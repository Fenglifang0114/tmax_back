package svc

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbFormulaInfo struct {
	dbName string
}

// RawMaterialCategory 原料类别表
type RawMaterialCategory struct {
	// 类别ID（主键）
	CategoryID int `gorm:"primaryKey;autoincrement;not null"`
	// 类别名称
	CategoryName string `gorm:"not null"`
}

type UploadServerInfo struct {
	RecId     int `gorm:"primaryKey;autoincrement;not null"`
	Ip        string
	ShareName string
	Username  string
	Password  string
	Enable    bool `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}

type RawMaterialOutput struct {
	MaterialId string
	Output     int
}

// RawMaterial 原料表
type RawMaterial struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 原料编号(主键)
	MaterialID string `gorm:"not null"`
	// 原料名称
	MaterialName string `gorm:"not null"`
	// 原料类别
	CategoryID int `gorm:"not null"`
	// 成分说明
	Ingredient string
	// 创建时间
	CreatedAt time.Time
	// 修改时间
	UpdatedAt time.Time
	// 原料创建人
	CreatedBy string
	// 原料修改人
	UpdatedBy string
	// 备注
	Remark string
	// 备注1
	Remark1   string
	ScaleId   int
	CheckCode string // 原料的条码
	Output    int    `gorm:"default:0"` // 输出口
}

// FormulaCategory 配方类别表
type FormulaCategory struct {
	// 类别ID（主键）
	CategoryID int `gorm:"primaryKey;autoincrement;not null"`
	// 类别名称
	CategoryName string `gorm:"not null"`
}

// FormulaHeader 配方头表
type FormulaHeader struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`

	FormulaKey int `gorm:"default:0"` // 设置为唯一标识主键
	//加key的作用是不管如何删除和修改，这次生成的key是唯一的。便于查找关于此配方的称重记录  后来不需要分那么清楚，就没有用了
	// 配方编号
	FormulaID string `gorm:"not null"`
	// 配方名称
	FormulaName string `gorm:"not null"`
	// 配方类别
	CategoryID int `gorm:"not null"`
	// 配方模式
	FormulaMode string
	// 配方单位
	FormulaUnit string
	// 配方总重量
	TotalWeight float64
	// 原料数量
	MaterialCount int
	// 是否加密
	IsEncrypted bool
	//是否有容器
	NeedContainer bool
	// 创建时间
	CreatedAt time.Time
	// 修改时间
	UpdatedAt time.Time
	// 配方创建人
	CreatedBy string
	// 配方修改人
	UpdatedBy string
	// 备注
	Remark string
	// 备注1
	Remark1 string
	// 备注2
	Remark2 string
	// 备注3
	Remark3 string
	//是否删除
	IsUsed bool `gorm:"default:true"` // 默认值为 true，表示未删除

	//是否最新的
	IsLatest bool `gorm:"default:true"` // 默认值为 true，表示未修改
	//配方条码
	FormulaBarcode string
}

// 配方头表的 BeforeSave 钩子
func (f *FormulaHeader) BeforeSave(tx *gorm.DB) error {
	if f.FormulaKey == 0 {
		// 解引用并赋值
		f.FormulaKey = f.RecId
	}

	return nil
}

// 配方头表包含类别名称
type FormulaHeaderWithType struct {
	FormulaHeader       FormulaHeader
	FormulaCategoryName string
}

// FormulaDetail 配方明细表
type FormulaDetail struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 配方编号（主键）
	FormulaRecID int `gorm:"not null"` //改为配方的RecId
	// 原料编号（主键）
	MaterialID string `gorm:"not null"`
	// 原料重量
	MaterialWeight float64
	// 原料百分比
	MaterialPercentage float64
	// 序号
	Sequence int
	// 允许误差
	AllowableError float64
	// 备注
	Remark string
	// 备注1
	Remark1 string
}

// FormulaWgtRecHeader 配方称重记录头表
type FormulaWgtRecHeader struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 记录编号（主键）
	RecordID string `gorm:"not null"`
	// 记录保存时间
	RecordSaveTime time.Time
	// 记录操作员
	Operator string
	//配方唯一标识
	FormulaKey int `gorm:"not null;default:0"`
	// 配方编号
	FormulaID string
	// 配方名称
	FormulaName string
	// 配方类别
	FormulaTypeId int
	// 配方类别名称
	FormulaTypeName string
	// 配方模式
	FormulaMode string
	// 配方总重量
	TotalWeight float64
	//实际总重量
	ActualTotalWeight float64
	// 总重量单位
	TotalWeightUnit string
	// 原料数量
	MaterialCount int
	// 误差
	Error float64
	// 是否达标
	IsQualified string
	// 配方实际需要的重量
	ActualFmaTotalWgt float64
	// 是否加密
	IsEncrypted bool
	//是否有容器
	NeedContainer bool
	// 配方创建时间
	FormulaCreatedAt time.Time
	// 配方修改时间
	FormulaUpdatedAt time.Time
	// 配方创建人
	FormulaCreatedBy string
	// 配方修改人
	FormulaUpdatedBy string
	// 配方备注
	FormulaRemark string
	// 配方备注1
	FormulaRemark1 string
	//记录备注 备用字段
	RecRemark string
	//记录备注1
	RecRemark1 string
	ScaleId    int
	ScaleName  string
	ScaleModel string
	ScaleSn    string
	//配方条码
	FormulaBarcode string
}

// FormulaWgtRecDetail 配方称重记录详情表
type FormulaWgtRecDetail struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 记录编号（主键）
	RecordID string `gorm:"not null"`
	// 原料编号（主键）
	MaterialID string `gorm:"not null"`
	// 原料名称
	MaterialName string
	// 原料类别
	MaterialTypeID int
	// 原料类别名称
	MaterialTypeName string
	// 原料成分说明
	Ingredient string
	// 原料创建时间
	MaterialCreatedAt time.Time
	// 原料修改时间
	MaterialUpdatedAt time.Time
	// 原料创建人
	MaterialCreatedBy string
	// 原料修改人
	MaterialUpdatedBy string
	// 原料备注
	MaterialRemark string
	// 原料备注1
	MaterialRemark1 string
	// 原料重量
	MaterialWeight float64
	// 原料百分比
	MaterialPercentage float64
	// 序号
	Sequence int
	// 允许误差
	AllowableError float64
	// 配方原料备注
	FormulaRemark string
	// 配方原料备注1
	FormulaRemark1 string
	//目标重量
	TargetWgt float64
	// 实际重量
	ActualWeight float64
	// 实际百分比
	ActualPercentage float64
	// 实际误差重量
	ActualErrorWgt float64
	// 实际误差百分比
	ActualErrorPct float64
	// 达标情况
	IsQualified string
	// 最后称重时间
	LastWeighingTime time.Time
	//记录备注 备用字段
	RecRemark string
	//记录备注1
	RecRemark1 string
	//秤ID
	ScaleId int
	//秤名字
	ScaleName string
	//秤型号
	ScaleModel string
	//秤序列号
	ScaleSn string
	//原料条码
	CheckCode string
}

type FormulaList struct {
	Header  FormulaHeader
	Details []FormulaDetail
}

type FormulaWgtRecList struct {
	Header  FormulaWgtRecHeader
	Details []FormulaWgtRecDetail
}

// 初始化数据库连接和表结构
func NewFormulaInfo(dbName string) (*DbFormulaInfo, error) {
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// 自动迁移表结构
	if err := db.AutoMigrate(
		&RawMaterialCategory{},
		&RawMaterial{},
		&FormulaCategory{},
		&FormulaHeader{},
		&FormulaDetail{},
		&FormulaWgtRecHeader{},
		&FormulaWgtRecDetail{},
		&SetAutoNext{},
		&SetOutputPort{},
		&SetInputPort{},
		&DrafFmaWgtRecHeader{},
		&DrafFmaWgtRecDetail{},
		&SetReportPrint{},
		&UploadServerInfo{},
	); err != nil {
		return nil, err
	}

	// 启用 WAL 模式与高性能 PRAGMA 缓存
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA cache_size = -64000;")
	db.Exec("PRAGMA synchronous = NORMAL;")
	db.Exec("PRAGMA temp_store = MEMORY;")

	// 创建高性能非破坏性索引
	db.Exec("CREATE INDEX IF NOT EXISTS idx_formula_header_search ON formula_headers(is_used, is_latest, category_id, is_encrypted);")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_formula_detail_header ON formula_details(formula_rec_id);")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_raw_material_search ON raw_materials(category_id, material_id);")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_wgt_rec_header_search ON formula_wgt_rec_headers(record_id, formula_id, created_at);")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_wgt_rec_detail_search ON formula_wgt_rec_details(record_id);")

	// 检查并更新现有数据
	db.Exec("UPDATE formula_headers SET formula_key = rec_id WHERE formula_key = 0")
	db.Exec("UPDATE formula_headers SET formula_barcode = formula_id WHERE formula_barcode IS NULL OR formula_barcode = ''")
	db.Exec("UPDATE formula_wgt_rec_headers SET formula_barcode = formula_id WHERE formula_barcode IS NULL OR formula_barcode = ''")
	db.Exec("UPDATE formula_wgt_rec_details SET check_code = material_id WHERE check_code IS NULL OR check_code = ''")
	db.Exec("UPDATE raw_materials SET check_code = material_id WHERE check_code IS NULL OR check_code = ''")

	info := &DbFormulaInfo{dbName: dbName}
	// 通过实例调用方法
	if err := info.UpdateFormulaKeyInWgtRecHeader(); err != nil {
		return nil, err
	}

	// 检查 RawMaterialCategory 表中是否存在 CategoryID = 0 的记录，不存在则插入
	var rawMaterialCategoryCount int64
	db.Model(&RawMaterialCategory{}).Where("category_name = ?", "-").Count(&rawMaterialCategoryCount)
	if rawMaterialCategoryCount == 0 {
		err := db.Create(&RawMaterialCategory{
			CategoryID:   0,
			CategoryName: "-",
		}).Error
		if err != nil {
			return nil, err
		}
	}

	// 检查 FormulaCategory 表中是否存在 CategoryID = 0 的记录，不存在则插入
	var formulaCategoryCount int64
	db.Model(&FormulaCategory{}).Where("category_name = ?", "-").Count(&formulaCategoryCount)

	if formulaCategoryCount == 0 {
		err := db.Create(&FormulaCategory{
			CategoryID:   0,
			CategoryName: "-",
		}).Error
		if err != nil {
			return nil, err
		}
	}
	var rawMaterialCategory RawMaterialCategory
	err = db.Where("category_name = ?", "-").First(&rawMaterialCategory).Error
	if err == nil {
		// 找到记录，修改 CategoryID 为 0
		err = db.Model(&RawMaterialCategory{}).Where("category_name = ?", "-").Update("category_id", 0).Error
		if err != nil {
			return nil, err
		}
	}

	var formulaCategory FormulaCategory
	err = db.Where("category_name = ?", "-").First(&formulaCategory).Error
	if err == nil {
		// 找到记录，修改 CategoryID 为 0
		err = db.Model(&formulaCategory).Where("category_name = ?", "-").Update("category_id", 0).Error
		if err != nil {
			return nil, err
		}
	}

	//新增一条设置
	if err := info.CreateSetAutoNext(); err != nil {
		return nil, err
	}

	//新增一条设置
	if err := info.CreateSetReportPrint(); err != nil {
		return nil, err
	}

	//新增设置 配方秤输出口设置
	if err := info.CreateSetOutputPort(); err != nil {
		return nil, err
	}

	//新增设置 配方秤输入口设置
	if err := info.CreateSetInputPort(); err != nil {
		return nil, err
	}

	return info, nil
}

// 更新 FormulaWgtRecHeader 中 formula_key 为 0 的记录
func (d *DbFormulaInfo) UpdateFormulaKeyInWgtRecHeader() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 使用 gorm.Expr 实现子查询
	subQueryExpr := gorm.Expr("(SELECT rec_id FROM formula_headers WHERE formula_id = formula_wgt_rec_headers.formula_id AND is_used = 1 AND is_latest = 1)")

	// 更新 FormulaWgtRecHeader 中 formula_key 为 0 的记录
	err = tx.Model(&FormulaWgtRecHeader{}).
		Where("formula_key = ?", 0).
		Update("formula_key", subQueryExpr).
		Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 删除未使用的原料类别
func (d *DbFormulaInfo) DeleteUnusedRawMaterialCategories() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	return db.Where("NOT EXISTS (SELECT 1 FROM raw_materials WHERE raw_materials.category_id = raw_material_categories.category_id)").
		Delete(&RawMaterialCategory{}).Error
}

// 删除未使用的配方类别
func (d *DbFormulaInfo) DeleteUnusedFormulaCategories() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	return db.Where("NOT EXISTS (SELECT 1 FROM formula_headers WHERE formula_headers.category_id = formula_categories.category_id)").
		Delete(&FormulaCategory{}).Error
}

// 新增配方类别
func (d *DbFormulaInfo) CreateFormulaCategory(category FormulaCategory) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&category).Error
}

// 新增配方类别列表
func (d *DbFormulaInfo) CreateFormulaCategoryList(categories []string) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	for _, name := range categories {
		// 检查是否已存在相同名称的类别
		var count int64
		if err := tx.Model(&FormulaCategory{}).Where("category_name = ?", name).Count(&count).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 如果不存在，则添加新类别
		if count == 0 {
			category := FormulaCategory{
				CategoryName: name,
			}
			if err := tx.Create(&category).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// 获取所有配方类别
func (d *DbFormulaInfo) GetAllFormulaCategories() ([]FormulaCategory, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var categories []FormulaCategory
	err = db.Find(&categories).Error
	return categories, err
}

// 根据 ID 获取配方类别
func (d *DbFormulaInfo) GetFormulaCategoryByID(id int) (FormulaCategory, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return FormulaCategory{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return FormulaCategory{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var category FormulaCategory
	err = db.First(&category, id).Error
	return category, err
}

// 更新配方类别
func (d *DbFormulaInfo) UpdateFormulaCategory(category FormulaCategory) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Save(&category).Error
}

// 删除配方类别
func (d *DbFormulaInfo) DeleteFormulaCategory(name string) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Delete(&FormulaCategory{}, "category_name  = ?", name).Error
}

// 新增原料类别
func (d *DbFormulaInfo) CreateRawMaterialCategory(category RawMaterialCategory) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	tx := db.Create(&category)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

// 新增原料类别
func (d *DbFormulaInfo) CreateRawCategoryList(categories []string) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 遍历所有类别名称
	for _, categoryName := range categories {
		// 查询名字是否存在，不存在就新增，存在就跳过
		var count int64
		db.Model(&RawMaterialCategory{}).Where("category_name = ?", categoryName).Count(&count)

		if count == 0 {
			// 如果类别不存在，则创建新类别
			category := RawMaterialCategory{
				CategoryName: categoryName,
			}
			tx := db.Create(&category)
			if tx.Error != nil {
				return tx.Error
			}
		}
		// 如果类别已存在，则跳过继续处理下一个
	}

	return nil
}

// 获取所有原料类别
func (d *DbFormulaInfo) GetAllRawMaterialCategories() ([]RawMaterialCategory, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []RawMaterialCategory{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []RawMaterialCategory{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var categories []RawMaterialCategory

	err = db.Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// 根据 ID 获取原料类别
func (d *DbFormulaInfo) GetRawMaterialCategoryByID(id int) (RawMaterialCategory, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return RawMaterialCategory{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return RawMaterialCategory{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var category RawMaterialCategory
	err = db.First(&category, id).Error

	if err != nil {
		return category, err
	}
	return category, err
}

// 更新原料类别
func (d *DbFormulaInfo) UpdateRawMaterialCategory(category RawMaterialCategory) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	return db.Save(&category).Error
}

// 删除原料类别
func (d *DbFormulaInfo) DeleteRawMaterialCategory(name string) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	return db.Delete(&RawMaterialCategory{}, "category_name = ?", name).Error
}

// 新增原料
func (d *DbFormulaInfo) CreateRawMaterial(material RawMaterial) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	return db.Create(&material).Error
}

// 获取最大的原料ID
func (d *DbFormulaInfo) GetMaxRawRecId() (int, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return 0, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return 0, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var maxId int
	err = db.Raw("SELECT MAX(rec_id) FROM raw_materials").Scan(&maxId).Error
	if err != nil {
		return 0, err
	}
	return maxId, nil
}

// 根据FMAID获取原料数据的输出口
// 根据配方的ID 找到原料ID，再找到原料的输出口
func (d *DbFormulaInfo) GetRawOutputByFmaId(fmaId string) ([]RawMaterialOutput, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []RawMaterialOutput{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []RawMaterialOutput{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var outputs []RawMaterialOutput

	// 1. 先根据 fmaId 查询配方头表，获取配方记录ID (RecId)
	var formulaHeader FormulaHeader
	err = db.Where("formula_id = ? AND is_used = ? AND is_latest = ?", fmaId, true, true).First(&formulaHeader).Error
	if err != nil {
		return []RawMaterialOutput{}, err
	}

	// 2. 如果找不到配方，返回空结果
	if formulaHeader.RecId == 0 {
		return outputs, nil
	}

	// 3. 联查配方明细表和原料表，获取原料ID和对应的输出口
	err = db.Table("formula_details").
		Select("raw_materials.material_id, raw_materials.output").
		Joins("left join raw_materials on formula_details.material_id = raw_materials.material_id").
		Where("formula_details.formula_rec_id = ?", formulaHeader.RecId).
		Scan(&outputs).Error

	if err != nil {
		return []RawMaterialOutput{}, err
	}

	return outputs, nil
}

// 批量新增原料
func (d *DbFormulaInfo) CreateRawMaterialList(materials []RawMaterial) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 批量创建原料
	if err := tx.Create(&materials).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 获取所有原料
func (d *DbFormulaInfo) GetAllRawMaterials() ([]RawMaterial, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []RawMaterial{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []RawMaterial{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var materials []RawMaterial
	err = db.Find(&materials).Error
	return materials, err
}

// 获取原料
func (d *DbFormulaInfo) GetRawMaterial(recId int) ([]RawMaterial, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []RawMaterial{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []RawMaterial{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var materials []RawMaterial
	err = db.Find(&materials).Error
	return materials, err
}

// 根据 ID 获取原料
func (d *DbFormulaInfo) GetRawMaterialByID(id int) ([]RawMaterial, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []RawMaterial{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []RawMaterial{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var materials []RawMaterial
	err = db.Where("rec_id = ?", id).Find(&materials).Error
	return materials, err
}

// 更新原料
func (d *DbFormulaInfo) UpdateRawMaterial(material RawMaterial) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 忽略 CreatedAt 字段，根据 RecId 更新原料信息，并更新 UpdatedAt 为当前时间
	return db.Model(&RawMaterial{}).Where("rec_id = ?", material.RecId).Omit("CreatedAt").Updates(map[string]interface{}{
		"material_name": material.MaterialName,
		"category_id":   material.CategoryID,
		"ingredient":    material.Ingredient,
		"updated_at":    time.Now(),
		"updated_by":    material.UpdatedBy,
		"remark":        material.Remark,
		"remark1":       material.Remark1,
		"scale_id":      material.ScaleId,
		"check_code":    material.CheckCode,
		"output":        material.Output,
	}).Error
}

// 删除原料
func (d *DbFormulaInfo) DeleteRawMaterial(recId int) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 修改为根据 recId 删除原料
	return db.Where("rec_id = ?", recId).Delete(&RawMaterial{}).Error
}

// 删除所有原料 - 只删除未被使用的原料
func (d *DbFormulaInfo) DeleteAllRawMaterials(recIds []int) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 查找未被配方明细使用的原料ID
	var unusedMaterialIDs []int
	err = tx.Model(&RawMaterial{}).
		Where("rec_id IN ? AND material_id NOT IN (SELECT DISTINCT material_id FROM formula_details)", recIds).
		Pluck("rec_id", &unusedMaterialIDs).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// 只删除未被使用的原料
	if len(unusedMaterialIDs) > 0 {
		if err := tx.Where("rec_id IN ?", unusedMaterialIDs).Delete(&RawMaterial{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// 新增配方列表
func (d *DbFormulaInfo) InsertFormulaList(list []FmaDataImportInfo) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	//新增配方头和明细部分
	for i := range list {
		if err := tx.Create(&list[i].Head).Error; err != nil {
			// 回滚事务
			tx.Rollback()
			return err
		}
		// 新增配方明细
		for j := range list[i].Detail {
			if err := tx.Create(&list[i].Detail[j]).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				return err
			}
		}

	}

	// 提交事务
	return tx.Commit().Error
}

// 根据配方编号查询 FormulaList
func (d *DbFormulaInfo) GetFormulaListByFormulaID(formulaID string) (FormulaList, error) {
	var list FormulaList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return list, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return list, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 查询配方头信息
	var header FormulaHeader
	err = db.Where("formula_id = ? AND is_used = ? AND is_latest = ?", formulaID, true, true).First(&header).Error
	if err != nil {
		return list, err
	}

	list.Header = header

	// 查询配方明细信息
	var formulaDetails []FormulaDetail
	err = db.Where("formula_rec_id = ?", header.RecId).Find(&formulaDetails).Error
	if err != nil {
		return list, err
	}

	// 组合成 FormulaDetailWithRaw
	list.Details = formulaDetails

	return list, nil
}

// 根据rec_id 查询配方信息
func (d *DbFormulaInfo) GetFormulaByRecId(recId int) (FormulaList, error) {
	var list FormulaList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return list, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return list, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var header FormulaHeader
	// 查询配方头信息
	err = db.Where("rec_id = ? AND is_used = ? AND is_latest = ?", recId, true, true).First(&header).Error
	if err != nil {
		return list, err
	}
	list.Header = header

	// 查询配方明细信息
	var formulaDetails []FormulaDetail
	err = db.Where("formula_rec_id = ?", header.RecId).Find(&formulaDetails).Error
	if err != nil {
		return list, err
	}

	// 组合成 FormulaDetailWithRaw
	list.Details = formulaDetails

	return list, nil
}

// 根据记录编号查询 FormulaWgtRecList
func (d *DbFormulaInfo) GetFormulaWgtRecListByRecordID(recordID string) (FormulaWgtRecList, error) {
	var list FormulaWgtRecList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return list, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return list, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 查询配方称重记录头信息
	err = db.Where("record_id = ?", recordID).First(&list.Header).Error
	if err != nil {
		return list, err
	}

	// 查询配方称重记录详情信息
	err = db.Where("record_id = ?", recordID).Find(&list.Details).Error
	if err != nil {
		return list, err
	}

	return list, nil
}

// 查询单个配方信息
func (d *DbFormulaInfo) GetFormulaData(id int) ([]FormulaList, error) {

	var formulaLists []FormulaList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var headers []FormulaHeader
	err = db.Where("rec_id = ? AND is_used = ? AND is_latest = ?", id, true, true).Find(&headers).Error
	if err != nil {
		return nil, err
	}

	for _, header := range headers {
		var formulaDetails []FormulaDetail
		err = db.Where("formula_rec_id = ?", header.RecId).Find(&formulaDetails).Error
		if err != nil {
			return nil, err
		}

		formulaLists = append(formulaLists, FormulaList{
			Header:  header,
			Details: formulaDetails,
		})
	}

	return formulaLists, nil

}

// 检查配方ID和条码是否匹配
func (d *DbFormulaInfo) CheckFmaIdAndBarcode(recId int, formulaID string, formulaBarcode string) (bool, bool, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return false, false, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return false, false, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var FormulaHeaders []FormulaHeader
	var FormulaHeadersBarcode []FormulaHeader
	// 查询配方头信息
	err = db.Where("formula_id = ? AND is_used = ? AND is_latest = ?", formulaID, true, true).Find(&FormulaHeaders).Error
	if err != nil {
		return false, false, err
	}
	// 查询配方头信息
	err = db.Where("formula_barcode = ? AND is_used = ? AND is_latest = ?", formulaBarcode, true, true).Find(&FormulaHeadersBarcode).Error
	if err != nil {
		return false, false, err
	}
	//新增时判断配方ID和条码是否存在
	if recId == 0 {
		idFlag := true
		barcodeFlag := true
		if len(FormulaHeaders) > 0 {
			idFlag = false
		}
		if len(FormulaHeadersBarcode) > 0 {
			barcodeFlag = false
		}
		return idFlag, barcodeFlag, nil
	}

	//判断ID和条码是否已经存在，且recId 与存在的RecID 不一致
	//下面这个是修改配方时，判断配方ID和条码是否一致，ID是不能改的，所以，只需要判断条码是否存在即可
	barcodeFlag := true
	if len(FormulaHeadersBarcode) > 0 {
		for _, header := range FormulaHeadersBarcode {
			if header.RecId != recId {
				barcodeFlag = false
			}
		}
	}

	return true, barcodeFlag, nil
}

//GetFormulaDataByBarcode

func (d *DbFormulaInfo) GetFormulaDataByBarcode(barcode string) ([]FormulaList, error) {
	var formulaLists []FormulaList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})

	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()

	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var headers []FormulaHeader
	// 查询配方头信息
	err = db.Where("formula_barcode  = ? AND is_used = ? AND is_latest = ?", barcode, true, true).Find(&headers).Error
	if err != nil {
		return nil, err
	}
	for _, header := range headers {
		var formulaDetails []FormulaDetail
		err = db.Where("formula_rec_id = ?", header.RecId).Find(&formulaDetails).Error
		if err != nil {
			return nil, err
		}

		formulaLists = append(formulaLists, FormulaList{
			Header:  header,
			Details: formulaDetails,
		})
	}
	return formulaLists, nil
}

// 查询所有的 FormulaList
func (d *DbFormulaInfo) GetAllFormulaLists() ([]FormulaList, error) {
	var formulaLists []FormulaList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var headers []FormulaHeader
	err = db.Where("is_used = ? AND is_latest = ?", true, true).Find(&headers).Error
	if err != nil {
		return nil, err
	}

	if len(headers) == 0 {
		return formulaLists, nil
	}

	var headerIDs []int
	for _, h := range headers {
		headerIDs = append(headerIDs, h.RecId)
	}

	var allDetails []FormulaDetail
	db.Where("formula_rec_id IN (?)", headerIDs).Order("sequence ASC").Find(&allDetails)

	detailsMap := make(map[int][]FormulaDetail)
	for _, det := range allDetails {
		detailsMap[det.FormulaRecID] = append(detailsMap[det.FormulaRecID], det)
	}

	for _, header := range headers {
		formulaLists = append(formulaLists, FormulaList{
			Header:  header,
			Details: detailsMap[header.RecId],
		})
	}

	return formulaLists, nil
}

// 查询单个配方称重记录
func (d *DbFormulaInfo) GetFmaWgtRecByOrderId(orderId string) (FormulaWgtRecList, error) {
	var formulaWgtRecList FormulaWgtRecList

	// 设置重试参数
	maxRetries := 3
	retryDelay := 500 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 每次重试都建立新的数据库连接
		db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
		if err != nil {
			// 如果是最后一次尝试，直接返回错误
			if attempt == maxRetries {
				return formulaWgtRecList, err
			}
			// 否则等待并继续重试
			time.Sleep(retryDelay)
			continue
		}

		sqlDB, err := db.DB()
		if err != nil {
			db = nil
			if attempt == maxRetries {
				return formulaWgtRecList, err
			}
			time.Sleep(retryDelay)
			continue
		}

		// 确保数据库连接被关闭
		defer sqlDB.Close()

		var headers []FormulaWgtRecHeader
		err = db.Where("record_id = ?", orderId).Find(&headers).Error
		if err != nil {
			// 如果是最后一次尝试，直接返回错误
			if attempt == maxRetries {
				return formulaWgtRecList, err
			}
			// 否则等待并继续重试
			time.Sleep(retryDelay)
			continue
		}

		// 有且只能有一个记录
		if len(headers) != 1 {
			// 如果是最后一次尝试，返回特定错误
			if attempt == maxRetries {
				return formulaWgtRecList, fmt.Errorf("fail")
			}
			// 否则等待并继续重试
			time.Sleep(retryDelay)
			continue
		}

		var details []FormulaWgtRecDetail
		err = db.Where("record_id = ?", headers[0].RecordID).Find(&details).Error
		if err != nil {
			if attempt == maxRetries {
				return formulaWgtRecList, err
			}
			time.Sleep(retryDelay)
			continue
		}

		// 查询成功，构造返回结果
		formulaWgtRecList = FormulaWgtRecList{
			Header:  headers[0],
			Details: details,
		}

		// 成功获取数据，直接返回
		return formulaWgtRecList, nil
	}

	// 如果所有重试都失败（理论上不会执行到这里）
	return formulaWgtRecList, fmt.Errorf("fail: exceeded maximum retry attempts")
}

// 查询一个配方称重记录通过配方ID
func (d *DbFormulaInfo) GetOneFormulaWgtRecLists(fmaId string) ([]FormulaWgtRecList, error) {
	var formulaWgtRecLists []FormulaWgtRecList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var headers []FormulaWgtRecHeader
	err = db.Where("formula_id = ?", fmaId).Find(&headers).Error
	if err != nil {
		return nil, err
	}

	for _, header := range headers {
		var details []FormulaWgtRecDetail
		err = db.Where("record_id = ?", header.RecordID).Find(&details).Error
		if err != nil {
			return nil, err
		}
		formulaWgtRecLists = append(formulaWgtRecLists, FormulaWgtRecList{
			Header:  header,
			Details: details,
		})
	}

	return formulaWgtRecLists, nil
}

// 查询所有的 FormulaWgtRecList
func (d *DbFormulaInfo) GetAllFormulaWgtRecLists() ([]FormulaWgtRecList, error) {
	var formulaWgtRecLists []FormulaWgtRecList
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var headers []FormulaWgtRecHeader
	err = db.Order("rec_id DESC").Find(&headers).Error
	if err != nil {
		return nil, err
	}

	if len(headers) == 0 {
		return formulaWgtRecLists, nil
	}

	var recordIDs []string
	for _, h := range headers {
		if h.RecordID != "" {
			recordIDs = append(recordIDs, h.RecordID)
		}
	}

	var allDetails []FormulaWgtRecDetail
	if len(recordIDs) > 0 {
		db.Where("record_id IN (?)", recordIDs).Order("sequence ASC").Find(&allDetails)
	}

	detailsMap := make(map[string][]FormulaWgtRecDetail)
	for _, det := range allDetails {
		detailsMap[det.RecordID] = append(detailsMap[det.RecordID], det)
	}

	for _, header := range headers {
		formulaWgtRecLists = append(formulaWgtRecLists, FormulaWgtRecList{
			Header:  header,
			Details: detailsMap[header.RecordID],
		})
	}

	return formulaWgtRecLists, nil
}

// 新增配方称重记录头
func (d *DbFormulaInfo) CreateFormulaWgtRecHeader(header FormulaWgtRecHeader) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&header).Error
}

// 新增配方称重记录详情
func (d *DbFormulaInfo) CreateFormulaWgtRecDetail(detail FormulaWgtRecDetail) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&detail).Error
}

// 查询配方头中的最大的formula_key
func (d *DbFormulaInfo) GetMaxFormulaRecKey() (int, error) {
	var maxRecId int
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return 0, err

	}
	sqlDB, err := db.DB()
	if err != nil {
		return 0, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	err = db.Model(&FormulaHeader{}).Select("MAX(formula_key)").Row().Scan(&maxRecId)
	if err != nil {
		return 0, err
	}

	return maxRecId, nil
}

// 查询配方头中的最大的recId
func (d *DbFormulaInfo) GetMaxFormulaRecId() (int, error) {
	var maxRecId int
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return 0, err

	}
	sqlDB, err := db.DB()
	if err != nil {
		return 0, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	err = db.Model(&FormulaHeader{}).Select("MAX(rec_id)").Row().Scan(&maxRecId)
	if err != nil {
		return 0, err
	}

	return maxRecId, nil
}

// 新增配方头
func (d *DbFormulaInfo) CreateFormulaHeader(header FormulaHeader) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&header).Error
}

// 新增配方明细
func (d *DbFormulaInfo) CreateFormulaDetail(detail FormulaDetail) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&detail).Error
}

// 修改配方，包含配方头和明细
func (d *DbFormulaInfo) UpdateFormula(header FormulaHeader, details []FormulaDetail) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 更新配方头为新的数据
	if err := tx.Model(&FormulaHeader{}).Where("formula_key = ? AND is_latest = ?", header.FormulaKey, true).Updates(map[string]any{
		"formula_name":    header.FormulaName,
		"category_id":     header.CategoryID,
		"formula_mode":    header.FormulaMode,
		"formula_unit":    header.FormulaUnit,
		"total_weight":    header.TotalWeight,
		"material_count":  header.MaterialCount,
		"is_encrypted":    header.IsEncrypted,
		"need_container":  header.NeedContainer,
		"updated_at":      time.Now(),
		"updated_by":      header.UpdatedBy,
		"remark":          header.Remark,
		"remark1":         header.Remark1,
		"remark2":         header.Remark2,
		"remark3":         header.Remark3,
		"formula_barcode": header.FormulaBarcode,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 查询当前配方头的 RecId
	var currentHeader FormulaHeader
	if err := tx.Where("formula_key = ? AND is_latest = ?", header.FormulaKey, true).First(&currentHeader).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除旧的配方明细
	if err := tx.Where("formula_rec_id = ?", currentHeader.RecId).Delete(&FormulaDetail{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 使用当前配方头的 RecId 更新配方明细的 formulaRecId
	for i := range details {
		details[i].FormulaRecID = currentHeader.RecId
	}

	// 插入新的配方明细
	for _, detail := range details {
		if err := tx.Create(&detail).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// 新增配方称重记录，包含记录头和详情
func (d *DbFormulaInfo) CreateFormulaWgtRec(header FormulaWgtRecHeader, details []FormulaWgtRecDetail) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 插入配方称重记录头
	if err := tx.Create(&header).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 插入配方称重记录详情
	for _, detail := range details {
		if err := tx.Create(&detail).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// 根据记录编号删除配方称重记录头和详情
func (d *DbFormulaInfo) DeleteFormulaWgtRecByRecordID(recordID string) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 删除配方称重记录详情
	if err := tx.Where("record_id = ?", recordID).Delete(&FormulaWgtRecDetail{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除配方称重记录头
	if err := tx.Where("record_id = ?", recordID).Delete(&FormulaWgtRecHeader{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 根据recId将配方标记为未使用
func (d *DbFormulaInfo) DeleteFormulaByRecId(recId int) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 查询配方ID
	var header FormulaHeader
	if err := tx.Where("rec_id = ?", recId).First(&header).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除旧的配方明细
	if err := tx.Where("formula_rec_id = ?", header.RecId).Delete(&FormulaDetail{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 直接删除配方头记录
	if err := tx.Where("rec_id = ?", recId).Delete(&FormulaHeader{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除所有 isUsed 和 isLatest 都为 false 的配方头记录
	if err := tx.Where("is_used = ? AND is_latest = ?", false, false).Delete(&FormulaHeader{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 根据recId删除所有配方
// 根据recIds删除配方及其相关的暂存记录
func (d *DbFormulaInfo) DeleteAllFormulaByRecId(recIds []int) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 查询所有指定recIds的配方头信息
	var headers []FormulaHeader
	if err := tx.Where("rec_id IN ?", recIds).Find(&headers).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 收集所有需要删除明细的配方recIds和配方ID

	formulaIDs := make([]string, len(headers))
	for i, header := range headers {

		formulaIDs[i] = header.FormulaID
	}

	// 删除所有指定配方的明细记录
	if len(formulaIDs) > 0 {
		if err := tx.Where("formula_rec_id IN ?", recIds).Delete(&FormulaDetail{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 直接删除所有指定的配方头记录
		if err := tx.Where("rec_id IN ?", recIds).Delete(&FormulaHeader{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 根据配方ID查找相关的暂存配方记录头
	var draftHeaders []DrafFmaWgtRecHeader
	if len(formulaIDs) > 0 {
		if err := tx.Where("formula_id IN ?", formulaIDs).Find(&draftHeaders).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 收集需要删除的暂存记录的orderIds
		orderIds := make([]string, len(draftHeaders))
		for i, draftHeader := range draftHeaders {
			orderIds[i] = draftHeader.OrderId
		}

		// 根据orderIds删除暂存配方明细记录
		if len(orderIds) > 0 {
			if err := tx.Where("order_id IN ?", orderIds).Delete(&DrafFmaWgtRecDetail{}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		// 根据配方ID删除暂存配方头记录
		if len(formulaIDs) > 0 {
			if err := tx.Where("formula_id IN ?", formulaIDs).Delete(&DrafFmaWgtRecHeader{}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// 配方秤中的自动下一步设置
type SetAutoNext struct {
	RecID            int  `gorm:"primaryKey;autoincrement;not null"`
	AutoNext         bool `gorm:"not null"`
	StableTime       int  `gorm:"not null"`
	AutoTare         bool `gorm:"default:0; not null"`
	CheckCode        bool `gorm:"default:0; not null"`
	UnstableZeroTare bool `gorm:"default:0; not null"` // 新增字段
}

// 打印字段是否显示表
type SetReportPrint struct {
	// 编号（主键）
	RecID int `gorm:"primaryKey;autoincrement;not null"`
	// 配方ID
	FormulaID bool `gorm:"default:1; not null"`
	// 配方名称
	FormulaName bool `gorm:"default:1; not null"`
	//配方条码
	FormulaBarcode bool `gorm:"default:1; not null"`
	//单号
	OrderId bool `gorm:"default:1; not null"`
	//保存时间
	SaveTime bool `gorm:"default:1; not null"`
	// 操作员
	Operator bool `gorm:"default:1; not null"`
	//原料ID
	RawId bool `gorm:"default:1; not null"`
	//原料名称
	RawName bool `gorm:"default:1; not null"`
	//是否通过
	Pass bool `gorm:"default:1; not null"`
	//配方总重量
	FmaTotalWgt bool `gorm:"default:1; not null"`
	// 实际总重量
	ActualTotalWgt bool `gorm:"default:1; not null"`
	//设备名称
	DeviceName bool `gorm:"default:1; not null"`
	//实际误差
	RawActualErr bool `gorm:"default:1; not null"`
	//实际重量
	RawActualWgt bool `gorm:"default:1; not null"`
}

// 暂存配方的表头
type DrafFmaWgtRecHeader struct {
	// 记录编号（主键）
	RecID int `gorm:"primaryKey;autoincrement;not null"`
	// 配方ID
	FormulaID string `gorm:"not null"`
	//单号
	OrderId string `gorm:"not null"`
	//创建时间
	CreatedAt time.Time `gorm:"not null"`
	// 创建人
	CreatedBy string `gorm:"not null"`
	// 更新时间
	UpdatedAt time.Time `gorm:"not null"`
	// 更新人
	UpdatedBy string `gorm:"not null"`
	//状态
	Status int `gorm:"not null"` // 状态字段，0表示草稿状态，1表示已发布状态
	// 备注
	Remark string
	// 备注1
	Remark1 string
	// 备注2
	Remark2 string
}

// 暂存配方的明细
type DrafFmaWgtRecDetail struct {
	// 记录编号（主键）
	RecID int `gorm:"primaryKey;autoincrement;not null"`
	// 单号
	OrderId string `gorm:"not null"`
	// 原料ID
	RawMaterialID string `gorm:"not null"`
	// 原料序号
	Seq int `gorm:"not null"`
	//实际重量
	ActualWeight float64 `gorm:"not null"`
	//实际重量单位
	ActualWeightUnit string
	//是否是容器
	IsContainer bool `gorm:"not null"`
	// 备注
	Remark string
	// 备注1
	Remark1 string
	// 备注2
	Remark2 string
	//秤的信息
	ScaleId    int
	ScaleName  string
	ScaleModel string
	ScaleSn    string
}

//暂存配方的记录

type DrafFmaWgtRecInfo struct {
	// 暂存配方称重记录头
	Header DrafFmaWgtRecHeader `gorm:"embedded"`
	// 暂存配方称重记录详情
	Details []DrafFmaWgtRecDetail `gorm:"foreignKey:OrderId;references:OrderId"`
}

// 配方秤中的输出口设置
type SetOutputPort struct {
	RecID     int     `gorm:"primaryKey;autoincrement;not null"`
	Port      int     `gorm:"not null"`            //0-11   输出口
	Status    bool    `gorm:"default:0; not null"` // 开关状态
	StartTime int     `gorm:"not null"`            // 开始延迟时间，单位ms
	EndValue  float64 `gorm:"not null"`            // 关闭阈值，距离目标差值小于这个值就关闭输出口 比如目标重量是1000g，EndValue是50g，当实际重量达到950g时就关闭输出口
	Remark    string  // 备注
	UpdateAt  time.Time
}

// 配方秤中的输入口设置
type SetInputPort struct {
	RecID    int    `gorm:"primaryKey;autoincrement;not null"`
	Port     int    `gorm:"not null"`                //0-3   输入口
	Btn      string `gorm:"default:'None';not null"` // 按钮类型
	UpdateAt time.Time
}

func (d *DbFormulaInfo) CreateSetAutoNext() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 检查 SetAutoNext 表中是否有数据
	var count int64
	if err := tx.Model(&SetAutoNext{}).Count(&count).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 若没有数据，则插入一条
	if count == 0 {
		if err := tx.Create(&SetAutoNext{
			AutoNext:   true,
			StableTime: 5,
			AutoTare:   false,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// 创建打印字段是否显示表
func (d *DbFormulaInfo) CreateSetReportPrint() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	var count int64
	if err := tx.Model(&SetReportPrint{}).Count(&count).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 若没有数据，则插入一条
	if count == 0 {
		if err := tx.Create(&SetReportPrint{
			FormulaID:      true,
			FormulaName:    true,
			FormulaBarcode: true,
			OrderId:        true,
			SaveTime:       true,
			Operator:       true,
			RawId:          true,
			RawName:        true,
			Pass:           true,
			FmaTotalWgt:    true,
			ActualTotalWgt: true,
			DeviceName:     true,
			RawActualErr:   true,
			RawActualWgt:   true,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// 修改打印字段是否显示表
func (d *DbFormulaInfo) UpdateSetReportPrint(setReportPrint SetReportPrint) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// 更新 SetReportPrint 表
	if err := tx.Model(&SetReportPrint{}).Where("rec_id = ?", 1).Updates(map[string]interface{}{
		"formula_id":       setReportPrint.FormulaID,
		"formula_name":     setReportPrint.FormulaName,
		"formula_barcode":  setReportPrint.FormulaBarcode,
		"order_id":         setReportPrint.OrderId,
		"save_time":        setReportPrint.SaveTime,
		"operator":         setReportPrint.Operator,
		"raw_id":           setReportPrint.RawId,
		"raw_name":         setReportPrint.RawName,
		"pass":             setReportPrint.Pass,
		"raw_actual_wgt":   setReportPrint.RawActualWgt,
		"raw_actual_err":   setReportPrint.RawActualErr,
		"fma_total_wgt":    setReportPrint.FmaTotalWgt,
		"actual_total_wgt": setReportPrint.ActualTotalWgt,
		"device_name":      setReportPrint.DeviceName,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 查询SetReportPrint
func (d *DbFormulaInfo) GetSetReportPrint() (*SetReportPrint, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var setReportPrint SetReportPrint
	if err := db.First(&setReportPrint).Error; err != nil {
		return nil, err
	}
	return &setReportPrint, nil
}

// 修改SetAutoNext
func (d *DbFormulaInfo) UpdateSetAutoNext(autoNext bool, stableTime int, autoTare bool, checkCode bool) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Model(&SetAutoNext{}).Where("rec_id = ?", 1).Updates(map[string]interface{}{
		"auto_next":   autoNext,
		"stable_time": stableTime,
		"auto_tare":   autoTare,
		"check_code":  checkCode,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 查询SetAutoNext
func (d *DbFormulaInfo) GetSetAutoNext() (*SetAutoNext, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 查询
	var setAutoNext SetAutoNext
	if err := db.First(&setAutoNext).Error; err != nil {
		return nil, err
	}
	return &setAutoNext, nil
}

// 获取不稳定归零扣重开关
func (d *DbFormulaInfo) GetUnstableZeroTare() (bool, error) {
	set, err := d.GetSetAutoNext()
	if err != nil {
		return false, err
	}
	return set.UnstableZeroTare, nil
}

// 更新不稳定归零扣重开关
func (d *DbFormulaInfo) UpdateUnstableZeroTare(enable bool) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 更新第一条记录（通常配置表只有一条记录）
	return db.Model(&SetAutoNext{}).Where("1 = 1").Update("unstable_zero_tare", enable).Error
}

//暂存配方的增删改查

// 新增暂存配方称重记录头
func (d *DbFormulaInfo) CreateDraftFmaWgtRecHeader(header DrafFmaWgtRecHeader) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&header).Error
}

// 新增暂存配方称重记录详情
func (d *DbFormulaInfo) CreateDraftFmaWgtRecDetail(detail DrafFmaWgtRecDetail) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	return db.Create(&detail).Error
}

// 查询所有的暂存配方称重记录头和明细 List
func (d *DbFormulaInfo) GetAllDraftFmaWgtRecLists() ([]DrafFmaWgtRecInfo, error) {
	var draftFmaWgtRecLists []DrafFmaWgtRecInfo
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var headers []DrafFmaWgtRecHeader
	err = db.Find(&headers).Error
	if err != nil {
		return nil, err
	}

	for _, header := range headers {
		var details []DrafFmaWgtRecDetail
		err = db.Where("order_id = ?", header.OrderId).Find(&details).Error
		if err != nil {
			return nil, err
		}
		draftFmaWgtRecLists = append(draftFmaWgtRecLists, DrafFmaWgtRecInfo{
			Header:  header,
			Details: details,
		})
	}

	return draftFmaWgtRecLists, nil
}

// 查询暂存配方称重记录ByOrderId
func (d *DbFormulaInfo) GetDraftFmaWgtRecByOrderId(orderIds []string) ([]DrafFmaWgtRecInfo, error) {
	var draftFmaWgtRecLists []DrafFmaWgtRecInfo
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 查询暂存配方称重记录头
	var headers []DrafFmaWgtRecHeader
	err = db.Where("order_id IN ?", orderIds).Find(&headers).Error
	if err != nil {
		return nil, err
	}
	for _, header := range headers {
		var details []DrafFmaWgtRecDetail
		err = db.Where("order_id = ?", header.OrderId).Find(&details).Error
		if err != nil {
			return nil, err
		}
		draftFmaWgtRecLists = append(draftFmaWgtRecLists, DrafFmaWgtRecInfo{
			Header:  header,
			Details: details,
		})
	}
	return draftFmaWgtRecLists, nil
}

// 根据order_id删除某一条记录
func (d *DbFormulaInfo) DeleteDraftFmaWgtRec(orderId string) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// 删除暂存配方称重记录详情
	if err := tx.Where("order_id = ?", orderId).Delete(&DrafFmaWgtRecDetail{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 删除暂存配方称重记录头
	if err := tx.Where("order_id = ?", orderId).Delete(&DrafFmaWgtRecHeader{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 提交事务
	return tx.Commit().Error
}

// 删除所有暂存配方
func (d *DbFormulaInfo) DeleteAllDraftFmaWgtRec(orderIds []string) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("order_id IN ?", orderIds).Delete(&DrafFmaWgtRecDetail{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 删除暂存配方称重记录头
	if err := tx.Where("order_id IN ?", orderIds).Delete(&DrafFmaWgtRecHeader{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 提交事务
	return tx.Commit().Error
}

// 更新暂存配方称重记录头和明细
func (d *DbFormulaInfo) UpdateDraftFmaWgtRec(rec DrafFmaWgtRecInfo) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// 更新暂存配方称重记录头
	if err := tx.Model(&DrafFmaWgtRecHeader{}).Where("order_id = ?", rec.Header.OrderId).Updates(map[string]interface{}{
		"status":     rec.Header.Status,
		"updated_at": time.Now(),
		"updated_by": rec.Header.UpdatedBy,
		"remark":     rec.Header.Remark,
		"remark1":    rec.Header.Remark1,
		"remark2":    rec.Header.Remark2,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 删除旧的暂存配方称重记录详情
	if err := tx.Where("order_id = ?", rec.Header.OrderId).Delete(&DrafFmaWgtRecDetail{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 新增新的暂存配方称重记录详情
	for _, detail := range rec.Details {
		if err := tx.Create(&detail).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	// 提交事务
	return tx.Commit().Error
}

func (d *DbFormulaInfo) CheckFormulaRawData(scaleId int) (bool, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return false, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return false, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 检查配方材料里面是否使用了这个秤
	var count int64
	db.Model(&RawMaterial{}).Where("scale_id = ?", scaleId).Count(&count)
	if count > 0 {
		return true, nil
	}
	return false, nil

}

// 根据原料ID获取原料信息
func (d *DbFormulaInfo) GetRawDataByRawID(rawId string) (RawMaterial, error) {
	var raw RawMaterial
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return raw, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return raw, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 查询原料信息
	if err := db.Where("material_id = ?", rawId).First(&raw).Error; err != nil {
		return raw, err
	}
	return raw, nil
}

// 创建一个新的上传服务器信息 服务器信息只能有一个，如果有则覆盖
func (d *DbFormulaInfo) CreateUploadServerInfo(info UploadServerInfo) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// 检查上传服务器信息表是否已经存在数据
	var count int64
	db.Model(&UploadServerInfo{}).Count(&count)
	if count > 0 {
		// 如果存在数据，则删除旧数据
		if err := tx.Delete(&UploadServerInfo{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	// 创建新的上传服务器信息
	if err := tx.Create(&info).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 提交事务
	return tx.Commit().Error
}

// 获取上传服务器信息 如果不存在则返回空结构体
func (d *DbFormulaInfo) GetUploadServerInfo() (UploadServerInfo, error) {
	var info UploadServerInfo
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return info, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return info, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 查询上传服务器信息
	if err := db.First(&info).Error; err != nil {
		return info, err
	}
	return info, nil
}

// 修改上传服务器信息
func (d *DbFormulaInfo) UpdateUploadServerInfo(info UploadServerInfo) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// 更新上传服务器信息
	if err := tx.Model(&UploadServerInfo{}).Where("rec_id = ?", 1).Updates(map[string]interface{}{
		"ip":         info.Ip,
		"share_name": info.ShareName,
		"username":   info.Username,
		"password":   info.Password,
		"enable":     info.Enable,
		"updated_at": time.Now(),
		"updated_by": info.UpdatedBy,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 提交事务
	return tx.Commit().Error
}

// //////// 配方秤输出口设置的增删改查
func (d *DbFormulaInfo) CreateSetOutputPort() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 检查 SetOutputPort 表中是否有数据
	var count int64
	if err := tx.Model(&SetOutputPort{}).Count(&count).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 若没有数据，则插入一条
	if count == 0 {
		for i := 1; i <= 12; i++ {
			if err := tx.Create(&SetOutputPort{
				Port:      i,
				Status:    false,
				StartTime: 0,
				EndValue:  0,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// 提交事务
	return tx.Commit().Error
}

func (d *DbFormulaInfo) GetSetOutputPort() ([]SetOutputPort, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 查询 SetOutputPort 表中的所有数据
	var ports []SetOutputPort
	if err := db.Find(&ports).Error; err != nil {
		return nil, err
	}

	return ports, nil
}

// SetOutputPort
func (d *DbFormulaInfo) UpdateSetOutputPort(ports []SetOutputPort) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	for i := 0; i < len(ports); i++ {
		// 检查记录是否存在
		var count int64
		if err := tx.Model(&SetOutputPort{}).Where("port = ?", ports[i].Port).Count(&count).Error; err != nil {
			tx.Rollback()
			return err
		}

		if count > 0 {
			// 存在则更新
			if err := tx.Model(&SetOutputPort{}).Where("port = ?", ports[i].Port).Updates(map[string]interface{}{
				"status":     ports[i].Status,
				"start_time": ports[i].StartTime,
				"end_value":  ports[i].EndValue,
				"remark":     ports[i].Remark,
				"update_at":  time.Now(),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// //////// 配方秤输入口设置的增删改查
func (d *DbFormulaInfo) CreateSetInputPort() error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 检查 SetInputPort 表中是否有数据
	var count int64
	if err := tx.Model(&SetInputPort{}).Count(&count).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 若没有数据，则插入一条
	if count == 0 {
		for i := 1; i <= 4; i++ {
			if err := tx.Create(&SetInputPort{
				Port:     i,
				Btn:      "None",
				UpdateAt: time.Now(),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	// 提交事务
	return tx.Commit().Error
}

// GetSetInputPort
func (d *DbFormulaInfo) GetSetInputPort() ([]SetInputPort, error) {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 查询 SetInputPort 表中的所有数据
	var ports []SetInputPort
	if err := db.Find(&ports).Error; err != nil {
		return nil, err
	}
	return ports, nil
}

// SetInputPort
func (d *DbFormulaInfo) UpdateSetInputPort(ports []SetInputPort) error {
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	for i := 0; i < len(ports); i++ {
		// 检查记录是否存在
		var count int64
		if err := tx.Model(&SetInputPort{}).Where("port = ?", ports[i].Port).Count(&count).Error; err != nil {
			tx.Rollback()
			return err
		}

		if count > 0 {
			// 存在则更新
			if err := tx.Model(&SetInputPort{}).Where("port = ?", ports[i].Port).Updates(map[string]interface{}{
				"btn":       ports[i].Btn,
				"update_at": time.Now(),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	// 提交事务
	return tx.Commit().Error
}
