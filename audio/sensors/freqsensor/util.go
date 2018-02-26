package freqsensor

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type drivers struct {
	amplitude [][]float64
	phase     []float64
}

func newDrivers(rows int, columns int) drivers {
	amp := make([][]float64, columns)
	for i := range amp {
		amp[i] = make([]float64, rows)
	}
	return drivers{
		amp, make([]float64, rows),
	}
}

type filterValues struct {
	gain *mat.Dense
	diff *mat.Dense
}

type variableGainController struct {
	filterParams *mat.VecDense
	frame        *mat.VecDense
	gain         []float64
	size         int
}

func newVariableGainController(size int, params []float64) *variableGainController {
	init := make([]float64, size)
	for i := range init {
		init[i] = 1
	}
	return &variableGainController{
		filterParams: mat.NewVecDense(2, params),
		frame:        mat.NewVecDense(size, nil), // used to keep an internal LPF of the input
		gain:         init,
		size:         size,
	}
}

func (v *variableGainController) apply(input []float64) {
	m := mat.NewDense(2, v.size, append(input, v.frame.RawVector().Data...))
	v.frame.MulVec(m.T(), v.filterParams)

	var g = make([]float64, v.size)

	for i := range g {
		//g[i] = sigmoidCurve(1 - v.frame.AtVec(i))

		g[i] = customCurve(1 - v.frame.AtVec(i))
	}

	for i := range g {
		v.gain[i] += g[i]
	}
}

func Sigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

func sigmoidCurve(x float64) float64 {
	return 2*Sigmoid(x) - 1
}

func customCurve(x float64) float64 {
	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	return sign * x * x
}
