package svc

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	mcmd "tmaxsrv/cmd"
	m "tmaxsrv/comm"
	l "tmaxsrv/log"
	"tmaxsrv/util"
)

const (
	NORMAL_WEIGHT_MODE = 0
	CHECK_WEIGHT_MODE  = 1
	TACKE_IN_MODE      = 2
	TACKE_OUT_MODE     = 3
)

// 带校验的openblt升级
func (c *Scale) UpdateFirmware(name string) (*ScaleRespMsg, error) {

	var err error
	pickerFn := c.MySerial.pickerFn
	c.MySerial.Close()

	output := make(chan string)
	done := make(chan error)
	bootCommanderPath := m.GetExePath()
	print(bootCommanderPath)
	bootCommanderPath = filepath.Join(bootCommanderPath, "BootCommander.exe")
	go util.RunCommand(output, done, bootCommanderPath, "-t=xcp_rs232", "-d="+c.Pcnf.DevPath, "-b=57600", name)

	isFinish := false
	isStartUpdate := false
	var cmdOutput string
	var percentage float32 = 0.0
	var updateTimeMs float32 = 0.0
	var respMsg *ScaleRespMsg
	respOk := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "ok", ScaleId: c.Id}
	respFail := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail", ScaleId: c.Id}
	for {
		if isFinish {
			break
		}
		select {
		case line := <-output:
			cmdOutput += line
			fmt.Println(line) // Print each line of output as it is received
			l.Log.Debug(line)

			if !isStartUpdate && strings.Contains(cmdOutput, "Erasing") {
				isStartUpdate = true
				// inform UI update firmware is
				re := regexp.MustCompile(`Erasing (\d+) bytes`)
				match := re.FindStringSubmatch(cmdOutput)
				if len(match) > 1 {
					number := match[1]
					updateTimeInt, _ := strconv.Atoi(number)
					updateTimeMs = float32(updateTimeInt) / 2.6
					fmt.Println(number) // 输出: 64980
				}

				respMsg := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: "started", ScaleId: c.Id}
				result, _ := json.Marshal(respMsg)
				c.client.sendCh <- result
			}
		case err = <-done:
			if strings.Contains(cmdOutput, "Finishing programming session...[OK]") {
				respMsg100 := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(100), ScaleId: c.Id}
				result, _ := json.Marshal(respMsg100)
				c.client.sendCh <- result
				time.Sleep(500 * time.Millisecond)
				respMsg = respOk
			} else {
				respMsg = respFail
			}
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Command completed")
			}
			isFinish = true
		case <-time.After(time.Duration(250) * time.Millisecond):
			if isStartUpdate {
				percentage = percentage + 25000.0/updateTimeMs
				percentageInt := int(percentage)
				respMsg := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(percentageInt), ScaleId: c.Id}
				result, _ := json.Marshal(respMsg)
				c.client.sendCh <- result
			}

		}
	}
	if c.MySerial, err = NewSerial(c.Pcnf, pickerFn, true); err != nil {
		l.Log.Error(err.Error())
	}
	// os.Remove(name)
	return respMsg, nil
}

// 不带校验的openblt升级 比如小天平
func (c *Scale) UpdateFirmwareOld(name string) (*ScaleRespMsg, error) {

	pickerFn := c.MySerial.pickerFn
	c.MySerial.Close()
	output := make(chan string)
	done := make(chan error)
	bootCommanderPath := m.GetExePath()
	print(bootCommanderPath)
	bootCommanderPath = filepath.Join(bootCommanderPath, "oldBoot") //在oldBoot文件夹下
	bootCommanderPath = filepath.Join(bootCommanderPath, "BootCommander.exe")
	go util.RunCommand(output, done, bootCommanderPath, "-t=xcp_rs232", "-d="+c.Pcnf.DevPath, "-b=57600", name)
	var err error
	isFinish := false
	isStartUpdate := false
	var cmdOutput string
	var percentage float32 = 0.0
	var updateTimeMs float32 = 0.0
	var respMsg *ScaleRespMsg
	respOk := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "ok", ScaleId: c.Id}
	respFail := &ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_RESP, MsgBody: "fail", ScaleId: c.Id}
	for {
		if isFinish {
			break
		}
		select {
		case line := <-output:
			cmdOutput += line
			fmt.Println(line) // Print each line of output as it is received
			l.Log.Debug(line)

			if !isStartUpdate && strings.Contains(cmdOutput, "Erasing") {
				isStartUpdate = true
				// inform UI update firmware is
				re := regexp.MustCompile(`Erasing (\d+) bytes`)
				match := re.FindStringSubmatch(cmdOutput)
				if len(match) > 1 {
					number := match[1]
					updateTimeInt, _ := strconv.Atoi(number)
					updateTimeMs = float32(updateTimeInt) / 2.6
					fmt.Println(number) // 输出: 64980
				}

				respMsg := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: "started", ScaleId: c.Id}
				result, _ := json.Marshal(respMsg)
				c.client.sendCh <- result
			}
		case err = <-done:
			if strings.Contains(cmdOutput, "Finishing programming session...[OK]") {
				respMsg100 := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(100), ScaleId: c.Id}
				result, _ := json.Marshal(respMsg100)
				c.client.sendCh <- result
				time.Sleep(500 * time.Millisecond)
				respMsg = respOk
			} else {
				respMsg = respFail
			}
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Command completed")
			}
			isFinish = true
		case <-time.After(time.Duration(250) * time.Millisecond):
			if isStartUpdate {
				percentage = percentage + 25000.0/updateTimeMs
				percentageInt := int(percentage)
				respMsg := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(percentageInt), ScaleId: c.Id}
				result, _ := json.Marshal(respMsg)
				c.client.sendCh <- result
			}

		}
	}
	if c.MySerial, err = NewSerial(c.Pcnf, pickerFn, true); err != nil {
		l.Log.Error(err.Error())
	}
	// os.Remove(name)
	return respMsg, nil
}

func (c *Scale) CheckSerialPort() (*ScaleRespMsg, error) {
	l.Log.Debug("check serial port")
	if c.ScaleCat == m.SCALE_C51 {
		if !c.Conn.IsOnline {
			return &ScaleRespMsg{MsgType: m.CHECK_SERIAL_PORT_RESP, MsgBody: "fail", ScaleId: c.Id}, nil

		}
		var factoryInfo FIFromScale
		req := ReqModifyScaleSn{
			ScaleId:    c.Id,
			ScaleModel: "DC500",
			Sn:         "",
		}
		c.scaleMgr.UpdateScaleSn(req)

		factoryInfo.ModelName = "DC500"
		factoryInfo.ScaleSn = ""
		msgBody, _ := json.MarshalToString(factoryInfo)
		return &ScaleRespMsg{MsgType: m.CHECK_SERIAL_PORT_RESP, MsgBody: msgBody, ScaleId: c.Id}, nil
	}

	reqMsg, _ := excuteSimpCmd(c, m.CMD_GET_FACTORY_INFO, m.GET_FACTORY_INFO_RESP)
	if reqMsg.MsgBody == nil {
		return &ScaleRespMsg{MsgType: m.CHECK_SERIAL_PORT_RESP, MsgBody: "fail", ScaleId: c.Id}, nil
	}
	var dataStruct FIFromScale
	msgBodyStr, _ := reqMsg.MsgBody.(string)
	err := json.UnmarshalFromString(msgBodyStr, &dataStruct)
	if err == nil {
		req := ReqModifyScaleSn{
			ScaleId:    c.Id,
			ScaleModel: dataStruct.ModelName,
			Sn:         dataStruct.ScaleSn,
		}
		c.scaleMgr.UpdateScaleSn(req)

	}

	return &ScaleRespMsg{MsgType: m.CHECK_SERIAL_PORT_RESP, MsgBody: reqMsg.MsgBody, ScaleId: c.Id}, nil
}

//原来的打开工厂模式的写法
// func (c *Scale) CheckSerialPort() (*ScaleRespMsg, error) {
// 	l.Log.Debug("check serial port")
// 	reqMsg, _ := excuteSimpCmd(c, m.CMD_EN_FAC_MODE, m.EN_FAC_MODE_RESP)
// 	if reqMsg.MsgBody == "ok" {
// 		return &ScaleRespMsg{MsgType: m.CHECK_SERIAL_PORT_RESP, MsgBody: "ok", ScaleId: c.Id}, nil
// 	}
// 	return &ScaleRespMsg{MsgType: m.CHECK_SERIAL_PORT_RESP, MsgBody: "fail", ScaleId: c.Id}, nil
// }

func (c *Scale) GetBuildInfo() (*ScaleRespMsg, error) {
	l.Log.Debug("get build info")
	_, err, res := openFactory(c)
	if err != nil || !res {
		l.Log.Debug(err)
		respMsg := &ScaleRespMsg{MsgType: m.GET_BUILD_INFO_RESP, MsgBody: "failed to open factory", ScaleId: c.Id}
		return respMsg, err
	}
	reqMsg, err := excuteSimpCmd(c, m.CMD_GET_BUILD_INFO, m.GET_BUILD_INFO_RESP)
	return reqMsg, err
}

func (c *Scale) GetScaleTime() (*ScaleRespMsg, error) {
	l.Log.Debug("get scale time")
	_, err, res := openFactory(c)
	if err != nil || !res {
		l.Log.Debug(err)
		respMsg := &ScaleRespMsg{MsgType: m.GET_SCALE_TIME_RESP, MsgBody: "failed to open factory", ScaleId: c.Id}
		return respMsg, err
	}
	reqMsg, err := excuteSimpCmd(c, m.CMD_GET_SCALE_TIME, m.GET_SCALE_TIME_RESP)
	return reqMsg, err
}

func (c *Scale) GetScaleInfo() (*ScaleRespMsg, error) {
	l.Log.Debug("get scale info")
	reqMsg, err := excuteSimpCmd(c, m.CMD_GET_SCALE_INFO, m.GET_SCALE_INFO_RESP)
	return reqMsg, err
}

func (c *Scale) GetFactoryInfo() (*ScaleRespMsg, error) {
	l.Log.Debug("get factory info")
	reqMsg, err := excuteSimpCmd(c, m.CMD_GET_FACTORY_INFO, m.GET_FACTORY_INFO_RESP)
	return reqMsg, err
}

func getPercentage(data string) string {
	percentage := "0%"
	pattern := `\[ *(\d+)%\][^[]*$`
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(data)
	if len(match) > 1 {
		fmt.Println(match[1])
		percentage = match[1]
	} else {
		fmt.Println("No match found.")
	}

	return percentage + "%"
}

func (c *Scale) PerfZero() (*ScaleRespMsg, error) {
	l.Log.Debug("perform zero")
	return excuteSimpCmd(c, m.CMD_ZERO, m.ZERO_CMD_RESP)
}

func (c *Scale) PerfTare() (*ScaleRespMsg, error) {
	l.Log.Debug("perform tare")
	return excuteSimpCmd(c, m.CMD_TARE, m.TARE_CMD_RESP)
}

func (c *Scale) PerfZeroUnstable() (*ScaleRespMsg, error) {
	l.Log.Debug("perform zero unstable")
	return excuteSimpCmd(c, m.CMD_ZERO_UNSTABLE, m.ZERO_UNSTABLE_CMD_RESP)
}

func (c *Scale) PerfTareUnstable() (*ScaleRespMsg, error) {
	l.Log.Debug("perform tare unstable")
	return excuteSimpCmd(c, m.CMD_TARE_UNSTABLE, m.TARE_UNSTABLE_CMD_RESP)
}

func (c *Scale) ReadWeight() (*ScaleRespMsg, error) {
	// if c.isOldC51Scale {
	//     return perfCmd(c, []byte(GET_WEIGHT_CMD))
	// } else {
	//     _, err := perfCmdNwaitResult(c, TMAX_READ_WEIGHT_CMD, WEIGHT_DATA_RESP)
	//     return err == nil
	// }
	return nil, nil // FIXME:
}

func (c *Scale) RegWeightData() (*ScaleRespMsg, error) { //FLF

	l.Log.Debug("register weight data")
	c.isSendUnolicitedData = true
	c.isScalePassth = false //TODO:
	if c.ScaleCat == m.SCALE_C51 {
		return &ScaleRespMsg{MsgType: m.UNREG_WEIGHT_RESP, MsgBody: "ok", ScaleId: c.Id}, nil
	}
	msg, err, res := openFactory(c)
	if err != nil || !res {
		l.Log.Debug(err)
		msg.MsgType = m.REG_WEIGHT_RESP
		return msg, err
	}
	// DisFacMode(c) // TODO: check return value
	// enable scale sending weighing info continually
	return excuteSimpCmd(c, m.CMD_EN_CONTINUE_MODE, m.REG_WEIGHT_RESP)
}

// func sendErrMsg(c *Scale, msg *ScaleRespMsg) {
//     msgStr, err := json.MarshalToString(msg)
//     if err != nil {
//         if c.client != nil {
//             c.client.sendCh <- []byte(msgStr)
//         }
//     }
// }

func (c *Scale) UnRegWeightData() (*ScaleRespMsg, error) {

	c.isSendUnolicitedData = false
	c.isScalePassth = false
	if c.ScaleCat == m.SCALE_C51 {
		return &ScaleRespMsg{MsgType: m.UNREG_WEIGHT_RESP, MsgBody: "ok", ScaleId: c.Id}, nil
	}
	msg, err := perfCmdNwaitResult(c, mcmd.DIS_CONT_MODE_CMD_TMAX, m.UNREG_WEIGHT_RESP, mcmd.CMD_TIMEOUT_SHORT_1500_MS)

	if str, ok := msg.MsgBody.(string); ok && strings.Contains(str, "ok") {
		return msg, err
	}
	msg, err, res := openFactory(c)
	if err != nil || !res {
		l.Log.Debug(err)
		msg.MsgType = m.UNREG_WEIGHT_RESP
		return msg, err
	}
	msg, err = perfCmdNwaitResult(c, mcmd.DIS_CONT_MODE_CMD_TMAX, m.UNREG_WEIGHT_RESP, mcmd.CMD_TIMEOUT_SHORT_1500_MS)
	//sendErrMsg(c, msg)
	// _, _ = EnFacMode(c)
	return msg, err
}

func (c *Scale) OpenScalePassth() (*ScaleRespMsg, error) { //FLF
	if c.ScaleCat == m.SCALE_C51 {
		return &ScaleRespMsg{MsgType: m.UNREG_WEIGHT_RESP, MsgBody: "ok", ScaleId: c.Id}, nil
	}
	l.Log.Debug("register weight data")
	c.isSendUnolicitedData = true
	// _, err := EnFacMode(c)
	// if err != nil {
	// 	return &ScaleRespMsg{}, err //FLF
	// }
	// DisFacMode(c) // TODO: check return value
	// enable scale sending weighing info continually
	reqMsg, _ := excuteSimpCmd(c, m.CMD_DIS_FAC_MODE, m.DIS_FAC_MODE_RESP)
	if reqMsg.MsgBody == "ok" {
		return &ScaleRespMsg{MsgType: m.OPEN_SCALE_PASSTHROUGH_RESP, MsgBody: "ok", ScaleId: c.Id}, nil

	}
	return &ScaleRespMsg{MsgType: m.OPEN_SCALE_PASSTHROUGH_RESP, MsgBody: "fail", ScaleId: c.Id}, nil
}

func (c *Scale) CloseScalePassth() (*ScaleRespMsg, error) {
	if c.ScaleCat == m.SCALE_C51 {
		return &ScaleRespMsg{MsgType: m.UNREG_WEIGHT_RESP, MsgBody: "ok", ScaleId: c.Id}, nil
	}
	c.isSendUnolicitedData = false
	// _, err := EnFacMode(c)
	// if err != nil {
	// 	return &ScaleRespMsg{}, err
	// }
	// enable scale sending weighing info continually
	//sendErrMsg(c, msg)
	// _, _ = EnFacMode(c)
	reqMsg, _ := excuteSimpCmd(c, m.CMD_CHECK_FAC_MODE, m.EN_FAC_MODE_RESP)
	if reqMsg.MsgBody == "ok" {
		return &ScaleRespMsg{MsgType: m.CLOSE_SCALE_PASSTHROUGH_RESP, MsgBody: "ok", ScaleId: c.Id}, nil
	}
	return &ScaleRespMsg{MsgType: m.CLOSE_SCALE_PASSTHROUGH_RESP, MsgBody: "fail", ScaleId: c.Id}, nil
}

func (c *Scale) GetRecs(scaleMode string) ([]ScaleRec, error) {
	result := strings.Split(scaleMode, ",")
	var scaleModel = ""
	var scaleSn = ""
	var scaleName = ""
	var page = ""
	var pageSize = ""
	var columnName = "id"
	var direction = "descending"

	if len(result) == 8 {
		scaleModeInt, _ := strconv.Atoi(result[0])
		scaleModel = result[1]
		scaleSn = result[2]
		scaleName = result[3]
		page = result[4]
		pageSize = result[5]
		columnName = result[6]
		if result[7] != "" {
			direction = result[7]
		}

		if scaleModeInt == NORMAL_WEIGHT_MODE {
			return c.scaleMgr.recPb.GetRecsList(c, scaleModel, scaleSn, scaleName, page, pageSize, columnName, direction)
		}
		if scaleModeInt == CHECK_WEIGHT_MODE {
			return c.scaleMgr.recCheckWeigherPb.GetRecsList(c, scaleModel, scaleSn, scaleName, page, pageSize, columnName, direction)
		}
		if scaleModeInt == TACKE_IN_MODE {
			return c.scaleMgr.recTakeInPb.GetRecsList(c, scaleModel, scaleSn, scaleName, page, pageSize, columnName, direction)
		}
		if scaleModeInt == TACKE_OUT_MODE {
			return c.scaleMgr.recTakeOutPb.GetRecsList(c, scaleModel, scaleSn, scaleName, page, pageSize, columnName, direction)
		}
	}
	return []ScaleRec{}, nil // c.scaleMgr.recPb.GetRecsList(c, scaleModel, scaleSn, scaleName)
}

func (c *Scale) GetWgtRecs(scaleMode string, offset int, limit int) ([]ScaleRec, error) {
	result := strings.Split(scaleMode, ",")
	var scaleModel = ""
	var scaleSn = ""
	var scaleName = ""

	if len(result) == 5 {
		scaleModeInt, _ := strconv.Atoi(result[0])
		scaleModel = result[1]
		scaleSn = result[2]
		scaleName = result[3]

		if scaleModeInt == NORMAL_WEIGHT_MODE {
			return c.scaleMgr.recPb.GetWgtRecsList(c, scaleModel, scaleSn, scaleName, offset, limit)
		}
		if scaleModeInt == CHECK_WEIGHT_MODE {
			return c.scaleMgr.recCheckWeigherPb.GetWgtRecsList(c, scaleModel, scaleSn, scaleName, offset, limit)
		}
		if scaleModeInt == TACKE_IN_MODE {
			return c.scaleMgr.recTakeInPb.GetWgtRecsList(c, scaleModel, scaleSn, scaleName, offset, limit)
		}
		if scaleModeInt == TACKE_OUT_MODE {
			return c.scaleMgr.recTakeOutPb.GetWgtRecsList(c, scaleModel, scaleSn, scaleName, offset, limit)
		}
	}
	return []ScaleRec{}, nil // c.scaleMgr.recPb.GetRecsList(c, scaleModel, scaleSn, scaleName)
}

func (c *Scale) AddRec(rec ScaleRec) error {
	// rec.ScaleModel = c.Model
	// rec.ScaleSn = c.Sn
	scaleModeInt, _ := strconv.Atoi(rec.ScaleMode)

	LogScaleWgtOperation(scaleModeInt, rec, "")

	switch scaleModeInt {
	case NORMAL_WEIGHT_MODE:
		return c.scaleMgr.recPb.InsertRec(rec)
	case CHECK_WEIGHT_MODE:
		return c.scaleMgr.recCheckWeigherPb.InsertRec(rec)
	case TACKE_IN_MODE:
		return c.scaleMgr.recTakeInPb.InsertRec(rec)
	case TACKE_OUT_MODE:
		return c.scaleMgr.recTakeOutPb.InsertRec(rec)
	default:
		return c.scaleMgr.recPb.InsertRec(rec)
	}
}

func (c *Scale) DelRec(recId uint, scaleMode uint, modelName string, scaleSn string) error {
	switch scaleMode {
	case NORMAL_WEIGHT_MODE:
		switch recId {
		case 999999999:
			return c.scaleMgr.recPb.DeleteAllRec(modelName, scaleSn)
		default:
			return c.scaleMgr.recPb.DeleteRec(recId)
		}
	case CHECK_WEIGHT_MODE:
		switch recId {
		case 999999999:
			return c.scaleMgr.recCheckWeigherPb.DeleteAllRec(modelName, scaleSn)
		default:
			return c.scaleMgr.recCheckWeigherPb.DeleteRec(recId)
		}
	case TACKE_IN_MODE:
		switch recId {
		case 999999999:
			return c.scaleMgr.recTakeInPb.DeleteAllRec(modelName, scaleSn)
		default:
			return c.scaleMgr.recTakeInPb.DeleteRec(recId)
		}
	case TACKE_OUT_MODE:
		switch recId {
		case 999999999:
			return c.scaleMgr.recTakeOutPb.DeleteAllRec(modelName, scaleSn)
		default:
			return c.scaleMgr.recTakeOutPb.DeleteRec(recId)
		}
	default:
		return nil
	}
}

// 重启
func Reboot(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("send reboot cmd to scale")
	return excuteSimpCmd(s, m.CMD_REBOOT, m.REBOOT_RESP, 1)
}

// 打开工厂模式
func EnFacMode(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("send enable factory mode cmd to scale")
	return excuteSimpCmd(s, m.CMD_CHECK_FAC_MODE, m.EN_FAC_MODE_RESP)
}

// 关闭工厂模式
func DisFacMode(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("send disable factory mode cmd to scale")
	return excuteSimpCmd(s, m.CMD_DIS_FAC_MODE, m.DIS_FAC_MODE_RESP)
}

func excuteSimpCmd(s *Scale, cmdType m.CmdType, respType m.RespMsgType, perfTimes ...int) (*ScaleRespMsg, error) {
	// if s.ScaleCat != m.SCALE_TMAX {
	// 	return &ScaleRespMsg{}, nil //20250905

	// }

	scaleCmdExtractorFn := s.composer.ComposeCmd
	cmd, timeoutMs, err := scaleCmdExtractorFn(s.composer, cmdType, m.CmdData{})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	var curTimeoutMs = 3
	if len(perfTimes) > 0 {
		curTimeoutMs = perfTimes[0]
	}
	println(fmt.Sprintf("%x", cmd))

	if res, err := perfCmdNwaitResult(s, cmd, respType, timeoutMs, curTimeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else {
		return res, nil
	}
}

// 打开BT透传模式
func EnPassthrough(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("send enable passthrough mode cmd to scale")
	return excuteSimpCmd(s, m.CMD_EN_PASSTH, m.EN_PASSTH_MODE_RESP)
}

// 关闭BT透传模式
func DisPassthrough(s *Scale) (*ScaleRespMsg, error) {
	return excuteSimpCmd(s, m.CMD_DIS_PASSTH, m.DIS_PASSTH_MODE_RESP)
	// return &ScaleRespMsg{MsgType: m.DIS_PASSTH_MODE_RESP, MsgBody: "ok", ScaleId: s.Id}, nil
}

var GExpectBTResp m.RespMsgType

func (c *Scale) ModifyBTName(name string) (*ScaleRespMsg, error) {
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_MODIFY_BT_NAME, m.CmdData{Type: m.DATA_TYPE_STR, Data: name})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectBTResp = m.MODIFY_BT_NAME_RESP
	return perfCmdNwaitResult(c, cmd, m.MODIFY_BT_NAME_RESP, timeoutMs)
}

var GExpectWifiResp m.RespMsgType

// get At version
func GetWifiAtVersion(s *Scale) (*ScaleRespMsg, error) {
	GExpectWifiResp = m.GET_AT_VERSION_RESP
	// return excuteSimpCmd(c, m.CMD_WIFI_AT_VERSION, m.GET_AT_VERSION_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_AT_VERSION, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_AT_VERSION_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_AT_VERSION_RESP, timeoutMs)
}

// get At mode
func GetWifiAtMode(s *Scale) (*ScaleRespMsg, error) {
	GExpectWifiResp = m.GET_AT_MODE_RESP
	// return excuteSimpCmd(c, m.CMD_WIFI_AT_MODE, m.GET_AT_MODE_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_AT_MODE, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_AT_MODE_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_AT_MODE_RESP, timeoutMs)
}

// Get AP list
func GetApList(s *Scale) (*ScaleRespMsg, error) {
	GExpectWifiResp = m.GET_AP_LIST_RESP
	// return excuteSimpCmd(c, m.CMD_WIFI_GET_AP_LIST, m.GET_AP_LIST_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_AP_LIST, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_AP_LIST_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_AP_LIST_RESP, timeoutMs)
}

// Get Wifi AP info ESP32
func GetWifiApInfo32(s *Scale) (*ScaleRespMsg, error) {
	GExpectWifiResp = m.GET_WIFI_AP_INFO_RESP
	// return excuteSimpCmd(c, m.CMD_WIFI_GET_AP_INFO_32, m.GET_WIFI_AP_INFO_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_AP_INFO_32, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_WIFI_AP_INFO_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_WIFI_AP_INFO_RESP, timeoutMs)
}

// Get Wifi AP info
func GetWifiApInfo(s *Scale) (*ScaleRespMsg, error) {
	GExpectWifiResp = m.GET_WIFI_AP_INFO_RESP
	// return excuteSimpCmd(c, m.CMD_WIFI_GET_AP_INFO, m.GET_WIFI_AP_INFO_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_AP_INFO, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_WIFI_AP_INFO_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_WIFI_AP_INFO_RESP, timeoutMs)
}

// Send data to BT
func SendDataToBT(c *Scale, data string) (*ScaleRespMsg, error) {
	l.Log.Debug("Send data to BT")
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_BT_DATA_PASSTH, m.CmdData{Type: m.DATA_TYPE_STR, Data: data})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectBTResp = m.BT_PASSTH_DATA_RESP
	return perfCmdNwaitResult(c, cmd, m.BT_PASSTH_DATA_RESP, timeoutMs)
}

// Send data to Wifi
func SendDataToWifi(s *Scale, data string) (*ScaleRespMsg, error) {
	l.Log.Debug("Send data to WIFI")
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_DATA_PASSTH, m.CmdData{Type: m.DATA_TYPE_STR, Data: data})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectBTResp = m.WIFI_PASSTH_DATA_RESP
	return perfCmdNwaitResult(s, cmd, m.WIFI_PASSTH_DATA_RESP, timeoutMs)
}

func SetWifiDynamicIp(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("set wifi to dynamic IP")
	GExpectWifiResp = m.SET_WIFI_DYNAMIC_IP_RESP
	// return excuteSimpCmd(s, m.CMD_WIFI_EN_DHCP, m.SET_WIFI_DYNAMIC_IP_RESP, 1)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_EN_DHCP, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.SET_WIFI_DYNAMIC_IP_RESP
	return perfCmdNwaitResult(s, cmd, m.SET_WIFI_DYNAMIC_IP_RESP, timeoutMs)
}

func SetWifiDynamicIp32(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("set wifi to dynamic IP")
	GExpectWifiResp = m.SET_WIFI_DYNAMIC_IP_RESP
	// return excuteSimpCmd(s, m.CMD_WIFI_EN_DHCP_32, m.SET_WIFI_DYNAMIC_IP_RESP, 1)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_EN_DHCP_32, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.SET_WIFI_DYNAMIC_IP_RESP
	return perfCmdNwaitResult(s, cmd, m.SET_WIFI_DYNAMIC_IP_RESP, timeoutMs)
}

func SetWifiStaticIp(s *Scale, ip string, gateway string, netmask string) (*ScaleRespMsg, error) {
	l.Log.Debug("set wifi to static IP")
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_SET_STATIC_IP, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%s,%s,%s", ip, gateway, netmask)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.SET_WIFI_STATIC_IP_RESP
	return perfCmdNwaitResult(s, cmd, m.SET_WIFI_STATIC_IP_RESP, timeoutMs, 1)
}

func SetWifiStaticIp32(s *Scale, ip string, gateway string, netmask string) (*ScaleRespMsg, error) {
	l.Log.Debug("set wifi to static IP")
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_SET_STATIC_IP_32, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%s,%s,%s", ip, gateway, netmask)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.SET_WIFI_STATIC_IP_RESP
	return perfCmdNwaitResult(s, cmd, m.SET_WIFI_STATIC_IP_RESP, timeoutMs, 1)
}

// Connect to specifi AP
func ConnectWifiAp(s *Scale, ssid string, passwd string, bssid string) (*ScaleRespMsg, error) {
	l.Log.Debug("Connect to Wifi AP")
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_CONN_AP, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%s,%s,%s", ssid, passwd, bssid)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.CONNECT_AP_RESP
	return perfCmdNwaitResult(s, cmd, m.CONNECT_AP_RESP, timeoutMs, 1)
}

func ConnectWifiAp32(s *Scale, ssid string, passwd string, bssid string) (*ScaleRespMsg, error) {
	l.Log.Debug("Connect to Wifi AP")
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_CONN_AP32, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%s,%s,%s", ssid, passwd, bssid)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.CONNECT_AP_RESP
	msg, err := perfCmdNwaitResult(s, cmd, m.CONNECT_AP_RESP, timeoutMs, 1)
	if err != nil {
		return msg, err
	}
	if msg.MsgBody != "ok" {
		return msg, err
	}

	// if s.Model == "DPM" {
	// 	ReqSetServerMode(s, SRequest{})
	// } else {
	// 	return msg, nil
	// }

	return msg, nil
}

func ConnectWifiApOneKey(s *Scale, ssid string, bssid string, passwd string) (*ScaleRespMsg, error) {
	l.Log.Debug("Connect to Wifi AP")
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_CONN_AP_ONE_KEY, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%s,%s,%s", ssid, bssid, passwd)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.CONNECT_AP_ONE_KEY_RESP
	return perfCmdNwaitResult(s, cmd, m.CONNECT_AP_ONE_KEY_RESP, timeoutMs, 1)
}

// Get IP info from scale
func GetIpInfo(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("Get IP info from Scale")
	GExpectWifiResp = m.GET_IP_INFO_RESP
	// return excuteSimpCmd(s, m.CMD_WIFI_GET_IP_INFO, m.GET_IP_INFO_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_IP_INFO, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_IP_INFO_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_IP_INFO_RESP, timeoutMs)
}

// Get IP info from scale
func GetIpInfo32(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("Get IP info from Scale")
	GExpectWifiResp = m.GET_IP_INFO_RESP
	// return excuteSimpCmd(s, m.CMD_WIFI_GET_IP_INFO_32, m.GET_IP_INFO_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_IP_INFO_32, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_IP_INFO_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_IP_INFO_RESP, timeoutMs)
}

func ChangeWifiMode(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("Change wifi mode from Scale")
	GExpectWifiResp = m.CHANGE_WIFI_MODE_RESP
	// return excuteSimpCmd(s, m.CMD_CHANGE_WIFI_MODE, m.CHANGE_WIFI_MODE_RESP)

	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_CHANGE_WIFI_MODE, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.CHANGE_WIFI_MODE_RESP
	return perfCmdNwaitResult(s, cmd, m.CHANGE_WIFI_MODE_RESP, timeoutMs)
}

func GetIpMode(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("Get IP mode from Scale")
	GExpectWifiResp = m.GET_IP_MODE_RESP
	// return excuteSimpCmd(s, m.CMD_WIFI_GET_IP_MODE, m.GET_IP_MODE_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_IP_MODE, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_IP_MODE_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_IP_MODE_RESP, timeoutMs)
}

func GetIpMode32(s *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("Get IP mode from Scale")
	GExpectWifiResp = m.GET_IP_MODE_RESP
	// return excuteSimpCmd(s, m.CMD_WIFI_GET_IP_MODE_32, m.GET_IP_MODE_RESP)
	cmd, timeoutMs, err := s.composer.ComposeCmd(s.composer, m.CMD_WIFI_GET_IP_MODE_32, m.CmdData{Type: m.DATA_TYPE_STR, Data: ""})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	GExpectWifiResp = m.GET_IP_MODE_RESP
	return perfCmdNwaitResult(s, cmd, m.GET_IP_MODE_RESP, timeoutMs)
}

// func perfCmd(c *Scale, cmd []byte) bool {
// 	if err := writeScale(c, cmd); err != nil {
// 		l.Log.Error(err.Error())
// 		return false
// 	}
// 	return true
// }

func (c *Scale) AddPluDownRec(rec PluRec) error {
	return c.scaleMgr.pluFilePb.InsertRec(rec) //@FLF20240107
}

func (c *Scale) GetPluDownRec(md5Str string) ([]PluRec, error) {
	return c.scaleMgr.pluFilePb.GetPluPath(md5Str) //@FLF20240107
}

func perfCmdNwaitResult(c *Scale, cmd []byte, waitMsgType m.RespMsgType, timeoutMs ...int) (*ScaleRespMsg, error) {
	curTimeoutMs := 10000 //3000 // 3000 ms
	var ret *ScaleRespMsg
	var err error = nil
	var sendCmdTimes = 5

	if len(timeoutMs) > 0 {
		curTimeoutMs = timeoutMs[0]
	}

	if len(timeoutMs) > 1 {
		sendCmdTimes = timeoutMs[1]
	}
	for _, b := range cmd {
		fmt.Printf("%02x ", b) // 打印每个字节的 16 进制表示并用空格分隔
	}
	for i := 0; i < sendCmdTimes; i++ {

		if curTimeoutMs == mcmd.CMD_TIMEOUT_IMMEDIATE {
			if err = writeScale(c, cmd); err != nil {
				l.Log.Error(err.Error())
				ret = &ScaleRespMsg{MsgType: waitMsgType, ScaleId: c.Id, MsgBody: "error"}
				time.Sleep(500 * time.Millisecond)
				continue

			}
			return &ScaleRespMsg{MsgType: waitMsgType, ScaleId: c.Id, MsgBody: "done"}, nil
		}

		ch := make(chan *ScaleRespMsg, 10)
		c.RegisterNotif(waitMsgType, ch)
		defer func() {
			c.UnRegisterNotif(waitMsgType, ch)
			c.isWaintingResp = false
		}()

		GlastWantRespMsgType = waitMsgType
		c.isWaintingResp = true
		if err = writeScale(c, cmd); err != nil {
			l.Log.Errorf("【调试】writeScale报错: %v", err.Error())
			ret = &ScaleRespMsg{MsgType: waitMsgType, ScaleId: c.Id, MsgBody: "error"}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		l.Log.Infof("================wait: %v\n", waitMsgType)

		fmt.Printf("================wait: %v\n", waitMsgType)

		select {
		case ret = <-ch:
			l.Log.Infof("【调试】收到回复，提前结束等待: type=%v, body=%v", ret.MsgType, ret.MsgBody)
		case <-time.After(time.Duration(curTimeoutMs) * time.Millisecond):
			l.Log.Infof("【调试】等待超时: %v ms", curTimeoutMs)
			ret = &ScaleRespMsg{MsgType: waitMsgType, ScaleId: c.Id, MsgBody: "timeout"}
			err = fmt.Errorf("no response, time out")
			time.Sleep(500 * time.Millisecond)
			continue
			// default:
			// 	time.Sleep(time.Microsecond * 100)
			// 	continue
		}

		fmt.Printf("^^^^^^^^^^^^^^^^Got: %v\n", waitMsgType)
		l.Log.Infof("^^^^^^^^^^^^^^^^Got: %v\n", waitMsgType)

		/////////////

		break

	}

	return ret, err
}

func writeScale(c *Scale, data []byte) error {

	if c.MySerial != nil && c.MySerial.isDefault {
		if c.MySerial.toQuit {
			return fmt.Errorf("serial is closed")
		}
		if len(c.MySerial.sendCh) > SEND_CH_SIZE {
			return fmt.Errorf("serial sendCh full")
		}
		c.MySerial.sendCh <- data
	}

	if c.MyNet != nil && c.MyNet.isAlive && c.MyNet.isDefault {
		if c.MyNet.toQuit {
			return fmt.Errorf("net is closed")
		}
		if c.MyNet.conn != nil {
			if len(c.MyNet.sendCh) > SEND_CH_SIZE {
				return fmt.Errorf("net sendCh full")
			}
			c.MyNet.sendCh <- data //写数据到发送通道20250901
		}
	}

	if c.MyBluetooth != nil && c.MyBluetooth.isDefault {
		if c.MyBluetooth.toQuit {
			return fmt.Errorf("bluetooth is closed")
		}

		if len(c.MyBluetooth.sendCh) > SEND_CH_SIZE {
			return fmt.Errorf("bluetooth sendCh full")
		}
		c.MyBluetooth.sendCh <- data //写数据到发送通道20250901

	}
	return nil
}

// write send messages from the hub to scale.
// A goroutine running write is started for each scale. The
// application ensures that there is at most one writer to a scale by
// executing all writes from this goroutine.
// func (c *Scale) write() {
// 	for {
// 		select {
// 		case message, ok := <-c.toScaleMsgCh:
// 			if !ok {
// 				// The hub closed the channel.
// 				return
// 			}
// 			// send message to scale
// 			c.MySerial.Write([]byte(message))
// 		default:
// 			time.Sleep(time.Microsecond * 100)
// 			continue
// 		}
// 	}
// }

func GetToScaleCmd(scaleCat m.ScaleCat, cmdType SReqType) []byte {
	switch scaleCat {

	}

	return nil
}

func Req2CmdForC51(req SReqType) ([]byte, error) {
	fmt.Println("Req2CmdForC51")
	return nil, nil
}

// // 打开工厂模式
// func enFacModeCmdT2200() []byte {
//     // 构建包头
//     packet := make([]byte, OPEN_FAC_CHUNK_SIZE)
//     binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD)

//     // 构建命令ID与命令类型
//     packet[2] = 0x05
//     packet[3] = 0xF1

//     // 添加包尾
//     binary.BigEndian.PutUint16(packet[OPEN_FAC_CHUNK_SIZE-2:], PACKET_TAIL)

//     return packet
// }

// // 打开工厂模式
// func disFacModeCmdT2200() []byte {
//     // 构建包头
//     packet := make([]byte, OPEN_FAC_CHUNK_SIZE)
//     binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD)

//     // 构建命令ID与命令类型
//     packet[2] = 0x05
//     packet[3] = 0xF2

//     // 添加包尾
//     binary.BigEndian.PutUint16(packet[OPEN_FAC_CHUNK_SIZE-2:], PACKET_TAIL)

//     return packet
// }

// // 擦除原本秤上的打印格式
// func eraseCmdT2200(addr uint32) []byte { // erase size will 2K
//     // 构建包头
//     packet := make([]byte, EARSE_CHUNK_SIZE)
//     binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD)
//     // 构建命令ID与命令类型
//     packet[2] = CMD_ERASE
//     packet[3] = CMD_FLASH

//     // 构建地址
//     binary.BigEndian.PutUint32(packet[4:8], addr)

//     // 擦除长度
//     binary.BigEndian.PutUint16(packet[8:10], CMD_ERASE_SIZE)

//     // 计算与添加校验码
//     checksum := utils.Crc32MPEG2(packet[2 : EARSE_CHUNK_SIZE-6])
//     binary.BigEndian.PutUint32(packet[EARSE_CHUNK_SIZE-6:], checksum)

//     // 添加包尾
//     binary.BigEndian.PutUint16(packet[EARSE_CHUNK_SIZE-2:], PACKET_TAIL)

//     return packet
// }

// func wrDataCmdT2200(addr uint32, data []byte) []byte {
//     // 构建包头
//     packet := make([]byte, FILE_CHUNK_SIZE)
//     binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD)

//     // 构建命令ID与命令类型
//     packet[2] = CMD_IDENTIFY
//     packet[3] = CMD_TYPE

//     // 构建地址
//     binary.BigEndian.PutUint32(packet[4:8], addr)

//     // 构建数据长度
//     binary.BigEndian.PutUint16(packet[8:10], uint16(len(data)))

//     // 复制数据
//     copy(packet[10:], data)

//     // 计算与添加校验码
//     checksum := utils.Crc32MPEG2(packet[2 : FILE_CHUNK_SIZE-6])
//     binary.BigEndian.PutUint32(packet[FILE_CHUNK_SIZE-6:], checksum)

//     // 添加包尾
//     binary.BigEndian.PutUint16(packet[FILE_CHUNK_SIZE-2:], PACKET_TAIL)

//     return packet
// }

// func Req2CmdForT2200(req SReqType, reqData string) ([]byte, error) {
//     fmt.Println("Req2CmdForT2200")
//     switch req {
//     // case SREQ_EN_FAC_MODE:
//     //     return enFacModeCmdT2200(), nil
//     // case SREQ_DIS_FAC_MODE:
//     //     return disFacModeCmdT2200(), nil
//     // case SREQ_ERASE_FLASH:
//     //     // TODO: reqData: format sequence 0-3
//     //     seqNo, err := strconv.Atoi(reqData)
//     //     if err != nil {
//     //         return nil, err
//     //     }
//     //     return eraseCmdT2200(uint32(T2200_PRN_FMT_BASE_ADDR + seqNo*CMD_ERASE_SIZE)), nil
//     // case SREQ_READ_FLASH:
//     //     return T2200_READ_FLASH, nil
//     // case SREQ_WRITE_FLASH:
//     //     seqNo, err := strconv.Atoi(reqData[:1])
//     //     if err != nil {
//     //         fmt.Println("Invalid sequence number")
//     //         return nil, err
//     //     }
//     //     offset, err := strconv.Atoi(reqData[2:8])
//     //     if err != nil {
//     //         fmt.Println("Invalid offset number")
//     //         return nil, err
//     //     }

//     //     // Extract byte array
//     //     data, err := hex.DecodeString(reqData[9:])
//     //     if err != nil {
//     //         fmt.Println("Invalid hex string")
//     //         return nil, err
//     //     }
//     // return wrDataCmdT2200(uint32(T2200_PRN_FMT_BASE_ADDR+seqNo*CMD_ERASE_SIZE+offset), data), nil
//     }

//     return nil, nil
// }

// // func Req2CmdForJWP(req SReqType) ([]byte, error) {
// //     fmt.Println("Req2CmdForJWP")
// //     return nil, nil
// // }

// // func Req2CmdForTMAX(req SReqType) ([]byte, error) {
// //     fmt.Println("Req2CmdForTMAX")
// //     return nil, nil
// // }

// 创建ScaleCat的函数映射
// var scaleCatFuncMap map[ScaleCat]func(req SReqType, reqData string) ([]byte, error)
var cmdComposerFuncMap map[m.ScaleCat]m.CmdComposer

// scaleCatFuncMap := make(map[ScaleCat]func(req SReqType, reqData string) ([]byte, error))
// // scaleCatFuncMap[SCALE_C51] = Req2CmdForC51
// scaleCatFuncMap[SCALE_T2200] = Req2CmdForT2200
// scaleCatFuncMap[SCALE_JWP] = Req2CmdForJWP
// scaleCatFuncMap[SCALE_TMAX] = Req2CmdForTMAX

// cmdComposerFuncMap := make(map[m.ScaleCat]func(req SReqType, reqData string) ([]byte, error))
// cmdComposerFuncMap[m.SCALE_T2200] = ComposerT2200

func GetGrossWeight(s *Scale) (float32, error) {
	resp, err := excuteSimpCmd(s, m.CMD_GET_GROSS_WEIGHT, m.GET_GROSS_WEIGHT_RESP, 1)
	if err != nil {
		return 0.0, err
	}
	fVal, err := strconv.ParseFloat(resp.MsgBody.(string), 32)
	return float32(fVal), err
}

func GetNetWeight(s *Scale) (float32, error) {
	resp, err := excuteSimpCmd(s, m.CMD_GET_NET_WEIGHT, m.GET_NET_WEIGHT_RESP, 1)
	if err != nil {
		return 0.0, err
	}
	fVal, err := strconv.ParseFloat(resp.MsgBody.(string), 32)
	return float32(fVal), err
}

func GetTareWeight(s *Scale) (float32, error) {
	resp, err := excuteSimpCmd(s, m.CMD_GET_TARE_WEIGHT, m.GET_TARE_WEIGHT_RESP, 1)
	if err != nil {
		return 0.0, err
	}
	fVal, err := strconv.ParseFloat(resp.MsgBody.(string), 32)
	return float32(fVal), err
}

func GetPreTareWeight(s *Scale) (float32, error) {
	resp, err := excuteSimpCmd(s, m.CMD_GET_PRE_TARE_WEIGHT, m.GET_PRE_TARE_WEIGHT_RESP, 1)
	if err != nil {
		return 0.0, err
	}
	fVal, err := strconv.ParseFloat(resp.MsgBody.(string), 32)
	return float32(fVal), err
}

func GetWeightUnitCmd(s *Scale) (int, error) {
	resp, err := excuteSimpCmd(s, m.CMD_GET_WEIGHT_UNIT, m.GET_WEIGHT_UNIT_RESP, 1)
	if err != nil {
		return 0, err
	}
	unit, err := strconv.Atoi(resp.MsgBody.(string))
	return unit, err
}

func init() {
	cmdComposerFuncMap = make(map[m.ScaleCat]m.CmdComposer)
	cmdComposerFuncMap[m.SCALE_C51] = *mcmd.NewComposerC51()
	cmdComposerFuncMap[m.SCALE_T2200] = *mcmd.NewComposerT2200()
	cmdComposerFuncMap[m.SCALE_TMAX] = *mcmd.NewComposerTMAX()
}
