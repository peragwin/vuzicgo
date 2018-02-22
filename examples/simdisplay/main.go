package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/graphql-go/graphql"
	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/audio/fft"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	frameSize  = 1024
	sampleRate = 44100

	textureMode = gl.LINEAR
)

var (
	width  = flag.Int("width", 1200, "width of window")
	height = flag.Int("height", 800, "height of window")

	buckets = flag.Int("buckets", 64, "number of frequency buckets")
	columns = flag.Int("columns", 16, "number of cells per row")

	mode = flag.Int("mode", NormalMode, "which mode: 0=Normal, 1=Animate")
)

func initGfx(done chan struct{}) *grid.Grid {
	runtime.LockOSThread()

	g, err := grid.NewGrid(done, &grid.Config{
		Rows: *buckets, Columns: *columns,
		Width: *width, Height: *height,
		Title:       "Sim LED Display",
		TextureMode: textureMode,
	})
	if err != nil {
		log.Fatal("error creating display:", err)
	}
	return g
}

func main() {
	flag.Parse()

	render := make(chan struct{})
	defer close(render)
	done := make(chan struct{})
	defer close(done)

	// The graphics have to be the first thing we initialize on macOS; I'm guessing it's
	// because of the syscall that binds it to the main thread.
	g := initGfx(done)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source, errc := audio.NewSource(ctx, &audio.Config{
		BlockSize:  frameSize,
		SampleRate: sampleRate,
		Channels:   1,
	})

	// watch for errors
	go func() {
		defer close(done)
		err := <-errc
		log.Fatal(err)
	}()

	source64 := audio.Buffer(done, source)

	fftProc := fft.NewFFTProcessor(sampleRate, frameSize)
	fftOut := fftProc.Process(done, source64)

	specProc := new(fft.PowerSpectrumProcessor)
	specOut := specProc.Process(done, fftOut)

	defaultParameters.Mode = *mode
	defaultParameters.Period = 3 * *columns / 2
	cfg := NewConfig(&Config{
		Columns:    *columns,
		Buckets:    *buckets,
		SampleRate: sampleRate,
		Parameters: defaultParameters,
	})
	display := NewDisplay(cfg)
	frames := display.Process(done, specOut, render)

	g.SetRenderFunc(func(g *grid.Grid) {
		render <- struct{}{}
		img := <-frames
		g.SetImage(img)
	})

	go func() {
		http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			log.Println(query)
			res := graphql.Do(graphql.Params{
				Schema:        cfg.schema,
				RequestString: query,
			})
			json.NewEncoder(w).Encode(res)
		})
		// http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {

		// })
		http.ListenAndServe(":8080", nil)
	}()

	g.Start()
}
