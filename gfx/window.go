package gfx

import (
	"github.com/go-gl/glfw/v3.2/glfw"
)

const (
	openglVersionMajor = 4
	openglVersionMinor = 1
)

// Window represents a wrapped glfw window object.
type Window struct {
	Config     *WindowConfig
	GlfwWindow *glfw.Window
}

// WindowConfig contains a new window configuration
type WindowConfig struct {
	Width  int
	Height int
	Title  string
}

// NewWindow initializes a new window object with glfw.
func NewWindow(cfg *WindowConfig) (*Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, err
	}

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, openglVersionMajor)
	glfw.WindowHint(glfw.ContextVersionMinor, openglVersionMinor)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(cfg.Width, cfg.Height, cfg.Title, nil, nil)
	if err != nil {
		return nil, err
	}
	window.MakeContextCurrent()

	return &Window{Config: cfg, GlfwWindow: window}, nil
}
