package gfx

import (
	"image"
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
)

// Context is a context for doing opengl graphics
type Context struct {
	Window  *Window
	Program *Program

	uniforms   map[string]int32
	attributes map[string]int32

	vaos     []*VertexArrayObject
	textures []*TextureObject

	done chan struct{}
}

// NewContext creates a new opengl context
func NewContext(done chan struct{},
	windowConfig *WindowConfig, shaderConfigs []*ShaderConfig) (*Context, error) {
	window, err := NewWindow(windowConfig)
	if err != nil {
		return nil, err
	}

	if err := gl.Init(); err != nil {
		return nil, err
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	//gl.Enable(gl.BLEND)
	//gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	program, err := NewProgram()
	if err != nil {
		return nil, err
	}
	for _, cfg := range shaderConfigs {
		if err := program.AttachShader(cfg); err != nil {
			return nil, err
		}
	}
	if err := program.Link(); err != nil {
		return nil, err
	}

	uniforms := make(map[string]int32)
	attributes := make(map[string]int32)
	for _, sh := range program.Shaders {
		for uname, uloc := range sh.UniformLocations {
			uniforms[uname] = uloc
		}
		for aname, aloc := range sh.AttributeLocations {
			attributes[aname] = aloc
		}
	}

	return &Context{
		Window:     window,
		Program:    program,
		uniforms:   uniforms,
		attributes: attributes,
		done:       done,
	}, nil
}

// EventLoop clears the current framebuffer and executes render in a loop until
// the underlying glfw window tells it to stop. Calls glfw.Terminate when finished.
func (c *Context) EventLoop(render func(*Context)) {

	// OpenGL requires that rendering functions be called from the main thread
	runtime.LockOSThread()

	for !c.Window.GlfwWindow.ShouldClose() {
		select {
		case <-c.done:
			return
		default:
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(c.Program.ProgramID)

		render(c)

		c.Draw()

		glfw.PollEvents()
		c.Window.GlfwWindow.SwapBuffers()
	}
}

// Draw draws every VAO that's attached to the context.
func (c *Context) Draw() {
	for _, v := range c.vaos {
		v.Draw(c)
	}
}

// Terminate ends the glfw session
func (c *Context) Terminate() {
	glfw.Terminate()
}

// AddVertexArrayObject creates a VAO from a VertexBufferObject
// (todo implement that type)
func (c *Context) AddVertexArrayObject(cfg *VAOConfig) error {
	vao, err := c.NewVertexArrayObject(cfg)
	if err != nil {
		return err
	}
	c.vaos = append(c.vaos, vao)
	return nil
}

// GetUniformLocation returns the location of a uniform within the context's program.
func (c *Context) GetUniformLocation(uname string) int32 {
	uloc, ok := c.uniforms[uname]
	if !ok {
		panic("unknown uniform name: " + uname)
	}
	return uloc
}

// GetAttributeLocation returns the location of an attribute with context's program.
func (c *Context) GetAttributeLocation(name string) uint32 {
	loc, ok := c.attributes[name]
	if !ok {
		panic("unknown attribute name: " + name)
	}
	return uint32(loc)
}

// TextureConfig is a configuration for creating a new TextureObject
type TextureConfig struct {
	Image       *image.RGBA
	UniformName string
}

// TextureObject represents a texture that is
type TextureObject struct {
	texID  uint32
	texLoc int32
	pbo    uint32
	image  *image.RGBA
}

// AddTextureObject creates a new TextureObject by first creating a PixelBufferObject
// that will be used to store the texture. The PBO can be updated by calling
// TextureObject.Update() which will read the current state of cfg.Image.
func (c *Context) AddTextureObject(cfg *TextureConfig) (*TextureObject, error) {

	// var pbo uint32
	// gl.GenBuffers(1, &pbo)
	// gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, pbo)
	// // Write PBO with nil to initialize the space
	// gl.BufferData(gl.PIXEL_UNPACK_BUFFER, len(cfg.Image.Pix), nil, gl.STREAM_DRAW)

	var texID uint32
	gl.GenTextures(1, &texID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	// write texture with nil pointer to initialize the space
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA,
		int32(cfg.Image.Rect.Size().X), int32(cfg.Image.Rect.Size().Y),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(cfg.Image.Pix))

	texLoc := c.GetUniformLocation(cfg.UniformName)
	//fmt.Println("texLoc", texLoc, texID)
	gl.Uniform1i(texLoc, 0)

	tex := &TextureObject{
		texID:  texID,
		texLoc: texLoc,
		//pbo:    pbo,
		image: cfg.Image,
	}
	c.textures = append(c.textures, tex)
	return tex, nil
}

func (t *TextureObject) Update() {
	// Update the PBO with image
	//gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, t.pbo)
	//gl.BufferData(gl.PIXEL_UNPACK_BUFFER, len(t.image.Pix), gl.Ptr(t.image.Pix), gl.WRITE_ONLY)

	// Write PBO to texture object
	//fmt.Println("bind texture")
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.texID)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0,
		int32(t.image.Rect.Size().X), int32(t.image.Rect.Size().Y),
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(t.image.Pix)) //nil)
}
