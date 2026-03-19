// internal/service/rabbitmq/consumer.go

package rabbitmq

import (
	"awesomeProject/internal/config"
	"awesomeProject/pkg"
	"encoding/json"
	"log"
)

func StartSensorDataConsumer() {
	ch, err := GetChannel()
	if err != nil {
		log.Printf("消费者获取通道失败: %v", err)
		return
	}

	queueName := config.AppConfig.RabbitMQ.Queue

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Printf("设置 QoS 失败: %v", err)
		return
	}

	msgs, err := ch.Consume(
		queueName,
		"sensor-consumer-1",
		false,
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
		//go ProcessDistanceAlert(req.DeviceID, req.Distance)

		// 处理成功，手动确认
		msg.Ack(false)
	}
}
