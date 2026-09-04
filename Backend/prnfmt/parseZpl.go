package prnfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"tmaxsrv/comm"
)

// 解析 ZPL II 打印机指令协议 (支持 BIXOLON SLP-DX220 及 Zebra 斑马打印机)
const (
	ZPL_HEAD           = "^XA\r\n"
	ZPL_TAIL           = "^XZ\r\n"
	ZPL_LINE_END       = "\r\n"
	ZPL_BAR_CODE_EXCEL = "barcode.xlsx"
	ZPL_LANGUAGE_ZPL   = "ZPL"
)

// ParseZplLines 解析模板 CSV 格式字符串并转换为 ZPL II 指令 Buffer
func ParseZplLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
	currentPath := comm.GetSrvDataPath()
	VarTable = ReadTableFromFile(currentPath + "/varTable.json") // 获取变量ID表

	buf := dataBuffer
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(buff))

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		buf, lastvarPos = ZplLines(rowArray, buf, lastvarPos, currentPath)
	}

	// 写入打印命令结尾 ^XZ
	buf.WriteString(ZPL_TAIL)
	return buf
}

func ZplLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	if len(line) == 0 {
		return dataBuffer, lastvarPos
	}

	switch line[0] {
	case "P":
		dataBuffer = ParseZplPage(line, dataBuffer)
	case "TB":
		if len(line) >= 11 {
			if line[10] == "TEXT" {
				dataBuffer = ParseZplText(line, dataBuffer)
			} else if line[10] == "DATA" {
				dataBuffer, lastvarPos = ParseZplVar(line, dataBuffer, lastvarPos)
			}
		}
	case "B":
		dataBuffer, lastvarPos = ParseZplBarcode(line, dataBuffer, lastvarPos, path)
	case "L":
		dataBuffer = ParseZplLine(line, dataBuffer)
	case "R":
		dataBuffer = ParseZplRectangle(line, dataBuffer)
	case "QR":
		if len(line) > 9 {
			dataBuffer, lastvarPos = ParseZplQRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

// ParseZplPage 解析页面配置与纸张大小: P,width,height
// 输出 ZPL 头: ^XA\r\n^PWwidth\r\n^LLheight\r\n
func ParseZplPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 3 {
		return dataBuffer
	}

	widthStr := strings.TrimSpace(strings.ReplaceAll(tempRowArr[1], "\r\n", ""))
	heightStr := strings.TrimSpace(strings.ReplaceAll(tempRowArr[2], "\r\n", ""))

	dataBuffer.WriteString(ZPL_HEAD)
	dataBuffer.WriteString("^PW" + widthStr + ZPL_LINE_END)
	dataBuffer.WriteString("^LL" + heightStr + ZPL_LINE_END)

	return dataBuffer
}

// ParseZplText 解析静态文本: TB,x,y,w,h,font_size,x_mul,y_mul,style,rot,TEXT,content,idx
// 输出: ^FOx,y^A0N,h,w^FDcontent^FS
func ParseZplText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = ZplTextVarPosInfo(tempRowArr, dataBuffer)
	if len(tempRowArr) > 11 {
		dataBuffer.WriteString(tempRowArr[11])
	}
	dataBuffer.WriteString("^FS" + ZPL_LINE_END)
	return dataBuffer
}

// ParseZplVar 解析动态变量文本
func ParseZplVar(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarID(tempRowArr[11])

	if isFind {
		dataBuffer = ZplTextVarPosInfo(tempRowArr, dataBuffer)
		if varId != 1 && varId != 2 {
			tempVarData.id = uint16(varId)
			tempVarData.startPos = uint16(lastvarPos)
			tempVarData.endPos = uint16(dataBuffer.Len() - lastvarPos)
			if len(tempRowArr) > 13 {
				intAlign, err := strconv.Atoi(tempRowArr[13])
				if err == nil {
					tempVarData.align = uint16(intAlign)
				}
			}
			if len(tempRowArr) > 14 {
				intMaxLen, err := strconv.Atoi(tempRowArr[14])
				if err == nil {
					tempVarData.maxlen = uint16(intMaxLen)
				}
			}
			VarList = append(VarList, tempVarData)
			lastvarPos = int(tempVarData.endPos) + lastvarPos
		} else {
			dataBuffer, lastvarPos = VarDateTime(dataBuffer, lastvarPos, varId)
		}
		dataBuffer.WriteString("^FS" + ZPL_LINE_END)
	} else {
		fmt.Println("ZPL parse: can't find var id, please check var name:", tempRowArr[11])
	}
	return dataBuffer, lastvarPos
}

// ZplTextVarPosInfo 构建文本坐标与字体配置指令
// 输出格式: ^FOx,y^A0{rot},23,23[^FR]^FD
func ZplTextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	x := tempRowArr[1]
	y := tempRowArr[2]

	dataBuffer.WriteString("^FO" + x + "," + y)

	rot := getZplRotationChar(tempRowArr[9])

	style := "0"
	if len(tempRowArr) > 8 {
		style = tempRowArr[8]
	}

	fontH := 23
	fontW := 23

	// 样式处理: 2=加粗, 3=反白加粗 (加粗时字宽微调为 27)
	if style == "2" || style == "3" {
		fontW = 27
	}

	dataBuffer.WriteString(fmt.Sprintf("^A0%s,%d,%d", rot, fontH, fontW))

	// 样式处理: 1=反白, 3=反白加粗 (ZPL 中使用 ^FR 指令开启 Field Reverse 反显)
	if style == "1" || style == "3" {
		dataBuffer.WriteString("^FR")
	}

	dataBuffer.WriteString("^FD")
	return dataBuffer
}

// ParseZplLine 解析直线: L,x1,y1,x2,y2,thickness
// 输出: ^FOx,y^GBw,h,t^FS
func ParseZplLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 6 {
		return dataBuffer
	}

	x1, _ := strconv.Atoi(tempRowArr[1])
	y1, _ := strconv.Atoi(tempRowArr[2])
	x2, _ := strconv.Atoi(tempRowArr[3])
	y2, _ := strconv.Atoi(tempRowArr[4])
	t, _ := strconv.Atoi(tempRowArr[5])

	if t <= 0 {
		t = 1
	}

	width := x2 - x1
	if width < 0 {
		width = -width
	}
	height := y2 - y1
	if height < 0 {
		height = -height
	}

	if y1 == y2 { // 横线
		height = t
	} else if x1 == x2 { // 竖线
		width = t
	}

	minX := x1
	if x2 < minX {
		minX = x2
	}
	minY := y1
	if y2 < minY {
		minY = y2
	}

	dataBuffer.WriteString(fmt.Sprintf("^FO%d,%d^GB%d,%d,%d^FS%s", minX, minY, width, height, t, ZPL_LINE_END))
	return dataBuffer
}

// ParseZplRectangle 解析矩形框: R,x,y,x1,y1,thickness
// 输出: ^FOx,y^GBw,h,t^FS
func ParseZplRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 6 {
		return dataBuffer
	}

	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x1, _ := strconv.Atoi(tempRowArr[3])
	y1, _ := strconv.Atoi(tempRowArr[4])
	t, _ := strconv.Atoi(tempRowArr[5])

	if t <= 0 {
		t = 1
	}

	width := x1 - x
	if width < 0 {
		width = -width
	}
	height := y1 - y
	if height < 0 {
		height = -height
	}

	dataBuffer.WriteString(fmt.Sprintf("^FO%d,%d^GB%d,%d,%d^FS%s", x, y, width, height, t, ZPL_LINE_END))
	return dataBuffer
}

// ParseZplBarcode 解析一维条码: B,x,y,w,h,narrow,codeType,rot,readable,TEXT/DATA...
// 支持 Code 128 (^BC), Code 39 (^B3), EAN-13 (^BE), EAN-8 (^B8)
func ParseZplBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	if len(tempRowArr) < 9 {
		return dataBuffer, lastvarPos
	}

	x := tempRowArr[1]
	y := tempRowArr[2]
	height := tempRowArr[4]
	rot := getZplRotationChar(tempRowArr[7])

	readable := "Y"
	if tempRowArr[8] == "N" || tempRowArr[8] == "0" {
		readable = "N"
	}

	excelPath := path + "\\" + ZPL_BAR_CODE_EXCEL
	codeType := getBarCodeTypeId(excelPath, ZPL_LANGUAGE_ZPL, tempRowArr[6])
	if codeType == "" {
		codeType = tempRowArr[6]
	}

	dataBuffer.WriteString("^FO" + x + "," + y)

	switch strings.ToUpper(codeType) {
	case "CODE39", "39", "B3":
		dataBuffer.WriteString(fmt.Sprintf("^B3%s,N,%s,%s,N^FD", rot, height, readable))
	case "EAN13", "E13", "BE":
		dataBuffer.WriteString(fmt.Sprintf("^BE%s,%s,%s,N^FD", rot, height, readable))
	case "EAN8", "E8", "B8":
		dataBuffer.WriteString(fmt.Sprintf("^B8%s,%s,%s,N^FD", rot, height, readable))
	default: // 默认 Code 128
		dataBuffer.WriteString(fmt.Sprintf("^BC%s,%s,%s,N,N^FD", rot, height, readable))
	}

	parseContentArr := tempRowArr[9:]
	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString("^FS" + ZPL_LINE_END)
	return dataBuffer, lastvarPos
}

// ParseZplQRcode 解析二维码: QR,x,y,mode,size,ec,mask,TEXT/DATA...
// 输出: ^FOx,y^BQrot,2,size^FDcontent^FS
func ParseZplQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	if len(tempRowArr) < 7 {
		return dataBuffer, lastvarPos
	}

	x := tempRowArr[1]
	y := tempRowArr[2]
	size := tempRowArr[4]
	if size == "" || size == "0" {
		size = "4"
	}

	rot := "N"
	if len(tempRowArr) > 8 {
		rot = getZplRotationChar(tempRowArr[8])
	}

	dataBuffer.WriteString(fmt.Sprintf("^FO%s,%s^BQ%s,2,%s^FD", x, y, rot, size))

	parseContentArr := tempRowArr[7:]
	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString("^FS" + ZPL_LINE_END)
	return dataBuffer, lastvarPos
}

// 旋转参数转换: "0"/"90"/"180"/"270" 或 "0"/"1"/"2"/"3" -> "N"/"R"/"I"/"B"
func getZplRotationChar(rotStr string) string {
	switch rotStr {
	case "90", "1":
		return "R"
	case "180", "2":
		return "I"
	case "270", "3":
		return "B"
	default:
		return "N"
	}
}
