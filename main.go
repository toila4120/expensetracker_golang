package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Không tìm thấy file .env (Chạy trên Server Production)")
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

	r := gin.Default()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server đang khởi chạy tại cổng:", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("❌ Lỗi khi khởi chạy Server: ", err)
	}
}
