package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Aud           string `json:"aud"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
}

type GoogleOAuthService struct {
	ClientID     string
	ClientSecret string
}

func NewGoogleOAuthService() *GoogleOAuthService {
	return &GoogleOAuthService{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	}
}

// VerifyToken verifies a Google ID token and returns user info
func (s *GoogleOAuthService) VerifyToken(idToken string) (*GoogleUserInfo, error) {
	// Exchange ID token for user info using Google's tokeninfo endpoint
	// Or use the OAuth2 userinfo endpoint
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Google token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid Google token: %s", string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	// Verify the token is issued to our client
	if userInfo.ID == "" {
		return nil, fmt.Errorf("invalid Google token: missing user ID")
	}

	// Validate audience (aud) matches our client ID
	if s.ClientID != "" && userInfo.Aud != s.ClientID {
		return nil, fmt.Errorf("invalid Google token: audience mismatch")
	}

	return &userInfo, nil
}

// VerifyAccessToken verifies a Google access token and returns user info
func (s *GoogleOAuthService) VerifyAccessToken(accessToken string) (*GoogleUserInfo, error) {
	url := "https://www.googleapis.com/oauth2/v2/userinfo"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Google access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid Google access token: %s", string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	return &userInfo, nil
}
