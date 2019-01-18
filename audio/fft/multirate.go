package fft

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"sync"

	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"github.com/peragwin/vuzicgo/audio/util"
)

// MultiRateFFT is a processor that returns a number of layered FFT's at different rates
// corresponding to different segments of the frequency spectrum.
// Given the inputs @minFrameSize and @maxFrameSize (both powers of 2), a number of FFT's
// equating to `N = log(2)(maxFrameSize / minFrameSize)` will be constructed. A chunk
// of each of these FFTs is used to return N layers which trade off on resolution of
// time and frequency. The "highest" layer has the best frequency resolution but higest
// latency, while this is traded exponentially with each later until the "lowest" has
// the least latency but also with the least frequency resolution.
//
// Layers are calculated by exapanding the Cooley-Tukey algorithm and building each higher
// layer in the recursion using a sliding tree of FFT windows, each of @minFrameSize.
//
// For example, a @maxFrameSize of 4096 with a @minFrameSize of 128 produces `(12 - 6) = 6`
// layers of FFT's. We'll call the size of each of these FFT's `N_k for k=0...5`. Each layer
// corresponds to a segment of `FFT_k[ N_k / (2 ^ (k+1)) : N_k / (2 ^ k) ]`. Therefore,
// the number of bins in each layer is always (@minFrameSize / 2), and each layer corresponds
// to a segment of the overall frequency spectrum with resolution increasing exponentially
// with `k`. The upper and lower frequency of each layer can be calculated as
// `Fmin = Fs / (2 ^ (k+1))` and `Fmax = Fs / (2 ^ k)`
//
// The @synchronous argument determines that all FFT's should be returned on one channel
// in a single concatenated array at the rate @minFrameSize frames are generted. If false,
// N channels are returned with frames being produced at the rate that a unique frame
// would be generated for the @minFrameSize.
type MultiRateFFT struct {
	minFrameSize, maxFrameSize int
	sync                       bool
}

// NewMultiRateFFT constructs a new MultiRateFFT processor
func NewMultiRateFFT(minFrameSize, maxFrameSize int, sync bool) (MultiRateFFT, error) {
	if !powerOf2(minFrameSize) {
		return MultiRateFFT{}, errors.New("minFrameSize is not a power of 2")
	}
	if !powerOf2(maxFrameSize) {
		return MultiRateFFT{}, errors.New("maxFrameSize is not a power of 2")
	}
	return MultiRateFFT{minFrameSize: minFrameSize, maxFrameSize: maxFrameSize, sync: sync}, nil
}

// Process returns a channel where incoming []float64 frames are processed and output
// over a number of channels for each layer (or on one channel with layers concatenated
// if sync == true.)
func (m MultiRateFFT) Process(done chan struct{}, in chan []float64) []chan []complex128 {
	numLayers := int(math.Log2(float64(m.maxFrameSize/m.minFrameSize))) + 1

	out := make([]chan []complex128, numLayers)
	var syncOut chan []complex128
	if m.sync {
		syncOut = make(chan []complex128)
	}
	proc := newMultiproc(m.minFrameSize, m.maxFrameSize, out, syncOut)

	go func() {
		defer close(done)

		var frame []float64

		for {
			select {
			case <-done:
				return
			case frame = <-in:
			}

			proc.push(frame)
		}
	}()

	if m.sync {
		return []chan []complex128{syncOut}
	}

	return out
}

type multiproc struct {
	sync.RWMutex
	mods      []int
	out       []chan []complex128
	windows   [][]float64
	buffer    *util.RingBuffer
	index     int
	syncOutCh chan []complex128
	syncOut   []complex128
	sync      bool
}

func newMultiproc(minFrameSize, maxFrameSize int,
	out []chan []complex128, syncOutCh chan []complex128) *multiproc {
	p := int(math.Log2(float64(maxFrameSize / minFrameSize)))
	mods := make([]int, len(out))
	s := syncOutCh != nil
	var syncOut []complex128
	if s {
		syncOut = make([]complex128, (p+2)*minFrameSize/4)
		fmt.Println("@@@@@@@@@ len sync out", len(syncOut))
	}

	windows := make([][]float64, len(mods))
	for i := range mods {
		pow := int(math.Pow(2, float64(i)))
		size := pow * minFrameSize
		windows[i] = window.Hamming(size)
	}

	return &multiproc{
		mods:      mods,
		buffer:    util.NewRingBuffer(2 * maxFrameSize),
		out:       out,
		windows:   windows,
		syncOutCh: syncOutCh,
		sync:      s,
		syncOut:   syncOut,
	}
}

func (m *multiproc) push(frame []float64) {
	m.buffer.Push(frame)

	wg := &sync.WaitGroup{}
	if m.sync {
		defer func() {
			wg.Wait()
			m.syncOutCh <- m.syncOut
		}()
	}

	for i, v := range m.mods {
		pow := int(math.Pow(2, float64(i)))
		size := pow * len(frame)
		if (v+len(frame))%(size/2) == 0 {
			wg.Add(1)
			go func(i int, out chan []complex128) {
				defer wg.Done()
				w := m.windows[i]

				fx := m.buffer.Get(size)
				fw := make([]float64, size)
				for j := range fx {
					fw[j] = w[j] * fx[j]
				}

				start := len(frame) / 4
				end := len(frame) / 2
				if i == len(m.mods)-1 {
					start = 0
				}
				X := fft.FFTReal(fw)[start:end]

				if m.sync {
					for j := range X {
						l := len(frame) / 4
						s := i * l
						m.syncOut[j+s] = X[len(X)-j-1]
					}
				} else {
					out <- X
				}
			}(i, m.out[i])
		}
		m.mods[i] = (m.mods[i] + len(frame)) % size
	}
}

type ffter interface {
	len() int
	fft() []complex128
}

type node struct {
	size   int
	even   ffter //*layer
	odd    ffter //*layer
	frames [][]complex128
	mod    int

	twiddle []complex128
}

func newNode(size int, even, odd ffter, twiddle []complex128) *node {
	return &node{
		size: size, even: even, odd: odd, twiddle: twiddle,
	}
}

func (m *node) push(frame []complex128) {
	if len(frame) != m.size {
		panic(fmt.Sprintf("pushed frame size %d != %d layer size", len(frame), m.size))
	}
	m.frames[m.mod] = frame
	m.mod = (m.mod + 1) % 2
}

func (m *node) fft() []complex128 {
	even := m.even.fft()
	odd := m.odd.fft()

	fmt.Println("even fft", even[:10])
	fmt.Println("odd fft", odd[:10])

	fft := make([]complex128, m.size)

	for i := 0; i < m.size/2; i++ {
		fft[i] = even[i] + m.twiddle[i]*odd[i]
		fft[i+m.size/2] = even[i] + m.twiddle[i+m.size/2]*odd[i]
	}

	return fft
}

func (m *node) len() int { return m.size }

func makeLayer(nodes []ffter) []*node {
	l := len(nodes) / 2
	layer := make([]*node, l)
	size := 2 * nodes[0].len()
	twiddle := makeTwiddle(size * l)
	for i := 0; i < l; i++ {
		layer[i] = newNode(size, nodes[2*i], nodes[2*i+1], twiddle[i*size:(i+1)*size])
	}
	return layer
}

type leaf struct {
	size  int
	frame []float64
}

func (l *leaf) fft() []complex128 {
	fx := make([]float64, len(l.frame))
	w := window.Hamming(len(l.frame))
	for i := range fx {
		fx[i] = l.frame[i] * w[i]
	}
	fmt.Println("leaf vals", fx[:10])
	return fft.FFTReal(fx)
}

func (l *leaf) len() int { return l.size }

func makeLeaves(frame []float64, n int) []*leaf {
	l := len(frame) / n
	leaves := make([]*leaf, n)
	for i := 0; i < n; i++ {
		f := make([]float64, l)
		for j := 0; j < l; j++ {
			f[j] = frame[n*i+j]
		}
		leaves[i] = &leaf{frame: f, size: l}
	}
	return leaves
}

func powerOf2(x int) bool {
	l := math.Log2(float64(x))
	return l == math.Floor(l)
}

func makeTwiddle(n int) []complex128 {
	// factor = np.exp(-2j * np.pi * np.arange(N) / N)
	t := make([]complex128, n)
	for i := 0; i < n; i++ {
		t[i] = cmplx.Exp(-2i * math.Pi * complex(float64(i)/float64(n), 0))
	}
	return t
}
