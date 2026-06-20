// File: routes/routes.go
// Mục tiêu: Định nghĩa các đường dẫn (URL endpoints) cho ứng dụng và gom nhóm API.
//
// Nhiệm vụ của bạn:
// 1. Viết hàm SetupRoutes(r *gin.Engine).
// 2. Tạo một Group route cho API công khai (không cần token):
//   - authGroup := r.Group("/auth")
//   - POST /auth/register -> gọi đến hàm controllers.Register
//   - POST /auth/login -> gọi đến hàm controllers.Login
//
// 3. Tạo một Group route cho API cần bảo vệ, sử dụng middleware `middleware.AuthMiddleware()`:
//   - apiGroup := r.Group("/api")
//   - apiGroup.Use(middleware.AuthMiddleware())
//   - GET /api/transactions -> gọi đến controllers.GetTransactions
//   - POST /api/transactions -> gọi đến controllers.CreateTransaction
//   - PUT /api/transactions/:id -> gọi đến controllers.UpdateTransaction
//   - DELETE /api/transactions/:id -> gọi đến controllers.DeleteTransaction
//
// Kiến thức cần học:
// - Route Grouping trong Gin (r.Group) để code gọn gàng, tránh lặp lại đường dẫn gốc.
// - Cách áp dụng middleware cho một group route cụ thể (group.Use).
package routes

import (
	"expensetracker/controllers"
	"expensetracker/middleware"
	"expensetracker/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes cấu hình toàn bộ các đường dẫn cho ứng dụng
func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	// Health check endpoint cho Render deploy
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Endpoint tạm thời để reset dữ liệu và tạo lại các bảng theo cách 2 (sẽ xóa sau khi hoàn thành)
	r.GET("/reset-db-dangerous", func(c *gin.Context) {
		err := db.Migrator().DropTable(&models.User{}, &models.Transaction{}, &models.Budget{}, &models.RecurringTransaction{})
		if err != nil {
			c.JSON(500, gin.H{"error": "Không thể xóa các bảng: " + err.Error()})
			return
		}

		err = db.AutoMigrate(&models.User{}, &models.Transaction{}, &models.Budget{}, &models.RecurringTransaction{})
		if err != nil {
			c.JSON(500, gin.H{"error": "Không thể tạo lại các bảng: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "Đã xóa toàn bộ dữ liệu và cấu trúc lại bảng thành công!",
		})
	})

	authGroup := r.Group("/auth")
	{
		// Cú pháp: authGroup.POST("đường_dẫn", hàm_xử_lý)
		authGroup.POST("/register", controllers.Register(db))
		authGroup.POST("/login", controllers.Login(db))
		authGroup.POST("/refresh", controllers.RefreshToken(db))
	}

	apiGroup := r.Group("/api")

	// Áp dụng Middleware cho toàn bộ các route nằm trong apiGroup
	apiGroup.Use(middleware.AuthMiddleware())

	{
		// Các route này mặc định đã được bảo vệ, chỉ ai có Token mới vào được
		apiGroup.GET("/transactions", controllers.GetAllTransaction(db))
		apiGroup.POST("/transactions", controllers.CreateTransaction(db))
		apiGroup.PUT("/transactions/:id", controllers.UpdateTransaction(db))
		apiGroup.DELETE("/transactions/:id", controllers.DeleteTransaction(db))

		// Dashboard
		apiGroup.GET("/dashboard", controllers.GetDashboard(db))

		// Profile
		apiGroup.GET("/profile", controllers.GetProfile(db))
		apiGroup.PUT("/profile", controllers.UpdateProfile(db))

		// Budget
		apiGroup.GET("/budgets", controllers.GetBudgets(db))
		apiGroup.POST("/budgets", controllers.CreateBudget(db))
		apiGroup.DELETE("/budgets/:id", controllers.DeleteBudget(db))

		// Search & Analytics
		apiGroup.GET("/transactions/search", controllers.SearchTransactions(db))
		apiGroup.GET("/analytics/categories", controllers.GetCategoryStats(db))

		// Security
		apiGroup.PUT("/change-password", controllers.ChangePassword(db))

		// Recurring Transactions
		apiGroup.GET("/recurring", controllers.GetRecurrings(db))
		apiGroup.POST("/recurring", controllers.CreateRecurring(db))
		apiGroup.PUT("/recurring/:id/toggle", controllers.ToggleRecurring(db))
		apiGroup.DELETE("/recurring/:id", controllers.DeleteRecurring(db))
	}
}
