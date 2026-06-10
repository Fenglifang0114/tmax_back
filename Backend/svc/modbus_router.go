package svc

import (
	"encoding/binary"
	"math"
	"strconv"
	"tmaxsrv/log"
)

// RouteModbusRequest 处理解析好的 Modbus 请求帧并返回响应帧
func RouteModbusRequest(packet []byte, targetModbusId int, srvMgr *SrvMgr) []byte {
	if len(packet) < 8 {
		return nil
	}

	slaveID := packet[0]
	funcCode := packet[1]
	startAddr := binary.BigEndian.Uint16(packet[2:4])
	
	// 查找目标秤
	var targetScale *Scale
	scaleMgr := srvMgr.scaleMgr
	for _, scale := range scaleMgr.scales {
		if (scale.Conn != nil && scale.Conn.ModbusId == targetModbusId) || scale.Id == int64(targetModbusId) {
			targetScale = scale
			break
		}
	}

	if targetScale == nil {
		log.Log.Warnf("Modbus Router: Target Scale not found for ModbusID %d", targetModbusId)
		return buildExceptionResponse(slaveID, funcCode, 0x0B) // 0x0B: Gateway Target Device Failed to Respond
	}

	switch funcCode {
	case 0x03:
		count := binary.BigEndian.Uint16(packet[4:6])
		return handleReadHoldingRegisters(slaveID, startAddr, count, targetScale)
	case 0x06:
		value := binary.BigEndian.Uint16(packet[4:6])
		return handleWriteSingleRegister(slaveID, startAddr, value, targetScale)
	default:
		return buildExceptionResponse(slaveID, funcCode, 0x01)
	}
}

func handleReadHoldingRegisters(slaveID byte, startAddr uint16, count uint16, scale *Scale) []byte {
	// 特殊处理：01 03 00 00 00 01 84 0A 作为建立连接的心跳包，需要原样回复特定数据
	if startAddr == 0 && count == 1 {
		response := make([]byte, 7)
		response[0] = slaveID
		response[1] = 0x03
		response[2] = 0x02
		response[3] = 0x19
		response[4] = 0x98
		return appendModbusCRC(response)
	}

	currentWeight := scale.LastWeight

	byteCount := count * 2
	data := make([]byte, byteCount)

	for i := uint16(0); i < count; i+=2 {
		addr := startAddr + i
		var val float32 = 0.0
		
		switch addr {
		case 0, 40001, 2, 40003: // 0/2 为标准地址，40001/40003 为兼容地址 (毛重)
			if weight, err := GetGrossWeight(scale); err == nil {
				val = weight
			} else {
				val = currentWeight
			}
		case 4, 40005: // 皮重 (扣重)
			if weight, err := GetTareWeight(scale); err == nil {
				val = weight
			}
		case 6, 40007: // 净重
			if weight, err := GetNetWeight(scale); err == nil {
				val = weight
			}
		case 8, 40009: // 未圆整毛重 (暂用毛重代替)
			if weight, err := GetGrossWeight(scale); err == nil {
				val = weight
			}
		case 10, 40011: // 预扣重
			if weight, err := GetPreTareWeight(scale); err == nil {
				val = weight
			}
		case 12, 40013: // 未圆整净重 (暂用净重代替)
			if weight, err := GetNetWeight(scale); err == nil {
				val = weight
			}
		case 14, 40015: // 单位 (0: kg, 1: g, 4: lb)
			if unit, err := GetWeightUnitCmd(scale); err == nil {
				if unit == 2 {
					val = float32(4) // 秤端 2 代表 lb，Modbus 中 4 代表 lb
				} else {
					val = float32(unit) // 0 代表 kg，1 代表 g，保持不变
				}
			}
		}

		if addr == 14 || addr == 40015 {
			// 单位寄存器必须返回整数，不能当做浮点数处理！
			if i+1 < count {
				binary.BigEndian.PutUint32(data[i*2:(i*2)+4], uint32(val))
			} else {
				binary.BigEndian.PutUint16(data[i*2:(i*2)+2], uint16(val))
			}
		} else {
			if i+1 < count {
				bits := math.Float32bits(val)
				binary.BigEndian.PutUint32(data[i*2:(i*2)+4], bits)
			} else {
				// 只请求了一个寄存器，返回高16位以防客户端报错
				bits := math.Float32bits(val)
				binary.BigEndian.PutUint16(data[i*2:(i*2)+2], uint16(bits>>16))
			}
		}
	}

	response := make([]byte, 3+byteCount+2)
	response[0] = slaveID
	response[1] = 0x03
	response[2] = byte(byteCount)
	copy(response[3:], data)

	return appendModbusCRC(response)
}

func handleWriteSingleRegister(slaveID byte, startAddr uint16, value uint16, scale *Scale) []byte {
	success := false
	if value == 256 {
		switch startAddr {
		case 21, 40022: // 21 (0x15) 是标准地址，40022 (0x9C56) 是文档原始十六进制地址
			log.Log.Infof("Modbus Router: Executing Tare on Scale %d", scale.Id)
			scale.PerfTare()
			success = true
		case 23, 40024: // 0x17 or 0x9C58
			log.Log.Infof("Modbus Router: Executing Zero on Scale %d", scale.Id)
			scale.PerfZero()
			success = true
		case 25, 40026: // 0x19 or 0x9C5A
			log.Log.Infof("Modbus Router: Executing Clear Tare on Scale %d", scale.Id)
			// scale.PerfClearTare() // 暂无直接方法，通常调用取消扣重
			success = true
		}
	}

	// 针对 S15 的标定功能
	switch startAddr {
	case 30, 40030: // 0x9C5E
		success = true // 无论是否 S15，都返回正确，以免主机报错
		if scale.Model == "S15" {
			if value == 3 {
				log.Log.Infof("Modbus Router: S15 Clear Flag")
			} else if value == 1 {
				log.Log.Infof("Modbus Router: S15 Set Zero Point")
				ReqCalWeight(scale, SRequest{ReqData: "0"})
			} else if value == 2 {
				calWeightStr := strconv.Itoa(int(scale.ModbusCalWeight))
				log.Log.Infof("Modbus Router: S15 Start Calibration with weight %s", calWeightStr)
				ReqCalWeight(scale, SRequest{ReqData: calWeightStr})
			}
		}
	case 32, 40032: // 0x9C60 (MSB)
		success = true
		if scale.Model == "S15" {
			scale.ModbusCalWeightMSB = value
			log.Log.Infof("Modbus Router: S15 Set Cal Weight MSB to 0x%X", value)
		}
	case 33, 40033: // 0x9C61 (LSB)
		success = true
		if scale.Model == "S15" {
			bits := (uint32(scale.ModbusCalWeightMSB) << 16) | uint32(value)
			scale.ModbusCalWeight = math.Float32frombits(bits)
			log.Log.Infof("Modbus Router: S15 Set Cal Weight to %f", scale.ModbusCalWeight)
		}
	}

	if !success {
		log.Log.Warnf("Modbus Router: Unhandled Write Addr: %d (0x%X), Value: %d", startAddr, startAddr, value)
	}

	response := make([]byte, 6+2)
	response[0] = slaveID
	response[1] = 0x06
	binary.BigEndian.PutUint16(response[2:4], startAddr)
	binary.BigEndian.PutUint16(response[4:6], value)

	return appendModbusCRC(response)
}

func buildExceptionResponse(slaveID byte, funcCode byte, exceptionCode byte) []byte {
	response := make([]byte, 5)
	response[0] = slaveID
	response[1] = funcCode | 0x80
	response[2] = exceptionCode
	return appendModbusCRC(response)
}

func appendModbusCRC(data []byte) []byte {
	crcData := data[:len(data)-2]
	crc := calculateModbusCRC16(crcData)
	data[len(data)-2] = byte(crc & 0xFF)
	data[len(data)-1] = byte((crc >> 8) & 0xFF)
	return data
}
