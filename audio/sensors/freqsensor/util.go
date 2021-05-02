package freqsensor

import (
	gomath "math"

	math "github.com/chewxy/math32"
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

var sigmoidTable = make([]float32, 2000)
var sigmoidRange = float32(10.0)
var sigmoidScale = float32(len(sigmoidTable)) / (2 * sigmoidRange)

func init() {
	hl := len(sigmoidTable) / 2
	for i := range sigmoidTable {
		v := float32(i-hl) / sigmoidScale
		sigmoidTable[i] = 1 / (1 + math.Exp(-v))
	}
}

func Sigmoid(x float32) float32 {
	if x >= sigmoidRange {
		return sigmoidTable[len(sigmoidTable)-1]
	} else if x <= -sigmoidRange {
		return sigmoidTable[0]
	}
	idx := int(x*sigmoidScale) + len(sigmoidTable)/2
	// fmt.Println("sig for", x, idx)
	return sigmoidTable[idx]

	// if r, ok := sigmoidCache[v]; ok {
	// 	return r
	// }

	// log.Println("sigmoid cache miss", v)

	// r := 1 / (1 + math.Exp(-x))

	// return r
}

func sigmoidCurve(x float32) float32 {
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
	return sign * (gomath.Log2(gomath.Abs(x)))
}

func errCurve(x float32) float32 {
	if x > 0 {
		return -1 * x * x
	} else {
		return math.Log2(1.0000001 - x)
	}
}
