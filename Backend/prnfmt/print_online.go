package prnfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"tmaxsrv/log"

	"go.bug.st/serial"
)

type PrintOnlineRequest struct {
	SerialPort string `json:"serialPort"`
	BaudRate   int    `json:"baudRate"`
	FormatCsv  string `json:"formatCsv"`
	WeightVal  string `json:"weightVal"`
	WeightUnit string `json:"weightUnit"`
	IsNet      bool   `json:"isNet"`
	IsZero     bool   `json:"isZero"`
}

func HandlePrintOnline(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}

	var req PrintOnlineRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// 1. 先将真实变量值直接预填充至 CSV 模板文本中
	updatedCsv := injectVariablesToCsv(req.FormatCsv, req)

	// 2. 解析预填充后的 CSV，根据 CSV 内含的 F,协议,模式 自动提取并生成完整无缺失的纯打印指令流
	templateBuffer, _ := ParserFmtToRawCmd(updatedCsv)

	finalBytes := templateBuffer.Bytes()

	log.Log.Infof("PrintOnline: sending %d raw command bytes to %s", len(finalBytes), req.SerialPort)

	// 3. 直接发送完整指令字节流至串口
	err = sendToSerial(req.SerialPort, req.BaudRate, finalBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send to serial port: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func injectVariablesToCsv(csvStr string, req PrintOnlineRequest) string {
	lines := strings.Split(csvStr, "\n")
	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		parts := strings.Split(trimmed, ",")

		for p := 0; p < len(parts)-1; p++ {
			if parts[p] == "DATA" {
				varName := parts[p+1]
				varVal := ""
				switch strings.ToLower(varName) {
				case "gross", "gross_weight":
					varVal = req.WeightVal
				case "net", "net_weight":
					varVal = req.WeightVal
				case "weightunit", "weight_unit", "unit":
					varVal = req.WeightUnit
				}

				if varVal != "" && p+2 < len(parts) {
					maxLen := 0
					if p+4 < len(parts) {
						fmt.Sscanf(parts[p+4], "%d", &maxLen)
					}
					if maxLen > 0 {
						if len(varVal) > maxLen {
							varVal = varVal[:maxLen]
						} else if len(varVal) < maxLen {
							align := 0
							if p+3 < len(parts) {
								fmt.Sscanf(parts[p+3], "%d", &align)
							}
							padLen := maxLen - len(varVal)
							if align == 1 { // Right align
								varVal = strings.Repeat(" ", padLen) + varVal
							} else {
								varVal = varVal + strings.Repeat(" ", padLen)
							}
						}
					}
					parts[p+2] = varVal
				}
			}
		}
		lines[i] = strings.Join(parts, ",")
	}
	return strings.Join(lines, "\n")
}

func sendToSerial(portName string, baudRate int, data []byte) error {
	log.Log.Infof("PrintOnline: Opening serial port %s at %d", portName, baudRate)
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	s, err := serial.Open(portName, mode)
	if err != nil {
		log.Log.Errorf("PrintOnline: failed to open port %s: %v", portName, err)
		return err
	}
	defer s.Close()

	n, err := s.Write(data)
	if err != nil {
		log.Log.Errorf("PrintOnline: failed to write to port %s: %v", portName, err)
		return err
	}
	log.Log.Infof("PrintOnline: Successfully sent %d bytes to %s", n, portName)
	return nil
}
