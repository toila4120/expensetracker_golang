package services

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"expensetracker/models"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

type FCMService struct {
	messaging *messaging.Client
	db        *gorm.DB
}

func NewFCMService(db *gorm.DB) *FCMService {
	ctx := context.Background()

	// Cách 1: Đọc từ file (local development)
	credPath := os.Getenv("FIREBASE_CREDENTIALS")
	if credPath != "" {
		opt := option.WithCredentialsFile(credPath)
		app, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			log.Println("⚠️ Firebase init error:", err)
			return &FCMService{db: db}
		}

		client, err := app.Messaging(ctx)
		if err != nil {
			log.Println("⚠️ FCM client error:", err)
			return &FCMService{db: db}
		}

		log.Println("✅ FCM Service initialized (from file)")
		return &FCMService{messaging: client, db: db}
	}

	// Cách 2: Đọc từ base64 env var (Render, production)
	credBase64 := os.Getenv("FIREBASE_CREDENTIALS_BASE64")
	if credBase64 != "" {
		jsonData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(credBase64))
		if err != nil {
			log.Println("⚠️ Firebase base64 decode error:", err)
			return &FCMService{db: db}
		}

		opt := option.WithCredentialsJSON(jsonData)
		app, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			log.Println("⚠️ Firebase init error:", err)
			return &FCMService{db: db}
		}

		client, err := app.Messaging(ctx)
		if err != nil {
			log.Println("⚠️ FCM client error:", err)
			return &FCMService{db: db}
		}

		log.Println("✅ FCM Service initialized (from base64)")
		return &FCMService{messaging: client, db: db}
	}

	log.Println("⚠️ FIREBASE_CREDENTIALS or FIREBASE_CREDENTIALS_BASE64 not set, FCM disabled")
	return &FCMService{db: db}
}

func (s *FCMService) SendToUser(userID uint, title, body string, data map[string]string) error {
	if s.messaging == nil {
		return nil
	}

	var tokens []models.FCMToken
	if err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&tokens).Error; err != nil {
		return err
	}

	if len(tokens) == 0 {
		return nil
	}

	var tokenStrings []string
	for _, t := range tokens {
		tokenStrings = append(tokenStrings, t.Token)
	}

	ctx := context.Background()
	msg := &messaging.MulticastMessage{
		Tokens: tokenStrings,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	resp, err := s.messaging.SendEachForMulticast(ctx, msg)
	if err != nil {
		log.Println("FCM send error:", err)
		return err
	}

	log.Printf("FCM sent: %d success, %d failure", resp.SuccessCount, resp.FailureCount)
	return nil
}
