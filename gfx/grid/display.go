package grid

import (
	"image"
	"image/color"

	"github.com/go-gl/gl/v4.1-core/gl"
	ml "github.com/go-gl/mathgl/mgl32"
	"github.com/peragwin/vuzicgo/gfx"
)

const (
	vertexShaderSource = `
	#version 410
	layout (location=1) in vec3  vertPos;
	layout (location=2) in vec2  texPos;
	out vec2 fragTexPos;
	void main() {
		fragTexPos = texPos;
		gl_Position = vec4(vertPos, 1.0);
	}`

	fragmenShaderSource = `
	#version 410
	uniform sampler2D tex;
	in vec2 fragTexPos;
	out vec4 frag_color;
	void main() {
		frag_color = texture(tex, fragTexPos);
	}`
)

var (
	square = [6]ml.Vec2{
		{-1, 1},
		{-1, -1},
		{1, -1},

		{-1, 1},
		{1, 1},
		{1, -1},
	}
	uvCord = [6]ml.Vec2{
		{0, 0},
		{0, 1},
		{1, 1},

		{0, 0},
		{1, 0},
		{1, 1},
	}
)

// Grid is a 2D-grid display engine.
type Grid struct {
	rows    int
	columns int

	image   *image.RGBA
	texture *gfx.TextureObject

	Gfx  *gfx.Context
	Done chan struct{}
}

// Config is a configuration for creating a new Grid.
type Config struct {
	Width       int
	Height      int
	Columns     int
	Rows        int
	Title       string
	TextureMode int32

	Render func(*Grid)
}

// NewGrid creates a new Grid display
func NewGrid(done chan struct{}, cfg *Config) (*Grid, error) {
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	g, err := gfx.NewContext(done, &gfx.WindowConfig{
		Width: cfg.Width, Height: cfg.Height, Title: cfg.Title,
	}, []*gfx.ShaderConfig{
		&gfx.ShaderConfig{
			Typ:    gfx.VertexShaderType,
			Source: vertexShaderSource,
		},
		&gfx.ShaderConfig{
			Typ:            gfx.FragmentShaderType,
			Source:         fragmenShaderSource,
			AttributeNames: []string{"vertPos", "texPos"},
			UniformNames:   []string{"tex"},
		},
	})
	if err != nil {
		return nil, err
	}
	// XXX needed?
	//gl.BindFragDataLocation(g.Program.ProgramID, 0, gl.Str("frag_color\x00"))

	img := image.NewRGBA(image.Rect(0, 0, cfg.Columns, cfg.Rows))
	//img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for i := 0; i < cfg.Columns; i++ {
		for j := 0; j < cfg.Rows; j++ {
			if i%2^j%2 == 0 {
				img.SetRGBA(i, j, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	tex, err := g.AddTextureObject(&gfx.TextureConfig{
		Image:       img,
		UniformName: "tex",
		Mode:        cfg.TextureMode,
	})
	if err != nil {
		return nil, err
	}

	grid := &Grid{
		rows:    cfg.Rows,
		columns: cfg.Columns,

		texture: tex,
		image:   img,

		Gfx:  g,
		Done: done,
	}

	if err := grid.createCells(cfg.Columns, cfg.Rows, g); err != nil {
		return nil, err
	}

	go func() {
		defer g.Terminate()
		defer close(done)

		g.EventLoop(func(g *gfx.Context) {
			cfg.Render(grid)
		})
	}()

	return grid, nil
}

// SetColor sets a cell in the grid to a color
func (g *Grid) SetColor(i, j int, clr color.RGBA) {
	//g.colors[g.getColorIndex(i, j)] = color
	g.image.SetRGBA(i, j, clr)
}

// SetImage sets the entire display at once
func (g *Grid) SetImage(img *image.RGBA) {
	g.image = img
}

// Clear sets all the cells to black
func (g *Grid) Clear() {
	// for i := range g.colors {
	// 	g.colors[i] = ml.Vec4{}
	// }
	for i := range g.image.Pix {
		g.image.Pix[i] = 0
	}
}

func (g *Grid) getColorIndex(i, j int) int {
	return i*g.rows + j
}

func (g *Grid) createCells(columns, rows int, ctx *gfx.Context) error {

	//g.colors = make([]ml.Vec4, columns*rows)

	// scaleX := 2.0 / float32(columns)
	// scaleY := 2.0 / float32(rows)
	// scale := ml.Scale3D(scaleX, scaleY, 0)

	// for i := 0; i < columns; i++ {
	// 	for j := 0; j < rows; j++ {

	// tx := float32(i)*scaleX - 1.0
	// ty := float32(j)*scaleY - 1.0
	// translate := ml.Translate3D(tx, ty, 0)

	// verts := make([]float32, 3*len(square))
	// for u := 0; u < len(square); u++ {
	// 	vec := square[u].Vec4(0, 1)
	// 	vec = scale.Mul4x1(vec)
	// 	vec = translate.Mul4x1(vec)

	// 	for k := 0; k < 3; k++ {
	// 		verts[3*u+k] = vec[k]
	// 	}
	// }

	// index := g.getColorIndex(i, j)
	// g.colors[index] = ml.Vec4{0, 0, 0, 0}
	// uloc := g.Gfx.GetUniformLocation("u_color")

	// if err := ctx.AddVertexArrayObject(&gfx.VAOConfig{
	// 	Vertices:   verts,
	// 	Size:       3,
	// 	GLDrawType: gl.TRIANGLE_STRIP,
	// 	OnDraw: func(ctx *gfx.Context) bool {
	// 		v := g.colors[index]
	// 		gl.Uniform4f(uloc, v[0], v[1], v[2], v[3])
	// 		return true
	// 	},
	// }); err != nil {
	// 	return err
	// }
	// 	}
	// }

	verts := make([]float32, 5*len(square))
	for i := range square {
		// xyz coord
		verts[5*i] = square[i][0]
		verts[5*i+1] = square[i][1]
		verts[5*i+2] = 0

		// uv coord
		verts[5*i+3] = uvCord[i][0]
		verts[5*i+4] = uvCord[i][1]
	}

	return ctx.AddVertexArrayObject(&gfx.VAOConfig{
		Vertices:   verts,
		Size:       3,
		GLDrawType: gl.TRIANGLE_STRIP,
		VertAttr:   "vertPos",
		TexAttr:    "texPos",
		Stride:     5,
		OnDraw:     g.drawTexture,
	})
}

var count int

func (g *Grid) drawTexture(*gfx.Context) bool {
	// if count%60 == 0 {
	// 	for i := range g.image.Pix {
	// 		if g.image.Pix[i] != 0 {
	// 			g.image.Pix[i] = 0
	// 		} else {
	// 			g.image.Pix[i] = 255
	// 		}
	// 	}
	// }
	g.texture.Update(g.image)
	return true
}
