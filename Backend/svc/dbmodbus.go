package svc

import (
	"errors"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbModbusServices struct {
	dbName string
	db     *gorm.DB
}

func NewDbModbusServices(dbName string) (*DbModbusServices, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&ModbusServiceInfo{}); err != nil {
		l.Log.Errorf("failed to migrate database of modbus services: %v", err)
	}
	return &DbModbusServices{dbName: dbName, db: db}, nil
}

func (d *DbModbusServices) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbModbusServices) GetModbusServiceList() ([]ModbusServiceInfo, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var services []ModbusServiceInfo
	d.db.Find(&services)
	return services, nil
}

func (d *DbModbusServices) InsertModbusService(service *ModbusServiceInfo) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(service)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbModbusServices) UpdateModbusService(service ModbusServiceInfo) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	rowAffected := d.db.Model(&service).Where("id=?", service.Id).Select("*").Updates(&service).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateModbusService failed, maybe record not existing")
	}

	return nil
}

func (d *DbModbusServices) DeleteModbusService(id uint) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	d.db.Where("id=?", id).Delete(&ModbusServiceInfo{})
	return nil
}
