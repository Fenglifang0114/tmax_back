package svc

// import (
// 	"context"
// 	"encoding/hex"
// 	"fmt"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"tmaxsrv/comm"
// 	"tmaxsrv/log"
// 	"tmaxsrv/picker"
// 	"tmaxsrv/util"

// 	"tinygo.org/x/bluetooth"

// 	"bytes"

// 	"os"
// 	"os/exec"
// 	"runtime"
// 	"strings"
// 	"syscall"
// )

// // 常量定义
// const (
// 	BLUETOOTH_SEND_CH_SIZE = 100
// 	BLUETOOTH_RECV_CH_SIZE = 100
// 	BLUETOOTH_QUEUE_SIZE   = 1024
// 	BLUETOOTH_TMP_BUF_SIZE = 512
// 	NOTIFY_CH_SIZE         = 100
// )

// var adapter = bluetooth.DefaultAdapter

// // TBluetooth 蓝牙连接管理器
// type TBluetooth struct {
// 	// 蓝牙设备配置
// 	deviceAddress string
// 	serviceUUID   string
// 	charReadUUID  string
// 	charWriteUUID string

// 	// 连接对象
// 	device           *bluetooth.Device
// 	notificationChan chan []byte
// 	quitChan         chan struct{}
// 	disconnectChan   chan struct{}
// 	wg               sync.WaitGroup
// 	isAlive          atomic.Bool
// 	isDisconnecting  atomic.Bool
// 	mu               sync.RWMutex

// 	// 通道和缓冲区
// 	sendCh     chan []byte
// 	recvCh     chan comm.Packet
// 	queue      *util.CircularBuffer
// 	pickerFn   picker.PickerFunc
// 	tmpbuf     []byte
// 	toQuit     bool
// 	maxPackLen int
// 	isDefault  bool

// 	// 特征对象
// 	readChar  *bluetooth.DeviceCharacteristic
// 	writeChar *bluetooth.DeviceCharacteristic
// }

// const (
// 	ServiceUUID = "A002"
// 	ReadUUID    = "C305"
// 	WriteUUID   = "C304"
// )

// // NewBluetooth 创建蓝牙连接
// func NewBluetooth(bInfo BtInfo, pickerFn picker.PickerFunc, isDefault bool) (*TBluetooth, error) {
// 	// func NewBluetooth(bInfo BluetoothInfo, isDefault bool) (*TBluetooth, error) {
// 	if pickerFn == nil {
// 		return nil, fmt.Errorf("NewBluetooth(): should provide a picker function")
// 	}

// 	// 初始化蓝牙适配器
// 	if err := adapter.Enable(); err != nil {
// 		// return nil, fmt.Errorf("enable BLE stack failed: %v", err)
// 	}

// 	// 设置默认值
// 	charWriteUUID := WriteUUID
// 	if charWriteUUID == "" {
// 		charWriteUUID = ReadUUID // 使用相同的UUID
// 	}

// 	bt := &TBluetooth{
// 		deviceAddress:    string(bInfo.Mac),
// 		serviceUUID:      ServiceUUID,
// 		charReadUUID:     ReadUUID,
// 		charWriteUUID:    charWriteUUID,
// 		sendCh:           make(chan []byte, BLUETOOTH_SEND_CH_SIZE),
// 		recvCh:           make(chan comm.Packet, BLUETOOTH_RECV_CH_SIZE),
// 		queue:            util.NewCircularBuffer(BLUETOOTH_QUEUE_SIZE),
// 		pickerFn:         pickerFn,
// 		tmpbuf:           make([]byte, BLUETOOTH_TMP_BUF_SIZE),
// 		toQuit:           false,
// 		isDefault:        isDefault,
// 		notificationChan: make(chan []byte, NOTIFY_CH_SIZE),
// 		quitChan:         make(chan struct{}),
// 		disconnectChan:   make(chan struct{}),
// 	}

// 	// 启动连接循环
// 	go bt.connectLoop()

// 	return bt, nil
// }

// // connectLoop 连接循环，保持蓝牙连接
// func (bt *TBluetooth) connectLoop() {
// 	bt.wg.Add(1)
// 	defer bt.wg.Done()

// 	// 设置连接状态回调
// 	adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
// 		if connected {
// 			log.Log.Info("设备已连接")
// 			bt.isAlive.Store(true)
// 		} else {
// 			log.Log.Info("设备已断开")
// 			bt.isAlive.Store(false)
// 			select {
// 			case bt.disconnectChan <- struct{}{}:
// 			default:
// 			}
// 		}
// 	})

// 	for {
// 		if bt.toQuit {
// 			break
// 		}

// 		if !bt.isAlive.Load() {
// 			if err := bt.connectDevice(); err != nil {
// 				log.Log.Errorf("连接失败，5秒后重试: %v", err)
// 				time.Sleep(5 * time.Second)
// 				continue
// 			}
// 		}

// 		time.Sleep(2 * time.Second) // 定期检查连接状态
// 	}
// }

// func (bt *TBluetooth) connectDevice() error {
// 	maxRetries := 2 // 最大重试次数（包括初始连接）
// 	var lastErr error

// 	for attempt := 0; attempt < maxRetries; attempt++ {
// 		log.Log.Infof("连接尝试 %d/%d: 设备 %s", attempt+1, maxRetries, bt.deviceAddress)

// 		if attempt == 0 {
// 			// 第一次尝试：直接连接
// 			lastErr = bt.tryDirectConnect()
// 		} else {
// 			// 后续尝试：先扫描再连接
// 			lastErr = bt.scanAndConnect()
// 		}

// 		if lastErr == nil {
// 			// 连接成功
// 			log.Log.Infof("蓝牙设备 %s 连接成功 (第%d次尝试)", bt.deviceAddress, attempt+1)
// 			return nil
// 		}

// 		log.Log.Warnf("连接尝试 %d 失败: %v", attempt+1, lastErr)

// 		// 如果不是最后一次尝试，等待一下再重试
// 		if attempt < maxRetries-1 {
// 			waitTime := time.Duration(attempt+1) * 2 * time.Second
// 			log.Log.Infof("等待 %v 后重试...", waitTime)
// 			time.Sleep(waitTime)
// 		}
// 	}

// 	return fmt.Errorf("连接设备失败，已尝试 %d 次，最后错误: %v", maxRetries, lastErr)
// }

// // tryDirectConnect 尝试直接连接已知地址的设备
// func (bt *TBluetooth) tryDirectConnect() error {
// 	log.Log.Info("尝试直接连接设备: " + bt.deviceAddress)

// 	mac, err := stringToMACAddress(bt.deviceAddress)
// 	if err != nil {
// 		return fmt.Errorf("MAC地址格式错误: %v", err)
// 	}

// 	// 尝试连接
// 	device, err := adapter.Connect(bluetooth.Address{MACAddress: mac}, bluetooth.ConnectionParams{
// 		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
// 	})
// 	if err != nil {
// 		return fmt.Errorf("直接连接失败: %v", err)
// 	}

// 	// 连接成功，进行服务发现和特征配置
// 	return bt.setupConnectedDevice(&device)
// }

// // scanAndConnect 扫描设备并连接
// func (bt *TBluetooth) scanAndConnect() error {
// 	log.Log.Info("开始扫描设备...")

// 	// 扫描设备
// 	devices, err := bt.scanDevices(20 * time.Second)
// 	if err != nil {
// 		return fmt.Errorf("扫描设备失败: %v", err)
// 	}

// 	// 查找目标设备
// 	var targetDevice bluetooth.ScanResult
// 	found := false
// 	for _, dev := range devices {
// 		log.Log.Debugf("发现设备: %s - %s", dev.Address.String(), dev.LocalName())
// 		if dev.Address.String() == bt.deviceAddress {
// 			targetDevice = dev
// 			found = true
// 			log.Log.Infof("找到目标设备: %s", bt.deviceAddress)
// 			break
// 		}
// 	}

// 	if !found {
// 		return fmt.Errorf("未找到目标设备: %s (共扫描到 %d 个设备)", bt.deviceAddress, len(devices))
// 	}

// 	// 连接设备
// 	device, err := adapter.Connect(targetDevice.Address, bluetooth.ConnectionParams{
// 		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
// 	})
// 	if err != nil {
// 		return fmt.Errorf("连接扫描到的设备失败: %v", err)
// 	}

// 	// 连接成功，进行服务发现和特征配置
// 	return bt.setupConnectedDevice(&device)
// }

// // setupConnectedDevice 处理连接后的服务发现和特征配置
// func (bt *TBluetooth) setupConnectedDevice(device *bluetooth.Device) error {
// 	bt.mu.Lock()
// 	bt.device = device
// 	bt.mu.Unlock()

// 	// 发现服务
// 	serviceUUID, err := bluetooth.ParseUUID(bt.serviceUUID)
// 	if err != nil {
// 		return fmt.Errorf("无效的服务UUID: %v", err)
// 	}

// 	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
// 	if err != nil {
// 		return fmt.Errorf("发现服务失败: %v", err)
// 	}
// 	if len(services) == 0 {
// 		return fmt.Errorf("未找到服务 %s", bt.serviceUUID)
// 	}
// 	targetService := services[0]
// 	log.Log.Infof("找到服务: %s", bt.serviceUUID)

// 	// 发现读取特征并订阅通知
// 	charReadUUID, err := bluetooth.ParseUUID(bt.charReadUUID)
// 	if err != nil {
// 		return fmt.Errorf("无效的读取特征UUID: %v", err)
// 	}

// 	readChars, err := targetService.DiscoverCharacteristics([]bluetooth.UUID{charReadUUID})
// 	if err != nil {
// 		return fmt.Errorf("发现读取特征失败: %v", err)
// 	}
// 	if len(readChars) == 0 {
// 		return fmt.Errorf("未找到读取特征 %s", bt.charReadUUID)
// 	}
// 	bt.readChar = &readChars[0]

// 	// 订阅通知
// 	bt.wg.Add(1)
// 	go func() {
// 		defer bt.wg.Done()
// 		err := bt.readChar.EnableNotifications(func(data []byte) {
// 			if bt.isDisconnecting.Load() {
// 				return
// 			}
// 			select {
// 			case bt.notificationChan <- data:
// 				log.Log.Debugf("收到蓝牙数据: %s", hex.EncodeToString(data))
// 			case <-bt.quitChan:
// 				return
// 			}
// 		})
// 		if err != nil {
// 			log.Log.Errorf("订阅特征失败: %v", err)
// 		} else {
// 			log.Log.Info("订阅成功，等待接收数据...")
// 		}
// 	}()

// 	// 发现写入特征
// 	charWriteUUID, err := bluetooth.ParseUUID(bt.charWriteUUID)
// 	if err != nil {
// 		return fmt.Errorf("无效的写入特征UUID: %v", err)
// 	}

// 	writeChars, err := targetService.DiscoverCharacteristics([]bluetooth.UUID{charWriteUUID})
// 	if err != nil {
// 		return fmt.Errorf("发现写入特征失败: %v", err)
// 	}
// 	if len(writeChars) == 0 {
// 		return fmt.Errorf("未找到写入特征 %s", bt.charWriteUUID)
// 	}
// 	bt.writeChar = &writeChars[0]

// 	// 启动读写协程
// 	go bt.read()
// 	go bt.write()

// 	return nil
// }

// func stringToMACAddress(macStr string) (bluetooth.MACAddress, error) {

// 	// 1. 先将字符串转换为 MACAddress
// 	mac, err := bluetooth.ParseMAC(macStr)
// 	if err != nil {
// 		return bluetooth.MACAddress{},
// 			err
// 	}

// 	// 2. 创建 Address 结构体
// 	addr := bluetooth.MACAddress{
// 		MAC: mac, // MAC地址
// 		// 是否为随机地址（false表示公共地址）
// 	}

// 	return addr, nil
// }

// // scanDevices 扫描蓝牙设备
// func (bt *TBluetooth) scanDevices(timeout time.Duration) ([]bluetooth.ScanResult, error) {
// 	devices := make(map[string]bluetooth.ScanResult)
// 	var results []bluetooth.ScanResult

// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()

// 	log.Log.Info("正在扫描蓝牙设备...")
// 	start := time.Now()

// 	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
// 		select {
// 		case <-ctx.Done():
// 			adapter.StopScan()
// 			return
// 		default:
// 		}

// 		address := device.Address.String()
// 		name := device.LocalName()

// 		if existing, exists := devices[address]; exists {
// 			if existing.LocalName() == "" && name != "" {
// 				devices[address] = device
// 				for i, d := range results {
// 					if d.Address.String() == address {
// 						results[i] = device
// 						break
// 					}
// 				}
// 			}
// 		} else {
// 			devices[address] = device
// 			results = append(results, device)

// 			if name == "" {
// 				name = "未知设备"
// 			}

// 			elapsed := time.Since(start).Seconds()
// 			log.Log.Debugf("[%.1fs] 发现设备: %s, 名称: %s", elapsed, address, name)
// 		}
// 	})

// 	if err != nil {
// 		return nil, fmt.Errorf("扫描失败: %v", err)
// 	}

// 	adapter.StopScan()
// 	log.Log.Infof("扫描完成，发现 %d 个设备", len(results))
// 	println("scanDevices: ", len(results))
// 	return results, nil
// }

// // Close 关闭蓝牙连接
// func (bt *TBluetooth) Close() error {
// 	bt.toQuit = true

// 	// 标记为断开中
// 	if bt.isDisconnecting.Swap(true) {
// 		return nil // 已经在断开过程中
// 	}

// 	log.Log.Info("开始断开蓝牙连接...")
// 	// 关闭通知通道
// 	close(bt.quitChan)

// 	// 等待一小段时间让通知停止
// 	time.Sleep(100 * time.Millisecond)

// 	// 断开设备连接
// 	bt.mu.RLock()
// 	device := bt.device
// 	bt.mu.RUnlock()

// 	if device != nil && bt.isAlive.Load() {
// 		log.Log.Info("执行设备断开...")
// 		err := device.Disconnect()
// 		if err != nil {
// 			log.Log.Errorf("断开设备连接失败: %v", err)
// 		} else {
// 			log.Log.Info("设备断开命令已发送")
// 		}
// 	}

// 	// 清理适配器状态
// 	adapter.SetConnectHandler(nil)

// 	// 等待所有goroutine完成
// 	bt.wg.Wait()

// 	// 等待断开确认
// 	select {
// 	case <-bt.disconnectChan:
// 		log.Log.Info("收到断开确认")
// 	case <-time.After(10 * time.Second):
// 		log.Log.Warn("断开确认超时")
// 	}

// 	// 关闭通道
// 	if !IsClosed(bt.sendCh) {
// 		close(bt.sendCh)
// 	}

// 	if !IsPacketChClosed(bt.recvCh) {
// 		close(bt.recvCh)
// 	}

// 	close(bt.notificationChan)

// 	// 清空设备引用
// 	bt.mu.Lock()
// 	bt.device = nil
// 	bt.readChar = nil
// 	bt.writeChar = nil
// 	bt.mu.Unlock()

// 	// 停止扫描
// 	adapter.StopScan()

// 	log.Log.Info("蓝牙连接已关闭")
// 	time.Sleep(100 * time.Millisecond)
// 	go func() {
// 		controller := NewBluetoothController()
// 		fmt.Println("\n正在重置蓝牙...")

// 		if err := controller.DisableBluetooth(); err != nil {
// 			fmt.Printf("禁用失败: %v\n", err)
// 		}
// 		if err := controller.EnableBluetooth(); err != nil {
// 			fmt.Printf("启用蓝牙失败: %v\n", err)
// 		}
// 	}()

// 	return nil
// }

// // Write 发送数据到蓝牙设备
// func (bt *TBluetooth) Write(data []byte) error {
// 	if len(bt.sendCh) >= BLUETOOTH_SEND_CH_SIZE {
// 		log.Log.Error("sendCh is full")
// 		return util.ErrFull
// 	}

// 	select {
// 	case bt.sendCh <- data:
// 		log.Log.Debugf("数据已放入发送队列: %s", hex.EncodeToString(data))
// 		return nil
// 	case <-bt.quitChan:
// 		return fmt.Errorf("蓝牙连接已关闭")
// 	default:
// 		return util.ErrFull
// 	}
// }

// // read 从蓝牙读取数据（处理通知通道）
// func (bt *TBluetooth) read() {
// 	bt.wg.Add(1)
// 	defer bt.wg.Done()

// 	packCnt := 0

// 	for {
// 		select {
// 		case data := <-bt.notificationChan:
// 			if data != nil && len(data) > 0 {
// 				bt.processReceivedData(data)
// 			}
// 			// 处理队列中的数据包
// 			bt.processQueue(&packCnt)

// 		case <-bt.quitChan:
// 			return

// 		case <-time.After(10 * time.Millisecond):
// 			// 定期处理队列，即使没有新数据
// 			bt.processQueue(&packCnt)
// 		}
// 	}
// }

// // processReceivedData 处理接收到的数据
// func (bt *TBluetooth) processReceivedData(data []byte) {
// 	if bt.queue.IsFull() {
// 		bt.queue.DequeueN(bt.queue.Capacity)
// 	}

// 	log.Log.Debugf("蓝牙接收数据 HEX: %x", data)
// 	log.Log.Debugf("蓝牙接收数据 STR: %s", string(data))

// 	if err := bt.queue.EnqueueN(data, len(data)); err != nil {
// 		log.Log.Errorf("数据入队失败: %v", err)
// 		bt.queue.Reset()
// 	}
// }

// // processQueue 处理队列中的数据包
// func (bt *TBluetooth) processQueue(packCnt *int) {
// 	hasPack := true
// 	for hasPack {
// 		if bt.queue.GetDataLen() > MIN_PACK_SIZE {
// 			data := bt.queue.PeekAll()
// 			_, packLen, removeLen, pack := bt.pickerFn(data, bt.queue.GetDataLen())

// 			if packLen > 0 {
// 				if len(bt.recvCh) >= BLUETOOTH_RECV_CH_SIZE {
// 					log.Log.Errorf("recvCh full, size: %v", len(bt.recvCh))
// 				} else {
// 					bt.recvCh <- pack
// 				}
// 				(*packCnt)++
// 			} else {
// 				hasPack = false
// 			}

// 			if removeLen > 0 {
// 				bt.queue.DequeueN(int(removeLen))
// 			}
// 		} else {
// 			hasPack = false
// 		}
// 	}
// }

// // write 向蓝牙写入数据
// func (bt *TBluetooth) write() {
// 	bt.wg.Add(1)
// 	defer bt.wg.Done()

// 	for {
// 		select {
// 		case message := <-bt.sendCh:
// 			if bt.isAlive.Load() && bt.writeChar != nil {
// 				log.Log.Debugf("发送数据到蓝牙: %s", hex.EncodeToString(message))

// 				// 写入数据
// 				n, err := bt.writeChar.Write(message)
// 				if err != nil {
// 					log.Log.Errorf("写入蓝牙失败: %v", err)
// 					bt.isAlive.Store(false)
// 				} else if n != len(message) {
// 					log.Log.Errorf("数据写入不完整，期望 %d 字节，实际 %d 字节", len(message), n)
// 				} else {
// 					log.Log.Debugf("数据写入成功: %d 字节", n)
// 				}

// 				// 避免发送过快
// 				time.Sleep(10 * time.Millisecond)
// 			}

// 		case <-bt.quitChan:
// 			return
// 		}
// 	}
// }

// // ChangePickFunc 更改数据包解析函数
// func (bt *TBluetooth) ChangePickFunc(pickerFn picker.PickerFunc) {
// 	bt.pickerFn = pickerFn
// }

// // GetConnectionStatus 获取连接状态
// func (bt *TBluetooth) GetConnectionStatus() bool {
// 	return bt.isAlive.Load()
// }

// // ScanDevices 扫描附近的蓝牙设备（公开方法）
// func (bt *TBluetooth) ScanDevices(timeout time.Duration) ([]bluetooth.ScanResult, error) {
// 	return bt.scanDevices(timeout)
// }

// // // 辅助函数
// // func IsClosed(ch chan []byte) bool {
// // 	select {
// // 	case <-ch:
// // 		return false
// // 	default:
// // 		return false
// // 	}
// // }

// // func IsPacketChClosed(ch chan comm.Packet) bool {
// // 	select {
// // 	case <-ch:
// // 		return false
// // 	default:
// // 		return false
// // 	}
// // }

// ///////////////////////////////////////////////////////////

// // BluetoothController Windows蓝牙控制器
// type BluetoothController struct {
// 	isAdmin bool
// }

// // NewBluetoothController 创建蓝牙控制器
// func NewBluetoothController() *BluetoothController {
// 	return &BluetoothController{isAdmin: checkAdmin()}
// }

// // 检查管理员权限
// func checkAdmin() bool {
// 	if runtime.GOOS != "windows" {
// 		return false
// 	}

// 	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
// 	if err != nil {
// 		return false
// 	}

// 	// 额外检查：尝试写入系统目录
// 	tempFile := "C:\\Windows\\Temp\\test_admin.tmp"
// 	err = os.WriteFile(tempFile, []byte("test"), 0644)
// 	if err == nil {
// 		os.Remove(tempFile)
// 		return true
// 	}
// 	return false
// }

// // 运行PowerShell命令
// func runPowerShell(command string) (string, error) {
// 	if runtime.GOOS != "windows" {
// 		return "", fmt.Errorf("PowerShell only available on Windows")
// 	}

// 	cmd := exec.Command("powershell", "-Command", command)
// 	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

// 	var stdout, stderr bytes.Buffer
// 	cmd.Stdout = &stdout
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	output := strings.TrimSpace(stdout.String())

// 	// 如果命令执行出错，但仍然有输出，也返回输出
// 	if err != nil && stderr.Len() > 0 {
// 		output += "\nError: " + stderr.String()
// 	}

// 	return output, err
// }

// // GetBluetoothStatus 获取蓝牙状态
// func (bc *BluetoothController) GetBluetoothStatus() (map[string]string, error) {
// 	fmt.Println("正在检查蓝牙状态...")

// 	results := make(map[string]string)

// 	// 1. 检查设备管理器中的蓝牙设备
// 	deviceMgrCmd := `
// 		# 检查所有蓝牙相关设备
// 		Write-Host "=== 设备管理器检查 ==="

// 		# 获取所有蓝牙设备
// 		$btDevices = Get-PnpDevice | Where-Object {
// 			$_.Class -eq 'Bluetooth' -or
// 			$_.FriendlyName -like '*Bluetooth*' -or
// 			$_.FriendlyName -like '*BTH*' -or
// 			$_.Class -eq 'Net' -and $_.FriendlyName -like '*Bluetooth*'
// 		} | Select-Object Status, FriendlyName, InstanceId, Class

// 		if ($btDevices) {
// 			Write-Host "找到蓝牙设备:"
// 			foreach ($device in $btDevices) {
// 				Write-Host "  - $($device.FriendlyName) [$($device.Class)]: $($device.Status)"
// 			}
// 		} else {
// 			Write-Host "未找到蓝牙设备"
// 		}

// 		# 检查无线网卡
// 		$wirelessAdapters = Get-PnpDevice | Where-Object {
// 			$_.FriendlyName -like '*Wireless*' -or
// 			$_.FriendlyName -like '*WiFi*' -or
// 			$_.FriendlyName -like '*WLAN*'
// 		} | Select-Object Status, FriendlyName

// 		if ($wirelessAdapters) {
// 			Write-Host "无线网卡:"
// 			foreach ($adapter in $wirelessAdapters) {
// 				Write-Host "  - $($adapter.FriendlyName): $($adapter.Status)"
// 			}
// 		}
// 	`

// 	deviceMgrResult, _ := runPowerShell(deviceMgrCmd)
// 	results["device_manager"] = deviceMgrResult

// 	// 2. 检查网络适配器
// 	networkCmd := `
// 		Write-Host "=== 网络适配器检查 ==="
// 		$btAdapters = Get-NetAdapter | Where-Object {
// 			$_.InterfaceDescription -like '*Bluetooth*' -or
// 			$_.Name -like '*Bluetooth*'
// 		}

// 		if ($btAdapters) {
// 			Write-Host "蓝牙网络适配器:"
// 			foreach ($adapter in $btAdapters) {
// 				Write-Host "  - $($adapter.Name): $($adapter.InterfaceDescription), 状态: $($adapter.Status)"
// 			}
// 		} else {
// 			Write-Host "未找到蓝牙网络适配器"
// 		}
// 	`

// 	networkResult, _ := runPowerShell(networkCmd)
// 	results["network_adapters"] = networkResult

// 	// 3. 检查蓝牙服务
// 	serviceCmd := `
// 		Write-Host "=== 蓝牙服务检查 ==="

// 		$services = @(
// 			@{Name='BthServ'; DisplayName='蓝牙支持服务'},
// 			@{Name='BluetoothUserService'; DisplayName='蓝牙用户服务'}
// 		)

// 		foreach ($svc in $services) {
// 			$service = Get-Service -Name $svc.Name -ErrorAction SilentlyContinue
// 			if ($service) {
// 				Write-Host "  - $($svc.DisplayName): $($service.Status)"
// 			} else {
// 				Write-Host "  - $($svc.DisplayName): 未找到"
// 			}
// 		}
// 	`

// 	serviceResult, _ := runPowerShell(serviceCmd)
// 	results["services"] = serviceResult

// 	return results, nil
// }

// // AnalyzeStatus 分析状态并给出建议
// func (bc *BluetoothController) AnalyzeStatus() string {
// 	status, _ := bc.GetBluetoothStatus()

// 	var issues []string
// 	var suggestions []string

// 	// 分析设备管理器结果
// 	devicesText := status["device_manager"]
// 	if strings.Contains(devicesText, "未找到蓝牙设备") {
// 		issues = append(issues, "未检测到蓝牙硬件设备")
// 		suggestions = append(suggestions,
// 			"1. 检查设备管理器中是否有蓝牙设备\n"+
// 				"2. 检查BIOS/UEFI中的蓝牙设置是否启用\n"+
// 				"3. 检查是否有物理无线开关")
// 	} else if strings.Contains(devicesText, "Error") {
// 		issues = append(issues, "蓝牙设备存在错误")
// 		suggestions = append(suggestions, "更新蓝牙驱动程序")
// 	} else if strings.Contains(devicesText, "Disabled") {
// 		issues = append(issues, "蓝牙设备被禁用")
// 		suggestions = append(suggestions, "在设备管理器中启用蓝牙设备")
// 	}

// 	// 分析网络适配器结果
// 	networkText := status["network_adapters"]
// 	if strings.Contains(networkText, "未找到蓝牙网络适配器") {
// 		issues = append(issues, "蓝牙网络适配器未安装")
// 		suggestions = append(suggestions, "安装蓝牙驱动程序")
// 	} else if strings.Contains(networkText, "状态: Disabled") {
// 		issues = append(issues, "蓝牙网络适配器被禁用")
// 		suggestions = append(suggestions, "在网络连接中启用蓝牙适配器")
// 	}

// 	// 分析服务结果
// 	serviceText := status["services"]
// 	if strings.Contains(serviceText, "Stopped") {
// 		issues = append(issues, "蓝牙服务未运行")
// 		suggestions = append(suggestions, "启动蓝牙相关服务")
// 	}

// 	// 构建结果
// 	result := "分析结果:\n"

// 	if len(issues) > 0 {
// 		result += "\n发现问题:\n"
// 		for i, issue := range issues {
// 			result += fmt.Sprintf("  %d. %s\n", i+1, issue)
// 		}

// 		result += "\n建议解决方案:\n"
// 		for i, suggestion := range suggestions {
// 			result += fmt.Sprintf("  %d. %s\n", i+1, suggestion)
// 		}
// 	} else {
// 		result += "\n未发现明显问题。蓝牙状态正常。\n"
// 	}

// 	return result
// }

// // EnableBluetooth 启用蓝牙
// func (bc *BluetoothController) EnableBluetooth() error {
// 	fmt.Println("=== 开始启用蓝牙 ===")

// 	// 1. 启动蓝牙服务
// 	fmt.Println("1. 启动蓝牙服务...")
// 	serviceCmd := `
// 		# 设置服务为自动启动
// 		Set-Service -Name BthServ -StartupType Automatic -ErrorAction SilentlyContinue
// 		Set-Service -Name BluetoothUserService -StartupType Automatic -ErrorAction SilentlyContinue

// 		# 启动服务
// 		Start-Service -Name BthServ -ErrorAction SilentlyContinue
// 		Start-Service -Name BluetoothUserService -ErrorAction SilentlyContinue

// 		Write-Host "服务启动完成"
// 	`

// 	runPowerShell(serviceCmd)
// 	time.Sleep(2 * time.Second)

// 	// 2. 启用蓝牙设备
// 	fmt.Println("2. 启用蓝牙设备...")
// 	enableDeviceCmd := `
// 		# 启用所有蓝牙设备
// 		$btDevices = Get-PnpDevice | Where-Object {
// 			$_.Class -eq 'Bluetooth' -or
// 			$_.FriendlyName -like '*Bluetooth*'
// 		}

// 		foreach ($device in $btDevices) {
// 			try {
// 				if ($device.Status -ne 'OK') {
// 					Enable-PnpDevice -InstanceId $device.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
// 					Write-Host "已启用: $($device.FriendlyName)"
// 				}
// 			} catch {
// 				Write-Host "启用失败: $($device.FriendlyName)"
// 			}
// 		}

// 		Write-Host "设备启用完成"
// 	`

// 	runPowerShell(enableDeviceCmd)
// 	time.Sleep(2 * time.Second)

// 	// 3. 启用网络适配器
// 	fmt.Println("3. 启用蓝牙网络适配器...")
// 	enableAdapterCmd := `
// 		# 启用蓝牙网络适配器
// 		$btAdapters = Get-NetAdapter | Where-Object {
// 			$_.InterfaceDescription -like '*Bluetooth*'
// 		}

// 		foreach ($adapter in $btAdapters) {
// 			try {
// 				if ($adapter.Status -ne 'Up') {
// 					Enable-NetAdapter -Name $adapter.Name -Confirm:$false -ErrorAction SilentlyContinue
// 					Write-Host "已启用网络适配器: $($adapter.Name)"
// 				}
// 			} catch {
// 				Write-Host "启用失败: $($adapter.Name)"
// 			}
// 		}

// 		Write-Host "网络适配器启用完成"
// 	`

// 	runPowerShell(enableAdapterCmd)
// 	time.Sleep(3 * time.Second)

// 	// 4. 检查结果
// 	fmt.Println("4. 检查启用结果...")
// 	finalStatus, _ := bc.GetBluetoothStatus()

// 	// 简单判断是否成功
// 	success := false
// 	if strings.Contains(finalStatus["services"], "Running") &&
// 		(strings.Contains(finalStatus["network_adapters"], "状态: Up") ||
// 			strings.Contains(finalStatus["device_manager"], "OK")) {
// 		success = true
// 	}

// 	if success {
// 		fmt.Println("✓ 蓝牙启用成功")
// 		return nil
// 	} else {
// 		fmt.Println("✗ 蓝牙启用失败")
// 		fmt.Println("\n详细信息:")
// 		for key, value := range finalStatus {
// 			fmt.Printf("\n%s:\n%s\n", key, value)
// 		}
// 		return fmt.Errorf("蓝牙启用失败")
// 	}
// }

// // DisableBluetooth 禁用蓝牙
// func (bc *BluetoothController) DisableBluetooth() error {
// 	fmt.Println("=== 开始禁用蓝牙 ===")

// 	// 1. 禁用网络适配器
// 	fmt.Println("1. 禁用蓝牙网络适配器...")
// 	disableAdapterCmd := `
// 		$btAdapters = Get-NetAdapter | Where-Object {
// 			$_.InterfaceDescription -like '*Bluetooth*'
// 		}

// 		foreach ($adapter in $btAdapters) {
// 			try {
// 				Disable-NetAdapter -Name $adapter.Name -Confirm:$false -ErrorAction SilentlyContinue
// 				Write-Host "已禁用: $($adapter.Name)"
// 			} catch {
// 				Write-Host "禁用失败: $($adapter.Name)"
// 			}
// 		}
// 	`

// 	runPowerShell(disableAdapterCmd)
// 	time.Sleep(1 * time.Second)

// 	// 2. 禁用蓝牙设备
// 	fmt.Println("2. 禁用蓝牙设备...")
// 	disableDeviceCmd := `
// 		$btDevices = Get-PnpDevice | Where-Object {
// 			$_.Class -eq 'Bluetooth' -or
// 			$_.FriendlyName -like '*Bluetooth*'
// 		}

// 		foreach ($device in $btDevices) {
// 			try {
// 				Disable-PnpDevice -InstanceId $device.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
// 				Write-Host "已禁用设备: $($device.FriendlyName)"
// 			} catch {
// 				Write-Host "禁用失败: $($device.FriendlyName)"
// 			}
// 		}
// 	`

// 	runPowerShell(disableDeviceCmd)
// 	time.Sleep(1 * time.Second)

// 	// 3. 停止蓝牙服务
// 	fmt.Println("3. 停止蓝牙服务...")
// 	stopServiceCmd := `
// 		Stop-Service -Name BthServ -Force -ErrorAction SilentlyContinue
// 		Stop-Service -Name BluetoothUserService -Force -ErrorAction SilentlyContinue

// 		# 设置为手动启动
// 		Set-Service -Name BthServ -StartupType Manual -ErrorAction SilentlyContinue
// 		Set-Service -Name BluetoothUserService -StartupType Manual -ErrorAction SilentlyContinue

// 		Write-Host "服务已停止"
// 	`

// 	runPowerShell(stopServiceCmd)

// 	fmt.Println("✓ 蓝牙禁用完成")
// 	return nil
// }

// // ResetBluetooth 重置蓝牙
// func (bc *BluetoothController) ResetBluetooth() error {
// 	fmt.Println("=== 重置蓝牙 ===")

// 	// 1. 先禁用
// 	fmt.Println("1. 禁用蓝牙...")
// 	bc.DisableBluetooth()

// 	// 2. 等待
// 	fmt.Println("2. 等待5秒...")
// 	time.Sleep(5 * time.Second)

// 	// 3. 再启用
// 	fmt.Println("3. 重新启用蓝牙...")
// 	return bc.EnableBluetooth()
// }

// // ShowMenu 显示菜单
// func ShowMenu() {
// 	fmt.Println("\n=== 蓝牙修复工具菜单 ===")
// 	fmt.Println("1. 检查蓝牙状态")
// 	fmt.Println("2. 分析问题并给出建议")
// 	fmt.Println("3. 启用蓝牙")
// 	fmt.Println("4. 禁用蓝牙")
// 	fmt.Println("5. 重置蓝牙（禁用后重新启用）")
// 	fmt.Println("6. 显示此菜单")
// 	fmt.Println("7. 退出")
// 	fmt.Print("\n请选择操作 (1-7): ")
// }

// // 主函数
// // func main() {
// // 	// 设置控制台编码（Windows）
// // 	if runtime.GOOS == "windows" {
// // 		exec.Command("chcp", "65001").Run()
// // 	}

// // 	fmt.Println("蓝牙修复工具 v1.0")
// // 	fmt.Println("=================")

// // 	controller := NewBluetoothController()

// // 	// 检查平台
// // 	if runtime.GOOS != "windows" {
// // 		fmt.Println("错误：此工具仅支持Windows系统")
// // 		fmt.Println("按Enter键退出...")
// // 		fmt.Scanln()
// // 		return
// // 	}

// // 	// 检查管理员权限
// // 	if controller.isAdmin {
// // 		fmt.Println("✓ 以管理员权限运行")
// // 	} else {
// // 		fmt.Println("⚠ 未以管理员权限运行")
// // 		fmt.Println("部分功能可能需要管理员权限")
// // 		fmt.Println("建议以管理员身份运行此程序")
// // 	}

// // 	ShowMenu()

// // 	for {
// // 		var choice int
// // 		fmt.Scanln(&choice)

// // 		switch choice {
// // 		case 1:
// // 			fmt.Println("\n正在检查蓝牙状态...")
// // 			status, err := controller.GetBluetoothStatus()
// // 			if err != nil {
// // 				fmt.Printf("检查失败: %v\n", err)
// // 			} else {
// // 				fmt.Println("\n检查结果:")
// // 				for key, value := range status {
// // 					fmt.Printf("\n%s:\n%s\n", key, value)
// // 				}
// // 			}

// // 		case 2:
// // 			fmt.Println("\n正在分析蓝牙问题...")
// // 			result := controller.AnalyzeStatus()
// // 			fmt.Println(result)

// // 		case 3:
// // 			fmt.Println("\n正在启用蓝牙...")
// // 			if err := controller.EnableBluetooth(); err != nil {
// // 				fmt.Printf("启用失败: %v\n", err)
// // 			}

// // 		case 4:
// // 			fmt.Println("\n正在禁用蓝牙...")
// // 			if err := controller.DisableBluetooth(); err != nil {
// // 				fmt.Printf("禁用失败: %v\n", err)
// // 			}

// // 		case 5:
// // 			fmt.Println("\n正在重置蓝牙...")
// // 			if err := controller.ResetBluetooth(); err != nil {
// // 				fmt.Printf("重置失败: %v\n", err)
// // 			}

// // 		case 6:
// // 			ShowMenu()

// // 		case 7:
// // 			fmt.Println("退出程序...")
// // 			return

// // 		default:
// // 			fmt.Println("无效的选择，请输入1-7")
// // 		}

// // 		fmt.Println("\n按Enter键继续...")
// // 		fmt.Scanln()
// // 		ShowMenu()
// // 	}

// // }
