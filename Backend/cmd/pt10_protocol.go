package cmd

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// PT10Params holds all printer configuration parameter fields.
type PT10Params struct {
	Date      string `json:"date"`       // 0x01: YYYY-MM-DD
	Time      string `json:"time"`       // 0x02: HH:mm:ss
	BaudRate  int    `json:"baud_rate"`  // 0x03: 0=115200, 1=9600
	PaperType int    `json:"paper_type"` // 0x04: 0=Continuous, 1=Label
	PaperTake int    `json:"paper_take"` // 0x05: 0=Off, 1=On
	Speed     int    `json:"speed"`      // 0x06: 0=50mm/s, 1=75mm/s, 2=100mm/s, 3=125mm/s
	Density   int    `json:"density"`    // 0x07: 0=Pale, 1=Light, 2=Standard, 3=Medium, 4=Dark
	CoverFeed int    `json:"cover_feed"` // 0x08: 0=Off, 1=On
	PowerFeed int    `json:"power_feed"` // 0x09: 0=Off, 1=On
	Language  int    `json:"language"`   // 0x0A: 0=Traditional, 1=Simplified, 2=Multi
	AutoPaper int    `json:"auto_paper"` // 0x0B: 0=Off, 1=On
	Beep      int    `json:"beep"`       // 0x0C: 0=Off, 1=On
}

// Command IDs
const (
	PT10_CMD_DATE        byte = 0x01
	PT10_CMD_TIME        byte = 0x02
	PT10_CMD_BAUDRATE    byte = 0x03
	PT10_CMD_PAPER_TYPE  byte = 0x04
	PT10_CMD_PAPER_TAKE  byte = 0x05
	PT10_CMD_SPEED       byte = 0x06
	PT10_CMD_DENSITY     byte = 0x07
	PT10_CMD_COVER_FEED  byte = 0x08
	PT10_CMD_POWER_FEED  byte = 0x09
	PT10_CMD_LANGUAGE    byte = 0x0A
	PT10_CMD_AUTO_PAPER  byte = 0x0B
	PT10_CMD_BEEP        byte = 0x0C
	PT10_CMD_SELF_TEST   byte = 0x0F
	PT10_CMD_PING        byte = 0x10
	PT10_CMD_GET_PARAMS  byte = 0x20
)

// Frame Headers and Tails
var (
	PT10_REQ_HEADER  = []byte{0xFF, 0xFE}
	PT10_RESP_HEADER = byte(0xAF)
	PT10_FRAME_TAIL  = []byte{0x0D, 0x0A}
)

// BuildPT10Frame builds a complete command frame: FF FE [Cmd] [Len] [Payload] 0D 0A
func BuildPT10Frame(cmdType byte, payload []byte) []byte {
	buf := make([]byte, 0, 4+len(payload)+2)
	buf = append(buf, PT10_REQ_HEADER...)
	buf = append(buf, cmdType)
	buf = append(buf, byte(len(payload)))
	buf = append(buf, payload...)
	buf = append(buf, PT10_FRAME_TAIL...)
	return buf
}

// BuildPT10PingFrame returns frame for 0x10 ping test
func BuildPT10PingFrame() []byte {
	return BuildPT10Frame(PT10_CMD_PING, []byte{0x00})
}

// BuildPT10QueryFrame returns frame for 0x20 parameter query
func BuildPT10QueryFrame() []byte {
	return BuildPT10Frame(PT10_CMD_GET_PARAMS, []byte{0x00})
}

// BuildPT10SelfTestFrame returns frame for 0x0F self test page
func BuildPT10SelfTestFrame() []byte {
	return BuildPT10Frame(PT10_CMD_SELF_TEST, []byte{0x00})
}

// BuildPT10SetParamFrame builds individual parameter setting frame
func BuildPT10SetParamFrame(cmdType byte, val string) ([]byte, error) {
	var payload []byte
	switch cmdType {
	case PT10_CMD_DATE:
		// Expect YYYY-MM-DD or YYYYMMDD
		clean := strings.ReplaceAll(val, "-", "")
		if len(clean) != 8 {
			return nil, fmt.Errorf("invalid date format: %s", val)
		}
		payload = []byte(clean)
	case PT10_CMD_TIME:
		// Expect HH:mm:ss or HHMMSS
		clean := strings.ReplaceAll(val, ":", "")
		if len(clean) != 6 {
			return nil, fmt.Errorf("invalid time format: %s", val)
		}
		payload = []byte(clean)
	default:
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid integer param: %s", val)
		}
		payload = []byte{byte(n)}
	}
	return BuildPT10Frame(cmdType, payload), nil
}

// ParsePT10Response parses response buffer from printer, scanning for a valid frame: AF [Cmd] [Len] [Data...] 0D 0A
func ParsePT10Response(buf []byte) (byte, []byte, error) {
	for i := 0; i < len(buf); i++ {
		if buf[i] == PT10_RESP_HEADER {
			sub := buf[i:]
			if len(sub) < 5 {
				continue // Too short, keep waiting for more data
			}
			cmdType := sub[1]
			dataLen := int(sub[2])
			if len(sub) < 3+dataLen+2 {
				continue // Payload incomplete, keep waiting for more data
			}
			tail := sub[3+dataLen : 3+dataLen+2]
			if bytes.Equal(tail, PT10_FRAME_TAIL) {
				payload := sub[3 : 3+dataLen]
				return cmdType, payload, nil
			}
		}
	}
	return 0, nil, fmt.Errorf("no valid PT10 response frame found")
}

// ParsePT100x20Payload parses concatenated parameters separated by ';' (0x3B)
func ParsePT100x20Payload(payload []byte) (*PT10Params, error) {
	params := &PT10Params{}
	chunks := bytes.Split(payload, []byte{0x3B})

	for _, chunk := range chunks {
		if len(chunk) < 2 {
			continue
		}
		itemType := chunk[0]
		itemData := chunk[1:]

		switch itemType {
		case PT10_CMD_DATE:
			dStr := string(itemData)
			if len(dStr) == 8 {
				params.Date = fmt.Sprintf("%s-%s-%s", dStr[0:4], dStr[4:6], dStr[6:8])
			} else {
				params.Date = dStr
			}
		case PT10_CMD_TIME:
			tStr := string(itemData)
			if len(tStr) == 6 {
				params.Time = fmt.Sprintf("%s:%s:%s", tStr[0:2], tStr[2:4], tStr[4:6])
			} else {
				params.Time = tStr
			}
		case PT10_CMD_BAUDRATE:
			if len(itemData) > 0 {
				params.BaudRate = int(itemData[0])
			}
		case PT10_CMD_PAPER_TYPE:
			if len(itemData) > 0 {
				params.PaperType = int(itemData[0])
			}
		case PT10_CMD_PAPER_TAKE:
			if len(itemData) > 0 {
				params.PaperTake = int(itemData[0])
			}
		case PT10_CMD_SPEED:
			if len(itemData) > 0 {
				params.Speed = int(itemData[0])
			}
		case PT10_CMD_DENSITY:
			if len(itemData) > 0 {
				params.Density = int(itemData[0])
			}
		case PT10_CMD_COVER_FEED:
			if len(itemData) > 0 {
				params.CoverFeed = int(itemData[0])
			}
		case PT10_CMD_POWER_FEED:
			if len(itemData) > 0 {
				params.PowerFeed = int(itemData[0])
			}
		case PT10_CMD_LANGUAGE:
			if len(itemData) > 0 {
				params.Language = int(itemData[0])
			}
		case PT10_CMD_AUTO_PAPER:
			if len(itemData) > 0 {
				params.AutoPaper = int(itemData[0])
			}
		case PT10_CMD_BEEP:
			if len(itemData) > 0 {
				params.Beep = int(itemData[0])
			}
		}
	}

	return params, nil
}
