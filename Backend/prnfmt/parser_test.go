package prnfmt

import (
	"bytes"
	"testing"
)

func TestParserFmtToBuf(t *testing.T) {
	var mybuf bytes.Buffer
	mybuf.WriteString("Hello, World!")
	type args struct {
		utf8Buff string
	}
	tests := []struct {
		name string
		args args
		want *bytes.Buffer
	}{
		{name: "TestParserFmtToBuf", args: args{utf8Buff: `P,396,360
TB,29,38,130,30,0,1,1,0,0,DATA,Date,2023-03-16,1,10,2
TB,165,38,130,30,0,1,1,0,0,DATA,Time,10:38:16,1,10,3
TB,100,116,121,30,0,1,1,0,0,DATA,percent,percent,3,7,13
TB,29,116,109,30,0,1,1,0,0,TEXT,PERCENT:,14
TB,227,116,130,30,0,1,1,0,0,TEXT,%,15
ROTATE,0`}, want: &mybuf},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParserFmtToBuf(tt.args.utf8Buff, "EPM205", 2048); len(got.Bytes()) != 2048 {
				t.Errorf("ParserFmtToBuf() len = %d, expected 2048", len(got.Bytes()))
			}
		})
	}
}

func TestParserFmtToRawCmd(t *testing.T) {
	utf8Buff := `P,396,360
TB,29,38,130,30,0,1,1,0,0,DATA,Net,000000,1,10,10
ROTATE,0
F,EPM205,L`

	buf, vars := ParserFmtToRawCmd(utf8Buff)
	if buf == nil || buf.Len() == 0 {
		t.Fatalf("ParserFmtToRawCmd() returned empty buffer")
	}
	if len(buf.Bytes()) >= 2048 {
		t.Errorf("ParserFmtToRawCmd() len = %d, expected raw command (much smaller than 2048)", len(buf.Bytes()))
	}
	if len(vars) == 0 {
		t.Errorf("ParserFmtToRawCmd() vars length is 0, expected variables parsed")
	}
}

func TestReadTableFromFile_Fallback(t *testing.T) {
	// 模拟在不存在文件的路径（如 macOS 上未复制 varTable.json 的情况）
	order := ReadTableFromFile("/nonexistent/path/on/macos/varTable.json")
	if len(order.ScaleVarTable) == 0 {
		t.Errorf("ReadTableFromFile failed to fallback to embedded varTable.json")
	}

	// 验证包含核心变量
	foundNet := false
	for _, v := range order.ScaleVarTable {
		if v.ValueVame == "Net" {
			foundNet = true
			break
		}
	}
	if !foundNet {
		t.Errorf("Expected embedded varTable to contain 'Net' variable")
	}
}

func TestParserFmtToBytes_NoPanic(t *testing.T) {
	utf8Buff := `P,396,360
TB,29,38,130,30,0,1,1,0,0,DATA,Date,2023-03-16,1,10,2
TB,165,38,130,30,0,1,1,0,0,DATA,Time,10:38:16,1,10,3
TB,100,116,121,30,0,1,1,0,0,DATA,Net,000000,1,10,10
ROTATE,0
F,EPM205,L`

	// 模拟执行标签格式下发解析，验证不会发生 panic 崩溃
	data, success := ParserFmtToBytes(utf8Buff, "EPM205", 2048)
	if !success || len(data) == 0 {
		t.Errorf("ParserFmtToBytes returned false or empty data")
	}

	// 模拟执行票据格式下发解析，验证不会发生 panic 崩溃
	receiptBuff := `P,396,360
TB,29,38,130,30,0,1,1,0,0,DATA,Net,000000,1,10,10
ROTATE,0
F,EPM205,R`
	dataRpt, successRpt := ParserFmtToBytes(receiptBuff, "EPM205", 2048)
	if !successRpt || len(dataRpt) == 0 {
		t.Errorf("ParserFmtToBytes (Receipt) returned false or empty data")
	}
}

