package audio

import (
	"context"
	"fmt"
	"log"

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
	// Device ID selects which audio device to use
	DeviceID int
	// PrintDevices prints the current audio devices then exits
	PrintDevices bool
	// LowLatency selects whether to use low latency for the audio source
	LowLatency bool
}

// NewSource initializes a new streaming source with portaudio and returns a channel on which
// to receive frames.
func NewSource(ctx context.Context, cfg *Config) (<-chan []float32, <-chan error) {
	out := make(chan []float32, 16)
	errc := make(chan error, 1)
	done := ctx.Done()

	go func() {
		defer close(out)

		portaudio.Initialize()
		defer portaudio.Terminate()

		in := make([]float32, cfg.BlockSize)

		var stream *portaudio.Stream
		var err error
		if cfg.DeviceID != -1 || cfg.PrintDevices {
			devices, err := portaudio.Devices()
			if err != nil {
				errc <- err
				return
			}
			if cfg.PrintDevices {
				for _, dev := range devices {
					fmt.Println("Device:", dev)
				}
				return
			}
			params := portaudio.StreamDeviceParameters{
				Device:   devices[cfg.DeviceID],
				Channels: 1,
				Latency:  devices[cfg.DeviceID].DefaultHighInputLatency,
			}
			if cfg.LowLatency {
				params.Latency = devices[cfg.DeviceID].DefaultLowInputLatency
			}
			stream, err = portaudio.OpenStream(portaudio.StreamParameters{
				Input:           params,
				SampleRate:      cfg.SampleRate,
				FramesPerBuffer: cfg.BlockSize,
			}, in)

		} else {
			stream, err = portaudio.OpenDefaultStream(
				cfg.Channels, 0, cfg.SampleRate, cfg.BlockSize, in)
			if err != nil {
				errc <- fmt.Errorf("Error opening stream: %v", err)
				return
			}
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
				log.Println("[INFO] [Audio]", err, len(out))
				switch err {
				case portaudio.InputOverflowed:
				default:
					errc <- fmt.Errorf("Error reading from stream: %v", err)
					return
				}
			}

			out <- in
		}
	}()

	return out, errc
}
