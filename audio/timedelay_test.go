package audio

import (
	"fmt"
	"testing"
)

func TestCircSlice(t *testing.T) {
	frameSize := 4
	numFrames := 4
	bufferSize := frameSize * numFrames
	delay := 12
	f := make([]float64, bufferSize)
	for i := range f {
		f[i] = float64(i)
	}

	for i := 0; i < numFrames; i++ {
		offset := frameSize * i
		start := offset - delay
		stop := start + frameSize
		fmt.Println("bounds:", start, stop)
		o := circSlice(f, bufferSize, start, stop)
		fmt.Println(o)
		if len(o) != frameSize {
			t.Error(o)
		}
		if o[0] != float64((bufferSize+start)%bufferSize) {
			t.Error(o)
		}
	}
}
