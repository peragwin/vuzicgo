package audio

import (
	"context"
	"fmt"

	"github.com/gordonklaus/portaudio"
)

// Config represents a config that is used to open a new Stream.
type Config struct {
	// BlockSize refers to the buffer size for each block
	BlockSize int
	// Channels is the number of input channeles
	Channels int
	// SampleRate is the sample rate (Fs).
	SampleRate float64
}

// NewSource initializes a new streaming source with portaudio and returns a channel on which
// to receive frames.
func NewSource(ctx context.Context, cfg *Config) (<-chan []float32, <-chan error) {
	out := make(chan []float32)
	errc := make(chan error, 1)
	done := ctx.Done()

	go func() {
		defer close(out)

		in := make([]float32, cfg.BlockSize)

		stream, err := portaudio.OpenDefaultStream(
			cfg.Channels, 0, cfg.SampleRate, cfg.BlockSize, in)
		if err != nil {
			errc <- fmt.Errorf("Error opening stream: %v", err)
			return
		}
		defer stream.Close()
		if err := stream.Start(); err != nil {
			errc <- fmt.Errorf("Error starting stream: %v", err)
			return
		}

		for {
			select {
			case <-done:
				return
			default:
			}

			err := stream.Read()
			if err != nil {
				errc <- fmt.Errorf("Error reading from stream: %v", err)
				return
			}

			out <- in
		}
	}()

	return out, errc
}
