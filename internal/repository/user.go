package repository

import (
	"awesomeProject/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

// GetUserByUsername 根据用户名查询用户
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}
