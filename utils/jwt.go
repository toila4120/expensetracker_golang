// File: utils/jwt.go
// Mục tiêu: Tạo và giải mã JSON Web Token (JWT).
//
// Nhiệm vụ của bạn:
// 1. Viết hàm GenerateToken(userID uint) trả về chuỗi token. Cần thiết lập thời gian hết hạn (Expiration time - ví dụ 24h).
// 2. Viết hàm ValidateToken(tokenString string) để kiểm tra token có hợp lệ không và trích xuất userID từ payload của token.
//
// Kiến thức cần học:
// - JWT là gì? Cấu trúc của JWT gồm những phần nào (Header, Payload, Signature)?
// - Cài đặt và sử dụng thư viện `github.com/golang-jwt/jwt/v5`.
// - Khái niệm secret key (khóa bí mật) để ký token. Bảo mật chuỗi secret này như thế nào?

package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func getSecretKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback_secret_key_just_in_case"
	}
	return []byte(secret)
}

type CustomClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint) (string, error) {
	return GenerateAccessToken(userID)
}

func GenerateAccessToken(userID uint) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(getSecretKey())
}

func GenerateRefreshToken(userID uint) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(getSecretKey())
}

func ValidateToken(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("thuật toán không hợp lệ")
		}

		return getSecretKey(), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims.UserID, nil
	}

	return 0, errors.New("token không hợp lệ")
}
