// File: utils/hash.go
// Mục tiêu: Mã hóa (Hash) mật khẩu trước khi lưu vào DB và Kiểm tra mật khẩu lúc đăng nhập.
//
// Nhiệm vụ của bạn:
// 1. Viết hàm HashPassword(password string) trả về chuỗi mật khẩu đã được băm an toàn.
// 2. Viết hàm CheckPasswordHash(password, hash string) trả về bool (đúng/sai) xem mật khẩu người dùng nhập có khớp với chuỗi băm trong DB không.
//
// Kiến thức cần học:
// - Cách cài đặt và sử dụng package `golang.org/x/crypto/bcrypt`.
// - Khái niệm băm mật khẩu (hashing) vs mã hóa (encryption). Tại sao bắt buộc phải dùng bcrypt thay vì md5 hay mã hóa 2 chiều?

package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
