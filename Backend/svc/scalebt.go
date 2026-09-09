// BluetoothManager.go
package svc

import (
	"bytes"
	"context"

	"encoding/hex"
	"os"
	"os/exec"
	"runtime"

	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"tmaxsrv/comm"
	"tmaxsrv/log"
	"tmaxsrv/picker"
	"tmaxsrv/util"

	"tinygo.org/x/bluetooth"
)

// 常量定义
const (
	BLUETOOTH_SEND_CH_SIZE = 100
	BLUETOOTH_RECV_CH_SIZE = 100
	BLUETOOTH_QUEUE_SIZE   = 1024
	BLUETOOTH_TMP_BUF_SIZE = 512
	NOTIFY_CH_SIZE         = 100
	// MIN_PACK_SIZE          = 4 // 假设最小包大小
)

const (
	ServiceUUID = "A002"
	ReadUUID    = "C305"
	WriteUUID   = "C304"
)

// BtInfoList 蓝牙设备信息
type BtInfoList struct {
	Mac  string `json:"mac"`
	Name string `json:"name"`
	RSSI int    `json:"rssi"`
}

// 全局蓝牙管理器
type BluetoothManager struct {
	adapter     *bluetooth.Adapter
	isEnabled   bool
	enabledOnce sync.Once
	mu          sync.RWMutex

	// 管理所有连接的设备
	connectedDevices map[string]*BluetoothDevice
	devicesMu        sync.RWMutex

	// 扫描控制
	scanning   bool
	scanMu     sync.Mutex
	scanCancel context.CancelFunc
}

// 蓝牙设备包装器
type BluetoothDevice struct {
	ID          string
	Address     string
	Device      *bluetooth.Device
	Connection  *TBluetooth
	ConnectedAt time.Time
	LastSeen    time.Time
}

var (
	bluetoothManager *BluetoothManager
	managerOnce      sync.Once
)

// 获取蓝牙管理器单例
func GetBluetoothManager() *BluetoothManager {
	managerOnce.Do(func() {
		bluetoothManager = &BluetoothManager{
			connectedDevices: make(map[string]*BluetoothDevice),
		}
	})
	return bluetoothManager
}

// 安全地启用适配器
func (bm *BluetoothManager) EnableAdapter() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.isEnabled {
		return nil
	}

	if bm.adapter == nil {
		bm.adapter = bluetooth.DefaultAdapter
		if bm.adapter == nil {
			return fmt.Errorf("no bluetooth adapter available")
		}
	}

	// 尝试启用
	err := bm.adapter.Enable()
	if err != nil {
		// 检查是否已经启用
		if isAlreadyEnabled(err) {
			bm.isEnabled = true
			return nil
		}
		return fmt.Errorf("enable adapter failed: %v", err)
	}

	bm.isEnabled = true
	log.Log.Info("蓝牙适配器初始化成功")
	return nil
}

// 检查错误是否是"already enabled"
func isAlreadyEnabled(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "already") ||
		strings.Contains(errStr, "already enabled") ||
		strings.Contains(errStr, "already in use")
}

// 扫描设备（统一管理）
func (bm *BluetoothManager) ScanDevices(timeout time.Duration) ([]bluetooth.ScanResult, error) {
	bm.scanMu.Lock()
	if bm.scanning {
		bm.scanMu.Unlock()
		return nil, fmt.Errorf("扫描正在进行中")
	}
	bm.scanning = true
	bm.scanMu.Unlock()

	defer func() {
		bm.scanMu.Lock()
		bm.scanning = false
		bm.scanMu.Unlock()
	}()

	// 确保适配器已启用
	if err := bm.EnableAdapter(); err != nil {
		return nil, err
	}

	// 调用实际扫描逻辑
	return bm.doScan(timeout)
}

// 实际扫描逻辑
func (bm *BluetoothManager) doScan(timeout time.Duration) ([]bluetooth.ScanResult, error) {
	devices := make(map[string]bluetooth.ScanResult)
	var results []bluetooth.ScanResult

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 保存取消函数
	bm.scanMu.Lock()
	if bm.scanCancel != nil {
		bm.scanCancel()
	}
	bm.scanCancel = cancel
	bm.scanMu.Unlock()

	log.Log.Info("正在扫描蓝牙设备...")
	start := time.Now()

	err := bm.adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		select {
		case <-ctx.Done():
			adapter.StopScan()
			return
		default:
		}

		address := device.Address.String()
		name := device.LocalName()

		if existing, exists := devices[address]; exists {
			if existing.LocalName() == "" && name != "" {
				devices[address] = device
				for i, d := range results {
					if d.Address.String() == address {
						results[i] = device
						break
					}
				}
			}
		} else {
			devices[address] = device
			results = append(results, device)

			if name == "" {
				name = "未知设备"
			}

			elapsed := time.Since(start).Seconds()
			log.Log.Debugf("[%.1fs] 发现设备: %s, 名称: %s", elapsed, address, name)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("扫描失败: %v", err)
	}

	bm.adapter.StopScan()
	log.Log.Infof("扫描完成，发现 %d 个设备", len(results))
	return results, nil
}

// 停止当前扫描
func (bm *BluetoothManager) StopScan() {
	bm.scanMu.Lock()
	defer bm.scanMu.Unlock()

	if bm.scanning && bm.scanCancel != nil {
		bm.scanCancel()
		bm.adapter.StopScan()
		bm.scanning = false
		log.Log.Info("扫描已停止")
	}
}

// 连接设备
func (bm *BluetoothManager) ConnectDevice(address string, pickerFn picker.PickerFunc, isDefault bool) (*TBluetooth, error) {
	bm.devicesMu.Lock()
	defer bm.devicesMu.Unlock()

	// 检查是否已连接
	if dev, exists := bm.connectedDevices[address]; exists && dev.Connection != nil {
		if dev.Connection.GetConnectionStatus() {
			return dev.Connection, nil // 已连接，返回现有连接
		} else {
			// 已断开，清理
			delete(bm.connectedDevices, address)
		}
	}

	// 检查连接数限制
	if len(bm.connectedDevices) >= 3 { // 限制最多3个连接
		return nil, fmt.Errorf("已达到最大连接数限制(%d)", 3)
	}

	// 确保适配器已启用
	if err := bm.EnableAdapter(); err != nil {
		return nil, err
	}

	// 创建新的蓝牙连接
	bt, err := NewBluetoothConnection(address, pickerFn, isDefault, bm.adapter)
	if err != nil {
		return nil, err
	}

	// 保存到管理器
	bm.connectedDevices[address] = &BluetoothDevice{
		ID:          address,
		Address:     address,
		Connection:  bt,
		ConnectedAt: time.Now(),
	}

	log.Log.Infof("设备 %s 已添加到管理器，当前连接数: %d", address, len(bm.connectedDevices))
	return bt, nil
}

// 断开连接
func (bm *BluetoothManager) DisconnectDevice(address string) error {
	bm.devicesMu.Lock()
	defer bm.devicesMu.Unlock()

	dev, exists := bm.connectedDevices[address]
	if !exists {
		return nil // 设备不存在
	}

	log.Log.Infof("正在断开设备 %s ...", address)

	// 断开连接
	if dev.Connection != nil {
		dev.Connection.Close()
	}

	// 从管理器中移除
	delete(bm.connectedDevices, address)

	log.Log.Infof("设备 %s 已断开，剩余连接数: %d", address, len(bm.connectedDevices))

	time.Sleep(100 * time.Millisecond)
	// go func() {
	// 	controller := NewBluetoothController()
	// 	fmt.Println("\n正在重置蓝牙...")

	// 	if err := controller.DisableBluetooth(); err != nil {
	// 		fmt.Printf("禁用失败: %v\n", err)
	// 	}
	// 	if err := controller.EnableBluetooth(); err != nil {
	// 		fmt.Printf("启用蓝牙失败: %v\n", err)
	// 	}
	// }()
	return nil
}

// 获取所有已连接设备
func (bm *BluetoothManager) GetConnectedDevices() []*BluetoothDevice {
	bm.devicesMu.RLock()
	defer bm.devicesMu.RUnlock()

	devices := make([]*BluetoothDevice, 0, len(bm.connectedDevices))
	for _, dev := range bm.connectedDevices {
		devices = append(devices, dev)
	}
	return devices
}

// TBluetooth 修改为不管理适配器
type TBluetooth struct {
	deviceAddress string
	serviceUUID   string
	charReadUUID  string
	charWriteUUID string

	device           *bluetooth.Device
	notificationChan chan []byte
	quitChan         chan struct{}
	disconnectChan   chan struct{}
	wg               sync.WaitGroup
	isAlive          atomic.Bool
	isDisconnecting  atomic.Bool
	mu               sync.RWMutex

	// 使用共享适配器的引用
	adapterRef *bluetooth.Adapter

	// 其他字段保持不变...
	sendCh     chan []byte
	recvCh     chan comm.Packet
	queue      *util.CircularBuffer
	pickerFn   picker.PickerFunc
	tmpbuf     []byte
	toQuit     bool
	maxPackLen int
	isDefault  bool

	readChar  *bluetooth.DeviceCharacteristic
	writeChar *bluetooth.DeviceCharacteristic
}

// 新的构造函数
func NewBluetoothConnection(address string, pickerFn picker.PickerFunc, isDefault bool, adapter *bluetooth.Adapter) (*TBluetooth, error) {
	if pickerFn == nil {
		return nil, fmt.Errorf("NewBluetoothConnection(): should provide a picker function")
	}

	if adapter == nil {
		return nil, fmt.Errorf("adapter is nil")
	}

	charWriteUUID := WriteUUID
	if charWriteUUID == "" {
		charWriteUUID = ReadUUID
	}

	bt := &TBluetooth{
		deviceAddress:    address,
		serviceUUID:      ServiceUUID,
		charReadUUID:     ReadUUID,
		charWriteUUID:    charWriteUUID,
		adapterRef:       adapter, // 使用共享适配器
		sendCh:           make(chan []byte, BLUETOOTH_SEND_CH_SIZE),
		recvCh:           make(chan comm.Packet, BLUETOOTH_RECV_CH_SIZE),
		queue:            util.NewCircularBuffer(BLUETOOTH_QUEUE_SIZE),
		pickerFn:         pickerFn,
		tmpbuf:           make([]byte, BLUETOOTH_TMP_BUF_SIZE),
		toQuit:           false,
		isDefault:        isDefault,
		notificationChan: make(chan []byte, NOTIFY_CH_SIZE),
		quitChan:         make(chan struct{}),
		disconnectChan:   make(chan struct{}),
	}

	// 启动连接循环
	go bt.connectLoop()

	return bt, nil
}

// connectLoop 连接循环，保持蓝牙连接
func (bt *TBluetooth) connectLoop() {
	bt.wg.Add(1)
	defer bt.wg.Done()

	for {
		if bt.toQuit {
			break
		}

		if !bt.isAlive.Load() {
			if err := bt.connectDevice(); err != nil {
				log.Log.Errorf("连接失败，5秒后重试: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
		}

		time.Sleep(2 * time.Second) // 定期检查连接状态
	}
}

func stringToMACAddress(macStr string) (bluetooth.MACAddress, error) {

	// 1. 先将字符串转换为 MACAddress
	mac, err := bluetooth.ParseMAC(macStr)
	if err != nil {
		return bluetooth.MACAddress{},
			err
	}

	// 2. 创建 Address 结构体
	addr := bluetooth.MACAddress{
		MAC: mac, // MAC地址
		// 是否为随机地址（false表示公共地址）
	}

	return addr, nil
}

// connectDevice 连接蓝牙设备（带重试机制）
func (bt *TBluetooth) connectDevice() error {
	log.Log.Info("尝试连接蓝牙设备: " + bt.deviceAddress)

	// 尝试最大次数
	maxRetries := 2

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Log.Infof("连接尝试 %d/%d", attempt, maxRetries)

		var device bluetooth.Device
		var err error

		if attempt == 1 {
			// 第一次尝试：直接连接

			btAddr, err := bluetooth.ParseAddress(bt.deviceAddress)
			if err != nil {
				return fmt.Errorf("MAC地址格式错误: %v", err)
			}

			// 尝试连接
			device, err = bt.adapterRef.Connect(btAddr, bluetooth.ConnectionParams{
				ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
			})

			if err != nil {
				// 重要：返回错误，不要继续执行
				return fmt.Errorf("未找到设备 %s: %v", bt.deviceAddress, err)
			}

		} else {
			// 后续尝试：扫描后连接
			log.Log.Warnf("第%d次尝试：扫描后连接", attempt)

			scanMutex.Lock()
			devices, scanErr := bt.scanDevicesWithAdapter(bt.adapterRef, 20*time.Second)
			scanMutex.Unlock()

			if scanErr != nil {
				log.Log.Errorf("扫描失败: %v", scanErr)
				continue
			}

			// 查找目标设备
			var targetDevice bluetooth.ScanResult
			found := false
			for _, dev := range devices {
				if dev.Address.String() == bt.deviceAddress {
					targetDevice = dev
					found = true
					break
				}
			}
			if !found {
				log.Log.Warn("扫描后未找到目标设备")
				continue
			}

			device, err = bt.adapterRef.Connect(targetDevice.Address, bluetooth.ConnectionParams{
				ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
			})
		}

		if err == nil {
			// 连接成功
			return bt.setupDevice(device)
		}

		log.Log.Warnf("连接尝试 %d 失败: %v", attempt, err)

		// 如果不是最后一次尝试，等待后重试
		if attempt < maxRetries {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("连接设备失败，已尝试 %d 次", maxRetries)
}

// setupDevice 设置设备的服务和特征（从原函数提取的公共部分）
func (bt *TBluetooth) setupDevice(device bluetooth.Device) error {
	bt.mu.Lock()
	bt.device = &device
	bt.mu.Unlock()

	// 发现服务
	serviceUUID, err := bluetooth.ParseUUID(bt.serviceUUID)
	if err != nil {
		return fmt.Errorf("无效的服务UUID: %v", err)
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil || len(services) == 0 {
		return fmt.Errorf("未找到服务 %s: %v", bt.serviceUUID, err)
	}
	targetService := services[0]
	log.Log.Infof("找到服务: %s", bt.serviceUUID)

	// 发现读取特征并订阅通知
	charReadUUID, err := bluetooth.ParseUUID(bt.charReadUUID)
	if err != nil {
		return fmt.Errorf("无效的读取特征UUID: %v", err)
	}

	readChars, err := targetService.DiscoverCharacteristics([]bluetooth.UUID{charReadUUID})
	if err != nil || len(readChars) == 0 {
		return fmt.Errorf("未找到读取特征 %s: %v", bt.charReadUUID, err)
	}
	bt.readChar = &readChars[0]

	// 订阅通知
	bt.wg.Add(1)
	go func() {
		defer bt.wg.Done()

		err := bt.readChar.EnableNotifications(func(data []byte) {
			if bt.isDisconnecting.Load() {
				return
			}
			select {
			case bt.notificationChan <- data:
				log.Log.Debugf("收到蓝牙数据: %s", hex.EncodeToString(data))
			case <-bt.quitChan:
				return
			}
		})
		if err != nil {
			log.Log.Errorf("订阅特征失败: %v", err)
		} else {
			log.Log.Info("订阅成功，等待接收数据...")
		}
	}()

	// 发现写入特征
	charWriteUUID, err := bluetooth.ParseUUID(bt.charWriteUUID)
	if err != nil {
		return fmt.Errorf("无效的写入特征UUID: %v", err)
	}

	writeChars, err := targetService.DiscoverCharacteristics([]bluetooth.UUID{charWriteUUID})
	if err != nil || len(writeChars) == 0 {
		return fmt.Errorf("未找到写入特征 %s: %v", bt.charWriteUUID, err)
	}
	bt.writeChar = &writeChars[0]

	bt.isAlive.Store(true)
	log.Log.Info("蓝牙连接成功")

	// 启动读写协程
	go bt.read()
	go bt.write()

	return nil
}

// scanDevicesWithAdapter 使用指定适配器扫描设备
func (bt *TBluetooth) scanDevicesWithAdapter(adapter *bluetooth.Adapter, timeout time.Duration) ([]bluetooth.ScanResult, error) {
	devices := make(map[string]bluetooth.ScanResult)
	var results []bluetooth.ScanResult

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Log.Info("正在扫描蓝牙设备...")
	start := time.Now()

	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		select {
		case <-ctx.Done():
			adapter.StopScan()
			return
		default:
		}

		address := device.Address.String()
		name := device.LocalName()

		if existing, exists := devices[address]; exists {
			if existing.LocalName() == "" && name != "" {
				devices[address] = device
				for i, d := range results {
					if d.Address.String() == address {
						results[i] = device
						break
					}
				}
			}
		} else {
			devices[address] = device
			results = append(results, device)

			if name == "" {
				name = "未知设备"
			}

			elapsed := time.Since(start).Seconds()
			log.Log.Debugf("[%.1fs] 发现设备: %s, 名称: %s", elapsed, address, name)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("扫描失败: %v", err)
	}

	adapter.StopScan()
	log.Log.Infof("扫描完成，发现 %d 个设备", len(results))
	return results, nil
}

// Close 关闭蓝牙连接
func (bt *TBluetooth) Close() error {
	bt.toQuit = true

	// 标记为断开中
	if bt.isDisconnecting.Swap(true) {
		return nil
	}

	log.Log.Info("开始断开蓝牙连接...")

	// 1. 先取消通知订阅（关键步骤）
	bt.disableNotifications()

	// 2. 关闭通知通道
	close(bt.quitChan)

	// 3. 等待通知完全停止
	time.Sleep(200 * time.Millisecond)

	// 4. 断开设备连接
	bt.mu.RLock()
	device := bt.device
	bt.mu.RUnlock()

	if device != nil && bt.isAlive.Load() {
		log.Log.Info("执行设备断开...")
		err := device.Disconnect()
		if err != nil {
			log.Log.Errorf("断开设备连接失败: %v", err)
		} else {
			log.Log.Info("设备断开命令已发送")
		}
	}

	// 6. 等待断开确认
	select {
	case <-bt.disconnectChan:
		log.Log.Info("收到断开确认")
	case <-time.After(3 * time.Second):
		log.Log.Warn("断开确认超时")
	}

	// 7. 关闭通道
	close(bt.sendCh)
	close(bt.recvCh)
	close(bt.notificationChan)

	// 8. 清空设备引用
	bt.mu.Lock()
	bt.device = nil
	bt.readChar = nil
	bt.writeChar = nil
	bt.mu.Unlock()

	bt.isAlive.Store(false)
	log.Log.Info("蓝牙连接已关闭")

	return nil
}

// disableNotifications 禁用通知（ESP32-C3 TinyGo专用）
func (bt *TBluetooth) disableNotifications() {
	bt.mu.RLock()
	char := bt.readChar
	bt.mu.RUnlock()

	if char == nil {
		return
	}

	log.Log.Info("正在禁用蓝牙通知...")

	bt.stopNotificationCallback()

}

// stopNotificationCallback 停止通知回调
func (bt *TBluetooth) stopNotificationCallback() {
	// TinyGo的蓝牙库中，停止通知的方法取决于具体实现
	// 通常有几种方式：

	// 1. 如果特征值有DisableNotifications方法
	if char, ok := interface{}(bt.readChar).(interface {
		DisableNotifications() error
	}); ok {
		err := char.DisableNotifications()
		if err != nil {
			log.Log.Warnf("DisableNotifications失败: %v", err)
		} else {
			log.Log.Info("成功禁用通知")
		}
		return
	}

	// 2. 重新订阅一个空回调函数
	bt.resubscribeWithNilCallback()
}

// resubscribeWithNilCallback 重新订阅空回调函数
func (bt *TBluetooth) resubscribeWithNilCallback() {
	bt.mu.RLock()
	char := bt.readChar
	bt.mu.RUnlock()

	if char == nil {
		return
	}

	// 尝试重新启用通知，但使用空回调
	// 这可能会覆盖原来的回调函数
	err := char.EnableNotifications(func(data []byte) {
		// 空回调，什么也不做
		// 检查是否需要退出
		select {
		case <-bt.quitChan:
			return
		default:
		}
	})

	if err != nil {
		log.Log.Warnf("重新订阅空回调失败: %v", err)
	} else {
		log.Log.Info("成功重新订阅空回调")
	}
}

// Write 发送数据到蓝牙设备
func (bt *TBluetooth) Write(data []byte) error {
	if len(bt.sendCh) >= BLUETOOTH_SEND_CH_SIZE {
		log.Log.Error("sendCh is full")
		return util.ErrFull
	}

	select {
	case bt.sendCh <- data:
		log.Log.Debugf("数据已放入发送队列: %s", hex.EncodeToString(data))
		return nil
	case <-bt.quitChan:
		return fmt.Errorf("蓝牙连接已关闭")
	default:
		return util.ErrFull
	}
}

// read 从蓝牙读取数据（处理通知通道）
func (bt *TBluetooth) read() {
	bt.wg.Add(1)
	defer bt.wg.Done()

	packCnt := 0

	for {
		select {
		case data := <-bt.notificationChan:
			if data != nil && len(data) > 0 {
				bt.processReceivedData(data)
			}
			// 处理队列中的数据包
			bt.processQueue(&packCnt)

		case <-bt.quitChan:
			return

		case <-time.After(10 * time.Millisecond):
			// 定期处理队列，即使没有新数据
			bt.processQueue(&packCnt)
		}
	}
}

// processReceivedData 处理接收到的数据
func (bt *TBluetooth) processReceivedData(data []byte) {
	if bt.queue.IsFull() {
		bt.queue.DequeueN(bt.queue.Capacity)
	}

	log.Log.Debugf("蓝牙接收数据 HEX: %x", data)
	log.Log.Debugf("蓝牙接收数据 STR: %s", string(data))

	if err := bt.queue.EnqueueN(data, len(data)); err != nil {
		log.Log.Errorf("数据入队失败: %v", err)
		bt.queue.Reset()
	}
}

// processQueue 处理队列中的数据包
func (bt *TBluetooth) processQueue(packCnt *int) {
	hasPack := true
	for hasPack {
		if bt.queue.GetDataLen() > MIN_PACK_SIZE {
			data := bt.queue.PeekAll()
			_, packLen, removeLen, pack := bt.pickerFn(data, bt.queue.GetDataLen())

			if packLen > 0 {
				if len(bt.recvCh) >= BLUETOOTH_RECV_CH_SIZE {
					log.Log.Errorf("recvCh full, size: %v", len(bt.recvCh))
				} else {
					bt.recvCh <- pack
				}
				(*packCnt)++
			} else {
				hasPack = false
			}

			if removeLen > 0 {
				bt.queue.DequeueN(int(removeLen))
			}
		} else {
			hasPack = false
		}
	}
}

// write 向蓝牙写入数据
func (bt *TBluetooth) write() {
	bt.wg.Add(1)
	defer bt.wg.Done()

	for {
		select {
		case message := <-bt.sendCh:
			if bt.isAlive.Load() && bt.writeChar != nil {
				log.Log.Debugf("发送数据到蓝牙: %s", hex.EncodeToString(message))

				// 写入数据
				n, err := bt.writeChar.Write(message)
				if err != nil {
					log.Log.Errorf("写入蓝牙失败: %v", err)
					bt.isAlive.Store(false)
				} else if n != len(message) {
					log.Log.Errorf("数据写入不完整，期望 %d 字节，实际 %d 字节", len(message), n)
				} else {
					log.Log.Debugf("数据写入成功: %d 字节", n)
				}

				// 避免发送过快
				time.Sleep(10 * time.Millisecond)
			}

		case <-bt.quitChan:
			return
		}
	}
}

// GetConnectionStatus 获取连接状态
func (bt *TBluetooth) GetConnectionStatus() bool {
	return bt.isAlive.Load()
}

// ChangePickFunc 更改数据包解析函数
func (bt *TBluetooth) ChangePickFunc(pickerFn picker.PickerFunc) {
	bt.pickerFn = pickerFn
}

// GetDeviceAddress 获取设备地址
func (bt *TBluetooth) GetDeviceAddress() string {
	return bt.deviceAddress
}

// 对外提供的API

// 初始化蓝牙（应用启动时调用一次）
func InitializeBluetooth() error {
	mgr := GetBluetoothManager()
	return mgr.EnableAdapter()
}

// 获取蓝牙列表
func GetBluetoothList(timeout time.Duration) (string, error) {
	mgr := GetBluetoothManager()
	results, err := mgr.ScanDevices(timeout)
	if err != nil {
		return "", err
	}

	// 转换为 BtInfoList
	list := make([]BtInfoList, len(results))
	for i, device := range results {
		list[i] = BtInfoList{
			Mac:  device.Address.String(),
			Name: device.LocalName(),
			RSSI: int(device.RSSI),
		}
	}

	jsonBytes, err := json.Marshal(list)
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %w", err)
	}

	return string(jsonBytes), nil
}

// 连接蓝牙设备
func ConnectBluetoothDevice(address string, pickerFn picker.PickerFunc) (*TBluetooth, error) {
	mgr := GetBluetoothManager()
	return mgr.ConnectDevice(address, pickerFn, false)
}

// 断开蓝牙设备
func DisconnectBluetoothDevice(address string) error {
	mgr := GetBluetoothManager()
	return mgr.DisconnectDevice(address)
}

// 停止蓝牙扫描
func StopBluetoothScan() {
	mgr := GetBluetoothManager()
	mgr.StopScan()
}

// 检查是否正在扫描
func IsBluetoothScanning() bool {
	mgr := GetBluetoothManager()
	mgr.scanMu.Lock()
	defer mgr.scanMu.Unlock()
	return mgr.scanning
}

// 获取已连接设备列表
func GetConnectedDevices() []string {
	mgr := GetBluetoothManager()
	devices := mgr.GetConnectedDevices()

	addresses := make([]string, len(devices))
	for i, dev := range devices {
		addresses[i] = dev.Address
	}
	return addresses
}

/////////////////////////////////////////////
// 在您的服务中使用蓝牙管理器

// BluetoothController Windows蓝牙控制器
type BluetoothController struct {
	isAdmin bool
}

// NewBluetoothController 创建蓝牙控制器
func NewBluetoothController() *BluetoothController {
	return &BluetoothController{isAdmin: checkAdmin()}
}

// 检查管理员权限
func checkAdmin() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}

	// 额外检查：尝试写入系统目录
	tempFile := "C:\\Windows\\Temp\\test_admin.tmp"
	err = os.WriteFile(tempFile, []byte("test"), 0644)
	if err == nil {
		os.Remove(tempFile)
		return true
	}
	return false
}

// DisableBluetooth 禁用蓝牙
func (bc *BluetoothController) DisableBluetooth() error {
	fmt.Println("=== 开始禁用蓝牙 ===")

	// 1. 禁用网络适配器
	fmt.Println("1. 禁用蓝牙网络适配器...")
	disableAdapterCmd := `
		$btAdapters = Get-NetAdapter | Where-Object {
			$_.InterfaceDescription -like '*Bluetooth*'
		}

		foreach ($adapter in $btAdapters) {
			try {
				Disable-NetAdapter -Name $adapter.Name -Confirm:$false -ErrorAction SilentlyContinue
				Write-Host "已禁用: $($adapter.Name)"
			} catch {
				Write-Host "禁用失败: $($adapter.Name)"
			}
		}
	`

	runPowerShell(disableAdapterCmd)
	time.Sleep(1 * time.Second)

	// 2. 禁用蓝牙设备
	fmt.Println("2. 禁用蓝牙设备...")
	disableDeviceCmd := `
		$btDevices = Get-PnpDevice | Where-Object {
			$_.Class -eq 'Bluetooth' -or
			$_.FriendlyName -like '*Bluetooth*'
		}

		foreach ($device in $btDevices) {
			try {
				Disable-PnpDevice -InstanceId $device.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
				Write-Host "已禁用设备: $($device.FriendlyName)"
			} catch {
				Write-Host "禁用失败: $($device.FriendlyName)"
			}
		}
	`

	runPowerShell(disableDeviceCmd)
	time.Sleep(1 * time.Second)

	// 3. 停止蓝牙服务
	fmt.Println("3. 停止蓝牙服务...")
	stopServiceCmd := `
		Stop-Service -Name BthServ -Force -ErrorAction SilentlyContinue
		Stop-Service -Name BluetoothUserService -Force -ErrorAction SilentlyContinue

		# 设置为手动启动
		Set-Service -Name BthServ -StartupType Manual -ErrorAction SilentlyContinue
		Set-Service -Name BluetoothUserService -StartupType Manual -ErrorAction SilentlyContinue

		Write-Host "服务已停止"
	`

	runPowerShell(stopServiceCmd)

	fmt.Println("✓ 蓝牙禁用完成")
	return nil
}

// EnableBluetooth 启用蓝牙
func (bc *BluetoothController) EnableBluetooth() error {
	fmt.Println("=== 开始启用蓝牙 ===")

	// 1. 启动蓝牙服务
	fmt.Println("1. 启动蓝牙服务...")
	serviceCmd := `
		# 设置服务为自动启动
		Set-Service -Name BthServ -StartupType Automatic -ErrorAction SilentlyContinue
		Set-Service -Name BluetoothUserService -StartupType Automatic -ErrorAction SilentlyContinue

		# 启动服务
		Start-Service -Name BthServ -ErrorAction SilentlyContinue
		Start-Service -Name BluetoothUserService -ErrorAction SilentlyContinue

		Write-Host "服务启动完成"
	`

	runPowerShell(serviceCmd)
	time.Sleep(2 * time.Second)

	// 2. 启用蓝牙设备
	fmt.Println("2. 启用蓝牙设备...")
	enableDeviceCmd := `
		# 启用所有蓝牙设备
		$btDevices = Get-PnpDevice | Where-Object {
			$_.Class -eq 'Bluetooth' -or
			$_.FriendlyName -like '*Bluetooth*'
		}

		foreach ($device in $btDevices) {
			try {
				if ($device.Status -ne 'OK') {
					Enable-PnpDevice -InstanceId $device.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
					Write-Host "已启用: $($device.FriendlyName)"
				}
			} catch {
				Write-Host "启用失败: $($device.FriendlyName)"
			}
		}

		Write-Host "设备启用完成"
	`

	runPowerShell(enableDeviceCmd)
	time.Sleep(2 * time.Second)

	// 3. 启用网络适配器
	fmt.Println("3. 启用蓝牙网络适配器...")
	enableAdapterCmd := `
		# 启用蓝牙网络适配器
		$btAdapters = Get-NetAdapter | Where-Object {
			$_.InterfaceDescription -like '*Bluetooth*'
		}

		foreach ($adapter in $btAdapters) {
			try {
				if ($adapter.Status -ne 'Up') {
					Enable-NetAdapter -Name $adapter.Name -Confirm:$false -ErrorAction SilentlyContinue
					Write-Host "已启用网络适配器: $($adapter.Name)"
				}
			} catch {
				Write-Host "启用失败: $($adapter.Name)"
			}
		}

		Write-Host "网络适配器启用完成"
	`

	runPowerShell(enableAdapterCmd)
	time.Sleep(3 * time.Second)

	// 4. 检查结果
	fmt.Println("4. 检查启用结果...")
	finalStatus, _ := bc.GetBluetoothStatus()

	// 简单判断是否成功
	success := false
	if strings.Contains(finalStatus["services"], "Running") &&
		(strings.Contains(finalStatus["network_adapters"], "状态: Up") ||
			strings.Contains(finalStatus["device_manager"], "OK")) {
		success = true
	}

	if success {
		fmt.Println("✓ 蓝牙启用成功")
		return nil
	} else {
		fmt.Println("✗ 蓝牙启用失败")
		fmt.Println("\n详细信息:")
		for key, value := range finalStatus {
			fmt.Printf("\n%s:\n%s\n", key, value)
		}
		return fmt.Errorf("蓝牙启用失败")
	}
}

// GetBluetoothStatus 获取蓝牙状态
func (bc *BluetoothController) GetBluetoothStatus() (map[string]string, error) {
	fmt.Println("正在检查蓝牙状态...")

	results := make(map[string]string)

	// 1. 检查设备管理器中的蓝牙设备
	deviceMgrCmd := `
		# 检查所有蓝牙相关设备
		Write-Host "=== 设备管理器检查 ==="

		# 获取所有蓝牙设备
		$btDevices = Get-PnpDevice | Where-Object {
			$_.Class -eq 'Bluetooth' -or
			$_.FriendlyName -like '*Bluetooth*' -or
			$_.FriendlyName -like '*BTH*' -or
			$_.Class -eq 'Net' -and $_.FriendlyName -like '*Bluetooth*'
		} | Select-Object Status, FriendlyName, InstanceId, Class

		if ($btDevices) {
			Write-Host "找到蓝牙设备:"
			foreach ($device in $btDevices) {
				Write-Host "  - $($device.FriendlyName) [$($device.Class)]: $($device.Status)"
			}
		} else {
			Write-Host "未找到蓝牙设备"
		}

		# 检查无线网卡
		$wirelessAdapters = Get-PnpDevice | Where-Object {
			$_.FriendlyName -like '*Wireless*' -or
			$_.FriendlyName -like '*WiFi*' -or
			$_.FriendlyName -like '*WLAN*'
		} | Select-Object Status, FriendlyName

		if ($wirelessAdapters) {
			Write-Host "无线网卡:"
			foreach ($adapter in $wirelessAdapters) {
				Write-Host "  - $($adapter.FriendlyName): $($adapter.Status)"
			}
		}
	`

	deviceMgrResult, _ := runPowerShell(deviceMgrCmd)
	results["device_manager"] = deviceMgrResult

	// 2. 检查网络适配器
	networkCmd := `
		Write-Host "=== 网络适配器检查 ==="
		$btAdapters = Get-NetAdapter | Where-Object {
			$_.InterfaceDescription -like '*Bluetooth*' -or
			$_.Name -like '*Bluetooth*'
		}

		if ($btAdapters) {
			Write-Host "蓝牙网络适配器:"
			foreach ($adapter in $btAdapters) {
				Write-Host "  - $($adapter.Name): $($adapter.InterfaceDescription), 状态: $($adapter.Status)"
			}
		} else {
			Write-Host "未找到蓝牙网络适配器"
		}
	`

	networkResult, _ := runPowerShell(networkCmd)
	results["network_adapters"] = networkResult

	// 3. 检查蓝牙服务
	serviceCmd := `
		Write-Host "=== 蓝牙服务检查 ==="

		$services = @(
			@{Name='BthServ'; DisplayName='蓝牙支持服务'},
			@{Name='BluetoothUserService'; DisplayName='蓝牙用户服务'}
		)

		foreach ($svc in $services) {
			$service = Get-Service -Name $svc.Name -ErrorAction SilentlyContinue
			if ($service) {
				Write-Host "  - $($svc.DisplayName): $($service.Status)"
			} else {
				Write-Host "  - $($svc.DisplayName): 未找到"
			}
		}
	`

	serviceResult, _ := runPowerShell(serviceCmd)
	results["services"] = serviceResult

	return results, nil
}

// 运行PowerShell命令
func runPowerShell(command string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("PowerShell only available on Windows")
	}

	cmd := exec.Command("powershell", "-Command", command)
	util.SetHideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())

	// 如果命令执行出错，但仍然有输出，也返回输出
	if err != nil && stderr.Len() > 0 {
		output += "\nError: " + stderr.String()
	}

	return output, err
}
