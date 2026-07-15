package svc

import (
	"fmt"
	"regexp"
	"strings"

	m "tmaxsrv/comm"
	l "tmaxsrv/log"
)

// ProtocolParser 定义了协议解析的函数签名
type ProtocolParser func(scaleId int64, data []byte) (*ScaleRespMsg, error)

// protocolParsers 保存所有的协议解析器
var protocolParsers = map[string]ProtocolParser{
	"SCP-01": parseSCP01,
	"SCP-02": parseSCP02,
	"SCP-03": parseSCP03,
	"SCP-04": parseSCP04,
	"SCP-05": parseSCP05,
	"SCP-06": parseSCP06,
	"SCP-07": parseSCP07,
	"SCP-08": parseSCP08,
	"SCP-09": parseSCP09,
	"SCP-10": parseSCP10,
	"SCP-11": parseSCP11,
	"SCP-12": parseSCP12,
	"SCP-13": parseSCP13,
	"SCP-14": parseSCP14,
	"SCP-15": parseSCP15,
	"SCP-16": parseSCP16,
	"SCP-17": parseSCP17,
	"SCP-18": parseSCP18,
	"SCP-19": parseSCP19,
	"SCP-20": parseSCP20,
	"SCP-21": parseSCP21,
}

var overloadRegex = regexp.MustCompile(`(?:^|[^a-zA-Z])(OL|UL)(?:[^a-zA-Z]|$)`)

// DispatchProtocolParser 根据协议名调度相应的解析函数
func DispatchProtocolParser(protocolName string, scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	l.Log.Debugf("DispatchProtocolParser invoked - ProtocolName: '%s', ScaleId: %d, Data: %q", protocolName, scaleId, dataStr)

	// 使用正则严格匹配，避免误杀类似 "VOLTAGE", "HOLD", "NULL" 等词汇以及原生的二进制数据中的 0x4F 0x4C
	if matches := overloadRegex.FindStringSubmatch(dataStr); matches != nil {
		val := "-OL-"
		if matches[1] == "UL" {
			val = "-UL-"
		}
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  val,
			WeightUnit: "",
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 如果在注册表里能找到该协议，就用特定的协议解析
	if parser, exists := protocolParsers[protocolName]; exists {
		return parser(scaleId, data)
	}

	// 默认兜底：如果没找到或者没有配置 ProtocolName，走之前的通用解析逻辑
	return retreiveRespMsgC51(scaleId, data)
}

// defaultSCPParser 包装一下通用的解析以适应 ProtocolParser 签名
func defaultSCPParser(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	var err error
	respMsg := &ScaleRespMsg{MsgType: GlastWantRespMsgType, MsgBody: "", ScaleId: scaleId}

	if len(data) == 1 {
		switch data[0] {
		case 0x06:
			respMsg.MsgBody = "ok"
			err = nil
		case 0x15:
			respMsg.MsgBody = "fail"
			err = nil
		default:
			return respMsg, fmt.Errorf("unknown response")
		}
	} else {
		// 先尝试用特定的 ST/US GS/NT 格式去解析
		msg, parseErr := parseStandardSCP(data)
		if parseErr == nil {
			respMsg.MsgType = m.WEIGHT_DATA
			respMsg.MsgBody = msg
		} else {
			// 如果特定解析失败，回退到原来的逻辑
			var oldMsg WeightMsg
			if oldMsg, err = retreiveWeightNewC51(data); err == nil {
				respMsg.MsgType = m.WEIGHT_DATA
				respMsg.MsgBody = oldMsg
			}
		}
	}
	return respMsg, err
}

// parseStandardSCP 专门用于解析带有 ST(稳定)/US(不稳定), GS(毛重)/NT(净重) 的协议文本
func parseStandardSCP(data []byte) (WeightMsg, error) {
	dataStr := string(data)
	// 去掉可能包含的括号
	dataStr = strings.ReplaceAll(dataStr, "(", "")
	dataStr = strings.ReplaceAll(dataStr, ")", "")

	// 匹配模式：ST或US, 接着 GS或NT, 可能有正负号，然后是数字，最后是单位
	re := regexp.MustCompile(`(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)`)
	matches := re.FindStringSubmatch(dataStr)

	if matches != nil && len(matches) >= 6 {
		status := matches[1] // "ST" 或 "US"
		mode := matches[2]   // "GS" 或 "NT"
		sign := matches[3]   // "+" 或 "-"
		value := matches[4]  // "123.45"
		unit := matches[5]   // "kg", "g" 等

		if sign == "-" {
			value = "-" + value
		}

		isZero := (value == "0" || value == "0.0" || value == "0.00" || value == "0.000" || value == "0.0000" || value == "-0.0" || value == "-0.00")

		return WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			WeightVal:  value,
			WeightUnit: unit,
			IsZero:     isZero,
		}, nil
	}

	// 匹配异常状态：------ 或者 --OL-- / --UL--
	reErr := regexp.MustCompile(`(--(?:OL|UL)--|-{6,})`)
	if matchesErr := reErr.FindStringSubmatch(dataStr); matchesErr != nil {
		return WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  matchesErr[1],
			WeightUnit: "",
		}, nil
	}

	return WeightMsg{}, fmt.Errorf("SCP parse error on %v", dataStr)
}

// 下面是 SCP 协议的具体解析实现，目前先默认指向 defaultSCPParser（内部使用 retreiveWeightNewC51）
// 之后可以根据各图片协议的特定格式，在这里逐个细化修改

// parseSCP01 解析 SCP-01 协议
func parseSCP01(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-01 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP02 解析 SCP-02 协议
func parseSCP02(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-02 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP03 解析 SCP-03 协议
func parseSCP03(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-03 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP04 解析 SCP-04 协议
func parseSCP04(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-04 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP05 解析 SCP-05 协议
func parseSCP05(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-05 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP06 解析 SCP-06 协议
func parseSCP06(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-06 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP07 解析 SCP-07 协议
func parseSCP07(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-07 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP08 解析 SCP-08 协议
func parseSCP08(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-08 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP09 解析 SCP-09 协议
func parseSCP09(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-09 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP10 解析 SCP-10 协议
func parseSCP10(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-10 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP11 解析 SCP-11 协议
func parseSCP11(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-11 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP12 解析 SCP-12 协议
func parseSCP12(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-12 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP13 解析 SCP-13 协议
func parseSCP13(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-13 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP14 解析 SCP-14 协议
func parseSCP14(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-14 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP15 解析 SCP-15 协议
func parseSCP15(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-15 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP16 解析 SCP-16 协议
func parseSCP16(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-16 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP17 解析 SCP-17 协议
func parseSCP17(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-17 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP18 解析 SCP-18 协议
func parseSCP18(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-18 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP19 解析 SCP-19 协议
func parseSCP19(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-19 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP20 解析 SCP-20 协议
func parseSCP20(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-20 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

// parseSCP21 解析 SCP-21 协议
func parseSCP21(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	
	// TODO: 根据 SCP-21 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)")
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {
		msg := WeightMsg{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}

