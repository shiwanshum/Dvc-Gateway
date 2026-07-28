package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	
	"github.com/program-dg/dvc-gateway/internal/api/routes"
	"github.com/program-dg/dvc-gateway/internal/database/timeseries"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/nats"
	"github.com/program-dg/dvc-gateway/internal/orchestrator"
)

func main() {
	fmt.Println("Starting Dvc-Gateway with Make-Wise Structure...")

	postgres.ConnectDB()
	postgres.SeedRoles()
	postgres.SeedPLCs()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222" // Use docker service name
	}
	if err := nats.InitNATS(natsURL); err != nil {
		log.Println("WARNING: NATS not available yet")
	}

	// Start NATS logger for TimescaleDB
	go timeseries.StartNATSLogConsumer()

	go orchestrator.StartManager()

	app := fiber.New(fiber.Config{
		AppName: "i-tips-device-gateway-go-api",
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	routes.SetupRoutes(app)

	log.Println("Gateway API running on :8080")
	log.Fatal(app.Listen(":8080"))
}
