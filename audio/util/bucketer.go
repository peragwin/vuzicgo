package util

import (
	"log"
	"math"
)

// Scale represents the scale that is used to calculate bucket indices.
type Scale interface {
	To(float64) float64
	From(float64) float64
}

type melScale struct{}

// MelScale is a scale defined by how humans perceive pitch differences.
var MelScale *melScale

func (s *melScale) To(val float64) float64 {
	return 1127 * math.Log(1+val/700)
}

func (s *melScale) From(val float64) float64 {
	return 700 * (math.Exp(val/1127.0) - 1)
}

type logScale struct{}

// LogScale is a typical log2 scale.
var LogScale *logScale

func (s *logScale) To(val float64) float64 {
	return math.Log(1 + val)
}
func (s *logScale) From(val float64) float64 {
	return math.Exp2(val) - 1
}

type logScaleX struct {
	x float64
}

// LogScale2 is log2 scale but with a 2x slope
var LogScale2 = &logScaleX{1}

func (s *logScaleX) To(val float64) float64 {
	return math.Log2(math.Log2(1 + val))
}
func (s *logScaleX) From(val float64) float64 {
	return math.Exp2(math.Exp2(val)) - 1
}


// Bucketer puts the specturn into N buckets using a mel frequency scale
type Bucketer struct {
	Buckets int
	Size    int
	Scale   Scale

	// generate N-1 indices to split a frame into N Buckets
	indices []int
}

// NewBucketer creates a new Bucketer for a frame of @frameSize based on @scale and N @buckets,
// starting with [0:@fMin] and going to [@fMax:<nyquist>].
func NewBucketer(scale Scale, buckets, frameSize int, fMin, fMax float64) *Bucketer {
	sMin := scale.To(fMin)
	sMax := scale.To(fMax)
	space := (sMax - sMin) / float64(buckets)
	indices := make([]int, buckets-1)
	lastIdx := 0
	offset := 1
	// in scale space, how far is a unit index
	offsetDelta := (sMax - sMin) / float64(frameSize)

	for i := range indices {
		// the bucket spacing needs to adjust if we've accumulated offset
		adjSpace := space - offsetDelta*float64(offset)/float64(buckets)

		v := scale.From(float64(i+1)*(adjSpace) + sMin + offsetDelta*float64(offset))
		idx := int(math.Ceil(
			float64(frameSize) * v / fMax))

		// this is a special edge case when requested number of buckets is too high
		// and we'd otherwise have two sequential index values that round to the same number.
		if idx <= lastIdx {
			idx = lastIdx + 1
			offset++
		}
		if idx >= frameSize {
			idx = frameSize - 1
		}

		indices[i] = idx
		lastIdx = idx
	}

	return &Bucketer{
		Buckets: buckets,
		Size:    frameSize,
		Scale:   scale,
		indices: indices,
	}
}

// Bucket applys b.Buckets rectangular windows on the incoming frame and returns the sum in
// each window in a len==b.Buckets []float64.
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
		buckets[i] = sum / float64(stop-start)
	}
	return buckets
}

// BucketProcessor is an asynchronous processor that puts incoming frames into buckets.
type BucketProcessor struct {
	Bucketer *Bucketer
}

// NewBucketProcessor creates a new bucket processor using a Bucketer.
func NewBucketProcessor(b *Bucketer) *BucketProcessor {
	return &BucketProcessor{b}
}

// Process kicks off a goroutine to process incoming @in frames and returns the output channel.
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
			if x == nil {
				return
			}
			out <- b.Bucketer.Bucket(x)
		}
	}()

	return out
}
