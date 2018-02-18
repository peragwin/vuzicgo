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
			out <- fft.FFTReal(fx)
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
