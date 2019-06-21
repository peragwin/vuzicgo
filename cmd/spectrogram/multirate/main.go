package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/audio/fft"
	"github.com/peragwin/vuzicgo/audio/util"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	minFrameSize = 256
	maxFrameSize = 4096
	sampleRate   = 44100

	width  = 1200
	height = 800

	rows = 512

	textureMode = gl.LINEAR
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})

	source, errc := audio.NewSource(ctx, &audio.Config{
		BlockSize:  minFrameSize,
		SampleRate: sampleRate,
		Channels:   1,
	})

	// watch for errors
	go func() {
		defer close(done)
		err := <-errc
		log.Fatal(err)
	}()

	//lock := new(sync.Mutex)

	source64 := audio.Buffer(done, source, maxFrameSize)

	// source64 := make(chan []float64)
	// // convert intput to float64
	// go func() {
	// 	defer close(done)
	// 	for {
	// 		select {
	// 		case <-done:
	// 			return
	// 		default:
	// 		}
	// 		x := <-source
	// 		y := make([]float64, len(x))
	// 		for i := range x {
	// 			y[i] = float64(x[i])
	// 		}
	// 		source64 <- y
	// 	}
	// }()

	fftProc := fft.NewFFTProcessor(sampleRate, maxFrameSize)
	fftOut := fftProc.Process(done, source64)
	// fftProc, err := fft.NewMultiRateFFT(minFrameSize, maxFrameSize, true)
	// if err != nil {
	// 	panic(err)
	// }
	// fftOuts := fftProc.Process(done, source64)
	// fftOut := fftOuts[0]

	specProc := new(fft.PowerSpectrumProcessor)
	specOut := specProc.Process(done, fftOut)
	//specOut := fft.SpectrumProcessor(done, source, frameSize)

	fmt.Println("@@@@ SET UP")

	s := <-specOut
	l := len(s)

	frames := make([][]float64, rows)
	outframes := make([][]float64, rows)
	for i := range outframes {
		frames[i] = make([]float64, l)
	}
	frameIndex := 0

	alpha := 1.0

	fmt.Println("@@@@ SET UP 2")

	// goroutine responsible for writing to frames
	go func() {
		i := 0
		defer func() { r := recover(); log.Fatal("frmaes", r, i+frameIndex) }()

		var frame []float64
		for {
			select {
			case <-done:
				return
			default:
			}
			//lock.Lock()

			frame = <-specOut
			copy(frames[frameIndex], frame)

			max := -10000.0
			for i := range frames {
				for j := range frames[i] {
					v := frames[i][j]
					if v > max {
						max = v
					}
				}
			}

			alpha = 0.99*alpha + 0.01*max

			frameIndex = (frameIndex + 1) % rows
			//lock.Unlock()
		}
	}()

	indexInFrames := func(i int) int {
		if i < rows-frameIndex {
			return i + frameIndex
		}
		return i - rows + frameIndex
	}
	// rotateIndex := func(i int) int {
	// 	if i < frameSize/2 {
	// 		return i + frameSize/2
	// 	}
	// 	return i - frameSize/2
	// }
	colorMap := util.NewColorMap(256)

	g, err := grid.NewGrid(done, &grid.Config{
		Rows: l, Columns: rows,
		Width: width, Height: height,
		Title:       "Spectrogram Display",
		TextureMode: textureMode,
		Render: func(g *grid.Grid) {
			// lock.Lock()
			// defer lock.Unlock()
			// defer func() {
			// 	r := recover()
			// 	if r != nil {
			// 		log.Fatal("render", r)
			// 	}
			// }()
			for i := range frames {
				for j := range frames[i] {
					ifr := indexInFrames(i)
					jr := j //rotateIndex(j)
					if ifr >= len(frames) || ifr < 0 {
						log.Fatal("ifr bad index", i, frameIndex, ifr)
					}
					if jr >= len(frames[i]) || jr < 0 {
						log.Fatal("jr bad index", j, jr)
					}
					scaled := frames[ifr][jr] * 4.0 / alpha
					s := uint8(255 * scaled)
					if s > 255 {
						s = 255
					}
					if s < 0 {
						s = 0
					}
					c := colorMap[s]
					// r, _g, b := colorMap.GetInterpolatedColorFor(scaled).RGB255()
					// c := color.RGBA{r, _g, b, 255}
					g.SetColor(i, l-1-j, c)
				}
			}
		},
	})
	if err != nil {
		log.Fatal("error creating display:", err)
	}
	g.Start()

	<-done
}
