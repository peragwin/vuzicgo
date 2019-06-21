package util

import (
	"sync"
)

// RingBuffer implements a circular buffer.
type RingBuffer struct {
	sync.RWMutex
	buf   []float64
	index int
}

// NewRingBuffer creates a new ring buffer with the given size.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{buf: make([]float64, size)}
}

// Push data onto the ring buffer.
func (r *RingBuffer) Push(data []float64) {
	if len(data) > len(r.buf) {
		panic("cant push data longer than size of buffer")
	}

	r.Lock()
	defer r.Unlock()

	wrap := false
	en := r.index + len(data)
	if en > len(r.buf) {
		en = len(r.buf)
		wrap = true
	}
	for i := r.index; i < en; i++ {
		r.buf[i] = data[i-r.index]
	}
	if wrap {
		os := len(r.buf) - r.index
		for i := 0; i < len(data)-os; i++ {
			r.buf[i] = data[i+os]
		}
	}

	r.index = (r.index + len(data)) % len(r.buf)
}

// Get the most recent N data points from the buffer.
func (r *RingBuffer) Get(size int) []float64 {
	return r.GetOffset(size, 0)
}

// GetOffset gets the most recent N data points from the buffer, offset minus M samples.
func (r *RingBuffer) GetOffset(size, offset int) []float64 {
	if size > len(r.buf) {
		panic("cant get size greater than size of buffer")
	}

	r.RLock()
	defer r.RUnlock()

	ret := make([]float64, size)

	wrap := false
	index := r.index - offset
	st := index - size
	en := index
	if st < 0 {
		st = len(r.buf) + st
		en = len(r.buf)
		wrap = true
	}
	for i := st; i < en; i++ {
		ret[i-st] = r.buf[i]
	}
	if wrap {
		for i := 0; i < index; i++ {
			ret[i+en-st] = r.buf[i]
		}
	}

	return ret
}
