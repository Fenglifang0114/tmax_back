package svc

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"tmaxsrv/comm" // for the message types.  It is not a direct part of the code.  It is a "hel
	"tmaxsrv/lic"
	l "tmaxsrv/log"
	"tmaxsrv/util"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var SERVICE_ID int64 = 999999900 //小服务的ID是从999999900开始的

type SrvMgr struct {
	scaleMgr *ScaleMgr
	// Registered clients.
	clients map[*Client]bool
	// Inbound messages from the clients.
	recvWsClientMsg chan []byte
	// Register requests from the clients.
	register chan *Client
	// Unregister requests from clients.
	unregister chan *Client
	// add a new scale
	addScale chan *Scale
	// remove a scale
	removeScale chan *Scale
	// Inbound messages from the scales.
	recvScaleMsg chan *ScaleRespMsg
	// Inbound messages from the scale manager
	recvScaleMgrMsg    chan *ScaleMgrRespMsg
	recvScaleMgrMsgSrv chan *SrvMgrRespMsg
	// Inbound messages from the scale's notification
	recvScaleNotifyMsg chan ScaleRespMsg
	// ScaleId to Scale map, used to access the scale
	scales map[int64]*Scale
	// scale to client map, used to send message from scale to the associated client
	clientOfScales map[*Scale]*Client
	//连接的服务器
	clientOfService map[int64]*Client
	// quitch channel to close this application
	quitch chan bool
	// productPb
	productPd *ProductRecProvider
	userPd    *UserRecProvider
	wifiPd    *WifiRecProvider
	// formulaPd
	formulaPd   *FormulaRecProvider
	flowRatePd  *FlowRateProvider
	uiConfig    *UiConfig
	modeSetting *ModeSettingProvider
	sysUserPd   *SysUserProvider
	sysLogPd    *LogRecProvider

	//服务与秤的关系
	srvScaleRel []*SrvScaleRel

	// ========== 新增串口管理字段 ==========

	// 串口实例
	serialPort *SerialPort

	// 串口配置
	serialConfig *SerialConfigWrapper

	// 串口命令通道
	serialCmdChan chan *SerialCommand

	// 串口数据通道
	serialDataChan chan *SerialDataMessage

	//串口分析接收的输入状态
	inputBuffer chan []byte // 全局或函数内维持的缓冲区

	parseBuffer []byte     // 用于累积数据的缓冲区
	parseMu     sync.Mutex // 保护 parseBuffer（如果可能被多个 goroutine 访问）
	lastStates  [4]bool    // 上次开关状态，用于检测下降沿

	// 串口状态
	serialStatus struct {
		IsOpen        bool
		LastError     string
		LastErrorTime time.Time
		BytesRead     uint64
		BytesWritten  uint64
		PacketsRecv   uint64
	}

	// 串口读写锁
	serialMu sync.RWMutex

	// 串口数据处理器映射
	serialHandlers map[string]func(*SerialDataMessage)

	// 配置文件路径
	configPath string

	// 是否自动打开串口
	autoOpenSerial bool

	// Modbus 网关管理器
	ModbusGateway *ModbusGatewayManager
}

var SrvIdList []int64 = []int64{999999999}

const (
	SRV_STATUS_UNINSTALLED = "status1" //服务未安装
	SRV_STATUS_INSTALLED   = "status2" //服务已安装  服务未启动
	SRV_STATUS_STARTED     = "status3" //服务已安装 服务已启动
)

type LicenseInfo struct {
	Id         string
	ModuleName string
	IsValid    bool
	ValidDate  string
}

var (
	gIsKeyValid      bool
	gMachineId       string
	gLicValidDate    string
	gModuleName      string
	gLicenseInfoList []LicenseInfo
)

func NewSrvMgr(scaleMgr *ScaleMgr, quitch chan bool) *SrvMgr {
	productPb := NewProductRecProvider()
	userPb := NewUserRecProvider()
	modeSettingPb := NewModeSettingProvider()
	wifiPb := NewWifiRecProvider()
	formulaPb := NewFormulaRecProvider()
	flowRatePb := NewFlowRateProvider()
	sysUserPb := NewSysUserProvider()
	sysLogPb := NewLogRecProvider()

	var licKey string

	var licKeyList []string

	licFilePath := filepath.Join(comm.GetExePath(), comm.LICENSE_FILE)

	licKey, _ = lic.ReadLicFile(licFilePath)

	licKeyList = strings.Split(licKey, "\r\n")
	for _, item := range licKeyList {
		if len(item) == 74 || len(item) == 78 || strings.Contains(item, "++==") {
			gIsKeyValid, gMachineId, gLicValidDate, gModuleName = lic.IsKeyValid(item)
			if gIsKeyValid {
				gLicenseInfoList = append(gLicenseInfoList, LicenseInfo{Id: gMachineId, ValidDate: gLicValidDate, ModuleName: gModuleName, IsValid: gIsKeyValid})
			}
		}
	}

	if len(gLicenseInfoList) == 0 {
		var temp []string
		licKeyList = temp
		gIsKeyValid, gMachineId, gLicValidDate, gModuleName = lic.IsKeyValid("d7a0a41239d92ee1724cd1a311ffffff2023-05-2594df26ebd828dbff03ede5f76effffff")
		gLicenseInfoList = append(gLicenseInfoList, LicenseInfo{Id: gMachineId, ValidDate: gLicValidDate, ModuleName: gModuleName, IsValid: gIsKeyValid})

	}

	configPath := "config.yaml" // 默认配置文件路径
	configPath = filepath.Join(comm.GetExePath(), configPath)

	l.Log.Debugf("configPath yaml:%s\n", configPath)

	sm := &SrvMgr{
		scaleMgr:           scaleMgr,
		scales:             map[int64]*Scale{},
		clientOfScales:     map[*Scale]*Client{},
		clientOfService:    map[int64]*Client{},
		recvWsClientMsg:    make(chan []byte),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		addScale:           make(chan *Scale, 2),
		removeScale:        make(chan *Scale, 2),
		recvScaleMsg:       make(chan *ScaleRespMsg, 2),
		recvScaleMgrMsg:    make(chan *ScaleMgrRespMsg, 2),
		recvScaleMgrMsgSrv: make(chan *SrvMgrRespMsg, 2),
		recvScaleNotifyMsg: make(chan ScaleRespMsg, 2),
		clients:            make(map[*Client]bool),
		quitch:             quitch,
		productPd:          productPb,
		userPd:             userPb,
		wifiPd:             wifiPb,
		formulaPd:          formulaPb,
		flowRatePd:         flowRatePb,
		uiConfig:           NewUiConfig(),
		modeSetting:        modeSettingPb,
		sysUserPd:          sysUserPb,
		sysLogPd:           sysLogPb,
		srvScaleRel:        make([]*SrvScaleRel, 0),

		// 串口相关初始化
		serialPort:     NewSerialPort(configPath),
		serialConfig:   &SerialConfigWrapper{},
		serialCmdChan:  make(chan *SerialCommand, 50),
		serialDataChan: make(chan *SerialDataMessage, 200),
		serialHandlers: make(map[string]func(*SerialDataMessage)),
		configPath:     configPath,
		autoOpenSerial: true,
		lastStates:     [4]bool{false, false, false, false},
		parseBuffer:    make([]byte, 2048),
		parseMu:        sync.Mutex{},
		inputBuffer:    make(chan []byte, 1024),
	}

	// 加载配置
	if err := sm.LoadSerialConfig(); err != nil {
		fmt.Printf("加载串口配置失败: %v\n", err)
	}

	// 设置串口事件监听
	sm.setupSerialEventListeners()

	sm.StartParsing()

	// 启动查询协程
	go sm.queryLoop()

	// 启动串口管理协程
	go sm.serialManager()

	// 启动串口数据分发协程
	go sm.serialDataDispatcher()

	// 注册默认处理器
	sm.registerDefaultHandlers()

	// 初始化并启动 Modbus 网关
	sm.ModbusGateway = NewModbusGatewayManager(sm)
	sm.ModbusGateway.StartAll()

	// 如果配置了自动打开，则打开串口

	l.Log.Debugf("sm.autoOpenSerial:%v\n", sm.autoOpenSerial)
	if sm.autoOpenSerial {
		go func() {
			time.Sleep(1 * time.Second) // 等待系统初始化完成
			if err := sm.OpenSerial(); err != nil {
				fmt.Printf("自动打开串口失败: %v\n", err)
				l.Log.Debugf("自动打开串口失败:%s\n", err)
			}
		}()
	}

	return sm

}

func (h *SrvMgr) Run() {
	for {
		select {
		case client := <-h.register:
			scaleId := client.scaleId
			if scaleId > 0 && scaleId < SERVICE_ID { // scaleId 0 is for management
				scale := h.scales[scaleId]
				if scale == nil {
					l.Log.Errorf("The scale: %v is not existed", scaleId)
					break
				}
			}
			isRegisted := false
			for client := range h.clients {
				if client.scaleId == scaleId {
					l.Log.Errorf("The scale: %v is already registered", scaleId)
					isRegisted = true
					break
				}
			}
			if !isRegisted {
				h.clients[client] = true
				// if scaleId != 0 { // 0 reserved for common information channel, 9999 reserved for legacy MCU scale, only support one scale with this ID
				if scaleId < SERVICE_ID {
					h.clientOfScales[h.scales[scaleId]] = client
				}
				if scaleId != 0 && scaleId < SERVICE_ID { // id 0 is reserved for common information channel
					h.scales[scaleId].SetClient(client)
				}

				if scaleId > SERVICE_ID { // id 0 is reserved for common information channel
					h.clientOfService[scaleId] = client
				}

				// h.clientOfScales[h.scales[scaleId]] = client
				//}
			} else {
				// close(client.send) // either other client is already registered the scale or the scale is not yet registered
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				// TODO: handle client disconnect
				//0 通道也断了，如何处理，先将下面的程序改为0的话断掉也直接断开。
				if client.scaleId != -1 && client.scaleId != 0 { //20250220    -1 原来是 0
					if h.scales[client.scaleId] != nil {
						h.scales[client.scaleId].HandleClientDisconnect()
					}

				}
				// client.Close()
				delete(h.clients, client)
				if client.scaleId > SERVICE_ID {
					delete(h.clientOfService, client.scaleId)
				} else {
					fmt.Printf("删除的 scale id: %v\n", client.scaleId)
					delete(h.clientOfScales, h.scales[client.scaleId])
				}

				// close(client.send)
			}
		case scale := <-h.addScale: // from scale manager
			if len(h.scales) == 0 {
				h.scales[scale.Id] = scale
			} else {
				if h.scales[scale.Id] == nil {
					h.scales[scale.Id] = scale
				} else {
					l.Log.Warnf("scale id already registered, just ignore it")
				}
			}
		case scale := <-h.removeScale: // from scale manager
			if h.scales[scale.Id] != nil { // scale not existing
				h.scales[scale.Id] = nil
			} else {
				l.Log.Warn("scale id not registered and can't be removed, just ignore it")
			}
		case userMessage := <-h.recvWsClientMsg: // message from websocket client
			var data map[string][]byte
			json.Unmarshal(userMessage, &data)
			scaleId := new(big.Int).SetBytes(data["scaleId"]).Int64()
			if scaleId == 0 { // not for scale communication but for information purposes
				// request := parseMsg(string(data["message"]))
				// TODO: send request to scale manager to get scale list or get serial ports
				// parse request for "get port list", "get scale list", "update scale conneciton",
				//                   "create a new scale", delete a scale" or "close application"
				l.Log.Infof("Got request from common channel %v\n", string(data["message"]))
				parseMsgAndTrigEvt(h.scaleMgr, string(data["message"]))

			} else if scaleId > SERVICE_ID { // not for scale communication but for information purposes
				//小服务的接口
				l.Log.Infof("Got request from common channel %v\n", string(data["message"]))
				parseMsgAndTrigEvtService(h.scaleMgr, string(data["message"]), scaleId)
			} else {
				scale := h.scales[scaleId]
				if scale == nil { // something wrong about scale id
					l.Log.Warnf("cannot find the scale with id: %v\n", scaleId)
				} else if scale.Id != scaleId { // something wrong about scale id
					l.Log.Errorf("scale id: %v is not consistent with the id: %v recorded in the hub", scaleId, scale.Id)
				} else {
					// send data to the scale
					// if req, err := parseToScaleReq(string(data["message"])); err == nil {
					// 	go procToScaleReq(req, scaleId, h, scale) // TODO: handle error
					// }

					l.Log.Warnf("Test : %v\n", scaleId)

				}
			}

		case scaleMessage := <-h.recvScaleMsg:
			// handle the message from the scale
			if h.scales[scaleMessage.ScaleId] != nil {
				client := h.clientOfScales[h.scales[scaleMessage.ScaleId]]
				if client != nil {
					outData, _ := json.Marshal(scaleMessage)
					client.sendCh <- outData
				}
			}
			// default:
			// 	fmt.Println("    .")
			// 	time.Sleep(1 * time.Millisecond)

		case scaleMgrMessage := <-h.recvScaleMgrMsg:
			// handle the message from the scale

			l.Log.Debugf("秤客户端的数量---------- %v\n", (len(h.clientOfScales)))
			if len(h.clientOfScales) == 0 {
				break
			}
			if h.clientOfScales[h.scales[0]] == nil {
				break
			}

			client := h.clientOfScales[h.scales[0]]
			println(h.scales)
			outData, _ := json.Marshal(scaleMgrMessage)
			if client != nil {
				client.sendCh <- outData
				l.Log.Debugf("ClientSendch---------- %v\n", string(outData))
			}
		case recvScaleMgrMsgSrv := <-h.recvScaleMgrMsgSrv: //20241118 如何将数据传出去？9999999999  99999998
			// handle the message from the scale
			// 只要是服务，全送
			outData, _ := json.Marshal(recvScaleMgrMsgSrv)
			if recvScaleMgrMsgSrv.ScaleId > SERVICE_ID {
				client := h.clientOfService[recvScaleMgrMsgSrv.ScaleId]
				if client != nil {
					client.sendCh <- outData
				}

			} else {

				for _, srvRel := range h.srvScaleRel {
					if srvRel.ScaleId == recvScaleMgrMsgSrv.ScaleId && srvRel.IsUsed {
						client := h.clientOfService[srvRel.SrvId]
						if client != nil {
							client.sendCh <- outData
						}

					}
				}

			}

		case scaleMessage := <-h.recvScaleNotifyMsg:
			// handle the message from the scale
			if len(h.clientOfScales) == 0 {
				break
			}

			if h.clientOfScales[h.scales[scaleMessage.ScaleId]] == nil {
				break
			}

			client := h.clientOfScales[h.scales[scaleMessage.ScaleId]]
			if client != nil { // handle the transient situation
				outData, _ := json.Marshal(scaleMessage)
				// fmt.Printf("%v\n", scaleMessage)
				l.Log.Debugf("%v\n", string(outData))
				client.sendCh <- outData
			}
		default:
			time.Sleep(time.Microsecond * 100)
			continue
		}
	}
}

func parseMsgAndTrigEvt(scaleMgr *ScaleMgr, reqJson string) {
	var req Request
	if err := json.UnmarshalFromString(reqJson, &req); err != nil {
		l.Log.Error(err)
		return
	}

	switch req.Req {
	case REQ_GET_SCALE_LIST:
		ScaleListed.Trigger(scalesListed, scaleMgr)
	case REQ_GET_PORT_LIST:
		PortsListed.Trigger(portsListed)
	case REQ_GET_BT_LIST:
		BtListed.Trigger(btListed)
	case REQ_ADD_SCALE:
		jsonStr := req.ReqData
		var data ReqAddScale
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ScaleAdded.Trigger(scaleAdded, data)
		}
	case REQ_DEL_SCALE:
		jsonStr := req.ReqData
		var data ReqDelScale
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			scaleDeleted.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_MODIFY_SCALE:
		jsonStr := req.ReqData
		var data ReqModifyScale
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ScaleModified.Trigger(scaleModified, data)
		}
	case REQ_MODIFY_SCALE_NAME:
		jsonStr := req.ReqData
		var data ReqModifyScaleName
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ScaleNameModified.Trigger(scaleNameModified, data)
		}

	case REQ_GET_PLU_BY_PAGE:
		jsonStr := req.ReqData
		var data ReqGetPluByPage
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			PluByPageListed.Trigger(pluByPageListed, scaleMgr.srvMgr, data)
		}

	case REQ_GET_PRODUCT_LIST:
		productsListed.Trigger(scaleMgr.srvMgr)

	case REQ_DOWN_ALL_PLU:

		exportPluToFile.Trigger(scaleMgr.srvMgr, req.ReqData)

	case REQ_SET_PLU:
		jsonStr := req.ReqData
		var data ReqPluSetting
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			pluSetting.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_GET_PLU_SETTING:
		getPluSetting.Trigger(scaleMgr.srvMgr)

	case REQ_CLEAR_PRODUCT:
		productCleared.Trigger(scaleMgr.srvMgr)

	case REQ_EXPORT_PRODUCT:
		jsonStr := req.ReqData
		var data ReqExportProduct
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			exportProduct.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_ADD_PRODUCT:
		jsonStr := req.ReqData
		var data ReqAddPlu
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ProductAdded.Trigger(productAdded, scaleMgr.srvMgr, data)
		}
	case REQ_CHECK_PLU_EXIST:

		checkPluExist.Trigger(scaleMgr.srvMgr, req.ReqData)

	case REQ_ADD_ONE_PRODUCT:
		jsonStr := req.ReqData
		var data AddProduct
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ProductAddedOne.Trigger(productAddedOne, scaleMgr.srvMgr, data)

		}

	case REQ_DEL_PRODUCT:
		jsonStr := req.ReqData
		var data ReqDelProduct
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ProductDeleted.Trigger(productDeleted, scaleMgr.srvMgr, data)
		}
	case REQ_DEL_ALL_PRODUCT:
		productDeletedAll.Trigger(scaleMgr.srvMgr)
	case REQ_GET_LAST_PRODUCT_REC:

		getLastProductRec.Trigger(scaleMgr.srvMgr)

	case REQ_UPDATE_ENABLED_PLU:
		jsonStr := req.ReqData
		var data ReqUpdateEnabledPlu
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateEnabledPlu.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_MODIFY_PRODUCT:
		jsonStr := req.ReqData
		var data AddProduct
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			ProductModified.Trigger(productModified, scaleMgr.srvMgr, data)
		}
	case REQ_GET_USER_LIST:
		usersListed.Trigger(scaleMgr.srvMgr)
	case REQ_ADD_USER:
		jsonStr := req.ReqData
		var data ReqAddUser
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			UserAdded.Trigger(userAdded, scaleMgr.srvMgr, data)
		}
	case REQ_GET_MODBUS_SERVICES:
		getModbusServices.Trigger(scaleMgr)
	case REQ_ADD_MODBUS_SERVICE:
		var data ReqAddModbusService
		if err := json.UnmarshalFromString(req.ReqData, &data); err != nil {
			l.Log.Error(err)
		} else {
			addModbusService.Trigger(data)
		}
	case REQ_EDIT_MODBUS_SERVICE:
		var data ReqEditModbusService
		if err := json.UnmarshalFromString(req.ReqData, &data); err != nil {
			l.Log.Error(err)
		} else {
			editModbusService.Trigger(data)
		}
	case REQ_DEL_MODBUS_SERVICE:
		var data ReqDelModbusService
		if err := json.UnmarshalFromString(req.ReqData, &data); err != nil {
			l.Log.Error(err)
		} else {
			delModbusService.Trigger(data)
		}
	case REQ_DEL_USER:
		jsonStr := req.ReqData
		var data ReqDelUser
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			UserDeleted.Trigger(userDeleted, scaleMgr.srvMgr, data)
		}
	case REQ_MODIFY_USER:
		jsonStr := req.ReqData
		var data ReqModifyUser
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			UserModified.Trigger(userModified, scaleMgr.srvMgr, data)
		}

	case REQ_QUIT_APPLICATION:
		l.Log.Warn("Got quit application")
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_QUIT_APPLICATION, MsgBody: ""}
		mSrvMgr.quitch <- true
	case REQ_GET_UI_CONF:
		l.Log.Info("Got get UI Config request")
		modeInt, _ := strconv.Atoi(req.ReqData)
		modeUint := uint(modeInt)
		config, _ := mSrvMgr.modeSetting.GetModeSetting(modeUint)
		configStr, _ := json.MarshalToString(config[0])
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_UI_CONFIG, MsgBody: configStr}
	case REQ_UPDATE_UI_CONF:
		l.Log.Info("Got update UI Config request")
		var config ModeSetting
		if err := json.UnmarshalFromString(req.ReqData, &config); err != nil {
			l.Log.Error(err)
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_UI_CONFIG, MsgBody: "failed to parse update UI Config"}
		} else {
			err := mSrvMgr.modeSetting.settingPb.UpdateModeSetting(config)
			if err != nil {
				l.Log.Error(err)
			}
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_UI_CONFIG, MsgBody: "ok"}
		}
	case REQ_GET_LICENSE:
		getLicenseList()
		licListStr, _ := json.MarshalToString(gLicenseInfoList)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_LICENSE, MsgBody: licListStr}

	case REQ_CHECK_LICENSE_KEY:

		isValid, machineId, licValidDate, moduleName := lic.IsKeyValid(req.ReqData)
		var isValidStr string
		if isValid {
			isValidStr = "true"
		} else {
			isValidStr = "false"
		}
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CHECK_LICENSE_KEY, MsgBody: moduleName + "," + isValidStr + "," + machineId + "," + licValidDate}

	case REQ_UPDATE_LICENSE:
		licPath := filepath.Join(comm.GetExePath(), comm.LICENSE_FILE)
		if err := lic.SaveKey(licPath, req.ReqData); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_LICENSE, MsgBody: "fail"}
		}
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_LICENSE, MsgBody: "ok"}

	case REQ_GET_DETAIL_LIST: // TODO: should we check the input parameters?
		DetailListed.Trigger(detailListed, scaleMgr)
	case REQ_GET_SCALE_SRV_LIST: // TODO: should we check the input parameters?
		jsonStr := req.ReqData
		scaleSrvList.Trigger(scaleMgr, jsonStr)
	case REQ_SET_SCALE_SRV_VAL: // TODO: should we check the input parameters?
		jsonStr := req.ReqData
		var data SrvScaleRel
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			setScaleSrvVal.Trigger(scaleMgr, data)
		}

	case REQ_GET_WIFI_PWD_LIST:
		wifiListed.Trigger(scaleMgr.srvMgr)
	case REQ_WIFI_PWD:
		jsonStr := req.ReqData
		var data ReqAddWifi
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			wifiAdded.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_SEND_TO_SRV1:
		jsonStr := req.ReqData
		sendToSrv1.Trigger(scaleMgr.srvMgr, jsonStr)

	case REQ_SET_DO_SERVICE_ACTION:
		jsonStr := req.ReqData
		var data ReqDoServiceAction
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			doServiceAction.Trigger(scaleMgr.srvMgr, data)
		}
		//新增原料类型
	case REQ_ADD_RAW_TYPE:
		jsonStr := req.ReqData
		var data ReqAddRawType
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			RawTypeAdded.Trigger(rawTypeAdded, scaleMgr.srvMgr, data)
		}
		//删除未使用的配方类型
	case REQ_DEL_UNUSED_FMA_TYPE:
		FmaTypeUnusedDeleted.Trigger(fmaTypeUnusedDeleted, scaleMgr.srvMgr)
	case REQ_DEL_UNUSED_RAW_TYPE:
		RawTypeUnusedDeleted.Trigger(rawTypeUnusedDeleted, scaleMgr.srvMgr)

	case REQ_EDIT_RAW_TYPE:
		jsonStr := req.ReqData
		var data ReqEditRawType
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			RawTypeModified.Trigger(rawTypeModified, scaleMgr.srvMgr, data)
		}
	case REQ_DEL_RAW_TYPE:
		jsonStr := req.ReqData
		var data ReqAddRawType
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			RawTypeDeleted.Trigger(rawTypeDeleted, scaleMgr.srvMgr, data)
		}

		//新增配方类型
	case REQ_ADD_FORMULA_TYPE:
		jsonStr := req.ReqData
		var data ReqAddFormulaType
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FormulaTypeAdded.Trigger(formulaTypeAdded, scaleMgr.srvMgr, data)
		}
	case REQ_GET_FORMULA_TYPE_LIST:
		formulaTypeListed.Trigger(scaleMgr.srvMgr)
		//获取原料类型表
	case REQ_GET_RAW_TYPE_LIST:
		rawTypeListed.Trigger(scaleMgr.srvMgr)
		//新增原料数据
	case REQ_ADD_RAW_DATA:
		jsonStr := req.ReqData
		var data ReqAddRawData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			RawDataAdded.Trigger(rawDataAdded, scaleMgr.srvMgr, data)
		}

		//根据配方ID获取原料输出端口
	case REQ_GET_RAW_OUTPUT_BY_FMA_ID:
		jsonStr := req.ReqData
		var data ReqGetRawOutputByFmaId
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getRawOutputByFmaId.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_IMPORT_RAW_LIST:
		jsonStr := req.ReqData
		var data ReqImportRawList
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			RawListImported.Trigger(rawListImported, scaleMgr.srvMgr, data)
		}
	case REQ_IMPORT_FMA_LIST:
		jsonStr := req.ReqData
		var data ReqImportFmaList
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			formulaListImported.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_GET_RAW_DATA_LIST:
		rawDataListed.Trigger(scaleMgr.srvMgr)

	case REQ_GET_OUTPUT_PORT:
		getOutputPort.Trigger(scaleMgr.srvMgr)

	case REQ_UPDATE_OUTPUT_PORT:
		jsonStr := req.ReqData
		var data []ReqUpdateOutputPort
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateOutputPort.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_GET_INPUT_PORT:
		getInputPort.Trigger(scaleMgr.srvMgr)

	case REQ_UPDATE_INPUT_PORT:
		jsonStr := req.ReqData
		var data []ReqUpdateInputPort
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateInputPort.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_GET_AUTO_NEXT:
		getAutoNext.Trigger(scaleMgr.srvMgr)

	case REQ_GET_REPORT_PRINT_SETTING:
		getSetReportPrint.Trigger(scaleMgr.srvMgr)

	case REQ_UPLOAD_SERVER_EDIT:
		jsonStr := req.ReqData
		var data UploadServerInfo
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			editUploadFmaServer.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_UPLOAD_SERVER_GET:
		getUploadFmaServer.Trigger(scaleMgr.srvMgr)

	case REQ_GET_ALL_SEAL_LOG:
		jsonStr := req.ReqData
		var data GetSealLogReq
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getAllSealLog.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_UNSEAL_BY_MASTER_KEY:
		data := req.ReqData
		unsealByMasterKey.Trigger(scaleMgr.srvMgr, data)

	case REQ_UPDATE_REPORT_PRINT_SETTING:
		jsonStr := req.ReqData
		var data SetReportPrint
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateSetReportPrint.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_UPDATE_AUTO_NEXT:
		jsonStr := req.ReqData
		var data ReqUpdateAutoNext
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateAutoNext.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_GET_UNSTABLE_ZERO_TARE:
		getUnstableZeroTare.Trigger(scaleMgr.srvMgr)

	case REQ_UPDATE_UNSTABLE_ZERO_TARE:
		jsonStr := req.ReqData
		var data ReqUpdateUnstableZeroTare
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateUnstableZeroTare.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_EDIT_RAW_DATA:
		jsonStr := req.ReqData
		var data ReqEditRawData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			RawDataEdited.Trigger(rawDataEdited, scaleMgr.srvMgr, data)
		}
	case REQ_DELETE_RAW_DATA:
		jsonStr := req.ReqData
		var data ReqDelRawData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		}
		rawDataDeleted.Trigger(scaleMgr.srvMgr, data)
	case REQ_EDIT_FORMULA_TYPE:
		jsonStr := req.ReqData
		var data ReqEditRawType
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FmaTypeModified.Trigger(fmaTypeModified, scaleMgr.srvMgr, data)
		}
	case REQ_DEL_FORMULA_TYPE:
		jsonStr := req.ReqData
		var data ReqAddRawType
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		}
		fmaTypeDeleted.Trigger(scaleMgr.srvMgr, data)

	//新增配方
	case REQ_ADD_FORMULA_DATA:
		jsonStr := req.ReqData
		var data ReqAddFormulaData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FormulaDataAdded.Trigger(formulaDataAdded, scaleMgr.srvMgr, data)
		}
	//修改配方
	case REQ_EDIT_FORMULA_DATA:
		jsonStr := req.ReqData
		var data ReqAddFormulaData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			formulaDataEdited.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_GET_FORMULA_BY_BARCODE:
		jsonStr := req.ReqData
		var data ReqGetFormulaByBarcode
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getFormulaByBarcode.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_CHECK_FMA_ID_AND_BARCODE:
		jsonStr := req.ReqData
		var data ReqCheckFmaIdAndBarcode
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			checkFmaIdAndBarcode.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_GET_FORMULA_LIST:
		formulaRecList.Trigger(scaleMgr.srvMgr)
	case REQ_GET_FORMULA_DATA:
		jsonStr := req.ReqData
		formulaData.Trigger(scaleMgr.srvMgr, jsonStr)
	case REQ_GET_RAW_DATA:
		jsonStr := req.ReqData
		rawDataGetted.Trigger(scaleMgr.srvMgr, jsonStr)
		//新增配方称重记录
	case REQ_ADD_FORMULA_REC:
		jsonStr := req.ReqData
		var data ReqFormulaWgtRec
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FormulaWgtRecAdded.Trigger(formulaWgtRecAdded, scaleMgr.srvMgr, data)
		}
		//获取配方称重记录
	case REQ_GET_FORMULA_REC_LIST:
		formulaWgtRecList.Trigger(scaleMgr.srvMgr)
		//根据订单号获取配方称重记录

	case REQ_GET_ONE_FORMULA_REC_LIST:
		jsonStr := req.ReqData
		oneFmaWgtRecList.Trigger(scaleMgr.srvMgr, jsonStr)
	case REQ_GET_FMA_REC_BY_ORDER:
		jsonStr := req.ReqData
		getFmaRecByOrderId.Trigger(scaleMgr.srvMgr, jsonStr)

		//删除配方
	case REQ_DELETE_FORMULA_DATA:
		jsonStr := req.ReqData
		var data ReqDelFmaData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FormulaDeleted.Trigger(formulaDeleted, scaleMgr.srvMgr, data)
		}
		//删除所有配方
	case REQ_DELETE_MANY_FORMULA:
		jsonStr := req.ReqData
		var data ReqDelAllFmaData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FormulaDeletedAll.Trigger(formulaDeletedAll, scaleMgr.srvMgr, data)
		}
	//删除暂存配方
	case REQ_DEL_MANY_DRAFT_FMA:
		jsonStr := req.ReqData
		var data ReqDeleteAllDraftFmaWgtRec
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			formulaDataDeletedAll.Trigger(scaleMgr.srvMgr, data)
		}
	//删除所有原料数据
	case REQ_DEL_MANY_RAW_DATA:
		jsonStr := req.ReqData
		var data ReqDelAllRawData
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			rawDataDeletedAll.Trigger(scaleMgr.srvMgr, data)
		}

		//获取暂存配方列表
	case REQ_GET_DRAFT_FMA_WGT_REC_LIST:
		getDraftFmaWgtRecList.Trigger(scaleMgr.srvMgr)
		//删除暂存配方称重记录
	case REQ_DELETE_DRAFT_FMA_WGT_REC:
		jsonStr := req.ReqData
		var data ReqDeleteDraftFmaWgtRec
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			deleteDraftFmaWgtRec.Trigger(scaleMgr.srvMgr, data)
		}
		//更新暂存配方称重记录
	case REQ_UPDATE_DRAFT_FMA_WGT_REC:
		jsonStr := req.ReqData
		var data DrafFmaWgtRecInfo
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateDraftFmaWgtRec.Trigger(scaleMgr.srvMgr, data)
		}
		//创建暂存配方称重记录
	case REQ_CREATE_DRAFT_FMA_WGT_REC:
		jsonStr := req.ReqData
		var data DrafFmaWgtRecInfo
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			addDraftFmaWgtRec.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_ADD_SYS_USER:
		jsonStr := req.ReqData
		var data ReqAddSysUser
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			addSysUser.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_DELETE_SYS_USER:
		jsonStr := req.ReqData
		var data ReqSysUserIdList
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			deleteSysUser.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_UPDATE_SYS_USER:
		jsonStr := req.ReqData
		var data ReqUpdateSysUser
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			updateSysUser.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_DISABLE_SYS_USER:
		jsonStr := req.ReqData
		var data ReqEnabledSysUserId
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			disableSysUser.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_CHANGE_PASSWORD:
		jsonStr := req.ReqData
		var data ReqChangePassword
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			changePassword.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_LOGIN:
		jsonStr := req.ReqData
		var data ReqLogin
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			login.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_LOGOUT:
		logout.Trigger(scaleMgr.srvMgr)
	case REQ_GET_ALL_USERS:
		getAllUsers.Trigger(scaleMgr.srvMgr)

	case REQ_GET_USER_DETAIL:
		jsonStr := req.ReqData
		var data ReqSysUserName
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getUserDetail.Trigger(scaleMgr.srvMgr, data)
		}

		//新增流速
	case REQ_ADD_FLOW_RATE:
		jsonStr := req.ReqData
		var data ReqFlowRateRec
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			FlowRateAdded.Trigger(flowRateAdded, scaleMgr.srvMgr, data)
		}
		//获取流速
	case REQ_GET_FLOW_RATE_LIST:
		flowRateList.Trigger(scaleMgr.srvMgr)
		//获取称重记录列表
	case REQ_GET_ALL_WGT_REC_LIST:
		jsonStr := req.ReqData
		var data ReqGetAllWgtRecList
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getAllWgtRecList.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_GET_SEARCH_REC_LIST:
		jsonStr := req.ReqData
		var data ReqGetSearchRecList
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getSearchRecList.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_ADD_WGT_REC:
		jsonStr := req.ReqData
		var data ReqAddWgtRec
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			addWgtRec.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_DEL_WGT_REC:
		jsonStr := req.ReqData
		var data ReqDelWgtRec
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			delWgtRec.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_DEL_WGT_REC_BY_ID:
		jsonStr := req.ReqData
		var data ReqDelWgtRecById
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			delWgtRecById.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_EXPORT_ALL_RECS:
		jsonStr := req.ReqData
		var data ReqExportAllRecs
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			exportAllRecs.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_KILL_BOOT_COMMANDER:
		var OUR_USED_APP_NAMES []string = []string{"BootCommander.exe"}
		for _, app := range OUR_USED_APP_NAMES {
			if err := util.KillApp(app); err == nil {
				fmt.Println("wait 5 seconds...")
				time.Sleep(2 * time.Second)
				fmt.Println("done")
			}
		}
		// mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_KILL_BOOT_COMMANDER, MsgBody: "ok"}
	case REQ_ADD_SYS_LOG:
		jsonStr := req.ReqData
		var data ReqAddSysLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			addSysLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_ADD_CAL_LOG:
		jsonStr := req.ReqData
		var data CalibrationLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			addCalRecord.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_ADD_SCALE_LOG:
		jsonStr := req.ReqData
		var data ReqAddScaleLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			addScaleLog.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_DEL_SYS_LOG:
		jsonStr := req.ReqData
		var data ReqDelLogs
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			delMultiSysLog.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_DEL_CAL_LOG:
		jsonStr := req.ReqData
		var data ReqDelLogs
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			delMultiCalLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_DEL_SCALE_LOG:
		jsonStr := req.ReqData
		var data ReqDelLogs
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			delMultiScaleLog.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_DEL_ALL_SYS_LOG:
		delAllSysLog.Trigger(scaleMgr.srvMgr)

	case REQ_DEL_ALL_CAL_LOG:
		delAllCalLog.Trigger(scaleMgr.srvMgr)
	case REQ_DEL_ALL_SCALE_LOG:
		delAllScaleLog.Trigger(scaleMgr.srvMgr)

	case REQ_EXPORT_SYS_LOG:
		jsonStr := req.ReqData
		var data ReqExportLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			exportSysLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_EXPORT_CAL_LOG:
		jsonStr := req.ReqData
		var data ReqExportLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			exportCalLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_EXPORT_SCALE_LOG:
		jsonStr := req.ReqData
		var data ReqExportLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			exportScaleLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_GET_SYS_LOG:
		jsonStr := req.ReqData
		var data ReqGetLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getAllSysLog.Trigger(scaleMgr.srvMgr, data)
		}

	case REQ_GET_CAL_LOG:
		jsonStr := req.ReqData
		var data ReqGetLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getAllCalLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_GET_SCALE_LOG:
		jsonStr := req.ReqData
		var data ReqGetLog
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			getAllScaleLog.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_OPEN_OUTPUT_PORT:
		jsonStr := req.ReqData
		var data ReqPortInfo
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			openOutputPort.Trigger(scaleMgr.srvMgr, data)
		}
	case REQ_READ_OUTPUT_PORT:
		jsonStr := req.ReqData
		var data ReqPortInfo
		if err := json.UnmarshalFromString(jsonStr, &data); err != nil {
			l.Log.Error(err)
		} else {
			readOutputPort.Trigger(scaleMgr.srvMgr, data)
		}

	}

}

func getLicenseList() {
	var licKey string
	var newLicList []LicenseInfo
	var licKeyList []string

	licFilePath := filepath.Join(comm.GetExePath(), comm.LICENSE_FILE)

	licKey, _ = lic.ReadLicFile(licFilePath)

	licKeyList = strings.Split(licKey, "\r\n")
	for _, item := range licKeyList {
		if len(item) == 74 || len(item) == 78 || strings.Contains(item, "++==") {
			gIsKeyValid, gMachineId, gLicValidDate, gModuleName = lic.IsKeyValid(item)
			if gIsKeyValid {
				newLicList = append(newLicList, LicenseInfo{Id: gMachineId, ValidDate: gLicValidDate, ModuleName: gModuleName, IsValid: gIsKeyValid})
			}
		}
	}

	if len(newLicList) == 0 {
		var temp []string
		licKeyList = temp
		gIsKeyValid, gMachineId, gLicValidDate, gModuleName = lic.IsKeyValid("d7a0a41239d92ee1724cd1a311ffffff2023-05-2594df26ebd828dbff03ede5f76effffff")
		newLicList = append(newLicList, LicenseInfo{Id: gMachineId, ValidDate: gLicValidDate, ModuleName: gModuleName, IsValid: gIsKeyValid})

	}
	gLicenseInfoList = newLicList
}

func parseMsgAndTrigEvtService(scaleMgr *ScaleMgr, reqJson string, scaleId int64) {
	var req Request
	if err := json.UnmarshalFromString(reqJson, &req); err != nil {
		l.Log.Error(err)
		return
	}

	switch req.Req {
	case REQ_GET_SCALE_LIST:
		scalesListedSrv.Trigger(scaleMgr, scaleId)
	case REQ_SEND_TO_UI:
		jsonStr := req.ReqData
		println(jsonStr)
		println("--------------------")
		sendToUi.Trigger(scaleMgr.srvMgr, jsonStr)

	}
}

//分页获取PLU

type SendPluList struct {
	Total   int
	PluList []ProductRec
}

func (p pluByPageListedNotifier) Handle(mgr *SrvMgr, payload ReqGetPluByPage) {

	products, total, _ := mgr.productPd.GetPluByPage(payload.Page, payload.PageSize, payload.FieldName, payload.Direction, payload.Search)
	sendPlu := SendPluList{Total: total, PluList: products}
	sendPluStr, err := json.MarshalToString(sendPlu)
	if err != nil {
		l.Log.Error(err)
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PLU_LIST, MsgBody: sendPluStr}
}

// 获取PLU设置字段
func (p getPluSettingNotifier) Handle(mgr *SrvMgr) {
	pluSetting, err := mgr.productPd.GetPluSetting()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PLU_SETTING, MsgBody: ""}
		return
	}

	plus := []string{}
	for _, value := range pluSetting {
		if value == "plu" || value == "productName" {
			continue
		}
		plus = append(plus, value)
	}

	Req := ReqPluSetting{Plu: plus}
	jsonStr, err := json.MarshalToString(Req)
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PLU_SETTING, MsgBody: ""}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PLU_SETTING, MsgBody: jsonStr}
}

// 导出产品列表到excel文件
func (p exportPluToFileNotifier) Handle(mgr *SrvMgr, payload string) {
	var products []ProductRec
	err := error(nil)
	products, err = mgr.productPd.GetRecsList()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DOWN_ALL_PLU, MsgBody: "fail to get product list"}
	}

	var productsEnabled []ProductRec
	for _, product := range products {
		if product.Enabled {
			productsEnabled = append(productsEnabled, product)
		}
	}

	err = SavePluToFile(payload, productsEnabled)
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DOWN_ALL_PLU, MsgBody: "fail to save plu to file"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DOWN_ALL_PLU, MsgBody: "ok," + payload}

}

// 导出产品列表
func (p exportProductNotifier) Handle(mgr *SrvMgr, payload ReqExportProduct) {
	fmt.Println(payload)
	fmt.Println(payload)
	translate := payload.Translation
	products := []ProductRec{}
	err := error(nil)
	if products, err = mgr.productPd.GetExportProductList(payload.SearchPlu); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_PLU_LIST, MsgBody: "fail to get product list"}
		return
	}
	//获取plu设置字段
	pluSetting, err := mgr.productPd.GetPluSetting()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_PLU_LIST, MsgBody: "fail to get plu setting"}
		return
	}
	//写入csv文件表头
	// 打开 CSV 文件
	file, err := os.Create(payload.Path)
	if err != nil {
		mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: "fail to create csv file"}
		return
	}
	defer file.Close()

	// 创建 CSV 写入器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	selectPluFields := []string{}
	selectPluFields = append(selectPluFields, "plu")
	selectPluFields = append(selectPluFields, "productName")
	if len(pluSetting) > 0 {

		for _, value := range pluSetting {
			selectPluFields = append(selectPluFields, value)
		}
	}

	//将recs写入csv文件，文件路径为payload.Path
	//下面是总的表头
	headers := []string{}
	for _, value := range selectPluFields {
		headers = append(headers, getPluTranslate(translate, value))
	}

	//明细的头和表头共用即可，不需要重新写

	// 写入总的标题
	if err := writer.Write(headers); err != nil {
		mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: "fail to write csv file"}
		return
	}

	for _, product := range products {
		row := []string{}
		for _, value := range selectPluFields {
			row = append(row, getFieldValue(product, value))
		}
		if err := writer.Write(row); err != nil {
			mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: "fail to write csv file"}
			return
		}
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_PLU_LIST, MsgBody: payload.Path}
	type Total struct {
		Total int
	}
	jsonStr, _ := json.Marshal(Total{Total: len(products)})
	LogSysOperation(MenuPLUManage, OpExportStr, OpExportStr, string(jsonStr), "ok", "")
}

// 根据字段名获取产品字段值
func getFieldValue(product ProductRec, fieldName string) string {
	switch fieldName {
	case "plu":
		return product.Plu
	case "productName":
		return product.ProductName
	case "category":
		return product.Category
	case "generalUnit":
		unitMap := map[string]string{"0": "kg", "1": "100g", "2": "pcs", "3": "lb", "4": "g", "5": "oz", "6": "lboz", "7": "tj", "8": "hj", "9": "t"}
		if unit, ok := unitMap[product.GeneralUnit]; ok {
			return unit
		}
		return product.GeneralUnit
	case "taxType":
		taxMap := map[string]string{"0": "tax1", "1": "tax2", "2": "tax3"}
		if tax, ok := taxMap[product.TaxType]; ok {
			return tax
		}
		return product.TaxType
	case "price":
		return product.Price
	case "unitWeight":
		return product.UnitWeight
	case "pretare":
		return product.Pretare
	case "limitHigh":
		return product.LimitHigh
	case "limitLow":
		return product.LimitLow
	case "productCode":
		return product.ProductCode
	case "itemCode":
		return product.ItemCode
	default:
		return ""
	}
}

func getPluTranslate(pluMap map[string]string, title string) string {

	for k, v := range pluMap {
		if title == k {
			return v
		}
	}
	return title

}

// 设置PLU字段
func (p pluSettingNotifier) Handle(mgr *SrvMgr, payload ReqPluSetting) {
	l.Log.Debug("Handle pluSettingNotifier called")
	_, username, _ := GetCurrentUser()

	err := mgr.productPd.SetPluSetting(payload.Plu, username)
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PLU_SETTING, MsgBody: "fail to get plu setting"}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PLU_SETTING, MsgBody: "ok"}

	jsonStr, _ := json.Marshal(payload.Plu)
	LogSysOperation(MenuPLUManage, OpSetStr, OpSetStr, string(jsonStr), "ok", "")
}

func (p productClearedNotifier) Handle(mgr *SrvMgr) {
	l.Log.Debug("Handle productClearedNotifier called")
	if err := mgr.productPd.DeleteAllRec(); err != nil {
		l.Log.Error(err)
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_DEL, MsgBody: "ok"}
	LogSysOperation(MenuPLUManage, OpClearStr, OpClearStr, "", "ok", "")

}

// 获取产品列表
func (p productListedNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle productListedNotifier called")
	// Do something with this event

	products, _ := mgr.productPd.GetRecsList()

	batchSize := 100 //每次发送1000条
	numBatches := (len(products) + batchSize - 1) / batchSize

	for i := 0; i < numBatches; i++ {
		startIndex := i * batchSize
		endIndex := (i + 1) * batchSize
		if endIndex > len(products) {
			endIndex = len(products)
		}
		batchProducts := products[startIndex:endIndex]
		var productsStr string
		var err error
		if productsStr, err = json.MarshalToString(batchProducts); err != nil {
			l.Log.Error(err)
			continue
		}
		// 发送每一批次的数据
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCTS_LIST, MsgBody: productsStr}
		time.Sleep(100 * time.Millisecond)
	}
}

// checkPluExist
func (p checkPluExistNotifier) Handle(mgr *SrvMgr, payload string) {
	// Do something for this event
	l.Log.Debug("Handle checkPluExistNotifier called")

	pluIdNum := strings.Split(payload, ",")
	id := pluIdNum[0]
	pluId, _ := strconv.Atoi(id)
	plu := pluIdNum[1]
	exist, _ := mgr.productPd.CheckPluExist(pluId, plu)
	if exist {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CHECK_PLU_EXIST, MsgBody: "true"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CHECK_PLU_EXIST, MsgBody: "false"}
}

func (p addProductNotifier) Handle(mgr *SrvMgr, payload ReqAddPlu) {
	// Do something for this event
	l.Log.Debug("Handle addProductNotifier called")

	productList := payload.PluList
	recList := make([]ProductRec, 0)
	for _, product := range productList {
		rec := ProductRec{
			Plu:         product.Plu,
			ProductCode: product.ProductCode,
			ItemCode:    product.ItemCode,
			Category:    product.Category,
			ProductName: product.ProductName,
			GeneralUnit: product.GeneralUnit,
			TaxType:     product.TaxType,
			Price:       product.Price,
			UnitWeight:  product.UnitWeight,
			Pretare:     product.Pretare,
			LimitHigh:   product.LimitHigh,
			LimitLow:    product.LimitLow,
			CreateBy:    product.CreateBy,
			UpdateBy:    product.CreateBy,
			CreateUser:  product.CreateUser,
			UpdateUser:  product.UpdateUser,
		}
		recList = append(recList, rec)

	}
	if err := mgr.productPd.Insert100Rec(recList); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_ADD, MsgBody: err.Error()}
		return
	}

	if payload.Index*100 < payload.Total {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_ADD, MsgBody: "ok"}
		return
	}
	LogSysOperation(MenuPLUManage, OpImportStr, OpImportStr, "Total: "+strconv.Itoa(payload.Total), "ok", "")
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_ADD, MsgBody: "ok"}

}

func (p addOneProductNotifier) Handle(mgr *SrvMgr, payload AddProduct) {
	// Do something for this event
	l.Log.Debug("Handle addOneProductNotifier called")

	product := ProductRec{
		Plu:         payload.Plu,
		ProductCode: payload.ProductCode,
		ItemCode:    payload.ItemCode,
		Category:    payload.Category,
		ProductName: payload.ProductName,
		GeneralUnit: payload.GeneralUnit,
		TaxType:     payload.TaxType,
		Price:       payload.Price,
		UnitWeight:  payload.UnitWeight,
		Pretare:     payload.Pretare,
		LimitHigh:   payload.LimitHigh,
		LimitLow:    payload.LimitLow,
		CreateBy:    payload.CreateBy,
		UpdateBy:    payload.UpdateBy,
		CreateUser:  payload.CreateUser,
		UpdateUser:  payload.UpdateUser,
	}

	if err := mgr.productPd.InsertRec(product); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_ADD_ONE, MsgBody: "failed to add product"}
		return
	}
	// send result back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_ADD_ONE, MsgBody: "ok"}
}

func (p delProductNotifier) Handle(mgr *SrvMgr, payload ReqDelProduct) {
	// Do something for this event
	l.Log.Debug("Handle delProductNotifier called")
	if err := mgr.productPd.DeleteRec((payload.RecId)); err != nil {
		// TODO: error handling
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_DEL, MsgBody: "failed to delete product"}
		return
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_DEL, MsgBody: "ok"}
}

func (p delAllProductNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event

	l.Log.Debug("Handle delProductNotifier called")
	if err := mgr.productPd.DeleteAllRec(); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_DEL, MsgBody: "failed to delete all products"}
		return
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_DEL, MsgBody: "ok"}
}

func (p updateEnabledPluNotifier) Handle(mgr *SrvMgr, payload ReqUpdateEnabledPlu) {
	// Do something for this event
	l.Log.Debug("Handle updateEnabledPluNotifier called")

	pluList := payload.PluList
	enabled := payload.Enabled
	updateBy := payload.UpdateBy

	if err := mgr.productPd.UpdateProductRecEnabled(pluList, enabled, updateBy); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_ENABLED_PLU, MsgBody: "failed to update enabled plu"}
		return
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_ENABLED_PLU, MsgBody: "ok"}
}

func (p modifyProductNotifier) Handle(mgr *SrvMgr, payload AddProduct) {

	product := ProductRec{
		RecId:       payload.RecId,
		Plu:         payload.Plu,
		ProductCode: payload.ProductCode,
		ItemCode:    payload.ItemCode,
		Category:    payload.Category,
		ProductName: payload.ProductName,
		GeneralUnit: payload.GeneralUnit,
		TaxType:     payload.TaxType,
		Price:       payload.Price,
		UnitWeight:  payload.UnitWeight,
		Pretare:     payload.Pretare,
		LimitHigh:   payload.LimitHigh,
		LimitLow:    payload.LimitLow,
		UpdateBy:    payload.UpdateBy,
		UpdateUser:  payload.UpdateUser,
	}

	if err := mgr.productPd.ModifyRec(product); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_MODIFY, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PRODUCT_MODIFY, MsgBody: "ok"}
}

func (p getLastProductRecNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getLastProductRecNotifier called")
	lastRec, err := mgr.productPd.GetLastProductRec()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_LAST_PRODUCT_REC, MsgBody: "failed to get last product record"}
		return
	}
	lastRecStr, _ := json.MarshalToString(lastRec)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_LAST_PRODUCT_REC, MsgBody: lastRecStr}
}

func (p userListedNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle userListedNotifier called")
	// Do something with this event
	users, _ := mgr.userPd.GetRecsList()

	var userStr string
	var err error
	if userStr, err = json.MarshalToString(users); err != nil {
		l.Log.Error(err)
		// TODO: error handling
	}
	// send users list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USERS_LIST, MsgBody: userStr}
}

func (p addUserNotifier) Handle(mgr *SrvMgr, payload ReqAddUser) {
	// Do something for this event
	l.Log.Debug("Handle addUserNotifier called")
	var rec UserRec = UserRec{Id: payload.Id, Name: payload.Name, Phone: payload.Phone, IsFemale: payload.IsFemale, Remarks: payload.Remarks}
	if err := NewUserRecProvider().InsertRec(rec); err != nil {
		// TODO: error handling
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USER_ADD, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USER_ADD, MsgBody: ""}
}

func (p delUserNotifier) Handle(mgr *SrvMgr, payload ReqDelUser) {
	// Do something for this event
	l.Log.Debug("Handle delUserNotifier called")
	if err := NewUserRecProvider().DeleteRec(uint(payload.RecId)); err != nil {
		// TODO: error handling
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USER_DEL, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USER_DEL, MsgBody: ""}
}

func (p modifyUserNotifier) Handle(mgr *SrvMgr, payload ReqModifyUser) {
	// Do something for this event
	l.Log.Debug("Handle modifyUserNotifier called")
	var rec UserRec = UserRec{RecId: uint(payload.RecId), Id: payload.Id, Name: payload.Name, Phone: payload.Phone, IsFemale: payload.IsFemale, Remarks: payload.Remarks}
	if err := mgr.userPd.ModifyRec(rec); err != nil {
		// TODO: error handling
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USER_MODIFY, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_USER_MODIFY, MsgBody: ""}
}

func (p wifiPwdListedNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle wifiPwdListedNotifier called")
	// Do something with this event
	wifis, _ := mgr.wifiPd.GetRecsList()

	var wifiStr string
	var err error
	if wifiStr, err = json.MarshalToString(wifis); err != nil {
		l.Log.Error(err)
		// TODO: error handling
	}
	// send wifi pwd list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_WIFI_PWD_LIST, MsgBody: wifiStr}
}

func (p addWifiPwdNotifier) Handle(mgr *SrvMgr, payload ReqAddWifi) {
	// Do something for this event
	l.Log.Debug("Handle addWifiNotifier called")
	var rec WifiRec = WifiRec{Ssid: payload.Ssid, Pwd: payload.Pwd}
	if err := NewWifiRecProvider().InsertRec(rec); err != nil {
		// TODO: error handling
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_WIFI_PWD_ADD, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_WIFI_PWD_ADD, MsgBody: "ok"}
}

func (p doServiceActionNotifier) Handle(mgr *SrvMgr, payload ReqDoServiceAction) {
	// Do something for this event
	l.Log.Debug("Handle addWifiNotifier called")
	msgStr := ""
	// if mgr.clientOfService[payload.ServiceId] != nil {
	// 	msgStr = "Service Started"
	// 	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: msgStr}
	// 	return
	// }

	srvName, srvPath := getServiceNameAndPath(payload.ServiceId)
	if srvPath == "" {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: msgStr}
		return
	}

	serviceName := srvName
	servicePath := srvPath
	serviceManager := ServiceManager{
		ServiceName: serviceName,
		ServicePath: servicePath,
	}

	switch payload.Action {
	case "Install":
		if err := serviceManager.Install(); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: err.Error()}
			return
		}
		msgStr = SRV_STATUS_INSTALLED

	case "Start":
		if err := serviceManager.Start(); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: err.Error()}
			return
		}
		msgStr = SRV_STATUS_STARTED
	case "Stop":
		if err := serviceManager.Stop(); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: err.Error()}
			return
		}
		msgStr = SRV_STATUS_INSTALLED
	case "Uninstall":
		if err := serviceManager.Uninstall(); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: err.Error()}
			return
		}
		msgStr = SRV_STATUS_UNINSTALLED

	case "Status":
		status := getServiceStatus(serviceName)
		msgStr = "Status:" + status
	default:
		break

	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DO_SERVICE_ACTION, MsgBody: msgStr}
}

func getServiceStatus(serviceName string) string {
	res := IsServiceInstalled(serviceName)
	if !res {
		//服务没有安装
		return SRV_STATUS_UNINSTALLED
	} else if !IsServiceRunning(serviceName) {
		return SRV_STATUS_INSTALLED
	} else {
		return SRV_STATUS_STARTED
	}
}

func getServiceNameAndPath(srvId int64) (string, string) {
	switch srvId {
	case 999999999:
		return "RetailDetailService", filepath.Join(comm.GetServicePath(), "detailservice.exe")
	default:
		return "", ""
	}
}

// ServiceManager结构体用于管理服务的操作
type ServiceManager struct {
	ServiceName string
	ServicePath string
}

// Install方法用于安装服务
func (sm *ServiceManager) Install() error {
	installCmd := fmt.Sprintf("sc create %s binPath= \"%s\"  start= auto", sm.ServiceName, sm.ServicePath)
	// installCmd := "sc create RetailDetailService binpath=\"G:\\T-max\\wifi_tmax\\service\\TmaxService\\Backend\\srvdata\\service\\detailservice.exe\""

	cmd := exec.Command("cmd", "/C", installCmd)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("fail: %v", err)
	}
	fmt.Println("ok")
	return nil
}

// Uninstall方法用于卸载服务
func (sm *ServiceManager) Uninstall() error {
	uninstallCmd := fmt.Sprintf("sc delete %s", sm.ServiceName)
	cmd := exec.Command("cmd", "/C", uninstallCmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("fail: %v", err)
	}
	fmt.Println("ok")
	return nil
}

// Start方法用于启动服务
func (sm *ServiceManager) Start() error {
	startCmd := fmt.Sprintf("sc start %s", sm.ServiceName)
	cmd := exec.Command("cmd", "/C", startCmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("fail: %v", err)
	}
	fmt.Println("ok")
	return nil
}

// Stop方法用于停止服务
func (sm *ServiceManager) Stop() error {
	stopCmd := fmt.Sprintf("sc stop %s", sm.ServiceName)
	cmd := exec.Command("cmd", "/C", stopCmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("fail: %v", err)
	}
	fmt.Println("ok")
	return nil
}

// func isServiceRunning(serviceName string) (bool, error) {
// 	m, err := mgr.Connect()
// 	if err != nil {
// 		return false, err
// 	}
// 	defer m.Disconnect()

// 	s, err := m.OpenService(serviceName)
// 	if err != nil {
// 		return false, err
// 	}
// 	defer s.Close()

// 	status, err := s.Query()
// 	if err != nil {
// 		return false, err
// 	}

// 	return status.State == svc.Running, nil
// }

func IsServiceInstalled(serviceName string) bool {
	m, err := mgr.Connect()
	if err != nil {
		log.Printf("%v", err)
	}
	defer m.Disconnect()

	services, err := m.ListServices()
	if err != nil {
		log.Printf("%v", err)
	}

	for _, s := range services {
		if s == serviceName {
			return true
		}
	}
	return false
}

func IsServiceRunning(serviceName string) bool {
	m, err := mgr.Connect()
	if err != nil {
		log.Printf("mgr.Connect error: %v", err)
		return false
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		log.Printf("OpenService error: %v", err)
		return false
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		log.Printf("Query error: %v", err)
		return false
	}

	return status.State == svc.Running
}

// 配方秤原料类型新增
func (p addRawTypeNotifier) Handle(mgr *SrvMgr, payload ReqAddRawType) {
	// Do something for this event
	l.Log.Debug("Handle addRawTypeNotifier called")
	var rec RawMaterialCategory = RawMaterialCategory{CategoryName: payload.Name}
	if err := mgr.formulaPd.InsertRawType(rec); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_ADD, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_ADD, MsgBody: ""}
	jsonStr, _ := json.MarshalToString(TypeName{Type: payload.Name})
	LogSysOperation(MenuFormulaManage, SubFmaRawTypeAdd, OpAddStr, jsonStr, "ok", "")
}

// 配方类型新增
func (p addFormulaTypeNotifier) Handle(mgr *SrvMgr, payload ReqAddFormulaType) {
	// Do something for this event
	l.Log.Debug("Handle addFormulaTypeNotifier called")
	var rec FormulaCategory = FormulaCategory{CategoryName: payload.Name}
	if err := mgr.formulaPd.InsertFormulaType(rec); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_TYPE_ADD, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_TYPE_ADD, MsgBody: ""}

	jsonStr, _ := json.MarshalToString(TypeName{Type: payload.Name})
	LogSysOperation(MenuFormulaManage, SubFormulaTypeAdd, OpAddStr, jsonStr, "ok", "")
}

// 配方秤原料类型清除未使用的
func (p delRawTypeUnusedNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle delRawTypeUnusedNotifier called")
	if err := mgr.formulaPd.DeleteUnusedRawType(); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_DEL_UNUSED, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_DEL_UNUSED, MsgBody: "ok"}
	LogSysOperation(MenuFormulaManage, SubFmaRawTypeClearUnused, OpClearStr, "", "ok", "")
}

// 配方类型清除未使用的
func (p delFmaTypeUnusedNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle delFmaTypeUnusedNotifier called")
	if err := mgr.formulaPd.DeleteUnusedFormulaType(); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_TYPE_DEL_UNUSED, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_TYPE_DEL_UNUSED, MsgBody: "ok"}
	LogSysOperation(MenuFormulaManage, SubFormulaTypeClearUnused, OpClearStr, "", "ok", "")
}

// 配方秤原料类型列表
func (p getRawTypeListNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getRawTypeListNotifier called")
	types, _ := mgr.formulaPd.GetRawTypeList()
	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(types); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_LIST, MsgBody: ""}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_LIST, MsgBody: typesStr}
}

// 配方类型列表
func (p getFormulaTypeListNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getFormulaTypeListNotifier called")
	types, _ := mgr.formulaPd.GetFormulaTypeList()
	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(types); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_TYPE_LIST, MsgBody: ""}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_TYPE_LIST, MsgBody: typesStr}
}

// 原料数据新增
func (p rawDataAddedNotifier) Handle(mgr *SrvMgr, payload ReqAddRawData) {
	// Do something for this event
	l.Log.Debug("Handle addRawTypeNotifier called")
	var checkCode string
	checkCode = payload.CheckCode
	if checkCode == "" {
		checkCode = payload.MaterialID
	}
	var rec RawMaterial = RawMaterial{
		MaterialID:   payload.MaterialID,
		MaterialName: payload.MaterialName,
		CategoryID:   payload.CategoryID,
		Ingredient:   payload.Ingredient,
		Remark:       payload.Remark,
		Remark1:      payload.Remark1,
		CreatedBy:    payload.CreatedBy,
		UpdatedBy:    payload.UpdatedBy,
		ScaleId:      payload.ScaleId,
		CheckCode:    checkCode,
		Output:       payload.Output,
	}
	if err := mgr.formulaPd.InsertRawInfo(rec); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_ADD, MsgBody: "failed to get max raw id"}
		return
	}

	//获取最大的原料ID
	maxId, err := mgr.formulaPd.GetMaxRawRecId()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_ADD, MsgBody: "failed to get max raw id"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_ADD, MsgBody: "ok," + strconv.Itoa(maxId)}
	SaveFmaRawDataLog(payload)
}

// 获取原料数据的输出口
func (p getRawOutputByFmaIdNotifier) Handle(mgr *SrvMgr, payload ReqGetRawOutputByFmaId) {
	// Do something for this event
	l.Log.Debug("Handle getRawOutputByFmaIdNotifier called")
	output, err := mgr.formulaPd.GetRawOutputByFmaId(payload.FormulaId)

	outputStr := ""

	if outputStr, err = json.MarshalToString(output); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_OUTPUT_BY_FMA_ID, MsgBody: "failed to get raw output by fma id"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_OUTPUT_BY_FMA_ID, MsgBody: outputStr}

}

// //导入原料数据列表
func (p rawListImportedNotifier) Handle(mgr *SrvMgr, payload ReqImportRawList) {
	// Do something for this event
	l.Log.Debug("Handle rawListImportedNotifier called")

	//先导入原料类型
	var categoryNames []string
	for _, v := range payload.RawInfo {
		if v.CategoryName != "" {
			categoryNames = append(categoryNames, v.CategoryName)
		}
	}
	//去掉重复值
	categoryNames = removeDuplicates(categoryNames)
	if len(categoryNames) != 0 {
		if err := mgr.formulaPd.InsertRawTypeList(categoryNames); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST_IMPORT, MsgBody: "failed to import raw type list"}
		}
	}

	//查出所有类型列表
	rawTypes, err := mgr.formulaPd.GetRawTypeList()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST_IMPORT, MsgBody: "failed to get raw type list"}
		return
	}

	rawList := []RawMaterial{}
	//再导入原料数据
	for _, v := range payload.RawInfo {

		//根据类型名称查出类型ID
		categoryId := 0
		if v.CategoryName != "" {
			for _, t := range rawTypes {
				if t.CategoryName == v.CategoryName {
					categoryId = t.CategoryID
					break
				}
			}
			if categoryId == 0 {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST_IMPORT, MsgBody: "failed to get raw type id"}
				return
			}

		}

		var checkCode string
		checkCode = v.CheckCode
		if v.CheckCode == "" {
			checkCode = v.MaterialID
		}

		rec := RawMaterial{
			MaterialID:   v.MaterialID,
			MaterialName: v.MaterialName,
			CategoryID:   categoryId,
			Ingredient:   v.Ingredient,
			Remark:       "",
			Remark1:      "",
			CreatedBy:    payload.CreatedBy,
			UpdatedBy:    payload.CreatedBy,
			ScaleId:      v.ScaleId,
			CheckCode:    checkCode,
		}

		rawList = append(rawList, rec)

	}
	if err := mgr.formulaPd.InsertRawInfoList(rawList); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST_IMPORT, MsgBody: "failed to import raw info list"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST_IMPORT, MsgBody: "ok"}

}

// 去掉重复值
func removeDuplicates(strings []string) []string {
	encountered := map[string]bool{}
	result := []string{}
	for _, v := range strings {
		if !encountered[v] {
			encountered[v] = true
			result = append(result, v)
		}
	}
	return result
}

type FmaDataImportInfo struct {
	Head   FormulaHeader
	Detail []FormulaDetail
}

// //导入配方数据列表
func (p formulaListImportedNotifier) Handle(mgr *SrvMgr, payload ReqImportFmaList) {
	// Do something for this event
	l.Log.Debug("Handle formulaListImportedNotifier called")

	//先导入配方类型
	var categoryNames []string
	for _, v := range payload.FmaInfo {
		if v.Category != "" {
			categoryNames = append(categoryNames, v.Category)
		}
	}
	//去掉重复值
	categoryNames = removeDuplicates(categoryNames)
	if len(categoryNames) != 0 {
		if err := mgr.formulaPd.InsertFormulaTypeList(categoryNames); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_LIST_IMPORT, MsgBody: "failed to import formula type list"}
		}
	}

	//查出所有类型列表
	formulaTypes, err := mgr.formulaPd.GetFormulaTypeList()
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_LIST_IMPORT, MsgBody: "failed to get formula type list"}
		return
	}

	//再导入配方数据
	headerKey, _ := mgr.formulaPd.GetMaxFormulaRecKey() //获取最新的配方ID
	fmaDataList := []FmaDataImportInfo{}
	for _, v := range payload.FmaInfo {
		//根据类型名称查出类型ID
		categoryId := 0
		if v.Category != "" {
			for _, t := range formulaTypes {
				if t.CategoryName == v.Category {
					categoryId = t.CategoryID
					break
				}
			}
			if categoryId == 0 {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_LIST_IMPORT, MsgBody: "failed to get formula type id"}
				return
			}
		}

		FmaDataImportInfo := FmaDataImportInfo{}
		mode := "wgt"
		if v.Mode == "Percent" || v.Mode == "percent" {
			mode = "pct"
		}
		unit := "kg"
		if v.WeightUnit != "" {
			unit = v.WeightUnit
		}

		details := []FormulaDetail{}
		totalWgt := 0.0

		for _, detail := range v.Ingredients {

			roundedValue := math.Round(detail.WeightOrPercent*1000) / 1000

			tempRec := FormulaDetail{
				FormulaRecID:       headerKey + 1,
				MaterialID:         detail.IngredientId,
				MaterialWeight:     roundedValue,
				MaterialPercentage: roundedValue,
				Sequence:           detail.IngredientNo,
				AllowableError:     detail.AllowError,
				Remark:             "",
			}
			totalWgt += roundedValue

			details = append(details, tempRec)
		}

		totalWgt = math.Round(totalWgt*1000) / 1000

		barcode := v.FormulaId
		if v.Barcode != "" {
			barcode = v.Barcode
		}

		var header FormulaHeader = FormulaHeader{
			FormulaID:      v.FormulaId,
			FormulaName:    v.FormulaName,
			FormulaKey:     headerKey + 1,
			CategoryID:     categoryId,
			Remark:         v.Notes,
			CreatedBy:      payload.CreateBy,
			UpdatedBy:      payload.CreateBy,
			FormulaMode:    mode,
			FormulaUnit:    unit,
			TotalWeight:    totalWgt,
			MaterialCount:  len(details),
			IsEncrypted:    v.IsConfidential,
			NeedContainer:  v.NeedContainer,
			FormulaBarcode: barcode,
		}
		FmaDataImportInfo.Head = header
		FmaDataImportInfo.Detail = details
		headerKey = headerKey + 1
		fmaDataList = append(fmaDataList, FmaDataImportInfo)

	}
	if err := mgr.formulaPd.InsertFmaInfoList(fmaDataList); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_LIST_IMPORT, MsgBody: "failed to import formula info list"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_LIST_IMPORT, MsgBody: "ok"}

}

// 修改原料类型
func (p rawTypeEditedNotifier) Handle(mgr *SrvMgr, payload ReqEditRawType) {
	// Do something for this event
	l.Log.Debug("Handle rawTypeEditedNotifier called")
	var rec RawMaterialCategory = RawMaterialCategory{
		CategoryID:   payload.Id,
		CategoryName: payload.Name,
	}
	rawType, _ := mgr.formulaPd.GetRawTypeByID(payload.Id)
	if err := mgr.formulaPd.UpdateRawType(rec); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_EDIT, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_EDIT, MsgBody: "ok"}
	jsonStr, _ := json.MarshalToString(UpdateTypeInfo{NewType: payload.Name, OldType: rawType.CategoryName})
	LogSysOperation(MenuFormulaManage, SubFmaRawTypeUpdate, OpUpdateStr, jsonStr, "ok", "")
}

// 删除原料类型
func (p rawTypeDeletedNotifier) Handle(mgr *SrvMgr, payload ReqAddRawType) {
	// Do something for this event
	l.Log.Debug("Handle rawTypeDeletedNotifier called")
	if err := mgr.formulaPd.DeleteRawType(payload.Name); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_DELETE, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_TYPE_DELETE, MsgBody: "ok"}
	jsonStr, _ := json.MarshalToString(TypeName{Type: payload.Name})
	LogSysOperation(MenuFormulaManage, SubFmaRawTypeDel, OpDeleteStr, jsonStr, "ok", "")
}

// 修改配方类型
func (p fmaTypeEditedNotifier) Handle(mgr *SrvMgr, payload ReqEditRawType) {
	// Do something for this event
	l.Log.Debug("Handle fmaTypeEditedNotifier called")
	var rec FormulaCategory = FormulaCategory{
		CategoryID:   payload.Id,
		CategoryName: payload.Name,
	}

	formulaType, _ := mgr.formulaPd.GetFormulaCategoryByID(payload.Id)
	if err := mgr.formulaPd.UpdateFmaType(rec); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_TYPE_EDIT, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_TYPE_EDIT, MsgBody: "ok"}
	jsonStr, _ := json.MarshalToString(UpdateTypeInfo{NewType: payload.Name, OldType: formulaType.CategoryName})
	LogSysOperation(MenuFormulaManage, SubFormulaTypeUpdate, OpUpdateStr, jsonStr, "ok", "")
}

// 删除配方类型
func (p fmaTypeDeletedNotifier) Handle(mgr *SrvMgr, payload ReqAddRawType) {
	// Do something for this event
	l.Log.Debug("Handle fmaTypeDeletedNotifier called")
	if err := mgr.formulaPd.DeleteFmaType(payload.Name); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_TYPE_DELETE, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FMA_TYPE_DELETE, MsgBody: "ok"}
	jsonStr, _ := json.MarshalToString(TypeName{Type: payload.Name})
	LogSysOperation(MenuFormulaManage, SubFormulaTypeDel, OpDeleteStr, jsonStr, "ok", "")
}

// 获取原料列表
func (p rawDataListedNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle rawDataListedNotifier called")
	recs, _ := mgr.formulaPd.GetRawDataList()

	batchSize := 100 //每次发送1000条
	numBatches := (len(recs) + batchSize - 1) / batchSize

	for i := 0; i < numBatches; i++ {
		startIndex := i * batchSize
		endIndex := (i + 1) * batchSize
		if endIndex > len(recs) {
			endIndex = len(recs)
		}
		batchRecs := recs[startIndex:endIndex]

		recStr := ""
		var err error
		if recStr, err = json.MarshalToString(batchRecs); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST, MsgBody: ""}
			return
		}
		// 发送每一批次的数据
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST, MsgBody: recStr}
		time.Sleep(100 * time.Millisecond)
	}
	if len(recs) == 0 {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_LIST, MsgBody: ""}
	}

}

// 修改原料信息
func (p rawDataEditedNotifier) Handle(mgr *SrvMgr, payload ReqEditRawData) {
	// Do something for this event
	l.Log.Debug("Handle rawDataEditedNotifier called")

	oldRawData, _ := mgr.formulaPd.GetRawData(payload.RecId)
	var checkCode string
	checkCode = payload.CheckCode

	if checkCode == "" {
		checkCode = payload.MaterialID
	}

	var rec RawMaterial = RawMaterial{
		RecId:        payload.RecId,
		MaterialID:   payload.MaterialID,
		MaterialName: payload.MaterialName,
		CategoryID:   payload.CategoryID,
		Ingredient:   payload.Ingredient,
		Remark:       payload.Remark,
		Remark1:      payload.Remark1,
		CreatedBy:    payload.CreatedBy,
		UpdatedBy:    payload.UpdatedBy,
		ScaleId:      payload.ScaleId,
		CheckCode:    checkCode,
		Output:       payload.Output,
	}

	if err := mgr.formulaPd.UpdateRawInfo(rec); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_EDIT, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_EDIT, MsgBody: "ok," + strconv.Itoa(payload.RecId)}
	SaveUpdateFmaRawDataLog(payload, oldRawData)
}

func (p rawDataDeletedNotifier) Handle(mgr *SrvMgr, payload ReqDelRawData) {
	// Do something for this event
	l.Log.Debug("Handle rawDataDeletedNotifier called")
	rec := payload.RecId
	if err := mgr.formulaPd.DeleteRawInfo(rec); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_DELETE, MsgBody: "failed to delete raw info"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW_DATA_DELETE, MsgBody: "ok," + strconv.Itoa(rec)}
	//TODO: 增加日志记录
}

// 增加配方
func (p addFormulaRecNotifier) Handle(mgr *SrvMgr, payload ReqAddFormulaData) {
	// Do something for this event
	l.Log.Debug("Handle addFormulaRecNotifier called")

	headerKey, _ := mgr.formulaPd.GetMaxFormulaRecKey() //获取最新的配方ID

	barcode := payload.Header.FormulaID
	if payload.Header.FormulaBarcode != "" {
		barcode = payload.Header.FormulaBarcode
	}

	var header FormulaHeader = FormulaHeader{
		FormulaID:      payload.Header.FormulaID,
		FormulaName:    payload.Header.FormulaName,
		FormulaKey:     headerKey + 1,
		CategoryID:     payload.Header.CategoryID,
		Remark:         payload.Header.Remark,
		CreatedBy:      payload.Header.CreatedBy,
		UpdatedBy:      payload.Header.UpdatedBy,
		FormulaMode:    payload.Header.FormulaMode,
		FormulaUnit:    payload.Header.FormulaUnit,
		TotalWeight:    payload.Header.TotalWeight,
		MaterialCount:  payload.Header.MaterialCount,
		IsEncrypted:    payload.Header.IsEncrypted,
		NeedContainer:  payload.Header.NeedContainer,
		FormulaBarcode: barcode,
	}

	if err := mgr.formulaPd.InsertFormulaHeader(header); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_ADD, MsgBody: err.Error()}
		return
	}

	headerId, _ := mgr.formulaPd.GetMaxFormulaRecId() //获取最新的配方ID

	for _, detail := range payload.Detail {
		tempRec := FormulaDetail{
			FormulaRecID:       headerId,
			MaterialID:         detail.MaterialID,
			MaterialWeight:     detail.MaterialWeight,
			MaterialPercentage: detail.MaterialPercentage,
			Sequence:           detail.Sequence,
			AllowableError:     detail.AllowableError,
			Remark:             detail.Remark,
		}
		if err := mgr.formulaPd.InsertFormulaBody(tempRec); err != nil {

			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_ADD, MsgBody: err.Error()}
			return
		}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_ADD, MsgBody: "ok," + strconv.Itoa(headerId)}
	SaveFmaDataAddLog(payload)

}

// 修改配方
func (p editFormulaRecNotifier) Handle(mgr *SrvMgr, payload ReqAddFormulaData) {
	l.Log.Debug("Handle editFormulaRecNotifier called")

	oldData, _ := mgr.formulaPd.GetFormulaListByFormulaID(payload.Header.FormulaID)
	barcode := payload.Header.FormulaBarcode
	if barcode == "" {
		barcode = payload.Header.FormulaID
	}

	var header FormulaHeader = FormulaHeader{
		RecId:          payload.Header.RecId,
		FormulaID:      payload.Header.FormulaID,
		FormulaName:    payload.Header.FormulaName,
		FormulaKey:     payload.Header.FormulaKey,
		CategoryID:     payload.Header.CategoryID,
		Remark:         payload.Header.Remark,
		CreatedBy:      payload.Header.CreatedBy,
		UpdatedBy:      payload.Header.UpdatedBy,
		FormulaMode:    payload.Header.FormulaMode,
		FormulaUnit:    payload.Header.FormulaUnit,
		TotalWeight:    payload.Header.TotalWeight,
		MaterialCount:  payload.Header.MaterialCount,
		IsEncrypted:    payload.Header.IsEncrypted,
		NeedContainer:  payload.Header.NeedContainer,
		FormulaBarcode: barcode,
	}

	details := []FormulaDetail{}
	for _, detail := range payload.Detail {
		tempRec := FormulaDetail{
			FormulaRecID:       payload.Header.FormulaKey,
			MaterialID:         detail.MaterialID,
			MaterialWeight:     detail.MaterialWeight,
			MaterialPercentage: detail.MaterialPercentage,
			Sequence:           detail.Sequence,
			AllowableError:     detail.AllowableError,
			Remark:             detail.Remark,
		}
		details = append(details, tempRec)
	}

	if err := mgr.formulaPd.UpdateFormula(header, details); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_UPDATE, MsgBody: "failed to update formula"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_UPDATE, MsgBody: "ok," + strconv.Itoa(payload.Header.RecId)}

	SaveUpdateFmaDataLog(payload, oldData)

}

// 检查配方ID和条码是否匹配
func (p checkFmaIdAndBarcodeNotifier) Handle(mgr *SrvMgr, payload ReqCheckFmaIdAndBarcode) {
	// Do something for this event
	l.Log.Debug("Handle checkFmaIdAndBarcodeNotifier called")
	barcode := payload.FormulaBarcode
	if barcode == "" {
		barcode = payload.FormulaID
	}
	idFlag, barcodeFlag, _ := mgr.formulaPd.CheckFmaIdAndBarcode(payload.RecId, payload.FormulaID, barcode)
	result := strconv.FormatBool(idFlag) + "," + strconv.FormatBool(barcodeFlag)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CHECK_FMA_ID_AND_BARCODE, MsgBody: result}

}

// 获取配方数据by 条码
func (p getFormulaByBarcodeNotifier) Handle(mgr *SrvMgr, payload ReqGetFormulaByBarcode) {
	// Do something for this event
	l.Log.Debug("Handle getFormulaByBarcodeNotifier called")
	recs, _ := mgr.formulaPd.GetFormulaDataByBarcode(payload.Barcode)
	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(recs); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_LIST_BY_BARCODE, MsgBody: ""}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_LIST_BY_BARCODE, MsgBody: typesStr}

}

// 获取配方
func (p getFormulaListNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getFormulaListNotifier called")
	recs, _ := mgr.formulaPd.GetFormulaDataList()
	//分批次发送
	const batchSize = 100
	numBatches := (len(recs) + batchSize - 1) / batchSize
	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(recs) {
			end = len(recs)
		}
		batch := recs[start:end]
		var typesStr string
		var err error
		if typesStr, err = json.MarshalToString(batch); err != nil {
			l.Log.Error(err)
			continue
		}
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_LIST, MsgBody: typesStr}
	}

	if len(recs) == 0 {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_LIST, MsgBody: ""}
	}

}

// 获取单个配方信息
func (p getFormulaDataNotifier) Handle(mgr *SrvMgr, payload string) {
	// Do something for this event
	l.Log.Debug("Handle getFormulaDataNotifier called")

	recId, err := strconv.Atoi(payload)
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA, MsgBody: ""}
		return
	}
	if recId == 0 {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA, MsgBody: ""}
		return
	}

	recs, _ := mgr.formulaPd.GetFormulaData(recId)
	var typesStr string

	if typesStr, err = json.MarshalToString(recs); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA, MsgBody: ""}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA, MsgBody: typesStr}

}

// 获取单个原料信息
func (p getRawDataNotifier) Handle(mgr *SrvMgr, payload string) {
	// Do something for this event
	l.Log.Debug("Handle getRawDataNotifier called")

	recId, err := strconv.Atoi(payload)
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW, MsgBody: ""}
		return
	}
	if recId == 0 {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW, MsgBody: ""}
		return
	}

	recs, _ := mgr.formulaPd.GetRawData(recId)
	var typesStr string

	if typesStr, err = json.MarshalToString(recs); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW, MsgBody: ""}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_RAW, MsgBody: typesStr}

}

// 传入原来的序号从payload中找出对应的ReqFormulaWgtRecDetail

func getDetailRecFromPayload(payload ReqFormulaWgtRec, seq int) ReqFormulaWgtRecDetail {
	for _, detail := range payload.RecDetail {
		if detail.Sequence == seq {
			return detail
		}
	}
	return ReqFormulaWgtRecDetail{}
}

// 新增配方称重记录
func (p addFormulaWgtRecNotifier) Handle(mgr *SrvMgr, payload ReqFormulaWgtRec) {
	// Do something for this event
	l.Log.Debug("Handle addFormulaWgtRecNotifier called")
	//先找出配方信息 写记录的时候，将原来的配方信息也写进去
	fmaId := payload.RecHeader.FormulaID
	fmaInfo, err := mgr.formulaPd.GetFormulaListByFormulaID(fmaId)

	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_ADD, MsgBody: "Formula not found"}
		return
	}

	formulaWgtRecList := FormulaWgtRecList{}

	newFmaHeader := FormulaWgtRecHeader{
		RecordID:          payload.RecHeader.RecordID,
		RecordSaveTime:    time.Now(),
		Operator:          payload.RecHeader.Operator,
		FormulaID:         fmaId,
		FormulaKey:        fmaInfo.Header.FormulaKey,
		FormulaName:       fmaInfo.Header.FormulaName,
		FormulaMode:       fmaInfo.Header.FormulaMode,
		FormulaTypeId:     fmaInfo.Header.CategoryID,
		FormulaTypeName:   payload.RecHeader.FormulaTypeName,
		TotalWeight:       payload.RecHeader.TotalWeight,
		ActualTotalWeight: payload.RecHeader.ActualTotalWeight,
		TotalWeightUnit:   payload.RecHeader.TotalWeightUnit,
		MaterialCount:     fmaInfo.Header.MaterialCount,
		Error:             0.0,
		IsQualified:       payload.RecHeader.IsQualified,
		ActualFmaTotalWgt: payload.RecHeader.ActualFmaTotalWgt, //实际配方总重量,包括修正后需要的重量
		IsEncrypted:       fmaInfo.Header.IsEncrypted,
		NeedContainer:     fmaInfo.Header.NeedContainer,
		FormulaCreatedAt:  fmaInfo.Header.CreatedAt,
		FormulaUpdatedAt:  fmaInfo.Header.UpdatedAt,
		FormulaCreatedBy:  fmaInfo.Header.CreatedBy,
		FormulaUpdatedBy:  fmaInfo.Header.UpdatedBy,
		FormulaRemark:     fmaInfo.Header.Remark,
		FormulaRemark1:    fmaInfo.Header.Remark1,
		ScaleId:           payload.RecHeader.ScaleId,
		ScaleName:         payload.RecHeader.ScaleName,
		ScaleModel:        payload.RecHeader.ScaleModel,
		ScaleSn:           payload.RecHeader.ScaleSn,
		FormulaBarcode:    fmaInfo.Header.FormulaBarcode,
	}

	formulaWgtRecList.Header = newFmaHeader

	//插入头
	if err := mgr.formulaPd.InsertFormulaWgtHeader(newFmaHeader); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_ADD, MsgBody: err.Error()}
		return
	}
	//如果有容器
	if fmaInfo.Header.NeedContainer {
		detailRec := getDetailRecFromPayload(payload, 0)
		newFmaDetail := FormulaWgtRecDetail{
			RecordID:           payload.RecHeader.RecordID,
			MaterialID:         "-",
			MaterialName:       "-",
			MaterialTypeID:     0,
			MaterialTypeName:   "-",
			Ingredient:         "-",
			MaterialCreatedAt:  time.Now(),
			MaterialUpdatedAt:  time.Now(),
			MaterialCreatedBy:  "-",
			MaterialUpdatedBy:  "-",
			MaterialRemark:     "-",
			MaterialRemark1:    "-",
			TargetWgt:          0,
			MaterialWeight:     0,
			MaterialPercentage: 0,
			Sequence:           0,
			AllowableError:     0,
			ActualWeight:       detailRec.ActualWeight,
			ActualPercentage:   0,
			ActualErrorWgt:     0,
			ActualErrorPct:     0,
			IsQualified:        detailRec.IsQualified,
			LastWeighingTime:   time.Now(),
			ScaleId:            detailRec.ScaleId,
			ScaleName:          detailRec.ScaleName,
			ScaleModel:         detailRec.ScaleModel,
			ScaleSn:            detailRec.ScaleSn,
			CheckCode:          "-",
		}

		formulaWgtRecList.Details = append(formulaWgtRecList.Details, newFmaDetail)
		//插入容器
		if err := mgr.formulaPd.InsertFormulaWgtBody(newFmaDetail); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_ADD, MsgBody: err.Error()}
			return
		}
	}

	//插入体
	for _, detail := range fmaInfo.Details {
		//先找到payload中对应的detail
		// newFmaDetail := FormulaWgtRecDetail{}
		//先找出是不是容器
		//不是容器
		//计算出配方的单重

		var rawWgt float64 = 0.0
		if fmaInfo.Header.FormulaMode == "pct" {
			rawWgt = math.Round(payload.RecHeader.ActualFmaTotalWgt*detail.MaterialPercentage/100*1000) / 1000
		} else {
			rawWgt = detail.MaterialWeight
		}
		detailRec := getDetailRecFromPayload(payload, detail.Sequence)
		rawInfo, _ := mgr.formulaPd.GetRawDataByRawID(detailRec.MaterialID)

		materialTypeId, err := strconv.Atoi(detailRec.MaterialTypeId)
		if err != nil {
			materialTypeId = 0
		}
		newFmaDetail := FormulaWgtRecDetail{
			RecordID:           payload.RecHeader.RecordID,
			MaterialID:         detailRec.MaterialID,
			MaterialName:       rawInfo.MaterialName,
			MaterialTypeID:     materialTypeId,
			MaterialTypeName:   detailRec.MaterialTypeName,
			Ingredient:         rawInfo.Ingredient,
			MaterialCreatedAt:  time.Now(),
			MaterialUpdatedAt:  time.Now(),
			MaterialCreatedBy:  rawInfo.CreatedBy,
			MaterialUpdatedBy:  rawInfo.UpdatedBy,
			MaterialRemark:     detail.Remark,
			MaterialRemark1:    detail.Remark1,
			TargetWgt:          detailRec.TargetWgt,
			MaterialWeight:     rawWgt,
			MaterialPercentage: detail.MaterialPercentage,
			Sequence:           detail.Sequence,
			AllowableError:     detail.AllowableError,
			ActualWeight:       detailRec.ActualWeight,
			ActualPercentage:   detailRec.ActualPercentage,
			ActualErrorWgt:     detailRec.ActualErrorWgt,
			ActualErrorPct:     detailRec.ActualErrorPct,
			IsQualified:        detailRec.IsQualified,
			LastWeighingTime:   time.Now(),
			ScaleId:            detailRec.ScaleId,
			ScaleName:          detailRec.ScaleName,
			ScaleModel:         detailRec.ScaleModel,
			ScaleSn:            detailRec.ScaleSn,
			CheckCode:          detailRec.CheckCode,
		}
		formulaWgtRecList.Details = append(formulaWgtRecList.Details, newFmaDetail)
		//插入详细
		if err := mgr.formulaPd.InsertFormulaWgtBody(newFmaDetail); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_ADD, MsgBody: err.Error()}
			return
		}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_ADD, MsgBody: "ok"}
	rec, err := mgr.formulaPd.GetFmaWgtRecByOrderId(payload.RecHeader.RecordID)
	if err != nil {
		return
	}
	SaveFmaWgtRecLog(rec)
	if fmaInfo.Header.IsEncrypted {
		return
	}

	SendFmaWgtRecToPcServer(rec, payload.RecHeader.RecordID, mgr)
}

func createRptCsvFile(rec FormulaWgtRecList, orderId string) (string, error) {
	// 创建CSV格式文件
	csvFilePath := filepath.Join(comm.GetExePath(), orderId+".csv")

	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		return "", fmt.Errorf("创建CSV文件失败: %v", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)

	// 确保在函数返回前刷新缓冲区
	defer func() {
		writer.Flush()
		// 检查写入过程中是否有错误
		if writer.Error() != nil {
			fmt.Printf("警告：CSV写入过程中可能出现错误: %v\n", writer.Error())
		}
	}()

	// 写入表头
	header := []string{"No.", "Formula Id", "Formula Name", "Barcode", "Ingredient Name", "Ingredient Id", "Mode", "Confidential", "Formula Total Weight", "Actual Total Weight", "Ingredient Weight", "Actual Ingredient Weight", "Allowable Error", "Actual Error", "Pass", "Created Time", "Operator"}
	if err := writer.Write(header); err != nil {
		return csvFilePath, fmt.Errorf("写入CSV表头失败: %v", err)
	}

	// 写入头行
	rowHeader := []string{
		rec.Header.RecordID,
		rec.Header.FormulaID,
		rec.Header.FormulaName,
		rec.Header.FormulaBarcode,
		"",
		"",
		rec.Header.FormulaMode,
		fmt.Sprintf("%v", rec.Header.IsEncrypted),
		fmt.Sprintf("%.3f", rec.Header.ActualFmaTotalWgt) + rec.Header.TotalWeightUnit,
		fmt.Sprintf("%.3f", rec.Header.ActualTotalWeight) + rec.Header.TotalWeightUnit,
		"",
		"",
		"",
		"",
		rec.Header.IsQualified,
		rec.Header.RecordSaveTime.Format("2006-01-02 15:04:05"),
		rec.Header.Operator,
	}
	if err := writer.Write(rowHeader); err != nil {
		return csvFilePath, fmt.Errorf("写入CSV头行失败: %v", err)
	}

	// 写入数据行
	for _, detail := range rec.Details {
		row := []string{
			"",
			"",
			"",
			"",
			detail.MaterialName,
			detail.MaterialID,
			"",
			"",
			"",
			"",
			fmt.Sprintf("%.3f", detail.MaterialWeight) + rec.Header.TotalWeightUnit,
			fmt.Sprintf("%.3f", detail.ActualWeight) + rec.Header.TotalWeightUnit,
			fmt.Sprintf("%.3f", detail.AllowableError) + rec.Header.TotalWeightUnit,
			fmt.Sprintf("%.3f", detail.ActualErrorWgt) + rec.Header.TotalWeightUnit,
			fmt.Sprintf("%v", detail.IsQualified),
			"",
			"",
		}
		if err := writer.Write(row); err != nil {
			return csvFilePath, fmt.Errorf("写入CSV数据行失败: %v", err)
		}
	}

	// 强制刷新缓冲区
	writer.Flush()
	if writer.Error() != nil {
		return csvFilePath, fmt.Errorf("刷新CSV缓冲区失败: %v", writer.Error())
	}

	// 可选：强制同步到磁盘（如果需要更高的数据安全性）
	if err := csvFile.Sync(); err != nil {
		return csvFilePath, fmt.Errorf("同步文件到磁盘失败: %v", err)
	}

	return csvFilePath, nil
}

func SendFmaWgtRecToPcServer(rec FormulaWgtRecList, orderId string, mgr *SrvMgr) {

	uploadServerInfo, err := mgr.formulaPd.GetUploadServerInfo()
	if err != nil {
		return
	}
	if uploadServerInfo == (UploadServerInfo{}) {
		return
	}

	if !uploadServerInfo.Enable {
		return
	}

	csvFilePath, err := createRptCsvFile(rec, orderId)
	if err != nil {
		fmt.Println("创建CSV文件失败:", err)
		return
	}

	config := SMBUploader{
		ServerIP:  uploadServerInfo.Ip,
		ShareName: uploadServerInfo.ShareName,
		Username:  uploadServerInfo.Username,
		Password:  uploadServerInfo.Password,
	}

	uploader := NewSMBUploader(
		config.ServerIP,
		config.ShareName,
		config.Username,
		config.Password,
	)

	UploadFileByPath(uploader, csvFilePath)

}

// 根据订单号获取配方称重记录
func (p getFmaRecByOrderIdNotifier) Handle(mgr *SrvMgr, payload string) {
	//停留几秒再获取数据

	rec, err := mgr.formulaPd.GetFmaWgtRecByOrderId(payload)
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_BY_ORDER, MsgBody: "fail"}
		return
	}
	recStr, _ := json.MarshalToString(rec)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_BY_ORDER, MsgBody: recStr}

}

// 获取配方称重记录
func (p getFormulaWgtRecListNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getFormulaWgtRecListNotifier called")
	recs, _ := mgr.formulaPd.GetFormulaWgtRecList()

	//分批发送
	const batchSize = 100
	for i := 0; i < len(recs); i += batchSize {
		end := i + batchSize
		if end > len(recs) {
			end = len(recs)
		}
		batchRecs := recs[i:end]

		var typesStr string
		var err error
		if typesStr, err = json.MarshalToString(batchRecs); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_LIST, MsgBody: ""}
			return
		}
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_LIST, MsgBody: typesStr}
		time.Sleep(time.Millisecond * 100)
	}
	if len(recs) == 0 {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_REC_LIST, MsgBody: ""}
	}

}

// 根据配方ID获取配方称重记录
func (p getOneFormulaWgtRecListNotifier) Handle(mgr *SrvMgr, payload string) {
	// Do something for this event
	l.Log.Debug("Handle getOneFormulaWgtRecListNotifier called")
	recs, _ := mgr.formulaPd.GetOneFormulaWgtRecList(payload)

	//分批发送
	const batchSize = 100
	for i := 0; i < len(recs); i += batchSize {
		end := i + batchSize
		if end > len(recs) {
			end = len(recs)
		}
		batchRecs := recs[i:end]

		var typesStr string
		var err error
		if typesStr, err = json.MarshalToString(batchRecs); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ONE_FORMULA_REC_LIST, MsgBody: ""}
			return
		}
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ONE_FORMULA_REC_LIST, MsgBody: typesStr}
		time.Sleep(time.Millisecond * 100)
	}
	if len(recs) == 0 {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ONE_FORMULA_REC_LIST, MsgBody: ""}
	}

}

// 删除配方
func (p delFormulaNotifier) Handle(mgr *SrvMgr, payload ReqDelFmaData) {
	// Do something for this event
	l.Log.Debug("Handle delFormulaNotifier called")

	oldFmaData, _ := mgr.formulaPd.GetFormulaByRecId(payload.RecId)

	rec := payload.RecId
	if err := mgr.formulaPd.DeleteFormula(rec); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_DELETE, MsgBody: "failed to delete formula "}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FORMULA_DELETE, MsgBody: "ok," + strconv.Itoa(rec)}
	SaveDeleteFormulaLog(oldFmaData)

}

// 删除所有配方
func (p delAllFormulaNotifier) Handle(mgr *SrvMgr, payload ReqDelAllFmaData) {
	// Do something for this event
	l.Log.Debug("Handle delAllFormulaNotifier called")
	recs := payload.RecID
	if err := mgr.formulaPd.DeleteAllFormulas(recs); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MANY_FMA_DELETE, MsgBody: "failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MANY_FMA_DELETE, MsgBody: "ok"}
	//TODO: 增加日志记录
	//记录删除的配方，以及配方相关的暂存的称重记录
	// SaveDeleteManyFormulaLog(oldFmaData)

}

// 删除所有原料数据
func (p delAllRawDataNotifier) Handle(mgr *SrvMgr, payload ReqDelAllRawData) {
	// Do something for this event
	l.Log.Debug("Handle delAllRawDataNotifier called")
	recs := payload.RecId
	if err := mgr.formulaPd.DeleteAllRawInfo(recs); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MANY_RAW_DELETE, MsgBody: "failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MANY_RAW_DELETE, MsgBody: "ok"}
	//TODO: 增加日志记录

}

// 删除所有暂存配方称重记录
func (p delAllDraftFmaWgtRecNotifier) Handle(mgr *SrvMgr, payload ReqDeleteAllDraftFmaWgtRec) {
	// Do something for this event
	l.Log.Debug("Handle delAllDraftFmaWgtRecNotifier called")

	// 查询暂存配方称重记录ByOrderId
	draftFmaWgtRecLists, _ := mgr.formulaPd.GetDraftFmaWgtRecByOrderId(payload.OrderId)
	recs := payload.OrderId
	if err := mgr.formulaPd.DeleteAllDraftFmaWgtRec(recs); err != nil {

		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MANY_DRAFT_FMA_WGT_REC_DELETE, MsgBody: "failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MANY_DRAFT_FMA_WGT_REC_DELETE, MsgBody: "ok"}
	//记录删除的暂存配方称重记录
	if len(draftFmaWgtRecLists) == 0 {
		return
	}
	SaveDeleteManyDraftFmaWgtRecLog(draftFmaWgtRecLists)

}

// 增加流速
func (p addFlowRateNotifier) Handle(mgr *SrvMgr, payload ReqFlowRateRec) {
	// Do something for this event
	l.Log.Debug("Handle addFlowRateNotifier called")
	var header FlowRateHeader = FlowRateHeader{
		TotalWeight:     payload.RecHeader.TotalWeight,
		TotalTime:       payload.RecHeader.TotalTime,
		AverageFlowRate: payload.RecHeader.AverageFlowRate,
		MinFlowRate:     payload.RecHeader.MinFlowRate,
		MaxFlowRate:     payload.RecHeader.MaxFlowRate,
		WgtUnit:         payload.RecHeader.WgtUnit,
	}
	if err := mgr.flowRatePd.infoPb.AddFlowRateHeader(&header); err != nil {
		// TODO: error handling
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FLOW_RATE_ADD, MsgBody: err.Error()}
	}

	//找出头表中最大的recid
	recId, err := mgr.flowRatePd.infoPb.FindMaxRecId()
	if err != nil {
		// TODO: error handling
		recId = 0
	}
	for _, detail := range payload.RecDetail {
		tempRec := FlowRateDetail{
			HeaderId: recId,
			Id:       detail.Id,
			Rate:     detail.Rate,
			Time:     detail.Time,
		}
		//插入详细
		if err := mgr.flowRatePd.infoPb.AddFlowRateDetail(&tempRec); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FLOW_RATE_ADD, MsgBody: err.Error()}
		}
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FLOW_RATE_ADD, MsgBody: "ok"}

}

// 获取流速
func (p getFlowRateListNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getFlowRateNotifier called")
	recs, _ := mgr.flowRatePd.GetFlowRateList()

	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(recs); err != nil {
		l.Log.Error(err)

	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_FLOW_RATE_LIST, MsgBody: typesStr}

}

// 删除称重记录
func (p delWgtRecNotifier) Handle(mgr *SrvMgr, payload ReqDelWgtRec) {
	// Do something for this event
	l.Log.Debug("Handle delWgtRecNotifier called")
	mode := payload.Mode

	switch mode {
	case 0:
		if err := NewScaleRecProvider().NewDeleteAllRec(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC, MsgBody: err.Error()}
			return
		}

	case 1:
		if err := NewScaleRecCheckWeigherProvider().NewDeleteAllRec(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC, MsgBody: err.Error()}
			return
		}

	case 2:
		if err := NewScaleRecTakeInProvider().NewDeleteAllRec(); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC, MsgBody: err.Error()}
			return
		}

	case 3:
		if err := NewScaleRecTakeOutProvider().NewDeleteAllRec(); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC, MsgBody: err.Error()}
			return
		}

	}

	SaveClearScaleWgtLog(int(mode))

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC, MsgBody: "ok"}

}

// 删除称重记录
func (p delWgtRecByIdNotifier) Handle(mgr *SrvMgr, payload ReqDelWgtRecById) {
	// Do something for this event
	l.Log.Debug("Handle delWgtRecByIdNotifier called")
	mode := payload.Mode
	recId := payload.RecId
	switch mode {
	case 0:
		if err := NewScaleRecProvider().DeleteRec(recId); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC_BY_ID, MsgBody: err.Error()}
			return
		}

	case 1:
		if err := NewScaleRecCheckWeigherProvider().DeleteRec(recId); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC_BY_ID, MsgBody: err.Error()}
			return
		}

	case 2:
		if err := NewScaleRecTakeInProvider().DeleteRec(recId); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC_BY_ID, MsgBody: err.Error()}
			return
		}

	case 3:
		if err := NewScaleRecTakeOutProvider().DeleteRec(recId); err != nil {
			// TODO: error handling
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC_BY_ID, MsgBody: err.Error()}
			return
		}

	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_WGT_REC_BY_ID, MsgBody: "ok"}
	//TODO: 增加日志记录
}

// 获取称重记录
func (p getAllWgtRecListNotifier) Handle(mgr *SrvMgr, payload ReqGetAllWgtRecList) {
	// Do something for this event
	l.Log.Debug("Handle getAllWgtRecListNotifier called")
	var recs PagedScaleRecInfo
	var err error
	mode := payload.Mode
	switch mode {
	case 0:
		if recs, err = NewScaleRecProvider().NewGetRecsList(payload.Page, payload.PageSize, payload.ColumnName, payload.Direction); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_WGT_REC_LIST, MsgBody: err.Error()}
			return
		}

	case 1:
		if recs, err = NewScaleRecCheckWeigherProvider().NewGetRecsList(payload.Page, payload.PageSize, payload.ColumnName, payload.Direction); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_WGT_REC_LIST, MsgBody: err.Error()}
			return
		}

	case 2:
		if recs, err = NewScaleRecTakeInProvider().NewGetRecsList(payload.Page, payload.PageSize, payload.ColumnName, payload.Direction); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_WGT_REC_LIST, MsgBody: err.Error()}
			return
		}

	case 3:
		if recs, err = NewScaleRecTakeOutProvider().NewGetRecsList(payload.Page, payload.PageSize, payload.ColumnName, payload.Direction); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_WGT_REC_LIST, MsgBody: err.Error()}
			return
		}

	}
	var typesStr string
	if typesStr, err = json.MarshalToString(recs); err != nil {
		l.Log.Error(err)
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_WGT_REC_LIST, MsgBody: typesStr}
}

// 获取称重记录
func (p getSearchRecListNotifier) Handle(mgr *SrvMgr, payload ReqGetSearchRecList) {
	// Do something for this event
	// l.Log.Debug("Handle getSearchRecListNotifier called")
	// recs, _ := NewWgtRecProvider().GetSearchRecList(payload.SearchKey)

	var typesStr string
	// var err error
	// if typesStr, err = json.MarshalToString(recs); err != nil {
	// 	l.Log.Error(err)

	// }
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SEARCH_REC_LIST, MsgBody: typesStr}
}

// 增加称重记录
func (p addWgtRecNotifier) Handle(mgr *SrvMgr, payload ReqAddWgtRec) {
	// Do something for this event
	l.Log.Debug("Handle addWgtRecNotifier called")
	// 找出头表中最大的recid
	var err error
	mode := payload.Mode
	switch mode {
	case 0:
		if err := NewScaleRecProvider().InsertRec(payload.HeadRec); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}
		var maxId uint
		if maxId, err = NewScaleRecProvider().FindMaxRecId(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}
		for _, detail := range payload.DetailRec {
			detail.HeadId = maxId
			if err := NewScaleRecProvider().InsertScaleRecDetail(detail); err != nil {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
				return
			}
		}

	case 1:

		if err := NewScaleRecCheckWeigherProvider().InsertRec(payload.HeadRec); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}
		var maxId uint
		if maxId, err = NewScaleRecCheckWeigherProvider().FindMaxRecId(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}

		for _, detail := range payload.DetailRec {
			detail.HeadId = maxId
			if err := NewScaleRecCheckWeigherProvider().InsertScaleRecDetail(detail); err != nil {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
				return
			}
		}

	case 2:

		if err := NewScaleRecTakeInProvider().InsertRec(payload.HeadRec); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}
		var maxId uint
		if maxId, err = NewScaleRecTakeInProvider().FindMaxRecId(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}

		for _, detail := range payload.DetailRec {
			detail.HeadId = maxId
			if err := NewScaleRecTakeInProvider().InsertScaleRecDetail(detail); err != nil {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
				return
			}
		}

	case 3:

		if err := NewScaleRecTakeOutProvider().InsertRec(payload.HeadRec); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}
		var maxId uint
		if maxId, err = NewScaleRecTakeOutProvider().FindMaxRecId(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
			return
		}

		for _, detail := range payload.DetailRec {
			detail.HeadId = maxId
			if err := NewScaleRecTakeOutProvider().InsertScaleRecDetail(detail); err != nil {
				mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: err.Error()}
				return
			}
		}

	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_WGT_REC, MsgBody: "ok"}
	SaveAddScaleWgtLog(payload)

}

// 导出所有记录
func (p exportAllRecsNotifier) Handle(mgr *SrvMgr, payload ReqExportAllRecs) {
	// Do something for this event
	l.Log.Debug("Handle exportAllRecsNotifier called")
	// Do something with this event
	// 导出所有记录
	var recs []ScaleRecInfo
	var config []ModeSetting
	var err error
	switch payload.Mode {
	case 0:
		if recs, err = NewScaleRecProvider().GetAllScaleRecInfos(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
			return
		}
		config, _ = mSrvMgr.modeSetting.GetModeSetting(0)
	case 1:
		if recs, err = NewScaleRecCheckWeigherProvider().GetAllScaleRecInfos(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
			return
		}
		config, _ = mSrvMgr.modeSetting.GetModeSetting(1)
	case 2:
		if recs, err = NewScaleRecTakeInProvider().GetAllScaleRecInfos(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
			return
		}
		config, _ = mSrvMgr.modeSetting.GetModeSetting(2)
	case 3:
		if recs, err = NewScaleRecTakeOutProvider().GetAllScaleRecInfos(); err != nil {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
			return
		}
		config, _ = mSrvMgr.modeSetting.GetModeSetting(3)

	}

	selFields := payload.FieldName
	translation := payload.Translation

	// 打开 CSV 文件
	file, err := os.Create(payload.Path)
	if err != nil {
		mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
		return
	}
	defer file.Close()

	// 创建 CSV 写入器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	//将recs写入csv文件，文件路径为payload.Path
	//下面是总的表头

	headers := []string{}

	for _, field := range selFields {
		switch field {
		case "Id":
			headers = append(headers, getTranslation(translation, "Id"))
		case "Weight":
			headers = append(headers, getTranslation(translation, "Weight"))
		case "Weight Unit":
			headers = append(headers, getTranslation(translation, "Weight Unit"))
		case "Scale Name":
			headers = append(headers, getTranslation(translation, "Scale Name"))
		case "PLU":
			headers = append(headers, getTranslation(translation, "PLU"))
		case "PLU Name":
			headers = append(headers, getTranslation(translation, "PLU Name"))
		case "Price":
			headers = append(headers, getTranslation(translation, "Price"))
		case "Product Code":
			headers = append(headers, getTranslation(translation, "Product Code"))
		case "Item Code":
			headers = append(headers, getTranslation(translation, "Item Code"))
		case "Category":
			headers = append(headers, getTranslation(translation, "Category"))
		case "GeneralUnit":
			headers = append(headers, getTranslation(translation, "GeneralUnit"))
		case "TaxType":
			headers = append(headers, getTranslation(translation, "TaxType"))
		case "UnitWeight":
			headers = append(headers, getTranslation(translation, "UnitWeight"))
		case "Pretare":
			headers = append(headers, getTranslation(translation, "Pretare"))
		case "LimitHigh":
			headers = append(headers, getTranslation(translation, "LimitHigh"))
		case "LimitLow":
			headers = append(headers, getTranslation(translation, "LimitLow"))
		case "User Name":
			headers = append(headers, getTranslation(translation, "User Name"))
		case "Date Time":
			headers = append(headers, getTranslation(translation, "Date Time"))
		}
	}
	//明细的头和表头共用即可，不需要重新写
	// 写入总的标题
	if err := writer.Write(headers); err != nil {
		mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
		return
	}

	//日期和分隔符的格式来源于config[0].DateFormat 和 config[0].Delimiter
	dateFormat := config[0].DateFormat
	delimiter := config[0].DateSeparator

	// 定义日期格式模板
	var formatTemplate string
	switch dateFormat {
	case "1":
		formatTemplate = "2006" + delimiter + "01" + delimiter + "02" + " 15:04:05"
	case "2":
		formatTemplate = "02" + delimiter + "01" + delimiter + "2006" + " 15:04:05"
	case "3":
		formatTemplate = "01" + delimiter + "02" + delimiter + "2006" + " 15:04:05"
	default:
		// 默认使用 yyyy-mm-dd hh:mm:ss 格式
		formatTemplate = "2006" + delimiter + "01" + delimiter + "02 15:04:05"
	}

	// 遍历每个 ScaleRecInfo
	for _, info := range recs {
		// 提取表头数据，需要根据 ScaleRec 结构体实际字段调整
		createdAtFormatted := info.Header.CreatedAt.Format(formatTemplate)

		headerData := []string{}
		for _, field := range selFields {
			switch field {
			case "Id":
				headerData = append(headerData, fmt.Sprint(info.Header.RecId))
			case "Weight":
				headerData = append(headerData, fmt.Sprint(info.Header.Weight))
			case "Weight Unit":
				headerData = append(headerData, fmt.Sprint(info.Header.WeightUnit))
			case "Scale Name":
				headerData = append(headerData, fmt.Sprint(info.Header.ScaleName))
			case "PLU":
				headerData = append(headerData, fmt.Sprint(info.Header.Plu))
			case "PLU Name":
				headerData = append(headerData, fmt.Sprint(info.Header.ProductName))
			case "Price":
				headerData = append(headerData, fmt.Sprint(info.Header.Price))
			case "Product Code":
				headerData = append(headerData, fmt.Sprint(info.Header.ProductCode))
			case "Item Code":
				headerData = append(headerData, fmt.Sprint(info.Header.ItemCode))
			case "Category":
				headerData = append(headerData, fmt.Sprint(info.Header.Category))
			case "GeneralUnit":
				unitMap := map[string]string{"0": "kg", "1": "100g", "2": "pcs", "3": "lb", "4": "g", "5": "oz", "6": "lboz", "7": "tj", "8": "hj", "9": "t"}
				if unit, ok := unitMap[info.Header.GeneralUnit]; ok {
					headerData = append(headerData, unit)
				} else {
					headerData = append(headerData, fmt.Sprint(info.Header.GeneralUnit))
				}
			case "TaxType":
				taxMap := map[string]string{"0": "tax1", "1": "tax2", "2": "tax3"}
				if tax, ok := taxMap[info.Header.TaxType]; ok {
					headerData = append(headerData, tax)
				} else {
					headerData = append(headerData, fmt.Sprint(info.Header.TaxType))
				}
			case "UnitWeight":
				headerData = append(headerData, fmt.Sprint(info.Header.UnitWeight))
			case "Pretare":
				headerData = append(headerData, fmt.Sprint(info.Header.Pretare))
			case "LimitHigh":
				headerData = append(headerData, fmt.Sprint(info.Header.LimitHigh))
			case "LimitLow":
				headerData = append(headerData, fmt.Sprint(info.Header.LimitLow))
			case "User Name":
				headerData = append(headerData, fmt.Sprint(info.Header.UserName))
			case "Date Time":
				headerData = append(headerData, createdAtFormatted)
			}
		}

		if err := writer.Write(headerData); err != nil {
			mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
			return
		}

		// 若有明细信息，写入明细
		if len(info.Details) > 0 {
			for _, detail := range info.Details {
				// 假设明细和表头字段相同，若不同需要调整

				headerData := []string{}

				for _, field := range selFields {
					switch field {
					case "Id":
						headerData = append(headerData, "")
					case "Weight":
						headerData = append(headerData, fmt.Sprint(detail.Weight))
					case "Weight Unit":
						headerData = append(headerData, fmt.Sprint(detail.WeightUnit))
					case "Scale Name":
						headerData = append(headerData, fmt.Sprint(detail.ScaleName))
					case "PLU":
						headerData = append(headerData, "")
					case "PLU Name":
						headerData = append(headerData, "")
					case "Price":
						headerData = append(headerData, "")
					case "Product Code":
						headerData = append(headerData, "")
					case "Item Code":
						headerData = append(headerData, "")
					case "Category":
						headerData = append(headerData, "")
					case "GeneralUnit":
						headerData = append(headerData, "")
					case "TaxType":
						headerData = append(headerData, "")
					case "UnitWeight":
						headerData = append(headerData, "")
					case "Pretare":
						headerData = append(headerData, "")
					case "LimitHigh":
						headerData = append(headerData, "")
					case "LimitLow":
						headerData = append(headerData, "")
					case "User Name":
						headerData = append(headerData, "")
					case "Date Time":
						headerData = append(headerData, createdAtFormatted)
					}
				}

				if err := writer.Write(headerData); err != nil {
					mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: err.Error()}
					return
				}
			}
		}
	}

	// 发送成功消息
	mgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_ALL_RECS, MsgBody: "ok," + payload.Path}
	//TODO: 增加日志记录
}

func getTranslation(trans map[string]string, field string) string {
	if val, ok := trans[field]; ok {
		return val
	}
	return field
}

//获取自动下一步设置

func (p getAutoNextNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle rawDataListedNotifier called")
	formulaRecProvider := mgr.formulaPd
	rec, _ := formulaRecProvider.GetSetAutoNext()

	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(rec); err != nil {
		l.Log.Error(err)

	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_AUTO_NEXT, MsgBody: typesStr}
}

// 获取不稳定归零扣重设置
func (p getUnstableZeroTareNotifier) Handle(mgr *SrvMgr) {
	l.Log.Debug("Handle getUnstableZeroTareNotifier called")
	val, _ := mgr.formulaPd.GetUnstableZeroTare()
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_UNSTABLE_ZERO_TARE, MsgBody: fmt.Sprintf("%v", val)}
}

// 更新不稳定归零扣重设置
func (p updateUnstableZeroTareNotifier) Handle(mgr *SrvMgr, payload ReqUpdateUnstableZeroTare) {
	l.Log.Debug("Handle updateUnstableZeroTareNotifier called")
	err := mgr.formulaPd.UpdateUnstableZeroTare(payload.Enable)
	msgBody := "ok"
	if err != nil {
		msgBody = "fail"
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_UNSTABLE_ZERO_TARE, MsgBody: msgBody}
}

// 更新自动下一步设置
func (p updateAutoNextNotifier) Handle(mgr *SrvMgr, payload ReqUpdateAutoNext) {
	// Do something for this event
	l.Log.Debug("Handle updateAutoNextNotifier called")

	setAutoNext := SetAutoNext{
		AutoNext:   payload.AutoNext,
		StableTime: payload.StableTime,
		AutoTare:   payload.AutoTare,
		CheckCode:  payload.CheckCode,
	}
	if err := mgr.formulaPd.UpdateSetAutoNext(setAutoNext); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_AUTO_NEXT, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_AUTO_NEXT, MsgBody: "ok"}
	//TODO: 增加日志记录
}

// 修改打印报表字段设置
func (p updateSetReportPrintNotifier) Handle(mgr *SrvMgr, payload SetReportPrint) {
	// Do something for this event
	l.Log.Debug("Handle updateSetReportPrintNotifier called")

	if err := mgr.formulaPd.UpdateSetReportPrint(payload); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SET_REPORT_PRINT, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SET_REPORT_PRINT, MsgBody: "ok"}
	//TODO: 增加日志记录

}

// 获取打印报表字段设置
func (p getSetReportPrintNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getSetReportPrintNotifier called")
	setReportPrint, err := mgr.formulaPd.GetSetReportPrint()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SET_REPORT_PRINT, MsgBody: err.Error()}
		return
	}
	var typesStr string
	if typesStr, err = json.MarshalToString(setReportPrint); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SET_REPORT_PRINT, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SET_REPORT_PRINT, MsgBody: typesStr}

}

// 创建暂存配方
func (p addDraftFmaWgtRecNotifier) Handle(mgr *SrvMgr, payload DrafFmaWgtRecInfo) {
	// Do something for this event
	l.Log.Debug("Handle createDraftFmaWgtRecNotifier called")

	tempHeader := DrafFmaWgtRecHeader{
		OrderId:   payload.Header.OrderId,
		FormulaID: payload.Header.FormulaID,
		//创建人
		CreatedBy: payload.Header.CreatedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UpdatedBy: payload.Header.UpdatedBy,
		Status:    0, // 草稿状态
		Remark:    payload.Header.Remark,
		Remark1:   payload.Header.Remark1,
		Remark2:   payload.Header.Remark2,
	}
	//插入头
	if err := mgr.formulaPd.CreateDraftFmaWgtRecHeader(tempHeader); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CREATE_DRAFT_FMA_WGT_REC, MsgBody: err.Error()}
		return
	}

	for _, detail := range payload.Details {
		tempRec := DrafFmaWgtRecDetail{
			OrderId:          payload.Header.OrderId,
			RawMaterialID:    detail.RawMaterialID,
			ActualWeight:     detail.ActualWeight,
			ActualWeightUnit: detail.ActualWeightUnit,
			IsContainer:      detail.IsContainer,
			Seq:              detail.Seq,
			Remark:           detail.Remark,
			Remark1:          detail.Remark1,
			Remark2:          detail.Remark2,
			ScaleId:          detail.ScaleId,
			ScaleName:        detail.ScaleName,
			ScaleModel:       detail.ScaleModel,
			ScaleSn:          detail.ScaleSn,
		}

		if err := mgr.formulaPd.CreateDraftFmaWgtRecDetail(tempRec); err != nil {
			l.Log.Error(err)
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CREATE_DRAFT_FMA_WGT_REC, MsgBody: err.Error()}
			return
		}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CREATE_DRAFT_FMA_WGT_REC, MsgBody: "ok"}
	SaveFmaDarftAddLog(payload)
}

// 删除暂存的配方
func (p deleteDraftFmaWgtRecNotifier) Handle(mgr *SrvMgr, payload ReqDeleteDraftFmaWgtRec) {
	// Do something for this event
	l.Log.Debug("Handle deleteDraftFmaWgtRecNotifier called")
	orders := []string{payload.OrderId}

	draftFmaWgtRecLists, _ := mgr.formulaPd.GetDraftFmaWgtRecByOrderId(orders)

	if err := mgr.formulaPd.DeleteDraftFmaWgtRec(payload.OrderId); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DELETE_DRAFT_FMA_WGT_REC, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DELETE_DRAFT_FMA_WGT_REC, MsgBody: "ok"}
	SaveFmaDarftDelLog(draftFmaWgtRecLists)
}

// 更新暂存配方
func (p updateDraftFmaWgtRecNotifier) Handle(mgr *SrvMgr, payload DrafFmaWgtRecInfo) {
	// Do something for this event
	l.Log.Debug("Handle updateDraftFmaWgtRecNotifier called")

	orders := []string{payload.Header.OrderId}
	draftFmaWgtRecLists, _ := mgr.formulaPd.GetDraftFmaWgtRecByOrderId(orders)

	if err := mgr.formulaPd.UpdateDraftFmaWgtRec(payload); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_DRAFT_FMA_WGT_REC, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_DRAFT_FMA_WGT_REC, MsgBody: "ok"}
	SaveFmaDarftUpdateLog(payload, draftFmaWgtRecLists)
}

// 获取暂存配方列表
func (p getDraftFmaWgtRecListNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getDraftFmaWgtRecListNotifier called")

	recs, err := mgr.formulaPd.GetDraftFmaWgtRec()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_DRAFT_FMA_WGT_REC_LIST, MsgBody: err.Error()}
		return
	}
	var typesStr string
	if typesStr, err = json.MarshalToString(recs); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_DRAFT_FMA_WGT_REC_LIST, MsgBody: err.Error()}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_DRAFT_FMA_WGT_REC_LIST, MsgBody: typesStr}
}

// 新增用户
func (p addSysUserNotifier) Handle(mgr *SrvMgr, payload ReqAddSysUser) {
	// Do something for this event
	l.Log.Debug("Handle addSysUserNotifier called")

	_, username, _ := GetCurrentUser()

	userInfo := SysUser{
		CreatedBy:     payload.CreatedBy,
		UpdatedBy:     payload.UpdatedBy,
		RoleId:        payload.RoleId,
		UserName:      payload.Username,
		NickName:      payload.NickName,
		Password:      payload.Password,
		Email:         payload.Email,
		Phone:         payload.Phone,
		InitialPageId: payload.InitialPageId,
		Remark:        payload.Remark,
		CreatedByName: username,
		UpdatedByName: username,
	}
	err := mSrvMgr.sysUserPd.AddUser(&userInfo)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_SYS_USER, MsgBody: "fail,add user failed"}
		return
	}

	newUser, err := mSrvMgr.sysUserPd.GetUserInfo(payload.Username)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_SYS_USER, MsgBody: "fail,add user failed"}
		return
	}

	pagesId := payload.PagesId
	err = mSrvMgr.sysUserPd.UpdateUserPageId(newUser.UserId, pagesId)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_SYS_USER, MsgBody: "fail,add user failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_ADD_SYS_USER, MsgBody: "ok"}

	jsonStr := SaveAddUserFunc(newUser, pagesId)
	LogSysOperation(MenuUserManage, SubAdd, OpAddStr, jsonStr, "ok", "")
}

// 删除用户
func (p deleteSysUserNotifier) Handle(mgr *SrvMgr, payload ReqSysUserIdList) {
	// Do something for this event
	l.Log.Debug("Handle deleteSysUserNotifier called")
	// 记录操作日志
	users, _ := mSrvMgr.sysUserPd.GetManyUserInfoById(payload.UserIds)
	if err := mSrvMgr.sysUserPd.DeleteUser((payload.UserIds)); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DELETE_SYS_USER, MsgBody: "fail,delete user failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DELETE_SYS_USER, MsgBody: "ok"}

	jsonStr := SaveDelUserFunc(users)
	LogSysOperation(MenuUserManage, SubDel, OpDeleteStr, jsonStr, "ok", "")
}

// 更新用户
func (p updateSysUserNotifier) Handle(mgr *SrvMgr, payload ReqUpdateSysUser) {
	l.Log.Debug("Handle updateSysUserNotifier called")
	// 检查用户是否存在
	user, err := mSrvMgr.sysUserPd.GetUserInfoById(payload.UpdateUser.UserId)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SYS_USER, MsgBody: "fail,user not exist"}
		return
	}

	// 比较新旧数据，找出变更的字段和原始值
	updatedFields, oldFieldValues := CompareUserFields(user, payload.UpdateUser)

	// 检查页面权限是否有变化
	pagesChanged, oldPages := ComparePages(user.UserName, payload.PagesId)
	_, username, _ := GetCurrentUser()
	userInfo := SysUser{
		UserId:        user.UserId,
		CreatedBy:     user.CreatedBy,
		Password:      payload.UpdateUser.Password,
		IsEnabled:     payload.UpdateUser.IsEnabled,
		CreatedTime:   user.CreatedTime,
		UpdatedTime:   time.Now(),
		UserName:      payload.UpdateUser.UserName,
		NickName:      payload.UpdateUser.NickName,
		RoleId:        payload.UpdateUser.RoleId,
		Email:         payload.UpdateUser.Email,
		Phone:         payload.UpdateUser.Phone,
		InitialPageId: payload.UpdateUser.InitialPageId,
		Remark:        payload.UpdateUser.Remark,
		UpdatedBy:     payload.UpdateUser.UpdatedBy,
		UpdatedByName: username,
	}

	// 密码是否更新
	pswUpdated := false
	if payload.UpdateUser.Password != user.Password {
		pswUpdated = true
		updatedFields["Password"] = "***"  // 新密码用***表示
		oldFieldValues["Password"] = "***" // 旧密码也用***表示
	}

	// 如果没有字段更新且页面权限没有变化，则不记录日志
	if len(updatedFields) == 0 && !pagesChanged && !pswUpdated {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SYS_USER, MsgBody: "ok"}
		return
	}

	err = mSrvMgr.sysUserPd.UpdateUser(&userInfo, pswUpdated)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SYS_USER, MsgBody: "fail,update user failed"}
		return
	}

	// 检查页面权限是否有变化
	if pagesChanged {
		// 清除用户所有页面权限
		err = mSrvMgr.sysUserPd.ClearAllPagePermissions(userInfo.UserId)
		if err != nil {
			l.Log.Error(err)
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SYS_USER, MsgBody: "fail,update user failed"}
			return
		}

		// 更新用户页面权限
		pagesId := payload.PagesId
		err = mSrvMgr.sysUserPd.UpdateUserPageId(userInfo.UserId, pagesId)
		if err != nil {
			l.Log.Error(err)
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SYS_USER, MsgBody: "fail,update user failed"}
			return
		}

		updatedFields["PagesId"] = payload.PagesId
		oldFieldValues["PagesId"] = oldPages
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_SYS_USER, MsgBody: "ok"}

	// 只记录变更的字段
	if len(updatedFields) > 0 {
		jsonStr := SaveUpdateUserLog(user.UserName, user.NickName, updatedFields, oldFieldValues)
		LogSysOperation(MenuUserManage, SubUpdate, OpUpdateStr, jsonStr, "ok", "")
	}
}

// 禁用用户
func (p disableSysUserNotifier) Handle(mgr *SrvMgr, payload ReqEnabledSysUserId) {
	// Do something for this event
	l.Log.Debug("Handle disableSysUserNotifier called")

	userid, username, _ := GetCurrentUser()

	err := mSrvMgr.sysUserPd.DisableUser(payload.UserId, payload.IsEnabled, userid, username)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DISABLE_SYS_USER, MsgBody: "fail,disable user failed"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DISABLE_SYS_USER, MsgBody: "ok"}

	userPerm, _ := mSrvMgr.sysUserPd.GetUserInfoById(payload.UserId)
	type ReqEnabledSysUserName struct {
		Account   string
		UserName  string
		IsEnabled bool
	}
	reqInfo := ReqEnabledSysUserName{
		Account:   userPerm.UserName,
		UserName:  userPerm.NickName,
		IsEnabled: payload.IsEnabled,
	}

	jsonStr, _ := json.MarshalToString(reqInfo)
	LogSysOperation(MenuUserManage, SubUserManageEnabled, OpEnabledStr, jsonStr, "ok", "")

}

// 密码修改
func (p changePasswordNotifier) Handle(mgr *SrvMgr, payload ReqChangePassword) {
	l.Log.Debug("Handle changePasswordNotifier called")

	err := mSrvMgr.sysUserPd.UpdateUserPassword(payload.UserId, payload.NewPassword)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CHANGE_PASSWORD, MsgBody: "fail,change password failed"}
		return
	}
	LogSysOperation(MenuUserManage, SubUserManageEditPswd, OpUpdateStr, "", "ok", "")
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CHANGE_PASSWORD, MsgBody: "ok"}
}

// 登录
func (p loginNotifier) Handle(mgr *SrvMgr, payload ReqLogin) {
	// Do something for this event
	l.Log.Debug("Handle loginNotifier called")

	if payload.AutoLogin {
		userPerm, _ := mSrvMgr.sysUserPd.GetUserDetail(payload.UserName)
		SetCurrentUser(userPerm.UserID, userPerm.NickName, userPerm.RoleID)
		// 记录登录日志
		LogSysOperation(MenuSystem, SubSysLogin, OpLoginStr, "", "ok", "")

	} else {

		res, err := mSrvMgr.sysUserPd.Login(payload.UserName, payload.Password)
		if err != nil {
			l.Log.Error(err)
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_LOGIN, MsgBody: "fail,login failed"}
			return
		}
		if res {

			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_LOGIN, MsgBody: "ok"}
			userPerm, _ := mSrvMgr.sysUserPd.GetUserDetail(payload.UserName)
			SetCurrentUser(userPerm.UserID, userPerm.NickName, userPerm.RoleID)
			// 记录登录日志
			LogSysOperation(MenuSystem, SubSysLogin, OpLoginStr, "", "ok", "")
		} else {
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_LOGIN, MsgBody: "fail,login failed"}
		}

	}

}

// 登出
func (p logoutNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle logoutNotifier called")
	LogSysOperation(MenuSystem, SubSysLogout, OpLogoutStr, "", "ok", "")
	SetCurrentUser(0, "", 0)
}

// 获取所有用户列表
func (p getAllUsersNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getAllUsersNotifier called")
	users, err := mSrvMgr.sysUserPd.GetAllUsers()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_USERS, MsgBody: "fail,get all users failed"}

		return
	}
	// 转换为字符串
	usersStr, err := json.Marshal(users)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_USERS, MsgBody: "fail,get all users failed"}
		return
	}
	// 发送消息
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_USERS, MsgBody: string(usersStr)}
}

// 获取用户详细信息
func (p getUserDetailNotifier) Handle(mgr *SrvMgr, payload ReqSysUserName) {
	// Do something for this event
	l.Log.Debug("Handle getUserDetailNotifier called")
	userPerm, err := mSrvMgr.sysUserPd.GetUserDetail(payload.UserName)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_USER_DETAIL, MsgBody: "fail,get user detail failed"}
		return
	}
	// 转换为字符串
	userStr, err := json.Marshal(userPerm)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_USER_DETAIL, MsgBody: "fail,get user detail failed"}
		return
	}
	// 发送消息
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_USER_DETAIL, MsgBody: string(userStr)}
}

// 新增系统日志记录
func (p addSysLogNotifier) Handle(mgr *SrvMgr, payload ReqAddSysLog) {
	// Do something for this event
	l.Log.Debug("Handle addSysLogNotifier called")

	newLog := Syslog{
		CreateTime:    time.Now(),
		FuncName:      payload.FuncName,
		Module:        payload.Module,
		Operation:     payload.Operation,
		OperationType: payload.OperationType,
		Operator:      payload.Operator,
		RecId:         payload.RecId,
		Remarks:       payload.Remarks,
	}

	err := mSrvMgr.sysLogPd.AddSyslog(newLog)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SYS_LOG_ADD, MsgBody: "fail,add sys log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SYS_LOG_ADD, MsgBody: "ok"}
}

// 新增称重记录
func (p addScaleLogNotifier) Handle(mgr *SrvMgr, payload ReqAddScaleLog) {
	// Do something for this event
	l.Log.Debug("Handle addScaleLogNotifier called")
	newLog := ScaleWgtLog{
		Operator:   payload.Operator,
		RoleId:     payload.RoleId,
		ScaleName:  payload.ScaleName,
		Module:     payload.Module,
		ModelName:  payload.ModelName,
		Sn:         payload.Sn,
		Unit:       payload.Unit,
		Weight:     payload.Weight,
		Remarks:    payload.Remarks,
		CreateTime: time.Now(),
	}

	err := mSrvMgr.sysLogPd.AddScaleLog(newLog)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_LOG_ADD, MsgBody: "fail,add scale log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_LOG_ADD, MsgBody: "ok"}
	//校准日志已经是日志了。所以这里不需要记录操作日志。
}

// 删除系统日志记录
func (p delSysLogNotifier) Handle(mgr *SrvMgr, payload ReqDelLogs) {
	// Do something for this event
	l.Log.Debug("Handle delSysLogNotifier called")

	err := mSrvMgr.sysLogPd.DeleteSyslog(payload.RecId)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_SYS_LOG, MsgBody: "fail,delete sys log failed"}
		return
	}

	jsonStr, _ := json.Marshal(payload)
	LogSysOperation(MenuSysLog, SubLogDel, OpDeleteStr, string(jsonStr), "ok", "")
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_SYS_LOG, MsgBody: "ok"}

}

// 删除校准日志
func (p delCalLogNotifier) Handle(mgr *SrvMgr, payload ReqDelLogs) {
	// Do something for this event
	l.Log.Debug("Handle delCalLogNotifier called")

	logs, err := mSrvMgr.sysLogPd.GetCalibrationLogByID(payload.RecId)
	if err != nil {
		l.Log.Error(err)
	}

	err = mSrvMgr.sysLogPd.DeleteCalibrationLog(payload.RecId)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_CAL_LOG, MsgBody: "fail,delete calibration log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_CAL_LOG, MsgBody: "ok"}
	jsonStr := SaveDelCalLog(logs)
	LogSysOperation(MenuCalLog, SubLogDel, OpDeleteStr, string(jsonStr), "ok", "")
}

// 删除称重日志
func (p delScaleLogNotifier) Handle(mgr *SrvMgr, payload ReqDelLogs) {
	// Do something for this event
	l.Log.Debug("Handle delScaleLogNotifier called")

	logs, err := mSrvMgr.sysLogPd.GetScaleLogByID(payload.RecId)
	if err != nil {
		l.Log.Error(err)
	}
	err = mSrvMgr.sysLogPd.DeleteScaleLog(payload.RecId)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_SCALE_LOG, MsgBody: "fail,delete scale log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_SCALE_LOG, MsgBody: "ok"}
	jsonStr := SaveDelScaleLog(logs)

	LogSysOperation(MenuScaleLog, SubLogDel, OpDeleteStr, string(jsonStr), "ok", "")
}

// 删除所有系统日志记录
func (p delAllSysLogNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle delAllSysLogNotifier called")
	err := mSrvMgr.sysLogPd.DeleteAllSyslog()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_ALL_SYS_LOG, MsgBody: "fail,delete all sys log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_ALL_SYS_LOG, MsgBody: "ok"}
	LogSysOperation(MenuSysLog, SubLogClear, OpClearStr, "", "ok", "")
}

// 删除所有校准日志记录
func (p delAllCalLogNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle delAllCalLogNotifier called")
	err := mSrvMgr.sysLogPd.DeleteAllCalibrationLog()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_ALL_CAL_LOG, MsgBody: "fail,delete all calibration log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_ALL_CAL_LOG, MsgBody: "ok"}
	LogSysOperation(MenuCalLog, SubLogClear, OpClearStr, "", "ok", "")
}

// 删除所有称重日志记录
func (p delAllScaleLogNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle delAllScaleLogNotifier called")
	err := mSrvMgr.sysLogPd.DeleteAllScaleLog()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_ALL_SCALE_LOG, MsgBody: "fail,delete all scale log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DEL_ALL_SCALE_LOG, MsgBody: "ok"}
	LogSysOperation(MenuScaleLog, SubLogClear, OpClearStr, "", "ok", "")
}

// 获取系统日志记录
func (p getSysLogNotifier) Handle(mgr *SrvMgr, payload ReqGetLog) {
	// Do something for this event
	l.Log.Debug("Handle getSysLogNotifier called")
	syslogQuery := GetSearchLog(payload.Search)
	logs, total, err := mSrvMgr.sysLogPd.GetSyslog(payload.Page, payload.PageSize, payload.FieldName, payload.Direction, syslogQuery)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SYS_LOG_LIST, MsgBody: "fail,get sys log failed"}
		return
	}

	SyslogList := SyslogList{Total: int(total), Logs: logs}
	jsonStr, _ := json.Marshal(SyslogList)

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SYS_LOG_LIST, MsgBody: string(jsonStr)}
}

// 获取校准日志记录
func (p getCalLogNotifier) Handle(mgr *SrvMgr, payload ReqGetLog) {
	// Do something for this event
	l.Log.Debug("Handle getCalLogNotifier called")
	syslogQuery := GetSearchLog(payload.Search)
	logs, total, err := mSrvMgr.sysLogPd.GetCalibrationLog(payload.Page, payload.PageSize, payload.FieldName, payload.Direction, syslogQuery)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_CAL_LOG_LIST, MsgBody: "fail,get calibration log failed"}
		return
	}

	CalibrationLogList := CalibrationLogList{Total: int(total), Logs: logs}
	jsonStr, _ := json.Marshal(CalibrationLogList)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_CAL_LOG_LIST, MsgBody: string(jsonStr)}

}

// 获取称重日志记录
func (p getScaleLogNotifier) Handle(mgr *SrvMgr, payload ReqGetLog) {
	// Do something for this event
	l.Log.Debug("Handle getScaleLogNotifier called")
	syslogQuery := GetSearchLog(payload.Search)

	logs, total, err := mSrvMgr.sysLogPd.GetScaleLog(payload.Page, payload.PageSize, payload.FieldName, payload.Direction, syslogQuery)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SCALE_LOG_LIST, MsgBody: "fail,get scale log failed"}
		return
	}

	ScaleLogList := ScaleLogList{Total: int(total), Logs: logs}
	jsonStr, _ := json.Marshal(ScaleLogList)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SCALE_LOG_LIST, MsgBody: string(jsonStr)}
}

// 导出系统日志
func (p exportSysLogNotifier) Handle(mgr *SrvMgr, payload ReqExportLog) {
	// Do something for this event
	l.Log.Debug("Handle exportSysLogNotifier called")
	syslogQuery := GetSearchLog(payload.Search)

	logs, err := mSrvMgr.sysLogPd.ExportSyslog(payload.FieldName, payload.Direction, syslogQuery)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SYS_LOG, MsgBody: "fail,export sys log failed"}
		return
	}
	trans := payload.Translation
	fmt.Print(len(logs))

	// 导出日志到文件
	err = ExportSysLogsToFile(logs, payload.FilePath, trans, payload.Headers)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SYS_LOG, MsgBody: "fail,export sys log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SYS_LOG, MsgBody: "ok," + payload.FilePath}

	type Total struct {
		Total int
	}
	jsonStr, _ := json.Marshal(Total{Total: len(logs)})
	LogSysOperation(MenuSysLog, SubLogExport, OpExportStr, string(jsonStr), "ok", "")
}

// 导出校准日志
func (p exportCalLogNotifier) Handle(mgr *SrvMgr, payload ReqExportLog) {
	// Do something for this event
	l.Log.Debug("Handle exportCalLogNotifier called")
	syslogQuery := GetSearchLog(payload.Search)
	logs, err := mSrvMgr.sysLogPd.ExportCalibrationLog(payload.FieldName, payload.Direction, syslogQuery)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_CAL_LOG, MsgBody: "fail,export calibration log failed"}
		return
	}
	fmt.Print(len(logs))

	// 导出日志到文件
	err = ExportCalLogsToFile(logs, payload.FilePath, payload.Translation, payload.Headers)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_CAL_LOG, MsgBody: "fail,export calibration log failed"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_CAL_LOG, MsgBody: "ok," + payload.FilePath}
}

// 导出称重日志
func (p exportScaleLogNotifier) Handle(mgr *SrvMgr, payload ReqExportLog) {
	// Do something for this event
	l.Log.Debug("Handle exportScaleLogNotifier called")
	syslogQuery := GetSearchLog(payload.Search)
	logs, err := mSrvMgr.sysLogPd.ExportScaleLog(payload.FieldName, payload.Direction, syslogQuery)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SCALE_LOG, MsgBody: "fail,export scale log failed"}
		return
	}
	fmt.Print(len(logs))
	trans := payload.Translation
	fmt.Print(len(logs))

	// 导出日志到文件
	err = ExportWgtLogsToFile(logs, payload.FilePath, trans, payload.Headers)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SCALE_LOG, MsgBody: "fail,export sys log failed"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SCALE_LOG, MsgBody: "ok," + payload.FilePath}

	type Total struct {
		Total int
	}
	jsonStr, _ := json.Marshal(Total{Total: len(logs)})
	LogSysOperation(MenuScaleLog, SubLogExport, OpExportStr, string(jsonStr), "ok", "")

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_EXPORT_SCALE_LOG, MsgBody: "ok"}
}

// 新增标定记录
func (p addCalLogNotifier) Handle(mgr *SrvMgr, payload CalibrationLog) {
	l.Log.Debug("Handle addCalLogNotifier called")

	for _, v := range mgr.scales {
		if v.Conn.ScaleId == int64(payload.ScaleId) {
			payload.ScaleName = v.Conn.ScaleName
			payload.Sn = v.Conn.ScaleSn
			payload.ModelName = v.Conn.ScaleModel
			break
		}
	}
	LogCalLogOperation(payload)

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_CAL_LOG_ADD, MsgBody: "ok"}

}

// 新增上传配方称重记录服务器
func (p editUploadFmaServerNotifier) Handle(mgr *SrvMgr, payload UploadServerInfo) {
	l.Log.Debug("Handle editUploadServerNotifier called")
	if payload.RecId == 0 {
		srvInfo := UploadServerInfo{
			Ip:        payload.Ip,
			ShareName: payload.ShareName,
			Username:  payload.Username,
			Password:  payload.Password,
			Enable:    payload.Enable,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: payload.UpdatedBy,
			UpdatedBy: payload.UpdatedBy,
		}
		mgr.formulaPd.CreateUploadServerInfo(srvInfo)
	} else {
		srvInfo := UploadServerInfo{
			Ip:        payload.Ip,
			ShareName: payload.ShareName,
			Username:  payload.Username,
			Password:  payload.Password,
			Enable:    payload.Enable,
			UpdatedAt: time.Now(),
			UpdatedBy: payload.UpdatedBy,
		}

		mgr.formulaPd.UpdateUploadServerInfo(srvInfo)
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPLOAD_SERVER_EDIT, MsgBody: "ok"}
}

// 获取上传配方称重记录服务器
func (p getUploadFmaServerNotifier) Handle(mgr *SrvMgr) {
	l.Log.Debug("Handle getUploadServerNotifier called")

	serverInfo, err := mgr.formulaPd.GetUploadServerInfo()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPLOAD_SERVER_GET, MsgBody: "fail"}
		return
	}

	typesStr, err := json.MarshalToString(serverInfo)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPLOAD_SERVER_GET, MsgBody: "fail"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPLOAD_SERVER_GET, MsgBody: typesStr}
}

// 获取所有铅封日志记录
func (p getAllSealLogNotifier) Handle(mgr *SrvMgr, payload GetSealLogReq) {
	l.Log.Debug("Handle getAllSealLogNotifier called")

	sealLogs, err := mgr.sysLogPd.GetAllSealLog(payload.Model, payload.Sn)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_SEAL_LOG, MsgBody: "fail"}
		return
	}
	typesStr, err := json.MarshalToString(sealLogs)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_SEAL_LOG, MsgBody: "fail"}
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_ALL_SEAL_LOG, MsgBody: typesStr}
	return
}

// 万能钥匙解除铅封
func (p unsealByMasterKeyNotifier) Handle(mgr *SrvMgr, payload string) {
	l.Log.Debug("Handle unsealByMasterKeyNotifier called")

	privateKey := `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCiaVmdfO6DCcVh
Gx+qYJ3v0xavdYAPbQ+LSV0oQLawnGYN4SzjfzoufMS5V3QDUmXNEkpMVOI/WZsT
SfqAyr4EwK9zVE0JL/Q1jNbzw4H5iq0ryyURJqVXpVn+mlERvv3zphvBVvQMfnzT
G6WwD3ijz3gAkZ6LziUay+KNIrA4nFXC7IQq2SHKGu/tlDYU5iab0xN/mfnVII8A
MgxLyYdh1RI4I6s6h6AayPx8+/uRWQjKJZxRgUZl0EL6BZJWdHrIYQaqIGcXnyM/
BD+c/pppb0XQc38sQ4H1xrpqQ2e4Uum51j76918IkFCk2wI9IVfjFyUCtKBzTHC2
MQWaNqV5AgMBAAECggEAfvLwLI0J9n19vjCwaMIK0fpjAhVLW0N5YfufiKZE5vnp
P7IiH1VEii/WqbU1Jp+SmWBRmSbEjpYhBEvQNjnDm/1tZy2e5a6JKg6Dupi4kPEX
+WJZ//UASukh1kSTV9a9tGTDzzWDn/yC35T9xwfg2dKCz5cDoe4pzK9Pz9gsfKJf
xQeay1HMbWaqWaoNs4bkqsjQoCWYxvi1NMVQ8tw2w3rozW4ibxY9TfFHBKoVsFCc
Z7QTEvULzX6OBpNT6700HIklmXFwNEzDf6LS5nE1stOuk/A4VBMf4q4jKYoGiuYn
mFg2DF7NSLnNMMhUKhsVg60HuzFoy2jTxPFhfH5KTQKBgQDDB75+WhSmatupkFXb
GzgE36qMYQMjGz8HHI7Q+KEIrzvZ3mYEyw9rO+A14/no5mZhDiGhFSSYqOq9oaqr
I/1KakzAZ1ohWfw2u8ejGxORY01bk/K+tYxMU+HN8QMJbArzuDdRdSScFtdF6TFs
TnezO13FrP9Ag9Zhr7Yyj6WygwKBgQDVLyG65n+CcoQ8F87kb1cWMpvCju8RT/bj
f54lVJ8hFWNXlRb/qtHkbBjPhfUnKPMua/2H6VSIvqOMTknjvRY8XhKqJSzHtqpO
VYTuN9Od7N0owqDS1SN7i+fc6yAdbXSCj0l9rH+m7SZZhgnCndWEVqJXrsacWC2G
qxorD3wXUwKBgA6R5YlK8X/9O6vPPJrBzc2PaA5UsQdOYccGOyUhbeZYMQB1vOle
wiggsP9VqLXdgIh/pcOC8Nj2xZKlITrn1WRZzKITFoinUFBGdwOYYj3aTU0qIFhe
97w8CAJ6nt91UtwiRv+u4K1Ih4yRfz+4HPkm1jqOUgNf1gQ2PEZKtPZBAoGAKYyP
EWM9NMpm9WNagnEk0wG4E9pRw9kG8F3+D56HiSYm/3niSqAbWl6rEz8zgZdclg6c
EjIqtKAbNgxIIGfI/qkDEEBAkwgJ90x5pQgiaWQx0nDkcVLzIHArF4aH8tRTYeLV
WvYUxw7va4FRQ6oJZEqSR26b7PrOnLGaXwwcjlsCgYAMzOWUpsmHgdwIfvLsVmP2
8IFhFkm01e13FUzRyfRO/8SVzg1H764GDawN8XhusKYagobpBKtEmN3DXixqKw6r
sc27fFJGwCCLfWHk/pfzyZqJbzdKNNa0KfsbwCkXNLmGc64EpkJtlys0YHmNzREE
pTolgx1VELrkotW1tuLGJA==
-----END PRIVATE KEY-----`

	cipherText := payload
	// 尝试用OAEP模式解密（对应C#默认的true）
	decrypted, err := RSADecrypt(privateKey, cipherText, true)
	if err != nil {
		fmt.Printf("OAEP解密失败: %v\n", err)
		// 如果OAEP失败，尝试PKCS1模式
		decrypted, err = RSADecrypt(privateKey, cipherText, false)
		if err != nil {
			fmt.Printf("PKCS1解密失败: %v\n", err)
			mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UNSEAL_BY_MASTER_KEY, MsgBody: "fail,invalid ciphertext"}
			return
		}

	}
	println(decrypted)

	MasterKeyResp := MasterKeyResp{}
	err = json.Unmarshal([]byte(decrypted), &MasterKeyResp)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UNSEAL_BY_MASTER_KEY, MsgBody: "fail,invalid json"}
		return
	}

	//获取时间戳，比较两个时间戳
	currentTimestamp := time.Now().Unix()

	// 转换时间戳字符串为int64
	timestamp, err := strconv.ParseInt(MasterKeyResp.TimeStamp, 10, 64)
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UNSEAL_BY_MASTER_KEY, MsgBody: "fail,invalid timestamp"}
		return
	}

	// 检查时间戳是否在48小时以内
	if !IsWithin48Hours(currentTimestamp, timestamp) {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UNSEAL_BY_MASTER_KEY, MsgBody: "fail,timestamp expired"}
		return
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UNSEAL_BY_MASTER_KEY, MsgBody: "ok"}
	return
}

type MasterKeyResp struct {
	Code      string `json:"Code"`
	TimeStamp string `json:"TimeStamp"`
}

// IsWithin48Hours 判断两个10位时间戳是否在48小时以内
func IsWithin48Hours(timestamp1, timestamp2 int64) bool {
	// 计算时间差的绝对值（秒）
	diff := timestamp1 - timestamp2
	if diff < 0 {
		diff = -diff
	}

	// 48小时 = 48 * 3600 秒
	const hours48InSeconds = 48 * 3600

	// 如果时间差小于48小时的秒数，说明在48小时以内
	return diff < hours48InSeconds
}

func (p openOutputPortNotifier) Handle(mgr *SrvMgr, payload ReqPortInfo) {
	l.Log.Debug("Handle openOutputPortNotifier called")

	portStatus, err := mgr.GetSerialStatus()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_open_output_port", MsgBody: "fail,get serial status failed"}
		mgr.serialPort.Restart()
		return
	}

	openFlag, ok := portStatus["isOpen"].(bool)
	if !ok {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_open_output_port", MsgBody: "fail,get serial status failed"}
		mgr.serialPort.Restart()
		return
	}

	if !openFlag {
		l.Log.Error("serial port is not open")
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_open_output_port", MsgBody: "fail,serial status is not open"}
		mgr.serialPort.Restart()
		return
	}

	status := byte(0x00)
	if payload.Status {
		status = byte(0xFF)
	}

	port := 0
	if payload.PortId > 0 {
		port = payload.PortId - 1
	}

	addr := byte(0x01)                    // 从站地址
	funcCode := byte(0x05)                // 功能码：写线圈
	startAddr := []byte{0x00, byte(port)} // 起始地址 9
	quantity := []byte{status, 0x00}      // 读取1个线圈

	mgr.SendModbusCommand([]byte{addr, funcCode}, startAddr, quantity)
}

func (p readOutputPortNotifier) Handle(mgr *SrvMgr, payload ReqPortInfo) {
	l.Log.Debug("Handle readOutputPortNotifier called")
	// mgr.WriteSerialHex("0101000900012DC8")

	portStatus, err := mgr.GetSerialStatus()
	if err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_read_output_port", MsgBody: "fail,get serial status failed"}
		return
	}

	openFlag, ok := portStatus["isOpen"].(bool)
	if !ok {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_read_output_port", MsgBody: "fail,get serial status failed"}
	}

	if !openFlag {
		l.Log.Error("serial port is not open")
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: "resp_read_output_port", MsgBody: "fail,serial status is not open"}
		return
	}

	port := 0
	if payload.PortId > 0 {
		port = payload.PortId - 1
	}

	addr := byte(0x01)                    // 从站地址
	funcCode := byte(0x01)                // 功能码：读线圈
	startAddr := []byte{0x00, byte(port)} // 起始地址 9
	quantity := []byte{0x00, 0x01}        // 读取1个线圈

	mgr.SendModbusCommand([]byte{addr, funcCode}, startAddr, quantity)
}

// 获取输出端口设置
func (p getOutputPortNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getOutputPortNotifier called")
	formulaRecProvider := mgr.formulaPd
	rec, _ := formulaRecProvider.GetSetOutput()

	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(rec); err != nil {
		l.Log.Error(err)

	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_OUTPUT_PORT, MsgBody: typesStr}
}

// 更新输出端口设置
func (p updateOutputPortNotifier) Handle(mgr *SrvMgr, payload []ReqUpdateOutputPort) {
	// Do something for this event
	l.Log.Debug("Handle updateOutputPortNotifier called")

	portList := payload

	setPortList := []SetOutputPort{}

	for i := 0; i < len(portList); i++ {
		setPortList = append(setPortList, SetOutputPort{
			Port:      portList[i].Port,
			Status:    portList[i].Status,
			StartTime: portList[i].StartTime,
			EndValue:  portList[i].EndValue,
			Remark:    portList[i].Remark,
		})
	}

	if err := mgr.formulaPd.UpdateSetOutput(setPortList); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_OUTPUT_PORT, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_OUTPUT_PORT, MsgBody: "ok"}
	//TODO: 增加日志记录
}

// 获取输入端口设置
func (p getInputPortNotifier) Handle(mgr *SrvMgr) {
	// Do something for this event
	l.Log.Debug("Handle getInputPortNotifier called")
	formulaRecProvider := mgr.formulaPd
	rec, _ := formulaRecProvider.GetSetInput()

	var typesStr string
	var err error
	if typesStr, err = json.MarshalToString(rec); err != nil {
		l.Log.Error(err)

	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_INPUT_PORT, MsgBody: typesStr}
}

// 更新输入端口设置
func (p updateInputPortNotifier) Handle(mgr *SrvMgr, payload []ReqUpdateInputPort) {
	// Do something for this event
	l.Log.Debug("Handle updateInputPortNotifier called")
	portList := payload
	setPortList := []SetInputPort{}

	for i := 0; i < len(portList); i++ {
		setPortList = append(setPortList, SetInputPort{
			Port: portList[i].Port,
			Btn:  portList[i].Btn,
		})
	}

	if err := mgr.formulaPd.UpdateSetInput(setPortList); err != nil {
		l.Log.Error(err)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_INPUT_PORT, MsgBody: err.Error()}
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_UPDATE_INPUT_PORT, MsgBody: "ok"}
	//TODO: 增加日志记录
}
