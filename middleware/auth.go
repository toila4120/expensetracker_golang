// File: middleware/auth.go
// Mục tiêu: Lớp khiên bảo vệ các API riêng tư (ví dụ: thêm giao dịch, xem báo cáo). Chỉ cho phép user có token đi qua.
//
// Nhiệm vụ của bạn:
// 1. Viết một middleware function trả về `gin.HandlerFunc`.
// 2. Lấy token từ header của request (thường nằm ở header `Authorization: Bearer <token>`).
// 3. Sử dụng utils.ValidateToken để kiểm tra token.
// 4. Nếu hợp lệ: Lưu userID vào context của Gin (c.Set) và cho phép đi tiếp (c.Next).
// 5. Nếu không hợp lệ (không có token, hoặc token hết hạn/sai): Trả về lỗi 401 Unauthorized và chặn lại ngay lập tức (c.Abort).
//
// Kiến thức cần học:
// - Middleware trong web framework là gì? Luồng xử lý request qua Middleware.
// - Cách hoạt động của Gin Middleware (c.Next() vs c.Abort()).
// - Cách truyền dữ liệu giữa middleware và controller bằng Context (c.Set() và c.Get()).

package middleware

import (
	"expensetracker/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Yêu cầu cung cấp token xác thực"})
			ctx.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Định dạng token không hợp lệ (Phải là Bearer <token>)"})
			ctx.Abort()
			return
		}
		tokenString := parts[1]
		userID, err := utils.ValidateToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Token không hợp lệ hoặc đã hết hạn",
				"details": err.Error(),
			})
			ctx.Abort()
			return
		}
		ctx.Set("currentUserID", userID)
		ctx.Next()
	}
}
