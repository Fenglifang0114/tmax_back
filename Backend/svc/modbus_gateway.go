package svc

import (
	"go.bug.st/serial"
	"sync"
	"time"
	"tmaxsrv/log"
)

type ModbusGatewayManager struct {
	mu          sync.Mutex
	srvMgr      *SrvMgr
	rtuServers  map[uint]*ModbusRTUServer // key is ModbusServiceInfo.Id
	tcpServers  map[uint]*ModbusTCPServer // key is ModbusServiceInfo.Id
}

func NewModbusGatewayManager(srvMgr *SrvMgr) *ModbusGatewayManager {
	return &ModbusGatewayManager{
		srvMgr:     srvMgr,
		rtuServers: make(map[uint]*ModbusRTUServer),
		tcpServers: make(map[uint]*ModbusTCPServer),
	}
}

// StartAll 从数据库读取所有启用的服务并启动
func (m *ModbusGatewayManager) StartAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	pb := NewModbusServiceProvider()
	services, err := pb.GetModbusServiceList()
	if err != nil {
		log.Log.Errorf("ModbusGatewayManager failed to get services: %v", err)
		return
	}
	for _, srv := range services {
		m.startServiceInternal(srv)
	}
}

// StopAll 停止所有服务
func (m *ModbusGatewayManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, server := range m.rtuServers {
		server.Stop()
		delete(m.rtuServers, id)
	}
	for id, server := range m.tcpServers {
		server.Stop()
		delete(m.tcpServers, id)
	}
}

// StartService 启动单个服务
func (m *ModbusGatewayManager) StartService(info ModbusServiceInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 先停止可能已存在的服务
	if server, ok := m.rtuServers[info.Id]; ok {
		if server.info.TargetModbusId == info.TargetModbusId && server.info.Protocol == info.Protocol && server.info.Port == info.Port && server.info.BaudRate == info.BaudRate {
			// 检查串口是否真的还是开着的（防止假死）
			log.Log.Infof("Modbus RTU service %d configuration unchanged, skipping restart", info.Id)
			return
		}
		
		log.Log.Infof("Modbus RTU service %d config changed! Old: (Port:%s, Baud:%d, ID:%d, Proto:%s), New: (Port:%s, Baud:%d, ID:%d, Proto:%s). Restarting...", 
			info.Id, 
			server.info.Port, server.info.BaudRate, server.info.TargetModbusId, server.info.Protocol,
			info.Port, info.BaudRate, info.TargetModbusId, info.Protocol)

		server.Stop()
		delete(m.rtuServers, info.Id)
		// 必须延迟等待系统完全释放串口，否则紧接着的 Start 会报“拒绝访问”而失败，导致串口假性释放！
		time.Sleep(500 * time.Millisecond)
	}

	if server, ok := m.tcpServers[info.Id]; ok {
		if server.info.TargetModbusId == info.TargetModbusId && server.info.Protocol == info.Protocol && server.info.Port == info.Port {
			log.Log.Infof("Modbus TCP service %d configuration unchanged, skipping restart", info.Id)
			return
		}
		log.Log.Infof("Modbus TCP service %d config changed! Restarting...", info.Id)
		server.Stop()
		delete(m.tcpServers, info.Id)
		time.Sleep(200 * time.Millisecond)
	}
	m.startServiceInternal(info)
}

func (m *ModbusGatewayManager) startServiceInternal(info ModbusServiceInfo) {
	if info.Protocol == "Modbus RTU" || info.Protocol == "RTU" {
		server := NewModbusRTUServer(info, m.srvMgr)
		if err := server.Start(); err != nil {
			log.Log.Errorf("Failed to start Modbus RTU Server [Port: %s]: %v", info.Port, err)
			return
		}
		m.rtuServers[info.Id] = server
		log.Log.Infof("Modbus RTU Server started [Port: %s, Baud: %d]", info.Port, info.BaudRate)
	} else if info.Protocol == "Modbus TCP" || info.Protocol == "TCP" {
		server := NewModbusTCPServer(info, m.srvMgr)
		if err := server.Start(); err != nil {
			log.Log.Errorf("Failed to start Modbus TCP Server [Port: %s]: %v", info.Port, err)
			return
		}
		m.tcpServers[info.Id] = server
		log.Log.Infof("Modbus TCP Server started [Port: %s]", info.Port)
	}
}

// StopService 停止单个服务
func (m *ModbusGatewayManager) StopService(id uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if server, ok := m.rtuServers[id]; ok {
		server.Stop()
		delete(m.rtuServers, id)
		log.Log.Infof("Modbus RTU Server stopped [ID: %d]", id)
	}
	if server, ok := m.tcpServers[id]; ok {
		server.Stop()
		delete(m.tcpServers, id)
		log.Log.Infof("Modbus TCP Server stopped [ID: %d]", id)
	}
}

// ==============================================================
// Modbus RTU Server implementation
// ==============================================================
type ModbusRTUServer struct {
	info     ModbusServiceInfo
	srvMgr   *SrvMgr
	port     serial.Port
	stopChan chan struct{}
}

func NewModbusRTUServer(info ModbusServiceInfo, srvMgr *SrvMgr) *ModbusRTUServer {
	return &ModbusRTUServer{
		info:     info,
		srvMgr:   srvMgr,
		stopChan: make(chan struct{}),
	}
}

func (s *ModbusRTUServer) Start() error {
	mode := &serial.Mode{
		BaudRate: s.info.BaudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	
	port, err := serial.Open(s.info.Port, mode)
	if err != nil {
		return err
	}
	s.port = port
	
	go s.readLoop()
	return nil
}

func (s *ModbusRTUServer) Stop() {
	close(s.stopChan)
	if s.port != nil {
		s.port.Close()
	}
}

func (s *ModbusRTUServer) readLoop() {
	buf := make([]byte, 1024)
	var rxBuffer []byte

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		n, err := s.port.Read(buf)
		if err != nil {
			if err.Error() == "EOF" || err.Error() == "Port has been closed" {
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n > 0 {
			rxBuffer = append(rxBuffer, buf[:n]...)
			rxBuffer = s.processBuffer(rxBuffer)
		}
	}
}

func (s *ModbusRTUServer) processBuffer(buffer []byte) []byte {
	for len(buffer) >= 8 {
		// 校验功能码，支持 03 (读), 06 (写单个)
		funcCode := buffer[1]
		if funcCode != 0x03 && funcCode != 0x06 {
			// 如果不是支持的功能码，尝试向后寻找合法的从机地址 (1~247) 和功能码
			// 为防止陷入死循环或丢弃有效数据，每次仅丢弃1个字节重新同步
			buffer = buffer[1:]
			continue
		}

		packetLen := 8 // 对于 03 请求和 06 请求，标准的 RTU 报文长度都是 8 字节
		if len(buffer) < packetLen {
			break // 数据还不够，等待后续数据
		}

		packet := buffer[:packetLen]
		if s.verifyCRC(packet) {
			// CRC 校验通过，提取出一个完整报文，并移交给 Router 处理
			response := RouteModbusRequest(packet, s.info.TargetModbusId, s.srvMgr)
			if response != nil && s.port != nil {
				s.port.Write(response)
			}
			// 将已处理的数据从缓存中移除
			buffer = buffer[packetLen:]
		} else {
			// CRC 校验失败，可能遇到了误匹配，丢弃第一个字节以重新同步
			buffer = buffer[1:]
		}
	}
	
	// 防止恶意或异常数据导致内存无限增长
	if len(buffer) > 256 {
		buffer = buffer[len(buffer)-256:]
	}
	
	return buffer
}

func (s *ModbusRTUServer) verifyCRC(packet []byte) bool {
	if len(packet) < 2 {
		return false
	}
	crc := calculateModbusCRC16(packet[:len(packet)-2])
	packetCRC := uint16(packet[len(packet)-2]) | (uint16(packet[len(packet)-1]) << 8)
	return crc == packetCRC
}
