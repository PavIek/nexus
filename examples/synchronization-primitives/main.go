package main

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

var requests atomic.Int64

func handleRequest() {
	requests.Add(1)
}

var shutdown atomic.Int32

func mainLoop() {
	for {
		if shutdown.Load() == 1 {
			break
		}
		// do work
	}
}

func stop() {
	shutdown.Store(1)
}

type MyResource struct{}

var resource unsafe.Pointer
var initStatus int32

func getResource() *MyResource {
	if atomic.LoadInt32(&initStatus) == 2 {
		return (*MyResource)(atomic.LoadPointer(&resource))
	}

	if atomic.CompareAndSwapInt32(&initStatus, 0, 1) {
		newRes := expensiveInit()
		atomic.StorePointer(&resource, unsafe.Pointer(newRes))
		atomic.StoreInt32(&initStatus, 2)
		return newRes
	}

	for atomic.LoadInt32(&initStatus) != 2 {
		runtime.Gosched()
	}
	return (*MyResource)(atomic.LoadPointer(&resource))
}

func expensiveInit() *MyResource {
	return &MyResource{}
}

type node struct {
	next *node
	val  any
}

var head atomic.Pointer[node]

func push(n *node) {
	for {
		old := head.Load()
		n.next = old
		if head.CompareAndSwap(old, n) {
			return
		}
	}
}
