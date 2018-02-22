package main

import (
	"math"

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

type Drivers struct {
	// Amplitude is the immediate amplitudes of the frequency response for each frame
	Amplitude [][]float64
	// Energy is the overall accumulated energies
	Energy []float64
}

// FrequencySensor is the main object that generate the visualization
type FrequencySensor struct {
	Frames  int
	Buckets int

	Drivers

	params       *Parameters
	filterParams filterValues
	filterValues filterValues
	vgc          *variableGainController
}

// NewFrequencySensor creates a new FrequencySensor from a Config
func NewFrequencySensor(cfg *Config) *FrequencySensor {
	amp := make([][]float64, cfg.Columns)
	for i := range amp {
		amp[i] = make([]float64, cfg.Buckets)
	}

	return &FrequencySensor{
		Frames:  cfg.Columns,
		Buckets: cfg.Buckets,
		Drivers: Drivers{
			Amplitude: amp,
			Energy:    make([]float64, cfg.Buckets),
		},
		params:       cfg.Parameters,
		filterParams: defaultFilterParams,
		filterValues: filterValues{
			gain: mat.NewDense(2, cfg.Buckets, nil),
			diff: mat.NewDense(2, cfg.Buckets, nil),
		},
		vgc: newVariableGainController(cfg.Buckets, defaultVGCParams),
	}
}

// Process generates the frames of the visualization from input
func (d *FrequencySensor) Process(done chan struct{}, in chan []float64) chan *Drivers {

	x := <-in
	bucketer := util.NewBucketer(util.LogScale, d.Buckets, len(x), 32, 16000)
	buckets := util.NewBucketProcessor(bucketer).Process(done, in)

	out := make(chan *Drivers)

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

			out <- &d.Drivers
		}
	}()

	return out
}

func (d *FrequencySensor) applyFilters(frame []float64) {
	// apply variable gain
	for i := range frame {
		frame[i] *= d.vgc.gain[i]
	}
	d.adjustVariableGain(frame)

	var diffInput = mat.NewDense(2, d.Buckets, nil)
	d.applyFilter(frame, d.filterValues.gain, d.filterParams.gain, diffInput)
	d.applyFilter(diffInput.RawRowView(0), d.filterValues.diff, d.filterParams.diff, nil)

}

func (d *FrequencySensor) applyFilter(frame []float64, output, fp, di *mat.Dense) {
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
func (d *FrequencySensor) adjustVariableGain(frame []float64) {
	d.vgc.apply(frame)
}

func (d *FrequencySensor) applyChannelEffects() {
	dg := d.params.DifferentialGain
	ag := d.params.Gain
	ao := d.params.Offset

	gain := d.filterValues.gain.RawRowView(0)
	diff := d.filterValues.diff.RawRowView(0)

	if d.params.Mode == AnimateMode {
		decay := 1 - (2.0 / float64(d.Frames))
		for i := len(d.Amplitude) / 2; i >= 0; i-- { // -2
			for j := range d.Amplitude[i] {
				d.Amplitude[i+1][j] = decay * d.Amplitude[i][j]
			}
		}
	}

	for i := range gain {
		d.Amplitude[0][i] = ao + ag*gain[i]
	}
	for i := range diff {
		ph := d.Energy[i] + .001 // apply a constant opposing pressure
		ph -= dg * math.Abs(diff[i])
		d.Energy[i] = ph
	}
}

func (d *FrequencySensor) applyChannelSync() {
	avg := floats.Sum(d.Energy) / float64(d.Buckets)
	if avg < -2*math.Pi {
		for i := range d.Energy {
			d.Energy[i] += 2 * math.Pi
		}
	}
	if avg > 2*math.Pi {
		for i := range d.Energy {
			d.Energy[i] -= 2 * math.Pi
		}
	}
	for i, ph := range d.Energy {
		diff := avg - d.Energy[i]
		sign := math.Signbit(diff)
		diff *= diff
		if sign {
			diff = -diff
		}
		ph += d.params.Sync * diff
		d.Energy[i] = ph
	}
}
