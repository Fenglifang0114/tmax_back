package svc

import (
	"errors"
	"fmt"
	"time"

	l "tmaxsrv/log"

	"github.com/gitteamer/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openDatabase 创建带有高并发优化（busy_timeout与连接池）的 GORM 数据库长连接
func openDatabase(dbName string) (*gorm.DB, error) {
	if dbName == "" {
		return nil, errors.New("dbName is empty")
	}
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		l.Log.Errorf("failed to open database %s: %v", dbName, err)
		return nil, err
	}
	sqlDB, err := db.DB()
	if err == nil && sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(1 * time.Hour)
	}
	return db, nil
}

type DbScaleConn struct {
	dbName string
	db     *gorm.DB
}

func NewDbScaleConn(dbName string) (*DbScaleConn, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}

	// Migrate the schema
	if err = db.AutoMigrate(&ScaleConnMedia{}, &SrvScaleRel{}); err != nil {
		l.Log.Errorf("failed to migrate database of scale connection: %v", err)
	}
	return &DbScaleConn{
		dbName: dbName,
		db:     db,
	}, nil
}

func (d *DbScaleConn) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func (d *DbScaleConn) GetScaleConnList() ([]*ScaleConnMedia, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var conns []*ScaleConnMedia
	if err := d.db.Find(&conns).Error; err != nil {
		return nil, err
	}
	return conns, nil
}

func (d *DbScaleConn) InsertScaleConn(conn ScaleConnMedia) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	var conn1 ScaleConnMedia
	d.db.Where("scale_model=?", conn.ScaleModel).Where("scale_sn=?", conn.ScaleSn).First(&conn1)
	if conn1.ScaleModel != "" {
		log.Error("scale is already exist, with ScaleModel: %v, ScaleSn: %v", conn.ScaleModel, conn.ScaleSn)
		return errors.New("scale_id is already exist")
	}

	tx := d.db.Create(&conn)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbScaleConn) UpdateScaleConn(conn ScaleConnMedia) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	tx := d.db.Model(&ScaleConnMedia{})
	if conn.ScaleId != 0 {
		tx = tx.Where("scale_id = ?", conn.ScaleId)
	} else {
		tx = tx.Where("scale_model = ? AND scale_sn = ?", conn.ScaleModel, conn.ScaleSn)
	}

	rowAffected := tx.Updates(map[string]interface{}{
		"mediainfo_type":            conn.MediaConf.Type,
		"mediainfo_media_info_json": conn.MediaConf.MediaInfoJson,
		"modbus_id":                 conn.ModbusId,
		"protocol_name":             conn.ProtocolName,
	}).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleConn failed, maybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) UpdateScaleInfo(conn ScaleConnMedia) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	rowAffected := d.db.Model(&conn).Where("scale_id=?", conn.ScaleId).Updates(&conn).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleConn failed, mybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) UpdateScaleSn(conn ScaleConnMedia) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Model(&conn).Where("scale_id=?", conn.ScaleId).Updates(map[string]interface{}{
		"InnerModel": conn.InnerModel,
		"ScaleSn":    conn.ScaleSn,
	})
	rowAffected := result.RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleConn failed, maybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) UpdateScaleName(conn ScaleConnMedia) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Model(&conn).Where("scale_id=?", conn.ScaleId).Updates(map[string]interface{}{
		"ScaleName": conn.ScaleName,
	})
	rowAffected := result.RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleConn failed, maybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) DeleteScaleConn(inConn ScaleConnMedia) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	d.db.Where("scale_id=?", inConn.ScaleId).Delete((&inConn))
	return nil
}

func (d *DbScaleConn) InsertSrvScaleRel(rel SrvScaleRel) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	tx := d.db.Create(&rel)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *DbScaleConn) GetSrvScaleRelList() ([]*SrvScaleRel, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}

	var rels []*SrvScaleRel
	d.db.Find(&rels)

	return rels, nil
}

func (d *DbScaleConn) UpdateSrvScaleRel(rel SrvScaleRel) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	updateFields := make(map[string]interface{})
	updateFields["is_used"] = rel.IsUsed

	result := d.db.Model(&rel).Where("scale_id=?", rel.ScaleId).Where("srv_id=?", rel.SrvId).Updates(updateFields)
	rowAffected := result.RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateSrvScaleRel failed, maybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) DeleteSrvScaleRel(rel SrvScaleRel) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	d.db.Where("scale_id=?", rel.ScaleId).Where("srv_id=?", rel.SrvId).Delete(&rel)

	return nil
}

func (d *DbScaleConn) DeleteSrvScaleRelByScaleId(scaleId int64) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Where("scale_id =?", scaleId).Delete(&SrvScaleRel{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (d *DbScaleConn) DeleteSrvScaleRelAll() error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	result := d.db.Delete(&SrvScaleRel{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (d *DbScaleConn) getNameById(id int64) string {
	if d == nil || d.db == nil {
		return ""
	}
	var scale ScaleConnMedia

	result := d.db.Where("scale_id=?", id).First(&scale)

	if result.RowsAffected == 0 {
		return ""
	}

	return scale.ScaleName
}
