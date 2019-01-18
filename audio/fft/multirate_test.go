package fft

import (
	"math"
	"testing"

	"github.com/mjibson/go-dsp/fft"
)

func allClose(a, b []complex128) bool {
	for i := range a {
		if math.Abs(real(a[i])-real(b[i])) > 1e-9 || math.Abs(imag(a[i])-imag(b[i])) > 1e-9 {
			return false
		}
	}
	return true
}

func TestLayers(t *testing.T) {
	f := make([]float64, 4096)
	for i := range f {
		//f[i] = rand.Float64() - 0.5
		x := (float64(i)/128.0 - 32.0)
		f[i] = math.Exp(-0.5 * x * x)
	}

	F := fft.FFTReal(f)

	leaves := makeLeaves(f, 4096/128)
	nodes := make([]ffter, len(leaves))
	for i, lf := range leaves {
		nodes[i] = lf
	}
	for len(nodes) > 1 {
		// if len(nodes) == 1 { break }
		ns := makeLayer(nodes)
		nodes = make([]ffter, len(ns))
		for i, n := range ns {
			nodes[i] = n
		}
	}

	G := nodes[0].fft()

	if !allClose(F, G) {
		t.Fatal("nope", F[:10], G[:10])
	}
}
