package orchestrator

import (
	"context"
	"log"
	"sync"

	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/protocols/mitsubishi"
	"github.com/program-dg/dvc-gateway/internal/protocols/rockwell"
	"github.com/program-dg/dvc-gateway/internal/protocols/siemens"
	"github.com/program-dg/dvc-gateway/internal/protocols/fanuc"
	"github.com/program-dg/dvc-gateway/internal/protocols/allen_bradley"
)

var (
	activePLCs sync.Map // Map of ID to context.CancelFunc
)

// StartManager boots up all configured PLCs from the database
func StartManager() {
	log.Println("[Orchestrator] Starting dynamic PLC manager...")

	var plcs []models.MitsubishiPlc
	// Fetch all PLCs from the database (assuming postgres.DB is initialized)
	// We need to import the postgres package to use postgres.DB
	if err := postgres.DB.Find(&plcs).Error; err != nil {
		log.Printf("[Orchestrator] Error fetching PLCs from DB: %v", err)
		return
	}

	for _, plc := range plcs {
		AddPLC(plc)
	}
}

// AddPLC dynamically starts polling for a new PLC config
func AddPLC(config models.MitsubishiPlc) error {
	log.Printf("[Orchestrator] Request to add PLC: %s (Make: %s)", config.IpAddress, config.Maker)
	
	if _, exists := activePLCs.Load(config.ID); exists {
		log.Printf("[Orchestrator] PLC %s is already running", config.IpAddress)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	activePLCs.Store(config.ID, cancel)

	// Spawns the Make-specific poller
	go func() {
		defer activePLCs.Delete(config.ID)
		switch config.Maker {
		case "mitsubishi", "Mitsubishi", "MITSUBISHI":
			mitsubishi.Poll(ctx, config)
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
			mitsubishi.Poll(ctx, config)
		}
	}()

	return nil
}

// RemovePLC gracefully stops polling for a PLC
func RemovePLC(plcID string, make string) error {
	log.Printf("[Orchestrator] Stopping PLC ID: %s", plcID)
	if cancelFunc, exists := activePLCs.Load(plcID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			cancel()
		}
		activePLCs.Delete(plcID)
	}
	return nil
}

// RestartPLC stops and then starts a PLC poller to force a reconnect or port scan
func RestartPLC(config models.MitsubishiPlc) error {
	RemovePLC(config.ID, config.Maker)
	return AddPLC(config)
}
