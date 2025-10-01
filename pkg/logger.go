package pkg

import (
	"log"
	"net"
	"os"
)

var logger *log.Logger

// 打印日志在终端
func LogDetail(level string, message string, detail net.Interface) {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}
	logger.Printf("[%s] %s | detail=%v\n", level, message, detail)
}

// 创建日志写到文件中
func InitLogger(logFile string) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("打开日志文件失败: %v", err)
	}
	logger = log.New(file, "", log.LstdFlags|log.Lshortfile) // 标准日志格式 + 文件行号
}

func Info(message string, detail net.Interface) {
	LogDetail("INFO", message, detail)
}

func Error(message string, detail net.Interface) {
	LogDetail("ERROR", message, detail)
}
