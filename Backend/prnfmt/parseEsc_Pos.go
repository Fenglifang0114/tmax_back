package prnfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"tmaxsrv/comm"
)

var (
	ESC_INIT                 = []byte{0x1B, 0x40}
	ESC_LINE_END             = []byte{0x0a}
	ESC_LINE_OFFSET_X        = []byte{0x1B, 0x24, 0x00, 0x00}
	ESC_LINE_NUM             = 0
	ESC_TAIL                 = []byte{0x1B, 0x64, 0x03}
	ESC_FONT_NORMAL          = []byte{0x1B, 0x45, 0x00, 0x1D, 0x42, 0x00}
	ESC_FONT_BOLD            = []byte{0x1B, 0x45, 0x01}
	ESC_FONT_REVERTED        = []byte{0x1D, 0x42, 0x01}
	ESC_FONT_CANCEL_REVERTED = []byte{0x1D, 0x42, 0x00}
	ESC_REVOLVE_90           = []byte{0x1B, 0x56, 0x01}
	ESC_CANCEL_REVOLVE_90    = []byte{0x1B, 0x56, 0x00}
	ESC_UTF8_MODE            = []byte{0x1C, 0x26, 0x1B, 0x39, 0x01}
	ESC_LOOP_FLAG            = 0
	ESC_CHANGE_ESC_205       = []byte{0x1F, 0x28, 0x4C, 0x03, 0x00, 0x43, 0x4E, 0x0D}

	//12 23 0F 打印机浓度  00-0F
)

const (
	escLineEnd      = "\r\n"
	escTxtLineHead  = "A"
	escInnerLineSep = ","

	ESC_ROTATE_ZB      = "ZB"
	ESC_ROTATE_ZT      = "ZT"
	ESC_BAR_CODE_EXCEL = "barcode.xlsx"
	LANGUAGE_ESC       = "EPL"
	ESC_LINE_HEIGHT    = 31
)

func ParseEscLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
	// 获取VarTable.json和barcode.xlsx文件路径
	// file, _ := exec.LookPath(os.Args[0])
	// path, _ := filepath.Abs(file)
	// index := strings.LastIndex(path, string(os.PathSeparator))
	// currentPath := path[:index]
	// currentPath = filepath.Join(currentPath, comm.SRV_DATA_PATH)
	currentPath := comm.GetSrvDataPath()                         //20241107
	VarTable = ReadTableFromFile(currentPath + "/varTable.json") // 获取变量ID表TODO:

	// 处理字符串并解析
	buf := dataBuffer
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(buff))

	for scanner.Scan() {
		lines = append(lines, scanner.Text()) // 每行字符串添加到切片
	}
	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		buf, lastvarPos = EscLines(rowArray, buf, lastvarPos, currentPath)
	}

	for _, b := range buf.Bytes() {
		fmt.Printf("%02X ", b)
	}

	// 写入打印命令结尾
	buf.Write(ESC_TAIL)
	return buf
}

func EscLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	switch line[0] {
	case "P":
		ESC_LINE_NUM = 0
		ESC_LOOP_FLAG = 0
		ParseEscPage(line, dataBuffer)
	case "TB":
		// fmt.Println(len(line))
		if line[10] == "TEXT" && len(line) == 13 {
			dataBuffer = ParseEscText(line, dataBuffer)
		} else if line[10] == "DATA" && len(line) == 16 {
			dataBuffer = ParseEscVar(line, dataBuffer)
		}

		// case "B":
		// 	dataBuffer, lastvarPos = ParsEscBarcode(line, dataBuffer, lastvarPos, path)
		// case "ROTATE":
		// 	// dataBuffer = ParseEscRotate(line, dataBuffer)
		// case "L":
		// 	dataBuffer = ParseEscLine(line, dataBuffer)
		// case "R":
		// 	dataBuffer = ParseEscRectangle(line, dataBuffer)
		// case "O":

		// case "GP":

		// case "QR":
		// 	if len(line) > 9 {
		// 		dataBuffer, lastvarPos = ParsEscQRcode(line, dataBuffer, lastvarPos)
		// 	}
	}
	return dataBuffer, lastvarPos
}

func ParseEscPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.Write(ESC_INIT)
	dataBuffer.Write(ESC_UTF8_MODE)

	return dataBuffer
}

// L,63,133,153,133,2,0,0
func ParseEscRotate(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if tempRowArr[1] == "0" {
		dataBuffer.WriteString(ESC_ROTATE_ZB)
	} else if tempRowArr[1] == "2" {
		dataBuffer.WriteString(ESC_ROTATE_ZT)
	}

	dataBuffer.WriteString(escLineEnd)

	return dataBuffer
}

// L,63,133,153,133,2,0,0
func ParseEscLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {

	dataBuffer.WriteString(tempRowArr[1])
	dataBuffer.WriteString(escInnerLineSep)

	dataBuffer.WriteString(tempRowArr[2])
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[5])
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[3])
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[4])
	dataBuffer.WriteString(escLineEnd)

	return dataBuffer
}

// R,63,133,153,133,2,0,0
func ParseEscRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {

	dataBuffer.WriteString(tempRowArr[1])
	dataBuffer.WriteString(escInnerLineSep)

	dataBuffer.WriteString(tempRowArr[2])
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[5])
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[3])
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[4])
	dataBuffer.WriteString(escLineEnd)

	return dataBuffer
}

func getXOffset(offsetValue int) []byte {
	if offsetValue < 65535 {
		ESC_LINE_OFFSET_X[2] = byte(offsetValue & 0xFF)        // 低位 nL
		ESC_LINE_OFFSET_X[3] = byte((offsetValue >> 8) & 0xFF) // 高位 nH
	}
	return ESC_LINE_OFFSET_X
}

// TB,12,43,130,30,0,1,1,2,0,TEXT,TIME,0
func ParseEscText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = EscTextVarPosInfo(tempRowArr, dataBuffer)

	if len(tempRowArr[11]) > 0 {
		dataBuffer.WriteString(tempRowArr[11])
	}
	return dataBuffer
}

func ParseEscVar(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	var tempVarData RptVarStruct
	varId, isFind := findEscVarID(tempRowArr[11])
	if varId == 254 && ESC_LOOP_FLAG == 0 {
		ESC_LOOP_FLAG++
	} else if varId == 254 && ESC_LOOP_FLAG == 1 {
		varId = 255
		ESC_LOOP_FLAG = 0
	}
	if isFind {
		dataBuffer = EscTextVarPosInfo(tempRowArr, dataBuffer)

		// 变量为日期格式需要单独处理
		if varId != 1 && varId != 2 {
			tempVarData.id = uint8(varId)
			tempVarData.startPos = uint16(dataBuffer.Len())
			intAlign, err := strconv.Atoi(tempRowArr[13])
			if err == nil {
				tempVarData.align = uint8(intAlign)
			}
			intMaxLen, err := strconv.Atoi(tempRowArr[14])
			if err == nil {
				tempVarData.maxLen = uint8(intMaxLen)
			}
			RptVarList = append(RptVarList, tempVarData)

		} else {
			dataBuffer = VarEscDateTime(dataBuffer, varId)
		}

	} else {
		fmt.Println("can't find var id ,please check var name")
	}

	return dataBuffer
}

func getLineNum(yPos int) int {
	return int(math.Round(float64(yPos) / ESC_LINE_HEIGHT))

}

func insertRowEnd(tempYLine int) []byte {
	count := tempYLine - ESC_LINE_NUM
	escLineEnd := make([]byte, count*len(ESC_LINE_END)) // 为了性能考虑，避免在循环中多次调用 Write
	for i := 0; i < count; i++ {
		copy(escLineEnd[i*len(ESC_LINE_END):], ESC_LINE_END)
	}

	ESC_LINE_NUM += count

	return escLineEnd
}

func EscTextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {

	xOffset, err := strconv.Atoi(tempRowArr[1])
	if err != nil {
		return dataBuffer
	}
	yOffset, err := strconv.Atoi(tempRowArr[2])
	if err != nil {
		return dataBuffer
	}
	tempYLine := getLineNum(yOffset)
	if tempYLine > ESC_LINE_NUM {
		escLineEnd := insertRowEnd(tempYLine) //先看要不要增加换行
		dataBuffer.Write(escLineEnd)
	}
	dataBuffer.Write(getXOffset(xOffset)) //X
	if tempRowArr[9] != "0" {             //旋转
		dataBuffer.Write(ESC_REVOLVE_90)
	} else {
		dataBuffer.Write(ESC_CANCEL_REVOLVE_90)
	}
	// dataBuffer.WriteString(tempRowArr[5]) // 字体大小
	// dataBuffer.WriteString(tempRowArr[6]) // 宽度倍数
	// dataBuffer.WriteString(tempRowArr[7]) // 高度倍数
	if tempRowArr[8] == "0" {
		dataBuffer.Write(ESC_FONT_NORMAL)
	} else if tempRowArr[8] == "1" {
		dataBuffer.Write(ESC_FONT_NORMAL)
		dataBuffer.Write(ESC_FONT_REVERTED)
	} else if tempRowArr[8] == "2" {
		dataBuffer.Write(ESC_FONT_NORMAL)
		dataBuffer.Write(ESC_FONT_BOLD)
	} else if tempRowArr[8] == "3" {
		dataBuffer.Write(ESC_FONT_REVERTED)
		dataBuffer.Write(ESC_FONT_BOLD)
	}

	return dataBuffer
}

/*
转换条码
*/
// content: TEXT,2,TEXT,0,TEXT,0,DATA,Net,01234567,3,8,TEXT,7,TEXT,8,DATA,Net,01234567,3,8

func ParsEscBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	lineHead := "B"
	escInnerLineSep := ","
	dataSep := "\""
	bottom := "BC" // 显示数据在条码下方，居中
	// noCode := "N"  //不显示数据
	top := "TC" // 显示数据在条码上方，居中（没有适配的打印机命令）
	parseContentArr := []string{}

	dataBuffer.WriteString(lineHead)
	dataBuffer.WriteString(tempRowArr[1]) // X

	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[2]) // Y

	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[7]) // 旋转

	dataBuffer.WriteString(escInnerLineSep)
	path = filepath.Join(path, ESC_BAR_CODE_EXCEL)
	codeType := getBarCodeTypeId(path, LANGUAGE_ESC, tempRowArr[6])
	dataBuffer.WriteString(codeType)
	dataBuffer.WriteString(escInnerLineSep)

	dataBuffer.WriteString(tempRowArr[5]) // 条码中窄条的宽度
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[5]) // 条码中宽条的宽度 TTC中只送了一个宽度
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(tempRowArr[4]) // 条码整体高度
	dataBuffer.WriteString(escInnerLineSep)
	if tempRowArr[8] == top {
		dataBuffer.WriteString(bottom)
	} else {
		dataBuffer.WriteString(tempRowArr[8])
	}
	dataBuffer.WriteString(escInnerLineSep)
	dataBuffer.WriteString(dataSep) // 条码数据开始
	for i := 9; i < len(tempRowArr); i++ {
		parseContentArr = append(parseContentArr, tempRowArr[i])
	}

	if len(parseContentArr) == 0 {
		fmt.Println("barCode data is missing ,please check barcode format")
	}

	dataBuffer, lastvarPos = FindEscVarInfo(parseContentArr, dataBuffer, lastvarPos)
	dataBuffer.WriteString(dataSep)
	dataBuffer.WriteString(escLineEnd)
	return dataBuffer, lastvarPos
}

func findEscVarID(name string) (int, bool) {
	lenth := len(VarTable.ScaleVarTable)
	isFind := false
	id := 0
	for i := 0; i < lenth; i++ {
		if name == VarTable.ScaleVarTable[i].ValueVame {
			id = VarTable.ScaleVarTable[i].Id
			isFind = true
			i = lenth
		}
	}
	return id, isFind
}

// QR,248,176,1,6,1,,DATA,Net,000000,0,6,DATA,Unit_weight,000000,0,4,TEXT,SN9000,0

// 转换后 b100,200,Q,m2,s6,"123456789"  //Q 代表QR m2 代表模式2，样式， s6代表大小  “” 数据
func ParsEscQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	lineHead := "b"
	escInnerLineSep := ","
	dataSep := "\""
	QRcodeType := "Q,"
	styleType := "m2,"
	QRsize := "s"

	parseContentArr := []string{}
	dataBuffer.WriteString(lineHead)
	dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(escInnerLineSep)

	dataBuffer.WriteString(tempRowArr[2]) // Y
	dataBuffer.WriteString(escInnerLineSep)

	dataBuffer.WriteString(QRcodeType) // Q，
	dataBuffer.WriteString(styleType)  // m2,
	dataBuffer.WriteString(QRsize)     // s

	sizedata, err := strconv.Atoi(tempRowArr[4])
	if err != nil {
		fmt.Println("change size to int error")
	}
	if 0 < sizedata && sizedata < 99 {
		dataBuffer.WriteString(tempRowArr[4]) // 二维码大小
	} else {
		dataBuffer.WriteString("6")
	}
	dataBuffer.WriteString(escInnerLineSep)

	dataBuffer.WriteString(dataSep) // 二维码数据开始
	for i := 7; i < len(tempRowArr); i++ {
		parseContentArr = append(parseContentArr, tempRowArr[i])
	}

	if len(parseContentArr) == 0 {
		fmt.Println("QRcode data is missing ,please check QR format")
	}
	dataBuffer, lastvarPos = FindEscVarInfo(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString(dataSep)
	dataBuffer.WriteString(escLineEnd)
	return dataBuffer, lastvarPos
}

func VarEscDateTime(dataBuffer *bytes.Buffer, varId int) *bytes.Buffer {
	var tempVarData RptVarStruct
	if varId == 1 {
		for i := 1; i < 4; i++ {
			if i == 1 {
				tempVarData.maxLen = uint8(4)
			} else {
				dataBuffer.WriteString("/")
				tempVarData.maxLen = uint8(2)
			}
			tempVarData.id = uint8(i)
			tempVarData.startPos = uint16(dataBuffer.Len())
			tempVarData.align = uint8(1)
			RptVarList = append(RptVarList, tempVarData)

		}
	} else {
		for i := 4; i < 7; i++ {
			if i == 4 {
				tempVarData.maxLen = uint8(2)
			} else {
				dataBuffer.WriteString(":")
				tempVarData.maxLen = uint8(2)
			}
			tempVarData.maxLen = uint8(2)
			tempVarData.id = uint8(i)
			tempVarData.startPos = uint16(dataBuffer.Len())
			tempVarData.align = uint8(1)
			RptVarList = append(RptVarList, tempVarData)
		}
	}
	return dataBuffer
}

func FindEscVarInfo(parseContentArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct

	for i := 0; i < len(parseContentArr)-1; i++ {
		if parseContentArr[i] == "TEXT" {
			dataBuffer.WriteString(parseContentArr[i+1])
			i = i + 1
		} else if parseContentArr[i] == "DATA" {
			varId, isFind := findEscVarID(parseContentArr[i+1])
			i = i + 1
			if isFind {
				if varId != 1 && varId != 2 {

					tempVarData.id = uint16(varId)
					tempVarData.startPos = uint16(lastvarPos)
					tempVarData.endPos = uint16(dataBuffer.Len() - lastvarPos)

					i = i + 1
					alignData, err := strconv.Atoi(parseContentArr[i+1])
					if err != nil {
						fmt.Println("data alignment error,please check it")
					}
					tempVarData.align = uint16(alignData)
					i = i + 1
					maxLen, err := strconv.Atoi(parseContentArr[i+1])
					if err != nil {
						fmt.Println("data maxLength error,please check it")
					}
					i = i + 1
					tempVarData.maxlen = uint16(maxLen)
					VarList = append(VarList, tempVarData)
					lastvarPos = int(tempVarData.endPos) + lastvarPos

				} else {
					dataBuffer = VarEscDateTime(dataBuffer, varId)
				}
			} else {
				fmt.Println("Var name is not find ")
			}
		}
	}
	return dataBuffer, lastvarPos
}
