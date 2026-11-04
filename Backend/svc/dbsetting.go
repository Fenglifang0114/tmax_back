package svc

import (
	l "tmaxsrv/log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbModeSetting struct {
	dbName string
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
	var err error
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// Migrate the schema
	if err = db.AutoMigrate(&ModeSetting{}); err != nil {
		l.Log.Debug("failed to migrate database of scale connection")
	}

	return &DbModeSetting{dbName: dbName}, nil
}

func (d *DbModeSetting) GetModeSetting(scaleMode uint) ([]ModeSetting, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// Migrate the schema
	if err = db.AutoMigrate(&ModeSetting{}); err != nil {
		l.Log.Debug("failed to migrate database of scale connection")
	}

	// 读取内容
	var setting []ModeSetting
	db.Where("scale_mode=?", scaleMode).Find(&setting)
	if len(setting) == 0 {
		defaultSetting := ModeSetting{
			ScaleMode:     scaleMode,
			DateFormat:    "0",
			DateSeparator: "-",
			StableTime:    "1",
			ZeroRange:     "1",
			RecMode:       "0",
			ScaleSn:       "",
			SaveMode:      "0",
			WgtMode:       0,
		}
		db.Create(&defaultSetting)
		setting = append(setting, defaultSetting)
	}
	return setting, nil
}

func (d *DbModeSetting) UpdateModeSetting(setting ModeSetting) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 明确指定要更新的字段
	rowAffected := db.Model(&setting).Where("scale_mode=?", setting.ScaleMode).Select("*").Updates(&setting).RowsAffected
	if rowAffected == 0 {
		return db.Create(&setting).Error
	}
	return nil
}
