package gfx

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Program represents an OpenGL program.
type Program struct {
	ProgramID uint32
	Shaders   []*Shader
}

// NewProgram creates a new Program
func NewProgram() (*Program, error) {
	prog := gl.CreateProgram()
	if prog < 0 {
		return nil, fmt.Errorf("no programs available: %d", prog)
	}
	return &Program{
		ProgramID: prog,
		Shaders:   []*Shader{},
	}, nil
}

// AttachShader attaches a shader from source to a program, defering compilation
// so that calls can be chained together and finished with a call to Link()
func (p *Program) AttachShader(cfg *ShaderConfig) error {
	shader, err := NewShader(cfg)
	if err != nil {
		return err
	}
	p.Shaders = append(p.Shaders, shader)
	gl.AttachShader(p.ProgramID, shader.ShaderID)

	return nil
}

// Link links the program and retrieves all variable locations
func (p *Program) Link() error {
	gl.LinkProgram(p.ProgramID)

	for _, sh := range p.Shaders {
		for uname := range sh.UniformLocations {
			uloc := gl.GetUniformLocation(p.ProgramID, gl.Str(uname+"\x00"))
			if uloc == -1 {
				return fmt.Errorf("location of uniform '%s' not found", uname)
			} else if uloc < 0 {
				return fmt.Errorf("unknown error %d", uloc)
			}
			sh.UniformLocations[uname] = uloc
		}
	}

	return nil
}
