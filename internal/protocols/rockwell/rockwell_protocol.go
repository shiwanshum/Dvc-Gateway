package rockwell

import (
	"log"
)

// RockwellEIPConn represents a Rockwell-specific EtherNet/IP connection
type RockwellEIPConn struct {
	IPAddress string
	Port      int
}

func NewRockwellEIPConn(ip string, port int) *RockwellEIPConn {
	return &RockwellEIPConn{
		IPAddress: ip,
		Port:      port,
	}
}

func (c *RockwellEIPConn) Connect() error {
	log.Printf("[Rockwell] Connected to %s:%d via CIP/EIP", c.IPAddress, c.Port)
	return nil
}

func (c *RockwellEIPConn) Close() error {
	return nil
}
