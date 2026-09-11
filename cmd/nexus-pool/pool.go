package main

type Pool struct {
	workers []*
}

type Worker struct {
	id int
	pool *Pool
	queue *taskQueue
}

type Task struct {
	Key string
	Data []byte
}