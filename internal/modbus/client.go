package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"modbusbaby/internal/logger"
	"modbusbaby/pkg/datatypes"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

// ConnectionType 连接类型
type ConnectionType int

const (
	TCP ConnectionType = iota
	RTU
)

func (ct ConnectionType) String() string {
	switch ct {
	case TCP:
		return "Modbus TCP"
	case RTU:
		return "Modbus RTU"
	default:
		return "Unknown"
	}
}

// RegisterType 寄存器类型
type RegisterType int

const (
	HoldingRegister RegisterType = iota
	InputRegister
	DiscreteInput
	Coil
)

func (rt RegisterType) String() string {
	switch rt {
	case HoldingRegister:
		return "Holding Register"
	case InputRegister:
		return "Input Register"
	case DiscreteInput:
		return "Discrete Input"
	case Coil:
		return "Coil"
	default:
		return "Unknown"
	}
}

// Client Modbus客户端
type Client struct {
	client         modbus.Client
	handler        io.Closer // Store the handler for closing
	connectionType ConnectionType
	isConnected    bool

	// data Converter 数据转换器
	converter *datatypes.Converter

	// telemetry log 报文记录
	lastSentPacket     []byte
	lastReceivedPacket []byte
	packetMutex        sync.RWMutex
	transactionID      uint16
	transactionIDMutex sync.Mutex
}

// NewClient 创建新的Modbus客户端
func NewClient() *Client {
	return &Client{
		converter: datatypes.NewConverter(datatypes.AB, datatypes.WORD_1234),
	}
}

// ConnectTCP 连接TCP设备
func (c *Client) ConnectTCP(host string, port int) error {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", host, port))
	handler.Timeout = 10 * time.Second
	err := handler.Connect()
	if err != nil {
		logger.Error("TCP Connection failed:", err)
		return err
	}

	c.client = modbus.NewClient(handler)
	c.handler = handler
	c.connectionType = TCP
	c.isConnected = true

	logger.Info(fmt.Sprintf("TCP Connection successful: %s:%d", host, port))
	return nil
}

// ConnectRTU 连接RTU设备
func (c *Client) ConnectRTU(port string, baudRate int, dataBits, stopBits int, parity string) error {
	handler := modbus.NewRTUClientHandler(port)
	handler.BaudRate = baudRate
	handler.DataBits = dataBits
	handler.StopBits = stopBits
	handler.Timeout = 10 * time.Second

	switch parity {
	case "Even":
		handler.Parity = "E"
	case "Odd":
		handler.Parity = "O"
	default:
		handler.Parity = "N"
	}

	err := handler.Connect()
	if err != nil {
		logger.Error("RTU Connection failed:", err)
		return err
	}

	c.client = modbus.NewClient(handler)
	c.handler = handler
	c.connectionType = RTU
	c.isConnected = true

	logger.Info(fmt.Sprintf("RTU Connection successful: %s, BaudRate: %d", port, baudRate))
	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect() error {
	if c.handler == nil {
		return nil
	}
	err := c.handler.Close()
	c.handler = nil
	c.client = nil
	if err != nil {
		logger.Error("Disconnection failed:", err)
		return err
	}
	logger.Info("Connection closed")
	return nil
}

// IsClientReady 检查客户端是否已初始化 (但不保证连接)
func (c *Client) IsClientReady() bool {
	return c.client != nil
}

// SetDataConverter 设置数据转换器
func (c *Client) SetDataConverter(byteOrder datatypes.ByteOrder, wordOrder datatypes.WordOrder) {
	c.converter = datatypes.NewConverter(byteOrder, wordOrder)
}

// IsConnected 检查客户端是否已连接
func (c *Client) IsConnected() bool {
	return c.isConnected
}

// ReadHoldingRegisters 读取保持寄存器
func (c *Client) ReadHoldingRegisters(slaveID byte,address, count uint16, dataType datatypes.DataType) (interface{}, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("device not connected")
	}


	if c.connectionType == TCP {
		if tcpHandler, ok := c.handler.(*modbus.TCPClientHandler); ok {
			originalSlaveID := tcpHandler.SlaveId
			tcpHandler.SlaveId = slaveID
			defer func() {
				tcpHandler.SlaveId = originalSlaveID
			}()
			logger.Debug(fmt.Sprintf("ReadHoldingRegisters (TCP): Setting handler SlaveId to %d", slaveID))
		} else {
			logger.Warn("TCP handler type assertion failed in ReadHoldingRegisters. Unit ID might not be set.")
		}
	}

	logger.Debug(fmt.Sprintf("Attempting to read holding registers for SlaveID: %d, Address: %d, Count: %d", slaveID, address, count))

	requestPDU := make([]byte, 5)
	requestPDU[0] = 0x03
	binary.BigEndian.PutUint16(requestPDU[1:3], address)
	binary.BigEndian.PutUint16(requestPDU[3:5], count)

	logger.Debug(fmt.Sprintf("ReadHoldingRegisters: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
	
	results, err := c.client.ReadHoldingRegisters( address, count)

	logger.Debug(fmt.Sprintf("ReadHoldingRegisters: Raw results from goburrow/modbus: %x, Error: %v", results, err))

	if err == nil {
		logger.Debug(fmt.Sprintf("Received Modbus Holding Registers response (PDU): %x", results))
	}

	if err != nil {
		// Add this explicit log
		if len(results) == 0 {
			logger.Info(fmt.Sprintf("Modbus Read Error: No response bytes received (results is empty/nil). Error: %v", err))
		} else {
			logger.Info(fmt.Sprintf("Modbus Read Error: Received partial/error response bytes: %x. Error: %v", results, err))
		}
		c.recordADU(requestPDU, nil, slaveID) // Pass nil for responsePDU on error
		return nil, fmt.Errorf("failed to read holding registers: %w", err)
	}
	c.recordADU(requestPDU, results, slaveID) // Pass results as responsePDU

	// 转换数据类型
	registers := bytesToUint16Array(results)
	return c.converter.ConvertFromRegisters(registers, dataType)
}

// ReadInputRegisters 读取输入寄存器
func (c *Client) ReadInputRegisters(slaveID byte, address, count uint16, dataType datatypes.DataType) (interface{}, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("device not connected")
	}

	// For TCP connections, ensure the SlaveId (Unit ID) is set on the handler
	if c.connectionType == TCP {
		if tcpHandler, ok := c.handler.(*modbus.TCPClientHandler); ok {
			originalSlaveID := tcpHandler.SlaveId
			tcpHandler.SlaveId = slaveID
			defer func() {
				tcpHandler.SlaveId = originalSlaveID
			}()
			logger.Debug(fmt.Sprintf("ReadInputRegisters (TCP): Setting handler SlaveId to %d", slaveID))
		} else {
			logger.Warn("TCP handler type assertion failed in ReadInputRegisters. Unit ID might not be set.")
		}
	}

	logger.Debug(fmt.Sprintf("Attempting to read input registers for SlaveID: %d, Address: %d, Count: %d", slaveID, address, count))

	requestPDU := make([]byte, 5)
	requestPDU[0] = 0x04
	binary.BigEndian.PutUint16(requestPDU[1:3], address)
	binary.BigEndian.PutUint16(requestPDU[3:5], count)

	logger.Debug(fmt.Sprintf("ReadInputRegisters: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
	var response []byte
	results, err := c.client.ReadInputRegisters(address, count)

	if err == nil {
		logger.Debug(fmt.Sprintf("Received Modbus Input Registers response (PDU): %x", results))
	}

	if err != nil {
		c.recordADU(requestPDU, nil, slaveID)
		return nil,  fmt.Errorf("failed to read input registers: %w", err)
	}

	
	c.recordADU(requestPDU, results, slaveID)
	response = results
	registers := bytesToUint16Array(response)
	return c.converter.ConvertFromRegisters(registers, dataType)
}

// ReadCoils 读取线圈
func (c *Client) ReadCoils(slaveID byte, address, count uint16) ([]bool, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("device not connected")
	}

	// For TCP connections, ensure the SlaveId (Unit ID) is set on the handler
	if c.connectionType == TCP {
		if tcpHandler, ok := c.handler.(*modbus.TCPClientHandler); ok {
			originalSlaveID := tcpHandler.SlaveId
			tcpHandler.SlaveId = slaveID
			defer func() {
				tcpHandler.SlaveId = originalSlaveID
			}()
			logger.Debug(fmt.Sprintf("readCoils (TCP): Setting handler SlaveId to %d", slaveID))
		} else {
			logger.Warn("TCP handler type assertion failed in ReadCoils. Unit ID might not be set.")
		}
	}

	logger.Debug(fmt.Sprintf("attempting to read coils for SlaveID: %d, Address: %d, Count: %d", slaveID, address, count))

	requestPDU := make([]byte, 5)
	requestPDU[0] = 0x01
	binary.BigEndian.PutUint16(requestPDU[1:3], address)
	binary.BigEndian.PutUint16(requestPDU[3:5], count)

	logger.Debug(fmt.Sprintf("ReadCoils: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
	// var response []byte
	results, err := c.client.ReadCoils(address, count)

	if err == nil {
		logger.Debug(fmt.Sprintf("Received Modbus Coils response (PDU): %x", results))
	}

	if err != nil {
		c.recordADU(requestPDU, nil, slaveID)
		return nil, fmt.Errorf("failed to read coils: %w", err)
	}

	c.recordADU(requestPDU, results, slaveID) // Pass results as responsePDU
	// response = results
	// 转换为bool数组
	var bools []bool
	for i := 0; i < int(count); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < len(results) { // Use results directly
			bools = append(bools, (results[byteIndex]&(1<<bitIndex)) != 0)
		}
	}
	return bools, nil
}

// ReadDiscreteInputs 读取离散输入
func (c *Client) ReadDiscreteInputs(slaveID byte, address, count uint16) ([]bool, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("device not connected")
	}

	// For TCP connections, ensure the SlaveId (Unit ID) is set on the handler
	if c.connectionType == TCP {
		if tcpHandler, ok := c.handler.(*modbus.TCPClientHandler); ok {
			originalSlaveID := tcpHandler.SlaveId
			tcpHandler.SlaveId = slaveID
			defer func() {
				tcpHandler.SlaveId = originalSlaveID
			}()
			logger.Debug(fmt.Sprintf("ReadDiscreteInputs (TCP): Setting handler SlaveId to %d", slaveID))
		} else {
			logger.Warn("TCP handler type assertion failed in ReadDiscreteInputs. Unit ID might not be set.")
		}
	}

	logger.Debug(fmt.Sprintf("Attempting to read discrete inputs for SlaveID: %d, Address: %d, Count: %d", slaveID, address, count))

	requestPDU := make([]byte, 5)
	requestPDU[0] = 0x02
	binary.BigEndian.PutUint16(requestPDU[1:3], address)
	binary.BigEndian.PutUint16(requestPDU[3:5], count)

	logger.Debug(fmt.Sprintf("ReadDiscreteInputs: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))

	results, err := c.client.ReadDiscreteInputs(address, count)
	// var response []byte

	if err == nil {
		logger.Debug(fmt.Sprintf("Received Modbus Discrete Inputs response (PDU): %x", results))
	}

	if err != nil {
		c.recordADU(requestPDU, nil, slaveID)
		return nil, fmt.Errorf("failed to read discrete inputs: %w", err)
	}
	
	c.recordADU(requestPDU, results, slaveID)
	// response = results
	// 转换为bool数组
	var bools []bool
	for i := 0; i < int(count); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < len(results) { // Use results directly
			bools = append(bools, (results[byteIndex]&(1<<bitIndex)) != 0)
		}
	}
	return bools, nil
}

// WriteHoldingRegisters 写入保持寄存器
func (c *Client) WriteHoldingRegisters(slaveID byte, address uint16, values interface{}) error {
	if !c.isConnected {
		return fmt.Errorf("device not connected")
	}

	// For TCP connections, ensure the SlaveId (Unit ID) is set on the handler
	if c.connectionType == TCP {
		if tcpHandler, ok := c.handler.(*modbus.TCPClientHandler); ok {
			originalSlaveID := tcpHandler.SlaveId
			tcpHandler.SlaveId = slaveID
			defer func() {
				tcpHandler.SlaveId = originalSlaveID
			}()
			logger.Debug(fmt.Sprintf("WriteHoldingRegisters (TCP): Setting handler SlaveId to %d", slaveID))
		} else {
			logger.Warn("TCP handler type assertion failed in WriteHoldingRegisters. Unit ID might not be set.")
		}
	}

	registers, err := c.converter.ConvertToRegisters(values)
	if err != nil {
		return fmt.Errorf("unsupported data type or conversion failed: %v", err)
	}
	
	quantity := uint16(len(registers))
	
	// 根据寄存器数量选择功能码
	if quantity == 1 {
		// 使用功能码 0x06 (Write Single Register)
		data := uint16ArrayToBytes(registers)
		logger.Debug(fmt.Sprintf("Attempting to write single holding register for SlaveID: %d, Address: %d", slaveID, address))

		// Request PDU: FC(1) + Addr(2) + Value(2)
		requestPDU := make([]byte, 5)
		requestPDU[0] = 0x06
		binary.BigEndian.PutUint16(requestPDU[1:3], address)
		copy(requestPDU[3:5], data)

		logger.Debug(fmt.Sprintf("WriteSingleRegister: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
		
		results, err := c.client.WriteSingleRegister(address, registers[0])
		if err != nil {
			if modbusErr, ok := err.(*modbus.ModbusError); ok {
				response := []byte{modbusErr.ExceptionCode}
				logger.Debug(fmt.Sprintf("Modbus Write Single Register error response (PDU): %x", response))
			}
			c.recordADU(requestPDU, nil, slaveID)
			return fmt.Errorf("failed to write single holding register: %w", err)
		}
		c.recordADU(requestPDU, results, slaveID)
		logger.Info(fmt.Sprintf("successfully wrote single holding register: Address=%d", address))

	} else {
		// 使用功能码 0x10 (Write Multiple Registers)
		data := uint16ArrayToBytes(registers)
		logger.Debug(fmt.Sprintf("attempting to write multiple holding registers for SlaveID: %d, Address: %d, Quantity: %d", slaveID, address, quantity))

		// Request PDU: FC(1) + Addr(2) + Qty(2) + ByteCount(1) + Data(N)
		requestPDU := make([]byte, 6+len(data))
		requestPDU[0] = 0x10
		binary.BigEndian.PutUint16(requestPDU[1:3], address)
		binary.BigEndian.PutUint16(requestPDU[3:5], quantity)
		requestPDU[5] = byte(len(data))
		copy(requestPDU[6:], data)

		logger.Debug(fmt.Sprintf("writeMultipleRegisters: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
		
		results, err := c.client.WriteMultipleRegisters(address, quantity, data)
		if err != nil {
			if modbusErr, ok := err.(*modbus.ModbusError); ok {
				response := []byte{modbusErr.ExceptionCode}
				logger.Debug(fmt.Sprintf("modbus Write Holding Registers error response (PDU): %x", response))
			}
			c.recordADU(requestPDU, nil, slaveID)
			return fmt.Errorf("failed to write multiple holding registers: %w", err)
		}
		c.recordADU(requestPDU, results, slaveID)
		logger.Info(fmt.Sprintf("successfully wrote multiple holding registers: Address=%d, Quantity=%d", address, quantity))
	}
	return nil
}

// WriteCoils 写入线圈
func (c *Client) WriteCoils(slaveID byte, address uint16, values []bool) error {
	if !c.isConnected {
		return fmt.Errorf("device not connected")
	}

	// For TCP connections, ensure the SlaveId (Unit ID) is set on the handler
	if c.connectionType == TCP {
		if tcpHandler, ok := c.handler.(*modbus.TCPClientHandler); ok {
			originalSlaveID := tcpHandler.SlaveId
			tcpHandler.SlaveId = slaveID
			defer func() {
				tcpHandler.SlaveId = originalSlaveID
			}()
			logger.Debug(fmt.Sprintf("WriteCoils (TCP): Setting handler SlaveId to %d", slaveID))
		} else {
			logger.Warn("TCP handler type assertion failed in WriteCoils. Unit ID might not be set.")
		}
	}

	quantity := uint16(len(values))

	if quantity == 1 {
		// 使用功能码 0x05 (Write Single Coil)
		value := uint16(0x0000)
		if values[0] {
			value = 0xFF00
		}
		logger.Debug(fmt.Sprintf("Attempting to write single coil for SlaveID: %d, Address: %d, Value: %v", slaveID, address, values[0]))

		// Request PDU: FC(1) + Addr(2) + Value(2)
		requestPDU := make([]byte, 5)
		requestPDU[0] = 0x05
		binary.BigEndian.PutUint16(requestPDU[1:3], address)
		binary.BigEndian.PutUint16(requestPDU[3:5], value)

		logger.Debug(fmt.Sprintf("WriteSingleCoil: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
		
		results, err := c.client.WriteSingleCoil(address, value)
		if err != nil {
			if modbusErr, ok := err.(*modbus.ModbusError); ok {
				response := []byte{0x85, modbusErr.ExceptionCode}
				logger.Debug(fmt.Sprintf("Modbus Write Single Coil error response (PDU): %x", response))
			}
			c.recordADU(requestPDU, nil, slaveID)
			return fmt.Errorf("failed to write single coil: %w", err)
		}
		c.recordADU(requestPDU, results, slaveID)
		logger.Info(fmt.Sprintf("successfully wrote single coil: Address=%d, Value=%v", address, values[0]))

	} else {
		// 使用功能码 0x0F (Write Multiple Coils)
		byteCount := (len(values) + 7) / 8
		data := make([]byte, byteCount)
		logger.Debug(fmt.Sprintf("attempting to write multiple coils for SlaveID: %d, Address: %d, Quantity: %d", slaveID, address, quantity))

		for i, val := range values {
			if val {
				byteIndex := i / 8
				bitIndex := i % 8
				data[byteIndex] |= 1 << bitIndex
			}
		}

		// Request PDU: FC(1) + Addr(2) + Qty(2) + ByteCount(1) + Data(N)
		requestPDU := make([]byte, 6+len(data))
		requestPDU[0] = 0x0F
		binary.BigEndian.PutUint16(requestPDU[1:3], address)
		binary.BigEndian.PutUint16(requestPDU[3:5], quantity)
		requestPDU[5] = byte(len(data))
		copy(requestPDU[6:], data)

		logger.Debug(fmt.Sprintf("WriteCoils: Constructed Request PDU: %x (Length: %d)", requestPDU, len(requestPDU)))
		
		results, err := c.client.WriteMultipleCoils(address, quantity, data)
		if err != nil {
			if modbusErr, ok := err.(*modbus.ModbusError); ok {
				response := []byte{0x8F, modbusErr.ExceptionCode}
				logger.Debug(fmt.Sprintf("Modbus Write Coils error response (PDU): %x", response))
			}
			c.recordADU(requestPDU, nil, slaveID)
			return fmt.Errorf("failed to write multiple coils: %w", err)
		}
		c.recordADU(requestPDU, results, slaveID)
		logger.Info(fmt.Sprintf("successfully wrote multiple coils: Address=%d, Quantity=%d", address, quantity))
	}
	return nil
}

// 辅助函数
func bytesToUint16Array(data []byte) []uint16 {
	count := len(data) / 2
	result := make([]uint16, count)
	for i := 0; i < count; i++ {
		result[i] = binary.BigEndian.Uint16(data[i*2 : (i+1)*2])
	}
	return result
}

func uint16ArrayToBytes(data []uint16) []byte {
	result := make([]byte, len(data)*2)
	for i, val := range data {
		binary.BigEndian.PutUint16(result[i*2:], val)
	}
	return result
}

// GetLastPackets 获取最后的发送和接收报文
func (c *Client) GetLastPackets() ([]byte, []byte) {
	c.packetMutex.RLock()
	defer c.packetMutex.RUnlock()
	return c.lastSentPacket, c.lastReceivedPacket
}



// calculateCRC 计算Modbus RTU的CRC-16校验码
func calculateCRC(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i])
		for j := 8; j != 0; j-- {
			if (crc & 0x0001) != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	// 返回低字节在前，高字节在后的结果
	return (crc >> 8) | (crc << 8)
}

// recordADU 构建并记录完整的请求和响应ADU
// requestPDU: 仅包含功能码和数据部分的PDU (e.g., [0x03, addr_high, addr_low, count_high, count_low])
// responsePDU: 仅包含数据部分的PDU (e.g., [reg1_high, reg1_low, reg2_high, reg2_low])
func (c *Client) recordADU(requestPDU, responsePDU []byte, slaveID byte) {
	c.packetMutex.Lock()
	defer c.packetMutex.Unlock()

	logger.Debug(fmt.Sprintf("recordADU: Request PDU: %x, Response PDU (data only): %x, Slave ID: %d", requestPDU, responsePDU, slaveID))

	// --- 构建请求ADU ---
	if c.connectionType == TCP {
		c.transactionIDMutex.Lock()
		c.transactionID++
		tid := c.transactionID
		c.transactionIDMutex.Unlock()

		// MBAP Header: TransactionID(2) + ProtocolID(2) + Length(2) + UnitID(1)
		header := make([]byte, 7)
		binary.BigEndian.PutUint16(header[0:2], tid)
		binary.BigEndian.PutUint16(header[2:4], 0) // Protocol ID is 0
		binary.BigEndian.PutUint16(header[4:6], uint16(len(requestPDU)+1))
		header[6] = slaveID // Use passed slaveID
		c.lastSentPacket = append(header, requestPDU...)
		logger.Info(fmt.Sprintf("Modbus TCP Sent ADU: %x", c.lastSentPacket))
		logger.Debug(fmt.Sprintf("recordADU (TCP): Constructed Sent ADU: %x", c.lastSentPacket))

		// --- 构建响应ADU (TCP) ---
		if responsePDU != nil {
			var fullResponsePDU []byte
			requestFuncCode := requestPDU[0] // Get the function code from the original request

			// Reconstruct the full response PDU (Function Code + Byte Count/Echo Data + Data)
			switch requestFuncCode {
			case 0x01, 0x02, 0x03, 0x04: // Read Coils/Inputs
				byteCount := byte(len(responsePDU))
				fullResponsePDU = make([]byte, 2+len(responsePDU))
				fullResponsePDU[0] = requestFuncCode
				fullResponsePDU[1] = byteCount
				copy(fullResponsePDU[2:], responsePDU)
			case 0x05, 0x06, 0x0F, 0x10: // Write Single/Multiple
				// For write responses, the PDU is echoed back (FC + Addr + Qty/Value)
				fullResponsePDU = make([]byte, 1+len(responsePDU))
				fullResponsePDU[0] = requestFuncCode
				copy(fullResponsePDU[1:], responsePDU)
			default:
				logger.Warn(fmt.Sprintf("recordADU (TCP): Unknown Modbus function code %x for response PDU reconstruction. Using raw responsePDU.", requestFuncCode))
				fullResponsePDU = responsePDU
			}

			responseHeader := make([]byte, 7)
			binary.BigEndian.PutUint16(responseHeader[0:2], tid) // Use the same transaction ID
			binary.BigEndian.PutUint16(responseHeader[2:4], 0)
			binary.BigEndian.PutUint16(responseHeader[4:6], uint16(len(fullResponsePDU)+1))
			responseHeader[6] = slaveID
			c.lastReceivedPacket = append(responseHeader, fullResponsePDU...)
			logger.Info(fmt.Sprintf("Modbus TCP Received ADU: %x", c.lastReceivedPacket))
			logger.Debug(fmt.Sprintf("recordADU (TCP): Constructed Received ADU: %x", c.lastReceivedPacket))
		} else {
			c.lastReceivedPacket = nil // Clear if no response
			logger.Info("Modbus TCP Received ADU: (No response received)")
			logger.Debug("recordADU (TCP): No response PDU provided, clearing lastReceivedPacket.")
		}

	} else { // RTU
		// Request ADU
		adu := append([]byte{slaveID}, requestPDU...) // Use passed slaveID
		crc := calculateCRC(adu)
		c.lastSentPacket = append(adu, byte(crc&0xFF), byte(crc>>8))
		logger.Info(fmt.Sprintf("Modbus RTU Sent ADU: %x", c.lastSentPacket))
		logger.Debug(fmt.Sprintf("recordADU (RTU): Constructed Sent ADU: %x", c.lastSentPacket))

		// --- 构建响应ADU (RTU) ---
		if responsePDU != nil {
			var fullResponsePDU []byte
			requestFuncCode := requestPDU[0] // Get the function code from the original request

			// Reconstruct the full response PDU (Function Code + Byte Count/Echo Data + Data)
			switch requestFuncCode {
			case 0x01, 0x02, 0x03, 0x04: // Read Coils/Inputs
				byteCount := byte(len(responsePDU))
				fullResponsePDU = make([]byte, 2+len(responsePDU))
				fullResponsePDU[0] = requestFuncCode
				fullResponsePDU[1] = byteCount
				copy(fullResponsePDU[2:], responsePDU)
			case 0x05, 0x06, 0x0F, 0x10: // Write Single/Multiple
				// For write responses, the PDU is echoed back (FC + Addr + Qty/Value)
				fullResponsePDU = make([]byte, 1+len(responsePDU))
				fullResponsePDU[0] = requestFuncCode
				copy(fullResponsePDU[1:], responsePDU)
			default:
				logger.Warn(fmt.Sprintf("recordADU (RTU): Unknown Modbus function code %x for response PDU reconstruction. Using raw responsePDU.", requestFuncCode))
				fullResponsePDU = responsePDU
			}

			// RTU response: SlaveID(1) + FullPDU(N) + CRC(2)
			responseADU := append([]byte{slaveID}, fullResponsePDU...)
			responseCRC := calculateCRC(responseADU)
			c.lastReceivedPacket = append(responseADU, byte(responseCRC&0xFF), byte(responseCRC>>8))
			logger.Info(fmt.Sprintf("Modbus RTU Received ADU: %x", c.lastReceivedPacket))
			logger.Debug(fmt.Sprintf("recordADU (RTU): Constructed Received ADU: %x", c.lastReceivedPacket))
		} else {
			c.lastReceivedPacket = nil // Clear if no response
			logger.Info("Modbus RTU Received ADU: (No response received)")
			logger.Debug("recordADU (RTU): No response PDU provided, clearing lastReceivedPacket.")
		}
	}
}