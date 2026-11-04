package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbWifiRec struct {
	dbName string
	db     *gorm.DB
}

type WifiRec struct {
	Id        uint `gorm:"primaryKey;autoincrement;not null"`
	Ssid      string
	Pwd       string
	CreatedAt time.Time
}

func NewDbWifiRec(dbName string) (*DbWifiRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&WifiRec{}); err != nil {
		l.Log.Errorf("failed to migrate database of wifi connection: %v", err)
	}
	return &DbWifiRec{dbName: dbName, db: db}, nil
}

func (d *DbWifiRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbWifiRec) GetWifiRecsList() ([]WifiRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []WifiRec
	d.db.Find(&recs)
	return recs, nil
}

func (d *DbWifiRec) GetWifiRecs(start int, quantity int) ([]WifiRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []WifiRec
	d.db.Limit(quantity).Offset(start).Find(&recs)
	return recs, nil
}

func (d *DbWifiRec) InsertWifiRec(rec WifiRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	d.db.Where("ssid=?", rec.Ssid).Delete((&rec)) //先删除一样的

	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbWifiRec) UpdateWifiRec(rec WifiRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var oldRec WifiRec
	if err := d.db.Where("ssid = ?", rec.Ssid).First(&oldRec).Error; err != nil {
		return errors.New("@UpdateWifiRec failed, mybe record not existing")
	}
	return d.db.Save(&rec).Error
}

func (d *DbWifiRec) DeleteWifiRec(id uint) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var rec WifiRec
	rec.Id = id
	return d.db.Where("rec_id=?", id).Delete(&rec).Error
}
