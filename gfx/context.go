package gfx

import (
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

	if err := gl.Init(); err != nil {
		return nil, err
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	window, err := NewWindow(windowConfig)
	if err != nil {
		return nil, err
	}

	gl.Flush()
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

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

		gl.Flush()

		c.Window.GlfwWindow.SwapBuffers()

		glfw.PollEvents()
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
