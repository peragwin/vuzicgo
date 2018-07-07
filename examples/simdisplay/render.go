package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
	flaschen "github.com/peragwin/vuzicgo/gfx/flaschen-taschen/api/go"
	"github.com/peragwin/vuzicgo/gfx/skgrid"
)

type renderer struct {
	src     *fs.FrequencySensor
	columns int
	rows    int
	params  *fs.Parameters

	renderCount int
	lastRender  time.Time

	display *image.RGBA
	warp    []float32
	scale   float32
}

func newRenderer(columns int, params *fs.Parameters, src *fs.FrequencySensor) *renderer {
	display := image.NewRGBA(image.Rect(0, 0, columns, src.Buckets))
	return &renderer{
		params:  params,
		columns: columns,
		rows:    src.Buckets,
		src:     src,
		display: display,
		warp:    make([]float32, src.Buckets),
	}
}

type renderValues struct {
	img   *image.RGBA
	warp  []float32
	scale float32
}

func (r *renderer) Render(done, request chan struct{}) chan *renderValues {
	out := make(chan *renderValues)

	// set up a goroutine to render a frame only when requested
	go func() {
		for {
			select {
			case <-done:
				return
			case <-request:
				r.render()
				out <- &renderValues{
					r.display,
					r.warp,
					r.scale,
				}
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
			"fps":  diff / 100.0,
			"amp":  r.src.Amplitude[0],
			"pha":  r.src.Energy,
			"diff": r.src.Diff,
		}
		bs, err := json.Marshal(m) //Indent(m, "", "  ")
		if err != nil {
			fmt.Printf("%#v", r.src.Drivers)
			panic(err)
		}
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
	for i, d := range r.src.Diff {
		r.warp[i] = float32(r.params.WarpOffset + r.params.WarpScale*math.Abs(d))
	}
	for i := 1; i < len(r.src.Diff)-1; i++ {
		wl := r.warp[i-1]
		wr := r.warp[i+1]
		w := r.warp[i]
		// dl := w - wl
		// sl := float32(1.0)
		// if dl < 0 {sl = -1}
		// dr :=  w - wr
		// sr := float32(1.0)
		// if dr < 0 {sr = -1}
		// r.warp[i] += .01 * (sl * wl * wl + sr * wr * wr)
		r.warp[i] = (wl + wr + w) / 3
	}
	r.scale = float32(1 + r.params.Scale*r.src.Bass)
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
	sat := fs.Sigmoid(br - 2 + amp)
	val := fs.Sigmoid(gbr/255*(1+amp) - 4)
	al := fs.Sigmoid(.25*amp - 4)

	r, g, b := colorful.Hsv(hue, sat, val).RGB255()
	return color.RGBA{r, g, b, uint8(256 * al)}
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

func (r *renderer) skgridRender(skRem skgrid.Driver, done chan struct{}) {
	defer skRem.Close()
	defer close(done)

	skGrid := skgrid.NewGrid(60, 16, skRem)

	render := make(chan struct{})
	frames := r.Render(done, render)

	xinput := make([]float64, skGrid.Height/2)
	for i := range xinput {
		xinput[i] = 1 - 2*float64(i)/float64(skGrid.Height)
	}
	warpIndices := func(warp float64) []int {
		ws := fs.DefaultParameters.WarpScale
		wo := fs.DefaultParameters.WarpOffset
		warp = wo + ws*math.Abs(warp)
		wv := make([]int, skGrid.Height)
		b := skGrid.Height / 2
		for i := 0; i < skGrid.Height/2; i++ {
			scaled := 1 - math.Pow(xinput[i], warp)
			xp := scaled * float64(skGrid.Height/2)
			wv[b-i] = b - int(xp+0.5)
			wv[b+i] = b + int(xp+0.5)
		}
		return wv
	}
	scaleIndex := func(y int, scale float64) int {
		yi := 1 - float64(y)/float64(skGrid.Width)
		warp := fs.DefaultParameters.Scale * scale
		scaled := 1 - math.Pow(yi, warp)
		return int((float64(skGrid.Width) * scaled) + .5)
	}

	ticker := time.NewTicker(20000 * time.Microsecond)

	for {
		<-ticker.C
		render <- struct{}{}
		frame := <-frames
		if frame == nil {
			break
		}
		img := resize.Resize(uint(skGrid.Height), uint(skGrid.Width),
			frame.img, resize.NearestNeighbor)

		wvs := make([][]int, skGrid.Width)
		yv := make([]int, skGrid.Width)
		for i := range wvs {
			wvs[i] = warpIndices(float64(frame.warp[i]))
			yv[i] = scaleIndex(i, float64(frame.scale))
		}

		for y := 0; y < 60; y++ {
			for x := 0; x < 16; x++ {
				px := img.At(x, skGrid.Width-1-y).(color.RGBA)
				px.G /= 2
				px.B /= 2
				px.A = uint8(float64(px.A)/8 + 0.5)

				xstart := 0
				if x != 0 {
					xstart = wvs[y][x]
				}
				xend := skGrid.Height
				if x != skGrid.Height-1 {
					//log.Println(y, x, wvs[y])
					xend = wvs[y][x+1]
				}

				ystart := yv[y]
				yend := skGrid.Width
				if y != skGrid.Width-1 {
					yend = yv[y+1]
				}

				if xend > xstart && yend > ystart {
					for j := ystart; j < yend; j++ {
						for k := xstart; k < xend; k++ {
							skGrid.Pixel(j, k, px)
						}
					}
				}
			}
		}

		if err := skGrid.Show(); err != nil {
			log.Println("sk grid error!", err)
			break
		}
	}
}

func (r *renderer) flaschenRender(fl *flaschen.Flaschen, done chan struct{}) {
	defer fl.Close()
	defer close(done)

	rect := fl.Rect()
	width := rect.Dx()
	height := rect.Dy()

	render := make(chan struct{})
	frames := r.Render(done, render)

	mod := width % 2
	xinput := make([]float64, width/2+mod)
	for i := range xinput {
		xinput[i] = 1 - 2*float64(i)/float64(width)
	}
	warpIndices := func(warp float64) []int {
		ws := fs.DefaultParameters.WarpScale
		wo := fs.DefaultParameters.WarpOffset
		warp = wo + ws*math.Abs(warp)
		wv := make([]int, width)
		b := width / 2
		for i := 0; i < width/2+mod; i++ {
			scaled := 1 - math.Pow(xinput[i], warp)
			xp := scaled * float64(width) / 2
			wv[b-i] = b - int(xp+0.5)
			wv[b+i] = b + int(xp+0.5)
		}
		return wv
	}
	scaleIndex := func(y int, scale float64) int {
		yi := 1 - float64(y)/float64(height)
		warp := 1.0 //fs.DefaultParameters.Scale * scale
		scaled := 1 - math.Pow(yi, warp)
		return int((float64(height) * scaled) + .5)
	}

	ticker := time.NewTicker(20000 * time.Microsecond)

	for {
		<-ticker.C
		render <- struct{}{}
		frame := <-frames
		if frame == nil {
			break
		}
		img := resize.Resize(uint(width), uint(height),
			frame.img, resize.NearestNeighbor)

		wvs := make([][]int, height)
		yv := make([]int, height)
		for i := range wvs {
			wvs[i] = warpIndices(float64(frame.warp[i]))
			yv[i] = scaleIndex(i, float64(frame.scale))
		}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				px := img.At(x, y).(color.RGBA)

				xstart := 0
				if x != 0 {
					xstart = wvs[y][x]
				}
				xend := width
				if x != width-1 {
					// log.Println(y, x, wvs[y])
					xend = wvs[y][x+1]
				}

				ystart := yv[y]
				yend := height
				if y != height-1 {
					yend = yv[y+1]
				}

				if xend > xstart && yend > ystart {
					// fmt.Println(xstart, xend, ystart, yend)
					for j := ystart; j < yend; j++ {
						for k := xstart; k < xend; k++ {
							fl.Pixel(k, height-1-j, px)
						}
					}
				}
			}
		}

		if err := fl.Show(); err != nil {
			log.Println("flaschen grid error!", err)
			break
		}
	}
}
