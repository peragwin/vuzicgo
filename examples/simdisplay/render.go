package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
)

type renderer struct {
	src     *FrequencySensor
	columns int
	rows    int
	params  *Parameters

	renderCount int
	lastRender  time.Time

	display *image.RGBA
}

func newRenderer(columns int, params *Parameters, fs *FrequencySensor) *renderer {
	display := image.NewRGBA(image.Rect(0, 0, columns, fs.Buckets))
	return &renderer{
		params:  params,
		columns: columns,
		rows:    fs.Buckets,
		src:     fs,
		display: display,
	}
}

func (r *renderer) Render(done, request chan struct{}) chan *image.RGBA {
	out := make(chan *image.RGBA)

	// set up a goroutine to render a frame only when requested
	go func() {
		for {
			select {
			case <-done:
				return
			case <-request:
				r.render()
				out <- r.display
			}
		}
	}()

	return out
}

func (r *renderer) render() {
	r.renderCount++
	if r.params.Debug && r.renderCount%100 == 0 {
		diff := time.Now().Sub(r.lastRender)
		fmt.Println("fps:", diff/100.0)
		fmt.Println("amp:", r.src.Amplitude[0])
		fmt.Println("pha:", r.src.Energy)
		r.lastRender = time.Now()
	}
	hl := r.columns / 2
	for i := 0; i < hl; i++ {
		col := r.renderColumn(i)
		for j, c := range col {
			r.display.SetRGBA(hl+i, r.rows-j-1, c)
			r.display.SetRGBA(hl-1-i, r.rows-j-1, c)
		}
	}
}

func (r *renderer) renderColumn(col int) []color.RGBA {

	amp := r.src.Amplitude[0]
	if r.params.Mode == AnimateMode {
		amp = r.src.Amplitude[col]
	}
	phase := r.src.Energy
	ws := 2.0 * math.Pi / float64(r.params.Period)
	phi := ws * float64(col)

	colors := make([]color.RGBA, r.rows)

	for i, ph := range phase {
		//colors[i] = getRGB(d.params, amp[i], ph, phi)
		colors[i] = getHSV(r.params, amp[i], ph, phi)
	}

	return colors
}

func getHSV(params *Parameters, amp, ph, phi float64) color.RGBA {
	br := params.Brightness
	gbr := params.GlobalBrightness

	hue := math.Mod((ph+phi)*180/math.Pi, 360)
	if hue < 0 {
		hue += 360
	}
	sat := sigmoid(br * amp)
	val := sigmoid(gbr / 255 * (1 + amp))

	r, g, b := colorful.Hsv(hue, sat, val).RGB255()
	return color.RGBA{r, g, b, 255}
}

func getRGB(params *Parameters, amp, ph, phi float64) color.RGBA {
	br := params.Brightness
	gbr := params.GlobalBrightness

	r := math.Sin(ph + phi)
	g := math.Sin(ph + phi + 2*math.Pi/3)
	b := math.Sin(ph + phi - 2*math.Pi/3)

	// TODO print norm and see if it's contant to optimize
	norm := math.Abs(r) + math.Abs(g) + math.Abs(b)
	r /= norm
	g /= norm
	b /= norm

	r = gbr * (1 + br*amp*r)
	g = gbr * (1 + br*amp*g)
	b = gbr * (1 + br*amp*b)
	// WAS b = gbr / br * (br + amp*b)

	r = math.Max(0, math.Min(255, r))
	g = math.Max(0, math.Min(255, g))
	b = math.Max(0, math.Min(255, b))

	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}
