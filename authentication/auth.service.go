package authentication

import (
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"colossa-pm/helpers"
	"colossa-pm/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSamePassword       = errors.New("new password must differ from current password")
)

type RegisterInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"fullName" binding:"required,min=4"`
}

type LoginInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshInput struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword"     binding:"required,min=8"`
}

type AuthResponse struct {
	User   *models.UsersModel `json:"user"`
	Tokens *helpers.TokenPair `json:"tokens"`
}

type Service interface {
	Register(input RegisterInput) (*AuthResponse, error)
	Login(input LoginInput) (*AuthResponse, error)
	RefreshTokens(input RefreshInput) (*helpers.TokenPair, error)
	ChangePassword(userID uuid.UUID, input ChangePasswordInput) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Register(input RegisterInput) (*AuthResponse, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.UsersModel{
		Email:    input.Email,
		Password: string(hashed),
		FullName: input.FullName,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	tokens, err := helpers.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: user, Tokens: tokens}, nil
}

func (s *service) Login(input LoginInput) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(input.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := helpers.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: user, Tokens: tokens}, nil
}

func (s *service) RefreshTokens(input RefreshInput) (*helpers.TokenPair, error) {
	claims, err := helpers.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Confirm user still exists
	_, err = s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	return helpers.GenerateTokenPair(claims.UserID)
}

func (s *service) ChangePassword(userID uuid.UUID, input ChangePasswordInput) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.CurrentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Prevent reuse of the same password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.NewPassword)); err == nil {
		return ErrSamePassword
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(userID, string(hashed))
}
