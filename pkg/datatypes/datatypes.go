package datatypes

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// DataType 数据类型枚举
type DataType int

const (
	BYTE DataType = iota
	INT16
	UINT16
	INT32
	UINT32
	INT64
	UINT64
	FLOAT32
	FLOAT64
	BOOL
	ASCII
	UNIX_TIMESTAMP
)

// String 返回数据类型的字符串表示
func (dt DataType) String() string {
	switch dt {
	case BYTE:
		return "BYTE"
	case INT16:
		return "INT16"
	case UINT16:
		return "UINT16"
	case INT32:
		return "INT32"
	case UINT32:
		return "UINT32"
	case INT64:
		return "INT64"
	case UINT64:
		return "UINT64"
	case FLOAT32:
		return "FLOAT32"
	case FLOAT64:
		return "FLOAT64"
	case BOOL:
		return "BOOL"
	case ASCII:
		return "ASCII"
	case UNIX_TIMESTAMP:
		return "UNIX_TIMESTAMP"
	default:
		return "UNKNOWN"
	}
}

// ByteOrder 字节序
type ByteOrder int

const (
	AB ByteOrder = iota // Big Endian
	BA                  // Little Endian
)

func (bo ByteOrder) String() string {
	switch bo {
	case AB:
		return "AB"
	case BA:
		return "BA"
	default:
		return "AB"
	}
}

// WordOrder 字序
type WordOrder int

const (
	WORD_1234 WordOrder = iota // Big Endian
	WORD_4321                  // Little Endian
)

func (wo WordOrder) String() string {
	switch wo {
	case WORD_1234:
		return "1234"
	case WORD_4321:
		return "4321"
	default:
		return "1234"
	}
}

// RegistersPerValue 返回每个值需要的寄存器数量
func (dt DataType) RegistersPerValue() int {
	switch dt {
	case BYTE, INT16, UINT16, BOOL:
		return 1
	case INT32, UINT32, FLOAT32, UNIX_TIMESTAMP:
		return 2
	case INT64, UINT64, FLOAT64:
		return 4
	case ASCII:
		return 1 // 每个寄存器2个字符
	default:
		return 1
	}
}

// Converter 数据转换器
type Converter struct {
	byteOrder ByteOrder
	wordOrder WordOrder
}

// NewConverter 创建新的数据转换器
func NewConverter(byteOrder ByteOrder, wordOrder WordOrder) *Converter {
	return &Converter{
		byteOrder: byteOrder,
		wordOrder: wordOrder,
	}
}

// ConvertFromRegisters 从寄存器数据转换为指定类型
func (c *Converter) ConvertFromRegisters(registers []uint16, dataType DataType) (interface{}, error) {
	if len(registers) == 0 {
		return nil, fmt.Errorf("寄存器数据为空")
	}

	switch dataType {
	case BYTE:
		return c.convertToBytes(registers), nil
	case INT16:
		return c.convertToInt16Array(registers), nil
	case UINT16:
		return c.convertToUint16Array(registers), nil
	case INT32:
		return c.convertToInt32Array(registers), nil
	case UINT32:
		return c.convertToUint32Array(registers), nil
	case INT64:
		return c.convertToInt64Array(registers), nil
	case UINT64:
		return c.convertToUint64Array(registers), nil
	case FLOAT32:
		return c.convertToFloat32Array(registers), nil
	case FLOAT64:
		return c.convertToFloat64Array(registers), nil
	case BOOL:
		return c.convertToBoolArray(registers), nil
	case ASCII:
		return c.convertToASCII(registers), nil
	case UNIX_TIMESTAMP:
		return c.convertToTimestamp(registers), nil
	default:
		return registers, nil
	}
}

// ConvertToRegisters 将值转换为寄存器数据
func (c *Converter) ConvertToRegisters(value interface{}) ([]uint16, error) {
	var registers []uint16
	switch v := value.(type) {
	case []int16:
		for _, val := range v {
			registers = append(registers, uint16(val))
		}
	case []uint16:
		registers = v
	case []int32:
		for _, val := range v {
			registers = append(registers, c.int32ToRegisters(val)...)
		}
	case []uint32:
		for _, val := range v {
			registers = append(registers, c.uint32ToRegisters(val)...)
		}
	case []int64:
		for _, val := range v {
			registers = append(registers, c.int64ToRegisters(val)...)
		}
	case []uint64:
		for _, val := range v {
			registers = append(registers, c.uint64ToRegisters(val)...)
		}
	case []float32:
		for _, val := range v {
			registers = append(registers, c.float32ToRegisters(val)...)
		}
	case []float64:
		for _, val := range v {
			registers = append(registers, c.float64ToRegisters(val)...)
		}
	case string: // For ASCII
		registers = c.asciiToRegisters(v)
	default:
		return nil, fmt.Errorf("unsupported type for conversion to registers: %T", value)
	}
	return registers, nil
}

// ParseStringToType parses a comma-separated string of values into a slice of the specified data type.
func ParseStringToType(valueStr string, dataType DataType) (interface{}, error) {
	parts := strings.Split(valueStr, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("value string is empty")
	}

	switch dataType {
	case INT16:
		var values []int16
		for _, p := range parts {
			val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid INT16 value: %s", p)
			}
			values = append(values, int16(val))
		}
		return values, nil
	case UINT16:
		var values []uint16
		for _, p := range parts {
			val, err := strconv.ParseUint(strings.TrimSpace(p), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid UINT16 value: %s", p)
			}
			values = append(values, uint16(val))
		}
		return values, nil
	case INT32:
		var values []int32
		for _, p := range parts {
			val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid INT32 value: %s", p)
			}
			values = append(values, int32(val))
		}
		return values, nil
	case UINT32:
		var values []uint32
		for _, p := range parts {
			val, err := strconv.ParseUint(strings.TrimSpace(p), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid UINT32 value: %s", p)
			}
			values = append(values, uint32(val))
		}
		return values, nil
	case INT64:
		var values []int64
		for _, p := range parts {
			val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid INT64 value: %s", p)
			}
			values = append(values, val)
		}
		return values, nil
	case UINT64:
		var values []uint64
		for _, p := range parts {
			val, err := strconv.ParseUint(strings.TrimSpace(p), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid UINT64 value: %s", p)
			}
			values = append(values, val)
		}
		return values, nil
	case FLOAT32:
		var values []float32
		for _, p := range parts {
			val, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
			if err != nil {
				return nil, fmt.Errorf("invalid FLOAT32 value: %s", p)
			}
			values = append(values, float32(val))
		}
		return values, nil
	case FLOAT64:
		var values []float64
		for _, p := range parts {
			val, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return nil, fmt.Errorf("invalid FLOAT64 value: %s", p)
			}
			values = append(values, val)
		}
		return values, nil
	case ASCII:
		return valueStr, nil // Keep as a single string
	case BOOL:
		var values []bool
		for _, p := range parts {
			val, err := strconv.ParseBool(strings.TrimSpace(p))
			if err != nil {
				return nil, fmt.Errorf("invalid BOOL value: %s", p)
			}
			values = append(values, val)
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported data type for string parsing: %s", dataType.String())
	}
}

// 内部转换方法
func (c *Converter) convertToBytes(registers []uint16) []byte {
	var result []byte
	for _, reg := range registers {
		if c.byteOrder == AB {
			result = append(result, byte(reg>>8), byte(reg&0xFF))
		} else {
			result = append(result, byte(reg&0xFF), byte(reg>>8))
		}
	}
	return result
}

func (c *Converter) convertToInt16Array(registers []uint16) []int16 {
	result := make([]int16, len(registers))
	for i, reg := range registers {
		result[i] = int16(reg)
	}
	return result
}

func (c *Converter) convertToUint16Array(registers []uint16) []uint16 {
	return registers
}

func (c *Converter) convertToInt32Array(registers []uint16) []int32 {
	var result []int32
	for i := 0; i < len(registers); i += 2 {
		if i+1 < len(registers) {
			var val uint32
			if c.wordOrder == WORD_1234 {
				val = uint32(registers[i])<<16 | uint32(registers[i+1])
			} else {
				val = uint32(registers[i+1])<<16 | uint32(registers[i])
			}
			result = append(result, int32(val))
		}
	}
	return result
}

func (c *Converter) convertToUint32Array(registers []uint16) []uint32 {
	var result []uint32
	for i := 0; i < len(registers); i += 2 {
		if i+1 < len(registers) {
			var val uint32
			if c.wordOrder == WORD_1234 {
				val = uint32(registers[i])<<16 | uint32(registers[i+1])
			} else {
				val = uint32(registers[i+1])<<16 | uint32(registers[i])
			}
			result = append(result, val)
		}
	}
	return result
}

func (c *Converter) convertToInt64Array(registers []uint16) []int64 {
	var result []int64
	for i := 0; i < len(registers); i += 4 {
		if i+3 < len(registers) {
			var val uint64
			if c.wordOrder == WORD_1234 {
				val = uint64(registers[i])<<48 | uint64(registers[i+1])<<32 |
					uint64(registers[i+2])<<16 | uint64(registers[i+3])
			} else {
				val = uint64(registers[i+3])<<48 | uint64(registers[i+2])<<32 |
					uint64(registers[i+1])<<16 | uint64(registers[i])
			}
			result = append(result, int64(val))
		}
	}
	return result
}

func (c *Converter) convertToUint64Array(registers []uint16) []uint64 {
	var result []uint64
	for i := 0; i < len(registers); i += 4 {
		if i+3 < len(registers) {
			var val uint64
			if c.wordOrder == WORD_1234 {
				val = uint64(registers[i])<<48 | uint64(registers[i+1])<<32 |
					uint64(registers[i+2])<<16 | uint64(registers[i+3])
			} else {
				val = uint64(registers[i+3])<<48 | uint64(registers[i+2])<<32 |
					uint64(registers[i+1])<<16 | uint64(registers[i])
			}
			result = append(result, val)
		}
	}
	return result
}

func (c *Converter) convertToFloat32Array(registers []uint16) []float32 {
	var result []float32
	for i := 0; i < len(registers); i += 2 {
		if i+1 < len(registers) {
			var bits uint32
			if c.wordOrder == WORD_1234 {
				bits = uint32(registers[i])<<16 | uint32(registers[i+1])
			} else {
				bits = uint32(registers[i+1])<<16 | uint32(registers[i])
			}
			result = append(result, math.Float32frombits(bits))
		}
	}
	return result
}

func (c *Converter) convertToFloat64Array(registers []uint16) []float64 {
	var result []float64
	for i := 0; i < len(registers); i += 4 {
		if i+3 < len(registers) {
			var bits uint64
			if c.wordOrder == WORD_1234 {
				bits = uint64(registers[i])<<48 | uint64(registers[i+1])<<32 |
					uint64(registers[i+2])<<16 | uint64(registers[i+3])
			} else {
				bits = uint64(registers[i+3])<<48 | uint64(registers[i+2])<<32 |
					uint64(registers[i+1])<<16 | uint64(registers[i])
			}
			result = append(result, math.Float64frombits(bits))
		}
	}
	return result
}

func (c *Converter) convertToBoolArray(registers []uint16) []bool {
	var result []bool
	for _, reg := range registers {
		for i := 0; i < 16; i++ {
			result = append(result, (reg&(1<<i)) != 0)
		}
	}
	return result
}

func (c *Converter) convertToASCII(registers []uint16) string {
	var chars []byte
	for _, reg := range registers {
		if c.byteOrder == AB {
			chars = append(chars, byte(reg>>8), byte(reg&0xFF))
		} else {
			chars = append(chars, byte(reg&0xFF), byte(reg>>8))
		}
	}
	// 移除尾部的空字符
	return strings.TrimRight(string(chars), "\x00")
}

func (c *Converter) convertToTimestamp(registers []uint16) string {
	if len(registers) >= 2 {
		var timestamp uint32
		if c.wordOrder == WORD_1234 {
			timestamp = uint32(registers[0])<<16 | uint32(registers[1])
		} else {
			timestamp = uint32(registers[1])<<16 | uint32(registers[0])
		}

		t := time.Unix(int64(timestamp), 0)
		return t.Format("2006-01-02 15:04:05")
	}
	return "无效时间戳"
}

// 转换为寄存器的辅助方法
func (c *Converter) int32ToRegisters(value int32) []uint16 {
	bits := uint32(value)
	if c.wordOrder == WORD_1234 {
		return []uint16{uint16(bits >> 16), uint16(bits & 0xFFFF)}
	} else {
		return []uint16{uint16(bits & 0xFFFF), uint16(bits >> 16)}
	}
}

func (c *Converter) uint32ToRegisters(value uint32) []uint16 {
	if c.wordOrder == WORD_1234 {
		return []uint16{uint16(value >> 16), uint16(value & 0xFFFF)}
	} else {
		return []uint16{uint16(value & 0xFFFF), uint16(value >> 16)}
	}
}

func (c *Converter) int64ToRegisters(value int64) []uint16 {
	bits := uint64(value)
	if c.wordOrder == WORD_1234 {
		return []uint16{
			uint16(bits >> 48), uint16(bits >> 32),
			uint16(bits >> 16), uint16(bits & 0xFFFF),
		}
	} else {
		return []uint16{
			uint16(bits & 0xFFFF), uint16(bits >> 16),
			uint16(bits >> 32), uint16(bits >> 48),
		}
	}
}

func (c *Converter) uint64ToRegisters(value uint64) []uint16 {
	bits := value
	if c.wordOrder == WORD_1234 {
		return []uint16{
			uint16(bits >> 48), uint16(bits >> 32),
			uint16(bits >> 16), uint16(bits & 0xFFFF),
		}
	} else {
		return []uint16{
			uint16(bits & 0xFFFF), uint16(bits >> 16),
			uint16(bits >> 32), uint16(bits >> 48),
		}
	}
}

func (c *Converter) float32ToRegisters(value float32) []uint16 {
	bits := math.Float32bits(value)
	if c.wordOrder == WORD_1234 {
		return []uint16{uint16(bits >> 16), uint16(bits & 0xFFFF)}
	} else {
		return []uint16{uint16(bits & 0xFFFF), uint16(bits >> 16)}
	}
}

func (c *Converter) float64ToRegisters(value float64) []uint16 {
	bits := math.Float64bits(value)
	if c.wordOrder == WORD_1234 {
		return []uint16{
			uint16(bits >> 48), uint16(bits >> 32),
			uint16(bits >> 16), uint16(bits & 0xFFFF),
		}
	} else {
		return []uint16{
			uint16(bits & 0xFFFF), uint16(bits >> 16),
			uint16(bits >> 32), uint16(bits >> 48),
		}
	}
}

func (c *Converter) asciiToRegisters(value string) []uint16 {
	bytes := []byte(value)
	// Pad with null byte if odd length
	if len(bytes)%2 != 0 {
		bytes = append(bytes, 0)
	}
	var registers []uint16
	for i := 0; i < len(bytes); i += 2 {
		if c.byteOrder == AB {
			registers = append(registers, uint16(bytes[i])<<8|uint16(bytes[i+1]))
		} else {
			registers = append(registers, uint16(bytes[i+1])<<8|uint16(bytes[i]))
		}
	}
	return registers
}
