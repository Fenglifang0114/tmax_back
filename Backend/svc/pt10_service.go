package svc

import (
	"bytes"
	"fmt"
	"sync"
	"time"
	"tmaxsrv/cmd"

	"go.bug.st/serial"
)

var pt10Mutex sync.Mutex

// PT10SendAndReceive opens serial port, sends request frame, and waits for AF...0D0A response.
func PT10SendAndReceive(portName string, baudRate int, reqFrame []byte, timeout time.Duration) (byte, []byte, error) {
	pt10Mutex.Lock()
	defer pt10Mutex.Unlock()

	if portName == "" {
		portName = "COM1"
	}
	if baudRate <= 0 {
		baudRate = 9600
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to open serial port %s: %v", portName, err)
	}
	defer port.Close()

	// Wait 1000ms for the printer hardware to stabilize/boot up after port opening (DTR/RTS reset)
	time.Sleep(1000 * time.Millisecond)

	if err := port.SetReadTimeout(20 * time.Millisecond); err != nil {
		return 0, nil, fmt.Errorf("failed to set read timeout: %v", err)
	}

	// Flush any leftover buffer without blocking
	_ = port.ResetInputBuffer()

	// Write request frame
	fmt.Printf("[PT10 Serial TX] Port: %s, BaudRate: %d, Frame: %X\n", portName, baudRate, reqFrame)
	_, err = port.Write(reqFrame)
	if err != nil {
		fmt.Printf("[PT10 Serial TX Error] %v\n", err)
		return 0, nil, fmt.Errorf("failed to write to serial port: %v", err)
	}

	// Read response loop
	start := time.Now()
	var rxBuf []byte
	readBuf := make([]byte, 256)

	for time.Since(start) < timeout {
		n, err := port.Read(readBuf)
		if n > 0 {
			fmt.Printf("[PT10 Serial RX Chunk] %X\n", readBuf[:n])
			rxBuf = append(rxBuf, readBuf[:n]...)
			// Check if response contains AF header and 0D 0A tail
			if bytes.Contains(rxBuf, []byte{cmd.PT10_RESP_HEADER}) && bytes.Contains(rxBuf, cmd.PT10_FRAME_TAIL) {
				cmdType, payload, parseErr := cmd.ParsePT10Response(rxBuf)
				if parseErr == nil {
					fmt.Printf("[PT10 Serial RX Full Success] %X\n", rxBuf)
					return cmdType, payload, nil
				}
			}
		}
		// If read failed, sleep briefly and keep trying until overall timeout
		_ = err
		time.Sleep(10 * time.Millisecond)
	}

	if len(rxBuf) > 0 {
		cmdType, payload, parseErr := cmd.ParsePT10Response(rxBuf)
		if parseErr == nil {
			fmt.Printf("[PT10 Serial RX Full Success (Post-Loop)] %X\n", rxBuf)
			return cmdType, payload, nil
		}
	}

	fmt.Printf("[PT10 Serial RX Timeout Error] Timeout reached. Full rxBuf: %X\n", rxBuf)
	return 0, nil, fmt.Errorf("timeout waiting for PT10 printer response")
}

// PT10TestConnection sends 0x10 ping test to check printer connectivity
func PT10TestConnection(portName string, baudRate int) (bool, error) {
	req := cmd.BuildPT10PingFrame()
	cmdType, payload, err := PT10SendAndReceive(portName, baudRate, req, 2*time.Second)
	if err != nil {
		return false, err
	}
	if cmdType == cmd.PT10_CMD_PING {
		if len(payload) > 0 && payload[0] == 0x00 {
			return true, nil
		}
		return true, nil // Response received successfully
	}
	return false, fmt.Errorf("unexpected ping response cmd: 0x%X", cmdType)
}

// PT10ReadAllParams sends 0x20 query frame and returns parsed PT10Params
func PT10ReadAllParams(portName string, baudRate int) (*cmd.PT10Params, error) {
	req := cmd.BuildPT10QueryFrame()
	cmdType, payload, err := PT10SendAndReceive(portName, baudRate, req, 3*time.Second)
	if err != nil {
		return nil, err
	}
	if cmdType != cmd.PT10_CMD_GET_PARAMS {
		return nil, fmt.Errorf("unexpected query response cmd: 0x%X", cmdType)
	}
	return cmd.ParsePT100x20Payload(payload)
}

// PT10WriteSingleParam sends parameter setting frame for a specific cmdType
func PT10WriteSingleParam(portName string, baudRate int, cmdType byte, val string) error {
	req, err := cmd.BuildPT10SetParamFrame(cmdType, val)
	if err != nil {
		return err
	}
	respCmd, payload, err := PT10SendAndReceive(portName, baudRate, req, 2*time.Second)
	if err != nil {
		return err
	}
	if respCmd != cmdType {
		return fmt.Errorf("unexpected setting response cmd: 0x%X (expected 0x%X)", respCmd, cmdType)
	}
	if len(payload) > 0 && payload[0] == 0x01 {
		return fmt.Errorf("printer returned failure status (0x01)")
	}
	return nil
}

// PT10WriteParams writes multiple parameters to the printer sequentially, ensuring baud rate is written last and using a single open port session (Write-Only)
func PT10WriteParams(portName string, baudRate int, params map[string]string) error {
	pt10Mutex.Lock()
	defer pt10Mutex.Unlock()

	if portName == "" {
		portName = "COM1"
	}
	if baudRate <= 0 {
		baudRate = 9600
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port %s: %v", portName, err)
	}
	defer port.Close()

	// Wait 1000ms for the printer hardware to stabilize/boot up after port opening (DTR/RTS reset)
	time.Sleep(1000 * time.Millisecond)

	// Discard any startup noise
	_ = port.ResetInputBuffer()

	// Helper function to send command without waiting for response
	sendOnly := func(reqFrame []byte) error {
		fmt.Printf("[PT10 Batch TX (Write-Only)] Frame: %X\n", reqFrame)
		_, err := port.Write(reqFrame)
		if err != nil {
			fmt.Printf("[PT10 Batch TX Error] %v\n", err)
			return fmt.Errorf("failed to write: %v", err)
		}
		// Sleep 50ms to allow the hardware UART module to fully transmit the frame
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	var baudRateVal string
	var hasBaudRate bool

	writeSingle := func(cmdType byte, val string) error {
		req, err := cmd.BuildPT10SetParamFrame(cmdType, val)
		if err != nil {
			return err
		}
		return sendOnly(req)
	}

	for k, v := range params {
		var cmdType byte
		_, err := fmt.Sscanf(k, "%d", &cmdType)
		if err != nil {
			_, err = fmt.Sscanf(k, "0x%X", &cmdType)
		}
		if err != nil {
			return fmt.Errorf("invalid param key format: %s", k)
		}

		if cmdType == cmd.PT10_CMD_BAUDRATE {
			baudRateVal = v
			hasBaudRate = true
			continue
		}

		err = writeSingle(cmdType, v)
		if err != nil {
			return fmt.Errorf("failed to write param %d (%s): %v", cmdType, v, err)
		}
	}

	if hasBaudRate {
		err := writeSingle(cmd.PT10_CMD_BAUDRATE, baudRateVal)
		if err != nil {
			return fmt.Errorf("failed to write baud rate param (%s): %v", baudRateVal, err)
		}
	}

	return nil
}
