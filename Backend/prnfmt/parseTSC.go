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

// 解析TSC 打印机 指令协议是 TSPL2
//指令的特点是先发 Size

const (
	TSPL_TAIL           = "PRINT 1\r\n"
	TSPL_LINE_END       = "\r\n"
	TSPL_TEXT_LINE_HEAD = "TEXT "
	TSPL_INNER_LINE_SEP = ","
	TSPL_TEXT_SEP       = "\"" // 文本分割符""
	TSPL_LINE_HEAD      = "BAR "
	TSPL_BAR_CODE_HEAD  = "BARCODE "
	TSPL_RECTANGLE_HEAD = "X"
	TSPL_ROTATE_ZB      = "DIRECTION 0\r\n"
	TSPL_ROTATE_ZT      = "DIRECTION 1\r\n"
	TSPL_BAR_CODE_EXCEL = "barcode.xlsx"
	TSPL_LANGUAGE_TSPL  = "TSPL"
	TSPL_CLEAR          = "CLS\r\n"
)

var direction = TSPL_ROTATE_ZB

func ParseTscLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
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
		buf, lastvarPos = TscLines(rowArray, buf, lastvarPos, currentPath)
	}

	// 写入打印命令结尾
	buf.WriteString(TSPL_TAIL)
	return buf
}

func TscLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	switch line[0] {
	case "P":
		ParseTscPage(line, dataBuffer)
	case "TB":
		// fmt.Println(len(line))
		if line[10] == "TEXT" && len(line) == 13 {
			dataBuffer = ParseTscText(line, dataBuffer)
		} else if line[10] == "DATA" && len(line) == 16 {
			dataBuffer, lastvarPos = ParseTscVar(line, dataBuffer, lastvarPos)
		}

	case "B":
		dataBuffer, lastvarPos = ParseTscBarcode(line, dataBuffer, lastvarPos, path)
	case "ROTATE":
		dataBuffer = ParseTscRotate(line, dataBuffer)
	case "L":
		dataBuffer = ParseTscLine(line, dataBuffer)
	case "R":
		dataBuffer = ParseTscRectangle(line, dataBuffer)
	case "O":

	case "GP":

	case "QR":
		if len(line) > 9 {
			dataBuffer, lastvarPos = ParseTscQRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

// SIZE 100 mm,100 mm   宽度  高度
func ParseTscPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	//300DPI 纸张大小可能要转换
	width, _ := strconv.Atoi(tempRowArr[1])
	height, _ := strconv.Atoi(tempRowArr[2])

	if width != 0 && height != 0 {
		width = width / 8
		height = height / 8
	}

	SizeHead := "SIZE "
	FirstMM := " mm,"
	SecMM := " mm"

	dataBuffer.WriteString(SizeHead)
	dataBuffer.WriteString(strconv.Itoa(width)) // 宽度
	dataBuffer.WriteString(FirstMM)
	dataBuffer.WriteString(strconv.Itoa(height)) // 高度
	dataBuffer.WriteString(SecMM)
	dataBuffer.WriteString(TSPL_LINE_END)

	dataBuffer.WriteString(direction)
	dataBuffer.WriteString(TSPL_CLEAR) // 清除打印区域

	// fmt.Println(dataBuffer.String())
	return dataBuffer
}

// 解析TSC指令中的打印方向指令， DIRECTION 1 和 DIRECTION 0 （打印位置为出纸方向的标签左上角）
func ParseTscRotate(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	//此处写入过早，Size指令在前, 所以先写入Size指令
	if tempRowArr[1] == "0" {
		direction = TSPL_ROTATE_ZB
	} else if tempRowArr[1] == "2" {
		direction = TSPL_ROTATE_ZT
	}
	return dataBuffer
}

// L,63,133,153,133,2,0,0
// BAR x,y,width,height
// 起始位置  宽度和高度

func ParseTscLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
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
	dataBuffer.WriteString(TSPL_LINE_HEAD)

	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x = x * 12 / 8
	y = y * 12 / 8

	width = width * 12 / 8
	height = height * 12 / 8

	dataBuffer.WriteString(strconv.Itoa(x))
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(y))
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(width))
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(height))
	dataBuffer.WriteString(TSPL_LINE_END)

	return dataBuffer
}

// R,63,133,153,133,2,0,0
func ParseTscRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(TSPL_RECTANGLE_HEAD)
	dataBuffer.WriteString(tempRowArr[1])
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[2])
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[5])
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[3])
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[4])
	dataBuffer.WriteString(TSPL_LINE_END)

	return dataBuffer
}

// TB,12,43,130,30,0,1,1,2,0,TEXT,TIME,0
// TEXT 10,10,"2",0,1,1,"Human readable alignment"

func ParseTscText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = TscTextVarPosInfo(tempRowArr, dataBuffer)
	dataBuffer.WriteString(TSPL_TEXT_SEP)

	if len(tempRowArr[11]) > 0 {
		dataBuffer.WriteString(tempRowArr[11])
		dataBuffer.WriteString(TSPL_TEXT_SEP)
		dataBuffer.WriteString(TSPL_LINE_END)
	} else {
		dataBuffer.WriteString(TSPL_TEXT_SEP)
		dataBuffer.WriteString(TSPL_LINE_END)
	}
	return dataBuffer
}

func ParseTscVar(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarIDTsc(tempRowArr[11])

	if isFind {
		dataBuffer = TscTextVarPosInfo(tempRowArr, dataBuffer)
		dataBuffer.WriteString(TSPL_TEXT_SEP)
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
			dataBuffer, lastvarPos = VarDateTimeTsc(dataBuffer, lastvarPos, varId)
		}

		dataBuffer.WriteString(TSPL_TEXT_SEP)
		dataBuffer.WriteString(TSPL_LINE_END)

	} else {
		fmt.Println("can't find var id ,please check var name")
	}

	return dataBuffer, lastvarPos
}

// TB,80,146,70.0,30.0,4,1,1,0,0,TEXT,GROSS:,2
// X,Y,宽度，高度，字体大小,宽度倍数,高度倍数,样式,旋转,字体内容,序号
// 指令TEXT x,y,"font",rotation,x-multiplication,y-multiplication,[alignment,]"content"

// 0 : 不旋转
// 90: 顺时针旋转 90 度
// 180 :顺时针旋转 180 度
// 270 :顺时针旋转 270 度

func TscTextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {

	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x = (x*12 + 7) / 8
	y = (y*12 + 7) / 8
	//转坐标 300DPI  除以8 * 12 取整数
	dataBuffer.WriteString(TSPL_TEXT_LINE_HEAD)
	dataBuffer.WriteString(strconv.Itoa(x)) // X
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(y)) // Y
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	if tempRowArr[5] == "0" { // 字体大小不能为0
		tempRowArr[5] = "4"
	}

	isChinese := false

	//判断字体是不是中文
	if len(tempRowArr[11]) > 0 && containsGB2312(tempRowArr[11]) {
		tempRowArr[5] = "FONT001" // 中文字体
		isChinese = true
	}

	dataBuffer.WriteString(TSPL_TEXT_SEP)
	dataBuffer.WriteString(tempRowArr[5]) // 字体大小
	dataBuffer.WriteString(TSPL_TEXT_SEP)
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	rotation := "0"
	if tempRowArr[9] == "1" {
		rotation = "90"
	} else if tempRowArr[9] == "2" {
		rotation = "180"
	} else if tempRowArr[9] == "3" {
		rotation = "270"
	}
	dataBuffer.WriteString(rotation) // 旋转
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	//如果是中文，倍高和倍宽最少为3，最大为10
	if isChinese {
		if tempRowArr[6] == "1" {
			tempRowArr[6] = "3"
			tempRowArr[7] = "3"
		} else if tempRowArr[6] == "2" {
			tempRowArr[6] = "4"
			tempRowArr[7] = "4"
		} else if tempRowArr[6] == "3" {
			tempRowArr[6] = "5"
			tempRowArr[7] = "5"
		} else if tempRowArr[6] == "4" {
			tempRowArr[6] = "6"
			tempRowArr[7] = "6"
		} else if tempRowArr[6] == "5" {
			tempRowArr[6] = "7"
			tempRowArr[7] = "7"
		} else if tempRowArr[6] == "6" {
			tempRowArr[6] = "8"
			tempRowArr[7] = "8"
		} else if tempRowArr[6] == "7" {
			tempRowArr[6] = "9"
			tempRowArr[7] = "9"
		} else {
			tempRowArr[6] = "10"
			tempRowArr[7] = "10"
		}
	}
	dataBuffer.WriteString(tempRowArr[6]) // 宽度倍数
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[7]) // 高度倍数
	dataBuffer.WriteString(TSPL_INNER_LINE_SEP)

	return dataBuffer
}

/*
转换条码
*/
// B,27,460,150,80,2,CODE39,0,BC,TEXT,444,DATA,Gross,0123456,1,7,0
// B,24,333,150,80,2,1,0,BC,TEXT,555,TEXT,666,0
//指令
// BARCODE X,Y,”code type”,height,human readable,rotation,narrow,wide,[alignment,]”content“
//实际举例
//BARCODE 200,50,"128",100,2,0,2,2,"center"

// Height 条码高度(单位 dot)
// human readable
// 0: 不显示码文
// 1: 显示码文，码文左对齐
// 2: 显示码文，码文居中
// 3: 显示码文，码文右对齐
// rotation
// 0 : 不旋转
// 90 : 顺时针旋转 90 度
// 180 : 顺时针旋转 90 度
// 270 : 顺时针旋转 270 度
// narrow 窄比例因子宽度 (单位 dot)
// wide 宽比例因子宽度 (单位 dot)

func ParseTscBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	innerLineSep := ","
	dataSep := "\""
	bottom := "2" // 显示数据在条码下方，居中
	noCode := "0" //不显示数据
	parseContentArr := []string{}

	// X,Y 条码位置 转换为 300DPI 坐标
	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x = (x*12 + 7) / 8
	y = (y*12 + 7) / 8
	dataBuffer.WriteString(TSPL_BAR_CODE_HEAD) // lineHead
	dataBuffer.WriteString(strconv.Itoa(x))    // X
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(strconv.Itoa(y)) // Y
	dataBuffer.WriteString(innerLineSep)
	path = filepath.Join(path, TSPL_BAR_CODE_EXCEL)
	codeType := getBarCodeTypeId(path, TSPL_LANGUAGE_TSPL, tempRowArr[6])

	dataBuffer.WriteString(TSPL_TEXT_SEP)
	dataBuffer.WriteString(codeType)
	dataBuffer.WriteString(TSPL_TEXT_SEP)

	dataBuffer.WriteString(innerLineSep)
	height, _ := strconv.Atoi(tempRowArr[4])
	height = (height*12 + 7) / 8
	dataBuffer.WriteString(strconv.Itoa(height)) // 条码整体高度
	dataBuffer.WriteString(innerLineSep)
	if tempRowArr[8] == "BC" {
		dataBuffer.WriteString(bottom)
	} else {
		dataBuffer.WriteString(noCode)
	}
	dataBuffer.WriteString(innerLineSep)
	if tempRowArr[7] == "1" {
		tempRowArr[7] = "90"
	} else if tempRowArr[7] == "2" {
		tempRowArr[7] = "180"
	} else if tempRowArr[7] == "3" {
		tempRowArr[7] = "270"
	} else {
		tempRowArr[7] = "0"
	}
	dataBuffer.WriteString(tempRowArr[7]) // 旋转
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(tempRowArr[5]) // 条码中窄条的宽度
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(tempRowArr[5]) // 条码中宽条的宽度 TTC中只送了一个宽度
	dataBuffer.WriteString(innerLineSep)

	dataBuffer.WriteString(dataSep) // 条码数据开始
	for i := 9; i < len(tempRowArr); i++ {
		parseContentArr = append(parseContentArr, tempRowArr[i])
	}

	if len(parseContentArr) == 0 {
		fmt.Println("barCode data is missing ,please check barcode format")
	}

	dataBuffer, lastvarPos = FindVarInfoTsc(parseContentArr, dataBuffer, lastvarPos)
	dataBuffer.WriteString(dataSep)
	dataBuffer.WriteString(TSPL_LINE_END)
	return dataBuffer, lastvarPos
}

func findVarIDTsc(name string) (int, bool) {
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

// QRCODE x,y,  ECC Level,cell width,mode,rotation,[justification,]model,]mask,]area] "content"

// ECC level 纠错等级
// L : 7%
// M : 15%
// Q : 25%
// H : 30%
// cell width 1~10
// mode 自动 / 手动编码
// A : 自动
// M : 手动

// cell width 1~10

// QRCODE 310,310,M,6,A,0,M2,"ABCabc123"

func ParseTscQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	lineHead := "QRCODE "
	innerLineSep := ","
	dataSep := "\""
	QRcodeType := "M,"
	styleType := "A,"
	QRModel := "M2,"
	rotation := "0,"

	parseContentArr := []string{}
	dataBuffer.WriteString(lineHead)
	x, _ := strconv.Atoi(tempRowArr[1])
	x = (x*12 + 7) / 8
	y, _ := strconv.Atoi(tempRowArr[2])
	y = (y*12 + 7) / 8
	dataBuffer.WriteString(strconv.Itoa(x)) // X
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(strconv.Itoa(y)) // Y
	dataBuffer.WriteString(innerLineSep)

	dataBuffer.WriteString(QRcodeType) // M，
	sizedata, _ := strconv.Atoi(tempRowArr[4])

	if 0 < sizedata && sizedata <= 10 {
		dataBuffer.WriteString(tempRowArr[4]) // 二维码大小
	} else {
		dataBuffer.WriteString("6")
	}
	dataBuffer.WriteString(innerLineSep)
	dataBuffer.WriteString(styleType) // A,
	dataBuffer.WriteString(rotation)  // 0,
	dataBuffer.WriteString(QRModel)   // M2,

	dataBuffer.WriteString(dataSep) // 二维码数据开始
	for i := 7; i < len(tempRowArr); i++ {
		parseContentArr = append(parseContentArr, tempRowArr[i])
	}

	if len(parseContentArr) == 0 {
		fmt.Println("QRcode data is missing ,please check QR format")
	}
	dataBuffer, lastvarPos = FindVarInfoTsc(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString(dataSep)
	dataBuffer.WriteString(TSPL_LINE_END)
	return dataBuffer, lastvarPos
}

func VarDateTimeTsc(dataBuffer *bytes.Buffer, lastvarPos int, varId int) (*bytes.Buffer, int) {
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

func FindVarInfoTsc(parseContentArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct

	for i := 0; i < len(parseContentArr)-1; i++ {
		if parseContentArr[i] == "TEXT" {
			dataBuffer.WriteString(parseContentArr[i+1])
			i = i + 1
		} else if parseContentArr[i] == "DATA" {
			varId, isFind := findVarIDTsc(parseContentArr[i+1])
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
					dataBuffer, lastvarPos = VarDateTimeTsc(dataBuffer, lastvarPos, varId)
				}
			} else {
				fmt.Println("Var name is not find ")
			}
		}
	}
	return dataBuffer, lastvarPos
}
