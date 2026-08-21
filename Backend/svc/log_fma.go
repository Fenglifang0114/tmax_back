package svc

type UpdateTypeInfo struct {
	NewType string
	OldType string
}

type TypeName struct {
	Type string
}

type SaveAddRawData struct {
	ID         string
	Name       string
	Category   string
	Ingredient string
	ScaleName  string
	CheckCode  string
}

func GetScaleName(id int) string {
	return mSrvMgr.scaleMgr.connPb.GetScaleNameById(int64(id))
}

// 保存原料数据新增日志
func SaveFmaRawDataLog(payload ReqAddRawData) {
	rawType, _ := mSrvMgr.formulaPd.GetRawTypeByID(payload.CategoryID)
	scaleName := GetScaleName(payload.ScaleId)
	saveData := SaveAddRawData{
		ID:         payload.MaterialID,
		Name:       payload.MaterialName,
		Category:   rawType.CategoryName,
		Ingredient: payload.Ingredient,
		ScaleName:  scaleName,
		CheckCode:  payload.CheckCode,
	}
	jsonStr, _ := json.MarshalToString(saveData)
	LogSysOperation(MenuFormulaManage, SubFmaRawAdd, OpAddStr, jsonStr, "ok", "")
}

// 保存原料数据更新日志
func SaveUpdateFmaRawDataLog(payload ReqEditRawData, data []RawMaterial) {
	if len(data) <= 0 {
		return
	}
	oldRaw := data[0]
	oldRawType, _ := mSrvMgr.formulaPd.GetRawTypeByID(oldRaw.CategoryID)
	newRawType, _ := mSrvMgr.formulaPd.GetRawTypeByID(payload.CategoryID)
	updateFields := make(map[string]interface{})
	originalFields := make(map[string]interface{})

	if oldRaw.MaterialID != payload.MaterialID {
		updateFields["ID"] = payload.MaterialID
		originalFields["ID"] = oldRaw.MaterialID
	} else {
		originalFields["ID"] = oldRaw.MaterialID
	}
	if oldRaw.MaterialName != payload.MaterialName {
		updateFields["Name"] = payload.MaterialName
		originalFields["Name"] = oldRaw.MaterialName
	} else {
		originalFields["Name"] = oldRaw.MaterialName
	}
	if oldRawType.CategoryName != newRawType.CategoryName {
		updateFields["Category"] = newRawType.CategoryName
		originalFields["Category"] = oldRawType.CategoryName
	}
	if oldRaw.Ingredient != payload.Ingredient {
		updateFields["Ingredient"] = payload.Ingredient
		originalFields["Ingredient"] = oldRaw.Ingredient
	}
	if oldRaw.ScaleId != payload.ScaleId {
		updateFields["ScaleName"] = GetScaleName(payload.ScaleId)
		originalFields["ScaleName"] = GetScaleName(oldRaw.ScaleId)
	}

	if oldRaw.CheckCode != payload.CheckCode {
		updateFields["CheckCode"] = payload.CheckCode
		originalFields["CheckCode"] = oldRaw.CheckCode
	}

	if len(updateFields) <= 0 {
		return
	}

	saveData := UpdateLog{
		UpdatedFields:  updateFields,
		OriginalFields: originalFields,
	}
	jsonStr, _ := json.MarshalToString(saveData)
	LogSysOperation(MenuFormulaManage, SubFmaRawUpdate, OpUpdateStr, jsonStr, "ok", "")

}

type SaveAddFormulaData struct {
	Header FmaHeader
	Detail []FmaDetail
}

// 配方头表
type FmaHeader struct {
	ID   string
	Name string
	// 配方类别
	Category string
	// 配方模式
	Mode string
	// 配方单位
	Unit string
	// 配方总重量
	TotalWeight float64
	// 原料数量
	MaterialCount int
	// 是否加密
	IsEncrypted bool
	//是否需要容器
	NeedContainer bool
	// 备注
	Remark string
}

// 配方明细表
type FmaDetail struct {
	Sequence       int
	ID             string
	Name           string
	WeightOrPct    float64
	AllowableError float64
}

func getFmaWgtMode(mode string) string {
	if mode == "wgt" {
		return "weight"
	}
	return "percentage"
}

func getFmaUnit(unit string, mode string) string {

	if mode == "wgt" {
		return unit
	}
	return ""
}

// 保存配方数据新增日志
func SaveFmaDataAddLog(payload ReqAddFormulaData) {
	formulaType, _ := mSrvMgr.formulaPd.GetFormulaCategoryByID(payload.Header.CategoryID)

	header := FmaHeader{
		ID:            payload.Header.FormulaID,
		Name:          payload.Header.FormulaName,
		Category:      formulaType.CategoryName,
		Mode:          getFmaWgtMode(payload.Header.FormulaMode),
		Unit:          getFmaUnit(payload.Header.FormulaUnit, payload.Header.FormulaMode),
		TotalWeight:   payload.Header.TotalWeight,
		MaterialCount: payload.Header.MaterialCount,
		IsEncrypted:   payload.Header.IsEncrypted,
		NeedContainer: payload.Header.NeedContainer,
		Remark:        payload.Header.Remark,
	}
	saveData := SaveAddFormulaData{Header: header}
	for _, raw := range payload.Detail {
		rawInfo, _ := mSrvMgr.formulaPd.GetRawDataByRawID(raw.MaterialID)
		saveData.Detail = append(saveData.Detail, FmaDetail{
			ID:             raw.MaterialID,
			Name:           rawInfo.MaterialName,
			WeightOrPct:    raw.MaterialWeight,
			Sequence:       raw.Sequence,
			AllowableError: raw.AllowableError,
		})
	}

	jsonStr, _ := json.MarshalToString(saveData)
	LogSysOperation(MenuFormulaManage, SubFormulaAdd, OpAddStr, jsonStr, "ok", "")
}

func getOldFmaDetails(oldDetail []FormulaDetail) []FmaDetail {
	details := []FmaDetail{}
	for _, raw := range oldDetail {
		rawInfo, _ := mSrvMgr.formulaPd.GetRawDataByRawID(raw.MaterialID)
		details = append(details, FmaDetail{
			ID:             raw.MaterialID,
			Name:           rawInfo.MaterialName,
			WeightOrPct:    raw.MaterialWeight,
			Sequence:       raw.Sequence,
			AllowableError: raw.AllowableError,
		})
	}
	return details
}

func getNewFmaDetails(newDetail []ReqAddFormulaDetail) []FmaDetail {
	details := []FmaDetail{}
	for _, raw := range newDetail {
		rawInfo, _ := mSrvMgr.formulaPd.GetRawDataByRawID(raw.MaterialID)
		details = append(details, FmaDetail{
			ID:             raw.MaterialID,
			Name:           rawInfo.MaterialName,
			WeightOrPct:    raw.MaterialWeight,
			Sequence:       raw.Sequence,
			AllowableError: raw.AllowableError,
		})
	}
	return details
}

func SaveUpdateFmaDataLog(newData ReqAddFormulaData, oldData FormulaList) {
	if oldData.Header.RecId <= 0 {
		return
	}
	isDetailUpdate := false

	//判断字段是否有更新
	updateFields := make(map[string]interface{})
	originalFields := make(map[string]interface{})

	if newData.Header.FormulaID != oldData.Header.FormulaID {
		updateFields["ID"] = newData.Header.FormulaID
		originalFields["ID"] = oldData.Header.FormulaID
	} else {
		originalFields["ID"] = oldData.Header.FormulaID
	}
	if newData.Header.FormulaName != oldData.Header.FormulaName {
		updateFields["Name"] = newData.Header.FormulaName
		originalFields["Name"] = oldData.Header.FormulaName
	} else {
		originalFields["Name"] = oldData.Header.FormulaName
	}
	// 配方类别
	if newData.Header.CategoryID != oldData.Header.CategoryID {
		newCategory, _ := mSrvMgr.formulaPd.GetFormulaCategoryByID(newData.Header.CategoryID)
		oldCategory, _ := mSrvMgr.formulaPd.GetFormulaCategoryByID(oldData.Header.CategoryID)
		updateFields["Category"] = newCategory.CategoryName
		originalFields["Category"] = oldCategory.CategoryName
	}
	if newData.Header.FormulaMode != oldData.Header.FormulaMode {
		updateFields["Mode"] = getFmaWgtMode(newData.Header.FormulaMode)
		originalFields["Mode"] = getFmaWgtMode(oldData.Header.FormulaMode)
		isDetailUpdate = true
	}
	if newData.Header.FormulaUnit != oldData.Header.FormulaUnit {
		updateFields["Unit"] = getFmaUnit(newData.Header.FormulaUnit, newData.Header.FormulaMode)
		originalFields["Unit"] = getFmaUnit(oldData.Header.FormulaUnit, oldData.Header.FormulaMode)
	}
	if newData.Header.TotalWeight != oldData.Header.TotalWeight {
		updateFields["TotalWeight"] = newData.Header.TotalWeight
		originalFields["TotalWeight"] = oldData.Header.TotalWeight
		isDetailUpdate = true
	}
	// 原料数量
	if newData.Header.MaterialCount != oldData.Header.MaterialCount {
		updateFields["Count"] = newData.Header.MaterialCount
		originalFields["Count"] = oldData.Header.MaterialCount
		isDetailUpdate = true
	}
	// 是否加密
	if newData.Header.IsEncrypted != oldData.Header.IsEncrypted {
		updateFields["IsEncrypted"] = newData.Header.IsEncrypted
		originalFields["IsEncrypted"] = oldData.Header.IsEncrypted
	}
	// 是否需要容器
	if newData.Header.NeedContainer != oldData.Header.NeedContainer {
		updateFields["NeedContainer"] = newData.Header.NeedContainer
		originalFields["NeedContainer"] = oldData.Header.NeedContainer
	}
	// 备注
	if newData.Header.Remark != oldData.Header.Remark {
		updateFields["Remark"] = newData.Header.Remark
		originalFields["Remark"] = oldData.Header.Remark
	}
	//配方条码
	if newData.Header.FormulaBarcode != oldData.Header.FormulaBarcode {
		updateFields["Barcode"] = newData.Header.FormulaBarcode
		originalFields["Barcode"] = oldData.Header.FormulaBarcode
	}

	oldDetails := getOldFmaDetails(oldData.Details)
	newDetails := getNewFmaDetails(newData.Detail)

	if !isDetailUpdate {
		if len(newDetails) != len(oldDetails) {
			isDetailUpdate = true
		}
		for i := range newDetails {
			if newDetails[i].ID != oldDetails[i].ID ||
				newDetails[i].Name != oldDetails[i].Name ||
				newDetails[i].WeightOrPct != oldDetails[i].WeightOrPct ||
				newDetails[i].Sequence != oldDetails[i].Sequence ||
				newDetails[i].AllowableError != oldDetails[i].AllowableError {
				isDetailUpdate = true
			}
		}
	}

	if isDetailUpdate {
		updateFields["Detail"] = newDetails
		originalFields["Detail"] = oldDetails
	}
	if len(updateFields) <= 0 {
		return
	}
	updateLog := UpdateLog{
		UpdatedFields:  updateFields,
		OriginalFields: originalFields,
	}

	jsonStr, _ := json.MarshalToString(updateLog)
	LogSysOperation(MenuFormulaManage, SubFormulaUpdate, OpUpdateStr, jsonStr, "ok", "")
}

func SaveDeleteFormulaLog(oldData FormulaList) {
	if oldData.Header.RecId <= 0 {
		return
	}
	formulaType, _ := mSrvMgr.formulaPd.GetFormulaCategoryByID(oldData.Header.CategoryID)
	header := FmaHeader{
		ID:            oldData.Header.FormulaID,
		Name:          oldData.Header.FormulaName,
		Category:      formulaType.CategoryName,
		Mode:          getFmaWgtMode(oldData.Header.FormulaMode),
		Unit:          getFmaUnit(oldData.Header.FormulaUnit, oldData.Header.FormulaMode),
		TotalWeight:   oldData.Header.TotalWeight,
		MaterialCount: oldData.Header.MaterialCount,
		IsEncrypted:   oldData.Header.IsEncrypted,
		NeedContainer: oldData.Header.NeedContainer,
		Remark:        oldData.Header.Remark,
	}
	saveData := SaveAddFormulaData{Header: header}
	saveData.Detail = getOldFmaDetails(oldData.Details)
	jsonStr, _ := json.MarshalToString(saveData)
	LogSysOperation(MenuFormulaManage, SubFormulaDel, OpDeleteStr, jsonStr, "ok", "")

}

type SaveAddDarftFmaData struct {
	Header AddDraftFmaHeader
	Detail []AddDraftDetail
}

// 暂存配方的表头
type AddDraftFmaHeader struct {
	FormulaID   string
	FormulaName string
	OrderId     string
}

// 暂存配方的明细
type AddDraftDetail struct {
	Sequence         int
	ID               string
	Name             string
	ActualWeight     float64
	ActualWeightUnit string
	IsContainer      bool
	ScaleId          int
	ScaleName        string
	ScaleModel       string
	ScaleSn          string
}

func getSaveDarftFma(tempDraft DrafFmaWgtRecInfo) SaveAddDarftFmaData {
	formula, _ := mSrvMgr.formulaPd.GetFormulaListByFormulaID(tempDraft.Header.FormulaID)
	header := AddDraftFmaHeader{
		FormulaID:   tempDraft.Header.FormulaID,
		FormulaName: formula.Header.FormulaName,
		OrderId:     tempDraft.Header.OrderId,
	}
	addDraftDetails := []AddDraftDetail{}
	for _, detail := range tempDraft.Details {
		rawInfo, _ := mSrvMgr.formulaPd.GetRawDataByRawID(detail.RawMaterialID)
		addDraftDetails = append(addDraftDetails, AddDraftDetail{
			Sequence:         detail.Seq,
			ID:               detail.RawMaterialID,
			Name:             rawInfo.MaterialName,
			ActualWeight:     detail.ActualWeight,
			ActualWeightUnit: detail.ActualWeightUnit,
			IsContainer:      detail.IsContainer,
			ScaleId:          detail.ScaleId,
			ScaleName:        detail.ScaleName,
			ScaleModel:       detail.ScaleModel,
			ScaleSn:          detail.ScaleSn,
		})
	}
	saveData := SaveAddDarftFmaData{Header: header, Detail: addDraftDetails}
	return saveData
}

func SaveFmaDarftAddLog(payload DrafFmaWgtRecInfo) {
	saveData := getSaveDarftFma(payload)
	jsonStr, _ := json.MarshalToString(saveData)
	LogSysOperation(MenuFormulaManage, SubFmaDarftAdd, OpAddStr, jsonStr, "ok", "")

}

func SaveFmaDarftUpdateLog(newDarft DrafFmaWgtRecInfo, oldDarft []DrafFmaWgtRecInfo) {
	if len(oldDarft) <= 0 {
		return
	}
	saveNewData := getSaveDarftFma(newDarft)
	saveOldData := getSaveDarftFma(oldDarft[0])

	type UpdateDarft struct {
		Header     AddDraftFmaHeader
		NewDetails []AddDraftDetail
		OldDetails []AddDraftDetail
	}
	updateDarft := UpdateDarft{
		Header:     saveNewData.Header,
		NewDetails: saveNewData.Detail,
		OldDetails: saveOldData.Detail,
	}
	jsonStr, _ := json.MarshalToString(updateDarft)
	LogSysOperation(MenuFormulaManage, SubFmaDarftUpdate, OpUpdateStr, jsonStr, "ok", "")
}

func SaveDeleteManyDraftFmaWgtRecLog(drafts []DrafFmaWgtRecInfo) {
	saveDatasList := []SaveAddDarftFmaData{}
	for _, draft := range drafts {
		saveData := getSaveDarftFma(draft)
		saveDatasList = append(saveDatasList, saveData)
	}
	jsonStr, _ := json.MarshalToString(saveDatasList)
	LogSysOperation(MenuFormulaManage, SubFmaDarftDel, OpDeleteStr, jsonStr, "ok", "")
}

func SaveFmaDarftDelLog(draft []DrafFmaWgtRecInfo) {
	if len(draft) <= 0 {
		return
	}
	saveData := getSaveDarftFma(draft[0])
	jsonStr, _ := json.MarshalToString(saveData)
	LogSysOperation(MenuFormulaManage, SubFmaDarftDel, OpDeleteStr, jsonStr, "ok", "")
}

func SaveFmaWgtRecLog(recInfo FormulaWgtRecList) {
	jsonStr, _ := json.MarshalToString(recInfo)
	LogSysOperation(MenuFormulaManage, SubFmaWgtRecAdd, OpAddStr, jsonStr, "ok", "")

}

// 上传配方称重记录CSV日志
type UploadFmaWgtRecCsv struct {
	Ip        string
	ShareName string
	Result    string
}

func UploadFmaWgtRecCsvLog(ip string, shareName string, res string) {
	record := UploadFmaWgtRecCsv{
		Ip:        ip,
		ShareName: shareName,
		Result:    res,
	}
	jsonStr, _ := json.MarshalToString(record)
	LogSysOperation(MenuFormulaManage, SubFmaWgtRecUpload, OpUploadStr, jsonStr, "", "")
}

// 配方称重记录删除日志结构体
type SaveDelFmaWgtRecLogData struct {
	RecordID          string  `json:"recordId"`
	FormulaID         string  `json:"formulaId"`
	FormulaName       string  `json:"formulaName"`
	FormulaBarcode    string  `json:"formulaBarcode"`
	TotalWeight       float64 `json:"totalWeight"`
	ActualTotalWeight float64 `json:"actualTotalWeight"`
	TotalWeightUnit   string  `json:"totalWeightUnit"`
	IsQualified       string  `json:"isQualified"`
	Operator          string  `json:"operator"`
	RecordSaveTime    string  `json:"recordSaveTime"`
}

// 记录批量删除配方称重记录日志
func SaveDeleteFmaWgtRecBatchLog(headers []FormulaWgtRecHeader) {
	if len(headers) == 0 {
		return
	}
	var logList []SaveDelFmaWgtRecLogData
	for _, header := range headers {
		saveTimeStr := ""
		if !header.RecordSaveTime.IsZero() {
			saveTimeStr = header.RecordSaveTime.Format("2006-01-02 15:04:05")
		}
		logList = append(logList, SaveDelFmaWgtRecLogData{
			RecordID:          header.RecordID,
			FormulaID:         header.FormulaID,
			FormulaName:       header.FormulaName,
			FormulaBarcode:    header.FormulaBarcode,
			TotalWeight:       header.TotalWeight,
			ActualTotalWeight: header.ActualTotalWeight,
			TotalWeightUnit:   header.TotalWeightUnit,
			IsQualified:       header.IsQualified,
			Operator:          header.Operator,
			RecordSaveTime:    saveTimeStr,
		})
	}
	if len(logList) > 0 {
		jsonStr, _ := json.MarshalToString(logList)
		LogSysOperation(MenuFormulaManage, SubFmaWgtRecDel, OpDeleteStr, jsonStr, "ok", "")
	}
}

// 记录清空全库配方称重记录日志
func SaveClearAllFmaWgtRecLog(total int64) {
	type ClearLogData struct {
		Total  int64  `json:"total"`
		Action string `json:"action"`
	}
	clearData := ClearLogData{
		Total:  total,
		Action: "clear_all_formula_wgt_records",
	}
	jsonStr, _ := json.MarshalToString(clearData)
	LogSysOperation(MenuFormulaManage, SubFmaWgtRecDel, OpClearStr, jsonStr, "ok", "")
}
