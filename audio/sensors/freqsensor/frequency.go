package freqsensor

import (
	"errors"
	"log"
	"math"

	"github.com/graphql-go/graphql"
	"github.com/peragwin/vuzicgo/audio/util"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

var (
	defaultFilterParams = filterValues{
		gain: mat.NewDense(2, 2, []float64{
			0.80, +0.20,
			-0.0005, 0.9995,
		}),
		diff: mat.NewDense(2, 2, []float64{
			0.263, .737,
			-0.0028, 0.2272,
		}),
	}
	defaultVGCParams = []float64{0.005, 0.995}

	// DefaultParameters is a set of default parameters that work okay
	DefaultParameters = &Parameters{
		GlobalBrightness: 127,
		Brightness:       4,
		Offset:           0,
		Period:           24,
		Gain:             4,
		Preemphasis:      2,
		DifferentialGain: 22e-4,
		Sync:             36e-4,
		Mode:             AnimateMode,
		WarpOffset:       0.68,
		WarpScale:        1.33,
		WarpSpring:       0.01,
		WarpFriction:     0.001,
		Scale:            1.5,
		SaturationOffset: 0.0,
		SaturationScale:  0.7,
		ValueOffset1:     2.0,
		ValueOffset2:     0.0,
		Alpha:            0.0,
		AlphaOffset:      0.0,
		ScaleOffset:      0.71,
		ColumnDivider:    2,
	}
)

type Drivers struct {
	// Amplitude is the immediate amplitudes of the frequency response for each frame
	Amplitude [][]float64
	// Diff is how much the frame chanced since the last input
	Diff []float64
	// Energy is the abs overall accumulated differential
	Energy []float64
	// Bass keeps track of how intense the current base is
	Bass float64
	// Scales is a scale that tries to keep the variation of Amplitude between -1 and 1
	Scales []float64
}

// FrequencySensor is the main object that generates visualization parameters from incoming
// log power spectral frames.
type FrequencySensor struct {
	Frames  int
	Buckets int

	Drivers

	params       *Parameters
	filterParams filterValues
	filterValues filterValues
	vgc          *variableGainController

	valueScales     []float64
	valueOffsets    []float64
	valueHistory    []float64
	valueMaxHistory []float64

	schema graphql.Schema

	frameCount int
}

// NewFrequencySensor creates a new FrequencySensor from a Config
func NewFrequencySensor(cfg *Config) *FrequencySensor {
	amp := make([][]float64, cfg.Columns)
	vos := make([]float64, cfg.Buckets)
	vhs := make([]float64, cfg.Buckets)
	vmhs := make([]float64, cfg.Buckets)
	vss := make([]float64, cfg.Buckets)
	for i := range amp {
		amp[i] = make([]float64, cfg.Buckets)
	}
	for i := range vos {
		vos[i], vss[i], vhs[i], vmhs[i] = -1, 1, 1, 1
	}
	fs := &FrequencySensor{
		Frames:  cfg.Columns,
		Buckets: cfg.Buckets,
		Drivers: Drivers{
			Amplitude: amp,
			Scales:    vss,
			Energy:    make([]float64, cfg.Buckets),
			Diff:      make([]float64, cfg.Buckets),
		},
		params:       cfg.Parameters,
		filterParams: defaultFilterParams,
		filterValues: filterValues{
			gain: mat.NewDense(2, cfg.Buckets, nil),
			diff: mat.NewDense(2, cfg.Buckets, nil),
		},
		vgc: newVariableGainController(cfg.Buckets, defaultVGCParams),

		valueOffsets:    vos,
		valueHistory:    vhs,
		valueMaxHistory: vmhs,
	}
	if err := fs.initGraphql(); err != nil {
		panic(err)
	}
	return fs
}

// Process generates the frames of the visualization from input
func (d *FrequencySensor) Process(done chan struct{}, in chan []float64) chan *Drivers {

	x := <-in
	bucketer := util.NewBucketer(util.LogScale2, d.Buckets, len(x), 32, 16000)
	buckets := util.NewBucketProcessor(bucketer).Process(done, in)

	out := make(chan *Drivers)

	bucketSum := 0.0
	isBadFrame := func(x []float64) bool {
		sum := 0.0
		for i := range x {
			sum += x[i]
		}
		var bad bool
		if sum > 32*bucketSum {
			bad = true
			log.Println("[Info] Bad frame!")
		}
		bucketSum = .01*sum + .99*bucketSum
		return bad
	}

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

			if isBadFrame(x) {
				continue
			}

			d.applyPreemphasis(x)

			d.applyFilters(x)
			d.applyChannelEffects()
			d.applyChannelSync()
			d.adjustValueRanges()
			d.applyBase(d.Amplitude[0])

			d.frameCount++

			out <- &d.Drivers
		}
	}()

	return out
}

// tao is a value  >=1 which determines the time constant of the filter. A value of 1 means
// no lowpass, where a large value means a long time delay.
func (d *FrequencySensor) SetFilterParams(typ string, level int, gain, tao float64) error {
	if math.Abs(tao) < 1 {
		//return errors.New("|tao| < 1 undefined")
		sign := math.Signbit(tao)
		tao = 1
		if sign {
			tao = -1
		}
	}

	a := 1 / math.Abs(tao)
	b := 1 - a
	a *= gain
	b *= gain
	if tao < 0 {
		a = -a
	}
	params := []float64{a, b}

	var m *mat.Dense
	switch typ {
	case "amp":
		m = d.filterParams.gain
	case "diff":
		m = d.filterParams.diff
	default:
		return errors.New("typ must be either 'amp' or 'diff'")
	}
	rows, _ := m.Dims()
	if level >= rows {
		return errors.New("level not defined for filter typ")
	}

	m.SetRow(level, params)
	return nil
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
	// if d.params.Debug && d.frameCount%200 == 0 {
	// 	bs, _ := json.Marshal(map[string]interface{}{"vgc.gain": d.vgc.gain})
	// 	fmt.Println(string(bs))
	// }
}

func (d *FrequencySensor) adjustValueRanges() {
	frame := d.Drivers.Amplitude[0]
	for i := range frame {
		vh := d.valueHistory[i]
		// vo := d.valueOffsets[i]
		vs := d.Drivers.Scales[i]
		vmh := d.valueMaxHistory[i]

		// vo is clamped to -1 instead
		// vh = 0.005*frame[i] + 0.995*vh
		// e := logCurve(0.000001 - (vh + vo))
		// vo += .001 * e

		sval := vs * (frame[i] - 1)
		sval *= sval
		if sval < vmh {
			vmh = 0.001*sval + .999*vmh
		} else {
			vmh = .001*sval + .999*vmh
		}
		if vmh < .001 {
			vmh = .001
		}
		vs = 1 / vmh

		d.valueHistory[i] = vh
		// d.valueOffsets[i] = vo
		d.valueMaxHistory[i] = vmh
		d.Drivers.Scales[i] = vs
	}
}

func (d *FrequencySensor) applyChannelEffects() {
	dg := d.params.DifferentialGain
	ag := d.params.Gain
	ao := d.params.Offset

	gain := d.filterValues.gain.RawRowView(0)
	diff := d.filterValues.diff.RawRowView(0)

	if d.params.Mode == AnimateMode && d.frameCount%d.params.ColumnDivider == 0 {
		decay := 1 - (2.0 / float64(d.Frames))
		for i := len(d.Amplitude) - 2; i >= 0; i-- { // -2
			for j := range d.Amplitude[i] {
				d.Amplitude[i+1][j] = decay * d.Amplitude[i][j]
			}
		}
	}

	for i := range gain {
		d.Amplitude[0][i] = ao + ag*gain[i]
	}
	for i := range diff {
		d.Diff[i] = diff[i]

		ph := d.Energy[i] + .001 // apply a constant opposing pressure
		ph -= dg * math.Abs(diff[i])
		d.Energy[i] = ph
	}
}

func (d *FrequencySensor) applyChannelSync() {
	avg := floats.Sum(d.Energy) / float64(d.Buckets)
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
	for i := 1; i < len(d.Energy)-1; i++ {
		diff := d.Energy[i-1] - d.Energy[i]
		sign := 1.0
		if diff < 0 {
			sign = -1
		}
		diff = sign * diff * diff
		d.Energy[i] += 10 * d.params.Sync * diff

		diff = d.Energy[i+1] - d.Energy[i]
		sign = 1.0
		if diff < 0 {
			sign = -1
		}
		diff = sign * diff * diff
		d.Energy[i] += 10 * d.params.Sync * diff
	}

	avg = floats.Sum(d.Energy) / float64(d.Buckets)
	if avg < -2*math.Pi {
		for _, e := range d.Energy {
			if e >= -2*math.Pi {
				return
			}
		}
		for i := range d.Energy {
			d.Energy[i] = 2*math.Pi + math.Mod(d.Energy[i], 2*math.Pi)
		}
		avg = 2*math.Pi + math.Mod(avg, 2*math.Pi)
	}
	if avg > 2*math.Pi {
		for _, e := range d.Energy {
			if e <= 2*math.Pi {
				return
			}
		}
		for i := range d.Energy {
			d.Energy[i] = math.Mod(d.Energy[i], 2*math.Pi)
		}
		avg = math.Mod(avg, 2*math.Pi)
	}
}

var baseFilter = []float64{1, .75, 0.5, 0.25}

func (d *FrequencySensor) applyBase(frame []float64) {
	//cutoff := 4

	bass := 0.0
	for i := range baseFilter {
		v := frame[i]
		if v < 0 {
			v = 0
		}
		bass += v * baseFilter[i]
	}
	// if d.params.Debug && d.frameCount%10 == 0 {
	// 	fmt.Println("@@@ BASE", bass)
	// }
	bass /= 2
	bass = math.Log(1 + bass)
	// if d.params.Debug && d.frameCount%10 == 0 {
	// 	fmt.Println("@@@ BASE", bass)
	// }
	d.Bass = .25*bass + .75*d.Bass
}

func (d *FrequencySensor) applyPreemphasis(frame []float64) {
	incr := (d.params.Preemphasis - 1) / float64(d.Buckets)
	for i := range frame {
		frame[i] *= 1 + float64(i)*incr
	}
}
