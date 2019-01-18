package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"

	"github.com/peragwin/vuzicgo/gfx/video"
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

	//mode   = flag.Int("mode", fs.NormalMode, "which mode: 0=Normal, 1=Animate")
	remote = flag.String("remote", "", "ip:port of remote grid")
)

func initGfx(done chan struct{}) *video.Grid {
	runtime.LockOSThread()

	g, err := video.NewGrid(done, &video.Config{
		Width: *width, Height: *height,
		Title:       "LED Display 1",
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

	{
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// source, errc := audio.NewSource(ctx, &audio.Config{
	// 	BlockSize:  frameSize,
	// 	SampleRate: sampleRate,
	// 	Channels:   1,
	// })

	// // watch for errors
	// go func() {
	// 	defer close(done)
	// 	err := <-errc
	// 	log.Fatal(err)
	// }()

	// source64 := audio.Buffer(done, source)

	// fftProc := fft.NewFFTProcessor(sampleRate, frameSize)
	// fftOut := fftProc.Process(done, source64)

	// specProc := new(fft.PowerSpectrumProcessor)
	// specOut := specProc.Process(done, fftOut)

	// fs.DefaultParameters.Mode = *mode
	// fs.DefaultParameters.Period = 3 * *columns / 2
	// f := fs.NewFrequencySensor(&fs.Config{
	// 	Columns:    *columns,
	// 	Buckets:    *buckets,
	// 	SampleRate: sampleRate,
	// 	Parameters: fs.DefaultParameters,
	// })
	// fsOut := f.Process(done, specOut)
	// // this output isn't needed here so throw it away
	// go func() {
	// 	for {
	// 		<-fsOut
	// 	}
	// }()
	}

	src := &rgbSrc{
		NewOscillator(XYVal{1.0 / 401, 0}, XYVal{0,0}, XYVal{0, 0}, SawWave),
		NewOscillator(XYVal{1.0/400,0}, XYVal{0,0}, XYVal{0, 0}, SawWave),
		NewOscillator(XYVal{1.0/399,0}, XYVal{0,0}, XYVal{0, 0}, SawWave),
	}

	rndr := newRenderer(&params{*height, *width}, src)
	frames := rndr.Render(done, render)
	// skframes := make(chan *renderValues)

	g.SetRenderFunc(func(g *video.Grid) {
		render <- struct{}{}
		//_ = <-frames
		rv := <-frames
		g.SetImage(rv.img)
		//fmt.Println(rv.img.At(0, 0))
	})

	g.Start()
}

	// if *remote != "" {
	// 	if skRem, err := skgrid.NewRemote(*remote); err != nil {
	// 		log.Println("[ERROR] could not connect to remote skgrid controller")
	// 	} else {
	// 		go func() {
	// 			defer skRem.Close()
	// 			skGrid := skgrid.NewGrid(60, 16, skRem)

	// 			render := make(chan struct{})
	// 			frames := rndr.Render(done, render)

	// 			xinput := make([]float64, skGrid.Height/2)
	// 			for i := range xinput {
	// 				xinput[i] = 1 - 2*float64(i)/float64(skGrid.Height)
	// 			}
	// 			warpIndices := func(warp float64) []int {
	// 				ws := fs.DefaultParameters.WarpScale
	// 				wo := fs.DefaultParameters.WarpOffset
	// 				warp = wo + ws*math.Abs(warp)
	// 				wv := make([]int, skGrid.Height)
	// 				b := skGrid.Height / 2
	// 				for i := 0; i < skGrid.Height/2; i++ {
	// 					scaled := 1 - math.Pow(xinput[i], warp)
	// 					xp := scaled * float64(skGrid.Height/2)
	// 					wv[b-i] = b - int(xp+0.5)
	// 					wv[b+i] = b + int(xp+0.5)
	// 				}
	// 				return wv
	// 			}
	// 			scaleIndex := func(y int, scale float64) int {
	// 				yi := 1 - float64(y)/float64(skGrid.Width)
	// 				warp := fs.DefaultParameters.Scale * scale
	// 				scaled := 1 - math.Pow(yi, warp)
	// 				return int((float64(skGrid.Width) * scaled) + .5)
	// 			}

	// 			ticker := time.NewTicker(30000 * time.Microsecond)

	// 			for {
	// 				<-ticker.C
	// 				render <- struct{}{}
	// 				frame := <-frames
	// 				if frame == nil {
	// 					break
	// 				}
	// 				img := resize.Resize(uint(skGrid.Height), uint(skGrid.Width),
	// 					frame.img, resize.NearestNeighbor)

	// 				wvs := make([][]int, skGrid.Width)
	// 				yv := make([]int, skGrid.Width)
	// 				for i := range wvs {
	// 					wvs[i] = warpIndices(float64(frame.warp[i]))
	// 					yv[i] = scaleIndex(i, float64(frame.scale))
	// 				}

	// 				for y := 0; y < 60; y++ {
	// 					for x := 0; x < 16; x++ {
	// 						px := img.At(x, skGrid.Width-1-y).(color.RGBA)
	// 						px.G /= 2
	// 						px.B /= 2
	// 						px.A = uint8(float64(px.A)/8 + 0.5)

	// 						xstart := 0
	// 						if x != 0 {
	// 							xstart = wvs[y][x]
	// 						}
	// 						xend := skGrid.Height
	// 						if x != skGrid.Height-1 {
	// 							//log.Println(y, x, wvs[y])
	// 							xend = wvs[y][x+1]
	// 						}

	// 						ystart := yv[y]
	// 						yend := skGrid.Width
	// 						if y != skGrid.Width-1 {
	// 							yend = yv[y+1]
	// 						}

	// 						if xend > xstart && yend > ystart {
	// 							for j := ystart; j < yend; j++ {
	// 								for k := xstart; k < xend; k++ {
	// 									skGrid.Pixel(j, k, px)
	// 								}
	// 							}
	// 						}
	// 					}
	// 				}

	// 				if err := skGrid.Show(); err != nil {
	// 					log.Println("sk grid error!", err)
	// 				}
	// 			}
	// 		}()
	// 	}
	// }

	// go func() {
	// 	http.HandleFunc("/api/v1/graphql", func(w http.ResponseWriter, r *http.Request) {
	// 		query := r.URL.Query().Get("query")
	// 		log.Println(query)
	// 		res := f.Query(query, nil)
	// 		json.NewEncoder(w).Encode(res)
	// 	})

	// 	http.HandleFunc("/api/v2/graphql", func(w http.ResponseWriter, r *http.Request) {
	// 		body, err := ioutil.ReadAll(r.Body)
	// 		if err != nil {
	// 			http.Error(w, err.Error(), http.StatusInternalServerError)
	// 			return
	// 		}

	// 		apolloQuery := make(map[string]interface{})
	// 		if err := json.Unmarshal(body, &apolloQuery); err != nil {
	// 			http.Error(w, err.Error(), http.StatusInternalServerError)
	// 			return
	// 		}
	// 		log.Println(apolloQuery)

	// 		query := apolloQuery["query"].(string)
	// 		variables := apolloQuery["variables"].(map[string]interface{})
	// 		res := f.Query(query, variables)
	// 		if len(res.Errors) > 0 {
	// 			for _, err := range res.Errors {
	// 				log.Println("[ERROR]", err)
	// 			}
	// 		}
	// 		json.NewEncoder(w).Encode(res)
	// 	})

	// 	http.ListenAndServe(":8080", nil)
	// }()

// 	g.Start()
// }
