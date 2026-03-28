package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/advent-calendar-backend/src/configs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
	"gorm.io/gorm"
)

type GoogleUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type MicrosoftUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

func generateStateCookie(c *gin.Context) string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	c.SetCookie("oauth_state", state, 900, "/", "", false, true)
	return state
}

func getGoogleConfig(cfg *configs.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.OauthGoogle.ClientID,
		ClientSecret: cfg.OauthGoogle.ClientSecret,
		RedirectURL:  fmt.Sprintf("http://%s/auth/google/callback", cfg.Origin.OriginBack),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

func getMicrosoftConfig(cfg *configs.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.OauthMicrosoft.ClientID,
		ClientSecret: cfg.OauthMicrosoft.ClientSecret,
		RedirectURL:  fmt.Sprintf("http://%s/auth/microsoft/callback", cfg.Origin.OriginBack),
		Scopes:       []string{"openid", "profile", "email", "User.Read"},
		Endpoint:     microsoft.AzureADEndpoint("common"),
	}
}

// --- GOOGLE ---

func GoogleLogin(cfg *configs.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := getGoogleConfig(cfg).AuthCodeURL(generateStateCookie(c))
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func GoogleCallback(cfg *configs.Config, jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		state, _ := c.Cookie("oauth_state")
		if state == "" || state != c.Query("state") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid CSRF state"})
			return
		}

		token, err := getGoogleConfig(cfg).Exchange(context.Background(), c.Query("code"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Exchange failed"})
			return
		}

		resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer resp.Body.Close()

		var gUser GoogleUser
		json.NewDecoder(resp.Body).Decode(&gUser)

		processOAuthUser(c, gUser.Email, gUser.Name, jwtKey, cfg)
	}
}

// --- MICROSOFT ---

func MicrosoftLogin(cfg *configs.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := getMicrosoftConfig(cfg).AuthCodeURL(generateStateCookie(c))
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func MicrosoftCallback(cfg *configs.Config, jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		state, _ := c.Cookie("oauth_state")
		if state == "" || state != c.Query("state") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid CSRF state"})
			return
		}

		conf := getMicrosoftConfig(cfg)
		token, err := conf.Exchange(context.Background(), c.Query("code"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Exchange failed"})
			return
		}

		client := conf.Client(context.Background(), token)
		resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer resp.Body.Close()

		var msUser MicrosoftUser
		json.NewDecoder(resp.Body).Decode(&msUser)

		email := msUser.Mail
		if email == "" {
			email = msUser.UserPrincipalName
		}

		processOAuthUser(c, email, msUser.DisplayName, jwtKey, cfg)
	}
}

// --- LOGIC ---

func processOAuthUser(c *gin.Context, email, name string, jwtKey []byte, cfg *configs.Config) {
	db := c.MustGet("db").(*gorm.DB)

	var user User
	err := db.Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = User{
				Username: name,
				Email:    email,
				Password: "",
				Country:  "Unknown",
			}

			if user.Username == "" {
				user.Username = email
			}

			if err := db.Create(&user).Error; err != nil {
				zap.L().Error("DB Error while creating OAuth user", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				return
			}
		} else {
			zap.L().Error("DB Error while finding OAuth user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	}

	finalToken, err := generateJWT(user.ID, user.Username, jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	frontendURL := fmt.Sprintf("http://%s/login-success?token=%s", cfg.Origin.OriginFront, finalToken)
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}
