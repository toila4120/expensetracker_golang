package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Không tìm thấy file .env (Có thể bạn đang chạy trên Server Production)")
	}

	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		log.Fatal("❌ Lỗi: DATABASE_URL chưa được thiết lập!")
	}

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Không thể kết nối Database: ", err)
	}

	log.Println("✅ Kết nối Neon PostgreSQL thành công thông qua biến môi trường!")

	DB = database
}
