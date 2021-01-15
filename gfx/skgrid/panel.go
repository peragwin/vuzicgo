package skgrid

import (
	"image"
	"image/color"

	"github.com/peragwin/go-rpi-rgb-led-matrix"
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
	var panelType string
	panelTypeI, ok := opts["paneltype"]
	if ok {
		panelType = panelTypeI.(string)
	}
	chain := 1
	chainI, ok := opts["chainLength"]
	if ok {
		chain = chainI.(int)
	}
	parallel := 1
	parI, ok := opts["parallel"]
	if ok {
		parallel = parI.(int)
	}
	hwMapping := "regular"
	hwI, ok := opts["hardwareMapping"]
	if ok {
		hwMapping = hwI.(string)
	}
	showRefreshRate := false
	srI, ok := opts["showRefreshRate"]
	if ok {
		showRefreshRate = srI.(bool)
	}
	pwmLSBNano := 130
	pwmLI, ok := opts["pwmLSBNano"]
	if ok {
		pwmLSBNano = pwmLI.(int)
	}

	cfg := &rgbmatrix.HardwareConfig{
		Rows:              h,
		Cols:              w,
		ChainLength:       chain,
		Parallel:          parallel,
		PWMBits:           11,
		Brightness:        100,
		PWMLSBNanoseconds: pwmLSBNano,
		ScanMode:          rgbmatrix.Progressive,
		ShowRefreshRate:   showRefreshRate,
		PanelType:	   panelType,
		HardwareMapping:   hwMapping,
	}

	m, err := rgbmatrix.NewRGBLedMatrix(cfg)
	if err != nil {
		return nil, err
	}
	p := &panel{
		w: w * chain,
		h: h * parallel,
		close: make(chan struct{}),
		m: m,
		c: rgbmatrix.NewCanvas(m),
	}

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
