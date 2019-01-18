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
	err          []float64
	size         int
	kp           float64
	kd           float64
}

func newVariableGainController(size int, params []float64) *variableGainController {
	gain := make([]float64, size)
	err := make([]float64, size)
	for i := range gain {
		gain[i] = 1
	}
	return &variableGainController{
		filterParams: mat.NewVecDense(2, params),
		frame:        mat.NewVecDense(size, nil), // used to keep an internal LPF of the input
		gain:         gain,
		err:          err,
		size:         size,
		kp:           2,
		kd:           8,
	}
}

func (v *variableGainController) apply(input []float64) {
	m := mat.NewDense(2, v.size, append(input, v.frame.RawVector().Data...))
	v.frame.MulVec(m.T(), v.filterParams)

	var e = make([]float64, v.size)

	for i := range e {
		//e[i] = sigmoidCurve(1 - v.frame.AtVec(i))
		//e[i] = quadraticCurve(1 - v.frame.AtVec(i))
		e[i] = logCurve(.0000001 + v.frame.AtVec(i))
		//e[i] = errCurve(v.frame.AtVec(i) - 1)
	}

	for i := range e {
		u := v.kp*e[i] + v.kd*(e[i]-v.err[i])
		v.gain[i] += u
		if v.gain[i] > 1000000 {
			v.gain[i] = 1000000
		} else if v.gain[i] < .000001 {
			v.gain[i] = .000001
		}
		// bs, _ := json.Marshal(v.gain)
		// fmt.Println(string(bs))
		//fmt.Println(v.frame.AtVec(i), u, v.gain[i])
		v.err[i] = e[i]
	}
}

func Sigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

func sigmoidCurve(x float64) float64 {
	return 2*Sigmoid(x) - 1
}

func quadraticCurve(x float64) float64 {
	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	return sign * x * x
}

func logCurve(x float64) float64 {
	// if math.IsNaN(x) {
	// 	panic("nan")
	// }
	sign := 1.0
	if x > 0 {
		sign = -1.0
	}
	return sign * (math.Log2(math.Abs(x)))
}

func errCurve(x float64) float64 {
	if x > 0 {
		return -1 * x * x
	} else {
		return math.Log2(1.0000001 - x)
	}
}
