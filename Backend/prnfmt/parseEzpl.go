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
	EZPL_HEAD           = "^L" + EZPL_LINE_END
	EZPL_TAIL           = "E" + EZPL_LINE_END
	EZPL_LINE_END       = "\r\n"
	EZPL_TEXT_LINE_HEAD = "AC"
	EZPL_INNER_LINE_SEP = ","
	EZPL_TEXT_SEP       = "" // EZPL typically doesn't use quotes for AT command unless data starts with space
	EZPL_BARCODE_HEAD   = "B"
	EZPL_QRCODE_HEAD    = "W"
	EZPL_LINE_HEAD      = "Lo"
	EZPL_RECTANGLE_HEAD = "Re"
)

func ParseEzplLines(buff string, dataBuffer *bytes.Buffer, lastvarPos int) *bytes.Buffer {
	// 获取VarTable.json和barcode.xlsx文件路径
	currentPath := comm.GetSrvDataPath()
	VarTable = ReadTableFromFile(currentPath + "/varTable.json") // 获取变量ID表

	// 处理字符串并解析
	buf := dataBuffer
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(buff))

	for scanner.Scan() {
		lines = append(lines, scanner.Text()) // 每行字符串添加到切片
	}

	// Configuration and Label start will be handled by ParseEzplPage (P command)
	for i := 0; i < len(lines); i++ {
		rowArray := strings.Split(lines[i], ",")
		buf, lastvarPos = EzplLines(rowArray, buf, lastvarPos, currentPath)
	}

	// 写入打印命令结尾
	buf.WriteString(EZPL_TAIL)
	return buf
}

func EzplLines(line []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	if len(line) == 0 {
		return dataBuffer, lastvarPos
	}

	switch line[0] {
	case "P":
		dataBuffer = ParseEzplPage(line, dataBuffer)
	case "TB":
		if len(line) >= 11 {
			if line[10] == "TEXT" {
				dataBuffer = ParseEzplText(line, dataBuffer)
			} else if line[10] == "DATA" {
				dataBuffer, lastvarPos = ParseEzplVar(line, dataBuffer, lastvarPos)
			}
		}
	case "B":
		dataBuffer, lastvarPos = ParsEzplBarcode(line, dataBuffer, lastvarPos, path)
	case "L":
		dataBuffer = ParseEzplLine(line, dataBuffer)
	case "R":
		dataBuffer = ParseEzplRectangle(line, dataBuffer)
	case "QR":
		if len(line) >= 9 {
			dataBuffer, lastvarPos = ParsEzplQRcode(line, dataBuffer, lastvarPos)
		}
	}
	return dataBuffer, lastvarPos
}

func ParseEzplPage(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	// P,width(dots),height(dots)
	if len(tempRowArr) < 3 {
		return dataBuffer
	}

	wDots, _ := strconv.Atoi(tempRowArr[1])
	hDots, _ := strconv.Atoi(tempRowArr[2])

	// Convert dots to mm (assuming 8 dots/mm)
	width := wDots / 8
	height := hDots / 8

	// Configuration commands (should be outside ^L...E block or at start)
	// ^W width (mm)
	dataBuffer.WriteString("^W" + strconv.Itoa(width) + EZPL_LINE_END)
	// ^Q height (mm)
	dataBuffer.WriteString("^Q" + strconv.Itoa(height) + EZPL_LINE_END)

	// Start Label command
	dataBuffer.WriteString(EZPL_HEAD) // ^L

	return dataBuffer
}

// AT,x,y,xs,ys,gh,rot,cls,data
func ParseEzplText(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer = EzplTextVarPosInfo(tempRowArr, dataBuffer)
	// EZPL data part
	if len(tempRowArr) > 11 {
		dataBuffer.WriteString(tempRowArr[11])
	}
	dataBuffer.WriteString(EZPL_LINE_END)
	return dataBuffer
}

func ParseEzplVar(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	var tempVarData VarStruct
	varId, isFind := findVarID(tempRowArr[11])
	if isFind {
		dataBuffer = EzplTextVarPosInfo(tempRowArr, dataBuffer)

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
		dataBuffer.WriteString(EZPL_LINE_END)
	} else {
		fmt.Println("can't find var id ,please check var name")
	}
	return dataBuffer, lastvarPos
}

// AD,x,y,x_mul,y_mul,gap,rotationInverse,data
func EzplTextVarPosInfo(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	dataBuffer.WriteString(EZPL_TEXT_LINE_HEAD)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[2]) // Y
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// x_mul,y_mul (Magnification)
	xs := tempRowArr[6]
	if xs == "0" || xs == "" {
		xs = "1"
	}
	dataBuffer.WriteString(xs)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	ys := tempRowArr[7]
	if ys == "0" || ys == "" {
		ys = "1"
	}
	dataBuffer.WriteString(ys)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// gap (Gap between characters)
	dataBuffer.WriteString("0")
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// rotationInverse (Rotation: 0, 90, 180, 270 -> 0, 1, 2, 3)
	rot := "0"
	switch tempRowArr[9] {
	case "0":
		rot = "0"
	case "90":
		rot = "1"
	case "180":
		rot = "2"
	case "270":
		rot = "3"
	default:
		rot = "0"
	}
	dataBuffer.WriteString(rot)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	return dataBuffer
}

// Lo,x,y,x1,y1
func ParseEzplLine(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 6 {
		return dataBuffer
	}

	x, _ := strconv.Atoi(tempRowArr[1])
	y, _ := strconv.Atoi(tempRowArr[2])
	x1, _ := strconv.Atoi(tempRowArr[3])
	y1, _ := strconv.Atoi(tempRowArr[4])
	thickness, _ := strconv.Atoi(tempRowArr[5])

	if y == y1 { // Horizontal line
		y1 = y + thickness
	} else if x == x1 { // Vertical line
		x1 = x + thickness
	}

	dataBuffer.WriteString(EZPL_LINE_HEAD)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(x))
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(y))
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(x1))
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(strconv.Itoa(y1))
	dataBuffer.WriteString(EZPL_LINE_END)

	return dataBuffer
}

// Re,x,y,x1,y1,t1,t2
func ParseEzplRectangle(tempRowArr []string, dataBuffer *bytes.Buffer) *bytes.Buffer {
	if len(tempRowArr) < 6 {
		return dataBuffer
	}
	dataBuffer.WriteString(EZPL_RECTANGLE_HEAD)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[1]) // x
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[2]) // y
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[3]) // x1
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[4]) // y1
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[5]) // t1
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[5]) // t2 (use same)
	dataBuffer.WriteString(EZPL_LINE_END)
	return dataBuffer
}

// Bt,x,y,narrow,wide,height,rotation,readable,data
func ParsEzplBarcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int, path string) (*bytes.Buffer, int) {
	if len(tempRowArr) < 9 {
		return dataBuffer, lastvarPos
	}

	dataBuffer.WriteString(EZPL_BARCODE_HEAD)
	// t (Type)
	barcodeExcelPath := filepath.Join(path, EPL_BAR_CODE_EXCEL)
	codeType := getBarCodeTypeId(barcodeExcelPath, "EZPL", tempRowArr[6]) // Need EZPL column in excel
	if codeType == "" {
		codeType = "Q" // Default to Code39 or similar for EZPL
	}
	dataBuffer.WriteString(codeType)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[2]) // Y
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// narrow, wide
	dataBuffer.WriteString(tempRowArr[5]) // narrow
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[5]) // wide (simple mapping)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	dataBuffer.WriteString(tempRowArr[4]) // height
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// rotation (0-3)
	rot := "0"
	switch tempRowArr[7] {
	case "0":
		rot = "0"
	case "90":
		rot = "1"
	case "180":
		rot = "2"
	case "270":
		rot = "3"
	default:
		rot = "0"
	}
	dataBuffer.WriteString(rot)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// readable (0/1)
	readable := "0"
	if tempRowArr[8] != "N" {
		readable = "1"
	}
	dataBuffer.WriteString(readable)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// data
	parseContentArr := tempRowArr[9:]
	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString(EZPL_LINE_END)
	return dataBuffer, lastvarPos
}

// Wx,y,mode,type,ec,mask,mul,len,rotate
func ParsEzplQRcode(tempRowArr []string, dataBuffer *bytes.Buffer, lastvarPos int) (*bytes.Buffer, int) {
	if len(tempRowArr) < 7 {
		return dataBuffer, lastvarPos
	}

	parseContentArr := tempRowArr[7:]
	totalLen := GetQRDataLen(parseContentArr)

	dataBuffer.WriteString(EZPL_QRCODE_HEAD)
	dataBuffer.WriteString(tempRowArr[1]) // X
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString(tempRowArr[2]) // Y
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	dataBuffer.WriteString("2") // mode: 2 (usually for generic data)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString("2") // type: 2 (auto)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString("L") // ec: L (Medium)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)
	dataBuffer.WriteString("8") // mask: 7
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// multiplier (size)
	size := tempRowArr[4]
	if size == "" || size == "0" {
		size = "6"
	}
	dataBuffer.WriteString(size)
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	// len (Data Length) - calculate exactly for EZPL
	dataBuffer.WriteString(fmt.Sprintf("%d", totalLen))
	dataBuffer.WriteString(EZPL_INNER_LINE_SEP)

	dataBuffer.WriteString("0")           // rotate
	dataBuffer.WriteString(EZPL_LINE_END) // <CR> before data

	// data
	dataBuffer, lastvarPos = FindVarInfo(parseContentArr, dataBuffer, lastvarPos)

	dataBuffer.WriteString(EZPL_LINE_END)
	return dataBuffer, lastvarPos
}

func GetQRDataLen(parseContentArr []string) int {
	totalLen := 0
	for i := 0; i < len(parseContentArr)-1; i++ {
		if parseContentArr[i] == "TEXT" {
			totalLen += len(parseContentArr[i+1])
			i = i + 1
		} else if parseContentArr[i] == "DATA" {
			varId, isFind := findVarID(parseContentArr[i+1])
			i = i + 1
			if isFind {
				if varId == 1 { // Date YYYY/MM/DD
					totalLen += 10
				} else if varId == 2 { // Time HH:MM:SS
					totalLen += 8
				} else {
					// 寻找 maxlen: DATA (i-1), name (i), dummy (i+1), align (i+2), maxlen (i+3)
					if i+3 < len(parseContentArr) {
						maxLen, err := strconv.Atoi(parseContentArr[i+3])
						if err == nil {
							totalLen += maxLen
						}
						// 跳过 DATA 的后续 3 个参数
						i = i + 3
					}
				}
			}
		}
	}
	return totalLen
}
