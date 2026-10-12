package svc

import (
	"context"

	"fmt"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	// 单例适配器，避免重复创建
	btAdapter      *bluetooth.Adapter
	adapterOnce    sync.Once
	adapterInitErr error

	// 扫描互斥锁，防止并发扫描
	scanMutex  sync.Mutex
	isScanning bool

	// 用于停止当前扫描的上下文
	currentScanCtx    context.Context
	currentScanCancel context.CancelFunc
)

// 初始化适配器
func initAdapter() (*bluetooth.Adapter, error) {
	if btAdapter == nil {
		btAdapter = bluetooth.DefaultAdapter
	}

	if err := btAdapter.Enable(); err != nil {
		if !isAlreadyEnabled(err) {
			return nil, fmt.Errorf("enable adapter failed: %w", err)
		}
	}

	return btAdapter, nil
}

// 清理适配器状态
func resetAdapter() {
	scanMutex.Lock()
	defer scanMutex.Unlock()

	if btAdapter != nil {
		// 停止扫描
		_ = btAdapter.StopScan()
		isScanning = false

		// 取消当前扫描上下文
		if currentScanCancel != nil {
			currentScanCancel()
			currentScanCancel = nil
		}

		// 等待一段时间确保扫描完全停止
		time.Sleep(100 * time.Millisecond)
	}
}

// 强制停止扫描（外部可调用）
func ForceStopScan() {
	resetAdapter()
}

// 可独立使用的扫描函数
func GetBtList() (string, error) {
	// 防止并发扫描
	scanMutex.Lock()
	if isScanning {
		scanMutex.Unlock()
		return "", fmt.Errorf("扫描正在进行中，请稍后再试")
	}
	isScanning = true
	scanMutex.Unlock()

	// 确保扫描结束后重置状态
	defer func() {
		scanMutex.Lock()
		isScanning = false
		scanMutex.Unlock()
	}()

	// 1. 初始化适配器
	adapter, err := initAdapter()
	if err != nil {
		return "", fmt.Errorf("初始化适配器失败: %w", err)
	}

	devices := make(map[string]bluetooth.ScanResult)
	var results []bluetooth.ScanResult
	var resultsMu sync.Mutex

	// 创建扫描上下文（用于超时控制，默认10秒）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// 保存当前扫描上下文，以便外部可以取消
	scanMutex.Lock()
	if currentScanCancel != nil {
		currentScanCancel()
	}
	currentScanCtx = ctx
	currentScanCancel = cancel
	scanMutex.Unlock()

	defer cancel()

	fmt.Println("正在扫描蓝牙设备（最多10秒）...")
	start := time.Now()

	// 启动 watchdog 协程：当超时或取消时，主动调用 adapter.StopScan() 打断阻塞
	stopWatchdog := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = adapter.StopScan()
		case <-stopWatchdog:
		}
	}()
	defer close(stopWatchdog)

	// 开始阻塞扫描
	err = adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		select {
		case <-ctx.Done():
			_ = adapter.StopScan()
			return
		default:
		}

		address := device.Address.String()
		name := device.LocalName()

		resultsMu.Lock()
		defer resultsMu.Unlock()

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
				name = "No Name"
			}

			elapsed := time.Since(start).Seconds()
			fmt.Printf("[%.1fs] 发现设备: %s, 名称: %s, RSSI: %d\n",
				elapsed, address, name, device.RSSI)
		}
	})

	_ = adapter.StopScan()

	if err != nil && ctx.Err() == nil {
		return "", fmt.Errorf("启动扫描失败: %v", err)
	}

	fmt.Printf("\n扫描完成，发现 %d 个设备\n", len(results))

	// 转换为 JSON
	btInfoList := make([]BtInfoList, 0)
	for _, device := range results {
		btInfoList = append(btInfoList, BtInfoList{
			Mac:  device.Address.String(),
			Name: device.LocalName(),
			RSSI: int(device.RSSI),
		})
	}

	jsonBytes, err := json.Marshal(btInfoList)
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %w", err)
	}

	return string(jsonBytes), nil
}

// 带手动停止的扫描函数
func GetBtListWithStopControl() (string, context.CancelFunc, error) {
	scanMutex.Lock()
	if isScanning {
		scanMutex.Unlock()
		return "", nil, fmt.Errorf("扫描正在进行中，请稍后再试")
	}
	isScanning = true
	scanMutex.Unlock()

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 保存取消函数
	scanMutex.Lock()
	if currentScanCancel != nil {
		currentScanCancel()
	}
	currentScanCtx = ctx
	currentScanCancel = cancel
	scanMutex.Unlock()

	devices := make(map[string]bluetooth.ScanResult)
	var results []bluetooth.ScanResult
	var resultsMu sync.Mutex

	adapter, err := initAdapter()
	if err != nil {
		cancel()
		return "", nil, fmt.Errorf("初始化适配器失败: %w", err)
	}

	fmt.Println("正在扫描蓝牙设备（手动停止）...")
	start := time.Now()

	scanDone := make(chan struct{})
	scanErr := make(chan error, 1)

	go func() {
		defer close(scanDone)
		defer func() {
			_ = adapter.StopScan()
			time.Sleep(100 * time.Millisecond)
		}()

		stopWatchdog := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				_ = adapter.StopScan()
			case <-stopWatchdog:
			}
		}()
		defer close(stopWatchdog)

		err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			select {
			case <-ctx.Done():
				_ = adapter.StopScan()
				return
			default:
			}

			address := device.Address.String()
			name := device.LocalName()

			resultsMu.Lock()
			defer resultsMu.Unlock()

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
					name = "No Name"
				}

				elapsed := time.Since(start).Seconds()
				fmt.Printf("[%.1fs] 发现设备: %s, 名称: %s, RSSI: %d\n",
					elapsed, address, name, device.RSSI)
			}
		})

		if err != nil && ctx.Err() == nil {
			scanErr <- err
			return
		}
	}()

	// 返回取消函数，让调用者可以随时停止扫描
	stopFunc := func() {
		cancel()
		<-scanDone

		scanMutex.Lock()
		isScanning = false
		currentScanCancel = nil
		currentScanCtx = nil
		scanMutex.Unlock()

		fmt.Printf("\n扫描已手动停止，发现 %d 个设备\n", len(results))
	}

	// 在另一个goroutine中等待扫描完成
	go func() {
		select {
		case <-scanDone:
			// 扫描完成
			scanMutex.Lock()
			isScanning = false
			currentScanCancel = nil
			currentScanCtx = nil
			scanMutex.Unlock()
		case err := <-scanErr:
			fmt.Printf("扫描错误: %v\n", err)
			scanMutex.Lock()
			isScanning = false
			currentScanCancel = nil
			currentScanCtx = nil
			scanMutex.Unlock()
		}
	}()

	// 立即返回，调用者可以通过cancelFunc停止扫描
	return "", stopFunc, nil
}

// 带超时控制的扫描函数（更简洁版本）
func GetBtListWithTimeout(timeout time.Duration) (string, error) {
	scanMutex.Lock()
	if isScanning {
		scanMutex.Unlock()
		return "", fmt.Errorf("扫描正在进行中，请稍后再试")
	}
	isScanning = true
	scanMutex.Unlock()

	defer func() {
		scanMutex.Lock()
		isScanning = false
		scanMutex.Unlock()
	}()

	adapter, err := initAdapter()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	devices := make(map[string]bluetooth.ScanResult)
	var results []bluetooth.ScanResult
	var mu sync.Mutex

	fmt.Printf("正在扫描蓝牙设备（最多%v）...\n", timeout)
	start := time.Now()

	// 使用WaitGroup确保扫描完全停止
	var wg sync.WaitGroup
	wg.Add(1)

	scanErr := make(chan error, 1)

	// 启动扫描
	err = adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		select {
		case <-ctx.Done():
			// 超时，停止扫描
			adapter.StopScan()
			wg.Done()
			return
		default:
		}

		mu.Lock()
		defer mu.Unlock()

		address := device.Address.String()
		if _, exists := devices[address]; !exists {
			devices[address] = device
			results = append(results, device)

			name := device.LocalName()
			if name == "" {
				name = "No Name"
			}

			elapsed := time.Since(start).Seconds()
			fmt.Printf("[%.1fs] 发现设备: %s, 名称: %s, RSSI: %d\n",
				elapsed, address, name, device.RSSI)
		}
	})

	if err != nil {
		return "", fmt.Errorf("启动扫描失败: %v", err)
	}

	// 等待超时
	go func() {
		<-ctx.Done()
		adapter.StopScan()
		// 等待扫描回调完成
		wg.Wait()
		scanErr <- nil
	}()

	// 等待扫描停止
	<-scanErr

	fmt.Printf("\n扫描完成，发现 %d 个设备\n", len(results))

	// 转换为JSON
	btInfoList := make([]BtInfoList, 0)
	for _, device := range results {
		btInfoList = append(btInfoList, BtInfoList{
			Mac:  device.Address.String(),
			Name: device.LocalName(),
			RSSI: int(device.RSSI),
		})
	}

	jsonBytes, err := json.Marshal(btInfoList)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// 测试函数
func TestScan() {
	fmt.Println("=== 测试蓝牙扫描 ===")

	// 测试1：正常扫描10秒
	fmt.Println("\n1. 正常扫描10秒")
	result, err := GetBtListWithTimeout(10 * time.Second)
	if err != nil {
		fmt.Printf("扫描失败: %v\n", err)
	} else {
		fmt.Printf("扫描成功，发现设备: %s\n", result)
	}

	// 等待2秒
	time.Sleep(2 * time.Second)

	// 测试2：快速扫描5秒
	fmt.Println("\n2. 快速扫描5秒")
	result, err = GetBtListWithTimeout(5 * time.Second)
	if err != nil {
		fmt.Printf("扫描失败: %v\n", err)
	} else {
		fmt.Printf("扫描成功，发现设备: %s\n", result)
	}

	// 测试3：检查扫描状态
	scanMutex.Lock()
	fmt.Printf("\n扫描状态: isScanning=%v\n", isScanning)
	scanMutex.Unlock()
}
