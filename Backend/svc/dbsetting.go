package svc

import (
	"errors"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbModeSetting struct {
	dbName string
	db     *gorm.DB
}

type ModeSetting struct {
	//ScaleMode 0 =DC500 1=check Weigher 2=take in  3=take out
	Id            uint `gorm:"primaryKey;autoincrement;not null"`
	ScaleMode     uint
	DateFormat    string
	DateSeparator string
	StableTime    string
	ZeroRange     string
	RecMode       string
	ScaleSn       string
	SaveMode      string
	WgtMode       uint `gorm:"default:0"` //0 单台模式  1 合并称重模式
}

func NewDbModeSetting(dbName string) (*DbModeSetting, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&ModeSetting{}); err != nil {
		l.Log.Errorf("failed to migrate database of scale connection: %v", err)
	}

	return &DbModeSetting{dbName: dbName, db: db}, nil
}

func (d *DbModeSetting) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbModeSetting) GetModeSetting(scaleMode uint) ([]ModeSetting, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var setting []ModeSetting
	d.db.Where("scale_mode=?", scaleMode).Find(&setting)
	return setting, nil
}

func (d *DbModeSetting) UpdateModeSetting(setting ModeSetting) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	rowAffected := d.db.Model(&setting).Where("scale_mode=?", setting.ScaleMode).Select("*").Updates(&setting).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleRec failed, maybe record not existing")
	}
	return nil
}
