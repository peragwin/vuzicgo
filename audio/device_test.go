package audio

import (
	"os"
	"testing"

	"github.com/gordonklaus/portaudio"
)

func TestMain(m *testing.M) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	os.Exit(m.Run())
}

func TestPrintDevices(t *testing.T) {
	PrintDevices()
}

func chk(t *testing.T, err error) {
	if err != nil {
		panic(err)
	}
}
