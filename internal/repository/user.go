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
// 修改：统一返回 gorm.ErrRecordNotFound 以便服务层区分业务逻辑
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound // 修改：返回标准 GORM 错误，便于服务层处理注册/登录逻辑
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
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
// 修改：统一返回 gorm.ErrRecordNotFound
func (r *UserRepository) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound // 修改：返回标准 GORM 错误
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱查询用户
// 修改：统一返回 gorm.ErrRecordNotFound，并确保 err == nil 时返回非零值用户
func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound // 新增：处理未找到情况
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByPhone 根据手机号查询用户
// 修改：同上，统一处理
func (r *UserRepository) GetUserByPhone(phone string) (*model.User, error) {
	var user model.User
	err := r.db.Where("phone = ?", phone).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound // 新增：处理未找到情况
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
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
