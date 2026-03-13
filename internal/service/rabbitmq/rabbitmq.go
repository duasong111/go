package rabbitmq

import (
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

	// 构建连接字符串（添加虚拟主机参数）
	addr := "amqp://" + username + ":" + password + "@" + host + ":5672/%2F"

	var err error
	// 建立连接
	conn, err = amqp.Dial(addr)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return err
	}

	// 建立通道
	channel, err = conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v", err)
		conn.Close()
		return err
	}

	// 声明一个交换机
	exchangeName := "sensor_data"
	err = channel.ExchangeDeclare(
		exchangeName, // 交换机名称
		"direct",     // 交换机类型
		true,         // 持久化
		false,        // 自动删除
		false,        // 内部
		false,        // 非阻塞
		nil,          // 额外参数
	)
	if err != nil {
		log.Printf("Failed to declare exchange: %v", err)
		channel.Close()
		conn.Close()
		return err
	}

	// 声明一个队列
	queueName := "sensor_data_queue"
	_, err = channel.QueueDeclare(
		queueName, // 队列名称
		true,      // 持久化
		false,     // 自动删除
		false,     // 独占
		false,     // 非阻塞
		nil,       // 额外参数
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		channel.Close()
		conn.Close()
		return err
	}

	// 绑定队列到交换机
	routingKey := "sensor_data_key"
	err = channel.QueueBind(
		queueName,    // 队列名称
		routingKey,   // 路由键
		exchangeName, // 交换机名称
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to bind queue: %v", err)
		channel.Close()
		conn.Close()
		return err
	}

	// 设置连接监控
	go monitorConnection()

	log.Println("RabbitMQ connection initialized successfully")
	return nil
}

// GetChannel 获取 RabbitMQ 通道
func GetChannel() (*amqp.Channel, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if conn == nil || conn.IsClosed() || channel == nil {
		// 重新连接
		log.Println("RabbitMQ connection lost, reconnecting...")
		err := Init("rabbitmq", "rabbitmq", "rabbitmq")
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
			err := Init("rabbitmq", "rabbitmq", "rabbitmq")
			if err != nil {
				log.Printf("Failed to reconnect: %v", err)
			}
		}
		mutex.Unlock()
	}
}
