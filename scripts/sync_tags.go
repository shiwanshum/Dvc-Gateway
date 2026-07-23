package main

import (
	"fmt"
	"log"
	"github.com/google/uuid"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

func main() {
	postgres.InitPostgres() // Initialize DB connection

	var plcs []models.MitsubishiPlc
	if err := postgres.DB.Find(&plcs).Error; err != nil {
		log.Fatal(err)
	}

	var sourcePLC *models.MitsubishiPlc
	var targetPLCs []models.MitsubishiPlc

	// Find the PLC with tags to use as source
	for _, plc := range plcs {
		var count int64
		postgres.DB.Model(&models.MitsubishiTagList{}).Where("plc_id = ?", plc.ID).Count(&count)
		if count > 0 {
			if sourcePLC == nil {
				plcCopy := plc
				sourcePLC = &plcCopy
				fmt.Printf("Found Source PLC: %s with %d tags\n", plc.IpAddress, count)
			}
		} else {
			targetPLCs = append(targetPLCs, plc)
			fmt.Printf("Found Target PLC: %s with 0 tags\n", plc.IpAddress)
		}
	}

	if sourcePLC == nil {
		log.Fatal("No source PLC with tags found!")
	}

	if len(targetPLCs) == 0 {
		fmt.Println("All PLCs already have tags synced.")
		return
	}

	var sourceTags []models.MitsubishiTagList
	postgres.DB.Where("plc_id = ?", sourcePLC.ID).Find(&sourceTags)

	for _, target := range targetPLCs {
		fmt.Printf("Syncing %d tags to PLC %s...\n", len(sourceTags), target.IpAddress)
		
		var newTags []models.MitsubishiTagList
		for _, tag := range sourceTags {
			newTag := tag
			newTag.ID = uuid.New().String() // Generate new UUID
			newTag.PlcID = target.ID
			newTag.PlcIp = target.IpAddress
			newTag.FacilityName = target.FacilityName
			newTags = append(newTags, newTag)
		}

		// Insert in batches of 500
		if err := postgres.DB.CreateInBatches(newTags, 500).Error; err != nil {
			log.Printf("Error syncing tags to %s: %v\n", target.IpAddress, err)
		} else {
			fmt.Printf("Successfully synced tags to %s\n", target.IpAddress)
		}
	}
}
