package main

import (
	"fmt"
	"log"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

func main() {
	postgres.ConnectDB()

	// Update PLC 192.169.4.152 -> booth
	var plc1 models.MitsubishiPlc
	if err := postgres.DB.Where("ip_address = ?", "192.169.4.152").First(&plc1).Error; err == nil {
		postgres.DB.Model(&plc1).Update("facility_name", "booth")
		postgres.DB.Model(&models.MitsubishiTagList{}).Where("plc_id = ?", plc1.ID).Update("facility_name", "booth")
		fmt.Println("Updated PLC 192.169.4.152 facility name to 'booth'")
	}

	// Update PLC 192.169.4.201 -> oven
	var plc2 models.MitsubishiPlc
	if err := postgres.DB.Where("ip_address = ?", "192.169.4.201").First(&plc2).Error; err == nil {
		postgres.DB.Model(&plc2).Update("facility_name", "oven")
		postgres.DB.Model(&models.MitsubishiTagList{}).Where("plc_id = ?", plc2.ID).Update("facility_name", "oven")
		fmt.Println("Updated PLC 192.169.4.201 facility name to 'oven'")
	}
}
