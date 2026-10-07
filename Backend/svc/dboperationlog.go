package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbOperationLog struct {
	dbName string
	db     *gorm.DB
}

type OperationLog struct {
	RecId           int `gorm:"primaryKey;not null;autoincrement;"`
	OperatorTypeID  int // 关联operation_types.ID
	OperatorMenuID  int // 关联operation_menus.ID
	OperatorContent string
	Status          int
	Result          string

	DeviceId    int
	DeviceModel string
	DeviceSn    string

	Remark string

	OperationTime time.Time `gorm:"autoCreateTime"`
	OperatorId    int       `gorm:"not null;"`
	OperatorName  string    `gorm:"not null;"`
	RoleId        int       `gorm:"not null;"`

	OperatorType *OperationType `gorm:"foreignKey:OperatorTypeID"`
	OperatorMenu *OperationMenu `gorm:"foreignKey:OperatorMenuID"`
}

type OperationType struct {
	ID   int    `gorm:"primaryKey"`
	Name string `gorm:"not null;unique"` // 如"登录"、"增加"
	Desc string // 描述（可选）
}

type OperationMenu struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null;unique"` // 如"秤管理"、"PLU列表"
	Path     string // 菜单路径（可选，如"/scale/manage"）
	ParentID int    // 父菜单ID（如果有层级关系）
}

func NewDbOperationLog(dbName string) (*DbOperationLog, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(
		&OperationType{},
		&OperationMenu{},
		&OperationLog{},
	); err != nil {
		l.Log.Errorf("failed to migrate database of operation log: %v", err)
	}

	return &DbOperationLog{dbName: dbName, db: db}, nil
}

func (d *DbOperationLog) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// 创建操作日志
func (d *DbOperationLog) CreateOperationLog(log *OperationLog) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Create(log).Error
}

// 删除操作日志
func (d *DbOperationLog) DeleteOperationLog(logID []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	for _, id := range logID {
		if err := d.db.Delete(&SysUser{}, "user_id = ?", id).Error; err != nil {
			return err
		}
	}
	return nil
}

// 查询操作日志列表
func (d *DbOperationLog) ListOperationLogs() ([]OperationLog, error) {
	if d == nil || d.db == nil {
		return []OperationLog{}, errors.New("db connection is nil")
	}
	var logs []OperationLog
	result := d.db.Find(&logs)
	if result.Error != nil {
		return []OperationLog{}, result.Error
	}
	return logs, nil
}
