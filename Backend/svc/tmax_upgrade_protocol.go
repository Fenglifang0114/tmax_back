package svc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"strconv"
	"strings"
	"sync"
	"time"
	m "tmaxsrv/comm"

	mcrc "github.com/BertoldVdb/go-misc/multicrc"
	"go.bug.st/serial"
)

const (
	frameHeader1       = 0xFE
	frameHeader2       = 0xFE
	frameTail          = 0xFF
	commandEnter       = 0x02
	commandErase       = 0x04
	commandSend        = 0x03
	commandWrite       = 0x05
	commandReboot      = 0x08
	success            = 0x06
	failure            = 0x15
	receiveTimeout     = 50 * time.Millisecond
	eraseSectorTimeout = 50000 * time.Millisecond
	ringBufferSize     = 1024 // 环形缓冲区大小
)

// RingBuffer 环形缓冲区
type RingBuffer struct {
	buffer []byte
	head   int // 写入位置
	tail   int // 读取位置
	size   int // 当前数据长度
	lock   sync.Mutex
}

// NewRingBuffer 创建环形缓冲区
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]byte, capacity),
	}
}

// Put 写入数据（覆盖模式）
func (rb *RingBuffer) Put(data []byte) {
	rb.lock.Lock()
	defer rb.lock.Unlock()

	for _, b := range data {
		if rb.size >= len(rb.buffer) {
			// 缓冲区已满，覆盖最旧数据
			rb.tail = (rb.tail + 1) % len(rb.buffer)
			rb.size--
		}
		rb.buffer[rb.head] = b
		rb.head = (rb.head + 1) % len(rb.buffer)
		rb.size++
	}
}

// Get 读取指定长度的数据
func (rb *RingBuffer) Get(length int) []byte {
	rb.lock.Lock()
	defer rb.lock.Unlock()

	if rb.size < length {
		return nil
	}

	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = rb.buffer[rb.tail]
		rb.tail = (rb.tail + 1) % len(rb.buffer)
	}
	rb.size -= length
	return data
}

// Port 假设的串口结构体
type Port struct {
	// 这里可以添加串口相关的字段
}

// Send 发送数据
func (p *Port) Send(cmd []byte) error {
	// 这里需要实现具体的发送逻辑
	// 示例中简单返回 nil 表示发送成功
	fmt.Printf("Sending command: %v", cmd)
	return nil
}

// Receive 接收数据
func (p *Port) Receive() ([]byte, error) {
	// 这里需要实现具体的接收逻辑
	// 示例中简单返回模拟数据
	log.Println("Receiving data...")
	return []byte{0x01, 0x02, 0x03}, nil
}

// ReceiveTimeout 实现带有超时的接收方法
func (p *Port) ReceiveTimeout(timeout time.Duration) ([]byte, error) {
	receiveChan := make(chan []byte)
	errorChan := make(chan error)

	go func() {
		data, err := p.Receive()
		if err != nil {
			errorChan <- err
			return
		}
		receiveChan <- data
	}()

	select {
	case data := <-receiveChan:
		return data, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("receive timeout")
	}
}

// Upgrader 升级器结构体
type Upgrader struct {
	scale       *Scale
	port        serial.Port
	sendChan    chan []byte
	receiveChan chan []byte
	errorChan   chan error
	stopChan    chan struct{}
	wg          sync.WaitGroup
	ringBuffer  *RingBuffer
	tmpbuf      []byte
	// messageChan chan []byte

	// messageChan chan comm.ResponseCommand
}

// NewUpgrader 创建升级器实例
func NewUpgrader(c *Scale, port serial.Port) *Upgrader {
	// func NewUpgrader(port serial.Port, messageChan chan comm.ResponseCommand) *Upgrader {
	u := &Upgrader{
		scale:       c,
		port:        port,
		sendChan:    make(chan []byte, 10),
		receiveChan: make(chan []byte, 10),
		errorChan:   make(chan error, 10),
		stopChan:    make(chan struct{}),
		ringBuffer:  NewRingBuffer(ringBufferSize),
		tmpbuf:      make([]byte, 1024),
		// messageChan: c.client.sendCh,
		// messageChan: messageChan,
	}
	u.startCommunication()
	return u
}

// PerformUpgrader 运行升级流程
func (u *Upgrader) PerformUpgrader(srecData []byte) (*ScaleRespMsg, error) {
	// defer u.stopCommunication()

	var respMsg *ScaleRespMsg
	respOk := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "ok", ScaleId: u.scale.Id}
	respFail := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail", ScaleId: u.scale.Id}

	// 步骤 1: 进入升级模式
	if err := u.enterUpgradeMode(); err != nil {
		log.Println("Failed to enter upgrade mode: ", err)
		return respFail, fmt.Errorf("enter upgrade mode failed: %w", err)
	}
	respMsg = &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: "started", ScaleId: u.scale.Id}
	result, _ := json.Marshal(respMsg)
	u.scale.client.sendCh <- result

	if err := u.eraseFlash(); err != nil {
		log.Println("Failed to erase flash: ", err)
		return respFail, fmt.Errorf("erase flash failed: %w", err)
	}
	respMsg = &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: "erased flash", ScaleId: u.scale.Id}
	result, _ = json.Marshal(respMsg)
	u.scale.client.sendCh <- result

	// 步骤 4 - 6: 传输并写入固件
	if err := u.transferAndWriteFirmware(srecData); err != nil {
		log.Println("Failed to transfer and write firmware: ", err)
		respMsg = respFail
		result, _ = json.Marshal(respMsg)
		u.scale.client.sendCh <- result
		return respFail, fmt.Errorf("transfer and write firmware failed: %w", err)
	}
	// 步骤 7: 重启设备
	if err := u.rebootDevice(); err != nil {
		log.Println("Failed to reboot device:", err)
		return respFail, fmt.Errorf("reboot device failed: %w", err)
	}
	log.Println(" 升级流程完成 ")
	respMsg = &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: "completed", ScaleId: u.scale.Id}
	result, _ = json.Marshal(respMsg)
	u.scale.client.sendCh <- result

	return respOk, nil
}

// enterUpgradeMode 进入升级模式（修复通道关闭引发的panic）
func (u *Upgrader) enterUpgradeMode() error {
	initCmd := []byte{0x02, 0xff, 0x00}
	// expectedResp := []byte{0x00, 0xff, 0x08}
	expectedResp := []byte{0x03, 0xff, 0x08}
	timeout := 50 * time.Second
	successChan := make(chan struct{}, 1)
	done := make(chan struct{})
	var once sync.Once

	// 关闭done通道的函数，确保只关闭一次
	closeDone := func() {
		once.Do(func() {
			close(done)
		})
	}

	// 发送任务
	go func() {
		defer closeDone() // 使用once确保只关闭一次

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				select {
				case u.sendChan <- initCmd:
					// 命令已发送
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 接收任务 - 修改部分
	go func() {
		var buffer []byte // 用于存储累积的响应数据

		for {
			select {
			case resp := <-u.receiveChan:
				if len(resp) > 0 {
					// 将新接收到的数据添加到缓冲区
					buffer = append(buffer, resp...)

					// 检查缓冲区是否包含完整的预期响应
					if len(buffer) >= len(expectedResp) {
						// 查找预期响应的起始位置
						start := 0
						for start <= len(buffer)-len(expectedResp) {
							if bytes.Equal(buffer[start:start+len(expectedResp)], expectedResp) {
								// 找到完整响应
								select {
								case successChan <- struct{}{}:
									log.Println("收到已经进入升级模式信息，停止发送进入升级模式请求")
								default:
									// 已有响应被处理，忽略此响应
								}
								closeDone()
								return
							}
							start++
						}

						// 如果缓冲区太长，可以移除前面的数据以避免无限增长
						if len(buffer) > 2*len(expectedResp) {
							buffer = buffer[len(buffer)-len(expectedResp):]
						}
					}
				}
			case <-done:
				return
			}
		}
	}()

	// 等待成功、超时或停止
	select {
	case <-successChan:
		log.Println("成功进入升级模式")
		return nil
	case <-time.After(timeout):
		log.Println("进入升级模式超时")
		closeDone()
		return fmt.Errorf("enter upgrade mode timeout,fail")
	case <-u.stopChan:
		log.Println("通信在进入升级模式时停止")
		closeDone()
		return fmt.Errorf("fail,connect stoped")
	}
}

// eraseFlash 擦除APP扇区
func (u *Upgrader) eraseFlash() error {
	maxRetries := 3
	retryCount := 0

	for retryCount < maxRetries {
		// 创建擦除所有block的命令
		eraseCmd := createFrame(commandErase, []byte{0x99}) // 0x99 是擦除所有扇区的命令
		log.Println("正在尝试擦除所有扇区  尝试次数: ", retryCount)
		u.sendChan <- eraseCmd

		// 创建一个缓冲区来累积接收到的数据
		var buffer []byte
		success := false

		// 设置超时时间，考虑擦除操作需要2秒
		timeout := time.NewTimer(5 * time.Second)
		defer timeout.Stop()

	Loop:
		for {
			select {
			case resp := <-u.receiveChan:
				if len(resp) > 0 {
					// 将新数据添加到缓冲区
					buffer = append(buffer, resp...)
					// 处理缓冲区中的数据
					for len(buffer) >= 10 { // 至少需要10个字节才能包含完整响应
						// 查找响应前缀 0xfe, 0xfe, 0x07, 0x44, 0x99
						if buffer[0] == 0xfe && buffer[1] == 0xfe &&
							buffer[2] == 0x07 && buffer[3] == 0x44 && buffer[4] == 0x99 {

							// 验证CRC32 (bytes 5-8)
							data := buffer[2:5]
							crcReceived := uint32(buffer[5])<<24 | uint32(buffer[6])<<16 |
								uint32(buffer[7])<<8 | uint32(buffer[8])

							crcCalculated := CalculateCRC32MPEG2(data)

							// 验证CRC和结束字节
							if crcCalculated == crcReceived && buffer[9] == 0xff {
								log.Println("收到有效擦除完成通知，CRC验证成功")
								success = true
								break Loop
							}
						}

						// 如果不是有效的响应前缀，移除第一个字节继续查找
						buffer = buffer[1:]
					}
				}

			case err := <-u.errorChan:
				if !isRecoverableError(err) {
					return fmt.Errorf("通信错误: %w", err)
				}
				log.Printf("接收擦除完成通知时临时错误: %v", err)

			case <-timeout.C:
				log.Println("擦除超时-重试中 尝试次数: ", retryCount+1)
				retryCount++
				break Loop

			case <-u.stopChan:
				return fmt.Errorf("user stoped")
			}
		}

		if success {
			log.Println("所有扇区擦除成功")
			return nil
		}
	}

	return fmt.Errorf("erase flash failed (try %d time)", maxRetries)
}

// validateFrame 验证帧格式和CRC
func validateFrame(frame []byte) bool {
	if len(frame) < 8 || frame[0] != 0xFE || frame[1] != 0xFE || frame[len(frame)-1] != 0xFF {
		return false
	}

	// 验证长度
	expectedLength := int(frame[2])
	if len(frame)-3 != expectedLength {
		return false
	}

	// 验证CRC
	dataToCRC := frame[2 : len(frame)-5] // 长度、命令、数据
	expectedCRC := frame[len(frame)-5 : len(frame)-1]
	calculatedCRC := Uint32ToBytes(CalculateCRC32MPEG2(dataToCRC))

	return bytes.Equal(expectedCRC, calculatedCRC)
}

// 判断是否为可恢复错误（根据实际协议定义）
func isRecoverableError(err error) bool {
	return strings.Contains(err.Error(), "crc check failed") ||
		strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "EOF")
}

// transferAndWriteFirmware 传输并写入固件
func (u *Upgrader) transferAndWriteFirmware(srecData []byte) error {
	blockSize := 2048 // 2K块大小
	segmentSize := 64
	segmentCount := blockSize / segmentSize                   // 64字节段大小
	blockCount := (len(srecData) + blockSize - 1) / blockSize // 计算总块数

	// 配置参数 - 可根据实际硬件调整
	maxRetries := 32 // 每个段的最大重试次数

	// 启动块处理协程
	for i := blockCount - 1; i >= 0; i-- {
		//for i := 0; i < blockCount; i++ {
		if err := u.processBlock(srecData, i, blockSize, segmentSize, segmentCount, maxRetries); err != nil {
			return err
		}
		if err := u.writeBlockToFlash(i); err != nil {
			return err
		}
	}

	// 跟踪已完成的块数
	return nil
}

// processBlock 处理固件块传输 (同步等待回复)
func (u *Upgrader) processBlock(firmwareData []byte, blockIndex, blockSize, segmentSize, segmentCount, maxRetries int) error {
	blockData, err := u.prepareBlockData(firmwareData, blockIndex, blockSize)
	if err != nil {
		return err
	}

	for segIdx := 0; segIdx < segmentCount; segIdx++ {
		successFlag := false
		var lastError error

		for attempt := 0; attempt < maxRetries; attempt++ {
			err := u.sendSegment(blockData, segIdx, segmentSize)
			if err != nil {
				return fmt.Errorf("send segment %d failed: %v", segIdx, err)
			}

			// 同步等待当前段的 0x06 回复
			var buffer []byte
			timeout := time.NewTimer(300 * time.Millisecond) // 设置合适的超时时间

		receiveLoop:
			for {
				select {
				case resp, ok := <-u.receiveChan:
					if !ok {
						lastError = fmt.Errorf("接收通道已关闭")
						break receiveLoop
					}
					if !timeout.Stop() {
						<-timeout.C
					}
					timeout.Reset(300 * time.Millisecond)

					buffer = append(buffer, resp...)

					for len(buffer) >= 10 { // 至少需要10个字节
						if buffer[0] == 0xFE && buffer[1] == 0xFE {
							expectedLen := int(buffer[2])
							frameLen := expectedLen + 3
							if len(buffer) >= frameLen {
								frame := buffer[:frameLen]
								if validateFrame(frame) && frame[3] == 0x43 {
									// 协议没变：秤返回的是段号，不会返回 0x06。
									segmentNum := int(frame[4])
									if segmentNum == segIdx {
										successFlag = true
										// 从缓冲区移除已处理的帧，并跳出接收循环
										buffer = buffer[frameLen:]
										break receiveLoop
									} else {
										// 收到非预期的段号，可能是上一次超时的旧回复，忽略之，继续等待
										// 消费掉这个不需要的帧，继续外层 for len(buffer) >= 10 循环
										buffer = buffer[frameLen:]
										continue
									}
								}
							} else {
								// 长度不够，等待更多数据
								break
							}
						}
						// 找不到有效的帧头，移除第一个字节继续查找
						buffer = buffer[1:]
					}

				case err := <-u.errorChan:
					if !isRecoverableError(err) {
						return fmt.Errorf("通信错误: %w", err)
					}
					lastError = fmt.Errorf("临时通信错误: %v", err)
					break receiveLoop

				case <-timeout.C:
					lastError = fmt.Errorf("等待段 %d 响应超时", segIdx)
					break receiveLoop

				case <-u.stopChan:
					return fmt.Errorf("升级被用户取消")
				}
			}
			timeout.Stop()

			if successFlag {
				break // 当前段发送成功，退出重试循环，处理下一段
			}

			// 发送失败或超时，准备重试
			// log.Printf("段 %d 发送失败，准备重试 (%d/%d): %v", segIdx, attempt+1, maxRetries, lastError)
			time.Sleep(10 * time.Millisecond)
		}

		if !successFlag {
			return fmt.Errorf("段 %d 发送失败，达到最大重试次数: %v", segIdx, lastError)
		}
	}

	return nil
}

// prepareBlockData 准备块数据，处理边界情况
func (u *Upgrader) prepareBlockData(firmwareData []byte, blockIndex, blockSize int) ([]byte, error) {
	blockOffset := blockIndex * blockSize
	blockDataLen := min(blockSize, len(firmwareData)-blockOffset)

	// 预分配块数据缓冲区
	blockBuffer := make([]byte, blockSize)
	copy(blockBuffer[:blockDataLen], firmwareData[blockOffset:blockOffset+blockDataLen])

	// 不足部分用0xFF填充
	for i := blockDataLen; i < blockSize; i++ {
		blockBuffer[i] = 0xFF
	}

	return blockBuffer, nil
}

// sendSegment 发送单个段
func (u *Upgrader) sendSegment(blockData []byte, segIndex, segmentSize int) error {
	segOffset := segIndex * segmentSize
	relativeAddr := byte(segIndex)

	sendCmd := createFrame(commandSend, append([]byte{relativeAddr}, blockData[segOffset:segOffset+segmentSize]...))
	u.sendChan <- sendCmd
	return nil
}

func (u *Upgrader) writeBlockToFlash(blockIndex int) error {
	// 定义常量提升可维护性
	const (
		maxRetries     = 2
		receiveTimeout = receiveTimeout
		minProcess     = 90
		maxBlocks      = 61
	)

	block := byte(blockIndex)
	writeCmd := createFrame(commandWrite, []byte{block})
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 发送写入命令
		u.sendChan <- writeCmd
		// 初始化缓冲区和计时器
		var buffer []byte
		timeout := time.NewTimer(receiveTimeout)
		// 使用带标签的循环便于提前退出
	receiveLoop:
		for {
			select {
			case resp, ok := <-u.receiveChan:
				if !ok {
					lastError = fmt.Errorf("接收通道已关闭")
					break receiveLoop
				}
				// 重置计时器
				if !timeout.Stop() {
					<-timeout.C
				}
				timeout.Reset(receiveTimeout)

				// 拼接数据到缓冲区
				buffer = append(buffer, resp...)
				// log.Printf("收到数据，缓冲区长度: %d", len(buffer))

				// 循环处理可能包含的多个响应
				for len(buffer) >= 5 {
					// 检查是否是我们要找的响应
					if buffer[3] == commandWrite {
						// 确保有完整的响应数据
						if len(buffer) < 5 {
							lastError = fmt.Errorf("响应数据不完整，需要至少5字节，实际收到: %d字节", len(buffer))
							break
						}

						// 检查响应状态
						if buffer[4] != success {
							lastError = fmt.Errorf("写入Flash失败，块 %d，错误码: %d", blockIndex, buffer[4])
							break receiveLoop
						}

						// 计算进度并发送更新
						process := minProcess - blockIndex*100/maxBlocks
						respMsg := &ScaleRespMsg{
							MsgType: m.UPDATE_FIRMWARE_PROGRESS,
							MsgBody: strconv.Itoa(process),
							ScaleId: u.scale.Id,
						}
						// 优化JSON序列化错误处理
						result, _ := json.Marshal(respMsg)
						u.scale.client.sendCh <- result

						log.Printf("成功写入块 %d 到Flash，尝试次数: %d", blockIndex, attempt+1)
						return nil
					}

					// 不是我们要找的响应，跳过这个包
					// log.Printf("跳过不匹配的响应，命令: %s", hex.EncodeToString(buffer[:5]))
					buffer = buffer[1:]
				}

			case err := <-u.errorChan:
				lastError = fmt.Errorf("通信错误: %w", err)
				break receiveLoop

			case <-timeout.C:
				lastError = fmt.Errorf("写入响应超时，块 %d，尝试次数: %d，缓冲区长度: %d",
					blockIndex, attempt+1, len(buffer))
				break receiveLoop

			case <-u.stopChan:
				lastError = fmt.Errorf("升级已取消")
				break receiveLoop
			}
		}

		// 确保计时器被停止
		timeout.Stop()

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries-1 {
			return lastError
			// break
		}

		// 指数退避重试
		backoff := time.Duration(5<<attempt) * time.Millisecond
		log.Printf("写入块 %d 失败，准备重试 (%d/%d)，延迟: 10ms", blockIndex, attempt+1, maxRetries)
		time.Sleep(backoff)
	}

	if lastError != nil {
		return fmt.Errorf("写入块 %d 失败: %w", blockIndex, lastError)
	}
	return fmt.Errorf("写入块 %d 失败，达到最大重试次数 %d", blockIndex, maxRetries)
}

// writeBlockToFlash 将块写入Flash
// func (u *Upgrader) writeBlockToFlash(blockIndex int) error {
// 	block := byte(blockIndex)
// 	writeCmd := createFrame(commandWrite, []byte{block})

// 	const maxRetries = 3
// 	for attempt := 0; attempt < maxRetries; attempt++ {
// 		// 发送写入命令
// 		u.sendChan <- writeCmd

// 		// 初始化缓冲区和计时器
// 		var buffer []byte
// 		var lastError error
// 		timeout := time.NewTimer(2 * receiveTimeout)
// 		defer timeout.Stop()

// 		for {
// 			select {
// 			case resp := <-u.receiveChan:
// 				// 重置计时器
// 				if !timeout.Stop() {
// 					<-timeout.C
// 				}
// 				timeout.Reset(2 * receiveTimeout)

// 				// 拼接数据到缓冲区
// 				buffer = append(buffer, resp...)
// 				log.Printf("收到数据，缓冲区长度: %d", len(buffer))

// 				// 循环处理可能包含的多个响应
// 				for len(buffer) >= 5 {
// 					// 检查是否是我们要找的响应
// 					if buffer[3] == commandWrite {
// 						// 确保有完整的响应数据
// 						if len(buffer) < 5 {
// 							lastError = fmt.Errorf("响应数据不完整，长度: %d", len(buffer))

// 							break
// 						}

// 						// 检查响应状态
// 						if buffer[4] != success {
// 							return fmt.Errorf("写入Flash失败，块 %d，错误码: %d", blockIndex, buffer[4])
// 						}
// 						process := 90 - blockIndex*100/61
// 						respMsg := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(process), ScaleId: u.scale.Id}
// 						result, _ := json.Marshal(respMsg)
// 						u.scale.client.sendCh <- result
// 						// u.messageChan <- comm.ResponseCommand{
// 						// 	Code:    1,
// 						// 	Message: "写入 Flash",
// 						// 	Action:  "upgrade_tmax",
// 						// 	Process: 90 - blockIndex*100/61,
// 						// }
// 						log.Println("成功写入块 ", blockIndex, " 到Flash，尝试次数: ", attempt+1)
// 						return nil
// 					}

// 					// 不是我们要找的响应，跳过这个包
// 					log.Println("跳过不匹配的响应，命令: ", hex.EncodeToString(buffer[0:]))
// 					buffer = buffer[1:] // 移除第一个字节，继续检查剩余数据
// 				}

// 			case err := <-u.errorChan:
// 				return err

// 			case <-timeout.C:
// 				lastError = fmt.Errorf("写入响应超时，块 %d，尝试次数: %d，缓冲区长度: %d",
// 					blockIndex, attempt+1, len(buffer))
// 				return fmt.Errorf("超时")

// 			case <-u.stopChan:
// 				return fmt.Errorf("通信已停止")
// 			}
// 		}

// 		// 如果是最后一次尝试，返回错误
// 		if attempt == maxRetries-1 {
// 			return lastError
// 		}

// 		// 重试前等待一小段时间
// 		log.Printf("写入块 %d 失败，准备重试 (%d/%d)...", blockIndex, attempt+1, maxRetries)
// 		time.Sleep(10 * time.Millisecond) // 退避时间
// 	}

// 	return fmt.Errorf("写入块 %d 失败，达到最大重试次数", blockIndex)
// }

// 辅助函数：返回两个数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 重启秤
func (u *Upgrader) rebootDevice() error {
	rebootCmd := createFrame(commandReboot, []byte{10}) // 0x99 是擦除所有扇区的命令
	u.sendChan <- rebootCmd
	log.Println("重启秤指令已发送！")
	return nil
}

// 辅助函数：验证重启响应帧
// func validRebootResponse(frame []byte) bool {
// 	return len(frame) >= 10 &&
// 		frame[0] == 0xfe &&
// 		frame[1] == 0xfe &&
// 		frame[3] == commandReboot
// }

// createFrame 创建协议帧
func createFrame(cmd byte, data []byte) []byte {
	frameLength := byte(1 + len(data) + 4 + 1) // 命令 + 数据 + CRC + 帧尾
	frame := []byte{frameHeader1, frameHeader2, frameLength, cmd}
	frame = append(frame, data...)
	crc := CalculateCRC32MPEG2(frame[2:])
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc)
	frame = append(frame, crcBytes...)
	frame = append(frame, frameTail)
	return frame
}

// startCommunication 启动通信goroutines
func (u *Upgrader) startCommunication() {
	u.wg.Add(2)
	go u.sender()
	go u.receiver()
}

// sender 发送goroutine
func (u *Upgrader) sender() {
	defer u.wg.Done()
	for {
		select {
		case cmd := <-u.sendChan:
			n, err := u.port.Write(cmd)
			if err != nil || n != len(cmd) {
				log.Printf("Error while sending command: %v", err)
			}
			time.Sleep(5 * time.Millisecond)
		case <-u.stopChan:
			return
		}
	}
}

// receiver 接收goroutine（使用环形缓冲区）
func (u *Upgrader) receiver() {
	defer u.wg.Done()
	u.port.SetReadTimeout(100 * time.Millisecond)
	for {
		select {
		case <-u.stopChan:
			return
		default:
			n, err := u.port.Read(u.tmpbuf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // 忽略超时错误
				}
				log.Printf("Error reading scale: %v", err)
				u.errorChan <- err
				return // 遇到非超时错误直接退出
			}
			time.Sleep(5 * time.Millisecond)
			if n > 0 {
				u.ringBuffer.Put(u.tmpbuf[0:n])
				select {
				case u.receiveChan <- u.tmpbuf[0:n]:
				case <-u.stopChan:
					return
				}
			}
		}
	}
}

// stopCommunication 停止通信goroutines
// func (u *Upgrader) stopCommunication() {
// 	close(u.stopChan)
// 	u.wg.Wait()
// 	close(u.sendChan)
// 	close(u.receiveChan)
// 	close(u.errorChan)
// }

func CalculateCRC32MPEG2(data []byte) uint32 {
	crc := mcrc.NewCRC(mcrc.Crc32MPEG2)
	crc.Reset().AddBytes(data)
	return crc.Result32()
}
func Uint32ToBytes(u uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, u)
	return buf
}
