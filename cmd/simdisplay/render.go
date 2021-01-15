package main

import (
	"image/color"
	"log"
	"math"
	"sync"
	"time"

	hsluv "github.com/hsluv/hsluv-go"
	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
	"github.com/peragwin/vuzicgo/gfx/skgrid"
)

type renderer struct {
	src     *fs.FrequencySensor
	columns int
	rows    int
	params  *fs.Parameters
	mirror  bool

	renderCount int
	lastRender  time.Time

	display [][]ARGBf
	warp    []float32
	scale   []float32
	bass    float32
}

func newRenderer(columns int, mirror bool, params *fs.Parameters,
	src *fs.FrequencySensor) *renderer {
	rows := src.Buckets
	if mirror {
		rows *= 2
	}
	// display := image.NewRGBA(image.Rect(0, 0, columns, rows))
	display := make([][]ARGBf, columns)
	for i := 0; i < columns; i++ {
		display[i] = make([]ARGBf, src.Buckets)
	}
	for h := 0; h < numHues; h++ {
		for v := 0; v < numValues; v++ {
			r, g, b := hsluv.HsluvToRGB(float64(h), float64(100), 100*float64(v)/256)
			r *= r
			g *= g
			b *= b
			clut[h][v] = ARGBf{0, float32(r), float32(g), float32(b)}
		}
	}
	return &renderer{
		params:  params,
		columns: columns,
		rows:    src.Buckets,
		src:     src,
		display: display,
		warp:    make([]float32, src.Buckets),
		scale:   make([]float32, columns),
		mirror:  mirror,
	}
}

type renderValues struct {
	img   [][]ARGBf
	warp  []float32
	scale []float32
	bass  float32
}

func (r *renderer) Render(done, request chan struct{}) chan *renderValues {
	out := make(chan *renderValues)

	// set up a goroutine to render a frame only when requested
	go func() {
		for {
			select {
			case <-done:
				return
			case <-request:
				r.render()
				out <- &renderValues{
					r.display,
					r.warp,
					r.scale,
					r.bass,
				}
			}
		}
	}()

	return out
}

func (r *renderer) render() {
	r.renderCount++
	// if r.params.Debug && r.renderCount%100 == 0 {
	// 	diff := time.Now().Sub(r.lastRender)
	// 	m := map[string]interface{}{
	// 		"fps":  diff / 100.0,
	// 		"amp":  r.src.Amplitude[0],
	// 		"pha":  r.src.Energy,
	// 		"diff": r.src.Diff,
	// 	}
	// 	bs, err := json.Marshal(m) //Indent(m, "", "  ")
	// 	if err != nil {
	// 		fmt.Printf("%#v", r.src.Drivers)
	// 		panic(err)
	// 	}
	// 	fmt.Println(string(bs))
	// 	r.lastRender = time.Now()
	// }
	hl := r.columns
	for i := 0; i < hl; i++ {
		col := r.renderColumn(i)
		for j, c := range col {
			v := float64(i) / float64(hl)
			v = 1 - v*v
			c.A *= float32(v)
			c.R *= float32(v)
			c.G *= float32(v)
			c.B *= float32(v)
			// if r.mirror {
			// 	r.display.SetRGBA(hl+i, r.rows+j, c)
			// 	r.display.SetRGBA(hl+i, r.rows-1-j, c)
			// 	r.display.SetRGBA(hl-1-i, r.rows+j, c)
			// 	r.display.SetRGBA(hl-1-i, r.rows-1-j, c)
			// } else {
			// 	r.display.SetRGBA(hl+i, r.rows-j-1, c)
			// 	r.display.SetRGBA(hl-1-i, r.rows-j-1, c)
			// }
			r.display[i][j] = c
		}
	}
	ln := len(r.src.Diff)
	// if r.mirror {
	// 	ln *= 2
	// }
	for i := 0; i < ln; i++ {
		var ix int
		if i < len(r.src.Diff) {
			ix = i
		} else {
			ix = ln - 1 - i
		}
		d := r.src.Diff[ix]
		r.warp[i] = float32(d) //float32(r.params.WarpOffset + r.params.WarpScale*math.Abs(d))
	}
	for i := 1; i < ln-1; i++ {
		wl := r.warp[i-1]
		wr := r.warp[i+1]
		w := r.warp[i]
		// dl := w - wl
		// sl := float32(1.0)
		// if dl < 0 {sl = -1}
		// dr :=  w - wr
		// sr := float32(1.0)
		// if dr < 0 {sr = -1}
		// r.warp[i] += .01 * (sl * wl * wl + sr * wr * wr)
		r.warp[i] = (wl + wr + w) / 3
	}

	r.bass = float32(r.src.Bass)

	sc := len(r.src.Amplitude) / r.columns
	for i := 0; i < r.columns; i++ {
		var s float64
		for j := 0; j < sc; j++ {
			for k, v := range r.src.Amplitude[sc*i+j] {
				// todo: sum with linear scaling to target bass?
				s += r.src.Scales[k] * (v - 1)
			}
		}
		s /= float64(sc)
		s /= float64(len(r.src.Amplitude[sc*i]))
		// v := float64(i) / float64(r.columns)
		// s *= (1 - v*v)
		r.scale[i] = float32(s)
	}
}

func (r *renderer) renderColumn(col int) []ARGBf {

	amp := r.src.Amplitude[0]
	if r.params.Mode == fs.AnimateMode {
		amp = r.src.Amplitude[col]
	}

	phase := r.src.Energy
	ws := 2.0 * math.Pi / float64(r.params.Period)
	phi := ws * float64(col)

	colors := make([]ARGBf, r.rows)

	for i, ph := range phase {
		val := r.src.Scales[i] * (amp[i] - 1)
		//colors[i] = getRGB(d.params, amp[i], ph, phi)
		colors[i] = getHSV(r.params, val, ph, phi)
	}

	return colors
}

type ARGBf struct {
	A float32
	R float32
	G float32
	B float32
}

const numValues = 256
const numHues = 360

var clut [numHues][numValues]ARGBf

func getHSV(params *fs.Parameters, amp, ph, phi float64) ARGBf {
	// br := params.Brightness
	// gbr := params.GlobalBrightness
	ss := params.SaturationScale
	so := params.SaturationOffset
	vo1 := params.ValueOffset1
	vo2 := params.ValueOffset2
	alpha := params.Alpha
	ao := params.AlphaOffset

	hue := math.Mod((ph+phi)*180/math.Pi, 360)
	if hue < 0 {
		hue += 360
	}
	// sat := fs.Sigmoid(br + so + amp)
	val := ss*fs.Sigmoid(vo1*amp+vo2) + so
	al := fs.Sigmoid(alpha*amp + ao)

	vi := int(numValues * val)
	if vi >= numValues {
		vi = numValues - 1
	}
	if vi < 0 {
		vi = 0
	}

	// r, g, b := colorful.Hsv(hue, sat, val).RGB255()
	// rf, gf, bf := hsluv.HsluvToRGB(hue, sat*100, val*75)
	// r, g, b := uint8(255*rf), uint8(255*gf), uint8(255*bf)
	// fmt.Println(r, g, b)
	// fmt.Println(int(hue), vi)
	c := clut[int(hue)][vi]
	c.A = float32(al)

	return c
}

type gridPoint struct {
	x    float32
	y    float32
	srcX int
	srcY int
}

func (r *renderer) gridRender3(g skgrid.Grid, frameRate int, done chan struct{}) {
	defer g.Close()
	defer close(done)

	rect := g.Rect()
	displayWidth := rect.Dx()
	displayHeight := rect.Dy()

	render := make(chan struct{})
	frames := r.Render(done, render)

	columns := r.src.Frames
	rows := r.src.Buckets
	// aspect := float32(9.0 / 13.0)

	// initialize points in the top right quadrant due to symmetry
	points := make([]gridPoint, columns*rows)
	for x := 0; x < columns; x++ {
		for y := 0; y < rows; y++ {
			xf := float32(x) / float32(columns)
			yf := float32(y) / float32(rows) // * aspect
			points[x+y*columns] = gridPoint{
				x:    xf,
				y:    yf,
				srcX: columns - 1 - x,
				srcY: rows - 1 - y,
			}
		}
	}

	// 0,0 -> dw/2,dh/2
	// 1,1 -> dw,0
	// -1,-1 -> 0,dh
	// 1,-1 -> dw,dh
	// -1,1 -> 0,0

	getDisplayXY := func(g gridPoint) (int, int) {
		x := int(g.x*float32(displayWidth)/2 + 0.5)
		if x < 0 {
			x = 0
		}
		if x >= displayWidth/2 {
			x = displayWidth/2 - 1
		}
		y := int(g.y*float32(displayHeight)/2 + 0.5)
		if y < 0 {
			y = 0
		}
		if y >= displayHeight/2 {
			y = displayHeight/2 - 1
		}
		return x, y
	}

	applyWarp := func(g gridPoint, w, s float32) gridPoint {
		if g.x <= 0 {
			g.x = float32(math.Pow(float64(g.x+1), float64(w)) - 1)
		} else {
			g.x = float32(1 - math.Pow(float64(1-g.x), float64(w)))
		}
		if g.y <= 0 {
			s = (1 + g.y/2) * s
			g.y = float32(math.Pow(float64(1+g.y), float64(s)) - 1)
		} else {
			// ss := s
			s = (1 - g.y/2) * s
			// yy := g.y
			g.y = float32(1 - math.Pow(float64(1-g.y), float64(s)))
			// if g.y < 0 {
			// 	fmt.Println("wtf", yy, g.y, s, ss)
			// }
		}
		return g
	}

	delay := time.Second / time.Duration(frameRate)
	ticker := time.NewTicker(delay)

	var renderLoopTime time.Duration
	var writeLoopTime time.Duration
	frameCount := 0
	if r.params.Debug {
		go func() {
			t := time.NewTicker(10 * time.Second)
			for range t.C {
				log.Println("[Info] FPS:\t", frameCount/10)
				log.Println("[Info] Render:\t", renderLoopTime)
				log.Println("[Info] Write:\t", writeLoopTime)
				frameCount = 0
			}
		}()
	}

	// bufferAt := func(b []int, x, y int) (int, int, int, int) {
	// 	idx := x + displayWidth/2*y
	// 	idx *= 4
	// 	return b[idx], b[idx+1], b[idx+2], b[idx+3]
	// }
	// bufferSet := func(buf []int, x, y, a, r, g, b int) {
	// 	idx := x + displayWidth/2*y
	// 	idx *= 4
	// 	buf[idx] = a
	// 	buf[idx+1] = r
	// 	buf[idx+2] = g
	// 	buf[idx+3] = b
	// }
	const nThreads = 4

	// buffer := make([][]ARGBf, displayWidth/2)
	// for i := 0; i < displayWidth/2; i++ {
	// 	buffer[i] = make([]ARGBf, displayHeight/2)
	// }
	type frameReadyValues struct {
		frame  *renderValues
		buffer [][]ARGBf
	}
	frameReady := make(chan frameReadyValues, 2)
	defer close(frameReady)
	writeReady := make(chan [][]ARGBf, 2)
	defer close(writeReady)

	go func() {
		for {
			frameCount++
			<-ticker.C
			render <- struct{}{}
			frame := <-frames
			if frame == nil {
				break
			}

			//g.Fill(color.RGBA{0, 0, 0, 0})

			// for i := 0; i < displayWidth/2; i++ {
			// 	for j := 0; j < displayHeight/2; j++ {
			// 		buffer[i][j] = ARGBf{0, 0, 0, 0}
			// 	}
			// }
			buffer := make([][]ARGBf, displayWidth/2)
			for i := 0; i < displayWidth/2; i++ {
				buffer[i] = make([]ARGBf, displayHeight/2)
			}

			frameReady <- frameReadyValues{frame, buffer}
		}
	}()

	go func() {
		for fv := range frameReady {
			if fv.frame == nil {
				break
			}
			frame := fv.frame
			buffer := fv.buffer

			now := time.Now()

			var wg = new(sync.WaitGroup)

			for i := 0; i < nThreads; i++ {
				start := i * len(points) / nThreads
				end := (i + 1) * len(points) / nThreads
				wg.Add(1)
				go func() {
					defer wg.Done()

					// points := make([]gridPoint, len(pointSrc))
					for i := start; i < end; i++ {
						pbase := points[i]
						wi := rows - 1 - pbase.srcY
						si := columns - 1 - pbase.srcX

						// fmt.Println("g warp scale", g, len(frame.warp), len(frame.scale))
						warp := fs.DefaultParameters.WarpScale * float64(frame.warp[wi])
						warp += fs.DefaultParameters.WarpOffset
						scale := fs.DefaultParameters.Scale * float64(frame.scale[si])
						scale += fs.DefaultParameters.ScaleOffset
						p := applyWarp(pbase, float32(warp), float32(scale))
						x, y := getDisplayXY(p)
						// fmt.Println("g, p, x, y", g, p, x, y)

						c1 := frame.img[si][wi]
						c2 := buffer[x][y]

						// wc1 := (c1.A + c1.B + c1.G + c1.R) / 4
						// wc2 := (c2.A + c2.B + c2.G + c2.R) / 4
						// sw := (wc1 + wc2)
						// sc1 := wc1 / sw
						// sc2 := wc2 / sw

						sc1 := c1.A
						sc2 := c2.A

						a := c1.A*sc1 + c2.A*sc2
						if a > 1.0 {
							a = 1.0
						}
						r := c1.R*sc1 + c2.R*sc2
						if r > 1.0 {
							r = 1.0
						}
						g := c1.G*sc1 + c2.G*sc2
						if g > 1.0 {
							g = 1.0
						}
						b := c1.B*sc1 + c2.B*sc2
						if b > 1.0 {
							b = 1.0
						}

						buffer[x][y] = ARGBf{a, r, g, b}

						// buffer.SetRGBA(x, y, c1)
						//buffer.SetRGBA(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
						// bufferSet(buffer, x, y, a, r, g, b)
					}
				}()
			}

			wg.Wait()

			renderLoopTime = time.Now().Sub(now)

			writeReady <- buffer
		}
	}()
	// max := 255
	// for _, v := range buffer {
	// 	if v > max {
	// 		max = v
	// 	}
	// }
	// sc := 255.0 / float32(max)
	// for i, v := range buffer {
	// 	buffer[i] = int(float32(v) * sc)
	// }

	xo := displayWidth / 2
	yo := displayHeight / 2

	writePixels := []func(xo, xt, yo, yt int, c color.RGBA){
		func(xo, xt, yo, yt int, c color.RGBA) { g.Pixel(xo+xt, yo+yt, c) },
		func(xo, xt, yo, yt int, c color.RGBA) { g.Pixel(xo-xt-1, yo+yt, c) },
		func(xo, xt, yo, yt int, c color.RGBA) { g.Pixel(xo+xt, yo-yt-1, c) },
		func(xo, xt, yo, yt int, c color.RGBA) { g.Pixel(xo-xt-1, yo-yt-1, c) },
	}

	// go func() {
	for buffer := range writeReady {
		if buffer == nil {
			break
		}
		now := time.Now()

		var wg sync.WaitGroup
		for _, writePixel := range writePixels {
			writePixel := writePixel
			wg.Add(1)
			go func() {
				defer wg.Done()
				for x := 0; x < displayWidth/2; x++ {
					for y := 0; y < displayHeight/2; y++ {
						cf := buffer[x][y]
						c := color.RGBA{uint8(255 * cf.R), uint8(255 * cf.G), uint8(255 * cf.B), uint8(255 * cf.A)}
						writePixel(xo, x, yo, y, c)
					}
				}
			}()
		}
		wg.Wait()

		writeLoopTime = time.Now().Sub(now)

		if err := g.Show(); err != nil {
			log.Println("grid error!", err)
			break
		}
	}
	// }()

}

func (r *renderer) colorTest(g skgrid.Grid, frameRate int, done chan struct{}) {
	defer g.Close()
	defer close(done)

	rect := g.Rect()
	displayWidth := rect.Dx()
	displayHeight := rect.Dy()

	delay := time.Second / time.Duration(frameRate)
	ticker := time.NewTicker(delay)

	phase := 0.0

	for {
		<-ticker.C

		phase += 1.0
		phase = math.Mod(phase, 3*float64(displayWidth))

		for x := 0; x < displayWidth; x++ {
			for y := 0; y < displayHeight; y++ {

				// h := math.Mod(phase+float64(x), 360)
				// v := 60 * float64(y) / float64(displayHeight)
				// r, gr, b := hsluv.HpluvToRGB(h, 100, v)
				r := (float64(x)) / float64(displayWidth)
				gr := 0
				b := 0

				c := color.RGBA{uint8(255 * r), uint8(255 * gr), uint8(255 * b), 0}

				g.Pixel(x, y, c)
			}
		}
		if err := g.Show(); err != nil {
			log.Println("grid error!", err)
			break
		}
	}
}
