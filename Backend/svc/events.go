package svc

var portsListed PortsListed

type PortsListed struct {
	handlers []interface{ Handle() }
}

// Register adds an event handler for this event
func (u *PortsListed) Register(handler interface{ Handle() }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u PortsListed) Trigger() {
	for _, handler := range u.handlers {
		go handler.Handle()
	}
}

var btListed BtListed

type BtListed struct {
	handlers []interface{ Handle() }
}

// Register adds an event handler for this event
func (u *BtListed) Register(handler interface{ Handle() }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u BtListed) Trigger() {
	for _, handler := range u.handlers {
		go handler.Handle()
	}
}

var scalesListedSrv ScaleListedSrv

type ScaleListedSrv struct {
	handlers []interface {
		Handle(scaleMgr *ScaleMgr, scaleId int64)
	}
}

// Register adds an event handler for this event
func (u *ScaleListedSrv) Register(handler interface {
	Handle(payload *ScaleMgr, scaleId int64)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleListedSrv) Trigger(scaleMgr *ScaleMgr, scaleId int64) {
	for _, handler := range u.handlers {
		go handler.Handle(scaleMgr, scaleId)
	}
}

var scalesListed ScaleListed

type ScaleListed struct {
	handlers []interface{ Handle(scaleMgr *ScaleMgr) }
}

// Register adds an event handler for this event
func (u *ScaleListed) Register(handler interface{ Handle(payload *ScaleMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleListed) Trigger(payload *ScaleMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var scaleAdded ScaleAdded

type ScaleAdded struct {
	handlers []interface{ Handle(payload ReqAddScale) }
}

// Register adds an event handler for this event
func (u *ScaleAdded) Register(handler interface{ Handle(ReqAddScale) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleAdded) Trigger(payload ReqAddScale) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var scaleModified ScaleModified

type ScaleModified struct {
	handlers []interface{ Handle(payload ReqModifyScale) }
}

// Register adds an event handler for this event
func (u *ScaleModified) Register(handler interface{ Handle(payload ReqModifyScale) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleModified) Trigger(payload ReqModifyScale) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var scaleNameModified ScaleNameModified

type ScaleNameModified struct {
	handlers []interface {
		Handle(payload ReqModifyScaleName)
	}
}

// Register modify an event handler for this event
func (u *ScaleNameModified) Register(handler interface {
	Handle(payload ReqModifyScaleName)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleNameModified) Trigger(payload ReqModifyScaleName) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var scaleDeleted ScaleDeleted

type ScaleDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelScale)
	}
}

// Register adds an event handler for this event
func (u *ScaleDeleted) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelScale)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleDeleted) Trigger(mgr *SrvMgr, payload ReqDelScale) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 分页获取PLU列表
var pluByPageListed PluByPageListed

type PluByPageListed struct {
	handlers []interface {
		Handle(mgr *SrvMgr, data ReqGetPluByPage)
	}
}

// Register adds an event handler for this event
func (u *PluByPageListed) Register(handler interface {
	Handle(mgr *SrvMgr, data ReqGetPluByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u PluByPageListed) Trigger(mgr *SrvMgr, data ReqGetPluByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, data)
	}
}

// 清空产品列表
var productCleared ProductCleared

type ProductCleared struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *ProductCleared) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductCleared) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

var productsListed ProductListed

type ProductListed struct {
	handlers []interface{ Handle(mgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *ProductListed) Register(handler interface{ Handle(mgr *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductListed) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 导出PLU列表到excel文件

var exportPluToFile ExportPluToFile

type ExportPluToFile struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *ExportPluToFile) Register(handler interface {
	Handle(mgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ExportPluToFile) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取PLU设置字段
var getPluSetting GetPluSetting

type GetPluSetting struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *GetPluSetting) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetPluSetting) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 设置PLU字段
var pluSetting PluSetting

type PluSetting struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqPluSetting)
	}
}

// Register adds an event handler for this event
func (u *PluSetting) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqPluSetting)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u PluSetting) Trigger(mgr *SrvMgr, payload ReqPluSetting) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 导出
var exportProduct ExportProduct

type ExportProduct struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqExportProduct)
	}
}

// Register adds an event handler for this event
func (u *ExportProduct) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqExportProduct)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ExportProduct) Trigger(mgr *SrvMgr, payload ReqExportProduct) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var productAdded ProductAdded

type ProductAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddPlu)
	}
}

// Register adds an event handler for this event
func (u *ProductAdded) Register(handler interface {
	Handle(*SrvMgr, ReqAddPlu)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductAdded) Trigger(mgr *SrvMgr, payload ReqAddPlu) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 检查plu是否存在
var checkPluExist CheckPluExist

type CheckPluExist struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *CheckPluExist) Register(handler interface {
	Handle(mgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u CheckPluExist) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var productAddedOne ProductAddedOne

type ProductAddedOne struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload AddProduct)
	}
}

// Register adds an event handler for this event
func (u *ProductAddedOne) Register(handler interface {
	Handle(*SrvMgr, AddProduct)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductAddedOne) Trigger(mgr *SrvMgr, payload AddProduct) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var productModified ProductModified

type ProductModified struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload AddProduct)
	}
}

// Register adds an event handler for this event
func (u *ProductModified) Register(handler interface {
	Handle(mgr *SrvMgr, payload AddProduct)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductModified) Trigger(mgr *SrvMgr, payload AddProduct) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var getLastProductRec GetLastProductRec

type GetLastProductRec struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *GetLastProductRec) Register(handler interface {
	Handle(mgr *SrvMgr)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetLastProductRec) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

var productDeleted ProductDeleted

type ProductDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelProduct)
	}
}

// Register adds an event handler for this event
func (u *ProductDeleted) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelProduct)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductDeleted) Trigger(mgr *SrvMgr, payload ReqDelProduct) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var productDeletedAll ProductDeletedAll

type ProductDeletedAll struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event

func (u *ProductDeletedAll) Register(handler interface {
	Handle(mgr *SrvMgr)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ProductDeletedAll) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

var updateEnabledPlu UpdateEnabledPlu

type UpdateEnabledPlu struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqUpdateEnabledPlu)
	}
}

// Register adds an event handler for this event
func (u *UpdateEnabledPlu) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqUpdateEnabledPlu)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateEnabledPlu) Trigger(mgr *SrvMgr, payload ReqUpdateEnabledPlu) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var usersListed UserListed

type UserListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *UserListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UserListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var userAdded UserAdded

type UserAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddUser)
	}
}

// Register adds an event handler for this event
func (u *UserAdded) Register(handler interface{ Handle(*SrvMgr, ReqAddUser) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UserAdded) Trigger(mgr *SrvMgr, payload ReqAddUser) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var userModified UserModified

type UserModified struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqModifyUser)
	}
}

// Register adds an event handler for this event
func (u *UserModified) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqModifyUser)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UserModified) Trigger(mgr *SrvMgr, payload ReqModifyUser) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var userDeleted UserDeleted

type UserDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelUser)
	}
}

// Register adds an event handler for this event
func (u *UserDeleted) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelUser)
},
) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UserDeleted) Trigger(mgr *SrvMgr, payload ReqDelUser) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var detailListed DetailListed

type DetailListed struct {
	handlers []interface{ Handle(scaleMgr *ScaleMgr) }
}

// Register adds an event handler for this event
func (u *DetailListed) Register(handler interface{ Handle(payload *ScaleMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DetailListed) Trigger(payload *ScaleMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var scaleSrvList ScaleSrvList

type ScaleSrvList struct {
	handlers []interface {
		Handle(scaleMgr *ScaleMgr, scaleIdStr string)
	}
}

// Register adds an event handler for this event
func (u *ScaleSrvList) Register(handler interface {
	Handle(payload *ScaleMgr, scaleIdStr string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ScaleSrvList) Trigger(payload *ScaleMgr, scaleIdStr string) {
	for _, handler := range u.handlers {
		go handler.Handle(payload, scaleIdStr)
	}
}

var setScaleSrvVal SetScaleSrvVal

type SetScaleSrvVal struct {
	handlers []interface {
		Handle(scaleMgr *ScaleMgr, relInfo SrvScaleRel)
	}
}

// Register adds an event handler for this event
func (u *SetScaleSrvVal) Register(handler interface {
	Handle(payload *ScaleMgr, relInfo SrvScaleRel)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u SetScaleSrvVal) Trigger(payload *ScaleMgr, relInfo SrvScaleRel) {
	for _, handler := range u.handlers {
		go handler.Handle(payload, relInfo)
	}
}

var wifiListed WifiListed

type WifiListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *WifiListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u WifiListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var wifiAdded WifiAdded

type WifiAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddWifi)
	}
}

// Register adds an event handler for this event
func (u *WifiAdded) Register(handler interface{ Handle(*SrvMgr, ReqAddWifi) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u WifiAdded) Trigger(mgr *SrvMgr, payload ReqAddWifi) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var sendToSrv1 SendToSrv1

type SendToSrv1 struct {
	handlers []interface {
		Handle(mgr *SrvMgr, jsonStr string)
	}
}

// Register adds an event handler for this event
func (u *SendToSrv1) Register(handler interface {
	Handle(payload *SrvMgr, jsonStr string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u SendToSrv1) Trigger(mgr *SrvMgr, jsonStr string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, jsonStr)
	}
}

var sendToUi SendToUi

type SendToUi struct {
	handlers []interface {
		Handle(mgr *SrvMgr, jsonStr string)
	}
}

// Register adds an event handler for this event
func (u *SendToUi) Register(handler interface {
	Handle(payload *SrvMgr, jsonStr string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u SendToUi) Trigger(mgr *SrvMgr, jsonStr string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, jsonStr)
	}
}

var doServiceAction DoServiceAction

type DoServiceAction struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDoServiceAction)
	}
}

// Register adds an event handler for this event
func (u *DoServiceAction) Register(handler interface {
	Handle(*SrvMgr, ReqDoServiceAction)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DoServiceAction) Trigger(mgr *SrvMgr, payload ReqDoServiceAction) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 配方秤
// 新增原料类型
var rawTypeAdded RawTypeAdded

type RawTypeAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddRawType)
	}
}

// Register adds an event handler for this event
func (u *RawTypeAdded) Register(handler interface{ Handle(*SrvMgr, ReqAddRawType) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawTypeAdded) Trigger(mgr *SrvMgr, payload ReqAddRawType) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 修改原料类型
var rawTypeModified RawTypeModified

type RawTypeModified struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqEditRawType)
	}
}

// Register adds an event handler for this event
func (u *RawTypeModified) Register(handler interface {
	Handle(*SrvMgr, ReqEditRawType)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawTypeModified) Trigger(mgr *SrvMgr, payload ReqEditRawType) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除原料类型
var rawTypeDeleted RawTypeDeleted

type RawTypeDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddRawType)
	}
}

// Register adds an event handler for this event
func (u *RawTypeDeleted) Register(handler interface {
	Handle(*SrvMgr, ReqAddRawType)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawTypeDeleted) Trigger(mgr *SrvMgr, payload ReqAddRawType) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除未使用的原料类型
var rawTypeUnusedDeleted RawTypeUnusedDeleted

type RawTypeUnusedDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *RawTypeUnusedDeleted) Register(handler interface {
	Handle(*SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawTypeUnusedDeleted) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 删除未使用的配方类型
var fmaTypeUnusedDeleted FmaTypeUnusedDeleted

type FmaTypeUnusedDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *FmaTypeUnusedDeleted) Register(handler interface {
	Handle(*SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FmaTypeUnusedDeleted) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 修改配方类型
var fmaTypeModified FmaTypeModified

type FmaTypeModified struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqEditRawType)
	}
}

// Register adds an event handler for this event
func (u *FmaTypeModified) Register(handler interface {
	Handle(*SrvMgr, ReqEditRawType)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FmaTypeModified) Trigger(mgr *SrvMgr, payload ReqEditRawType) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除配方类型
var fmaTypeDeleted FmaTypeDeleted

type FmaTypeDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddRawType)
	}
}

// Register adds an event handler for this event
func (u *FmaTypeDeleted) Register(handler interface {
	Handle(*SrvMgr, ReqAddRawType)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FmaTypeDeleted) Trigger(mgr *SrvMgr, payload ReqAddRawType) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 新增配方类型
var formulaTypeAdded FormulaTypeAdded

type FormulaTypeAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddFormulaType)
	}
}

// Register adds an event handler for this event
func (u *FormulaTypeAdded) Register(handler interface {
	Handle(*SrvMgr, ReqAddFormulaType)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaTypeAdded) Trigger(mgr *SrvMgr, payload ReqAddFormulaType) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

//获取配方类型列表

var formulaTypeListed FormulaTypeListed

type FormulaTypeListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *FormulaTypeListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaTypeListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

//获取配方类型列表

var rawTypeListed RawTypeListed

type RawTypeListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *RawTypeListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawTypeListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

// 根据配方ID获取原料输出端口
var getRawOutputByFmaId GetRawOutputByFmaId

type GetRawOutputByFmaId struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqGetRawOutputByFmaId)
	}
}

// Register adds an event handler for this event
func (u *GetRawOutputByFmaId) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqGetRawOutputByFmaId)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetRawOutputByFmaId) Trigger(mgr *SrvMgr, payload ReqGetRawOutputByFmaId) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 新增原料数据
var rawDataAdded RawDataAdded

type RawDataAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddRawData)
	}
}

// Register adds an event handler for this event
func (u *RawDataAdded) Register(handler interface{ Handle(*SrvMgr, ReqAddRawData) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawDataAdded) Trigger(mgr *SrvMgr, payload ReqAddRawData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 导入原料数据列表
var rawListImported RawListImported

type RawListImported struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqImportRawList)
	}
}

// Register adds an event handler for this event
func (u *RawListImported) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqImportRawList)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawListImported) Trigger(mgr *SrvMgr, payload ReqImportRawList) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 导入配方数据列表

var formulaListImported FormulaListImported

type FormulaListImported struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqImportFmaList)
	}
}

// Register adds an event handler for this event
func (u *FormulaListImported) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqImportFmaList)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaListImported) Trigger(mgr *SrvMgr, payload ReqImportFmaList) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取原料数据列表
var rawDataListed RawDataListed

type RawDataListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *RawDataListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawDataListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

// 修改原料数据
var rawDataEdited RawDataEdited

type RawDataEdited struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqEditRawData)
	}
}

// Register adds an event handler for this event
func (u *RawDataEdited) Register(handler interface{ Handle(*SrvMgr, ReqEditRawData) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawDataEdited) Trigger(mgr *SrvMgr, payload ReqEditRawData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除原料数据
var rawDataDeleted RawDataDeleted

type RawDataDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelRawData)
	}
}

// Register adds an event handler for this event
func (u *RawDataDeleted) Register(handler interface{ Handle(*SrvMgr, ReqDelRawData) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawDataDeleted) Trigger(mgr *SrvMgr, payload ReqDelRawData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var formulaDataAdded FormulaDataAdded

type FormulaDataAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddFormulaData)
	}
}

// Register adds an event handler for this event
func (u *FormulaDataAdded) Register(handler interface {
	Handle(*SrvMgr, ReqAddFormulaData)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaDataAdded) Trigger(mgr *SrvMgr, payload ReqAddFormulaData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var formulaDataEdited FormulaDataEdited

type FormulaDataEdited struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddFormulaData)
	}
}

// Register adds an event handler for this event
func (u *FormulaDataEdited) Register(handler interface {
	Handle(*SrvMgr, ReqAddFormulaData)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaDataEdited) Trigger(mgr *SrvMgr, payload ReqAddFormulaData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 检查配方ID和条码是否匹配
var checkFmaIdAndBarcode CheckFmaIdAndBarcode

type CheckFmaIdAndBarcode struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqCheckFmaIdAndBarcode)
	}
}

// Register adds an event handler for this event
func (u *CheckFmaIdAndBarcode) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqCheckFmaIdAndBarcode)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u CheckFmaIdAndBarcode) Trigger(mgr *SrvMgr, payload ReqCheckFmaIdAndBarcode) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取配方数据by 条码
var getFormulaByBarcode GetFormulaByBarcode

type GetFormulaByBarcode struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqGetFormulaByBarcode)
	}
}

// Register adds an event handler for this event
func (u *GetFormulaByBarcode) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqGetFormulaByBarcode)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetFormulaByBarcode) Trigger(mgr *SrvMgr, payload ReqGetFormulaByBarcode) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var formulaRecList FormulaRecListed

type FormulaRecListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *FormulaRecListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaRecListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

// 获取原料数据
var rawDataGetted RawDataGetted

type RawDataGetted struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *RawDataGetted) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawDataGetted) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取配方数据
var formulaData FormulaData

type FormulaData struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *FormulaData) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaData) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 新增配方称重记录
var formulaWgtRecAdded FormulaWgtRecAdded

type FormulaWgtRecAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqFormulaWgtRec)
	}
}

// Register adds an event handler for this event
func (u *FormulaWgtRecAdded) Register(handler interface {
	Handle(*SrvMgr, ReqFormulaWgtRec)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaWgtRecAdded) Trigger(mgr *SrvMgr, payload ReqFormulaWgtRec) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 根据订单号获取配方称重记录
var getFmaRecByOrderId GetFmaRecByOrderId

type GetFmaRecByOrderId struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *GetFmaRecByOrderId) Register(handler interface {
	Handle(mgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetFmaRecByOrderId) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取配方称重记录列表
var oneFmaWgtRecList OneFmaWgtRecListed

type OneFmaWgtRecListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *OneFmaWgtRecListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u OneFmaWgtRecListed) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取配方称重记录列表
var formulaWgtRecList FormulaWgtRecListed

type FormulaWgtRecListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *FormulaWgtRecListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaWgtRecListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

// 分页与全局排序获取配方称重记录列表
var formulaWgtRecByPage FormulaWgtRecByPageListed

type FormulaWgtRecByPageListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetFormulaRecByPage)
	}
}

func (u *FormulaWgtRecByPageListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetFormulaRecByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u FormulaWgtRecByPageListed) Trigger(srvMgr *SrvMgr, payload ReqGetFormulaRecByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 导出获取所有配方称重记录列表
var getAllFormulaRecForExport GetAllFormulaRecForExportListed

type GetAllFormulaRecForExportListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr, payload ReqGetFormulaRecByPage) }
}

func (u *GetAllFormulaRecForExportListed) Register(handler interface{ Handle(srvMgr *SrvMgr, payload ReqGetFormulaRecByPage) }) {
	u.handlers = append(u.handlers, handler)
}

func (u GetAllFormulaRecForExportListed) Trigger(srvMgr *SrvMgr, payload ReqGetFormulaRecByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 分页获取暂存配方称重记录列表
var draftFormulaRecByPage DraftFormulaRecByPageListed

type DraftFormulaRecByPageListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetDraftFormulaRecByPage)
	}
}

func (u *DraftFormulaRecByPageListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetDraftFormulaRecByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u DraftFormulaRecByPageListed) Trigger(srvMgr *SrvMgr, payload ReqGetDraftFormulaRecByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 导出获取所有暂存配方称重记录列表
var getAllDraftFormulaRecForExport GetAllDraftFormulaRecForExportListed

type GetAllDraftFormulaRecForExportListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetDraftFormulaRecByPage)
	}
}

func (u *GetAllDraftFormulaRecForExportListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetDraftFormulaRecByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u GetAllDraftFormulaRecForExportListed) Trigger(srvMgr *SrvMgr, payload ReqGetDraftFormulaRecByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 分页获取原料列表
var rawMaterialByPage RawMaterialByPageListed

type RawMaterialByPageListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetRawMaterialByPage)
	}
}

func (u *RawMaterialByPageListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetRawMaterialByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u RawMaterialByPageListed) Trigger(srvMgr *SrvMgr, payload ReqGetRawMaterialByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 获取原料轻量字典
var rawMaterialDict RawMaterialDictListed

type RawMaterialDictListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr)
	}
}

func (u *RawMaterialDictListed) Register(handler interface {
	Handle(srvMgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u RawMaterialDictListed) Trigger(srvMgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr)
	}
}

// 导出获取所有原料列表
var getAllRawMaterialsForExport GetAllRawMaterialsForExportListed

type GetAllRawMaterialsForExportListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetRawMaterialByPage)
	}
}

func (u *GetAllRawMaterialsForExportListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetRawMaterialByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u GetAllRawMaterialsForExportListed) Trigger(srvMgr *SrvMgr, payload ReqGetRawMaterialByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 分页获取配方列表
var formulaByPage FormulaByPageListed

type FormulaByPageListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetFormulaByPage)
	}
}

func (u *FormulaByPageListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetFormulaByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u FormulaByPageListed) Trigger(srvMgr *SrvMgr, payload ReqGetFormulaByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 懒加载获取单条配方工序明细
var formulaDetailsByRecId FormulaDetailsByRecIdListed

type FormulaDetailsByRecIdListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload int)
	}
}

func (u *FormulaDetailsByRecIdListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload int)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u FormulaDetailsByRecIdListed) Trigger(srvMgr *SrvMgr, payload int) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 导出获取所有配方列表
var getAllFormulasForExport GetAllFormulasForExportListed

type GetAllFormulasForExportListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetFormulaByPage)
	}
}

func (u *GetAllFormulasForExportListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetFormulaByPage)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u GetAllFormulasForExportListed) Trigger(srvMgr *SrvMgr, payload ReqGetFormulaByPage) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}



// 批量删除配方称重记录
var delFormulaWgtRecBatch DelFormulaWgtRecBatch

type DelFormulaWgtRecBatch struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqDelFormulaWgtRecBatch)
	}
}

func (u *DelFormulaWgtRecBatch) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqDelFormulaWgtRecBatch)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u DelFormulaWgtRecBatch) Trigger(mgr *SrvMgr, payload ReqDelFormulaWgtRecBatch) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 清空所有配方称重记录
var delAllFormulaWgtRec DelAllFormulaWgtRecListed

type DelAllFormulaWgtRecListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr)
	}
}

func (u *DelAllFormulaWgtRecListed) Register(handler interface {
	Handle(srvMgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u DelAllFormulaWgtRecListed) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}


// 根据RecId删除配方
var formulaDeleted FormulaDeleted

type FormulaDeleted struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelFmaData)
	}
}

// Register adds an event handler for this event
func (u *FormulaDeleted) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelFmaData)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaDeleted) Trigger(mgr *SrvMgr, payload ReqDelFmaData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除所有配方
var formulaDeletedAll FormulaDeletedAll

type FormulaDeletedAll struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelAllFmaData)
	}
}

// Register adds an event handler for this event
func (u *FormulaDeletedAll) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelAllFmaData)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaDeletedAll) Trigger(mgr *SrvMgr, payload ReqDelAllFmaData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除所有原料数据
var rawDataDeletedAll RawDataDeletedAll

type RawDataDeletedAll struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelAllRawData)
	}
}

// Register adds an event handler for this event
func (u *RawDataDeletedAll) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelAllRawData)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u RawDataDeletedAll) Trigger(mgr *SrvMgr, payload ReqDelAllRawData) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除所有暂存配方称重记录
var formulaDataDeletedAll FormulaDataDeletedAll

type FormulaDataDeletedAll struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDeleteAllDraftFmaWgtRec)
	}
}

// Register adds an event handler for this event
func (u *FormulaDataDeletedAll) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDeleteAllDraftFmaWgtRec)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FormulaDataDeletedAll) Trigger(mgr *SrvMgr, payload ReqDeleteAllDraftFmaWgtRec) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 增加流速
var flowRateAdded FlowRateAdded

type FlowRateAdded struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqFlowRateRec)
	}
}

// Register adds an event handler for this event
func (u *FlowRateAdded) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqFlowRateRec)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FlowRateAdded) Trigger(mgr *SrvMgr, payload ReqFlowRateRec) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取流速列表
var flowRateList FlowRateListed

type FlowRateListed struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *FlowRateListed) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u FlowRateListed) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

// 获取所有称重记录
var getAllWgtRecList GetAllWgtRecListed

type GetAllWgtRecListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetAllWgtRecList)
	}
}

// Register adds an event handler for this event
func (u *GetAllWgtRecListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetAllWgtRecList)
}) {

	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAllWgtRecListed) Trigger(srvMgr *SrvMgr, payload ReqGetAllWgtRecList) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 获取搜索的称重记录
var getSearchRecList GetSearchRecListed

type GetSearchRecListed struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqGetSearchRecList)
	}
}

// Register adds an event handler for this event
func (u *GetSearchRecListed) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqGetSearchRecList)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetSearchRecListed) Trigger(srvMgr *SrvMgr, payload ReqGetSearchRecList) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

//新增称重记录

var addWgtRec AddWgtRec

type AddWgtRec struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqAddWgtRec)
	}
}

// Register adds an event handler for this event
func (u *AddWgtRec) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqAddWgtRec)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u AddWgtRec) Trigger(srvMgr *SrvMgr, payload ReqAddWgtRec) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 删除记录
var delWgtRec DelWgtRec

type DelWgtRec struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqDelWgtRec)
	}
}

// Register adds an event handler for this event
func (u *DelWgtRec) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqDelWgtRec)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DelWgtRec) Trigger(srvMgr *SrvMgr, payload ReqDelWgtRec) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 根据RecId删除记录
var delWgtRecById DelWgtRecById

type DelWgtRecById struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqDelWgtRecById)
	}
}

// Register adds an event handler for this event
func (u *DelWgtRecById) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqDelWgtRecById)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DelWgtRecById) Trigger(srvMgr *SrvMgr, payload ReqDelWgtRecById) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 导出所有记录
var exportAllRecs ExportAllRecs

type ExportAllRecs struct {
	handlers []interface {
		Handle(srvMgr *SrvMgr, payload ReqExportAllRecs)
	}
}

// Register adds an event handler for this event
func (u *ExportAllRecs) Register(handler interface {
	Handle(srvMgr *SrvMgr, payload ReqExportAllRecs)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ExportAllRecs) Trigger(srvMgr *SrvMgr, payload ReqExportAllRecs) {
	for _, handler := range u.handlers {
		go handler.Handle(srvMgr, payload)
	}
}

// 获取自动下一步设置
var getAutoNext GetAutoNext

type GetAutoNext struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *GetAutoNext) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAutoNext) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

//更新自动下一步

var updateAutoNext UpdateAutoNext

type UpdateAutoNext struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqUpdateAutoNext)
	}
}

// Register adds an event handler for this event
func (u *UpdateAutoNext) Register(handler interface {
	Handle(*SrvMgr, ReqUpdateAutoNext)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateAutoNext) Trigger(mgr *SrvMgr, payload ReqUpdateAutoNext) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 不稳定归零扣重
var getUnstableZeroTare GetUnstableZeroTare

type GetUnstableZeroTare struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *GetUnstableZeroTare) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetUnstableZeroTare) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var updateUnstableZeroTare UpdateUnstableZeroTare

type UpdateUnstableZeroTare struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqUpdateUnstableZeroTare)
	}
}

// Register adds an event handler for this event
func (u *UpdateUnstableZeroTare) Register(handler interface {
	Handle(*SrvMgr, ReqUpdateUnstableZeroTare)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateUnstableZeroTare) Trigger(mgr *SrvMgr, payload ReqUpdateUnstableZeroTare) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

var updateSetReportPrint UpdateSetReportPrint

type UpdateSetReportPrint struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload SetReportPrint)
	}
}

func (u *UpdateSetReportPrint) Register(handler interface {
	Handle(mgr *SrvMgr, payload SetReportPrint)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u UpdateSetReportPrint) Trigger(mgr *SrvMgr, payload SetReportPrint) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取上传服务器信息
var getUploadFmaServer GetUploadFmaServer

type GetUploadFmaServer struct {
	handlers []interface{ Handle(mgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *GetUploadFmaServer) Register(handler interface{ Handle(mgr *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetUploadFmaServer) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 获取所有铅封日志记录
var getAllSealLog GetAllSealLog

type GetAllSealLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload GetSealLogReq)
	}
}

// Register adds an event handler for this event
func (u *GetAllSealLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload GetSealLogReq)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAllSealLog) Trigger(mgr *SrvMgr, payload GetSealLogReq) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

//万能钥匙解封

var unsealByMasterKey UnsealByMasterKey

type UnsealByMasterKey struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload string)
	}
}

// Register adds an event handler for this event
func (u *UnsealByMasterKey) Register(handler interface {
	Handle(mgr *SrvMgr, payload string)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UnsealByMasterKey) Trigger(mgr *SrvMgr, payload string) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 修改上传服务器信息
var editUploadFmaServer EditUploadFmaServer

type EditUploadFmaServer struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload UploadServerInfo)
	}
}

// Register adds an event handler for this event
func (u *EditUploadFmaServer) Register(handler interface {
	Handle(mgr *SrvMgr, payload UploadServerInfo)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u EditUploadFmaServer) Trigger(mgr *SrvMgr, payload UploadServerInfo) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取报表打印设置
var getSetReportPrint GetSetReportPrint

type GetSetReportPrint struct {
	handlers []interface{ Handle(mgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *GetSetReportPrint) Register(handler interface{ Handle(mgr *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetSetReportPrint) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 新增暂存配方记录
var addDraftFmaWgtRec AddDraftFmaWgtRec

type AddDraftFmaWgtRec struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload DrafFmaWgtRecInfo)
	}
}

// Register adds an event handler for this event
func (u *AddDraftFmaWgtRec) Register(handler interface {
	Handle(mgr *SrvMgr, payload DrafFmaWgtRecInfo)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u AddDraftFmaWgtRec) Trigger(mgr *SrvMgr, payload DrafFmaWgtRecInfo) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取暂存配方记录
var getDraftFmaWgtRecList GetDraftFmaWgtRecList

type GetDraftFmaWgtRecList struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *GetDraftFmaWgtRecList) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetDraftFmaWgtRecList) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 删除暂存配方记录
var deleteDraftFmaWgtRec DeleteDraftFmaWgtRec

type DeleteDraftFmaWgtRec struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDeleteDraftFmaWgtRec)
	}
}

// Register adds an event handler for this event
func (u *DeleteDraftFmaWgtRec) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDeleteDraftFmaWgtRec)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DeleteDraftFmaWgtRec) Trigger(mgr *SrvMgr, payload ReqDeleteDraftFmaWgtRec) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 更新暂存配方记录
var updateDraftFmaWgtRec UpdateDraftFmaWgtRec

type UpdateDraftFmaWgtRec struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload DrafFmaWgtRecInfo)
	}
}

// Register adds an event handler for this event
func (u *UpdateDraftFmaWgtRec) Register(handler interface {
	Handle(mgr *SrvMgr, payload DrafFmaWgtRecInfo)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateDraftFmaWgtRec) Trigger(mgr *SrvMgr, payload DrafFmaWgtRecInfo) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 新增系统用户
var addSysUser AddSysUser

type AddSysUser struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddSysUser)
	}
}

// Register adds an event handler for this event
func (u *AddSysUser) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqAddSysUser)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u AddSysUser) Trigger(mgr *SrvMgr, payload ReqAddSysUser) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除系统用户
var deleteSysUser DeleteSysUser

type DeleteSysUser struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqSysUserIdList)
	}
}

// Register adds an event handler for this event
func (u *DeleteSysUser) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqSysUserIdList)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DeleteSysUser) Trigger(mgr *SrvMgr, payload ReqSysUserIdList) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 更新系统用户
var updateSysUser UpdateSysUser

type UpdateSysUser struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqUpdateSysUser)
	}
}

// Register adds an event handler for this event
func (u *UpdateSysUser) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqUpdateSysUser)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateSysUser) Trigger(mgr *SrvMgr, payload ReqUpdateSysUser) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 禁用系统用户
var disableSysUser DisableSysUser

type DisableSysUser struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqEnabledSysUserId)
	}
}

// Register adds an event handler for this event
func (u *DisableSysUser) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqEnabledSysUserId)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u DisableSysUser) Trigger(mgr *SrvMgr, payload ReqEnabledSysUserId) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 密码修改
var changePassword ChangePassword

type ChangePassword struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqChangePassword)
	}
}

// Register adds an event handler for this event
func (u *ChangePassword) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqChangePassword)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ChangePassword) Trigger(mgr *SrvMgr, payload ReqChangePassword) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 登录
var login Login

type Login struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqLogin)
	}
}

// Register adds an event handler for this event
func (u *Login) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqLogin)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u Login) Trigger(mgr *SrvMgr, payload ReqLogin) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// RFID 刷卡登录
var rfidLogin RfidLogin

type RfidLogin struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqRfidLogin)
	}
}

func (u *RfidLogin) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqRfidLogin)
}) {
	u.handlers = append(u.handlers, handler)
}

func (u RfidLogin) Trigger(mgr *SrvMgr, payload ReqRfidLogin) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 登出
var logout Logout

type Logout struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *Logout) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u Logout) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 获取所有用户
var getAllUsers GetAllUsers

type GetAllUsers struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

// Register adds an event handler for this event
func (u *GetAllUsers) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {

	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAllUsers) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 获取用户详情
var getUserDetail GetUserDetail

type GetUserDetail struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqSysUserName)
	}
}

// Register adds an event handler for this event
func (u *GetUserDetail) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqSysUserName)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetUserDetail) Trigger(mgr *SrvMgr, payload ReqSysUserName) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 新增系统日志记录
var addSysLog AddSysLog

type AddSysLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddSysLog)
	}
}

// Register adds an event handler for this event
func (u *AddSysLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqAddSysLog)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u AddSysLog) Trigger(mgr *SrvMgr, payload ReqAddSysLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 新增称重日志记录
var addScaleLog AddScaleLog

type AddScaleLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqAddScaleLog)
	}
}

// Register adds an event handler for this event
func (u *AddScaleLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqAddScaleLog)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u AddScaleLog) Trigger(mgr *SrvMgr, payload ReqAddScaleLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取系统日志列表
var getAllSysLog GetAllSysLog

type GetAllSysLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqGetLog)
	}
}

// Register adds an event handler for this event
func (u *GetAllSysLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqGetLog)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAllSysLog) Trigger(mgr *SrvMgr, payload ReqGetLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取校准记录列表
var getAllCalLog GetAllCalLog

type GetAllCalLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqGetLog)
	}
}

// Register adds an event handler for this event
func (u *GetAllCalLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqGetLog)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAllCalLog) Trigger(mgr *SrvMgr, payload ReqGetLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 打开输出端口
var openOutputPort OpenOutputPort

type OpenOutputPort struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqPortInfo)
	}
}

// Register adds an event handler for this event

func (u *OpenOutputPort) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqPortInfo)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u OpenOutputPort) Trigger(mgr *SrvMgr, payload ReqPortInfo) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// readOutputPort
// 读取输出端口
var readOutputPort ReadOutputPort

type ReadOutputPort struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqPortInfo)
	}
}

// Register adds an event handler for this event
func (u *ReadOutputPort) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqPortInfo)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u ReadOutputPort) Trigger(mgr *SrvMgr, payload ReqPortInfo) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取称重日志列表
var getAllScaleLog GetAllScaleLog

type GetAllScaleLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqGetLog)
	}
}

// Register adds an event handler for this event
func (u *GetAllScaleLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqGetLog)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetAllScaleLog) Trigger(mgr *SrvMgr, payload ReqGetLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除所有系统日志
var delAllSysLog DelAllSysLog

type DelAllSysLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

func (u *DelAllSysLog) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u DelAllSysLog) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 删除所有校准记录
var delAllCalLog DelAllCalLog

type DelAllCalLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

func (u *DelAllCalLog) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u DelAllCalLog) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 删除所有称重记录
var delAllScaleLog DelAllScaleLog

type DelAllScaleLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr)
	}
}

func (u *DelAllScaleLog) Register(handler interface {
	Handle(mgr *SrvMgr)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u DelAllScaleLog) Trigger(mgr *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr)
	}
}

// 删除多条系统日志
var delMultiSysLog DelMultiSysLog

type DelMultiSysLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelLogs)
	}
}

func (u *DelMultiSysLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelLogs)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u DelMultiSysLog) Trigger(mgr *SrvMgr, payload ReqDelLogs) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除多条校准记录
var delMultiCalLog DelMultiCalLog

type DelMultiCalLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelLogs)
	}
}

func (u *DelMultiCalLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelLogs)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u DelMultiCalLog) Trigger(mgr *SrvMgr, payload ReqDelLogs) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 删除多条称重日志
var delMultiScaleLog DelMultiScaleLog

type DelMultiScaleLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqDelLogs)
	}
}

func (u *DelMultiScaleLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqDelLogs)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u DelMultiScaleLog) Trigger(mgr *SrvMgr, payload ReqDelLogs) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 导出系统日志
var exportSysLog ExportSysLog

type ExportSysLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqExportLog)
	}
}

func (u *ExportSysLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqExportLog)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u ExportSysLog) Trigger(mgr *SrvMgr, payload ReqExportLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 导出校准记录
var exportCalLog ExportCalLog

type ExportCalLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqExportLog)
	}
}

func (u *ExportCalLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqExportLog)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u ExportCalLog) Trigger(mgr *SrvMgr, payload ReqExportLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 导出称重记录
var exportScaleLog ExportScaleLog

type ExportScaleLog struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload ReqExportLog)
	}
}

func (u *ExportScaleLog) Register(handler interface {
	Handle(mgr *SrvMgr, payload ReqExportLog)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u ExportScaleLog) Trigger(mgr *SrvMgr, payload ReqExportLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 增加标定记录
var addCalRecord AddCalRecord

type AddCalRecord struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload CalibrationLog)
	}
}

func (u *AddCalRecord) Register(handler interface {
	Handle(mgr *SrvMgr, payload CalibrationLog)
}) {
	u.handlers = append(u.handlers, handler)
}
func (u AddCalRecord) Trigger(mgr *SrvMgr, payload CalibrationLog) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取自动下一步设置
var getOutputPort GetOutputPort

type GetOutputPort struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *GetOutputPort) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetOutputPort) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

//更新自动下一步

var updateOutputPort UpdateOutputPort

type UpdateOutputPort struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload []ReqUpdateOutputPort)
	}
}

// Register adds an event handler for this event
func (u *UpdateOutputPort) Register(handler interface {
	Handle(*SrvMgr, []ReqUpdateOutputPort)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateOutputPort) Trigger(mgr *SrvMgr, payload []ReqUpdateOutputPort) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// 获取输入端口状态
var getInputPort GetInputPort

type GetInputPort struct {
	handlers []interface{ Handle(srvMgr *SrvMgr) }
}

// Register adds an event handler for this event
func (u *GetInputPort) Register(handler interface{ Handle(payload *SrvMgr) }) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u GetInputPort) Trigger(payload *SrvMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

// 更新输入端口状态
var updateInputPort UpdateInputPort

type UpdateInputPort struct {
	handlers []interface {
		Handle(mgr *SrvMgr, payload []ReqUpdateInputPort)
	}
}

// Register adds an event handler for this event
func (u *UpdateInputPort) Register(handler interface {
	Handle(*SrvMgr, []ReqUpdateInputPort)
}) {
	u.handlers = append(u.handlers, handler)
}

// Trigger sends out an event with the payload
func (u UpdateInputPort) Trigger(mgr *SrvMgr, payload []ReqUpdateInputPort) {
	for _, handler := range u.handlers {
		go handler.Handle(mgr, payload)
	}
}

// Modbus Events
var getModbusServices GetModbusServices

type GetModbusServices struct {
	handlers []interface{ Handle(scaleMgr *ScaleMgr) }
}

func (u *GetModbusServices) Register(handler interface{ Handle(payload *ScaleMgr) }) {
	u.handlers = append(u.handlers, handler)
}

func (u GetModbusServices) Trigger(payload *ScaleMgr) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var addModbusService AddModbusService

type AddModbusService struct {
	handlers []interface{ Handle(payload ReqAddModbusService) }
}

func (u *AddModbusService) Register(handler interface{ Handle(payload ReqAddModbusService) }) {
	u.handlers = append(u.handlers, handler)
}

func (u AddModbusService) Trigger(payload ReqAddModbusService) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var editModbusService EditModbusService

type EditModbusService struct {
	handlers []interface{ Handle(payload ReqEditModbusService) }
}

func (u *EditModbusService) Register(handler interface{ Handle(payload ReqEditModbusService) }) {
	u.handlers = append(u.handlers, handler)
}

func (u EditModbusService) Trigger(payload ReqEditModbusService) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}

var delModbusService DelModbusService

type DelModbusService struct {
	handlers []interface{ Handle(payload ReqDelModbusService) }
}

func (u *DelModbusService) Register(handler interface{ Handle(payload ReqDelModbusService) }) {
	u.handlers = append(u.handlers, handler)
}

func (u DelModbusService) Trigger(payload ReqDelModbusService) {
	for _, handler := range u.handlers {
		go handler.Handle(payload)
	}
}
