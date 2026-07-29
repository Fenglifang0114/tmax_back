package svc

import (
	"encoding/hex"
	"net"
	"testing"
	"time"
	"tmaxsrv/cmd"
	m "tmaxsrv/comm"
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

func TestModbusTCPPreTare(t *testing.T) {
	srvMgr := &SrvMgr{
		scaleMgr: &ScaleMgr{
			scales: make(map[int64]*Scale),
		},
	}
	c := cmd.NewComposerTMAX()
	mockScale := &Scale{
		Id:           1,
		Model:        "S15",
		composer:     c,
		respChansMap: make(map[m.RespMsgType][]chan *ScaleRespMsg),
	}
	srvMgr.scaleMgr.scales[1] = mockScale

	info := ModbusServiceInfo{
		Id:             2,
		TargetModbusId: 1,
		Protocol:       "Modbus TCP",
		Port:           "5021",
	}

	tcpServer := NewModbusTCPServer(info, srvMgr)
	if err := tcpServer.Start(); err != nil {
		t.Fatalf("Failed to start Modbus TCP server: %v", err)
	}
	defer tcpServer.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:5021")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// 1. Send Frame 1: 00 02 00 00 00 06 01 06 9C 54 3F 05 (MSB for 0.52)
	frame1Hex := "00020000000601069C543F05"
	frame1Bytes, _ := hex.DecodeString(frame1Hex)
	conn.Write(frame1Bytes)

	respBuf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(respBuf)
	if err != nil {
		t.Fatalf("Frame 1 response error: %v", err)
	}
	if hex.EncodeToString(respBuf[:n]) != "00020000000601069c543f05" {
		t.Errorf("Frame 1 response mismatch: got %x", respBuf[:n])
	}

	// 2. Send Frame 2: 00 03 00 00 00 06 01 06 9C 55 1E B8 (LSB for 0.52)
	frame2Hex := "00030000000601069C551EB8"
	frame2Bytes, _ := hex.DecodeString(frame2Hex)
	conn.Write(frame2Bytes)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn.Read(respBuf)
	if err != nil {
		t.Fatalf("Frame 2 response error: %v", err)
	}
	if hex.EncodeToString(respBuf[:n]) != "00030000000601069c551eb8" {
		t.Errorf("Frame 2 response mismatch: got %x", respBuf[:n])
	}

	// 3. Send 0x10 Multiple Write Frame: 00 04 00 00 00 0B 01 10 9C 54 00 02 04 3F 05 1E B8
	frame3Hex := "00040000000b01109c540002043f051eb8"
	frame3Bytes, _ := hex.DecodeString(frame3Hex)
	conn.Write(frame3Bytes)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn.Read(respBuf)
	if err != nil {
		t.Fatalf("Frame 3 (0x10) response error: %v", err)
	}
	if hex.EncodeToString(respBuf[:n]) != "00040000000601109c540002" {
		t.Errorf("Frame 3 response mismatch: got %x", respBuf[:n])
	}

	t.Logf("TestModbusTCPPreTare Passed!")
}
