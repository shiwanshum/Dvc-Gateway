package handlers

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
	"github.com/program-dg/dvc-gateway/internal/orchestrator"
	"github.com/program-dg/dvc-gateway/internal/protocols/mitsubishi"
	"time"
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

type ManualReadWriteReq struct {
	Device string `json:"device"`
	Offset int    `json:"offset"`
	IsBit  bool   `json:"is_bit"`
	Value  int    `json:"value"` // only used for write
}

func ManualReadHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req ManualReadWriteReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	var plc models.MitsubishiPlc
	if err := postgres.DB.First(&plc, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "PLC not found"})
	}

	start := time.Now()
	conn, err := mitsubishi.NewPlcConn(plc.IpAddress, plc.Port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "rtt_ms": 0})
	}
	defer conn.Close()

	var val int
	if req.IsBit {
		frame := mitsubishi.BuildReadFrame(req.Device, req.Offset, 1)
		res, err := conn.DoRead(frame)
		if err != nil {
			rtt := float64(time.Since(start).Microseconds()) / 1000.0
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "rtt_ms": rtt})
		}
		if len(res) > 0 { val = res[0] }
	} else {
		frame := mitsubishi.BuildReadWordFrame(req.Device, req.Offset, 1)
		res, err := conn.DoReadWords(frame, 1)
		if err != nil {
			rtt := float64(time.Since(start).Microseconds()) / 1000.0
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "rtt_ms": rtt})
		}
		if len(res) > 0 { val = res[0] }
	}
	
	rtt := float64(time.Since(start).Microseconds()) / 1000.0
	return c.JSON(fiber.Map{"value": val, "rtt_ms": rtt})
}

func ManualWriteHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req ManualReadWriteReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	var plc models.MitsubishiPlc
	if err := postgres.DB.First(&plc, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "PLC not found"})
	}

	start := time.Now()
	conn, err := mitsubishi.NewPlcConn(plc.IpAddress, plc.Port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "rtt_ms": 0})
	}
	defer conn.Close()

	if req.IsBit {
		err = conn.WriteSingleBit(req.Device, req.Offset, req.Value)
	} else {
		err = conn.WriteSingleWord(req.Device, req.Offset, req.Value)
	}

	rtt := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "rtt_ms": rtt})
	}
	
	return c.JSON(fiber.Map{"status": "success", "rtt_ms": rtt})
}
