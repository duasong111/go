package service

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"os"
	"time"
)

// 此处的目的是调用sql语句的
type UserService struct {
	repo *repository.UserRepository
}

// UserService 的工厂函数，用于依赖注入
func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// 用户注册服务
func (s *UserService) Register(username, password, email string) (*model.User, error) {
	existing, err := s.repo.GetUserByUsername(username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID != 0 {
		return nil, errors.New("用户名已存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// 用户登录验证
func (s *UserService) Login(username, password string) (*model.User, string, error) {
	// 查询用户信息
	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		return nil, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("密码错误")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key"
	}
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, "", err
	}
	return user, tokenString, nil
}

func (s *UserService) Logout(userID uint) error {
	return nil
}
