package orchestrator

import (
	"log"
	"sync"

	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
	"github.com/program-dg/dvc-gateway/internal/protocols/mitsubishi"
	"github.com/program-dg/dvc-gateway/internal/protocols/rockwell"
	"github.com/program-dg/dvc-gateway/internal/protocols/siemens"
	"github.com/program-dg/dvc-gateway/internal/protocols/fanuc"
	"github.com/program-dg/dvc-gateway/internal/protocols/allen_bradley"
)

var (
	activePLCs sync.Map // Map of ID to running goroutine status
)

// StartManager boots up all configured PLCs from the database
func StartManager() {
	// For now, assume we fetch this list from Postgres
	// e.g. plcs := postgres.GetAllPLCs()
	log.Println("[Orchestrator] Starting dynamic PLC manager...")
}

// AddPLC dynamically starts polling for a new PLC config
func AddPLC(config models.MitsubishiPlc) error {
	log.Printf("[Orchestrator] Request to add PLC: %s (Make: %s)", config.IpAddress, config.Maker)
	
	if _, exists := activePLCs.Load(config.ID); exists {
		log.Printf("[Orchestrator] PLC %s is already running", config.IpAddress)
		return nil
	}

	activePLCs.Store(config.ID, true)

	// Spawns the Make-specific poller
	go func() {
		switch config.Maker {
		case "mitsubishi", "Mitsubishi", "MITSUBISHI":
			mitsubishi.Poll(config)
		case "rockwell", "Rockwell", "ROCKWELL":
			rockwell.Poll(config)
		case "siemens", "Siemens", "SIEMENS":
			siemens.Poll(config)
		case "fanuc", "Fanuc", "FANUC":
			fanuc.Poll(config)
		case "allen_bradley", "AllenBradley", "ALLEN_BRADLEY":
			allen_bradley.Poll(config)
		default:
			// Fallback to mitsubishi if unknown
			mitsubishi.Poll(config)
		}
		activePLCs.Delete(config.ID)
	}()

	return nil
}

// RemovePLC gracefully stops polling for a PLC
func RemovePLC(plcID string, make string) error {
	log.Printf("[Orchestrator] Stopping PLC ID: %s", plcID)
	// In a real implementation, we'd send a context cancellation or stop signal
	return nil
}
