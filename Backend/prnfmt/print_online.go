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
	Protocol   string `json:"protocol"`
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

	// 1. Generate printer template buffer using existing parser logic
	// This function populates prnfmt.VarList with the variables
	// Use 2048 or a sufficiently large length for fmtLen
	templateBuffer := ParserFmtToBuf(req.FormatCsv, req.Protocol, 2048)

	// 3. Inject Variables into the buffer
	// Need to safely copy VarList because it's a global variable and might be overwritten
	currentVars := make([]VarStruct, len(VarList))
	copy(currentVars, VarList)

	finalBytes := injectVariablesToBuffer(templateBuffer.Bytes(), currentVars, req)

	// 4. Send to serial port
	err = sendToSerial(req.SerialPort, req.BaudRate, finalBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send to serial port: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func injectVariablesToBuffer(template []byte, vars []VarStruct, req PrintOnlineRequest) []byte {
	// We operate on a copy of the template so we don't modify the original (though in this case template is one-off)
	result := make([]byte, len(template))
	copy(result, template)

	// Pre-calculate values
	netWeight := req.WeightVal
	grossWeight := req.WeightVal // Assuming gross is net if Tare is 0
	// You might want to parse and calculate based on IsNet if you had Tare.

	for _, v := range vars {
		var valStr string
		switch v.id {
		case 8: // Gross
			valStr = grossWeight
		case 10: // Net
			valStr = netWeight
		case 15: // WeightUnit
			valStr = req.WeightUnit
		default:
			// Optionally handle more IDs (Date, Time, etc.)
			// Using blank for unsupported/unused variables to clean up output
			valStr = ""
		}

		// Pad or truncate string to maxlen
		if int(v.maxlen) > 0 {
			if len(valStr) > int(v.maxlen) {
				valStr = valStr[:v.maxlen]
			} else {
				// Pad according to alignment
				padLen := int(v.maxlen) - len(valStr)
				if v.align == 0 { // Left align
					valStr = valStr + strings.Repeat(" ", padLen)
				} else if v.align == 1 { // Right align
					valStr = strings.Repeat(" ", padLen) + valStr
				} else { // Center or default to left
					valStr = valStr + strings.Repeat(" ", padLen)
				}
			}
		}

		// Inject into result byte array
		// Make sure startPos and endPos are within bounds
		if int(v.startPos) < len(result) && int(v.endPos) <= len(result) && v.startPos < v.endPos {
			valBytes := []byte(valStr)
			// Truncate valBytes to fit exactly into (endPos - startPos)
			availableSpace := int(v.endPos - v.startPos)
			if len(valBytes) > availableSpace {
				valBytes = valBytes[:availableSpace]
			}

			// Copy injected bytes
			for i := 0; i < len(valBytes); i++ {
				result[int(v.startPos)+i] = valBytes[i]
			}
			// Pad the rest of the available space with spaces
			for i := len(valBytes); i < availableSpace; i++ {
				result[int(v.startPos)+i] = ' '
			}
		}
	}
	return result
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
