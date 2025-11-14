package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wtppaul/course-service/internal/models"
)

var DB *gorm.DB

func InitDB() {
	// 🎈 1. Baca environment variable yang diset di docker-compose.yml
	host := os.Getenv("DATABASE_HOST")
	user := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	dbname := os.Getenv("DATABASE_NAME")
	port := os.Getenv("DATABASE_PORT")
	sslmode := os.Getenv("DATABASE_SSL_MODE")

	// 🎈 2. Bangun DSN (Data Source Name) dari env vars
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)

	// 🎈 3. Cek apakah env var yang penting ada (cek host saja cukup)
	if host == "" {
		log.Fatal("❌ Database configuration not set (check DATABASE_HOST etc.)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v. DSN: %s", err, dsn) // Tambahkan DSN ke log error
	}

	// 🎈 4. Enable uuid-ossp extension (safe if already exists)
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error; err != nil {
		log.Fatalf("❌ Failed to enable uuid-ossp extension: %v", err)
	}

	// 🎈 5. Auto migrate all models
	fmt.Println("Running migrations...")
	if err := db.AutoMigrate(
		&models.Course{},
		&models.Chapter{},
		&models.Lesson{},
		&models.Category{},
		&models.Tag{},
		&models.Sale{},
		&models.Coupon{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}
	fmt.Println("✅ Migration done!")

	DB = db
	fmt.Println("✅ Database connected & migrated successfully.")
}