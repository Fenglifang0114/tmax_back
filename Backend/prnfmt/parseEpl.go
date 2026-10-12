package prnfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"tmaxsrv/comm"
)

const (
	EPL_HEAD           = "FK\"AA\"\r\nFS\"AA\"\r\nD8\r\nS4\r\nI0\r\nj0\r\nR0,0\r\n"
	EPL_TAIL           = "FE\r\nFR\"AA\"\r\nP1,1\r\n"
	EPL_LINE_END       = "\r\n"
	EPL_TEXT_LINE_HEAD = "A"
	EPL_INNER_LINE_SEP = ","
	EPL_REVERTED       = "R"  // 黑色背景白色字体
	EPL_TEXT_BOLD      = "B"  // 字体加粗
	EPL_FOND_NORMAL    = "N"  // 字体正常
	EPL_REVERTED_BOLD  = "W"  // 反白加粗
	EPL_TEXT_SEP       = "\"" // 文本分割符""
	EPL_LINE_HEAD      = "LS"
	EPL_RECTANGLE_HEAD = "X"
	EPL_ROTATE_ZB      = "ZB"
	EPL_ROTATE_ZT      = "ZT"
	EPL_BAR_CODE_EXCEL = "barcode.xlsx"
	EPL_LANGUAGE_EPL   = "EPL"
)

func ParseEplLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
	// 获取VarTable.json和barcode.xlsx文件路径
	// file, _ := exec.LookPath(os.Args[0])
	// path, _ := filepath.Abs(file)
	// index := strings.LastIndex(path, string(os.PathSeparator))
	// currentPath := path[:index]
	// currentPath = filepath.Join(currentPath, comm.SRV_DATA_PATH)
	currentPath := comm.GetSrvDataPath()                         //20241107
	VarTable = ReadTableFromFile(currentPath + "/varTable.json") // 获取变量ID表

	// 处理字符串并解析
	buf := dataBuffer
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(buff))

	for scanner.Scan() {
		lines = append(lines, scanner.Text()) // 每行字符串添加到切片
	}
	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		buf, lastvarPos = EplLines(rowArray, buf, lastvarPos, currentPath)
	}

	// 写入打印命令结尾
	buf.WriteString(EPL_TAIL)
	return buf
}

func EplLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	switch line[0] {
	case "P":
		ParseEplPage(line, dataBuffer)
	case "TB":
		// fmt.Println(len(line))
		if line[10] == "TEXT" && len(line) == 13 {
			dataBuffer = ParseEplText(line, dataBuffer)
		} else if line[10] == "DATA" && len(line) == 16 {
			dataBuffer, lastvarPos = ParseEplVar(line, dataBuffer, lastvarPos)
		}

	case "B":
		dataBuffer, lastvarPos = ParsEplBarcode(line, dataBuffer, lastvarPos, path)
	case "ROTATE":
		dataBuffer = ParseEplRotate(line, dataBuffer)
	case "L":
		dataBuffer = ParseEplLine(line, dataBuffer)
	case "R":
		dataBuffer = ParseEplRectangle(line, dataBuffer)
	case "O":

	case "GP":

	case "QR":
		if len(line) > 9 {
			dataBuffer, lastvarPos = ParsEplQRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

func ParseEplPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	widthHead := "q"
	hightHead := "Q"
	paperGap := ",24+8"

	dataBuffer.WriteString(EPL_HEAD)

	dataBuffer.WriteString(hightHead)
	if strings.Contains(tempRowArr[2], "\r\n") {
		tempRowArr[2] = strings.Replace(tempRowArr[2], "\r\n", "", 1)
	}
	dataBuffer.WriteString(tempRowArr[2]) // 高度
	dataBuffer.WriteString(paperGap)
	dataBuffer.WriteString(EPL_LINE_END)

	dataBuffer.WriteString(widthHead)
	dataBuffer.WriteString(tempRowArr[1]) // 宽度
	dataBuffer.WriteString(EPL_LINE_END)

	// fmt.Println(dataBuffer.String())
	return dataBuffer
}

// L,63,133,153,133,2,0,0
func ParseEplRotate(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if tempRowArr[1] == "0" {
		dataBuffer.WriteString(EPL_ROTATE_ZB)
	} else if tempRowArr[1] == "2" {
		dataBuffer.WriteString(EPL_ROTATE_ZT)
	}

	dataBuffer.WriteString(EPL_LINE_END)

	return dataBuffer
}

// L,63,133,153,133,2,0,0
// 	//  LS10,10,20,200,200  画对角线
// 	//LO10,10,200,10  画直线
//	[1] - 水平起始 X1
//	[2] - 垂直起始 Y1
//	[3] - 线宽
//	[4] - 线高

func ParseEplLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	// 参数有效性检查（至少需要索引1-5）
	if len(tempRowArr) < 6 {
		return dataBuffer
	}

	// 横线判断：Y1 == Y2
	width := 0
	height := 0
	if tempRowArr[2] == tempRowArr[4] {
		// 宽度 = |X2 - X1|
		x1, _ := strconv.Atoi(tempRowArr[1])
		x2, _ := strconv.Atoi(tempRowArr[3])
		width = x2 - x1
		if width < 0 {
			width = -width
		}
		height, _ = strconv.Atoi(tempRowArr[5])
		// 此处可根据需要处理 width，例如校验或日志
	}

	// 竖线判断：X1 == X2

	if tempRowArr[1] == tempRowArr[3] {
		// 高度 = |Y2 - Y1|
		y1, _ := strconv.Atoi(tempRowArr[2])
		y2, _ := strconv.Atoi(tempRowArr[4])
		height = y2 - y1
		if height < 0 {
			height = -height
		}
		width, _ = strconv.Atoi(tempRowArr[5])
		// 此处可根据需要处理 height
	}

	// 写入LO指令，参数顺序：X1, Y1, 线宽, 线高
	dataBuffer.WriteString("LO")
	dataBuffer.WriteString(tempRowArr[1])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[2])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(width))
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(height))
	dataBuffer.WriteString(EPL_LINE_END)

	return dataBuffer
}

// R,63,133,153,133,2,0,0
func ParseEplRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(EPL_RECTANGLE_HEAD)
	dataBuffer.WriteString(tempRowArr[1])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[2])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[5])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[3])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[4])
	dataBuffer.WriteString(EPL_LINE_END)

	return dataBuffer
}

// TB,12,43,130,30,0,1,1,2,0,TEXT,TIME,0
func ParseEplText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = TextVarPosInfo(tempRowArr, dataBuffer)
	dataBuffer.WriteString(EPL_TEXT_SEP)

	if len(tempRowArr[11]) > 0 {
		dataBuffer.WriteString(tempRowArr[11])
		dataBuffer.WriteString(EPL_TEXT_SEP)
		dataBuffer.WriteString(EPL_LINE_END)
	} else {
		dataBuffer.WriteString(EPL_TEXT_SEP)
		dataBuffer.WriteString(EPL_LINE_END)
	}
	return dataBuffer
}

func ParseEplVar(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarID(tempRowArr[11])

	if isFind {
		dataBuffer = TextVarPosInfo(tempRowArr, dataBuffer)
		dataBuffer.WriteString(EPL_TEXT_SEP)
		// 变量为日期格式需要单独处理
		if varId != 1 && varId != 2 {

			tempVarData.id = uint16(varId)

			tempVarData.startPos = uint16(lastvarPos)
			tempVarData.endPos = uint16(dataBuffer.Len() - lastvarPos)
			intAlign, err := strconv.Atoi(tempRowArr[13])
			if err == nil {
				tempVarData.align = uint16(intAlign)
			}
			intMaxLen, err := strconv.Atoi(tempRowArr[14])
			if err == nil {
				tempVarData.maxlen = uint16(intMaxLen)
			}
			VarList = append(VarList, tempVarData)
			lastvarPos = int(tempVarData.endPos) + lastvarPos

		} else {
			dataBuffer, lastvarPos = VarDateTime(dataBuffer, lastvarPos, varId)
		}

		dataBuffer.WriteString(EPL_TEXT_SEP)
		dataBuffer.WriteString(EPL_LINE_END)

	} else {
		fmt.Println("can't find var id ,please check var name")
	}

	return dataBuffer, lastvarPos
}

func TextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(EPL_TEXT_LINE_HEAD)
	dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[2]) // Y
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[9]) // 旋转
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	if tempRowArr[5] == "0" { // 字体大小不能为0
		tempRowArr[5] = "4"
	}
	dataBuffer.WriteString(tempRowArr[5]) // 字体大小
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[6]) // 宽度倍数
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[7]) // 高度倍数
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)

	if tempRowArr[8] == "0" {
		dataBuffer.WriteString(EPL_FOND_NORMAL)
		dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	} else if tempRowArr[8] == "1" {
		dataBuffer.WriteString(EPL_REVERTED)
		dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	} else if tempRowArr[8] == "2" {
		dataBuffer.WriteString(EPL_TEXT_BOLD)
		dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	} else if tempRowArr[8] == "3" {
		dataBuffer.WriteString(EPL_REVERTED_BOLD)
		dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	}

	return dataBuffer
}

/*
转换条码
*/
// content: TEXT,2,TEXT,0,TEXT,0,DATA,Net,01234567,3,8,TEXT,7,TEXT,8,DATA,Net,01234567,3,8

func ParsEplBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	lineHead := "B"
	innerLineSep := ","
	dataSep := "\""
	bottom := "BC" // 显示数据在条码下方，居中
	// noCode := "N"  //不显示数据
	top := "TC" // 显示数据在条码上方，居中（没有适配的打印机命令）
	parseContentArr := []string{}

	dataBuffer.WriteString(lineHead)
	dataBuffer.WriteString(tempRowArr[1]) // X

	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(tempRowArr[2]) // Y

	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(tempRowArr[7]) // 旋转

	dataBuffer.WriteString(innerLineSep)
	path = filepath.Join(path, EPL_BAR_CODE_EXCEL)
	codeType := getBarCodeTypeId(path, EPL_LANGUAGE_EPL, tempRowArr[6])
	dataBuffer.WriteString(codeType)
	dataBuffer.WriteString(innerLineSep)

	dataBuffer.WriteString(tempRowArr[5]) // 条码中窄条的宽度
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(tempRowArr[5]) // 条码中宽条的宽度 TTC中只送了一个宽度
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(tempRowArr[4]) // 条码整体高度
	dataBuffer.WriteString(innerLineSep)
	if tempRowArr[8] == top {
		dataBuffer.WriteString(bottom)
	} else {
		dataBuffer.WriteString(tempRowArr[8])
	}
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(dataSep) // 条码数据开始
	for i := 9; i < len(tempRowArr); i++ {
		parseContentArr = append(parseContentArr, tempRowArr[i])
	}

	if len(parseContentArr) == 0 {
		fmt.Println("barCode data is missing ,please check barcode format")
	}

	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)
	dataBuffer.WriteString(dataSep)
	dataBuffer.WriteString(EPL_LINE_END)
	return dataBuffer, lastvarPos
}

func findVarID(name string) (int, bool) {
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
func ParsEplQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	lineHead := "b"
	innerLineSep := ","
	dataSep := "\""
	QRcodeType := "Q,"
	styleType := "m2,"
	QRsize := "s"

	parseContentArr := []string{}
	dataBuffer.WriteString(lineHead)
	dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(innerLineSep)

	dataBuffer.WriteString(tempRowArr[2]) // Y
	dataBuffer.WriteString(innerLineSep)

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
	dataBuffer.WriteString(innerLineSep)

	dataBuffer.WriteString(dataSep) // 二维码数据开始
	for i := 7; i < len(tempRowArr); i++ {
		parseContentArr = append(parseContentArr, tempRowArr[i])
	}

	if len(parseContentArr) == 0 {
		fmt.Println("QRcode data is missing ,please check QR format")
	}
	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString(dataSep)
	dataBuffer.WriteString(EPL_LINE_END)
	return dataBuffer, lastvarPos
}

func VarDateTime(dataBuffer *bytes.Buffer, lastvarPos int, varId int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	if varId == 1 {
		for i := 1; i < 4; i++ {
			if i == 1 {
				tempVarData.maxlen = uint16(4)
			} else {
				dataBuffer.WriteString("/")
				tempVarData.maxlen = uint16(2)
			}
			tempVarData.id = uint16(i)
			tempVarData.startPos = uint16(lastvarPos)
			tempVarData.endPos = uint16(dataBuffer.Len() - lastvarPos)
			tempVarData.align = uint16(1)
			VarList = append(VarList, tempVarData)
			lastvarPos = int(tempVarData.endPos) + lastvarPos

		}
	} else {
		for i := 4; i < 7; i++ {
			if i == 4 {
				tempVarData.maxlen = uint16(2)
			} else {
				dataBuffer.WriteString(":")
				tempVarData.maxlen = uint16(2)
			}
			tempVarData.maxlen = uint16(2)
			tempVarData.id = uint16(i)
			tempVarData.startPos = uint16(lastvarPos)
			tempVarData.endPos = uint16(dataBuffer.Len() - lastvarPos)
			tempVarData.align = uint16(1)
			VarList = append(VarList, tempVarData)
			lastvarPos = int(tempVarData.endPos) + lastvarPos

		}
	}

	return dataBuffer, lastvarPos
}

func FindVarInfo(parseContentArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct

	for i := 0; i < len(parseContentArr)-1; i++ {
		if parseContentArr[i] == "TEXT" {
			dataBuffer.WriteString(parseContentArr[i+1])
			i = i + 1
		} else if parseContentArr[i] == "DATA" {
			varId, isFind := findVarID(parseContentArr[i+1])
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
					dataBuffer, lastvarPos = VarDateTime(dataBuffer, lastvarPos, varId)
				}
			} else {
				fmt.Println("Var name is not find ")
			}
		}
	}
	return dataBuffer, lastvarPos
}
