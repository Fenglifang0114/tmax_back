package svc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"

	m "tmaxsrv/comm"
	l "tmaxsrv/log"
)

var EnableAutoScan = true
var AutoScanWakeup = make(chan bool, 1)

var portCooldowns = make(map[string]time.Time)
var cooldownMutex sync.Mutex

// loadAutoScanConfig reads the auto-scan setting from disk
func loadAutoScanConfig() {
	path := filepath.Join(m.GetSrvDataPath(), "autoscan.json")
	data, err := os.ReadFile(path)
	if err == nil {
		if strings.Contains(string(data), "false") {
			EnableAutoScan = false
		} else {
			EnableAutoScan = true
		}
	}
}

// SaveAutoScanConfig saves the auto-scan setting to disk
func SaveAutoScanConfig(enable bool) {
	EnableAutoScan = enable
	path := filepath.Join(m.GetSrvDataPath(), "autoscan.json")
	val := "true"
	if !enable {
		val = "false"
	}
	os.WriteFile(path, []byte(fmt.Sprintf(`{"enable": %s}`, val)), 0644)

	if enable {
		select {
		case AutoScanWakeup <- true:
		default:
		}
	}
}

// StartAutoMountTask starts a background goroutine to periodically scan and auto-mount scales.
func StartAutoMountTask(s *ScaleMgr) {
	loadAutoScanConfig()
	go autoMountLoop(s)
}

func autoMountLoop(s *ScaleMgr) {
	// Wait a bit before starting the first scan
	time.Sleep(5 * time.Second)

	for {
		if !EnableAutoScan {
			select {
			case <-time.After(10 * time.Second):
			case <-AutoScanWakeup:
			}
			continue
		}

		ports, err := getPortsList()
		if err != nil {
			l.Log.Errorf("auto_mount: failed to get ports list: %v", err)
			select {
			case <-time.After(10 * time.Second):
			case <-AutoScanWakeup:
			}
			continue
		}

		l.Log.Infof("auto_mount: active ports found: %v", ports)

		// Find occupied ports
		occupiedPorts := make(map[string]bool)
		for _, conn := range s.medias {
			if conn.TMedia == MEDIA_COM {
				var comInfo ComInfo
				if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &comInfo); err == nil {
					occupiedPorts[comInfo.DevPath] = true
				}
			}
		}

		// Check unoccupied ports
		for _, port := range ports {
			if occupiedPorts[port] {
				l.Log.Infof("auto_mount: skipping occupied port %s", port)
				continue
			}

			cooldownMutex.Lock()
			cooldownEnd, hasCooldown := portCooldowns[port]
			if hasCooldown && time.Now().Before(cooldownEnd) {
				cooldownMutex.Unlock()
				continue
			}
			portCooldowns[port] = time.Now().Add(120 * time.Second)
			cooldownMutex.Unlock()

			l.Log.Infof("auto_mount: scheduling test for unoccupied port %s", port)
			go testAndMountPort(s, port)
		}

		// Scan every 10 seconds or until woken up
		select {
		case <-time.After(10 * time.Second):
		case <-AutoScanWakeup:
		}
	}
}

func testAndMountPort(s *ScaleMgr, port string) {
	if !EnableAutoScan {
		return
	}

	baudRates := []int{115200, 9600}

	for i, baud := range baudRates {
		if !EnableAutoScan {
			l.Log.Infof("auto_mount: auto-scan disabled, aborting test for %s", port)
			return
		}

		if i > 0 {
			// Give OS time to release the serial port lock from previous baud rate attempt
			for j := 0; j < 5; j++ {
				if !EnableAutoScan {
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		isFound, protocol, model, sn := tryPortAtBaud(s, port, baud)
		if !EnableAutoScan {
			l.Log.Infof("auto_mount: auto-scan disabled during test, aborting mount for %s", port)
			return
		}

		if isFound {
			l.Log.Infof("auto_mount: SUCCESS found %s scale on %s (baud: %d, model: %s, sn: %s)", protocol, port, baud, model, sn)

			// Auto mount the scale
			var comInfo ComInfo = ComInfo{DevPath: port, Baud: baud, DataBits: 8, Parity: 0, StopBits: 0}
			var conf MediaConf = MediaConf{}
			conf.Type = MEDIA_COM
			conf.MediaInfoJson, _ = json.MarshalToString(comInfo)

			// Find next available Modbus ID
			nextModbusId := 1
			if s != nil {
				s.mu.Lock()
				usedIds := make(map[int]bool)
				for _, sc := range s.scales {
					if sc != nil && sc.Conn != nil {
						usedIds[sc.Conn.ModbusId] = true
					}
				}
				s.mu.Unlock()

				for usedIds[nextModbusId] {
					nextModbusId++
				}
			}

			req := ReqAddScale{
				ScaleModel: model,
				ScaleSn:    sn,
				MediaConf:  conf,
				ModbusId:   nextModbusId,
			}

			// This will also trigger the UI update
			scaleAdded.Trigger(req)
			l.Log.Infof("auto_mount: UI update triggered for %s", port)
			return // Done testing this port
		}
	}
	l.Log.Infof("auto_mount: test completed for %s, no scale found", port)
}

func tryPortAtBaud(s *ScaleMgr, port string, baud int) (bool, string, string, string) {
	l.Log.Infof("auto_mount: trying port %s at baud %d", port, baud)
	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	p, err := serial.Open(port, mode)
	if err != nil {
		l.Log.Errorf("auto_mount: failed to open port %s at baud %d: %v", port, baud, err)
		return false, "", "", ""
	}
	defer p.Close()

	p.SetReadTimeout(time.Millisecond * 500)
	p.SetDTR(true)
	p.SetRTS(true)
	p.ResetInputBuffer()
	p.ResetOutputBuffer()

	// CRITICAL FIX: Give the hardware time to stabilize!
	// Opening a serial port often toggles DTR/RTS which resets the MCU.
	// We wait 1.5 seconds to be absolutely sure the scale has booted up.
	for i := 0; i < 15; i++ {
		if !EnableAutoScan {
			return false, "", "", ""
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Try TMax protocol
	composer := cmdComposerFuncMap[m.SCALE_TMAX]
	cmdBytes, _, err := composer.ComposeCmd(&composer, m.CMD_GET_FACTORY_INFO, m.CmdData{})

	if err == nil {
		for retry := 0; retry < 3; retry++ {
			p.Write(cmdBytes)
			l.Log.Infof("auto_mount: sent %X to %s at %d (attempt %d/3)", cmdBytes, port, baud, retry+1)

			var fullBuf []byte
			buf := make([]byte, 1024)
			startRead := time.Now()
			// scale responds very quickly (typically < 100ms), but we wait up to 5 seconds per user request to ensure recovery from wrong baud rate garbage
			for time.Since(startRead) < 5*time.Second {
				if !EnableAutoScan {
					return false, "", "", ""
				}
				n, _ := p.Read(buf)
				if n > 0 {
					fullBuf = append(fullBuf, buf[:n]...)

					idx := bytes.Index(fullBuf, []byte{0x5A, 0xA5})
					if idx >= 0 && len(fullBuf) >= idx+13 {
						l.Log.Infof("auto_mount: found TMax packet header on %s: %X", port, fullBuf[idx:])

						// Try to parse model/sn if it happens to be a GET_FACTORY_INFO response
						if idx+31 <= len(fullBuf) {
							payload := fullBuf[idx+7 : idx+31]
							resp, _ := handleGetFactoryInfoResp(0, payload)
							if resp.MsgType == m.GET_FACTORY_INFO_RESP && resp.MsgBody != "fail" && resp.MsgBody != nil {
								var factoryInfo FIFromScale
								if err := json.UnmarshalFromString(fmt.Sprintf("%v", resp.MsgBody), &factoryInfo); err == nil {
									if factoryInfo.ModelName != "" {
										l.Log.Infof("auto_mount: TMax successfully parsed: %s, %s", factoryInfo.ModelName, factoryInfo.ScaleSn)
										return true, "TMax", factoryInfo.ModelName, factoryInfo.ScaleSn
									}
								}
							}
						}

						// If we reach here, we found a TMax packet (like a weight response) but not factory info.
						l.Log.Infof("auto_mount: Identified TMax scale on %s without factory info", port)
						return true, "TMax", "TMAX", ""
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	} else {
		l.Log.Errorf("auto_mount: TMax compose error: %v", err)
	}

	// Try DC500 continuous data
	l.Log.Infof("auto_mount: trying DC500 continuous data on %s at %d", port, baud)
	start := time.Now()
	dataStr := ""
	buf := make([]byte, 256)

	// Read for 1.5 seconds max
	for time.Since(start) < 1500*time.Millisecond {
		if !EnableAutoScan {
			return false, "", "", ""
		}
		n, _ := p.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			dataStr += chunk
			l.Log.Infof("auto_mount: [DEBUG] DC500 chunk received: %q", chunk)

			// Check if we have complete lines
			lines := strings.Split(dataStr, "\n")
			for i := 0; i < len(lines)-1; i++ {
				line := strings.TrimSpace(lines[i])
				l.Log.Infof("auto_mount: [DEBUG] DC500 parsed line: %q", line)

				if strings.Contains(line, "ST") || strings.Contains(line, "UL") || strings.Contains(line, "OL") || strings.Contains(line, "ZE") {
					_, err := retrieveWeight([]byte(line))
					if err == nil {
						l.Log.Infof("auto_mount: DC500 valid data found on %s", port)
						return true, "DC500", "DC500", ""
					} else {
						l.Log.Errorf("auto_mount: [DEBUG] retrieveWeight failed for %q: %v", line, err)
					}
				}
			}

			// Keep only the last incomplete line
			if len(lines) > 0 {
				dataStr = lines[len(lines)-1]
			}
		}
	}

	return false, "", "", ""
}
