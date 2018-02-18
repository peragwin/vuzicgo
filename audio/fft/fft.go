package fft

import (
	"math"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

type FFTProcessor struct {
	SampleRate float64
	Size       int

	window []float64
}

func NewFFTProcessor(sampleRate float64, size int) *FFTProcessor {
	w := window.Hamming(size)
	return &FFTProcessor{
		SampleRate: sampleRate,
		Size:       size,
		window:     w,
	}
}

func (f *FFTProcessor) Process(done chan struct{}, in chan []float64) chan []complex128 {

	out := make(chan []complex128)
	//errc := make(chan error)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			fx := <-in
			window.Apply(fx, window.Hamming)
			out <- fft.FFTReal(fx)[1 : len(fx)/2]
		}
	}()

	return out
}

type PowerSpectrumProcessor struct {
}

func (p *PowerSpectrumProcessor) Process(done chan struct{}, in chan []complex128) chan []float64 {

	out := make(chan []float64)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			Fx := <-in
			Px := make([]float64, len(Fx))
			N := float64(len(Px))

			for i, f := range Fx {
				Px[i] = math.Sqrt(real(cmplx.Conj(f)*f)) / N
			}
			for i := range Px {
				Px[i] = math.Log(1 + Px[i])
			}

			out <- Px
		}
	}()

	return out
}

func SpectrumProcessor(done chan struct{}, in <-chan []float32, size int) chan []float64 {
	out := make(chan []float64)

	var x []float32
	var fx = make([]float64, size)
	var Px = make([]float64, size)
	var Fx []complex128
	var N = float64(size)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			x = <-in
			for i := range x {
				fx[i] = float64(x[i])
			}

			window.Apply(fx, window.Hamming)
			Fx = fft.FFTReal(fx)

			for i, f := range Fx {
				Px[i] = real(cmplx.Conj(f)*f) / N
			}
			for i := range Px {
				Px[i] = math.Log(1 + Px[i])
			}

			out <- Px
		}
	}()

	return out
}
