package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"sync"
	"time"
)

var (
	// Shared resources for mutex contention testing
	sharedMutex   sync.Mutex
	sharedCounter int64
	dataMutex     sync.RWMutex
	sharedData    = make(map[string]int)
)

// cpuIntensiveWork performs CPU-intensive operations
func cpuIntensiveWork() {
	// Perform mathematical calculations
	result := 0.0
	for i := 0; i < 100000; i++ {
		result += math.Sqrt(float64(i)) * math.Sin(float64(i))
	}
}

// memoryIntensiveWork allocates and uses memory
func memoryIntensiveWork() {
	// Allocate multiple chunks of memory
	data := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		// Allocate 1MB chunks
		data[i] = make([]byte, 1024*1024)
		// Fill with random data to ensure allocation
		rand.Read(data[i])
	}

	// Do some work with the data to prevent optimization
	for i := 0; i < len(data); i++ {
		for j := 0; j < 100; j++ {
			data[i][j] = byte(i + j)
		}
	}
}

// mutexContentionWork creates mutex contention
func mutexContentionWork(workerID int) {
	for i := 0; i < 1000; i++ {
		// Contend for shared counter
		sharedMutex.Lock()
		sharedCounter++
		sharedMutex.Unlock()

		// Contend for shared data with read/write pattern
		key := fmt.Sprintf("worker-%d", workerID)

		// Write operation
		dataMutex.Lock()
		sharedData[key] = i + 1
		dataMutex.Unlock()

		// Read operation
		dataMutex.RLock()
		_ = sharedData[key]
		dataMutex.RUnlock()

		// Small delay to simulate real work
		time.Sleep(time.Microsecond)
	}
}

// blockingWork creates blocking operations
func blockingWork(workerID int) {
	// Channel operations that will block
	ch := make(chan int, 1)
	done := make(chan bool)

	// Producer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ch <- i
			time.Sleep(time.Millisecond)
		}
		close(ch)
	}()

	// Consumer goroutine with blocking receives
	go func() {
		for range ch {
			// Simulate work
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for completion
	<-done

	// Additional blocking on timer
	timer := time.NewTimer(100 * time.Millisecond)
	<-timer.C
}

// goroutineContentionWork creates many goroutines
func goroutineContentionWork() {
	var wg sync.WaitGroup

	// Create multiple goroutines that do work
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine does some blocking work
			blockingWork(id)

			// And some CPU work
			for j := 0; j < 10000; j++ {
				_ = j * j
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}
