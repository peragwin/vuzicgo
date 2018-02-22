package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/lucasb-eyer/go-colorful"

	"github.com/peragwin/vuzicgo/audio/util"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

var (
	defaultFilterParams = filterValues{
		gain: mat.NewDense(2, 2, []float64{
			0.80, +0.200,
			-0.005, 0.995,
		}),
		diff: mat.NewDense(2, 2, []float64{
			0.95, +0.10,
			-0.04, 0.96,
		}),
	}
	defaultVGCParams = []float64{0.05, 0.95}

	// defaultParameters is a set of default parameters that work okay
	defaultParameters = &Parameters{
		GlobalBrightness: 127, // center around 50% brightness
		Brightness:       2,
		Offset:           1,
		Period:           24,
		Gain:             1,
		DifferentialGain: 4e-3,
		Sync:             1.8e-3,
		Mode:             AnimateMode,
	}
)

// Display is the main object that generate the visualization
type Display struct {
	Columns    int
	Buckets    int
	SampleRate float64

	params       *Parameters
	drivers      drivers
	filterParams filterValues
	filterValues filterValues
	vgc          *variableGainController

	display *image.RGBA
}

// NewDisplay creates a new Display from a Config
func NewDisplay(cfg *Config) *Display {
	display := image.NewRGBA(image.Rect(0, 0, cfg.Columns, cfg.Buckets))
	return &Display{
		Columns:      cfg.Columns,
		Buckets:      cfg.Buckets,
		SampleRate:   cfg.SampleRate,
		params:       cfg.Parameters,
		display:      display,
		drivers:      newDrivers(cfg.Buckets, cfg.Columns),
		filterParams: defaultFilterParams,
		filterValues: filterValues{
			gain: mat.NewDense(2, cfg.Buckets, nil),
			diff: mat.NewDense(2, cfg.Buckets, nil),
		},
		vgc: newVariableGainController(cfg.Buckets, defaultVGCParams),
	}
}

// Process generates the frames of the visualization from input
func (d *Display) Process(done chan struct{}, in chan []float64, render chan struct{}) chan *image.RGBA {

	x := <-in
	bucketer := util.NewBucketer(util.LogScale, d.Buckets, len(x), 32, 16000)
	buckets := util.NewBucketProcessor(bucketer).Process(done, in)

	out := make(chan *image.RGBA)

	// set up a goroutine to process the bucketed input
	go func() {
		defer close(out)
		var x []float64
		for {
			select {
			case <-done:
				return
			default:
			}

			x = <-buckets
			if x == nil {
				return
			}

			d.applyFilters(x)
			d.applyChannelEffects()
			d.applyChannelSync()
		}
	}()

	// set up a goroutine to render only when a frame is requested
	go func() {
		for {
			<-render
			d.render()
			out <- d.display
		}
	}()

	return out
}

func (d *Display) applyFilters(frame []float64) {
	// apply variable gain
	for i := range frame {
		frame[i] *= d.vgc.gain[i]
	}
	d.adjustVariableGain(frame)

	var diffInput = mat.NewDense(2, d.Buckets, nil)
	d.applyFilter(frame, d.filterValues.gain, d.filterParams.gain, diffInput)
	d.applyFilter(diffInput.RawRowView(0), d.filterValues.diff, d.filterParams.diff, nil)

}

func (d *Display) applyFilter(frame []float64, output, fp, di *mat.Dense) {
	for level := 0; level < 2; level++ {
		// m looks like:
		// [ frame0, ..., frameN ]
		// [ out0,   ..., outN   ]
		m := mat.NewDense(2, d.Buckets, append(frame, output.RawRowView(level)...))
		at := fp.RowView(level)

		// perform the fitler operation using out.T = [frame[:], output[:]] * params.T
		var out = mat.NewVecDense(d.Buckets, nil)
		out.MulVec(m.T(), at)

		if di != nil {
			// get the differential since the last output
			// XXX why diff against the output of the filter instead of on the input?
			var s = mat.NewVecDense(d.Buckets, nil)
			s.AddVec(out, output.RowView(level))
			di.SetRow(level, s.RawVector().Data)
		}

		copy(frame, out.RawVector().Data)
		output.SetRow(level, frame)
	}

	// apply output of the second filter as feedback
	var s = mat.NewVecDense(d.Buckets, nil)
	s.AddVec(output.RowView(0), output.RowView(1))
	y := s.RawVector().Data
	output.SetRow(0, y)
}

// The VGA works by taking the sigmoid function of the difference of the current
// long-term gain value with 1. This value is then applied as input to a low-pass
// filter whose output will be the gain of the 1st level filter for the next incoming frame.
func (d *Display) adjustVariableGain(frame []float64) {
	d.vgc.apply(frame)
}

func (d *Display) applyChannelEffects() {
	dg := d.params.DifferentialGain
	ag := d.params.Gain
	ao := d.params.Offset

	gain := d.filterValues.gain.RawRowView(0)
	diff := d.filterValues.diff.RawRowView(0)

	decay := 1 - (2.0 / float64(d.Columns))
	if d.params.Mode == AnimateMode {
		for i := len(d.drivers.amplitude) / 2; i >= 0; i-- { // -2
			for j := range d.drivers.amplitude[i] {
				d.drivers.amplitude[i+1][j] = decay * d.drivers.amplitude[i][j]
			}
		}
	}

	for i := range gain {
		d.drivers.amplitude[0][i] = ao + ag*gain[i]
	}
	for i := range diff {
		ph := d.drivers.phase[i] + .001 // apply a constant opposing pressure
		ph -= dg * math.Abs(diff[i])
		d.drivers.phase[i] = ph
	}
}

func (d *Display) applyChannelSync() {
	avg := floats.Sum(d.drivers.phase) / float64(d.Buckets)
	if avg < -2*math.Pi {
		for i := range d.drivers.phase {
			d.drivers.phase[i] += 2 * math.Pi
		}
	}
	if avg > 2*math.Pi {
		for i := range d.drivers.phase {
			d.drivers.phase[i] -= 2 * math.Pi
		}
	}
	for i, ph := range d.drivers.phase {
		diff := avg - d.drivers.phase[i]
		sign := math.Signbit(diff)
		diff *= diff
		if sign {
			diff = -diff
		}
		ph += d.params.Sync * diff
		d.drivers.phase[i] = ph
	}
}

var renderCount = 0
var lastRender time.Time

func (d *Display) render() {
	renderCount++
	if d.params.Debug && renderCount%100 == 0 {
		diff := time.Now().Sub(lastRender)
		fmt.Println("fps:", diff/100.0)
		fmt.Println("amp:", d.drivers.amplitude[0])
		fmt.Println("pha:", d.drivers.phase)
		lastRender = time.Now()
	}
	hl := d.Columns / 2
	for i := 0; i < hl; i++ {
		col := d.renderColumn(i)
		for j, c := range col {
			d.display.SetRGBA(hl+i, d.Buckets-j-1, c)
			d.display.SetRGBA(hl-1-i, d.Buckets-j-1, c)
		}
	}
}

func (d *Display) renderColumn(col int) []color.RGBA {

	amp := d.drivers.amplitude[0]
	if d.params.Mode == AnimateMode {
		amp = d.drivers.amplitude[col]
	}
	phase := d.drivers.phase
	ws := 2.0 * math.Pi / float64(d.params.Period)
	phi := ws * float64(col)

	colors := make([]color.RGBA, d.Buckets)

	for i, ph := range phase {
		//colors[i] = getRGB(d.params, amp[i], ph, phi)
		colors[i] = getHSV(d.params, amp[i], ph, phi)
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
