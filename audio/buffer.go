package audio

// Buffer turns every incoming frame into two outgoing frames which overlap by 50%.
// It also converts the float32 input from a raw audio source to float64 so it's easier
// to work with down the line using go's math package.
func Buffer(done chan struct{}, in <-chan []float32) chan []float64 {

	out := make(chan []float64, 2)

	x := <-in
	frameSize := len(x)

	go func() {
		defer close(out)
		y := make([]float64, frameSize*2)
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
			for i := range x {
				y[i+offset] = float64(x[i])
			}
			if bufferIndex == 1 {
				// z := y[frameSize/2 : 2*frameSize-frameSize/2]
				// out <- z
				out <- y[frameSize:]
			} else {
				// z := append(y[2*frameSize-frameSize/2:], y[:frameSize/2]...)
				// out <- z
				out <- y[:frameSize]
			}

			bufferIndex ^= 1
		}
	}()

	return out
}
