package handlers

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
	"github.com/program-dg/dvc-gateway/internal/orchestrator"
)

func GetPLCsHandler(c *fiber.Ctx) error {
	var plcs []models.MitsubishiPlc
	
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	search := c.Query("search", "")
	filter := c.Query("filter", "")

	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 10 }

	offset := (page - 1) * limit

	query := postgres.DB.Model(&models.MitsubishiPlc{})

	if search != "" {
		query = query.Where("ip_address LIKE ?", "%"+search+"%")
	}
	if filter != "" {
		query = query.Where("facility_name = ?", filter)
	}

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Find(&plcs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(fiber.Map{
		"data":        plcs,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func AddPLCHandler(c *fiber.Ctx) error {
	var plc models.MitsubishiPlc
	if err := c.BodyParser(&plc); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := postgres.DB.Create(&plc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	orchestrator.AddPLC(plc)
	return c.JSON(plc)
}

func UpdatePLCHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var plc models.MitsubishiPlc
	if err := postgres.DB.First(&plc, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "PLC not found"})
	}
	if err := c.BodyParser(&plc); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := postgres.DB.Save(&plc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(plc)
}

func DeletePLCHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := postgres.DB.Delete(&models.MitsubishiPlc{}, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func ScanPLCHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var plc models.MitsubishiPlc
	if err := postgres.DB.First(&plc, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "PLC not found"})
	}
	
	err := orchestrator.RestartPLC(plc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "scanning", "message": "Port scan and reconnect triggered."})
}
