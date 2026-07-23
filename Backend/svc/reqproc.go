package svc

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	m "tmaxsrv/comm"

	l "tmaxsrv/log"
	"tmaxsrv/picker"
	utils "tmaxsrv/util"
)

type (
	reqProcFun func(scale *Scale, req SRequest) (*ScaleRespMsg, error)
	IpMode     int
)

const (
	IPMODE_NONE IpMode = iota
	IPMODE_STATIC
	IPMODE_DYNAMIC
)

var handlers map[SReqType]reqProcFun

func procToScaleReq(s *Scale, req SRequest) {
	var resp *ScaleRespMsg
	var err error

	handler, ok := handlers[req.Req]
	if !ok {
		resp = &ScaleRespMsg{MsgType: GetResVsResp(req.Req), MsgBody: fmt.Sprintf("unknown req: %s", req.Req), ScaleId: s.Id}
	} else {
		resp, err = handler(s, req)
		if err != nil {
			resp = &ScaleRespMsg{MsgType: GetResVsResp(req.Req), MsgBody: fmt.Sprintf("error: %s", err.Error()), ScaleId: s.Id}
		}
	}
	// send msg to web socket client
	if resp == nil {
		return
	}
	result, _ := json.Marshal(resp)

	if s.client != nil {
		s.client.sendCh <- result

	}
}

var conversionMap map[SReqType]m.RespMsgType

// func procGetRecs(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
// 	recs, _ := scale.GetRecs(req.ReqData)
// 	recsStr, _ := json.MarshalToString(recs)
// 	resp := &ScaleRespMsg{MsgType: m.GET_RECS_RESP, MsgBody: recsStr, ScaleId: scale.Id}
// 	result, _ := json.Marshal(resp)
// 	scale.client.sendCh <- result
// 	return nil, nil
// }

func procGetRecs(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	// 获取记录
	recs, err := scale.GetRecs(req.ReqData)
	if err != nil {
		return nil,
			err
	}

	// 检查记录数量，如果为 0 则返回包含空列表的响应
	if len(recs) == 0 {
		emptyRecsStr, err := json.MarshalToString([]interface{}{})
		if err != nil {
			return nil, err
		}
		resp := &ScaleRespMsg{
			MsgType: m.GET_RECS_RESP,
			MsgBody: emptyRecsStr,
			ScaleId: scale.Id,
		}
		return resp, nil
	}

	batchSize := 1000
	// 计算记录的总数
	recsCount := len(recs)
	// 逐批处理记录
	for i := 0; i < recsCount; i += batchSize {
		end := i + batchSize
		if end > recsCount {
			end = recsCount
		}
		// 获取当前批次的记录
		batch := recs[i:end]
		// 将当前批次的记录序列化为 JSON 字符串
		recsStr, err := json.MarshalToString(batch)
		if err != nil {
			return nil, err
		}
		// 创建响应消息
		resp := &ScaleRespMsg{
			MsgType: m.GET_RECS_RESP,
			MsgBody: recsStr,
			ScaleId: scale.Id,
		}
		result, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		scale.client.sendCh <- result
	}

	return nil, nil
}

func procSendScaleAlive(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	if scale.ScaleCat != m.SCALE_TMAX {
		return nil, nil
	}
	sendRespMsgScale(scale)
	return &ScaleRespMsg{m.ANSWER_ALIVE_RESP, "ok", scale.Id}, nil
}

// 导出csv时用来写表头
func procExportRecs(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	// 获取记录
	dataReq := strings.Split(req.ReqData, ",")
	path := ""

	if len(dataReq) == 5 {

		path = dataReq[4]
	}
	file, err := os.Create(path)
	if err != nil {
		return &ScaleRespMsg{m.EXPORT_RECS_RESP, "Can not creat file! error!", scale.Id}, nil
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 获取结构体字段名并反向转换为原始字段作为表头
	headers := []string{
		"Id",
		"ScaleModel",
		"ScaleSn",
		"PLU",
		"Product Code",
		"Item Code",
		"Category",
		"PLU Name",
		"GeneralUnit",
		"TaxType",
		"Price",
		"UnitWeight",
		"Pretare",
		"LimitHigh",
		"LimitLow",
		"Weight",
		"Weight Unit",
		// "User NO.",
		"User Name",
		"Scale Name",
		"Date Time",
	}

	if err := writer.Write(headers); err != nil {
		return &ScaleRespMsg{m.EXPORT_RECS_RESP, "Failed to wirte headers!", scale.Id}, err
	}

	// 写入数据行
	// 分批写入数据
	limit := 100000 // 每批获取的记录数量
	offset := 0
	for {
		recs, err := scale.GetWgtRecs(req.ReqData, offset, limit)
		if err != nil {
			return &ScaleRespMsg{m.EXPORT_RECS_RESP, "get data error!", scale.Id}, err
		}

		if len(recs) == 0 {
			break // 没有更多记录，退出循环
		}

		// 写入当前批次的数据行
		for _, rec := range recs {
			record := []string{
				rec.Id,
				rec.ScaleModel,
				rec.ScaleSn,
				rec.Plu,
				rec.ProductCode,
				rec.ItemCode,
				rec.Category,
				rec.ProductName,
				rec.GeneralUnit,
				rec.TaxType,
				rec.Price,
				rec.UnitWeight,
				rec.Pretare,
				rec.LimitHigh,
				rec.LimitLow,
				rec.Weight,
				rec.WeightUnit,
				// rec.UserNo,
				rec.UserName,
				rec.ScaleName,
				rec.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			if err := writer.Write(record); err != nil {
				return &ScaleRespMsg{m.EXPORT_RECS_RESP, "Write to file failed!", scale.Id}, err
			}
		}

		offset += limit // 移动到下一批记录
	}

	return &ScaleRespMsg{m.EXPORT_RECS_RESP, path + "  Export ok !", scale.Id}, nil

}

func procAddRec(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	var rec ScaleRec
	if err := json.Unmarshal([]byte(req.ReqData), &rec); err != nil {
		l.Log.Errorf("Unmarshal ScaleRec error: %v", err)
		resp := &ScaleRespMsg{MsgType: m.GET_RECS_RESP, MsgBody: err.Error(), ScaleId: scale.Id}
		result, _ := json.Marshal(resp)
		scale.client.sendCh <- result
	} else {
		scale.AddRec(rec)
		resp := &ScaleRespMsg{MsgType: m.ADD_REC_RESP, MsgBody: "ok", ScaleId: scale.Id} //@FLF20231027
		result, _ := json.Marshal(resp)
		scale.client.sendCh <- result
	}
	return nil, nil
}

func procDelRec(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	var id uint64
	var scaleMode uint64
	var err error

	parts := strings.Split(req.ReqData, ",")
	if id, err = strconv.ParseUint(parts[0], 10, 64); err != nil {
		resp := &ScaleRespMsg{MsgType: m.DEL_REC_RESP, MsgBody: err.Error(), ScaleId: scale.Id}
		result, _ := json.Marshal(resp)
		scale.client.sendCh <- result
	} else if scaleMode, err = strconv.ParseUint(parts[1], 10, 64); err != nil {
		resp := &ScaleRespMsg{MsgType: m.DEL_REC_RESP, MsgBody: err.Error(), ScaleId: scale.Id}
		result, _ := json.Marshal(resp)
		scale.client.sendCh <- result
	} else {
		_ = scale.DelRec(uint(id), uint(scaleMode), parts[2], parts[3])
		resp := &ScaleRespMsg{MsgType: m.DEL_REC_RESP, MsgBody: "", ScaleId: scale.Id}
		result, _ := json.Marshal(resp)
		scale.client.sendCh <- result
	}

	// if id, err = strconv.ParseUint(req.ReqData, 10, 64); err != nil {
	// 	log.Log.Errorf(err.Error())
	// 	resp := &ScaleRespMsg{MsgType: m.DEL_REC_RESP, MsgBody: err.Error(), ScaleId: scale.Id}
	// 	result, _ := json.Marshal(resp)
	// 	scale.client.sendCh <- result
	// }
	//  else {
	// 	_ = scale.DelRec(uint(id))
	// 	resp := &ScaleRespMsg{MsgType: m.DEL_REC_RESP, MsgBody: "", ScaleId: scale.Id}
	// 	result, _ := json.Marshal(resp)
	// 	scale.client.sendCh <- result
	// }
	return nil, nil
}

func procDownPrnFmt(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownPrnFmt(scale, req)
}

func procSetMaxRange1(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetMaxRange1(scale, req)
}
func procSetMaxRange2(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetMaxRange2(scale, req)
}
func procGetMaxRange1(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetMaxRange1(scale, req)
}
func procGetMaxRange2(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetMaxRange2(scale, req)
}

func procSetDecimalValue(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetDecimalValue(scale, req)
}

func procSetGaduation1Value(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetGaduation1Value(scale, req)
}

func procSetGaduation2Value(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetGaduation2Value(scale, req)
}

func procGetGaduation1Value(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetGaduation1Value(scale, req)
}

func procGetGaduation2Value(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetGaduation2Value(scale, req)
}

func procGetDecimalValue(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetDecimalValue(scale, req)
}

func procSetSerialPort(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetSerialPort(scale, req)
}

func procGetSerialPort(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetSerialPort(scale, req)
}

func procGetWeightUnit(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetWeightUnit(scale, req)
}

func procSetWeightUnit(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetWeightUnit(scale, req)
}

func procSetInitialZero(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetInitialZero(scale, req)
}

func procGetInitialZero(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetInitialZero(scale, req)
}

func procSetManualZero(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetManualZero(scale, req)
}

func procGetManualZero(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetManualZero(scale, req)
}

func procSetZeroTracking(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetZeroTracking(scale, req)
}

func procGetZeroTracking(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetZeroTracking(scale, req)
}

func procSetGravAcc(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetGravAcc(scale, req)
}

func procGetGravAcc(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetGravAcc(scale, req)
}

func procSetForceUnTare(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetForceUnTare(scale, req)
}

func procGetSealStatus(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetSealStatus(scale, req)
}

// 获取型号
func procGetModel(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetModel(scale, req)
}

// 打开连续发送内码
func procEnCode(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqEnCode(scale, req)
}

// 关闭连续发送内码
func procDisCode(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDisCode(scale, req)
}

// 查询ROM版本号
func procAskRomVersion(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqAskRomVersion(scale, req)
}

func procSoftSeal(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSoftSeal(scale, req)
}

func procRemoveSoftSeal(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqRemoveSoftSeal(scale, req)
}

func procRemoveSoftSealOnce(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqRemoveSoftSealOnce(scale, req)
}

func procGetWiredIp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetWiredIp(scale, req)
}

func procSetWiredIp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetWiredIp(scale, req)
}
func procSetWiredDhcp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetWiredDhcp(scale, req)
}
func procGetWiredDhcp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetWiredDhcp(scale, req)
}

func procInitWifi(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqInitWifi(scale, req)
}

func procCalWeight(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqCalWeight(scale, req)
}

func procSendCalHeart(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSendCalHeart(scale, req)
}

func procDownDefaultPrnFmt(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownDefaultPrnFmt(scale, req)
}

func procBackupDefSetting(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqBackupDefSetting(scale, req)

}

func procDownPlu(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownPlu(scale, req)
}

func procDownFirmwareWifi(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownFirmware(scale, req)
}

func procGetBasicData(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetBasicData(scale)
}

func procDisPassthMode(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDisWifiPassthrough(scale)
}
func procCloseSerialPort(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqCloseSerialPort(scale)
}
func procOpenSerialPort(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqOpenSerialPort(scale)
}

func procSetLimitToScale(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetLimitToScale(scale, req)
}

func procOpenBillSend(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqOpenBillSend(scale)
}

func procUpdateFirmware(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqUpdateFirmware(scale, req)
}

func procDelPlu(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDelPlu(scale, req)
}

func procInsertPlu(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqInsertPlu(scale, req)
}

func procGetOneEepromInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetOneEepromInfo(scale, req)
}

func procGetAllEepromInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetAllEepromInfo(scale)
}

func ProcSetOutputFmt(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetOutputFmt(scale, req)
}

func procGetApList(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetApList(scale)
}

// func procGetWeightErr(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
// 	return ReqGetWeightErr(scale)
// }

func procRescanAp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return &ScaleRespMsg{ScaleId: scale.Id, MsgType: m.RESCAN_AP_LIST_RESP, MsgBody: "ok"}, nil
}

func procConnectAp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	data := utils.JsonToMap(req.ReqData)
	return ReqConnectAp(scale, data["ssid"].(string), data["bssid"].(string), data["password"].(string))
}

func procConnectApOneKey(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	data := utils.JsonToMap(req.ReqData)
	return ReqConnectApOneKey(scale, data["ssid"].(string), data["bssid"].(string), data["password"].(string))
}

func procSetWifiDynamicIp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetWifiDynamicIp(scale)
}

func procSetWifiStaticIp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	data := utils.JsonToMap(req.ReqData)
	return ReqSetWifiStaticIp(scale, data["ip"].(string), data["gateway"].(string), data["netmask"].(string))
}

func procGetIpInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetIpInfo(scale)
}

func procChangeWifiMode(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqChangeWifiMode(scale, req)
}

func procModifyEepromInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqModifyEepromInfo(scale, req)
}

func procDownEepromInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownEepromInfo(scale, req)
}

func procGetEepromToBin(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetEepromInfoToBin(scale, req)
}

func procSetEepromFromBin(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetEepromFromBin(scale, req)
}

func procSetEepromFromBinFc(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetEepromFromBinFc(scale, req)
}

func procDownFactoryInfoFc(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownFactoryInfoFc(scale, req)
}

func procDownFactoryInfoTmax(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqDownFactoryInfoTmax(scale, req)
}

func procModifyVarValue(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqModifyVarValue(scale, req)
}

func procSetServerIp(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetServerIp(scale, req)
}

func procEnFactoryMode(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetServerIp(scale, req)
	// return ReqEnFactoryMode(scale)TODO:
}

func procModifyBTName(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqModifyBTName(scale, req.ReqData)
}

func procSendDataToBT(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSendDataToBT(scale, req.ReqData)
}

func procGetIpMode(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetIpMode(scale)
}

func procGetWifiInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetWifiApInfo(scale)
}

func procSendDataToWifi(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSendDataToWifi(scale, req.ReqData)
}

func procRegWeight(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	l.Log.Debugf("process register weight data")
	return scale.RegWeightData()
}

func procUnRegWeight(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	// return scale.UnRegWeightData()

	scale.isSendUnolicitedData = false
	if scale.isScalePassth {
		scale.isScalePassth = false
		picker := picker.GetPickerFn(scale.ScaleCat)
		// scale.MySerial.ChangePickFunc(picker)
		if scale.MySerial != nil {
			scale.MySerial.ChangePickFunc(picker)

		} else if scale.MyNet != nil {
			scale.MyNet.ChangePickFunc(picker)
		} else {
			l.Log.Errorf("MySerial and MyNet are nil, cannot change pick function")
		}
	}
	return scale.UnRegWeightData()
}

func procOpenScalePassth(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	l.Log.Debugf("process Open Scale Passth")
	msg, err := scale.OpenScalePassth()
	if err == nil && msg.MsgBody == "ok" {
		scale.isScalePassth = true
		if req.ReqData == "hex" {
			scale.IsScalePassthHex = true
		} else {
			scale.IsScalePassthHex = false
		}
		picker := picker.GetPickerFn(scale.ScaleCat + 1)

		if scale.MySerial != nil {
			scale.MySerial.ChangePickFunc(picker)

		} else if scale.MyNet != nil {
			scale.MyNet.ChangePickFunc(picker)
		} else {
			l.Log.Errorf("MySerial and MyNet are nil, cannot change pick function")
		}
	}

	return msg, err
}

func procCloseScalePassth(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	l.Log.Debugf("process Close Scale Passth")
	scale.isScalePassth = false
	picker := picker.GetPickerFn(scale.ScaleCat)
	scale.MySerial.ChangePickFunc(picker)
	return scale.CloseScalePassth()
}

func procChangeScalePassthMode(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	l.Log.Debugf("process Open Scale Passth")
	if req.ReqData == "hex" {
		s.IsScalePassthHex = true
	} else {
		s.IsScalePassthHex = false
	}
	return &ScaleRespMsg{MsgType: m.CHANGE_SCALE_PASSTH_MODE_RESP, MsgBody: "ok", ScaleId: s.Id}, nil
}

func procTare(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.PerfTare()
}

func procGetWeight(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.ReadWeight()
}

func procZero(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.PerfZero()
}

func procZeroUnstable(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.PerfZeroUnstable()
}

func procTareUnstable(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.PerfTareUnstable()
}

func GetResVsResp(reqType SReqType) m.RespMsgType {
	return conversionMap[reqType]
}

// func procUpdateFirmware(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
// 	return scale.UpdateFirmware(req.ReqData)
// }

func ProcCheckSerialPort(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.CheckSerialPort()
}

func procGetBuildInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.GetBuildInfo()
}

func procGetScaleTime(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.GetScaleTime()
}
func procSetScaleTime(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqSetScaleTime(scale, req)
}

func procGetScaleInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.GetScaleInfo()
}

func procGetFactoryInfo(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return scale.GetFactoryInfo()
}

func procGetWeighErr(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	return ReqGetWeightErr(scale)
}

func procGetUiConf(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	l.Log.Info("Got get UI Config request")
	modeInt, _ := strconv.Atoi(req.ReqData)
	modeUint := uint(modeInt)
	// TODO:需要连上之后加sn  ScaleSn
	config, _ := mSrvMgr.modeSetting.GetModeSetting(modeUint)
	configStr, _ := json.MarshalToString(config[0])
	respMsg := &ScaleRespMsg{MsgType: m.GET_UI_CONF_RESP, MsgBody: configStr, ScaleId: scale.Id}
	return respMsg, nil
}

func procUpdateUiConf(scale *Scale, req SRequest) (*ScaleRespMsg, error) {
	l.Log.Info("Got get UI Config request")
	// TODO:需要连上之后加sn  ScaleSn
	var config ModeSetting
	var respMsg *ScaleRespMsg
	if err := json.UnmarshalFromString(req.ReqData, &config); err != nil {
		l.Log.Error(err)
		respMsg = &ScaleRespMsg{MsgType: m.UPDATE_UI_CONF_RESP, MsgBody: "failed to parse update UI Config", ScaleId: scale.Id}

	} else {
		err := mSrvMgr.modeSetting.settingPb.UpdateModeSetting(config)
		if err != nil {
			l.Log.Error(err)
		}
		respMsg = &ScaleRespMsg{MsgType: m.UPDATE_UI_CONF_RESP, MsgBody: "ok", ScaleId: scale.Id}
	}
	return respMsg, nil
}

//  处理请求

func init() {
	handlers = map[SReqType]reqProcFun{
		SREQ_GET_WEIGHT:               procGetWeight,
		SREQ_ZERO:                     procZero,
		SREQ_TARE:                     procTare,
		SREQ_ZERO_UNSTABLE:            procZeroUnstable,
		SREQ_TARE_UNSTABLE:            procTareUnstable,
		SREQ_REG_WEIGHT_DATA:          procRegWeight,
		SREQ_UNREG_WEIGHT_DATA:        procUnRegWeight,
		SREQ_GET_RECS:                 procGetRecs,
		SREQ_ADD_REC:                  procAddRec,
		SREQ_DEL_REC:                  procDelRec,
		SREQ_DOWN_PRN_FMT:             procDownPrnFmt,
		SREQ_GET_AP_LIST:              procGetApList,
		SREQ_RESCAN_AP_LIST:           procRescanAp,
		SREQ_CONNECT_AP:               procConnectAp,
		SREQ_CONNECT_AP_ONE_KEY:       procConnectApOneKey,
		SREQ_SET_WIFI_DYNAMIC_IP:      procSetWifiDynamicIp,
		SREQ_SET_WIFI_STATIC_IP:       procSetWifiStaticIp,
		SREQ_GET_IP_INFO:              procGetIpInfo,
		SREQ_MODIFY_BT_NAME:           procModifyBTName,
		SREQ_SEND_DATA_TO_BT:          procSendDataToBT,
		SREQ_SEND_DATA_TO_WIFI:        procSendDataToWifi,
		SREQ_GET_IP_MODE:              procGetIpMode,
		SREQ_GET_WIFI_INFO:            procGetWifiInfo,
		SREQ_UPDATE_FIRMWARE:          procUpdateFirmware,
		SREQ_CHECK_SERIAL_PORT:        ProcCheckSerialPort,
		SREQ_GET_BUILD_INFO:           procGetBuildInfo,
		SREQ_GET_SCALE_TIME:           procGetScaleTime,
		SREQ_SET_SCALE_TIME:           procSetScaleTime,
		SREQ_GET_ONE_EEPROM_INFO:      procGetOneEepromInfo,
		SREQ_GET_ALL_EEPROM_INFO:      procGetAllEepromInfo,
		SREQ_SET_OUTPUT_FMT:           ProcSetOutputFmt,
		SREQ_OPNE_SCALE_PASSTHROUGH:   procOpenScalePassth,
		SREQ_CLOSE_SCALE_PASSTHROUGH:  procCloseScalePassth,
		SREQ_CHANGE_SCALE_PASSTH_MODE: procChangeScalePassthMode,
		SREQ_GET_SCALE_INFO:           procGetScaleInfo,
		SREQ_GET_FACTORY_INFO:         procGetFactoryInfo,
		SREQ_GET_WEIGHT_ERR:           procGetWeighErr,
		SREQ_DOWN_PLU:                 procDownPlu,
		SREQ_DEL_PLU:                  procDelPlu,
		SREQ_INSERT_PLU:               procInsertPlu,
		SREQ_GET_UI_CONF:              procGetUiConf,
		SREQ_UPDATE_UI_CONF:           procUpdateUiConf,
		SREQ_CHANGE_WIFI_MODE:         procChangeWifiMode,
		SREQ_MODIFY_EEPROM_INFO:       procModifyEepromInfo,
		SREQ_DOWN_EEPROM_INFO:         procDownEepromInfo,
		SREQ_DOWN_FACTORY_INFO_FC:     procDownFactoryInfoFc,
		SREQ_DOWN_FACTORY_INFO:        procDownFactoryInfoTmax,
		SREQ_MODIFY_VAR_VALUE:         procModifyVarValue,
		SREQ_SET_SERVER_IP:            procSetServerIp,
		SREQ_EN_FACTORY_MODE:          procEnFactoryMode,
		SREQ_DOWN_DEFAULT_PRN_FMT:     procDownDefaultPrnFmt,
		SREQ_BACKUP_DEF_SETTING:       procBackupDefSetting,
		SREQ_GET_EEPROM_TO_BIN:        procGetEepromToBin,
		SREQ_SET_EEPROM_FROM_BIN:      procSetEepromFromBin,
		SREQ_SET_EEPROM_FROM_BIN_FC:   procSetEepromFromBinFc,
		SREQ_DOWN_FIRMWARE_WIFI:       procDownFirmwareWifi,
		SREQ_GET_BASIC_DATA:           procGetBasicData,
		SREQ_SET_LIMIT_TO_SCALE:       procSetLimitToScale,
		SREQ_OPEN_BILL_SEND:           procOpenBillSend,
		SREQ_DIS_PASSTH_MODE:          procDisPassthMode,
		SREQ_CLOSE_SERIAL_PORT:        procCloseSerialPort,
		SREQ_OPEN_SERIAL_PORT:         procOpenSerialPort,
		SREQ_EXPORT_RECS:              procExportRecs,
		SREQ_SEND_SCALE_ALIVE:         procSendScaleAlive,
		SREQ_SET_MAX_RANGE1:           procSetMaxRange1,
		SREQ_SET_MAX_RANGE2:           procSetMaxRange2,
		SREQ_GET_MAX_RANGE1:           procGetMaxRange1,
		SREQ_GET_MAX_RANGE2:           procGetMaxRange2,

		SREQ_CAL_WGT:               procCalWeight,
		SREQ_SEND_CAL_HEART_BEAT:   procSendCalHeart,
		SREQ_SET_DECIMAL_VALUE:     procSetDecimalValue,
		SREQ_SET_GADUATION1_VALUE:  procSetGaduation1Value,
		SREQ_SET_GADUATION2_VALUE:  procSetGaduation2Value,
		SREQ_GET_GADUATION1_VALUE:  procGetGaduation1Value,
		SREQ_GET_GADUATION2_VALUE:  procGetGaduation2Value,
		SREQ_GET_DECIMAL_VALUE:     procGetDecimalValue,
		SREQ_SET_SERIAL_PORT:       procSetSerialPort,
		SREQ_GET_SERIAL_PORT:       procGetSerialPort,
		SREQ_SET_WEIGHT_UNIT:       procSetWeightUnit,
		SREQ_GET_WEIGHT_UNIT:       procGetWeightUnit,
		SREQ_SET_INITIAL_ZERO:      procSetInitialZero,
		SREQ_GET_INITIAL_ZERO:      procGetInitialZero,
		SREQ_SET_MANUAL_ZERO:       procSetManualZero,
		SREQ_GET_MANUAL_ZERO:       procGetManualZero,
		SREQ_SET_ZERO_TRACKING:     procSetZeroTracking,
		SREQ_GET_ZERO_TRACKING:     procGetZeroTracking,
		SREQ_SET_GRAV_ACC:          procSetGravAcc,
		SREQ_GET_GRAV_ACC:          procGetGravAcc,
		SREQ_SET_FORCE_UNTARE:      procSetForceUnTare,
		SREQ_GET_SEAL_STATUS:       procGetSealStatus,
		SREQ_SOFT_SEAL:             procSoftSeal,
		SREQ_REMOVE_SOFT_SEAL:      procRemoveSoftSeal,
		SREQ_REMOVE_SOFT_SEAL_ONCE: procRemoveSoftSealOnce,
		SREQ_GET_WIRED_IP:          procGetWiredIp,
		SREQ_SET_WIRED_IP:          procSetWiredIp,
		SREQ_SET_WIRED_DHCP:        procSetWiredDhcp,
		SREQ_GET_WIRED_DHCP:        procGetWiredDhcp,
		SREQ_INIT_WIFI:             procInitWifi,
		SREQ_GET_MODEL:             procGetModel,
		SREQ_EN_CODE:               procEnCode,
		SREQ_DIS_CODE:              procDisCode,
		SREQ_ASK_ROM_VERSION:       procAskRomVersion,
	}

	conversionMap = map[SReqType]m.RespMsgType{
		SREQ_ZERO:                     m.ZERO_CMD_RESP,
		SREQ_TARE:                     m.TARE_CMD_RESP,
		SREQ_ZERO_UNSTABLE:            m.ZERO_UNSTABLE_CMD_RESP,
		SREQ_TARE_UNSTABLE:            m.TARE_UNSTABLE_CMD_RESP,
		SREQ_GET_WEIGHT:               m.WEIGHT_DATA_RESP,
		SREQ_SEND_WT_CONT:             m.WEIGHT_DATA_RESP,
		SREQ_STOP_SEND_WT:             m.WEIGHT_DATA_RESP,
		SREQ_REG_WEIGHT_DATA:          m.REG_WEIGHT_RESP,
		SREQ_UNREG_WEIGHT_DATA:        m.UNREG_WEIGHT_RESP,
		SREQ_GET_RECS:                 m.GET_RECS_RESP,
		SREQ_ADD_REC:                  m.ADD_REC_RESP,
		SREQ_DEL_REC:                  m.DEL_REC_RESP,
		SREQ_DOWN_PRN_FMT:             m.DOWN_PRN_FMT_RESP,
		SREQ_GET_AP_LIST:              m.GET_AP_LIST_RESP,
		SREQ_RESCAN_AP_LIST:           m.RESCAN_AP_LIST_RESP,
		SREQ_CONNECT_AP:               m.CONNECT_AP_RESP,
		SREQ_CONNECT_AP_ONE_KEY:       m.CONNECT_AP_ONE_KEY_RESP,
		SREQ_SET_WIFI_DYNAMIC_IP:      m.SET_WIFI_DYNAMIC_IP_RESP,
		SREQ_SET_WIFI_STATIC_IP:       m.SET_WIFI_STATIC_IP_RESP,
		SREQ_GET_IP_INFO:              m.GET_IP_INFO_RESP,
		SREQ_MODIFY_BT_NAME:           m.MODIFY_BT_NAME_RESP,
		SREQ_SEND_DATA_TO_BT:          m.SEND_DATA_TO_BT_RESP,
		SREQ_SEND_DATA_TO_WIFI:        m.SEND_DATA_TO_WIFI_RESP,
		SREQ_GET_IP_MODE:              m.GET_IP_MODE_RESP, //FLF
		SREQ_GET_WIFI_INFO:            m.GET_IP_INFO_RESP,
		SREQ_GET_BUILD_INFO:           m.GET_BUILD_INFO_RESP,
		SREQ_GET_SCALE_TIME:           m.GET_SCALE_TIME_RESP,
		SREQ_SET_SCALE_TIME:           m.SET_SCALE_TIME_RESP, //20240125@FLF
		SREQ_GET_ONE_EEPROM_INFO:      m.GET_ONE_EEPROM_INFO_RESP,
		SREQ_GET_ALL_EEPROM_INFO:      m.GET_ALL_EEPROM_INFO_RESP,
		SREQ_SET_OUTPUT_FMT:           m.SET_OUTPUT_FMT_RESP,
		SREQ_OPNE_SCALE_PASSTHROUGH:   m.OPEN_SCALE_PASSTHROUGH_RESP, //20231023@FLF
		SREQ_CLOSE_SCALE_PASSTHROUGH:  m.CLOSE_SCALE_PASSTHROUGH_RESP,
		SREQ_CHANGE_SCALE_PASSTH_MODE: m.CHANGE_SCALE_PASSTH_MODE_RESP,
		SREQ_GET_SCALE_INFO:           m.GET_SCALE_INFO_RESP,
		SREQ_GET_FACTORY_INFO:         m.GET_FACTORY_INFO_RESP,
		SREQ_GET_WEIGHT_ERR:           m.GET_WEIGHT_ERR_RESP,
		SREQ_DOWN_PLU:                 m.DOWN_PLU_RESP,
		SREQ_DEL_PLU:                  m.DEL_PLU_RESP,
		SREQ_INSERT_PLU:               m.INSERT_PLU_RESP,
		SREQ_CHANGE_WIFI_MODE:         m.CHANGE_WIFI_MODE_RESP,
		SREQ_MODIFY_VAR_VALUE:         m.MODIFY_VAR_RESP,
		SREQ_SET_SERVER_IP:            m.SET_SERVER_IP_RESP,
		SREQ_EN_FACTORY_MODE:          m.EN_FACTORY_MODE_RESP,
		SREQ_DOWN_DEFAULT_PRN_FMT:     m.DOWN_DEFAULT_PRN_FMT_RESP,
		SREQ_DOWN_FACTORY_INFO_FC:     m.DOWN_FACTORY_INFO_FC_RESP,
		SREQ_DOWN_FACTORY_INFO:        m.DOWN_FACTORY_INFO_RESP,
		SREQ_BACKUP_DEF_SETTING:       m.BACKUP_DEF_SETTING_RESP,
		SREQ_GET_EEPROM_TO_BIN:        m.GET_EEPROM_TO_BIN_RESP,
		SREQ_SET_EEPROM_FROM_BIN:      m.SET_EEPROM_FROM_BIN_RESP,
		SREQ_SET_EEPROM_FROM_BIN_FC:   m.SET_EEPROM_FROM_BIN_RESP,
		SREQ_DOWN_FIRMWARE_WIFI:       m.DOWN_FIRMWARE_WIFI_RESP,
		SREQ_GET_BASIC_DATA:           m.GET_BASIC_DATA_RESP,
		SREQ_SET_LIMIT_TO_SCALE:       m.SET_LIMIT_TO_SCALE_RESP,
		SREQ_OPEN_BILL_SEND:           m.OPEN_BILL_SEND_RESP,
		SREQ_CLOSE_SERIAL_PORT:        m.CLOSE_SERIAL_PORT_RESP,
		SREQ_OPEN_SERIAL_PORT:         m.OPEN_SERIAL_PORT_RESP,
		SREQ_EXPORT_RECS:              m.EXPORT_RECS_RESP,

		SREQ_GET_SEAL_STATUS:       m.GET_SEAL_STATUS_RESP,
		SREQ_SOFT_SEAL:             m.SOFT_SEAL_RESP,
		SREQ_REMOVE_SOFT_SEAL:      m.REMOVE_SOFT_SEAL_RESP,
		SREQ_GET_WIRED_IP:          m.GET_WIRED_IP_RESP,
		SREQ_SET_WIRED_IP:          m.SET_WIRED_IP_RESP,
		SREQ_SET_WIRED_DHCP:        m.SET_WIRED_DHCP_RESP,
		SREQ_GET_WIRED_DHCP:        m.GET_WIRED_DHCP_RESP,
		SREQ_REMOVE_SOFT_SEAL_ONCE: m.REMOVE_SOFT_SEAL_ONCE_RESP,

		SREQ_GET_MODEL: m.GET_MODEL_RESP,
	}
}
