package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbPluRec struct {
	dbName string
	db     *gorm.DB
}

type PluRec struct {
	RecId     uint `gorm:"primaryKey;autoincrement;not null"`
	Md5       string
	FileName  string
	CreatedAt time.Time
}

func NewDbPluRec(dbName string) (*DbPluRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&PluRec{}); err != nil {
		l.Log.Errorf("failed to migrate database of plu rec: %v", err)
	}
	return &DbPluRec{dbName: dbName, db: db}, nil
}

func (d *DbPluRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbPluRec) GetPluRecsList() ([]PluRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []PluRec
	d.db.Find(&recs)
	return recs, nil
}

func (d *DbPluRec) GetPluRecs(md5Str string) ([]PluRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []PluRec
	d.db.Where("md5=?", md5Str).Order("created_at desc").First(&recs)
	return recs, nil
}

func (d *DbPluRec) InsertPluRec(rec PluRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
