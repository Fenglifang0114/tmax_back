package svc

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tmaxsrv/comm"
	"tmaxsrv/log"
	"tmaxsrv/picker"
	"tmaxsrv/util"
)

type TNet struct {
	// net port
	port int
	ip   string
	// conn net.Conn
	conn *net.TCPConn
	// send to scale channel, message will be json string
	sendCh     chan []byte
	recvCh     chan comm.Packet
	wg         sync.WaitGroup
	queue      *util.CircularBuffer
	pickerFn   picker.PickerFunc
	tmpbuf     []byte // for storing temporary data read from scale
	toQuit     bool   // for informing the read/write goroutine to quit
	maxPackLen int    // max package size that scale can receive on time
	isAlive    bool
	isDefault  bool //用来判断同一台秤，多种连接方式的情况下，是否走这个通道
	timeOutCnt int  // 超时次数
	IsModbusTCP   bool
	transactionID uint32
}

// NewSerial creates a new serial port

func NewNet(ncnf NetInfo, pickerFn picker.PickerFunc, isDefault bool) (*TNet, error) {
	if pickerFn == nil {
		return nil, fmt.Errorf("user NewNet(), should provide a picker function")
	}
	ip := ncnf.Ip
	port := ncnf.Port
	var conn *net.TCPConn = nil
	connected := false
	tnet := &TNet{
		port:       port,
		ip:         ip,
		conn:       conn,
		sendCh:     make(chan []byte, SEND_CH_SIZE),
		recvCh:     make(chan comm.Packet, RECV_CH_SIZE),
		queue:      util.NewCircularBuffer(QUEUE_SIZE),
		pickerFn:   pickerFn,
		tmpbuf:     make([]byte, TMP_BUF_SIZE),
		toQuit:     false,
		isAlive:    connected,
		isDefault:  isDefault,
		timeOutCnt: 0,
	}

	go tnet.write()
	go tnet.read()

	return tnet, nil
}

func (tnet *TNet) Close() error {
	// inform read/write goroutines to quit
	tnet.toQuit = true
	println("toQuit==============")
	// time.Sleep(100000 * time.Millisecond) // to let goroutines run
	time.Sleep(100 * time.Millisecond) // to let goroutines run
	if tnet.conn != nil {
		err := tnet.conn.Close()
		if err != nil {
			return fmt.Errorf("tnet.conn closed error")
		}
	}
	fmt.Println("关闭 conn")
	tnet.wg.Wait()
	fmt.Println("关闭 read")
	if !IsClosed(tnet.sendCh) {
		close(tnet.sendCh)
	}
	if !IsPacketChClosed(tnet.recvCh) {
		close(tnet.recvCh)
	}
	if !tnet.toQuit {
		tnet.toQuit = true
	}
	if tnet.conn == nil {
		return fmt.Errorf("tnet.conn is nil")
	}
	log.Log.Info("close tcp connect")
	time.Sleep(50 * time.Millisecond) // to let tnet closed
	return nil
}

func (tnet *TNet) Write(data []byte) error {
	if len(tnet.sendCh) >= SEND_CH_SIZE {
		log.Log.Error("sendCh is full")
		return util.ErrFull
	}
	tnet.sendCh <- data
	return nil
}

// read messages from the scale and parses them into data record or command response
// The application runs read in a per-scale goroutine. The application
// ensures that there is at most one reader on a scale by executing all
// reads from this goroutine.
// func (tnet *TNet) read() {
// 	tnet.wg.Add(1)
// 	packCnt := 0

// 	for {
// 		if tnet.toQuit {
// 			break // quit immediately
// 		}

// 		if tnet.conn == nil {
// 			time.Sleep(10 * time.Millisecond) // to avoid consume too much cpu time
// 			continue
// 		}

// 		// read data from net at least PACK_MIN_LEN or timeout (2 * 1/baud)
// 		if n, err := tnet.readScale(); err != nil { // data will be stored in the queue
// 			log.Log.Errorf("@TNet read(), err: %v\n", err)
// 			if tnet.toQuit {
// 				continue
// 			}

// 			if err.Error() != "EOF" {
// 				println("断开连接，重新连")
// 				tnet.conn.Close()
// 				tnet.conn = nil
// 				continue
// 			}

// 			if err.Error() == "EOF" { // 20240801
// 				log.Log.Errorf("EOF")
// 				//20250901 断开TCP，重新连接
// 				if tnet.conn != nil {
// 					tnet.conn.Close()
// 					tnet.reconnect()
// 				}
// 				continue
// 			}
// 			if !IsPacketChClosed(tnet.recvCh) {
// 				// tnet.recvCh <- comm.Packet{PayloadLen: uint16(len(RESP_SERIAL_ERROR)), CmdID: 0, CmdSubId: 0, SeqNum: 0, Payload: RESP_SERIAL_ERROR}
// 			}
// 			time.Sleep(100 * time.Millisecond) // to avoid sending error too often to UI
// 			continue
// 		} else if n == 0 {
// 			time.Sleep(1 * time.Millisecond) // to avoid consume too much cpu time
// 		}
// 		hasPack := true
// 		for hasPack {
// 			if tnet.queue.GetDataLen() > MIN_PACK_SIZE {
// 				// call the packet picker function
// 				data := tnet.queue.PeekAll()
// 				_, packLen, removeLen, pack := tnet.pickerFn(data, tnet.queue.GetDataLen())
// 				if packLen > 0 {
// 					if len(tnet.recvCh) >= RECV_CH_SIZE {
// 						log.Log.Errorf("recvCh full, size: %v", len(tnet.recvCh))
// 						fmt.Printf("recvCh full, size: %v", len(tnet.recvCh))
// 					} else {
// 						// fmt.Printf("pack data: %v\n", pack.Payload)
// 						tnet.recvCh <- pack

// 					}
// 					packCnt++
// 				} else {
// 					fmt.Printf("no packet\n")
// 					hasPack = false
// 				}
// 				if removeLen < packLen {
// 					fmt.Printf("removelen error\n")
// 				}
// 				if removeLen > 0 {
// 					tnet.queue.DequeueN(int(removeLen))
// 				}
// 			} else {
// 				hasPack = false
// 			}

// 		}

// 	}
// 	tnet.wg.Done()

// }

//20260309 修复网络突然断开后，不能正常工作

func (tnet *TNet) read() {
	tnet.wg.Add(1)
	defer tnet.wg.Done()

	packCnt := 0

	for {
		if tnet.toQuit {
			break
		}

		if tnet.conn == nil {
			time.Sleep(1000 * time.Millisecond) // to avoid consume too much cpu time
			// 重连成功后继续循环
			continue
		}

		// 读取数据
		n, err := tnet.readScale()
		if err != nil {
			log.Log.Errorf("@TNet read(), err: %v", err)

			// 检查是否需要退出
			if tnet.toQuit {
				continue
			}

			// [优化] 统一处理所有异常：只要报错就认为断开了（readScale 内部已过滤掉正常的 200ms 超时）
			log.Log.Warn("Detected network fatal error, performing force disconnect...")
			tnet.forceDisconnect()
			tnet.conn = nil
			tnet.isAlive = false
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if n == 0 {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		// 处理接收到的数据包
		hasPack := true
		for hasPack {
			if tnet.queue.GetDataLen() > MIN_PACK_SIZE {
				// call the packet picker function
				data := tnet.queue.PeekAll()
				var packLen, removeLen uint
				var pack comm.Packet

				if tnet.IsModbusTCP {
					// Read MBAP: Transaction ID(2), Protocol ID(2), Length(2)
					if len(data) >= 7 { // MBAP(6) + at least 1 byte Unit ID
						mbapLen := int(data[4])<<8 | int(data[5])
						totalLen := 6 + mbapLen
						
						// Check if Protocol ID is 0x0000 (Modbus)
						if data[2] != 0 || data[3] != 0 {
							// Invalid MBAP, discard 1 byte and re-sync
							packLen = 0
							removeLen = 1
						} else if len(data) >= totalLen {
							// We have a full Modbus TCP frame
							payload := data[6:totalLen]
							
							// Reconstruct RTU by appending CRC
							rtuFrame := make([]byte, len(payload)+2)
							copy(rtuFrame, payload)
							crc := calculateModbusCRC16(payload)
							rtuFrame[len(payload)] = byte(crc & 0xFF)
							rtuFrame[len(payload)+1] = byte((crc >> 8) & 0xFF)
							
							_, packLenRtu, _, packParsed := tnet.pickerFn(rtuFrame, len(rtuFrame))
							if packLenRtu > 0 {
								packLen = uint(packLenRtu)
								removeLen = uint(totalLen) // Remove the whole TCP frame
								pack = packParsed
							} else {
								// Picker failed to parse? Shouldn't happen with valid RTU.
								removeLen = uint(totalLen)
							}
						} else {
							packLen = 0
							removeLen = 0
						}
					} else {
						packLen = 0
						removeLen = 0
					}
				} else {
					_, packLen, removeLen, pack = tnet.pickerFn(data, tnet.queue.GetDataLen())
				}

				if packLen > 0 {
					if len(tnet.recvCh) >= RECV_CH_SIZE {
						log.Log.Errorf("recvCh full, size: %v", len(tnet.recvCh))
						fmt.Printf("recvCh full, size: %v", len(tnet.recvCh))
					} else {
						// fmt.Printf("pack data: %v\n", pack.Payload)
						tnet.recvCh <- pack

					}
					packCnt++
				} else {
					fmt.Printf("no packet\n")
					hasPack = false
				}
				if removeLen < packLen {
					fmt.Printf("removelen error\n")
				}
				if removeLen > 0 {
					tnet.queue.DequeueN(int(removeLen))
				}
			} else {
				hasPack = false
			}

		}
	}
}

// 辅助函数：判断是否为网络错误
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "wsarecv") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "reset by peer") ||
		strings.Contains(errStr, "use of closed network connection")
}

func isNofError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "EOF")
}

// if n > 0 {
// 	hexData := hex.EncodeToString(tnet.tmpbuf[:n])
// 	fmt.Printf("Received HEX data: %s\n", hexData)
// }

func (tnet *TNet) readScale() (int, error) {
	if !tnet.isAlive {
		time.Sleep(1 * time.Second)
		return 0, nil //关闭了会报错
	}
	if tnet.toQuit {
		time.Sleep(1 * time.Second)
		return 0, nil //关闭了会报错

	}

	// 设置2秒读取超时，从而更快地检测到断开
	// [最大稳定优化] 放弃短周期感应检测，回归最传统的阻塞读取和致命错误监测。
	// 这能确保在不发送业务心跳时，不会因为忙碌或微小延迟导致跳变。
	err := tnet.conn.SetReadDeadline(time.Now().Add(300 * time.Second))
	if err != nil {
		log.Log.Errorf("SetReadDeadline error: %v", err)
	}

	n, err := tnet.conn.Read(tnet.tmpbuf)

	if err != nil {
		// 检查是否是超时错误 (由 300s 触发)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// 超时时间极长，通常不需要处理
			return 0, nil 
		}
		// 只有遇到真正的致命底层故障（如：Connection reset, EOF）时才上报错误
		return 0, err
	}
	// 只要收到了数据，重置计数器（虽然逻辑上已不再主动使用计数器做断开判定）
	tnet.timeOutCnt = 0

	// if err != nil {
	// 	log.Log.Errorf("Error reading scale: %v", err)
	// 	tnet.isAlive = false
	// 	return 0, err
	// }

	if tnet.queue.IsFull() {
		tnet.queue.DequeueN(tnet.queue.Capacity) // handle abnormal case
	}

	if n > 0 {
		// log.Log.Debug(tnet.tmpbuf[0:n])
		fmt.Printf("netrevdata:%x\n", string(tnet.tmpbuf[0:n]))
		fmt.Printf("netrevdata:%s\n", string(tnet.tmpbuf[0:n]))
		if err := tnet.queue.EnqueueN(tnet.tmpbuf[0:n], n); err != nil {
			tnet.queue.Reset()
		}
	}

	return n, nil
}

// A goroutine running write is started for the scale. The
// func (tnet *TNet) write() {
// 	for message := range tnet.sendCh {
// 		// send message to scale
// 		if tnet.conn != nil {
// 			n, err := tnet.conn.Write([]byte(message))

// 			if err != nil || n != len(message) {
// 				tnet.isAlive = false
// 				log.Log.Error(fmt.Sprintf("Error on sending message to scale, to send: %v, sent:%v, err:%v\n", len(message), n, err.Error()))

// 			}
// 			time.Sleep(10 * time.Millisecond)
// 		}
// 	}
// }

func (tnet *TNet) write() {
	for message := range tnet.sendCh {
		// 检查连接状态
		if tnet.conn == nil || !tnet.isAlive {
			time.Sleep(100 * time.Millisecond)
			continue

		}

		// 设置写入超时
		if err := tnet.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Log.Error(fmt.Sprintf("Set write deadline error: %v", err))
		}

		sendData := []byte(message)
		if tnet.IsModbusTCP && len(sendData) >= 4 {
			// Strips 2 bytes CRC
			payload := sendData[:len(sendData)-2]
			payloadLen := len(payload)
			
			tid := atomic.AddUint32(&tnet.transactionID, 1)
			header := make([]byte, 6)
			header[0] = byte((tid >> 8) & 0xFF)
			header[1] = byte(tid & 0xFF)
			header[2] = 0x00
			header[3] = 0x00
			header[4] = byte((payloadLen >> 8) & 0xFF)
			header[5] = byte(payloadLen & 0xFF)
			
			sendData = append(header, payload...)
		}

		n, err := tnet.conn.Write(sendData)
		if err != nil {
			// [最终稳定优化] 写入报错时的分级处理
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 判定为“写超时”。这通常不是链路断了，而是秤端忙着解析指令或网络拥塞
				// 此时绝对不能销毁连接，否则会触发无限重连死循环
				log.Log.Trace(fmt.Sprintf("Write timeout (5s), scale busy? IP: %v", tnet.ip))
				// 避让 100ms 再试下一个指令包
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// 只有真正的致命链路错误才断线
			log.Log.Errorf("Fatal write error: %v, IP: %v, message length: %v, sent: %v",
				err.Error(), tnet.ip, len(sendData), n)

			tnet.isAlive = false
			if tnet.conn != nil {
				tnet.conn.Close()
				tnet.conn = nil
			}

			// 延迟一秒给底层网络栈留出重平衡时间
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		if n != len(sendData) {
			log.Log.Warn(fmt.Sprintf("Partial write: %v of %v bytes", n, len(sendData)))
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (tnet *TNet) ChangePickFunc(pickerFn picker.PickerFunc) {
	tnet.pickerFn = pickerFn
	//TODO:  修改网络信息  待写
}

// func (tnet *TNet) monitorConnection() {
// 	ticker := time.NewTicker(2 * time.Second)
// 	for range ticker.C {
// 		if tnet.conn == nil || !tnet.isAlive {
// 			// 尝试重连
// 			var err error
// 			tnet.conn, err = tnet.reconnect()
// 			if err != nil {
// 				log.Log.Printf("Reconnection failed: %v", err)
// 			}
// 		} else {
// 			fmt.Println("Connection is alive")
// 		}
// 	}
// }

func (tnet *TNet) reconnect() (*net.TCPConn, error) {

	conn, err := tnet.connect()
	if err == nil {
		tnet.conn = conn
		tnet.isAlive = true
	}
	println("reconnect")
	println(tnet.ip)

	return conn, err
}

func (tnet *TNet) connect() (*net.TCPConn, error) {

	conn, err := tnet.dialTCP()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (tnet *TNet) dialTCP() (*net.TCPConn, error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", tnet.ip+":"+strconv.Itoa(tnet.port))
	if err != nil {
		log.Log.Printf("ResolveTCPAddr error: %v", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	// conn, err := net.Dial("tcp", tnet.ip+":"+strconv.Itoa(tnet.port))

	if err != nil {
		return nil, err
	}
	// 采用系统默认的 KeepAlive 设置，不进行高频探测。
	conn.SetKeepAlive(true)

	return conn, nil

}

func (tnet *TNet) forceDisconnect() {

	log.Log.Warn("正在强制断开连接...")

	if tnet.conn != nil {
		// 1. 先设置SO_LINGER，确保彻底关闭

		// 2. 关闭连接
		tnet.conn.Close()

		// 3. 设置为nil，确保不会再使用
		tnet.conn = nil
	}

	// 4. 重置所有状态
	tnet.isAlive = false

	// 5. 清空缓冲区
	if tnet.queue != nil {
		tnet.queue.Reset()
	}

	log.Log.Info("连接已彻底断开")
}

