package main

import (
	"fmt"
	"sync/atomic"
)

type RunePool struct {
	buffers chan []rune
	bufSize int
	gets int64
	puts int64
}

// NewRunePool creates a bounded rune buffer pool with specified buffer size and count
func NewRunePool(bufSize int, poolSize int) (*RunePool) {
	if bufSize <= 0 || poolSize <= 0 {
		panic("RunePool: Invalid parameters")
	}

	runePool := &RunePool{
		buffers: make(chan []rune, poolSize),
		bufSize: bufSize,
		gets: 0,
		puts: 0,
	}

	for i := 0; i < poolSize; i++ {
		runePool.buffers <- make([]rune, 0, bufSize)
	}

	return runePool
}

// Get returns a clean buffer from the pool. Blocks if pool is empty.
func (rp *RunePool) Get() []rune {
	atomic.AddInt64(&rp.gets, 1)
	return <-rp.buffers
}

// Put returns a buffer to the pool. Drops oversized buffers.
func (rp *RunePool) Put(buf []rune) {
	atomic.AddInt64(&rp.puts, 1)

	var replacement []rune
	if cap(buf) != rp.bufSize {
		replacement = make([]rune, 0, rp.bufSize)
	} else {
		replacement = buf[:0]
	}

	select {
	case rp.buffers <- replacement:
		// Buffer successfully added back into pool
	default:
		// GC to handle. This should not happen, but
		// this makes sure the program continues to run if it does
		fmt.Println("Extra buffer returned")
	}
}

func (rp *RunePool) Report() (int64, int64) {
	gets := atomic.LoadInt64(&rp.gets)
	puts := atomic.LoadInt64(&rp.puts)

	return gets, puts
}

func (rp *RunePool) PrintReport() {
	gets, puts := rp.Report()
	fmt.Printf("Pool stats - Gets: %d, Puts: %d", gets, puts)
}