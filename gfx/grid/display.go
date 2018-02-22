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
	render  func(*Grid)

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
	gl.BindFragDataLocation(g.Program.ProgramID, 0, gl.Str("frag_color\x00"))

	img := image.NewRGBA(image.Rect(0, 0, cfg.Columns, cfg.Rows))
	//fmt.Println(cfg.Columns, cfg.Rows)
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
		render:  cfg.Render,

		Gfx:  g,
		Done: done,
	}

	if err := grid.createCells(cfg.Columns, cfg.Rows, g); err != nil {
		return nil, err
	}

	return grid, nil
}

// Start initiates the graphics event loop
func (g *Grid) Start() {
	defer g.Gfx.Terminate()

	g.Gfx.EventLoop(func(_ *gfx.Context) {
		if g.render != nil {
			g.render(g)
		}
	})
}

// SetColor sets a cell in the grid to a color
func (g *Grid) SetColor(i, j int, clr color.RGBA) {
	g.image.SetRGBA(i, j, clr)
}

// SetImage sets the entire display at once
func (g *Grid) SetImage(img *image.RGBA) {
	g.image = img
}

// Clear sets all the cells to black
func (g *Grid) Clear() {
	for i := range g.image.Pix {
		g.image.Pix[i] = 0
	}
}

// SetRenderFunc sets the render function of the display grid
func (g *Grid) SetRenderFunc(render func(*Grid)) {
	g.render = render
}

func (g *Grid) getColorIndex(i, j int) int {
	return i*g.rows + j
}

func (g *Grid) createCells(columns, rows int, ctx *gfx.Context) error {
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

func (g *Grid) drawTexture(*gfx.Context) bool {
	g.texture.Update(g.image)
	return true
}
