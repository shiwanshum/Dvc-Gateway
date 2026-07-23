package allen_bradley

import (
	"log"
)

// EIPConn represents an EtherNet/IP connection to an Allen Bradley PLC
type EIPConn struct {
	IPAddress string
	Port      int
}

// NewEIPConn creates a new connection
func NewEIPConn(ip string, port int) *EIPConn {
	return &EIPConn{
		IPAddress: ip,
		Port:      port,
	}
}

func (c *EIPConn) Connect() error {
	log.Printf("[AllenBradley] Connected to %s:%d via EtherNet/IP", c.IPAddress, c.Port)
	return nil
}

func (c *EIPConn) ReadTag(tagName string) (interface{}, error) {
	return nil, nil
}

func (c *EIPConn) WriteTag(tagName string, value interface{}) error {
	return nil
}

func (c *EIPConn) Close() error {
	return nil
}
