package skgrid

import (
	"errors"
	"image"
	"image/color"

	"github.com/peragwin/vuzicgo/gfx/flaschen-taschen/api/go"
)

type skGrid struct {
	Width     int
	Height    int
	buffer    []byte
	driver    Driver
	transpose bool
}

type Driver interface {
	Send([]byte) error
	Close() error
}

type Grid interface {
	Rect() image.Rectangle
	Pixel(x, y int, col color.RGBA)
	Show() error
	Close() error
}

type initFunc func(int, int, map[string]interface{}) (Grid, error)

var drivers = map[string]initFunc{
	"skgrid":   newSkGrid,
	"flaschen": newFlaschen,
}

// NewGrid creates a new Grid display object using the given driver and options
func NewGrid(width, height int, driver string, opts map[string]interface{}) (Grid, error) {
	init, ok := drivers[driver]
	if !ok {
		return nil, errors.New("unknown grid driver: " + driver)
	}
	return init(width, height, opts)
}

func newSkGrid(width, height int, opts map[string]interface{}) (Grid, error) {
	ln := width * height
	endframe := make([]byte, 6+ln/16)
	endframe[0] = 0xff
	buffer := make([]byte, 4*(ln+1))
	drv := opts["driver"]
	if drv == nil {
		return nil, errors.New("skgrid driver missing required option: 'driver'")
	}
	driver, ok := drv.(Driver)
	if !ok {
		return nil, errors.New("skgrid option 'driver' is not a `skgrid.Driver`")
	}
	var transpose bool
	trans, ok := opts["transpose"]
	if ok {
		tp, ok := trans.(bool)
		if !ok {
			return nil, errors.New("skgrid option 'transpose' is not a `bool`")
		}
		transpose = tp
	}
	return &skGrid{
		Width:     width,
		Height:    height,
		transpose: transpose,
		buffer:    append(buffer, endframe...),
		driver:    driver,
	}, nil
}

func (s *skGrid) Rect() image.Rectangle {
	if s.transpose {
		return image.Rect(0, 0, s.Height, s.Width)
	}
	return image.Rect(0, 0, s.Width, s.Height)
}

func (s *skGrid) SetBuffer(idx int, col color.RGBA) {
	n := 4*idx + 4
	s.buffer[n] = 0xe0 | col.A
	s.buffer[n+1] = col.B
	s.buffer[n+2] = col.G
	s.buffer[n+3] = col.R
}

func (s *skGrid) Fill(col color.RGBA) {
	for i := 0; i < s.Width*s.Height; i++ {
		s.SetBuffer(i, col)
	}
}

func (s *skGrid) Pixel(x, y int, col color.RGBA) {
	// adjust B/G channels to match R
	col.G /= 2
	col.B /= 2
	// limit alpha to 32 (256/8)
	col.A = uint8(float64(col.A)/8 + 0.5)

	// skgrid is wired like a snake so we have to flip every other column
	//if s.transpose {
		if x%2 == 1 {
			y = s.Height - 1 - y
		}
	//} else {
	//	if y%2 == 1 {
	//		x = s.Height - 1 - x
	//	}
	//}
	var idx int
	if s.transpose {
		idx = s.Height*x + y
	} else {
		idx = s.Width*y + x // <- glitch. correct is s.Height*y + x
	}
	s.SetBuffer(idx, col)
}

func (s *skGrid) Show() error {
	return s.driver.Send(s.buffer) //[4:4+4*(s.Width*s.Height)])
}

func (s *skGrid) Close() error {
	return s.driver.Close()
}

func newFlaschen(width, height int, opts map[string]interface{}) (Grid, error) {
	lay, ok := opts["layer"]
	if !ok {
		return nil, errors.New("flashen driver missing 'layer' option")
	}
	layer, ok := lay.(int)
	if !ok {
		return nil, errors.New("flashen option 'layer' is not an `int`")
	}
	rem, ok := opts["remote"]
	if !ok {
		return nil, errors.New("flashen driver missing 'remote' option")
	}
	remote, ok := rem.(string)
	if !ok {
		return nil, errors.New("flashen option 'remote' is not a `string`")
	}
	return flaschen.NewFlaschen(width, height, layer, remote)
}
