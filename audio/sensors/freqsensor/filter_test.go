package freqsensor

import (
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/plot/vg"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
)

var testFilterParams = filterValues{
	gain: mat.NewDense(2, 2, []float64{
		0.2, 0.8,
		0, 0, //-0.01, 0.99,
	}),
	diff: mat.NewDense(2, 2, []float64{
		0.95, 0.10,
		-0.04, 0.96,
	}),
}

func TestFilter(t *testing.T) {
	//drivers := newDrivers(1)
	newDisplay := func() *FrequencySensor {
		return &FrequencySensor{
			Buckets: 1,
			params:  DefaultParameters,
			//drivers:      drivers,
			filterParams: testFilterParams,
			filterValues: filterValues{
				gain: mat.NewDense(2, 1, nil),
				diff: mat.NewDense(2, 1, nil),
			},
			vgc: newVariableGainController(1, []float64{.001, .999}),
		}
	}

	testFilterWithInput := func(t *testing.T, name string, input []float64, size int) {
		d := newDisplay()
		d.vgc.kp = 0.005
		d.vgc.kd = .02
		output0 := make([]float64, size)
		output1 := make([]float64, size)
		gain := make([]float64, size)
		vgcFrame := make([]float64, size)
		for i := range input {
			d.applyFilters([]float64{input[i]})
			output0[i] = d.filterValues.gain.At(0, 0)
			output1[i] = d.filterValues.gain.At(1, 0)
			gain[i] = d.vgc.gain[0] / 10
			vgcFrame[i] = d.vgc.frame.AtVec(0)
		}

		p, err := plot.New()
		if err != nil {
			t.Fatal(err)
		}

		if err := plotutil.AddLinePoints(p,
			"Input", newPlotter(input),
			"Output0", newPlotter(output0),
			"Output1", newPlotter(output1),
			"Gain", newPlotter(gain),
			"VGC Frame", newPlotter(vgcFrame),
		); err != nil {
			t.Fatal(err)
		}

		if err := p.Save(16*vg.Inch, 8*vg.Inch, name+".png"); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("Test Impulses", func(t *testing.T) {
		size := 1024 * 16
		input := make([]float64, size)
		for i := range input {
			// if (i/128/8)%2 == 1 {
			input[i] = (1 + math.Sin(2*math.Pi/float64(size)*32*float64(i))) / 2
			// } else {
			// 	input[i] = .2 * math.Sin(2*math.Pi/256*float64(i))
			// }

			// s := i / size
			// input[i] = float64((s + 1) % 2) //.1 * (float64(s) / 2 * (float64((s+1)%2) + .001))
		}
		testFilterWithInput(t, "testImpulses", input, size)
	})
	t.Run("Test Random Input", func(t *testing.T) {
		size := 1024
		input := make([]float64, size)
		a := make([]float64, size)
		for i := range input {
			if i == 0 {
				input[0] = .2 * rand.Float64()
				a[0] = .2 * rand.Float64()
				continue
			}
			if i%32 == 0 {
				a[i] = .5 * rand.Float64()
			} else {
				a[i] = a[i-1]
			}
			input[i] = (.1*(rand.Float64()+a[i]) + .9*input[i-1]) / 10
		}
		testFilterWithInput(t, "testRandom", input, size)
	})
}

func TestValueRanges(t *testing.T) {
	d := &FrequencySensor{
		valueHistory:    []float64{1},
		valueMaxHistory: []float64{1},
		valueScales:     []float64{1},
		valueOffsets:    make([]float64, 1),
	}

	size := 2048

	in := make([]float64, size)
	for i := range in {
		in[i] = rand.Float64()
	}
	values := make([]float64, size)
	for i := range values {
		v := 0.0
		for j := -5; j < 4; j++ {
			if (i+j) >= 0 && (i+j) < size {
				v += in[i+j]
			}
		}
		values[i] = v / 10
	}
	// t.Error(values)
	for i := 1024; i < size; i++ {
		if (i/64)%2 == 0 {
			values[i] = 2
		}
	}

	vhout := make([]float64, size)
	vmhout := make([]float64, size)
	vsout := make([]float64, size)
	voout := make([]float64, size)
	out := make([]float64, size)
	for i := 0; i < size; i++ {
		d.adjustValueRanges([]float64{values[i]})
		vhout[i] = d.valueHistory[0]
		vmhout[i] = d.valueMaxHistory[0]
		vsout[i] = d.valueScales[0]
		voout[i] = d.valueOffsets[0]

		out[i] = vsout[i] * (values[i] + voout[i])
	}

	p, err := plot.New()
	if err != nil {
		t.Fatal(err)
	}

	if err := plotutil.AddLinePoints(p,
		"Input", newPlotter(values),
		"ValueHistory", newPlotter(vhout),
		"ValueMaxHistory", newPlotter(vmhout),
		"ValueScale", newPlotter(vsout),
		"ValueOffset", newPlotter(voout),
		"Output", newPlotter(out),
	); err != nil {
		t.Fatal(err)
	}

	if err := p.Save(16*vg.Inch, 8*vg.Inch, "testValueRange.png"); err != nil {
		t.Fatal(err)
	}
}

func newPlotter(data []float64) plotter.XYs {
	pts := make(plotter.XYs, len(data))
	for i := range pts {
		pts[i].X = float64(i)
		pts[i].Y = data[i]
	}
	return pts
}
