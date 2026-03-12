package service

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/redis"
	"awesomeProject/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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
		secret = "K9f4zB2qX8vL7nA1pR6sT5wM3cN9xY2hV7jQ4mE6oI5uP8tW1rS3eD7yH6kL9vC4n"
	}
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, "", err
	}
	return user, tokenString, nil
}

// 用户登出

func (s *UserService) Logout(userID uint) error {
	return nil
}

// 用户信息更新服务

func (s *UserService) UpdateUser(id uint, updates map[string]interface{}) (*model.User, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%d", id)

	// 1. 先从缓存获取用户信息
	var user *model.User
	cached, err := redis.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cached), &user); err != nil {
			// 反序列化失败，从数据库查询
			user, err = s.repo.GetUserByID(id)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		user, err = s.repo.GetUserByID(id)
		if err != nil {
			return nil, err
		}
	}

	// 2. 验证唯一性字段（Email 和 Phone）
	if email, ok := updates["email"].(string); ok && email != "" && email != user.Email {
		existing, err := s.repo.GetUserByEmail(email) // 需添加 GetUserByEmail 方法
		if err == nil && existing != nil {
			return nil, errors.New("邮箱已存在")
		}
	}
	if phone, ok := updates["phone"].(string); ok && phone != "" && phone != user.Phone {
		existing, err := s.repo.GetUserByPhone(phone) // 需添加 GetUserByPhone 方法
		if err == nil && existing != nil {
			return nil, errors.New("手机号已存在")
		}
	}

	// 3. 执行更新
	if err := s.repo.UpdateUser(id, updates); err != nil {
		return nil, err
	}

	// 4. 清除缓存
	redis.Client.Del(ctx, cacheKey)

	// 5. 返回更新后的用户
	updatedUser, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// 6. 将更新后的用户信息写入缓存
	if data, err := json.Marshal(updatedUser); err == nil {
		redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
	}

	return updatedUser, nil
}

// 获取用户信息服务

func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%d", id)

	// 1. 先从缓存获取用户信息
	var user *model.User
	cached, err := redis.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cached), &user); err != nil {
			// 反序列化失败，从数据库查询
			user, err = s.repo.GetUserByID(id)
			if err != nil {
				return nil, err
			}
			// 写入缓存
			if data, err := json.Marshal(user); err == nil {
				redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		user, err = s.repo.GetUserByID(id)
		if err != nil {
			return nil, err
		}
		// 写入缓存
		if data, err := json.Marshal(user); err == nil {
			redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
		}
	}

	return user, nil
}

// 用户密码修改服务

func (s *UserService) ModifyPassword(id uint, oldPassword, newPassword string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%d", id)

	// 1. 查询用户（先从缓存获取）
	var user *model.User
	cached, err := redis.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cached), &user); err != nil {
			// 反序列化失败，从数据库查询
			user, err = s.repo.GetUserByID(id)
			if err != nil {
				return err
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		user, err = s.repo.GetUserByID(id)
		if err != nil {
			return err
		}
	}

	// 2. 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	// 3. 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// 4. 更新密码
	if err := s.repo.UpdateUserPassword(id, string(hashedPassword)); err != nil {
		return err
	}

	// 5. 清除缓存
	redis.Client.Del(ctx, cacheKey)

	return nil
}

// 暴露出去，供同目录的sse.go使用
// func (s *UserService) GetUserByID(id uint) (*model.User, error) {
// 	return s.repo.GetUserByID(id)
// }
