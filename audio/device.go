package audio

import (
	"bytes"
	"log"
	"text/template"

	"github.com/gordonklaus/portaudio"
)

var deviceTmpl = template.Must(template.New("").Parse(
	`{{. | len}} host APIs: {{range .}}
	Name:                   {{.Name}}
	{{if .DefaultInputDevice}}Default input device:   {{.DefaultInputDevice.Name}}{{end}}
	{{if .DefaultOutputDevice}}Default output device:  {{.DefaultOutputDevice.Name}}{{end}}
	Devices: {{range .Devices}}
		Name:                      {{.Name}}
		MaxInputChannels:          {{.MaxInputChannels}}
		MaxOutputChannels:         {{.MaxOutputChannels}}
		DefaultLowInputLatency:    {{.DefaultLowInputLatency}}
		DefaultLowOutputLatency:   {{.DefaultLowOutputLatency}}
		DefaultHighInputLatency:   {{.DefaultHighInputLatency}}
		DefaultHighOutputLatency:  {{.DefaultHighOutputLatency}}
		DefaultSampleRate:         {{.DefaultSampleRate}}
	{{end}}
{{end}}`,
))

// PrintDevices prints host devices using deviceTmpl
func PrintDevices() {
	hs, err := portaudio.HostApis()
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer([]byte{})
	err = deviceTmpl.Execute(buf, hs)
	if err != nil {
		panic(err)
	}
	log.Println(buf.String())
}
