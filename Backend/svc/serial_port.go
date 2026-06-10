package svc

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"tmaxsrv/comm"

	"github.com/spf13/viper"
	"go.bug.st/serial"
	"gopkg.in/yaml.v3"
)

// 组包模式常量
const (
	PacketModeNone = iota
	PacketModeHeaderTail
	PacketModeLine
)

// 串口状态常量
const (
	StateClosed = iota
	StateOpening
	StateOpen
	StateClosing
	StateError
)

// 默认配置
const (
	DefaultReadTimeout  = 100 * time.Millisecond
	DefaultMaxPacketLen = 1024
)

// 串口事件常量
const (
	EventPortOpened = iota
	EventPortClosed
	EventPortError
	EventDataReceived
	EventPacketReceived
)

// 添加串口相关的常量
const (
	SerialEventDataReceived = iota + 1000
	SerialEventPacketReceived
	SerialEventPortOpened
	SerialEventPortClosed
	SerialEventPortError
)

// 串口命令类型
type SerialCmdType int

const (
	SerialCmdOpen SerialCmdType = iota
	SerialCmdClose
	SerialCmdRestart
	SerialCmdWrite
	SerialCmdWriteString
	SerialCmdWriteHex
	SerialCmdGetStatus
	SerialCmdUpdateConfig
	SerialCmdReloadConfig
	SerialCmdClearBuffers
)

// 事件回调函数类型
type EventCallback func(eventType int, data interface{})

// 串口命令响应
type SerialCmdResponse struct {
	Success bool
	Data    interface{}
	Error   string
}

// 串口命令
type SerialCommand struct {
	CmdType  SerialCmdType
	Data     interface{}
	Response chan *SerialCmdResponse
}

// 串口数据消息
type SerialDataMessage struct {
	Type      string // "raw" 或 "packet"
	Data      []byte
	Timestamp time.Time
	PortName  string
	HexDump   string
}

// SerialConfig 串口配置结构
type SerialConfig struct {
	Name        string        `mapstructure:"name" json:"name" yaml:"name"`
	BaudRate    int           `mapstructure:"baudrate" json:"baudrate" yaml:"baudrate"`
	DataBits    int           `mapstructure:"data_bits" json:"data_bits" yaml:"data_bits"`
	StopBits    int           `mapstructure:"stop_bits" json:"stop_bits" yaml:"stop_bits"`
	Parity      string        `mapstructure:"parity" json:"parity" yaml:"parity"`
	ReadTimeout time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
}

// PacketConfig 组包配置
type PacketConfig struct {
	Mode         string `mapstructure:"mode" json:"mode" yaml:"mode"`                      // "header_tail", "line", "none"
	Header       string `mapstructure:"header" json:"header" yaml:"header"`                // 如 "5AA5"
	Tail         string `mapstructure:"tail" json:"tail" yaml:"tail"`                      // 如 "A55A"
	LineEnding   string `mapstructure:"line_ending" json:"line_ending" yaml:"line_ending"` // "CR", "LF", "CRLF"
	MaxPacketLen int    `mapstructure:"max_packet_len" json:"max_packet_len" yaml:"max_packet_len"`
}

// 串口配置包装（用于从配置文件读取）
type SerialConfigWrapper struct {
	Serial   SerialConfig    `mapstructure:"serial" json:"serial" yaml:"serial"`
	Packet   PacketConfig    `mapstructure:"packet" json:"packet" yaml:"packet"`
	AutoOpen bool            `mapstructure:"auto_open_serial" json:"auto_open_serial" yaml:"auto_open_serial"`
	Handlers []HandlerConfig `mapstructure:"serial_handlers" json:"serial_handlers" yaml:"serial_handlers"`
}

// 处理器配置
type HandlerConfig struct {
	Name    string `mapstructure:"name" json:"name" yaml:"name"`
	Enabled bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// PacketHandler 包处理器
type PacketHandler struct {
	mode           int
	buffer         []byte
	header         []byte
	tail           []byte
	lineEnding     []byte
	maxPacketLen   int
	packets        chan []byte
	foundHeader    bool
	packetStartIdx int
	mu             sync.Mutex
}

// SerialPort 串口控制器
type SerialPort struct {
	config        *SerialConfig
	packetConfig  *PacketConfig
	port          serial.Port
	packetHandler *PacketHandler
	state         int
	stateMu       sync.RWMutex

	// 控制通道
	quit      chan struct{}
	readDone  chan struct{}
	writeDone chan struct{}

	// 等待组
	readWG  sync.WaitGroup
	writeWG sync.WaitGroup

	// 数据通道
	readChan   chan []byte
	packetChan chan []byte

	// 事件回调
	eventCallbacks []EventCallback
	callbackMu     sync.RWMutex

	// 错误信息
	lastError error
	errorMu   sync.RWMutex

	// 调试模式
	debug bool

	// 配置文件路径
	configPath string
}

// ============= PacketHandler 方法实现 =============

// newPacketHandler 创建新的包处理器
func (sp *SerialPort) newPacketHandler(config *PacketConfig) *PacketHandler {
	ph := &PacketHandler{
		buffer:       make([]byte, 0),
		packets:      make(chan []byte, 100),
		maxPacketLen: config.MaxPacketLen,
	}

	// 设置组包模式
	switch strings.ToLower(config.Mode) {
	case "header_tail":
		ph.mode = PacketModeHeaderTail
		// 解析包头
		if config.Header != "" {
			header, err := hex.DecodeString(config.Header)
			if err == nil {
				ph.header = header
			}
		}
		// 解析包尾
		if config.Tail != "" {
			tail, err := hex.DecodeString(config.Tail)
			if err == nil {
				ph.tail = tail
			}
		}
		// 如果没有指定包头包尾，使用默认值
		if len(ph.header) == 0 {
			ph.header = []byte{0x5A, 0xA5}
		}
		if len(ph.tail) == 0 {
			ph.tail = []byte{0xA5, 0x5A}
		}

	case "line":
		ph.mode = PacketModeLine
		// 设置行结束符
		switch strings.ToUpper(config.LineEnding) {
		case "CR":
			ph.lineEnding = []byte{'\r'}
		case "LF":
			ph.lineEnding = []byte{'\n'}
		case "CRLF":
			ph.lineEnding = []byte{'\r', '\n'}
		default:
			ph.lineEnding = []byte{'\n'} // 默认使用 LF
		}
	}

	return ph
}

// HandleData 处理接收到的数据
func (ph *PacketHandler) HandleData(data []byte) {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	// 将新数据添加到缓冲区
	ph.buffer = append(ph.buffer, data...)

	// 尝试解析 Modbus 格式（所有模式下都优先尝试，因为它有严格的 CRC 校验，不会误伤其他数据）
	originalLen := len(ph.buffer)
	ph.processModbusPackets()

	// 如果成功处理了 Modbus 包，直接返回，剩下的等下一波
	if len(ph.buffer) < originalLen {
		return
	}

	// 根据模式处理剩余的数据
	switch ph.mode {
	case PacketModeHeaderTail:
		ph.processHeaderTailPackets()
	case PacketModeLine:
		ph.processLinePackets()
	default:
		// 如果没有指定组包模式
		if len(ph.buffer) > 0 {
			// 启发式检查：如果是有效的从机地址和功能码，可能是一个未收完的 modbus 包
			if len(ph.buffer) >= 2 && ph.buffer[0] >= 1 && ph.buffer[0] <= 247 && 
			   (ph.buffer[1] <= 6 || ph.buffer[1] == 0x0F || ph.buffer[1] == 0x10) {
				// 可能是没收完的 modbus 包，保留 buffer 等待下一批数据
				// 防护：如果太长肯定不是合法的 modbus 帧，清空
				if len(ph.buffer) > 256 {
					packet := make([]byte, len(ph.buffer))
					copy(packet, ph.buffer)
					ph.packets <- packet
					ph.buffer = ph.buffer[:0]
				}
			} else {
				// 不是 modbus 包特征，直接作为 raw 包输出并清空
				packet := make([]byte, len(ph.buffer))
				copy(packet, ph.buffer)
				ph.packets <- packet
				ph.buffer = ph.buffer[:0]
			}
		}
	}
}

// GetPackets 获取接收到的包
func (ph *PacketHandler) GetPackets() <-chan []byte {
	return ph.packets
}

// Clear 清空缓冲区
func (ph *PacketHandler) Clear() {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.buffer = ph.buffer[:0]
	ph.foundHeader = false
}

// 处理包头包尾模式的数据
func (ph *PacketHandler) processHeaderTailPackets() {
	bufferLen := len(ph.buffer)
	if bufferLen == 0 {
		return
	}

	i := 0
	for i < bufferLen {
		if !ph.foundHeader {
			// 查找包头
			if i+len(ph.header) <= bufferLen {
				if ph.matchPattern(ph.buffer[i:i+len(ph.header)], ph.header) {
					ph.foundHeader = true
					ph.packetStartIdx = i
					i += len(ph.header)
					continue
				}
			}
			i++
		} else {
			// 已经找到包头，查找包尾
			if i+len(ph.tail) <= bufferLen {
				if ph.matchPattern(ph.buffer[i:i+len(ph.tail)], ph.tail) {
					// 找到完整的包
					packetEnd := i + len(ph.tail)
					packet := make([]byte, packetEnd-ph.packetStartIdx)
					copy(packet, ph.buffer[ph.packetStartIdx:packetEnd])

					// 发送完整包
					ph.packets <- packet

					// 重置状态
					ph.foundHeader = false

					// 移除已处理的包
					ph.buffer = ph.buffer[packetEnd:]
					bufferLen = len(ph.buffer)
					i = 0
					continue
				}
			}

			// 检查包是否过长
			if i-ph.packetStartIdx > ph.maxPacketLen {
				// 包太长，可能是错误的开始，重新查找包头
				ph.foundHeader = false
				i = ph.packetStartIdx + 1
			} else {
				i++
			}
		}
	}
}

// 处理行模式的数据
func (ph *PacketHandler) processLinePackets() {

	// ph.processPackets()
	for {
		// 查找行结束符
		idx := ph.findLineEnding()
		if idx == -1 {
			break
		}

		// 提取一行数据（包括结束符）
		lineEnd := idx + len(ph.lineEnding)
		line := make([]byte, lineEnd)
		copy(line, ph.buffer[:lineEnd])

		// 发送这一行
		ph.packets <- line

		// 移除已处理的数据
		ph.buffer = ph.buffer[lineEnd:]
	}
}

// 查找行结束符
func (ph *PacketHandler) findLineEnding() int {
	bufferLen := len(ph.buffer)
	if bufferLen < len(ph.lineEnding) {
		return -1
	}

	for i := 0; i <= bufferLen-len(ph.lineEnding); i++ {
		if ph.matchPattern(ph.buffer[i:i+len(ph.lineEnding)], ph.lineEnding) {
			return i
		}
	}
	return -1
}

// 匹配模式
func (ph *PacketHandler) matchPattern(data, pattern []byte) bool {
	if len(data) != len(pattern) {
		return false
	}
	for i := range pattern {
		if data[i] != pattern[i] {
			return false
		}
	}
	return true
}

// ============= SerialPort 方法实现 =============

// NewSerialPort 创建新的串口控制器
func NewSerialPort(configPath string) *SerialPort {

	sp := &SerialPort{
		config: &SerialConfig{
			Name:        "",
			BaudRate:    57600,
			DataBits:    8,
			StopBits:    1,
			Parity:      "N",
			ReadTimeout: 100 * time.Millisecond,
		},
		packetConfig: &PacketConfig{
			Mode:         "line",
			Header:       "",
			Tail:         "",
			LineEnding:   "CRLF",
			MaxPacketLen: 1024,
		},
		state:          StateClosed,
		quit:           make(chan struct{}),
		readDone:       make(chan struct{}),
		writeDone:      make(chan struct{}),
		readChan:       make(chan []byte, 100),
		packetChan:     make(chan []byte, 100),
		eventCallbacks: make([]EventCallback, 0),
		configPath:     configPath,
		debug:          false,
	}

	// 初始化包处理器
	sp.packetHandler = sp.newPacketHandler(sp.packetConfig)

	return sp
}

// Open 打开串口
func (sp *SerialPort) Open() error {
	sp.stateMu.Lock()
	defer sp.stateMu.Unlock()

	if sp.state == StateOpen || sp.state == StateOpening {
		return fmt.Errorf("serial port is already open")
	}

	// 如果没有配置，尝试从文件加载
	if sp.config.Name == "" {
		if err := sp.ReloadConfig(); err != nil {
			return fmt.Errorf("failed to reload config: %v", err)
		}
	}

	sp.setState(StateOpening)

	// 解析校验位
	parity := serial.NoParity
	switch strings.ToUpper(sp.config.Parity) {
	case "E":
		parity = serial.EvenParity
	case "O":
		parity = serial.OddParity
	case "N":
		parity = serial.NoParity
	}

	// 解析停止位
	stopBits := serial.OneStopBit
	switch sp.config.StopBits {
	case 1:
		stopBits = serial.OneStopBit
	case 2:
		stopBits = serial.TwoStopBits
	}

	mode := &serial.Mode{
		BaudRate: sp.config.BaudRate,
		DataBits: sp.config.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	}

	if sp.debug {
		fmt.Printf("open serial port: %s, mode: %+v\n", sp.config.Name, mode)
	}

	port, err := serial.Open(sp.config.Name, mode)
	if err != nil {
		sp.setError(err)
		sp.setState(StateError)
		return fmt.Errorf("failed to open serial port: %v", err)
	}

	// 设置超时
	if err := port.SetReadTimeout(sp.config.ReadTimeout); err != nil {
		port.Close()
		sp.setError(err)
		sp.setState(StateError)
		return fmt.Errorf("failed to set read timeout: %v", err)
	}

	sp.port = port
	sp.setState(StateOpen)

	// 重新创建退出通道
	sp.quit = make(chan struct{})
	sp.readDone = make(chan struct{})
	sp.writeDone = make(chan struct{})

	// 启动读取协程
	sp.readWG.Add(1)
	go sp.readLoop()

	// 启动包处理协程
	sp.readWG.Add(1)
	go sp.packetLoop()

	// 触发事件
	sp.fireEvent(EventPortOpened, sp.config.Name)

	return nil
}

// Close 关闭串口
func (sp *SerialPort) Close() error {
	sp.stateMu.Lock()
	defer sp.stateMu.Unlock()

	if sp.state != StateOpen || sp.port == nil {
		return nil
	}

	sp.setState(StateClosing)

	// 发送退出信号
	if sp.quit != nil {
		close(sp.quit)
	}

	// 等待读取协程结束
	sp.readWG.Wait()

	// 关闭串口
	err := sp.port.Close()
	sp.port = nil
	sp.setState(StateClosed)

	// 触发事件
	sp.fireEvent(EventPortClosed, nil)

	return err
}

// Restart 重启串口
func (sp *SerialPort) Restart() error {
	if sp.debug {
		fmt.Println("restart serial port...")
	}

	// 先关闭
	if err := sp.Close(); err != nil {
		sp.setError(err)
	}

	// 等待一下确保资源释放
	time.Sleep(100 * time.Millisecond)

	// 再打开
	return sp.Open()
}

// Write 写入数据（原始字节）
func (sp *SerialPort) Write(data []byte) (int, error) {
	if !sp.IsOpen() {
		return 0, fmt.Errorf("serial port is not open")
	}

	return sp.port.Write(data)
}

// IsOpen 检查串口是否打开
func (sp *SerialPort) IsOpen() bool {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.state == StateOpen && sp.port != nil
}

// 设置状态
func (sp *SerialPort) setState(state int) {
	sp.state = state
}

// 设置错误
func (sp *SerialPort) setError(err error) {
	sp.errorMu.Lock()
	defer sp.errorMu.Unlock()
	sp.lastError = err
	sp.fireEvent(EventPortError, err.Error())
}

// 触发事件
func (sp *SerialPort) fireEvent(eventType int, data interface{}) {
	sp.callbackMu.RLock()
	callbacks := make([]EventCallback, len(sp.eventCallbacks))
	copy(callbacks, sp.eventCallbacks)
	sp.callbackMu.RUnlock()

	for _, callback := range callbacks {
		go callback(eventType, data)
	}
}

// AddEventListener 添加事件监听器
func (sp *SerialPort) AddEventListener(callback EventCallback) {
	sp.callbackMu.Lock()
	defer sp.callbackMu.Unlock()
	sp.eventCallbacks = append(sp.eventCallbacks, callback)
}

// 读取循环
func (sp *SerialPort) readLoop() {
	defer sp.readWG.Done()

	buffer := make([]byte, 1024)

	for {
		select {
		case <-sp.quit:
			return
		default:
		}

		// 检查串口是否打开
		if sp.port == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		n, err := sp.port.Read(buffer)
		if err != nil {
			// 超时错误忽略
			if err.Error() != "timeout" {
				if sp.debug {
					fmt.Printf("read error: %v\n", err)
				}
				sp.setError(err)

				// 如果是严重错误，可能串口已断开
				if strings.Contains(err.Error(), "device") ||
					strings.Contains(err.Error(), "端口") ||
					strings.Contains(err.Error(), "port") {
					// 尝试自动关闭
					sp.stateMu.Lock()
					sp.port = nil
					sp.state = StateError
					sp.stateMu.Unlock()
					sp.fireEvent(EventPortClosed, nil)
					return
				}
			}
			continue
		}

		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			
			// 打印串口原始接收数据，方便调试
			fmt.Printf("【串口 %s 原始接收】 %X\n", sp.config.Name, data)

			// 发送到原始数据通道
			select {
			case sp.readChan <- data:
			default:
				// 通道满了，丢弃旧数据
				if sp.debug {
					fmt.Println("read channel full, discard data")
				}
			}

			// 交给包处理器
			if sp.packetHandler != nil {
				sp.packetHandler.HandleData(data)
			}

			// 触发事件
			sp.fireEvent(EventDataReceived, data)
		}
	}
}

// 包处理循环
func (sp *SerialPort) packetLoop() {
	defer sp.readWG.Done()

	for {
		select {
		case <-sp.quit:
			return
		case packet := <-sp.packetHandler.GetPackets():
			// 发送到包通道
			select {
			case sp.packetChan <- packet:
			default:
				if sp.debug {
					fmt.Println("packet channel full, discard packet")
				}
			}

			// 触发事件
			sp.fireEvent(EventPacketReceived, packet)
		}
	}
}

// SetConfigPath 设置配置文件路径
func (sp *SerialPort) sendGetInput() error {
	if _, err := sp.WriteHex("01020000000479c9"); err != nil {
		return err
	}
	return nil
}

// SetConfigPath 设置配置文件路径
func (sp *SerialPort) SetConfigPath(path string) {
	sp.configPath = path
}

// GetConfig 获取当前串口配置
func (sp *SerialPort) GetConfig() *SerialConfig {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.config
}

// GetPacketConfig 获取当前组包配置
func (sp *SerialPort) GetPacketConfig() *PacketConfig {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.packetConfig
}

// SetConfig 更新配置
func (sp *SerialPort) SetConfig(config *SerialConfig) error {
	sp.stateMu.Lock()
	defer sp.stateMu.Unlock()

	// 如果串口已打开，需要先关闭
	if sp.state == StateOpen {
		return fmt.Errorf("serial port is open, please close it first before updating config")
	}

	sp.config = config
	return nil
}

// SetPacketConfig 更新包配置
func (sp *SerialPort) SetPacketConfig(packetConfig *PacketConfig) {
	sp.packetConfig = packetConfig
	sp.packetHandler = sp.newPacketHandler(packetConfig)
}

// GetStateString 获取状态字符串描述
func (sp *SerialPort) GetStateString() string {
	state := sp.GetState()
	switch state {
	case StateClosed:
		return "Closed"
	case StateOpening:
		return "Opening"
	case StateOpen:
		return "Open"
	case StateClosing:
		return "Closing"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// GetState 获取当前状态
func (sp *SerialPort) GetState() int {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.state
}

// WriteString 写入字符串
func (sp *SerialPort) WriteString(s string) (int, error) {
	return sp.Write([]byte(s))
}

// WriteHex 写入十六进制字符串
func (sp *SerialPort) WriteHex(hexStr string) (int, error) {
	// 移除空格
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "-", "")

	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return 0, fmt.Errorf("invalid hex string: %v", err)
	}

	return sp.Write(data)
}

// ClearBuffers 清空缓冲区
func (sp *SerialPort) ClearBuffers() error {
	if !sp.IsOpen() {
		return fmt.Errorf("serial port is not open")
	}

	// 清空输入缓冲区
	for {
		select {
		case <-sp.readChan:
		case <-sp.packetChan:
		default:
			goto CLEAN_DONE
		}
	}
CLEAN_DONE:

	// 清空包处理器缓冲区
	if sp.packetHandler != nil {
		sp.packetHandler.Clear()
	}

	return nil
}

func (sp *SerialPort) LoadConfig(path string) error {
	if path != "" {
		sp.configPath = path
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(sp.configPath); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		if err := sp.createDefaultConfig(); err != nil {
			return fmt.Errorf("failed to create default config file: %v", err)
		}
		fmt.Printf("Created default configuration file: %s\n", sp.configPath)
	} else if err != nil {
		return fmt.Errorf("failed to check config file: %v", err)
	}

	viper.SetConfigFile(sp.configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var appConfig struct {
		Serial SerialConfig `mapstructure:"serial"`
		Packet PacketConfig `mapstructure:"packet"`
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}

	sp.config = &appConfig.Serial
	sp.packetConfig = &appConfig.Packet
	sp.packetHandler = sp.newPacketHandler(sp.packetConfig)

	return nil
}

// createDefaultConfig 创建默认配置文件
func (sp *SerialPort) createDefaultConfig() error {
	// 确保目录存在
	dir := filepath.Dir(sp.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	defaultConfig := `# Serial port configuration
serial:
  name: COM3                    # Serial port name (Windows: COM3, Linux: /dev/ttyUSB0, /dev/ttyS0)
  baudrate: 57600                # Baud rate (common: 9600, 115200, 19200, 38400, 57600)
  data_bits: 8                  # Data bits (common: 8, 7)
  stop_bits: 1                  # Stop bits (1, 2)
  parity: "N"                   # Parity: "N"(none), "E"(even), "O"(odd)
  read_timeout: 100ms           # Read timeout (milliseconds)

# Packet configuration
packet:
  mode: "line"                  # Packet mode: "header_tail"(header and tail), "line"(line mode), "none"(none)
  header: "5AA5"                # Header (hex string)
  tail: "A55A"                  # Tail (hex string)
  line_ending: "CRLF"           # Line ending: "CR"(carriage return), "LF"(line feed), "CRLF"(carriage return + line feed)
  max_packet_len: 1024          # Maximum packet length (bytes)

# Auto open serial port configuration
auto_open_serial: false          # Whether to automatically open the serial port when the program starts

# Serial handlers configuration
serial_handlers:
  - name: "logger"              # Logger handler
    enabled: true               # Whether to enable
  - name: "modbus"              # Modbus handler
    enabled: true               # Whether to enable
  - name: "custom_handler"      # Custom handler
    enabled: false              # Whether to enable
`

	// 写入配置文件
	if err := os.WriteFile(sp.configPath, []byte(defaultConfig), 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write config file to %s", sp.configPath)
		}
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// ReloadConfig 重新加载配置
func (sp *SerialPort) ReloadConfig() error {
	configPath := "config.yaml" // 默认配置文件路径
	configPath = filepath.Join(comm.GetExePath(), configPath)

	if sp.configPath == "" {
		sp.configPath = configPath
	}
	return sp.LoadConfig("")
}

// LoadSerialConfig 从配置文件加载串口配置
func (sm *SrvMgr) LoadSerialConfig() error {
	viper.SetConfigFile(sm.configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// 加载串口配置
	var config SerialConfigWrapper
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal serial config: %v", err)
	}

	sm.serialMu.Lock()
	defer sm.serialMu.Unlock()

	sm.serialConfig = &config
	sm.autoOpenSerial = config.AutoOpen

	// 更新串口实例的配置
	if config.Serial.Name != "" {
		sm.serialPort.SetConfig(&config.Serial)
	}

	// 更新组包配置
	if config.Packet.Mode != "" {
		sm.serialPort.SetPacketConfig(&config.Packet)
	}

	// 设置配置文件路径
	sm.serialPort.SetConfigPath(sm.configPath)

	return nil
}

// SaveSerialConfig 保存串口配置到文件
func (sm *SrvMgr) SaveSerialConfig() error {
	sm.serialMu.RLock()
	defer sm.serialMu.RUnlock()

	data, err := yaml.Marshal(sm.serialConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal serial config: %v", err)
	}

	if err := ioutil.WriteFile(sm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// 设置串口事件监听
func (sm *SrvMgr) setupSerialEventListeners() {
	sm.serialPort.AddEventListener(func(eventType int, data interface{}) {
		switch eventType {
		case EventPortOpened:
			sm.serialMu.Lock()
			sm.serialStatus.IsOpen = true
			sm.serialStatus.LastError = ""
			sm.serialMu.Unlock()
			fmt.Println("serial port opened")

		case EventPortClosed:
			sm.serialMu.Lock()
			sm.serialStatus.IsOpen = false
			sm.serialMu.Unlock()
			fmt.Println("serial port closed")

		case EventPortError:
			errStr, _ := data.(string)
			sm.serialMu.Lock()
			sm.serialStatus.LastError = errStr
			sm.serialStatus.LastErrorTime = time.Now()
			sm.serialMu.Unlock()
			fmt.Printf("serial port error: %s\n", errStr)

		case EventDataReceived:
			if rawData, ok := data.([]byte); ok {
				sm.serialMu.Lock()
				sm.serialStatus.BytesRead += uint64(len(rawData))
				sm.serialMu.Unlock()

				sm.serialDataChan <- &SerialDataMessage{
					Type:      "raw",
					Data:      rawData,
					Timestamp: time.Now(),
					PortName:  sm.serialPort.GetConfig().Name,
					HexDump:   hex.EncodeToString(rawData),
				}
				sm.inputBuffer <- rawData
				// 解析输入状态

			}

		case EventPacketReceived:
			if packet, ok := data.([]byte); ok {
				sm.serialMu.Lock()
				sm.serialStatus.PacketsRecv++
				sm.serialMu.Unlock()

				sm.serialDataChan <- &SerialDataMessage{
					Type:      "packet",
					Data:      packet,
					Timestamp: time.Now(),
					PortName:  sm.serialPort.GetConfig().Name,
					HexDump:   hex.EncodeToString(packet),
				}
			}
		}
	})
}

// 启动消费者 goroutine
func (sm *SrvMgr) StartParsing() {
	go sm.consumeInputBuffer()
}

func (sm *SrvMgr) consumeInputBuffer() {

	for {
		select {
		case data, ok := <-sm.inputBuffer:
			if !ok {
				// 通道已关闭，退出
				return
			}
			// 将收到的数据追加到缓冲区
			sm.parseMu.Lock()
			sm.parseBuffer = append(sm.parseBuffer, data...)
			sm.parseMu.Unlock()

			// 尝试从缓冲区中解析所有完整帧
			sm.parseFrames()

			// 可选：如果希望定时清理缓冲区或处理超时，可以加一个 time.After
		}
	}
}

// 查询协程
func (sm *SrvMgr) queryLoop() {

	if !sm.autoOpenSerial {
		return
	}

	for {
		cmdWithCRC := []byte{0x01, 0x02, 0x00, 0x00, 0x00, 0x04, 0x79, 0xc9}

		_, err := sm.WriteSerial(cmdWithCRC)
		if err != nil {
			fmt.Printf("写入串口失败: %v\n", err)
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// 注册默认处理器
func (sm *SrvMgr) registerDefaultHandlers() {
	// 数据日志处理器
	sm.RegisterSerialHandler("logger", func(msg *SerialDataMessage) {
		if msg.Type == "raw" {
			// fmt.Printf("[串口原始数据] %s: % X\n",
			// 	msg.Timestamp.Format("15:04:05.000"), msg.Data)
		} else {
			fmt.Printf("[串口数据包] %s: % X\n",
				msg.Timestamp.Format("15:04:05.000"), msg.Data)
			fmt.Printf("[串口数据包] %s: %s\n",
				msg.Timestamp.Format("15:04:05.000"), msg.Data)
		}
	})

	// Modbus 处理器
	sm.RegisterSerialHandler("modbus", func(msg *SerialDataMessage) {
		if msg.Type == "packet" && len(msg.Data) >= 2 {
			fmt.Printf("【Modbus 处理器收到完整包】 %X\n", msg.Data)
			// 解析 Modbus 功能
			functionCode := msg.Data[1] //01 是读取， 05 是写入
			if functionCode == 0x01 {
				// 读取或写入操作
				if len(msg.Data) >= 6 {
					if msg.Data[3] == 0x00 {
						mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_read_output_port", MsgBody: "false"}
					} else if msg.Data[3] == 0x01 {
						mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_read_output_port", MsgBody: "true"}
					}
				}
			} else if functionCode == 0x05 {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_open_output_port", MsgBody: "ok"}
			} else if functionCode == 0x03 {
				// 测试：收到 01 03 00 00 00 01 84 0A 时回复特定数据以建立连接
				sm.WriteSerial([]byte{0x01, 0x03, 0x02, 0x19, 0x98, 0xB2, 0x7E})
			} else {
				return
			}

			// // 根据地址找到对应的秤
			// sm.serialMu.RLock()
			// scale, exists := sm.scales[int64(address)]
			// sm.serialMu.RUnlock()

			// 将数据转发给对应的秤
			// select {
			// case sm.recvScaleMsg <- &ScaleRespMsg{}:

			// default:
			// 	// fmt.Printf("秤消息通道已满，丢弃数据: %d\n", scale.ID)
			// }

		}
	})
}

// 串口管理器主循环
func (sm *SrvMgr) serialManager() {
	for {
		select {
		case cmd := <-sm.serialCmdChan:
			sm.handleSerialCommand(cmd)
		case <-sm.quitch:
			return
		}
	}
}

// 处理串口命令
func (sm *SrvMgr) handleSerialCommand(cmd *SerialCommand) {
	var response SerialCmdResponse

	switch cmd.CmdType {
	case SerialCmdOpen:
		err := sm.serialPort.Open()
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Success = true
			response.Data = "串口打开成功"
		}

	case SerialCmdClose:
		err := sm.serialPort.Close()
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Success = true
			response.Data = "串口关闭成功"
		}

	case SerialCmdRestart:
		err := sm.serialPort.Restart()
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Success = true
			response.Data = "串口重启成功"
		}

	case SerialCmdWrite:
		if data, ok := cmd.Data.([]byte); ok {
			n, err := sm.serialPort.Write(data)
			if err != nil {
				response.Error = err.Error()
			} else {
				sm.serialMu.Lock()
				sm.serialStatus.BytesWritten += uint64(n)
				sm.serialMu.Unlock()

				response.Success = true
				response.Data = map[string]interface{}{
					"bytesWritten": n,
					"hex":          hex.EncodeToString(data),
				}
			}
		} else {
			response.Error = "无效的数据格式"
		}

	case SerialCmdWriteString:
		if str, ok := cmd.Data.(string); ok {
			n, err := sm.serialPort.WriteString(str)
			if err != nil {
				response.Error = err.Error()
			} else {
				response.Success = true
				response.Data = map[string]interface{}{
					"bytesWritten": n,
					"string":       str,
				}
			}
		} else {
			response.Error = "无效的字符串格式"
		}

	case SerialCmdWriteHex:
		if hexStr, ok := cmd.Data.(string); ok {
			n, err := sm.serialPort.WriteHex(hexStr)
			if err != nil {
				response.Error = err.Error()
			} else {
				response.Success = true
				response.Data = map[string]interface{}{
					"bytesWritten": n,
					"hex":          hexStr,
				}
			}
		} else {
			response.Error = "无效的十六进制字符串"
		}

	case SerialCmdGetStatus:
		sm.serialMu.RLock()
		config := sm.serialPort.GetConfig()
		packetConfig := sm.serialPort.GetPacketConfig()
		status := map[string]interface{}{
			"isOpen":        sm.serialStatus.IsOpen,
			"config":        config,
			"packetConfig":  packetConfig,
			"lastError":     sm.serialStatus.LastError,
			"lastErrorTime": sm.serialStatus.LastErrorTime,
			"state":         sm.serialPort.GetStateString(),
			"bytesRead":     sm.serialStatus.BytesRead,
			"bytesWritten":  sm.serialStatus.BytesWritten,
			"packetsRecv":   sm.serialStatus.PacketsRecv,
			"portName":      config.Name,
		}
		sm.serialMu.RUnlock()

		response.Success = true
		response.Data = status

	case SerialCmdUpdateConfig:
		if newConfig, ok := cmd.Data.(*SerialConfig); ok {
			err := sm.serialPort.SetConfig(newConfig)
			if err != nil {
				response.Error = err.Error()
			} else {
				// 更新配置文件中的配置
				sm.serialMu.Lock()
				sm.serialConfig.Serial = *newConfig
				sm.serialMu.Unlock()
				sm.SaveSerialConfig()

				response.Success = true
				response.Data = "配置更新成功"
			}
		} else {
			response.Error = "无效的配置格式"
		}

	case SerialCmdReloadConfig:
		err := sm.LoadSerialConfig()
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Success = true
			response.Data = "配置重新加载成功"
		}

	case SerialCmdClearBuffers:
		err := sm.serialPort.ClearBuffers()
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Success = true
			response.Data = "缓冲区清空成功"
		}

	default:
		response.Error = "未知的命令类型"
	}

	if cmd.Response != nil {
		cmd.Response <- &response
	}
}

// 串口数据分发器
func (sm *SrvMgr) serialDataDispatcher() {
	for {
		select {
		case msg := <-sm.serialDataChan:
			sm.dispatchSerialData(msg)
		case <-sm.quitch:
			return
		}
	}
}

// 分发串口数据到所有注册的处理器
func (sm *SrvMgr) dispatchSerialData(msg *SerialDataMessage) {
	sm.serialMu.RLock()
	handlers := make([]func(*SerialDataMessage), 0, len(sm.serialHandlers))
	for _, handler := range sm.serialHandlers {
		handlers = append(handlers, handler)
	}
	sm.serialMu.RUnlock()

	// 并发执行所有处理器
	for _, handler := range handlers {
		go handler(msg)
	}

}

// ========== 对外提供的串口操作方法 ==========

// OpenSerial 打开串口
func (sm *SrvMgr) OpenSerial() error {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdOpen,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// CloseSerial 关闭串口
func (sm *SrvMgr) CloseSerial() error {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdClose,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// RestartSerial 重启串口
func (sm *SrvMgr) RestartSerial() error {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdRestart,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// WriteSerial 写入原始数据
func (sm *SrvMgr) WriteSerial(data []byte) (int, error) {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdWrite,
		Data:     data,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return 0, fmt.Errorf("%s", result.Error)
	}

	if writeResult, ok := result.Data.(map[string]interface{}); ok {
		if n, ok := writeResult["bytesWritten"].(int); ok {
			return n, nil
		}
	}
	return 0, nil
}

// WriteSerialString 写入字符串
func (sm *SrvMgr) WriteSerialString(str string) (int, error) {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdWriteString,
		Data:     str,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return 0, fmt.Errorf("%s", result.Error)
	}

	if writeResult, ok := result.Data.(map[string]interface{}); ok {
		if n, ok := writeResult["bytesWritten"].(int); ok {
			return n, nil
		}
	}
	return 0, nil
}

// WriteSerialHex 写入十六进制字符串
func (sm *SrvMgr) WriteSerialHex(hexStr string) (int, error) {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdWriteHex,
		Data:     hexStr,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return 0, fmt.Errorf("%s", result.Error)
	}

	if writeResult, ok := result.Data.(map[string]interface{}); ok {
		if n, ok := writeResult["bytesWritten"].(int); ok {
			return n, nil
		}
	}
	return 0, nil
}

// SendModbusCommand 发送Modbus命令
func (sm *SrvMgr) SendModbusCommand(function []byte, address []byte, data []byte) error {
	// 构建Modbus命令
	cmd := append(function, address...)
	cmd = append(cmd, data...)

	// 计算CRC
	crc := calculateModbusCRC16(cmd)

	// 添加CRC（低字节在前）
	cmdWithCRC := append(cmd, byte(crc&0xFF), byte((crc>>8)&0xFF))

	// 发送
	_, err := sm.WriteSerial(cmdWithCRC)
	return err
}

// 计算Modbus CRC16
func calculateModbusCRC16(data []byte) uint16 {
	var crc uint16 = 0xFFFF

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 == 1 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc = crc >> 1
			}
		}
	}

	return crc
}

// GetSerialStatus 获取串口状态
func (sm *SrvMgr) GetSerialStatus() (map[string]interface{}, error) {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdGetStatus,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	if status, ok := result.Data.(map[string]interface{}); ok {
		return status, nil
	}
	return nil, fmt.Errorf("获取状态失败")
}

// UpdateSerialConfig 更新串口配置
func (sm *SrvMgr) UpdateSerialConfig(config *SerialConfig) error {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdUpdateConfig,
		Data:     config,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// ReloadSerialConfig 重新加载配置文件
func (sm *SrvMgr) ReloadSerialConfig() error {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdReloadConfig,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// ClearSerialBuffers 清空串口缓冲区
func (sm *SrvMgr) ClearSerialBuffers() error {
	resp := make(chan *SerialCmdResponse)

	sm.serialCmdChan <- &SerialCommand{
		CmdType:  SerialCmdClearBuffers,
		Response: resp,
	}

	result := <-resp
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// RegisterSerialHandler 注册串口数据处理器
func (sm *SrvMgr) RegisterSerialHandler(name string, handler func(*SerialDataMessage)) {
	sm.serialMu.Lock()
	defer sm.serialMu.Unlock()
	sm.serialHandlers[name] = handler
}

// UnregisterSerialHandler 注销串口数据处理器
func (sm *SrvMgr) UnregisterSerialHandler(name string) {
	sm.serialMu.Lock()
	defer sm.serialMu.Unlock()
	delete(sm.serialHandlers, name)
}

// IsSerialOpen 检查串口是否打开
func (sm *SrvMgr) IsSerialOpen() bool {
	sm.serialMu.RLock()
	defer sm.serialMu.RUnlock()
	return sm.serialStatus.IsOpen
}

// GetSerialPortNames 获取可用串口列表
func (sm *SrvMgr) GetSerialPortNames() ([]string, error) {
	return serial.GetPortsList()
}

// 处理Modbus数据包
func (ph *PacketHandler) processModbusPackets() {
	for {
		if len(ph.buffer) < 4 { // 至少需要地址(1) + 功能码(1) + CRC(2)
			break
		}

		// 获取地址和功能码
		address := ph.buffer[0]
		functionCode := ph.buffer[1]

		// 检查是否是我们关心的地址（0x01）
		if address != 0x01 {
			// 不是目标地址，移除一个字节继续查找
			ph.buffer = ph.buffer[1:]
			continue
		}

		// 根据功能码确定数据包长度
		packetLen := ph.getModbusPacketLength(functionCode, ph.buffer)

		if packetLen == 0 {
			// 无法确定长度，移除一个字节尝试重新同步
			ph.buffer = ph.buffer[1:]
			continue
		}

		// 检查缓冲区是否包含完整的数据包
		if len(ph.buffer) < packetLen {
			break // 数据不足，等待更多数据
		}

		// 提取完整的Modbus数据包
		packet := make([]byte, packetLen)
		copy(packet, ph.buffer[:packetLen])

		// 验证CRC校验
		if ph.verifyModbusCRC(packet) {
			fmt.Printf("【Modbus CRC校验通过,成功提取】 %X\n", packet)
			// CRC验证通过，发送数据包
			ph.packets <- packet
			// 移除已处理的数据
			ph.buffer = ph.buffer[packetLen:]
		} else {
			// CRC验证失败，可能是错误的数据包或同步问题
			// 移除一个字节尝试重新同步
			ph.buffer = ph.buffer[1:]
		}
	}
}

// 根据功能码获取Modbus数据包长度
func (ph *PacketHandler) getModbusPacketLength(functionCode byte, buffer []byte) int {
	switch functionCode {
	case 0x01:
		// 读命令的响应帧：地址(1) + 功能码(1) + 数据长度(1) + N字节数据 + CRC(2)
		if len(buffer) < 3 {
			return 0 // 数据不足，无法获取数据长度字节
		}
		dataLen := int(buffer[2]) // 第3个字节是数据长度
		return 3 + dataLen + 2    // 地址+功能码+长度字节 + 数据 + CRC

	case 0x03, 0x05:
		// 读保持寄存器请求（03）或写单个线圈的请求/响应（05）：固定8字节
		return 8

	case 0x02:

		return 6

	// case 0x0F, 0x10:
	// 	// 写多个线圈/寄存器的请求：需要第6字节是数据长度
	// 	if len(buffer) < 6 {
	// 		return 0
	// 	}
	// 	dataLen := int(buffer[5]) // 第6字节是数据长度
	// 	return 6 + dataLen + 2    // 地址+功能码+4字节参数 + 数据 + CRC

	default:
		// 未知功能码，返回0表示无法确定长度
		return 0
	}
}

// 验证Modbus CRC16校验
func (ph *PacketHandler) verifyModbusCRC(packet []byte) bool {
	if len(packet) < 2 {
		return false
	}

	// 计算CRC（不包括最后2个字节）
	crc := calculateModbusCRC16(packet[:len(packet)-2])

	// 获取包中的CRC（低位在前，高位在后）
	packetCRC := uint16(packet[len(packet)-2]) | (uint16(packet[len(packet)-1]) << 8)

	return crc == packetCRC
}

// 可选的：保留行模式处理，但专门处理Modbus
func (ph *PacketHandler) processPackets() {
	for {
		if len(ph.buffer) == 0 {
			break
		}

		// 先尝试作为Modbus数据包处理
		modbusProcessed := ph.tryProcessAsModbus()
		if modbusProcessed {
			continue
		}

		// 如果不是Modbus格式，回退到行模式处理
		ph.processLinePackets()
	}
}

// 尝试作为Modbus数据包处理
func (ph *PacketHandler) tryProcessAsModbus() bool {
	originalLen := len(ph.buffer)

	// 处理Modbus数据包
	ph.processModbusPackets()

	// 如果缓冲区长度减少了，说明成功处理了Modbus数据包
	return len(ph.buffer) < originalLen
}
