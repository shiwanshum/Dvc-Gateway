package fanuc

import (
	"log"
	"time"

	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

// Poll starts the continuous polling cycle for this specific PLC
func Poll(config models.MitsubishiPlc) {
	log.Printf("[Fanuc] Starting poller for %s:%d", config.IpAddress, config.Port)
	
	conn := NewFocasConn(config.IpAddress, config.Port)
	if err := conn.Connect(); err != nil {
		log.Printf("[Fanuc] Connection failed: %v", err)
		return
	}
	defer conn.Close()

	for {
		// Mock health check and data poll
		// conn.ReadMacro(100)
		time.Sleep(5 * time.Second)
	}
}

// Stop halts the polling cycle
func Stop(ipAddress string) {
	log.Printf("[Fanuc] Stopping poller for %s", ipAddress)
}
