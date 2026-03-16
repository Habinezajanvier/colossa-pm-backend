package users

import (
	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
)

// UserProfile is a safe public representation — no sensitive fields exposed
type UserProfile struct {
	ID         uuid.UUID `json:"id"`
	FullName   string    `json:"fullName"`
	Email      string    `json:"email"`
	IsVerified bool      `json:"isVerified"`
}

type Service interface {
	GetUsers(params helpers.PaginationParams) (*helpers.PaginatedResult[models.UsersModel], error)
	GetProfile(userID uuid.UUID) (*UserProfile, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetUsers(params helpers.PaginationParams) (*helpers.PaginatedResult[models.UsersModel], error) {
	return s.repo.FindAll(params)
}

func (s *service) GetProfile(userID uuid.UUID) (*UserProfile, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	return &UserProfile{
		ID:         user.ID,
		FullName:   user.FullName,
		Email:      user.Email,
		IsVerified: user.IsVerified,
	}, nil
}
