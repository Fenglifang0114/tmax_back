package prnfmt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

type InfoStr struct {
	addr      uint32
	formatLen uint32
	varNum    uint32
}
type printInfo struct {
	printerName     [23]byte //22个用于存储打印机的名字，后面的一个用于写类型，比如0是lable，1是receipt
	formatNum       uint8
	everyFormatInfo [12]InfoStr
	varInfo         [30]VarStruct
}

type printInfoRpt struct {
	printerName     [23]byte
	formatNum       uint8
	everyFormatInfo [12]InfoStr
	varInfo         [60]RptVarStruct
}

type printInfoDef struct {
	printerName     [23]byte
	formatNum       uint8
	everyFormatInfo [12]InfoStr    //最大12个打印格式
	varInfo         [150]VarStruct //最大150个变量
}

const (
	headBufLen        = 468  //一个printInfo占用的字节是固定的。
	headBufLenReceipt = 468  //一个printInfoReceipt占用的字节是固定的。
	headBufLenDef     = 1668 //默认打印格式占用的字节是固定的。
)

var (
	FMT_FILL_TAIL      []byte = []byte{0x5a, 0xa5, 0xa5, 0x5a, 0x00, 0x00, 0x00, 0x00}
	ESC_CHANGE_EPL_205        = []byte{0x1F, 0x28, 0x4C, 0x03, 0x00, 0x43, 0x45, 0x06}
)

func findPrinterName(s string) string {
	parts := strings.Split(s, "\r\n")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}
func ParserFmtToFile(utf8Buff string, printerModel string, fmtLen int) bool {
	var buffer *bytes.Buffer
	printMode := printerModel
	printerName := ""
	prtInfo := findPrinterName(utf8Buff)
	if strings.Contains(prtInfo, "F,") {
		parts := strings.Split(prtInfo, ",")
		if len(parts) == 3 {
			printerName = parts[1]
			if parts[2] == "L" {
				printMode = "Lable"
			} else {
				printMode = "Receipt"
			}

		}

	}
	if printMode == "Lable" {
		//此处需要进一步判断是哪个打印机，哪种模式，上面的Lable只是初步判断是从标签格式下发的路径来的
		buffer = ParserFmtToBuf(utf8Buff, printerName, fmtLen)
	} else {
		buffer = ParserRptFmtToBuf(utf8Buff, printerName, fmtLen)
	}
	// 7.创建bin文件

	if creatFile("formatBin.bin", buffer) {
		fmt.Println("creat binary file success")
		return true
	} else {
		fmt.Println("creat binary file fail")
		return false
	}

	// 8.写数据到串口
}

func ParserFmtToBuf(utf8Buff string, printerModel string, fmtLen int) *bytes.Buffer {
	var clearList []VarStruct
	VarList = clearList // 用于清空数据

	var clearTable ScaleVarOrder
	VarTable = clearTable // 用于清空数据

	var FinalFormatInfo printInfo
	var everyBufLen []int                    // 每个打印格式的命令集合
	totalbuffer := bytes.NewBufferString("") // 打印命令集合
	dataCamp := bytes.NewBufferString("")
	TotalVarDataIndex := 0 // 每个打印格式信息的索引

	fillchar := 0xff
	lastVarPos := 0
	lastVarNum := 0
	lastAddr := 0

	buff, _ := Utf8ToGb2312(utf8Buff)
	var formatbuf *bytes.Buffer
	if printerModel == "EPM205" {
		dataCamp.Write(ESC_CHANGE_EPL_205)
	}

	// 解析SBPL指令

	if printerModel == "ZEBRA" {
		formatbuf = ParseEplZebraLines(buff, dataCamp, lastVarPos)
	} else if printerModel == "LP50" {
		formatbuf = ParseEplLp50Lines(buff, dataCamp, lastVarPos)
	} else if printerModel == "GODEX" {
		formatbuf = ParseEzplLines(buff, dataCamp, lastVarPos)
	} else if printerModel == "EPM205" {
		formatbuf = ParseEplLines(buff, dataCamp, lastVarPos)
	} else if printerModel == "TSC" {
		formatbuf = ParseTscLines(buff, dataCamp, lastVarPos)
	} else if printerModel == "SATO" {
		formatbuf = ParseSbplLines(buff, dataCamp, lastVarPos)
	} else {
		formatbuf = ParseEplLines(buff, dataCamp, lastVarPos)
	}

	everyBufLen = append(everyBufLen, formatbuf.Len())

	fmt.Println(string(dataCamp.Bytes()))

	totalbuffer.WriteString(formatbuf.String())
	div := ((everyBufLen[TotalVarDataIndex] / 4) + 1) * 4
	for i := formatbuf.Len(); i < div; i++ {
		totalbuffer.WriteByte(byte(fillchar))
	}

	if TotalVarDataIndex > 0 {
		lastVarNum = len(VarList) - lastVarNum
	} else {
		lastAddr = headBufLen
		lastVarNum = len(VarList)
	}

	FinalFormatInfo.formatNum = 1 ///打印格式总数，根据打印格式文件数量决定
	var prtName [23]byte
	prtName[22] = 0x00

	if len(printerModel) > 22 {
		printerModel = printerModel[:22]
	}
	copy(prtName[:len(printerModel)], []byte(printerModel))

	FinalFormatInfo.printerName = prtName
	FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].addr = uint32(lastAddr)
	FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].formatLen = uint32(everyBufLen[TotalVarDataIndex])
	FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].varNum = uint32(lastVarNum)
	dataCamp.Reset()

	TotalVarDataIndex = TotalVarDataIndex + 1
	lastVarNum = len(VarList)
	lastAddr = div + lastAddr

	// 4.将变量信息写入结构体
	for i := 0; i < len(VarList); i++ {
		FinalFormatInfo.varInfo[i] = VarList[i]
	}
	// 5.转换为bin文件
	buffer := binaryData(FinalFormatInfo)
	binary.Write(buffer, binary.LittleEndian, totalbuffer.Bytes())

	if buffer.Len() < (fmtLen - 8) {
		for i := buffer.Len(); i < (fmtLen - 8); i++ {
			binary.Write(buffer, binary.LittleEndian, byte(fillchar))
		}
	}

	// 6.写结尾5a a5 a5 5a 00 00 00 00

	dataCamp1 := bytes.NewBufferString("")

	for i := 0; i < len(FMT_FILL_TAIL); i++ {
		dataCamp1.WriteByte(FMT_FILL_TAIL[i])
	}

	binary.Write(buffer, binary.LittleEndian, dataCamp1.Bytes())

	println(buffer)

	return buffer
}

func ParserRptFmtToBuf(utf8Buff string, printerModel string, fmtLen int) *bytes.Buffer {
	var clearList []RptVarStruct
	RptVarList = clearList // 用于清空数据

	var clearTable ScaleVarOrder
	VarTable = clearTable // 用于清空数据

	var FinalRptFmtInfo printInfoRpt
	var everyBufLen []int                    // 每个打印格式的命令集合
	totalbuffer := bytes.NewBufferString("") // 打印命令集合
	dataCamp := bytes.NewBufferString("")
	TotalVarDataIndex := 0 // 每个打印格式信息的索引

	fillchar := 0xff
	lastVarPos := 0
	lastVarNum := 0
	lastAddr := 0

	// buff, _ := Utf8ToGb2312(utf8Buff)
	buff := utf8Buff //用UTF8 做
	var formatbuf *bytes.Buffer
	if printerModel == "EPM205" {
		dataCamp.Write(ESC_CHANGE_ESC_205)
	}
	if printerModel == "LP50" { //此处对接的是OS2130打印机
		formatbuf = ParseLP50Lines(buff, dataCamp, lastVarPos)
	} else if printerModel == "ZEBRA" {
		formatbuf = ParseRptZebraLines(buff, dataCamp, lastVarPos)
	} else {
		formatbuf = ParseEscLines(buff, dataCamp, lastVarPos)
	}
	everyBufLen = append(everyBufLen, formatbuf.Len())
	fmt.Println(string(dataCamp.Bytes()))
	totalbuffer.WriteString(formatbuf.String())
	div := ((everyBufLen[TotalVarDataIndex] / 4) + 1) * 4
	for i := formatbuf.Len(); i < div; i++ {
		totalbuffer.WriteByte(byte(fillchar))
	}

	if TotalVarDataIndex > 0 {
		lastVarNum = len(RptVarList) - lastVarNum
	} else {
		lastAddr = headBufLen
		lastVarNum = len(RptVarList)
	}

	FinalRptFmtInfo.formatNum = 1 ///打印格式总数，根据打印格式文件数量决定
	var prtName [23]byte
	prtName[22] = 0x01 // 打印格式类型 0x01 票据格式 0x00 标签格式
	tmpNameStr := ""
	if printerModel == "LP50" {
		printerModel = "LP50*31"
	} else if printerModel == "ZEBRA" {
		printerModel = "ZEBRA*44"
	}
	if len(printerModel) > 22 {
		tmpNameStr = printerModel[:22]
	} else {
		tmpNameStr = printerModel
	}
	copy(prtName[:len(tmpNameStr)], []byte(tmpNameStr))
	FinalRptFmtInfo.printerName = prtName
	FinalRptFmtInfo.everyFormatInfo[TotalVarDataIndex].addr = uint32(lastAddr)
	FinalRptFmtInfo.everyFormatInfo[TotalVarDataIndex].formatLen = uint32(everyBufLen[TotalVarDataIndex])
	FinalRptFmtInfo.everyFormatInfo[TotalVarDataIndex].varNum = uint32(lastVarNum)
	dataCamp.Reset()

	TotalVarDataIndex = TotalVarDataIndex + 1
	lastVarNum = len(RptVarList)
	lastAddr = div + lastAddr

	// 4.将变量信息写入结构体
	for i := 0; i < len(RptVarList); i++ {
		FinalRptFmtInfo.varInfo[i] = RptVarList[i]
	}
	// 5.转换为bin文件
	buffer := toBinaryDataRpt(FinalRptFmtInfo)
	binary.Write(buffer, binary.LittleEndian, totalbuffer.Bytes())

	if buffer.Len() < (fmtLen - 8) {
		for i := buffer.Len(); i < (fmtLen - 8); i++ {
			binary.Write(buffer, binary.LittleEndian, byte(fillchar))
		}
	}
	// 6.写结尾5a a5 a5 5a 00 00 00 00

	dataCamp1 := bytes.NewBufferString("")

	for i := 0; i < len(FMT_FILL_TAIL); i++ {
		dataCamp1.WriteByte(FMT_FILL_TAIL[i])
	}
	binary.Write(buffer, binary.LittleEndian, dataCamp1.Bytes())
	return buffer
}

// 转二进制
func toBinaryDataRpt(tempInfo printInfoRpt) *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, tempInfo.printerName)
	binary.Write(buf, binary.LittleEndian, tempInfo.formatNum)
	binary.Write(buf, binary.LittleEndian, tempInfo.everyFormatInfo)
	binary.Write(buf, binary.LittleEndian, tempInfo.varInfo)
	return buf
}

// 转二进制
func binaryData(tempInfo printInfo) *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, tempInfo.printerName)
	binary.Write(buf, binary.LittleEndian, tempInfo.formatNum)
	binary.Write(buf, binary.LittleEndian, tempInfo.everyFormatInfo)
	binary.Write(buf, binary.LittleEndian, tempInfo.varInfo)
	return buf
}

// 创建bin文件
func creatFile(fileName string, tempBuf *bytes.Buffer) bool {
	//先找到exe运行的路径
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	// 拼接文件路径
	filePath := filepath.Join(exeDir, fileName)

	fp, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer fp.Close()
	fp.Write(tempBuf.Bytes())
	return true
}

// Utf8ToGb2312 将UTF-8字符串转换为GB2312编码
func Utf8ToGb2312(buff string) (string, error) {
	utf8str := buff
	enc := simplifiedchinese.GB18030.NewEncoder()
	utf8Bytes := []byte(utf8str)
	gb2312Bytes, err := enc.Bytes(utf8Bytes)
	if err != nil {
		log.Printf("Utf8ToGb2312 error: %v", err)
		return "", err
	}
	gb2312str := string(gb2312Bytes)

	return gb2312str, err
}

func ParserDefFmtToFile(fmtDataList []string, printerModel string, fmtLen int) bool {
	var buffer *bytes.Buffer
	if printerModel == "EPM205" {
		buffer = ParserDefFmtToBuf(fmtDataList, printerModel, fmtLen)
	} else if printerModel == "LP50" {
		buffer = ParserLp50DefFmtToBuf(fmtDataList, printerModel, fmtLen)
	} else {
		return false
	}
	// 7.创建bin文件
	if buffer.Len() > 0 {
		if creatFile("formatBin.bin", buffer) {
			fmt.Println("creat binary file success")
			return true
		} else {
			fmt.Println("creat binary file fail")
			return false
		}

	}
	return false

	// 8.写数据到串口
}

func ParserDefFmtToBuf(fmtDataList []string, printerModel string, fmtLen int) *bytes.Buffer {
	var FinalFormatInfo printInfoDef

	dataCamp1 := bytes.NewBufferString("")
	for i := 0; i < len(FMT_FILL_TAIL); i++ {
		dataCamp1.WriteByte(FMT_FILL_TAIL[i])
	}

	// var fillchar byte
	fillchar := 0xff
	var lastVarPos int //变量位置

	FinalFormatInfo.formatNum = uint8(len(fmtDataList)) ///打印格式总数，根据打印格式文件数量决定

	totalbuffer := bytes.NewBufferString("") //打印命令集合
	// var everyAddr []int                                                 //每个打印格式偏移量
	var everyBufLen []int  //每个打印格式的命令集合
	TotalVarDataIndex := 0 //每个打印格式信息的索引
	// formatinfo.VarTable = formatinfo.ReadTableFromFile(currentPath + "\\varTable.json") //获取变量ID表
	dataCamp := bytes.NewBufferString("") //临时buf 存放命令数据
	lastVarNum := 0
	lastAddr := 0
	var clearList []VarStruct
	VarList = clearList // 用于清空数据

	//for循环解析文件
	for i := 0; i < len(fmtDataList); i++ {
		utf8Buff := fmtDataList[i]
		lastVarPos = 0

		buff, _ := Utf8ToGb2312(utf8Buff)
		if printerModel == "EPM205" {
			dataCamp.Write(ESC_CHANGE_EPL_205)
		}
		formatbuf := ParseEplLines(buff, dataCamp, lastVarPos)
		everyBufLen = append(everyBufLen, formatbuf.Len())

		fmt.Println(string(dataCamp.Bytes()))

		totalbuffer.WriteString(formatbuf.String())
		div := ((everyBufLen[TotalVarDataIndex] / 4) + 1) * 4

		for i := formatbuf.Len(); i < div; i++ {
			totalbuffer.WriteByte(byte(fillchar))
		}

		if TotalVarDataIndex > 0 {
			lastVarNum = len(VarList) - lastVarNum
		} else {
			lastAddr = headBufLenDef
			lastVarNum = len(VarList)
		}
		// fmt.Printf("%x", templen)
		FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].addr = uint32(lastAddr)
		FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].formatLen = uint32(everyBufLen[TotalVarDataIndex])
		FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].varNum = uint32(lastVarNum)
		dataCamp.Reset()
		// fmt.Println(dataCamp.Len())
		TotalVarDataIndex = TotalVarDataIndex + 1
		lastVarNum = len(VarList)
		lastAddr = div + lastAddr

	}
	if len(VarList) > 150 {
		return bytes.NewBufferString("")

	}

	//复制变量
	copy(FinalFormatInfo.varInfo[:], VarList)

	buffer := binaryDataDef(FinalFormatInfo)
	binary.Write(buffer, binary.LittleEndian, totalbuffer.Bytes())
	// fmt.Println(buffer.Len())
	if buffer.Len() < (fmtLen - 8) {
		for i := buffer.Len(); i < (fmtLen - 8); i++ {
			binary.Write(buffer, binary.LittleEndian, byte(fillchar))
		}
	} else {
		return bytes.NewBufferString("")
	}
	binary.Write(buffer, binary.LittleEndian, dataCamp1.Bytes())

	return buffer

}

func binaryDataDef(tempInfo printInfoDef) *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, tempInfo.printerName)
	binary.Write(buf, binary.LittleEndian, tempInfo.formatNum)
	binary.Write(buf, binary.LittleEndian, tempInfo.everyFormatInfo)
	binary.Write(buf, binary.LittleEndian, tempInfo.varInfo)
	return buf
}

func ParserLp50DefFmtToBuf(fmtDataList []string, printerModel string, fmtLen int) *bytes.Buffer {
	var FinalFormatInfo printInfoDef

	dataCamp1 := bytes.NewBufferString("")
	for i := 0; i < len(FMT_FILL_TAIL); i++ {
		dataCamp1.WriteByte(FMT_FILL_TAIL[i])
	}

	// var fillchar byte
	fillchar := 0xff
	var lastVarPos int //变量位置

	FinalFormatInfo.formatNum = uint8(len(fmtDataList)) ///打印格式总数，根据打印格式文件数量决定

	totalbuffer := bytes.NewBufferString("") //打印命令集合
	// var everyAddr []int                                                 //每个打印格式偏移量
	var everyBufLen []int  //每个打印格式的命令集合
	TotalVarDataIndex := 0 //每个打印格式信息的索引
	// formatinfo.VarTable = formatinfo.ReadTableFromFile(currentPath + "\\varTable.json") //获取变量ID表
	dataCamp := bytes.NewBufferString("") //临时buf 存放命令数据
	lastVarNum := 0
	lastAddr := 0
	var clearList []VarStruct
	VarList = clearList // 用于清空数据

	//for循环解析文件
	for i := 0; i < len(fmtDataList); i++ {
		utf8Buff := fmtDataList[i]
		lastVarPos = 0

		buff, _ := Utf8ToGb2312(utf8Buff)
		buff = ModifyDataSimple(buff)
		formatbuf := ParseEplLp50Lines(buff, dataCamp, lastVarPos)
		everyBufLen = append(everyBufLen, formatbuf.Len())

		fmt.Println(string(dataCamp.Bytes()))

		totalbuffer.WriteString(formatbuf.String())
		div := ((everyBufLen[TotalVarDataIndex] / 4) + 1) * 4

		for i := formatbuf.Len(); i < div; i++ {
			totalbuffer.WriteByte(byte(fillchar))
		}

		if TotalVarDataIndex > 0 {
			lastVarNum = len(VarList) - lastVarNum
		} else {
			lastAddr = headBufLenDef
			lastVarNum = len(VarList)
		}
		// fmt.Printf("%x", templen)
		FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].addr = uint32(lastAddr)
		FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].formatLen = uint32(everyBufLen[TotalVarDataIndex])
		FinalFormatInfo.everyFormatInfo[TotalVarDataIndex].varNum = uint32(lastVarNum)
		dataCamp.Reset()
		// fmt.Println(dataCamp.Len())
		TotalVarDataIndex = TotalVarDataIndex + 1
		lastVarNum = len(VarList)
		lastAddr = div + lastAddr

	}
	if len(VarList) > 150 {
		return bytes.NewBufferString("")

	}

	//复制变量
	copy(FinalFormatInfo.varInfo[:], VarList)

	buffer := binaryDataDef(FinalFormatInfo)
	binary.Write(buffer, binary.LittleEndian, totalbuffer.Bytes())
	// fmt.Println(buffer.Len())
	if buffer.Len() < (fmtLen - 8) {
		for i := buffer.Len(); i < (fmtLen - 8); i++ {
			binary.Write(buffer, binary.LittleEndian, byte(fillchar))
		}
	} else {
		return bytes.NewBufferString("")
	}
	binary.Write(buffer, binary.LittleEndian, dataCamp1.Bytes())

	return buffer

}

// EPM205 打印机 解析P命令 存为整数
func ModifyDataSimple(s string) string {
	if !strings.Contains(s, "EPM205") {
		return s
	}
	lines := strings.Split(s, "\r\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "P,") {
			// 解析line
			parts := strings.Split(line, ",")
			if len(parts) == 3 {
				// parts[0]是"P"，parts[1]和parts[2]是数字字符串（可能含小数）
				num1, err1 := strconv.ParseFloat(parts[1], 64)
				num2, err2 := strconv.ParseFloat(parts[2], 64)
				if err1 == nil && err2 == nil {
					int1 := int(num1) // 截断小数
					int2 := int(num2)
					lines[i] = fmt.Sprintf("P,%d,%d", int1, int2)
				}
				// 如果解析失败，保留原行
			}
			break

		}
	}
	return strings.Join(lines, "\r\n")
}
