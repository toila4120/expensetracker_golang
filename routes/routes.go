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

	r.HEAD("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
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

		// Wallets
		apiGroup.GET("/wallets", controllers.GetWallets(db))
		apiGroup.POST("/wallets", controllers.CreateWallet(db))
		apiGroup.GET("/wallets/:id", controllers.GetWalletDetails(db))
		apiGroup.POST("/wallets/:id/invite", controllers.InviteMember(db))

		// Financial Goals
		apiGroup.GET("/goals", controllers.GetGoals(db))
		apiGroup.POST("/goals", controllers.CreateGoal(db))
		apiGroup.GET("/goals/:id", controllers.GetGoalDetails(db))
		apiGroup.PUT("/goals/:id", controllers.UpdateGoal(db))
		apiGroup.DELETE("/goals/:id", controllers.DeleteGoal(db))
		apiGroup.POST("/goals/:id/allocate", controllers.AllocateToGoal(db))
		apiGroup.POST("/goals/:id/withdraw", controllers.WithdrawFromGoal(db))

		// Groups (Share Bill)
		apiGroup.POST("/groups", controllers.CreateGroup(db))
		apiGroup.GET("/groups", controllers.GetGroups(db))
		apiGroup.GET("/groups/:id", controllers.GetGroupDetails(db))
		apiGroup.POST("/groups/:id/members", controllers.AddMember(db))
		apiGroup.DELETE("/groups/:id/members/:member_id", controllers.RemoveMember(db))

		// Shared Bills
		apiGroup.POST("/groups/:id/bills", controllers.CreateSharedBill(db))
		apiGroup.GET("/groups/:id/bills", controllers.GetSharedBills(db))

		// Quick Bill (peer_to_peer)
		apiGroup.POST("/bills/quick", controllers.CreateQuickBill(db))

		// Balances & Settlement
		apiGroup.GET("/groups/:id/balances", controllers.GetBalances(db))
		apiGroup.POST("/groups/:id/settle", controllers.SettleDebt(db))
	}
}
