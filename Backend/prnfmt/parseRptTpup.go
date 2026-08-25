package prnfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"tmaxsrv/comm"
)

var (
	TPUP_LINE_END         = "\n"       // 0x0A 换行符
	TPUP_TAIL             = "\n\n\n\n" // 结尾发送 4 个 0x0A
	TPUP_LINE_NUM         = 0
	TPUP_CURRENT_X        = 0   // 当前行已使用的字符数
	TPUP_LOOP_FLAG        = 0
	TPUP_LINE_HEIGHT      = 31.0 // 票据行高 (dot/行)
	TPUP_CHAR_WIDTH       = 12.0 // 每个字符宽度 (dot/字符)
	TPUP_LEFT_MARGIN_DOTS = 0.0  // 左侧不可打印边距点数偏移
)

var TpupFirstLoopY = 0  // 记录第一行循环变量的Y坐标
var TpupSecondLoopY = 0 // 记录第二行循环变量的Y坐标
var TpupLoopYNum = 0    // 记录循环变量的Y行数数量

// ParseRptTpupLines 解析票据格式设计字符串并转换为 TPUP 打印协议
func ParseRptTpupLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
	currentPath := comm.GetSrvDataPath()
	VarTable = ReadTableFromFile(currentPath + "/varTable.json") // 获取变量ID表

	buf := dataBuffer
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(buff))

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return buf
	}

	var countLoop = 0
	var firstLoopIndex = 0
	var secondLoopIndex = 0
	TpupFirstLoopY = 0
	TpupSecondLoopY = 0
	TpupLoopYNum = 0

	// 去掉负数位置
	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		if strings.Contains(lines[i], "TB") && len(rowArray) > 10 {
			rowArray[1] = strings.ReplaceAll(rowArray[1], "-", "")
			rowArray[2] = strings.ReplaceAll(rowArray[2], "-", "")
			lines[i] = strings.Join(rowArray, ",")
		}
	}

	// 循环变量 StartLoop 处理
	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		if strings.Contains(lines[i], "DATA,StartLoop") {
			countLoop += 1
			if countLoop == 1 && len(rowArray) > 2 {
				firstLoopIndex = i
				TpupFirstLoopY, _ = strconv.Atoi(rowArray[2])
			} else if countLoop == 2 && len(rowArray) > 2 {
				secondLoopIndex = i
				TpupSecondLoopY, _ = strconv.Atoi(rowArray[2])
				break
			}
		}
	}

	TpupLoopYNum = int(math.Round(math.Abs(float64(TpupSecondLoopY-TpupFirstLoopY) / TPUP_LINE_HEIGHT)))

	if TpupLoopYNum > 0 && strings.Contains(lines[firstLoopIndex], "DATA,StartLoop") && strings.Contains(lines[secondLoopIndex], "DATA,StartLoop") {
		firstLine := lines[firstLoopIndex]
		parts := strings.Split(firstLine, ",")
		parts[14] = strconv.Itoa(TpupLoopYNum)
		result := strings.Join(parts, ",")
		lines[firstLoopIndex] = result

		secondLine := lines[secondLoopIndex]
		parts2 := strings.Split(secondLine, ",")
		parts2[14] = strconv.Itoa(TpupLoopYNum)
		result2 := strings.Join(parts2, ",")
		lines[secondLoopIndex] = result2

		// 处理 Y 坐标
		for i := 0; i < len(lines); i++ {
			rowArray := strings.Split(lines[i], ",")
			if len(rowArray) > 10 {
				var yPos, _ = strconv.Atoi(rowArray[2])
				if yPos >= TpupFirstLoopY && i != firstLoopIndex && i != secondLoopIndex {
					rowArray[2] = "*#" + rowArray[2]
					lines[i] = strings.Join(rowArray, ",")
				}
			}
		}
	}

	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		buf, lastvarPos = tpupLines(rowArray, buf, lastvarPos, currentPath)
	}

	fmt.Println("ParseRptTpupLines finished, buffer length:", buf.Len())

	// 写入打印命令结尾 4 个 0x0A
	buf.WriteString(TPUP_TAIL)
	return buf
}

func tpupLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	switch line[0] {
	case "P":
		TPUP_LINE_NUM = 0
		TPUP_CURRENT_X = 0
		TPUP_LOOP_FLAG = 0
		ParseRptTpupPage(line, dataBuffer)
	case "TB":
		if line[10] == "TEXT" && len(line) == 13 {
			dataBuffer = ParseRptTpupText(line, dataBuffer)
		} else if line[10] == "DATA" && len(line) == 16 {
			dataBuffer = ParseRptTpupVar(line, dataBuffer)
		}
	// 忽略条码(B)和二维码(QR)等复杂图形，TPUP 打印协议无需解析，直接过滤丢弃
	}
	return dataBuffer, lastvarPos
}

func ParseRptTpupPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	// TPUP 打印机无特定初始化指令
	return dataBuffer
}

func ParseRptTpupText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = tpupTextVarPosInfo(tempRowArr, dataBuffer)
	if len(tempRowArr[11]) > 0 {
		textVal := tempRowArr[11]
		dataBuffer.WriteString(textVal)
		TPUP_CURRENT_X += len(textVal)
	}
	return dataBuffer
}

func ParseRptTpupVar(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	var tempVarData RptVarStruct
	varId, isFind := findTpupVarID(tempRowArr[11])
	if varId == 254 && TPUP_LOOP_FLAG == 0 {
		TPUP_LOOP_FLAG++
	} else if varId == 254 && TPUP_LOOP_FLAG == 1 {
		varId = 255
		TPUP_LOOP_FLAG = 0
	}
	if isFind {
		dataBuffer = tpupTextVarPosInfo(tempRowArr, dataBuffer)

		// 变量为日期时间格式需要单独处理
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
			TPUP_CURRENT_X += int(tempVarData.maxLen)
		} else {
			dataBuffer = VarTpupDateTime(dataBuffer, varId)
		}
	} else {
		fmt.Println("can't find var id, please check var name:", tempRowArr[11])
	}

	return dataBuffer
}

func tpupTextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	xOffset, err := strconv.Atoi(tempRowArr[1])
	if err != nil {
		return dataBuffer
	}
	yOffset, err := strconv.Atoi(strings.ReplaceAll(tempRowArr[2], "*#", ""))
	if err != nil {
		return dataBuffer
	}

	// 1. 换行计算 (Y 坐标 -> \n 0x0A)
	tempYLine := int(math.Round(float64(yOffset) / TPUP_LINE_HEIGHT))
	if tempYLine > TPUP_LINE_NUM {
		lineCount := tempYLine - TPUP_LINE_NUM
		for i := 0; i < lineCount; i++ {
			dataBuffer.WriteString(TPUP_LINE_END)
		}
		TPUP_LINE_NUM = tempYLine
		TPUP_CURRENT_X = 0 // 换行后重置水平字符索引
	}

	// 2. 左边距点数补偿
	adjX := float64(xOffset) - TPUP_LEFT_MARGIN_DOTS
	if adjX < 0 {
		adjX = 0
	}
	targetCharCol := int(math.Round(adjX / TPUP_CHAR_WIDTH))

	// 3. 空格填充 (X 坐标 -> ' ' 0x20)
	if targetCharCol > TPUP_CURRENT_X {
		spaceCount := targetCharCol - TPUP_CURRENT_X
		for i := 0; i < spaceCount; i++ {
			dataBuffer.WriteString(" ")
		}
		TPUP_CURRENT_X = targetCharCol
	}

	return dataBuffer
}

func findTpupVarID(name string) (int, bool) {
	lenth := len(VarTable.ScaleVarTable)
	isFind := false
	id := 0
	for i := 0; i < lenth; i++ {
		if name == VarTable.ScaleVarTable[i].ValueVame {
			id = VarTable.ScaleVarTable[i].Id
			isFind = true
			break
		}
	}
	return id, isFind
}

func VarTpupDateTime(dataBuffer *bytes.Buffer, varId int) *bytes.Buffer {
	var tempVarData RptVarStruct
	if varId == 1 {
		for i := 1; i < 4; i++ {
			if i == 1 {
				tempVarData.maxLen = uint8(4)
			} else {
				dataBuffer.WriteString("/")
				TPUP_CURRENT_X += 1
				tempVarData.maxLen = uint8(2)
			}
			tempVarData.id = uint8(i)
			tempVarData.startPos = uint16(dataBuffer.Len())
			tempVarData.align = uint8(1)
			RptVarList = append(RptVarList, tempVarData)
			TPUP_CURRENT_X += int(tempVarData.maxLen)
		}
	} else {
		for i := 4; i < 7; i++ {
			if i == 4 {
				tempVarData.maxLen = uint8(2)
			} else {
				dataBuffer.WriteString(":")
				TPUP_CURRENT_X += 1
				tempVarData.maxLen = uint8(2)
			}
			tempVarData.id = uint8(i)
			tempVarData.startPos = uint16(dataBuffer.Len())
			tempVarData.align = uint8(1)
			RptVarList = append(RptVarList, tempVarData)
			TPUP_CURRENT_X += int(tempVarData.maxLen)
		}
	}
	return dataBuffer
}
