// Waveform opens an input stream and displays the output as a simple waveform
// like an oscilliscope.

package main

import (
	"context"
	"image/color"
	"log"
	"sync"

	"github.com/peragwin/vuzicgo/audio"
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
	buffer := [2][frameSize]int{}
	bufferIdx := 0
	lock := new(sync.Mutex)

	// goroutine responsible for receiving from the source
	go func() {
		defer cancel()
		for {
			select {
			case in := <-source:
				if in == nil {
					return
				}
				lock.Lock()

				var max float32 = 0.00001
				for i, val := range in {
					if val > max {
						max = val
					}

					v := int(rows * (0.5 + 2.0/rows*scale*val))
					if v >= rows {
						v = rows - 1
					} else if v < 0 {
						v = 0
					}
					buffer[bufferIdx][i] = v
				}
				bufferIdx ^= 1

				// sum = float32(math.Sqrt(float64(sum)))
				// sum /= float32(len(in))
				scale = .95*scale + float32(.5/float64(max))

				lock.Unlock()
			case err := <-errc:
				log.Println("steam error:", err)
				return
			}
		}
	}()

	_, err := grid.NewGrid(done, &grid.Config{
		Rows: rows, Columns: frameSize,
		Width: width, Height: height,
		Title: "Waveform Display",
		Render: func(g *grid.Grid) {
			lock.Lock()
			defer lock.Unlock()

			g.Clear()

			prev := bufferIdx
			recent := bufferIdx ^ 1
			for i, val := range buffer[prev] {
				g.SetColor(i, val, color.RGBA{0, 255, 0, 127})
			}
			for i, val := range buffer[recent] {
				g.SetColor(i, val, color.RGBA{0, 255, 0, 255})
			}
		},
	})
	if err != nil {
		log.Fatal("error creating display:", err)
	}

	<-done
}
