package svc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gitteamer/log"

	"tmaxsrv/cmd"
	m "tmaxsrv/comm"
	"tmaxsrv/util"
)

const (
	MODIFY_BT_OK_RESP string = "TTM:OK\r\n"
)

const (
	CONNECT_AP_OK_RESP          string = "\r\nOK\r\n"
	CONNECT_AP_FAIL_RESP        string = "+CWJAP:"
	SET_WIFI_DYNAMIC_IP_OK_RESP string = "\r\nOK\r\n"
	SET_WIFI_STATIC_IP_OK_RESP  string = "\r\nOK\r\n"
	GET_AP_INFO_OK_RESP         string = "\r\nOK\r\n"
	GET_IP_INFO_OK_RESP         string = "\r\nOK\r\n"
	GET_IP_MODE_OK_RESP         string = "\r\nOK\r\n"
	CHANG_WIFI_MODE_OK_RESP     string = "\r\nOK\r\n"
	CHANG_WIFI_MODE_FAIL_RESP   string = "\r\n+CWMODE:\r\n"
	GET_AT_VERSION_OK_RESP      string = "\r\nOK\r\n"
	GET_AT_MODE_OK_RESP         string = "\r\nOK\r\n"
)

const (
	RSSI_MAX = -50  // maximum strength of signal in dBm
	RSSI_MIN = -100 // minimum strength of signal in dBm
)

// SI_  代表scale info 的缩写
const (
	SI_DEF_PRN_INFO  = 1
	SI_FREE_PRN_INFO = 2
	SI_SERIAL_OUTPUT = 3
	SI_PLU_INFO      = 4
	SI_OL_INFO       = 5
	SI_UL_INFO       = 6
	SI_SN_INFO       = 7
)

var responseHandlerMap map[m.RespMsgType]func(int64, []byte) (ScaleRespMsg, int)

func init() {
	util.CmdsRespMap = util.CmdMap{
		0xe107:                           m.WEIGHT_DATA,
		0xe401:                           m.ZERO_UNSTABLE_CMD_RESP,
		0xe402:                           m.TARE_UNSTABLE_CMD_RESP,
		0xe103:                           m.ZERO_CMD_RESP,
		0xe105:                           m.TARE_CMD_RESP,
		0xe101:                           m.WEIGHT_DATA_RESP,
		0xfff3:                           m.REG_WEIGHT_RESP,
		0xe108:                           m.UNREG_WEIGHT_RESP,
		cmd.CMDID_PAY_BILL_ON_TMAX:       m.OPEN_BILL_SEND_RESP,
		cmd.CMDID_SET_LIMIT_TMAX:         m.SET_LIMIT_TO_SCALE_RESP,
		cmd.CMDID_SWITCH_LIMIT_TMAX:      m.SWITCH_LIMIT_RESP,
		cmd.CMDID_PAY_BILL_HEAD_TMAX:     m.REV_DETAIl_HEAD_RESP,
		cmd.CMDID_PAY_BILL_MID_TMAX:      m.REV_DETAIl_MID_RESP,
		cmd.CMDID_PAY_BILL_TAIL_TMAX:     m.REV_DETAIl_TAIL_RESP,
		0xfff6:                           m.GET_RECS_RESP,
		0xfff7:                           m.ADD_REC_RESP,
		0xfff8:                           m.DEL_REC_RESP,
		cmd.CMDID_REBOOT_TMAX:            m.REBOOT_RESP,
		0x0556:                           m.GET_BUILD_INFO_RESP,
		cmd.CMDID_READ_SCALE_INFO_TMAX:   m.GET_SCALE_INFO_RESP,
		cmd.CMDID_GET_SCALE_TIME_TMAX:    m.GET_SCALE_TIME_RESP,
		0x05f1:                           m.EN_FAC_MODE_RESP,
		0x05f2:                           m.DIS_FAC_MODE_RESP,
		0x05f3:                           m.EN_PASSTH_MODE_RESP,
		0x05f4:                           m.DIS_PASSTH_MODE_RESP,
		cmd.CMDID_GET_FACTORY_INFO_TMAX:  m.GET_FACTORY_INFO_RESP,
		cmd.CMDID_GET_RANDOM_DATA_TMAX:   m.GET_RANDOM_DATA_RESP,
		cmd.CMDID_GET_BASIC_DATA_TMAX:    m.GET_BASIC_DATA_RESP,
		cmd.CMDID_GET_WIRED_IP_TMAX:      m.GET_WIRED_IP_RESP,
		cmd.CMDID_SET_WIRED_IP_TMAX:      m.SET_WIRED_IP_RESP,
		cmd.CMDID_SET_WIRED_DHCP_TMAX:    m.SET_WIRED_DHCP_RESP,
		cmd.CMDID_GET_WIRED_DHCP_TMAX:    m.GET_WIRED_DHCP_RESP,
		cmd.CMDID_ANSWER_ALIVE_TMAX:      m.ANSWER_ALIVE_RESP,
		cmd.CMDID_ERASE_FLASH_TMAX:       m.ERASE_FLASH_RESP, //FLF//
		cmd.CMDID_WRITE_FLASH_TMAX:       m.WRITE_DATA_FLASH_RESP,
		0xff11:                           m.DOWN_PRN_FMT_RESP,
		0xff12:                           m.ERR_SERIAL_RESP,
		0xff13:                           m.GET_AP_LIST_RESP,
		0xff14:                           m.RESCAN_AP_LIST_RESP,
		0xff15:                           m.SET_WIFI_DYNAMIC_IP_RESP,
		0xff16:                           m.SET_WIFI_STATIC_IP_RESP,
		0xff17:                           m.GET_IP_INFO_RESP,
		0xff18:                           m.MODIFY_BT_NAME_RESP,
		0xff19:                           m.NO_RESP,
		0xf201:                           m.BT_PASSTH_DATA_RESP,
		0xf202:                           m.WIFI_PASSTH_DATA_RESP,
		0xff22:                           m.PRT_PASSTH_DATA_RESP,
		cmd.CMDID_SCALE_PASSTH_DATA_TMAX: m.SCALE_PASSTH_DATA,
		cmd.CMDID_DOWN_PLU_TMAX:          m.DOWN_PLU_RESP,
		cmd.CMDID_DEL_PLU_TMAX:           m.DEL_PLU_RESP,
		cmd.CMDID_SET_SCALE_TIME_TMAX:    m.SET_SCALE_TIME_RESP,
		cmd.CMDID_INSERT_PLU_TMAX:        m.INSERT_PLU_ADDR_RESP,
		cmd.CMDID_READ_FLASH_TMAX:        m.READ_FLASH_DATA_RESP,
		cmd.CMDID_ERASE_INSERT_PLU_TMAX:  m.ERASE_INSERT_PLU_RESP,
		cmd.CMDID_MODIFY_VAR_TMAX:        m.MODIFY_VAR_RESP,
		cmd.CMDID_EN_FACTORY_MODE:        m.EN_FACTORY_MODE_RESP,

		cmd.CMDID_SET_CAL_WGT_TMAX: m.SET_CAL_WGT_RESP,

		cmd.CMDID_SET_DECIMAL_VALUE_TMAX: m.SET_DECIMAL_VALUE_RESP,
		cmd.CMDID_GET_DECIMAL_VALUE_TMAX: m.GET_DECIMAL_VALUE_RESP,

		cmd.CMDID_SET_MAX_RANGE1_TMAX: m.SET_MAX_RANGE1_RESP,
		cmd.CMDID_GET_MAX_RANGE1_TMAX: m.GET_MAX_RANGE1_RESP,
		cmd.CMDID_SET_MAX_RANGE2_TMAX: m.SET_MAX_RANGE2_RESP,
		cmd.CMDID_GET_MAX_RANGE2_TMAX: m.GET_MAX_RANGE2_RESP,

		cmd.CMDID_SET_GADUATION_VALUE1_TMAX: m.SET_GADUATION1_VALUE_RESP,
		cmd.CMDID_GET_GADUATION_VALUE1_TMAX: m.GET_GADUATION1_VALUE_RESP,
		cmd.CMDID_SET_GADUATION_VALUE2_TMAX: m.SET_GADUATION2_VALUE_RESP,
		cmd.CMDID_GET_GADUATION_VALUE2_TMAX: m.GET_GADUATION2_VALUE_RESP,

		cmd.CMDID_SET_ZERO_TRACK_TMAX: m.SET_ZERO_TRACKING_RESP,
		cmd.CMDID_GET_ZERO_TRACK_TMAX: m.GET_ZERO_TRACKING_RESP,

		cmd.CMDID_SET_INIT_ZERO_TMAX: m.SET_INITIAL_ZERO_RESP,
		cmd.CMDID_GET_INIT_ZERO_TMAX: m.GET_INITIAL_ZERO_RESP,

		cmd.CMDID_SET_MANUAL_ZERO_TMAX: m.SET_MANUAL_ZERO_RESP,
		cmd.CMDID_GET_MANUAL_ZERO_TMAX: m.GET_MANUAL_ZERO_RESP,

		cmd.CMDID_SET_GRAVITY_ACCEL_TMAX: m.SET_GRAV_ACC_RESP,
		cmd.CMDID_GET_GRAVITY_ACCEL_TMAX: m.GET_GRAV_ACC_RESP,

		cmd.CMDID_SET_WGT_UNIT_TMAX: m.SET_WEIGHT_UNIT_RESP,
		cmd.CMDID_GET_WGT_UNIT_TMAX: m.GET_WEIGHT_UNIT_RESP,

		cmd.CMDID_GET_SEAL_STATUS_TMAX:      m.GET_SEAL_STATUS_RESP,
		cmd.CMDID_SET_SOFT_SEAL_TMAX:        m.SOFT_SEAL_RESP,
		cmd.CMDID_REMOVE_SOFT_SEAL_TMAX:     m.REMOVE_SOFT_SEAL_RESP,
		cmd.CMDID_REMOVE_SOFT_SEAL_ONCE_TMX: m.REMOVE_SOFT_SEAL_ONCE_RESP,

		cmd.CMDID_SET_SERIAL_PORT_TMAX: m.SET_SERIAL_PORT_RESP,
		cmd.CMDID_GET_SERIAL_PORT_TMAX: m.GET_SERIAL_PORT_RESP,

		cmd.CMDID_GET_MODEL_TMAX: m.GET_MODEL_RESP,
		cmd.CMDID_CONT_CODE_TMAX: m.CONT_CODE_RESP,

		0xff25: m.UNKNOWN_DATA,
	}

	responseHandlerMap = map[m.RespMsgType]func(int64, []byte) (ScaleRespMsg, int){
		m.WEIGHT_DATA:   handleWeightDataMsg,
		m.ZERO_CMD_RESP: handleZeroCmdResp,
		m.TARE_CMD_RESP: handleTareCmdResp,

		m.ZERO_UNSTABLE_CMD_RESP: handleZeroUnstableCmdResp,
		m.TARE_UNSTABLE_CMD_RESP: handleTareUnstableCmdResp,

		m.WEIGHT_DATA_RESP:          handleWeightDataResp,
		m.REG_WEIGHT_RESP:           handleRegWeightResp,
		m.UNREG_WEIGHT_RESP:         handleUnregWeightResp,
		m.GET_RECS_RESP:             handleGetRecsResp,
		m.ADD_REC_RESP:              handleAddRecResp,
		m.DEL_REC_RESP:              handleDelRecResp,
		m.EN_FAC_MODE_RESP:          handleEnFacModeResp,
		m.DIS_FAC_MODE_RESP:         handleDisFacModeResp,
		m.EN_PASSTH_MODE_RESP:       handleEnPassthModeResp,
		m.DIS_PASSTH_MODE_RESP:      handleDisPassthModeResp,
		m.ERASE_FLASH_RESP:          handleEraseFlashResp,
		m.WRITE_DATA_FLASH_RESP:     handleWriteDataFlashResp,
		m.DOWN_PRN_FMT_RESP:         handleDownPrnFmtResp,
		m.ERR_SERIAL_RESP:           handleErrSerialResp,
		m.GET_AP_LIST_RESP:          handleGetApListResp,
		m.RESCAN_AP_LIST_RESP:       handleRescanApListResp,
		m.SET_WIFI_DYNAMIC_IP_RESP:  handleSetWifiDynamicIpResp,
		m.SET_WIFI_STATIC_IP_RESP:   handleSetWifiStaticIpResp,
		m.GET_IP_INFO_RESP:          handleGetIpInfoResp,
		m.GET_IP_MODE_RESP:          handleGetIpModeResp,
		m.MODIFY_BT_NAME_RESP:       handleModifyBtNameResp,
		m.BT_PASSTH_DATA_RESP:       handleBTPassthResp,
		m.WIFI_PASSTH_DATA_RESP:     handleWifiPassthResp,
		m.GET_BUILD_INFO_RESP:       handleGetBuildInfoResp,
		m.GET_SCALE_INFO_RESP:       handleGetScaleInfoResp,
		m.GET_FACTORY_INFO_RESP:     handleGetFactoryInfoResp,
		m.GET_SCALE_TIME_RESP:       handleGetScaleTimeResp,
		m.SET_SCALE_TIME_RESP:       handleSetScaleTimeResp,
		m.DOWN_PLU_RESP:             handleDownPluResp,
		m.DOWN_FIRMWARE_WIFI_RESP:   handleDownFirmwareWifiResp,
		m.DEL_PLU_RESP:              handleDelPluResp,
		m.INSERT_PLU_ADDR_RESP:      handleInsertPluResp,
		m.READ_FLASH_DATA_RESP:      handleReadFlashDataResp,
		m.ERASE_INSERT_PLU_RESP:     handleEraseInsertPluResp,
		m.REBOOT_RESP:               handleRebootResp,
		m.MODIFY_VAR_RESP:           handleModifyVarResp,
		m.EN_FACTORY_MODE_RESP:      handleEnFactoryModeResp,
		m.GET_RANDOM_DATA_RESP:      handleGetRandomDataResp,
		m.DOWN_DEFAULT_PRN_FMT_RESP: handleDownDefaultPrnFmtResp,
		m.DOWN_FACTORY_INFO_FC_RESP: handleDownFactorInfoFcResp,
		m.DOWN_FACTORY_INFO_RESP:    handleDownFactorInfoResp,
		m.GET_EEPROM_TO_BIN_RESP:    handleGetEepromToBinResp,
		m.SET_EEPROM_FROM_BIN_RESP:  handleSetEepromFromBinResp,
		m.GET_BASIC_DATA_RESP:       handleGetBasicDataResp,
		m.SET_LIMIT_TO_SCALE_RESP:   handleSetLimitResp,
		m.SWITCH_LIMIT_RESP:         handleSwitchLimitResp,
		m.REV_DETAIl_HEAD_RESP:      handleRevDetailHeadResp,
		m.REV_DETAIl_MID_RESP:       handleRevDetailMidResp,
		m.REV_DETAIl_TAIL_RESP:      handleRevDetailTailResp,
		m.OPEN_BILL_SEND_RESP:       handleOpenBillSendResp,
		m.ANSWER_ALIVE_RESP:         handleAnswerAliveResp,
		m.SET_MAX_RANGE1_RESP:       handleSetMaxRange1Resp,
		m.SET_MAX_RANGE2_RESP:       handleSetMaxRange2Resp,
		m.SET_CAL_WGT_RESP:          handleCalWgtResp,
		m.SET_DECIMAL_VALUE_RESP:    handleSetDecimalValueResp,
		m.GET_DECIMAL_VALUE_RESP:    handleGetDecimalValueResp,
		m.GET_MAX_RANGE1_RESP:       handleGetMaxRange1Resp,
		m.GET_MAX_RANGE2_RESP:       handleGetMaxRange2Resp,
		m.SET_GADUATION1_VALUE_RESP: handleSetGaduation1ValueResp,
		m.GET_GADUATION1_VALUE_RESP: handleGetGaduation1ValueResp,
		m.SET_GADUATION2_VALUE_RESP: handleSetGaduation2ValueResp,
		m.GET_GADUATION2_VALUE_RESP: handleGetGaduation2ValueResp,
		m.SET_WEIGHT_UNIT_RESP:      handleSetWeightUnitResp,
		m.GET_WEIGHT_UNIT_RESP:      handleGetWeightUnitResp,
		m.SET_INITIAL_ZERO_RESP:     handleSetInitialZeroResp,
		m.SET_MANUAL_ZERO_RESP:      handleSetManualZeroResp,
		m.SET_ZERO_TRACKING_RESP:    handleSetZeroTrackingResp,
		m.SET_GRAV_ACC_RESP:         handleSetGravAccResp,
		m.GET_INITIAL_ZERO_RESP:     handleGetInitialZeroResp,
		m.GET_MANUAL_ZERO_RESP:      handleGetManualZeroResp,
		m.GET_ZERO_TRACKING_RESP:    handleGetZeroTrackingResp,
		m.GET_GRAV_ACC_RESP:         handleGetGravAccResp,

		m.GET_WIRED_IP_RESP:   handleGetWiredIpResp,
		m.GET_WIRED_DHCP_RESP: handleGetWiredDhcpResp,
		m.SET_WIRED_IP_RESP:   handleSetWiredIpResp,
		m.SET_WIRED_DHCP_RESP: handleSetWiredDhcpResp,

		m.GET_SEAL_STATUS_RESP:       handleGetSealStatusResp,
		m.SOFT_SEAL_RESP:             handleSoftSealResp,
		m.REMOVE_SOFT_SEAL_RESP:      handleRemoveSoftSealResp,
		m.REMOVE_SOFT_SEAL_ONCE_RESP: handleRemoveSoftSealOnceResp,
		m.CONT_CODE_RESP:             handleContCodeResp,

		m.SET_SERIAL_PORT_RESP: handleSetSerialPortResp,
		m.GET_SERIAL_PORT_RESP: handleGetSerialPortResp,

		m.GET_MODEL_RESP: handleGetModelResp,
	}

	// example usage: call the handler for the WEIGHT_DATA message
	// msg := "some message"
	// responseHandlerMap[WEIGHT_DATA](msg)
}

func extractMessageTMAX(scaleId int64, bufs *util.CircularBuffer, msgType m.RespMsgType) ScaleRespMsg {
	data := bufs.PeekAll()
	handler := responseHandlerMap[msgType]
	if handler == nil {
		log.Error("handler not found, msgType: %v", msgType)
	}
	resp, shouldRemoveLen := handler(scaleId, data)
	bufs.DequeueN(shouldRemoveLen)

	return resp
}

func extractScalePassthDataTMAX(s *Scale, bufs *util.CircularBuffer, msgType m.RespMsgType) ScaleRespMsg {
	println(msgType)
	data := bufs.PeekAll()
	resp, shouldRemoveLen := handleScalePassthData(s.Id, data, s.IsScalePassthHex)
	bufs.DequeueN(shouldRemoveLen)

	return resp
}

func handleWeightDataMsg(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// fmt.Printf("recived%s", data)
	weightMsg, err := retrieveWeight(data)
	if err != nil {
		return ScaleRespMsg{}, len(data)
	}
	weightStr, err := json.MarshalToString(weightMsg)
	if err != nil {
		fmt.Printf("%v", weightStr)
	}
	respMsg := ScaleRespMsg{MsgType: m.WEIGHT_DATA, MsgBody: weightStr, ScaleId: scaleId}
	return respMsg, len(data)
}

func retrieveWeight(data []byte) (WeightMsg, error) {
	dataStr := string(data)
	dataStr = strings.TrimSpace(dataStr)
	fields := strings.Split(dataStr, ",")

	if len(fields) != 3 {
		if len(fields[0]) < MIN_PACK_SIZE {
			return WeightMsg{}, fmt.Errorf("no packet")
		}

		weightVal := strings.TrimRight(dataStr, "\r\n")
		weightVal = strings.TrimSpace(weightVal)
		if weightVal == "--OL--" || weightVal == "--UL--" {
			return WeightMsg{WeightVal: weightVal, WeightUnit: ""}, nil
		}

		// Regex pattern to match "ST,NT90PCS" or "ST,GS 80%"
		pattern := `^ZE,ST,\s*([A-Za-z0-9]+)\s*([%A-Za-z]+)$`
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(weightVal)
		if len(match) == 3 {
			weightVal := match[1]
			weightUnit := match[2]
			return WeightMsg{WeightVal: weightVal, WeightUnit: weightUnit}, nil
		}

		return WeightMsg{}, fmt.Errorf("invalid weight format: %v", weightVal)
	}

	weightMsg := WeightMsg{}
	weightMsg.IsZero = strings.Contains(strings.TrimSpace(fields[0]), "ZE")
	weightMsg.IsStable = strings.Contains(strings.TrimSpace(fields[1]), "ST")
	weightMsg.IsNet = strings.Contains(strings.TrimSpace(fields[2]), "NT")

	regexp, err := regexp.Compile(`([0-9:.-]+)\s*([a-zA-Z%:]+)`)
	if err != nil {
		return WeightMsg{}, err
	}

	match := regexp.FindStringSubmatch(strings.ReplaceAll(fields[2], " ", ""))
	if len(match) != 3 {
		return WeightMsg{}, fmt.Errorf("finding substring error: %v", fields[2])
	}

	weightMsg.WeightVal = strings.TrimSpace(match[1])
	weightMsg.WeightUnit = strings.TrimSpace(match[2])

	return weightMsg, nil
}

// 处理连续发送内码响应
func handleContCodeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	// fmt.Printf("recived%s", data)
	weightMsg, err := retrieveCode(data)
	if err != nil {
		return ScaleRespMsg{}, len(data)
	}
	weightStr, err := json.MarshalToString(weightMsg)
	if err != nil {
		fmt.Printf("%v", weightStr)
	}
	respMsg := ScaleRespMsg{MsgType: m.CONT_CODE_RESP, MsgBody: weightStr, ScaleId: scaleId}

	return respMsg, len(data)
}

func retrieveCode(data []byte) (CodeMsg, error) {
	dataStr := strings.TrimSpace(string(data))

	// 按空格分割字符串
	fields := strings.Split(dataStr, " ")

	// 检查是否正好有两部分
	if len(fields) != 2 {
		return CodeMsg{}, fmt.Errorf("invalid format: expected 2 fields, got %d", len(fields))
	}

	// 检查第一部分是否是0或1
	stableFlag := strings.TrimSpace(fields[0])
	if stableFlag != "0" && stableFlag != "1" {
		return CodeMsg{}, fmt.Errorf("invalid stable flag: must be 0 or 1, got %s", stableFlag)
	}

	// 解析第二部分为内码值
	codeValStr := strings.TrimSpace(fields[1])
	codeVal, err := strconv.ParseInt(codeValStr, 10, 64)
	if err != nil {
		return CodeMsg{}, fmt.Errorf("invalid code value: %v", err)
	}

	// 检查内码值是否大于0
	if codeVal <= 0 {
		return CodeMsg{}, fmt.Errorf("code value must be greater than 0, got %d", codeVal)
	}

	// 创建并返回CodeMsg
	return CodeMsg{
		IsStable: stableFlag == "1", // 1表示稳定，0表示不稳定
		CodeVal:  codeVal,
	}, nil
}

func handleZeroCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	if data[0] == 0x06 {
		msg.MsgType = m.ZERO_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "ok"
	} else {
		msg.MsgType = m.ZERO_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleTareCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	if data[0] == 0x06 {
		msg.MsgType = m.TARE_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "ok"
	} else {
		msg.MsgType = m.TARE_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleZeroUnstableCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	if data[0] == 0x06 {
		msg.MsgType = m.ZERO_UNSTABLE_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "ok"
	} else {
		msg.MsgType = m.ZERO_UNSTABLE_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleTareUnstableCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	if data[0] == 0x06 {
		msg.MsgType = m.TARE_UNSTABLE_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "ok"
	} else {
		msg.MsgType = m.TARE_UNSTABLE_CMD_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleWeightDataResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleRegWeightResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	if data[0] == 0x06 {
		msg.MsgType = m.REG_WEIGHT_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "ok"
	} else {
		msg.MsgType = m.REG_WEIGHT_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleUnregWeightResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	if data[0] == 0x06 {
		msg.MsgType = m.UNREG_WEIGHT_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "ok"
	} else {
		msg.MsgType = m.UNREG_WEIGHT_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleGetBuildInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	// 5A A5 00 30 05 56 00 78 78 78 78 78 78 78 78 2D 78 78 78 78 2D 78 78 78 78 2D 78 78 78 78 2D 78 78 78 78 78 78 78 78 78 78 78 78 00 1C EE D5 0B A5 5A 5A A5 00 0C 05 56 00 06 8E 0B A1 55 A5 5A
	str := string(data)

	if len(str) >= 36 {
		msg.MsgType = m.GET_BUILD_INFO_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = str[:36]
	} else if data[0] == 0x06 {

	} else {
		msg.MsgType = m.GET_BUILD_INFO_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

type SIFromScale struct {
	ScaleSn   string   `json:"ScaleSn"`
	ModelName string   `json:"ModelName"`
	AddrInfos []string `json:"AddrInfos"`
}

type SIAddrInfos struct {
	Type     int `json:"Type"`
	Addr     int `json:"Addr"`
	Lenth    int `json:"Lenth"`
	EraseLen int `json:"EraseLen"`
}

type FIFromScale struct {
	ScaleSn   string `json:"ScaleSn"`
	ModelName string `json:"ModelName"`
}

func handleSetScaleTimeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_SCALE_TIME_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_SCALE_TIME_RESP, MsgBody: "fail"}, len(data)
	}
	// TODO: Implement function
}

func handleGetScaleInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	//此处处理收到的数据
	var scaleInfo SIFromScale
	var siAddrInfos SIAddrInfos
	msg := ScaleRespMsg{}
	// 0 75 1 5 84 45 77 65 88 2 10 84 45 83 67 65 76 69 48 48 49 3 12 8 0 48 0 0 0 32 0 0 0 8 0 4 12 8 1 224 0 0 0 32 0 0 0 8 0 5 12 8 1 216 0 0 0 8 0 0 0 8 0 6 12 0 6 160 0 0 16 0 0 0 0 16 0
	if data[0] == 0x15 || data[0] == 0x06 {
		msg.MsgType = m.GET_SCALE_INFO_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)

	}
	dataLen := int(binary.BigEndian.Uint32(data[:4]))

	loopTime := dataLen / 16
	dataStart := 4
	for i := 0; i < loopTime; i++ {

		id := int(binary.BigEndian.Uint32(data[dataStart : dataStart+4]))
		switch id {
		case SI_DEF_PRN_INFO, SI_FREE_PRN_INFO, SI_SERIAL_OUTPUT, SI_PLU_INFO, SI_OL_INFO, SI_UL_INFO, SI_SN_INFO:

			var infoByte = data[dataStart+4 : dataStart+16]
			siAddrInfos.Type = id
			siAddrInfos.Addr, siAddrInfos.Lenth, siAddrInfos.EraseLen = getInfoAddrLen(infoByte)
			addrInfoStr, _ := json.MarshalToString(siAddrInfos)
			scaleInfo.AddrInfos = append(scaleInfo.AddrInfos, addrInfoStr)

		default:

		}
		dataStart = dataStart + 16
	}
	msg.MsgType = m.GET_SCALE_INFO_RESP
	msg.ScaleId = scaleId
	msg.MsgBody, _ = json.MarshalToString(scaleInfo)

	return msg, len(data)
}

func handleGetFactoryInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	//     20240417@FLF
	// 5A A5 00 1E 05 F6 00 05 54 2D 4D 41 58 0C 31 30 38 30 30 38 30 32 30 30 30 38 62 C6 A8 39 A5 5A  //旧的
	//前8位是model name 后16位是SN
	var factoryInfo FIFromScale
	msg := ScaleRespMsg{}
	msg.MsgType = m.GET_FACTORY_INFO_RESP
	msg.ScaleId = scaleId
	msg.MsgBody = "fail"
	modelNameStr := ""
	snStr := ""
	if len(data) != 24 {
		return msg, len(data)
	}
	for i := 0; i < 8; i++ {
		if data[i] != 0xff {
			modelNameStr = modelNameStr + string(data[i])
		} else {
			break
		}
	}
	for i := 8; i < 24; i++ {
		if data[i] != 0xff {
			snStr = snStr + string(data[i])
		} else {
			break
		}
	}
	factoryInfo.ModelName = modelNameStr
	factoryInfo.ScaleSn = snStr
	msg.MsgBody, _ = json.MarshalToString(factoryInfo)
	return msg, len(data)
}

func handleGetScaleTimeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	//TODO:       20231101@FLF
	//此处处理收到的数据
	msg := ScaleRespMsg{}

	if len(data) < 4 {
		msg.MsgType = m.GET_SCALE_TIME_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}
	dataTimeInt := int(binary.BigEndian.Uint32(data[0:4]))
	strTime := strconv.Itoa(dataTimeInt)
	if strTime == "" {
		msg.MsgType = m.GET_SCALE_TIME_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	msg.MsgType = m.GET_SCALE_TIME_RESP
	msg.ScaleId = scaleId
	msg.MsgBody = "ok" + "," + strTime

	return msg, len(data)
}

func getInfoAddrLen(data []byte) (int, int, int) {
	return int(binary.BigEndian.Uint32(data[:4])), int(binary.BigEndian.Uint32(data[4:8])), int(binary.BigEndian.Uint32(data[8:12]))

}

func handleGetRecsResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleAddRecResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleDelRecResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleEnFacModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_FAC_MODE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_FAC_MODE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleAnswerAliveResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x05 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.ANSWER_ALIVE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.ANSWER_ALIVE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleSetMaxRange1Resp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MAX_RANGE1_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MAX_RANGE1_RESP, MsgBody: "fail"}, len(data)
	}
}
func handleSetMaxRange2Resp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MAX_RANGE2_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MAX_RANGE2_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleGetMaxRange2Resp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) < 4 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MAX_RANGE2_RESP, MsgBody: "fail"}, len(data)
	}
	maxRange := int(binary.BigEndian.Uint32(data[0:4]))
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MAX_RANGE2_RESP, MsgBody: fmt.Sprintf("%d", maxRange)}, len(data)

}

func handleGetMaxRange1Resp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) < 4 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MAX_RANGE1_RESP, MsgBody: "fail"}, len(data)
	}
	maxRange := int(binary.BigEndian.Uint32(data[0:4]))
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MAX_RANGE1_RESP, MsgBody: fmt.Sprintf("%d", maxRange)}, len(data)

}

func handleGetGravAccResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) < 4 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_GRAV_ACC_RESP, MsgBody: "fail"}, len(data)
	}
	gravInt := int(binary.BigEndian.Uint32(data[0:4]))
	gravDouble := float64(gravInt) / 100000
	gravStr := strconv.FormatFloat(gravDouble, 'f', 5, 64)
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_GRAV_ACC_RESP, MsgBody: gravStr}, len(data)

}

func formatIPv4(b []byte) string {
	if len(b) != 4 {
		return "0.0.0.0"
	}
	return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
}

type IpInfo struct {
	Ip      string `json:"Ip"`
	Gateway string `json:"Gateway"`
	Netmask string `json:"Netmask"`
}

func handleGetWiredIpResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) < 12 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIRED_IP_RESP, MsgBody: "fail"}, len(data)
	}
	ip := formatIPv4(data[0:4])
	gateway := formatIPv4(data[4:8])
	netmask := formatIPv4(data[8:12])

	IpInfo := IpInfo{
		Ip:      ip,
		Gateway: gateway,
		Netmask: netmask,
	}
	jsonStr, _ := json.MarshalToString(IpInfo)
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIRED_IP_RESP, MsgBody: jsonStr}, len(data)
}

func handleGetWiredDhcpResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	if len(data) != 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIRED_DHCP_RESP, MsgBody: "fail"}, len(data)
	}

	if data[0] == 0x00 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIRED_DHCP_RESP, MsgBody: "false"}, len(data)
	}

	if data[0] == 0x01 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIRED_DHCP_RESP, MsgBody: "true"}, len(data)
	}

	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIRED_DHCP_RESP, MsgBody: "fail"}, len(data)
}

func handleSetWiredIpResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) != 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIRED_IP_RESP, MsgBody: "fail"}, len(data)
	}

	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIRED_IP_RESP, MsgBody: "ok"}, len(data)
	}

	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIRED_IP_RESP, MsgBody: "fail"}, len(data)
}

func handleSetWiredDhcpResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) != 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIRED_DHCP_RESP, MsgBody: "fail"}, len(data)
	}
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIRED_DHCP_RESP, MsgBody: "ok"}, len(data)
	}
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIRED_DHCP_RESP, MsgBody: "fail"}, len(data)
}

// 获取型号
func handleGetModelResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) < 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MODEL_RESP, MsgBody: "fail"}, len(data)
	}
	model := string(data)
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MODEL_RESP, MsgBody: model}, len(data)
}

func handleGetSealStatusResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) != 2 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_SEAL_STATUS_RESP, MsgBody: "fail"}, len(data)
	}

	status1 := data[0] != 0 // 第一个字节
	status2 := data[1] != 0 // 第二个字节

	msgBody := fmt.Sprintf("%v,%v", status1, status2)

	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_SEAL_STATUS_RESP, MsgBody: msgBody}, len(data)

}

func handleSetSerialPortResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) >= 1 && data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_SERIAL_PORT_RESP, MsgBody: "ok"}, len(data)
	}
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_SERIAL_PORT_RESP, MsgBody: "fail"}, len(data)
}

func handleGetSerialPortResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) < 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_SERIAL_PORT_RESP, MsgBody: "fail"}, len(data)
	}
	str := string(data)
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_SERIAL_PORT_RESP, MsgBody: str}, len(data)
}

func handleSoftSealResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) != 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SOFT_SEAL_RESP, MsgBody: "fail"}, len(data)
	}
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SOFT_SEAL_RESP, MsgBody: "ok"}, len(data)
	}
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SOFT_SEAL_RESP, MsgBody: "fail"}, len(data)

}

func handleRemoveSoftSealResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) != 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REMOVE_SOFT_SEAL_RESP, MsgBody: "fail"}, len(data)
	}
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REMOVE_SOFT_SEAL_RESP, MsgBody: "ok"}, len(data)
	}
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REMOVE_SOFT_SEAL_RESP, MsgBody: "fail"}, len(data)

}

func handleRemoveSoftSealOnceResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) != 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REMOVE_SOFT_SEAL_ONCE_RESP, MsgBody: "fail"}, len(data)
	}
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REMOVE_SOFT_SEAL_ONCE_RESP, MsgBody: "ok"}, len(data)
	}
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REMOVE_SOFT_SEAL_ONCE_RESP, MsgBody: "fail"}, len(data)
}

func handleSetDecimalValueResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_DECIMAL_VALUE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_DECIMAL_VALUE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleGetDecimalValueResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_DECIMAL_VALUE_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	//解析data[0] int值应该小于4
	value := int(data[0])
	if value < 4 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_DECIMAL_VALUE_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_DECIMAL_VALUE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleGetGaduation1ValueResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_GADUATION1_VALUE_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	value := int(data[0])
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_GADUATION1_VALUE_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)

}

func handleGetGaduation2ValueResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_GADUATION2_VALUE_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	value := int(data[0])
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_GADUATION2_VALUE_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)

}

func handleGetManualZeroResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_MANUAL_ZERO_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	value := int(data[0])
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_MANUAL_ZERO_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)

}

func handleGetInitialZeroResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_INITIAL_ZERO_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	value := int(data[0])
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_INITIAL_ZERO_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)
}

func handleGetZeroTrackingResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_ZERO_TRACKING_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	value := int(data[0])
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_ZERO_TRACKING_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)
}

func handleGetWeightUnitResp(scaleId int64, data []byte) (ScaleRespMsg, int) {

	msg := ScaleRespMsg{}
	if len(data) != 1 {
		msg.MsgType = m.GET_WEIGHT_UNIT_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
		return msg, len(data)
	}

	value := int(data[0])
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WEIGHT_UNIT_RESP, MsgBody: fmt.Sprintf("%d", value)}, len(data)
}

func handleSetGaduation1ValueResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_GADUATION1_VALUE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_GADUATION1_VALUE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleSetGaduation2ValueResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_GADUATION2_VALUE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_GADUATION2_VALUE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleSetWeightUnitResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WEIGHT_UNIT_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WEIGHT_UNIT_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleSetInitialZeroResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_INITIAL_ZERO_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_INITIAL_ZERO_RESP, MsgBody: "fail"}, len(data)
	}
}
func handleSetManualZeroResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MANUAL_ZERO_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MANUAL_ZERO_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleSetGravAccResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_GRAV_ACC_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_GRAV_ACC_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleSetZeroTrackingResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_ZERO_TRACKING_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_ZERO_TRACKING_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleCalWgtResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_CAL_WGT_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_CAL_WGT_RESP, MsgBody: "fail"}, len(data)
	}
}

// 最新修改的打开工厂模式
func handleEnFactoryModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_FACTORY_MODE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_FACTORY_MODE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleGetRandomDataResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if len(data) == 2 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_RANDOM_DATA_RESP, MsgBody: data}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_RANDOM_DATA_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleRebootResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REBOOT_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REBOOT_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleDisFacModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_FAC_MODE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_FAC_MODE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleEnPassthModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_PASSTH_MODE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_PASSTH_MODE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleDisPassthModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_PASSTH_MODE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_PASSTH_MODE_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleEraseFlashResp(scaleId int64, data []byte) (ScaleRespMsg, int) { //FLF
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.ERASE_FLASH_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.ERASE_FLASH_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleWriteDataFlashResp(scaleId int64, data []byte) (ScaleRespMsg, int) { //FLF
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.WRITE_DATA_FLASH_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.WRITE_DATA_FLASH_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleModifyVarResp(scaleId int64, data []byte) (ScaleRespMsg, int) { //FLF
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.MODIFY_VAR_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.MODIFY_VAR_RESP, MsgBody: "fail"}, len(data)
	}
}

func handleDownDefaultPrnFmtResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}
func handleDownFactorInfoFcResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleDownFactorInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleGetEepromToBinResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0

}
func handleSetEepromFromBinResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0

}

func handleDownPrnFmtResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

type ScaleBasicInfo struct {
	ForcedShutdownCnt int `json:"ForcedShutdownCnt"`
	PowerOnCnt        int `json:"PowerOnCnt"`
	RunningTime       int `json:"RunningTime"`
	CaliCnt           int `json:"CaliCnt"`
	CalSwitchCnt      int `json:"CalSwitchCnt"`
	Err4Cnt           int `json:"Err4Cnt"`
	Err19Cnt          int `json:"Err19Cnt"`
	OlTime            int `json:"OlTime"`
	UlTime            int `json:"UlTime"`
	WgtCnt            int `json:"WgtCnt"`
}

func handleGetBasicDataResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	msg.MsgType = m.GET_BASIC_DATA_RESP
	msg.ScaleId = scaleId
	msg.MsgBody = "fail"

	if len(data) < 36 {
		return msg, len(data)
	}
	if len(data)%4 != 0 {
		return msg, len(data)
	}
	basicInfo := ScaleBasicInfo{
		PowerOnCnt:        0,
		RunningTime:       0,
		ForcedShutdownCnt: 0,
		WgtCnt:            0,
		OlTime:            0,
		UlTime:            0,
		CalSwitchCnt:      0,
		Err4Cnt:           0,
		Err19Cnt:          0,
		CaliCnt:           0,
	}
	start := 0
	for i := 0; i < len(data)/4; i++ {
		switch i {
		case 0:
			basicInfo.PowerOnCnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 1:
			basicInfo.RunningTime = int(binary.BigEndian.Uint32(data[start:start+4])) * 10
		case 2:
			basicInfo.ForcedShutdownCnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 3:
			basicInfo.WgtCnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 4:
			basicInfo.OlTime = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 5:
			basicInfo.UlTime = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 6:
			basicInfo.CalSwitchCnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 7:
			basicInfo.Err4Cnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 8:
			basicInfo.Err19Cnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		case 9:
			basicInfo.CaliCnt = int(binary.BigEndian.Uint32(data[start : start+4]))
		}
		start += 4

	}

	msg.MsgBody, _ = json.MarshalToString(basicInfo)

	return msg, len(data)
}

func handleDownPluResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleDownFirmwareWifiResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

// func handleGetAllEepromInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
// 	// TODO: Implement function
// 	return ScaleRespMsg{}, 0
// }

func handleDelPluResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DEL_PLU_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DEL_PLU_RESP, MsgBody: "fail"}, len(data)
	}
	// TODO: Implement function
}

func handleSetLimitResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_LIMIT_TO_SCALE_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_LIMIT_TO_SCALE_RESP, MsgBody: "fail"}, len(data)
	}
	// TODO: Implement function
}

func handleSwitchLimitResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SWITCH_LIMIT_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SWITCH_LIMIT_RESP, MsgBody: "fail"}, len(data)
	}
	// TODO: Implement function
}

func handleOpenBillSendResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.OPEN_BILL_SEND_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.OPEN_BILL_SEND_RESP, MsgBody: "fail"}, len(data)
	}
	// TODO: Implement function
}

// func handleEnUserContResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
// 	if data[0] == 0x06 {
// 		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.OPEN_BILL_SEND_RESP, MsgBody: "ok"}, len(data)
// 	} else {
// 		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.OPEN_BILL_SEND_RESP, MsgBody: "fail"}, len(data)
// 	}
// 	// TODO: Implement function
// }

type DetailHeadData struct {
	SettleAccountTimes string
	TotalCount         string
	PayPrice           string
	TotalPrice         string
	TaxKind            string
}

func parseDetailHeadData(str string) DetailHeadData {
	result := DetailHeadData{}
	parts := splitData(str)
	for _, part := range parts {
		keyValue := splitKeyValue(part)
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			switch key {
			case "settle_account_times":
				result.SettleAccountTimes = value
			case "total_count":
				result.TotalCount = value
			case "pay_price":
				result.PayPrice = value
			case "total_price":
				result.TotalPrice = value
			case "tax_kind":
				result.TaxKind = value
			}
		}
	}
	return result
}

func splitData(str string) []string {
	return splitByDelimiter(str, ',')
}

func splitKeyValue(str string) []string {
	return splitByDelimiter(str, ':')
}

func splitByDelimiter(str string, delimiter byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(str); i++ {
		if str[i] == delimiter {
			result = append(result, str[start:i])
			start = i + 1
		}
	}
	result = append(result, str[start:])
	return result
}

func handleRevDetailHeadResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	detailHeadData := parseDetailHeadData(string(data))
	println(string(data))

	model := mSrvMgr.scales[scaleId].Model
	sn := mSrvMgr.scales[scaleId].Sn
	rec := DetailTotal{}
	rec.PayPrice = detailHeadData.PayPrice
	rec.ScaleModel = model
	rec.ScaleSn = sn
	rec.SettleAccountTimes = detailHeadData.SettleAccountTimes
	rec.TaxKind = detailHeadData.TaxKind
	rec.TotalCount = detailHeadData.TotalCount
	rec.TotalPrice = detailHeadData.TotalPrice

	mSrvMgr.scales[scaleId].detailInfo = DetailList{}
	mSrvMgr.scales[scaleId].detailInfo.Total = rec
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_HEAD_RESP, MsgBody: "ok"}, len(data)
}

type DetailMidData struct {
	SettleAccountTimes string
	PluIndex           string
	PluNum             string
	PluTotalPrice      string
	PluUnitPrice       string
	PluTotalWeight     string
	PluTare            string
	PluQuantity        string
	PluUnit            string
	PluTaxType         string
	PluReturnFlag      string
	PluYear            string
	PluMonth           string
	PluDay             string
	PluTaxPrice        string
	PluChangeType      string
	PluName            string
}

// 单笔交易分包
type PackDetailMidData struct {
	SettleAccountTimes string
	PluIndex           string
	PackT              string
	PackS              string
	PluNum             string
	PluTotalPrice      string
	PluUnitPrice       string
	PluTotalWeight     string
	PluTare            string
	PluQuantity        string
	PluUnit            string
	PluTaxType         string
	PluReturnFlag      string
	PluYear            string
	PluMonth           string
	PluDay             string
	PluTaxPrice        string
	PluChangeType      string
	PluName            string
}

func parseDetailMidData(str string) PackDetailMidData {
	result := PackDetailMidData{}
	parts := splitData(str)
	for _, part := range parts {
		keyValue := splitKeyValue(part)
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			switch key {
			case "settle_account_times":
				result.SettleAccountTimes = value
			case "plu_index":
				result.PluIndex = value
			case "pack_t":
				result.PackT = value
			case "pack_s":
				result.PackS = value
			case "plu_num":
				result.PluNum = value
			case "plu_total_price":
				result.PluTotalPrice = value
			case "plu_unit_price":
				result.PluUnitPrice = value
			case "plu_total_weight":
				result.PluTotalWeight = value
			case "plu_tare":
				result.PluTare = value
			case "plu_quantity":
				result.PluQuantity = value
			case "plu_unit":
				result.PluUnit = value
			case "plu_tax_type":
				result.PluTaxType = value
			case "plu_return_flag":
				result.PluReturnFlag = value
			case "plu_year":
				result.PluYear = value
			case "plu_month":
				result.PluMonth = value
			case "plu_day":
				result.PluDay = value
			case "plu_tax_price":
				result.PluTaxPrice = value
			case "plu_change_type":
				result.PluChangeType = value
			case "plu_name":
				result.PluName = value

			}
		}
	}
	return result
}

func handleRevDetailMidResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	detailMidData := parseDetailMidData(string(data))
	if detailMidData.PackS == "1" {
		mSrvMgr.scales[scaleId].packDetailMid = []PackDetailMidData{}
		mSrvMgr.scales[scaleId].packDetailMid = append(mSrvMgr.scales[scaleId].packDetailMid, detailMidData)
	} else {
		if len(mSrvMgr.scales[scaleId].packDetailMid) < 1 {
			return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_MID_RESP, MsgBody: "ok"}, len(data)
		}
		if detailMidData.SettleAccountTimes != mSrvMgr.scales[scaleId].packDetailMid[0].SettleAccountTimes || detailMidData.PluIndex != mSrvMgr.scales[scaleId].packDetailMid[0].PluIndex {
			return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_MID_RESP, MsgBody: "ok"}, len(data)
		}
		mSrvMgr.scales[scaleId].packDetailMid = append(mSrvMgr.scales[scaleId].packDetailMid, detailMidData)
	}
	packLen := len(mSrvMgr.scales[scaleId].packDetailMid)
	pcakLenString := strconv.Itoa(packLen)

	if packLen < 1 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_MID_RESP, MsgBody: "ok"}, len(data)
	}
	if pcakLenString == mSrvMgr.scales[scaleId].packDetailMid[0].PackT {
		model := mSrvMgr.scales[scaleId].Model
		sn := mSrvMgr.scales[scaleId].Sn
		rec := DetailRec{}
		rec.ScaleModel = model
		rec.ScaleSn = sn
		for _, packMid := range mSrvMgr.scales[scaleId].packDetailMid {
			if packMid.SettleAccountTimes != "" {
				rec.SettleAccountTimes = packMid.SettleAccountTimes
			}
			if packMid.PluIndex != "" {
				rec.PluIndex = packMid.PluIndex
			}
			if packMid.PluNum != "" {
				rec.PluNum = packMid.PluNum
			}
			if packMid.PluTotalPrice != "" {
				rec.PluTotalPrice = packMid.PluTotalPrice
			}
			if packMid.PluUnitPrice != "" {
				rec.PluUnitPrice = packMid.PluUnitPrice
			}
			if packMid.PluTotalWeight != "" {
				rec.PluTotalWeight = packMid.PluTotalWeight
			}
			if packMid.PluTare != "" {
				rec.PluTare = packMid.PluTare
			}
			if packMid.PluQuantity != "" {
				rec.PluQuantity = packMid.PluQuantity
			}
			if packMid.PluUnit != "" {
				rec.PluUnit = packMid.PluUnit
			}
			if packMid.PluTaxType != "" {
				rec.PluTaxType = packMid.PluTaxType
			}
			if packMid.PluReturnFlag != "" {
				rec.PluReturnFlag = packMid.PluReturnFlag
			}
			if packMid.PluYear != "" {
				rec.PluYear = packMid.PluYear
			}
			if packMid.PluMonth != "" {
				rec.PluMonth = packMid.PluMonth
			}
			if packMid.PluDay != "" {
				rec.PluDay = packMid.PluDay
			}
			if packMid.PluTaxPrice != "" {
				rec.PluTaxPrice = packMid.PluTaxPrice
			}
			if packMid.PluChangeType != "" {
				rec.PluChangeType = packMid.PluChangeType
			}
			if packMid.PluName != "" {
				rec.PluName = packMid.PluName
			}
		}

		head := mSrvMgr.scales[scaleId].detailInfo.Total
		if head.SettleAccountTimes != "" && head.SettleAccountTimes == rec.SettleAccountTimes {
			mSrvMgr.scales[scaleId].detailInfo.Details = append(mSrvMgr.scales[scaleId].detailInfo.Details, rec)
		}

	}

	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_MID_RESP, MsgBody: "ok"}, len(data)
	// TODO: Implement function
}

type DetailTailData struct {
	SettleAccountTimes string
}

func parseDetailTailData(str string) DetailTailData {
	result := DetailTailData{}
	parts := splitData(str)
	for _, part := range parts {
		keyValue := splitKeyValue(part)
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			if key == "settle_account_times" {
				result.SettleAccountTimes = value
			}
		}
	}
	return result
}

func handleRevDetailTailResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	parseDetailTailData := parseDetailTailData(string(data))
	head := mSrvMgr.scales[scaleId].detailInfo.Total

	if parseDetailTailData.SettleAccountTimes != head.SettleAccountTimes {
		mSrvMgr.scales[scaleId].detailInfo = DetailList{}
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_TAIL_RESP, MsgBody: "fail"}, len(data)
	}
	mid := mSrvMgr.scales[scaleId].detailInfo.Details

	if head.TotalCount != strconv.Itoa(len(mid)) {
		mSrvMgr.scales[scaleId].detailInfo = DetailList{}
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_TAIL_RESP, MsgBody: "fail"}, len(data)
	}

	//直接失败不要数据了
	// // mSrvMgr.scaleMgr.detailPb.InsertTotalRec(head)//主服务不再写数据库
	// totalStr, _ := json.MarshalToString(head)
	// recStr := ""
	// mSrvMgr.recvScaleMgrMsgSrv <- &SrvMgrRespMsg{MsgType: SCALE_MGR_RESP_DETAIL_LIST, MsgBody: totalStr, ScaleId: scaleId}
	// for _, rec := range mid {
	// 	// mSrvMgr.scaleMgr.detailPb.InsertDetailRec(rec) //主服务不再写数据库
	// 	recStr, _ = json.MarshalToString(rec)
	// 	mSrvMgr.recvScaleMgrMsgSrv <- &SrvMgrRespMsg{MsgType: SCALE_MGR_RESP_DETAIL_LIST, MsgBody: recStr, ScaleId: scaleId}
	// }
	list := mSrvMgr.scales[scaleId].detailInfo
	recStr, _ := json.MarshalToString(list)
	mSrvMgr.recvScaleMgrMsgSrv <- &SrvMgrRespMsg{MsgType: SCALE_MGR_RESP_DETAIL_LIST, MsgBody: recStr, ScaleId: scaleId}

	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.REV_DETAIl_TAIL_RESP, MsgBody: "ok"}, len(data)
	// TODO: Implement function
}

func handleEraseInsertPluResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if data[0] == 0x06 {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.ERASE_INSERT_PLU_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.ERASE_INSERT_PLU_RESP, MsgBody: "fail"}, len(data)
	}

}

func handleInsertPluResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	str := string(data)
	if len(str) >= 12 {
		msg.MsgType = m.INSERT_PLU_ADDR_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = str[:12]
	} else if data[0] == 0x06 {

	} else {
		msg.MsgType = m.INSERT_PLU_ADDR_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleReadFlashDataResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}
	str := string(data)
	if len(str) >= 0 {
		msg.MsgType = m.READ_FLASH_DATA_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = str[:len(data)]
	} else if data[0] == 0x06 {

	} else {
		msg.MsgType = m.READ_FLASH_DATA_RESP
		msg.ScaleId = scaleId
		msg.MsgBody = "fail"
	}
	return msg, len(data)
}

func handleErrSerialResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	msg := ScaleRespMsg{}

	msg.MsgType = m.ERR_SERIAL_RESP
	msg.ScaleId = scaleId
	msg.MsgBody = ""
	return msg, 0
}
func checkData(data []byte) bool {
	for _, value := range data {
		if value == 0x5a || value == 0xa5 {
			return true
		}
	}
	return false
}

func handleScalePassthData(scaleId int64, data []byte, isHexMode bool) (ScaleRespMsg, int) {
	// find \r\n
	target := []byte{0x0d, 0x0a}

	index := bytes.LastIndex(data, target)
	if index == -1 {
		return ScaleRespMsg{}, 0
	}
	if checkData(data) {
		return ScaleRespMsg{}, index + 2

	}
	var passthStr string
	if isHexMode {
		for _, b := range data[0 : index+2] {
			passthStr += fmt.Sprintf("%02X ", b)
		}

	} else {
		passthStr = string(data[0 : index+2])
	}

	respMsg := ScaleRespMsg{MsgType: m.SCALE_PASSTH_DATA, MsgBody: passthStr, ScaleId: scaleId}

	return respMsg, index + 2
}

// var okBytes = []byte("\r\nOK\r\n")

// var okBytes = []byte("\r\n\r\nOK\r\n") //20240718  更换wifi模块

// func containsOK(response []byte) bool {
// 	println(string(response))
// 	return bytes.Contains(response, okBytes)
// }

type CWLAPResponse struct {
	NetworkType int
	SSID        string
	RSSI        int
	BSSID       string
	Channel     int
	Offset      int
	Security    int
}

func convertResponseToInfo(seqNo int, response CWLAPResponse) APInfo {
	var info APInfo

	info.SeqNo = seqNo
	info.Ssid = response.SSID
	info.Rssi = getRssiLevel(response.RSSI)
	info.Mac = response.BSSID
	info.EncryptType = getEncryptType(response.Security)

	return info
}

func getRssiLevel(rssi int) int {
	signalQuality := dBmtoPercentage(rssi)
	switch {
	case signalQuality < 25:
		return 1
	case signalQuality >= 25 && signalQuality < 50:
		return 2
	case signalQuality >= 50 && signalQuality < 75:
		return 3
	case signalQuality >= 75 && signalQuality <= 100:
		return 4
	default:
		return 1
	}
}

func dBmtoPercentage(rssiDbm int) int { // -50 - -100dbm
	var quality int
	if rssiDbm <= RSSI_MIN {
		quality = 0
	} else if rssiDbm >= RSSI_MAX {
		quality = 100
	} else {
		quality = 2 * (rssiDbm + 100)
	}

	return quality
} // dBmtoPercentage

func getEncryptType(security int) string {
	switch security {
	case 1:
		return "NONE"
	case 2:
		return "WEP"
	case 3:
		return "WPA"
	case 4:
		return "WPA2-PSK"
	default:
		return ""
	}
}

// +CWLAP:<ecn>,<ssid>,<rssi>,<mac>,<channel>,<freq_offset>,<freqcal_val>,<pairwise_cipher>,<group_cipher>,<bgn>,<wps> ESP32-C3
// <ecn>：加密方式
// <ssid>：字符串参数，AP 的 SSID
// <rssi>：信号强度
// <mac>：字符串参数，AP 的 MAC 地址
// <channel>：信道号
// <scan_type>：Wi-Fi 扫描类型，默认值为：0
// <scan_time_min>：每个信道最短扫描时间，单位：毫秒，范围：[0,1500]，如果扫描类型为被动扫描，本参数无效
// <scan_time_max>：每个信道最长扫描时间，单位：毫秒，范围：[0,1500]，如果设为 0，固件采用参数默认值，主动扫描为 120 ms，被动扫描为 360 ms
// <freq_offset>：频偏（保留项目）
// <freqcal_val>：频率校准值（保留项目）
// <pairwise_cipher>：成对加密类型
// <group_cipher>：组加密类型，与 <pairwise_cipher> 参数的枚举值相同
// <bgn>：802.11 b/g/n，若 bit 设为 1，则表示使能对应模式，若设为 0，则表示禁用对应模式
// <wps>：wps flag
// +CWLAP:<ecn>, <ssid>, <rssi>, <mac>, <ch>, <freq offset>, <freq calibration>  ESP8266

//"LDM",-86,"c4:c0:63:9c:86:f0"   DPM 机种设置了三个参数

func parseCWLAPResponse(data []byte) []CWLAPResponse {

	response := bytes.NewBuffer(data).String()
	lines := strings.Split(response, "\n")
	regex := regexp.MustCompile(`\+CWLAP:\(([^)]+)\)`)

	var results []CWLAPResponse

	for _, line := range lines {
		match := regex.FindStringSubmatch(line)
		if len(match) > 0 {
			fields := strings.Split(match[1], ",")
			if len(fields) == 7 {
				networkType, _ := strconv.Atoi(fields[0])
				ssid := strings.Trim(fields[1], "\"")
				rssi, _ := strconv.Atoi(fields[2])
				bssid := strings.Trim(fields[3], "\"")
				channel, _ := strconv.Atoi(fields[4])
				offset, _ := strconv.Atoi(fields[5])
				security, _ := strconv.Atoi(fields[6])

				result := CWLAPResponse{
					NetworkType: networkType,
					SSID:        ssid,
					RSSI:        rssi,
					BSSID:       bssid,
					Channel:     channel,
					Offset:      offset,
					Security:    security,
				}

				results = append(results, result)
			} else if len(fields) == 11 { //ESP32-C3
				networkType, _ := strconv.Atoi(fields[0])
				ssid := strings.Trim(fields[1], "\"")
				rssi, _ := strconv.Atoi(fields[2])
				bssid := strings.Trim(fields[3], "\"")
				channel, _ := strconv.Atoi(fields[4])
				offset, _ := strconv.Atoi(fields[8])
				security, _ := strconv.Atoi(fields[9])

				result := CWLAPResponse{
					NetworkType: networkType,
					SSID:        ssid,
					RSSI:        rssi,
					BSSID:       bssid,
					Channel:     channel,
					Offset:      offset,
					Security:    security,
				}

				results = append(results, result)
			} else if len(fields) == 3 {
				networkType := 0
				ssid := strings.Trim(fields[0], "\"")
				rssi, _ := strconv.Atoi(fields[1])
				bssid := strings.Trim(fields[2], "\"")
				channel := 0
				offset := 0
				security := 0

				result := CWLAPResponse{
					NetworkType: networkType,
					SSID:        ssid,
					RSSI:        rssi,
					BSSID:       bssid,
					Channel:     channel,
					Offset:      offset,
					Security:    security,
				}
				results = append(results, result)

			}
		}
	}

	return results
}

func convertResponsesToInfos(responses []CWLAPResponse) []APInfo {
	infos := make([]APInfo, len(responses))

	for i, response := range responses {
		infos[i] = convertResponseToInfo(i, response)
	}

	return infos
}

func handleWifiPassthResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	switch GExpectWifiResp {
	case m.GET_AP_LIST_RESP:
		return handleGetApListResp(scaleId, data)
	case m.CONNECT_AP_RESP:
		return handleConnectApResp(scaleId, data)
	case m.CONNECT_AP_ONE_KEY_RESP:
		return handleConnectApOneKeyResp(scaleId, data)
	case m.SET_WIFI_DYNAMIC_IP_RESP:
		return handleSetWifiDynamicIpResp(scaleId, data)
	case m.SEND_DATA_TO_WIFI_RESP:
		return handleSendDataToWifiResp(scaleId, data)
	case m.GET_WIFI_AP_INFO_RESP:
		return handleGetApInfoResp(scaleId, data)
	case m.GET_IP_INFO_RESP:
		return handleGetIpInfoResp(scaleId, data)
	case m.GET_IP_MODE_RESP:
		return handleGetIpModeResp(scaleId, data)
	case m.SET_WIFI_STATIC_IP_RESP:
		return handleSetWifiStaticIpResp(scaleId, data)
	case m.CHANGE_WIFI_MODE_RESP:
		return handleChangeWifiModeResp(scaleId, data)
	case m.GET_AT_VERSION_RESP:
		return handleGetWifiAtVersionResp(scaleId, data)
	case m.GET_AT_MODE_RESP:
		return handleGetWifiAtModeResp(scaleId, data)

	case m.CLOSE_SERVER_CMD_RESP:
		return handleCloseServerCmdResp(scaleId, data)
	case m.DIS_BT_CMD_RESP:
		return handleDisBtCmdResp(scaleId, data)
	case m.EN_AUTO_CONN_CMD_RESP:
		return handleEnAutoConnCmdResp(scaleId, data)
	case m.SET_WIFI_STATION_MODE_CMD_RESP:
		return handleSetWifiStationModeCmdResp(scaleId, data)
	case m.SET_MULTI_CONN_CMD_RESP:
		return handleSetMultiConnCmdResp(scaleId, data)
	case m.DIS_RECONN_CMD_RESP:
		return handleDisReconnCmdResp(scaleId, data)
	case m.DIS_IP_PORT_INFO_CMD_RESP:
		return handleDisIpPortInfoCmdResp(scaleId, data)
	case m.SET_SINGLE_CONN_CMD_RESP:
		return handleSetSingleConnCmdResp(scaleId, data)
	case m.SET_TCP_SERVER_CMD_RESP:
		return handleSetTcpServerCmdResp(scaleId, data)
	case m.SET_TIME_OUT_CMD_RESP:
		return handleSetTimeOutCmdResp(scaleId, data)
	case m.SET_PASSTH_MODE_CMD_RESP:
		return handleSetPassthModeCmdResp(scaleId, data)
	case m.SET_SCAN_AP_PARAM_CMD_RESP:
		return handleSetScanApParamCmdResp(scaleId, data)

	default:
		return ScaleRespMsg{}, 0
	}
}

func handleCloseServerCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPSERVER=0") {
			return ScaleRespMsg{m.CLOSE_SERVER_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.CLOSE_SERVER_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CLOSE_SERVER_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleDisBtCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+BLEINIT=0") {
			return ScaleRespMsg{m.DIS_BT_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.DIS_BT_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_BT_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleEnAutoConnCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CWAUTOCONN=1") {
			return ScaleRespMsg{m.EN_AUTO_CONN_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.EN_AUTO_CONN_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.EN_AUTO_CONN_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetWifiStationModeCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CWMODE=1,1") {
			return ScaleRespMsg{m.SET_WIFI_STATION_MODE_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_WIFI_STATION_MODE_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIFI_STATION_MODE_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetMultiConnCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPMUX=1") {
			return ScaleRespMsg{m.SET_MULTI_CONN_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_MULTI_CONN_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_MULTI_CONN_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleDisReconnCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CWRECONNCFG=20,0") {
			return ScaleRespMsg{m.DIS_RECONN_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.DIS_RECONN_CMD_RESP, "ok", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_RECONN_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleDisIpPortInfoCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPDINFO=0") {
			return ScaleRespMsg{m.DIS_IP_PORT_INFO_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.DIS_IP_PORT_INFO_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.DIS_IP_PORT_INFO_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetSingleConnCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPSERVERMAXCONN=1") {
			return ScaleRespMsg{m.SET_SINGLE_CONN_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_SINGLE_CONN_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_SINGLE_CONN_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetTcpServerCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPSERVER=") {
			return ScaleRespMsg{m.SET_TCP_SERVER_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_TCP_SERVER_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_TCP_SERVER_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}
func handleSetTimeOutCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPSTO=0") {
			return ScaleRespMsg{m.SET_TIME_OUT_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_TIME_OUT_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_TIME_OUT_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetPassthModeCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CIPMODE=0") {
			return ScaleRespMsg{m.SET_PASSTH_MODE_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_PASSTH_MODE_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_PASSTH_MODE_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetScanApParamCmdResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "AT+CWLAPOPT") {
			return ScaleRespMsg{m.SET_SCAN_AP_PARAM_CMD_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.SET_SCAN_AP_PARAM_CMD_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_SCAN_AP_PARAM_CMD_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleGetWifiAtModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_MODE_OK_RESP) { // success
		if strings.Contains(string(data), "+CWMODE:1") {
			return ScaleRespMsg{m.GET_AT_MODE_RESP, "ok", scaleId}, len(data)
		} else {
			return ScaleRespMsg{m.GET_AT_MODE_RESP, "fail", scaleId}, len(data)

		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_AT_MODE_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleGetWifiAtVersionResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AT_VERSION_OK_RESP) { // success
		if strings.Contains(string(data), m.AT_VERSION) {
			return ScaleRespMsg{m.GET_AT_VERSION_RESP, m.AT_VERSION, scaleId}, len(data)
		} else {
			if strings.Contains(string(data), m.AT_VERSION8266) {
				return ScaleRespMsg{m.GET_AT_VERSION_RESP, "8266", scaleId}, len(data)
			} else {
				return ScaleRespMsg{m.GET_AT_VERSION_RESP, m.AT_VERSION, scaleId}, len(data)
			}
		}
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_AT_VERSION_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleBTPassthResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	switch GExpectBTResp {
	case m.MODIFY_BT_NAME_RESP:
		return handleModifyBtNameResp(scaleId, data)
	case m.BT_PASSTH_DATA_RESP:
		return handleSendDataToBTResp(scaleId, data)
	default:
		return ScaleRespMsg{}, 0
	}
}

func handleGetApListResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	hexData := fmt.Sprintf("%x", data)
	println("at get ap list resp:" + hexData)
	println("at get ap list resp:" + string(data))

	if !bytes.Contains(data, []byte("OK")) {
		if len(data) < 2000 {
			return ScaleRespMsg{}, 0
		}
	}

	apList := parseCWLAPResponse(data)
	if apList == nil {
		return ScaleRespMsg{}, 0
	}
	println("at get ap list resp:" + string(data))

	// compose response
	apInfoList := convertResponsesToInfos(apList)

	jsonData, err := json.MarshalToString(apInfoList)
	if err != nil {
		fmt.Println("Error:", err)
		return ScaleRespMsg{}, 0
	}
	return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_AP_LIST_RESP, MsgBody: jsonData}, len(data)
}

func handleConnectApResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	println("at connect resp:" + string(data))
	if strings.Contains(string(data), CONNECT_AP_OK_RESP) { // success
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CONNECT_AP_RESP, MsgBody: "ok"}, len(data)
	} else if strings.Contains(string(data), CONNECT_AP_FAIL_RESP) { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CONNECT_AP_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, len(data)
	}
}

func handleConnectApOneKeyResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), CONNECT_AP_OK_RESP) { // success
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CONNECT_AP_ONE_KEY_RESP, MsgBody: "ok"}, len(data)
	} else if strings.Contains(string(data), CONNECT_AP_FAIL_RESP) { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CONNECT_AP_ONE_KEY_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, len(data)
	}
}

func handleChangeWifiModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), CHANG_WIFI_MODE_OK_RESP) { // success
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CHANGE_WIFI_MODE_RESP, MsgBody: "ok"}, len(data)
	} else if strings.Contains(string(data), CONNECT_AP_FAIL_RESP) { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.CHANGE_WIFI_MODE_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, len(data)
	}
}

func handleRescanApListResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// TODO: Implement function
	return ScaleRespMsg{}, 0
}

func handleSetWifiDynamicIpResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), SET_WIFI_DYNAMIC_IP_OK_RESP) { // success
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIFI_DYNAMIC_IP_RESP, MsgBody: "ok"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleSetWifiStaticIpResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), SET_WIFI_STATIC_IP_OK_RESP) { // success
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.SET_WIFI_STATIC_IP_RESP, MsgBody: "ok"}, len(data)
	} else {
		return ScaleRespMsg{}, 0
	}
}

type IPInfo struct {
	IP      string
	Gateway string
	Netmask string
}

type WifiAPInfo struct {
	Ssid    string
	Bssid   string
	Channel string
	Rssi    int
}

func extractWifiAPInfo(response string) (WifiAPInfo, error) {
	var info WifiAPInfo

	// 	+CWJAP_DEF:<ssid>, <bssid>, <channel>, <rssi>
	// OK
	// split response string into multiple lines

	// iterates on each lines to extract ssid, bssid, channel, rssi
	if strings.Contains(response, "CWJAP_DEF") {
		lines := strings.Split(response, "\r\n")
		for _, line := range lines {
			line = strings.Trim(line, "\t")
			line = strings.Replace(line, `\"`, "", -1)
			line = strings.Replace(line, `"`, "", -1)
			if strings.HasPrefix(line, "+CWJAP_DEF:") {
				data := line[len("+CWJAP_DEF:"):]
				dataSplit := strings.Split(data, ",")
				if len(dataSplit) < 4 {
					continue
				}
				info.Ssid = dataSplit[0]
				info.Bssid = dataSplit[1]
				info.Channel = dataSplit[2]
				level, _ := strconv.ParseInt(dataSplit[3], 10, 64)
				info.Rssi = getRssiLevel(int(level))
			}
		}

	} else {
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			line = strings.Trim(line, "\t")
			line = strings.Replace(line, `\"`, "", -1)
			line = strings.Replace(line, `"`, "", -1)
			if strings.HasPrefix(line, "+CWJAP:") {
				data := line[len("+CWJAP:"):]
				dataSplit := strings.Split(data, ",")
				if len(dataSplit) < 4 {
					continue
				}
				info.Ssid = dataSplit[0]
				info.Bssid = dataSplit[1]
				info.Channel = dataSplit[2]
				level, _ := strconv.ParseInt(dataSplit[3], 10, 64)
				info.Rssi = getRssiLevel(int(level))
			}
		}
	}

	return info, nil
}

func extractIPInfo(response string) (IPInfo, error) {
	var info IPInfo

	// 根据字符串中的换行符分割字符串
	lines := strings.Split(response, "\r\n")

	//遍历每一行字符串，提取 IP、网关和子网掩码的值
	if strings.Contains(response, "+CIPSTA_CUR:ip:") {
		for _, line := range lines {
			if strings.HasPrefix(line, "+CIPSTA_CUR:ip:") {
				info.IP = strings.Trim(line[len("+CIPSTA_CUR:ip:\""):], "\"")
			} else if strings.HasPrefix(line, "+CIPSTA_CUR:gateway:") {
				info.Gateway = strings.Trim(line[len("+CIPSTA_CUR:gateway:\""):], "\"")
			} else if strings.HasPrefix(line, "+CIPSTA_CUR:netmask:") {
				info.Netmask = strings.Trim(line[len("+CIPSTA_CUR:netmask:\""):], "\"")
			}
		}

	} else if strings.Contains(response, "+CIPSTA:ip:") {
		//ESP_32
		for _, line := range lines {
			if strings.HasPrefix(line, "+CIPSTA:ip:") {
				info.IP = strings.Trim(line[len("+CIPSTA:ip:\""):], "\"")
			} else if strings.HasPrefix(line, "+CIPSTA:gateway:") {
				info.Gateway = strings.Trim(line[len("+CIPSTA:gateway:\""):], "\"")
			} else if strings.HasPrefix(line, "+CIPSTA:netmask:") {
				info.Netmask = strings.Trim(line[len("+CIPSTA:netmask:\""):], "\"")
			}
		}
	}

	// 检查是否成功提取了所有值
	if info.IP == "" || info.Gateway == "" || info.Netmask == "" {
		return info, fmt.Errorf("failed to extract IP information")
	}

	return info, nil
}

func extractIPMode(response string) (bool, error) {
	var mode bool
	var err error = nil

	// 根据字符串中的换行符分割字符串
	lines := strings.Split(response, "\r\n")

	// 遍历每一行字符串，提取 IP mode
	if strings.Contains(response, "CWDHCP_CUR") {
		for _, line := range lines {
			if strings.HasPrefix(line, "+CWDHCP_CUR:") {
				modeNo := line[len("+CWDHCP_CUR:"):]
				switch modeNo {
				case "2", "3":
					mode = true
				case "0", "1":
					mode = false
				default:
					mode = false
					err = fmt.Errorf("invalid response")
				}
				break
			}
		}

	} else {
		for _, line := range lines {
			if strings.HasPrefix(line, "+CWDHCP:") {
				modeNo := line[len("+CWDHCP:"):]

				switch modeNo {
				case "3":
					mode = true
				case "0", "1", "2":
					mode = false
				default:
					mode = false
					err = fmt.Errorf("invalid response")
				}

				break
			}
		}

	}

	return mode, err
}

func handleGetApInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_AP_INFO_OK_RESP) { // success
		println("AP:" + string(data))
		apInfo, err := extractWifiAPInfo(string(data))
		if err != nil {
			return ScaleRespMsg{}, len(data)
		}
		apInfoStr, _ := json.MarshalToString(apInfo)
		return ScaleRespMsg{m.GET_WIFI_AP_INFO_RESP, apInfoStr, scaleId}, len(data)
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_WIFI_AP_INFO_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleGetIpInfoResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	println("IP:" + string(data))
	if strings.Contains(string(data), GET_IP_INFO_OK_RESP) { // success
		ipInfo, err := extractIPInfo(string(data))
		if err != nil {
			return ScaleRespMsg{m.GET_IP_INFO_RESP, "fail", scaleId}, len(data)
		}
		ipInfoStr, _ := json.MarshalToString(ipInfo)
		return ScaleRespMsg{m.GET_IP_INFO_RESP, ipInfoStr, scaleId}, len(data)
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_IP_INFO_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleGetIpModeResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if strings.Contains(string(data), GET_IP_MODE_OK_RESP) { // success
		isDhcpEnabled, err := extractIPMode(string(data))
		if err != nil {
			return ScaleRespMsg{}, len(data)
		}
		var dhcpStr string
		if isDhcpEnabled {
			dhcpStr = "dhcp"
		} else {
			dhcpStr = "static"
		}

		return ScaleRespMsg{m.GET_IP_MODE_RESP, dhcpStr, scaleId}, len(data)
	} else if strings.Contains(string(data), "Error") { // fail
		return ScaleRespMsg{ScaleId: scaleId, MsgType: m.GET_IP_MODE_RESP, MsgBody: "fail"}, len(data)
	} else { // unkown
		return ScaleRespMsg{}, 0
	}
}

func handleModifyBtNameResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	// if bytes.Contains(data, []byte(MODIFY_BT_OK_RESP)) {
	if bytes.Contains(data, []byte("OK")) {
		return ScaleRespMsg{m.MODIFY_BT_NAME_RESP, "ok", scaleId}, len(data)
	} else if bytes.Contains(data, []byte("ERROR")) {
		return ScaleRespMsg{m.MODIFY_BT_NAME_RESP, "fail", scaleId}, len(data)
	} else {
		return ScaleRespMsg{}, 0
		// return ScaleRespMsg{m.MODIFY_BT_NAME_RESP, "fail", scaleId}, len(data)
	}
}

func handleSendDataToBTResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	if bytes.Contains(data, []byte("OK")) {
		return ScaleRespMsg{m.SEND_DATA_TO_BT_RESP, string(data), scaleId}, len(data)
	} else if bytes.Contains(data, []byte("ERROR")) {
		return ScaleRespMsg{m.SEND_DATA_TO_BT_RESP, "fail", scaleId}, len(data)
	} else {
		return ScaleRespMsg{}, 0
	}

}

func handleSendDataToWifiResp(scaleId int64, data []byte) (ScaleRespMsg, int) {
	return ScaleRespMsg{m.SEND_DATA_TO_WIFI_RESP, string(data), scaleId}, len(data)
}
