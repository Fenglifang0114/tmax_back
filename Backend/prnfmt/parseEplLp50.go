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

func ParseEplLp50Lines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
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
		buf, lastvarPos = EplLp50Lines(rowArray, buf, lastvarPos, currentPath)
	}

	// 写入打印命令结尾
	buf.WriteString(EPL_TAIL)
	return buf
}

func EplLp50Lines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	switch line[0] {
	case "P":
		ParseEplPage(line, dataBuffer)
	case "TB":
		// fmt.Println(len(line))
		if line[10] == "TEXT" && len(line) == 13 {
			dataBuffer = ParseEplLp50Text(line, dataBuffer)
		} else if line[10] == "DATA" && len(line) == 16 {
			dataBuffer, lastvarPos = ParseEplLp50Var(line, dataBuffer, lastvarPos)
		}

	case "B":
		dataBuffer, lastvarPos = ParsEplLp50Barcode(line, dataBuffer, lastvarPos, path)
	case "ROTATE":
		dataBuffer = ParseEplLp50Rotate(line, dataBuffer)
	case "L":
		dataBuffer = ParseEplLp50Line(line, dataBuffer)
	case "R":
		dataBuffer = ParseEplLp50Rectangle(line, dataBuffer)
	case "O":

	case "GP":

	case "QR":
		if len(line) >= 9 {
			dataBuffer, lastvarPos = ParsEplLp50QRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

func ParseEplLp50Page(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	widthHead := "q"
	hightHead := "Q"
	paperGap := ",24"

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
func ParseEplLp50Rotate(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if tempRowArr[1] == "0" {
		dataBuffer.WriteString(EPL_ROTATE_ZT)
	} else if tempRowArr[1] == "2" {
		dataBuffer.WriteString(EPL_ROTATE_ZB)
	}
	dataBuffer.WriteString(EPL_LINE_END)
	return dataBuffer
}

// L,63,133,153,133,2,0,0
// 打印机指令LS46,302,3,346,302
func ParseEplLp50Line(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(EPL_LINE_HEAD)
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

// R,63,133,153,133,2,0,0
func ParseEplLp50Rectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
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
func ParseEplLp50Text(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = TextVarPosInfoLp50(tempRowArr, dataBuffer)
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

func ParseEplLp50Var(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarID(tempRowArr[11])
	if isFind {
		dataBuffer = TextVarPosInfoLp50(tempRowArr, dataBuffer)
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

func TextVarPosInfoLp50(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
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
	if len(tempRowArr) > 12 && containsGB2312(tempRowArr[11]) && tempRowArr[11] != "" {
		tempRowArr[5] = "6"
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

func containsGB2312(s string) bool {
	bytes := []byte(s)
	for i := 0; i < len(bytes)-1; i++ {
		first := bytes[i]
		second := bytes[i+1]
		if first >= 0xA1 && first <= 0xF7 && second >= 0xA1 && second <= 0xFE {
			return true
		}
	}
	return false
}

/*
转换条码
*/
// content: TEXT,2,TEXT,0,TEXT,0,DATA,Net,01234567,3,8,TEXT,7,TEXT,8,DATA,Net,01234567,3,8

func ParsEplLp50Barcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
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

// QR,248,176,1,6,1,,DATA,Net,000000,0,6,DATA,Unit_weight,000000,0,4,TEXT,SN9000,0

// 转换后 b100,200,Q,m2,s6,"123456789"  //Q 代表QR m2 代表模式2，样式， s6代表大小  “” 数据
func ParsEplLp50QRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
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
