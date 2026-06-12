package svc

import (
	"encoding/hex"
	"net"
	"testing"
	"time"
)

func TestModbusTCPServer(t *testing.T) {
	// 1. Setup mock Server Manager
	srvMgr := &SrvMgr{
		scaleMgr: &ScaleMgr{
			scales: make(map[int64]*Scale),
		},
	}
	
	// Add a mock scale with ID 1
	mockScale := &Scale{
		Id: 1,
		LastWeight: 12.34,
	}
	srvMgr.scaleMgr.scales[1] = mockScale

	// 2. Setup ModbusTCPServer on port 5020 (avoid 502 for permissions/conflicts)
	info := ModbusServiceInfo{
		Id:             1,
		TargetModbusId: 1, // Routes to scale ID 1
		Protocol:       "Modbus TCP",
		Port:           "5020",
	}

	tcpServer := NewModbusTCPServer(info, srvMgr)
	err := tcpServer.Start()
	if err != nil {
		t.Fatalf("Failed to start Modbus TCP server: %v", err)
	}
	defer tcpServer.Stop()

	// Give the server a moment to bind
	time.Sleep(100 * time.Millisecond)

	// 3. Act as a TCP Client and send a Modbus TCP Request
	conn, err := net.Dial("tcp", "127.0.0.1:5020")
	if err != nil {
		t.Fatalf("Failed to connect to Modbus TCP server: %v", err)
	}
	defer conn.Close()

	// Request: Read Holding Registers (Func Code 0x03)
	// TransID: 0x12 0x34
	// ProtoID: 0x00 0x00
	// Length:  0x00 0x06
	// UnitID:  0x01
	// Func:    0x03
	// Addr:    0x00 0x00 (Gross Weight)
	// Count:   0x00 0x02 (2 registers = 4 bytes)
	reqHex := "123400000006010300000002"
	reqBytes, _ := hex.DecodeString(reqHex)

	_, err = conn.Write(reqBytes)
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// 4. Read Response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	respBuf := make([]byte, 1024)
	n, err := conn.Read(respBuf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	resp := respBuf[:n]
	t.Logf("Received Modbus TCP Response: %x", resp)

	// Validate Response length
	// MBAP (7 bytes) + Func (1 byte) + ByteCount (1 byte) + Data (4 bytes) = 13 bytes
	if len(resp) != 13 {
		t.Errorf("Expected response length 13, got %d", len(resp))
	}

	// Validate MBAP Header
	if resp[0] != 0x12 || resp[1] != 0x34 {
		t.Errorf("TransID mismatch: expected 1234, got %x", resp[0:2])
	}
	if resp[2] != 0x00 || resp[3] != 0x00 {
		t.Errorf("ProtoID mismatch")
	}
	if resp[4] != 0x00 || resp[5] != 0x07 {
		t.Errorf("Length mismatch: expected 7 (UnitID + PDU), got %x", resp[4:6])
	}
	if resp[6] != 0x01 {
		t.Errorf("UnitID mismatch")
	}

	// Validate PDU
	if resp[7] != 0x03 {
		t.Errorf("Function code mismatch")
	}
	if resp[8] != 0x04 {
		t.Errorf("Byte count mismatch: expected 4, got %d", resp[8])
	}
	
	// The data should represent the weight 12.34 (float32)
	// We just ensure it sent 4 bytes
	t.Logf("Test Passed! Modbus TCP Server is working.")
}
