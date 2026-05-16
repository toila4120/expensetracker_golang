package main

import (
	"expensetracker/models"
	"expensetracker/routes"
	"log"
	"os"

	"github.com/gin-contrib/cors"
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

	database, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		PrepareStmt: false,
	})
	if err != nil {
		log.Fatal("❌ Không thể kết nối Database: ", err)
	}
	log.Println("✅ Kết nối Neon PostgreSQL thành công thông qua biến môi trường!")
	DB = database

	// Tự động Migrate các Model để đồng bộ schema với Database (tạo bảng, thêm cột còn thiếu)
	err = database.AutoMigrate(&models.User{}, &models.Transaction{}, &models.Budget{}, &models.RecurringTransaction{})
	if err != nil {
		log.Println("⚠️ Lỗi khi AutoMigrate:", err)
	}

	r := gin.Default()
	
	// Cấu hình CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	routes.SetupRoutes(r, DB)

	log.Println("🚀 Server đang khởi chạy tại cổng:", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("❌ Lỗi khi khởi chạy Server: ", err)
	}
}
