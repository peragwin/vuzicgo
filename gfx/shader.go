package gfx

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Shader represents a shader with attached attribute pointers.
type Shader struct {
	ShaderID           uint32
	UniformLocations   map[string]int32
	AttributeLocations map[string]int32

	// whether shader has been attached to a program
	init bool
}

// ShaderConfig is used to create new shaders
type ShaderConfig struct {
	Source         string
	Typ            ShaderType
	AttributeNames []string
	UniformNames   []string
}

// ShaderType tells NewShader what type of shader it's creating.
type ShaderType int

// Types of shaders
const (
	VertexShaderType ShaderType = iota
	FragmentShaderType
)

// NewShader loads and compiles a new shader, but does not attach it to a program.
func NewShader(cfg *ShaderConfig) (*Shader, error) {
	id, err := compileShader(cfg.Source, cfg.Typ)
	if err != nil {
		return nil, err
	}
	uloc := make(map[string]int32)
	for _, un := range cfg.UniformNames {
		uloc[un] = -1
	}
	aloc := make(map[string]int32)
	for _, an := range cfg.AttributeNames {
		aloc[an] = -1
	}
	return &Shader{ShaderID: id, UniformLocations: uloc, AttributeLocations: aloc}, nil
}

// // Verify checks that all variables are accounted for in UniformLocations
// func Verify

func compileShader(src string, typ ShaderType) (uint32, error) {
	var glShaderType uint32
	switch typ {
	case VertexShaderType:
		glShaderType = gl.VERTEX_SHADER
	case FragmentShaderType:
		glShaderType = gl.FRAGMENT_SHADER
	}

	shaderID := gl.CreateShader(glShaderType)

	csources, free := gl.Strs(src + "\x00")
	gl.ShaderSource(shaderID, 1, csources, nil)
	free()
	gl.CompileShader(shaderID)

	var status int32
	gl.GetShaderiv(shaderID, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shaderID, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shaderID, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", src, log)
	}

	return shaderID, nil
}
