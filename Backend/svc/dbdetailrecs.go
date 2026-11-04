package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbDetailRec struct {
	dbName string
	db     *gorm.DB
}

type DetailRec struct {
	RecId              uint `gorm:"primaryKey;autoincrement;not null"`
	ScaleModel         string
	ScaleSn            string
	SettleAccountTimes string //流水号
	PluIndex           string //第几个PLU，相当于第几笔明细
	PluNum             string //PLu 编号
	PluTotalPrice      string //plu总价
	PluUnitPrice       string //plu单价
	PluTotalWeight     string //plu总重
	PluTare            string //plu扣重
	PluQuantity        string //plu总数
	PluUnit            string //单位 0-kg 1-100g 2-pcs
	PluTaxType         string //税类型 0-tax1 1-tax2 2-tax3
	PluReturnFlag      string //退货标志位 0-正常 1-退货 2-取消
	PluYear            string //年
	PluMonth           string //月
	PluDay             string //日
	PluTaxPrice        string //税价
	PluChangeType      string //找零类型 0-cash 1-card
	PluName            string //Plu 名字

	CreatedAt time.Time
}

type DetailTotal struct {
	RecId              uint `gorm:"primaryKey;autoincrement;not null"`
	ScaleModel         string
	ScaleSn            string
	SettleAccountTimes string //流水号
	TotalCount         string //打印次数
	PayPrice           string //付款总价
	TotalPrice         string //总价
	TaxKind            string //税类型 0-off 1-include 2-exclude

	CreatedAt time.Time
}

type DetailList struct {
	Total   DetailTotal
	Details []DetailRec
}

func NewDbDetailRec(dbName string) (*DbDetailRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&DetailRec{}, &DetailTotal{}); err != nil {
		l.Log.Errorf("failed to migrate database of detail recs: %v", err)
	}
	return &DbDetailRec{dbName: dbName, db: db}, nil
}

func (d *DbDetailRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbDetailRec) GetDetailRecsList() ([]DetailList, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var totals []DetailTotal
	err := d.db.Find(&totals).Error
	if err != nil {
		return nil, err
	}

	var detailLists []DetailList
	for _, total := range totals {
		var details []DetailRec
		err = d.db.Where("scale_model =? AND scale_sn =? AND settle_account_times =?", total.ScaleModel, total.ScaleSn, total.SettleAccountTimes).Find(&details).Error
		if err != nil {
			return nil, err
		}
		detailLists = append(detailLists, DetailList{Total: total, Details: details})
	}

	return detailLists, nil
}

func (d *DbDetailRec) InsertDetailRec(rec DetailRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbDetailRec) InsertDetailTotal(total DetailTotal) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&total)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbDetailRec) UpdateDetailRec(rec ScaleRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	rowAffected := d.db.Model(&rec).Where("rec_id=?", rec.RecId).Updates(&rec).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleRec failed, mybe record not existing")
	}
	return nil
}

func (d *DbDetailRec) DeleteDetailRec(id uint) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var rec ScaleRec
	rec.RecId = id
	return d.db.Where("rec_id=?", id).Delete(&rec).Error
}

func (d *DbDetailRec) DeleteAllDetailRec(scaleModel string, scaleSn string) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var rec ScaleRec
	return d.db.Where("scale_model = ? AND scale_sn = ?", scaleModel, scaleSn).Delete(&rec).Error
}
