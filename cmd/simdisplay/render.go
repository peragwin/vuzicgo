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
	"github.com/peragwin/vuzicgo/gfx/skgrid"
)

type renderer struct {
	src     *fs.FrequencySensor
	columns int
	rows    int
	params  *fs.Parameters
	mirror  bool

	renderCount int
	lastRender  time.Time

	display *image.RGBA
	warp    []float32
	scale   []float32
	bass    float32
}

func newRenderer(columns int, mirror bool, params *fs.Parameters,
	src *fs.FrequencySensor) *renderer {
	rows := src.Buckets
	if mirror {
		rows *= 2
	}
	display := image.NewRGBA(image.Rect(0, 0, columns, rows))
	return &renderer{
		params:  params,
		columns: columns,
		rows:    src.Buckets,
		src:     src,
		display: display,
		warp:    make([]float32, src.Buckets),
		scale:   make([]float32, columns),
		mirror:  mirror,
	}
}

type renderValues struct {
	img   *image.RGBA
	warp  []float32
	scale []float32
	bass  float32
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
					r.bass,
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
			if r.mirror {
				r.display.SetRGBA(hl+i, r.rows+j, c)
				r.display.SetRGBA(hl+i, r.rows-1-j, c)
				r.display.SetRGBA(hl-1-i, r.rows+j, c)
				r.display.SetRGBA(hl-1-i, r.rows-1-j, c)
			} else {
				r.display.SetRGBA(hl+i, r.rows-j-1, c)
				r.display.SetRGBA(hl-1-i, r.rows-j-1, c)
			}
		}
	}
	ln := len(r.src.Diff)
	// if r.mirror {
	// 	ln *= 2
	// }
	for i := 0; i < ln; i++ {
		var ix int
		if i < len(r.src.Diff) {
			ix = i
		} else {
			ix = ln - 1 - i
		}
		d := r.src.Diff[ix]
		r.warp[i] = float32(d) //float32(r.params.WarpOffset + r.params.WarpScale*math.Abs(d))
	}
	for i := 1; i < ln-1; i++ {
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

	r.bass = float32(r.src.Bass)

	for i := 0; i < r.columns; i++ {
		var s float64
		for _, v := range r.src.Amplitude[i] {
			// todo: sum with linear scaling to target bass?
			s += v
		}
		r.scale[i] = float32(s)
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
	so := params.SaturationOffset
	vo1 := params.ValueOffset1
	vo2 := params.ValueOffset2
	alpha := params.Alpha
	ao := params.AlphaOffset

	hue := math.Mod((ph+phi)*180/math.Pi, 360)
	if hue < 0 {
		hue += 360
	}
	sat := fs.Sigmoid(br + so + amp)
	val := fs.Sigmoid(gbr/255*(vo1+amp) + vo2)
	al := fs.Sigmoid(alpha*amp + ao)

	r, g, b := colorful.Hsv(hue, sat, val).RGB255()
	return color.RGBA{r, g, b, uint8(255 * al)}
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

func (r *renderer) gridRender(g skgrid.Grid, frameRate int, done chan struct{}) {
	defer g.Close()
	defer close(done)

	rect := g.Rect()
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
		if r.mirror {
			wv2 := make([]int, width)
			for i, v := range wv {
				wv2[width-1-i] = v
			}
			wv = append(wv, wv2...)
		}
		return wv
	}
	scaleIndex := func(y int, scale float64) int {
		yi := 1 - float64(y)/float64(height) // was 1 - float64(y) / ..
		warp := 1 + fs.DefaultParameters.Scale*scale
		offset := fs.DefaultParameters.ScaleOffset
		scaled := 1 - math.Pow(yi, warp)
		return int((float64(height) * scaled) + offset)
	}

	delay := time.Second / time.Duration(frameRate)
	ticker := time.NewTicker(delay)

	for {
		<-ticker.C
		//fmt.Println("render sk frame")
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
			yv[i] = scaleIndex(i, float64(frame.bass))
		}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				px := img.At(x, height-1-y).(color.RGBA)

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
							g.Pixel(k, j, px)
						}
					}
				}
			}
		}

		if err := g.Show(); err != nil {
			log.Println("grid error!", err)
			break
		}
	}
}

func calculateWarpIndices(input []float64, scale, offset, intensity float64) []int {
	warp := offset + scale*math.Abs(intensity)

	mod := len(input) % 2
	b := len(input) / 2
	size := b + mod
	bf := float64(len(input)) / 2

	out := make([]int, len(input))

	for i := 0; i < size; i++ {
		s := 1 - math.Pow(input[i], warp)
		p := s * bf
		out[b-i] = b - int(p+0.5)
		out[b+i] = b + int(p+0.5)
	}

	return out
}

func (r *renderer) gridRender2(g skgrid.Grid, frameRate int, done chan struct{}) {

	defer g.Close()
	defer close(done)

	rect := g.Rect()
	width := rect.Dx()
	height := rect.Dy()

	render := make(chan struct{})
	frames := r.Render(done, render)

	// mod := width % 2
	xinput := make([]float64, width) ///2+mod)
	for i := range xinput {
		xinput[i] = 1 - 2*float64(i)/float64(width)
	}
	// mod = height % 2
	yinput := make([]float64, height) ///2+mod)
	for i := range yinput {
		yinput[i] = 1 - 2*float64(i)/float64(height)
	}

	delay := time.Second / time.Duration(frameRate)
	ticker := time.NewTicker(delay)

	for {
		<-ticker.C
		render <- struct{}{}
		frame := <-frames
		if frame == nil {
			break
		}

		xvs := make([][]int, height)
		h := height/2 + (height % 2)
		sc := float64(len(frame.warp)) / float64(height/2)
		for i := 0; i < height/2; i++ {
			wstart := int(sc*float64(i) + 0.5)
			wend := int(sc*float64(i+1) + 0.5)
			var s float32
			for j := wstart; j < wend; j++ {
				s += frame.warp[j]
			}
			s /= float32(wend - wstart)
			wix := calculateWarpIndices(xinput,
				fs.DefaultParameters.WarpScale,
				fs.DefaultParameters.WarpOffset,
				float64(s),
			)
			xvs[h+i] = wix
			xvs[h-i-1] = wix
		}

		yvs := make([][]int, width)
		h = width/2 + (width % 2)
		b := float64(width) / 2
		sc = float64(len(frame.scale)) / 2 / b
		for i := 0; i < width/2; i++ {
			sstart := int(sc*float64(i) + 0.5)
			send := int(sc*float64(i+1) + 0.5)
			var s float32
			for j := sstart; j < send; j++ {
				s += frame.scale[j]
			}
			s /= float32(send - sstart)

			scale := 1 - math.Abs(b-float64(i)/2)/b
			// fmt.Println("@@@ scale", sstart, send, h+i, h-i-1, s)
			wix := calculateWarpIndices(yinput,
				scale*fs.DefaultParameters.Scale,
				fs.DefaultParameters.ScaleOffset,
				float64(s),
			)
			yvs[h+i] = wix
			yvs[h-i-1] = wix
		}

		img := resize.Resize(uint(width), uint(height),
			frame.img, resize.NearestNeighbor)

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				px := img.At(x, y).(color.RGBA)

				xstart := 0
				if x != 0 {
					xstart = xvs[y][x]
				}
				xend := width
				if x != width-1 {
					// log.Println(y, x, wvs[y])
					xend = xvs[y][x+1]
				}

				ystart := 0
				if y != 0 {
					ystart = yvs[x][y]
				}
				yend := height
				if y != height-1 {
					yend = yvs[x][y+1]
				}

				if xend > xstart && yend > ystart {
					for j := ystart; j < yend; j++ {
						for k := xstart; k < xend; k++ {
							g.Pixel(k, j, px)
						}
					}
				}
			}
		}

		if err := g.Show(); err != nil {
			log.Println("grid error!", err)
			break
		}
	}
}
