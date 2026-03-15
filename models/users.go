package models

type UsersModel struct {
	Base
	Email      string `gorm:"uniqueIndex;not null"              json:"email"`
	Password   string `gorm:"not null"                          json:"-"`
	FullName   string `gorm:"not null"                          json:"fullName"`
	AvatarUrl  string `                                         json:"avatarUrl"`
	IsVerified bool   `gorm:"not null;default:false"            json:"isVerified"`
}

func (UsersModel) TableName() string {
	return "users"
}
