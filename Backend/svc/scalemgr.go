package svc

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"tmaxsrv/comm"
	"tmaxsrv/log"
)

var nextScaleId int64 = 2 // this scale id will be incremented as new scale is added, 0 is reserved for not used

type ScaleMgr struct {
	mu                sync.Mutex
	srvMgr            *SrvMgr
	scales            map[int64]*Scale // map with scale id
	connPb            *ScaleConnProvider
	recPb             *ScaleRecProvider
	infoPb            *ScaleInfosProvider
	pluFilePb         *PluRecProvider
	recCheckWeigherPb *ScaleRecCheckWeigherProvider
	recTakeInPb       *ScaleRecTakeInProvider
	recTakeOutPb      *ScaleRecTakeOutProvider
	medias            []*ScaleConnMedia // scale connections meida
	detailPb          *DetailRecProvider
	bluetoothMgr      *BluetoothManager
}

func NewScaleMgr() *ScaleMgr {
	connPb := NewScaleConnProvider()
	recPb := NewScaleRecProvider()

	infoPb := NewScaleInfosProvider()
	pluFilePb := NewPluRecProvider()
	recCheckWeigherPb := NewScaleRecCheckWeigherProvider()
	recTakeInPb := NewScaleRecTakeInProvider()
	recTakeOutPb := NewScaleRecTakeOutProvider()
	scales := make(map[int64]*Scale)

	detailPb := NewDetailRecProvider()

	bluetoothMgr := GetBluetoothManager()
	bluetoothMgr.EnableAdapter()

	return &ScaleMgr{connPb: connPb, recPb: recPb, infoPb: infoPb, pluFilePb: pluFilePb, recCheckWeigherPb: recCheckWeigherPb, recTakeInPb: recTakeInPb, recTakeOutPb: recTakeOutPb, scales: scales, detailPb: detailPb, bluetoothMgr: bluetoothMgr}
}

func (s *ScaleMgr) SetSrvMsg(srvMgr *SrvMgr) {
	s.srvMgr = srvMgr
}

func init() {
	createNotifier := portListedNotifier{}
	portsListed.Register(createNotifier)

	createBtListNotifier := btListedNotifier{}
	btListed.Register(createBtListNotifier)

	createScaleListNotifier := scaleListedNotifier{}
	scalesListed.Register(createScaleListNotifier)

	createScaleListSrvNotifier := scaleListedNotifierSrv{}
	scalesListedSrv.Register(createScaleListSrvNotifier)

	createAddScaleNotifier := addScaleNotifier{}
	scaleAdded.Register(createAddScaleNotifier)

	createDelScaleNotifier := delScaleNotifier{}
	scaleDeleted.Register(createDelScaleNotifier)

	createModifyScaleNotifier := modifyScaleNotifier{}
	scaleModified.Register(createModifyScaleNotifier)

	createModifyScaleNameNotifier := modifyScaleNameNotifier{}
	scaleNameModified.Register(createModifyScaleNameNotifier)

	createProductListNotifier := productListedNotifier{}
	productsListed.Register(createProductListNotifier)

	createCheckPluExistNotifier := checkPluExistNotifier{}
	checkPluExist.Register(createCheckPluExistNotifier)

	//分页获取PLU列表
	createPluByPageListNotifier := pluByPageListedNotifier{}
	pluByPageListed.Register(createPluByPageListNotifier)

	createProductClearedNotifier := productClearedNotifier{}
	productCleared.Register(createProductClearedNotifier)

	createExportProductNotifier := exportProductNotifier{}
	exportProduct.Register(createExportProductNotifier)

	createPluSettingNotifier := pluSettingNotifier{}
	pluSetting.Register(createPluSettingNotifier)

	createGetPluSettingNotifier := getPluSettingNotifier{}
	getPluSetting.Register(createGetPluSettingNotifier)

	createExportPluToFileNotifier := exportPluToFileNotifier{}
	exportPluToFile.Register(createExportPluToFileNotifier)

	createAddProductNotifier := addProductNotifier{}
	productAdded.Register(createAddProductNotifier)

	createAddOneProductNotifier := addOneProductNotifier{}
	productAddedOne.Register(createAddOneProductNotifier)

	createDelProductNotifier := delProductNotifier{}
	productDeleted.Register(createDelProductNotifier)

	createDelAllProductNotifier := delAllProductNotifier{}
	productDeletedAll.Register(createDelAllProductNotifier)

	createupdateEnabledPluNotifier := updateEnabledPluNotifier{}
	updateEnabledPlu.Register(createupdateEnabledPluNotifier)

	createModifyProductNotifier := modifyProductNotifier{}
	productModified.Register(createModifyProductNotifier)

	createGetLastProductRecNotifier := getLastProductRecNotifier{}
	getLastProductRec.Register(createGetLastProductRecNotifier)

	createUserListNotifier := userListedNotifier{}
	usersListed.Register(createUserListNotifier)

	createAddUserNotifier := addUserNotifier{}
	userAdded.Register(createAddUserNotifier)

	createDelUserNotifier := delUserNotifier{}
	userDeleted.Register(createDelUserNotifier)

	createModifyUserNotifier := modifyUserNotifier{}
	userModified.Register(createModifyUserNotifier)

	createDetailListNotifier := detailListedNotifier{}
	detailListed.Register(createDetailListNotifier)

	createScaleSrvListNotifier := scaleSrvListNotifier{}
	scaleSrvList.Register(createScaleSrvListNotifier)

	// Modbus
	createGetModbusSrvNotifier := getModbusServicesNotifier{}
	getModbusServices.Register(createGetModbusSrvNotifier)

	createAddModbusSrvNotifier := addModbusServiceNotifier{}
	addModbusService.Register(createAddModbusSrvNotifier)

	createEditModbusSrvNotifier := editModbusServiceNotifier{}
	editModbusService.Register(createEditModbusSrvNotifier)

	createDelModbusSrvNotifier := delModbusServiceNotifier{}
	delModbusService.Register(createDelModbusSrvNotifier)

	createsetScaleSrvValNotifier := setScaleSrvValNotifier{}
	setScaleSrvVal.Register(createsetScaleSrvValNotifier)

	createWifiPwdListNotifier := wifiPwdListedNotifier{}
	wifiListed.Register(createWifiPwdListNotifier)

	createAddWifiPwdNotifier := addWifiPwdNotifier{}
	wifiAdded.Register(createAddWifiPwdNotifier)

	createsendToSrv1Notifier := sendToSrv1Notifier{}
	sendToSrv1.Register(createsendToSrv1Notifier)

	createsendToUiNotifier := sendToUiNotifier{}
	sendToUi.Register(createsendToUiNotifier)

	createdoServiceActionNotifier := doServiceActionNotifier{}
	doServiceAction.Register(createdoServiceActionNotifier)

	//配方秤
	createAddRewTypeNotifier := addRawTypeNotifier{}
	rawTypeAdded.Register(createAddRewTypeNotifier)

	createAddFormulaTypeNotifier := addFormulaTypeNotifier{}
	formulaTypeAdded.Register(createAddFormulaTypeNotifier)

	createDelRawTypeNotifier := delRawTypeUnusedNotifier{}
	rawTypeUnusedDeleted.Register(createDelRawTypeNotifier)

	createDelFmaTypeNotifier := delFmaTypeUnusedNotifier{}
	fmaTypeUnusedDeleted.Register(createDelFmaTypeNotifier)

	createGetRawTypeListNotifier := getRawTypeListNotifier{}
	rawTypeListed.Register(createGetRawTypeListNotifier)

	creategetFormulaTypeListNotifier := getFormulaTypeListNotifier{}
	formulaTypeListed.Register(creategetFormulaTypeListNotifier)

	creategetAddRawDataNotifier := rawDataAddedNotifier{}
	rawDataAdded.Register(creategetAddRawDataNotifier)

	createGetRawOutputByFmaIdNotifier := getRawOutputByFmaIdNotifier{}
	getRawOutputByFmaId.Register(createGetRawOutputByFmaIdNotifier)

	creategetEditRawTypeNotifier := rawTypeEditedNotifier{}
	rawTypeModified.Register(creategetEditRawTypeNotifier)

	creategetDelRawTypeNotifier := rawTypeDeletedNotifier{}
	rawTypeDeleted.Register(creategetDelRawTypeNotifier)

	creategetEditFmaTypeNotifier := fmaTypeEditedNotifier{}
	fmaTypeModified.Register(creategetEditFmaTypeNotifier)

	creategetDelFmaTypeNotifier := fmaTypeDeletedNotifier{}
	fmaTypeDeleted.Register(creategetDelFmaTypeNotifier)

	//导入原料数据列表
	createRawListImportedNotifier := rawListImportedNotifier{}
	rawListImported.Register(createRawListImportedNotifier)

	//导入配方数据列表
	createFormulaListImportedNotifier := formulaListImportedNotifier{}
	formulaListImported.Register(createFormulaListImportedNotifier)

	createGetrawDataListedNotifier := rawDataListedNotifier{}
	rawDataListed.Register(createGetrawDataListedNotifier)

	createRawDataEditedNotifier := rawDataEditedNotifier{}
	rawDataEdited.Register(createRawDataEditedNotifier)

	createRawDataDeletedNotifier := rawDataDeletedNotifier{}
	rawDataDeleted.Register(createRawDataDeletedNotifier)

	createFormulaRecAddedNotifier := addFormulaRecNotifier{}
	formulaDataAdded.Register(createFormulaRecAddedNotifier)

	createFormulaRecUpdateNotifier := editFormulaRecNotifier{}
	formulaDataEdited.Register(createFormulaRecUpdateNotifier)

	createFormulaRecListNotifier := getFormulaListNotifier{}
	formulaRecList.Register(createFormulaRecListNotifier)

	//获取配方数据by 条码
	createGetFormulaByBarcodeNotifier := getFormulaByBarcodeNotifier{}
	getFormulaByBarcode.Register(createGetFormulaByBarcodeNotifier)

	//检查配方ID和条码是否匹配
	createCheckFmaIdAndBarcodeNotifier := checkFmaIdAndBarcodeNotifier{}
	checkFmaIdAndBarcode.Register(createCheckFmaIdAndBarcodeNotifier)

	creategetFormulaDataNotifier := getFormulaDataNotifier{}
	formulaData.Register(creategetFormulaDataNotifier)

	createGetRawDataNotifier := getRawDataNotifier{}
	rawDataGetted.Register(createGetRawDataNotifier)

	createFormulaWgtRecAddedNotifier := addFormulaWgtRecNotifier{}
	formulaWgtRecAdded.Register(createFormulaWgtRecAddedNotifier)

	createFormulaWgtRecListNotifier := getFormulaWgtRecListNotifier{}
	formulaWgtRecList.Register(createFormulaWgtRecListNotifier)

	createGetFormulaWgtRecByPageNotifier := getFormulaWgtRecByPageNotifier{}
	formulaWgtRecByPage.Register(createGetFormulaWgtRecByPageNotifier)

	createGetAllFormulaRecForExportNotifier := getAllFormulaRecForExportNotifier{}
	getAllFormulaRecForExport.Register(createGetAllFormulaRecForExportNotifier)

	createDraftFormulaRecByPageNotifier := getDraftFormulaRecByPageNotifier{}
	draftFormulaRecByPage.Register(createDraftFormulaRecByPageNotifier)

	createGetAllDraftFormulaRecForExportNotifier := getAllDraftFormulaRecForExportNotifier{}
	getAllDraftFormulaRecForExport.Register(createGetAllDraftFormulaRecForExportNotifier)

	createRawMaterialByPageNotifier := getRawMaterialByPageNotifier{}
	rawMaterialByPage.Register(createRawMaterialByPageNotifier)

	createRawMaterialDictNotifier := getRawMaterialDictNotifier{}
	rawMaterialDict.Register(createRawMaterialDictNotifier)

	createGetAllRawMaterialsForExportNotifier := getAllRawMaterialsForExportNotifier{}
	getAllRawMaterialsForExport.Register(createGetAllRawMaterialsForExportNotifier)

	createFormulaByPageNotifier := getFormulaByPageNotifier{}
	formulaByPage.Register(createFormulaByPageNotifier)

	createFormulaDetailsByRecIdNotifier := getFormulaDetailsByRecIdNotifier{}
	formulaDetailsByRecId.Register(createFormulaDetailsByRecIdNotifier)

	createGetAllFormulasForExportNotifier := getAllFormulasForExportNotifier{}
	getAllFormulasForExport.Register(createGetAllFormulasForExportNotifier)



	oneFormulaWgtRecListNotifier := getOneFormulaWgtRecListNotifier{}
	oneFmaWgtRecList.Register(oneFormulaWgtRecListNotifier)

	createDelFormulaWgtRecBatchNotifier := delFormulaWgtRecBatchNotifier{}
	delFormulaWgtRecBatch.Register(createDelFormulaWgtRecBatchNotifier)

	createDelAllFormulaWgtRecNotifier := delAllFormulaWgtRecNotifier{}
	delAllFormulaWgtRec.Register(createDelAllFormulaWgtRecNotifier)


	getFmaRecByOrderIdNotifier := getFmaRecByOrderIdNotifier{}
	getFmaRecByOrderId.Register(getFmaRecByOrderIdNotifier)

	createFmaDelNotifier := delFormulaNotifier{}
	formulaDeleted.Register(createFmaDelNotifier)

	createDelAllFormulaNotifier := delAllFormulaNotifier{}
	formulaDeletedAll.Register(createDelAllFormulaNotifier)

	createDelAllRawDataNotifier := delAllRawDataNotifier{}
	rawDataDeletedAll.Register(createDelAllRawDataNotifier)

	createDelAllDraftFmaWgtRecNotifier := delAllDraftFmaWgtRecNotifier{}
	formulaDataDeletedAll.Register(createDelAllDraftFmaWgtRecNotifier)

	addFlowRateNotifier := addFlowRateNotifier{}
	flowRateAdded.Register(addFlowRateNotifier)

	createFlowRateListNotifier := getFlowRateListNotifier{}
	flowRateList.Register(createFlowRateListNotifier)

	createDelWgtRecNotifier := delWgtRecNotifier{}
	delWgtRec.Register(createDelWgtRecNotifier)

	createDelWgtRecByIdNotifier := delWgtRecByIdNotifier{}
	delWgtRecById.Register(createDelWgtRecByIdNotifier)

	createGetAllWgtRecListNotifier := getAllWgtRecListNotifier{}
	getAllWgtRecList.Register(createGetAllWgtRecListNotifier)

	createGetSearchRecListNotifier := getSearchRecListNotifier{}
	getSearchRecList.Register(createGetSearchRecListNotifier)

	createAddWgtRecNotifier := addWgtRecNotifier{}
	addWgtRec.Register(createAddWgtRecNotifier)

	createExportAllRecsNotifier := exportAllRecsNotifier{}
	exportAllRecs.Register(createExportAllRecsNotifier)

	createUpdateAutoNextNotifier := updateAutoNextNotifier{}
	updateAutoNext.Register(createUpdateAutoNextNotifier)

	createGetAutoNextNotifier := getAutoNextNotifier{}
	getAutoNext.Register(createGetAutoNextNotifier)

	createGetUnstableZeroTareNotifier := getUnstableZeroTareNotifier{}
	getUnstableZeroTare.Register(createGetUnstableZeroTareNotifier)

	createUpdateUnstableZeroTareNotifier := updateUnstableZeroTareNotifier{}
	updateUnstableZeroTare.Register(createUpdateUnstableZeroTareNotifier)

	createUpdateOutputPortNotifier := updateOutputPortNotifier{}
	updateOutputPort.Register(createUpdateOutputPortNotifier)

	createGetOutputPortNotifier := getOutputPortNotifier{}
	getOutputPort.Register(createGetOutputPortNotifier)

	createGetInputPortNotifier := getInputPortNotifier{}
	getInputPort.Register(createGetInputPortNotifier)

	createUpdateInputPortNotifier := updateInputPortNotifier{}
	updateInputPort.Register(createUpdateInputPortNotifier)

	createDraftFmaWgtRecNotifier := addDraftFmaWgtRecNotifier{}
	addDraftFmaWgtRec.Register(createDraftFmaWgtRecNotifier)

	createDeleteDraftFmaWgtRecNotifier := deleteDraftFmaWgtRecNotifier{}
	deleteDraftFmaWgtRec.Register(createDeleteDraftFmaWgtRecNotifier)

	createUpdateDraftFmaWgtRecNotifier := updateDraftFmaWgtRecNotifier{}
	updateDraftFmaWgtRec.Register(createUpdateDraftFmaWgtRecNotifier)

	createGetDraftFmaWgtRecListNotifier := getDraftFmaWgtRecListNotifier{}
	getDraftFmaWgtRecList.Register(createGetDraftFmaWgtRecListNotifier)

	createAddSysUserNotifier := addSysUserNotifier{}
	addSysUser.Register(createAddSysUserNotifier)

	createDeleteSysUserNotifier := deleteSysUserNotifier{}
	deleteSysUser.Register(createDeleteSysUserNotifier)

	createUpdateSysUserNotifier := updateSysUserNotifier{}
	updateSysUser.Register(createUpdateSysUserNotifier)

	createDisableSysUserNotifier := disableSysUserNotifier{}
	disableSysUser.Register(createDisableSysUserNotifier)

	createChangePasswordNotifier := changePasswordNotifier{}
	changePassword.Register(createChangePasswordNotifier)

	createLoginNotifier := loginNotifier{}
	login.Register(createLoginNotifier)

	createLogoutNotifier := logoutNotifier{}
	logout.Register(createLogoutNotifier)

	createGetAllUsersNotifier := getAllUsersNotifier{}
	getAllUsers.Register(createGetAllUsersNotifier)

	createGetUserDetailNotifier := getUserDetailNotifier{}
	getUserDetail.Register(createGetUserDetailNotifier)

	createSysLogNotifier := addSysLogNotifier{}
	addSysLog.Register(createSysLogNotifier)

	createScaleLogNotifier := addScaleLogNotifier{}
	addScaleLog.Register(createScaleLogNotifier)

	deleteSysLogNotifier := delSysLogNotifier{}
	delMultiSysLog.Register(deleteSysLogNotifier)

	deleteCalLogNotifier := delCalLogNotifier{}
	delMultiCalLog.Register(deleteCalLogNotifier)

	deleteScaleLogNotifier := delScaleLogNotifier{}
	delMultiScaleLog.Register(deleteScaleLogNotifier)

	deleteAllSysLogNotifier := delAllSysLogNotifier{}
	delAllSysLog.Register(deleteAllSysLogNotifier)

	deleteAllCalLogNotifier := delAllCalLogNotifier{}
	delAllCalLog.Register(deleteAllCalLogNotifier)

	deleteAllScaleLogNotifier := delAllScaleLogNotifier{}
	delAllScaleLog.Register(deleteAllScaleLogNotifier)

	getAllSysLogNotifier := getSysLogNotifier{}
	getAllSysLog.Register(getAllSysLogNotifier)

	getAllCalLogNotifier := getCalLogNotifier{}
	getAllCalLog.Register(getAllCalLogNotifier)

	getAllScaleLogNotifier := getScaleLogNotifier{}
	getAllScaleLog.Register(getAllScaleLogNotifier)

	exportAllSysLogNotifier := exportSysLogNotifier{}
	exportSysLog.Register(exportAllSysLogNotifier)

	exportAllCalLogNotifier := exportCalLogNotifier{}
	exportCalLog.Register(exportAllCalLogNotifier)

	exportAllScaleLogNotifier := exportScaleLogNotifier{}
	exportScaleLog.Register(exportAllScaleLogNotifier)

	addCalRecordNotifier := addCalLogNotifier{}
	addCalRecord.Register(addCalRecordNotifier)

	createUpdateSetReportPrintNotifier := updateSetReportPrintNotifier{}
	updateSetReportPrint.Register(createUpdateSetReportPrintNotifier)

	createGetSetReportPrintNotifier := getSetReportPrintNotifier{}
	getSetReportPrint.Register(createGetSetReportPrintNotifier)

	creatEditUploadFmaServerNotifier := editUploadFmaServerNotifier{}
	editUploadFmaServer.Register(creatEditUploadFmaServerNotifier)

	createGetUploadFmaServerNotifier := getUploadFmaServerNotifier{}
	getUploadFmaServer.Register(createGetUploadFmaServerNotifier)

	createGetAllSealLogNotifier := getAllSealLogNotifier{}
	getAllSealLog.Register(createGetAllSealLogNotifier)

	creatUnsealByMasterKeyNotifier := unsealByMasterKeyNotifier{}
	unsealByMasterKey.Register(creatUnsealByMasterKeyNotifier)

	creatReadOutputPortNotifier := readOutputPortNotifier{}
	readOutputPort.Register(creatReadOutputPortNotifier)

	createOpenOutputPortNotifier := openOutputPortNotifier{}
	openOutputPort.Register(createOpenOutputPortNotifier)

}

type portListedNotifier struct{}

type btListedNotifier struct{}

type scaleListedNotifier struct{}

type scaleListedNotifierSrv struct{}

type sendToSrv1Notifier struct{}

type sendToUiNotifier struct{}

type addScaleNotifier struct{}

type delScaleNotifier struct{}

type modifyScaleNotifier struct{}

type modifyScaleNameNotifier struct{}

type productListedNotifier struct{}

type checkPluExistNotifier struct{}

type pluByPageListedNotifier struct{}

type productClearedNotifier struct{}

type exportProductNotifier struct{}

type pluSettingNotifier struct{}

type getPluSettingNotifier struct{}

type exportPluToFileNotifier struct{}

type addProductNotifier struct{}

type addOneProductNotifier struct{}

type delProductNotifier struct{}

type delAllProductNotifier struct{}

type updateEnabledPluNotifier struct{}

type modifyProductNotifier struct{}

type getLastProductRecNotifier struct{}

type userListedNotifier struct{}

type addUserNotifier struct{}

type delUserNotifier struct{}

type modifyUserNotifier struct{}

type wifiPwdListedNotifier struct{}

type addWifiPwdNotifier struct{}

type detailListedNotifier struct{}

type scaleSrvListNotifier struct{}

type setScaleSrvValNotifier struct{}

type doServiceActionNotifier struct{}

// Modbus Notifiers
type getModbusServicesNotifier struct{}
type addModbusServiceNotifier struct{}
type editModbusServiceNotifier struct{}
type delModbusServiceNotifier struct{}

type addRawTypeNotifier struct{}

type addFormulaTypeNotifier struct{}

type delRawTypeUnusedNotifier struct{}

type delFmaTypeUnusedNotifier struct{}

type getFormulaTypeListNotifier struct{}

type getRawTypeListNotifier struct{}

type rawDataAddedNotifier struct{}

type getRawOutputByFmaIdNotifier struct{}

type rawTypeEditedNotifier struct{}

type rawTypeDeletedNotifier struct{}

type fmaTypeEditedNotifier struct{}

type fmaTypeDeletedNotifier struct{}

type rawListImportedNotifier struct{}

// 导入配方数据列表
type formulaListImportedNotifier struct{}

type rawDataListedNotifier struct{}

type rawDataEditedNotifier struct{}

type rawDataDeletedNotifier struct{}

type addFormulaRecNotifier struct{}

type editFormulaRecNotifier struct{}

type getFormulaListNotifier struct{}

type getFormulaByBarcodeNotifier struct{}

// 检查配方ID和条码是否匹配
type checkFmaIdAndBarcodeNotifier struct{}

type getFormulaDataNotifier struct{}

type getRawDataNotifier struct{}

type addFormulaWgtRecNotifier struct{}

type getFormulaWgtRecListNotifier struct{}

type getFormulaWgtRecByPageNotifier struct{}

type getAllFormulaRecForExportNotifier struct{}



type getOneFormulaWgtRecListNotifier struct{}

type delFormulaWgtRecBatchNotifier struct{}

type delAllFormulaWgtRecNotifier struct{}


type getFmaRecByOrderIdNotifier struct{}

type delFormulaNotifier struct{}

type delAllFormulaNotifier struct{}

type delAllRawDataNotifier struct{}

type delAllDraftFmaWgtRecNotifier struct{}

type addFlowRateNotifier struct{}

type getFlowRateListNotifier struct{}

type delWgtRecNotifier struct{}

type delWgtRecByIdNotifier struct{}

type getAllWgtRecListNotifier struct{}

type getSearchRecListNotifier struct{}

type addWgtRecNotifier struct{}

type exportAllRecsNotifier struct{}

type getAutoNextNotifier struct{}

type getUnstableZeroTareNotifier struct{}

type updateUnstableZeroTareNotifier struct{}

type updateAutoNextNotifier struct{}

type updateOutputPortNotifier struct{}

type getOutputPortNotifier struct{}

type addDraftFmaWgtRecNotifier struct{}

type deleteDraftFmaWgtRecNotifier struct{}

type updateDraftFmaWgtRecNotifier struct{}

type getDraftFmaWgtRecListNotifier struct{}

type addSysUserNotifier struct{}

type deleteSysUserNotifier struct{}

type updateSysUserNotifier struct{}

type disableSysUserNotifier struct{}

type changePasswordNotifier struct{}

type loginNotifier struct{}

type logoutNotifier struct{}

type getAllUsersNotifier struct{}

type getUserDetailNotifier struct{}

type addSysLogNotifier struct{}

type addScaleLogNotifier struct{}

type delSysLogNotifier struct{}

type delCalLogNotifier struct{}

type delScaleLogNotifier struct{}

type delAllSysLogNotifier struct{}

type delAllCalLogNotifier struct{}

type delAllScaleLogNotifier struct{}

type getSysLogNotifier struct{}

type getCalLogNotifier struct{}

type getScaleLogNotifier struct{}

type exportSysLogNotifier struct{}

type exportCalLogNotifier struct{}

type exportScaleLogNotifier struct{}

type addCalLogNotifier struct{}

type updateSetReportPrintNotifier struct{}

type getSetReportPrintNotifier struct{}

type editUploadFmaServerNotifier struct{}

type getUploadFmaServerNotifier struct{}

type getAllSealLogNotifier struct{}

type unsealByMasterKeyNotifier struct{}

type readOutputPortNotifier struct{}

type openOutputPortNotifier struct{}

type getInputPortNotifier struct{}

type updateInputPortNotifier struct{}

func (p portListedNotifier) Handle() {
	// Do something for this event
	log.Log.Debug("Handle portListedNotifier called")
	// Do something with this event
	ports, _ := getPortsList()
	var portsStr string
	var err error
	if portsStr, err = json.MarshalToString(ports); err != nil {
		// fmt.Printf("%v\n", err)
		// TODO: error handling
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_PORTS_LIST, MsgBody: portsStr}
}

func (p btListedNotifier) Handle() {
	// Do something for this event
	log.Log.Debug("Handle btListedNotifier called")
	// Do something with this event
	btList, _ := GetBtList()
	var btListStr string
	var err error
	if btListStr, err = json.MarshalToString(btList); err != nil {
		fmt.Printf("%v\n", err)
		// TODO: error handling
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_BT_LIST, MsgBody: btListStr}
}

func (p scaleListedNotifier) Handle(scaleMgr *ScaleMgr) {
	// Do something for this event
	log.Log.Debug("Handle scaleListedNotifier called")
	// Do something with this event
	// scaleMedias, _ := NewScaleConnProvider().GetScaleConnsList()
	scaleMedias := scaleMgr.medias
	var scalesStr string
	var err error
	if scalesStr, err = json.MarshalToString(scaleMedias); err != nil {
		log.Log.Errorf("%v\n", err)
		// TODO: error handling
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALES_LIST, MsgBody: scalesStr}
}

func (p scaleListedNotifierSrv) Handle(scaleMgr *ScaleMgr, scaleId int64) {
	// Do something for this event
	log.Log.Debug("Handle scaleListedNotifier called")
	// Do something with this event
	// scaleMedias, _ := NewScaleConnProvider().GetScaleConnsList()
	scaleMedias := scaleMgr.medias
	var scalesStr string
	var err error
	if scalesStr, err = json.MarshalToString(scaleMedias); err != nil {
		log.Log.Errorf("%v\n", err)
		// TODO: error handling
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsgSrv <- &SrvMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALES_LIST, MsgBody: scalesStr, ScaleId: scaleId}
}

func (p sendToSrv1Notifier) Handle(mSrvMgr *SrvMgr, jsonStr string) {
	// Do something for this event
	log.Log.Debug("Handle sendToSrv1Notifier called")
	// Do something with this event
	mSrvMgr.recvScaleMgrMsgSrv <- &SrvMgrRespMsg{MsgType: ScaleMgrRespMsgType(REQ_SEND_TO_SRV1), MsgBody: jsonStr, ScaleId: 999999999}
}

func (p sendToUiNotifier) Handle(mSrvMgr *SrvMgr, jsonStr string) {
	// Do something for this event
	log.Log.Debug("Handle sendToUiNotifier called")
	// Do something with this event

	var msg ScaleMgrRespMsg

	if err := json.UnmarshalFromString(jsonStr, &msg); err != nil {
		log.Log.Error(err)
	}

	mSrvMgr.recvScaleMgrMsg <- &msg
}

func (p addScaleNotifier) Handle(payload ReqAddScale) {
	// Do something for this event
	log.Log.Debug("Handle addScaleNotifier called")
	var scalesStr string
	var err error

	if err := mSrvMgr.scaleMgr.AddScale(payload); err != nil {
		log.Log.Errorf("%v\n", err)
		// return NACK to requester
		scalesStr = err.Error()
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_ADD, MsgBody: scalesStr}
		return
	}
	// TODO: check if this connection is already existed
	scaleConns, _ := NewScaleConnProvider().GetScaleConnsList()
	if scalesStr, err = json.MarshalToString(scaleConns); err != nil {
		log.Log.Errorf("%v\n", err)
		// TODO: error handling
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALES_LIST, MsgBody: scalesStr}
	SaveAddScaleLog(payload)
}

func (p detailListedNotifier) Handle(scaleMgr *ScaleMgr) {
	// Do something for this event
	log.Log.Debug("Handle detailListedNotifier called")
	// Do something with this event
	detailLists, _ := NewDetailRecProvider().GetRecsList()

	var detailsStr string
	var err error
	if detailsStr, err = json.MarshalToString(detailLists); err != nil {
		log.Log.Errorf("%v\n", err)
		// TODO: error handling
	}
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_DETAIL_LIST, MsgBody: detailsStr}
}

func (p scaleSrvListNotifier) Handle(scaleMgr *ScaleMgr, srvIdStr string) {
	// Do something for this event
	log.Log.Debug("Handle scaleSrvListNotifier called")
	// Do something with this event
	srvScaleList, _ := scaleMgr.connPb.connPb.GetSrvScaleRelList()
	srvId, _ := strconv.Atoi(srvIdStr)
	var relsStr string
	var sendRelList []*SrvScaleRel

	if srvId < 999999900 {
		var err error
		if relsStr, err = json.MarshalToString(sendRelList); err != nil {
			log.Log.Errorf("%v\n", err)
		}
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SCALE_SRV_LIST, MsgBody: relsStr}
		return
	}

	for _, rel := range srvScaleList {
		if rel.SrvId == int64(srvId) {
			sendRelList = append(sendRelList, rel)
		}
	}
	var err error
	if relsStr, err = json.MarshalToString(srvScaleList); err != nil {
		log.Log.Errorf("%v\n", err)
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_GET_SCALE_SRV_LIST, MsgBody: relsStr}
}

func (p setScaleSrvValNotifier) Handle(scaleMgr *ScaleMgr, rel SrvScaleRel) {
	// Do something for this event
	log.Log.Debug("Handle setScaleSrvValNotifier called")
	// Do something with this event
	if err := mSrvMgr.scaleMgr.UpdateSrvScaleVal(rel); err != nil {
		log.Log.Errorf("%v\n", err)
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SET_SCALE_SRV_VAL, MsgBody: "ok"}
}

func (p delScaleNotifier) Handle(mgr *SrvMgr, payload ReqDelScale) { //修改秤的属性
	// Do something for this event
	log.Log.Debug("Handle delScaleNotifier called")
	//先判断是否有配方使用了这个秤，使用了，不能删除
	isUsed, err := mgr.formulaPd.CheckFormulaRawData(int(payload.ScaleId))
	if err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_DEL, MsgBody: "fail,open db error"}
		return
	}

	if isUsed {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_DEL, MsgBody: "fail,this scale is used by formula"}
		return
	}

	scale, ok := mSrvMgr.scaleMgr.scales[payload.ScaleId]
	if !ok || scale == nil || scale.Conn == nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_DEL, MsgBody: "fail,scale not found"}
		return
	}
	delScaleInfo := scale.Conn.MediaConf
	delScaleName := scale.Conn.ScaleName

	if err := mSrvMgr.scaleMgr.DelScale(payload.ScaleId); err != nil {
		log.Log.Errorf("%v\n", err)
		resp := MgrRespMsg{IsAck: true, AckData: err.Error()}
		jsonStr, _ := json.MarshalToString(resp)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_DEL, MsgBody: jsonStr}
		return
	}
	resp := MgrRespMsg{IsAck: true, AckData: "ok"}
	jsonStr, _ := json.MarshalToString(resp)
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_DEL, MsgBody: jsonStr}
	SaveDelScaleInfoLog(delScaleInfo, delScaleName)
}

func (p modifyScaleNotifier) Handle(payload ReqModifyScale) { //修改秤的属性
	// Do something for this event
	log.Log.Debug("Handle modifyScaleNotifier called")

	scale, ok := mSrvMgr.scaleMgr.scales[payload.ScaleId]
	if !ok || scale == nil || scale.Conn == nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_MODIFY, MsgBody: "fail,scale not found"}
		return
	}

	oldMedia := scale.Conn.MediaConf
	newMedia := payload.MediaConf
	scaleName := scale.Conn.ScaleName

	if err := mSrvMgr.scaleMgr.UpdateScale(payload); err != nil {
		log.Log.Errorf("%v\n", err)
		resp := MgrRespMsg{IsAck: true, AckData: err.Error()}
		jsonStr, _ := json.MarshalToString(resp)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_MODIFY, MsgBody: jsonStr}
	}
	resp := MgrRespMsg{IsAck: true, AckData: ""}
	jsonStr, _ := json.MarshalToString(resp)
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_MODIFY, MsgBody: jsonStr}

	SaveModifyScaleLog(newMedia, oldMedia, scaleName)
}

func (p modifyScaleNameNotifier) Handle(payload ReqModifyScaleName) { //修改秤的属性
	// Do something for this event
	log.Log.Debug("Handle modifyScaleNameNotifier called")

	scale, ok := mSrvMgr.scaleMgr.scales[payload.ScaleId]
	if !ok || scale == nil || scale.Conn == nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_MODIFY, MsgBody: "fail,scale not found"}
		return
	}

	oldName := scale.Conn.ScaleName

	if err := mSrvMgr.scaleMgr.UpdateScaleName(payload); err != nil {
		log.Log.Errorf("%v\n", err)
		resp := MgrRespMsg{IsAck: true, AckData: err.Error()}
		jsonStr, _ := json.MarshalToString(resp)
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_MODIFY, MsgBody: jsonStr}
	}
	resp := MgrRespMsg{IsAck: true, AckData: ""}
	jsonStr, _ := json.MarshalToString(resp)
	// send ports list back to requestee
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_SCALE_MODIFY, MsgBody: jsonStr}
	SaveModifyScaleNameLog(payload.ScaleName, oldName)

}

// Run function will scan the scale from the scale list that from database, will inform srvMgr if any scale's online state is changed
func (s *ScaleMgr) Run() {
	// list serial from time to time to check if the port that connecting to scale is varied
	s.medias, _ = s.connPb.GetScaleConnsList() // scaleId "0000" is for get all scale connections
	//如果没有秤则新增一个串口
	//统一由此处去新增一个秤，当需要新增的时候只需要将秤加入数据库即可

	// if len(s.medias) == 0 {
	// 	var comInfo ComInfo = ComInfo{DevPath: "COM3", Baud: 115200, DataBits: 8, Parity: 0, StopBits: 0}
	// 	var conf MediaConf = MediaConf{}
	// 	conf.Type = MEDIA_COM
	// 	conf.MediaInfoJson, _ = json.MarshalToString(comInfo)
	// 	scaleConn := &ScaleConnMedia{IsOnline: false, ScaleCat: comm.SCALE_TMAX, ScaleId: 1, ScaleModel: "T-Max", ScaleSn: "123456", TMedia: MEDIA_COM, MediaConf: conf, IsDefault: true, ScaleName: "ComScale"}
	// 	s.connPb.connPb.InsertScaleConn(*scaleConn)
	// }

	s.medias, _ = s.connPb.GetScaleConnsList()
	//将所有的秤都与服务建立对应关系
	s.srvMgr.srvScaleRel, _ = s.connPb.GetScaleSrvRelList()

	if len(s.srvMgr.srvScaleRel) == 0 {
		for _, conn := range s.medias {
			var srvScaleRel SrvScaleRel
			for _, srvId := range SrvIdList {
				srvScaleRel.ScaleId = conn.ScaleId
				srvScaleRel.SrvId = srvId
				srvScaleRel.IsUsed = true
				s.srvMgr.srvScaleRel = append(s.srvMgr.srvScaleRel, &srvScaleRel)
				s.connPb.InsertSrvScaleRel(srvScaleRel)
			}
		}

	}
	for _, conn := range s.medias {
		if conn.scale == nil {
			// 在实例化之前，强行修正一次 ScaleCat，防止数据库里的历史脏数据导致类型错误
			if conn.ProtocolName != "" {
				conn.ScaleCat = comm.SCALE_TMAX
				if conn.ProtocolName != "SCP-X" {
					conn.ScaleCat = comm.SCALE_C51
				}
			}

			// new scale and assign scaleid to the instance
			var scale *Scale
			scale, _ = NewScale(s, conn, conn.ScaleCat, conn.ScaleModel, conn.ScaleSn, false)
			scale.Id = conn.ScaleId
			conn.scale = scale
			conn.ScaleId = scale.Id
			s.scales[scale.Id] = scale
			s.srvMgr.addScale <- scale // register new scale instance to srvMgr

		}
	}

	// StartAutoMountTask(s)

	for {

		// construct scale instance if it doesn't exist

		ports, _ := getPortsList()
		fmt.Printf("ports: %v\n", ports)

		// portsNotInUse := handlePortState(&ports, &s.conns)
		// portsNotInUse := handlePortState(&ports, &s.medias)
		_ = handleNetState(&s.medias)
		// fmt.Printf("portsNotInUse:  %v\n", portsNotInUse)
		// TODO: scan ports for finding scales and handle new finding scales

		// TODO: handle scales that offline
		time.Sleep(2 * time.Second) // detect connectivity every 2 seconds
	}
}

func (s *ScaleMgr) ModifyMediaList(scaleId int64, conf MediaConf) error {
	isFound := false
	for i, media := range s.medias {
		if media.ScaleId == scaleId {
			s.medias[i].MediaConf = conf
			isFound = true
			break
		}
	}
	if !isFound {
		return fmt.Errorf("not found the scale conf")

	}

	return nil
}

func (s *ScaleMgr) ModifyScaleInfo(scaleId int64, modelName string, sn string) error {
	isFound := false
	for i, media := range s.medias {
		if media.ScaleId == scaleId {
			s.medias[i].InnerModel = modelName
			s.medias[i].ScaleSn = sn
			isFound = true
			break
		}
	}
	if !isFound {
		return fmt.Errorf("not found the scale conf")
	}

	return nil
}

func (s *ScaleMgr) ModifyScaleName(scaleId int64, name string) error {
	isFound := false
	for i, media := range s.medias {
		if media.ScaleId == scaleId {
			s.medias[i].ScaleName = name
			isFound = true
			break
		}
	}
	if !isFound {
		return fmt.Errorf("not found the scale conf")
	}

	return nil
}

func (s *ScaleMgr) AddMediaList(scaleId int64, conn ScaleConnMedia) error {
	isFound := false
	for i, media := range s.medias {
		if media.ScaleId == scaleId {
			s.medias[i].MediaConf = conn.MediaConf
			isFound = true
			break
		}
	}
	if isFound {
		return fmt.Errorf("exsit the scale conf")
	}
	s.medias = append(s.medias, &conn)

	return nil
}

func (s *ScaleMgr) DelMediaList(scaleId int64, conn ScaleConnMedia) error {
	result := []*ScaleConnMedia{}
	for _, m := range s.medias {
		if m.ScaleId != scaleId {
			result = append(result, m)
		}
	}
	s.medias = result
	return nil

}

func (s *ScaleMgr) DelSrvScaleList(scaleId int64) error {
	result := []*SrvScaleRel{}
	for _, rel := range s.srvMgr.srvScaleRel {
		if rel.ScaleId != scaleId {
			result = append(result, rel)
		}
	}
	s.srvMgr.srvScaleRel = result
	return nil

}

func (s *ScaleMgr) AddSrvScaleList(scaleId int64) error {
	for _, rel := range SrvIdList {
		relScale := SrvScaleRel{ScaleId: scaleId, SrvId: rel, IsUsed: true}
		s.srvMgr.srvScaleRel = append(s.srvMgr.srvScaleRel, &relScale)
		s.connPb.InsertSrvScaleRel(relScale)
	}
	return nil

}

func (s *ScaleMgr) UpdateSrvScaleVal(relInfo SrvScaleRel) error {

	result := []*SrvScaleRel{}
	for _, rel := range s.srvMgr.srvScaleRel {
		if rel.ScaleId == relInfo.ScaleId && rel.SrvId == relInfo.SrvId {
			continue
		} else {
			result = append(result, rel)
		}
	}
	result = append(result, &relInfo)
	s.srvMgr.srvScaleRel = result
	s.connPb.UpdateSrvScaleRel(relInfo)
	return nil

}

// func sContainsConnMedia(conn *ScaleConnMedia) bool {
// 	return conn.scale != nil
// }

func handlePortState(inPorts *[]string, conns *[]*ScaleConnMedia) (portsNotInUse []string) {
	var comInfo ComInfo

	for _, port := range *inPorts {
		isFound := false
		for _, conn := range *conns {
			if conn.TMedia == MEDIA_COM {
				if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &comInfo); err != nil {
					return nil // TODO: check error
				}

				if port == comInfo.DevPath {
					isFound = true
					break
				}
			}
		}
		if !isFound {
			portsNotInUse = append(portsNotInUse, port)
		}
	}

	for i, conn := range *conns {
		if conn.TMedia == MEDIA_COM {
			if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &comInfo); err != nil {
				return nil // TODO: check error
			}
			if _, isFound := contains(*inPorts, comInfo.DevPath); isFound {
				if !(*conns)[i].IsOnline {
					(*conns)[i].IsOnline = true
				}

				continue
			}
			(*conns)[i].IsOnline = false
		}

	}
	return portsNotInUse
}

func handleNetState(conns *[]*ScaleConnMedia) (netsNotInUse []string) {

	var netInfo NetInfo

	for i, conn := range *conns {
		if conn.TMedia == MEDIA_NET {
			if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &netInfo); err != nil {
				return nil // TODO: check error
			}
			if i >= len(*conns) {
				return nil
			}
			if conn.scale == nil {
				if (*conns)[i].IsOnline {
					(*conns)[i].IsOnline = false
				}
				continue
			}

			if conn.scale.MyNet == nil {
				if (*conns)[i].IsOnline {
					(*conns)[i].IsOnline = false
				}
				continue
			}
			if conn.scale.MyNet.conn == nil {
				if (*conns)[i].IsOnline {
					(*conns)[i].IsOnline = false
					// fmt.Println("false" + (*conns)[i].scale.MyNet.ip)
				}
				// fmt.Println("false" + (*conns)[i].scale.MyNet.ip)
				continue

			}

			if conn.scale.MyNet.conn != nil && conn.scale.MyNet.isAlive {
				if !(*conns)[i].IsOnline {
					(*conns)[i].IsOnline = true
					// fmt.Println("true" + (*conns)[i].scale.MyNet.ip)
				}
				// fmt.Println("true" + (*conns)[i].scale.MyNet.ip)
				continue

			}

		}
		time.Sleep(1000 * time.Millisecond)

	}

	return
}

func contains(s []string, e string) (int, bool) {
	for i, a := range s {
		if a == e {
			return i, true
		}
	}
	return -1, false
}

func (m *ScaleMgr) GetScale(id int64) (*Scale, error) {
	scale := m.scales[id]
	if scale == nil {
		return nil, fmt.Errorf("can't find scale")
	}

	return scale, nil
}

// 随机写个sn初始化的时候，并不知道SN是什么
func getSn() string {
	timestamp := time.Now().Unix()
	// 将时间戳转换为字符串
	strTimestamp := strconv.FormatInt(timestamp, 10)
	return strTimestamp

}

// for user to add a scale, should avoid to overwrite existing scale
func (s *ScaleMgr) AddScale(req ReqAddScale) error {
	// check if the scaleConn is existing via checking the scale's model and scale's sn
	//TODO:要用于增加管理的秤  202406
	s.medias, _ = s.connPb.GetScaleConnsList()

	var reqComInfo ComInfo

	if req.MediaConf.Type == MEDIA_COM {
		if err := json.UnmarshalFromString(req.MediaConf.MediaInfoJson, &reqComInfo); err != nil {
			return err
		}

		for _, conn := range s.medias {
			if conn.MediaConf.Type == MEDIA_COM {
				var existingComInfo ComInfo
				if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &existingComInfo); err == nil {
					if reqComInfo.DevPath == existingComInfo.DevPath {
						return fmt.Errorf("this serial port already exists")
					}
				}
			}
		}

		if len(s.medias) == 0 {
			nextScaleId = 1
		} else {
			maxScaleID := int64(0)
			// 遍历 s.medias 找出最大的 ScaleId
			for _, media := range s.medias {
				if media.ScaleId > maxScaleID {
					maxScaleID = media.ScaleId
				}
			}
			nextScaleId = maxScaleID + 1
		}
		scaleCat := comm.SCALE_TMAX
		if req.ProtocolName != "SCP-X" {
			scaleCat = comm.SCALE_C51
		}
		scaleName := "Scale" + strconv.FormatInt(nextScaleId, 10)
		var comInfo ComInfo = ComInfo{DevPath: reqComInfo.DevPath, Baud: reqComInfo.Baud, DataBits: 8, Parity: 0, StopBits: 0}
		var conf MediaConf = MediaConf{}
		conf.Type = MEDIA_COM
		conf.MediaInfoJson, _ = json.MarshalToString(comInfo)

		customModel := req.ScaleModel
		var mSn string
		var modbusId int
		if req.ProtocolName == "SCP-X" {
			mSn = req.ScaleSn
			if mSn == "" {
				mSn = getSn()
			}
			modbusId = req.ModbusId
		} else {
			mSn = ""
			modbusId = 0
		}

		scaleConn := &ScaleConnMedia{
			IsOnline:     false,
			ScaleCat:     scaleCat,
			ScaleId:      nextScaleId,
			ScaleModel:   customModel,
			CustomModel:  customModel,
			InnerModel:   customModel,
			ProtocolName: req.ProtocolName,
			ScaleSn:      mSn,
			TMedia:       MEDIA_COM,
			MediaConf:    conf,
			IsDefault:    true,
			ScaleName:    scaleName,
			ModbusId:     modbusId,
		}
		s.connPb.connPb.InsertScaleConn(*scaleConn)
		s.AddMediaList(scaleConn.ScaleId, *scaleConn)

		var scale *Scale

		scale, _ = NewScale(s, scaleConn, scaleConn.ScaleCat, scaleConn.ScaleModel, scaleConn.ScaleSn, false)
		scale.Id = scaleConn.ScaleId
		scaleConn.scale = scale

		s.scales[scale.Id] = scale
		s.srvMgr.addScale <- scale // register new scale instance to srvMgr

		s.AddSrvScaleList(scale.Id)

		return nil
	}

	var reqBtInfo BtInfo
	var btInfo BtInfo

	if req.MediaConf.Type == MEDIA_BT {
		if err := json.UnmarshalFromString(req.MediaConf.MediaInfoJson, &reqBtInfo); err != nil {
			return err
		}
		for _, conn := range s.medias {
			if conn.MediaConf.Type == MEDIA_BT {
				if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &btInfo); err != nil {
					return err
				}
				if reqBtInfo.Mac == btInfo.Mac {
					return fmt.Errorf("this address already exists")
				}
			}
		}

		if len(s.medias) == 0 {
			nextScaleId = 1
		} else {
			maxScaleID := int64(0)
			// 遍历 s.medias 找出最大的 ScaleId
			for _, media := range s.medias {
				if media.ScaleId > maxScaleID {
					maxScaleID = media.ScaleId
				}
			}
			nextScaleId = maxScaleID + 1
		}
		scaleCat := comm.SCALE_TMAX
		if req.ProtocolName != "SCP-X" {
			scaleCat = comm.SCALE_C51
		}
		scaleName := "Scale" + strconv.FormatInt(nextScaleId, 10)

		var btInfo BtInfo = BtInfo{Mac: reqBtInfo.Mac, Name: reqBtInfo.Name}
		var conf MediaConf = MediaConf{}
		conf.Type = MEDIA_BT
		conf.MediaInfoJson, _ = json.MarshalToString(btInfo)

		customModel := req.ScaleModel
		var mSn string
		var modbusId int
		if req.ProtocolName == "SCP-X" {
			mSn = req.ScaleSn
			if mSn == "" {
				mSn = getSn()
			}
			modbusId = req.ModbusId
		} else {
			mSn = ""
			modbusId = 0
		}

		scaleConn := &ScaleConnMedia{
			IsOnline:     false,
			ScaleCat:     scaleCat,
			ScaleId:      nextScaleId,
			ScaleModel:   customModel,
			CustomModel:  customModel,
			InnerModel:   customModel,
			ProtocolName: req.ProtocolName,
			ScaleSn:      mSn,
			TMedia:       MEDIA_BT,
			MediaConf:    conf,
			IsDefault:    true,
			ScaleName:    scaleName,
			ModbusId:     modbusId,
		}
		s.connPb.connPb.InsertScaleConn(*scaleConn)
		s.AddMediaList(scaleConn.ScaleId, *scaleConn)

		var scale *Scale

		scale, _ = NewScale(s, scaleConn, scaleConn.ScaleCat, scaleConn.ScaleModel, scaleConn.ScaleSn, false)
		scale.Id = scaleConn.ScaleId
		scaleConn.scale = scale

		s.scales[scale.Id] = scale
		s.srvMgr.addScale <- scale // register new scale instance to srvMgr

		s.AddSrvScaleList(scale.Id)

		return nil
	}

	//下面是新增网络秤
	var netInfo NetInfo
	var reqNetInfo NetInfo

	if req.MediaConf.Type != MEDIA_NET {
		return fmt.Errorf("only can add net scale")
	}

	if err := json.UnmarshalFromString(req.MediaConf.MediaInfoJson, &reqNetInfo); err != nil {
		return err
	}
	for _, conn := range s.medias {
		if conn.MediaConf.Type == MEDIA_NET {
			if err := json.UnmarshalFromString(conn.MediaConf.MediaInfoJson, &netInfo); err != nil {
				return err
			}
			if reqNetInfo.Ip == netInfo.Ip && reqNetInfo.Port == netInfo.Port {
				return fmt.Errorf("this IP address already exists")
			}
		}
	}

	if len(s.medias) == 0 {
		nextScaleId = 1
	} else {
		maxScaleID := int64(0)
		// 遍历 s.medias 找出最大的 ScaleId
		for _, media := range s.medias {
			if media.ScaleId > maxScaleID {
				maxScaleID = media.ScaleId
			}
		}
		nextScaleId = maxScaleID + 1
	}

	customModel := req.ScaleModel
	var mSn string
	var modbusId int
	if req.ProtocolName == "SCP-X" {
		mSn = req.ScaleSn
		if mSn == "" {
			mSn = getSn()
		}
		modbusId = req.ModbusId
	} else {
		mSn = ""
		modbusId = 0
	}

	scaleCat := comm.SCALE_TMAX
	if req.ProtocolName != "SCP-X" {
		scaleCat = comm.SCALE_C51
	}

	conn := &ScaleConnMedia{
		ScaleModel:   customModel,
		CustomModel:  customModel,
		InnerModel:   customModel,
		ProtocolName: req.ProtocolName,
		ScaleSn:      mSn,
		TMedia:       req.MediaConf.Type,
		MediaConf:    req.MediaConf,
		ScaleId:      nextScaleId,
		IsDefault:    true,
		IsOnline:     true,
		ScaleCat:     scaleCat,
		ScaleName:    "Scale" + strconv.FormatInt(nextScaleId, 10),
		ModbusId:     modbusId,
	}

	var scale *Scale

	scale, _ = NewScale(s, conn, conn.ScaleCat, conn.ScaleModel, conn.ScaleSn, false)
	scale.Id = conn.ScaleId
	conn.scale = scale

	s.scales[scale.Id] = scale
	s.srvMgr.addScale <- scale // register new scale instance to srvMgr

	s.connPb.connPb.InsertScaleConn(*conn)
	s.AddMediaList(scale.Id, *conn)
	//增加服务的对应关系

	s.AddSrvScaleList(scale.Id)

	return nil

}

// for user to delete a scale
func (s *ScaleMgr) DelScale(id int64) error {
	//用于删除管理的秤  202406
	scale := s.scales[id]
	if scale == nil {
		return fmt.Errorf("can't find scale with id: %v", id)
	}
	conn := scale.Conn
	if conn == nil {
		return fmt.Errorf("can't find connection associated with the scale Id")
	}
	if scale.Conn.TMedia == MEDIA_COM {
		// return fmt.Errorf("serial connect can not delete")
		client := s.srvMgr.clientOfScales[scale]
		if client != nil && client.scaleId == id {
			s.srvMgr.unregister <- client
			if client.conn != nil {
				client.conn.Close()
			} // terminate the socket that associate with the scale
		}
		// remove the conn then add new one s.conns
		if scale.MySerial != nil {
			scale.MySerial.toQuit = true
		}

		time.Sleep(500 * time.Millisecond)
		scale.Close()
		s.srvMgr.removeScale <- scale
		s.connPb.connPb.DeleteScaleConn(*conn)
		if s.scales[scale.Id] != nil { // scale not existing
			delete(s.scales, scale.Id)
		}
		s.DelMediaList(scale.Id, *conn)
		//删除连接关系
		s.connPb.DeleteSrvScaleRelByScaleId(scale.Id)
		s.DelSrvScaleList(scale.Id)
		return nil
	}

	if scale.Conn.TMedia == MEDIA_NET {
		client := s.srvMgr.clientOfScales[scale]
		if client != nil && client.scaleId == id {
			s.srvMgr.unregister <- client
			if client.conn != nil {
				client.conn.Close()
			}
		}
		if scale.MyNet != nil {
			scale.MyNet.toQuit = true
		}
		time.Sleep(500 * time.Millisecond)
		scale.Close()
		s.srvMgr.removeScale <- scale
		s.connPb.connPb.DeleteScaleConn(*conn)
		if s.scales[scale.Id] != nil { // scale not existing
			delete(s.scales, scale.Id)
		}
		s.DelMediaList(scale.Id, *conn)
		s.connPb.DeleteSrvScaleRelByScaleId(scale.Id)
		s.DelSrvScaleList(scale.Id)
		return nil

	}

	if scale.Conn.TMedia == MEDIA_BT {
		if scale.MyBluetooth != nil {
			scale.MyBluetooth.toQuit = true
		}
		time.Sleep(500 * time.Millisecond)
		scale.Close()

		client := s.srvMgr.clientOfScales[scale]
		if client != nil && client.scaleId == id {
			s.srvMgr.unregister <- client
			if client.conn != nil {
				client.conn.Close()
			}
		}

		s.srvMgr.removeScale <- scale
		s.connPb.connPb.DeleteScaleConn(*conn)
		if s.scales[scale.Id] != nil {
			delete(s.scales, scale.Id)
		}
		s.DelMediaList(scale.Id, *conn)
		s.connPb.DeleteSrvScaleRelByScaleId(scale.Id)
		s.DelSrvScaleList(scale.Id)
		return nil
	}
	return nil
}

// for user to update a scale
func (s *ScaleMgr) UpdateScale(req ReqModifyScale) error {
	id := req.ScaleId
	scale := s.scales[id]
	if scale == nil {
		return fmt.Errorf("can't find scale with id: %v", id)
	}

	conn := scale.Conn
	if conn == nil {
		return fmt.Errorf("can't find connection associated with the scale Id")
	}

	conn.MediaConf = req.MediaConf

	// 允许修改协议名称和机种名称，前提是前端传了有效值（非空）。防止空数据覆盖现有配置
	if req.ProtocolName != "" {
		conn.ProtocolName = req.ProtocolName
	}

	if req.ScaleModel != "" {
		conn.ScaleModel = req.ScaleModel
		conn.CustomModel = req.ScaleModel
	}

	if conn.ProtocolName == "SCP-X" {
		// 仅允许修改 ModbusId (如果有传)
		if req.ModbusId != 0 {
			conn.ModbusId = req.ModbusId
		}
	} else {
		conn.ModbusId = 0
		conn.ScaleSn = ""
		conn.InnerModel = conn.ScaleModel
	}

	// 重新计算 ScaleCat
	conn.ScaleCat = comm.SCALE_TMAX
	if conn.ProtocolName != "SCP-X" {
		conn.ScaleCat = comm.SCALE_C51
	}

	scale.Model = conn.ScaleModel
	scale.ScaleCat = conn.ScaleCat
	scale.Sn = conn.ScaleSn
	composer := cmdComposerFuncMap[scale.ScaleCat]
	scale.composer = &composer

	s.scales[id].ModifyMedia(req.MediaConf)
	s.srvMgr.scaleMgr.ModifyMediaList(id, conn.MediaConf)

	s.connPb.connPb.UpdateScaleConn(*conn)

	return nil
}

// for user to update a scale name
func (s *ScaleMgr) UpdateScaleName(req ReqModifyScaleName) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := req.ScaleId
	scale := s.scales[id]
	if scale == nil {
		return fmt.Errorf("can't find scale with id: %v", id)
	}

	if scale.Conn.ScaleName == req.ScaleName {
		return nil
	}

	conn := scale.Conn
	conn.ScaleName = req.ScaleName

	s.srvMgr.scaleMgr.ModifyScaleName(id, req.ScaleName)
	s.connPb.connPb.UpdateScaleName(*conn)
	return nil
}

// for user to update a scale
func (s *ScaleMgr) UpdateScaleSn(req ReqModifyScaleSn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := req.ScaleId
	scale := s.scales[id]
	if scale == nil {
		return fmt.Errorf("can't find scale with id: %v", id)
	}

	if scale.Conn.InnerModel == req.InnerModel && scale.Sn == req.Sn {
		return nil
	}
	scale.Sn = req.Sn
	conn := scale.Conn
	conn.InnerModel = req.InnerModel
	conn.ScaleSn = scale.Sn
	s.srvMgr.scaleMgr.ModifyScaleInfo(id, req.InnerModel, req.Sn)
	s.connPb.connPb.UpdateScaleSn(*conn)
	return nil
}

// func (s *ScaleMgr) GetScaleRecs(scale *Scale) ([]ScaleRec, error) {
// 	var recs []ScaleRec
// 	var err error
// 	if recs, err = s.recPb.GetRecsList(*scale, "", "", ""); err != nil {
// 		return recs, err
// 	}
// 	return recs, nil
// }

func (s *ScaleMgr) InsertScaleRec(rec ScaleRec) error {
	return s.recPb.InsertRec(rec)
}

func (s *ScaleMgr) DeleteScaleRec(recId uint) error {
	return s.recPb.DeleteRec(recId)
}

// Modbus Service Handlers
func (p getModbusServicesNotifier) Handle(scaleMgr *ScaleMgr) {
	log.Log.Debug("Handle getModbusServicesNotifier called")
	pb := NewModbusServiceProvider()
	services, err := pb.GetModbusServiceList()
	if err != nil {
		log.Log.Errorf("Error getting modbus services: %v", err)
	}
	jsonStr, _ := json.MarshalToString(services)
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_SERVICES, MsgBody: jsonStr}
}

func (p addModbusServiceNotifier) Handle(payload ReqAddModbusService) {
	log.Log.Debug("Handle addModbusServiceNotifier called")
	pb := NewModbusServiceProvider()
	service := ModbusServiceInfo{
		TargetModbusId: payload.TargetModbusId,
		Protocol:       payload.Protocol,
		Port:           payload.Port,
		BaudRate:       payload.BaudRate,
	}
	if err := pb.InsertModbusService(&service); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_ADD, MsgBody: err.Error()}
		return
	}

	// 启动对应的 Modbus 网关服务
	if mSrvMgr != nil && mSrvMgr.ModbusGateway != nil {
		mSrvMgr.ModbusGateway.StartService(service)
	}

	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_ADD, MsgBody: "ok"}
}

func (p editModbusServiceNotifier) Handle(payload ReqEditModbusService) {
	log.Log.Debug("Handle editModbusServiceNotifier called")
	pb := NewModbusServiceProvider()
	service := ModbusServiceInfo{
		Id:             payload.Id,
		TargetModbusId: payload.TargetModbusId,
		Protocol:       payload.Protocol,
		Port:           payload.Port,
		BaudRate:       payload.BaudRate,
	}
	if err := pb.UpdateModbusService(service); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_EDIT, MsgBody: err.Error()}
		return
	}

	if mSrvMgr != nil && mSrvMgr.ModbusGateway != nil {
		mSrvMgr.ModbusGateway.StartService(service) // 重新启动该服务
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_EDIT, MsgBody: "ok"}
}

func (p delModbusServiceNotifier) Handle(payload ReqDelModbusService) {
	log.Log.Debug("Handle delModbusServiceNotifier called")
	pb := NewModbusServiceProvider()
	if err := pb.DeleteModbusService(payload.Id); err != nil {
		mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_DEL, MsgBody: err.Error()}
		return
	}

	if mSrvMgr != nil && mSrvMgr.ModbusGateway != nil {
		mSrvMgr.ModbusGateway.StopService(payload.Id)
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: SCALE_MGR_RESP_MODBUS_DEL, MsgBody: "ok"}
}
