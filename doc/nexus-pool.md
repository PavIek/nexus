可以。`nexus-pool` 這版和 `nexus-noblock` 的關注點不同：

```text
nexus-noblock：重點是 socket / epoll / non-blocking IO
nexus-pool：重點是 goroutine pool / CPU 綁定 / work stealing
```

它的網路入口還是標準庫 `net.Listen`，也就是 Go runtime 幫你處理底層 netpoll；這個版本主要研究「消息讀出來之後，怎麼調度任務」。

**1. main 啟動 Pool**

入口在 [cmd/nexus-pool/main.go](/home/euclid/src/nexus/cmd/nexus-pool/main.go:29)：

```go
pool = NewPool(runtime.NumCPU())
defer pool.Close()
```

這裡根據 CPU 核數建立 worker pool。比如機器有 8 核，就啟動 8 個 worker。

`NewPool` 在 [pool.go](/home/euclid/src/nexus/cmd/nexus-pool/pool.go:19)。

**2. NewPool 初始化 worker**

```go
runtime.GOMAXPROCS(numWorkers)
```

這行設置 Go scheduler 同時可執行 Go code 的 P 數量。簡單理解：

```text
numWorkers 個 worker
numWorkers 個 GOMAXPROCS
盡量讓 worker 和 CPU 並行能力對齊
```

然後：

```go
w := &Worker{
    id:    i,
    pool:  p,
    queue: newTaskQueue(queueSize),
}
go w.Run()
```

每個 worker 有自己的本地隊列：

```text
worker 0 -> queue 0
worker 1 -> queue 1
worker 2 -> queue 2
...
```

這是 work stealing 的基礎：先吃自己的隊列，自己的空了，再去偷別人的。

**3. server 接收 TCP 連線**

回到 [main.go](/home/euclid/src/nexus/cmd/nexus-pool/main.go:33)：

```go
ln, err := net.Listen("tcp", ":8080")
```

這裡仍然是標準庫 TCP server。

然後：

```go
for {
    conn, err := ln.Accept()
    go handleConn(conn)
}
```

也就是原 nexus 的模型：

```text
一個 TCP connection
  -> 一個 goroutine handleConn
```

但 Go 的底層 fd 還是 non-blocking + runtime netpoll，這點和你前面追 Go 源碼能對上。

**4. handleConn 讀消息**

[handleConn](/home/euclid/src/nexus/cmd/nexus-pool/main.go:49) 裡：

```go
buf := bufferPool.Get().([]byte)
defer bufferPool.Put(buf)
```

每條連線拿一塊 4096 bytes 的 buffer，用完放回 `sync.Pool`，減少分配。

然後循環讀消息：

```go
msg, err := readMessageBuffer(conn, buf)
```

這裡協議和原 nexus 一樣：

```text
4 字節 length
後面跟 length 字節 body
```

解析在 [readMessageBuffer](/home/euclid/src/nexus/cmd/nexus-pool/main.go:72)。

**5. 把消息變成 Task**

讀到完整消息後：

```go
task := Task{
    Key:  extractKey(msg),
    Data: append([]byte(nil), msg...),
}
```

`append([]byte(nil), msg...)` 是拷貝一份消息。因為 `msg` 底層用的是復用 buffer，如果不拷貝，下一次讀取會覆蓋數據。

然後提交到 pool：

```go
for !pool.Submit(task) {
    fmt.Println("pool busy...")
    time.Sleep(10 * time.Millisecond)
}
```

這裡表示：如果所有 worker 隊列都滿了，就等待一下再重試。

**6. Pool.Submit 分配任務**

`Submit` 在 [pool.go](/home/euclid/src/nexus/cmd/nexus-pool/pool.go:42)。

```go
start := int(p.next.Add(1)-1) % len(p.workers)
```

這是 round-robin 起點。每次提交任務，從不同 worker 開始嘗試，避免所有任務都先塞 worker 0。

然後：

```go
for i := 0; i < len(p.workers); i++ {
    idx := (start + i) % len(p.workers)
    if p.workers[idx].queue.push(taskPtr) {
        return true
    }
}
```

意思是：

```text
先嘗試某個 worker 的隊列
如果滿了，試下一個
全部都滿，Submit 返回 false
```

**7. worker 執行循環**

worker 的主循環在 [pool.go](/home/euclid/src/nexus/cmd/nexus-pool/pool.go:88)。

啟動後先做：

```go
runtime.LockOSThread()
```

這表示這個 worker goroutine 綁定在當前 OS thread 上，不再被 Go scheduler 移到其他 thread。

然後 Linux 下嘗試：

```go
pinCurrentThread(w.id % runtime.GOMAXPROCS(0))
```

實現在 [affinity_linux.go](/home/euclid/src/nexus/cmd/nexus-pool/affinity_linux.go:7)。

它用：

```go
unix.SchedSetaffinity(0, &set)
```

把當前 thread 綁到指定 CPU。這就是 Day 36-38 的「CPU 綁定」。

**8. worker 先取自己的任務**

worker 循環裡：

```go
task, ok := w.queue.pop()
```

先從自己的本地隊列取任務。

如果有任務：

```go
processTask(*task)
```

`processTask` 在 [main.go](/home/euclid/src/nexus/cmd/nexus-pool/main.go:90)：

```go
store.Set(task.Key, int64(len(task.Data)))
```

也就是把消息長度寫進 sharded map。

**9. 自己沒任務就偷**

如果自己的隊列空了：

```go
task = w.pool.Steal(w.id)
```

`Steal` 在 [pool.go](/home/euclid/src/nexus/cmd/nexus-pool/pool.go:58)。

```go
for offset := 1; offset < len(p.workers); offset++ {
    victim := (from + offset) % len(p.workers)
    if task, ok := p.workers[victim].queue.pop(); ok {
        return task
    }
}
```

意思是：

```text
worker 3 沒任務
  -> 試著從 worker 4 偷
  -> 再試 worker 5
  -> ...
  -> 繞一圈
```

偷到就處理，偷不到就：

```go
runtime.Gosched()
```

主動讓出 CPU，避免空轉太兇。

**10. taskQueue 的 CAS 實現**

隊列在 [queue.go](/home/euclid/src/nexus/cmd/nexus-pool/queue.go:7)。

它是 bounded ring queue：

```go
type taskQueue struct {
    buffer []queueSlot
    mask   uint64
    head   atomic.Uint64
    tail   atomic.Uint64
}
```

每個 slot 有一個 sequence number：

```go
type queueSlot struct {
    seq  atomic.Uint64
    task atomic.Pointer[Task]
}
```

`push` 用 CAS 推進 `tail`：

```go
if q.tail.CompareAndSwap(tail, tail+1) {
    slot.task.Store(task)
    slot.seq.Store(tail + 1)
    return true
}
```

`pop` 用 CAS 推進 `head`：

```go
if q.head.CompareAndSwap(head, head+1) {
    task := slot.task.Swap(nil)
    slot.seq.Store(head + q.mask + 1)
    return task, task != nil
}
```

所以這個 queue 的重點是：

```text
不用 mutex
用 atomic CAS 搶 head/tail
用 seq 判斷 slot 是空、滿、還是被其他 goroutine 搶走了
```

**整體流程**

把整個 `nexus-pool` 串起來就是：

```text
main
  -> NewPool(runtime.NumCPU())
       -> GOMAXPROCS(numWorkers)
       -> 建 numWorkers 個 Worker
       -> 每個 Worker 一個本地 queue
       -> go worker.Run()

  -> net.Listen(":8080")
  -> Accept()
  -> go handleConn(conn)

handleConn
  -> readMessageBuffer()
  -> Task{Key, Data}
  -> pool.Submit(task)

Pool.Submit
  -> round-robin 找 worker queue
  -> push 成功就返回
  -> 全滿則返回 false

Worker.Run
  -> LockOSThread
  -> pinCurrentThread
  -> 先 pop 自己的 queue
  -> 沒有就 Steal()
  -> 偷不到就 Gosched()
  -> 偷到/取到就 processTask()
```

**和原 nexus 的差異**

原 nexus 是：

```text
根據 key hash 到固定 worker
worker 只處理自己的 ringbuffer
沒有偷任務
```

`nexus-pool` 是：

```text
Submit 時盡量分散到不同 worker
worker 空了會從其他 worker 偷任務
worker 可嘗試綁定 CPU
```

所以這版解決的是：

```text
某些 worker 忙，某些 worker 閒
```

通過 work stealing 讓閒 worker 幫忙消化其他隊列裡的任務。

**一個重要提醒**

這版是教學 MVP。現在的 `taskQueue` 是 CAS bounded queue，但不是嚴格經典的 Chase-Lev work-stealing deque。經典 work stealing 通常是：

```text
owner worker 從 bottom push/pop
thief 從 top steal
```

而目前這版是：

```text
本地 pop 和偷任務都調同一個 pop
```

它能展示 CAS + stealing 的概念，但後續如果你要更貼近論文和 Go scheduler，可以再演進成真正的 deque。