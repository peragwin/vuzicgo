package grid

import (
	"context"

	"github.com/go-gl/gl/v4.1-core/gl"
	ml "github.com/go-gl/mathgl/mgl32"
	"github.com/peragwin/vuzicgo/gfx"
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
	uniform vec4 u_color = vec4(1, 1, 1, 1);
	out vec4 frag_color;
	void main() {
		frag_color = u_color;
	}`
)

var (
	square = [6]ml.Vec2{
		{0, 1},
		{0, 0},
		{1, 0},

		{0, 1},
		{1, 1},
		{1, 0},
	}
)

// Grid is a 2D-grid display engine.
type Grid struct {
	rows    int
	columns int

	colors []ml.Vec4

	Gfx *gfx.Context

	Done chan struct{}
}

// Config is a configuration for creating a new Grid.
type Config struct {
	Width   int
	Height  int
	Columns int
	Rows    int
	Title   string

	Render func(*Grid)
}

// NewGrid creates a new Grid display
func NewGrid(ctx context.Context, cfg *Config) (*Grid, error) {
	g, err := gfx.NewContext(ctx, &gfx.WindowConfig{
		Width: cfg.Width, Height: cfg.Height, Title: cfg.Title,
	}, []*gfx.ShaderConfig{
		&gfx.ShaderConfig{
			Typ:    gfx.VertexShaderType,
			Source: vertexShaderSource,
		},
		&gfx.ShaderConfig{
			Typ:          gfx.FragmentShaderType,
			Source:       fragmenShaderSource,
			UniformNames: []string{"u_color"},
		},
	})
	if err != nil {
		return nil, err
	}

	grid := &Grid{
		rows:    cfg.Rows,
		columns: cfg.Columns,

		Gfx:  g,
		Done: make(chan struct{}),
	}

	if err := grid.createCells(cfg.Columns, cfg.Rows, g); err != nil {
		return nil, err
	}

	go func() {
		defer g.Terminate()
		g.EventLoop(func(g *gfx.Context) {
			cfg.Render(grid)
		})
		grid.Done <- struct{}{}
	}()

	return grid, nil
}

// SetColor sets a cell in the grid to a color
func (g *Grid) SetColor(i, j int, color ml.Vec4) {
	g.colors[g.getColorIndex(i, j)] = color
}

// Clear sets all the cells to black
func (g *Grid) Clear() {
	for i := range g.colors {
		g.colors[i] = ml.Vec4{}
	}
}

func (g *Grid) getColorIndex(i, j int) int {
	return i*g.rows + j
}

func (g *Grid) createCells(columns, rows int, ctx *gfx.Context) error {

	g.colors = make([]ml.Vec4, columns*rows)

	scaleX := 2.0 / float32(columns)
	scaleY := 2.0 / float32(rows)
	scale := ml.Scale3D(scaleX, scaleY, 0)

	for i := 0; i < columns; i++ {
		for j := 0; j < rows; j++ {

			tx := float32(i)*scaleX - 1.0
			ty := float32(j)*scaleY - 1.0
			translate := ml.Translate3D(tx, ty, 0)

			verts := make([]float32, 3*len(square))
			for u := 0; u < len(square); u++ {
				vec := square[u].Vec4(0, 1)
				vec = scale.Mul4x1(vec)
				vec = translate.Mul4x1(vec)

				for k := 0; k < 3; k++ {
					verts[3*u+k] = vec[k]
				}
			}

			index := g.getColorIndex(i, j)
			g.colors[index] = ml.Vec4{0, 0, 0, 0}
			uloc := g.Gfx.GetUniformLocation("u_color")

			if err := ctx.AddVertexArrayObject(&gfx.VAOConfig{
				Vertices:   verts,
				Size:       3,
				GLDrawType: gl.TRIANGLE_STRIP,
				OnDraw: func(ctx *gfx.Context) bool {
					v := g.colors[index]
					gl.Uniform4f(uloc, v[0], v[1], v[2], v[3])
					return true
				},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
