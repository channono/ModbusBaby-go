package gui

import (
	"fmt"
	"modbusbaby/internal/config"
	"modbusbaby/internal/modbus"
	"modbusbaby/pkg/datatypes"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
)
type AppRefined struct {
	fyneApp fyne.App
	window  fyne.Window
	config  *config.Config
	modbus  *modbus.Client
	version string
	author  string

	// === 标题区域 ===
	logoLabel   *widget.Label
	authorLabel *widget.Label

	// === 连接设置区域 ===
	connectionType *widget.Select
	connectBtn     *widget.Button

	// TCP设置
	ipAddressEntry *widget.Entry
	portEntry      *widget.Entry
	

	// RTU设置
	serialPort *widget.Select
	baudRate   *widget.Select
	dataBits   *widget.Select
	stopBits   *widget.Select
	parity     *widget.Select


	// === 寄存器操作区域 ===
	slaveIdTcp        *widget.Entry
	slaveIdRtu        *widget.Entry	
	startAddressInput *widget.Entry
	endAddressInput   *widget.Entry
	registerTypeCombo *widget.Select
	dataTypeCombo     *widget.Select
	byteOrderCombo    *widget.Select
	wordOrderCombo    *widget.Select
	valueInput        *widget.Entry
	readButton        *widget.Button
	writeButton       *widget.Button

	// === 显示区域 ===
	logOutput             *widget.Entry
	sentPacketDisplay     *widget.Entry
	receivedPacketDisplay *widget.Entry
	clearInfoButton       *widget.Button

	// === 轮询设置 ===
	pollingIntervalInput *widget.Entry
	startPollingButton   *widget.Button
	stopPollingButton    *widget.Button

	// 状态管理
	isConnected bool
	pollingStop chan bool

	// 从站地址字节
	slaveIDByte byte  
}
 
func NewAppRefined(cfg *config.Config, version, author string) *AppRefined {
	fyneApp := app.NewWithID("com.biggiantbaby.modbusbaby")

	window := fyneApp.NewWindow("ModbusBaby - by Daniel BigGiantBaby")
	window.Resize(fyne.NewSize(1200, 800)) // 稍微加宽以适应布局
	window.CenterOnScreen()

	return &AppRefined{
		fyneApp:     fyneApp,
		window:      window,
		config:      cfg,
		modbus:      modbus.NewClient(),
		version:     version,
		author:      author,
		pollingStop: make(chan bool),
	}
}

// ShowAndRun 显示并运行应用程序
func (a *AppRefined) ShowAndRun() {
	a.initUI()
	a.window.ShowAndRun()
}

// initUI 初始化用户界面 
func (a *AppRefined) initUI() {
	a.createUIElements()
	a.setupValidators()


	// 设置按钮事件
	a.connectBtn.OnTapped = a.toggleConnection
	a.readButton.OnTapped = func() {
		if a.connectionType.Selected == "Modbus TCP" {
			if a.slaveIdTcp.Text != "" {
				slaveID, err := strconv.Atoi(a.slaveIdTcp.Text)
				if err == nil {
					a.slaveIDByte = byte(slaveID)
					a.readRegister(a.slaveIDByte)
				}
			}
		} else if a.slaveIdRtu.Text != "" {
			slaveID, err := strconv.Atoi(a.slaveIdRtu.Text)
			if err == nil {
				a.slaveIDByte = byte(slaveID)
				a.readRegister(a.slaveIDByte)
			}
		}
	}
	a.writeButton.OnTapped = func() {
		if a.connectionType.Selected == "Modbus TCP" {
			// 如果是TCP连接，使用TCP从站ID
			if a.slaveIdTcp.Text != "" {
				slaveID, err := strconv.Atoi(a.slaveIdTcp.Text)
				if err == nil {
					a.slaveIDByte = byte(slaveID)
					a.writeRegister(a.slaveIDByte)
				}
			}
		} else if a.slaveIdRtu.Text != "" {
			slaveID, err := strconv.Atoi(a.slaveIdRtu.Text)
			if err == nil {
				a.slaveIDByte = byte(slaveID)
				a.writeRegister(a.slaveIDByte)
			}
		}
	}
	a.startPollingButton.OnTapped = func() {
		a.startPolling(a.slaveIDByte)
	}
	a.stopPollingButton.OnTapped = a.stopPolling

	a.clearInfoButton.OnTapped = a.clearAll

	// 设置数据转换器更新事件
	a.byteOrderCombo.OnChanged = func(s string) {
		a.modbus.SetDataConverter(stringToByteOrder(a.byteOrderCombo.Selected), stringToWordOrder(a.wordOrderCombo.Selected))
	}
	a.wordOrderCombo.OnChanged = func(s string) {
		a.modbus.SetDataConverter(stringToByteOrder(a.byteOrderCombo.Selected), stringToWordOrder(a.wordOrderCombo.Selected))
	}
	// 创建主布局
	centralWidget := a.createMainLayout()
	a.window.SetContent(centralWidget)
	a.updateConnectionStateUI() // Set initial UI state
}

// createUIElements 创建UI元素
func (a *AppRefined) createUIElements() {
	// === 标题元素 ===
	a.logoLabel = widget.NewLabel("👶 Big GiantBaby 😧")
	a.authorLabel = widget.NewLabel("😄 大牛大巨婴 😊")
	a.authorLabel.TextStyle = fyne.TextStyle{Bold: true}
	a.authorLabel.Alignment = fyne.TextAlignTrailing

	// === 连接类型选择 ===
	a.connectionType = widget.NewSelect([]string{"Modbus TCP", "Modbus RTU"}, nil)
	a.connectionType.SetSelected("Modbus TCP")

	a.connectBtn = widget.NewButton("连接", nil)

	// === TCP设置元素 ===
	a.ipAddressEntry = widget.NewEntry()
	a.ipAddressEntry.SetText(a.config.TCP.IP)
	a.ipAddressEntry.PlaceHolder = "e.g., 192.168.1.100"

	a.portEntry = widget.NewEntry()
	a.portEntry.SetText(strconv.Itoa(a.config.TCP.Port))

	a.slaveIdTcp = widget.NewEntry()
	a.slaveIdTcp.SetText(strconv.Itoa(a.config.TCP.SlaveID))

	// === RTU设置元素 ===
	a.serialPort = widget.NewSelect([]string{}, nil)
	a.serialPort.PlaceHolder = "Select Serial Port"

	a.baudRate = widget.NewSelect([]string{"2400", "4800", "9600", "19200", "38400", "57600", "115200","230400"}, nil)
	a.baudRate.SetSelected("9600")

	a.dataBits = widget.NewSelect([]string{"8", "7"}, nil)
	a.dataBits.SetSelected("8")

	a.stopBits = widget.NewSelect([]string{"1", "2"}, nil)
	a.stopBits.SetSelected("1")

	a.parity = widget.NewSelect([]string{"None", "Even", "Odd"}, nil)
	a.parity.SetSelected("None")

	a.slaveIdRtu = widget.NewEntry()
	a.slaveIdRtu.SetText("1")

	// === 操作区域元素 ===
	a.startAddressInput = widget.NewEntry()
	a.startAddressInput.SetText("1")

	a.endAddressInput = widget.NewEntry()
	a.endAddressInput.SetText("32")

	a.registerTypeCombo = widget.NewSelect([]string{
		"Holding Register", "Input Register", "Discrete Input", "Coil",
	}, nil)
	a.registerTypeCombo.PlaceHolder = "Select Register Type"
	a.registerTypeCombo.SetSelected("Holding Register")

	a.dataTypeCombo = widget.NewSelect([]string{
		"BYTE", "INT16", "UINT16", "INT32", "UINT32", "INT64", "UINT64",
		"FLOAT32", "FLOAT64", "BOOL", "ASCII", "UNIX_TIMESTAMP",
	}, nil)
	a.dataTypeCombo.SetSelected("UINT16")

	a.byteOrderCombo = widget.NewSelect([]string{"AB", "BA"}, nil)
	a.byteOrderCombo.SetSelected("AB")

	a.wordOrderCombo = widget.NewSelect([]string{"1234", "4321"}, nil)
	a.wordOrderCombo.SetSelected("1234")

	a.readButton = widget.NewButton("读取", nil)
	a.readButton.Disable()

	a.valueInput = widget.NewMultiLineEntry()
	a.valueInput.Wrapping = fyne.TextWrapWord
	a.valueInput.SetMinRowsVisible(3)

	a.writeButton = widget.NewButton("写入", nil)
	a.writeButton.Disable()

	// === 显示区域元素 ===
	a.logOutput = widget.NewMultiLineEntry()
	a.logOutput.Wrapping = fyne.TextWrapWord

	a.sentPacketDisplay = widget.NewMultiLineEntry()
	a.sentPacketDisplay.Wrapping = fyne.TextWrapWord

	a.receivedPacketDisplay = widget.NewMultiLineEntry()
	a.receivedPacketDisplay.Wrapping = fyne.TextWrapWord

	a.clearInfoButton = widget.NewButton("清空", nil)

	// === 轮询设置元素 ===
	a.pollingIntervalInput = widget.NewEntry()
	a.pollingIntervalInput.PlaceHolder = "e.g., 1000"
	a.pollingIntervalInput.SetText(strconv.Itoa(a.config.PollingInterval))

	a.startPollingButton = widget.NewButton("开始轮询", nil)
	a.startPollingButton.Disable()

	a.stopPollingButton = widget.NewButton("停止轮询", nil)
	a.stopPollingButton.Disable()

	a.populateSerialPorts() // Populate serial ports after all UI elements are created
}

// createMainLayout 创建主布局 - 完全对应Python版本布局
func (a *AppRefined) createMainLayout() fyne.CanvasObject {
	titleRow := a.addTitleRow()
	settingsArea := a.addSettingsArea()
	displayArea := a.addDisplayArea()
	pollingSettings := a.addPollingSettings()

	return container.NewBorder(
		container.NewVBox(titleRow, widget.NewSeparator(), settingsArea, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), pollingSettings),
		nil, nil,
		displayArea,
	)
}

// addTitleRow 添加标题行
func (a *AppRefined) addTitleRow() fyne.CanvasObject {
	return container.NewBorder(nil, nil, a.logoLabel, a.authorLabel)
}

// addSettingsArea 添加设置区域
func (a *AppRefined) addSettingsArea() fyne.CanvasObject {
	connectionLayout := container.NewHBox(
		widget.NewLabel("连接类型:"),
		a.connectionType,
		layout.NewSpacer(),
		a.connectBtn,
	)

	tcpSettings := a.createTCPSettingsLayout()
	rtuSettings := a.createRTUSettingsLayout()
	rtuSettings.Hide()

	settingsContainer := container.NewStack(tcpSettings, rtuSettings)

	a.connectionType.OnChanged = func(selected string) {
		if selected == "Modbus TCP" {
			rtuSettings.Hide()
			tcpSettings.Show()
		} else {
			tcpSettings.Hide()
			rtuSettings.Show()
			a.populateSerialPorts() // Enumerate serial ports when switching to Modbus RTU
		}
	}

	registerLayout := a.createRegisterLayout()

	valueLayout := container.NewBorder(
		nil, nil, widget.NewLabel("数值:"), a.writeButton, a.valueInput,
	)

	settingsContent := container.NewVBox(
		connectionLayout,
		settingsContainer,
		registerLayout,
		valueLayout,
	)

	return widget.NewCard("", "", settingsContent)
}

// createTCPSettingsLayout 创建TCP设置行
func (a *AppRefined) createTCPSettingsLayout() fyne.CanvasObject {
	ipContainer := container.New(&minWidthLayout{width: 200}, a.ipAddressEntry)
	portContainer := container.New(&fixedWidthLayout{width: 80}, a.portEntry)
	slaveIDContainer := container.New(&fixedWidthLayout{width: 80}, a.slaveIdTcp)

	return container.NewHBox(
		widget.NewLabel("IP 地址:"),
		ipContainer, // Min width
		widget.NewLabel("端口:"),
		portContainer, // Fixed width
		widget.NewLabel("从站地址:"),
		slaveIDContainer, // Fixed width
		layout.NewSpacer(),
	)
}

// createRTUSettingsLayout 创建RTU设置行
func (a *AppRefined) createRTUSettingsLayout() fyne.CanvasObject {
	serialPortContainer := container.New(&minWidthLayout{width: 280}, a.serialPort)
	baudRateContainer := container.New(&minWidthLayout{width: 120}, a.baudRate)
	dataBitsContainer := container.New(&fixedWidthLayout{width: 70}, a.dataBits)
	stopBitsContainer := container.New(&fixedWidthLayout{width: 70}, a.stopBits)
	parityContainer := container.New(&fixedWidthLayout{width: 90}, a.parity)
	slaveIDContainer := container.New(&fixedWidthLayout{width: 80}, a.slaveIdRtu)

	return container.NewHBox(
		widget.NewLabel("串口:"),
		serialPortContainer, // Min width
		widget.NewLabel("波特率:"),
		baudRateContainer, // Min width
		widget.NewLabel("数据位:"),
		dataBitsContainer, // Fixed width
		widget.NewLabel("停止位:"),
		stopBitsContainer, // Fixed width
		widget.NewLabel("校验:"),
		parityContainer, // Fixed width
		widget.NewLabel("从站地址:"),
		slaveIDContainer, // Fixed width
		layout.NewSpacer(),
	)
}

// createRegisterLayout 创建寄存器操作行
func (a *AppRefined) createRegisterLayout() fyne.CanvasObject {
	startAddrContainer := container.New(&fixedWidthLayout{width: 80}, a.startAddressInput)
	endAddrContainer := container.New(&fixedWidthLayout{width: 80}, a.endAddressInput)
	dataTypeContainer := container.New(&minWidthLayout{width: 150}, a.dataTypeCombo)
	byteOrderContainer := container.New(&fixedWidthLayout{width: 80}, a.byteOrderCombo)
	wordOrderContainer := container.New(&fixedWidthLayout{width: 80}, a.wordOrderCombo)

	return container.NewHBox(
		widget.NewLabel("起始地址:"),
		startAddrContainer, // Fixed
		widget.NewLabel("结束地址:"),
		endAddrContainer, // Fixed
		widget.NewLabel("寄存器类型:"),
		a.registerTypeCombo, // Flexible
		widget.NewLabel("数据类型:"),
		dataTypeContainer, // Min width
		widget.NewLabel("字节序:"),
		byteOrderContainer, // Fixed
		widget.NewLabel("字序:"),
		wordOrderContainer, // Fixed
		layout.NewSpacer(),
		a.readButton,
	)
}

// addDisplayArea 添加显示区域
func (a *AppRefined) addDisplayArea() fyne.CanvasObject {
	infoHeader := container.NewHBox(
		widget.NewLabel("信息:"),
		layout.NewSpacer(),
		a.clearInfoButton,
	)
	infoContainer := container.NewBorder(infoHeader, nil, nil, nil, a.logOutput)

	sentWithLabel := container.NewBorder(
		container.NewHBox(widget.NewLabel("发送的报文:")), nil, nil, nil, a.sentPacketDisplay,
	)

	receivedWithLabel := container.NewBorder(
		container.NewHBox(widget.NewLabel("接收的报文:")), nil, nil, nil, a.receivedPacketDisplay,
	)

	packetSplitter := container.NewHSplit(sentWithLabel, receivedWithLabel)
	packetSplitter.SetOffset(0.5)

	mainSplitter := container.NewVSplit(infoContainer, packetSplitter)
	mainSplitter.SetOffset(0.6)

	return mainSplitter
}

// addPollingSettings 添加轮询设置
func (a *AppRefined) addPollingSettings() fyne.CanvasObject {
	pollingIntervalContainer := container.New(&minWidthLayout{width: 120}, a.pollingIntervalInput)
	return container.NewHBox(
		layout.NewSpacer(),
		widget.NewLabel("轮询间隔 (ms):"),
		pollingIntervalContainer, 
		a.startPollingButton,
		a.stopPollingButton,
		layout.NewSpacer(),
	)
}

// setupValidators 设置验证器
func (a *AppRefined) setupValidators() {
	// Placeholder for validation logic
}

// === 事件处理方法 ===

func (a *AppRefined) toggleConnection() {
	if a.isConnected {
		a.disconnectFromDevice()
	} else {
		a.connectToDevice()
	}
}

func (a *AppRefined) connectToDevice() {
	a.connectBtn.SetText("连接中...")
	a.connectBtn.Disable()

	connType := a.connectionType.Selected
	var err error
	switch connType {
	case "Modbus TCP":
		ip := a.ipAddressEntry.Text
		port, _ := strconv.Atoi(a.portEntry.Text)
		err = a.modbus.ConnectTCP(ip, port, )
	case "Modbus RTU":
		portName := a.serialPort.Selected
		baudRate, _ := strconv.Atoi(a.baudRate.Selected)
		dataBits, _ := strconv.Atoi(a.dataBits.Selected)
		stopBits, _ := strconv.Atoi(a.stopBits.Selected)
		parity := a.parity.Selected
		err = a.modbus.ConnectRTU(portName, baudRate, dataBits, stopBits, parity)
	default:
		err = fmt.Errorf("未知连接类型: %s", connType)
	}

	if err != nil {
		a.appendLog(fmt.Sprintf("连接失败: %v", err))
		a.isConnected = false
	} else {
		a.appendLog("Connection successful!")
		a.isConnected = true
	}

	a.updateConnectionStateUI()
}

func (a *AppRefined) disconnectFromDevice() {
	if a.modbus.IsConnected() {
		err := a.modbus.Disconnect()
		if err != nil {
			a.appendLog(fmt.Sprintf("断开连接失败: %v", err))
		} else {
			a.appendLog("连接已断开。")
		}
	}
	a.isConnected = false
	a.stopPolling() // Stop polling when disconnected
	a.updateConnectionStateUI()
}

func (a *AppRefined) updateConnectionStateUI() {
	if a.isConnected {
		a.connectBtn.SetText("断开")
		a.connectBtn.Enable()
		a.readButton.Enable()
		a.writeButton.Enable()
		a.startPollingButton.Enable()
		a.stopPollingButton.Disable() // Initially disable stop polling
	} else {
		a.connectBtn.SetText("连接")
		a.connectBtn.Enable()
		a.readButton.Disable()
		a.writeButton.Disable()
		a.startPollingButton.Disable()
		a.stopPollingButton.Disable()
	}
	a.window.Content().Refresh()
}

func (a *AppRefined) readRegister(slaveIDByte byte) {
	if !a.modbus.IsConnected() {
		a.appendLog("设备未连接，无法读取寄存器。")
		return
	}
	startAddr, err := strconv.ParseUint(a.startAddressInput.Text, 10, 16)
	if err != nil {
		a.appendLog(fmt.Sprintf("起始地址无效: %v", err))
		return
	}
	endAddr, err := strconv.ParseUint(a.endAddressInput.Text, 10, 16)
	if err != nil {
		a.appendLog(fmt.Sprintf("结束地址无效: %v", err))
		return
	}

	if endAddr < startAddr {
		a.appendLog("结束地址不能小于起始地址。")
		return
	}
	count := uint16(endAddr - startAddr + 1)

	regType := a.registerTypeCombo.Selected
	dataTypeStr := a.dataTypeCombo.Selected
	dataType := stringToDataType(dataTypeStr)

	var result interface{}
	var readErr error

	a.appendLog(fmt.Sprintf("正在读取: %s, 地址: %d, 数量: %d", regType, startAddr, count))

	switch regType {
	case "Holding Register":
		result, readErr = a.modbus.ReadHoldingRegisters(slaveIDByte, uint16(startAddr), count, dataType)
	case "Input Register":
		result, readErr = a.modbus.ReadInputRegisters(slaveIDByte, uint16(startAddr), count, dataType)
	case "Coil":
		result, readErr = a.modbus.ReadCoils(slaveIDByte, uint16(startAddr), count)
	case "Discrete Input":
		result, readErr = a.modbus.ReadDiscreteInputs(slaveIDByte, uint16(startAddr), count)
	default:
		readErr = fmt.Errorf("不支持的寄存器类型: %s", regType)
	}

	if readErr != nil {
		a.appendLog(fmt.Sprintf("读取失败: %v", readErr))
	} else {
		a.appendLog(fmt.Sprintf("读取成功: %v", result))
		// Format and display the result in valueInput
		displayValue := ""
		switch v := result.(type) {
		case []uint16:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatUint(uint64(val), 10)
			}
			displayValue = strings.Join(strValues, ",")
		case []int16:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatInt(int64(val), 10)
			}
			displayValue = strings.Join(strValues, ",")
		case []int32:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatInt(int64(val), 10)
			}
			displayValue = strings.Join(strValues, ",")
		case []uint32:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatUint(uint64(val), 10)
			}
			displayValue = strings.Join(strValues, ",")
		case []int64:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatInt(int64(val), 10)
			}
			displayValue = strings.Join(strValues, ",")
		case []uint64:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatUint(uint64(val), 10)
			}
			displayValue = strings.Join(strValues, ",")
		case []float32:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatFloat(float64(val), 'f', -1, 32)
			}
			displayValue = strings.Join(strValues, ",")
		case []float64:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatFloat(val, 'f', -1, 64)
			}
			displayValue = strings.Join(strValues, ",")
		case []bool:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = strconv.FormatBool(val)
			}
			displayValue = strings.Join(strValues, ",")
		case string: // ASCII
			displayValue = v
		default:
			displayValue = fmt.Sprintf("%v", result) // Fallback for unknown types
		}
		a.valueInput.SetText(displayValue)
	}

	sent, received := a.modbus.GetLastPackets()
	timestamp := time.Now().Format("15:04:05.000")
	a.sentPacketDisplay.SetText(a.sentPacketDisplay.Text + fmt.Sprintf("[%s] Sent: %X\n", timestamp, sent))
	a.receivedPacketDisplay.SetText(a.receivedPacketDisplay.Text + fmt.Sprintf("[%s] Received: %X\n", timestamp, received))
}

func (a *AppRefined) writeRegister(slaveIDByte byte) {
	if !a.modbus.IsConnected() {
		a.appendLog("设备未连接，无法写入寄存器。")
		return
	}
	regType := a.registerTypeCombo.Selected
	dataTypeStr := a.dataTypeCombo.Selected
	dataType := stringToDataType(dataTypeStr)
	valueStr := a.valueInput.Text

	startAddr, err := strconv.ParseUint(a.startAddressInput.Text, 10, 16)
	if err != nil {
		a.appendLog(fmt.Sprintf("起始地址无效: %v", err))
		return
	}

	var writeErr error
	a.appendLog(fmt.Sprintf("正在写入: %s, 地址: %d", regType, startAddr))

	switch regType {
	case "Holding Register":
		values, err := datatypes.ParseStringToType(valueStr, dataType)
		if err != nil {
			a.appendLog(fmt.Sprintf("解析数值失败: %v", err))
			return
		}
		writeErr = a.modbus.WriteHoldingRegisters(slaveIDByte, uint16(startAddr), values)
	case "Coil":
		values, err := datatypes.ParseStringToType(valueStr, dataType)
		if err != nil {
			a.appendLog(fmt.Sprintf("解析线圈数值失败: %v", err))
			return
		}
		boolValues, ok := values.([]bool)
		if !ok {
			a.appendLog(fmt.Sprintf("内部错误: 无法将解析结果转换为 []bool 类型: %T", values))
			return
		}
		writeErr = a.modbus.WriteCoils(slaveIDByte, uint16(startAddr), boolValues)
	default:
		writeErr = fmt.Errorf("不支持的写入寄存器类型: %s", regType)
	}

	if writeErr != nil {
		a.appendLog(fmt.Sprintf("写入失败: %v", writeErr))
	} else {
		a.appendLog("写入成功！")
	}

	sent, received := a.modbus.GetLastPackets()
	timestamp := time.Now().Format("15:04:05.000")
	a.sentPacketDisplay.SetText(a.sentPacketDisplay.Text + fmt.Sprintf("[%s] Sent: %X\n", timestamp, sent))
	a.receivedPacketDisplay.SetText(a.receivedPacketDisplay.Text + fmt.Sprintf("[%s] Received: %X\n", timestamp, received))
}

func (a *AppRefined) startPolling(slaveIDByte byte) {
	if !a.isConnected {
		a.appendLog("设备未连接，无法开始轮询。")
		return
	}
	intervalStr := a.pollingIntervalInput.Text
	intervalMs, err := strconv.Atoi(intervalStr)
	if err != nil || intervalMs <= 0 {
		//a.appendLog("轮询间隔无效，请输入正整数。")
		return
	}
	// 确保没有重复的轮询goroutine
	if a.pollingStop != nil {
		a.stopPolling()
	}
	a.pollingStop = make(chan bool)

	a.appendLog(fmt.Sprintf("开始轮询，间隔 %d ms...", intervalMs))
	a.startPollingButton.Disable()
	a.stopPollingButton.Enable()

	go func() {
		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-a.pollingStop:
				a.appendLog("轮询已停止。")
				return
			case <-ticker.C:
				// 执行读取操作
				a.readRegister(slaveIDByte) // Re-use existing read logic
			}
		}
	}()	

}

func (a *AppRefined) stopPolling(	) {
	if a.pollingStop == nil {
		a.startPollingButton.Enable()
		a.stopPollingButton.Disable()
	}
}

func (a *AppRefined) clearAll() {
	a.logOutput.SetText("")
	a.sentPacketDisplay.SetText("")
	a.receivedPacketDisplay.SetText("")
}

func (a *AppRefined) appendLog(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s\n", timestamp, message)
	a.logOutput.SetText(a.logOutput.Text + logMessage)
}

// populateSerialPorts 枚举并填充串口列表
func (a *AppRefined) populateSerialPorts() {
	allPorts, err := serial.GetPortsList()
	if err != nil {
		a.appendLog(fmt.Sprintf("获取串口列表失败: %v", err))
		return
	}

	var filteredPorts []string
	for _, port := range allPorts {
		// On macOS/Linux, prefer /dev/cu. over /dev/tty.
		if strings.HasPrefix(port, "/dev/tty.") {
			continue
		}
		filteredPorts = append(filteredPorts, port)
	}

	if len(filteredPorts) == 0 {
		a.appendLog("未找到可用串口。")
		a.serialPort.SetOptions([]string{"无可用串口"})
		a.serialPort.SetSelected("无可用串口")
		return
	}

	a.serialPort.SetOptions(filteredPorts)

	// Prioritize USB serial ports for default selection
	defaultPort := filteredPorts[0] // Default to the first filtered port
	for _, port := range filteredPorts {
		if strings.Contains(port, "usbmodem") || strings.Contains(port, "usbserial") {
			defaultPort = port
			break
		}
	}
	a.serialPort.SetSelected(defaultPort)
}

// Helper functions for string to enum conversion

func stringToByteOrder(s string) datatypes.ByteOrder {
	switch s {
	case "AB":
		return datatypes.AB
	case "BA":
		return datatypes.BA
	default:
		return datatypes.AB // Default to AB
	}
}

func stringToWordOrder(s string) datatypes.WordOrder {
	switch s {
	case "1234":
		return datatypes.WORD_1234
	case "4321":
		return datatypes.WORD_4321
	default:
		return datatypes.WORD_1234 // Default to 1234
	}
}

func stringToDataType(s string) datatypes.DataType {
	switch s {
	case "BYTE":
		return datatypes.BYTE
	case "INT16":
		return datatypes.INT16
	case "UINT16":
		return datatypes.UINT16
	case "INT32":
		return datatypes.INT32
	case "UINT32":
		return datatypes.UINT32
	case "INT64":
		return datatypes.INT64
	case "UINT64":
		return datatypes.UINT64
	case "FLOAT32":
		return datatypes.FLOAT32
	case "FLOAT64":
		return datatypes.FLOAT64
	case "BOOL":
		return datatypes.BOOL
	case "ASCII":
		return datatypes.ASCII
	case "UNIX_TIMESTAMP":
		return datatypes.UNIX_TIMESTAMP
	default:
		return datatypes.UINT16 // Default to UINT16
	}
}


// fixedWidthLayout is a custom layout that gives its content a fixed width.
type fixedWidthLayout struct {
	width float32
}

func (f *fixedWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(fyne.NewSize(f.width, objects[0].MinSize().Height))
	objects[0].Move(fyne.NewPos(0, (size.Height-objects[0].MinSize().Height))) // Center vertically
}

func (f *fixedWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return fyne.NewSize(f.width, objects[0].MinSize().Height)
}

// minWidthLayout is a custom layout that ensures its content has a minimum width.
type minWidthLayout struct {
	width float32
}

func (m *minWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(size)
	objects[0].Move(fyne.NewPos(0, 0))
}

func (m *minWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	childMin := objects[0].MinSize()
	actualWidth := childMin.Width
	if actualWidth < m.width {
		actualWidth = m.width
	}
	return fyne.NewSize(actualWidth, childMin.Height)
}