package audio

// StreamMultiplier creates N output channel copies of the input
func StreamMultiplier(done chan struct{}, in chan []float64, n int) []chan []float64 {
	out := make([]chan []float64, n)
	for i := range out {
		out[i] = make(chan []float64)
	}

	go func() {
		for i := range out {
			defer close(out[i])
		}

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

			for i := range out {
				out[i] <- x
			}
		}
	}()

	return out
}
