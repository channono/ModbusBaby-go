package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// Init 初始化日志系统
func Init() {
	Logger = logrus.New()
	
	// 设置日志格式
	Logger.SetFormatter(&CustomFormatter{})

	// 设置日志级别
	Logger.SetLevel(logrus.InfoLevel)

	// 创建日志目录
	logDir := getLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logrus.Errorf("无法创建日志目录 %s: %v", logDir, err)
		return // 无法创建目录，直接返回，日志将输出到stderr
	}

	// 创建日志文件
	logFile := filepath.Join(logDir, "modbusbaby.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Errorf("无法打开日志文件 %s: %v", logFile, err)
		return // 无法打开文件，直接返回，日志将输出到stderr
	}
	Logger.SetOutput(file)
}

// getLogDir 获取日志目录
func getLogDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".modbusbaby", "logs")
}

// Info 信息日志
func Info(args ...interface{}) {
	if Logger != nil {
		Logger.Info(args...)
	}
}

// Error 错误日志
func Error(args ...interface{}) {
	if Logger != nil {
		Logger.Error(args...)
	}
}

// Debug 调试日志
func Debug(args ...interface{}) {
	if Logger != nil {
		Logger.Debug(args...)
	}
}

// Warn 警告日志
func Warn(args ...interface{}) {
	if Logger != nil {
		Logger.Warn(args...)
	}
}