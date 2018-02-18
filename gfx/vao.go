package gfx

import (
	"errors"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// VertexArrayObject points to a vertex buffer that has already been
// loaded into grpahics memory.
type VertexArrayObject struct {
	vaoID      uint32
	length     int32
	glDrawType uint32
	onDraw     func(ctx *Context) bool
}

// VAOConfig represents a configuration for creating a new VAO.
// OnDraw is a function that returns true if the VAO should be drawn, but can
// also be used to set uniforms.
type VAOConfig struct {
	Vertices   []float32
	VertAttr   string
	TexAttr    string
	Stride     int32
	Size       int
	GLDrawType uint32
	OnDraw     func(ctx *Context) bool
}

// NewVertexArrayObject creates a VertexArrayObject
func (c *Context) NewVertexArrayObject(cfg *VAOConfig) (*VertexArrayObject, error) {
	if len(cfg.Vertices)%cfg.Size != 0 {
		return nil, errors.New("invalid length for vertices must be multiple of size")
	}
	stride := 4 * cfg.Stride

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(cfg.Vertices), gl.Ptr(cfg.Vertices), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	vattr := c.GetAttributeLocation(cfg.VertAttr)
	gl.EnableVertexAttribArray(vattr)
	gl.VertexAttribPointer(vattr, int32(cfg.Size), gl.FLOAT, false, stride, gl.PtrOffset(0))

	tattr := c.GetAttributeLocation(cfg.TexAttr)
	gl.EnableVertexAttribArray(tattr)
	gl.VertexAttribPointer(tattr, 2, gl.FLOAT, false, stride, gl.PtrOffset(cfg.Size*4))

	gl.BindVertexArray(0)

	return &VertexArrayObject{
		vao, int32(len(cfg.Vertices)) / cfg.Stride, cfg.GLDrawType,
		cfg.OnDraw,
	}, nil
}

// Draw draws a VertexArrayObject to the current frame buffer
func (v *VertexArrayObject) Draw(ctx *Context) {
	gl.BindVertexArray(v.vaoID)
	if v.onDraw != nil {
		if !v.onDraw(ctx) {
			return
		}
	}
	gl.DrawArrays(v.glDrawType, 0, v.length)
}
