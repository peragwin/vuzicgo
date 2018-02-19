package main

import (
	"context"
	"log"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/audio/fft"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	frameSize  = 512
	sampleRate = 44100

	width  = 1200
	height = 800

	buckets = 16
	columns = 60

	textureMode = gl.LINEAR
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})

	source, errc := audio.NewSource(ctx, &audio.Config{
		BlockSize:  frameSize,
		SampleRate: sampleRate,
		Channels:   1,
	})

	// watch for errors
	go func() {
		defer close(done)
		err := <-errc
		log.Fatal(err)
	}()

	source64 := make(chan []float64)
	// convert intput to float64
	// overlap frames by 50%
	go func() {
		defer close(done)
		y := make([]float64, frameSize*2)
		bufferIndex := 0
		for {
			select {
			case <-done:
				return
			default:
			}
			x := <-source

			offset := bufferIndex * frameSize
			for i := range x {
				y[i+offset] = float64(x[i])
			}
			if bufferIndex == 1 {
				z := y[frameSize/2 : 2*frameSize-frameSize/2]
				source64 <- z
				source64 <- y[frameSize:]
			} else {
				z := append(y[2*frameSize-frameSize/2:], y[:frameSize/2]...)
				source64 <- z
				source64 <- y[:frameSize]
			}

			bufferIndex ^= 1
		}
	}()

	fftProc := fft.NewFFTProcessor(sampleRate, frameSize)
	fftOut := fftProc.Process(done, source64)

	specProc := new(fft.PowerSpectrumProcessor)
	specOut := specProc.Process(done, fftOut)

	render := make(chan struct{})
	defer close(render)
	display := NewDisplay(&Config{
		Columns:    columns,
		Buckets:    buckets,
		SampleRate: sampleRate,
		Parameters: defaultParameters,
	})
	frames := display.Process(done, specOut, render)

	_, err := grid.NewGrid(done, &grid.Config{
		Rows: buckets, Columns: columns,
		Width: width, Height: height,
		Title:       "Sim LED Display",
		TextureMode: textureMode,
		Render: func(g *grid.Grid) {
			render <- struct{}{}
			img := <-frames
			g.SetImage(img)
		},
	})
	if err != nil {
		log.Fatal("error creating display:", err)
	}

	<-done
}
