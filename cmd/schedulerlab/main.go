package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var lockedAtInit bool

func init() {
	if os.Getenv("LOCK_OS_THREAD") == "1" {
		runtime.LockOSThread()
		lockedAtInit = true
	}
}

func main() {
	procs := flag.Int("procs", 4, "GOMAXPROCS value")
	workers := flag.Int("workers", 8, "number of busy worker goroutines")
	duration := flag.Duration("duration", 6*time.Second, "experiment duration")
	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	fmt.Printf("pid=%d main_tid=%d locked_at_init=%v gomaxprocs=%d workers=%d duration=%s\n",
		os.Getpid(), syscall.Gettid(), lockedAtInit, runtime.GOMAXPROCS(0), *workers, *duration)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			lastTid := -1
			changes := 0

			for {
				select {
				case <-stop:
					return
				default:
				}

				tid := syscall.Gettid()
				if tid != lastTid && changes < 8 {
					fmt.Printf("worker=%02d tid=%d\n", id, tid)
					lastTid = tid
					changes++
				}

				busyFor(40 * time.Millisecond)
				runtime.Gosched()
			}
		}(i)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.After(*duration)
	for {
		select {
		case <-ticker.C:
			fmt.Printf("main_tid=%d goroutines=%d\n", syscall.Gettid(), runtime.NumGoroutine())
		case <-deadline:
			close(stop)
			wg.Wait()
			fmt.Printf("done main_tid=%d\n", syscall.Gettid())
			return
		}
	}
}

func busyFor(d time.Duration) {
	deadline := time.Now().Add(d)
	var x uint64
	for time.Now().Before(deadline) {
		x++
	}
	if x == 0 {
		fmt.Println("unreachable")
	}
}
