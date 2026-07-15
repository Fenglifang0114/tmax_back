import sys

try:
    with open('d:/tmaxbackend/tmaxBack/Backend/svc/protocol_scp.go', 'r', encoding='utf-8') as f:
        content = f.read()

    start_idx = content.find('func parseSCP01')
    if start_idx == -1:
        print('Could not find start_idx')
        sys.exit(1)

    content = content[:start_idx]
    
    stubs = []
    for i in range(1, 22):
        num = f'{i:02d}'
        stub = f"""// parseSCP{num} 解析 SCP-{num} 协议
func parseSCP{num}(scaleId int64, data []byte) (*ScaleRespMsg, error) {{
	dataStr := string(data)
	
	// TODO: 根据 SCP-{num} 图片的特定格式，在这里细化修改
	// 例如修改正则，或者改为按字节索引 data[x:y] 截取
	re := regexp.MustCompile(`(ST|US)[^A-Za-z]+(GS|NT)[^0-9+-]*([+-]?)[^0-9]*([0-9]+\.[0-9]+|[0-9]+)[^a-zA-Z]*([a-zA-Z]+)`)
	matches := re.FindStringSubmatch(dataStr)
	
	if matches != nil && len(matches) >= 6 {{
		msg := WeightMsg{{
			IsStable:   matches[1] == "ST",
			IsNet:      matches[2] == "NT",
			WeightVal:  matches[3] + matches[4],
			WeightUnit: matches[5],
			IsZero:     matches[4] == "0" || matches[4] == "0.0" || matches[4] == "0.00" || matches[4] == "0.000",
		}}
		return &ScaleRespMsg{{MsgType: m.WEIGHT_DATA, MsgBody: msg, ScaleId: scaleId}}, nil
	}}
	
	// 如果特定解析失败，回退到通用解析
	return defaultSCPParser(scaleId, data)
}}
"""
        stubs.append(stub)

    with open('d:/tmaxbackend/tmaxBack/Backend/svc/protocol_scp.go', 'w', encoding='utf-8') as f:
        f.write(content + '\n'.join(stubs))
        
    print('SUCCESS')
except Exception as e:
    print(e)
    sys.exit(1)
