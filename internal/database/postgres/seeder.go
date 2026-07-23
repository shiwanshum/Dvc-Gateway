package postgres

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type PLCConfig struct {
	FacilityName string `json:"Facility name"`
	Driver       string `json:"Driver"`
	IPAddress    string `json:"Ip address"`
	ComType      string `json:"Comtype"`
	Rack         int    `json:"Rack"`
	Slot         int    `json:"Slot"`
	Ports      int    `json:"Ports"`
	WritePort  int    `json:"Write port"`
	AlarmPort  int    `json:"Alarm port"`
	Maker string `json:"Maker"`
	ID    string `json:"Id"`
}

type PLCFile struct {
	PLCs []PLCConfig `json:"plcs"`
}

func SeedRoles() {
	roles := []models.Role{
		{Name: "superadmin", Label: "Superadmin", Description: "Full system access", Level: 1},
		{Name: "co-admin", Label: "Co-Admin", Description: "Limited administrative access", Level: 5},
		{Name: "user", Label: "User", Description: "Basic user with read access", Level: 10},
	}

	for _, r := range roles {
		var existing models.Role
		if DB.Where("name = ?", r.Name).First(&existing).RowsAffected == 0 {
			DB.Create(&r)
			log.Println("✅ Seeded role:", r.Name)
		}
	}

	var adminRole models.Role
	DB.Where("name = ?", "superadmin").First(&adminRole)

	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{
			Email:    "admin@example.com",
			Password: string(hashed),
			Name:     "Super Admin",
			RoleID:   adminRole.ID,
			Active:   true,
		}
		DB.Create(&admin)
		log.Println("✅ Seeded default superadmin: admin@example.com / admin123")
	}
}

func SeedPLCs() {
	data, err := os.ReadFile("config/plc_info.json")
	if err != nil {
		log.Println("⚠️ plc_info.json not found, skipping seeder")
		return
	}

	var file PLCFile
	if err := json.Unmarshal(data, &file); err != nil {
		log.Println("❌ JSON parse error:", err)
		return
	}

	for _, p := range file.PLCs {

		ip := strings.TrimSpace(p.IPAddress)
		if ip == "" {
			continue
		}

		var existing models.MitsubishiPlc

		err := DB.Where("ip_address = ?", ip).First(&existing).Error

		// NOT FOUND → INSERT
		if errors.Is(err, gorm.ErrRecordNotFound) {

			newPLC := models.MitsubishiPlc{
				FacilityName: p.FacilityName,
				Driver:       p.Driver,
				IpAddress:    ip,
				ComType:      p.ComType,
				Rack:         p.Rack,
				Slot:         p.Slot,

				Port: p.Ports,
				WritePort: p.WritePort,
				AlarmPort: p.AlarmPort,

				Maker: p.Maker,
			}

			if err := DB.Create(&newPLC).Error; err != nil {
				log.Println("❌ Insert failed:", ip, err)
				continue
			}

			log.Println("✅ Seeded PLC:", ip)
			continue
		} else if err == nil {
			// EXISTING → UPDATE
			existing.FacilityName = p.FacilityName
			existing.Driver = p.Driver
			existing.ComType = p.ComType
			existing.Rack = p.Rack
			existing.Slot = p.Slot
			existing.Port = p.Ports
			existing.WritePort = p.WritePort
			existing.AlarmPort = p.AlarmPort
			existing.Maker = p.Maker

			if err := DB.Save(&existing).Error; err != nil {
				log.Println("❌ Update failed:", ip, err)
			} else {
				log.Println("✅ Updated existing PLC:", ip)
			}
			continue
		}

		// real DB error
		if err != nil {
			log.Println("❌ DB error:", ip, err)
			continue
		}
	}

	SeedCSVTagLists()
}

func SeedCSVTagLists() {
	csvPath := "deployments/seed_data/public.mitsubishi_tag_lists.csv"
	f, err := os.Open(csvPath)
	if err != nil {
		log.Println("ℹ️ public.mitsubishi_tag_lists.csv not found, skipping tag import:", err)
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		log.Println("❌ CSV parse error:", err)
		return
	}

	if len(records) < 2 {
		log.Println("ℹ️ CSV has no data rows")
		return
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[strings.TrimSpace(h)] = i
	}

	required := []string{"facility_name", "tag_name", "tag_address"}
	for _, r := range required {
		if _, ok := headers[r]; !ok {
			log.Printf("❌ CSV missing required column: %s", r)
			return
		}
	}

	facilityIdx := headers["facility_name"]
	robotIDIdx := headers["robot_id"]
	plcIDIdx := headers["plc_id"]
	tagNameIdx := headers["tag_name"]
	tagAddrIdx := headers["tag_address"]
	commentIdx := headers["comment"]
	dataTypeIdx := headers["data_type"]
	actionIdx := headers["action"]
	screenIdx := headers["screen"]
	svgElemIdx := headers["svg_element"]
	trueColorIdx := headers["true_condition_color"]
	falseColorIdx := headers["false_condition_color"]
	blinkingIdx := headers["blinking"]
	refreshRateIdx := headers["refresh_rate"]

	type plcInfo struct {
		ID           string
		FacilityName string
	}
	var plcs []plcInfo
	DB.Model(&models.MitsubishiPlc{}).Select("id, facility_name").Find(&plcs)
	plcByFacility := make(map[string]string)
	plcIDs := make(map[string]bool)
	for _, p := range plcs {
		plcByFacility[p.FacilityName] = p.ID
		plcIDs[p.ID] = true
	}

	var count int64
	DB.Model(&models.MitsubishiTagList{}).Count(&count)
	if count > 0 {
		log.Printf("ℹ️ %d tag_lists already exist, skipping CSV import", count)
		return
	}

	robotCreated := make(map[string]bool)
	inserted := 0
	skipped := 0

	for _, row := range records[1:] {
		facility := strings.TrimSpace(row[facilityIdx])
		plcID := ""
		if plcIDIdx >= 0 && plcIDIdx < len(row) {
			plcID = strings.TrimSpace(row[plcIDIdx])
		}

		if plcID != "" && !plcIDs[plcID] {
			if realID, ok := plcByFacility[facility]; ok {
				plcID = realID
			} else {
				skipped++
				continue
			}
		}
		if plcID == "" {
			if realID, ok := plcByFacility[facility]; ok {
				plcID = realID
			} else {
				skipped++
				continue
			}
		}

		robotID := ""
		if robotIDIdx >= 0 && robotIDIdx < len(row) {
			robotID = strings.TrimSpace(row[robotIDIdx])
		}
		if robotID != "" && !robotCreated[robotID] {
			var robotCount int64
			DB.Model(&models.MitsubishiRobot{}).Where("id = ?", robotID).Count(&robotCount)
			if robotCount == 0 {
				robot := models.MitsubishiRobot{
					PlcID: plcID,
					Name:  facility + "_Robot",
				}
				robot.ID = robotID
				if err := DB.Create(&robot).Error; err != nil {
					log.Printf("⚠️ Failed to create robot %s: %v", robotID, err)
				} else {
					log.Printf("✅ Created robot %s for %s", robotID, facility)
				}
			}
			robotCreated[robotID] = true
		}

		tagName := strings.TrimSpace(row[tagNameIdx])
		tagAddr := strings.TrimSpace(row[tagAddrIdx])
		if tagName == "" || tagAddr == "" {
			skipped++
			continue
		}

		tag := models.MitsubishiTagList{
			FacilityName: facility,
			RobotID:      robotID,
			PlcID:        plcID,
			TagName:      tagName,
			TagAddress:   tagAddr,
		}

		if commentIdx >= 0 && commentIdx < len(row) {
			tag.Comment = strings.TrimSpace(row[commentIdx])
		}
		if dataTypeIdx >= 0 && dataTypeIdx < len(row) {
			tag.DataType = strings.TrimSpace(row[dataTypeIdx])
		}
		if actionIdx >= 0 && actionIdx < len(row) {
			tag.Action = strings.TrimSpace(row[actionIdx])
		}
		if screenIdx >= 0 && screenIdx < len(row) {
			tag.Screen = strings.TrimSpace(row[screenIdx])
		}
		if svgElemIdx >= 0 && svgElemIdx < len(row) {
			tag.SvgElement = strings.TrimSpace(row[svgElemIdx]) == "true"
		}
		if trueColorIdx >= 0 && trueColorIdx < len(row) {
			tag.TrueConditionColor = strings.TrimSpace(row[trueColorIdx])
		}
		if falseColorIdx >= 0 && falseColorIdx < len(row) {
			tag.FalseConditionColor = strings.TrimSpace(row[falseColorIdx])
		}
		if blinkingIdx >= 0 && blinkingIdx < len(row) {
			tag.Blinking = strings.TrimSpace(row[blinkingIdx]) == "true"
		}
		if refreshRateIdx >= 0 && refreshRateIdx < len(row) {
			if v, err := strconv.Atoi(strings.TrimSpace(row[refreshRateIdx])); err == nil {
				tag.RefreshRate = v
			}
		}

		if err := DB.Create(&tag).Error; err != nil {
			log.Printf("⚠️ Failed to insert tag %s (%s): %v", tagName, tagAddr, err)
			skipped++
			continue
		}
		inserted++
	}

	log.Printf("✅ CSV import complete: %d inserted, %d skipped", inserted, skipped)
}
