package main

import (
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/peragwin/vuzicgo/gfx"
	"github.com/peragwin/vuzicgo/gfx/grid"
)

const (
	width  = 800
	height = 600

	rows    = 128 * 2
	columns = 128 * 2
)

var (
	ageColors = map[int]color.RGBA{
		1: color.RGBA{230, 0., 120, 255},
		2: color.RGBA{222, 79, 0, 255},
		3: color.RGBA{158, 214, 0, 255},
		4: color.RGBA{0, 207, 28, 255},
		5: color.RGBA{0, 194, 199, 255},
		6: color.RGBA{0, 15, 191, 255},
	}
)

func main() {
	cells := makeCells()
	done := make(chan struct{})

	_, err := grid.NewGrid(done, &grid.Config{
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

	<-done
}

type cell struct {
	vao *gfx.VertexArrayObject

	x, y int

	alive, aliveNext bool
	age              int
}

func (c *cell) getColor() color.RGBA {
	cl, ok := ageColors[c.age]
	if !ok {
		cl = color.RGBA{0, 0, 0, 0} // ageColors[len(ageColors)]
	}
	return cl
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
