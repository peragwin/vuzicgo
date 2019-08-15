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
	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
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

	mode = flag.Int("mode", fs.NormalMode, "which mode: 0=Normal, 1=Animate")
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

	sources := audio.StreamMultiplier(done, source64, numSources)
	drivers := make([]chan *fs.Drivers, numSources)
	for i, src := range sources {

		fftProc := fft.NewFFTProcessor(sampleRate, frameSize)
		fftOut := fftProc.Process(done, src)

		specProc := new(fft.PowerSpectrumProcessor)
		specOut := specProc.Process(done, fftOut)

		fs.DefaultParameters.Mode = *mode
		fs.DefaultParameters.Period = 3 * *columns / 2
		cfg := fs.NewConfig(&fs.Config{
			Columns:    1,
			Buckets:    *buckets,
			SampleRate: sampleRate,
			Parameters: fs.DefaultParameters,
		})
		f := fs.NewFrequencySensor(cfg)
		fsOut := f.Process(done, specOut)
	}
	// this output isn't needed here so throw it away
	go func() {
		for {
			<-fsOut
		}
	}()

	rndr := newRenderer(*columns, fs.DefaultParameters, f)
	frames := rndr.Render(done, render)

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
				Schema:        cfg.Schema,
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