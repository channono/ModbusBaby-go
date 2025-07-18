package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 应用配置结构
type Config struct {
	TCP             TCPConfig `json:"tcp"`
	RTU             RTUConfig `json:"rtu"`
	PollingInterval int       `json:"polling_interval"`
	DefaultConnType string    `json:"default_connection_type"`
	LogLevel        string    `json:"log_level"`
	Theme           string    `json:"theme"`
}

// TCPConfig TCP连接配置
type TCPConfig struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	SlaveID int    `json:"slave_id"`
}

// RTUConfig RTU连接配置
type RTUConfig struct {
	Port     string `json:"port"`
	BaudRate int    `json:"baud_rate"`
	DataBits int    `json:"data_bits"`
	StopBits int    `json:"stop_bits"`
	Parity   string `json:"parity"`
	SlaveID  int    `json:"slave_id"`
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		TCP: TCPConfig{
			IP:      "192.168.0.31",
			Port:    502,
			SlaveID: 1,
		},
		RTU: RTUConfig{
			Port:     "COM1",
			BaudRate: 9600,
			DataBits: 8,
			StopBits: 1,
			Parity:   "None",
			SlaveID:  1,
		},
		PollingInterval: 1000,
		DefaultConnType: "TCP",
		LogLevel:        "INFO",
		Theme:           "auto",
	}
}

// Load 加载配置文件
func Load() (*Config, error) {
	configPath := getConfigPath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save 保存配置文件
func (c *Config) Save() error {
	configPath := getConfigPath()
	
	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDir := filepath.Dir(exePath)
	// Look for config.json in the same directory as the executable
	return filepath.Join(exeDir, "config.json")
}