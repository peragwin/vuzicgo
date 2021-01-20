package audio

// NewNodeF64F64 creates a new audio processing node F64->F64
func NewNodeF64F64(done chan struct{}, in chan []float64, nodeFunc func([]float64) []float64) chan []float64 {
	out := make(chan []float64)

	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case frame := <-in:
				if frame == nil {
					return
				}
				out <- nodeFunc(frame)
			}
		}
	}()

	return out
}
