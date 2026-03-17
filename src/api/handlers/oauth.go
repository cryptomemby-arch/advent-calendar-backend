package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/advent-calendar-backend/src/configs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
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
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

func getMicrosoftConfig(cfg *configs.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.OauthMicrosoft.ClientID,
		ClientSecret: cfg.OauthMicrosoft.ClientSecret,
		RedirectURL:  "http://localhost:8080/auth/microsoft/callback",
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

		resp, _ := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
		var gUser GoogleUser
		json.NewDecoder(resp.Body).Decode(&gUser)

		processOAuthUser(c, gUser.Email, gUser.Name, jwtKey)
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
		resp, _ := client.Get("https://graph.microsoft.com/v1.0/me")
		var msUser MicrosoftUser
		json.NewDecoder(resp.Body).Decode(&msUser)

		email := msUser.Mail
		if email == "" {
			email = msUser.UserPrincipalName
		}

		processOAuthUser(c, email, msUser.DisplayName, jwtKey)
	}
}

func processOAuthUser(c *gin.Context, email, name string, jwtKey []byte) {
	db := c.MustGet("db").(*sql.DB)

	var username string
	err := db.QueryRow("SELECT username FROM users WHERE email = $1", email).Scan(&username)

	if err == sql.ErrNoRows {
		username = email
		_, err = db.Exec("INSERT INTO users (username, email, password, country) VALUES ($1, $2, $3, $4)",
			username, email, "", "Unknown")
		if err != nil {
			zap.L().Error("DB Error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
			return
		}
	}

	finalToken, _ := generateJWT(username, jwtKey)

	frontendURL := fmt.Sprintf("http://localhost:3000/login-success?token=%s", finalToken)
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}
