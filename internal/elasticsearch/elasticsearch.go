package elasticsearch

import (
	"awesomeProject/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Client struct {
	client *elasticsearch.Client
	config *config.ElasticsearchConfig
}

var ESClient *Client

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp   time.Time              `json:"@timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Category    string                 `json:"category"`
	DeviceID    string                 `json:"device_id,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Humidity    float64                `json:"humidity,omitempty"`
	Distance    float64                `json:"distance,omitempty"`
	UserID      uint                   `json:"user_id,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// Init 初始化 Elasticsearch 连接
func Init() error {
	if !config.AppConfig.Elasticsearch.Enabled {
		log.Println("Elasticsearch 已禁用")
		return nil
	}

	esConfig := config.AppConfig.Elasticsearch

	// 构建连接配置
	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%d", esConfig.Host, esConfig.Port),
		},
		Username: esConfig.Username,
		Password: esConfig.Password,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("创建 Elasticsearch 客户端失败: %w", err)
	}

	// 测试连接
	resp, err := client.Info()
	if err != nil {
		return fmt.Errorf("连接 Elasticsearch 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("Elasticsearch 连接失败: %s", resp.Status())
	}

	ESClient = &Client{
		client: client,
		config: &esConfig,
	}

	// 创建索引（如果不存在）
	if err := createIndex(); err != nil {
		log.Printf("创建索引失败: %v", err)
	}

	log.Println("Elasticsearch 连接成功")
	return nil
}

// createIndex 创建索引
func createIndex() error {
	index := config.AppConfig.Elasticsearch.Index
	// 检查索引是否存在
	resp, err := ESClient.client.Indices.Exists([]string{index})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// 索引不存在，创建
		resp, err := ESClient.client.Indices.Create(index)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.IsError() {
			return fmt.Errorf("创建索引失败: %s", resp.Status())
		}
		log.Printf("创建索引 %s 成功", index)
	}

	return nil
}

// Index 索引日志
func (c *Client) Index(entry LogEntry) error {
	if !config.AppConfig.Elasticsearch.Enabled {
		return nil
	}

	// 序列化日志条目
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// 创建索引请求
	req := esapi.IndexRequest{
		Index:   config.AppConfig.Elasticsearch.Index,
		Body:    strings.NewReader(string(data)),
		Refresh: "false",
	}

	// 执行请求
	resp, err := req.Do(context.Background(), c.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("索引失败: %s", resp.Status())
	}

	return nil
}

// Search 搜索日志
func (c *Client) Search(query map[string]interface{}) (map[string]interface{}, error) {
	if !config.AppConfig.Elasticsearch.Enabled {
		return nil, fmt.Errorf("Elasticsearch 已禁用")
	}

	// 序列化查询
	data, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	// 创建搜索请求
	req := esapi.SearchRequest{
		Index: []string{config.AppConfig.Elasticsearch.Index},
		Body:  strings.NewReader(string(data)),
	}

	// 执行请求
	resp, err := req.Do(context.Background(), c.client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("搜索失败: %s", resp.Status())
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// IndexLog 索引日志的便捷函数
func IndexLog(level, message, service, category string, details map[string]interface{}) error {
	if ESClient == nil || !config.AppConfig.Elasticsearch.Enabled {
		return nil
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Service:   service,
		Category:  category,
		Details:   details,
	}

	return ESClient.Index(entry)
}

// IndexSensorData 索引传感器数据
func IndexSensorData(deviceID string, temperature, humidity, distance float64) error {
	if ESClient == nil || !config.AppConfig.Elasticsearch.Enabled {
		return nil
	}

	entry := LogEntry{
		Timestamp:   time.Now(),
		Level:       "INFO",
		Message:     "传感器数据",
		Service:     "sensor",
		Category:    "data",
		DeviceID:    deviceID,
		Temperature: temperature,
		Humidity:    humidity,
		Distance:    distance,
	}

	return ESClient.Index(entry)
}
