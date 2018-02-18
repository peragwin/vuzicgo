// Waveform opens an input stream and displays the output as a simple waveform
// like an oscilliscope.

package main

import (
	"context"
	"image/color"
	"log"
	"sync"

	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/audio/fft"
	"github.com/peragwin/vuzicgo/audio/util"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	frameSize  = 512
	sampleRate = 44100

	width  = 640
	height = 480

	rows = 256
)

var scale float32 = 1.0

func main() {
	// runtime.LockOSThread()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})

	source, errc := audio.NewSource(ctx, &audio.Config{
		BlockSize:  frameSize,
		SampleRate: sampleRate,
		Channels:   1,
	})

	// since frameSize / sampleRate ~= 12ms, and we expect 60fps ~= 17ms
	// we'll buffer the two most recent frames to display them together
	//buffer := [2][frameSize]int{}
	//bufferIdx := 0
	lock := new(sync.Mutex)

	source64 := make(chan []float64)
	// convert intput to float64
	go func() {
		defer close(done)
		for {
			select {
			case <-done:
				return
			case err := <-errc:
				log.Fatal(err)
			default:
			}
			x := <-source
			y := make([]float64, len(x))
			for i := range x {
				y[i] = float64(x[i])
			}
			source64 <- y
		}
	}()

	fftProc := fft.NewFFTProcessor(sampleRate, frameSize)
	fftOut := fftProc.Process(done, source64)

	specProc := new(fft.PowerSpectrumProcessor)
	specOut := specProc.Process(done, fftOut)

	frames := make([][]float64, rows)
	outframes := make([][]float64, rows)
	for i := range outframes {
		frames[i] = make([]float64, frameSize)
		outframes[i] = make([]float64, frameSize)
	}
	frameIndex := 0

	//img := image.NewRGBA(image.Rect(0, 0, frameSize, rows))
	colorMap := util.NewColorMap()
	alpha := 1.0

	// goroutine responsible for writing to frames
	go func() {
		i := 0
		defer func() { r := recover(); log.Fatal("frmaes", r, i+frameIndex) }()

		for {
			select {
			case <-done:
				return
			default:
			}
			frame := <-specOut
			frames[frameIndex] = frame

			// // shift image forward by one frame
			// for i := 4*frameSize*rows; i > 4*frameSize; i-- {
			// 	img.Pix[i] = img.Pix[i-4*frameSize]
			// }

			//fmt.Println(frame[0], alpha)

			max := -10000.0
			for i := range frames {
				for j := range frames[i] {
					v := frames[i][j]
					if v > max {
						max = v
					}
				}
			}

			alpha = 0.95*alpha + 0.05*max

			for i = frameIndex; i < len(frames); i++ {
				//fmt.Println(i - frameIndex)
				outframes[rows-i] = frames[i]
			}
			for i = 0; i < rows-frameIndex; i++ {
				//fmt.Println(i + frameIndex)
				outframes[i+frameIndex] = frames[i]
			}

			frameIndex = (frameIndex + 1) % rows
		}
	}()

	_, err := grid.NewGrid(done, &grid.Config{
		Rows: rows, Columns: frameSize,
		Width: width, Height: height,
		Title: "Waveform Display",
		Render: func(g *grid.Grid) {
			lock.Lock()
			defer lock.Unlock()
			defer func() {
				r := recover()
				if r != nil {
					log.Fatal("render", r)
				}
			}()

			for i := range outframes {
				for j := range outframes[i] {
					scaled := outframes[i][j] * 1.0 / alpha
					r, _g, b := colorMap.GetInterpolatedColorFor(1 - scaled).RGB255()
					c := color.RGBA{r, _g, b, 255}
					g.SetColor(j, i, c)
				}
			}
		},
	})
	if err != nil {
		log.Fatal("error creating display:", err)
	}

	<-done
}
