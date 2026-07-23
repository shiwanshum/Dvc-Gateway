package main

import (
	"fmt"
	"log"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

func main() {
	postgres.ConnectDB()

	// Update PLC 192.169.4.152
	if err := postgres.DB.Model(&models.MitsubishiPlc{}).Where("ip_address = ?", "192.169.4.152").Update("facility_name", "booth").Error; err != nil {
		log.Fatal(err)
	}
	if err := postgres.DB.Model(&models.MitsubishiTagList{}).Where("plc_ip = ?", "192.169.4.152").Update("facility_name", "booth").Error; err != nil {
		log.Printf("Error updating tags for 152: %v", err)
	}
	fmt.Println("Updated PLC 192.169.4.152 facility name to 'booth'")

	// Update PLC 192.169.4.201
	if err := postgres.DB.Model(&models.MitsubishiPlc{}).Where("ip_address = ?", "192.169.4.201").Update("facility_name", "oven").Error; err != nil {
		log.Fatal(err)
	}
	if err := postgres.DB.Model(&models.MitsubishiTagList{}).Where("plc_ip = ?", "192.169.4.201").Update("facility_name", "oven").Error; err != nil {
		log.Printf("Error updating tags for 201: %v", err)
	}
	fmt.Println("Updated PLC 192.169.4.201 facility name to 'oven'")
}
