package main

import (
	"fmt"

	"example.com/nexus/internal/ringbuffer"
	"example.com/nexus/internal/shardmap"
)

const workerCount = 16

var workers []*Worker
var store = shardmap.New(64)

func initWorkers() {
	workers = make([]*Worker, workerCount)

	for i := 0; i < workerCount; i++ {
		w := &Worker{
			rb: ringbuffer.NewRingBuffer[Task](1024),
			sm: store,
		}
		workers[i] = w
		go w.Run()
	}
}

type Task struct {
	Key  string
	Data []byte
}

type Worker struct {
	rb *ringbuffer.RingBuffer[Task]
	sm *shardmap.ShardedMap
}

func (w *Worker) Run() {
	for {
		task, ok := w.rb.Pop()
		if !ok {
			continue
		}

		fmt.Println(fmt.Sprintf("process task: [%+v]", task), string(task.Data))

		w.sm.Set(task.Key, int64(len(task.Data)))
	}
}
