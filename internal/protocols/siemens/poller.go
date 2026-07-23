package siemens

import (
	"log"
	"time"

	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

// Poll starts the continuous polling cycle for this specific PLC
func Poll(config models.MitsubishiPlc) {
	log.Printf("[Siemens] Starting poller for %s:%d", config.IpAddress, config.Port)
	
	conn := NewS7Conn(config.IpAddress, config.Port)
	if err := conn.Connect(); err != nil {
		log.Printf("[Siemens] Connection failed: %v", err)
		return
	}
	defer conn.Close()

	for {
		// Mock health check and data poll
		// conn.ReadBlock(1, 0, 100)
		time.Sleep(5 * time.Second)
	}
}

// Stop halts the polling cycle
func Stop(ipAddress string) {
	log.Printf("[Siemens] Stopping poller for %s", ipAddress)
}
