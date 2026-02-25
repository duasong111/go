# Go后端项目

## 项目介绍

这是一个使用Go语言开发的后端系统，主要用于设备控制和用户管理。系统集成了WebSocket和SSE实时通信技术，通过MQTT协议与设备进行交互，支持RGB灯控制、屏幕显示控制和蜂鸣器控制等功能。

## 技术栈

- **框架**：Gin
- **数据库**：PostgreSQL (GORM)
- **实时通信**：WebSocket, SSE (Server-Sent Events)
- **设备通信**：MQTT
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
│   ├── controllers/    # 控制器
│   ├── middleware/     # 中间件
│   ├── model/          # 数据模型
│   ├── repository/     # 数据仓库
│   ├── routes/         # 路由配置
│   └── service/        # 业务逻辑
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
- **更新用户信息**：修改用户个人资料
- **修改密码**：更新用户密码

### 设备控制

- **RGB灯控制**：开关、颜色调整、亮度调节
- **屏幕文字显示**：文本内容、滚动效果、颜色设置、字体设置
- **屏幕图片显示**：上传本地图片、从URL获取图片
- **蜂鸣器控制**：开关、频率调节、持续时间设置、循环次数设置

### 实时通信

- **WebSocket**：双向实时通信，用于设备数据传输
- **SSE**：服务器推送事件，用于实时数据更新

### MQTT集成

- 通过MQTT协议与设备进行通信，发送控制命令和接收设备状态

## 快速开始

### 前置要求

- Go 1.16+ 环境
- PostgreSQL 数据库
- MQTT Broker

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

3. **配置数据库**

   修改 `main.go` 文件中的数据库连接信息：

   ```go
   dsn := "host=your_host user=your_user dbname=your_db port=5432 password=your_password sslmode=disable"
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

### 受保护API（需要认证）

| 路径 | 方法 | 描述 |
|------|------|------|
| `/api/logout` | POST | 用户登出 |
| `/api/update` | PUT | 更新用户信息 |
| `/api/modify` | PUT | 修改用户密码 |

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

## 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `DB_HOST` | 数据库主机 | `60.205.140.163` |
| `DB_USER` | 数据库用户 | `user_yrh7kC` |
| `DB_NAME` | 数据库名称 | `go_pg` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_PASSWORD` | 数据库密码 | `password_asY8fN` |
| `MQTT_BROKER` | MQTT broker地址 | - |
| `SERVER_PORT` | 服务器端口 | `8000` |

## 注意事项

1. **安全**：生产环境中应修改CORS配置，限制允许的源
2. **性能**：图片上传大小限制为200KB，以适应ESP32的接收缓冲区
3. **可靠性**：建议使用MQTT持久化消息，确保设备离线后仍能接收到命令
4. **扩展**：可根据需要添加更多设备类型和控制功能

## 许可证

[MIT License](LICENSE)

## 贡献

欢迎提交Issue和Pull Request来帮助改进这个项目！
