// Waveform opens an input stream and displays the output as a simple waveform
// like an oscilliscope.

package main

import (
	"context"
	"log"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/audio/fft"
	"github.com/peragwin/vuzicgo/audio/util"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	frameSize  = 4096
	sampleRate = 44100

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

	//lock := new(sync.Mutex)

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
				for j := frameSize - 512; j >= 0; j -= 512 {
					source64 <- y[frameSize-j : 2*frameSize-j]
				}
				// source64 <- y[offset/2 : frameSize-offset/2]
				// source64 <- y[offset:]
			} else {
				for j := frameSize - 512; j >= 0; j -= 512 {
					source64 <- append(y[2*frameSize-1-j:], y[:frameSize-j]...)
				}
				// source64 <- append(y[frameSize-frameSize/2:], y[:frameSize/2]...)
				// source64 <- y[:frameSize]
			}
		}
	}()

	fftProc := fft.NewFFTProcessor(sampleRate, frameSize)
	fftOut := fftProc.Process(done, source64)

	specProc := new(fft.PowerSpectrumProcessor)
	specOut := specProc.Process(done, fftOut)
	//specOut := fft.SpectrumProcessor(done, source, frameSize)

	frames := make([][]float64, rows)
	outframes := make([][]float64, rows)
	for i := range outframes {
		frames[i] = make([]float64, 256) //frameSize/2)
	}
	frameIndex := 0

	alpha := 1.0

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
			copy(frames[frameIndex], frame[:256])
			//fmt.Println("frame", frameIndex, frames[frameIndex][0])

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
		Rows:/*frameSize / 2*/ 256, Columns: rows,
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
					g.SetColor(i, 256 /*frameSize/2*/ -1-j, c)
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
