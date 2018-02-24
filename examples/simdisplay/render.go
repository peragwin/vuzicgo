package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
)

type renderer struct {
	src     *fs.FrequencySensor
	columns int
	rows    int
	params  *fs.Parameters

	renderCount int
	lastRender  time.Time

	display *image.RGBA
}

func newRenderer(columns int, params *fs.Parameters, src *fs.FrequencySensor) *renderer {
	display := image.NewRGBA(image.Rect(0, 0, columns, src.Buckets))
	return &renderer{
		params:  params,
		columns: columns,
		rows:    src.Buckets,
		src:     src,
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
		m := map[string]interface{}{
			"fps": diff / 100.0,
			"amp": r.src.Amplitude[0],
			"pha": r.src.Energy,
		}
		bs, _ := json.Marshal(m)
		fmt.Println(string(bs))
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
	if r.params.Mode == fs.AnimateMode {
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

func getHSV(params *fs.Parameters, amp, ph, phi float64) color.RGBA {
	br := params.Brightness
	gbr := params.GlobalBrightness

	hue := math.Mod((ph+phi)*180/math.Pi, 360)
	if hue < 0 {
		hue += 360
	}
	sat := fs.Sigmoid(br * amp)
	val := fs.Sigmoid(gbr / 255 * (1 + amp))

	r, g, b := colorful.Hsv(hue, sat, val).RGB255()
	return color.RGBA{r, g, b, 255}
}

func getRGB(params *fs.Parameters, amp, ph, phi float64) color.RGBA {
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
