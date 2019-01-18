package main

import (
	"image"
	"image/color"
	"math"
	"time"
)

type renderer struct {
	src     *rgbSrc
	params  *params
	sources []*Oscillator

	renderCount int
	lastRender  time.Time

	display *image.RGBA
}

type params struct {
	rows    int
	columns int
	// buckets int
	// frames  int
}

type XYVal struct {
	X float64
	Y float64
}

type Oscillator struct {
	SFreq  XYVal
	MFreq  XYVal
	Phase  XYVal
	mapper WaveMapper
	m      float64
}

func NewOscillator(sfreq, mfreq, phase XYVal, mapper WaveMapper) *Oscillator {
	return &Oscillator{sfreq, mfreq, phase, mapper, 0}
}

func (o *Oscillator) GetSample(x, y float64) float64 {
	x = o.Phase.X + x*o.SFreq.X + o.m*o.MFreq.X
	y = o.Phase.Y + y*o.SFreq.Y + o.m*o.MFreq.Y
	return o.mapper(x + y)
}

func (o *Oscillator) Increment() {
	o.m++
}

// func (o *Oscillator) GetSamples(n int) []float64 {
// 	r := make([]float64, n)
// 	for i := range r {
// 		r[i] = o.GetSample(0, 0)
// 	}
// 	return r
// }

type WaveMapper func(float64) float64

func SinWave(x float64) float64 {
	return (1 + math.Sin(2*math.Pi*x)) / 2
}

func SquareWave(x float64) float64 {
	_, x = math.Modf(x)
	if x <= 0.5 {
		return 0
	} else {
		return 1
	}
}

func SawWave(x float64) float64 {
	_, x = math.Modf(x)
	return x
}

type rgbSrc struct {
	r *Oscillator
	g *Oscillator
	b *Oscillator
}

func newRenderer(p *params, src *rgbSrc) *renderer {
	display := image.NewRGBA(image.Rect(0, 0, p.columns, p.rows))
	return &renderer{
		params:  p,
		src:     src,
		display: display,
	}
}

type renderValues struct {
	img *image.RGBA
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
				}
			}
		}
	}()

	return out
}

func (r *renderer) render() {


	// log.Println("render")
	
	for i := 0; i < r.params.columns; i++ {
		for j := 0; j < r.params.rows; j++ {
			r.renderCount++
			x := float64(r.params.rows*i) 
			y := float64(j+r.renderCount) / 1.8
			x = x + y
			r_ := uint8(255*r.src.r.GetSample(x, 0) + .5)
			g_ := uint8(255*r.src.g.GetSample(x, 0) + .5)
			b_ := uint8(255*r.src.b.GetSample(x, 0) + .5)
			c := color.RGBA{r_, g_, b_, 255}
			//fmt.Println("set color %d, %d: %v", i, j, c)
			r.display.SetRGBA(i, j, c)
		}
	}
}

// if r.params.Debug && r.renderCount%100 == 0 {
// 	diff := time.Now().Sub(r.lastRender)
// 	m := map[string]interface{}{
// 		"fps":  diff / 100.0,
// 		"amp":  r.src.Amplitude[0],
// 		"pha":  r.src.Energy,
// 		"diff": r.src.Diff,
// 	}
// 	bs, err := json.Marshal(m) //Indent(m, "", "  ")
// 	if err != nil {
// 		fmt.Printf("%#v", r.src.Drivers)
// 		panic(err)
// 	}
// 	fmt.Println(string(bs))
// 	r.lastRender = time.Now()
// }
// hl := r.columns / 2
// for i := 0; i < hl; i++ {
// 	col := r.renderColumn(i)
// 	for j, c := range col {
// 		r.display.SetRGBA(hl+i, r.rows-j-1, c)
// 		r.display.SetRGBA(hl-1-i, r.rows-j-1, c)
// 	}
// }
// for i, d := range r.src.Diff {
// 	r.warp[i] = float32(r.params.WarpOffset + r.params.WarpScale*math.Abs(d))
// }
// for i := 1; i < len(r.src.Diff)-1; i++ {
// 	wl := r.warp[i-1]
// 	wr := r.warp[i+1]
// 	w := r.warp[i]
// 	// dl := w - wl
// 	// sl := float32(1.0)
// 	// if dl < 0 {sl = -1}
// 	// dr :=  w - wr
// 	// sr := float32(1.0)
// 	// if dr < 0 {sr = -1}
// 	// r.warp[i] += .01 * (sl * wl * wl + sr * wr * wr)
// 	r.warp[i] = (wl + wr + w) / 3
// }
// r.scale = float32(1 + r.params.Scale*r.src.Bass)
// }

// func (r *renderer) renderColumn(col int) []color.RGBA {

// 	amp := r.src.Amplitude[0]
// 	if r.params.Mode == fs.AnimateMode {
// 		amp = r.src.Amplitude[col]
// 	}
// 	phase := r.src.Energy
// 	ws := 2.0 * math.Pi / float64(r.params.Period)
// 	phi := ws * float64(col)

// 	colors := make([]color.RGBA, r.rows)

// 	for i, ph := range phase {
// 		//colors[i] = getRGB(d.params, amp[i], ph, phi)
// 		colors[i] = getHSV(r.params, amp[i], ph, phi)
// 	}

// 	return colors
// }

// func getHSV(params *fs.Parameters, amp, ph, phi float64) color.RGBA {
// 	br := params.Brightness
// 	gbr := params.GlobalBrightness

// 	hue := math.Mod((ph+phi)*180/math.Pi, 360)
// 	if hue < 0 {
// 		hue += 360
// 	}
// 	sat := fs.Sigmoid(br - 2 + amp)
// 	val := fs.Sigmoid(gbr/255*(1+amp) - 4)
// 	al := fs.Sigmoid(.25*amp - 4)

// 	r, g, b := colorful.Hsv(hue, sat, val).RGB255()
// 	return color.RGBA{r, g, b, uint8(256 * al)}
// }

// func getRGB(params *fs.Parameters, amp, ph, phi float64) color.RGBA {
// 	br := params.Brightness
// 	gbr := params.GlobalBrightness

// 	r := math.Sin(ph + phi)
// 	g := math.Sin(ph + phi + 2*math.Pi/3)
// 	b := math.Sin(ph + phi - 2*math.Pi/3)

// 	// TODO print norm and see if it's contant to optimize
// 	norm := math.Abs(r) + math.Abs(g) + math.Abs(b)
// 	r /= norm
// 	g /= norm
// 	b /= norm

// 	r = gbr * (1 + br*amp*r)
// 	g = gbr * (1 + br*amp*g)
// 	b = gbr * (1 + br*amp*b)
// 	// WAS b = gbr / br * (br + amp*b)

// 	r = math.Max(0, math.Min(255, r))
// 	g = math.Max(0, math.Min(255, g))
// 	b = math.Max(0, math.Min(255, b))

// 	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
// }
