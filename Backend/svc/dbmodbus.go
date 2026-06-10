package svc

import (
	"errors"
	l "tmaxsrv/log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbModbusServices struct {
	dbName string
}

func NewDbModbusServices(dbName string) (*DbModbusServices, error) {
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
	if err = db.AutoMigrate(&ModbusServiceInfo{}); err != nil {
		l.Log.Debug("failed to migrate database of modbus services")
	}
	return &DbModbusServices{dbName: dbName}, nil
}

func (d *DbModbusServices) GetModbusServiceList() ([]ModbusServiceInfo, error) {
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

	var services []ModbusServiceInfo
	db.Find(&services)
	return services, nil
}

func (d *DbModbusServices) InsertModbusService(service *ModbusServiceInfo) error {
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

	tx := db.Create(service)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbModbusServices) UpdateModbusService(service ModbusServiceInfo) error {
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

	rowAffected := db.Model(&service).Where("id=?", service.Id).Select("*").Updates(&service).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateModbusService failed, maybe record not existing")
	}

	return nil
}

func (d *DbModbusServices) DeleteModbusService(id uint) error {
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

	db.Where("id=?", id).Delete(&ModbusServiceInfo{})
	return nil
}
