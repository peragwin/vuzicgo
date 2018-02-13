package main

import (
	"log"
	"math/rand"
	"time"

	ml "github.com/go-gl/mathgl/mgl32"
	"github.com/peragwin/vuzicgo/gfx"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	width  = 800
	height = 600

	rows    = 100
	columns = 100
)

var (
	ageColors = map[int]ml.Vec4{
		1: ml.Vec4{0.90, 0.00, 0.47, 1.0},
		2: ml.Vec4{0.87, 0.31, 0.00, 1.0},
		3: ml.Vec4{0.62, 0.84, 0.00, 1.0},
		4: ml.Vec4{0.00, 0.81, 0.11, 1.0},
		5: ml.Vec4{0.00, 0.76, 0.78, 1.0},
		6: ml.Vec4{0.00, 0.06, 0.75, 1.0},
	}
)

func main() {
	cells := makeCells()

	ctx, err := grid.NewGrid(&grid.Config{
		Width: 800, Height: 600, Title: "Game of Life",
		Columns: columns, Rows: rows,
		Render: func(ctx *grid.Grid) {
			for i := range cells {
				for j := range cells[i] {
					cells[i][j].checkState(cells)
				}
			}
			for i := range cells {
				for j := range cells[i] {
					ctx.SetColor(i, j, cells[i][j].getColor())
				}
			}
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done
}

type cell struct {
	vao *gfx.VertexArrayObject

	x, y int

	alive, aliveNext bool
	age              int
}

func (c *cell) getColor() ml.Vec4 {
	color, ok := ageColors[c.age]
	if !ok {
		color = ml.Vec4{0, 0, 0, 0} // ageColors[len(ageColors)]
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

func makeCells() [][]*cell {
	rand.Seed(time.Now().UnixNano())

	cells := make([][]*cell, rows, rows)
	for i := range cells {
		cells[i] = make([]*cell, columns, columns)
		for j := range cells[i] {
			c := &cell{x: i, y: j}
			cells[i][j] = c
			c.alive = rand.Float64() < 0.15
			c.aliveNext = c.alive
		}
	}
	return cells
}
