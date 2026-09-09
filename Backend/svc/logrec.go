package svc

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"time"
	l "tmaxsrv/log"
)

// 系统日志列表
type SyslogList struct {
	Total int
	Logs  []Syslog
}

// 校准日志列表
type CalibrationLogList struct {
	Total int
	Logs  []CalibrationLog
}

// 称重日志列表
type ScaleLogList struct {
	Total int
	Logs  []ScaleWgtLog
}

//辅助记录日志

type UpdateLog struct {
	UpdatedFields  map[string]interface{} `json:"UpdatedFields"`  // 更新的字段及新值
	OriginalFields map[string]interface{} `json:"OriginalFields"` // 被更新字段的原始值
}

type SaveDelUser struct {
	UserId        int
	Account       string
	UserName      string
	Role          string
	Password      string
	IsEnabled     bool
	Email         string
	Phone         string
	InitialPageId int
	Remark        string
	CreatedTime   string
	UpdatedTime   string
	Rfid          string
}

type SaveAddUser struct {
	UserId        int
	Account       string
	UserName      string
	Role          string
	Password      string
	IsEnabled     bool
	Email         string
	Phone         string
	InitialPageId int
	Remark        string
	PageIds       []int
	Rfid          string
}

type DelCalibrationLog struct {
	RecId      int
	Operator   string
	Role       string
	ScaleId    int    // 秤ID
	ScaleName  string // 秤名称
	ModleName  string // 秤机种
	Sn         string // 秤号
	Type       string // 标定类型  单点，线性
	Mode       string // 标定单点，2点，3点
	Unit       string // 标定单位
	Value      string // 标定值
	Before     string // 标定前值
	After      string // 标定后值
	Error      string // 标定误差
	Result     string // 标定结果  成功，失败
	Remarks    string // 备注
	CreateTime string
}

type DelScaleLog struct {
	RecId      int
	Operator   string // 操作员
	Role       string // 角色ID
	Module     string // app
	ScaleId    int    // 秤ID
	ScaleName  string // 秤名称
	ModleName  string // 秤机种
	Sn         string // 秤号
	Weight     string // 称重值
	Unit       string // 单位
	Remarks    string // 备注
	CreateTime string // 创建时间
}

func GetRoleById(roleId int) string {
	switch roleId {
	case 1:
		return "Super Admin" // 新值
	case 2:
		return "Admin" // 新值
	default:
		return "Operator" // 新值
	}
}

// 比较用户字段，返回变更的字段映射和原始值
func CompareUserFields(oldUser *SysUser, newUser UpdateUser) (map[string]interface{}, map[string]interface{}) {
	updatedFields := make(map[string]interface{})
	oldFieldValues := make(map[string]interface{})

	if newUser.UserName != oldUser.UserName {
		updatedFields["Account"] = newUser.UserName  // 新值
		oldFieldValues["Account"] = oldUser.UserName // 原始值
	} else {
		oldFieldValues["Account"] = oldUser.UserName
	}
	if newUser.NickName != oldUser.NickName {
		updatedFields["UserName"] = newUser.NickName  // 新值
		oldFieldValues["UserName"] = oldUser.NickName // 原始值
	} else {
		oldFieldValues["UserName"] = oldUser.UserName
	}
	if newUser.RoleId != oldUser.RoleId {
		updatedFields["Role"] = GetRoleById(newUser.RoleId)
		oldFieldValues["Role"] = GetRoleById(oldUser.RoleId)
	}
	if newUser.IsEnabled != oldUser.IsEnabled {
		updatedFields["IsEnabled"] = newUser.IsEnabled  // 新值
		oldFieldValues["IsEnabled"] = oldUser.IsEnabled // 原始值
	}
	if newUser.Email != oldUser.Email {
		updatedFields["Email"] = newUser.Email  // 新值
		oldFieldValues["Email"] = oldUser.Email // 原始值
	}
	if newUser.Phone != oldUser.Phone {
		updatedFields["Phone"] = newUser.Phone  // 新值
		oldFieldValues["Phone"] = oldUser.Phone // 原始值
	}
	if newUser.InitialPageId != oldUser.InitialPageId {
		updatedFields["InitialPageId"] = newUser.InitialPageId  // 新值
		oldFieldValues["InitialPageId"] = oldUser.InitialPageId // 原始值
	}
	if newUser.Remark != oldUser.Remark {
		updatedFields["Remark"] = newUser.Remark  // 新值
		oldFieldValues["Remark"] = oldUser.Remark // 原始值
	}
	if newUser.Rfid != oldUser.Rfid {
		updatedFields["Rfid"] = newUser.Rfid  // 新值
		oldFieldValues["Rfid"] = oldUser.Rfid // 原始值
	}
	return updatedFields, oldFieldValues
}

// 比较页面权限是否有变化
func ComparePages(userName string, newPages []int) (bool, []int) {
	// 获取用户当前的页面权限
	currentPages, err := mSrvMgr.sysUserPd.infoPb.GetUserRolePermission(userName)
	if err != nil {
		l.Log.Error("Failed to get user page permissions:", err)
		return true, currentPages.PageIDList // 如果获取失败，认为有变化
	}

	// 如果长度不同，肯定有变化
	if len(currentPages.PageIDList) != len(newPages) {
		return true, currentPages.PageIDList
	}

	// 排序后比较内容
	sort.Ints(currentPages.PageIDList)
	sortedNewPages := make([]int, len(newPages))
	copy(sortedNewPages, newPages)
	sort.Ints(sortedNewPages)

	for i, page := range currentPages.PageIDList {
		if page != sortedNewPages[i] {
			return true, currentPages.PageIDList
		}
	}

	return false, nil
}

//导出日志到文件

func ExportSysLogsToFile(logs []Syslog, filePath string, trans map[string]string, headers []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		l.Log.Error(err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	if len(headers) == 0 {
		headers = []string{"ID", "Operator", "Role", "Module", "Function Module", "Operation Type", "Details", "Result", "Created Time"}
	}
	writer.Write(headers)

	// 写入日志数据
	for _, log := range logs {

		// 转换RoleId为字符串
		roleIdStr := strconv.Itoa(log.RoleId)
		roleName := roleIdStr
		module := log.Module
		funcModule := log.FuncName
		operationType := log.OperationType
		operation := log.Operation
		result := log.Result

		for key, value := range trans {
			if roleName == key {
				roleName = value
				continue
			}
			if operationType == key {
				operationType = value
				continue
			}
			if funcModule == key {
				funcModule = value
				continue
			}
			if module == key {
				module = value
				continue
			}

			if result == key {
				result = value
				continue
			}
		}

		row := []string{
			strconv.Itoa(log.RecId),
			log.Operator,
			roleName,
			module,
			funcModule,
			operationType,
			operation,
			result,
			log.CreateTime.Format("2006-01-02 15:04:05"),
		}
		writer.Write(row)
	}
	return nil
}

func GetSearchLog(logQuery LogQuery) SyslogQuery {
	dateStartStr := logQuery.StartTime
	if dateStartStr == "" {
		dateStartStr = ("2006-01-02")
	} else {
		dateStartStr = logQuery.StartTime[:10]
	}

	startTime, err := time.Parse("2006-01-02", dateStartStr)
	if err != nil {
		l.Log.Errorf("日期解析失败: %v", err)
		return SyslogQuery{}
	}

	dateEndStr := logQuery.EndTime
	if dateEndStr == "" {
		dateEndStr = time.Now().Format("2006-01-02")
	} else {
		dateEndStr = dateEndStr[:10]
	}

	endTime, err := time.Parse("2006-01-02", dateEndStr)
	if err != nil {
		l.Log.Errorf("日期解析失败: %v", err)
		return SyslogQuery{}
	}

	syslogQuery := SyslogQuery{
		Operator:  logQuery.Operator,
		Module:    logQuery.Module,
		RoleId:    logQuery.RoleId,
		StartTime: startTime,
		EndTime:   endTime.AddDate(0, 0, 1),
	}
	return syslogQuery
}

func SaveAddUserFunc(newUser *SysUser, pagesId []int) string {

	userTemp := SaveAddUser{
		UserId:        newUser.UserId,
		Account:       newUser.UserName,
		UserName:      newUser.NickName,
		Role:          GetRoleById(newUser.RoleId),
		Password:      "******",
		IsEnabled:     newUser.IsEnabled,
		Email:         newUser.Email,
		Phone:         newUser.Phone,
		InitialPageId: newUser.InitialPageId,
		Remark:        newUser.Remark,
		PageIds:       pagesId,
		Rfid:          newUser.Rfid,
	}

	jsonStr, _ := json.MarshalToString(userTemp)

	return jsonStr
}

// 保存删除校准日志
func SaveDelCalLog(logs []CalibrationLog) string {

	logsTemp := make([]DelCalibrationLog, len(logs))
	for i, log := range logs {
		logsTemp[i] = DelCalibrationLog{
			RecId:      log.RecId,
			Operator:   log.Operator,
			Role:       GetRoleById(log.RoleId),
			ScaleId:    log.ScaleId,
			ScaleName:  log.ScaleName,
			ModleName:  log.ModelName,
			Sn:         log.Sn,
			Type:       log.Type,
			Mode:       log.Mode,
			Unit:       log.Unit,
			Value:      log.Value,
			Before:     log.Before,
			After:      log.After,
			Error:      log.Error,
			Result:     log.Result,
			Remarks:    log.Remarks,
			CreateTime: log.CreateTime.Format("2006-01-02 15:04:05"),
		}

	}

	jsonStr, _ := json.MarshalToString(logsTemp)
	return jsonStr
}

// 保存删除称重日志
func SaveDelScaleLog(logs []ScaleWgtLog) string {

	logsTemp := make([]DelScaleLog, len(logs))
	for i, log := range logs {
		logsTemp[i] = DelScaleLog{
			RecId:      log.RecId,
			Operator:   log.Operator,
			Role:       GetRoleById(log.RoleId),
			ScaleName:  log.ScaleName,
			ModleName:  log.ModelName,
			Sn:         log.Sn,
			Weight:     log.Weight,
			Unit:       log.Unit,
			Remarks:    log.Remarks,
			CreateTime: log.CreateTime.Format("2006-01-02 15:04:05"),
		}
	}
	jsonStr, _ := json.MarshalToString(logsTemp)
	return jsonStr
}

// 保存更新用户日志
func SaveUpdateUserLog(accout string, username string, updatedFields map[string]interface{}, oldFieldValues map[string]interface{}) string {
	// 检查并添加缺失的字段

	updateLog := UpdateLog{
		UpdatedFields:  updatedFields,  // 更新的字段及新值
		OriginalFields: oldFieldValues, // 被更新字段的原始值
	}

	jsonStr, _ := json.MarshalToString(updateLog)

	return jsonStr
}

func SaveDelUserFunc(users []SysUser) string {
	var delUsers []SaveDelUser

	for i := 0; i < len(users); i++ {

		user := users[i]
		reqInfo := SaveDelUser{
			UserId:        user.UserId,
			Account:       user.UserName,
			UserName:      user.NickName,
			Role:          GetRoleById(user.RoleId),
			Password:      "***",
			IsEnabled:     user.IsEnabled,
			Email:         user.Email,
			Phone:         user.Phone,
			InitialPageId: user.InitialPageId,
			Remark:        user.Remark,
			CreatedTime:   user.CreatedTime.Format("2006-01-02 15:04:05"),
			UpdatedTime:   user.UpdatedTime.Format("2006-01-02 15:04:05"),
			Rfid:          user.Rfid,
		}
		delUsers = append(delUsers, reqInfo)
	}
	jsonStr, _ := json.MarshalToString(delUsers)
	return jsonStr
}

// 导出称重日志到文件
func ExportWgtLogsToFile(logs []ScaleWgtLog, filePath string, trans map[string]string, headers []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		l.Log.Error(err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	if len(headers) == 0 {
		headers = []string{"ID", "Operator", "Role", "Module", "Total Weight", "Weight Unit", "Scale Name", "Scale Module", "SN", "Created Time"}
	}
	writer.Write(headers)

	// 写入日志数据
	for _, log := range logs {

		// 转换RoleId为字符串
		roleIdStr := strconv.Itoa(log.RoleId)
		roleName := roleIdStr
		module := log.Module

		for key, value := range trans {
			if roleName == key {
				roleName = value
				continue
			}
			if module == key {
				module = value
				continue
			}
		}

		row := []string{
			strconv.Itoa(log.RecId),
			log.Operator,
			roleName,
			module,
			log.Weight,
			log.Unit,
			log.ScaleName,
			log.ModelName,
			log.Sn,
			log.CreateTime.Format("2006-01-02 15:04:05"),
		}

		writer.Write(row)
	}
	return nil
}

func ExportCalLogsToFile(logs []CalibrationLog, filePath string, trans map[string]string, headers []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		l.Log.Error(err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	if len(headers) == 0 {
		headers = []string{"ID", "Operator", "Role", "Calibration Type", "Calibration Mode", "Calibration Unit", "Calibration Value", "Before", "After", "Error", "Result", "Scale Name", "Scale Module", "SN", "Created Time"}
	}
	writer.Write(headers)

	// 写入日志数据
	for _, log := range logs {

		// 转换RoleId为字符串
		roleIdStr := strconv.Itoa(log.RoleId)
		roleName := roleIdStr
		typeName := log.Type

		for key, value := range trans {
			if roleName == key {
				roleName = value
				continue
			}
			if typeName == key {
				typeName = value
				continue
			}
		}

		row := []string{
			strconv.Itoa(log.RecId),
			log.Operator,
			roleName,
			typeName,
			log.Mode,
			log.Unit,
			log.Value,
			log.Before,
			log.After,
			log.Error,
			log.Result,
			log.ScaleName,
			log.ModelName,
			log.Sn,
			log.CreateTime.Format("2006-01-02 15:04:05"),
		}

		writer.Write(row)
	}
	return nil
}

 


