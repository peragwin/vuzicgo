package gfx

import (
	"image"

	"github.com/go-gl/gl/v4.1-core/gl"
)

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
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.texID)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0,
		int32(t.image.Rect.Size().X), int32(t.image.Rect.Size().Y),
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(t.image.Pix)) //nil)
}
