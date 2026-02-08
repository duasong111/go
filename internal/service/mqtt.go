package service

import (
	"encoding/json"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	mqttClient mqtt.Client
	once       sync.Once
)

type SensorData struct {
	Time        string      `json:"time"`
	Temperature float64     `json:"temperature"`
	Humidity    float64     `json:"humidity"`
	Distance    interface{} `json:"distance"`
	// 如果后续有更多字段，可以继续加
}

func generateShortID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetMQTTClient 返回单例 MQTT 客户端
func GetMQTTClient() mqtt.Client {
	once.Do(func() {
		opts := mqtt.NewClientOptions()
		opts.AddBroker("tcp://60.205.140.163:1883") // ← 这里改！

		opts.SetClientID("backend-sensor-" + generateShortID())
		opts.SetUsername("admin")
		opts.SetPassword("password")

		opts.SetCleanSession(false)         // false 更适合后端持久连接；如需每次干净重连可改 true
		opts.SetKeepAlive(60 * time.Second) // 与工具里的 Keep Alive 60 匹配
		opts.SetAutoReconnect(true)
		opts.SetMaxReconnectInterval(30 * time.Second)

		opts.SetConnectionLostHandler(onConnectionLost)
		opts.SetReconnectingHandler(onReconnecting)
		opts.SetOnConnectHandler(onConnect)

		mqttClient = mqtt.NewClient(opts)
	})

	return mqttClient
}

func InitMQTT() error {
	client := GetMQTTClient()
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt connect failed: %w", token.Error())
	}

	// 订阅规则：根据你的工具，目前是 sensor/data
	// 如果未来有多个设备或更多类型，可以扩展为 "sensor/#" 或 "devices/+/sensor/#"
	topics := map[string]byte{
		"sensor/#": 1, // 覆盖 sensor/data, sensor/xxx 等
		// "devices/+/sensor/#": 1,  // 如果后续要支持多设备，可打开这行
		// "devices/+/status":   1,
	}

	token := client.SubscribeMultiple(topics, messageHandler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt subscribe failed: %w", token.Error())
	}

	//logger.Info("MQTT connected and subscribed successfully")
	return nil
}

// 连接成功回调
func onConnect(client mqtt.Client) {
	logger.Info("MQTT client connected")
	// 可在这里做额外初始化订阅（断线重连后自动重新订阅）
}

// 连接丢失回调
func onConnectionLost(client mqtt.Client, err error) {
	logger.Warnf("MQTT connection lost: %v", err)
}

// 重连中回调
func onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	logger.Info("MQTT reconnecting...")
}

// 统一消息处理入口
func messageHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payloadStr := string(msg.Payload())
	// 根据你的实际 topic 做分发
	if strings.HasPrefix(topic, "sensor/") {
		handleSensorData(topic, payloadStr)
		return
	}
}

// 处理传感器数据（你的示例 JSON）
func handleSensorData(topic string, payload string) {
	var data SensorData
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		logger.Errorf("Failed to parse sensor JSON: %v, payload: %s", err, payload)
		return
	}
	//logger.Infof("Sensor data parsed → temp: %.1f°C, hum: %.1f%%, dist: %.1f cm, time: %s", data.Temperature, data.Humidity, data.Distance, data.Time)

	BroadcastToWS(topic, payload)
}

// 对外暴露的发布函数（供控制器调用，例如前端发命令控制设备）
func Publish(topic string, qos byte, retained bool, payload interface{}) error {
	client := GetMQTTClient()
	if !client.IsConnectionOpen() {
		return fmt.Errorf("mqtt not connected")
	}

	var data []byte
	var err error

	switch v := payload.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("json marshal failed: %w", err)
		}
	}

	token := client.Publish(topic, qos, retained, data)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("publish failed: %w", token.Error())
	}

	//logger.Debugf("Published to %s (QoS:%d, retained:%v)", topic, qos, retained)
	return nil
}
