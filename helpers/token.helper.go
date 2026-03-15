package helpers

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrTokenExpired = errors.New("token has expired")
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID    uuid.UUID `json:"userId"`
	TokenType TokenType `json:"tokenType"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func accessSecret() []byte {
	return []byte(os.Getenv("JWT_ACCESS_SECRET"))
}

func refreshSecret() []byte {
	return []byte(os.Getenv("JWT_REFRESH_SECRET"))
}

func accessTTL() time.Duration {
	minutes, err := strconv.Atoi(os.Getenv("JWT_ACCESS_TTL_MINUTES"))
	if err != nil || minutes == 0 {
		return 15 * time.Minute // default: 15 minutes
	}
	return time.Duration(minutes) * time.Minute
}

func refreshTTL() time.Duration {
	days, err := strconv.Atoi(os.Getenv("JWT_REFRESH_TTL_DAYS"))
	if err != nil || days == 0 {
		return 7 * 24 * time.Hour // default: 7 days
	}
	return time.Duration(days) * 24 * time.Hour
}

// GenerateTokenPair issues a new access + refresh token pair for a user
func GenerateTokenPair(userID uuid.UUID) (*TokenPair, error) {
	accessToken, err := generateToken(userID, AccessToken, accessTTL(), accessSecret())
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateToken(userID, RefreshToken, refreshTTL(), refreshSecret())
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func generateToken(userID uuid.UUID, tokenType TokenType, ttl time.Duration, secret []byte) (string, error) {
	claims := Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateAccessToken parses and validates an access token, returning its claims
func ValidateAccessToken(tokenStr string) (*Claims, error) {
	return validateToken(tokenStr, AccessToken, accessSecret())
}

// ValidateRefreshToken parses and validates a refresh token, returning its claims
func ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return validateToken(tokenStr, RefreshToken, refreshSecret())
}

func validateToken(tokenStr string, expectedType TokenType, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
