# 基础镜像
FROM golang:1.25-alpine as builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制项目文件
COPY . .

# 构建应用
RUN go build -o app main.go

# 运行镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 复制构建好的应用
COPY --from=builder /app/app .

# 复制配置文件
COPY config/ config/

# 暴露端口
EXPOSE 8000

# 启动应用
CMD ["./app"]
