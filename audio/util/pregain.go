package util

import (
	"fmt"
	"math"

	"github.com/golang/glog"
)

// PreGain applies scaling by looking at RMS energy of the audio signal.
type PreGain struct {
	filterParams [4]float64 // a, b, kp, kd
	rms          float64
	gain         float64
	err          float64

	kp float64
	kd float64
}

// NewPreGain returns a new PreGain stage.
func NewPreGain(filterParams [4]float64) *PreGain {
	return &PreGain{
		filterParams: filterParams,
		gain:         1.0,
		rms:          1.0,
	}
}

// Apply pre-gain to the frame.
func (p *PreGain) Apply(frame []float64) {
	sum := 0.0
	for i := range frame {
		frame[i] *= p.gain
		sum += frame[i] * frame[i]
	}

	rms := math.Sqrt(2.0 * sum / float64(len(frame)))
	p.rms = p.filterParams[0]*rms + p.filterParams[1]*p.rms
	rms = p.rms

	e := logCurve(0.0000001 + rms)
	u := p.filterParams[2]*e + p.filterParams[3]*(e-p.err)
	p.gain += u
	if p.gain > 1e6 {
		p.gain = 1e6
	} else if p.gain < 1e-6 {
		p.gain = 1e-6
	}
	p.err = e

	if glog.V(3) {
		fmt.Printf("rms = %.02f\tpregain = %.02f\n", rms, p.gain)
	}
}

func logCurve(x float64) float64 {
	sign := 1.0
	if x > 0 {
		sign = -1.0
	}
	return sign * (math.Log2(math.Abs(x)))
}
