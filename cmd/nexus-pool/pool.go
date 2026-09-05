package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

const queueSize = 1024

type Pool struct {
	workers []*Worker
	next    atomic.Uint64
	stop    chan struct{}
	wg      sync.WaitGroup
}

func NewPool(numWorkers int) *Pool {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(numWorkers)

	p := &Pool{
		workers: make([]*Worker, 0, numWorkers),
		stop:    make(chan struct{}),
	}
	for i := 0; i < numWorkers; i++ {
		w := &Worker{
			id:    i,
			pool:  p,
			queue: newTaskQueue(queueSize),
		}
		p.workers = append(p.workers, w)
		p.wg.Add(1)
		go w.Run()
	}
	return p
}

func (p *Pool) Submit(task Task) bool {
	if len(p.workers) == 0 {
		return false
	}

	taskPtr := &task
	start := int(p.next.Add(1)-1) % len(p.workers)
	for i := 0; i < len(p.workers); i++ {
		idx := (start + i) % len(p.workers)
		if p.workers[idx].queue.push(taskPtr) {
			return true
		}
	}
	return false
}

func (p *Pool) Steal(from int) *Task {
	if len(p.workers) < 2 {
		return nil
	}

	for offset := 1; offset < len(p.workers); offset++ {
		victim := (from + offset) % len(p.workers)
		if task, ok := p.workers[victim].queue.pop(); ok {
			return task
		}
	}
	return nil
}

func (p *Pool) Close() {
	close(p.stop)
	p.wg.Wait()
}

type Worker struct {
	id    int
	pool  *Pool
	queue *taskQueue
}

type Task struct {
	Key  string
	Data []byte
}

func (w *Worker) Run() {
	defer w.pool.wg.Done()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := pinCurrentThread(w.id % runtime.GOMAXPROCS(0)); err != nil {
		fmt.Printf("worker %d cpu bind skipped: %v\n", w.id, err)
	}

	for {
		select {
		case <-w.pool.stop:
			return
		default:
		}

		task, ok := w.queue.pop()
		if !ok {
			task = w.pool.Steal(w.id)
		}
		if task == nil {
			runtime.Gosched()
			continue
		}

		processTask(*task)
	}
}
