package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbSyslogRec struct {
	dbName string
	db     *gorm.DB
}

func NewDbSyslogRec(dbName string) (*DbSyslogRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}

	if err = db.AutoMigrate(
		&Syslog{},
		&CalibrationLog{},
		&ScaleWgtLog{},
		&SealLog{},
	); err != nil {
		l.Log.Errorf("failed to migrate database of scale connection: %v", err)
	}

	return &DbSyslogRec{dbName: dbName, db: db}, nil
}

func (d *DbSyslogRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// 操作日志表
type Syslog struct {
	RecId         int       `gorm:"primaryKey;autoincrement;not null"` // 记录ID
	Operator      string    // 操作员
	RoleId        int       // 角色ID
	Module        string    // 模块  系统，app
	FuncName      string    // 功能模块  新增配方等具体功能
	OperationType string    // 操作类型  新增，删除，修改，查询等
	Operation     string    // 具体操作  新增了哪个配方，删除了哪个记录等
	Result        string    // 操作结果  成功，失败
	Remarks       string    // 备注
	CreateTime    time.Time // 创建时间
}

// 标定日志
type CalibrationLog struct {
	RecId      int       `gorm:"primaryKey;autoincrement;not null"` // 记录ID
	Operator   string    // 操作员
	RoleId     int       // 角色ID
	ScaleId    int       // 秤ID
	ScaleName  string    // 秤名称
	ModelName  string    // 秤机种
	Sn         string    // 秤号
	Type       string    // 标定类型  单点，线性
	Mode       string    // 标定单点，2点，3点
	Unit       string    // 标定单位
	Value      string    // 标定值
	Before     string    // 标定前值
	After      string    // 标定后值
	Error      string    // 标定误差
	Result     string    // 标定结果  成功，失败
	Remarks    string    // 备注
	CreateTime time.Time // 创建时间
}

// 称重日志
type ScaleWgtLog struct {
	RecId      int       `gorm:"primaryKey;autoincrement;not null"` // 记录ID
	Operator   string    // 操作员
	RoleId     int       // 角色ID
	Module     string    // app
	ScaleName  string    // 秤名称
	ModelName  string    // 秤机种
	Sn         string    // 秤号
	Weight     string    // 称重值
	Unit       string    // 单位
	Remarks    string    // 备注
	CreateTime time.Time // 创建时间
}

// 创建系统操作日志
func (d *DbSyslogRec) CreateSyslog(log Syslog) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Create(&log).Error; err != nil {
		l.Log.Errorf("failed to create syslog: %v", err)
		return err
	}
	return nil
}

// 创建标定日志
func (d *DbSyslogRec) CreateCalibrationLog(log CalibrationLog) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Create(&log).Error; err != nil {
		l.Log.Errorf("failed to create calibration log: %v", err)
		return err
	}
	return nil
}

// 创建称重日志
func (d *DbSyslogRec) CreateScaleLog(log ScaleWgtLog) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Create(&log).Error; err != nil {
		l.Log.Errorf("failed to create scale log: %v", err)
		return err
	}
	return nil
}

// 删除系统操作日志
func (d *DbSyslogRec) DeleteSyslog(recIds []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&Syslog{}, "rec_id IN ?", recIds).Error; err != nil {
		l.Log.Errorf("failed to delete syslog: %v", err)
		return err
	}
	return nil
}

// 删除所有系统操作日志
func (d *DbSyslogRec) DeleteAllSyslog() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Where("rec_id > ?", 0).Delete(&Syslog{}).Error
}

// 删除标定日志
func (d *DbSyslogRec) DeleteCalibrationLog(recIds []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&CalibrationLog{}, "rec_id IN ?", recIds).Error; err != nil {
		l.Log.Errorf("failed to delete calibration log: %v", err)
		return err
	}
	return nil
}

// 删除所有标定日志
func (d *DbSyslogRec) DeleteAllCalibrationLog() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&CalibrationLog{}).Error; err != nil {
		l.Log.Errorf("failed to delete all calibration log: %v", err)
		return err
	}
	return nil
}

// 删除称重日志
func (d *DbSyslogRec) DeleteScaleLog(recIds []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&ScaleWgtLog{}, "rec_id IN ?", recIds).Error; err != nil {
		l.Log.Errorf("failed to delete scale log: %v", err)
		return err
	}
	return nil
}

// 删除所有称重日志
func (d *DbSyslogRec) DeleteAllScaleLog() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Where("rec_id > ?", 0).Delete(&ScaleWgtLog{}).Error
}

// 删除多条系统操作日志
func (d *DbSyslogRec) DeleteMultiSyslog(recIds []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&Syslog{}, recIds).Error; err != nil {
		l.Log.Errorf("failed to delete multi syslog: %v", err)
		return err
	}
	return nil
}

// 删除多条标定日志
func (d *DbSyslogRec) DeleteMultiCalibrationLog(recIds []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&CalibrationLog{}, recIds).Error; err != nil {
		l.Log.Errorf("failed to delete multi calibration log: %v", err)
		return err
	}
	return nil
}

// 删除多条称重日志
func (d *DbSyslogRec) DeleteMultiScaleLog(recIds []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	if err := d.db.Delete(&ScaleWgtLog{}, recIds).Error; err != nil {
		l.Log.Errorf("failed to delete multi scale log: %v", err)
		return err
	}
	return nil
}

// 获取所有系统操作日志
func (d *DbSyslogRec) GetAllSyslog() ([]Syslog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var syslogs []Syslog
	err := d.db.Find(&syslogs).Error
	return syslogs, err
}

// 获取所有标定日志
func (d *DbSyslogRec) GetAllCalibrationLog() ([]CalibrationLog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var calibrationLogs []CalibrationLog
	err := d.db.Find(&calibrationLogs).Error
	return calibrationLogs, err
}

// 获取所有称重日志
func (d *DbSyslogRec) GetAllScaleLog() ([]ScaleWgtLog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var scaleLogs []ScaleWgtLog
	err := d.db.Find(&scaleLogs).Error
	return scaleLogs, err
}

// 定义查询条件结构体
type SyslogQuery struct {
	Operator  string // 操作员
	Module    string
	RoleId    int       // 角色ID
	StartTime time.Time // 创建时间起始
	EndTime   time.Time // 创建时间结束
}

// 分页获取系统操作日志（带组合条件、排序、分页）
func (d *DbSyslogRec) GetSyslogWithSort(
	pageNum int,
	pageSize int,
	sortField string,
	sortOrder string,
	query SyslogQuery,
) ([]Syslog, int64, error) {
	if pageNum <= 0 || d == nil || d.db == nil {
		return []Syslog{}, 0, nil
	}

	tx := d.db.Model(&Syslog{})

	if query.Operator != "" {
		tx = tx.Where("operator LIKE ?", "%"+query.Operator+"%")
	}
	if query.Module != "" {
		tx = tx.Where("module LIKE ?", "%"+query.Module+"%")
	}
	if query.RoleId != 0 {
		tx = tx.Where("role_id = ?", query.RoleId)
	}
	if !query.StartTime.IsZero() {
		tx = tx.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		tx = tx.Where("create_time <= ?", query.EndTime)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		l.Log.Errorf("failed to count syslogs: %v", err)
		return nil, 0, err
	}

	if sortField == "" {
		sortField = "create_time"
	}
	if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
		sortOrder = "desc"
	}
	orderStr := sortField + " " + sortOrder

	var syslogs []Syslog
	if err := tx.
		Order(orderStr).
		Limit(pageSize).
		Offset((pageNum - 1) * pageSize).
		Find(&syslogs).Error; err != nil {
		l.Log.Errorf("failed to query syslogs: %v", err)
		return nil, 0, err
	}

	return syslogs, total, nil
}

// 分页获取标定日志（带组合条件、排序、分页）
func (d *DbSyslogRec) GetCalibrationLogWithSort(
	pageNum int,
	pageSize int,
	sortField string,
	sortOrder string,
	query SyslogQuery,
) ([]CalibrationLog, int64, error) {
	if pageNum <= 0 || d == nil || d.db == nil {
		return []CalibrationLog{}, 0, nil
	}

	tx := d.db.Model(&CalibrationLog{})

	if query.Operator != "" {
		tx = tx.Where("operator LIKE ?", "%"+query.Operator+"%")
	}
	if query.RoleId != 0 {
		tx = tx.Where("role_id = ?", query.RoleId)
	}
	if !query.StartTime.IsZero() {
		tx = tx.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		tx = tx.Where("create_time <= ?", query.EndTime)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		l.Log.Errorf("failed to count calibration logs: %v", err)
		return nil, 0, err
	}

	if sortField == "" {
		sortField = "create_time"
	}
	if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
		sortOrder = "desc"
	}
	orderStr := sortField + " " + sortOrder

	var calibrationLogs []CalibrationLog
	if err := tx.
		Order(orderStr).
		Limit(pageSize).
		Offset((pageNum - 1) * pageSize).
		Find(&calibrationLogs).Error; err != nil {
		l.Log.Errorf("failed to query calibration logs: %v", err)
		return nil, 0, err
	}
	return calibrationLogs, total, nil
}

// 分页获取称重日志（带组合条件、排序、分页）
func (d *DbSyslogRec) GetScaleLogWithSort(
	pageNum int,
	pageSize int,
	sortField string,
	sortOrder string,
	query SyslogQuery,
) ([]ScaleWgtLog, int64, error) {
	if pageNum <= 0 || d == nil || d.db == nil {
		return []ScaleWgtLog{}, 0, nil
	}

	tx := d.db.Model(&ScaleWgtLog{})

	if query.Operator != "" {
		tx = tx.Where("operator LIKE ?", "%"+query.Operator+"%")
	}
	if query.RoleId != 0 {
		tx = tx.Where("role_id = ?", query.RoleId)
	}
	if !query.StartTime.IsZero() {
		tx = tx.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		tx = tx.Where("create_time <= ?", query.EndTime)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		l.Log.Errorf("failed to count weight logs: %v", err)
		return nil, 0, err
	}

	if sortField == "" {
		sortField = "create_time"
	}
	if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
		sortOrder = "desc"
	}
	orderStr := sortField + " " + sortOrder

	var weightLogs []ScaleWgtLog
	if err := tx.
		Order(orderStr).
		Limit(pageSize).
		Offset((pageNum - 1) * pageSize).
		Find(&weightLogs).Error; err != nil {
		l.Log.Errorf("failed to query weight logs: %v", err)
		return nil, 0, err
	}
	return weightLogs, total, nil
}

func (d *DbSyslogRec) ExportSyslogWithSort(
	sortField string,
	sortOrder string,
	query SyslogQuery,
) ([]Syslog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	tx := d.db.Model(&Syslog{})

	if query.Operator != "" {
		tx = tx.Where("operator LIKE ?", "%"+query.Operator+"%")
	}
	if query.Module != "" {
		tx = tx.Where("module LIKE ?", "%"+query.Module+"%")
	}
	if query.RoleId != 0 {
		tx = tx.Where("role_id = ?", query.RoleId)
	}
	if !query.StartTime.IsZero() {
		tx = tx.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		tx = tx.Where("create_time <= ?", query.EndTime)
	}

	if sortField == "" {
		sortField = "create_time"
	}
	if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
		sortOrder = "desc"
	}
	orderStr := sortField + " " + sortOrder

	var syslogs []Syslog
	if err := tx.
		Order(orderStr).
		Find(&syslogs).Error; err != nil {
		l.Log.Errorf("failed to query syslogs: %v", err)
		return nil, err
	}
	return syslogs, nil
}

func (d *DbSyslogRec) ExportCalibrationLogWithSort(
	sortField string,
	sortOrder string,
	query SyslogQuery,
) ([]CalibrationLog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	tx := d.db.Model(&CalibrationLog{})

	if query.Operator != "" {
		tx = tx.Where("operator LIKE ?", "%"+query.Operator+"%")
	}
	if query.RoleId != 0 {
		tx = tx.Where("role_id = ?", query.RoleId)
	}
	if !query.StartTime.IsZero() {
		tx = tx.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		tx = tx.Where("create_time <= ?", query.EndTime)
	}

	if sortField == "" {
		sortField = "create_time"
	}
	if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
		sortOrder = "desc"
	}
	orderStr := sortField + " " + sortOrder

	var calibrationLogs []CalibrationLog
	if err := tx.
		Order(orderStr).
		Find(&calibrationLogs).Error; err != nil {
		l.Log.Errorf("failed to query calibration logs: %v", err)
		return nil, err
	}
	return calibrationLogs, nil
}

func (d *DbSyslogRec) ExportScaleLogWithSort(
	sortField string,
	sortOrder string,
	query SyslogQuery,
) ([]ScaleWgtLog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	tx := d.db.Model(&ScaleWgtLog{})

	if query.Operator != "" {
		tx = tx.Where("operator LIKE ?", "%"+query.Operator+"%")
	}
	if query.RoleId != 0 {
		tx = tx.Where("role_id = ?", query.RoleId)
	}
	if !query.StartTime.IsZero() {
		tx = tx.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		tx = tx.Where("create_time <= ?", query.EndTime)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		l.Log.Errorf("failed to count weight logs: %v", err)
		return nil, err
	}

	if sortField == "" {
		sortField = "create_time"
	}
	if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
		sortOrder = "desc"
	}
	orderStr := sortField + " " + sortOrder

	var weightLogs []ScaleWgtLog
	if err := tx.
		Order(orderStr).
		Find(&weightLogs).Error; err != nil {
		l.Log.Errorf("failed to query weight logs: %v", err)
		return nil, err
	}
	return weightLogs, nil
}

func (d *DbSyslogRec) GetSyslogByID(RecIds []int) ([]Syslog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	tx := d.db.Model(&Syslog{})
	var syslogs []Syslog
	if err := tx.
		Where("rec_id IN ?", RecIds).
		Find(&syslogs).Error; err != nil {
		l.Log.Errorf("failed to query syslog by rec_id: %v", err)
		return nil, err
	}
	return syslogs, nil
}

func (d *DbSyslogRec) GetCalibrationLogByID(RecIds []int) ([]CalibrationLog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	tx := d.db.Model(&CalibrationLog{})
	var calibrationLogs []CalibrationLog
	if err := tx.
		Where("rec_id IN ?", RecIds).
		Find(&calibrationLogs).Error; err != nil {
		l.Log.Errorf("failed to query calibration log by rec_id: %v", err)
		return nil, err
	}
	return calibrationLogs, nil
}

func (d *DbSyslogRec) GetScaleLogByID(RecIds []int) ([]ScaleWgtLog, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	tx := d.db.Model(&ScaleWgtLog{})
	var scaleLogs []ScaleWgtLog
	if err := tx.
		Where("rec_id IN ?", RecIds).
		Find(&scaleLogs).Error; err != nil {
		l.Log.Errorf("failed to query scale log by rec_id: %v", err)
		return nil, err
	}
	return scaleLogs, nil
}

type SealLog struct {
	RecId         int `gorm:"primaryKey;not null;autoincrement;"`
	Model         string
	ScaleId       int
	Sn            string
	RoleId        int `gorm:"not null;"`
	Operator      string
	Operation     string    //seal unseal
	OperationTime time.Time `gorm:"autoCreateTime"`
	Remark        string
	Result        string
}

func (d *DbSyslogRec) CreateSealLog(log *SealLog) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Create(log).Error
}

func (d *DbSyslogRec) ListSealLogs(model string, sn string) ([]SealLog, error) {
	if d == nil || d.db == nil {
		return []SealLog{}, errors.New("db connection is nil")
	}
	var logs []SealLog
	result := d.db.Where("model = ? and sn = ?", model, sn).Find(&logs)
	if result.Error != nil {
		return []SealLog{}, result.Error
	}
	return logs, nil
}
