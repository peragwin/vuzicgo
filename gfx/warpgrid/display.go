package warpgrid

import (
	"image"
	"image/color"

	"github.com/go-gl/gl/v4.1-core/gl"
	ml "github.com/go-gl/mathgl/mgl32"
	"github.com/peragwin/vuzicgo/gfx"
)

const (
	// warp controlls the zoom in the center of the display
	// scale controlls the vertical scaling factor
	vertexShaderSource = `
	#version 410
	uniform float warp = 1;
	uniform float scale = 1;
	in vec3 vertPos;
	in vec2 texPos;
	out vec2 fragTexPos;
	
	float x, y;
	void main() {
		x = vertPos.x;
		if (x <= 0) {
			x = pow(x + 1, warp) - 1;
		} else {
			x = 1 - pow(abs(x - 1), warp);
		}

		y = vertPos.y;
		if (y <= 0) {
			y = pow(y + 1, scale) - 1;
		} else {
			y = 1 - pow(abs(y - 1), scale);
		}

		fragTexPos = texPos;
		gl_Position = vec4(x, y, vertPos.z, 1.0);
	}`

	fragmenShaderSource = `
	#version 410
	precision highp float;

	uniform vec2 iResolution = vec2(1920, 1080);
	uniform sampler2D tex;
	in vec2 fragTexPos;
	out vec4 frag_color;

	vec4 blur(sampler2D image, vec2 uv, vec2 resolution) {
		vec4 color = vec4(0.0);
		int radius = 2;
		for (int scale = 1; scale < radius; scale++)
		{
		vec2 direction = vec2(scale,0);

		vec2 off1 = vec2(1.411764705882353) * direction;
		vec2 off2 = vec2(3.2941176470588234) * direction;
		vec2 off3 = vec2(5.176470588235294) * direction;

		color += texture(image, uv) * 0.1964825501511404;
		color += texture(image, uv + (off1 / resolution)) * 0.2969069646728344;
		color += texture(image, uv - (off1 / resolution)) * 0.2969069646728344;
		color += texture(image, uv + (off2 / resolution)) * 0.09447039785044732;
		color += texture(image, uv - (off2 / resolution)) * 0.09447039785044732;
		color += texture(image, uv + (off3 / resolution)) * 0.010381362401148057;
		color += texture(image, uv - (off3 / resolution)) * 0.010381362401148057;

		direction = vec2(0,scale);
		off1 = vec2(1.411764705882353) * direction;
		off2 = vec2(3.2941176470588234) * direction;
		off3 = vec2(5.176470588235294) * direction;

		color += texture(image, uv) * 0.1964825501511404;
		color += texture(image, uv + (off1 / resolution)) * 0.2969069646728344;
		color += texture(image, uv - (off1 / resolution)) * 0.2969069646728344;
		color += texture(image, uv + (off2 / resolution)) * 0.09447039785044732;
		color += texture(image, uv - (off2 / resolution)) * 0.09447039785044732;
		color += texture(image, uv + (off3 / resolution)) * 0.010381362401148057;
		color += texture(image, uv - (off3 / resolution)) * 0.010381362401148057;
		}

		return color / 2 / (radius-1);
	}

	void main() {
		//vec2 uv = fragTexPos.xy / iResolution;
		frag_color = vec4(blur(tex, fragTexPos.xy, iResolution).rgb, 1);
		// vec3 v = texture(tex, fragTexPos).rgb;
		// frag_color = vec4(v, 1);
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
	warp    []float32
	scale   []float32

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
			Typ:            gfx.VertexShaderType,
			Source:         vertexShaderSource,
			AttributeNames: []string{"vertPos", "texPos"},
			UniformNames:   []string{"warp", "scale"},
		},
		&gfx.ShaderConfig{
			Typ:          gfx.FragmentShaderType,
			Source:       fragmenShaderSource,
			UniformNames: []string{"tex", "iResolution"},
		},
	})
	if err != nil {
		return nil, err
	}
	// XXX needed?
	gl.BindFragDataLocation(g.Program.ProgramID, 0, gl.Str("frag_color\x00"))
	//gl.Uniform2f(g.GetUniformLocation("iResolution"), float32(cfg.Width), float32(cfg.Height))
	//gl.TextureParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	// gl.TextureParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

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

	warp := make([]float32, cfg.Rows)
	for i := range warp {
		warp[i] = 1.0
	}
	scale := make([]float32, cfg.Columns)
	for i := range scale {
		scale[i] = 1.0
	}

	grid := &Grid{
		rows:    cfg.Rows,
		columns: cfg.Columns,

		texture: tex,
		image:   img,
		render:  cfg.Render,

		warp:  warp,
		scale: scale,

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

	g.Gfx.EventLoop(func(fx *gfx.Context) {
		if g.render != nil {
			g.render(g)
		}
		// uloc := fx.GetUniformLocation("scale")
		// gl.Uniform1f(uloc, g.scale)
		g.drawTexture(fx)
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
	//verts := make([]float32, 5*len(square)*columns*rows)

	wuloc := ctx.GetUniformLocation("warp")
	suloc := ctx.GetUniformLocation("scale")

	sx, sy := 1.0/float32(columns), 1.0/float32(rows)
	vscale := ml.Scale3D(sx, sy, 1)
	uscale := ml.Scale2D(sx, sy)

	var x, y float32
	for y = 0.0; y < float32(rows); y++ {
		//verts := make([]float32, 5*len(square)*columns)
		for x = 0.0; x < float32(columns); x++ {
			tx := sx * (1 + 2*(x-float32(columns)/2))
			ty := sy * -(1 + 2*(y-float32(rows)/2))

			vtrans := ml.Translate3D(tx, ty, 0)

			tx, ty = sx*x, sy*y

			utrans := ml.Translate2D(tx, ty)

			verts := make([]float32, 5*len(square))
			for i := range square {
				vec := ml.Vec4{square[i][0], square[i][1], 0, 1}
				vec = vtrans.Mul4x1(vscale.Mul4x1(vec))

				tex := ml.Vec3{uvCord[i][0], uvCord[i][1], 1}
				tex = utrans.Mul3x1(uscale.Mul3x1(tex))

				idx := 5 * i //(i + len(square)*cellNum)

				// if idx < 30 {
				// 	// fmt.Println("utrans")
				// 	// fmt.Println(utrans)
				// 	// fmt.Println("uscale")
				// 	// fmt.Println(uscale)
				// 	fmt.Println(idx, len(verts), vec, tex)
				// }

				copy(verts[idx:idx+3], vec[:3])
				copy(verts[idx+3:idx+5], tex[:2])
				// 	}
				// }
			}
			cellNum := int(x)
			rowNum := int(y)
			if err := ctx.AddVertexArrayObject(&gfx.VAOConfig{
				Vertices:   verts,
				Size:       3,
				GLDrawType: gl.TRIANGLE_STRIP,
				VertAttr:   "vertPos",
				TexAttr:    "texPos",
				Stride:     5,
				OnDraw: func(fx *gfx.Context) bool {
					gl.Uniform1f(wuloc, g.warp[rowNum])
					gl.Uniform1f(suloc, g.scale[cellNum])
					return true
				},
			}); err != nil {
				return err
			}
		}
	}

	// bs, _ := json.Marshal(verts[:20])
	// fmt.Println(string(bs))
	return nil
}

func (g *Grid) drawTexture(_ *gfx.Context) {
	g.texture.Update(g.image)
}

func (g *Grid) SetWarp(i int, w float32) {
	g.warp[i] = w
}

func (g *Grid) SetScale(i int, s float32) {
	g.scale[i] = s
}
