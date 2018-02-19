package main

import (
	"image"
	"image/color"
	"math"

	"github.com/peragwin/vuzicgo/audio/util"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

var (
	defaultFilterParams = filterValues{
		gain: mat.NewDense(2, 2, []float64{
			0.950, +0.100,
			-0.005, 0.995,
		}),
		diff: mat.NewDense(2, 2, []float64{
			0.95, +0.10,
			-0.04, 0.96,
		}),
	}
	defaultVGCParams = []float64{0.05, 0.95}

	// defaultParameters is a set of default parameters that work okay
	defaultParameters = Parameters{
		GlobalBrightness: 127, // center around 50% brightness
		Brightness:       2,
		Offset:           1.5,
		Period:           150,
		Gain:             1,
		DifferentialGain: 4e-3,
		Sync:             1.8e-3,
	}
)

// Display is the main object that generate the visualization
type Display struct {
	Columns    int
	Buckets    int
	SampleRate float64

	params       Parameters
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
		drivers:      newDrivers(cfg.Buckets),
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
	bucketer := util.NewBucketer(util.MelScale, d.Buckets, len(x), 32, 16000)
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

	var meangain float64
	for i := range gain {
		meangain += gain[i]
		d.drivers.amplitude[i] = ao + ag*gain[i]
	}
	for i := range diff {
		ph := d.drivers.phase[i] + .001 // apply a constant opposing pressure
		ph -= dg * math.Log(1+math.Abs(diff[i]))
		d.drivers.phase[i] = ph
	}
}

func (d *Display) applyChannelSync() {
	avg := floats.Sum(d.drivers.phase) / float64(d.Buckets)
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

func (d *Display) render() {
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
	br := d.params.Brightness
	gbr := d.params.GlobalBrightness
	amp := d.drivers.amplitude
	phase := d.drivers.phase
	ws := 2.0 * math.Pi / float64(d.params.Period)
	phi := ws * float64(col)

	colors := make([]color.RGBA, d.Buckets)

	for i, ph := range phase {
		r := math.Sin(ph + phi)
		g := math.Sin(ph + phi + 2*math.Pi/3)
		b := math.Sin(ph + phi - 2*math.Pi/3)

		// TODO print norm and see if it's contant to optimize
		norm := math.Abs(r) + math.Abs(g) + math.Abs(b)
		r /= norm
		g /= norm
		b /= norm

		r = gbr / br * (br + amp[i]*r)
		g = gbr / br * (br + amp[i]*g)
		b = gbr / br * (br + amp[i]*b)

		r = math.Max(0, math.Min(255, r))
		g = math.Max(0, math.Min(255, g))
		b = math.Max(0, math.Min(255, b))

		colors[i] = color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}

	return colors
}
