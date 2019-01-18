package audio

import (
	"log"

	"github.com/peragwin/vuzicgo/audio/util"
)

// Buffer turns every incoming frame into overlapping outgoing frames of the given size.
// @size must be strictly >= the size of the input, which is calculated from the first frame.
// It also converts the float32 input from a raw audio source to float64 so it's easier
// to work with down the line using go's math package.
func Buffer(done chan struct{}, in <-chan []float32, size int) chan []float64 {

	out := make(chan []float64, 16) // allocate a small buffer for

	go func() {
		defer close(out)
		var (
			x      []float32
			y      []float64
			buffer = util.NewRingBuffer(size)
		)

		for {
			select {
			case <-done:
				return
			case x = <-in:
				if x == nil {
					return
				}
				if y == nil {
					y = make([]float64, len(x))
				}

				for i := range x {
					y[i] = float64(x[i])
				}
				buffer.Push(y)

				select {
				case out <- buffer.Get(size):
				default:
					log.Println("[WARNING] Input buffer overrun! Frame was dropped.")
				}
			}
		}
	}()

	return out
}
