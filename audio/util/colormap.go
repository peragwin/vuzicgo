// Sourced from https://github.com/lucasb-eyer/go-colorful/blob/master/doc/gradientgen/gradientgen.go

package util

import (
	"image/color"

	colorful "github.com/lucasb-eyer/go-colorful"
)

// This table contains the "keypoints" of the colorgradient you want to generate.
// The position of each keypoint has to live in the range [0,1]
type colorTable []struct {
	Col colorful.Color
	Pos float64
}

// This is the meat of the gradient computation. It returns a HCL-blend between
// the two colors around `t`.
// Note: It relies heavily on the fact that the gradient keypoints are sorted.
func (g colorTable) getInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(g)-1; i++ {
		c1 := g[i]
		c2 := g[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}

	// Nothing found? Means we're at (or past) the last gradient keypoint.
	return g[len(g)-1].Col
}

// This is a very nice thing Golang forces you to do!
// It is necessary so that we can write out the literal of the colortable below.
func mustParseHex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("MustParseHex: " + err.Error())
	}
	return c
}

// ColorMap represents a linear colormap array.
type ColorMap []color.RGBA

// NewColorMap creates a colormap of @size by interpolating colors for each value along
// the hard-coded gradient.
func NewColorMap(size int) ColorMap {
	// The "keypoints" of the gradient.
	table := colorTable{
		{mustParseHex("#5e4fa2"), 0.0},
		{mustParseHex("#3288bd"), 0.1},
		{mustParseHex("#66c2a5"), 0.2},
		{mustParseHex("#abdda4"), 0.3},
		{mustParseHex("#e6f598"), 0.4},
		{mustParseHex("#ffffbf"), 0.5},
		{mustParseHex("#fee090"), 0.6},
		{mustParseHex("#fdae61"), 0.7},
		{mustParseHex("#f46d43"), 0.8},
		{mustParseHex("#d53e4f"), 0.9},
		{mustParseHex("#9e0142"), 1.0},
	}

	colors := make([]color.RGBA, size)
	for i := 0; i < size; i++ {
		r, g, b := table.getInterpolatedColorFor(float64(i) / float64(size)).RGB255()
		colors[i] = color.RGBA{r, g, b, 255}
	}
	return colors
}
