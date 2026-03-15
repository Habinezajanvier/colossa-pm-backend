package authentication

import (
	"colossa-pm/models"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TokenTypeEmailVerification models.TokenType = "email_verification"
	TokenTypeChangePassword    models.TokenType = "change_password"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenUsed     = errors.New("token has already been used")
	ErrTokenExpired  = errors.New("token has expired")
	ErrNotVerified   = errors.New("email not verified")
)

func generateOTP() (string, error) {
	const digits = "0123456789"
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}

func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(sum[:])
}

type TokenRepository interface {
	Create(userID uuid.UUID, tokenType models.TokenType) (*models.VerificationTokenModel, string, error)
	FindValid(userID uuid.UUID, token string, tokenType models.TokenType) (*models.VerificationTokenModel, error)
	MarkUsed(id uuid.UUID) error
	InvalidatePrevious(userID uuid.UUID, tokenType models.TokenType) error
}

type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(userID uuid.UUID, tokenType models.TokenType) (*models.VerificationTokenModel, string, error) {
	// Invalidate any existing unused tokens of the same type for this user
	// so only one active OTP exists at a time
	if err := r.InvalidatePrevious(userID, tokenType); err != nil {
		return nil, "", err
	}

	otp, err := generateOTP()
	if err != nil {
		return nil, "", err
	}

	vt := &models.VerificationTokenModel{
		UserID:    userID,
		Token:     hashOTP(otp),
		Type:      tokenType,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := r.db.Create(vt).Error; err != nil {
		return nil, "", err
	}

	// Return raw OTP to caller so it can be emailed — we never store it
	return vt, otp, nil
}

func (r *tokenRepository) FindValid(userID uuid.UUID, token string, tokenType models.TokenType) (*models.VerificationTokenModel, error) {
	var vt models.VerificationTokenModel
	err := r.db.Where(
		"user_id = ? AND token = ? AND type = ? AND used = FALSE AND expires_at > ?",
		userID, hashOTP(token), tokenType, time.Now(),
	).First(&vt).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTokenNotFound
	}
	return &vt, err
}

func (r *tokenRepository) MarkUsed(id uuid.UUID) error {
	return r.db.Model(&models.VerificationTokenModel{}).
		Where("id = ?", id).
		Update("used", true).Error
}

func (r *tokenRepository) InvalidatePrevious(userID uuid.UUID, tokenType models.TokenType) error {
	return r.db.Model(&models.VerificationTokenModel{}).
		Where("user_id = ? AND type = ? AND used = FALSE", userID, tokenType).
		Update("used", true).Error
}
