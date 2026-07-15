package svc

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"tmaxsrv/cmd"
	m "tmaxsrv/comm"
	"tmaxsrv/eeprom"
	l "tmaxsrv/log"
	"tmaxsrv/picker"
	"tmaxsrv/prnfmt"
	utils "tmaxsrv/util"
)

const (
	DATA_LENGTH_512_TMAX = 512
	DATA_LENGTH_256_TMAX = 256
	DATA_LENGTH_8_TMAX   = 8
)

var GlastWantRespMsgType m.RespMsgType = m.NO_RESP

const RECV_MAX_BUF_LEN = 2048

const MD5SEED = "*T-Scale*"

const (
	SCALE_RECV_CH_SIZE = 16
	SCALE_SEND_CH_SIZE = 16
)

const (
	FCX_ADDR            = 0x1001DC00
	FCX_MODEL_NAME_ADDR = FCX_ADDR                   //8字节
	FCX_SN_ADDR         = FCX_MODEL_NAME_ADDR + 0x08 //16字节
	FCX_MD5_ADDR        = FCX_SN_ADDR + 0x10         //8字节
	FCX_TAIL_ADDR       = FCX_ADDR + 0x200 - 0x08    //尾巴8字节

	FCX_DEF_FLASH_ADDR = 0x1001DE00 //默认参数位置
)

var WRITE_FACTORY_TAIL_CMD_TMAX []byte = []byte{0xff, 0xff, 0xff, 0xff, 0x5a, 0xa5, 0xa5, 0x5a}

const (
	FCX_MODEL_NAME_ID = 1
	FCX_SN_ID         = 2
	FCX_MD5_ID        = 3
)

const (
	GET_WEIGHT_CMD string = "W\r\n"
	TARE_CMD       string = "T\r\n"
	ZERO_CMD       string = "Z\r\n"
)

const (
	SCALE_TIME_OUT_S = 3 // 3 seconds
)

const (
	WIFI_BT_INFO            = "wifi_or_bt"
	WIFI_BT_INFO_FIELD_NAME = "M_WIRELESS_PERIPHERAL_TYPE"
)

type APInfo struct {
	SeqNo       int    `json:"seqno"`
	Ssid        string `json:"ssid"`
	Rssi        int    `json:"rssi"`
	Mac         string `json:"mac"`
	EncryptType string `json:"security"`
}

type WifiInfo struct {
	Ssid  string `json:"ssid"`
	Rssi  int    `json:"rssi"`
	Bssid string `json:"mac"`
}

type MsgBuf struct {
	CurIdx int
	Buf    [RECV_MAX_BUF_LEN]byte
}

type TAddress struct {
	Address string `json:"address"`
	Mask    int    `json:"mask"`
	Proto   string `json:"proto"`
	Family  string `json:"family"`
}

type NetworkConfig struct {
	Addresses []TAddress `json:"address"`
	Gateway   string     `json:"gateway"`
	DNS       []string   `json:"dns"`
	MAC       string     `json:"mac"`
}

type ScaleIsOnlineInfo struct {
	ScaleId   int64  `json:"scaleId"`
	IsOnline  bool   `json:"isOnline"`
	ModelName string `json:"modelName"`
	Sn        string `json:"sn"`
}

type Scale struct {
	scaleMgr *ScaleMgr
	// connectivity Media interface
	Conn *ScaleConnMedia
	// media config, this is a json string that will be unmarshaled to specific structure typed value
	// MediaConf string
	// com port
	MySerial *TSerial
	MyNet    *TNet
	// Network socket
	// tcpSocket net.Socket
	MyBluetooth *TBluetooth
	// btConn BtCom
	// scale Id
	Id int64
	// scale Model, if not supported then the default Model is "legacy"
	Model string
	// scale serial number, if not supported then the default serial number is "123456789"
	ScaleCat m.ScaleCat // scale type: C51, T2200, JWP, TMAX
	Sn       string
	// 最新收到的重量，用于 Modbus 网关查询
	LastWeight float32
	// 用于 Modbus S15 标定
	ModbusCalWeightMSB uint16
	ModbusCalWeight    float32
	// 用于 Modbus S15 预扣重
	ModbusPreTareMSB uint16
	// send to scale channel, message will be json string
	toScaleMsgCh chan string
	// receive from scale channel, message will be json string
	fromScaleMsgCh chan string
	// added time
	EnterAt      time.Time
	respChansMap map[m.RespMsgType][]chan *ScaleRespMsg

	// recvMsgBufsMap   map[RespMsgType][]MsgBuf // should be removed
	bufs utils.RingBuffers
	//isOldC51Scale bool
	// if client needs unsolicited data from scale
	isSendUnolicitedData bool
	// to scale command is issued and waiting response
	isWaintingResp bool
	// client that will communicate with scale
	client *Client
	// this channel is to inform the procScaleRespMessage goroutine, which process the data from serial port, to quit
	quitProcScaleRespMessageCh chan bool
	// this channel is to inform the procToScaleMsg, which process the data from serial port, to quit
	quitProcToScaleMsgCh chan bool

	// function pointer to handle message from scale
	// composer object
	composer *m.CmdComposer
	Pcnf     ComInfo
	Ncnf     NetInfo
	BtInfo   BtInfo

	isScalePassth    bool
	IsScalePassthHex bool
	detailInfo       DetailList
	packDetailMid    []PackDetailMidData
	mu               sync.Mutex
	closed           bool
}

// NewScale creates a new scale
func NewScale(scaleMgr *ScaleMgr, conn *ScaleConnMedia, scaleCat m.ScaleCat, model string, sn string, isTest bool) (*Scale, error) {
	var pcnf ComInfo
	var sport *TSerial
	var net *TNet
	var ncnf NetInfo
	var scale *Scale
	var detailInfo DetailList
	var packDetailMid []PackDetailMidData

	var btInfo BtInfo
	var bt *TBluetooth

	switch conn.TMedia {
	case MEDIA_COM:
		// open COM connection
		if err := json.Unmarshal([]byte(conn.MediaConf.MediaInfoJson), &pcnf); err != nil {
			l.Log.Errorf("error unmarshalling: %v", err)
		}

		picker := picker.GetPickerFn(scaleCat)
		sport, _ = NewSerial(pcnf, picker, conn.IsDefault)

		scale = &Scale{
			scaleMgr: scaleMgr, Conn: conn, ScaleCat: scaleCat, Model: model, Sn: sn, toScaleMsgCh: make(chan string, SCALE_SEND_CH_SIZE),
			fromScaleMsgCh: make(chan string, SCALE_RECV_CH_SIZE), MySerial: sport, Pcnf: pcnf,
			quitProcScaleRespMessageCh: make(chan bool, 1), quitProcToScaleMsgCh: make(chan bool, 1),
			detailInfo:    detailInfo,
			packDetailMid: packDetailMid,
		}

	case MEDIA_NET:
		if err := json.Unmarshal([]byte(conn.MediaConf.MediaInfoJson), &ncnf); err != nil {
			l.Log.Errorf("error unmarshalling: %v", err)
		}

		picker := picker.GetPickerFn(scaleCat)
		net, _ = NewNet(ncnf, picker, conn.IsDefault)
		if net != nil && (conn.ProtocolName == "Modbus TCP" || conn.ProtocolName == "ModbusTCP" || conn.ProtocolName == "TCP") {
			net.IsModbusTCP = true
		}
		scale = &Scale{
			scaleMgr: scaleMgr, Conn: conn, ScaleCat: scaleCat, Model: model, Sn: sn, toScaleMsgCh: make(chan string, SCALE_SEND_CH_SIZE),
			fromScaleMsgCh: make(chan string, SCALE_RECV_CH_SIZE), MyNet: net, Ncnf: ncnf,
			quitProcScaleRespMessageCh: make(chan bool, 1), quitProcToScaleMsgCh: make(chan bool, 1),
			detailInfo:    detailInfo,
			packDetailMid: packDetailMid,
		}

	case MEDIA_BT:
		if err := json.Unmarshal([]byte(conn.MediaConf.MediaInfoJson), &btInfo); err != nil {
			l.Log.Errorf("error unmarshalling: %v", err)
		}
		picker := picker.GetPickerFn(scaleCat)
		bt, _ = NewBluetoothConnection(btInfo.Mac, picker, conn.IsDefault, scaleMgr.bluetoothMgr.adapter)
		scale = &Scale{
			scaleMgr: scaleMgr, Conn: conn, ScaleCat: scaleCat, Model: model, Sn: sn, toScaleMsgCh: make(chan string, SCALE_SEND_CH_SIZE),
			fromScaleMsgCh: make(chan string, SCALE_RECV_CH_SIZE), MyBluetooth: bt, BtInfo: btInfo,
			quitProcScaleRespMessageCh: make(chan bool, 1), quitProcToScaleMsgCh: make(chan bool, 1),
			detailInfo:    detailInfo,
			packDetailMid: packDetailMid,
		}

	}

	scale.respChansMap = map[m.RespMsgType][]chan *ScaleRespMsg{}
	composer := cmdComposerFuncMap[scale.ScaleCat]
	scale.composer = &composer

	var respTypes []string
	for _, respType := range utils.CmdsRespMap {
		respTypes = append(respTypes, string(respType))
	}

	// create buffer for each message
	scale.bufs = utils.NewRingBuffers(respTypes...)

	responseChannels := make(map[m.RespMsgType]chan interface{})

	for _, respType := range utils.CmdsRespMap {
		if _, ok := responseChannels[respType]; !ok {
			responseChannels[respType] = make(chan interface{})
		}
	}

	// example usage: send a message to the WEIGHT_DATA_RESP channel
	// weightDataRespCh := responseChannels[WEIGHT_DATA_RESP]
	// weightDataRespCh <- "some message"

	scale.isSendUnolicitedData = false
	scale.isWaintingResp = false
	scale.isScalePassth = false
	scale.IsScalePassthHex = false

	scale.scaleMgr = scaleMgr
	scale.EnterAt = time.Now()
	switch conn.TMedia {
	case MEDIA_COM:
		scale.MySerial = sport
		go scale.keepSerialPortState()
	case MEDIA_NET:
		scale.MyNet = net
		go scale.keepNetState(scale.MyNet)
		// scale.keepNetState(scale.MyNet)
	}

	go scale.procScaleRespMsg()
	go scale.procToScaleMsg()
	if scale.ScaleCat == m.SCALE_C51 {
		go scale.setC51Offline()
	}

	return scale, nil
}

func (s *Scale) setC51Offline() {
	for {
		s.Conn.IsOnline = false
		time.Sleep(5 * time.Second)
	}
}

func (s *Scale) keepSerialPortState() {
	cont := 1
	isReconnecting := false // 添加一个标记表示是否正在重连

	for {
		if s.MySerial == nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if s.MySerial.toQuit {
			time.Sleep(5 * time.Second)
			continue
		}
		if !s.Conn.IsOnline {

			if !isReconnecting {
				cont = 0
				time.Sleep(5000 * time.Millisecond)

			} else {
				// 正在重连过程中，跳过本次循环
				time.Sleep(5 * time.Second)
				continue
			}
		}
		if s.Conn.IsOnline && cont == 0 {
			isReconnecting = true
			req := ReqModifyScale{ScaleId: s.Id, MediaConf: s.Conn.MediaConf, ScaleModel: s.Conn.ScaleModel}
			s.scaleMgr.UpdateScale(req)
			isReconnecting = false
			cont = 1

			continue
		}
		if s.Conn.IsOnline {
			cont = 1
			time.Sleep(5000 * time.Millisecond)
			continue
		}
	}
}

func (s *Scale) keepNetState(myNet *TNet) {
	for {
		if myNet == nil {
			break
		}
		// 用户要求退出
		if myNet.toQuit {
			if myNet.conn != nil {
				myNet.conn.Close()
			}
			break
		}

		// [核心逻辑] 如果当前是存活状态，休眠并继续检测
		if myNet.isAlive && myNet.conn != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// [核心逻辑] 如果代码运行到这里，说明断开了。
		// 1. 发送离线通知给界面
		sendScaleOnlineToUi(s, false, "", "")

		// 2. 检查连接状态是否需要清理
		if myNet.conn != nil {
			myNet.conn.Close()
			myNet.conn = nil
		}
		myNet.isAlive = false

		// 3. 尝试重连
		fmt.Printf("Attempting to reconnect: %v\n", myNet.ip)
		var err error
		myNet.conn, err = myNet.reconnect()
		if err == nil {
			myNet.isAlive = true
			fmt.Printf("Reconnect successful: %v\n", myNet.ip)
			// 重连成功，同步一次在线状态
			sendScaleOnlineToUi(s, true, s.Conn.ScaleModel, s.Sn)
		} else {
			// 连不上则等待 2 秒后再试
			time.Sleep(2 * time.Second)
		}
	}
}

// 移除 keepNetOnline 函数，因为不再需要通过指令获取 SN 和型号

func sendScaleOnlineToUi(s *Scale, isOnline bool, modelName string, sn string) {
	sta := &ScaleIsOnlineInfo{ScaleId: s.Conn.ScaleId, IsOnline: isOnline, ModelName: modelName, Sn: sn}
	recsStr, _ := json.MarshalToString(sta)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_ONLINE, MsgBody: recsStr}
}

func (s *Scale) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	close(s.quitProcScaleRespMessageCh)
	close(s.quitProcToScaleMsgCh)
	if s.MySerial != nil {
		s.MySerial.Close()
	}

	if s.MyNet != nil {
		s.MyNet.Close() //20240801
	}

	if s.MyBluetooth != nil {
		s.MyBluetooth.Close() //20260206
	}
	return nil
}

func (s *Scale) SetClient(client *Client) error {
	s.client = client
	return nil
}

func (s *Scale) HandleClientDisconnect() error {
	l.Log.Warn("Client disconnected, HandleClientDisconnect called")

	s.client = nil

	return nil
}

// to process msg from serial port
func (s *Scale) procScaleRespMsg() {
	quit := false
	if s.MySerial != nil {
		for {
			if quit {
				break
			}
			if s.MySerial == nil {
				time.Sleep(time.Microsecond * 100)
				continue
			}
			select {
			case <-s.quitProcScaleRespMessageCh:
				quit = true
			case inPack := <-s.MySerial.recvCh:
				if inPack.PayloadLen == 0 {
					continue
				}
				l.Log.Debugf("From sport: %v", inPack)

				if s.ScaleCat == m.SCALE_C51 {

					s.Conn.IsOnline = true

					protocolName := ""
					if s.Conn != nil {
						protocolName = s.Conn.ProtocolName
					}
					l.Log.Debugf("DispatchProtocolParser - ProtocolName is: '%s' for scale Id: %d", protocolName, s.Id)
					msg, err := DispatchProtocolParser(protocolName, s.Id, inPack.Payload)
					if err != nil {
						continue
					}
					sendMsgIntoChsOrWeightToClient(s, msg)
				} else if s.ScaleCat == m.SCALE_T2200 {
					msg, err := retreiveRespMsgT2200(s.Id, inPack.Payload)
					if err != nil {
						continue
					}
					sendMsgIntoChsOrWeightToClient(s, msg)
				} else if s.ScaleCat == m.SCALE_TMAX || s.ScaleCat == m.SCALE_TMAX_PASSTH {
					// find message buffer that associate to the message

					if s.isScalePassth {
						var cmdHex uint16 = cmd.CMDID_SCALE_PASSTH_DATA_TMAX
						bufName := utils.CmdsRespMap[utils.CmdID(cmdHex)]
						if bufName == "" {
							l.Log.Errorf("error on getting bufName for the cmd: %v", cmdHex)
							continue
						}

						fmt.Println(bufName)
						s.bufs.Write(string(bufName), inPack.Payload)
						msg := extractScalePassthDataTMAX(s, s.bufs[string(bufName)], bufName)
						sendMsgIntoChsOrWeightToClient(s, &msg) //20231023  @111
						continue
					}
					var cmdHex uint16 = (uint16(inPack.CmdID) << 8) | uint16(inPack.CmdSubId)
					bufName := utils.CmdsRespMap[utils.CmdID(cmdHex)]
					if bufName == "" {
						l.Log.Errorf("error on getting bufName for the cmd: %v", cmdHex)
						continue
					}

					fmt.Println(bufName)
					s.bufs.Write(string(bufName), inPack.Payload)
					msg := extractMessageTMAX(s.Id, s.bufs[string(bufName)], bufName)
					sendMsgIntoChsOrWeightToClient(s, &msg) //20231023  @111
				} else {

				}
			default:
				time.Sleep(time.Microsecond * 100)
				continue
			}
		}

	} else if s.MyNet != nil {
		for {
			if quit {
				break
			}
			select {
			case <-s.quitProcScaleRespMessageCh:
				quit = true
			case inPack := <-s.MyNet.recvCh:
				if inPack.PayloadLen == 0 {
					continue
				}
				// l.Log.Debugf("From net: %v", inPack)

				if s.ScaleCat == m.SCALE_C51 {

					s.Conn.IsOnline = true

					protocolName := ""
					if s.Conn != nil {
						protocolName = s.Conn.ProtocolName
					}
					msg, err := DispatchProtocolParser(protocolName, s.Id, inPack.Payload)
					if err != nil {
						continue
					}
					sendMsgIntoChsOrWeightToClient(s, msg)
				} else if s.ScaleCat == m.SCALE_T2200 {
					msg, err := retreiveRespMsgT2200(s.Id, inPack.Payload)
					if err != nil {
						continue
					}
					sendMsgIntoChsOrWeightToClient(s, msg)
				} else if s.ScaleCat == m.SCALE_TMAX || s.ScaleCat == m.SCALE_TMAX_PASSTH {
					// find message buffer that associate to the message

					if s.isScalePassth {
						var cmdHex uint16 = cmd.CMDID_SCALE_PASSTH_DATA_TMAX
						bufName := utils.CmdsRespMap[utils.CmdID(cmdHex)]
						if bufName == "" {
							l.Log.Errorf("error on getting bufName for the cmd: %v", cmdHex)
							continue
						}

						// fmt.Println(bufName)
						s.bufs.Write(string(bufName), inPack.Payload)
						msg := extractScalePassthDataTMAX(s, s.bufs[string(bufName)], bufName)
						sendMsgIntoChsOrWeightToClient(s, &msg) //20231023  @111
						continue
					}
					var cmdHex uint16 = (uint16(inPack.CmdID) << 8) | uint16(inPack.CmdSubId)
					bufName := utils.CmdsRespMap[utils.CmdID(cmdHex)]
					if bufName == "" {
						l.Log.Errorf("error on getting bufName for the cmd: %v", cmdHex)
						continue
					}
					fmt.Println(bufName)
					s.bufs.Write(string(bufName), inPack.Payload)
					msg := extractMessageTMAX(s.Id, s.bufs[string(bufName)], bufName)
					sendMsgIntoChsOrWeightToClient(s, &msg) //20231023  @111
				} else {

				}
			default:
				time.Sleep(time.Microsecond * 100)
				continue
			}
		}

	} else if s.MyBluetooth != nil {
		for {
			if quit {
				break
			}
			select {
			case <-s.quitProcScaleRespMessageCh:
				quit = true
			case inPack := <-s.MyBluetooth.recvCh:
				if inPack.PayloadLen == 0 {
					continue
				}
				// l.Log.Debugf("From net: %v", inPack)

				if s.ScaleCat == m.SCALE_C51 {

					s.Conn.IsOnline = true

					protocolName := ""
					if s.Conn != nil {
						protocolName = s.Conn.ProtocolName
					}
					msg, err := DispatchProtocolParser(protocolName, s.Id, inPack.Payload)
					if err != nil {
						continue
					}
					sendMsgIntoChsOrWeightToClient(s, msg)
				} else if s.ScaleCat == m.SCALE_T2200 {
					msg, err := retreiveRespMsgT2200(s.Id, inPack.Payload)
					if err != nil {
						continue
					}
					sendMsgIntoChsOrWeightToClient(s, msg)
				} else if s.ScaleCat == m.SCALE_TMAX || s.ScaleCat == m.SCALE_TMAX_PASSTH {
					// find message buffer that associate to the message

					if s.isScalePassth {
						var cmdHex uint16 = cmd.CMDID_SCALE_PASSTH_DATA_TMAX
						bufName := utils.CmdsRespMap[utils.CmdID(cmdHex)]
						if bufName == "" {
							l.Log.Errorf("error on getting bufName for the cmd: %v", cmdHex)
							continue
						}

						// fmt.Println(bufName)
						s.bufs.Write(string(bufName), inPack.Payload)
						msg := extractScalePassthDataTMAX(s, s.bufs[string(bufName)], bufName)
						sendMsgIntoChsOrWeightToClient(s, &msg) //20231023  @111
						continue
					}
					var cmdHex uint16 = (uint16(inPack.CmdID) << 8) | uint16(inPack.CmdSubId)
					bufName := utils.CmdsRespMap[utils.CmdID(cmdHex)]
					if bufName == "" {
						l.Log.Errorf("error on getting bufName for the cmd: %v", cmdHex)
						continue
					}
					fmt.Println(bufName)
					s.bufs.Write(string(bufName), inPack.Payload)
					msg := extractMessageTMAX(s.Id, s.bufs[string(bufName)], bufName)
					sendMsgIntoChsOrWeightToClient(s, &msg) //20231023  @111
				} else {

				}
			default:
				time.Sleep(time.Microsecond * 100)
				continue
			}
		}

	}

}

func (s *Scale) procToScaleMsg() {
	quit := false
	for {
		if quit {
			break
		}
		if s.client == nil {
			time.Sleep(10 * time.Millisecond) // to avoid consume too many cpu resource
			continue
		}
		select {
		case <-s.quitProcToScaleMsgCh:
			quit = true
		case userMessage, ok := <-s.client.recvCh:
			if !ok {
				l.Log.Error("client's recvCh closed")
				continue
			}
			var data map[string][]byte
			json.Unmarshal(userMessage, &data)
			l.Log.Debugf("userMessage: %v", data)
			scaleId := new(big.Int).SetBytes(data["scaleId"]).Int64()
			if scaleId != s.Id {
				l.Log.Errorf("wrong scale id :%v received, our Id is: %v", scaleId, s.Id)
				continue
			}
			if data == nil { // TODO: maybe caused by ...client?
				continue
			}
			l.Log.Debugf("From wsclient: %v", string(data["message"]))
			if req, err := parseToScaleReq(string(data["message"])); err == nil {
				go procToScaleReq(s, req) // TODO: handle error
			}
		default:
			time.Sleep(time.Microsecond * 100)
			continue
		}
	}
}

func (s *Scale) ModifyMedia(conf MediaConf) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	switch conf.Type {
	case MEDIA_COM:
		var pcnf ComInfo
		var err error
		// open COM connection
		if err := json.Unmarshal([]byte(conf.MediaInfoJson), &pcnf); err != nil {
			l.Log.Errorf("error unmarshalling: %v", err)
		}
		var pickFun picker.PickerFunc = nil
		if s.MySerial != nil {
			select {
			case s.quitProcScaleRespMessageCh <- true:
			default:
			}
			select {
			case s.quitProcToScaleMsgCh <- true:
			default:
			}
			pickFun = s.MySerial.pickerFn
			s.MySerial.Close()
			s.MySerial = nil
		} else {
			pickFun = picker.GetPickerFn(s.ScaleCat)
		}
		// 释放锁再进行休眠
		s.mu.Unlock()
		time.Sleep(1 * time.Second)
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return false
		}

		if s.MySerial, err = NewSerial(pcnf, pickFun, true); err != nil {
			l.Log.Error(err.Error())
		}
		s.Pcnf = pcnf
	case MEDIA_NET:

		var ncnf NetInfo
		var err error
		if err := json.Unmarshal([]byte(conf.MediaInfoJson), &ncnf); err != nil {
			l.Log.Errorf("error unmarshalling: %v", err)
		}
		var pickFun picker.PickerFunc = nil
		if s.MyNet != nil {
			select {
			case s.quitProcScaleRespMessageCh <- true:
			default:
			}
			select {
			case s.quitProcToScaleMsgCh <- true:
			default:
			}
			pickFun = s.MyNet.pickerFn
			s.MyNet.Close()
			s.MyNet = nil
			s.mu.Unlock()
			time.Sleep(500 * time.Millisecond)
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				return false
			}

		} else {
			pickFun = picker.GetPickerFn(s.ScaleCat)
		}
		s.mu.Unlock()
		time.Sleep(500 * time.Millisecond)
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return false
		}
		if s.MyNet, err = NewNet(ncnf, pickFun, true); err != nil {
			l.Log.Error(err.Error())
		} else if s.MyNet != nil && (s.Conn.ProtocolName == "Modbus TCP" || s.Conn.ProtocolName == "ModbusTCP" || s.Conn.ProtocolName == "TCP") {
			s.MyNet.IsModbusTCP = true
		}
		time.Sleep(500 * time.Millisecond)
		s.Ncnf = ncnf
		go s.keepNetState(s.MyNet)

	case MEDIA_BT:
		//TODO: 要做修改蓝牙    202406
		s.mu.Unlock()
		return false
	default:
		s.mu.Unlock()
		return false

	}
	defer s.mu.Unlock()
	go s.procScaleRespMsg()
	go s.procToScaleMsg()
	return true
}

// 移除不再使用的全局锁
// var mu sync.Mutex

func (c *Scale) RegisterNotif(msgType m.RespMsgType, inCh chan *ScaleRespMsg) {
	addNotif(c, msgType, inCh)
}

func (c *Scale) UnRegisterNotif(msgType m.RespMsgType, inCh chan *ScaleRespMsg) {
	removeNotif(c, msgType, inCh)
}

func addNotif(s *Scale, msgType m.RespMsgType, inCh chan *ScaleRespMsg) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.respChansMap[msgType] = append(s.respChansMap[msgType], inCh)
}

func removeNotif(s *Scale, msgType m.RespMsgType, inCh chan *ScaleRespMsg) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.respChansMap[msgType] = remove(s.respChansMap[msgType], inCh)
}

func remove(s []chan *ScaleRespMsg, m chan *ScaleRespMsg) []chan *ScaleRespMsg {
	for i, msgCh := range s {
		if msgCh == m {
			s[i] = s[len(s)-1]
			return s[:len(s)-1]
		}
	}
	return s
}

func parseToScaleReq(reqStr string) (SRequest, error) {
	var req SRequest
	if err := json.UnmarshalFromString(reqStr, &req); err != nil {
		l.Log.Error(err)
		return SRequest{}, err
	}

	return req, nil
}

func enablePassthrough(s *Scale, respType m.RespMsgType) (*ScaleRespMsg, error) {
	// msg, err := EnFacMode(s)
	// if err != nil {
	// 	return &ScaleRespMsg{respType, "fail", s.Id}, err
	// }
	// if msg.MsgBody != "ok" {
	// 	return &ScaleRespMsg{respType, "fail", s.Id}, nil
	// }
	// time.Sleep(100 * time.Millisecond)
	println(respType)
	msg, err := EnPassthrough(s)

	return msg, err
}

//20250829备份
// func ReqModifyBTName(s *Scale, name string) (*ScaleRespMsg, error) {
// 	reg, err, res1 := openFactory(s)
// 	if err != nil || !res1 {
// 		reg.MsgType = m.MODIFY_BT_NAME_RESP
// 		return reg, err

// 	}
// 	defer DisPassthrough(s)

// 	msg, err := enablePassthrough(s, m.MODIFY_BT_NAME_RESP)
// 	if msg.MsgBody != "ok" {
// 		msg.MsgType = m.MODIFY_BT_NAME_RESP
// 		return msg, err
// 	}
// 	return s.ModifyBTName(name)
// }

func ReqModifyBTName(s *Scale, name string) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.MODIFY_BT_NAME_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.MODIFY_BT_NAME_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.MODIFY_BT_NAME_RESP
		return msg, err
	}

	SaveUpdateBtNameLog(s.Conn.ScaleName, name)

	return s.ModifyBTName(name)
}

func ReqSendDataToBT(s *Scale, data string) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.SEND_DATA_TO_BT_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.SEND_DATA_TO_BT_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.SEND_DATA_TO_BT_RESP
		return msg, err
	}

	SaveUpdateBtPowerLog(s.Conn.ScaleName, data)

	return SendDataToBT(s, data+"\r\n\x00")
}

func ReqSendDataToWifi(s *Scale, data string) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.SEND_DATA_TO_WIFI_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.SEND_DATA_TO_WIFI_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.SEND_DATA_TO_WIFI_RESP
		return msg, err
	}

	return SendDataToWifi(s, data)
}

func ReqGetWifiApInfo(s *Scale) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.GET_WIFI_AP_INFO_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.GET_WIFI_AP_INFO_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.GET_WIFI_AP_INFO_RESP
		return msg, err
	}
	// msg, _ = getAtVersion(s)
	// switch msg.MsgBody {
	// case m.AT_VERSION:
	// 	return GetWifiApInfo32(s)
	// case m.AT_VERSION8266:
	// 	return GetWifiApInfo(s)
	// default:
	// 	return GetWifiApInfo32(s)
	// }

	return GetWifiApInfo32(s)

}

func ReqGetApList(s *Scale) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(s)
	if err != nil || !res {
		reg.MsgType = m.GET_AP_LIST_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.GET_AP_LIST_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.GET_AP_LIST_RESP
		return msg, err
	}

	return GetApList(s)
}

func getAtVersion(s *Scale) (*ScaleRespMsg, error) {
	msg, err := GetWifiAtVersion(s)
	return msg, err
}

func ReqConnectAp(s *Scale, ssid string, bssid string, password string) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.CONNECT_AP_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.CONNECT_AP_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.CONNECT_AP_RESP
		return msg, err
	}
	// msg, _ = getAtVersion(s)

	// switch msg.MsgBody {
	// case m.AT_VERSION:
	// 	res, err := ConnectWifiAp32(s, ssid, password, bssid)
	// 	return res, err
	// case m.AT_VERSION8266:
	// 	res, err := ConnectWifiAp(s, ssid, password, bssid)
	// 	return res, err
	// default:
	// 	res, err := ConnectWifiAp32(s, ssid, password, bssid)
	// 	return res, err
	// }

	res, err := ConnectWifiAp32(s, ssid, password, bssid)
	return res, err

}

func ReqConnectApOneKey(s *Scale, ssid string, password string, bssid string) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.CONNECT_AP_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.CONNECT_AP_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.CONNECT_AP_RESP
		return msg, err
	}

	res, err := ConnectWifiApOneKey(s, ssid, password, bssid)
	return res, err
}

func ReqSetWifiDynamicIp(s *Scale) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(s)
	if err != nil || !res {
		reg.MsgType = m.SET_WIFI_DYNAMIC_IP_RESP
		return reg, err

	}
	defer DisPassthrough(s)

	msg, err := enablePassthrough(s, m.SET_WIFI_DYNAMIC_IP_RESP)

	if msg.MsgBody != "ok" {
		msg.MsgType = m.SET_WIFI_DYNAMIC_IP_RESP
		return msg, err
	}
	// msg, _ = getAtVersion(s)
	// switch msg.MsgBody {
	// case m.AT_VERSION:
	// 	return SetWifiDynamicIp32(s)
	// case m.AT_VERSION8266:
	// 	return SetWifiDynamicIp(s)
	// default:
	// 	return SetWifiDynamicIp32(s)
	// }

	return SetWifiDynamicIp32(s)
}

func ReqSetWifiStaticIp(s *Scale, ip string, gateway string, netmask string) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(s)
	if err != nil || !res {
		reg.MsgType = m.SET_WIFI_STATIC_IP_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.SET_WIFI_STATIC_IP_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.SET_WIFI_STATIC_IP_RESP
		return msg, err
	}
	// msg, _ = getAtVersion(s)
	// switch msg.MsgBody {
	// case m.AT_VERSION:
	// 	return SetWifiStaticIp32(s, ip, gateway, netmask)
	// case m.AT_VERSION8266:
	// 	return SetWifiStaticIp(s, ip, gateway, netmask)
	// default:
	// 	return SetWifiStaticIp32(s, ip, gateway, netmask)
	// }
	return SetWifiStaticIp32(s, ip, gateway, netmask)
}

func ReqGetIpInfo(s *Scale) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(s)
	if err != nil || !res {
		reg.MsgType = m.SET_WIFI_STATIC_IP_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.SET_WIFI_STATIC_IP_RESP)
	if msg.MsgBody != "ok" {
		msg.MsgType = m.SET_WIFI_STATIC_IP_RESP
		return msg, err
	}

	// msg, _ = getAtVersion(s)
	// switch msg.MsgBody {
	// case m.AT_VERSION:
	// 	return GetIpInfo32(s)
	// case m.AT_VERSION8266:
	// 	return GetIpInfo(s)
	// default:
	// 	return GetIpInfo32(s)
	// }
	return GetIpInfo32(s)

}

func getAtMode(s *Scale) (*ScaleRespMsg, error) {
	msg, err := GetWifiAtMode(s)
	return msg, err
}

func ReqChangeWifiMode(s *Scale, req SRequest) (*ScaleRespMsg, error) {

	reg, err, res := openFactory(s)
	if err != nil || !res {
		reg.MsgType = m.CHANGE_WIFI_MODE_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.CHANGE_WIFI_MODE_RESP)

	if msg.MsgBody != "ok" {
		msg.MsgType = m.DIS_BT_CMD_RESP
		return msg, err
	}

	// 如果机种是DPM 就初始化
	// if s.Model == "DPM" {
	// 	ReqInitWifi(s, req)
	// } else {
	// 	msg.MsgType = m.CHANGE_WIFI_MODE_RESP

	// 	//问了模式不对再切换模式
	// 	msg, err = getAtMode(s)
	// 	if err != nil {
	// 		msg.MsgType = m.CHANGE_WIFI_MODE_RESP
	// 		return msg, err
	// 	} else if msg.MsgBody != "ok" {
	// 		return ChangeWifiMode(s)
	// 	}
	// 	msg.MsgType = m.CHANGE_WIFI_MODE_RESP
	// }

	ReqInitWifiAPListRef(s, req)

	msg.MsgType = m.CHANGE_WIFI_MODE_RESP

	//问了模式不对再切换模式
	msg, err = getAtMode(s)
	if err != nil {
		msg.MsgType = m.CHANGE_WIFI_MODE_RESP
		return msg, err
	} else if msg.MsgBody != "ok" {
		return ChangeWifiMode(s)
	}
	msg.MsgType = m.CHANGE_WIFI_MODE_RESP

	return &ScaleRespMsg{m.CHANGE_WIFI_MODE_RESP, "ok", s.Id}, nil

}

func ReqGetIpMode(s *Scale) (*ScaleRespMsg, error) {
	reg, err, res1 := openFactory(s)
	if err != nil || !res1 {
		reg.MsgType = m.GET_IP_MODE_RESP
		return reg, err

	}
	defer DisPassthrough(s)
	msg, err := enablePassthrough(s, m.GET_IP_MODE_RESP)

	if msg.MsgBody != "ok" {
		msg.MsgType = m.GET_IP_MODE_RESP
		return msg, err
	}
	// msg, _ = getAtVersion(s)
	// switch msg.MsgBody {
	// case m.AT_VERSION:
	// 	return GetIpMode32(s)
	// case m.AT_VERSION8266:
	// 	return GetIpMode(s)
	// default:
	// 	return GetIpMode32(s)
	// }
	return GetIpMode32(s)

}
func ReqDownEepromInfo(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	var eepromDataDown EepromDataDownResp
	if err := json.UnmarshalFromString(req.ReqData, &eepromDataDown.EepromDataDown); err != nil {
		return &ScaleRespMsg{}, err
	}
	modifyData := eepromDataDown.EepromDataDown

	print(len(modifyData))
	composer := s.composer
	fn := composer.ComposeCmd
	for i := 0; i < len(modifyData); i++ {
		switch modifyData[i].Type {
		case "string":
			packetData := stringToLittleEndianBytes(modifyData[i].Value, modifyData[i].Size)
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}

			// 发送数据包
			if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
			}
		case "int":
			if modifyData[i].Size == 1 {
				num, err := strconv.Atoi(modifyData[i].Value)
				if err != nil {

					return &ScaleRespMsg{m.DOWN_EEPROM_INFO_RESP, "fail,data error", s.Id}, nil
				}
				byteVal := []byte{byte(num)}
				packDataHexStr := hex.EncodeToString(byteVal)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			} else if modifyData[i].Size == 4 && modifyData[i].SubType == "ip" {
				packetData := ipv4StringToBytes(modifyData[i].Value)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			} else if modifyData[i].Size == 2 || modifyData[i].Size == 4 || modifyData[i].Size == 8 {
				packetData, _ := intToLittleEndianBytes(modifyData[i].Value, modifyData[i].Size)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			}

		case "double":
			switch modifyData[i].Size {
			case 8:
				packetData := StringToFloat64Bytes(modifyData[i].Value)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			case 4:
				packetData := StringToFloat32Bytes(modifyData[i].Value)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			}
		}
	}
	return &ScaleRespMsg{m.DOWN_EEPROM_INFO_RESP, "ok", s.Id}, nil
}

// Tmax 的更新默认参数 bin
func ReqSetEepromFromBin(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	filePath := req.ReqData
	composer := c.composer
	fn := composer.ComposeCmd

	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	dataInfo, err := os.ReadFile(filePath)
	if err != nil {
		l.Log.Errorf("SetEepromFromBin read file error: %v", err)
		return &ScaleRespMsg{}, err
	}

	if len(dataInfo) != 512 {
		return &ScaleRespMsg{}, fmt.Errorf("fail, get def value fail")
	}

	dataInfo[len(dataInfo)-2] = 0x5A
	dataInfo[len(dataInfo)-1] = 0xA5

	packetCount := (512 - 61) / 8
	if (512-61)%8 != 0 {
		packetCount += 1
	}
	addr := 61

	for i := 0; i < packetCount; i++ {
		// 计算本包数据
		start := i*8 + 61
		end := start + 8
		if end > len(dataInfo) {
			end = len(dataInfo)
		}
		packetData := dataInfo[start:end]
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
		// 地址自增
		addr += 0x08
	}

	l.Log.Info("save bin ok")

	return &ScaleRespMsg{m.SET_EEPROM_FROM_BIN_RESP, "ok", c.Id}, nil
}

// 下面是FCX的
func ReqSetEepromFromBinFc(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	filePath := req.ReqData
	composer := c.composer
	fn := composer.ComposeCmd

	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	dataInfo, err := os.ReadFile(filePath)
	if err != nil {
		l.Log.Errorf("SetEepromFromBinFc read file error: %v", err)
		return &ScaleRespMsg{}, err
	}

	if len(dataInfo) != 512 {
		return &ScaleRespMsg{}, fmt.Errorf("fail,get def value fail")
	}

	dataInfo[len(dataInfo)-2] = 0x5A
	dataInfo[len(dataInfo)-1] = 0xA5

	packetCount := (512 - 61) / 8
	if (512-61)%8 != 0 {
		packetCount += 1
	}
	addr := 61

	for i := 0; i < packetCount; i++ {
		// 计算本包数据
		start := i*8 + 61
		end := start + 8
		if end > len(dataInfo) {
			end = len(dataInfo)
		}
		packetData := dataInfo[start:end]
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
		// 地址自增
		addr += 0x08
	}

	l.Log.Info("save bin ok")

	// 备份eeprom 擦除原本秤上的flash  一次擦512  DATA_LENGTH_512_TMAX
	l.Log.Debug("erase flash on scale")
	addr = FCX_DEF_FLASH_ADDR

	addrInLoop := addr

	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH_512, m.CmdData{Type: m.DATA_TYPE_INT, Data: addrInLoop})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("erase fail")
	}

	// 计算数据包数量
	packetCount = len(dataInfo) / DATA_LENGTH_8_TMAX
	if len(dataInfo)%DATA_LENGTH_8_TMAX != 0 {
		packetCount += 1
	}
	// 遍历所有数据包
	l.Log.Debug("send data package to scale")
	for i := 0; i < packetCount; i++ {
		// 计算本包数据
		start := i * DATA_LENGTH_8_TMAX
		end := start + DATA_LENGTH_8_TMAX
		if end > len(dataInfo) {
			end = len(dataInfo)
		}
		packetData := dataInfo[start:end]

		// 构建数据包
		// dataPackCmd := buildSendDataPacket(addr, packetData)
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_8, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
		}
		// 地址自增
		addr += 0x08
	}
	l.Log.Info("send bin ok")

	return &ScaleRespMsg{m.SET_EEPROM_FROM_BIN_RESP, "ok", c.Id}, nil
}

func ReqGetEepromInfoToBin(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	filePath := req.ReqData

	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}

	dataInfo, _ := getEepromData(c, 512)
	if len(dataInfo) == 512 {
		//前面61个字节不写
		for i := 0; i < 61; i++ {
			dataInfo[i] = 0xff
		}
		if !saveByteToBin(filePath, dataInfo) {
			return &ScaleRespMsg{}, fmt.Errorf("fail,write bin fail")
		}

		l.Log.Info("save bin ok")
	} else {
		return &ScaleRespMsg{}, fmt.Errorf("fail,get def value fail")
	}

	return &ScaleRespMsg{m.GET_EEPROM_TO_BIN_RESP, "ok", c.Id}, nil
}

func saveByteToBin(fileName string, tempBuf []byte) bool {
	fp, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer fp.Close()
	fp.Write(tempBuf)
	return true
}

func ReqDownFactoryInfoTmax(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	var factoryDataDown FactoryDataDownResp
	if err := json.UnmarshalFromString(req.ReqData, &factoryDataDown.FactoryDataDown); err != nil {
		return &ScaleRespMsg{}, err
	}
	reg, err, res := openFactory(s)
	if err != nil || !res {
		return reg, err
	}

	modifyData := factoryDataDown.FactoryDataDown

	print(len(modifyData))
	composer := s.composer
	fn := composer.ComposeCmd

	addrInfo, res := getAddrFromScale(s, SI_SN_INFO)
	if !res {
		return &ScaleRespMsg{}, fmt.Errorf("get print address fail")
	}

	tmax_Addr := addrInfo.Addr

	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH_512, m.CmdData{Type: m.DATA_TYPE_INT, Data: tmax_Addr})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	if res, err := perfCmdNwaitResult(s, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("erase fail")
	}

	tmaxNameAddr := tmax_Addr
	tmaxSnAddr := tmaxNameAddr + 0x08
	tmaxMd5Addr := tmaxSnAddr + 0x10
	tmaxTailAddr := tmax_Addr + 0x200 - 0x08
	tmaxTailAddrFactory := tmax_Addr + addrInfo.Lenth - 0x08 //工厂校验地址

	modelNameSnStr := ""
	for i := 0; i < len(modifyData); i++ {
		if modifyData[i].Type == "string" {
			packetData := fillStrBySize(modifyData[i].Value, modifyData[i].Size)
			packDataHexStr := hex.EncodeToString(packetData)
			var addr = tmax_Addr
			switch modifyData[i].Id {
			case FCX_MODEL_NAME_ID:
				addr = tmaxNameAddr
				modelNameSnStr = modelNameSnStr + modifyData[i].Value
			case FCX_SN_ID:
				addr = tmaxSnAddr
				modelNameSnStr = modelNameSnStr + modifyData[i].Value
			case FCX_MD5_ID:
				addr = tmaxMd5Addr
			default:
				return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
			}
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}

			// 发送数据包
			if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
			}
		}
	}
	//计算MD5值
	if len(modelNameSnStr) > 0 {
		modelNameSnStr = modelNameSnStr + MD5SEED
		crc16Byte := calculateMD5(modelNameSnStr)
		byte4Md5 := crc16Byte[0:4]
		result := make([]byte, 0, 8)
		result = append(result, byte4Md5[:4]...)
		if len(result) < 8 {
			result = append(result, bytes.Repeat([]byte{0xFF}, 8-len(result))...)
		}
		packetData := result
		packDataHexStr := hex.EncodeToString(packetData)
		addr := tmaxMd5Addr
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
		}

	}
	//写校验  秤上说是老版本校验
	packetData := WRITE_FACTORY_TAIL_CMD_TMAX
	packDataHexStr := hex.EncodeToString(packetData)
	addr := tmaxTailAddr
	cmd, timeoutMs, err = fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	// 发送数据包
	if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
	}

	// 秤上说是工厂校验

	addr = tmaxTailAddrFactory
	cmd, timeoutMs, err = fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	// 发送数据包
	if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
	}

	return &ScaleRespMsg{m.DOWN_FACTORY_INFO_RESP, "ok", s.Id}, nil
}

func ReqDownFactoryInfoFc(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	var factoryDataDown FactoryDataDownResp
	if err := json.UnmarshalFromString(req.ReqData, &factoryDataDown.FactoryDataDown); err != nil {
		return &ScaleRespMsg{}, err
	}
	reg, err, res := openFactory(s)
	if err != nil || !res {
		return reg, err
	}

	modifyData := factoryDataDown.FactoryDataDown

	print(len(modifyData))
	composer := s.composer
	fn := composer.ComposeCmd

	eraseAddr := FCX_ADDR

	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH_512, m.CmdData{Type: m.DATA_TYPE_INT, Data: eraseAddr})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	if res, err := perfCmdNwaitResult(s, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("erase fail")
	}

	modelNameSnStr := ""
	for i := 0; i < len(modifyData); i++ {
		if modifyData[i].Type == "string" {
			packetData := fillStrBySize(modifyData[i].Value, modifyData[i].Size)
			packDataHexStr := hex.EncodeToString(packetData)
			var addr = 0
			switch modifyData[i].Id {
			case FCX_MODEL_NAME_ID:
				addr = FCX_MODEL_NAME_ADDR
				modelNameSnStr = modelNameSnStr + modifyData[i].Value
			case FCX_SN_ID:
				addr = FCX_SN_ADDR
				modelNameSnStr = modelNameSnStr + modifyData[i].Value
			case FCX_MD5_ID:
				addr = FCX_MD5_ADDR
			default:
				return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
			}
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}

			// 发送数据包
			if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
			}
		}
	}
	//计算MD5值
	if len(modelNameSnStr) > 0 {
		modelNameSnStr = modelNameSnStr + MD5SEED
		crc16Byte := calculateMD5(modelNameSnStr)
		byte4Md5 := crc16Byte[0:4]
		result := make([]byte, 0, 8)
		result = append(result, byte4Md5[:4]...)
		if len(result) < 8 {
			result = append(result, bytes.Repeat([]byte{0xFF}, 8-len(result))...)
		}
		packetData := result
		packDataHexStr := hex.EncodeToString(packetData)
		addr := FCX_MD5_ADDR
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
		}

	}

	packetData := WRITE_FACTORY_TAIL_CMD_TMAX
	packDataHexStr := hex.EncodeToString(packetData)
	addr := FCX_TAIL_ADDR
	cmd, timeoutMs, err = fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	// 发送数据包
	if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("write flase fail")
	}

	return &ScaleRespMsg{m.DOWN_FACTORY_INFO_FC_RESP, "ok", s.Id}, nil
}

func ReqModifyEepromInfo(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	var eepromData EepromDataResp
	if err := json.UnmarshalFromString(req.ReqData, &eepromData.EepromData); err != nil {
		return &ScaleRespMsg{}, err
	}
	modifyData := eepromData.EepromData
	l.Log.Debug("send enable factory mode cmd to scale")
	reg, err, res := openFactory(s)
	if err != nil || !res {
		return reg, err
	}

	print(len(modifyData))
	composer := s.composer
	fn := composer.ComposeCmd
	for i := 0; i < len(modifyData); i++ {
		switch modifyData[i].Type {
		case "string":
			packetData := stringToLittleEndianBytes(modifyData[i].CurrValue, modifyData[i].Size)
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})

			if err != nil {
				return &ScaleRespMsg{}, err
			}

			// 发送数据包
			if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
			}
		case "int":
			if modifyData[i].Size == 1 {
				num, err := strconv.Atoi(modifyData[i].CurrValue)
				if err != nil {

					return &ScaleRespMsg{m.MODIFY_EEPROM_INFO_RESP, "fail,data error", s.Id}, nil
				}
				byteVal := []byte{byte(num)}
				packDataHexStr := hex.EncodeToString(byteVal)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			} else if modifyData[i].Size == 4 && modifyData[i].SubType == "ip" {
				packetData := ipv4StringToBytes(modifyData[i].CurrValue)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			} else if modifyData[i].Size == 2 || modifyData[i].Size == 4 || modifyData[i].Size == 8 {
				packetData, _ := intToLittleEndianBytes(modifyData[i].CurrValue, modifyData[i].Size)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			}

		case "double":
			if modifyData[i].Size == 8 {
				packetData := StringToFloat64Bytes(modifyData[i].CurrValue)
				packDataHexStr := hex.EncodeToString(packetData)
				cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", modifyData[i].Addr, packDataHexStr)})
				if err != nil {
					return &ScaleRespMsg{}, err
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
				}
			}
		}
	}
	return &ScaleRespMsg{m.MODIFY_EEPROM_INFO_RESP, "ok", s.Id}, nil
}

type ServerIpData struct {
	Ip         string `json:"ip"`
	Netmask    string `json:"netmask"`
	Gateway    string `json:"gateway"`
	ServerIp   string `json:"serverIp"`
	ServerPort string `json:"serverPort"`
}

type ServerIpAddr struct {
	IpAddr         int
	NetmaskAddr    int
	GatewayAddr    int
	ServerIpAddr   int
	ServerPortAddr int
}

func getServerAddr() ServerIpAddr {
	var serverAddr ServerIpAddr
	var eepromResp EepromDataResp
	fieldData, res := eeprom.GetExcelData()
	if !res {
		return serverAddr
	}
	filedDataLen := len(fieldData.EepromStruct)
	if filedDataLen <= 0 {
		return serverAddr
	}
	for i := 0; i < filedDataLen; i++ {
		var eData EData
		eData.FiledName = fieldData.EepromStruct[i].FieldName
		eData.Permission = fieldData.EepromStruct[i].Permission
		eData.Comment = fieldData.EepromStruct[i].Comments
		eData.Category = fieldData.EepromStruct[i].Category
		eData.Addr = fieldData.EepromStruct[i].Addr
		eData.Size = fieldData.EepromStruct[i].Size
		eData.Description = fieldData.EepromStruct[i].Description
		eData.Values = fieldData.EepromStruct[i].Values
		eData.CurrValue = ""
		eData.Type = fieldData.EepromStruct[i].Type
		eData.SubType = fieldData.EepromStruct[i].SubType
		eepromResp.EepromData = append(eepromResp.EepromData, eData)
	}

	count := 0
	for _, eData := range eepromResp.EepromData {
		if eData.FiledName == "M_SCALE_IP" || eData.FiledName == "M_SCALE_MASK" || eData.FiledName == "M_SCALE_GATEWAY" || eData.FiledName == "M_PORT_NUMBER" || eData.FiledName == "M_NET_IP_ADDRESS" {
			fmt.Printf("FieldName: %s, Addr: %d\n", eData.FiledName, eData.Addr)
			if eData.FiledName == "M_SCALE_IP" {
				serverAddr.IpAddr = eData.Addr
			}
			if eData.FiledName == "M_SCALE_MASK" {
				serverAddr.NetmaskAddr = eData.Addr
			}
			if eData.FiledName == "M_SCALE_GATEWAY" {
				serverAddr.GatewayAddr = eData.Addr
			}
			if eData.FiledName == "M_NET_IP_ADDRESS" {
				serverAddr.ServerIpAddr = eData.Addr
			}
			if eData.FiledName == "M_PORT_NUMBER" {
				serverAddr.ServerPortAddr = eData.Addr
			}
			count++
			if count == 6 {
				break // 找到五个后跳出循环
			}
		}
	}
	return serverAddr

}

//ReqEnFactoryMode TODO:

func ReqSetServerIp(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	var serverData ServerIpData
	if err := json.UnmarshalFromString(req.ReqData, &serverData); err != nil {
		return &ScaleRespMsg{}, err
	}
	reg, err, res := openFactory(s)
	if err != nil || !res {
		return reg, err
	}

	serverAddr := getServerAddr()
	if serverAddr.IpAddr < 1 && serverAddr.GatewayAddr < 1 && serverAddr.NetmaskAddr < 1 && serverAddr.ServerIpAddr < 1 && serverAddr.ServerPortAddr < 1 {
		return &ScaleRespMsg{}, fmt.Errorf("get eeprom addr fail")
	}

	composer := s.composer
	fn := composer.ComposeCmd

	if serverData.Ip != "" {
		packetData := ipv4StringToBytes(serverData.Ip)
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", serverAddr.IpAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
	}
	if serverData.Gateway != "" {
		packetData := ipv4StringToBytes(serverData.Gateway)
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", serverAddr.GatewayAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
	}

	if serverData.Netmask != "" {
		packetData := ipv4StringToBytes(serverData.Netmask)
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", serverAddr.NetmaskAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
	}

	if serverData.ServerIp != "" {
		packetData := ipv4StringToBytes(serverData.ServerIp)
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", serverAddr.ServerIpAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
	}

	if serverData.ServerPort != "" {
		packetData, _ := intToLittleEndianBytes(serverData.ServerPort, 2)
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_EEPROM, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", serverAddr.ServerPortAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		// 发送数据包
		if res, err := perfCmdNwaitResult(s, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("write eeprom fail")
		}
	}

	return &ScaleRespMsg{m.SET_SERVER_IP_RESP, "ok", s.Id}, nil

}

type HeaderData struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

type HeaderList struct {
	ListData []HeaderData `json:"listData"`
}

func ReqModifyVarValue(s *Scale, req SRequest) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(s)
	if err != nil || !res {
		return reg, err
	}
	var headerList HeaderList
	err = json.Unmarshal([]byte(req.ReqData), &headerList)
	if err != nil {
		return &ScaleRespMsg{}, err
	}

	composer := s.composer
	fn := composer.ComposeCmd
	var sendArray []byte
	for i := 0; i < len(headerList.ListData); i++ {
		tmpData := headerList.ListData[i]
		idByte := make([]byte, 2)
		dataLen := make([]byte, 2)
		valueBytes := []byte(tmpData.Value)
		if sendArray != nil && len(sendArray)+len(valueBytes) > 200 {
			packDataHexStr := hex.EncodeToString(sendArray)
			cmd, timeoutMs, err := fn(composer, m.CMD_MODIFY_VAR_VALUE, m.CmdData{Type: m.DATA_TYPE_STR, Data: packDataHexStr})
			if err != nil {
				return &ScaleRespMsg{m.MODIFY_VAR_RESP, "fail", s.Id}, err
			}

			if res, err := perfCmdNwaitResult(s, cmd, m.MODIFY_VAR_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{m.MODIFY_VAR_RESP, "fail", s.Id}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{m.MODIFY_VAR_RESP, "fail", s.Id}, fmt.Errorf("write eeprom fail")
			}

			i--
			sendArray = nil

		} else {
			binary.BigEndian.PutUint16(idByte, uint16(tmpData.ID))
			sendArray = append(sendArray, idByte...)
			binary.BigEndian.PutUint16(dataLen, uint16(len(tmpData.Value)))
			sendArray = append(sendArray, dataLen...)
			sendArray = append(sendArray, valueBytes...)
			if i == len(headerList.ListData)-1 {
				packDataHexStr := hex.EncodeToString(sendArray)
				cmd, timeoutMs, err := fn(composer, m.CMD_MODIFY_VAR_VALUE, m.CmdData{Type: m.DATA_TYPE_STR, Data: packDataHexStr})

				if err != nil {
					return &ScaleRespMsg{m.MODIFY_VAR_RESP, "fail", s.Id}, err
				}
				for _, b := range cmd {
					fmt.Printf("%02x ", b) // 打印每个字节的 16 进制表示并用空格分隔
				}
				// 发送数据包
				if res, err := perfCmdNwaitResult(s, cmd, m.MODIFY_VAR_RESP, timeoutMs); err != nil {
					return &ScaleRespMsg{m.MODIFY_VAR_RESP, "fail", s.Id}, err
				} else if res.MsgBody != "ok" {
					return &ScaleRespMsg{m.MODIFY_VAR_RESP, "fail", s.Id}, fmt.Errorf("write eeprom fail")
				}
			}
		}
	}

	return &ScaleRespMsg{m.MODIFY_VAR_RESP, "ok", s.Id}, nil
}

func StringToFloat64Bytes(s string) []byte {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	bits := math.Float64bits(f)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

// StringToFloat32Bytes 将字符串转换为 4 个字节的 32 位浮点数
func StringToFloat32Bytes(s string) []byte {
	// 解析字符串为浮点数
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return nil
	}
	bits := math.Float32bits(float32(f))
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}

func ipv4StringToBytes(ipStr string) []byte {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}

	ip = ip.To4()
	if ip == nil {
		return nil
	}

	return []byte(ip)
}
func intToLittleEndianBytes(dataStr string, size int) ([]byte, error) {
	num, err := strconv.Atoi(dataStr)
	if err != nil {
		return []byte{}, nil
	}
	result := make([]byte, size)
	switch size {
	case 2:
		binary.LittleEndian.PutUint16(result, uint16(num))
		return result, nil
	case 4:
		binary.LittleEndian.PutUint32(result, uint32(num))
		return result, nil
	case 8:
		binary.LittleEndian.PutUint64(result, uint64(num))
		return result, nil
	default:
		return result, nil
	}

}

func fillStrBySize(dataStr string, size int) []byte {
	runes := []rune(dataStr)
	result := make([]byte, 0, size)

	if isASCII(dataStr) {
		for i, r := range runes {
			if i >= size {
				break
			}
			result = append(result, byte(r))
		}
	}
	for i := len(result); i < size; i++ {
		result = append(result, byte(0xff))
	}

	return result
}

func isASCII(s string) bool {
	for _, char := range s {
		if char > 127 {
			return false
		}
	}
	return true
}

func stringToLittleEndianBytes(dataStr string, size int) []byte {
	runes := []rune(dataStr)
	result := make([]byte, 0, size)

	if isASCII(dataStr) {
		for i, r := range runes {
			if i >= size {
				break
			}
			result = append(result, byte(r))
		}

	} else {
		for i := 0; i < len(runes); i++ {
			if i*2+1 >= size {
				break
			}
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(runes[i]))
			result = append(result, b...)
		}
	}

	return result
}

func calculateMD5(input string) [16]byte {
	hash := md5.Sum([]byte(input)) // 计算 MD5 校验值
	return hash                    // 转换为十六进制并返回
}
func openFactory(c *Scale) (*ScaleRespMsg, error, bool) {
	if c.ScaleCat != m.SCALE_TMAX {
		return &ScaleRespMsg{}, nil, false //20250905
	}
	if c.Conn != nil && c.Conn.ProtocolName != "SCP-X" {
		sendScaleOnlineToUi(c, true, c.Model, c.Sn)
		return &ScaleRespMsg{}, nil, true
	}
	res := false
	composer := c.composer
	fn := composer.ComposeCmd

	reqMsg, _ := excuteSimpCmd(c, m.CMD_CHECK_FAC_MODE, m.EN_FAC_MODE_RESP)
	if reqMsg.MsgBody == "ok" {

		sendScaleOnlineToUi(c, true, c.Model, c.Sn)
		return &ScaleRespMsg{}, nil, true
	}

	scaleModel, sn, err := getModelNameSn(c)

	if err != nil {
		return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail"), false
	}

	if scaleModel == "" || sn == "" {
		// return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail"), false
	}

	crc16Byte := calculateMD5(scaleModel + sn + MD5SEED)
	byte4Md5 := crc16Byte[0:4]
	fmt.Println(byte4Md5)
	//------拿到随机数
	var nums []uint8
	cmd, timeoutMs, err := fn(composer, m.CMD_GET_RANDOM_DATA, m.CmdData{})
	if err != nil {
		return &ScaleRespMsg{}, err, false
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.GET_RANDOM_DATA_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err, false
	} else if res.MsgBody == "" {
		return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail"), false
	} else {
		numsInt, ok := res.MsgBody.([]uint8)
		nums = numsInt
		if !ok || len(numsInt) != 2 {
			return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail"), false
		}
	}

	dataLast := make([]byte, 6)
	dataLast[0] = nums[0]
	dataLast[1] = nums[1]
	copy(dataLast[2:6], byte4Md5)

	//打开工厂模式
	packDataHexStr := hex.EncodeToString(dataLast)
	cmd, timeoutMs, err = fn(composer, m.CMD_EN_FACTORY_MODE, m.CmdData{Type: m.DATA_TYPE_STR, Data: packDataHexStr})
	if err != nil {
		return &ScaleRespMsg{}, err, false
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.EN_FACTORY_MODE_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err, false
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail"), false
	}

	res = true
	return &ScaleRespMsg{}, err, res

}

// func ReqDownPrnFmt(c *Scale, csvPrnFmt string, seqno string) error {
func ReqDownPrnFmt(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	composer := c.composer
	fn := composer.ComposeCmd

	reqMsg, _ := excuteSimpCmd(c, m.CMD_GET_SCALE_INFO, m.GET_SCALE_INFO_RESP)
	var scaleInfo SIFromScale
	prnFmtAddr := 0
	prnFmtMaxLenth := 0
	eraseLen := 0
	strData := reqMsg.MsgBody
	if str, ok := strData.(string); ok {
		if err := json.UnmarshalFromString(str, &scaleInfo); err != nil {
			l.Log.Error(err)
			return &ScaleRespMsg{}, fmt.Errorf("get print format address fail")
		}
	} else {
		return &ScaleRespMsg{}, fmt.Errorf("get print format address fail")
	}
	siAddrInfos := scaleInfo.AddrInfos
	fmt.Println(siAddrInfos)
	for _, jsonStr := range siAddrInfos {
		addrInfo, err := parseJSON(jsonStr)
		if err != nil {
			return &ScaleRespMsg{}, fmt.Errorf("get print format address fail")
		}
		if addrInfo.Type == SI_FREE_PRN_INFO {
			prnFmtAddr = addrInfo.Addr
			prnFmtMaxLenth = addrInfo.Lenth
			eraseLen = addrInfo.EraseLen
		}
	}
	if prnFmtAddr == 0 || prnFmtMaxLenth == 0 || eraseLen == 0 {
		return &ScaleRespMsg{}, fmt.Errorf("get print format address fail")
	}
	var reqData ReqPrnData
	if err := json.UnmarshalFromString(req.ReqData, &reqData); err != nil {
		return &ScaleRespMsg{}, err
	}
	for _, file := range reqData.FilePaths {
		fileOrderNo := file[0:1]
		file = file[1:]

		csvFmtContent, err := os.ReadFile(file)
		if err != nil {
			return &ScaleRespMsg{}, err
		}

		tempStr := decryptCsv(string(csvFmtContent))
		println(tempStr)
		if !strings.Contains(tempStr, "ROTATE") {
			return &ScaleRespMsg{}, fmt.Errorf("format error,download fail! ")
		}

		data, success := prnfmt.ParserFmtToBytes(tempStr, reqData.PrinterModel, eraseLen)
		if !success {
			return &ScaleRespMsg{}, fmt.Errorf("parse print format fail")
		}

		// os.WriteFile("formatbin.bin", data, 0644)

		// 擦除原本秤上的打印格式
		l.Log.Debug("erase flash on scale")
		no, err := strconv.Atoi(fileOrderNo)
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		addr := eraseLen*(no-1) + prnFmtAddr
		size := eraseLen
		if addr > prnFmtAddr+prnFmtMaxLenth {
			return &ScaleRespMsg{}, err
		}

		loopCnt := size / eraseLen
		addrInLoop := addr
		for i := 0; i < loopCnt; i++ {
			cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH, m.CmdData{Type: m.DATA_TYPE_INT, Data: addrInLoop})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("erase fail")
			}
			addrInLoop += eraseLen
		}
		// 计算数据包数量
		packetCount := len(data) / DATA_LENGTH_256_TMAX
		if len(data)%DATA_LENGTH_256_TMAX != 0 {
			packetCount += 1
		}
		// 遍历所有数据包
		l.Log.Debug("send data package to scale")
		for i := 0; i < packetCount; i++ {
			// 计算本包数据
			start := i * DATA_LENGTH_256_TMAX
			end := start + DATA_LENGTH_256_TMAX
			if end > len(data) {
				end = len(data)
			}
			packetData := data[start:end]

			// 构建数据包
			// dataPackCmd := buildSendDataPacket(addr, packetData)
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})

			if err != nil {
				return &ScaleRespMsg{}, err
			}
			// 发送数据包
			if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
			}
			// 地址自增
			addr += 0x100
		}
		l.Log.Info("send bin ok")
	}

	SaveDownLabelFmtToScaleLog(c.Conn.ScaleName, req.ReqData)
	return &ScaleRespMsg{m.DOWN_PRN_FMT_RESP, "ok", c.Id}, nil
}

func getAddrFromScale(c *Scale, typeInt int) (SIAddrInfos, bool) {
	reqMsg, _ := excuteSimpCmd(c, m.CMD_GET_SCALE_INFO, m.GET_SCALE_INFO_RESP)
	var scaleInfo SIFromScale
	var addrInfo SIAddrInfos

	strData := reqMsg.MsgBody
	if str, ok := strData.(string); ok {
		if err := json.UnmarshalFromString(str, &scaleInfo); err != nil {
			l.Log.Error(err)
			return addrInfo, false
		}
	} else {
		return addrInfo, false
	}
	siAddrInfos := scaleInfo.AddrInfos
	fmt.Println(siAddrInfos)
	for _, jsonStr := range siAddrInfos {
		addrInfo, err := parseJSON(jsonStr)
		if err != nil {
			return addrInfo, false
		}
		if addrInfo.Type == typeInt {
			return addrInfo, true
		}
	}
	return addrInfo, false

}

func ReqDownDefaultPrnFmt(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}

	prnFmtAddr := 0
	prnFmtMaxLenth := 0
	eraseLen := 0
	addrInfo, res := getAddrFromScale(c, SI_DEF_PRN_INFO)
	if !res {
		return &ScaleRespMsg{}, fmt.Errorf("get print address fail")
	}

	prnFmtAddr = addrInfo.Addr
	prnFmtMaxLenth = addrInfo.Lenth
	eraseLen = addrInfo.EraseLen

	if prnFmtAddr == 0 || prnFmtMaxLenth == 0 || eraseLen == 0 {
		return &ScaleRespMsg{}, fmt.Errorf("get print address fail")
	}
	var reqData ReqDefaultPrnData
	if err := json.UnmarshalFromString(req.ReqData, &reqData); err != nil {
		return &ScaleRespMsg{}, err
	}

	zipFilePath := reqData.FilePaths
	//将所有文件解密后的内容和md5都存储到数组里
	strFileDataArray, res := getFileData(zipFilePath)
	if !res {
		return &ScaleRespMsg{}, fmt.Errorf("fail,file error")
	}

	//分析打印格式
	composer := c.composer
	fn := composer.ComposeCmd
	data, success := prnfmt.ParserDefFmtToBytes(strFileDataArray, reqData.PrinterModel, prnFmtMaxLenth)
	if success {
		// 擦除原本秤上的打印格式
		l.Log.Debug("erase flash on scale")

		addr := prnFmtAddr
		size := prnFmtMaxLenth

		loopCnt := size / eraseLen
		addrInLoop := addr
		for i := 0; i < loopCnt; i++ {
			cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH, m.CmdData{Type: m.DATA_TYPE_INT, Data: addrInLoop})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("erase fail")
			}
			addrInLoop += eraseLen
		}
		// 计算数据包数量
		packetCount := len(data) / DATA_LENGTH_256_TMAX
		if len(data)%DATA_LENGTH_256_TMAX != 0 {
			packetCount += 1
		}
		// 遍历所有数据包
		l.Log.Debug("send data package to scale")
		for i := 0; i < packetCount; i++ {
			// 计算本包数据
			start := i * DATA_LENGTH_256_TMAX
			end := start + DATA_LENGTH_256_TMAX
			if end > len(data) {
				end = len(data)
			}
			packetData := data[start:end]

			// 构建数据包
			// dataPackCmd := buildSendDataPacket(addr, packetData)
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			// 发送数据包
			if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
			}
			// 地址自增
			addr += 0x100
		}
		l.Log.Info("send bin ok")
	} else {
		return &ScaleRespMsg{}, fmt.Errorf("fail,Check for over 150 variables, excessive 8K print format, or incorrect print format")
	}

	SaveDownLabelFmtToScaleLog(c.Conn.ScaleName, req.ReqData)
	return &ScaleRespMsg{m.DOWN_DEFAULT_PRN_FMT_RESP, "ok", c.Id}, nil
}
func getEepromData(c *Scale, readLen int) ([]byte, error) {

	composer := c.composer
	l.Log.Debug("get eeprom data from scale")
	dataBuffer := make([]byte, 0)

	loopAddr := 0
	packetCount := readLen / 8
	for i := 0; i < packetCount; i++ {
		cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_READ_EEPROM_8, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", loopAddr, "")})
		if err != nil {
			return dataBuffer, nil
		}
		if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
			return dataBuffer, nil
		} else if res.MsgBody == "fail" {
			return dataBuffer, nil
		} else {
			strData := res.MsgBody
			if str, ok := strData.(string); ok {
				addrBytes := []byte(str)
				if len(addrBytes) == 8 {
					dataBuffer = append(dataBuffer, addrBytes...)
				} else {
					return dataBuffer, nil
				}
			} else {
				return dataBuffer, nil
			}
		}
		loopAddr = loopAddr + 8

	}

	return dataBuffer, nil

}

// func getEepromDataOiml(c *Scale, readLen int) ([]byte, error) {

// 	composer := c.composer
// 	l.Log.Debug("get eeprom data from scale")
// 	dataBuffer := make([]byte, 0)

// 	loopAddr := 152
// 	packetCount := readLen / 8
// 	for i := 0; i < packetCount; i++ {
// 		cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_READ_EEPROM_8, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", loopAddr, "")})
// 		if err != nil {
// 			return dataBuffer, nil
// 		}
// 		if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
// 			return dataBuffer, nil
// 		} else if res.MsgBody == "fail" {
// 			return dataBuffer, nil
// 		} else {
// 			strData := res.MsgBody
// 			if str, ok := strData.(string); ok {
// 				addrBytes := []byte(str)
// 				if len(addrBytes) == 8 {
// 					dataBuffer = append(dataBuffer, addrBytes...)
// 				} else {
// 					return dataBuffer, nil
// 				}
// 			} else {
// 				return dataBuffer, nil
// 			}
// 		}
// 		loopAddr = loopAddr + 8

// 	}

// 	return dataBuffer, nil

// }

func ReqBackupDefSetting(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}

	//分析打印格式
	composer := c.composer
	fn := composer.ComposeCmd

	dataInfo, _ := getEepromData(c, 512)
	if len(dataInfo) == 512 {

		// 擦除原本秤上的flash  一次擦512  DATA_LENGTH_512_TMAX
		l.Log.Debug("erase flash on scale")
		addr := FCX_DEF_FLASH_ADDR

		addrInLoop := addr

		cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH_512, m.CmdData{Type: m.DATA_TYPE_INT, Data: addrInLoop})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("erase fail")
		}

		// 计算数据包数量
		packetCount := len(dataInfo) / DATA_LENGTH_8_TMAX
		if len(dataInfo)%DATA_LENGTH_8_TMAX != 0 {
			packetCount += 1
		}
		// 遍历所有数据包
		l.Log.Debug("send data package to scale")
		for i := 0; i < packetCount; i++ {
			// 计算本包数据
			start := i * DATA_LENGTH_8_TMAX
			end := start + DATA_LENGTH_8_TMAX
			if end > len(dataInfo) {
				end = len(dataInfo)
			}
			packetData := dataInfo[start:end]

			// 构建数据包
			// dataPackCmd := buildSendDataPacket(addr, packetData)
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_8, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			// 发送数据包
			if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
			}
			// 地址自增
			addr += 0x08
		}
		l.Log.Info("send bin ok")
	} else {
		return &ScaleRespMsg{}, fmt.Errorf("fail,get def value fail")
	}

	return &ScaleRespMsg{m.BACKUP_DEF_SETTING_RESP, "ok", c.Id}, nil
}

func getFileData(zipFilePath []string) ([]string, bool) {
	var strFileDataArray []string

	for _, f := range zipFilePath {

		// 打开文件
		file, err := os.Open(f)
		if err != nil {
			return strFileDataArray, false
		}
		defer file.Close()

		tempContent, err := io.ReadAll(file)
		if err != nil {
			return strFileDataArray, false
		}

		tempStr := decryptCsv(string(tempContent))
		if !strings.Contains(tempStr, "ROTATE") {
			return strFileDataArray, false
		}
		strFileDataArray = append(strFileDataArray, tempStr)

	}
	return strFileDataArray, true
}

// func getFileDataAndMd5(zipFilePath string) ([12]string, [12]string, bool) {
// 	var strFileDataArray [12]string
// 	var strMd5Array [12]string
// 	r, err := zip.OpenReader(zipFilePath)
// 	if err != nil {
// 		return strFileDataArray, strMd5Array, false
// 	}
// 	defer r.Close()
// 	for _, f := range r.File {
// 		fmt.Println("File:", f.Name)
// 		// 打开文件
// 		rc, err := f.Open()
// 		if err != nil {
// 			return strFileDataArray, strMd5Array, false
// 		}
// 		defer rc.Close()
// 		num, res := getFileNum(f.Name)
// 		if !res {
// 			return strFileDataArray, strMd5Array, false
// 		}
// 		tempContent, err := io.ReadAll(rc)
// 		if err != nil {
// 			return strFileDataArray, strMd5Array, false
// 		}
// 		if strings.Contains(f.Name, "fmt") {
// 			tempStr := decryptCsv(string(tempContent))
// 			if !strings.Contains(tempStr, "ROTATE") {
// 				return strFileDataArray, strMd5Array, false
// 			}
// 			strFileDataArray[num-1] = tempStr
// 		} else if strings.Contains(f.Name, "txt") {
// 			strMd5Array[num-1] = string(tempContent)
// 		}
// 	}
// 	return strFileDataArray, strMd5Array, true
// }

// func checkFilesMd5(strFileDataArray [12]string, strMd5Array [12]string) bool {
// 	for i := 0; i < 12; i++ {
// 		if strFileDataArray[i] != "" {
// 			tempMd5Arr := calculateMD5(strFileDataArray[i] + MD5SEED) //统一为seed,要改打包的UI，此处先暂定这样
// 			var md5TmpStr string
// 			for _, b := range tempMd5Arr {
// 				md5TmpStr += fmt.Sprintf("%02X", b) // 将每个字节转换为两位16进制格式的字符串并拼接
// 			}
// 			if !strings.EqualFold(md5TmpStr, strMd5Array[i]) {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// func getFileNum(fNameStr string) (int, bool) {
// 	re := regexp.MustCompile(`(\d+)`) // 使用正则表达式提取数字部分
// 	match := re.FindStringSubmatch(fNameStr)

// 	if len(match) > 1 {
// 		numStr := match[1]               // 提取到的数字部分
// 		num, err := strconv.Atoi(numStr) // 将字符串转换为int类型
// 		if err == nil {
// 			return num, true
// 		} else {
// 			return 0, false
// 		}
// 	}
// 	return 0, false
// }

func decryptByte(encryptedBytes []byte) []byte {
	var decryptedBytes []byte
	for i := 0; i < len(encryptedBytes); i += 2 {
		decryptedByte := (int32(encryptedBytes[i])-3)<<4 | (int32(encryptedBytes[i+1])-3)&0x0F
		decryptedBytes = append(decryptedBytes, byte(decryptedByte))
	}
	return decryptedBytes
}

func decryptCsv(encryptedCsv string) string {
	var decryptedBytes []byte
	encryptedBytes := []byte(encryptedCsv)

	for i := 0; i < len(encryptedBytes); i += 2 {
		decryptedByte := (int32(encryptedBytes[i])-3)<<4 | (int32(encryptedBytes[i+1])-3)&0x0F
		decryptedBytes = append(decryptedBytes, byte(decryptedByte))
	}

	return string(decryptedBytes)
}

func ReqInsertPlu(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	var reqData ReqPluData

	if err := json.UnmarshalFromString(req.ReqData, &reqData); err != nil {
		return &ScaleRespMsg{}, err
	}
	l.Log.Debug("send enable factory mode cmd to scale")
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	var file = reqData.FilePath
	var nameMaxLen = reqData.NameMaxLen
	var headBytes []byte
	composer := c.composer

	l.Log.Debug("get plu address to scale")
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_GET_PLU_HEAD, m.CmdData{})
	if err != nil {
		return &ScaleRespMsg{}, err
	}

	if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody == "fail" {
		return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
	} else {
		strData := res.MsgBody
		if str, ok := strData.(string); ok {
			addrBytes := []byte(str)
			if len(addrBytes) >= 100 {
				for i := 0; i < len(addrBytes); i++ {
					if i+2 < len(addrBytes) {
						if addrBytes[i] == 0xff && addrBytes[i+1] == 0x01 && addrBytes[i+2] == 0x00 {
							headBytes = addrBytes[35 : i+3]
						}
					}
				}
				if len(headBytes) == 0 {
					return &ScaleRespMsg{}, fmt.Errorf("get plu head fail")
				}

				//前4个是起始地址，中间4个字节是现在可用的地址，最后四个是可用空间大小
				// insertPluAddr = int(binary.BigEndian.Uint32(addrBytes[4:8]))
				// insertPluSize = int(binary.BigEndian.Uint32(addrBytes[8:12]))
			} else {
				return &ScaleRespMsg{}, fmt.Errorf("get plu head fail")
			}
		} else {
			return &ScaleRespMsg{}, fmt.Errorf("get plu head fail")
		}
	}

	bufPluNumData, insertHeadBytes, resParser := ParserInsertPlu(string(file), nameMaxLen)

	if !bytes.Equal(headBytes, insertHeadBytes) {
		return &ScaleRespMsg{}, fmt.Errorf("plu head is inconsistent,fail")
	}
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	filePath := filepath.Join(exeDir, "plu.bin")

	if resParser {
		// 读取bin文件
		data, err := os.ReadFile(filePath)
		if err != nil {
			l.Log.Errorf("ReqDownDefaultPlu read file error: %v", err)
			return &ScaleRespMsg{}, err
		}
		l.Log.Debug("send enable factory mode cmd to scale")

		reg, err, res := openFactory(c)
		if err != nil || !res {
			return reg, err
		}
		composer := c.composer
		fn := composer.ComposeCmd

		l.Log.Debug("get plu address to scale")
		cmd, timeoutMs, err = composer.ComposeCmd(composer, m.CMD_INSERT_PLU_ADDR, m.CmdData{})
		if err != nil {
			return &ScaleRespMsg{}, err
		}

		var insertPluAddr int
		var insertPluSize int
		if res, err := perfCmdNwaitResult(c, cmd, m.INSERT_PLU_ADDR_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody == "fail" {
			return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
		} else {
			strData := res.MsgBody
			if str, ok := strData.(string); ok {
				addrBytes := []byte(str)
				if len(addrBytes) >= 12 {
					//前4个是起始地址，中间4个字节是现在可用的地址，最后四个是可用空间大小
					insertPluAddr = int(binary.BigEndian.Uint32(addrBytes[4:8]))
					insertPluSize = int(binary.BigEndian.Uint32(addrBytes[8:12]))
				} else {
					insertPluAddr = 0
					insertPluSize = 0
					return &ScaleRespMsg{}, fmt.Errorf("get plu address fail")
				}

			} else {
				return &ScaleRespMsg{}, fmt.Errorf("get plu address fail")
			}
		}

		// 比较空间大小
		addr := insertPluAddr
		totalSize := insertPluSize
		size := len(data)
		if totalSize < size {
			return &ScaleRespMsg{}, fmt.Errorf("no enough space to insert")
		}
		if len(bufPluNumData) > 256 {
			return &ScaleRespMsg{}, fmt.Errorf("the number cannot be more than 60")
		}
		l.Log.Debug("delete plu number cmd to scale")
		cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_DEL_PLU, m.CmdData{Type: m.DATA_TYPE_STR, Data: string(bufPluNumData)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}

		if res, err := perfCmdNwaitResult(c, cmd, m.DEL_PLU_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("delete plu id fail")
		}

		packetCount := len(data) / DATA_LENGTH_256_TMAX
		if len(data)%DATA_LENGTH_256_TMAX != 0 {
			packetCount += 1
		}
		l.Log.Debug("send data package to scale")
		for i := 0; i < packetCount; i++ {
			// 计算本包数据
			start := i * DATA_LENGTH_256_TMAX
			end := start + DATA_LENGTH_256_TMAX
			if end > len(data) {
				end = len(data)
			}
			packetData := data[start:end]
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err = fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
			}
			addr += DATA_LENGTH_256_TMAX
		}
		l.Log.Info("send bin ok")
	}
	return &ScaleRespMsg{m.INSERT_PLU_RESP, "ok", c.Id}, nil
}
func unzipAndReadFilesSrec(zipPath string) ([]byte, []byte) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		fmt.Println("Error opening ZIP:", err)
		return nil, nil
	}
	defer r.Close()

	var binData, infoData []byte
	res1 := false
	res2 := false

	if len(r.File) != 3 {
		return nil, nil
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			fmt.Println("Error opening file:", err)
			return nil, nil
		}
		defer rc.Close()

		buf := make([]byte, f.UncompressedSize64)
		_, err = io.ReadFull(rc, buf)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading file:", err)
			return nil, nil
		}

		if strings.Contains(f.Name, ".srec") {
			binData = buf
			res1 = true
		} else if strings.Contains(f.Name, ".json") {
			infoData = buf
			res2 = true
		}
	}
	if res1 && res2 {
		return binData, infoData
	}
	return nil, nil

}

func unzipAndReadFiles(zipPath string) ([]byte, []byte) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		fmt.Println("Error opening ZIP:", err)
		return nil, nil
	}
	defer r.Close()

	var binData, infoData []byte
	res1 := false
	res2 := false

	if len(r.File) != 3 {
		return nil, nil
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			fmt.Println("Error opening file:", err)
			return nil, nil
		}
		defer rc.Close()

		buf := make([]byte, f.UncompressedSize64)
		_, err = io.ReadFull(rc, buf)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading file:", err)
			return nil, nil
		}

		if strings.Contains(f.Name, ".bin") {
			binData = buf
			res1 = true
		} else if strings.Contains(f.Name, ".json") {
			infoData = buf
			res2 = true
		}
	}
	if res1 && res2 {
		return binData, infoData
	}
	return nil, nil

}

func bytesToMd5(data []byte) string {
	hash := md5.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

// 合并[]byte
func mergeByteSlices(slice1, slice2 []byte) []byte {
	result := make([]byte, len(slice1)+len(slice2))
	copy(result[:len(slice1)], slice1)
	copy(result[len(slice1):], slice2)
	return result
}

// 获取modelName 和sn
func getModelNameSn(c *Scale) (string, string, error) {
	composer := c.composer
	fn := composer.ComposeCmd
	var dataStruct FIFromScale

	cmd, timeoutMs, err := fn(composer, m.CMD_GET_FACTORY_INFO, m.CmdData{})
	if err != nil {
		return "", "", err
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.GET_FACTORY_INFO_RESP, timeoutMs); err != nil {
		return "", "", err
	} else if res.MsgBody == "" {
		return "", "", err
	} else {
		msgBodyStr, ok := res.MsgBody.(string)
		if !ok {
			return "", "", err
		}
		err := json.UnmarshalFromString(msgBodyStr, &dataStruct)
		if err != nil {
			return "", "", err
		}
	}
	if dataStruct.ModelName == "" || dataStruct.ScaleSn == "" {
		return "", "", err
	}

	sendScaleOnlineToUi(c, true, dataStruct.ModelName, dataStruct.ScaleSn)
	return dataStruct.ModelName, dataStruct.ScaleSn, nil
}

// 从zip中获取bin和机种名
func getZipInfo(fileName string) ([]byte, string, error) {
	readBinData, txtData := unzipAndReadFiles(fileName)
	if readBinData == nil || txtData == nil {
		return nil, "", fmt.Errorf("file error")
	}

	var firmwareInfo ReqFirmwareInfo
	if err := json.UnmarshalFromString(string(txtData), &firmwareInfo); err != nil {
		return nil, "", fmt.Errorf("file error")
	}
	readMd5 := firmwareInfo.BinKey
	tempByte := mergeByteSlices(readBinData, []byte(MD5SEED))
	calMd5Str := bytesToMd5(tempByte)
	if strings.Trim(readMd5, " ") != calMd5Str {
		return nil, "", fmt.Errorf("file error")
	}

	modelName := firmwareInfo.ModelName
	return readBinData, modelName, nil
}

// 从zip中获取srec 和机种
func getZipInfoSrec(fileName string) ([]byte, string, string, error) {

	readSrecData, txtData := unzipAndReadFilesSrec(fileName)
	if readSrecData == nil || txtData == nil {
		return nil, "", "", fmt.Errorf("file error")
	}
	var firmwareInfo ReqFirmwareInfo
	if err := json.UnmarshalFromString(string(txtData), &firmwareInfo); err != nil {
		return nil, "", "", fmt.Errorf("file error")
	}
	readMd5 := firmwareInfo.SrecKey

	tempByte := mergeByteSlices(readSrecData, []byte(MD5SEED))
	calMd5Str := bytesToMd5(tempByte)
	if strings.Trim(readMd5, " ") != calMd5Str {
		return nil, "", "", fmt.Errorf("file error")
	}

	return readSrecData, firmwareInfo.ModelName, firmwareInfo.BootloaderVersion, nil
}

const (
	BOOTLOADER_OLD  = "boot_no_crc"
	BOOTLOADER_NEW  = "boot_with_crc"
	BOOTLOADER_TMAX = "new_tmax_boot"
)

// 更新srec之前要先验证 机种是否匹配
func ReqUpdateFirmware(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	if c.ScaleCat == m.SCALE_C51 {
		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,scale type error.", c.Id}, nil
	}
	if req.ReqData == "" {
		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,file error.", c.Id}, nil
	}
	parts := strings.Split(req.ReqData, ",")
	if len(parts) != 2 {
		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,file error.", c.Id}, nil
	}

	readSrecData, modelName, bootloaderVersion, err := getZipInfoSrec(parts[0])

	if err != nil || len(readSrecData) == 0 {
		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,file error.", c.Id}, nil
	}
	println(modelName)
	//现在都是强制更新，不问机种名序列号
	// if parts[1] == "0" {
	// 	err, scaleName, _ := getModelNameSn(c)
	// 	if err != nil {
	// 		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,check connection.", c.Id}, nil
	// 	}
	// 	if scaleName != modelName {
	// 		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,the model does not match.", c.Id}, nil
	// 	}
	// }
	binData := decryptByte(readSrecData)
	tmpFile, err := os.CreateTemp("", "update_*.srec")
	if err != nil {
		return &ScaleRespMsg{m.UPDATE_FIRMWARE_RESP, "fail,write file error.", c.Id}, nil
	}
	defer func() {
		tmpFile.Close()
		if err := os.Remove(tmpFile.Name()); err != nil {
			l.Log.Printf("删除临时文件失败: %v, 路径: %s", err, tmpFile.Name())
		}
	}()

	SaveUpdateFirmwareLog(c.Conn.ScaleName, parts[0], "Serial Port")

	writeStringToFile(string(binData), tmpFile.Name())
	switch bootloaderVersion {
	case BOOTLOADER_NEW:
		return c.UpdateFirmware(tmpFile.Name()) //带有CRC的boot升级，bootcommander
	case BOOTLOADER_OLD:
		return c.UpdateFirmwareOld(tmpFile.Name()) //不带CRC的boot升级，bootcommander
	case BOOTLOADER_TMAX:
		return c.TmaxUpdateFirmware(tmpFile.Name()) //Tmax升级，串口自己升级
	default:
		return c.UpdateFirmware(tmpFile.Name()) //带有CRC的boot升级，bootcommander
	}
}

func writeStringToFile(str string, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(str)
	if err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

// 关闭wifi 蓝牙的透传
func ReqDisWifiPassthrough(c *Scale) (*ScaleRespMsg, error) {

	return excuteSimpCmd(c, m.CMD_DIS_PASSTH, m.DIS_PASSTH_MODE_RESP)
}

// 关闭串口
func ReqCloseSerialPort(c *Scale) (*ScaleRespMsg, error) {
	if c.MySerial == nil {

		return &ScaleRespMsg{m.CLOSE_SERIAL_PORT_RESP, "fail", c.Id}, nil
	}
	c.MySerial.Close()
	c.MySerial = nil

	return &ScaleRespMsg{m.CLOSE_SERIAL_PORT_RESP, "ok", c.Id}, nil

}

// 打开串口
func ReqOpenSerialPort(c *Scale) (*ScaleRespMsg, error) {
	var err error
	if c.MySerial != nil {
		return &ScaleRespMsg{m.OPEN_SERIAL_PORT_RESP, "fail", c.Id}, nil
	}
	scaleCat := c.ScaleCat
	picker := picker.GetPickerFn(scaleCat)

	if c.MySerial, err = NewSerial(c.Pcnf, picker, true); err != nil {
		l.Log.Error(err.Error())
		return &ScaleRespMsg{m.OPEN_SERIAL_PORT_RESP, "fail", c.Id}, nil
	}

	return &ScaleRespMsg{m.OPEN_SERIAL_PORT_RESP, "ok", c.Id}, nil
}

// 获取基础数据 OL UL 开关机次数等
func ReqGetBasicData(c *Scale) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	return excuteSimpCmd(c, m.CMD_GET_BASIC_DATA, m.GET_BASIC_DATA_RESP)
}

// 将string 转为 浮点数的小端模式的8个字节  Tmax是这样存的
// 123.456   77 BE 9F 1A 2F DD 5E 40
func stringToLittleEndianDouble(s string) ([]byte, error) {
	buf := make([]byte, 8)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return buf, err
	}
	binary.LittleEndian.PutUint64(buf, math.Float64bits(f))
	return buf, err
}

// 设置上下限
func ReqSetLimitToScale(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	// reg, err, res := openFactory(c)
	// if err != nil || !res {
	// 	return reg, err
	// }
	strList := strings.Split(req.ReqData, ",")

	if len(strList) != 2 {
		return &ScaleRespMsg{}, fmt.Errorf("fail,req data error")
	}
	dataLow, err := stringToLittleEndianDouble(strList[0])
	if err != nil {
		return &ScaleRespMsg{}, fmt.Errorf("fail,req data error")
	}
	dataHigh, err := stringToLittleEndianDouble(strList[1])
	if err != nil {
		return &ScaleRespMsg{}, fmt.Errorf("fail,req data error")
	}
	combined := mergeByteSlices(dataLow, dataHigh)
	packDataHexStr := hex.EncodeToString(combined)
	l.Log.Debug("send cmd to scale")
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_LIMIT_TO_SCALE, m.CmdData{Type: m.DATA_TYPE_STR, Data: packDataHexStr})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	if _, err := perfCmdNwaitResult(c, cmd, m.SET_LIMIT_TO_SCALE_RESP, timeoutMs); err != nil {
		// 	return &ScaleRespMsg{}, err
		// } else if res.MsgBody != "ok" {
		// 	return &ScaleRespMsg{}, fmt.Errorf("set limit fail")
		return &ScaleRespMsg{m.SET_LIMIT_TO_SCALE_RESP, "ok", c.Id}, nil
	}
	return &ScaleRespMsg{m.SET_LIMIT_TO_SCALE_RESP, "ok", c.Id}, nil

}

func ReqOpenBillSend(c *Scale) (*ScaleRespMsg, error) {
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	return excuteSimpCmd(c, m.CMD_OPEN_BILL_SEND, m.OPEN_BILL_SEND_RESP)
}

// 20260311备份
// 在线升级bin
func ReqDownFirmware(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	readBinData, modelName, err := getZipInfo(req.ReqData)
	if err != nil {
		return &ScaleRespMsg{}, fmt.Errorf("fail,file error")
	}

	if len(readBinData) == 0 {
		return &ScaleRespMsg{}, fmt.Errorf("fail,file error")
	}

	scaleName, _, err := getModelNameSn(c)
	if err != nil {
		return &ScaleRespMsg{}, fmt.Errorf("fail,check connection")
	}

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{}, fmt.Errorf("fail,check connection")
	}

	if scaleName != modelName {
		//如果modelname 不一样，不能升级
		// return &ScaleRespMsg{m.DOWN_FIRMWARE_WIFI_RESP, "fail,the model does not match.", c.Id}, nil
	}

	binData := decryptByte(readBinData)
	if len(binData) == 0 || len(binData) > 1024*200 {
		return &ScaleRespMsg{}, fmt.Errorf("fail,file error")
	}

	composer := c.composer
	fn := composer.ComposeCmd

	// 擦除
	addr := 0x2002A000 //这个地址是不是统一的？
	size := 128 * 1024
	loopCnt := size / 4096
	addrInLoop := addr
	for i := 0; i < loopCnt; i++ {
		cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH, m.CmdData{Type: m.DATA_TYPE_INT, Data: addrInLoop})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("erase fail")
		}
		addrInLoop += 4096
	}

	respMsg100 := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(10), ScaleId: c.Id}
	result, _ := json.Marshal(respMsg100)
	c.client.sendCh <- result

	//开始写
	//开辟4K的空间来存储校验和尾巴，尾巴为8个字节，前四个字节为bin长度，后四个字节为固定的 5a a5 a5 5a
	loopDataLen := 4096
	last4kByte := make([]byte, loopDataLen)
	// 	// 填充数据不够4096的部分
	binDataAdd := padOrReturnBytes(binData)

	crcLoop := len(binDataAdd) / loopDataLen
	crcLen := 4
	for i := 0; i < crcLoop; i++ {
		// 计算本包数据
		start := i * loopDataLen
		end := start + loopDataLen
		startCrc := i * crcLen

		if end > len(binDataAdd) {
			end = len(binDataAdd)
		}
		packetData := binDataAdd[start:end]
		checksum := utils.Crc32MPEG2(packetData)
		binary.BigEndian.PutUint32(last4kByte[startCrc:], checksum)
	}
	binary.LittleEndian.PutUint32(last4kByte[loopDataLen-8:], uint32(len(binData)))
	binary.BigEndian.PutUint16(last4kByte[loopDataLen-4:], 0x5aa5)
	binary.BigEndian.PutUint16(last4kByte[loopDataLen-2:], 0xa55a)

	packetCount := len(binDataAdd) / DATA_LENGTH_256_TMAX
	println(packetCount)

	process := 1.0

	if packetCount > 0 {
		// 浮点数除法，得到小数结果
		process = 85.0 / float64(packetCount)
		if process < 0.01 {
			process = 0.01
		}
	}

	l.Log.Debug("send data package to scale")
	loopAddr := addr
	for i := 0; i < packetCount; i++ {
		// 计算本包数据
		start := i * DATA_LENGTH_256_TMAX
		end := start + DATA_LENGTH_256_TMAX
		if end > len(binDataAdd) {
			end = len(binDataAdd)
		}
		packetData := binDataAdd[start:end]
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", loopAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
		}
		loopAddr += DATA_LENGTH_256_TMAX

		totalProcessFloat := 11.0 + float64(i)*process
		totalProcessInt := int(math.Round(totalProcessFloat))
		if totalProcessInt > 95 {
			totalProcessInt = 95
		}

		respMsg100 := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(totalProcessInt), ScaleId: c.Id}
		result, _ := json.Marshal(respMsg100)
		c.client.sendCh <- result

	}
	//写最后4K
	lastLoop := len(last4kByte) / DATA_LENGTH_256_TMAX
	println(lastLoop)

	l.Log.Debug("send data package to scale")
	lastLoopAddr := addr + 1024*124

	for i := 0; i < lastLoop; i++ {
		// 计算本包数据
		start := i * DATA_LENGTH_256_TMAX
		end := start + DATA_LENGTH_256_TMAX
		if end > len(last4kByte) {
			end = len(last4kByte)
		}
		packetData := last4kByte[start:end]
		packDataHexStr := hex.EncodeToString(packetData)
		cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", lastLoopAddr, packDataHexStr)})
		if err != nil {
			return &ScaleRespMsg{}, err
		}
		if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody != "ok" {
			return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
		}
		lastLoopAddr += DATA_LENGTH_256_TMAX
	}

	l.Log.Info("send bin ok")
	SaveUpdateFirmwareLog(c.Conn.ScaleName, req.ReqData, "Network")

	// 重启
	Reboot(c)
	return &ScaleRespMsg{m.DOWN_FIRMWARE_WIFI_RESP, "ok", c.Id}, nil
}

func padOrReturnBytes(data []byte) []byte {
	remainder := len(data) % 4096
	if remainder == 0 {
		return data
	}
	//补上最后一个不够4096的数据
	paddedData := make([]byte, len(data)+(4096-remainder))
	copy(paddedData, data)
	for i := len(data); i < len(paddedData); i++ {
		paddedData[i] = 0xFF
	}
	return paddedData
}

// 下发PLU
func ReqDownPlu(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	var reqData ReqPluData
	if err := json.UnmarshalFromString(req.ReqData, &reqData); err != nil {
		return &ScaleRespMsg{}, err
	}
	var file = reqData.FilePath
	var nameMaxLen = reqData.NameMaxLen
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	filePath := filepath.Join(exeDir, "plu.bin")

	if ParserPluFile(string(file), nameMaxLen) {
		// 读取bin文件
		data, err := os.ReadFile(filePath)
		if err != nil {
			l.Log.Errorf("ReqDownPlu read file error: %v", err)
			return &ScaleRespMsg{}, err
		}
		l.Log.Debug("send enable factory mode cmd to scale")
		reg, err, res := openFactory(c)
		if err != nil || !res {
			return reg, err
		}
		composer := c.composer
		fn := composer.ComposeCmd

		//获取秤上的PLU地址
		reqMsg, _ := excuteSimpCmd(c, m.CMD_GET_SCALE_INFO, m.GET_SCALE_INFO_RESP)
		var scaleInfo SIFromScale
		pluStartAddr := 0
		pluMaxLenth := 0

		strData := reqMsg.MsgBody
		if str, ok := strData.(string); ok {
			if err := json.UnmarshalFromString(str, &scaleInfo); err != nil {
				l.Log.Error(err)
				return &ScaleRespMsg{}, fmt.Errorf("get plu address fail")
			}
		} else {
			return &ScaleRespMsg{}, fmt.Errorf("get plu address fail")
		}

		siAddrInfos := scaleInfo.AddrInfos
		fmt.Println(siAddrInfos)

		for _, jsonStr := range siAddrInfos {
			addrInfo, err := parseJSON(jsonStr)
			if err != nil {
				return &ScaleRespMsg{}, fmt.Errorf("get plu address fail")
			}

			if addrInfo.Type == SI_PLU_INFO {
				pluStartAddr = addrInfo.Addr
				pluMaxLenth = addrInfo.Lenth
			}
		}
		if pluStartAddr == 0 || pluMaxLenth == 0 {
			return &ScaleRespMsg{}, fmt.Errorf("get plu address fail")
		}
		// 擦除原本秤上的PLU
		// addr, size := mycmd.GetPluRomAddrNSize(c.ScaleCat)
		addr := pluStartAddr
		maxSize := pluMaxLenth
		size := len(data)
		if maxSize < size {
			return &ScaleRespMsg{}, fmt.Errorf("no enough space to download")
		}
		loopCnt := size/4096 + 1
		addrInLoop := addr
		for i := 0; i < loopCnt; i++ {
			cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH, m.CmdData{Type: m.DATA_TYPE_INT, Data: addrInLoop})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("erase fail")
			}
			addrInLoop += 4096
		}
		//擦除完成后送进度
		respMsg100 := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(10), ScaleId: c.Id}
		result, _ := json.Marshal(respMsg100)
		c.client.sendCh <- result

		packetCount := len(data) / DATA_LENGTH_256_TMAX
		if len(data)%DATA_LENGTH_256_TMAX != 0 {
			packetCount += 1
		}

		process := 1.0

		if packetCount > 0 {
			// 浮点数除法，得到小数结果
			process = 85.0 / float64(packetCount)
			if process < 0.01 {
				process = 0.01
			}
		}

		l.Log.Debug("send data package to scale")
		for i := 0; i < packetCount; i++ {
			// 计算本包数据
			start := i * DATA_LENGTH_256_TMAX
			end := start + DATA_LENGTH_256_TMAX
			if end > len(data) {
				end = len(data)
			}
			packetData := data[start:end]
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
			}
			addr += DATA_LENGTH_256_TMAX

			totalProcessFloat := 11.0 + float64(i)*process
			totalProcessInt := int(math.Round(totalProcessFloat))
			if totalProcessInt > 95 {
				totalProcessInt = 95
			}

			respMsg100 := ScaleRespMsg{MsgType: m.UPDATE_FIRMWARE_PROGRESS, MsgBody: strconv.Itoa(totalProcessInt), ScaleId: c.Id}
			result, _ := json.Marshal(respMsg100)
			c.client.sendCh <- result

		}
		l.Log.Info("send bin ok")
	} else {
		return &ScaleRespMsg{}, fmt.Errorf("fail,data error")
	}
	//备份PLU表格 记录md5值和文件的对应关系
	destPath := getPluFilePath(getCurrPath())
	CopyFile(string(file), destPath) //备份plu下发的数据
	md5Str := fileToMd5(destPath)
	var pluDown PluRec
	pluDown.FileName = destPath
	pluDown.Md5 = md5Str
	c.AddPluDownRec(pluDown)
	l.Log.Info("erase insert addr")
	//下命令让秤擦除新增区
	composer := c.composer
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_INSERT_PLU, m.CmdData{})
	if err != nil {
		return &ScaleRespMsg{}, err
	}

	if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_INSERT_PLU_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody == "fail" {
		return &ScaleRespMsg{}, fmt.Errorf("erase insert plu fail")
	}
	return &ScaleRespMsg{m.DOWN_PLU_RESP, "ok", c.Id}, nil
}

func parseJSON(jsonStr string) (SIAddrInfos, error) {
	var addrInfo SIAddrInfos
	err := json.Unmarshal([]byte(jsonStr), &addrInfo)
	return addrInfo, err
}

func getPluFilePath(currPath string) string {
	timestamp := time.Now().Unix()
	timeStr := strconv.FormatInt(timestamp, 10)
	currPath = currPath + string(os.PathSeparator) + timeStr + ".xlsx"
	return currPath
}

func getCurrPath() string {

	currentPath :=
		m.GetSrvDataPath()
	pluBackPath := filepath.Join(currentPath, m.PLU_BACK_PATH)
	if _, err := os.Stat(pluBackPath); os.IsNotExist(err) {
		os.MkdirAll(pluBackPath, os.ModePerm)
	}
	currentPath = filepath.Join(currentPath, m.PLU_BACK_PATH)
	return currentPath
}

func CopyFile(srcFilePath, dstFilePath string) (written int64, err error) {

	srcfile, err := os.Open(srcFilePath)
	defer func() {
		if err = srcfile.Close(); err != nil {
			l.Log.Info(err)
		}
	}()
	if err != nil {
		l.Log.Info(err)
	}
	// 构建 src Reader
	srcReaer := bufio.NewReader(srcfile)

	//打开dstFilePath
	dstfile, err := os.OpenFile(dstFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	defer func() {
		if err := dstfile.Close(); err != nil {
			l.Log.Info(err)
		}
	}()
	if err != nil {
		l.Log.Info(err)
	}
	// 构建 src Writer
	dstWriter := bufio.NewWriter(dstfile)

	return io.Copy(dstWriter, srcReaer)

}

// 设置秤上的时间
func ReqSetScaleTime(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData
	num, err := strconv.ParseUint(reqData, 10, 32)
	if err != nil {
		return &ScaleRespMsg{m.SET_SCALE_TIME_RESP, "fail, data error", c.Id}, nil
	}
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}

	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, uint32(num))
	l.Log.Debug("send cmd to scale")
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_SCALE_TIME, m.CmdData{Type: m.DATA_TYPE_STR, Data: string(bytes)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_SCALE_TIME_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("set time fail")
	}

	SaveSetScaleTimeLog(c.Conn.ScaleName, num)

	return &ScaleRespMsg{m.SET_SCALE_TIME_RESP, "ok", c.Id}, nil

}

func ReqDelPlu(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	var reqData ReqDelPLuData
	if err := json.UnmarshalFromString(req.ReqData, &reqData); err != nil {
		return &ScaleRespMsg{}, err
	}

	if len(reqData.PluId) == 0 {
		return &ScaleRespMsg{m.DEL_PLU_RESP, "fail, no PLU id", c.Id}, nil
	}

	bufData, _ := ParserDelPlu(reqData.PluId)

	l.Log.Debug("send cmd to scale")
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_DEL_PLU, m.CmdData{Type: m.DATA_TYPE_STR, Data: string(bufData)})
	if err != nil {
		return &ScaleRespMsg{}, err
	}

	if res, err := perfCmdNwaitResult(c, cmd, m.DEL_PLU_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{}, fmt.Errorf("delete plu id fail")
	}

	return &ScaleRespMsg{m.DEL_PLU_RESP, "ok", c.Id}, nil
}

// func deleteFile(fileName string) error {
// 	currentDir, err := os.Getwd()
// 	if err != nil {
// 		l.Log.Debug("failed get current path")
// 		return nil
// 	}
// 	// 拼接文件的完整路径
// 	fullPath := filepath.Join(currentDir, fileName)
// 	// 检查文件是否存在
// 	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
// 		l.Log.Debug("file does not exist ")
// 		return nil
// 	}
// 	// 删除文件
// 	err = os.Remove(fullPath)
// 	if err != nil {
// 		l.Log.Debug("delete file failed")
// 		return err
// 	}
// 	return nil
// }

// 获取OL UL的异常数据
func ReqGetWeightErr(c *Scale) (*ScaleRespMsg, error) {
	l.Log.Debug("send enable factory mode cmd to scale")
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}

	//获取秤上的OLUL地址
	reqMsg, _ := excuteSimpCmd(c, m.CMD_GET_SCALE_INFO, m.GET_SCALE_INFO_RESP)
	var scaleInfo SIFromScale
	olAddr := 0
	ulAddr := 0

	strData := reqMsg.MsgBody
	if str, ok := strData.(string); ok {
		if err := json.UnmarshalFromString(str, &scaleInfo); err != nil {
			l.Log.Error(err)
			return &ScaleRespMsg{}, fmt.Errorf("get ol/ul address fail")
		}

	} else {
		return &ScaleRespMsg{}, fmt.Errorf("get ol/ul address fail")
	}

	siAddrInfos := scaleInfo.AddrInfos
	fmt.Println(siAddrInfos)

	for _, jsonStr := range siAddrInfos {
		addrInfo, err := parseJSON(jsonStr)
		if err != nil {
			return &ScaleRespMsg{}, fmt.Errorf("get ol/ul address fail")
		}

		if addrInfo.Type == SI_OL_INFO {
			olAddr = addrInfo.Addr
		}
		if addrInfo.Type == SI_UL_INFO {
			ulAddr = addrInfo.Addr
		}
	}
	if olAddr == 0 || ulAddr == 0 {
		return &ScaleRespMsg{}, fmt.Errorf("get address fail")
	}

	byteOLArray, resOL := getOlUlFromScale(c, olAddr)
	if !resOL {
		return &ScaleRespMsg{}, fmt.Errorf("get data fail")
	}
	byteULArray, resUL := getOlUlFromScale(c, ulAddr)
	if !resUL {
		return &ScaleRespMsg{}, fmt.Errorf("get data fail")
	}

	var respInfo OlUlInfo
	if len(byteOLArray) == 8 && len(byteULArray) == 8 {
		respInfo.OlTime = int(binary.LittleEndian.Uint32(byteOLArray[0:4]))
		respInfo.OlCnt = int(binary.LittleEndian.Uint32(byteOLArray[4:8]))
		respInfo.UlTime = int(binary.LittleEndian.Uint32(byteULArray[0:4]))
		respInfo.UlCnt = int(binary.LittleEndian.Uint32(byteULArray[4:8]))
	}

	respInfoStr, err := json.MarshalToString(respInfo)
	if err != nil {
		return &ScaleRespMsg{}, fmt.Errorf("get data fail")
	}

	return &ScaleRespMsg{m.GET_WEIGHT_ERR_RESP, respInfoStr, c.Id}, nil
}

func getOlUlFromScale(c *Scale, addr int) ([]byte, bool) {
	var byteArray []byte
	composer := c.composer
	fn := composer.ComposeCmd
	l.Log.Debug("read flash from scale")
	getData8Len := []byte{0x00, 0x08}
	getData256Len := []byte{0x01, 0x00}
	packDataHexStr := hex.EncodeToString(getData8Len)
	packData256HexStr := hex.EncodeToString(getData256Len)
	totalSpace := (8 * 1024)
	loopCnt := totalSpace / DATA_LENGTH_256_TMAX
	if totalSpace%DATA_LENGTH_256_TMAX != 0 {
		loopCnt += 1
	}
	addrInLoop := addr
	for i := 0; i < loopCnt; i++ {
		cmd, timeoutMs, err := fn(composer, m.CMD_GET_WEIGHT_ERR, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addrInLoop, packDataHexStr)})
		if err != nil {
			return byteArray, false
		}
		if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
			return byteArray, false
		} else if res.MsgBody == "fail" {
			return byteArray, false
		} else {
			strData := res.MsgBody
			if str, ok := strData.(string); ok {
				addrBytes := []byte(str)
				if len(addrBytes) == 8 &&
					addrBytes[0] == 0xFF && addrBytes[1] == 0xFF && addrBytes[2] == 0xFF && addrBytes[3] == 0xFF &&
					addrBytes[4] == 0xFF && addrBytes[5] == 0xFF && addrBytes[6] == 0xFF && addrBytes[7] == 0xFF {
					// 0到7位置都是 FF
					break
				} else if len(addrBytes) != 8 {
					return byteArray, false
				}
			} else {
				return byteArray, false
			}
		}
		addrInLoop += 256
	}

	if addrInLoop > addr {
		cmd, timeoutMs, err := fn(composer, m.CMD_GET_WEIGHT_ERR, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addrInLoop-256, packData256HexStr)})
		if err != nil {
			return byteArray, false
		}

		if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
			return byteArray, false
		} else if res.MsgBody == "fail" {
			return byteArray, false
		} else {
			strData := res.MsgBody
			if str, ok := strData.(string); ok {
				addrBytes := []byte(str)
				if len(addrBytes) == 256 {
					len := 0
					for i := 0; i < 256/8; i++ {
						if addrBytes[len] == 0xFF && addrBytes[len+1] == 0xFF && addrBytes[len+2] == 0xFF && addrBytes[len+3] == 0xFF &&
							addrBytes[len+4] == 0xFF && addrBytes[len+5] == 0xFF && addrBytes[len+6] == 0xFF && addrBytes[len+7] == 0xFF {

							byteArray = addrBytes[len-8 : len]
							break

						} else if len+8 == 256 {
							byteArray = addrBytes[256-8 : 256]
							break
						}
						len += 8
					}
				} else if len(addrBytes) != 256 {
					return byteArray, false
				}
			} else {
				return byteArray, false
			}
		}

	} else {
		byteArray = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}
	return byteArray, true

}

func ReqGetOneEepromInfo(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	var eepromAddr int
	var eepromSize int
	var dataVal int

	if req.ReqData == WIFI_BT_INFO {
		eepromAddr, eepromSize = eeprom.GetFuncAddrSize("M_WIRELESS_PERIPHERAL_TYPE")
	}
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	composer := c.composer

	if eepromSize > 0 && eepromAddr < 256 {

		l.Log.Debug("get eeprom data from scale")
		cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_READ_EEPROM_256, m.CmdData{})
		if err != nil {
			return &ScaleRespMsg{}, err
		}

		if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
			return &ScaleRespMsg{}, err
		} else if res.MsgBody == "fail" {
			return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
		} else {
			strData := res.MsgBody
			if str, ok := strData.(string); ok {
				addrBytes := []byte(str)
				if len(addrBytes) >= 256 {
					if eepromSize == 1 {
						dataVal = int(addrBytes[eepromAddr])
						switch dataVal {
						case 0:
							return &ScaleRespMsg{m.GET_ONE_EEPROM_INFO_RESP, "ok,off", c.Id}, nil
						case 1:
							return &ScaleRespMsg{m.GET_ONE_EEPROM_INFO_RESP, "ok,wifi", c.Id}, nil
						case 2:
							return &ScaleRespMsg{m.GET_ONE_EEPROM_INFO_RESP, "ok,bt", c.Id}, nil
						}
					}

				}

			} else {
				return &ScaleRespMsg{}, fmt.Errorf("get wifi&Bt fail")
			}
		}

	}

	return &ScaleRespMsg{m.GET_ONE_EEPROM_INFO_RESP, "", c.Id}, nil

}

type EData struct {
	FiledName   string `json:"filedName"`
	Size        int    `json:"size"`
	Addr        int    `json:"addr"`
	Type        string `json:"type"`
	SubType     string `json:"subType"`
	Permission  int    `json:"permission"`
	Values      string `json:"values"`
	CurrValue   string `json:"currValue"`
	Description string `json:"description"`
	Comment     string `json:"comment"`
	Category    string `json:"category"`
}
type EDataDown struct {
	FiledName   string `json:"filedName"`
	Size        int    `json:"size"`
	Addr        int    `json:"addr"`
	Type        string `json:"type"`
	SubType     string `json:"subType"`
	Permission  int    `json:"permission"`
	Value       string `json:"value"`
	Values      string `json:"values"`
	CurrValue   string `json:"currValue"`
	Description string `json:"description"`
	Comment     string `json:"comment"`
	Category    string `json:"category"`
	Id          int    `json:"id"`
}

type EepromDataResp struct {
	EepromData []EData
}
type EepromDataDownResp struct {
	EepromDataDown []EDataDown
}

type FactDataDown struct {
	Size  int    `json:"size"`
	Type  string `json:"type"`
	Value string `json:"value"`
	Id    int    `json:"id"`
}

type FactoryDataDownResp struct {
	FactoryDataDown []FactDataDown
}

func ReqGetAllEepromInfo(c *Scale) (*ScaleRespMsg, error) {

	var filedDataLen = 0
	var eepromResp EepromDataResp
	fieldData, res := eeprom.GetExcelData()
	if !res {
		return &ScaleRespMsg{}, fmt.Errorf("fail,no eeprom data file")
	}
	filedDataLen = len(fieldData.EepromStruct)
	if filedDataLen <= 0 {
		return &ScaleRespMsg{}, fmt.Errorf("fail,eeprom data file is empty")
	}
	for i := 0; i < filedDataLen; i++ {
		var eData EData
		eData.FiledName = fieldData.EepromStruct[i].FieldName
		eData.Permission = fieldData.EepromStruct[i].Permission
		eData.Comment = fieldData.EepromStruct[i].Comments
		eData.Category = fieldData.EepromStruct[i].Category
		eData.Addr = fieldData.EepromStruct[i].Addr
		eData.Size = fieldData.EepromStruct[i].Size
		eData.Description = fieldData.EepromStruct[i].Description
		eData.Values = fieldData.EepromStruct[i].Values
		eData.CurrValue = ""
		eData.Type = fieldData.EepromStruct[i].Type
		eData.SubType = fieldData.EepromStruct[i].SubType
		eepromResp.EepromData = append(eepromResp.EepromData, eData)
	}
	eepromStr, _ := json.MarshalToString(eepromResp.EepromData)
	composer := c.composer

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	}

	l.Log.Debug("get eeprom data from scale")
	dataBuffer := make([]byte, 0)
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_READ_EEPROM_256, m.CmdData{})
	if err != nil {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	} else if res.MsgBody == "fail" {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	} else {
		strData := res.MsgBody
		if str, ok := strData.(string); ok {
			addrBytes := []byte(str)
			if len(addrBytes) >= 256 {
				dataBuffer = append(dataBuffer, addrBytes...)
			} else {
				return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
			}
		} else {
			return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
		}
	}
	cmd, timeoutMs, err = composer.ComposeCmd(composer, m.CMD_READ_EEPROM_512, m.CmdData{})
	if err != nil {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.READ_FLASH_DATA_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	} else if res.MsgBody == "fail" {
		return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
	} else {
		strData := res.MsgBody
		if str, ok := strData.(string); ok {
			addrBytes := []byte(str)
			if len(addrBytes) >= 256 {
				dataBuffer = append(dataBuffer, addrBytes...)
			} else {
				return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
			}
		} else {
			return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil
		}
	}
	eepromResp, _ = paserEepromData(dataBuffer, eepromResp, 0)
	eepromStr, _ = json.MarshalToString(eepromResp.EepromData)
	return &ScaleRespMsg{m.GET_ALL_EEPROM_INFO_RESP, eepromStr, c.Id}, nil

}

func paserEepromData(eepromBytes []byte, eepromResp EepromDataResp, startAddr int) (EepromDataResp, bool) {
	var dataAddr int
	var dataSize int
	var dataType string
	var dataSubType string

	for i := 0; i < len(eepromResp.EepromData); i++ {
		dataAddr = eepromResp.EepromData[i].Addr
		dataSize = eepromResp.EepromData[i].Size
		dataType = eepromResp.EepromData[i].Type
		dataSubType = eepromResp.EepromData[i].SubType

		if dataAddr >= startAddr && (dataAddr-startAddr) < len(eepromBytes) && len(eepromBytes) > dataAddr+dataSize {

			switch dataType {
			case "int":
				switch dataSize {
				case 1:
					if eepromBytes[dataAddr] == 0xff {
						eepromResp.EepromData[i].Permission = 0
						eepromResp.EepromData[i].CurrValue = ""
					} else {
						dataInt := int(eepromBytes[dataAddr])
						eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt)
					}

				case 2:
					if eepromBytes[dataAddr] == 0xff {
						eepromResp.EepromData[i].Permission = 0
						eepromResp.EepromData[i].CurrValue = ""
					} else {
						dataInt := binary.LittleEndian.Uint16(eepromBytes[dataAddr : dataAddr+2])
						eepromResp.EepromData[i].CurrValue = strconv.Itoa(int(dataInt))
					}

				case 4:
					if eepromBytes[dataAddr] == 0xff && dataSubType != "ip" {
						eepromResp.EepromData[i].Permission = 0
						eepromResp.EepromData[i].CurrValue = ""
					} else {
						if dataSubType == "ip" {
							dataInt1 := int(eepromBytes[dataAddr])
							dataInt2 := int(eepromBytes[dataAddr+1])
							dataInt3 := int(eepromBytes[dataAddr+2])
							dataInt4 := int(eepromBytes[dataAddr+3])
							eepromResp.EepromData[i].CurrValue = fmt.Sprintf("%d.%d.%d.%d", dataInt1, dataInt2, dataInt3, dataInt4)
						} else {
							dataInt := int(binary.LittleEndian.Uint32(eepromBytes[dataAddr : dataAddr+4]))
							eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt)
						}
					}

				case 8:
					if eepromBytes[dataAddr] == 0xff {
						eepromResp.EepromData[i].Permission = 0
						eepromResp.EepromData[i].CurrValue = ""
					} else {
						dataInt := int(binary.LittleEndian.Uint64(eepromBytes[dataAddr : dataAddr+8]))
						eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt)
					}
				}

			case "double":
				switch dataSize {
				case 8:
					dataInt64 := int64(binary.LittleEndian.Uint64(eepromBytes[dataAddr : dataAddr+8]))
					dataDouble := math.Float64frombits(uint64(dataInt64))
					fmt.Printf("Converted data value: %f\n", dataDouble)
					fmt.Printf("8 Data 结果: %f\n", dataDouble)
					eepromResp.EepromData[i].CurrValue = strconv.FormatFloat(dataDouble, 'f', 4, 64)

				case 4:
					dataUint32 := binary.LittleEndian.Uint32(eepromBytes[dataAddr : dataAddr+4])
					fmt.Printf("4 Data in hexadecimal: %x\n", dataUint32)
					dataDouble := math.Float32frombits(dataUint32)
					eepromResp.EepromData[i].CurrValue = strconv.FormatFloat(float64(dataDouble), 'f', 2, 32)
				}

			case "string":
				if eepromBytes[dataAddr] == 0xff {
					eepromResp.EepromData[i].Permission = 0
					eepromResp.EepromData[i].CurrValue = ""
				} else {
					dataStr := string(eepromBytes[dataAddr : dataAddr+dataSize])
					eepromResp.EepromData[i].CurrValue = dataStr
				}
			}
		}
	}

	return eepromResp, true
}

// func paserEepromData(eepromBytes []byte, eepromResp EepromDataResp, startAddr int) (EepromDataResp, bool) {
// 	var dataAddr int
// 	var dataSize int
// 	var dataType string
// 	var dataSubType string

// 	for i := 0; i < len(eepromResp.EepromData); i++ {
// 		dataAddr = eepromResp.EepromData[i].Addr
// 		dataSize = eepromResp.EepromData[i].Size
// 		dataType = eepromResp.EepromData[i].Type
// 		dataSubType = eepromResp.EepromData[i].SubType

// 		if dataAddr >= startAddr && (dataAddr-startAddr) < len(eepromBytes) && len(eepromBytes) > dataAddr+dataSize {

// 			if dataType == "int" {
// 				if dataSize == 1 {
// 					if eepromBytes[dataAddr] == 0xff {
// 						eepromResp.EepromData[i].Permission = 0
// 						eepromResp.EepromData[i].CurrValue = ""
// 					} else {
// 						dataInt := int(eepromBytes[dataAddr])
// 						eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt)
// 					}

// 				} else if dataSize == 2 {
// 					if eepromBytes[dataAddr] == 0xff {
// 						eepromResp.EepromData[i].Permission = 0
// 						eepromResp.EepromData[i].CurrValue = ""
// 					} else {
// 						dataInt := binary.LittleEndian.Uint16(eepromBytes[dataAddr:(dataAddr + 2)])
// 						eepromResp.EepromData[i].CurrValue = strconv.Itoa(int(dataInt))
// 					}

// 				} else if dataSize == 4 {
// 					if eepromBytes[dataAddr] == 0xff && eepromResp.EepromData[i].SubType != "ip" {
// 						eepromResp.EepromData[i].Permission = 0
// 						eepromResp.EepromData[i].CurrValue = ""
// 					} else {
// 						if dataSubType == "ip" {
// 							dataInt1 := int(eepromBytes[dataAddr])
// 							dataInt2 := int(eepromBytes[(dataAddr + 1)])
// 							dataInt3 := int(eepromBytes[(dataAddr + 2)])
// 							dataInt4 := int(eepromBytes[(dataAddr + 3)])
// 							eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt1) + "." + strconv.Itoa(dataInt2) + "." + strconv.Itoa(dataInt3) + "." + strconv.Itoa(dataInt4)

// 						} else {
// 							dataInt := int(binary.LittleEndian.Uint32(eepromBytes[dataAddr:(dataAddr + 4)]))
// 							eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt)
// 						}
// 					}
// 				} else if dataSize == 8 {
// 					if eepromBytes[dataAddr] == 0xff {
// 						eepromResp.EepromData[i].Permission = 0
// 						eepromResp.EepromData[i].CurrValue = ""
// 					} else {
// 						dataInt := int(binary.LittleEndian.Uint64(eepromBytes[dataAddr:(dataAddr + 8)]))
// 						eepromResp.EepromData[i].CurrValue = strconv.Itoa(dataInt)
// 					}
// 				}

// 			} else if dataType == "double" {

// 				if dataSize == 8 {
// 					dataInt64 := int64(binary.LittleEndian.Uint64(eepromBytes[dataAddr:(dataAddr + 8)]))
// 					dataDouble := math.Float64frombits(uint64(dataInt64))
// 					fmt.Printf("Converted data value: %f\n", dataDouble)
// 					fmt.Printf("8 Data 结果: %f\n", dataDouble)
// 					eepromResp.EepromData[i].CurrValue = strconv.FormatFloat(dataDouble, 'f', 4, 64)
// 				} else if dataSize == 4 {
// 					dataUint32 := (binary.LittleEndian.Uint32(eepromBytes[dataAddr:(dataAddr + 4)]))
// 					fmt.Printf("4 Data in hexadecimal: %x\n", dataUint32)
// 					dataDouble := math.Float32frombits(dataUint32)
// 					eepromResp.EepromData[i].CurrValue = strconv.FormatFloat(float64(dataDouble), 'f', 2, 32)
// 				}

// 			} else if dataType == "string" {
// 				if eepromBytes[dataAddr] == 0xff {
// 					eepromResp.EepromData[i].Permission = 0
// 					eepromResp.EepromData[i].CurrValue = ""
// 				} else {
// 					dataStr := string(eepromBytes[dataAddr:(dataAddr + dataSize)])
// 					eepromResp.EepromData[i].CurrValue = dataStr
// 				}

// 			}

// 		}

// 	}

// 	return eepromResp, true

// }

type SerialOutputInfo struct {
	Id   [1]byte // 序号
	Data []byte  // 内容
}

func initSerialOutputInfo() []SerialOutputInfo {
	return []SerialOutputInfo{
		{[1]byte{1}, nil},
		{[1]byte{2}, nil},
		{[1]byte{3}, nil},
		{[1]byte{4}, nil},
		{[1]byte{5}, nil},
		{[1]byte{6}, nil},
	}

}

func ReqSetOutputFmt(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	//先获取秤上的地址
	reg, err, res := openFactory(c)
	if err != nil || !res {
		return reg, err
	}
	composer := c.composer
	fn := composer.ComposeCmd

	reqMsg, _ := excuteSimpCmd(c, m.CMD_GET_SCALE_INFO, m.GET_SCALE_INFO_RESP)
	var scaleInfo SIFromScale
	serialOutputAddr := 0
	serialOutputMaxLenth := 0
	strData := reqMsg.MsgBody
	if str, ok := strData.(string); ok {
		if err := json.UnmarshalFromString(str, &scaleInfo); err != nil {
			l.Log.Error(err)
			return &ScaleRespMsg{}, fmt.Errorf("get serial format address fail")
		}
	} else {
		return &ScaleRespMsg{}, fmt.Errorf("get serial format address fail")
	}
	siAddrInfos := scaleInfo.AddrInfos
	fmt.Println(siAddrInfos)
	for _, jsonStr := range siAddrInfos {
		addrInfo, err := parseJSON(jsonStr)
		if err != nil {
			return &ScaleRespMsg{}, fmt.Errorf("get serial format address fail")
		}
		if addrInfo.Type == SI_SERIAL_OUTPUT {
			serialOutputAddr = addrInfo.Addr
			serialOutputMaxLenth = addrInfo.Lenth
		}
	}
	if serialOutputAddr == 0 || serialOutputMaxLenth == 0 {
		return &ScaleRespMsg{}, fmt.Errorf("get serial format address fail")
	}
	//开始分析json
	if req.ReqData == "" {
		return &ScaleRespMsg{}, nil
	}
	recvFileNames := req.ReqData
	fileDataInfoList := initSerialOutputInfo()
	//1 解出路径
	var fileList ReqSerialFileList
	err = json.Unmarshal([]byte(recvFileNames), &fileList)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
	//2 根据路径分别作出完整的输出格式
	for i := 0; i < len(fileList.Paths); i++ {
		var fileNameStr = fileList.Paths[i]
		if len(fileNameStr) > 0 {
			fileId := int(fileNameStr[0] - '0')
			filePathStr := fileNameStr[1:]
			if csvFmtContent, err := os.ReadFile(filePathStr); err != nil {
				return &ScaleRespMsg{}, err
			} else {
				if bufferData, res := ParserSerialOutputFile(string(csvFmtContent)); !res {
					return &ScaleRespMsg{}, err
				} else {
					fileDataInfoList[fileId-1].Data = bufferData.Bytes()
				}
			}
		}
	}
	//3 删除当前程序下的output.bin  先屏蔽
	// if err := deleteFile("output.bin"); err != nil {
	// 	return &ScaleRespMsg{}, err
	// }
	//4 将所有的格式合并为一个bin
	var totalOutputBuffer bytes.Buffer
	outputAddr := 0
	outputAddr = outputAddr + 12 //12个字节是开头地址的位置
	//json数据不为空则填实际地址，为空则补2个字节的0
	for i := 0; i < len(fileDataInfoList); i++ {
		lengthBytes := make([]byte, 2)
		if fileDataInfoList[i].Data != nil {
			binary.LittleEndian.PutUint16(lengthBytes, uint16(outputAddr))
			totalOutputBuffer.Write(lengthBytes)
			outputAddr = outputAddr + len(fileDataInfoList[i].Data)
		} else {
			binary.LittleEndian.PutUint16(lengthBytes, uint16(0))
			totalOutputBuffer.Write(lengthBytes)
		}
	}
	for i := 0; i < 6; i++ {
		if fileDataInfoList[i].Data != nil {
			totalOutputBuffer.Write(fileDataInfoList[i].Data)
		}
	}
	binPath := WriteDataToBin(totalOutputBuffer)

	if true { //TODO:
		// 读取bin文件
		data, err := os.ReadFile(binPath)
		if err != nil {
			l.Log.Errorf("WriteDataToBinAndErase read file error: %v", err)
			return &ScaleRespMsg{}, err
		}

		// 擦除原本秤上的打印格式
		l.Log.Debug("erase flash on scale")
		addr := serialOutputAddr
		size := serialOutputMaxLenth
		loopCnt := size / 2048
		addrInLoop := addr
		for i := 0; i < loopCnt; i++ {
			cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_ERASE_FLASH, m.CmdData{Type: m.DATA_TYPE_INT, Data: addr})
			if err != nil {
				return &ScaleRespMsg{}, err
			}

			if res, err := perfCmdNwaitResult(c, cmd, m.ERASE_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("erase fail")
			}
			addrInLoop += 2048
		}

		// 计算数据包数量
		packetCount := len(data) / DATA_LENGTH_256_TMAX
		if len(data)%DATA_LENGTH_256_TMAX != 0 {
			packetCount += 1
		}
		// 遍历所有数据包
		l.Log.Debug("send data package to scale")
		for i := 0; i < packetCount; i++ {
			// 计算本包数据
			start := i * DATA_LENGTH_256_TMAX
			end := start + DATA_LENGTH_256_TMAX
			if end > len(data) {
				end = len(data)
			}
			packetData := data[start:end]
			// 构建数据包
			// dataPackCmd := buildSendDataPacket(addr, packetData)
			packDataHexStr := hex.EncodeToString(packetData)
			cmd, timeoutMs, err := fn(composer, m.CMD_WRITE_FLASH_256, m.CmdData{Type: m.DATA_TYPE_STR, Data: fmt.Sprintf("%08x:%s", addr, packDataHexStr)})
			if err != nil {
				return &ScaleRespMsg{}, err
			}
			// 发送数据包
			if res, err := perfCmdNwaitResult(c, cmd, m.WRITE_DATA_FLASH_RESP, timeoutMs); err != nil {
				return &ScaleRespMsg{}, err
			} else if res.MsgBody != "ok" {
				return &ScaleRespMsg{}, fmt.Errorf("enable factory mode fail")
			}
			// 地址自增
			addr += 0x100
		}
		l.Log.Info("send bin ok")
	}
	return &ScaleRespMsg{m.SET_OUTPUT_FMT_RESP, "ok", c.Id}, nil
}

func retreiveRespMsgT2200(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	// checkHead, get msgid, get msgtype, check if return code is 0x06, for success
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
	} else { // weight data, same as C51 scale
		var msg WeightMsg
		if msg, err = retreiveWeightC51(data); err == nil {
			respMsg.MsgType = m.WEIGHT_DATA
			respMsg.MsgBody = msg
		}
	}
	return respMsg, err
}

func sendMsgIntoChsOrWeightToClient(s *Scale, msg *ScaleRespMsg) {
	if (*msg == ScaleRespMsg{}) {
		l.Log.Errorf("extractMsgTmaxScale error")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	chs := s.respChansMap[msg.MsgType]

	for _, ch := range chs {
		if len(ch) == 0 { // to avoid blocking, this kind of channel should only be used once
			ch <- msg
		}
	}

	// 记录最新重量用于 Modbus 快速查询
	if msg.MsgType == m.WEIGHT_DATA {
		if weightMsg, ok := msg.MsgBody.(WeightMsg); ok {
			if fVal, err := strconv.ParseFloat(weightMsg.WeightVal, 32); err == nil {
				s.LastWeight = float32(fVal)
			}
		} else if strMsg, ok := msg.MsgBody.(string); ok {
			// TMax scale sends JSON string
			var wMsg WeightMsg
			if err := json.UnmarshalFromString(strMsg, &wMsg); err == nil {
				if fVal, err := strconv.ParseFloat(wMsg.WeightVal, 32); err == nil {
					s.LastWeight = float32(fVal)
				}
			}
		}
	}

	if msg.MsgType == m.WEIGHT_DATA && s.isSendUnolicitedData && s.client != nil { // skip sending weight data to client if it doesn't not register this message
		sendRespMsgClient(s, msg)
		return
	}
	if msg.MsgType == m.WEIGHT_DATA && !s.isSendUnolicitedData { // skip sending weight data to client if it doesn't not register this message
		// writeScale(s, cmd.DIS_CONT_MODE_CMD_TMAX)
		if s.ScaleCat != m.SCALE_TMAX {
			return
		}
		perfCmdNwaitResult(s, cmd.DIS_CONT_MODE_CMD_TMAX, m.UNREG_WEIGHT_RESP, -1, 1)
		return
	}
	if msg.MsgType == m.SCALE_PASSTH_DATA && s.isScalePassth && s.client != nil { // skip sending weight data to client if it doesn't not register this message
		sendRespMsgClient(s, msg)
		return
	}
	if msg.MsgType == m.ERR_SERIAL_RESP && s.client != nil {

		sendRespMsgClient(s, msg) //此处单独来了串口错误功能，直接送出去
		return
	}
	if msg.MsgType == m.REV_DETAIl_TAIL_RESP && s.client != nil {
		sendRespMsgClient(s, msg) //此处单独来了明细数据
		return
	}
	if msg.MsgType == m.ANSWER_ALIVE_RESP {
		if s.ScaleCat != m.SCALE_TMAX {
			return
		}
		sendRespMsgScale(s)
		return
	}

	if msg.MsgType == m.CONT_CODE_RESP && s.client != nil { // skip sending weight data to client if it doesn't not register this message
		sendRespMsgClient(s, msg)
		return
	}

}

func sendRespMsgScale(s *Scale) {
	time.Sleep(500 * time.Millisecond)
	perfCmdNwaitResult(s, cmd.ANSWER_ALIVE_CMD_TMAX, m.ANSWER_ALIVE_RESP, -1, 1)
	time.Sleep(1 * time.Second)
	perfCmdNwaitResult(s, cmd.ANSWER_ALIVE_CMD_TMAX, m.ANSWER_ALIVE_RESP, -1, 1)

}

func sendRespMsgClient(s *Scale, msg *ScaleRespMsg) {
	if (*msg == ScaleRespMsg{}) {
		l.Log.Warnf("msg is ScaleRespMsg{}")
		return
	}

	if len(s.fromScaleMsgCh) < RECV_CH_SIZE { // no use in this moment
		if msgStr, err := json.MarshalToString(msg); err == nil {
			if s.client == nil {
				l.Log.Errorf("s.client is nil")
				return
			}

			l.Log.Tracef("%%%%%%%%%%%%%%: %s", msgStr)
			s.client.sendCh <- []byte(msgStr)
		} else {
			l.Log.Errorf("marshal msg err: %v", err.Error())
		}
	} else {
		l.Log.Error("fromScaleMsgCh full")
	}

}

// 设置最大量程
func ReqSetMaxRange1(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData
	num, err := strconv.Atoi(reqData)
	if err != nil {
		return &ScaleRespMsg{m.SET_MAX_RANGE1_RESP, "fail, data error", c.Id}, nil
	}
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_MAX_RANGE1_RESP, err.Error(), c.Id}, nil
	}
	composer := c.composer
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_SET_MAX_RANGE1, m.CmdData{Type: m.DATA_TYPE_INT, Data: num})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_MAX_RANGE1_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_MAX_RANGE1_RESP, timeoutMs)
}
func ReqSetMaxRange2(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData
	num, err := strconv.Atoi(reqData)
	if err != nil {
		return &ScaleRespMsg{m.SET_MAX_RANGE2_RESP, "fail, data error", c.Id}, nil
	}

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_MAX_RANGE2_RESP, err.Error(), c.Id}, nil
	}
	composer := c.composer
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_SET_MAX_RANGE2, m.CmdData{Type: m.DATA_TYPE_INT, Data: num})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_MAX_RANGE2_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_MAX_RANGE2_RESP, timeoutMs)
}

// 获取最大量程1
func ReqGetMaxRange1(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_MAX_RANGE1_RESP, "fail", c.Id}, nil
	}

	return excuteSimpCmd(c, m.CMD_GET_MAX_RANGE1, m.GET_MAX_RANGE1_RESP)
}

// 获取最大量程2
func ReqGetMaxRange2(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_MAX_RANGE2_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_MAX_RANGE2, m.GET_MAX_RANGE2_RESP)
}

// 标定重量
func ReqCalWeight(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_CAL_WGT_RESP, "fail", c.Id}, nil
	}
	reqData := req.ReqData
	num, err := strconv.Atoi(reqData)
	if err != nil {
		return &ScaleRespMsg{m.SET_CAL_WGT_RESP, "fail, data error", c.Id}, nil
	}
	composer := c.composer
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_CAl_WGT, m.CmdData{Type: m.DATA_TYPE_INT, Data: num})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_CAL_WGT_RESP, fmt.Errorf("fail"), c.Id}, nil

	}
	return perfCmdNwaitResult(c, cmd, m.SET_CAL_WGT_RESP, timeoutMs)

}

// 设置S15预扣重
func ReqSetPreTareS15(c *Scale, preTareStr string) (*ScaleRespMsg, error) {
	composer := c.composer
	cmd, timeoutMs, err := composer.ComposeCmd(composer, m.CMD_SET_PRE_TARE_S15, m.CmdData{Type: m.DATA_TYPE_STR, Data: preTareStr})
	if err != nil {
		return &ScaleRespMsg{m.SET_PRE_TARE_S15_RESP, fmt.Errorf("fail").Error(), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_PRE_TARE_S15_RESP, timeoutMs)
}

// 标定时发送的心跳
func ReqSendCalHeart(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	excuteSimpCmd(c, m.CMD_SEND_CAL_HEART_BEAT, m.UNKNOWN_DATA, 1)
	return &ScaleRespMsg{}, nil
}

// 设置小数点位数
func ReqSetDecimalValue(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_DECIMAL_VALUE_RESP, "fail", c.Id}, nil
	}

	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_DECIMAl_VALUE, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})

	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_DECIMAL_VALUE_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_DECIMAL_VALUE_RESP, timeoutMs)
}

// 设置分度值
func ReqSetGaduation1Value(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_GADUATION1_VALUE_RESP, "fail", c.Id}, nil
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_GADUATION1_VALUE, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_GADUATION1_VALUE_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_GADUATION1_VALUE_RESP, timeoutMs)
}

// 获取重量单位
func ReqGetWeightUnit(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_WEIGHT_UNIT_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_WEIGHT_UNIT, m.GET_WEIGHT_UNIT_RESP)

}

// 设置重量单位
func ReqSetWeightUnit(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_WEIGHT_UNIT_RESP, "fail", c.Id}, nil
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_WEIGHT_UNIT, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_WEIGHT_UNIT_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_WEIGHT_UNIT_RESP, timeoutMs)

}

// 设置分度值2
func ReqSetGaduation2Value(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_GADUATION2_VALUE_RESP, "fail", c.Id}, nil
	}

	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_GADUATION2_VALUE, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})

	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_GADUATION2_VALUE_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_GADUATION2_VALUE_RESP, timeoutMs)

}

// 获取分度值2
func ReqGetGaduation2Value(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_GADUATION2_VALUE_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_GADUATION2_VALUE, m.GET_GADUATION2_VALUE_RESP)
}

// 获取分度值1
func ReqGetGaduation1Value(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_GADUATION1_VALUE_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_GADUATION1_VALUE, m.GET_GADUATION1_VALUE_RESP)
}

// 获取小数点位数
func ReqGetDecimalValue(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_DECIMAL_VALUE_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_DECIMAL_VALUE, m.GET_DECIMAL_VALUE_RESP)
}

func ReqSetSerialPort(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData
	// _, err, res := openFactory(c)
	// if err != nil || !res {
	// 	return &ScaleRespMsg{m.SET_SERIAL_PORT_RESP, "fail", c.Id}, nil
	// }
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_SERIAL_PORT, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})
	if err != nil {
		return &ScaleRespMsg{m.SET_SERIAL_PORT_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_SERIAL_PORT_RESP, timeoutMs)
}

func ReqGetSerialPort(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	// _, err, res := openFactory(c)
	// if err != nil || !res {
	// 	return &ScaleRespMsg{m.GET_SERIAL_PORT_RESP, "fail", c.Id}, nil
	// }
	return excuteSimpCmd(c, m.CMD_GET_SERIAL_PORT, m.GET_SERIAL_PORT_RESP)
}

// 设置初始置零
func ReqSetInitialZero(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_INITIAL_ZERO_RESP, "fail", c.Id}, nil
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_INITIAL_ZERO, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_INITIAL_ZERO_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_INITIAL_ZERO_RESP, timeoutMs)

}

// 获取零点内码
func ReqGetInitialZero(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_INITIAL_ZERO_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_INITIAL_ZERO, m.GET_INITIAL_ZERO_RESP)
}

// 设置手动零点
func ReqSetManualZero(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_MANUAL_ZERO_RESP, "fail", c.Id}, nil
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_MANUAL_ZERO, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_MANUAL_ZERO_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_MANUAL_ZERO_RESP, timeoutMs)
}

// 获取手动零点
func ReqGetManualZero(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_MANUAL_ZERO_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_MANUAL_ZERO, m.GET_MANUAL_ZERO_RESP)
}

// 设置零点跟踪
func ReqSetZeroTracking(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData

	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_ZERO_TRACKING_RESP, "fail", c.Id}, nil
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_ZERO_TRACKING, m.CmdData{Type: m.DATA_TYPE_STR, Data: reqData})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_ZERO_TRACKING_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_ZERO_TRACKING_RESP, timeoutMs)
}

// 获取零点跟踪
func ReqGetZeroTracking(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_ZERO_TRACKING_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_ZERO_TRACKING, m.GET_ZERO_TRACKING_RESP)
}

// 设置重力加速度
func ReqSetGravAcc(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	reqData := req.ReqData
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_GRAV_ACC_RESP, "fail", c.Id}, nil
	}
	num, err := strconv.Atoi(reqData)
	if err != nil {
		return &ScaleRespMsg{m.SET_GRAV_ACC_RESP, "fail, data error", c.Id}, nil
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_GRAV_ACC, m.CmdData{Type: m.DATA_TYPE_INT, Data: num})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_GRAV_ACC_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_GRAV_ACC_RESP, timeoutMs)
}

// 获取重力加速度
func ReqGetGravAcc(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_GRAV_ACC_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_GRAV_ACC, m.GET_GRAV_ACC_RESP)
}

func convertIPConfigToHex(inInfo IpInfo) (string, error) {
	// 解析 IP 地址
	ip := net.ParseIP(inInfo.Ip)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", inInfo.Ip)
	}
	ip = ip.To4()
	if ip == nil {
		return "", fmt.Errorf("IP address is not IPv4: %s", inInfo.Ip)
	}

	// 解析网关
	gateway := net.ParseIP(inInfo.Gateway)
	if gateway == nil {
		return "", fmt.Errorf("invalid gateway: %s", inInfo.Gateway)
	}
	gateway = gateway.To4()
	if gateway == nil {
		return "", fmt.Errorf("gateway is not IPv4: %s", inInfo.Gateway)
	}

	// 解析子网掩码（正确方式）
	var mask net.IPMask
	if strings.Contains(inInfo.Netmask, ".") {
		// 点分十进制格式：255.255.255.0
		maskIP := net.ParseIP(inInfo.Netmask)
		if maskIP == nil {
			return "", fmt.Errorf("invalid netmask: %s", inInfo.Netmask)
		}
		mask = net.IPMask(maskIP.To4())
		if mask == nil {
			return "", fmt.Errorf("netmask is not IPv4: %s", inInfo.Netmask)
		}
	} else {
		// CIDR 格式：24
		prefixLen, err := strconv.Atoi(inInfo.Netmask)
		if err != nil || prefixLen < 0 || prefixLen > 32 {
			return "", fmt.Errorf("invalid netmask prefix: %s", inInfo.Netmask)
		}
		mask = net.CIDRMask(prefixLen, 32)
	}

	// 构建字节数组
	data := make([]byte, 0, 12) // 3个IPv4地址 = 12字节
	data = append(data, ip...)
	data = append(data, gateway...)
	data = append(data, mask...)

	// 转为十六进制字符串
	dataByteStr := fmt.Sprintf("%x", data)
	return dataByteStr, nil
}

func ReqSetWiredIp(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_WIRED_IP_RESP, "fail", c.Id}, nil
	}
	inInfo := IpInfo{}

	err = json.Unmarshal([]byte(req.ReqData), &inInfo)
	if err != nil {
		return &ScaleRespMsg{m.SET_WIRED_IP_RESP, fmt.Errorf("fail, data error"), c.Id}, nil
	}

	dataByteStr, _ := convertIPConfigToHex(inInfo)

	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_WIRED_IP, m.CmdData{Type: m.DATA_TYPE_STR, Data: dataByteStr})

	println(fmt.Sprintf("%x", cmd))

	if err != nil {
		return &ScaleRespMsg{m.SET_WIRED_IP_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	return perfCmdNwaitResult(c, cmd, m.SET_WIRED_IP_RESP, timeoutMs)
}

func ReqSetWiredDhcp(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_WIRED_DHCP_RESP, "fail", c.Id}, nil
	}

	switch req.ReqData {
	case "true":
		cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_WIRED_DHCP, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x01})
		println(fmt.Sprintf("%x", cmd))
		if err != nil {
			return &ScaleRespMsg{m.SET_WIRED_DHCP_RESP, fmt.Errorf("fail"), c.Id}, nil
		}
		return perfCmdNwaitResult(c, cmd, m.SET_WIRED_DHCP_RESP, timeoutMs)
	case "false":
		cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_WIRED_DHCP, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
		println(fmt.Sprintf("%x", cmd))
		if err != nil {
			return &ScaleRespMsg{m.SET_WIRED_DHCP_RESP, fmt.Errorf("fail"), c.Id}, nil
		}
		return perfCmdNwaitResult(c, cmd, m.SET_WIRED_DHCP_RESP, timeoutMs)
	}

	return &ScaleRespMsg{m.SET_WIRED_DHCP_RESP, fmt.Errorf("fail"), c.Id}, nil
}
func ReqGetWiredDhcp(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_WIRED_DHCP_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_WIRED_DHCP, m.GET_WIRED_DHCP_RESP)
}

func ReqInitWifiAPListRef(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	//设置扫描AP参数
	GExpectWifiResp = m.SET_SCAN_AP_PARAM_CMD_RESP

	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_SCAN_AP_PARAM_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_SCAN_AP_PARAM_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_SCAN_AP_PARAM_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}

	return &ScaleRespMsg{m.INIT_WIFI_RESP, "ok", c.Id}, nil

}

func ReqInitWifi(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	//关闭TCP服务器
	GExpectWifiResp = m.CLOSE_SERVER_CMD_RESP

	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_WIFI_CLOSE_SERVER_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.CLOSE_SERVER_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.CLOSE_SERVER_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}

	//注销蓝牙
	GExpectWifiResp = m.DIS_BT_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_DIS_BT_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.DIS_BT_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.DIS_BT_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//打开自动连接
	GExpectWifiResp = m.EN_AUTO_CONN_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_EN_AUTO_CONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.EN_AUTO_CONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.EN_AUTO_CONN_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//设置Wi-Fi模式为station
	GExpectWifiResp = m.SET_WIFI_STATION_MODE_CMD_RESP
	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_WIFI_STATION_MODE_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_WIFI_STATION_MODE_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_WIFI_STATION_MODE_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	// //设置多连接
	// GExpectWifiResp = m.SET_MULTI_CONN_CMD_RESP
	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_MULTI_CONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.SET_MULTI_CONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.SET_MULTI_CONN_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	// //设置单连接
	// GExpectWifiResp = m.SET_SINGLE_CONN_CMD_RESP

	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_SINGLE_CONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.SET_SINGLE_CONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.SET_SINGLE_CONN_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	// //断开重连
	// GExpectWifiResp = m.DIS_RECONN_CMD_RESP
	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_DIS_RECONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.DIS_RECONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.DIS_RECONN_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	// //不提示对端IP及端口号
	// GExpectWifiResp = m.DIS_IP_PORT_INFO_CMD_RESP
	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_DIS_IP_PORT_INFO_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.DIS_IP_PORT_INFO_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.DIS_IP_PORT_INFO_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	// //设置端口号
	// GExpectWifiResp = m.SET_TCP_SERVER_CMD_RESP

	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_CONN_PORT_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.SET_TCP_SERVER_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.SET_TCP_SERVER_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	// //设置本地TCP服务器超时
	// GExpectWifiResp = m.SET_TIME_OUT_CMD_RESP

	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_TIME_OUT_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.SET_TIME_OUT_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.SET_TIME_OUT_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	// //设置传输模式 0-普通 1-透传
	// GExpectWifiResp = m.SET_PASSTH_MODE_CMD_RESP

	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_PASSTH_MODE_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.SET_PASSTH_MODE_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.SET_PASSTH_MODE_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	//设置扫描AP参数
	GExpectWifiResp = m.SET_SCAN_AP_PARAM_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_SCAN_AP_PARAM_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_SCAN_AP_PARAM_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_SCAN_AP_PARAM_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}

	return &ScaleRespMsg{m.INIT_WIFI_RESP, "ok", c.Id}, nil

}

func ReqSetServerMode(c *Scale, req SRequest) (*ScaleRespMsg, error) {

	//设置多连接
	GExpectWifiResp = m.SET_MULTI_CONN_CMD_RESP
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_MULTI_CONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_MULTI_CONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_MULTI_CONN_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//设置单连接
	GExpectWifiResp = m.SET_SINGLE_CONN_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_SINGLE_CONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_SINGLE_CONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_SINGLE_CONN_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//断开重连
	GExpectWifiResp = m.DIS_RECONN_CMD_RESP
	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_DIS_RECONN_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.DIS_RECONN_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.DIS_RECONN_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//不提示对端IP及端口号
	GExpectWifiResp = m.DIS_IP_PORT_INFO_CMD_RESP
	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_DIS_IP_PORT_INFO_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.DIS_IP_PORT_INFO_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.DIS_IP_PORT_INFO_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//设置端口号
	GExpectWifiResp = m.SET_TCP_SERVER_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_CONN_PORT_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_TCP_SERVER_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_TCP_SERVER_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//设置本地TCP服务器超时
	GExpectWifiResp = m.SET_TIME_OUT_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_TIME_OUT_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_TIME_OUT_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_TIME_OUT_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	//设置传输模式 0-普通 1-透传
	GExpectWifiResp = m.SET_PASSTH_MODE_CMD_RESP

	cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_PASSTH_MODE_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	println(fmt.Sprintf("%x", cmd))
	if err != nil {
		return &ScaleRespMsg{m.SET_PASSTH_MODE_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	}
	if res, err := perfCmdNwaitResult(c, cmd, m.SET_PASSTH_MODE_CMD_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{}, err
	} else if res.MsgBody != "ok" {
		return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	}
	// //设置扫描AP参数
	// GExpectWifiResp = m.SET_SCAN_AP_PARAM_CMD_RESP

	// cmd, timeoutMs, err = c.composer.ComposeCmd(c.composer, m.CMD_WIFI_SET_SCAN_AP_PARAM_CMD, m.CmdData{Type: m.DATA_TYPE_INT, Data: 0x00})
	// println(fmt.Sprintf("%x", cmd))
	// if err != nil {
	// 	return &ScaleRespMsg{m.SET_SCAN_AP_PARAM_CMD_RESP, fmt.Errorf("fail"), c.Id}, nil
	// }
	// if res, err := perfCmdNwaitResult(c, cmd, m.SET_SCAN_AP_PARAM_CMD_RESP, timeoutMs); err != nil {
	// 	return &ScaleRespMsg{}, err
	// } else if res.MsgBody != "ok" {
	// 	return &ScaleRespMsg{m.INIT_WIFI_RESP, "fail", c.Id}, nil
	// }
	return &ScaleRespMsg{m.INIT_WIFI_RESP, "ok", c.Id}, nil
}

func ReqGetWiredIp(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_WIRED_IP_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_WIRED_IP, m.GET_WIRED_IP_RESP)
}

// 强制解除扣重
func ReqSetForceUnTare(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SET_FORCE_UNTARE_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_SET_FORCE_UNTARE, m.SET_FORCE_UNTARE_RESP)
}

// 获取封印状态
func ReqGetSealStatus(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_SEAL_STATUS_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_SEAL_STATUS, m.GET_SEAL_STATUS_RESP)
}

// 获取型号
func ReqGetModel(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.GET_MODEL_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_GET_MODEL, m.GET_MODEL_RESP)
}

// 开启打开内码
func ReqEnCode(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.EN_CODE_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_EN_CODE, m.EN_CODE_RESP)
}

// 关闭内码
func ReqDisCode(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	// _, err, res := openFactory(c)
	// if err != nil || !res {
	// 	return &ScaleRespMsg{m.DIS_CODE_RESP, "fail", c.Id}, nil
	// }
	return excuteSimpCmd(c, m.CMD_DIS_CODE, m.DIS_CODE_RESP)
}

// 询问Tmax rom 版本号
func ReqAskRomVersion(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.ASK_ROM_VERSION_RESP, "fail", c.Id}, nil
	}
	return excuteSimpCmd(c, m.CMD_ASK_ROM_VERSION, m.ASK_ROM_VERSION_RESP)
}

func ReqSoftSeal(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.SOFT_SEAL_RESP, "fail", c.Id}, nil
	}

	dataStr := req.ReqData
	var hexStr string
	for _, ch := range dataStr {
		hexStr += fmt.Sprintf("%02X", ch-'0') // ch-'0' 将字符转换为数字值
	}

	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_SET_SOFT_SEAL, m.CmdData{Type: m.DATA_TYPE_STR, Data: hexStr})

	if res, err := perfCmdNwaitResult(c, cmd, m.SOFT_SEAL_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{m.SOFT_SEAL_RESP, "fail", c.Id}, nil
	} else if res.MsgBody == "ok" {
		_, username, roleID := GetCurrentUser()
		SealLog := SealLog{
			ScaleId:   int(c.Id),
			Model:     c.Model,
			Sn:        c.Sn,
			Result:    "ok",
			Operation: "seal", //seal  unseal
			RoleId:    roleID,
			Operator:  username,
			Remark:    "",
		}
		c.scaleMgr.srvMgr.sysLogPd.AddSealLog(SealLog)
		return &ScaleRespMsg{m.SOFT_SEAL_RESP, "ok", c.Id}, nil
	}
	return &ScaleRespMsg{m.SOFT_SEAL_RESP, "fail", c.Id}, nil
}

func ReqRemoveSoftSeal(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_RESP, "fail", c.Id}, nil
	}
	dataStr := req.ReqData
	var hexStr string
	for _, ch := range dataStr {
		hexStr += fmt.Sprintf("%02X", ch-'0') // ch-'0' 将字符转换为数字值
	}
	cmd, timeoutMs, err := c.composer.ComposeCmd(c.composer, m.CMD_REMOVE_SOFT_SEAL, m.CmdData{Type: m.DATA_TYPE_STR, Data: hexStr})
	println(fmt.Sprintf("%x", cmd))
	if res, err := perfCmdNwaitResult(c, cmd, m.REMOVE_SOFT_SEAL_RESP, timeoutMs); err != nil {
		return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_RESP, "fail", c.Id}, nil
	} else if res.MsgBody == "ok" {
		_, username, roleID := GetCurrentUser()
		SealLog := SealLog{
			ScaleId:   int(c.Id),
			Model:     c.Model,
			Sn:        c.Sn,
			Result:    "ok",
			Operation: "unseal", //seal  unseal
			RoleId:    roleID,
			Operator:  username,
			Remark:    "",
		}
		c.scaleMgr.srvMgr.sysLogPd.AddSealLog(SealLog)
		return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_RESP, "ok", c.Id}, nil
	}
	return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_RESP, "fail", c.Id}, nil
}

func ReqRemoveSoftSealOnce(c *Scale, req SRequest) (*ScaleRespMsg, error) {
	_, err, res := openFactory(c)
	if err != nil || !res {
		return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_ONCE_RESP, "fail", c.Id}, nil
	}
	reqMsg, _ := excuteSimpCmd(c, m.CMD_REMOVE_SOFT_SEAL_ONCE, m.REMOVE_SOFT_SEAL_ONCE_RESP)
	if reqMsg.MsgBody == "ok" {
		_, username, roleID := GetCurrentUser()
		SealLog := SealLog{
			ScaleId:   int(c.Id),
			Model:     c.Model,
			Sn:        c.Sn,
			Result:    "ok",
			Operation: "unseal", //seal  unseal
			RoleId:    roleID,
			Operator:  username,
			Remark:    "key",
		}
		c.scaleMgr.srvMgr.sysLogPd.AddSealLog(SealLog)
		return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_ONCE_RESP, "ok", c.Id}, nil
	}
	return &ScaleRespMsg{m.REMOVE_SOFT_SEAL_ONCE_RESP, "fail", c.Id}, nil
}

func retreiveRespMsgC51(scaleId int64, data []byte) (*ScaleRespMsg, error) {
	// checkHead, get msgid, get msgtype, check if return code is 0x06, for success
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
	} else { // weight data, same as C51 scale
		var msg WeightMsg
		if msg, err = retreiveWeightNewC51(data); err == nil {
			respMsg.MsgType = m.WEIGHT_DATA
			respMsg.MsgBody = msg
		}
	}
	return respMsg, err
}
