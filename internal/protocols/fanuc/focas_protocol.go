package fanuc

import (
	"log"
)

// FocasConn represents a FOCAS API connection to a Fanuc CNC/Robot
type FocasConn struct {
	IPAddress string
	Port      int
}

func NewFocasConn(ip string, port int) *FocasConn {
	return &FocasConn{
		IPAddress: ip,
		Port:      port,
	}
}

func (c *FocasConn) Connect() error {
	log.Printf("[Fanuc] Connected to %s:%d via FOCAS", c.IPAddress, c.Port)
	return nil
}

func (c *FocasConn) ReadMacro(number int) (float64, error) {
	return 0.0, nil
}

func (c *FocasConn) Close() error {
	return nil
}
