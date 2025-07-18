package utils

import (
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// SerialPortInfo 串口信息
type SerialPortInfo struct {
	Name        string
	Description string
	VID         string
	PID         string
}

// GetAvailableSerialPorts 获取可用的串口列表
func GetAvailableSerialPorts() ([]SerialPortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	var result []SerialPortInfo
	for _, port := range ports {
		info := SerialPortInfo{
			Name:        port.Name,
			Description: port.Product,
			VID:         port.VID,
			PID:         port.PID,
		}
		result = append(result, info)
	}

	return result, nil
}

// GetSimpleSerialPorts 获取简单的串口名称列表
func GetSimpleSerialPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}
	return ports, nil
}

// ValidateSerialPort 验证串口是否可用
func ValidateSerialPort(portName string) bool {
	ports, err := serial.GetPortsList()
	if err != nil {
		return false
	}

	for _, port := range ports {
		if port == portName {
			return true
		}
	}
	return false
}