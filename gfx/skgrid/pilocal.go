package skgrid

// import (
// 	"github.com/buttairfly/goPi/spi"
// )

// PiLocal is a object representing a local raspberry pi instance which implements the
// Driver interface.
type PiLocal struct {
	// dev *spi.SPIDevice
}

// NewPiLocal creates a new PiLocal instance.
func NewPiLocal(speed uint32) (*PiLocal, error) {
	// dev := spi.NewSPIDevice(spi.DEFAULT_BUS, spi.DEFAULT_CHIP, 0)
	// if err := dev.Open(); err != nil {
	// 	return nil, err
	// }
	// if err := dev.SetBitsPerWord(8); err != nil {
	// 	return nil, err
	// }
	// if err := dev.SetMode(0); err != nil {
	// 	return nil, err
	// }
	// if err := dev.SetSpeed(speed); err != nil {
	// 	return nil, err
	// }
	// if err := dev.Open(); err != nil {
	// 	return nil, err
	// }
	// return &PiLocal{dev: dev}, nil
	panic("foo")
}

// Send data over SPI
func (p *PiLocal) Send(b []byte) error {
	panic("foo")
	// _, err := p.dev.Send(b)
	// return err
}

// Close the RPI instance
func (p *PiLocal) Close() error {
	panic("foo")
	// return p.dev.Close()
}
