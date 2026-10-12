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

const EPL_ZEBRA_HEAD = "FK\"AA\"\r\nFS\"AA\"\r\nR0,0\r\n"

func ParseEplZebraLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
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
		buf, lastvarPos = EplZebraLines(rowArray, buf, lastvarPos, currentPath)
	}
	// 写入打印命令结尾
	buf.WriteString(EPL_TAIL)
	return buf
}

func EplZebraLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	switch line[0] {
	case "P":
		ParseEplZebraPage(line, dataBuffer)
	case "TB":
		// fmt.Println(len(line))
		if line[10] == "TEXT" && len(line) == 13 {
			dataBuffer = ParseEplZebraText(line, dataBuffer)
		} else if line[10] == "DATA" && len(line) == 16 {
			dataBuffer, lastvarPos = ParseEplZebraVar(line, dataBuffer, lastvarPos)
		}
	case "B":
		dataBuffer, lastvarPos = ParsEplZebarBarcode(line, dataBuffer, lastvarPos, path)
	case "ROTATE":
		dataBuffer = ParseEplRotate(line, dataBuffer)
	case "L":
		dataBuffer = ParseEplZebraLine(line, dataBuffer)
	case "R":
		dataBuffer = ParseEplRectangle(line, dataBuffer)
	case "O":
	case "GP":
	case "QR":
		if len(line) > 9 {
			dataBuffer, lastvarPos = ParsZebarEplQRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

func getZebraPageSize(pageDots string) string {
	if strings.Contains(pageDots, "-") {
		pageDots = "0"
	}
	num, err := strconv.ParseFloat(pageDots, 64)
	if err != nil {
		return pageDots
	}
	result := num / 8 * 11.8
	resultStr := strconv.Itoa(int(result))
	return resultStr
}

func ParseEplZebraPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	widthHead := "q"
	hightHead := "Q"
	paperGap := ",24"
	dataBuffer.WriteString(EPL_ZEBRA_HEAD)
	dataBuffer.WriteString(hightHead)
	tempRowArr[2] = strings.Replace(tempRowArr[2], "\r\n", "", 1)
	tempRowArr[1] = strings.Replace(tempRowArr[1], "\r\n", "", 1)
	dotsHeigh := getZebraPageSize(tempRowArr[2])
	dataBuffer.WriteString(dotsHeigh) // 高度
	dataBuffer.WriteString(paperGap)
	dataBuffer.WriteString(EPL_LINE_END)
	dotsWidth := getZebraPageSize(tempRowArr[1])
	dataBuffer.WriteString(widthHead)
	dataBuffer.WriteString(dotsWidth) // 宽度
	dataBuffer.WriteString(EPL_LINE_END)
	// fmt.Println(dataBuffer.String())
	return dataBuffer
}

// L,63,133,153,133,2,0,0
func ParseEplZebraRotate(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if tempRowArr[1] == "0" {
		dataBuffer.WriteString(EPL_ROTATE_ZB)
	} else if tempRowArr[1] == "2" {
		dataBuffer.WriteString(EPL_ROTATE_ZT)
	}
	dataBuffer.WriteString(EPL_LINE_END)
	return dataBuffer
}

// L,63,133,153,133,2,0,0
// LS183,53,3,183,353 打印机指令 (x1,y1,lineWidth, x2,y2)
func ParseEplZebraLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(EPL_LINE_HEAD)
	x1 := getZebraPageSize(tempRowArr[1])
	dataBuffer.WriteString(x1)
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	y1 := getZebraPageSize(tempRowArr[2])
	dataBuffer.WriteString(y1)
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[5])
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	x2 := getZebraPageSize(tempRowArr[3])
	dataBuffer.WriteString(x2)
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	y2 := getZebraPageSize(tempRowArr[4])
	dataBuffer.WriteString(y2)
	dataBuffer.WriteString(EPL_LINE_END)
	return dataBuffer
}

// R,63,133,153,133,2,0,0
func ParseEplZebarRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
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
func ParseEplZebraText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = TextZebraVarPosInfo(tempRowArr, dataBuffer)
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

func ParseEplZebraVar(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarID(tempRowArr[11])

	if isFind {
		dataBuffer = TextZebraVarPosInfo(tempRowArr, dataBuffer)
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

func TextZebraVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(EPL_TEXT_LINE_HEAD)
	dotsX := getZebraPageSize(tempRowArr[1])
	dataBuffer.WriteString(dotsX) // X
	dataBuffer.WriteString(EPL_INNER_LINE_SEP)
	dotsY := getZebraPageSize(tempRowArr[2])
	dataBuffer.WriteString(dotsY) // Y
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
//B10,115,0,1,2,2,45,B,V00
func ParsEplZebarBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	lineHead := "B"
	innerLineSep := ","
	dataSep := "\""
	// bottom := "BC" // 显示数据在条码下方，居中
	// // noCode := "N"  //不显示数据
	// top := "TC" // 显示数据在条码上方，居中（没有适配的打印机命令）
	parseContentArr := []string{}

	dataBuffer.WriteString(lineHead)
	dotsX := getZebraPageSize(tempRowArr[1])
	dataBuffer.WriteString(dotsX) // X
	// dataBuffer.WriteString(tempRowArr[1]) // X

	dataBuffer.WriteString(innerLineSep)
	dotsY := getZebraPageSize(tempRowArr[2])
	dataBuffer.WriteString(dotsY) // X
	// dataBuffer.WriteString(tempRowArr[2]) // Y

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
	dotsH := getZebraPageSize(tempRowArr[4])
	dataBuffer.WriteString(dotsH) // X
	// dataBuffer.WriteString(tempRowArr[4]) // 条码整体高度
	dataBuffer.WriteString(innerLineSep)
	// if tempRowArr[8] == top {
	// 	dataBuffer.WriteString(bottom)
	// } else {
	// 	dataBuffer.WriteString(tempRowArr[8])
	// }
	dataBuffer.WriteString("B")
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

func findZebarVarID(name string) (int, bool) {
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
//
//	b50,50,Q,m2,s20,"www.zebra.com"
func ParsZebarEplQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	lineHead := "b"
	innerLineSep := ","
	dataSep := "\""
	QRcodeType := "Q,"
	styleType := "m2,"
	QRsize := "s"

	parseContentArr := []string{}
	dataBuffer.WriteString(lineHead)
	dotsX := getZebraPageSize(tempRowArr[1])
	dataBuffer.WriteString(dotsX) // X
	// dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(innerLineSep)

	dotsY := getZebraPageSize(tempRowArr[2])
	dataBuffer.WriteString(dotsY) // Y
	// dataBuffer.WriteString(tempRowArr[2]) // Y
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

func VarZebraDateTime(dataBuffer *bytes.Buffer, lastvarPos int, varId int) (*bytes.Buffer, int) {
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

func FindZebraVarInfo(parseContentArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
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
