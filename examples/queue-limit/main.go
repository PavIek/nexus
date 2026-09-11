package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// 模拟队列处理，返回所有请求的排队等待时间（毫秒）
func simulateQueue(queueSize, totalRequests int, serviceTime time.Duration) []float64 {
	// 带缓冲的 channel 就是我们的“队列”
	ch := make(chan time.Time, queueSize)
	var wg sync.WaitGroup
	var mu sync.Mutex
	waitTimes := []float64{}

	// 1. 启动消费者（处理速度固定）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for enqueueTime := range ch {
			// 计算排队等待时间（从入队到出队）
			waitMs := float64(time.Since(enqueueTime).Microseconds()) / 1000.0
			mu.Lock()
			waitTimes = append(waitTimes, waitMs)
			mu.Unlock()

			// 模拟固定处理耗时（10ms）
			time.Sleep(serviceTime)
		}
	}()

	// 2. 生产者：瞬间突发 100 个请求
	for i := 0; i < totalRequests; i++ {
		select {
		case ch <- time.Now(): // 入队成功
		default:
			// 队列满了！立刻拒绝（快速失败），等待时间为 0（因为根本没处理）
			mu.Lock()
			waitTimes = append(waitTimes, 0)
			mu.Unlock()
			// 注意：为了模拟真实拒绝，这里不阻塞
		}
	}
	close(ch)
	wg.Wait()
	return waitTimes
}

// 计算 P99（百分位数）
func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func main() {
	serviceTime := 10 * time.Millisecond
	totalReqs := 100

	fmt.Println("========== 场景 1：大缓冲区（Queue Size = 1000）==========")
	times1 := simulateQueue(1000, totalReqs, serviceTime)
	fmt.Printf("平均等待时间: %.2f ms\n", avg(times1))
	fmt.Printf("P99 等待时间: %.2f ms\n", percentile(times1, 0.99))
	fmt.Printf("最大等待时间: %.2f ms (第 100 个请求被堵了 99 个坑位)\n", max(times1))

	fmt.Println("\n========== 场景 2：小缓冲区 + 快速拒绝（Queue Size = 10）==========")
	times2 := simulateQueue(10, totalReqs, serviceTime)
	fmt.Printf("平均等待时间: %.2f ms\n", avg(times2))
	fmt.Printf("P99 等待时间: %.2f ms\n", percentile(times2, 0.99))
	fmt.Printf("最大等待时间: %.2f ms (第 11 个请求直接拒绝，不排队)\n", max(times2))

	// 统计被拒绝的请求数（等待时间为 0 且排在后面的，这里简化计算）
	rejected := 0
	for _, v := range times2 {
		if v == 0 {
			rejected++
		}
	}
	fmt.Printf("被快速拒绝的请求数: %d 个 (这些请求耗费 0ms 返回错误，让上游重试)\n", rejected)
}

func avg(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func max(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data {
		if v > m {
			m = v
		}
	}
	return m
}
