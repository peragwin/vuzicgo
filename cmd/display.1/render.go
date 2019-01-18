package main

import (
	"image"
	"fmt"
	//"time"
	"math"
	"image/color"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/phrozen/blend"

	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
)

type params struct {
	width int
	height int
	depth int
}

type renderer struct {
	params *params
	src *fs.FrequencySensor
	grid [][][]color.RGBA
	display *image.RGBA
	projection mgl32.Mat4

	renderCount int
}

func newRenderer(p *params, src *fs.FrequencySensor) *renderer {
	grid := make([][][]color.RGBA, p.depth)
	for k := range grid {
		grid[k] = make([][]color.RGBA, p.height)
		for j := range grid[k] {
			grid[k][j] = make([]color.RGBA, p.width)
		}
	}
	display := image.NewRGBA(image.Rect(0,0,p.width, p.height))
	//asp := float32(p.width)/float32(p.height)
	//projection := mgl32.Perspective(mgl32.DegToRad(90), 1, 0.1, 10)
	projection := newPerspective(90, 1, 2)
	fmt.Println(projection)
	return &renderer{
		params: p,
		grid: grid,
		display: display,
		projection: projection,
	}
}

func newPerspective(fov, near, far float32) mgl32.Mat4 {
	s := float32(1.0 / math.Tan(float64(fov * math.Pi / 360)))
	return mgl32.Mat4{
		s, 0, 0, 0,
		0, s, 0, 0,
		0, 0, -(far / (far-near)), -1,
		0, 0, -(far*near / (far-near)), 0,
	}
}

type renderValues struct {
	img *image.RGBA
}

func (r *renderer) Render(done, request chan struct{}) chan *renderValues {
	out := make(chan *renderValues)

	//ticker := time.NewTicker(200 * time.Millisecond)

	// set up a goroutine to render a frame only when requested
	go func() {
		for {
			//<-ticker.C
			select {
			case <-done:
				return
			case <-request:
				r.render()
				out <- &renderValues{
					r.display,
				}
			}
		}
	}()

	return out
}

func drawRect(col color.RGBA, w, h int) [][]color.RGBA {
	frame := make([][]color.RGBA, h)
	for j := range frame {
		frame[j] = make([]color.RGBA, w)
		if j == 0 || j == h - 1 {
			for i := range frame[j] {
				frame[j][i] = col
			}
		} else {
			frame[j][0] = col
			frame[j][w-1] = col
		}
	}
	return frame
}

func (r *renderer) makeVec(i, j, k int) mgl32.Vec4 {
	x := float32(i) / float32(r.params.width - 1)
	x = 2*x - 1
	y := float32(j) / float32(r.params.height - 1)
	y = 2*y - 1
	z := -(float32(k) + 1) /// float32(r.params.depth - 1)
	// z = 2*z - 1
	return mgl32.Vec4{x, y, z, 1.0}
}

func (r *renderer) fromVec(v mgl32.Vec4) (i, j, k int) {
	x, y, z, w := v.Elem()
	i = int( (x/w+1) / 2 * float32(r.params.width - 1) + 0.5 )
	j = int( (y/w+1) / 2 * float32(r.params.height - 1) + 0.5 )
	k = int( z-1 + 0.5 )
	return
}

func max(a, b uint8) uint8 {
	if a > b {
		return a
	} else {
		return b
	}
}

func (r *renderer) render() {
	var col color.RGBA
	if r.renderCount % r.params.depth == 0 {
		col = color.RGBA{255, 0, 0, 4}
	}

	for i := r.params.depth - 1; i > 0; i-- {
		r.grid[i] = r.grid[i-1]
	}
	r.grid[0] = drawRect(col-, r.params.width, r.params.height)

	imgs := make([]*image.RGBA, len(r.grid))
	for k := range r.grid {
		img := image.NewRGBA(image.Rect(0,0,r.params.width, r.params.height))
		for j := range r.grid[k] {
			for i, col := range r.grid[k][j] {
				vec := r.makeVec(i, j, k)
				proj := r.projection.Mul4x1(vec)
				x, y, _ := r.fromVec(proj)
				if col.R != 0 {
					//fmt.Printf("%2d, %2d, %2d -> %2d, %2d, %2d\n", i, j, k, x, y, z)
					img.SetRGBA(x, y, col)
				}
				//}
			}
		}
		imgs[k] = img
	}
	fmt.Println("render")

	display := image.NewRGBA(image.Rect(0,0,r.params.width, r.params.height))
	for _, im := range imgs {
		blend.BlendImage(display, im, blend.Screen)
	}
	r.display = display

	r.renderCount++
}

