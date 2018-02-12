package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/peragwin/vuzicgo/gfx"
)

const (
	width  = 800
	height = 600

	rows    = 200
	columns = 200
)

var (
	square = []float32{
		-0.5, 0.5, 0,
		-0.5, -0.5, 0,
		0.5, -0.5, 0,

		-0.5, 0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
	}
	ageColors = map[int][]float32{
		1: {0.90, 0.00, 0.47, 1.0},
		2: {0.87, 0.31, 0.00, 1.0},
		3: {0.62, 0.84, 0.00, 1.0},
		4: {0.00, 0.81, 0.11, 1.0},
		5: {0.00, 0.76, 0.78, 1.0},
		6: {0.00, 0.06, 0.75, 1.0},
	}
)

const (
	vertexShaderSource = `
	#version 410
	in vec3 vp;
	void main() {
		gl_Position = vec4(vp, 1.0);
	}`

	fragmenShaderSource = `
	#version 410
	uniform vec4 uColor = vec4(1, 1, 1, 1);
	out vec4 frag_color;
	void main() {
		frag_color = uColor;
	}`
)

func main() {
	ctx, err := gfx.NewContext(&gfx.WindowConfig{
		Width: 800, Height: 600, Title: "Game of Life",
	}, []*gfx.ShaderConfig{
		&gfx.ShaderConfig{
			Typ:    gfx.VertexShaderType,
			Source: vertexShaderSource,
		},
		&gfx.ShaderConfig{
			Typ:          gfx.FragmentShaderType,
			Source:       fragmenShaderSource,
			UniformNames: []string{"uColor"},
		},
	})
	if err != nil {
		log.Println(err)
		return
	}

	defer ctx.Terminate()

	cells := makeCells(ctx)

	ctx.EventLoop(func(ctx *gfx.Context) {
		for i := range cells {
			for j := range cells[i] {
				cells[i][j].checkState(cells)
			}
		}

		ctx.Draw()
	})
}

type cell struct {
	vao *gfx.VertexArrayObject

	x, y int

	alive, aliveNext bool
	age              int
}

func newCell(x, y int, ctx *gfx.Context) *cell {
	points := make([]float32, len(square), len(square))
	copy(points, square)

	for i := range points {
		var pos, size float32
		switch i % 3 {
		case 0:
			size = 1.0 / float32(columns)
			pos = float32(x) * size
		case 1:
			size = 1.0 / float32(rows)
			pos = float32(y) * size
		default:
			continue
		}

		if points[i] < 0 {
			points[i] = (pos * 2) - 1
		} else {
			points[i] = ((pos + size) * 2) - 1
		}
	}

	c := &cell{
		x: x,
		y: y,
	}
	if err := ctx.AddVertexArrayObject(&gfx.VAOConfig{
		Vertices:   points,
		Size:       3,
		GLDrawType: gl.TRIANGLE_STRIP,
		OnDraw: func(ctx *gfx.Context) bool {
			if !c.alive {
				return false
			}
			v := c.getColor()
			uloc := ctx.GetUniformLocation("uColor")
			gl.Uniform4f(uloc, v[0], v[1], v[2], v[3])
			return true
		},
	}); err != nil {
		panic(err)
	}
	return c
}

func (c *cell) getColor() []float32 {
	color := ageColors[c.age]
	if color == nil {
		color = ageColors[len(ageColors)]
	}
	return color
}

func (c *cell) checkState(cells [][]*cell) {
	c.alive = c.aliveNext
	c.aliveNext = c.alive // default

	liveCount := c.liveNeighbors(cells)
	if c.alive {
		c.aliveNext = liveCount == 2 || liveCount == 3
	} else {
		c.aliveNext = liveCount == 3
	}

	if c.alive {
		c.age++
	} else {
		c.age = 0
	}
}

func (c *cell) liveNeighbors(cells [][]*cell) int {
	var liveCount int
	add := func(x, y int) {
		if x == len(cells) {
			x = 0
		} else if x == -1 {
			x = len(cells) - 1
		}
		if y == len(cells[x]) {
			y = 0
		} else if y == -1 {
			y = len(cells[x]) - 1
		}

		if cells[x][y].alive {
			liveCount++
		}
	}

	add(c.x-1, c.y)   // To the left
	add(c.x+1, c.y)   // To the right
	add(c.x, c.y+1)   // up
	add(c.x, c.y-1)   // down
	add(c.x-1, c.y+1) // top-left
	add(c.x+1, c.y+1) // top-right
	add(c.x-1, c.y-1) // bottom-left
	add(c.x+1, c.y-1) // bottom-right

	return liveCount
}

func makeCells(ctx *gfx.Context) [][]*cell {
	rand.Seed(time.Now().UnixNano())

	cells := make([][]*cell, rows, rows)
	for i := range cells {
		cells[i] = make([]*cell, columns, columns)
		for j := range cells[i] {
			c := newCell(i, j, ctx)
			cells[i][j] = c
			c.alive = rand.Float64() < 0.15
			c.aliveNext = c.alive
		}
	}
	return cells
}
