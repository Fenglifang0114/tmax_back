package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbScaleInfos struct {
	dbName string
	db     *gorm.DB
}

type ScaleInfos struct {
	RecId      uint `gorm:"primaryKey;autoincrement;not null"`
	ScaleModel string
	ScaleSn    string
	ScaleName  string
	CreatedAt  time.Time
}

func NewDbScaleInfos(dbName string) (*DbScaleInfos, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&ScaleInfos{}); err != nil {
		l.Log.Errorf("failed to migrate database of scale infos: %v", err)
	}
	return &DbScaleInfos{dbName: dbName, db: db}, nil
}

func (d *DbScaleInfos) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbScaleInfos) GetScaleInfo(model string, sn string) ([]ScaleInfos, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var infos []ScaleInfos
	d.db.Where("scale_model=?", model).Where("scale_sn=?", sn).Find(&infos)
	return infos, nil
}

func (d *DbScaleInfos) InsertScaleInfos(rec ScaleInfos) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbScaleInfos) UpdateScaleInfos(rec ScaleInfos) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	rowAffected := d.db.Model(&rec).Where("rec_id=?", rec.RecId).Updates(&rec).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleInfos failed, mybe record not existing")
	}
	return nil
}

func (d *DbScaleInfos) DeleteScaleInfos(id uint) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var info ScaleInfos
	info.RecId = id
	return d.db.Where("rec_id=?", id).Delete(&info).Error
}

func (d *DbScaleInfos) DeleteAllScaleInfos() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	err := d.db.Migrator().DropTable(&ScaleInfos{})
	if err != nil {
		l.Log.Errorf("failed to drop database: %v", err)
		return err
	}
	return nil
}
