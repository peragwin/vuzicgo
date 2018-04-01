
package skgrid

import (
	"image/color"
)

type Grid struct {
	Width int
	Height int
	buffer []byte
	driver Driver
}

type Driver interface {
	Send([]byte) error
}

func NewGrid(width, height int, driver Driver) *Grid {
	ln := width * height
	endframe := make([]byte, 6 + ln/16)
	endframe[0] = 0xff
	buffer := make([]byte, 4 * (ln+1))
	return &Grid{
		Width: width,
		Height: height,
		buffer: append(buffer, endframe...),
		driver: driver,
	}
}

func (s *Grid) SetBuffer(idx int, col color.RGBA) {
	n := 4*idx+4
	s.buffer[n] = 0xe0 | col.A
	s.buffer[n+1] = col.B
	s.buffer[n+2] = col.G
	s.buffer[n+3] = col.R
}

func (s *Grid) Fill(col color.RGBA) {
	for i := 0; i < s.Width * s.Height; i++ {
		s.SetBuffer(i, col)
	}
}

func (s *Grid) Pixel(x, y int, col color.RGBA) {
	if y % 2 == 1{
		x = s.Width - 1 - x
	}
	idx := s.Width * y + x
	s.SetBuffer(idx, col)
}

func (s *Grid) Show() error {
	return s.driver.Send(s.buffer)
}