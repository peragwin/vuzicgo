package util

import "testing"

func TestBucketer(t *testing.T) {
	size := 512

	b := NewBucketer(MelScale, 16, size, 32, 16000)
	t.Log(b.indices)

	frame := make([]float64, size)
	for i := range frame {
		frame[i] = 1.0
	}

	buckets := b.Bucket(frame)
	t.Log(buckets, len(buckets))
}
