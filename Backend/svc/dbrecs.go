package svc

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbScaleRec struct {
	dbName string
	db     *gorm.DB
}

type ScaleRec struct {
	RecId       uint   `gorm:"primaryKey;autoincrement;not null"`
	Id          string //秤记录的序号
	ScaleModel  string
	ScaleSn     string
	Plu         string
	ProductCode string
	ItemCode    string
	Category    string
	ProductName string
	GeneralUnit string
	TaxType     string
	Price       string
	UnitWeight  string
	Pretare     string
	LimitHigh   string
	LimitLow    string
	Weight      string
	WeightUnit  string
	UserNo      string
	UserName    string
	ScaleMode   string //0 =DC500 1=check Weigher 2=take in  3=take out
	ScaleName   string

	CreatedAt time.Time
}

type ScaleRecDetail struct {
	RecId      uint `gorm:"primaryKey;autoincrement;not null"`
	HeadId     uint //head中的recId
	No         uint //编号
	ScaleModel string
	ScaleSn    string
	Weight     string
	WeightUnit string
	ScaleName  string
	CreatedAt  time.Time
}

type ScaleRecInfo struct {
	Header  ScaleRec
	Details []ScaleRecDetail
}

// 将传入的字段转为数据库字段
func ConvertToDBColumnName(columnName string) string {
	var dbColumnName string
	switch columnName {
	case "Id":
		dbColumnName = "rec_id"
	case "ScaleModel":
		dbColumnName = "scale_model"
	case "ScaleSn":
		dbColumnName = "scale_sn"
	case "PLU":
		dbColumnName = "plu"
	case "Product Code":
		dbColumnName = "product_code"
	case "Item Code":
		dbColumnName = "item_code"
	case "Category":
		dbColumnName = "category"
	case "PLU Name":
		dbColumnName = "product_name"
	case "GeneralUnit":
		dbColumnName = "general_unit"
	case "TaxType":
		dbColumnName = "tax_type"
	case "Price":
		dbColumnName = "price"
	case "UnitWeight":
		dbColumnName = "unit_weight"
	case "Pretare":
		dbColumnName = "pretare"
	case "LimitHigh":
		dbColumnName = "limit_high"
	case "LimitLow":
		dbColumnName = "limit_low"
	case "Weight":
		dbColumnName = "weight"
	case "Weight Unit":
		dbColumnName = "weight_unit"
	case "User NO.":
		dbColumnName = "user_no"
	case "User Name":
		dbColumnName = "user_name"
	case "Scale Name":
		dbColumnName = "scale_name"
	case "Date Time":
		dbColumnName = "created_at"
	default:
		// 如果传入的字段名不匹配，默认使用 "created_at"
		dbColumnName = "created_at"
	}
	return dbColumnName
}

func NewDbScaleRec(dbName string) (*DbScaleRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(&ScaleRec{}, &ScaleRecDetail{}); err != nil {
		l.Log.Errorf("failed to migrate database of scale connection: %v", err)
	}
	return &DbScaleRec{dbName: dbName, db: db}, nil
}

func (d *DbScaleRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// GetScaleWgtRecsList 从数据库中分批获取数据
func (d *DbScaleRec) GetScaleWgtRecsList(model string, sn string, name string, offset int, limit int) ([]ScaleRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var recs []ScaleRec
	result := d.db.Where("scale_model = ?", model).
		Where("scale_sn = ?", sn).
		Offset(offset).
		Limit(limit).
		Find(&recs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query scale records: %w", result.Error)
	}

	return recs, nil
}

func (d *DbScaleRec) GetScaleRecsList(model string, sn string, name string, page string, pageSize string, columnName string, direction string) ([]ScaleRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	pageInt := 1
	var err error
	if page != "" {
		pageInt, err = strconv.Atoi(page)
		if err != nil {
			return nil, err
		}
	}

	pageSizeInt := 10
	if pageSize != "" {
		pageSizeInt, err = strconv.Atoi(pageSize)
		if err != nil {
			return nil, err
		}
	}

	offset := (pageInt - 1) * pageSizeInt

	var recs []ScaleRec
	query := d.db.Where("scale_model = ?", model).Where("scale_sn = ?", sn)
	dbColumnName := ConvertToDBColumnName(columnName)

	if dbColumnName != "" && direction != "" {
		sqlDirection := ""
		switch direction {
		case "ascending":
			sqlDirection = "ASC"
		case "descending":
			sqlDirection = "DESC"
		default:
			sqlDirection = "ASC"
		}

		if dbColumnName == "rec_id" {
			query = query.Order("CAST(" + dbColumnName + " AS INTEGER) " + sqlDirection)
		} else {
			query = query.Order(dbColumnName + " " + sqlDirection)
		}
	}

	var total int64
	if err := query.Model(&ScaleRec{}).Count(&total).Error; err != nil {
		return nil, err
	}

	totalPages := (int(total) + pageSizeInt - 1) / pageSizeInt
	if pageInt > totalPages {
		return []ScaleRec{}, nil
	}

	query = query.Offset(offset).Limit(pageSizeInt)
	if err := query.Find(&recs).Error; err != nil {
		return nil, err
	}

	return recs, nil
}

func (d *DbScaleRec) GetScaleRecs(model string, sn string, start int, quantity int) ([]ScaleRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var recs []ScaleRec
	d.db.Limit(quantity).Offset(start).Where("scale_model=?", model).Where("scale_sn=?", sn).Find(&recs)
	return recs, nil
}

func (d *DbScaleRec) InsertScaleRec(rec ScaleRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbScaleRec) UpdateScaleRec(rec ScaleRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	rowAffected := d.db.Model(&rec).Where("rec_id=?", rec.RecId).Updates(&rec).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleRec failed, mybe record not existing")
	}
	return nil
}

func (d *DbScaleRec) DeleteScaleRec(id uint) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var rec ScaleRec
	rec.RecId = id
	d.db.Where("rec_id=?", id).Delete((&rec))
	return nil
}

func (d *DbScaleRec) DeleteAllScaleRec(scaleModel string, scaleSn string) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var rec ScaleRec
	d.db.Where("scale_model = ? AND scale_sn = ?", scaleModel, scaleSn).Delete(&rec)
	return nil
}

func (d *DbScaleRec) NewDeleteAllRec() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	err := d.db.Exec("DELETE FROM scale_recs").Error
	if err != nil {
		return err
	}
	err = d.db.Exec("DELETE FROM scale_rec_details").Error
	if err != nil {
		return err
	}
	return nil
}

type PagedScaleRecInfo struct {
	ScaleRecInfos []ScaleRecInfo `json:"scale_rec_infos"`
	TotalCount    int64          `json:"total_count"`
}

func (d *DbScaleRec) NewGetScaleRecsList(page int, pageSize int, columnName string, direction string) (PagedScaleRecInfo, error) {
	if d == nil || d.db == nil {
		return PagedScaleRecInfo{}, errors.New("db connection is nil")
	}

	pageInt := page
	pageSizeInt := pageSize
	offset := (pageInt - 1) * pageSizeInt

	var query = d.db.Model(&ScaleRec{})
	dbColumnName := ConvertToDBColumnName(columnName)

	if dbColumnName != "" && direction != "" {
		sqlDirection := ""
		switch direction {
		case "ascending":
			sqlDirection = "ASC"
		case "descending":
			sqlDirection = "DESC"
		default:
			sqlDirection = "ASC"
		}

		if dbColumnName == "rec_id" {
			query = query.Order("CAST(" + dbColumnName + " AS INTEGER) " + sqlDirection)
		} else {
			query = query.Order(dbColumnName + " " + sqlDirection)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PagedScaleRecInfo{}, err
	}

	totalPages := (int(total) + pageSizeInt - 1) / pageSizeInt
	if pageInt > totalPages {
		return PagedScaleRecInfo{
			ScaleRecInfos: []ScaleRecInfo{},
			TotalCount:    total,
		}, nil
	}

	query = query.Offset(offset).Limit(pageSizeInt)
	var scaleRecs []ScaleRec
	if err := query.Find(&scaleRecs).Error; err != nil {
		return PagedScaleRecInfo{}, err
	}

	var scaleRecInfos []ScaleRecInfo
	for _, rec := range scaleRecs {
		var details []ScaleRecDetail
		if err := d.db.Where("head_id = ?", rec.RecId).Find(&details).Error; err != nil {
			return PagedScaleRecInfo{}, err
		}
		scaleRecInfos = append(scaleRecInfos, ScaleRecInfo{
			Header:  rec,
			Details: details,
		})
	}

	return PagedScaleRecInfo{
		ScaleRecInfos: scaleRecInfos,
		TotalCount:    total,
	}, nil
}

func (d *DbScaleRec) NewInsertScaleRec(rec ScaleRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	rec.ScaleModel = ""
	rec.ScaleName = ""
	rec.ScaleSn = ""

	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbScaleRec) FindMaxRecId() (uint, error) {
	if d == nil || d.db == nil {
		return 0, errors.New("db connection is nil")
	}
	var maxRecId uint
	if err := d.db.Model(&ScaleRec{}).Select("MAX(rec_id)").Scan(&maxRecId).Error; err != nil {
		return 0, err
	}
	if maxRecId == 0 {
		return 0, nil
	}
	return maxRecId, nil
}

func (d *DbScaleRec) InsertScaleRecDetail(rec ScaleRecDetail) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbScaleRec) GetAllScaleRecInfos() ([]ScaleRecInfo, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var scaleRecs []ScaleRec
	if err := d.db.Find(&scaleRecs).Error; err != nil {
		return nil, fmt.Errorf("failed to query scale records: %w", err)
	}

	var scaleRecInfos []ScaleRecInfo
	for _, rec := range scaleRecs {
		var details []ScaleRecDetail
		if err := d.db.Where("head_id = ?", rec.RecId).Find(&details).Error; err != nil {
			return nil, fmt.Errorf("failed to query scale record details: %w", err)
		}
		scaleRecInfos = append(scaleRecInfos, ScaleRecInfo{
			Header:  rec,
			Details: details,
		})
	}

	return scaleRecInfos, nil
}
