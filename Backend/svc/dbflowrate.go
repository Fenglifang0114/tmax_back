package svc

import (
	"errors"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

type DbFlowRate struct {
	dbName string
	db     *gorm.DB
}

// FlowRate 流速头表格
type FlowRateHeader struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 总重量
	TotalWeight float64
	//总时间
	TotalTime float64
	//平均流速
	AverageFlowRate float64
	//最小流速
	MinFlowRate float64
	//最大流速
	MaxFlowRate float64
	//流速单位
	WgtUnit string
	// 创建时间
	CreatedAt time.Time
}

// FlowRateDetail 流速明细表
type FlowRateDetail struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 头Id
	HeaderId int
	// 序号
	Id int
	// 速度
	Rate float64
	//时间
	Time float64
}

type FlowRateList struct {
	FlowRateHeader FlowRateHeader
	FlowRateDetail []FlowRateDetail
}

// 初始化数据库连接和表结构
func NewFlowRateInfo(dbName string) (*DbFlowRate, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err := db.AutoMigrate(
		&FlowRateHeader{},
		&FlowRateDetail{},
	); err != nil {
		l.Log.Errorf("failed to migrate database of flow rate: %v", err)
		return nil, err
	}

	return &DbFlowRate{dbName: dbName, db: db}, nil
}

func (d *DbFlowRate) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// 新增头
func (d *DbFlowRate) AddFlowRateHeader(header *FlowRateHeader) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Create(header).Error
}

// 新增明细表
func (d *DbFlowRate) AddFlowRateDetail(detail *FlowRateDetail) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Create(detail).Error
}

// 找出头表中最大的recid
func (d *DbFlowRate) FindMaxRecId() (int, error) {
	if d == nil || d.db == nil {
		return 0, errors.New("db connection is nil")
	}
	var header FlowRateHeader
	if err := d.db.Last(&header).Error; err != nil {
		return 0, err
	}
	return header.RecId, nil
}

// 获取所有的流速信息
func (d *DbFlowRate) GetAllFlowRateLists() ([]FlowRateList, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var headers []FlowRateHeader
	if err := d.db.Find(&headers).Error; err != nil {
		return nil, err
	}

	var flowRateLists []FlowRateList
	for _, header := range headers {
		var details []FlowRateDetail
		if err := d.db.Where("header_id = ?", header.RecId).Find(&details).Error; err != nil {
			return nil, err
		}
		flowRateLists = append(flowRateLists, FlowRateList{
			FlowRateHeader: header,
			FlowRateDetail: details,
		})
	}
	return flowRateLists, nil
}
