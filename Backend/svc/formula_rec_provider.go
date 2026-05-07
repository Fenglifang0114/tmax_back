package svc

import (
	"path/filepath"
	"tmaxsrv/comm"
)

type FormulaRecProvider struct {
	myId   string
	infoPb *DbFormulaInfo
}

func NewFormulaRecProvider() *FormulaRecProvider {
	database := filepath.Join(comm.GetSrvDataPath(), "formulainfo.db")
	infoPb, _ := NewFormulaInfo(database)
	return &FormulaRecProvider{myId: "FormulaRecProvider", infoPb: infoPb}
}

// 获取所有配方列表
func (p *FormulaRecProvider) GetFormulaRecsList() ([]FormulaList, error) {
	recs, err := p.infoPb.GetAllFormulaLists()
	return recs, err
}

// 获取所有重量模式配方记录列表
func (p *FormulaRecProvider) GetPluPath() ([]FormulaWgtRecList, error) {
	recs, err := p.infoPb.GetAllFormulaWgtRecLists()
	return recs, err
}

// 新增原料类别
func (p *FormulaRecProvider) InsertRawType(rec RawMaterialCategory) error {
	return p.infoPb.CreateRawMaterialCategory(rec)
}

// 新增原料类别列表
func (p *FormulaRecProvider) InsertRawTypeList(rec []string) error {
	return p.infoPb.CreateRawCategoryList(rec)
}

// 新增配方类别列表
func (p *FormulaRecProvider) InsertFormulaTypeList(rec []string) error {
	return p.infoPb.CreateFormulaCategoryList(rec)
}

// 新增原料信息
func (p *FormulaRecProvider) InsertRawInfo(rec RawMaterial) error {
	return p.infoPb.CreateRawMaterial(rec)
}

// 获取最大的原料ID
func (p *FormulaRecProvider) GetMaxRawRecId() (int, error) {
	return p.infoPb.GetMaxRawRecId()
}

// 根据FMAID获取原料数据的输出口
func (p *FormulaRecProvider) GetRawOutputByFmaId(fmaId string) ([]RawMaterialOutput, error) {
	return p.infoPb.GetRawOutputByFmaId(fmaId)
}

// 新增原料信息列表
func (p *FormulaRecProvider) InsertRawInfoList(rec []RawMaterial) error {
	return p.infoPb.CreateRawMaterialList(rec)
}

// 新增配方类别
func (p *FormulaRecProvider) InsertFormulaType(rec FormulaCategory) error {
	return p.infoPb.CreateFormulaCategory(rec)
}

// 删除未使用的原料类别
func (p *FormulaRecProvider) DeleteUnusedRawType() error {
	return p.infoPb.DeleteUnusedRawMaterialCategories()
}
func (p *FormulaRecProvider) DeleteUnusedFormulaType() error {
	return p.infoPb.DeleteUnusedFormulaCategories()
}

// 获取原料类别列表
func (p *FormulaRecProvider) GetRawTypeList() ([]RawMaterialCategory, error) {
	recs, err := p.infoPb.GetAllRawMaterialCategories()
	return recs, err
}

// 获取配方类别列表
func (p *FormulaRecProvider) GetFormulaTypeList() ([]FormulaCategory, error) {
	recs, err := p.infoPb.GetAllFormulaCategories()
	return recs, err
}

// 获取原料列表
func (p *FormulaRecProvider) GetRawDataList() ([]RawMaterial, error) {
	recs, err := p.infoPb.GetAllRawMaterials()
	return recs, err
}

// 修改原料信息
func (p *FormulaRecProvider) UpdateRawInfo(rec RawMaterial) error {
	return p.infoPb.UpdateRawMaterial(rec)
}

// 删除原料信息
func (p *FormulaRecProvider) DeleteRawInfo(recId int) error {
	return p.infoPb.DeleteRawMaterial(recId)
}

// 删除所有原料信息
func (p *FormulaRecProvider) DeleteAllRawInfo(recIds []int) error {
	return p.infoPb.DeleteAllRawMaterials(recIds)
}

// 获取配方信息头ID
func (p *FormulaRecProvider) GetMaxFormulaRecId() (int, error) {
	return p.infoPb.GetMaxFormulaRecId()
}

// 获取配方信息头key
func (p *FormulaRecProvider) GetMaxFormulaRecKey() (int, error) {
	return p.infoPb.GetMaxFormulaRecKey()
}

// 新增配方信息头
func (p *FormulaRecProvider) InsertFormulaHeader(rec FormulaHeader) error {
	return p.infoPb.CreateFormulaHeader(rec)
}

// 新增配方信息体
func (p *FormulaRecProvider) InsertFormulaBody(rec FormulaDetail) error {
	return p.infoPb.CreateFormulaDetail(rec)
}

// 获取配方信息列表
func (p *FormulaRecProvider) GetFormulaDataList() ([]FormulaList, error) {
	return p.infoPb.GetAllFormulaLists()
}

// 通过条码获取配方数据
func (p *FormulaRecProvider) GetFormulaDataByBarcode(barcode string) ([]FormulaList, error) {
	return p.infoPb.GetFormulaDataByBarcode(barcode)
}

// 检查配方ID和条码是否匹配
func (p *FormulaRecProvider) CheckFmaIdAndBarcode(recId int, formulaID string, formulaBarcode string) (bool, bool, error) {
	return p.infoPb.CheckFmaIdAndBarcode(recId, formulaID, formulaBarcode)
}

// 获取单个配方信息
func (p *FormulaRecProvider) GetFormulaData(recId int) ([]FormulaList, error) {

	return p.infoPb.GetFormulaData(recId)
}

// 获取单个原料信息
func (p *FormulaRecProvider) GetRawData(recId int) ([]RawMaterial, error) {

	return p.infoPb.GetRawMaterialByID(recId)
}

// 根据配方编号获取配方
func (p *FormulaRecProvider) GetFormulaListByFormulaID(formulaID string) (FormulaList, error) {
	recs, err := p.infoPb.GetFormulaListByFormulaID(formulaID)
	return recs, err
}

// 根据recId获取配方
func (p *FormulaRecProvider) GetFormulaByRecId(recId int) (FormulaList, error) {
	return p.infoPb.GetFormulaByRecId(recId)
}

// 新增配方称重记录头
func (p *FormulaRecProvider) InsertFormulaWgtHeader(rec FormulaWgtRecHeader) error {
	return p.infoPb.CreateFormulaWgtRecHeader(rec)
}

// 新增配方称重记录体
func (p *FormulaRecProvider) InsertFormulaWgtBody(rec FormulaWgtRecDetail) error {
	return p.infoPb.CreateFormulaWgtRecDetail(rec)
}

// 获取配方称重记录列表
func (p *FormulaRecProvider) GetFormulaWgtRecList() ([]FormulaWgtRecList, error) {
	recs, err := p.infoPb.GetAllFormulaWgtRecLists()
	return recs, err
}

// 获取配方称重记录列表
func (p *FormulaRecProvider) GetOneFormulaWgtRecList(fmaId string) ([]FormulaWgtRecList, error) {
	recs, err := p.infoPb.GetOneFormulaWgtRecLists(fmaId)
	return recs, err
}

// 获取配方称重记录
func (p *FormulaRecProvider) GetFmaWgtRecByOrderId(orderId string) (FormulaWgtRecList, error) {
	return p.infoPb.GetFmaWgtRecByOrderId(orderId)
}

// 删除配方
func (p *FormulaRecProvider) DeleteFormula(rec_id int) error {
	return p.infoPb.DeleteFormulaByRecId(rec_id)
}

// 删除所有配方
func (p *FormulaRecProvider) DeleteAllFormulas(rec_ids []int) error {
	return p.infoPb.DeleteAllFormulaByRecId(rec_ids)
}

// 修改配方
func (p *FormulaRecProvider) UpdateFormula(header FormulaHeader, details []FormulaDetail) error {
	return p.infoPb.UpdateFormula(header, details)

}

// 删除原料类型
func (p *FormulaRecProvider) DeleteRawType(recId string) error {
	return p.infoPb.DeleteRawMaterialCategory(recId)
}

// 获取原料类型ByID
func (p *FormulaRecProvider) GetRawTypeByID(recId int) (RawMaterialCategory, error) {
	return p.infoPb.GetRawMaterialCategoryByID(recId)
}

// 获取配方类型ByID
func (p *FormulaRecProvider) GetFormulaCategoryByID(recId int) (FormulaCategory, error) {
	return p.infoPb.GetFormulaCategoryByID(recId)
}

// 修改原料类型
func (p *FormulaRecProvider) UpdateRawType(rec RawMaterialCategory) error {
	return p.infoPb.UpdateRawMaterialCategory(rec)
}

// 新增配方
func (p *FormulaRecProvider) InsertFmaInfoList(rec []FmaDataImportInfo) error {
	return p.infoPb.InsertFormulaList(rec)
}

// 删除配方类型
func (p *FormulaRecProvider) DeleteFmaType(name string) error {
	return p.infoPb.DeleteFormulaCategory(name)

}

// 修改配方类型
func (p *FormulaRecProvider) UpdateFmaType(rec FormulaCategory) error {
	return p.infoPb.UpdateFormulaCategory(rec)
}

// 获取配方秤中的自动下一步设置
func (p *FormulaRecProvider) GetSetAutoNext() (*SetAutoNext, error) {
	return p.infoPb.GetSetAutoNext()
}

// 更新配方秤中的自动下一步设置
func (p *FormulaRecProvider) UpdateSetAutoNext(rec SetAutoNext) error {
	return p.infoPb.UpdateSetAutoNext(rec.AutoNext, rec.StableTime, rec.AutoTare, rec.CheckCode)
}

// 获取不稳定归零扣重设置
func (p *FormulaRecProvider) GetUnstableZeroTare() (bool, error) {
	return p.infoPb.GetUnstableZeroTare()
}

// 更新不稳定归零扣重设置
func (p *FormulaRecProvider) UpdateUnstableZeroTare(enable bool) error {
	return p.infoPb.UpdateUnstableZeroTare(enable)
}

// 创建暂存配方称重记录
func (p *FormulaRecProvider) CreateDraftFmaWgtRecHeader(rec DrafFmaWgtRecHeader) error {
	return p.infoPb.CreateDraftFmaWgtRecHeader(rec)
}

// 创建暂存配方称重记录体
func (p *FormulaRecProvider) CreateDraftFmaWgtRecDetail(rec DrafFmaWgtRecDetail) error {
	return p.infoPb.CreateDraftFmaWgtRecDetail(rec)
}

// 更新暂存配方称重记录
func (p *FormulaRecProvider) UpdateDraftFmaWgtRec(rec DrafFmaWgtRecInfo) error {
	return p.infoPb.UpdateDraftFmaWgtRec(rec)
}

// 获取暂存配方称重记录
func (p *FormulaRecProvider) GetDraftFmaWgtRec() ([]DrafFmaWgtRecInfo, error) {
	return p.infoPb.GetAllDraftFmaWgtRecLists()

}

// 获取暂存配方称重记录ByOrderId
func (p *FormulaRecProvider) GetDraftFmaWgtRecByOrderId(orderId []string) ([]DrafFmaWgtRecInfo, error) {
	return p.infoPb.GetDraftFmaWgtRecByOrderId(orderId)
}

// 删除暂存配方称重记录
func (p *FormulaRecProvider) DeleteDraftFmaWgtRec(orderId string) error {
	return p.infoPb.DeleteDraftFmaWgtRec(orderId)
}

// 删除所有暂存配方称重记录
func (p *FormulaRecProvider) DeleteAllDraftFmaWgtRec(orderIds []string) error {
	return p.infoPb.DeleteAllDraftFmaWgtRec(orderIds)
}

// 检查配方材料里面是否使用了这个秤
func (p *FormulaRecProvider) CheckFormulaRawData(scaleId int) (bool, error) {
	return p.infoPb.CheckFormulaRawData(scaleId)
}

// 获取原料名ByID
func (p *FormulaRecProvider) GetRawDataByRawID(radId string) (RawMaterial, error) {
	return p.infoPb.GetRawDataByRawID(radId)
}

// 修改打印报表字段设置
func (p *FormulaRecProvider) UpdateSetReportPrint(rec SetReportPrint) error {
	return p.infoPb.UpdateSetReportPrint(rec)
}

// 获取打印报表字段设置
func (p *FormulaRecProvider) GetSetReportPrint() (*SetReportPrint, error) {
	return p.infoPb.GetSetReportPrint()
}

// 获取上传服务器信息
func (p *FormulaRecProvider) GetUploadServerInfo() (UploadServerInfo, error) {
	return p.infoPb.GetUploadServerInfo()
}

// 更新上传服务器信息
func (p *FormulaRecProvider) UpdateUploadServerInfo(rec UploadServerInfo) error {
	return p.infoPb.UpdateUploadServerInfo(rec)
}

// 创建上传服务器信息
func (p *FormulaRecProvider) CreateUploadServerInfo(rec UploadServerInfo) error {
	return p.infoPb.CreateUploadServerInfo(rec)
}

// 更新配方秤中的输出设置
func (p *FormulaRecProvider) UpdateSetOutput(recs []SetOutputPort) error {
	return p.infoPb.UpdateSetOutputPort(recs)
}

// 获取配方秤中的输出设置
func (p *FormulaRecProvider) GetSetOutput() ([]SetOutputPort, error) {
	return p.infoPb.GetSetOutputPort()
}

// 获取配方秤中的输入设置
func (p *FormulaRecProvider) GetSetInput() ([]SetInputPort, error) {
	return p.infoPb.GetSetInputPort()
}

// 更新配方秤中的输入设置
func (p *FormulaRecProvider) UpdateSetInput(recs []SetInputPort) error {
	return p.infoPb.UpdateSetInputPort(recs)
}
