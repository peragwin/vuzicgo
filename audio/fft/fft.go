package fft

import (
	"math"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

// FFTProcessor is a processor that performs FFT on incoming frames
type FFTProcessor struct {
	SampleRate float64
	Size       int

	window []float64
}

// NewFFTProcessor creates a new processor that performs FFT on incoming frames
func NewFFTProcessor(sampleRate float64, size int) *FFTProcessor {
	w := window.Hamming(size)
	return &FFTProcessor{
		SampleRate: sampleRate,
		Size:       size,
		window:     w,
	}
}

// Process processes incoming frames by applying a Hamming window then performing a real FFT
func (f *FFTProcessor) Process(done chan struct{}, in chan []float64) chan []complex128 {

	out := make(chan []complex128)

	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			default:
			}

			fx := <-in
			if fx == nil {
				return
			}

			for i := range fx {
				fx[i] = f.window[i] * fx[i]
			}
			out <- fft.FFTReal(fx)[:len(fx)/2]
		}
	}()

	return out
}

// PowerSpectrumProcessor processes takes FFT frames as input and computes the log power spectrumm
type PowerSpectrumProcessor struct {
	FFTProcessor
}

func NewPowerSpectrumProcessor(sampleRate float64, size int) *PowerSpectrumProcessor {
	return &PowerSpectrumProcessor{
		FFTProcessor: *NewFFTProcessor(sampleRate, size),
	}
}

func (p *PowerSpectrumProcessor) Apply(fx []float64) []float64 {
	for i := range fx {
		fx[i] = p.window[i] * fx[i]
	}
	Fx := fft.FFTReal(fx)[:len(fx)/2]

	Px := make([]float64, len(Fx))
	N := float64(len(Px))
	for i, f := range Fx {
		Px[i] = math.Log(1+real(cmplx.Conj(f)*f)/N) / 2.0
	}

	return Px
}

// Process processes FFT input frames and outputs log power spectral frames
func (p *PowerSpectrumProcessor) Process(done chan struct{}, in chan []complex128) chan []float64 {

	out := make(chan []float64)

	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			default:
			}

			Fx := <-in
			if Fx == nil {
				return
			}

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
		defer close(out)
		for {
			select {
			case <-done:
				return
			default:
			}

			x = <-in
			if x == nil {
				return
			}
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
