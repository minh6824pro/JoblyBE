package database

import (
	entities "Jobly/internal/entities"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"time"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := os.Getenv("JOBLY_MYSQL_URI")
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatal("Failed to get sql.DB from gorm:", err)
	}

	sqlDB.SetMaxOpenConns(90)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	DB = database
	log.Println("Database connected with pool config")
}

func AutoMigrate() {
	err := DB.AutoMigrate(
		&entities.User{},
		&entities.Job{},
		&entities.Company{},
		&entities.UserTracking{},
	)

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migration successfully")
}
