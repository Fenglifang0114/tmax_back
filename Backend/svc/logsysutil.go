package svc

import (
	"fmt"
	"time"
	l "tmaxsrv/log"
)

// 模块
const (
	MenuSystem = "system" // 系统模块

	MenuSysLog   = "syslog"   // 系统日志模块
	MenuCalLog   = "callog"   // 校准日志模块
	MenuScaleLog = "scalelog" //称重日志模块

	MenuScalesManage   = "scale_manage"     // 秤管理模块
	MenuPLUManage      = "plu_manage"       // PLU管理模块
	MenuUserManage     = "user_manage"      // 用户管理模块
	MenuFormulaManage  = "formula_manage"   // 配方管理模块
	MenuDeviceTime     = "device_time"      // 设备时间管理模块
	MenuDeviceBt       = "device_bt"        // 设备蓝牙管理模块
	MenuUpdateFirmware = "update_firmware"  // 更新固件模块
	MenuDownLabelFmt   = "down_label_fmt"   // 下发打印格式
	MenuDownReceiptFmt = "down_receipt_fmt" // 下发打印格式

	// ... 其他菜单ID

	MenuWgtCol   = "wgt_col"   // 称重采集模块
	MenuCheckWgt = "check_wgt" // 重量校验模块
	MenuTakeIn   = "take_in"   // 加法秤模块
	MenuTakeOut  = "take_out"  // 减法秤模块
)

// 子模块类型
const (
	SubSysLogin  = "login"  // 登录模块
	SubSysLogout = "logout" // 登出模块

	SubLogDel    = "log_del"    //称重日志模块删除
	SubLogExport = "log_export" // 日志模块导出
	SubLogClear  = "log_clear"  //日志模块清空

	SubAdd    = "add"    // 新增
	SubDel    = "delete" // 删除
	SubUpdate = "update" // 更新

	SubUserManageEditPswd = "user_update_pswd" // 用户管理模块更新密码
	SubUserManageEnabled  = "user_enabled"     // 用户管理模块启用/禁用

	SubFormulaAdd    = "formula_add"    // 配方管理模块新增
	SubFormulaDel    = "formula_del"    // 配方管理模块删除
	SubFormulaUpdate = "formula_update" // 配方管理模块更新

	SubFormulaTypeAdd         = "formula_type_add"          // 配方类型管理模块新增
	SubFormulaTypeDel         = "formula_type_del"          // 配方类型管理模块删除
	SubFormulaTypeUpdate      = "formula_type_update"       // 配方类型管理模块更新
	SubFormulaTypeClearUnused = "formula_type_clear_unused" // 配方类型管理模块清除未使用的

	SubFmaRawTypeAdd         = "raw_type_add"          // 原料类型管理模块新增
	SubFmaRawTypeDel         = "raw_type_del"          // 原料类型管理模块删除
	SubFmaRawTypeUpdate      = "raw_type_update"       // 原料类型管理模块更新
	SubFmaRawTypeClearUnused = "raw_type_clear_unused" // 原料类型管理模块清除未使用的

	SubFmaRawAdd    = "raw_add"    // 原料数据管理模块新增
	SubFmaRawDel    = "raw_del"    // 原料数据管理模块删除
	SubFmaRawUpdate = "raw_update" // 原料数据管理模块更新

	SubFmaWgtRecAdd = "fma_wgt_rec_add" // 重量记录管理模块新增
	SubFmaWgtRecDel = "fma_wgt_rec_del" // 重量记录管理模块删除

	SubFmaWgtRecUpload = "fma_wgt_rec_upload" // 重量记录管理模块上传

	SubFmaDarftAdd    = "fma_draft_add"    // 草稿管理模块新增
	SubFmaDarftDel    = "fma_draft_del"    // 草稿管理模块删除
	SubFmaDarftUpdate = "fma_draft_update" // 草稿管理模块更新

	SubDeviceTimeSet = "device_time_set" // 设备时间管理模块设置

	SubBtSetName  = "bt_set_name"  // 设备蓝牙管理模块设置蓝牙名字
	SubBtSetPower = "bt_set_power" // 设备蓝牙管理模块设置蓝牙功率

	SubUpdateFirmware = "update_firmware"  // 更新固件模块
	SubDownLabelFmt   = "down_label_fmt"   // 下发打印格式
	SubDownReceiptFmt = "down_receipt_fmt" // 下发打印格式

	SubWgtCol   = "wgt_col"   // 称重采集模块
	SubCheckWgt = "check_wgt" // 重量校验模块
	SubTakeIn   = "take_in"   // 加法秤模块
	SubTakeOut  = "take_out"  // 减法秤模块

)

// 操作类型
const (
	OpLoginStr   = "login"   // 登录
	OpLogoutStr  = "logout"  // 登出
	OpAddStr     = "add"     // 新增
	OpDeleteStr  = "delete"  // 删除
	OpClearStr   = "clear"   // 清空
	OpUpdateStr  = "update"  // 更新
	OpEnabledStr = "enabled" // 启用/禁用
	OpQueryStr   = "query"   // 查询
	OpIssueStr   = "issue"   // 下发
	OpImportStr  = "import"  // 导入
	OpExportStr  = "export"  // 导出
	OpSetStr     = "setting" // 设置
	OpUploadStr  = "upload"  // 上传
)

// LogSysOperation 统一操作日志记录接口
func LogSysOperation(module string, subModule string, opType string, opContent string, result string, remarks string) {
	// 1. 获取当前用户信息
	userID, username, roleID := GetCurrentUser()
	if userID == 0 {
		return
	}

	// 2. 构造操作日志对象
	log := Syslog{
		Operator:      username,
		RoleId:        roleID,
		Module:        module,
		FuncName:      subModule,
		OperationType: opType,
		Operation:     opContent,
		Result:        result,
		Remarks:       remarks,
		CreateTime:    time.Now(),
	}

	fmt.Println("LogSysOperation: ", log)

	// 3. 异步写入数据库（不阻塞主流程）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Log.Errorf("panic in LogSysOperation goroutine: %v", r)
			}
		}()
		// 写入数据库
		if mSrvMgr == nil || mSrvMgr.sysLogPd == nil {
			l.Log.Error("mSrvMgr or sysLogPd is nil when writing syslog")
			return
		}
		if err := mSrvMgr.sysLogPd.AddSyslog(log); err != nil {
			l.Log.Error(fmt.Sprintf("Failed to write operation log: %v, log content: %+v", err, log))
		} else {
			l.Log.Debug(fmt.Sprintf("Operation log written successfully: %+v", log))
		}
	}()
}

// // 5. 记录操作日志（异步执行） 在请求的函数中实现
// LogOperation(req.Ctx, OpTypeIssue, MenuPrintFormat,
// 	fmt.Sprintf("下发打印格式: ID=%s, 格式名称=%s, 内容=%s",
// 		req.FormatID, req.FormatName, req.FormatContent),
// 	status, result, c.DeviceId, c.DeviceSn)

// getScaleWgtMode 获取称重模块名称
func GetScaleWgtMode(mode int) (module string) {
	switch mode {
	case 0:
		module = MenuWgtCol
	case 1:
		module = MenuCheckWgt
	case 2:
		module = MenuTakeIn
	case 3:
		module = MenuTakeOut
	default:
		module = "unknown"
	}
	return module
}

// LogSysOperation 统一操作日志记录接口
func LogScaleWgtOperation(mode int, wgtInfo ScaleRec, remarks string) {
	// 1. 获取当前用户信息
	userID, username, roleID := GetCurrentUser()
	if userID == 0 {
		return
	}

	module := GetScaleWgtMode(mode)

	// 2. 构造操作日志对象
	log := ScaleWgtLog{
		Operator:   username,
		RoleId:     roleID,
		Module:     module,
		ScaleName:  wgtInfo.ScaleName,
		ModelName:  wgtInfo.ScaleModel,
		Sn:         wgtInfo.ScaleSn,
		Weight:     wgtInfo.Weight,
		Unit:       wgtInfo.WeightUnit,
		Remarks:    remarks,
		CreateTime: time.Now(),
	}

	fmt.Println("LogSysOperation: ", log)

	// 3. 异步写入数据库（不阻塞主流程）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Log.Errorf("panic in LogScaleWgtOperation goroutine: %v", r)
			}
		}()
		// 写入数据库
		if mSrvMgr == nil || mSrvMgr.sysLogPd == nil {
			l.Log.Error("mSrvMgr or sysLogPd is nil when writing scale log")
			return
		}
		if err := mSrvMgr.sysLogPd.AddScaleLog(log); err != nil {
			l.Log.Error(fmt.Sprintf("Failed to write operation log: %v, log content: %+v", err, log))
		} else {
			l.Log.Debug(fmt.Sprintf("Operation log written successfully: %+v", log))
		}
	}()
}

// LogSysOperation 统一操作日志记录接口
func LogCalLogOperation(rec CalibrationLog) {
	// 1. 获取当前用户信息
	userID, username, roleID := GetCurrentUser()
	if userID == 0 {
		return
	}

	// 2. 构造操作日志对象
	log := CalibrationLog{
		Operator:   username,
		RoleId:     roleID,
		ScaleId:    rec.ScaleId,
		ScaleName:  rec.ScaleName,
		ModelName:  rec.ModelName,
		Sn:         rec.Sn,
		Type:       rec.Type,
		Mode:       rec.Mode,
		Unit:       rec.Unit,
		Value:      rec.Value,
		Before:     rec.Before,
		After:      rec.After,
		Error:      rec.Error,
		Result:     rec.Result,
		Remarks:    rec.Remarks,
		CreateTime: time.Now(),
	}

	fmt.Println("LogSysOperation: ", log)

	// 3. 异步写入数据库（不阻塞主流程）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Log.Errorf("panic in LogCalLogOperation goroutine: %v", r)
			}
		}()
		// 写入数据库
		if mSrvMgr == nil || mSrvMgr.sysLogPd == nil {
			l.Log.Error("mSrvMgr or sysLogPd is nil when writing cal log")
			return
		}
		if err := mSrvMgr.sysLogPd.AddCalibrationLog(log); err != nil {
			l.Log.Error(fmt.Sprintf("Failed to write operation log: %v, log content: %+v", err, log))
		} else {
			l.Log.Debug(fmt.Sprintf("Operation log written successfully: %+v", log))
		}
	}()
}
