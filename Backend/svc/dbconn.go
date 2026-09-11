package svc

import (
	"errors"
	"os"
	"path/filepath"
	l "tmaxsrv/log"

	"github.com/gitteamer/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbScaleConn struct {
	dbName string
}

func NewDbScaleConn(dbName string) (*DbScaleConn, error) {
	if dir := filepath.Dir(dbName); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		l.Log.Error("failed to connect database: ", err)
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Error("failed to get sqlDB: ", err)
		return nil, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// Migrate the schema
	if err = db.AutoMigrate(&ScaleConnMedia{}, &SrvScaleRel{}); err != nil {
		l.Log.Error("failed to migrate database of scale connection: ", err)
		return nil, err
	}
	return &DbScaleConn{dbName: dbName}, nil
}

func (d *DbScaleConn) GetScaleConnList() ([]*ScaleConnMedia, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// Migrate the schema
	if err = db.AutoMigrate(&ScaleConnMedia{}); err != nil {
		l.Log.Debug("failed to migrate database of scale connection")
	}

	// 读取内容
	var conns []*ScaleConnMedia
	db.Find(&conns)
	return conns, nil
}

func (d *DbScaleConn) InsertScaleConn(conn ScaleConnMedia) error {
	// if conn.Scale == nil {
	// 	return errors.New("No scale instance assigned to this scale connection")
	// }
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var conn1 ScaleConnMedia
	db.Where("scale_model=?", conn.ScaleModel).Where("scale_sn=?", conn.ScaleSn).First(&conn1)
	if conn1.ScaleModel != "" {
		log.Error("scale is already exist, with ScaleModel: %v, ScaleSn: %v", conn.ScaleModel, conn.ScaleSn)
		return errors.New("scale_id is already exist")
	}

	tx := db.Create(&conn)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DbScaleConn) UpdateScaleConn(conn ScaleConnMedia) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	tx := db.Model(&ScaleConnMedia{})
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
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	rowAffected := db.Model(&conn).Where("scale_id=?", conn.ScaleId).Updates(&conn).RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleConn failed, mybe record not existing")
	} //写成save模式不生效，又改回来了

	return nil
}

func (d *DbScaleConn) UpdateScaleSn(conn ScaleConnMedia) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 使用Updates函数时，只传入需要更新的字段
	result := db.Model(&conn).Where("scale_id=?", conn.ScaleId).Updates(map[string]interface{}{
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
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 使用Updates函数时，只传入需要更新的字段
	result := db.Model(&conn).Where("scale_id=?", conn.ScaleId).Updates(map[string]interface{}{
		"ScaleName": conn.ScaleName,
	})
	rowAffected := result.RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateScaleConn failed, maybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) DeleteScaleConn(inConn ScaleConnMedia) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	db.Where("scale_id=?", inConn.ScaleId).Delete((&inConn))
	return nil
}

func (d *DbScaleConn) InsertSrvScaleRel(rel SrvScaleRel) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	tx := db.Create(&rel)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *DbScaleConn) GetSrvScaleRelList() ([]*SrvScaleRel, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var rels []*SrvScaleRel
	db.Find(&rels)

	return rels, nil
}

func (d *DbScaleConn) UpdateSrvScaleRel(rel SrvScaleRel) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	updateFields := make(map[string]interface{})
	updateFields["is_used"] = rel.IsUsed

	result := db.Model(&rel).Where("scale_id=?", rel.ScaleId).Where("srv_id=?", rel.SrvId).Updates(updateFields)
	rowAffected := result.RowsAffected
	if rowAffected == 0 {
		return errors.New("@UpdateSrvScaleRel failed, maybe record not existing")
	}

	return nil
}

func (d *DbScaleConn) DeleteSrvScaleRel(rel SrvScaleRel) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	db.Where("scale_id=?", rel.ScaleId).Where("srv_id=?", rel.SrvId).Delete(&rel)

	return nil
}

func (d *DbScaleConn) DeleteSrvScaleRelByScaleId(scaleId int64) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	result := db.Where("scale_id =?", scaleId).Delete(&SrvScaleRel{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (d *DbScaleConn) DeleteSrvScaleRelAll() error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 使用Delete方法时不传入具体条件，即可删除所有记录
	result := db.Delete(&SrvScaleRel{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (d *DbScaleConn) getNameById(id int64) string {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		l.Log.Debug("failed to connect database")
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var scale ScaleConnMedia

	result := db.Where("scale_id=?", id).First(&scale)

	if result.RowsAffected == 0 {
		return ""
	}

	return scale.ScaleName
}
