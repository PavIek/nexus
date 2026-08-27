以下是完整的12周每日执行计划，附带所有资料链接。请严格按照 Day 1 → Day 84 的顺序推进，不要跳步骤。

🏗️ 第一阶段：基石与并发原语（第1-4周）
第1周：项目初始化与分片 Map
📖 本周必读资料：

Go Memory Model—— happens-before 关系的正式定义

runtime/HACKING.md—— 理解 //go:notinheap、系统栈 vs 用户栈等 Runtime 编程规范

Day 1：项目初始化

bash
mkdir nexus && cd nexus
go mod init github.com/<your_username>/nexus
mkdir -p cmd/nexus internal/shardmap internal/ringbuffer internal/netpoll pkg/metrics
touch cmd/nexus/main.go
在 cmd/nexus/main.go 中写入：

go
package main

func main() {}
🔗 资料：Go Modules Reference

Day 2-3：实现基础分片 Map

internal/shardmap/shardmap.go：

go
package shardmap

import "sync"

type Shard struct {
    mu   sync.RWMutex
    data map[string]int64
}

type ShardedMap struct {
    shards []*Shard
}

func New(numShards int) *ShardedMap {
    sm := &ShardedMap{shards: make([]*Shard, numShards)}
    for i := range sm.shards {
        sm.shards[i] = &Shard{data: make(map[string]int64)}
    }
    return sm
}

func (sm *ShardedMap) getShard(key string) *Shard {
    return sm.shards[hash(key)%len(sm.shards)]
}

func (sm *ShardedMap) Set(key string, val int64) {
    sh := sm.getShard(key)
    sh.mu.Lock()
    defer sh.mu.Unlock()
    sh.data[key] = val
}

func (sm *ShardedMap) Get(key string) (int64, bool) {
    sh := sm.getShard(key)
    sh.mu.RLock()
    defer sh.mu.RUnlock()
    val, ok := sh.data[key]
    return val, ok
}
// TODO: 实现 hash 函数
🔗 资料：sync 包文档

Day 4-5：内存对齐与 Cacheline 填充

改造 Shard 结构，通过填充避免伪共享（False Sharing）：

go
type Shard struct {
    _    [56]byte // 填充至64字节，让 mu 和 data 独占 Cache Line
    mu   sync.RWMutex
    data map[string]int64
    _    [56]byte // 尾部填充
}
验证对齐：

bash
go run golang.org/x/tools/go/analysis/passes/fieldalignment@latest ./...
🔗 资料：

Go Memory Model - Happens Before

The Go Memory Model（官方完整版）

Day 6-7：基准测试与性能对比

internal/shardmap/shardmap_test.go：

go
func BenchmarkShardedMapSet(b *testing.B) {
    sm := New(64)
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            sm.Set("key", 1)
        }
    })
}
运行并对比：

bash
go test -bench=. -benchmem ./internal/shardmap/...
go test -bench=. -benchmem -cpu=1,2,4,8 ./internal/shardmap/...
🔗 资料：testing 包文档

第2周：无锁 Ring Buffer
📖 本周必读资料：

sync/atomic 包文档——无锁编程的核心工具

Day 8-9：SPSC Ring Buffer 设计

internal/ringbuffer/ringbuffer.go：

go
package ringbuffer

type RingBuffer[T any] struct {
    buffer   []T
    writeIdx uint64
    readIdx  uint64
    mask     uint64
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
    // capacity 必须是2的幂
    return &RingBuffer[T]{
        buffer: make([]T, capacity),
        mask:   uint64(capacity - 1),
    }
}
🔗 资料：atomic 包文档

Day 10-11：实现 Push/Pop（CAS 操作）

go
import "sync/atomic"

func (rb *RingBuffer[T]) Push(val T) bool {
    for {
        w := atomic.LoadUint64(&rb.writeIdx)
        r := atomic.LoadUint64(&rb.readIdx)
        if w-r >= uint64(len(rb.buffer)) {
            return false
        }
        if atomic.CompareAndSwapUint64(&rb.writeIdx, w, w+1) {
            rb.buffer[w&rb.mask] = val
            return true
        }
    }
}

func (rb *RingBuffer[T]) Pop() (T, bool) {
    var zero T
    for {
        r := atomic.LoadUint64(&rb.readIdx)
        w := atomic.LoadUint64(&rb.writeIdx)
        if r >= w {
            return zero, false
        }
        if atomic.CompareAndSwapUint64(&rb.readIdx, r, r+1) {
            return rb.buffer[r&rb.mask], true
        }
    }
}
🔗 资料：CompareAndSwap 官方文档

Day 12-13：Pop 实现与性能测试

internal/ringbuffer/ringbuffer_test.go：

go
func BenchmarkRingBufferPush(b *testing.B) {
    rb := NewRingBuffer[int](1024)
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            rb.Push(1)
        }
    })
}
对比 channel：

bash
go test -bench=. -benchmem ./internal/ringbuffer/...
Day 14：第一周复盘

🔗 资料：Go Concurrency Patterns

第3周：基础网络层
📖 本周必读资料：

net 包文档

Day 15-16：TCP Server 搭建

cmd/nexus/main.go：

go
package main

import (
    "log"
    "net"
)

func main() {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    defer ln.Close()
    
    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Print(err)
            continue
        }
        go handleConn(conn)
    }
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    // TODO: 读取数据
}
🔗 资料：net.Conn 文档

Day 17-18：TCP 粘包/拆包处理

实现 4 字节长度前缀协议：

go
func readMessage(conn net.Conn) ([]byte, error) {
    var length uint32
    if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
        return nil, err
    }
    data := make([]byte, length)
    _, err := io.ReadFull(conn, data)
    return data, err
}
🔗 资料：io.ReadFull 文档

Day 19-20：集成 ShardedMap 与 RingBuffer

将解析到的数据通过分片路由到 RingBuffer：

go
type Worker struct {
    rb *ringbuffer.RingBuffer[[]byte]
}

func (w *Worker) Run() {
    for {
        data, ok := w.rb.Pop()
        if !ok {
            continue
        }
        // 处理数据
    }
}
Day 21：引入 sync.Pool

go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    // 使用 buf 读取数据
}
🔗 资料：sync.Pool 文档

第4周：深化并发控制
📖 本周必读资料：

Scalable Go Scheduler Design Doc—— Dmitry Vyukov 2012 年撰写，GMP 模型的起源

runtime 包文档

Day 22-23：深入 GMP 模型

阅读核心源码：

bash
cat $GOROOT/src/runtime/runtime2.go  # G、M、P 的结构定义
cat $GOROOT/src/runtime/proc.go      # schedule() 调度主循环
🔗 资料：

Scalable Go Scheduler Design Doc（转载）—— 完整阐述了 P 的引入动机

Go Preemptive Scheduler Design Doc—— 2013 年抢占式调度设计

Day 24-25：实验 runtime.LockOSThread

go
func init() {
    runtime.LockOSThread()  // 将当前 G 锁定在 OS 线程上
}

func main() {
    runtime.GOMAXPROCS(4)
    // 观察调度行为
}
观察调度：

bash
GODEBUG=schedtrace=1000 ./nexus
🔗 资料：runtime.LockOSThread 文档

Day 26-27：探索 Gosched 与 Goexit

go
func worker() {
    for i := 0; i < 10; i++ {
        if i%2 == 0 {
            runtime.Gosched()  // 主动让出时间片
        }
    }
    runtime.Goexit()  // 终止当前 Goroutine
}
🔗 资料：runtime.Gosched 文档

Day 28：月度复盘

🔗 资料：Go Blog: The Go Scheduler

⚙️ 第二阶段：深入运行时与网络层（第5-8周）
第5周：自制 Netpoll
📖 本周必读资料：

netpoll.go 源码

golang.org/x/sys/unix 文档

Day 29-31：Epoll 系统调用

internal/netpoll/epoll.go：

go
package netpoll

import "golang.org/x/sys/unix"

type Epoll struct {
    fd int
}

func NewEpoll() (*Epoll, error) {
    fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
    if err != nil {
        return nil, err
    }
    return &Epoll{fd: fd}, nil
}

func (e *Epoll) Add(fd int) error {
    return unix.EpollCtl(e.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
        Events: unix.EPOLLIN | unix.EPOLLET,
        Fd:     int32(fd),
    })
}
🔗 资料：unix.EpollCreate1 文档

Day 32-33：封装非阻塞 Conn

go
func SetNonblock(fd int) error {
    return unix.SetNonblock(fd, true)
}
🔗 资料：unix.SetNonblock 文档

Day 34-35：事件循环

go
func (e *Epoll) Wait() ([]unix.EpollEvent, error) {
    events := make([]unix.EpollEvent, 128)
    n, err := unix.EpollWait(e.fd, events, -1)
    if err != nil {
        return nil, err
    }
    return events[:n], nil
}
🔗 资料：unix.EpollWait 文档

第6周：调度策略与性能调优
📖 本周必读资料：

Go Preemptive Scheduler Design Doc—— 理解抢占式调度的设计动机

Day 36-38：Goroutine 池与 CPU 绑定

go
type Pool struct {
    workers []*Worker
}

func NewPool(numWorkers int) *Pool {
    p := &Pool{}
    for i := 0; i < numWorkers; i++ {
        w := &Worker{id: i}
        go w.Run()
        p.workers = append(p.workers, w)
    }
    return p
}
🔗 资料：runtime.GOMAXPROCS 文档

Day 39-40：Work Stealing 实现

go
func (p *Pool) Steal(from int) *Task {
    // 从其他 Worker 的队列中偷取任务
    // 使用 atomic CAS 实现无锁偷取
}
🔗 资料：Work-Stealing 论文

Day 41-42：压力测试与调度分析

bash
GODEBUG=schedtrace=1000,scheddetail=1 ./nexus
🔗 资料：GODEBUG 环境变量文档

第7周：内存分配器深度探索
📖 本周必读资料：

A Guide to the Go Garbage Collector—— Go 1.19+ GC 完整指南

malloc.go 源码

Day 43-45：无指针数据结构设计

go
type RingBuffer[T any] struct {
    buffer []T  // 存储值类型，而非指针
    // ...
}
🔗 资料：GC Guide - Stack Allocation

Day 46-48：手动内存管理

结合 sync.Pool 和数组索引管理对象：

go
type ObjectPool struct {
    pool sync.Pool
}

func (p *ObjectPool) Get() *Object {
    return p.pool.Get().(*Object)
}
🔗 资料：GC Guide - GOGC

Day 49：分析 GC 日志

bash
GODEBUG=gctrace=1 ./nexus 2>&1 | grep gc
输出示例：

text
gc 1 @0.001s 2%: 0.018+0.43+0.017 ms clock, 0.14+0.047/0.092/0.12+0.14 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
🔗 资料：GC Guide - Understanding GC

第8周：阶段整合
Day 50-53：整合 Netpoll + RingBuffer + Goroutine Pool

🔗 资料：runtime/trace 包文档

Day 54-56：集成测试与压力测试

bash
go test -bench=. -benchmem -cpu=1,2,4,8 ./...
go test -bench=. -benchmem -cpu=1,2,4,8 -benchtime=30s ./...
🔗 资料：testing - Benchmarks

🛠️ 第三阶段：系统化与可观测性（第9-12周）
第9周：构建可观测性体系
📖 本周必读资料：

runtime/metrics 包文档—— Go 1.16+ 运行时指标稳定接口

runtime/pprof 包文档—— 性能分析数据采集

Profiling Go Programs—— Go 官方性能分析教程

Day 57-60：集成 runtime/metrics

pkg/metrics/metrics.go：

go
package metrics

import (
    "runtime/metrics"
)

func Read() map[string]uint64 {
    descriptors := metrics.All()
    samples := make([]metrics.Sample, len(descriptors))
    for i, d := range descriptors {
        samples[i].Name = d.Name
    }
    metrics.Read(samples)
    // 处理 samples
}
🔗 资料：runtime/metrics 示例

Day 61-63：使用 runtime/trace

在关键路径埋点：

go
import "runtime/trace"

func processRequest(ctx context.Context) {
    ctx, task := trace.NewTask(ctx, "processRequest")
    defer task.End()
    
    trace.WithRegion(ctx, "decode", decode)
    trace.WithRegion(ctx, "process", process)
}
生成并分析 trace：

bash
go test -trace=trace.out ./...
go tool trace trace.out
🔗 资料：

runtime/trace 包文档

More powerful Go execution traces—— Go 官方执行追踪器详解

第10周：混沌工程与边缘测试
Day 64-67：模拟极端场景

bash
# 手动触发 GC
kill -SIGUSR1 <pid>

# 在代码中触发
runtime.GC()
🔗 资料：runtime.GC 文档

Day 68-70：韧性与恢复测试

go
func chaosTest() {
    // 模拟 Goroutine 暴涨
    for i := 0; i < 100000; i++ {
        go func() { time.Sleep(time.Hour) }()
    }
    // 观察调度器行为
}
第11周：代码硬核化
📖 本周必读资料：

A Quick Guide to Go's Assembler—— Plan 9 汇编语言官方文档

cmd/asm/doc.go—— go tool asm 命令文档

Day 71-74：性能极致优化

使用编译器指令：

go
//go:noescape
func fastHash(data []byte) uint64

//go:nosplit
func hotPath() {
    // 跳过栈溢出检查
}
查看汇编输出：

bash
go tool compile -S -N -l main.go
🔗 资料：Go Assembly 官方文档

Day 75-77：编写最终基准测试

bash
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...
go tool pprof -http=:8080 cpu.prof
🔗 资料：pprof README

第12周：社区反馈与总结
Day 78-80：提炼最小复现示例

bash
# 将发现的问题提炼成 < 100 行的测试用例
# 提交到 https://github.com/golang/go/issues
🔗 资料：Go Issue Tracker

Day 81-83：准备社区贡献

提交 Issue 时的模板：

markdown
### What version of Go are you using?
go version go1.21.0 linux/amd64

### Does this issue reproduce with the latest release?
Yes

### What did you do?
[最小复现代码]

### What did you expect to see?
[期望行为]

### What did you see instead?
[实际行为]
Day 84：项目总结与文章撰写

撰写技术文章，主题建议：

"Deep Dive into Go Scheduler: A Case Study"

"Building a Zero-GC Event Processing Engine in Go"

🔗 资料：Go 官方博客

📋 资料速查总表
类别	资料	链接
调度器设计	Scalable Go Scheduler Design Doc (2012)	Google Docs
抢占式调度	Go Preemptive Scheduler Design Doc (2013)	Google Docs
内存模型	The Go Memory Model	go.dev/ref/mem
GC 指南	A Guide to the Go Garbage Collector	tip.golang.org/doc/gc-guide
性能分析	Profiling Go Programs (Blog)	blog.golang.org/profiling-go-programs
执行追踪	More powerful Go execution traces	go.dev/blog/execution-traces-2024
汇编语言	A Quick Guide to Go's Assembler	tip.golang.org/doc/asm
Runtime 规范	runtime/HACKING.md	go.googlesource.com
这不是一份可以“浏览”的计划。每行代码都要亲手敲，每个命令都要亲自跑。84天后，Go Runtime 对你将不再是黑盒。
