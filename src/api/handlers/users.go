package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateUser(user *User) (*User, error) {
	if user == nil {
		return nil, gorm.ErrInvalidData
	}

	user.Email = strings.TrimSpace(strings.ToLower(user.Email))
	if user.Email == "" {
		return nil, gorm.ErrInvalidData
	}

	if user.Username == "" {
		user.Username = user.Email
	}

	var existing User
	if err := s.db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
		return &existing, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserByID(id uint) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetAllUsers() ([]User, error) {
	var users []User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserService) GetByEmail(email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, gorm.ErrInvalidData
	}

	var user User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetByAuthSubject(authProvider, authSubject string) (*User, error) {
	if authProvider == "" || authSubject == "" {
		return nil, gorm.ErrInvalidData
	}

	var user User
	if err := s.db.Where("auth_provider = ? AND auth_subject = ?", authProvider, authSubject).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpsertAuthUser(authProvider, authSubject, email, name, country string) (*User, error) {
	if authProvider == "" || authSubject == "" || email == "" {
		return nil, gorm.ErrInvalidData
	}

	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	country = strings.TrimSpace(country)

	var user User
	if err := s.db.Where("auth_provider = ? AND auth_subject = ?", authProvider, authSubject).First(&user).Error; err == nil {
		user.Email = email
		user.Name = name
		user.Country = country
		if err := s.db.Save(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// existing user by email should be linked
	if err := s.db.Where("email = ?", email).First(&user).Error; err == nil {
		user.AuthProvider = authProvider
		user.AuthSubject = authSubject
		user.Name = name
		user.Country = country
		if err := s.db.Save(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	newUser := User{
		Email:        email,
		Username:     email,
		Name:         name,
		Country:      country,
		AuthProvider: authProvider,
		AuthSubject:  authSubject,
	}

	if err := s.db.Create(&newUser).Error; err != nil {
		return nil, err
	}

	return &newUser, nil
}

func (s *UserService) CreateUserHandler(c *gin.Context) {
	var req User
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind user JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	created, err := s.CreateUser(&req)
	if err != nil {
		zap.L().Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	zap.L().Info("User created", zap.Uint("userId", created.ID))
	c.JSON(http.StatusCreated, created)
}

func (s *UserService) GetUserByIDHandler(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		zap.L().Warn("Invalid user ID", zap.String("idStr", idStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := s.GetUserByID(uint(id64))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("User not found", zap.Uint64("userId", id64))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		zap.L().Error("Database error getting user", zap.Uint64("userId", id64), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	zap.L().Info("User retrieved by ID", zap.Uint64("userId", id64))
	c.JSON(http.StatusOK, user)
}

func (s *UserService) GetUsersHandler(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	if email != "" {
		user, err := s.GetByEmail(email)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				zap.L().Warn("User not found by email", zap.String("email", email))
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			zap.L().Error("Database error getting user by email", zap.String("email", email), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		zap.L().Info("User retrieved by email", zap.String("email", email), zap.Uint("userId", user.ID))
		c.JSON(http.StatusOK, user)
		return
	}

	users, err := s.GetAllUsers()
	if err != nil {
		zap.L().Error("Database error getting all users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	zap.L().Info("All users retrieved", zap.Int("count", len(users)))
	c.JSON(http.StatusOK, users)
}
