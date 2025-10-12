package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"

	"goforum/internal/config"
	"goforum/internal/constants"
	"goforum/internal/models"
)

type Service struct {
	db     *gorm.DB
	Config *config.Config
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		Config: cfg,
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
	return token.SignedString([]byte(s.Config.JWTSecret))
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(s.Config.JWTSecret), nil
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
	lowerUsername := strings.ToLower(username)
	lowerEmail := strings.ToLower(email)
	// Check if user already exists (case-insensitive)
	var existingUser models.User
	if err := s.db.Where("LOWER(username) = ? OR LOWER(email) = ?", lowerUsername, lowerEmail).First(&existingUser).Error; err == nil {
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
	userCount, err := constants.Cache.CountAllUsers(s.db)
	if err != nil {
		return nil, err
	}

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

	if err := constants.Cache.CreateUser(user); err != nil {
		return nil, err
	}

	// Send verification email (if not the first user)
	if userCount > 0 {
		if err := s.SendVerificationEmail(user); err != nil {
			// Log error but don't fail registration
			log.Printf("Failed to send verification email: %v\n", err)
		}
	}

	return user, nil
}

func (s *Service) Login(username, password string) (*models.User, string, error) {
	user, ok := constants.Cache.GetUserByName(username)
	if !ok {
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

	return constants.Cache.UpdateUser(&user)
}

func (s *Service) SendResetPasswordEmail(user *models.User) error {
	if s.Config.SMTPHost == "" || s.Config.SMTPUsername == "" {
		return errors.New("email configuration not set")
	}
	if user.ResetToken == "" || user.ResetTokenExpiry == nil {
		return errors.New("reset token not set")
	}
	resetURL := fmt.Sprintf("%s/auth/set-password/%s", s.Config.SiteURL, user.ResetToken)
	subject := fmt.Sprintf("Reset your password - %s", s.Config.SiteName)
	body := fmt.Sprintf(`
Hello %s,

You requested a password reset. Please click the following link to set a new password (valid for 1 hour):
%s

If you didn't request this, you can ignore this email.

Best regards,
%s Team
`, user.Username, resetURL, s.Config.SiteName)

	m := gomail.NewMessage()
	m.SetHeader("From", s.Config.FromEmail)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(s.Config.SMTPHost, s.Config.SMTPPort, s.Config.SMTPUsername, s.Config.SMTPPassword)

	return d.DialAndSend(m)
}

func (s *Service) SendVerificationEmail(user *models.User) error {
	//return nil // disable email sending
	if s.Config.SMTPHost == "" || s.Config.SMTPUsername == "" {
		return errors.New("email configuration not set")
	}

	verificationURL := fmt.Sprintf("%s/auth/verify/%s", s.Config.SiteURL, user.VerificationToken)

	subject := fmt.Sprintf("Verify your email - %s", s.Config.SiteName)
	body := fmt.Sprintf(`
Hello %s,

Please click the following link to verify your email address:
%s

If you didn't create an account, please ignore this email.

Best regards,
%s Team
`, user.Username, verificationURL, s.Config.SiteName)

	m := gomail.NewMessage()
	m.SetHeader("From", s.Config.FromEmail)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(s.Config.SMTPHost, s.Config.SMTPPort, s.Config.SMTPUsername, s.Config.SMTPPassword)

	return d.DialAndSend(m)
}

func (s *Service) generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
