package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbUserRec struct {
	dbName string
	db     *gorm.DB
}

type UserRec struct {
	RecId     uint `gorm:"primaryKey;autoincrement;not null"`
	Id        string
	Name      string
	IsFemale  bool
	Phone     string
	Remarks   string
	CreatedAt time.Time
}

func NewDbUserRec(dbName string) (*DbUserRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&UserRec{}); err != nil {
		l.Log.Errorf("failed to migrate database of user: %v", err)
	}
	return &DbUserRec{dbName: dbName, db: db}, nil
}

func (d *DbUserRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbUserRec) GetUserRecsList() ([]UserRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []UserRec
	d.db.Find(&recs)
	return recs, nil
}

func (d *DbUserRec) GetUserRecs(start int, quantity int) ([]UserRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []UserRec
	d.db.Limit(quantity).Offset(start).Find(&recs)
	return recs, nil
}

func (d *DbUserRec) InsertUserRec(rec UserRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbUserRec) UpdateUserRec(rec UserRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var oldRec UserRec
	if err := d.db.Where("rec_id = ?", rec.RecId).First(&oldRec).Error; err != nil {
		return errors.New("@UpdateUserRec failed, maybe record not existing")
	}
	return d.db.Save(&rec).Error
}

func (d *DbUserRec) DeleteUserRec(id uint) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var rec UserRec
	rec.RecId = id
	return d.db.Where("rec_id=?", id).Delete(&rec).Error
}
