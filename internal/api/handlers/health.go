package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
	"github.com/program-dg/dvc-gateway/internal/protocols/mitsubishi"
	"math"
)

// GetPLCHealth returns the current connectivity status and latency for all PLCs
func GetPLCHealth(c *fiber.Ctx) error {
	var plcs []models.MitsubishiPlc
	if err := postgres.DB.Find(&plcs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	type HealthResponse struct {
		ID        string  `json:"id"`
		Name      string  `json:"name"`
		IPAddress string  `json:"ip_address"`
		Status    string  `json:"status"`
		LatencyMS float64 `json:"latency_ms"`
	}

	var response []HealthResponse

	for _, plc := range plcs {
		status := "offline"
		latency := 0.0

		// Look up real-time health from the Poller's sync map
		if val, ok := mitsubishi.PlcHealthMap.Load(plc.ID); ok {
			health := val.(mitsubishi.PLCHealth)
			status = health.Status
			latency = health.LatencyMS
		}

		response = append(response, HealthResponse{
			ID:        plc.ID,
			Name:      plc.FacilityName,
			IPAddress: plc.IpAddress,
			Status:    status,
			LatencyMS: latency,
		})
	}

	return c.JSON(response)
}

// GetPollerStats returns the global read/write operations and average latencies
func GetPollerStats(c *fiber.Ctx) error {
	stats := mitsubishi.GetPollerStats()

	avgRead := 0.0
	avgWrite := 0.0
	
	if stats.ReadOps > 0 {
		avgRead = float64(stats.ReadMicros) / float64(stats.ReadOps) / 1000.0
	}
	if stats.WriteOps > 0 {
		avgWrite = float64(stats.WriteMicros) / float64(stats.WriteOps) / 1000.0
	}

	return c.JSON(fiber.Map{
		"read_latency_ms":  math.Round(avgRead*100) / 100,
		"write_latency_ms": math.Round(avgWrite*100) / 100,
		"read_ops":         stats.ReadOps,
		"write_ops":        stats.WriteOps,
		"error_count":      stats.ErrorCount,
	})
}
