package util

import (
	"log"
	"math"
)

type Scale interface {
	To(float64) float64
	From(float64) float64
}

type melScale struct{}

var MelScale *melScale

func (s *melScale) To(val float64) float64 {
	return 1127 * math.Log(1+val/700)
}

func (s *melScale) From(val float64) float64 {
	return 700 * (math.Exp(val/1127.0) - 1)
}

// Bucketer puts the specturn into N buckets using a mel frequency scale
type Bucketer struct {
	Buckets int
	Size    int
	Scale   Scale

	// generate N-1 indices to split a frame into N Buckets
	indices []int
}

func NewBucketer(scale Scale, buckets, frameSize int, fMin, fMax float64) *Bucketer {
	sMin := scale.To(fMin)
	sMax := scale.To(fMax)
	space := (sMax - sMin) / float64(buckets)
	indices := make([]int, buckets-1)
	for i := range indices {
		idx := scale.From(float64(i+1) * space)
		indices[i] = int(math.Ceil(
			float64(frameSize) * idx / fMax))
	}
	return &Bucketer{
		Buckets: buckets,
		Size:    frameSize,
		Scale:   scale,
		indices: indices,
	}
}

func (b *Bucketer) Bucket(frame []float64) []float64 {
	buckets := make([]float64, b.Buckets)
	if len(frame) != b.Size {
		log.Fatalf("Frame size %d does not match bucket size %d", len(frame), b.Size)
	}
	for i := range buckets {
		var start, stop int
		if i == 0 {
			start = 0
		} else {
			start = b.indices[i-1]
		}
		if i == len(buckets)-1 {
			stop = len(frame)
		} else {
			stop = b.indices[i]
		}
		var sum float64
		for j := start; j < stop; j++ {
			sum += frame[j]
		}
		buckets[i] = sum
	}
	return buckets
}

type BucketProcessor struct {
	Bucketer *Bucketer
}

func NewBucketProcessor(b *Bucketer) *BucketProcessor {
	return &BucketProcessor{b}
}

func (b *BucketProcessor) Process(done chan struct{}, in chan []float64) chan []float64 {
	out := make(chan []float64)

	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			default:
			}
			x := <-in
			out <- b.Bucketer.Bucket(x)
		}
	}()

	return out
}
