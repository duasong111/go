package rabbitmq

import (
	"awesomeProject/internal/config"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn    *amqp.Connection
	channel *amqp.Channel
	mutex   sync.Mutex
)

// Init 初始化 RabbitMQ 连接
func Init(host, username, password string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if conn != nil && !conn.IsClosed() {
		return nil
	}

	var err error
	conn, err = amqp.Dial(config.AppConfig.RabbitMQ.GetRabbitMQURL())
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return err
	}

	channel, err = conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v", err)
		conn.Close()
		return err
	}

	exchangeName := config.AppConfig.RabbitMQ.Exchange
	err = channel.ExchangeDeclare(
		exchangeName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to declare exchange: %v", err)
		channel.Close()
		conn.Close()
		return err
	}

	queueName := config.AppConfig.RabbitMQ.Queue
	_, err = channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		channel.Close()
		conn.Close()
		return err
	}

	routingKey := config.AppConfig.RabbitMQ.RoutingKey
	err = channel.QueueBind(
		queueName,
		routingKey,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to bind queue: %v", err)
		channel.Close()
		conn.Close()
		return err
	}

	go monitorConnection()

	log.Println("RabbitMQ connection initialized successfully")
	return nil
}

// GetChannel 获取 RabbitMQ 通道
func GetChannel() (*amqp.Channel, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if conn == nil || conn.IsClosed() || channel == nil {
		log.Println("RabbitMQ connection lost, reconnecting...")
		err := Init(
			config.AppConfig.RabbitMQ.Host,
			config.AppConfig.RabbitMQ.User,
			config.AppConfig.RabbitMQ.Password,
		)
		if err != nil {
			log.Printf("Failed to reconnect to RabbitMQ: %v", err)
			return nil, err
		}
	}

	return channel, nil
}

// Close 关闭 RabbitMQ 连接
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	if channel != nil {
		channel.Close()
	}
	if conn != nil {
		conn.Close()
	}

	log.Println("RabbitMQ connection closed")
}

// monitorConnection 监控 RabbitMQ 连接状态
func monitorConnection() {
	for {
		time.Sleep(30 * time.Second)
		mutex.Lock()
		if conn != nil && conn.IsClosed() {
			log.Println("RabbitMQ connection closed, reconnecting...")
			err := Init(
				config.AppConfig.RabbitMQ.Host,
				config.AppConfig.RabbitMQ.User,
				config.AppConfig.RabbitMQ.Password,
			)
			if err != nil {
				log.Printf("Failed to reconnect: %v", err)
			}
		}
		mutex.Unlock()
	}
}
