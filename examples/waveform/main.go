// Waveform opens an input stream and displays the output as a simple waveform
// like an oscilliscope.

package main

import (
	"context"
	"log"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	frameSize  = 128
	sampleRate = 44100

	width  = 640
	height = 480

	rows = 100

	scale = 32
)

func main() {
	// runtime.LockOSThread()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
				for i, val := range in {
					v := int(rows * (1.0 + scale*val) / 2.0)
					if v >= rows {
						v = rows - 1
					} else if v < 0 {
						v = 0
					}
					buffer[bufferIdx][i] = v
				}
				bufferIdx ^= 1
				lock.Unlock()
			case err := <-errc:
				log.Println("steam error:", err)
				return
			}
		}
	}()

	display, err := grid.NewGrid(ctx, &grid.Config{
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
				g.SetColor(i, val, mgl32.Vec4{0.0, 1.0, 0.0, 0.5})
			}
			for i, val := range buffer[recent] {
				g.SetColor(i, val, mgl32.Vec4{0.0, 1.0, 0.0, 1.0})
			}
		},
	})
	if err != nil {
		log.Fatal("error creating display:", err)
	}

	<-display.Done
}
