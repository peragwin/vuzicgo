package util

import "testing"

func TestBucketer(t *testing.T) {
	size := 512
	frame := make([]float64, size)
	for i := range frame {
		frame[i] = 1.0
	}

	// test and print MelScale
	b := NewBucketer(MelScale, 64, size, 32, 16000)
	t.Log(b.indices)
	buckets := b.Bucket(frame)
	t.Log(buckets, len(buckets))

	// test and print LogScale
	b = NewBucketer(LogScale, 60, size, 32, 16000)
	t.Log(b.indices)
	buckets = b.Bucket(frame)
	t.Log(buckets, len(buckets))
}
