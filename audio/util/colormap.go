// Sourced from https://github.com/lucasb-eyer/go-colorful/blob/master/doc/gradientgen/gradientgen.go

package util

import (
	colorful "github.com/lucasb-eyer/go-colorful"
)

// This table contains the "keypoints" of the colorgradient you want to generate.
// The position of each keypoint has to live in the range [0,1]
type ColorMap []struct {
	Col colorful.Color
	Pos float64
}

// This is the meat of the gradient computation. It returns a HCL-blend between
// the two colors around `t`.
// Note: It relies heavily on the fact that the gradient keypoints are sorted.
func (g ColorMap) GetInterpolatedColorFor(t float64) colorful.Color {
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

func NewColorMap() ColorMap {
	// The "keypoints" of the gradient.
	return ColorMap{
		{mustParseHex("#9e0142"), 0.0},
		{mustParseHex("#d53e4f"), 0.1},
		{mustParseHex("#f46d43"), 0.2},
		{mustParseHex("#fdae61"), 0.3},
		{mustParseHex("#fee090"), 0.4},
		{mustParseHex("#ffffbf"), 0.5},
		{mustParseHex("#e6f598"), 0.6},
		{mustParseHex("#abdda4"), 0.7},
		{mustParseHex("#66c2a5"), 0.8},
		{mustParseHex("#3288bd"), 0.9},
		{mustParseHex("#5e4fa2"), 1.0},
	}
}

// func foo() {
// 	h := 1024
// 	w := 40

// 	if len(os.Args) == 3 {
// 		// Meh, I'm being lazy...
// 		w, _ = strconv.Atoi(os.Args[1])
// 		h, _ = strconv.Atoi(os.Args[2])
// 	}

// 	img := image.NewRGBA(image.Rect(0, 0, w, h))

// 	for y := h - 1; y >= 0; y-- {
// 		c := keypoints.GetInterpolatedColorFor(float64(y) / float64(h))
// 		draw.Draw(img, image.Rect(0, y, w, y+1), &image.Uniform{c}, image.ZP, draw.Src)
// 	}

// 	outpng, err := os.Create("gradientgen.png")
// 	if err != nil {
// 		panic("Error storing png: " + err.Error())
// 	}
// 	defer outpng.Close()

// 	png.Encode(outpng, img)
// }
