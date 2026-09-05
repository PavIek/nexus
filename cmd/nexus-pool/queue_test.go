package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestTaskQueuePushPop(t *testing.T) {
	q := newTaskQueue(2)

	if !q.push(&Task{Key: "a"}) {
		t.Fatal("push a failed")
	}
	if !q.push(&Task{Key: "b"}) {
		t.Fatal("push b failed")
	}
	if q.push(&Task{Key: "c"}) {
		t.Fatal("push succeeded on full queue")
	}

	task, ok := q.pop()
	if !ok || task.Key != "a" {
		t.Fatalf("first pop = %+v, ok=%v", task, ok)
	}
	task, ok = q.pop()
	if !ok || task.Key != "b" {
		t.Fatalf("second pop = %+v, ok=%v", task, ok)
	}
	if _, ok := q.pop(); ok {
		t.Fatal("pop succeeded on empty queue")
	}
}

func TestPoolSteal(t *testing.T) {
	p := &Pool{
		workers: []*Worker{
			{id: 0, queue: newTaskQueue(4)},
			{id: 1, queue: newTaskQueue(4)},
		},
	}

	if !p.workers[1].queue.push(&Task{Key: "stolen"}) {
		t.Fatal("push failed")
	}

	task := p.Steal(0)
	if task == nil || task.Key != "stolen" {
		t.Fatalf("stolen task = %+v", task)
	}
}

func TestPoolSubmitConcurrent(t *testing.T) {
	p := &Pool{
		workers: []*Worker{
			{id: 0, queue: newTaskQueue(128)},
			{id: 1, queue: newTaskQueue(128)},
			{id: 2, queue: newTaskQueue(128)},
			{id: 3, queue: newTaskQueue(128)},
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if !p.Submit(Task{Key: fmt.Sprintf("task-%d", i)}) {
				t.Errorf("submit %d failed", i)
			}
		}(i)
	}
	wg.Wait()

	count := 0
	for _, worker := range p.workers {
		for {
			_, ok := worker.queue.pop()
			if !ok {
				break
			}
			count++
		}
	}
	if count != 100 {
		t.Fatalf("queued task count = %d, want 100", count)
	}
}
