package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"

	"github.com/peragwin/vuzicgo/audio"
	"github.com/peragwin/vuzicgo/audio/fft"
	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
	"github.com/peragwin/vuzicgo/gfx/skgrid"
	"github.com/peragwin/vuzicgo/gfx/warpgrid"
)

const (
	sampleFrame = 64
	sampleRate  = 44100

	textureMode = gl.LINEAR
)

var (
	width  = flag.Int("width", 1200, "width of window")
	height = flag.Int("height", 800, "height of window")

	buckets = flag.Int("buckets", 64, "number of frequency buckets")
	columns = flag.Int("columns", 16, "number of cells per row")
	mirror  = flag.Bool("mirror", false, "display mirrored rows")

	headless  = flag.Bool("headless", false, "run without initializing OpenGL display")
	mode      = flag.Int("mode", fs.NormalMode, "which mode: 0=Normal, 1=Animate")
	remote    = flag.String("remote", "", "ip:port of remote grid")
	flRemote  = flag.String("fl-remote", "", "ip:port of flaschen grid")
	pilocal   = flag.Bool("pilocal", false, "use raspberry pi's SPI output")
	frameRate = flag.Int("frame-rate", 30,
		"frame rate to target when rendering to something other than opengl")
	lowLatency  = flag.Bool("low-latency", false, "use lower audio latency")
	audioDevice = flag.Int("audio-device", -1, "select a specific audio device")
	listDevices = flag.Bool("list-devices", false, "display a list of audio devices")

	frameSize = flag.Int("frame-size", 1024,
		"size of process frames. must be multiple of 256")

	httpDir = flag.String("http-dir", "./client/build", "where to host static client gui files")
)

func initGfx(done chan struct{}) *warpgrid.Grid {
	runtime.LockOSThread()

	rows := *buckets
	if *mirror {
		rows *= 2
	}
	g, err := warpgrid.NewGrid(done, &warpgrid.Config{
		Rows: rows, Columns: *columns,
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

	fsize := *frameSize
	if fsize%sampleFrame != 0 {
		log.Fatalf("frame size must be multiple of %d", sampleFrame)
	}

	render := make(chan struct{})
	defer close(render)
	done := make(chan struct{})
	defer close(done)

	var g *warpgrid.Grid
	if !*headless {
		// The graphics have to be the first thing we initialize on macOS; I'm guessing it's
		// because of the syscall that binds it to the main thread.
		g = initGfx(done)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source, errc := audio.NewSource(ctx, &audio.Config{
		BlockSize:    sampleFrame,
		SampleRate:   sampleRate,
		Channels:     1,
		LowLatency:   *lowLatency,
		DeviceID:     *audioDevice,
		PrintDevices: *listDevices,
	})

	// watch for errors
	go func() {
		defer close(done)
		err := <-errc
		log.Fatal(err)
	}()

	source64 := audio.Buffer(done, source, fsize)

	fftProc := fft.NewFFTProcessor(sampleRate, fsize)
	fftOut := fftProc.Process(done, source64)

	specProc := new(fft.PowerSpectrumProcessor)
	specOut := specProc.Process(done, fftOut)

	fs.DefaultParameters.Mode = *mode
	fs.DefaultParameters.Period = 3 * *columns / 2
	f := fs.NewFrequencySensor(&fs.Config{
		Columns:    *columns,
		Buckets:    *buckets,
		SampleRate: sampleRate,
		Parameters: fs.DefaultParameters,
	})
	fsOut := f.Process(done, specOut)
	// this output isn't needed here so throw it away
	// XXX added fsOut to render input
	go func() {
		for {
			<-fsOut
		}
	}()
	// XXX added fsOut arg
	rndr := newRenderer(*columns, *mirror, fs.DefaultParameters, f)

	if !*headless {
		frames := rndr.Render(done, render)
		g.SetRenderFunc(func(g *warpgrid.Grid) {
			render <- struct{}{}
			rv := <-frames
			g.SetImage(rv.img)

			sos := float32(fs.DefaultParameters.ScaleOffset)
			ss := float32(fs.DefaultParameters.Scale)
			h := len(rv.scale) / 2
			// fmt.Println("")
			for i := 0; i < h; i++ {
				sss := 1 - math.Abs(float64(h)-float64(i)/2)/float64(h)
				// fmt.Println(sss)
				s := sos + ss*float32(sss)*rv.scale[i]
				g.SetScale(h+i, s)
				g.SetScale(h-i-1, s)
			}

			wos := float32(fs.DefaultParameters.WarpOffset)
			ws := float32(fs.DefaultParameters.WarpScale)
			h = len(rv.warp) / 2
			for i, w := range rv.warp {
				wss := 1 - math.Abs(float64(h)-float64(i)/2)/float64(h)
				w = wos + float32(wss)*ws*w
				g.SetWarp(i, w)
				if *mirror {
					g.SetWarp(2*len(rv.warp)-1-i, w)
				}
			}
		})
	}

	// If a remote is passed try to stream to it. If we lose the connection, try again
	// after 10 seconds to reestablish a connection.
	if *remote != "" {
		go func() {
			delay := time.NewTicker(10 * time.Second)
			for {
				if skRem, err := skgrid.NewRemote(*remote); err != nil {
					log.Println("[ERROR] could not connect to remote skgrid controller. " +
						"Retrying in 10 seconds...")
					<-delay.C
				} else {
					grid, err := skgrid.NewGrid(16, 60, "skgrid", map[string]interface{}{
						"transpose": true,
						"driver":    skRem,
					})
					if err != nil {
						panic(err)
					}
					done := make(chan struct{})
					go rndr.gridRender2(grid, *frameRate, done)
					<-done
				}
			}
		}()
	}

	if *flRemote != "" {
		go func() {
			grid, err := skgrid.NewGrid(60, 16, "flaschen", map[string]interface{}{
				"layer":  0,
				"remote": *flRemote,
			})
			if err != nil {
				panic(err)
			}
			done := make(chan struct{})
			go rndr.gridRender2(grid, *frameRate, done)
			<-done
		}()
	}

	if *pilocal {
		go func() {
			if pi, err := skgrid.NewPiLocal(8e6); err != nil {
				log.Println("[ERROR] could not initialize raspberry pi:", err)
			} else {
				grid, err := skgrid.NewGrid(60, 16, "skgrid", map[string]interface{}{
					//"transpose": true,
					"driver": pi,
				})
				if err != nil {
					panic(err)
				}
				done := make(chan struct{})
				go rndr.gridRender2(grid, *frameRate, done)
				<-done
			}
		}()
	}

	go func() {
		http.HandleFunc("/api/v1/graphql", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			log.Println(query)
			res := f.Query(query, nil)
			json.NewEncoder(w).Encode(res)
		})

		http.HandleFunc("/api/v2/graphql", func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			apolloQuery := make(map[string]interface{})
			if err := json.Unmarshal(body, &apolloQuery); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Println(apolloQuery)

			query := apolloQuery["query"].(string)
			variables := apolloQuery["variables"].(map[string]interface{})
			res := f.Query(query, variables)
			if len(res.Errors) > 0 {
				for _, err := range res.Errors {
					log.Println("[ERROR]", err)
				}
			}
			json.NewEncoder(w).Encode(res)
		})

		http.Handle("/", http.FileServer(http.Dir(*httpDir)))

		http.ListenAndServe(":8080", nil)
	}()

	if !*headless {
		g.Start()
	} else {
		<-done
	}
}
