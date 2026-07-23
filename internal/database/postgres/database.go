package postgres

import (
	"fmt"
	"log"
	"os"
	"time"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

var DB *gorm.DB

func ConnectDB() {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	// ✅ Fail fast if critical values missing
	if host == "" {
		host = "postgres"
	}
	if user == "" || password == "" || dbname == "" {
		log.Fatal("❌ Missing required DB environment variables (DB_USER, DB_PASSWORD, DB_NAME)")
	}
	if port == "" {
		port = "5432"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
		host, user, password, dbname, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})

	if err != nil {
		log.Fatal("❌ Failed to connect database:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("❌ DB instance error:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	fmt.Println("✅ Connected to Database")
	DB = db

	// AutoMigrate the models
	err = DB.AutoMigrate(
		&models.User{},
		&models.MitsubishiPlc{},
		&models.MitsubishiRobotModel{},
		&models.MitsubishiRobot{},
		&models.MitsubishiTagList{},
		&models.MitsubishiTagValue{},
	)
	if err != nil {
		log.Println("⚠️ Failed to auto-migrate:", err)
	}
}