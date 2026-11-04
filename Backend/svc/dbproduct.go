package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbProductRec struct {
	dbName string
	db     *gorm.DB
}

type ProductRec struct {
	RecId       int `gorm:"primaryKey;autoincrement;not null"`
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
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreateBy    int  `gorm:"default:1"`
	UpdateBy    int  `gorm:"default:1"`
	Enabled     bool `gorm:"default:true"`
	CreateUser  string
	UpdateUser  string
}

type FieldDisplaySetting struct {
	Id        int `gorm:"primaryKey;autoincrement;not null"`
	FieldName string
	Sequence  int
	UpdatedAt time.Time
	UserName  string
}

func NewDbProductRec(dbName string) (*DbProductRec, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}

	if err = db.AutoMigrate(&ProductRec{}, &FieldDisplaySetting{}); err != nil {
		l.Log.Errorf("failed to migrate database of scale connection: %v", err)
	}

	db.Exec("UPDATE product_recs SET updated_at = created_at, create_by = 1, update_by = 1, enabled = true WHERE updated_at IS NULL")

	return &DbProductRec{
		dbName: dbName,
		db:     db,
	}, nil
}

func (d *DbProductRec) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}



// CheckPluExist 检查PLU是否存在
func (d *DbProductRec) CheckPluExist(id int, plu string) (bool, error) {
	if d == nil || d.db == nil {
		return false, errors.New("db connection is nil")
	}

	var recs []ProductRec
	query := d.db.Model(&ProductRec{}).Where("plu = ?", plu)
	query.Find(&recs)
	if id == 0 {
		return len(recs) > 0, nil
	}

	if len(recs) > 0 {
		return recs[0].RecId != id, nil
	}

	return false, nil
}

func (d *DbProductRec) GetProductRecsList() ([]ProductRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var recs []ProductRec
	d.db.Find(&recs)
	return recs, nil
}

// 获取分页数据
func (d *DbProductRec) GetPluByPage(page, pageSize int, fieldName, direction string, search ProductQuery) ([]ProductRec, int, error) {
	if d == nil || d.db == nil {
		return nil, 0, errors.New("db connection is nil")
	}

	query := d.db.Model(&ProductRec{})

	if search.Plu != "" {
		query = query.Where("plu LIKE ?", "%"+search.Plu+"%")
	}
	if search.PluName != "" {
		query = query.Where("product_name LIKE ?", "%"+search.PluName+"%")
	}
	if search.Category != "" {
		query = query.Where("category LIKE ?", "%"+search.Category+"%")
	}

	if search.SetEnabled {
		if search.Enabled {
			query = query.Where("enabled = ?", true)
		} else if !search.Enabled {
			query = query.Where("enabled = ?", false)
		}
	}

	var total int64
	query.Count(&total)

	if fieldName != "" {
		orderDirection := "ASC"
		if direction == "desc" {
			orderDirection = "DESC"
		}
		switch fieldName {
		case "plu":
			query = query.Order("CAST(plu AS INTEGER) " + orderDirection)
		case "price":
			query = query.Order("CAST(price AS REAL) " + orderDirection)
		case "limit_high":
			query = query.Order("CAST(limit_high AS REAL) " + orderDirection)
		case "limit_low":
			query = query.Order("CAST(limit_low AS REAL) " + orderDirection)
		case "unit_weight":
			query = query.Order("CAST(unit_weight AS REAL) " + orderDirection)
		case "pretare":
			query = query.Order("CAST(pretare AS REAL) " + orderDirection)
		case "tax_type":
			query = query.Order("CAST(tax_type AS INTEGER) " + orderDirection)
		default:
			query = query.Order(fieldName + " " + orderDirection)
		}
	}

	var recs []ProductRec
	result := query.Limit(pageSize).Offset((page - 1) * pageSize).Find(&recs)
	if result.Error != nil {
		l.Log.Errorf("failed to query products: %v", result.Error)
		return nil, 0, result.Error
	}

	return recs, int(total), nil
}

// 导出产品列表
func (d *DbProductRec) GetExportProductList(search ProductQuery) ([]ProductRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	query := d.db.Model(&ProductRec{})

	if search.Plu != "" {
		query = query.Where("plu LIKE ?", "%"+search.Plu+"%")
	}
	if search.PluName != "" {
		query = query.Where("product_name LIKE ?", "%"+search.PluName+"%")
	}
	if search.Category != "" {
		query = query.Where("category LIKE ?", "%"+search.Category+"%")
	}

	if search.SetEnabled {
		if search.Enabled {
			query = query.Where("enabled = ?", true)
		} else if !search.Enabled {
			query = query.Where("enabled = ?", false)
		}
	}

	var recs []ProductRec
	result := query.Find(&recs)
	if result.Error != nil {
		l.Log.Errorf("failed to query products: %v", result.Error)
		return nil, result.Error
	}
	return recs, nil
}

func (d *DbProductRec) GetProductRecs(start int, quantity int) ([]ProductRec, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var recs []ProductRec
	d.db.Limit(quantity).Offset(start).Find(&recs)
	return recs, nil
}

func (d *DbProductRec) InsertProductRec(rec ProductRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	tx := d.db.Create(&rec)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbProductRec) Insert100ProductsWithGorm(products []ProductRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	var pluList []string
	for _, p := range products {
		pluList = append(pluList, p.Plu)
	}

	var existPluList []string
	err := d.db.Model(&ProductRec{}).Where("plu IN ?", pluList).Pluck("plu", &existPluList).Error
	if err != nil {
		return err
	}

	existPluSet := make(map[string]bool)
	for _, plu := range existPluList {
		existPluSet[plu] = true
	}

	filteredProducts := make([]ProductRec, 0, len(products))
	for _, p := range products {
		if !existPluSet[p.Plu] {
			filteredProducts = append(filteredProducts, p)
		}
	}

	if len(filteredProducts) == 0 {
		return nil
	}

	result := d.db.Create(filteredProducts)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (d *DbProductRec) UpdateProductRec(rec ProductRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	var oldRec ProductRec
	d.db.Where("rec_id = ?", rec.RecId).First(&oldRec)
	if oldRec.RecId == 0 {
		return errors.New("record not found")
	}

	result := d.db.Model(&ProductRec{}).Where("rec_id = ?", rec.RecId).Updates(map[string]interface{}{
		"plu":          rec.Plu,
		"product_code": rec.ProductCode,
		"item_code":    rec.ItemCode,
		"category":     rec.Category,
		"product_name": rec.ProductName,
		"general_unit": rec.GeneralUnit,
		"tax_type":     rec.TaxType,
		"price":        rec.Price,
		"unit_weight":  rec.UnitWeight,
		"pretare":      rec.Pretare,
		"limit_high":   rec.LimitHigh,
		"limit_low":    rec.LimitLow,
		"update_by":    rec.UpdateBy,
		"enabled":      oldRec.Enabled,
		"updated_at":   time.Now(),
		"created_at":   oldRec.CreatedAt,
		"create_by":    oldRec.CreateBy,
		"update_user":  rec.UpdateUser,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

// 获取recId最大的记录，也就是最后一个新增的记录
func (d *DbProductRec) GetLastProductRec() (ProductRec, error) {
	if d == nil || d.db == nil {
		return ProductRec{}, errors.New("db connection is nil")
	}

	var lastRec ProductRec
	d.db.Order("rec_id desc").First(&lastRec)
	return lastRec, nil
}

// 批量更新数据
func (d *DbProductRec) BatchUpdateProductRec(multiRec []ProductRec) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	tx := d.db.Begin()
	for _, rec := range multiRec {
		var targetRec ProductRec
		if err := tx.Where("plu =?", rec.Plu).First(&targetRec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&rec).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return err
			}
		} else {
			rec.RecId = targetRec.RecId
			if err := tx.Save(&rec).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit().Error
}

func (d *DbProductRec) DeleteProductRec(id []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Where("rec_id IN ?", id).Delete(&ProductRec{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (d *DbProductRec) DeleteAllProductRecs() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Exec("DELETE FROM product_recs")
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// 批量启用或者停用PLU
func (d *DbProductRec) UpdateProductRecEnabled(pluList []int, enabled bool, updateUser string) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Model(&ProductRec{}).Where("rec_id IN ?", pluList).Updates(map[string]interface{}{
		"enabled":     enabled,
		"update_user": updateUser,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (d *DbProductRec) GetPluSetting() (map[int]string, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var results []FieldDisplaySetting
	err := d.db.Model(&FieldDisplaySetting{}).Order("sequence ASC").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	fieldDisplaySetting := make(map[int]string)
	for _, r := range results {
		fieldDisplaySetting[r.Sequence] = r.FieldName
	}

	return fieldDisplaySetting, nil
}

// 设置字段显示
func (d *DbProductRec) SetPluSetting(fieldDisplaySetting []string, updateUser string) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Exec("DELETE FROM field_display_settings")
	if result.Error != nil {
		return result.Error
	}
	sequence := 1
	for _, r := range fieldDisplaySetting {
		if r == "plu" || r == "productName" {
			continue
		}
		d.db.Create(&FieldDisplaySetting{
			Sequence:  sequence,
			FieldName: r,
			UpdatedAt: time.Now(),
			UserName:  updateUser,
		})
		sequence++
	}

	return nil
}
