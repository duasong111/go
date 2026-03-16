package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	RabbitMQ    RabbitMQConfig    `mapstructure:"rabbitmq"`
	MQTT        MQTTConfig        `mapstructure:"mqtt"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	CORS        CORSConfig        `mapstructure:"cors"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

type RabbitMQConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	VHost      string `mapstructure:"vhost"`
	Exchange   string `mapstructure:"exchange"`
	Queue      string `mapstructure:"queue"`
	RoutingKey string `mapstructure:"routing_key"`
}

type MQTTConfig struct {
	Broker   string `mapstructure:"broker"`
	ClientID string `mapstructure:"client_id"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type ElasticsearchConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Index    string `mapstructure:"index"`
	Sniff    bool   `mapstructure:"sniff"`
}

var AppConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置到结构体
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	log.Println("配置文件加载成功")
	return nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s user=%s dbname=%s port=%d password=%s sslmode=%s",
		c.Host, c.User, c.DBName, c.Port, c.Password, c.SSLMode)
}

// GetRedisAddr 获取 Redis 地址
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetRabbitMQURL 获取 RabbitMQ 连接 URL
func (c *RabbitMQConfig) GetRabbitMQURL() string {
	vhost := c.VHost
	if vhost == "/" {
		vhost = "%2F"
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		c.User, c.Password, c.Host, c.Port, vhost)
}
