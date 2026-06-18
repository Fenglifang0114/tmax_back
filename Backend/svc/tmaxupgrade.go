package svc

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	m "tmaxsrv/comm"
	"tmaxsrv/picker"

	"go.bug.st/serial"
)

type PragramConfig struct {
	Baud        int    `yaml:"Baud"`
	PrintFormat bool   `yaml:"PrintFormat"`
	DevPath     string `yaml:"DevPath"`
	IsFindCOM   bool   `yaml:"IsFindCOM"`
}

func (c *Scale) ProcessUpdate(srecName string) (*ScaleRespMsg, error) {
	var mode serial.Mode
	mode.BaudRate = 57600
	port, err := serial.Open(c.Pcnf.DevPath, &mode)
	if err != nil {
		respMsg := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail,open serial port error", ScaleId: c.Id}
		return &respMsg, err
	}

	// 读取 SREC 文件
	srecData, minAddress, err := SRECToByteArray(srecName)
	if err != nil {
		port.Close()
		return &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail,srec data error", ScaleId: c.Id}, err
	}

	var expectedBootResp []byte
	var wrongBootResp []byte
	if minAddress&0xFFFF == 0x0800 {
		expectedBootResp = []byte{0x03, 0xff, 0x08}
		wrongBootResp = []byte{0x00, 0xff, 0x08}
	} else if minAddress&0xFFFF == 0x1000 {
		expectedBootResp = []byte{0x00, 0xff, 0x08}
		wrongBootResp = []byte{0x03, 0xff, 0x08}
	} else {
		port.Close()
		return &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail,app address not match boot 2K/4K", ScaleId: c.Id}, fmt.Errorf("app address not match boot 2K/4K")
	}

	// 执行升级流程
	upgrader := NewUpgrader(c, port)

	_, err = upgrader.PerformUpgrader(srecData, expectedBootResp, wrongBootResp)
	if err != nil {
		log.Printf("外部调用捕获到错误: %v", err) // 添加日志
		respMsg := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail,upgrade failed", ScaleId: c.Id}
		port.Close()
		return &respMsg, err
	}
	respMsg := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "ok", ScaleId: c.Id}
	time.Sleep(200 * time.Millisecond)
	port.Close()
	return respMsg, nil
}

func (c *Scale) TmaxUpdateFirmware(srecName string) (*ScaleRespMsg, error) {
	var resp *ScaleRespMsg
	var err error

	if c.Conn.TMedia != MEDIA_COM {
		return &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail,only support serial port update", ScaleId: c.Id}, nil
	}

	var pickerFn picker.PickerFunc
	if c.MySerial != nil {
		pickerFn = c.MySerial.pickerFn
		c.MySerial.Close()
	} else {
		// 如果因为某些原因 MySerial 为空，我们提供一个默认的解析函数以防止再次崩溃
		pickerFn = picker.GetPickerFn(c.ScaleCat)
	}

	//关闭串口，将串口让出去
	resp, _ = c.ProcessUpdate(srecName)
	// 开启串口，将串口重新初始化
	if c.MySerial, err = NewSerial(c.Pcnf, pickerFn, true); err != nil {
		log.Printf("重新初始化串口失败: %v", err)
	}

	result, _ := json.Marshal(resp)
	if c.client != nil {
		c.client.sendCh <- result
	}

	return resp, nil
}

// readSrecFile 读取 SREC 文件内容
// func readSrecFile(filePath string) ([]string, error) {
// 	file, err := os.Open(filePath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	var lines []string
// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		lines = append(lines, scanner.Text())
// 	}
// 	return lines, scanner.Err()
// }

// SRECToByteArray 解析SREC文件并返回字节数组、最小地址和错误
func SRECToByteArray(filename string) ([]byte, uint32, error) {
	// 读取SREC文件
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	var records []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 && line[0] == 'S' {
			records = append(records, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %v", err)
	}

	// 确定地址范围
	var minAddress, maxAddress uint32
	var firstAddressSet bool

	for _, record := range records {
		srec, err := parseSRecord(record)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse SREC record: %v", err)
		}

		// 只处理数据记录类型 (S1, S2, S3)
		if srec.Type == "S1" || srec.Type == "S2" || srec.Type == "S3" {
			addressEnd := srec.Address + uint32(len(srec.Data))

			if !firstAddressSet {
				minAddress = srec.Address
				maxAddress = addressEnd
				firstAddressSet = true
			} else {
				if srec.Address < minAddress {
					minAddress = srec.Address
				}
				if addressEnd > maxAddress {
					maxAddress = addressEnd
				}
			}
		}
	}

	if !firstAddressSet {
		return nil, 0, fmt.Errorf("fail,no valid data records found")
	}

	// 创建二进制数据缓冲区并初始化为0xFF
	dataSize := maxAddress - minAddress
	if dataSize > 1024*1024 {
		return nil, 0, fmt.Errorf("fail,binary data size exceeds 1024KB: %d bytes", dataSize)
	}

	binaryData := make([]byte, dataSize)
	for i := range binaryData {
		binaryData[i] = 0xFF
	}

	// 填充二进制数据
	for _, record := range records {
		srec, err := parseSRecord(record)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse SREC record: %v", err)
		}

		if srec.Type == "S1" || srec.Type == "S2" || srec.Type == "S3" {
			offset := srec.Address - minAddress
			if offset+uint32(len(srec.Data)) > uint32(len(binaryData)) {
				return nil, 0, fmt.Errorf("fail,record data exceeds calculated address range: %s", record)
			}
			copy(binaryData[offset:], srec.Data)
		}
	}

	return binaryData, minAddress, nil
}

// 解析单个SREC记录
func parseSRecord(line string) (SRecord, error) {
	// 此处省略与之前相同的parseSRecord实现...
	// 与之前提供的代码相同
	if len(line) < 4 {
		return SRecord{}, fmt.Errorf("fail,record too short: %s", line)
	}

	srec := SRecord{
		Type: line[:2],
	}

	// 解析字节计数
	countStr := line[2:4]
	countBytes, err := hex.DecodeString(countStr)
	if err != nil {
		return SRecord{}, fmt.Errorf("fail,parse byte count failed: %v", err)
	}
	count := int(countBytes[0])

	// 验证行长度
	if len(line) != 4+count*2 {
		return SRecord{}, fmt.Errorf("fail,record length mismatch: %s", line)
	}

	// 解析地址和数据
	addressBytes := 0
	switch srec.Type {
	case "S0", "S1", "S5":
		addressBytes = 2
	case "S2", "S6":
		addressBytes = 3
	case "S3", "S7", "S8", "S9":
		addressBytes = 4
	default:
		return SRecord{}, fmt.Errorf("fail,unknown SREC type: %s", srec.Type)
	}

	addressStr := line[4 : 4+addressBytes*2]
	addressData, err := hex.DecodeString(addressStr)
	if err != nil {
		return SRecord{}, fmt.Errorf("fail,parse address failed: %v", err)
	}

	// 转换地址为uint32
	srec.Address = 0
	for _, b := range addressData {
		srec.Address = (srec.Address << 8) | uint32(b)
	}

	// 解析数据
	dataStart := 4 + addressBytes*2
	dataEnd := dataStart + (count-addressBytes-1)*2
	if dataEnd <= dataStart {
		srec.Data = []byte{}
	} else {
		dataStr := line[dataStart:dataEnd]
		srec.Data, err = hex.DecodeString(dataStr)
		if err != nil {
			return SRecord{}, fmt.Errorf("fail,parse data failed: %v", err)
		}
	}

	// 解析校验和
	checksumStr := line[dataEnd : dataEnd+2]
	checksumData, err := hex.DecodeString(checksumStr)
	if err != nil {
		return SRecord{}, fmt.Errorf("fail,parse checksum failed: %v", err)
	}
	srec.Checksum = checksumData[0]

	// 验证校验和
	if !verifyChecksum(line) {
		return SRecord{}, fmt.Errorf("fail,checksum verification failed: %s", line)
	}

	return srec, nil
}

// 验证SREC记录的校验和
func verifyChecksum(line string) bool {
	if len(line) < 4 {
		return false
	}

	// 解析字节计数（前两个十六进制字符，不包括'S'和类型字符）
	countStr := line[2:4]
	countBytes, err := hex.DecodeString(countStr)
	if err != nil {
		return false
	}
	count := int(countBytes[0])

	// 提取需要计算校验和的数据部分（从字节计数开始到校验和前一个字节）
	dataStr := line[2 : 4+count*2-2] // 注意这里的范围
	dataBytes, err := hex.DecodeString(dataStr)
	if err != nil {
		return false
	}

	// 提取记录中提供的校验和
	checksumStr := line[4+count*2-2 : 4+count*2]
	checksumBytes, err := hex.DecodeString(checksumStr)
	if err != nil {
		return false
	}
	expectedChecksum := checksumBytes[0]

	// 计算校验和：对除校验和外的所有字节求和，然后取反
	var sum byte
	for _, b := range dataBytes {
		sum += b
	}
	calculatedChecksum := ^sum // 取反

	// 调试输出（可以在生产环境中移除）
	if false { // 设为true可启用调试输出
		fmt.Printf("Line: %s\n", line)
		fmt.Printf("  Count: 0x%02X\n", count)
		fmt.Printf("  Data bytes: %d\n", len(dataBytes))
		fmt.Printf("  Data: % X\n", dataBytes)
		fmt.Printf("  Sum: 0x%02X\n", sum)
		fmt.Printf("  Calculated checksum: 0x%02X\n", calculatedChecksum)
		fmt.Printf("  Expected checksum: 0x%02X\n", expectedChecksum)
		fmt.Printf("  Checksum match: %v\n", calculatedChecksum == expectedChecksum)
	}
	return calculatedChecksum == expectedChecksum
}

// SRecord 表示一个SREC记录
type SRecord struct {
	Type     string
	Address  uint32
	Data     []byte
	Checksum byte
}
