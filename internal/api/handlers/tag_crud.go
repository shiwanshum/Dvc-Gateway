package handlers

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

func GetTagsHandler(c *fiber.Ctx) error {
	var tags []models.MitsubishiTagList
	
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	search := c.Query("search", "")
	filter := c.Query("filter", "")

	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 10 }

	offset := (page - 1) * limit

	query := postgres.DB.Model(&models.MitsubishiTagList{})

	if search != "" {
		query = query.Where("tag_name LIKE ?", "%"+search+"%")
	}
	if filter != "" {
		query = query.Where("fac_name = ?", filter)
	}

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Find(&tags).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(fiber.Map{
		"data":        tags,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func AddTagHandler(c *fiber.Ctx) error {
	var tag models.MitsubishiTagList
	if err := c.BodyParser(&tag); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := postgres.DB.Create(&tag).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tag)
}

func UpdateTagHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var tag models.MitsubishiTagList
	if err := postgres.DB.First(&tag, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tag not found"})
	}
	if err := c.BodyParser(&tag); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := postgres.DB.Save(&tag).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tag)
}

func DeleteTagHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := postgres.DB.Delete(&models.MitsubishiTagList{}, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
