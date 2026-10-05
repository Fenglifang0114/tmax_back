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
	"SCP-23": parseSCP23,
	"SCP-24": parseSCP24,
	"SCP-25": parseSCP25,
	"SCP-26": parseSCP26,
	"SCP-27": parseSCP27,
	"SCP-28": parseSCP28,
	"SCP-29": parseSCP29,
	"SCP-30": parseSCP30,
	"SCP-31": parseSCP31,
	"SCP-32": parseSCP32,
	"SCP-33": parseSCP33,
}

var overloadRegex = regexp.MustCompile(`(?:^|[^a-zA-Z])(OL|UL)(?:[^a-zA-Z]|$)`)

var validWeightRegex = regexp.MustCompile(`^(?:-OL-|-UL-|-{6,}|[+-]?[0-9]+\.?[0-9]*)$`)

// isRespMsgValid 检查解析出的 ScaleRespMsg 是否包含有效的重量数据
func isRespMsgValid(resp *ScaleRespMsg) bool {
	if resp == nil || resp.MsgBody == nil {
		return false
	}
	msg, ok := resp.MsgBody.(WeightMsg)
	if !ok {
		return false
	}
	val := strings.TrimSpace(msg.WeightVal)
	if val == "" || val == "unknown" {
		return false
	}
	return validWeightRegex.MatchString(val)
}

// parseFlexibleNumeric 第三级兜底：弹性数值与单位提取 (自动从文本中提取数字与重量单位)
func parseFlexibleNumeric(data []byte) (WeightMsg, error) {
	dataStr := string(data)
	re := regexp.MustCompile(`([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]{1,4})`)
	matches := re.FindStringSubmatch(dataStr)

	if matches != nil && len(matches) >= 4 {
		sign := matches[1]
		valStr := matches[2]
		unitStr := matches[3]

		if sign == "-" {
			valStr = "-" + valStr
		}
		isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")

		return WeightMsg{
			IsStable:   true,
			IsNet:      false,
			IsZero:     isZero,
			WeightVal:  valStr,
			WeightUnit: unitStr,
		}, nil
	}

	return WeightMsg{}, fmt.Errorf("flexible parse failed on %q", dataStr)
}

// DispatchProtocolParser 根据协议名调度相应的解析函数 (带自动兜底)
func DispatchProtocolParser(protocolName string, scaleId int64, data []byte) (*ScaleRespMsg, error) {
	dataStr := string(data)
	l.Log.Debugf("DispatchProtocolParser invoked - ProtocolName: '%s', ScaleId: %d, Data: %q", protocolName, scaleId, dataStr)

	// 0. 特殊标志 (-OL- / -UL-) 拦截处理
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

	// 1. 第一级：优先使用用户选择的特定 SCP 协议解析
	if parser, exists := protocolParsers[protocolName]; exists {
		respMsg, err := parser(scaleId, data)
		if err == nil && isRespMsgValid(respMsg) {
			return respMsg, nil
		}
		l.Log.Warnf("Protocol '%s' failed for data %q, fallback to universal parsers", protocolName, dataStr)
	}

	// 2. 第二级兜底：通用 SCP 格式解析 (ST/US, GS/NT)
	if msg, err := parseStandardSCP(data); err == nil {
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 3. 第三级兜底：弹性数值与单位提取 (提取连续文本中的数字与单位)
	if msg, err := parseFlexibleNumeric(data); err == nil {
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 4. 终极兜底：如果没找到或者解析失败，走之前的通用 C51 解析逻辑
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
	re := regexp.MustCompile(`(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)`)
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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
	re := regexp.MustCompile("(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\\.[0-9]+|[0-9]+)[^a-zA-Z%]*([a-zA-Z%]+)")
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

// parseSCP23 解析 SCP-23 协议 (12字节固定帧)
func parseSCP23(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) < 10 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 识别超载状态 (O L / OL)
	if strings.HasPrefix(dataStr, "O L") || strings.HasPrefix(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   true,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 识别欠载状态 (U L / UL)
	if strings.HasPrefix(dataStr, "U L") || strings.HasPrefix(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   true,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 解析 Byte 1 符号位 ('+', '-', '1')
	sign := ""
	if data[0] == '-' {
		sign = "-"
	}

	// 解析 Byte 2~8 (索引 1~7) 7字节重量数值
	endIdx := 8
	if len(data) < 8 {
		endIdx = len(data)
	}
	rawWeightStr := strings.TrimSpace(string(data[1:endIdx]))

	weightVal := sign + rawWeightStr

	// 解析 Byte 9~10 (索引 8~9) 重量单位
	unit := ""
	if len(data) >= 10 {
		unit = strings.TrimSpace(string(data[8:10]))
	}

	isZero := (rawWeightStr == "0" || rawWeightStr == "0.0" || rawWeightStr == "0.00" || rawWeightStr == "0.000")

	msg := WeightMsg{
		IsStable:   true,
		IsNet:      false,
		IsZero:     isZero,
		WeightVal:  weightVal,
		WeightUnit: unit,
	}

	return &ScaleRespMsg{
		MsgType: m.WEIGHT_DATA,
		MsgBody: msg,
		ScaleId: scaleId,
	}, nil
}

// parseSCP24 解析 SCP-24 协议 (HEADER1,HEADER2,SIGN WEIGHT_DATA,WEIGHT_UNIT\r\n)
// 格式说明:
// HEADER1: ST(稳定) / US(不稳定)
// HEADER2: GS(毛重) / NT(净重)
// SIGN / WEIGHT: - 或 ⊔(空格) + 8字节重量数值
// UNIT: kg / lb / g / oz
func parseSCP24(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 2. 按逗号切割字段解析
	parts := strings.Split(dataStr, ",")
	if len(parts) >= 4 {
		header1 := strings.TrimSpace(parts[0])
		header2 := strings.TrimSpace(parts[1])
		rawWeight := strings.TrimSpace(parts[2])
		rawUnit := strings.TrimSpace(parts[3])

		isStable := (header1 == "ST")
		isNet := (header2 == "NT")

		sign := ""
		if strings.Contains(rawWeight, "-") {
			sign = "-"
		}

		var numBuilder strings.Builder
		for _, ch := range rawWeight {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		numStr := numBuilder.String()

		if numStr != "" {
			weightVal := sign + numStr
			isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

			msg := WeightMsg{
				IsStable:   isStable,
				IsNet:      isNet,
				IsZero:     isZero,
				WeightVal:  weightVal,
				WeightUnit: rawUnit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 3. 正则兜底解析
	re := regexp.MustCompile(`(ST|US)\s*,\s*(GS|NT)\s*,\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*,\s*([a-zA-Z%]+)`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 6 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := matches[5]

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 4. 解析失败回退到通用解析器
	return defaultSCPParser(scaleId, data)
}

// parseSCP25 解析 SCP-25 协议
// 支持三种通信数据包格式:
// 1. 标准称重模式: HEADER1,HEADER2,SIGN WEIGHT_DATA,WEIGHT_UNIT\r\n (例: ST,GS,  150.00,kg\r\n)
// 2. 连续数量/计数模式: PCS:[Count value]PCS\r\n (例: PCS:    1000 PCS\r\n)
// 3. 单重/比例模式: U.W.[Single Weight DATA]WEIGHT_UNIT\r\n (例: U.W.   0.500 KG\r\n)
func parseSCP25(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	trimmedStr := strings.TrimSpace(dataStr)

	// 2. 连续数量/计数模式 (PCS: ... PCS)
	if strings.HasPrefix(trimmedStr, "PCS:") {
		content := strings.TrimPrefix(trimmedStr, "PCS:")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		unit := "PCS"
		if strings.HasSuffix(strings.ToUpper(content), "PCS") {
			content = content[:len(content)-3]
			content = strings.TrimSpace(content)
		}

		var numBuilder strings.Builder
		for _, ch := range content {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		valStr := numBuilder.String()

		if valStr != "" {
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 3. 单重/比例模式 (U.W. ... KG)
	if strings.HasPrefix(trimmedStr, "U.W.") {
		content := strings.TrimPrefix(trimmedStr, "U.W.")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		reUW := regexp.MustCompile(`^([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
		matches := reUW.FindStringSubmatch(content)
		if matches != nil && len(matches) >= 2 {
			valStr := matches[1]
			unitStr := "kg"
			if len(matches) >= 3 && matches[2] != "" {
				unitStr = matches[2]
			}
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unitStr,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 4. 按逗号切割字段解析标准称重包
	parts := strings.Split(dataStr, ",")
	if len(parts) >= 4 {
		header1 := strings.TrimSpace(parts[0])
		header2 := strings.TrimSpace(parts[1])
		rawWeight := strings.TrimSpace(parts[2])
		rawUnit := strings.TrimSpace(parts[3])

		isStable := (header1 == "ST")
		isNet := (header2 == "NT")

		sign := ""
		if strings.Contains(rawWeight, "-") {
			sign = "-"
		}

		var numBuilder strings.Builder
		for _, ch := range rawWeight {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		numStr := numBuilder.String()

		if numStr != "" {
			weightVal := sign + numStr
			isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

			msg := WeightMsg{
				IsStable:   isStable,
				IsNet:      isNet,
				IsZero:     isZero,
				WeightVal:  weightVal,
				WeightUnit: rawUnit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 5. 正则兜底解析
	re := regexp.MustCompile(`(ST|US)\s*,\s*(GS|NT)\s*,\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*,\s*([a-zA-Z%]+)`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 6 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := matches[5]

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 6. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP28 解析 SCP-28 协议
// 协议特征说明:
// 1. HEADER1: ST(稳定) / UT(不稳定，注意为 UT 而非 US)
// 2. HEADER2: GS(毛重) / NT(净重)
// 3. 称重帧仅在 HEADER1 后有 1 个逗号，后续无逗号分隔
// 4. 支持单重模式 (U.W. 前缀) 与 计数模式 (PCS 前缀)
// 例: ST,GS  150.00kg\r\n
//
//	UT,NT - 25.50 g\r\n
//	PCS    1000 PCS\r\n
//	U.W.   0.500 KG\r\n
func parseSCP28(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	trimmedStr := strings.TrimSpace(dataStr)

	// 2. 计数模式 (PCS 前缀)
	if strings.HasPrefix(trimmedStr, "PCS") {
		content := strings.TrimPrefix(trimmedStr, "PCS")
		if strings.HasPrefix(content, ":") {
			content = strings.TrimPrefix(content, ":")
		}
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		unit := "PCS"
		if strings.HasSuffix(strings.ToUpper(content), "PCS") {
			content = content[:len(content)-3]
			content = strings.TrimSpace(content)
		}

		var numBuilder strings.Builder
		for _, ch := range content {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		valStr := numBuilder.String()

		if valStr != "" {
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 3. 单重/比例模式 (U.W. 前缀)
	if strings.HasPrefix(trimmedStr, "U.W.") {
		content := strings.TrimPrefix(trimmedStr, "U.W.")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		reUW := regexp.MustCompile(`^([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
		matches := reUW.FindStringSubmatch(content)
		if matches != nil && len(matches) >= 2 {
			valStr := matches[1]
			unitStr := "kg"
			if len(matches) >= 3 && matches[2] != "" {
				unitStr = matches[2]
			}
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unitStr,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 4. 按第一个逗号切割: parts[0] 为 HEADER1 (ST/UT), parts[1] 为 HEADER2 (GS/NT) + SIGN/WEIGHT + UNIT
	parts := strings.SplitN(dataStr, ",", 2)
	if len(parts) == 2 {
		header1 := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")

		if len(rest) >= 2 {
			header2 := rest[:2]
			isNet := (header2 == "NT")

			restData := rest[2:] // SIGN + WEIGHT DATA + UNIT

			sign := ""
			if strings.Contains(restData, "-") {
				sign = "-"
			}

			re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
			matches := re.FindStringSubmatch(restData)
			if matches != nil && len(matches) >= 3 {
				numStr := matches[1]
				unitStr := matches[2]
				weightVal := sign + numStr
				isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

				msg := WeightMsg{
					IsStable:   isStable,
					IsNet:      isNet,
					IsZero:     isZero,
					WeightVal:  weightVal,
					WeightUnit: unitStr,
				}
				return &ScaleRespMsg{
					MsgType: m.WEIGHT_DATA,
					MsgBody: msg,
					ScaleId: scaleId,
				}, nil
			}
		}
	}

	// 5. 通用正则兜底解析 (匹配 ST/UT, GS/NT)
	re := regexp.MustCompile(`(ST|UT)\s*,\s*(GS|NT)\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 5 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := ""
		if len(matches) >= 6 {
			unit = matches[5]
		}

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 6. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP26 解析 SCP-26 协议
// 帧结构说明:
// HEADER1 , HEADER2 SIGN WEIGHT_DATA WEIGHT_UNIT CR LF
// 注意事项:
// 1. HEADER1: ST (稳定) / UT (不稳定，注意此处为 UT 而非 US)
// 2. HEADER2: GS (毛重) / NT (净重)
// 3. 全帧仅 HEADER1 后包含 1 个逗号，其余无逗号分隔
// 例: ST,GS  150.00kg\r\n
//
//	UT,NT - 25.50 g\r\n
func parseSCP26(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 2. 按第一个逗号切割: parts[0] 为 HEADER1, parts[1] 为 HEADER2 + SIGN/WEIGHT + UNIT
	parts := strings.SplitN(dataStr, ",", 2)
	if len(parts) == 2 {
		header1 := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")

		if len(rest) >= 2 {
			header2 := rest[:2]
			isNet := (header2 == "NT")

			restData := rest[2:] // SIGN + WEIGHT DATA + UNIT

			sign := ""
			if strings.Contains(restData, "-") {
				sign = "-"
			}

			re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
			matches := re.FindStringSubmatch(restData)
			if matches != nil && len(matches) >= 3 {
				numStr := matches[1]
				unitStr := matches[2]
				weightVal := sign + numStr
				isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

				msg := WeightMsg{
					IsStable:   isStable,
					IsNet:      isNet,
					IsZero:     isZero,
					WeightVal:  weightVal,
					WeightUnit: unitStr,
				}
				return &ScaleRespMsg{
					MsgType: m.WEIGHT_DATA,
					MsgBody: msg,
					ScaleId: scaleId,
				}, nil
			}
		}
	}

	// 3. 通用正则兜底解析 (匹配 ST/UT, GS/NT)
	re := regexp.MustCompile(`(ST|UT)\s*,\s*(GS|NT)\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 6 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := matches[5]

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 4. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP27 解析 SCP-27 协议
// 协议特征说明:
// 1. HEADER1: ST(稳定) / US(不稳定)
// 2. HEADER2: GS 或 GA(毛重 - 注意新增 GA 标识) / NT(净重)
// 3. 计数模式: PCS [Count value] PCS\r\n (PCS 前缀无冒号)
// 4. 单重模式: U.W. [Single Weight DATA] WEIGHT_UNIT\r\n
// 例: ST,GS,  150.00kg\r\n
//
//	US,GA,-  12.34lb\r\n
//	PCS    1000 PCS\r\n
func parseSCP27(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	trimmedStr := strings.TrimSpace(dataStr)

	// 2. 计数模式 (PCS 前缀，兼容有无冒号)
	if strings.HasPrefix(trimmedStr, "PCS") {
		content := strings.TrimPrefix(trimmedStr, "PCS")
		if strings.HasPrefix(content, ":") {
			content = strings.TrimPrefix(content, ":")
		}
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		unit := "PCS"
		if strings.HasSuffix(strings.ToUpper(content), "PCS") {
			content = content[:len(content)-3]
			content = strings.TrimSpace(content)
		}

		var numBuilder strings.Builder
		for _, ch := range content {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		valStr := numBuilder.String()

		if valStr != "" {
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 3. 单重/比例模式 (U.W. 前缀)
	if strings.HasPrefix(trimmedStr, "U.W.") {
		content := strings.TrimPrefix(trimmedStr, "U.W.")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		reUW := regexp.MustCompile(`^([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
		matches := reUW.FindStringSubmatch(content)
		if matches != nil && len(matches) >= 2 {
			valStr := matches[1]
			unitStr := "kg"
			if len(matches) >= 3 && matches[2] != "" {
				unitStr = matches[2]
			}
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unitStr,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 4. 按逗号切割字段 (HEADER1, HEADER2, REST)
	parts := strings.Split(dataStr, ",")
	if len(parts) >= 3 {
		header1 := strings.TrimSpace(parts[0])
		header2 := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")
		isNet := (header2 == "NT") // GS 与 GA 均为毛重 (isNet = false)

		restData := strings.Join(parts[2:], " ")
		sign := ""
		if strings.Contains(restData, "-") {
			sign = "-"
		}

		re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
		matches := re.FindStringSubmatch(restData)
		if matches != nil && len(matches) >= 3 {
			numStr := matches[1]
			unitStr := matches[2]
			weightVal := sign + numStr
			isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

			msg := WeightMsg{
				IsStable:   isStable,
				IsNet:      isNet,
				IsZero:     isZero,
				WeightVal:  weightVal,
				WeightUnit: unitStr,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 5. 正则兜底解析 (支持 GS 和 GA)
	re := regexp.MustCompile(`(ST|US)\s*,\s*(GS|GA|NT)\s*,\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 5 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := ""
		if len(matches) >= 6 {
			unit = matches[5]
		}

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 6. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP29 解析 SCP-29 协议
// 协议特征说明:
// 1. HEADER1: ST(稳定) / US(不稳定，注意为 US)
// 2. HEADER2: GS(毛重) / NT(净重)
// 3. 称重帧仅在 HEADER1 后有 1 个逗号，后续无逗号分隔
// 例: ST,GS  150.00kg\r\n
//
//	US,NT - 25.50 g\r\n
func parseSCP29(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 2. 按第一个逗号切割: parts[0] 为 HEADER1 (ST/US), parts[1] 为 HEADER2 (GS/NT) + SIGN/WEIGHT + UNIT
	parts := strings.SplitN(dataStr, ",", 2)
	if len(parts) == 2 {
		header1 := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")

		if len(rest) >= 2 {
			header2 := rest[:2]
			isNet := (header2 == "NT")

			restData := rest[2:] // SIGN + WEIGHT DATA + UNIT

			sign := ""
			if strings.Contains(restData, "-") {
				sign = "-"
			}

			re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
			matches := re.FindStringSubmatch(restData)
			if matches != nil && len(matches) >= 3 {
				numStr := matches[1]
				unitStr := matches[2]
				weightVal := sign + numStr
				isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

				msg := WeightMsg{
					IsStable:   isStable,
					IsNet:      isNet,
					IsZero:     isZero,
					WeightVal:  weightVal,
					WeightUnit: unitStr,
				}
				return &ScaleRespMsg{
					MsgType: m.WEIGHT_DATA,
					MsgBody: msg,
					ScaleId: scaleId,
				}, nil
			}
		}
	}

	// 3. 通用正则兜底解析 (匹配 ST/US, GS/NT)
	re := regexp.MustCompile(`(ST|US)\s*,\s*(GS|NT)\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 5 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := ""
		if len(matches) >= 6 {
			unit = matches[5]
		}

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 4. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP30 解析 SCP-30 协议
// 协议特征说明:
// 1. HEADER1: ST(稳定) / UT(不稳定，注意为 UT 而非 US)
// 2. HEADER2: GS(毛重) / NT(净重)
// 3. 逗号特征: HEADER1 和 HEADER2 后均有逗号 (例: ST,GS,)，重量与单位之间无逗号
// 例: ST,GS,  150.00kg\r\n
//
//	UT,NT,-  25.50 g\r\n
func parseSCP30(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 2. 按逗号切割: parts[0] 为 HEADER1 (ST/UT), parts[1] 为 HEADER2 (GS/NT), parts[2] 为 REST (SIGN/WEIGHT DATA + UNIT)
	parts := strings.Split(dataStr, ",")
	if len(parts) >= 3 {
		header1 := strings.TrimSpace(parts[0])
		header2 := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")
		isNet := (header2 == "NT")

		restData := strings.Join(parts[2:], " ")
		sign := ""
		if strings.Contains(restData, "-") {
			sign = "-"
		}

		re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
		matches := re.FindStringSubmatch(restData)
		if matches != nil && len(matches) >= 3 {
			numStr := matches[1]
			unitStr := matches[2]
			weightVal := sign + numStr
			isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

			msg := WeightMsg{
				IsStable:   isStable,
				IsNet:      isNet,
				IsZero:     isZero,
				WeightVal:  weightVal,
				WeightUnit: unitStr,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 3. 通用正则兜底解析 (匹配 ST/UT, GS/NT)
	re := regexp.MustCompile(`(ST|UT)\s*,\s*(GS|NT)\s*,\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)?`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 5 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := ""
		if len(matches) >= 6 {
			unit = matches[5]
		}

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 4. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP31 解析 SCP-31 协议
// 协议特征说明:
//  1. 标准称重包: HEADER1,HEADER2 +/- WEIGHT_DATA WEIGHT_UNIT\r\n
//     HEADER1: ST(稳定) / US(不稳定)
//     HEADER2: GS 或 GA(毛重) / NT(净重)
//     正负号支持: +, -, 空格
//  2. 计数包: PCS: +/- COUNT_VALUE PCS\r\n (例: PCS:+   1000 PCS\r\n)
//  3. 单重包: UW: SINGLE_VALUE G/P\r\n (例: UW:   0.500 G/P\r\n)
//  4. 百分比包: PERC: +/- PERCENT_VALUE %\r\n (例: PERC:+   100.00 %\r\n)
//  5. 斤两钱分包: ST,GS,1,8,5,2\r\n (转换为 "1.852", 单位 "斤")
func parseSCP31(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	trimmedStr := strings.TrimSpace(dataStr)

	// 2. 计数模式 (PCS: 前缀)
	if strings.HasPrefix(trimmedStr, "PCS:") {
		content := strings.TrimPrefix(trimmedStr, "PCS:")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		sign := ""
		if strings.Contains(content, "-") {
			sign = "-"
		}

		unit := "PCS"
		if strings.HasSuffix(strings.ToUpper(content), "PCS") {
			content = content[:len(content)-3]
			content = strings.TrimSpace(content)
		}

		var numBuilder strings.Builder
		for _, ch := range content {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		valStr := numBuilder.String()

		if valStr != "" {
			weightVal := sign + valStr
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  weightVal,
				WeightUnit: unit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 3. 单重包模式 (UW: 前缀)
	if strings.HasPrefix(trimmedStr, "UW:") {
		content := strings.TrimPrefix(trimmedStr, "UW:")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		reUW := regexp.MustCompile(`^([0-9]+\.?[0-9]*)\s*([a-zA-Z%/]+)?`)
		matches := reUW.FindStringSubmatch(content)
		if matches != nil && len(matches) >= 2 {
			valStr := matches[1]
			unitStr := "G/P"
			if len(matches) >= 3 && matches[2] != "" {
				unitStr = matches[2]
			}
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  valStr,
				WeightUnit: unitStr,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 4. 百分比模式 (PERC: 前缀)
	if strings.HasPrefix(trimmedStr, "PERC:") {
		content := strings.TrimPrefix(trimmedStr, "PERC:")
		content = strings.TrimSuffix(content, "\n")
		content = strings.TrimSuffix(content, "\r")
		content = strings.TrimSpace(content)

		sign := ""
		if strings.Contains(content, "-") {
			sign = "-"
		}

		unit := "%"
		var numBuilder strings.Builder
		for _, ch := range content {
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numBuilder.WriteRune(ch)
			}
		}
		valStr := numBuilder.String()

		if valStr != "" {
			weightVal := sign + valStr
			isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000")
			msg := WeightMsg{
				IsStable:   true,
				IsNet:      false,
				IsZero:     isZero,
				WeightVal:  weightVal,
				WeightUnit: unit,
			}
			return &ScaleRespMsg{
				MsgType: m.WEIGHT_DATA,
				MsgBody: msg,
				ScaleId: scaleId,
			}, nil
		}
	}

	// 5. 斤两钱分模式 (ST,GS,1,8,5,2\r\n - 多逗号风格)
	parts := strings.Split(dataStr, ",")
	if len(parts) >= 6 { // HEADER1, HEADER2, Jin, Two, Money, Minute
		header1 := strings.TrimSpace(parts[0])
		header2 := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")
		isNet := (header2 == "NT")

		jin := strings.TrimSpace(parts[2])
		two := strings.TrimSpace(parts[3])
		money := strings.TrimSpace(parts[4])
		minuteStr := strings.TrimSpace(parts[5])

		minute := strings.Fields(minuteStr)[0]

		valStr := fmt.Sprintf("%s.%s%s%s", jin, two, money, minute)
		valStr = strings.TrimRight(valStr, ".")

		msg := WeightMsg{
			IsStable:   isStable,
			IsNet:      isNet,
			IsZero:     false,
			WeightVal:  valStr,
			WeightUnit: "斤",
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 6. 按第一个逗号切割解析标准称重包: HEADER1, REST (REST 为 HEADER2 +/- WEIGHT_DATA UNIT)
	if len(parts) >= 2 {
		header1 := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])

		isStable := (header1 == "ST")

		if len(rest) >= 2 {
			header2 := rest[:2]
			isNet := (header2 == "NT")

			restData := rest[2:] // +/- WEIGHT DATA UNIT

			sign := ""
			if strings.Contains(restData, "-") {
				sign = "-"
			}

			re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%/]+)`)
			matches := re.FindStringSubmatch(restData)
			if matches != nil && len(matches) >= 3 {
				numStr := matches[1]
				unitStr := matches[2]
				weightVal := sign + numStr
				isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

				msg := WeightMsg{
					IsStable:   isStable,
					IsNet:      isNet,
					IsZero:     isZero,
					WeightVal:  weightVal,
					WeightUnit: unitStr,
				}
				return &ScaleRespMsg{
					MsgType: m.WEIGHT_DATA,
					MsgBody: msg,
					ScaleId: scaleId,
				}, nil
			}
		}
	}

	// 7. 通用正则兜底解析
	re := regexp.MustCompile(`(ST|US)\s*,\s*(GS|GA|NT)\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%/]+)?`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 5 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := ""
		if len(matches) >= 6 {
			unit = matches[5]
		}

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 8. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP32 解析 SCP-32 协议
// 协议特征说明:
// 1. HEADER1: ST(稳定) / UT(不稳定，注意为 UT 而非 US)
// 2. HEADER2: GS(毛重) / NT(净重)
// 3. 分隔符特征: HEADER1 后为逗号, HEADER2 后为冒号 ':' (例: ST,GS:  150.00kg\r\n)
// 4. 超载特征: 重量数据部分传输 "--01--" (即 ST,GS:--01--kg\r\n)
// 例: ST,GS:  150.00kg\r\n
//
//	UT,NT:-  25.50 g\r\n
//	ST,GS:--01--kg\r\n
func parseSCP32(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (--01-- / OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "--01--") || strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 2. 按冒号 ':' 分割获取头部与后续数据
	// 期望结构: ST,GS :   150.00kg\r\n
	partsCol := strings.SplitN(dataStr, ":", 2)
	if len(partsCol) == 2 {
		headerPart := strings.TrimSpace(partsCol[0]) // ST,GS 或 UT,NT
		restData := strings.TrimSpace(partsCol[1])   // WEIGHT_DATA WEIGHT_UNIT

		partsComma := strings.Split(headerPart, ",")
		if len(partsComma) >= 2 {
			header1 := strings.TrimSpace(partsComma[0])
			header2 := strings.TrimSpace(partsComma[1])

			isStable := (header1 == "ST")
			isNet := (header2 == "NT")

			sign := ""
			if strings.Contains(restData, "-") {
				sign = "-"
			}

			re := regexp.MustCompile(`([0-9]+\.?[0-9]*)\s*([a-zA-Z%/]+)`)
			matches := re.FindStringSubmatch(restData)
			if matches != nil && len(matches) >= 3 {
				numStr := matches[1]
				unitStr := matches[2]
				weightVal := sign + numStr
				isZero := (numStr == "0" || numStr == "0.0" || numStr == "0.00" || numStr == "0.000" || numStr == "0.0000")

				msg := WeightMsg{
					IsStable:   isStable,
					IsNet:      isNet,
					IsZero:     isZero,
					WeightVal:  weightVal,
					WeightUnit: unitStr,
				}
				return &ScaleRespMsg{
					MsgType: m.WEIGHT_DATA,
					MsgBody: msg,
					ScaleId: scaleId,
				}, nil
			}
		}
	}

	// 3. 通用正则兜底解析 (匹配 ST/UT, GS/NT, 冒号/逗号)
	re := regexp.MustCompile(`(ST|UT)\s*,\s*(GS|NT)\s*[:,\s]\s*([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%/]+)?`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 5 {
		status := matches[1]
		mode := matches[2]
		sign := matches[3]
		val := matches[4]
		unit := ""
		if len(matches) >= 6 {
			unit = matches[5]
		}

		if sign == "-" {
			val = "-" + val
		}
		isZero := (val == "0" || val == "0.0" || val == "0.00" || val == "0.000")

		msg := WeightMsg{
			IsStable:   (status == "ST"),
			IsNet:      (mode == "NT"),
			IsZero:     isZero,
			WeightVal:  val,
			WeightUnit: unit,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 4. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}

// parseSCP33 解析 SCP-33 协议
// 协议特征说明:
// 极简数据帧，无 HEADER1 (ST/US) 与 HEADER2 (GS/NT) 前缀
// 帧结构: [+/- 符号位] [8位重量数据] [重量单位] CR LF
// 例: "+  150.00kg\r\n"
//
//	"-   12.34 g\r\n"
//	"   100.50lb\r\n"
func parseSCP33(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	if len(data) == 0 {
		return defaultSCPParser(scaleId, data)
	}

	dataStr := string(data)

	// 1. 拦截超载 (OL) 与 欠载 (UL)
	if strings.Contains(dataStr, "OL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-OL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}
	if strings.Contains(dataStr, "UL") {
		msg := WeightMsg{
			IsStable:   false,
			IsNet:      false,
			IsZero:     false,
			WeightVal:  "-UL-",
			WeightUnit: "",
		}
		return &ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}, nil
	}

	// 2. 正则提取 [可选符号 +/-] [数字部分] [单位部分]
	re := regexp.MustCompile(`([+-]?)\s*([0-9]+\.?[0-9]*)\s*([a-zA-Z%]+)`)
	matches := re.FindStringSubmatch(dataStr)
	if matches != nil && len(matches) >= 4 {
		sign := matches[1]
		valStr := matches[2]
		unitStr := matches[3]

		if sign == "-" {
			valStr = "-" + valStr
		}

		isZero := (valStr == "0" || valStr == "0.0" || valStr == "0.00" || valStr == "0.000" || valStr == "0.0000")

		msg := WeightMsg{
			IsStable:   true,  // 极简数据包默认记为稳定称重数据
			IsNet:      false, // 默认记为普通重量
			IsZero:     isZero,
			WeightVal:  valStr,
			WeightUnit: unitStr,
		}
		return &ScaleRespMsg{
			MsgType: m.WEIGHT_DATA,
			MsgBody: msg,
			ScaleId: scaleId,
		}, nil
	}

	// 3. 解析失败回退到 defaultSCPParser
	return defaultSCPParser(scaleId, data)
}
