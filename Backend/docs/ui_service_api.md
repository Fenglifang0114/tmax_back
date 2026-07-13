# UI 与 Service (Backend) 通讯接口文档

## 1. 通讯协议概述
UI 与后台服务（Service）之间的通信采用了统一的消息包裹机制。底层通讯主要基于 WebSocket 或特定的进程内桥接传递，载荷（Payload）均为序列化后的 **JSON** 字符串。

### 1.1 请求基础结构 (Request)
所有从 UI 发起的请求，均需封装为以下格式：
```json
{
  "Req": "string (具体指令枚举)",
  "ReqData": "string (业务参数对象的 JSON 字符串)"
}
```
*注：部分只读请求无需参数时，`ReqData` 传空字符串 `""` 即可。*

### 1.2 响应基础结构 (Response)
所有后台返回的消息格式如下：
```json
{
  "MsgType": "string (响应指令枚举)",
  "MsgBody": "object (根据各类请求返回的实体对象或包含 IsAck 的通用提示层)",
  "ScaleId": "int64 (仅当消息与特定秤设备强相关时附带，如推送重量等)"
}
```
*注：通用操作成功回调经常返回 `MgrRespMsg` 结构，其内部为 `{"IsAck": true, "AckData": "ok"}`。*

---

## 2. 接口按模块详情
以下列出系统各核心模块的枚举、输入端（ReqData 内的 JSON 结构）与输出端（MsgBody 结构）。

### 2.1 秤连接与基础设备管理 (Scale)
处理秤设备的搜索、添加、修改与删除。

#### 2.1.1 获取可用端口或蓝牙设备
* **请求**: `get_port_list` / `get_bt_list`
* **参数 (ReqData)**: 无 (`""`)
* **响应 (MsgType)**: `resp_ports_list` / `resp_bt_list`
* **响应类型**: `PortsListMsg`
```json
{ "PortsList": [ { "Name": "COM1" } ] }
```

#### 2.1.2 新增秤设备
* **请求**: `add_scale`
* **参数 (ReqData)**: `ReqAddScale`
```json
{
  "ScaleModel": "T-Scale-Model",
  "ScaleSn": "SN123456",
  "MediaConf": {
    "Type": 0, // 0:COM, 1:NET, 2:BT
    "MediaInfoJson": "{\"Ip\":\"192.168.1.10\",\"Port\":8080}" // ComInfo/NetInfo/BtInfo 的 JSON
  }
}
```
* **响应**: `resp_scale_add` -> `MgrRespMsg` 确认成功。

#### 2.1.3 删除与修改秤设备
* **请求指令**: `del_scale` (入参 `ReqDelScale {ScaleId: int64}`) 
* **请求指令**: `modify_scale` (入参 `ReqModifyScale`)
* **响应**: `resp_scale_del` / `resp_scale_modify`

### 2.2 PLU 与商品管理 (Product/PLU)
管理生鲜商品基础资料、PLU下发等操作。

#### 2.2.1 获取商品列表 / 按页获取
* **请求**: `get_product_list` / `get_plu_by_page`
* **按页参数 (ReqGetPluByPage)**:
```json
{
  "Page": 1,
  "PageSize": 50,
  "FieldName": "Plu",
  "Direction": "asc",
  "Search": { "Plu": "1001", "Enabled": true }
}
```
* **响应**: `resp_product_list` -> `ProductsListMsg` (包含 `[]ProductRec`)

#### 2.2.2 新增商品 (支持批量)
* **请求**: `add_product` (批量) / `add_one_product` (单个)
* **参数 (ReqAddPlu)**:
```json
{
  "PluList": [
    {
      "Plu": "1001",
      "ProductName": "Apple",
      "Price": "5.99",
      "UnitWeight": "Kg",
      "Category": "Fruit"
    }
  ],
  "Total": 1,
  "Index": 0
}
```
* **响应**: `resp_product_add` -> `MgrRespMsg`

### 2.3 配方与原料管理 (Formula & Raw Data)
大型工业、配方秤的核心业务，包含原料库和配方组合的管理。

#### 2.3.1 原料基础库维度 (Raw Type & Raw Data)
* **请求指令**: 
  * `add_raw_type` / `get_raw_type_list`
  * `add_raw_data` / `edit_raw_data` / `get_raw_data_list`
* **新增原料入参 (ReqAddRawData)**:
```json
{
  "MaterialID": "M001",
  "MaterialName": "Flour",
  "CategoryID": 1,
  "ScaleId": 1,
  "CheckCode": "0000"
}
```
* **响应**: `resp_raw_data_add` / `resp_raw_type_list`

#### 2.3.2 配方录入与使用 (Formula)
* **请求指令**: `add_formula_data` / `get_formula_list`
* **新增配方入参 (ReqAddFormulaData)**:
```json
{
  "Header": {
    "FormulaID": "F-01",
    "FormulaName": "Cake Base",
    "TotalWeight": 100.0,
    "FormulaBarcode": "12345678"
  },
  "Detail": [
    {
      "MaterialID": "M001",
      "MaterialWeight": 50.0,
      "Sequence": 1,
      "AllowableError": 0.5
    }
  ]
}
```

### 2.4 系统用户与权限控制 (System Users)
管理登录、操作员、权限映射等体系。

#### 2.4.1 用户注册与分配
* **请求**: `add_sys_user` / `update_sys_user`
* **参数 (ReqAddSysUser)**:
```json
{
  "Username": "admin",
  "Password": "password_plaintext", // 服务端会做 MD5 处理
  "RoleId": 1, // 1:超级管理员 2:普通管理员 3:操作员
  "IsEnabled": true,
  "PagesId": [1, 2, 3, 4] // 分割的界面权限权限ID列表
}
```
* **响应**: `resp_add_sys_user` -> `MgrRespMsg`

#### 2.4.2 登录认证验证
* **请求**: `login`
* **参数 (ReqLogin)**:
```json
{
  "UserName": "admin",
  "Password": "123",
  "AutoLogin": false
}
```
* **响应**: `resp_login` 

### 2.5 记录与日志提取 (Records & Logs)
各种称重流水、流速记录以及系统安全日志留存。

#### 2.5.1 提取称重日志 / 校准日志
* **请求**: `get_scale_log` / `get_sys_log`
* **参数 (ReqGetLog)**:
```json
{
  "Page": 1,
  "PageSize": 20,
  "Search": {
    "Operator": "admin",
    "StartTime": "2026-04-01 00:00:00",
    "EndTime": "2026-04-14 00:00:00"
  }
}
```

#### 2.5.2 配方记录与流速
* **请求指令**: `get_formula_rec_list` / `get_flow_rate_list`
* **响应**: 返回包含 `ReqFormulaWgtRecDetail` 结构体的数组，包含了实际重量、配比误差百分比 (`ActualErrorPct`) 等明细。

### 2.6 输出/输入端口与硬件控制配置
用于直接下发参数开启外部连接功能，报警灯光和开关量 I/O 读取。

* **获取端口状态**: `get_output_port` 
* **更新端口指令**: `update_output_port` (入参 `ReqUpdateOutputPort`)
```json
{
  "Port": 1,
  "Status": true,
  "StartTime": 0,
  "EndValue": 100.5
}
```
*响应为对应 `resp_...` 开头的状态信息。*

---
> 备注：所有的具体常量及字段类型推导源自 `svc/srvproto.go` 的枚举表，该协议具备强拓展性。如果前端传 JSON 解析失败，Service 会不动作或通过打 log 的方式输出反序列化失败警告。
