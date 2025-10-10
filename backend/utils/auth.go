package utils

import (
	"AIGE/config"
	"AIGE/models"
	"crypto/rand"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var JWTSecret = []byte("your-secret-key-change-this-in-production")

func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedBytes), err
}

func CheckPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func GenerateJWT(userID uint, username string, isAdmin bool) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

func GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func CreateDefaultAdmin() {
	var count int64
	config.DB.Model(&models.User{}).Where("is_admin = ?", true).Count(&count)
	
	if count == 0 {
		hashedPassword, err := HashPassword("admin123")
		if err != nil {
			log.Fatal("创建默认管理员失败:", err)
		}

		admin := models.User{
			Username: "admin",
			Password: hashedPassword,
			Email:    "admin@example.com",
			IsAdmin:  true,
		}

		if err := config.DB.Create(&admin).Error; err != nil {
			log.Fatal("创建默认管理员失败:", err)
		}

		log.Println("默认管理员账号已创建 - 用户名: admin, 密码: admin123")
	}
}