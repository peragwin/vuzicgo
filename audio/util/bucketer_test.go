package util

import (
	"testing"
)

var (
	minFrameSize = 256
	maxFrameSize = 4096
)

func TestBucketer(t *testing.T) {
	// p := int(math.Log2(float64(maxFrameSize / minFrameSize)))

	size := 512
	frame := make([]float64, size)
	for i := range frame {
		frame[i] = float64(i) * (24000) / float64(size) //1.0
	}

	// test and print MelScale
	b := NewBucketer(MelScale, 18, size, 32, 16000)
	t.Error(b.indices)
	buckets := b.Bucket(frame)
	t.Log(buckets, len(buckets))

	// test and print LogScale
	b = NewBucketer(LogScale2, 18, size, 32, 16000)
	t.Error(b.indices)
	buckets = b.Bucket(frame)
	t.Log(buckets, len(buckets))

	// test and print LogScale
	b = NewBucketer(LogScale, 18, size, 32, 16384)
	t.Error(b.indices)
	buckets = b.Bucket(frame)
	t.Log(buckets, len(buckets))

	// mark := struct {
	// 	mark  int
	// 	index int
	// 	mul   float64
	// }{
	// 	minFrameSize / 2, 1.0,
	// }

	// adj := make([]int, len(b.indices))
	// for i, idx := range b.indices {
	// 	if idx > mark.mark {
	// 		mark.mark += 1
	// 		mark.index += minFrameSize / 4
	// 		mark.mul *= 2
	// 	}
	// 	adj[i] = idx
	// }
}

func convertIndex() {}
