package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type EndpointMeta struct {
	Type string
	Desc string
}

// Global struct map parsing all go files
var structs = make(map[string][][]string)

func parseStructs(dir string) error {
	structPattern := regexp.MustCompile(`(?s)type\s+([A-Za-z0-9_]+)\s+struct\s*\{([\s\S]*?)\}`)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			contentBytes, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(contentBytes)
			sMatches := structPattern.FindAllStringSubmatch(content, -1)
			for _, m := range sMatches {
				name := m[1]
				body := m[2]
				lines := strings.Split(body, "\n")
				var fields [][]string
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if l != "" && !strings.HasPrefix(l, "//") {
						parts := strings.Fields(l)
						if len(parts) >= 2 {
							fName := parts[0]
							fType := parts[1]
							// clean type formatting like json tags
							fType = strings.TrimSpace(strings.Split(fType, "`")[0])
							fields = append(fields, []string{fName, fType})
						}
					}
				}
				structs[name] = fields
			}
		}
		return nil
	})
}

var hardcodedResponseMapping = map[string]EndpointMeta{
	"resp_ports_list":           {"[]string", "返回服务器系统上可用的物理或虚拟串口名称列表（如 'COM1', '/dev/ttyS0'）。"},
	"resp_bt_list":              {"[]string", "返回周围可用蓝牙设备的名称或地址列表。"},
	"resp_scales_list":          {"[]ScaleConnMedia", "返回系统中所有已配置保存的秤设备的详细清单。"},
	"resp_scale_srv_list":       {"[]SrvScaleRel", "获取所有电子秤与服务的绑定关系列表。"},
	"resp_scale_add":            {"MgrRespMsg", "新增建秤操作的回执。"},
	"resp_scale_del":            {"MgrRespMsg", "删除电子秤的回执。"},
	"resp_scale_modify":         {"MgrRespMsg", "修改秤的属性（如波特率、设备名）的反馈。"},
	"resp_detail_list":          {"[]DetailRec", "返回明细记录列表（结算流水数据）。"},
	"resp_set_scale_srv_val":    {"string", "设定秤与服务关联关系的反馈（单字符'ok'等）。"},
	"resp_get_ui_config":        {"ModeSetting", "返回UI的配置参数数据。"},
	"resp_update_ui_config":     {"string", "更新 UI 配置反馈（成功通常返回 'ok'）。"},
	"resp_get_license":          {"[]LicenseInfo", "查询当前系统的授权许可信息列表。"},
	"resp_check_license_key":    {"string", "校验新授权码是否有效的反馈字串。"},
	"resp_get_all_users":        {"[]UserRec", "返回系统中注册的所有用户信息（列表）。"},
	"resp_get_user_detail":      {"UserRec", "返回单个用户的详细内容。"},
	"resp_sys_log_add":          {"string", "系统级日志写入操作回执。"},
	"resp_get_sys_log_list":     {"[]LogRec", "系统日常操作日志流水列表。"},
	"resp_export_sys_log":       {"string", "打包导出日志文件的回执文件路径。"},
	"resp_get_output_port":      {"string", "获取当前继电器/输出端口配置状态的数据结果。"},
	"resp_change_password":      {"MgrRespMsg", "修改密码请求的反馈。"},
	"resp_login":                {"string", "登录操作的结果反馈名串。"},
	"resp_formula_type_list":    {"[]FormulaTypeInfo", "获取配方类型列表数据返回。"},
	"resp_raw_type_list":        {"[]RawTypeInfo", "获取原料类型列表返回。"},
	"resp_raw_list":             {"[]RawInfo", "获取原料清单列表记录返回。"},
	"resp_formula_list":         {"[]FormulaInfo", "获取配方库列表数据。"},
	"resp_formula_data":         {"FormulaInfo", "获取具体的单个配方记录明细。"},
	"resp_get_all_wgt_rec_list": {"[]WgtRec", "获取所有的称重记录明细流水。"},
}

// Map the Req Command to Both the Send Request Struct Name, and the Receive Response String literal + Receive Response Struct Name
type APISpec struct {
	ReqStruct  string
	RespCmd    string
	RespStruct string
}

var apiSpecs = map[string]APISpec{
	"get_port_list":         {ReqStruct: "none", RespCmd: "resp_ports_list", RespStruct: "[]string"},
	"get_bt_list":           {ReqStruct: "none", RespCmd: "resp_bt_list", RespStruct: "[]string"},
	"get_scale_list":        {ReqStruct: "none", RespCmd: "resp_scales_list", RespStruct: "[]ScaleConnMedia"},
	"get_scale_srv_list":    {ReqStruct: "none", RespCmd: "resp_get_scale_srv_list", RespStruct: "[]SrvScaleRel"},
	"add_scale":             {ReqStruct: "ReqAddScale", RespCmd: "resp_scale_add", RespStruct: "MgrRespMsg"},
	"del_scale":             {ReqStruct: "ReqDelScaleRec", RespCmd: "resp_scale_del", RespStruct: "MgrRespMsg"},
	"modify_scale":          {ReqStruct: "ReqModifyScale", RespCmd: "resp_scale_modify", RespStruct: "MgrRespMsg"},
	"modify_scale_name":     {ReqStruct: "ReqModifyScaleName", RespCmd: "resp_modify_scale_name", RespStruct: "MgrRespMsg"},
	"get_ui_conf":           {ReqStruct: "none", RespCmd: "resp_get_ui_config", RespStruct: "ModeSetting"},
	"update_ui_conf":        {ReqStruct: "ModeSetting", RespCmd: "resp_update_ui_config", RespStruct: "string"},
	"get_license":           {ReqStruct: "none", RespCmd: "resp_get_license", RespStruct: "[]LicenseInfo"},
	"check_license_key":     {ReqStruct: "string", RespCmd: "resp_check_license_key", RespStruct: "string"},
	"get_all_users":         {ReqStruct: "none", RespCmd: "resp_get_all_users", RespStruct: "[]UserRec"},
	"get_user_detail":       {ReqStruct: "string", RespCmd: "resp_get_user_detail", RespStruct: "UserRec"},
	"add_user":              {ReqStruct: "UserRec", RespCmd: "resp_add_sys_user", RespStruct: "MgrRespMsg"},
	"modify_user":           {ReqStruct: "UserRec", RespCmd: "resp_update_sys_user", RespStruct: "MgrRespMsg"},
	"del_user":              {ReqStruct: "ReqDelScaleRec", RespCmd: "resp_delete_sys_user", RespStruct: "MgrRespMsg"},
	"login":                 {ReqStruct: "string", RespCmd: "resp_login", RespStruct: "string"},
	"change_password":       {ReqStruct: "string", RespCmd: "resp_change_password", RespStruct: "MgrRespMsg"},
	"get_detail_list":       {ReqStruct: "none", RespCmd: "resp_detail_list", RespStruct: "[]DetailRec"},
	"add_product":           {ReqStruct: "ReqAddPlu", RespCmd: "resp_scale_add", RespStruct: "MgrRespMsg"},
	"del_product":           {ReqStruct: "ReqDelScaleRec", RespCmd: "resp_scale_del", RespStruct: "MgrRespMsg"},
	"modify_product":        {ReqStruct: "ProductRec", RespCmd: "resp_scale_modify", RespStruct: "MgrRespMsg"},
	"get_product_list":      {ReqStruct: "none", RespCmd: "resp_scales_list", RespStruct: "[]ProductRec"},
	"add_wifi_pwd":          {ReqStruct: "WifiRec", RespCmd: "resp_wifi_pwd_add", RespStruct: "MgrRespMsg"},
	"get_wifi_pwd_list":     {ReqStruct: "none", RespCmd: "resp_wifi_pwd_list", RespStruct: "[]WifiRec"},
	"add_raw_type":          {ReqStruct: "RawTypeInfo", RespCmd: "resp_raw_type_add", RespStruct: "MgrRespMsg"},
	"edit_raw_type":         {ReqStruct: "RawTypeInfo", RespCmd: "resp_raw_type_edit", RespStruct: "MgrRespMsg"},
	"delete_raw_type":       {ReqStruct: "string", RespCmd: "resp_raw_type_delete", RespStruct: "MgrRespMsg"},
	"add_formula_type":      {ReqStruct: "FormulaTypeInfo", RespCmd: "resp_fma_type_add", RespStruct: "MgrRespMsg"},
	"get_formula_type_list": {ReqStruct: "none", RespCmd: "resp_formula_type_list", RespStruct: "[]FormulaTypeInfo"},
	"get_raw_type_list":     {ReqStruct: "none", RespCmd: "resp_raw_type_list", RespStruct: "[]RawTypeInfo"},
	"get_raw_list":          {ReqStruct: "none", RespCmd: "resp_raw_list", RespStruct: "[]RawInfo"},
	"get_formula_list":      {ReqStruct: "none", RespCmd: "resp_formula_list", RespStruct: "[]FormulaInfo"},
	"add_formula_data":      {ReqStruct: "FormulaInfo", RespCmd: "resp_formula_add", RespStruct: "MgrRespMsg"},
	"delete_formula_data":   {ReqStruct: "string", RespCmd: "resp_formula_delete", RespStruct: "MgrRespMsg"},
	"get_all_wgt_rec_list":  {ReqStruct: "none", RespCmd: "resp_get_all_wgt_rec_list", RespStruct: "[]WgtRec"},
	"add_wgt_rec":           {ReqStruct: "WgtRec", RespCmd: "resp_add_wgt_rec", RespStruct: "MgrRespMsg"},
	"unseal_by_master_key":  {ReqStruct: "string", RespCmd: "resp_unseal_by_master_key", RespStruct: "string"},
}

var fieldDescriptions = map[string]string{
	"Id": "唯一标识ID",
	"RecId": "数据库记录主键 / 自增ID",
	"Plu": "商品 PLU (内部代码)",
	"ProductCode": "商品条码 / 产品编号",
	"ItemCode": "物料编号 / 货号",
	"Category": "类别编码 / 分类",
	"ProductName": "商品名称",
	"GeneralUnit": "常用单位标识(公斤/克等)",
	"TaxType": "税收分类标识",
	"Price": "商品单价",
	"UnitWeight": "参考单位重量",
	"Pretare": "预置皮重数值",
	"LimitHigh": "称重公差上限",
	"LimitLow": "称重公差下限",
	"CreatedAt": "记录创建时间",
	"UpdatedAt": "记录更新时间",
	"CreateBy": "创建者归属ID",
	"UpdateBy": "更新者归属ID",
	"CreateUser": "创建此记录的用户名",
	"UpdateUser": "最后更新此记录的用户名",
	"Enabled": "记录启用状态(软删除标志)",
	"Name": "用户名 / 记录名称",
	"IsFemale": "性别标志位 (true代表女性)",
	"Password": "用户密码哈希/密码",
	"Avatar": "头像媒体文件路径或特征",
	"BirthDay": "出生日期",
	"Phone": "联络电话/手机号码",
	"IdentityNo": "身份证号 / 识别证件号",
	"IsAdmin": "管理员权限标志位",
	"IsOnline": "设备实时的在线/失联标志",
	"ScaleModel": "电子秤硬件设备型号",
	"ScaleCat": "秤体大类(枚举值)",
	"ScaleSn": "设备的唯一出厂序列号(SN)",
	"ScaleId": "系统为当前连接设备分配的主键ID",
	"TMedia": "网络通信所用媒体通道类型",
	"MediaConf": "通信链路详细配置对象(内含IP与端口等)",
	"ScaleName": "设定的常用别名(门店显示称呼)",
	"SendService": "允许数据反向推送给第三方服务",
	"IsDefault": "系统默认的首选设备",
	"CustomModel": "用户自定义的展示型号",
	"InnerModel": "硬件核心的真实型号",
	"ProtocolName": "该设备连接使用的报文通讯协议",
	"Type": "媒体分类枚举值",
	"MediaInfoJson": "连接参数的JSON全集(波特率/IP配置等)",
	"Sn": "序列号 (SN)",
	"PluList": "将需要下发的商品或记录明细列表",
	"Total": "本次交互包含的记录/条目总数",
	"Index": "游标索引，用于分批次加载数据",
	"FilePaths": "目标的相对/绝对文件路径集合",
	"FilePath": "需处理或读取的目标文件路径",
	"PluId": "商品条目主键列",
	"Mode": "工作运转/业务请求模式",
	"IsAck": "命令执行状态位，true为成功",
	"AckData": "执行附带的补充/原因数据反馈",
	"TotalGross": "累计毛重统计",
	"TotalNet": "累计净重统计",
	"TotalTare": "累计皮重统计",
	"IsQualified": "质检/允差分析最终是否合格",
	"ActualFmaTotalWgt": "配方的实时整体累加重量",
	"FormulaBarcode": "当前批次或配方对应的扫描条码",
	"Scale": "关联的实例化电子秤指针(一般前台忽略)",
	"Remark": "备注补充1",
    "Remark1": "备注补充2",
    "Remark2": "备注补充3",
	"Remarks": "备注详细信息",
}

func getFieldDesc(fname string) string {
	if val, ok := fieldDescriptions[fname]; ok {
		return val
	}
	return ""
}

func main() {
	svcDir := `d:\tmaxbackend\tmaxBack\Backend\svc\`
	outPath := `d:\tmaxbackend\tmaxBack\Backend\docs\ui_service_api.html`

	// Parse all structs natively across all go code
	parseStructs(svcDir)
	
	// Add universal message
	structs["MgrRespMsg"] = [][]string{{"IsAck", "bool"}, {"AckData", "string"}}

	path := filepath.Join(svcDir, "srvproto.go")
	b, _ := os.ReadFile(path)
	content := string(b)

	reqPattern := regexp.MustCompile(`(?m)^[ \t]*([A-Z0-9_]+)[ \t]+ReqType[ \t]*=[ \t]*"([^"]+)"(?:[ \t]*//[ \t]*(.*))?`)
	sreqPattern := regexp.MustCompile(`(?m)^[ \t]*([A-Z0-9_]+)[ \t]+SReqType[ \t]*=[ \t]*"([^"]+)"(?:[ \t]*//[ \t]*(.*))?`)

	var allReqs [][]string
	allReqs = append(allReqs, reqPattern.FindAllStringSubmatch(content, -1)...)
	allReqs = append(allReqs, sreqPattern.FindAllStringSubmatch(content, -1)...)

	// Build the HTML using India SW40 structure approach and standard HTML5 semantics
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>TMax WebSocket UI API Documentation</title>
    <style>
        body { font-family: "Helvetica Neue", Helvetica, Arial, sans-serif; background: #fff; margin: 0; padding: 0; color: #333; }
        .sidebar { width: 300px; position: fixed; top: 0; bottom: 0; left: 0; background: #fafafa; border-right: 1px solid #e5e5e5; padding: 20px; overflow-y: auto; box-sizing: border-box; }
        .sidebar h2 { margin-top: 0; font-size: 16px; font-weight: bold; color: #333; text-transform: uppercase;}
        .sidebar .nav-list { list-style: none; padding: 0; margin: 0 0 20px 0; }
        .sidebar .nav-list li a { display: block; padding: 6px 10px; color: #0088cc; text-decoration: none; border-radius: 4px; font-size: 14px;}
        .sidebar .nav-list li a:hover { background: #eee; }
        .content { margin-left: 300px; padding: 40px; box-sizing: border-box; max-width: 1000px; }
        h1.page-title { font-size: 36px; padding-bottom: 10px; border-bottom: 2px solid #ddd; margin-top: 0; }
        section.api-section { border-top: 1px solid #ebebeb; padding: 30px 0; }
        section.api-section h2.api-title { font-family: "Source Sans Pro Bold", sans-serif; font-size: 26px; color: #222; margin-top: 0; }
        h3 { font-size: 18px; color: #444; border-bottom: 1px solid #eee; padding-bottom: 5px; margin-top: 20px; }
        code.req-label { background-color: #f7f7f9; color: #d14; border: 1px solid #e1e1e8; border-radius: 3px; padding: 2px 4px; font-size: 13px; font-family: Consolas, monospace; }
        .table-wrapper { margin-top: 15px; margin-bottom: 30px; box-shadow: 0 1px 3px rgba(0,0,0,0.05); border-radius: 4px; overflow: auto; }
        table { width: 100%; border-collapse: collapse; background: #fff;}
        th, td { padding: 12px 15px; text-align: left; border-bottom: 1px solid #e1e4e8; border-right: 1px solid #f1f2f4; }
        th:last-child, td:last-child { border-right: none; }
        th { background: #f6f8fa; color: #24292e; font-weight: 500; font-size: 14px; }
        td { font-size: 14px; color: #24292e; }
        .param-name { font-weight: 600; color: #032f62; font-family: Consolas, monospace; }
        .param-type { font-weight: 600; color: #d73a49; }
        .alert-info { background-color: #d9edf7; border-color: #bce8f1; color: #31708f; padding: 15px; border-radius: 4px; border: 1px solid transparent; margin-bottom: 20px; }
        .desc-text { color: #6a737d; font-size: 13px;}
    </style>
</head>
<body>
    <aside class="sidebar">
        <h2>TMax PC API</h2>
        <nav>
            <ul class="nav-list">
`
	for _, m := range allReqs {
		html += fmt.Sprintf(`                <li><a href="#api-%s">%s</a></li>`+"\n", m[2], m[2])
	}
	html += `            </ul>
        </nav>
    </aside>
    <main class="content">
        <header>
            <h1 class="page-title">WebSocket Service Method Index</h1>
            <p>本文档详尽列出了所有支持的 JSON 字段名、参数数据类型以及对应的中文语义说明。在实际通信时，所有请求都必须包裹在标准的 <code>{"Req": "接口指令", "ReqData": ...}</code> 信封结构中。本文档里详细剖析的数据类型即为 <code>ReqData</code> 或 <code>MsgBody</code> 有效载荷内部最原生的真实代码结构解析。</p>
        </header>
`

	var renderTable func(string) string
	renderTable = func(structName string) string {
		if structName == "string" || structName == "int" || structName == "bool" {
			return `<div class="alert-info">原生 ` + structName + ` 特例：该参数为一个原始值而非 JSON 复杂包络。</div>`
		}
		if structName == "" || strings.ToLower(structName) == "none" {
			return `<div class="alert-info">此字段的参数无需填写任何复杂模型 (Without Data Parameters)。通常只需要传送空值即可响应。</div>`
		}
		if strings.HasPrefix(structName, "[]") {
			baseType := structName[2:]
			if baseType == "string" {
				return `<div class="alert-info">参数为一个纯原生的 String 数组类型 (List of Strings) 。<br>例如: <code>["Value1", "Value2"]</code></div>`
			}
			tableText := `<div class="alert-info">这是一个<strong>对象数组 ` + structName + ` (List)</strong> 集合。集合内包含若干个对象，具体的单个对象的元素字段解析如下：</div>`
			tableText += renderTable(baseType)
			return tableText
		}

		if fields, ok := structs[structName]; ok {
			res := `<div class="table-wrapper"><table><thead><tr><th style="width: 30%">JSON Parameter Name</th><th style="width: 20%">Data Type</th><th>Description</th></tr></thead><tbody>`
			for _, f := range fields {
				fName := f[0]
				fType := f[1]
				isNested := false
				if _, subOk := structs[strings.ReplaceAll(fType, "[]", "")]; subOk {
					isNested = true
				}
				
				descExt := ""
				if isNested {
					descExt = "<br><span class='desc-text'>（该字段含内嵌数据结构...请参见详细定义）</span>"
				}
				
				baseDesc := getFieldDesc(fName)
				res += fmt.Sprintf(`<tr><td class="param-name">%s</td><td class="param-type">%s</td><td>%s%s</td></tr>`, fName, fType, baseDesc, descExt)
				
				// Automatically expand 1 level deep for extreme SW40 tabular readability
				if isNested {
					innerStruct := strings.ReplaceAll(fType, "[]", "")
					if innerFields, innerOk := structs[innerStruct]; innerOk {
						res += `<tr><td colspan="3" style="padding:0; background:#fbfbfb;">`
						res += `<div style="padding: 10px 20px;"><strong>⮑ 展开内嵌级字典结构 (` + innerStruct + `):</strong><br>`
						res += `<table style="background:#fff; margin-top: 10px; font-size:13px; border:1px solid #ddd;"><tbody>`
						for _, iF := range innerFields {
							iDesc := getFieldDesc(iF[0])
							res += fmt.Sprintf(`<tr><td class="param-name">%s</td><td class="param-type">%s</td><td>%s</td></tr>`, iF[0], iF[1], iDesc)
						}
						res += `</tbody></table></div></td></tr>`
					}
				}
			}
			res += `</tbody></table></div>`
			return res
		}
		
		return `<div class="alert-info">系统中定义类型为 <code>` + structName + `</code>, 但缺乏进一步下钻暴露的字典细表。</div>`
	}

	for _, m := range allReqs {
		constName := m[1]
		reqVal := m[2]
		comment := ""
		if len(m) > 3 {
			comment = strings.TrimSpace(m[3])
		}

		spec, overrides := apiSpecs[reqVal]
		reqParamObj := "none"
		respValLiteral := ""
		respDataObj := "none"
		
		if overrides {
			reqParamObj = spec.ReqStruct
			respValLiteral = spec.RespCmd
			respDataObj = spec.RespStruct
		} else {
			possibleNames := []string{ 
				"Req" + strings.ReplaceAll(strings.Title(strings.ToLower(strings.ReplaceAll(constName, "SREQ_", ""))), "_", ""),
				"Req" + strings.ReplaceAll(strings.Title(strings.ToLower(strings.ReplaceAll(constName, "REQ_", ""))), "_", ""),
			}
			for _, p := range possibleNames {
				if _, exists := structs[p]; exists {
					reqParamObj = p
					break
				}
			}
			if strings.Contains(strings.ToLower(comment), "without parameter") {
				reqParamObj = "none"
			}
			
			// Guess response logic
			respValLiteral = "resp_" + reqVal
			respDataObj = "MgrRespMsg" 
		}

		html += fmt.Sprintf(`        <section id="api-%s" class="api-section">`+"\n", reqVal)
		html += fmt.Sprintf(`            <h2 class="api-title">%s</h2>`+"\n", reqVal)
		html += fmt.Sprintf(`            <p class="desc-text">触发方法常量名称机制: <code>%s</code></p>`+"\n", constName)
		
		// 发送请求表
		html += `            <article>`+"\n"
		html += `                <h3><span style="color:#d73a49">Request Parameters</span> (传输字段体)</h3>`+"\n"
		html += fmt.Sprintf(`                <p>通过下发 <code>{"Req": "%s", "ReqData": [...如下数据模型...]}</code> 来调用。</p>`+"\n", reqVal)
		html += "                " + renderTable(reqParamObj) + "\n"

		// 响应表
		html += `                <h3><span style="color:#28a745">Response Content</span> (接口返回预期)</h3>`+"\n"
		html += fmt.Sprintf(`                <p><code>MsgType</code> 为 <code class="req-label">%s</code> 指令。</p>`+"\n", respValLiteral)
		html += fmt.Sprintf(`                <p>该消息附带的核心 <code>MsgBody</code> 数据有效荷载被解析为类型：<strong>%s</strong>，其具体内部包含如下下发属性：</p>`+"\n", respDataObj)
		html += "                " + renderTable(respDataObj) + "\n"
		html += `            </article>`+"\n"
		html += `        </section>`+"\n"
	}

	html += `    </main>
</body>
</html>`

	err := os.WriteFile(outPath, []byte(html), 0644)
	if err != nil {
		fmt.Println("Err write:", err)
		return
	}
	fmt.Println("Generated OK")
}
