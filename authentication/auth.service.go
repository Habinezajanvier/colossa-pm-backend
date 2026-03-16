package authentication

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"colossa-pm/email"
	"colossa-pm/helpers"
	"colossa-pm/logger"
	"colossa-pm/messaging"
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

type RegisterResponse struct {
	UserID  string `json:"userId"`
	Message string `json:"message"`
}

type VerifyEmailInput struct {
	UserID string `json:"-"`
	OTP    string `json:"otp" binding:"required,len=6"`
}

type RequestChangePasswordInput struct {
	Email string `json:"email" binding:"required,email"`
}

type ConfirmChangePasswordInput struct {
	Email       string `json:"email"    binding:"required,email"`
	OTP         string `json:"otp"             binding:"required,len=6"`
	NewPassword string `json:"newPassword"     binding:"required,min=8"`
}

type Service interface {
	Register(input RegisterInput) (*RegisterResponse, error)
	VerifyEmail(input VerifyEmailInput) (*AuthResponse, error)
	Login(input LoginInput) (*AuthResponse, error)
	RefreshTokens(input RefreshInput) (*helpers.TokenPair, error)
	RequestChangePassword(input RequestChangePasswordInput) (*models.UsersModel, error)
	ConfirmChangePassword(input ConfirmChangePasswordInput) (*models.UsersModel, error)
}

type service struct {
	repo        Repository
	tokenRepo   TokenRepository
	mailer      email.Mailer
	messageRepo messaging.Repository
}

type OtpMessagingDto struct {
	Email     string
	Username  string
	Otp       string
	EmailType email.OTPEmailType
	UserID    uuid.UUID
	EventID   *uuid.UUID
	EventType *string
}

func NewService(repo Repository, tokenRepo TokenRepository, mailer email.Mailer, messageRepo messaging.Repository) Service {
	return &service{repo: repo, tokenRepo: tokenRepo, mailer: mailer, messageRepo: messageRepo}
}

func (s *service) sendMessage(msgOpt *OtpMessagingDto) {

	msg := &models.MessageModel{
		UserID:    msgOpt.UserID,
		Recipient: msgOpt.Email,
		Type:      messaging.MessageTypeEmailVerification,
		Status:    messaging.MessageStatusSent,
		EventType: msgOpt.EventType,
		EventID:   msgOpt.EventID,
	}

	if err := s.mailer.SendOTP(msgOpt.Email, msgOpt.Username, msgOpt.Otp, msgOpt.EmailType); err != nil {
		errStr := err.Error()
		msg.Status = messaging.MessageStatusFailed
		msg.Error = &errStr
		logger.Instance().ErrorMsg("failed to send verification email to " + msgOpt.Email + ": " + errStr)
	} else {
		logger.Instance().Log("verification email sent to " + msgOpt.Email)
	}
	if err := s.messageRepo.Save(msg); err != nil {
		logger.Instance().ErrorMsg("failed to save message log: " + err.Error())
	}
}

func (s *service) Register(input RegisterInput) (*RegisterResponse, error) {

	existingUser, err := s.repo.FindByEmail(input.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	var user *models.UsersModel

	if existingUser != nil && !existingUser.IsVerified {
		latest, err := s.tokenRepo.FindLatest(existingUser.ID, TokenTypeEmailVerification)
		if err != nil && !errors.Is(err, ErrTokenNotFound) {
			return nil, err
		}
		if latest != nil && time.Since(latest.CreatedAt) < 10*time.Minute {
			return nil, ErrResendTooSoon
		}
		user = existingUser
	} else if existingUser != nil && existingUser.IsVerified {
		return nil, ErrEmailTaken
	} else {
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user = &models.UsersModel{
			Email:    input.Email,
			Password: string(hashed),
			FullName: input.FullName,
		}
		if err := s.repo.Create(user); err != nil {
			return nil, err
		}
	}

	vt, rawOTP, err := s.tokenRepo.Create(user.ID, TokenTypeEmailVerification)
	if err != nil {
		return nil, err
	}

	eventType := string(TokenTypeEmailVerification)
	capturedVT := vt
	go s.sendMessage(&OtpMessagingDto{
		UserID:    user.ID,
		Username:  user.FullName,
		Email:     user.Email,
		Otp:       rawOTP,
		EventType: &eventType,
		EventID:   &capturedVT.ID,
		EmailType: email.OTPEmailType(messaging.MessageTypeEmailVerification),
	})

	return &RegisterResponse{
		UserID:  user.ID.String(),
		Message: "registration successful, please check your email for a verification code",
	}, nil
}

// VerifyEmail validates the OTP, marks the user verified, and returns tokens.
func (s *service) VerifyEmail(input VerifyEmailInput) (*AuthResponse, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	vt, err := s.tokenRepo.FindValid(userID, input.OTP, TokenTypeEmailVerification)
	if err != nil {
		return nil, err
	}

	if err := s.tokenRepo.MarkUsed(vt.ID); err != nil {
		return nil, err
	}

	if err := s.repo.MarkVerified(userID); err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	tokens, err := helpers.GenerateTokenPair(user.ID, user.FullName)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: user, Tokens: tokens}, nil
}

// Login checks credentials and that the email is verified before issuing tokens.
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

	if !user.IsVerified {
		return nil, ErrNotVerified
	}

	tokens, err := helpers.GenerateTokenPair(user.ID, user.FullName)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: user, Tokens: tokens}, nil
}

// RefreshTokens issues a new token pair from a valid refresh token.
func (s *service) RefreshTokens(input RefreshInput) (*helpers.TokenPair, error) {
	claims, err := helpers.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	return helpers.GenerateTokenPair(claims.UserID, claims.FullName)
}

// RequestChangePassword sends an OTP to the user email to authorize a password change.
func (s *service) RequestChangePassword(input RequestChangePasswordInput) (*models.UsersModel, error) {
	user, err := s.repo.FindByEmail(input.Email)
	if err != nil {
		// Return nil even if not found to avoid email enumeration
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if !user.IsVerified {
		return nil, ErrNotVerified
	}

	vt, rawOTP, err := s.tokenRepo.Create(user.ID, TokenTypeChangePassword)
	if err != nil {
		return nil, err
	}

	eventType := string(TokenTypeChangePassword)
	capturedVT := vt
	go s.sendMessage(&OtpMessagingDto{
		UserID:    user.ID,
		Username:  user.FullName,
		Email:     user.Email,
		Otp:       rawOTP,
		EventType: &eventType,
		EventID:   &capturedVT.ID,
		EmailType: email.OTPEmailType(messaging.MessageTypeChangePassword),
	})

	return user, nil
}

// ConfirmChangePassword validates the OTP then updates the password.
func (s *service) ConfirmChangePassword(input ConfirmChangePasswordInput) (*models.UsersModel, error) {
	user, err := s.repo.FindByEmail(input.Email)

	if err != nil {
		return nil, err
	}

	vt, err := s.tokenRepo.FindValid(user.ID, input.OTP, TokenTypeChangePassword)
	if err != nil {
		return nil, err
	}

	// Prevent reuse of the same password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.NewPassword)); err == nil {
		return nil, ErrSamePassword
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if err := s.tokenRepo.MarkUsed(vt.ID); err != nil {
		return nil, err
	}

	return user, s.repo.UpdatePassword(user.ID, string(hashed))
}
