package rabbitmq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish 发送消息到 RabbitMQ
func Publish(exchange, routingKey string, message interface{}) error {
	// 获取通道
	ch, err := GetChannel()
	if err != nil {
		log.Printf("Failed to get RabbitMQ channel: %v", err)
		return err
	}

	// 序列化消息
	body, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return err
	}

	// 发布消息
	err = ch.Publish(
		exchange,   // 交换机名称
		routingKey, // 路由键
		false,      // 强制
		false,      // 立即
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	return nil
}
