package svc

import (
	"net"
	"sync"
	"time"

	"tmaxsrv/log"
)

// ==============================================================
// Modbus TCP Server implementation
// ==============================================================
type ModbusTCPServer struct {
	info     ModbusServiceInfo
	srvMgr   *SrvMgr
	listener net.Listener
	stopChan chan struct{}
	conns    map[net.Conn]struct{}
	mu       sync.Mutex
}

func NewModbusTCPServer(info ModbusServiceInfo, srvMgr *SrvMgr) *ModbusTCPServer {
	return &ModbusTCPServer{
		info:     info,
		srvMgr:   srvMgr,
		stopChan: make(chan struct{}),
		conns:    make(map[net.Conn]struct{}),
	}
}

func (s *ModbusTCPServer) Start() error {
	l, err := net.Listen("tcp", "0.0.0.0:"+s.info.Port)
	if err != nil {
		return err
	}
	s.listener = l

	go s.acceptLoop()
	return nil
}

func (s *ModbusTCPServer) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.conns = make(map[net.Conn]struct{})
	s.mu.Unlock()
}

func (s *ModbusTCPServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return // Normal stop
			default:
				log.Log.Warnf("Modbus TCP Server Accept error: %v", err)
				return // Listener closed or error
			}
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		go s.handleConnection(conn)
	}
}

func (s *ModbusTCPServer) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	buf := make([]byte, 1024)
	var rxBuffer []byte

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return // Connection closed or other error
		}

		if n > 0 {
			rxBuffer = append(rxBuffer, buf[:n]...)
			rxBuffer = s.processBuffer(conn, rxBuffer)
		}
	}
}

func (s *ModbusTCPServer) processBuffer(conn net.Conn, buffer []byte) []byte {
	// Modbus TCP header is 7 bytes
	for len(buffer) >= 7 {
		transID := make([]byte, 2)
		copy(transID, buffer[0:2])
		protoID := make([]byte, 2)
		copy(protoID, buffer[2:4])
		
		length := int(buffer[4])<<8 | int(buffer[5])
		unitID := buffer[6]

		// For Modbus TCP, ProtoID must be 0x0000
		if protoID[0] != 0 || protoID[1] != 0 {
			// Invalid MBAP header, discard 1 byte and re-sync
			buffer = buffer[1:]
			continue
		}

		// length field includes unitID + PDU
		if len(buffer) < 6+length {
			break // Need more data
		}

		packetLen := 6 + length
		packet := buffer[:packetLen]

		// PDU is function code + data
		pdu := packet[7:packetLen]
		
		// Build fake RTU request for RouteModbusRequest: [UnitID] + [PDU...] + [DummyCRC(2 bytes)]
		// Because RouteModbusRequest checks if len(packet) < 8.
		fakeRtuReq := make([]byte, 1+len(pdu)+2)
		fakeRtuReq[0] = unitID
		copy(fakeRtuReq[1:], pdu)

		// Route to target scale
		responseRtu := RouteModbusRequest(fakeRtuReq, s.info.TargetModbusId, s.srvMgr)
		
		if responseRtu != nil {
			// Strip the 2-byte CRC from RTU response
			if len(responseRtu) >= 2 {
				responsePdu := responseRtu[1 : len(responseRtu)-2]
				
				// Build TCP response
				tcpRespLen := 1 + len(responsePdu)
				tcpResp := make([]byte, 6+tcpRespLen)
				copy(tcpResp[0:2], transID)
				copy(tcpResp[2:4], protoID)
				tcpResp[4] = byte(tcpRespLen >> 8)
				tcpResp[5] = byte(tcpRespLen & 0xFF)
				tcpResp[6] = unitID
				copy(tcpResp[7:], responsePdu)

				conn.Write(tcpResp)
			}
		}

		buffer = buffer[packetLen:]
	}

	if len(buffer) > 1024 {
		buffer = buffer[len(buffer)-1024:]
	}

	return buffer
}
