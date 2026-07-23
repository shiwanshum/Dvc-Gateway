package main

import (
	"fmt"
	"log"
	"github.com/google/uuid"
	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

func main() {
	postgres.ConnectDB()

	var plcs []models.MitsubishiPlc
	if err := postgres.DB.Find(&plcs).Error; err != nil {
		log.Fatal(err)
	}

	if len(plcs) < 2 {
		log.Fatal("Need at least 2 PLCs to split tags.")
	}
	plc1 := plcs[0]
	plc2 := plcs[1]

	fmt.Printf("Splitting tags between PLC 1 (%s) and PLC 2 (%s)\n", plc1.IpAddress, plc2.IpAddress)

	// Fetch ALL distinct tags by taking 10000 tags from the first PLC
	var allTags []models.MitsubishiTagList
	postgres.DB.Where("plc_id = ?", plc1.ID).Order("tag_address asc").Find(&allTags)

	// Delete all tags for both PLCs to start fresh
	postgres.DB.Where("plc_id IN ?", []string{plc1.ID, plc2.ID}).Delete(&models.MitsubishiTagList{})

	half := len(allTags) / 2
	var newTags []models.MitsubishiTagList

	for i, tag := range allTags {
		newTag := tag
		newTag.ID = uuid.New().String()
		if i < half {
			newTag.PlcID = plc1.ID
			newTag.FacilityName = plc1.FacilityName
		} else {
			newTag.PlcID = plc2.ID
			newTag.FacilityName = plc2.FacilityName
		}
		newTags = append(newTags, newTag)
	}

	if err := postgres.DB.CreateInBatches(newTags, 500).Error; err != nil {
		log.Fatal("Error splitting tags: ", err)
	}
	fmt.Printf("Successfully assigned %d tags to %s and %d tags to %s\n", half, plc1.IpAddress, len(allTags)-half, plc2.IpAddress)
}
