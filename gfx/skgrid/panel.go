package skgrid

import (
	"image"
	"image/color"

	"github.com/mcuadros/go-rpi-rgb-led-matrix"
)

type panel struct {
	m     rgbmatrix.Matrix
	c     *rgbmatrix.Canvas
	w     int
	h     int
	close chan struct{}
}

// newPanel returns a new led panel driver
func newPanel(w, h int, opts map[string]interface{}) (Grid, error) {
	p := &panel{
		w: w, h: h,
		close: make(chan struct{}),
	}
	var panelType string
	panelTypeI, ok := opts["paneltype"]
	if ok {
		panelType = panelTypeI.(string)
	}

	cfg := &rgbmatrix.HardwareConfig{
		Rows:              h,
		Cols:              w,
		ChainLength:       1,
		Parallel:          1,
		PWMBits:           11,
		Brightness:        100,
		PWMLSBNanoseconds: 50,
		ScanMode:          rgbmatrix.Progressive,
		// ShowRefreshRate:   true,
		PanelType:	   panelType,
	}

	m, err := rgbmatrix.NewRGBLedMatrix(cfg)
	if err != nil {
		return nil, err
	}
	p.m = m

	c := rgbmatrix.NewCanvas(m)
	p.c = c

	return p, nil
}

func (p *panel) Rect() image.Rectangle {
	return image.Rect(0, 0, p.w, p.h)
}

func (p *panel) Pixel(x, y int, col color.RGBA) {
	p.c.Set(x, y, col)
}

func (p *panel) Show() error {
	return p.m.Render()
}

func (p *panel) Close() error {
	return p.c.Close()
}

func (p *panel) Fill(col color.RGBA) {
	for x := 0; x < p.w; x++ {
		for y := 0; y < p.h; y++ {
			p.c.Set(x, y, col)
		}
	}
}
