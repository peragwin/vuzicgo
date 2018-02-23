package audio

// TimeDelay is a processor that outputs slice of the input that has been delayed
// by @delay samples.
func TimeDelay(done chan struct{}, in <-chan []float64, frameSize, delay int) chan []float64 {

	out := make(chan []float64)
	var x []float64

	additionalFrames := 1 + delay/frameSize
	numFrames := 1 + additionalFrames
	bufferSize := frameSize * numFrames

	go func() {
		defer close(out)
		y := make([]float64, bufferSize)
		bufferIndex := 0
		for {
			select {
			case <-done:
				return
			default:
			}
			x = <-in
			if x == nil {
				return
			}

			offset := bufferIndex * frameSize
			copy(y[offset:], x)

			start := offset - delay
			stop := start + frameSize
			out <- circSlice(y, bufferSize, start, stop)

			bufferIndex %= numFrames
		}
	}()

	return out
}

func circSlice(y []float64, bufferSize, start, stop int) []float64 {
	var o []float64
	if start >= 0 {
		if stop <= bufferSize {
			//fmt.Println("case1")
			o = y[start:stop]
		} else {
			//fmt.Println("case2")
			o = append(y[start:], y[:bufferSize-stop]...)
		}
	} else {
		start += bufferSize
		if stop < 0 {
			stop += bufferSize
			//fmt.Println("case3", start, stop)
			o = y[start:stop]
		} else {
			//fmt.Println("case4", start, stop)
			o = append(y[start:], y[:stop]...)
		}
	}
	return o
}
