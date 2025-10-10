package repository

// 此处的目的是写那些sql语句的，然后返回给service去进行处理
import (
	"awesomeProject/internal/model"
	"errors"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// CreateUser 创建用户

func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

// GetUserByUsername --> 获取用户信息

func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("用户不存在")
	}
	return &user, err
}

// 更新用户信息

func (r *UserRepository) UpdateUser(id uint, updates map[string]interface{}) error {
	result := r.db.Model(&model.User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetUserByID 根据 ID 查询用户
func (r *UserRepository) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("用户不存在")
	}
	return &user, err
}

// GetUserByEmail 根据邮箱查询用户
func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

// GetUserByPhone 根据手机号查询用户
func (r *UserRepository) GetUserByPhone(phone string) (*model.User, error) {
	var user model.User
	err := r.db.Where("phone = ?", phone).First(&user).Error
	return &user, err
}

// 更新用户密码

func (r *UserRepository) UpdateUserPassword(id uint, hashedPassword string) error {
	result := r.db.Model(&model.User{}).Where("id = ?", id).Update("password", hashedPassword)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
