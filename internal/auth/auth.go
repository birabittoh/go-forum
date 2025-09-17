package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"

	"goforum/internal/config"
	"goforum/internal/models"
)

type Service struct {
	db     *gorm.DB
	config *config.Config
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
	}
}

func (s *Service) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *Service) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *Service) GenerateToken(userID uint) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 30)), // 30 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *Service) Register(username, email, password string) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("username = ? OR email = ?", username, email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user with this username or email already exists")
	}

	// Hash password
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Generate verification token
	verificationToken, err := s.generateRandomToken()
	if err != nil {
		return nil, err
	}

	// Determine user type (first user is admin)
	var userCount int64
	s.db.Model(&models.User{}).Count(&userCount)

	userType := models.UserTypeUnverified

	if userCount == 0 {
		userType = models.UserTypeAdmin
	}

	// Create user
	user := &models.User{
		Username:          username,
		Email:             email,
		PasswordHash:      hashedPassword,
		UserType:          userType,
		VerificationToken: verificationToken,
		Theme:             "default",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	// Send verification email (if not the first user)
	if userCount > 0 {
		if err := s.SendVerificationEmail(user); err != nil {
			// Log error but don't fail registration
			fmt.Printf("Failed to send verification email: %v\n", err)
		}
	}

	return user, nil
}

func (s *Service) Login(username, password string) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("username = ? OR email = ?", username, username).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, "", errors.New("invalid credentials")
	}

	if !user.IsActive() {
		return nil, "", errors.New("account is banned")
	}

	token, err := s.GenerateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

func (s *Service) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) VerifyEmail(token string) error {
	var user models.User
	if err := s.db.Where("verification_token = ?", token).First(&user).Error; err != nil {
		return errors.New("invalid verification token")
	}

	if user.IsVerified() {
		return errors.New("email already verified")
	}

	user.VerificationToken = ""
	user.UserType = models.UserTypeUser // Promote to regular user

	return s.db.Save(&user).Error
}

func (s *Service) SendVerificationEmail(user *models.User) error {
	if s.config.SMTPHost == "" || s.config.SMTPUsername == "" {
		return errors.New("email configuration not set")
	}

	verificationURL := fmt.Sprintf("%s/auth/verify/%s", s.config.SiteURL, user.VerificationToken)

	subject := fmt.Sprintf("Verify your email - %s", s.config.SiteName)
	body := fmt.Sprintf(`
Hello %s,

Please click the following link to verify your email address:
%s

If you didn't create an account, please ignore this email.

Best regards,
%s Team
`, user.Username, verificationURL, s.config.SiteName)

	m := gomail.NewMessage()
	m.SetHeader("From", s.config.FromEmail)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(s.config.SMTPHost, s.config.SMTPPort, s.config.SMTPUsername, s.config.SMTPPassword)

	return d.DialAndSend(m)
}

func (s *Service) generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
