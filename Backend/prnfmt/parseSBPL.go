package prnfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"tmaxsrv/comm"
)

const (
	SBPL_ESC      = "\x1b"
	SBPL_HEAD     = SBPL_ESC + "A"
	SBPL_TAIL     = SBPL_ESC + "Q1" + SBPL_ESC + "Z"
	SBPL_LINE_END = ""
)

func ParseSbplLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
	currentPath := comm.GetSrvDataPath()
	VarTable = ReadTableFromFile(currentPath + "/varTable.json")

	buf := dataBuffer
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(buff))

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	buf.WriteString(SBPL_HEAD)

	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		buf, lastvarPos = SbplLines(rowArray, buf, lastvarPos, currentPath)
	}

	buf.WriteString(SBPL_TAIL)
	return buf
}

func SbplLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	if len(line) == 0 {
		return dataBuffer, lastvarPos
	}

	switch line[0] {
	case "P":
		dataBuffer = ParseSbplPage(line, dataBuffer)
	case "TB":
		if len(line) >= 11 {
			if line[10] == "TEXT" {
				dataBuffer = ParseSbplText(line, dataBuffer)
			} else if line[10] == "DATA" {
				dataBuffer, lastvarPos = ParseSbplVar(line, dataBuffer, lastvarPos)
			}
		}
	case "B":
		dataBuffer, lastvarPos = ParsSbplBarcode(line, dataBuffer, lastvarPos, path)
	case "L":
		dataBuffer = ParseSbplLine(line, dataBuffer)
	case "R":
		dataBuffer = ParseSbplRectangle(line, dataBuffer)
	case "QR":
		if len(line) >= 9 {
			dataBuffer, lastvarPos = ParsSbplQRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

func ParseSbplPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	// 彻底去掉 A1 指令，避免页面尺寸设置导致坐标超出纸张边界
	return dataBuffer
}

func SbplTextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	dataBuffer.WriteString(fmt.Sprintf("%sV%04d%sH%04d", SBPL_ESC, y, SBPL_ESC, x))

	xs, _ := strconv.Atoi(tempRowArr[6])
	ys, _ := strconv.Atoi(tempRowArr[7])
	if xs <= 0 {
		xs = 1
	}
	if ys <= 0 {
		ys = 1
	}

	rot := tempRowArr[9]
	switch rot {
	case "90":
		dataBuffer.WriteString(fmt.Sprintf("%s%%1", SBPL_ESC))
	case "180":
		dataBuffer.WriteString(fmt.Sprintf("%s%%2", SBPL_ESC))
	case "270":
		dataBuffer.WriteString(fmt.Sprintf("%s%%3", SBPL_ESC))
	default:
		dataBuffer.WriteString(fmt.Sprintf("%s%%0", SBPL_ESC))
	}

	dataBuffer.WriteString(fmt.Sprintf("%sL%02d%02d", SBPL_ESC, xs, ys))
	dataBuffer.WriteString(SBPL_ESC + "XM")

	return dataBuffer
}

func ParseSbplText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = SbplTextVarPosInfo(tempRowArr, dataBuffer)
	if len(tempRowArr) > 11 {
		dataBuffer.WriteString(tempRowArr[11])
	}
	return dataBuffer
}

func ParseSbplVar(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarID(tempRowArr[11])
	if isFind {
		dataBuffer = SbplTextVarPosInfo(tempRowArr, dataBuffer)

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
	} else {
		fmt.Println("can't find var id ,please check var name")
	}
	return dataBuffer, lastvarPos
}

func ParseSbplLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 6 {
		return dataBuffer
	}
	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x1, _ := strconv.Atoi(tempRowArr[3])
	y1, _ := strconv.Atoi(tempRowArr[4])
	thickness, _ := strconv.Atoi(tempRowArr[5])

	w := x1 - x
	h := y1 - y
	if h == 0 {
		h = thickness
	}
	if w == 0 {
		w = thickness
	}
	if w < 0 {
		w = -w
		x = x1
	}
	if h < 0 {
		h = -h
		y = y1
	}

	dataBuffer.WriteString(fmt.Sprintf("%sV%04d%sH%04d", SBPL_ESC, y, SBPL_ESC, x))
	dataBuffer.WriteString(fmt.Sprintf("%sFW%02d%02dV%04dH%04d", SBPL_ESC, thickness, thickness, h, w))

	return dataBuffer
}

func ParseSbplRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 6 {
		return dataBuffer
	}
	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x1, _ := strconv.Atoi(tempRowArr[3])
	y1, _ := strconv.Atoi(tempRowArr[4])
	thickness, _ := strconv.Atoi(tempRowArr[5])

	w := x1 - x
	h := y1 - y
	if w < 0 {
		w = -w
		x = x1
	}
	if h < 0 {
		h = -h
		y = y1
	}

	dataBuffer.WriteString(fmt.Sprintf("%sV%04d%sH%04d", SBPL_ESC, y, SBPL_ESC, x))
	dataBuffer.WriteString(fmt.Sprintf("%sFW%02d%02dV%04dH%04d", SBPL_ESC, thickness, thickness, h, w))

	return dataBuffer
}

func ParsSbplBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	if len(tempRowArr) < 9 {
		return dataBuffer, lastvarPos
	}

	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	dataBuffer.WriteString(fmt.Sprintf("%sV%04d%sH%04d", SBPL_ESC, y, SBPL_ESC, x))

	// 获取参数
	typeName := strings.ToUpper(tempRowArr[6]) // 转换为大写进行匹配
	narrow, _ := strconv.Atoi(tempRowArr[5])
	if narrow <= 0 {
		narrow = 2 // 默认窄条宽度
	}
	height, _ := strconv.Atoi(tempRowArr[4])
	if height <= 0 {
		height = 40 // 默认高度，确保 HRI 可见
	}

	switch typeName {
	case "EAN13", "2":
		// 格式：BD + 类型(3) + 倍率(2位) + 高度(3位)
		dataBuffer.WriteString(fmt.Sprintf("%sBD3%02d%03d", SBPL_ESC, narrow, height))
	case "EAN8", "3":
		// 格式：BD + 类型(4) + 倍率(2位) + 高度(3位)
		dataBuffer.WriteString(fmt.Sprintf("%sBD4%02d%03d", SBPL_ESC, narrow, height))
	case "UPCA", "4":
		// 格式：BD + 类型(H) + 倍率(2位) + 高度(3位)
		dataBuffer.WriteString(fmt.Sprintf("%sBDH%02d%03d", SBPL_ESC, narrow, height))
	case "UPCE", "5":
		// 格式：BD + 类型(E) + 倍率(2位) + 高度(3位)
		dataBuffer.WriteString(fmt.Sprintf("%sDE%02d%03d", SBPL_ESC, narrow, height))
	case "CODE128", "1":
		// Code 128 使用 BG 指令，倍率(2位) + 高度(3位)
		dataBuffer.WriteString(fmt.Sprintf("%sBG%02d%03d", SBPL_ESC, narrow, height))
	case "CODE39", "0":
		// 格式：D1 + 2位倍率 + 3位高度 (1:3比例，更易扫描)
		dataBuffer.WriteString(fmt.Sprintf("%sD1%02d%03d", SBPL_ESC, narrow, height))
	default:
		// 不支持的类型直接返回，不生成指令
		return dataBuffer, lastvarPos
	}

	// 针对 Code 39 自动补全起止符 *，确保扫描成功
	isCode39 := (typeName == "CODE39" || typeName == "0")
	if isCode39 {
		dataBuffer.WriteString("*")
	}

	parseContentArr := tempRowArr[9:]
	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	if isCode39 {
		dataBuffer.WriteString("*")
	}

	return dataBuffer, lastvarPos
}

func ParsSbplQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	if len(tempRowArr) < 7 {
		return dataBuffer, lastvarPos
	}

	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	dataBuffer.WriteString(fmt.Sprintf("%sV%04d%sH%04d", SBPL_ESC, y, SBPL_ESC, x))

	parseContentArr := tempRowArr[7:]
	totalLen := GetQRDataLen(parseContentArr)

	size := tempRowArr[4]
	if size == "" || size == "0" {
		size = "5"
	}
	sizeInt, _ := strconv.Atoi(size)

	// 使用经过验证的 2D30 指令 (Model 2 QR)
	dataBuffer.WriteString(fmt.Sprintf("%s2D30,M,%02d,0,0", SBPL_ESC, sizeInt))
	// 使用经过验证的 DN 指令指定数据长度
	dataBuffer.WriteString(fmt.Sprintf("%sDN%04d,", SBPL_ESC, totalLen))

	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	return dataBuffer, lastvarPos
}
