# Go后端项目

## 项目介绍

这是一个使用Go语言开发的后端系统，主要用于IoT设备监控和控制。系统集成了WebSocket和SSE实时通信技术，通过MQTT协议与设备进行交互，支持传感器数据采集、阈值预警、设备控制等功能。

## 技术栈

- **框架**：Gin
- **数据库**：PostgreSQL (GORM)
- **缓存**：Redis
- **消息队列**：RabbitMQ
- **实时通信**：WebSocket, SSE (Server-Sent Events)
- **设备通信**：MQTT
- **搜索引擎**：Elasticsearch
- **图片处理**：nfnt/resize
- **跨域处理**：gin-contrib/cors
- **API文档**：Swagger

## 目录结构

```
├── api/                # API文档
│   └── swagger.yaml    # Swagger配置文件
├── config/             # 配置文件
│   └── config.yaml     # 系统配置
├── internal/           # 内部代码
│   ├── config/         # 配置管理
│   ├── controllers/    # 控制器
│   ├── elasticsearch/  # Elasticsearch服务
│   ├── middleware/     # 中间件
│   ├── model/          # 数据模型
│   ├── redis/          # Redis服务
│   ├── repository/     # 数据仓库
│   ├── routes/         # 路由配置
│   └── service/        # 业务逻辑
│       └── rabbitmq/   # RabbitMQ服务
├── pkg/                # 公共包
│   ├── logger.go       # 日志工具
│   └── response.go     # 响应工具
├── .gitignore          # Git忽略文件
├── README.md           # 项目说明
├── go.mod              # Go模块文件
├── go.sum              # 依赖校验文件
└── main.go             # 主入口文件
```

## 主要功能

### 用户管理

- **用户注册**：创建新用户账号
- **用户登录**：获取认证令牌
- **用户登出**：注销当前会话
- **获取用户信息**：查询用户详情
- **更新用户信息**：修改用户个人资料
- **修改密码**：更新用户密码

### 设备控制

- **RGB灯控制**：开关、颜色调整、亮度调节
- **屏幕文字显示**：文本内容、滚动效果、颜色设置、字体设置
- **屏幕图片显示**：上传本地图片、从URL获取图片
- **蜂鸣器控制**：开关、频率调节、持续时间设置、循环次数设置

### 传感器数据处理

- **数据采集**：接收和处理温湿度、距离等传感器数据
- **阈值预警**：基于设定的阈值进行温度、湿度、距离预警
- **实时监控**：通过WebSocket和SSE实时推送数据

### 数据存储与分析

- **PostgreSQL**：存储用户信息、设备配置、阈值设置
- **Elasticsearch**：存储和查询传感器数据，支持复杂的搜索和聚合分析
- **Redis**：缓存频繁访问的数据，提高性能

### 消息队列

- **RabbitMQ**：处理异步任务，如消息通知、预警推送

### MQTT集成

- 通过MQTT协议与设备进行通信，发送控制命令和接收设备状态

### 预警系统

- **温度预警**：当温度超过设定阈值时触发
- **湿度预警**：当湿度超过设定阈值时触发
- **距离预警**：当距离低于设定阈值时触发
- **多渠道通知**：支持Bark推送等通知方式

### Elasticsearch查询

- **日志搜索**：支持多条件组合搜索传感器数据
- **统计分析**：获取数据的平均值、最大值、最小值等统计信息
- **设备日志**：查询特定设备的所有历史数据

## 快速开始

### 前置要求

- Go 1.25+ 环境
- PostgreSQL 数据库
- Redis 服务
- RabbitMQ 服务
- MQTT Broker
- Elasticsearch 服务（可选，用于日志存储和查询）

### 安装步骤

1. **克隆项目**

   ```bash
   git clone https://github.com/duasong111/go.git 
   cd go
   ```

2. **安装依赖**

   ```bash
   go mod tidy
   ```

3. **配置系统**

   修改 `config/config.yaml` 文件中的配置信息：

   ```yaml
   # 数据库配置
   database:
     host: your_host
     port: 5432
     user: your_user
     password: your_password
     dbname: your_db
     sslmode: disable

   # Redis 配置
   redis:
     host: your_redis_host
     port: 6379
     password: your_redis_password
     db: 0

   # RabbitMQ 配置
   rabbitmq:
     host: your_rabbitmq_host
     port: 5672
     user: guest
     password: guest
     vhost: /
     exchange: sensor_exchange
     routing_key: sensor_data
     queue: sensor_queue

   # MQTT 配置
   mqtt:
     broker: tcp://your_mqtt_broker:1883
     client_id: go_backend
     username: your_mqtt_username
     password: your_mqtt_password

   # Elasticsearch 配置
   elasticsearch:
     enabled: true
     host: your_elasticsearch_host
     port: 9200
     username: elastic
     password: your_elasticsearch_password
     index: sensor-log
     sniff: false
   ```

4. **运行项目**

   ```bash
   go run main.go
   ```

   项目将在 `http://localhost:8000` 启动

## API文档

### 公共API

| 路径 | 方法 | 描述 |
|------|------|------|
| `/api/register` | POST | 用户注册 |
| `/api/login` | POST | 用户登录 |
| `/api/sse` | GET | SSE实时数据推送 |
| `/esp32/data` | GET | WebSocket连接 |
| `/api/device/rgb` | POST | 控制RGB灯 |
| `/api/device/screen/text` | POST | 控制屏幕文字 |
| `/api/device/screen/image` | POST | 上传图片到屏幕 |
| `/api/device/screen/image_url` | POST | 从URL发送图片到屏幕 |
| `/api/device/buzzer` | POST | 控制蜂鸣器 |
| `/api/device/bark_alert` | POST | 发送Bark通知 |
| `/api/device/distance_alert` | POST | 发送距离预警 |
| `/api/device/sensor_data` | POST | 接收传感器数据 |
| `/api/device/control_self` | POST | 设备自我控制 |
| `/api/logs/search` | GET | 搜索日志 |
| `/api/logs/stats` | GET | 获取统计信息 |
| `/api/logs/device/:device_id` | GET | 获取设备日志 |

### 受保护API（需要认证）

| 路径 | 方法 | 描述 |
|------|------|------|
| `/api/logout` | POST | 用户登出 |
| `/api/user/info` | GET | 获取用户信息 |
| `/api/update` | PUT | 更新用户信息 |
| `/api/modify` | PUT | 修改用户密码 |
| `/api/device/accept_threshold` | POST | 接受温湿度阈值 |
| `/api/device/accept_bark_token` | POST | 接受Bark令牌 |
| `/api/device/manage_threshold` | POST | 管理温湿度阈值 |
| `/api/device/manage_distance_threshold` | POST | 管理距离阈值 |
| `/api/device/distance_threshold` | POST | 接受距离阈值 |

### 请求示例

#### 用户注册

```json
POST /api/register
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123",
  "email": "test@example.com"
}
```

#### 控制RGB灯

```json
POST /api/device/rgb
Content-Type: application/json

{
  "state": "on",
  "color": "red",
  "brightness": 80
}
```

#### 控制屏幕文字

```json
POST /api/device/screen/text
Content-Type: application/json

{
  "text": "Hello World",
  "duration": 10,
  "scroll": true,
  "font_size": 24,
  "text_color": "blue",
  "background_color": "white"
}
```

### 预警阈值设置示例

#### 温湿度阈值

```json
POST /api/device/accept_threshold
Content-Type: application/json
Authorization: Bearer {token}

{
  "device_id": "esp32_001",
  "temperature_max": 30.0,
  "temperature_min": 10.0,
  "humidity_max": 70.0,
  "humidity_min": 30.0,
  "alert_seconds": 300,
  "is_active": true
}
```

#### 距离阈值

```json
POST /api/device/distance_threshold
Content-Type: application/json
Authorization: Bearer {token}

{
  "device_id": "esp32_001",
  "distance_min": 20.0,
  "alert_seconds": 300,
  "is_active": true
}
```

### 传感器数据示例

```json
POST /api/device/sensor_data
Content-Type: application/json

{
  "device_id": "esp32_001",
  "temperature": 25.5,
  "humidity": 55.2,
  "distance": 15.3
}
```

### Elasticsearch查询示例

#### 搜索日志

```bash
# 搜索温度超过30的日志
GET /api/logs/search?min_temp=30

# 搜索特定设备的日志
GET /api/logs/search?device_id=esp32_001

# 搜索时间范围内的日志
GET /api/logs/search?start_date=2026-03-14&end_date=2026-03-16
```

#### 获取统计信息

```bash
# 获取所有数据的统计
GET /api/logs/stats

# 获取特定设备的统计
GET /api/logs/stats?device_id=esp32_001
```

#### 获取设备日志

```bash
# 获取设备的最新日志
GET /api/logs/device/esp32_001

# 分页获取
GET /api/logs/device/esp32_001?page=2&page_size=10
```

## 配置管理

项目使用YAML配置文件进行管理，配置文件位于 `config/config.yaml`。主要配置项包括：

| 配置项 | 描述 | 默认值 |
|--------|------|--------|
| `server.port` | 服务器端口 | `8000` |
| `database` | 数据库连接信息 | - |
| `redis` | Redis连接信息 | - |
| `rabbitmq` | RabbitMQ连接信息 | - |
| `mqtt` | MQTT连接信息 | - |
| `jwt.secret` | JWT密钥 | - |
| `cors` | CORS配置 | - |
| `elasticsearch` | Elasticsearch配置 | - |

## 数据流向

```
ESP32 设备
    ↓
POST /api/device/sensor_data
    ↓
Go 后端处理
    ↓
1. 存储到 PostgreSQL（用户信息、配置等）
2. 索引到 Elasticsearch（传感器数据）
3. 发送到 RabbitMQ（异步任务）
4. 缓存到 Redis（频繁访问的数据）
    ↓
API 接口查询
    ↓
  前端显示
```

## 注意事项

1. **安全**：生产环境中应修改CORS配置，限制允许的源，修改JWT密钥
2. **性能**：图片上传大小限制为200KB，以适应ESP32的接收缓冲区
3. **可靠性**：建议使用MQTT持久化消息，确保设备离线后仍能接收到命令
4. **扩展**：可根据需要添加更多设备类型和控制功能
5. **Elasticsearch**：如果不需要日志存储和查询功能，可以在配置中禁用

## 许可证

[MIT License](LICENSE)

## 贡献

欢迎提交Issue和Pull Request来帮助改进这个项目！