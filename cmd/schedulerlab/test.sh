#!/bin/bash

go build -o /tmp/schedulerlab ./cmd/schedulerlab

GODEBUG=schedtrace=1000 /tmp/schedulerlab -duration=6s -procs=4 -workers=8

LOCK_OS_THREAD=1 GODEBUG=schedtrace=1000 /tmp/schedulerlab -duration=6s -procs=4 -workers=8




# 看重點：
# - main_tid=...：main goroutine 當前所在的 OS thread id。
# - 不鎖時，main_tid 可能會變；我剛測到從 2 變到 7。
# - 鎖時，main_tid 應該固定；我剛測到一直是 2。
# - worker=xx tid=yy 會顯示 worker goroutine 在不同 OS thread 間遷移。
# - schedtrace 裡的 gomaxprocs=4 代表 P 的數量；threads=N 是 runtime 實際開出的 OS thread 數，兩者不是一回事。
# - 第一行 SCHED 0ms 可能還是機器 CPU 數，因為那是在 main() 裡 runtime.GOMAXPROCS(4) 執行前打出的啟動快照。