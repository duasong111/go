// internal/service/rabbitmq/consumer.go

package rabbitmq

import (
	"encoding/json"
	"log"

	"awesomeProject/pkg" // 假设 SensorDataRequest 在此包
	// 例如：
	// "awesomeProject/internal/service/alert"   ← 根据实际情况调整路径
)

func StartSensorDataConsumer() {
	ch, err := GetChannel() // 假设您已有 GetChannel() 方法返回 *amqp.Channel
	if err != nil {
		log.Printf("消费者获取通道失败: %v", err)
		return
	}

	// 使用已声明的队列（不要重新声明）
	queueName := "sensor_data_queue"

	// 限制每次只处理1条消息（防止并发爆炸）
	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Printf("设置 QoS 失败: %v", err)
		return
	}

	msgs, err := ch.Consume(
		queueName,
		"sensor-consumer-1", // consumer tag，可自定义
		false,               // 必须手动 ack
		false, false, false,
		nil,
	)
	if err != nil {
		log.Printf("注册消费者失败: %v", err)
		return
	}

	log.Printf("[RabbitMQ Consumer] 等待传感器数据消息... (队列: %s)", queueName)

	for msg := range msgs {
		var req pkg.SensorDataRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			log.Printf("消息解析失败: %v → body: %s", err, string(msg.Body))
			msg.Nack(false, true) // 重新入队
			continue
		}

		// 执行阈值检查与报警（原来的异步逻辑移到这里）
		//go processTempHumidityAlert(req.DeviceID, req.Temperature, req.Humidity)
		//go processDistanceAlert(req.DeviceID, req.Distance)

		// 处理成功，手动确认
		msg.Ack(false)
	}
}
