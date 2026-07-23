package cmd

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBuildPT10Frame(t *testing.T) {
	frame := BuildPT10PingFrame()
	expected := []byte{0xFF, 0xFE, 0x10, 0x01, 0x00, 0x0D, 0x0A}
	if !bytes.Equal(frame, expected) {
		t.Errorf("BuildPT10PingFrame expected %X, got %X", expected, frame)
	}

	qFrame := BuildPT10QueryFrame()
	expectedQ := []byte{0xFF, 0xFE, 0x20, 0x01, 0x00, 0x0D, 0x0A}
	if !bytes.Equal(qFrame, expectedQ) {
		t.Errorf("BuildPT10QueryFrame expected %X, got %X", expectedQ, qFrame)
	}
}

func TestParsePT10Response(t *testing.T) {
	respBuf := []byte{0xAF, 0x10, 0x01, 0x00, 0x0D, 0x0A}
	cmdType, payload, err := ParsePT10Response(respBuf)
	if err != nil {
		t.Fatalf("ParsePT10Response error: %v", err)
	}
	if cmdType != 0x10 {
		t.Errorf("Expected cmdType 0x10, got %X", cmdType)
	}
	if len(payload) != 1 || payload[0] != 0x00 {
		t.Errorf("Expected payload [0x00], got %X", payload)
	}
}

func TestParsePT100x20Payload(t *testing.T) {
	// Sample payload:
	// 0x01 + 20260729 + ';'
	// 0x02 + 122014 + ';'
	// 0x03 + 0x01 + ';'
	// 0x04 + 0x01 + ';'
	// 0x05 + 0x01 + ';'
	// 0x06 + 0x02 + ';'
	// 0x07 + 0x02 + ';'
	// 0x08 + 0x01 + ';'
	// 0x09 + 0x00 + ';'
	// 0x0A + 0x01 + ';'
	// 0x0B + 0x00 + ';'
	// 0x0C + 0x01 + ';'
	var payload []byte
	payload = append(payload, append([]byte{0x01}, []byte("20260729;")...)...)
	payload = append(payload, append([]byte{0x02}, []byte("122014;")...)...)
	payload = append(payload, []byte{0x03, 0x01, 0x3B}...)
	payload = append(payload, []byte{0x04, 0x01, 0x3B}...)
	payload = append(payload, []byte{0x05, 0x01, 0x3B}...)
	payload = append(payload, []byte{0x06, 0x02, 0x3B}...)
	payload = append(payload, []byte{0x07, 0x02, 0x3B}...)
	payload = append(payload, []byte{0x08, 0x01, 0x3B}...)
	payload = append(payload, []byte{0x09, 0x00, 0x3B}...)
	payload = append(payload, []byte{0x0A, 0x01, 0x3B}...)
	payload = append(payload, []byte{0x0B, 0x00, 0x3B}...)
	payload = append(payload, []byte{0x0C, 0x01, 0x3B}...)

	params, err := ParsePT100x20Payload(payload)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := &PT10Params{
		Date:      "2026-07-29",
		Time:      "12:20:14",
		BaudRate:  1,
		PaperType: 1,
		PaperTake: 1,
		Speed:     2,
		Density:   2,
		CoverFeed: 1,
		PowerFeed: 0,
		Language:  1,
		AutoPaper: 0,
		Beep:      1,
	}

	if !reflect.DeepEqual(params, expected) {
		t.Errorf("Expected %+v, got %+v", expected, params)
	}
}
