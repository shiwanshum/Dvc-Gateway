package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/program-dg/dvc-gateway/internal/api/handlers"
)

func SetupRoutes(app *fiber.App) {
	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	
	// WebSocket endpoint
	app.Get("/ws", websocket.New(handlers.WsHandler))

	api := app.Group("/api")

	plcs := api.Group("/plcs")
	plcs.Get("/", handlers.GetPLCsHandler)
	plcs.Post("/", handlers.AddPLCHandler)
	plcs.Put("/:id", handlers.UpdatePLCHandler)
	plcs.Delete("/:id", handlers.DeletePLCHandler)
	plcs.Post("/:id/scan", handlers.ScanPLCHandler)

	tags := api.Group("/tags")
	tags.Get("/", handlers.GetTagsHandler)
	tags.Post("/", handlers.AddTagHandler)
	tags.Put("/:id", handlers.UpdateTagHandler)
	tags.Delete("/:id", handlers.DeleteTagHandler)

	// Health and Stats Routes
	api.Get("/health/plcs", handlers.GetPLCHealth)
	api.Get("/poller-stats", handlers.GetPollerStats)
}
