package authentication

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"colossa-pm/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already in use")
)

type Repository interface {
	Create(user *models.UsersModel) error
	FindByEmail(email string) (*models.UsersModel, error)
	FindByID(id uuid.UUID) (*models.UsersModel, error)
	UpdatePassword(id uuid.UUID, hashedPassword string) error
	MarkVerified(id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(user *models.UsersModel) error {
	return r.db.Create(user).Error
}

func (r *repository) FindByEmail(email string) (*models.UsersModel, error) {
	var user models.UsersModel
	err := r.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *repository) FindByID(id uuid.UUID) (*models.UsersModel, error) {
	var user models.UsersModel
	err := r.db.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *repository) UpdatePassword(id uuid.UUID, hashedPassword string) error {
	result := r.db.Model(&models.UsersModel{}).Where("id = ?", id).Update("password", hashedPassword)
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return result.Error
}

func (r *repository) MarkVerified(id uuid.UUID) error {
	result := r.db.Model(&models.UsersModel{}).Where("id = ?", id).Update("is_verified", true)
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return result.Error
}
