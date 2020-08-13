package audio

import (
	"github.com/peragwin/vuzicgo/audio/util"
	"github.com/golang/glog"
)

// Buffer turns every incoming frame into overlapping outgoing frames of the given size.
// @size must be strictly >= the size of the input, which is calculated from the first frame.
// It also converts the float32 input from a raw audio source to float64 so it's easier
// to work with down the line using go's math package.
func Buffer(done chan struct{}, in <-chan []float32, size int) chan []float64 {

	out := make(chan []float64, 1) // allocate a small buffer for

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
					glog.V(3).Info("[WARNING] send buffer overrun! Frame was dropped.")
				}
			}
		}
	}()

	return out
}
