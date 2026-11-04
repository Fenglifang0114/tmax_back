package svc

import (
	"crypto/md5"
	"errors"
	"fmt"
	"time"
	l "tmaxsrv/log"

	"gorm.io/gorm"
)

//系统用户表

type DbSysUser struct {
	dbName string
	db     *gorm.DB
}

// 角色表
type SysRole struct {
	RoleId     int            `gorm:"primaryKey;not null;autoincrement;"`
	RoleName   string         `gorm:"not null;"`
	IsFullPerm int            `gorm:"not null;"` // 1-拥有全部权限，0-不拥有
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// 用户表
type SysUser struct {
	UserId        int    `gorm:"primaryKey;not null;autoincrement;"`
	UserName      string `gorm:"not null;unique"`         //账户名 用于登录
	NickName      string `gorm:"not null;default:'name'"` // 昵称 或者姓名
	RoleId        int    `gorm:"not null;"`
	Password      string `gorm:"not null"`
	IsEnabled     bool   `gorm:"not null;default:true;"`
	Email         string
	Phone         string
	InitialPageId int
	Remark        string
	CreatedTime   time.Time `gorm:"autoCreateTime"`
	UpdatedTime   time.Time `gorm:"autoUpdateTime"`
	CreatedBy     int
	UpdatedBy     int
	CreatedByName string
	UpdatedByName string
	IsChanged     bool `gorm:"not null;default:false;"`
	Rfid          string
}

// 操作员-页面关联表
type SysOperatorPage struct {
	Id        int       `gorm:"primaryKey;autoIncrement"`
	UserId    int       `gorm:"not null"`
	PageId    int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// 关联
	User SysUser `gorm:"foreignKey:UserId;references:UserId"`
}

func NewDbSysUser(dbName string) (*DbSysUser, error) {
	db, err := openDatabase(dbName)
	if err != nil {
		l.Log.Errorf("failed to connect database %s: %v", dbName, err)
		return nil, err
	}
	if err = db.AutoMigrate(
		&SysRole{},
		&SysUser{},
		&SysOperatorPage{},
	); err != nil {
		l.Log.Errorf("failed to migrate database: %v", err)
	}
	sysUser := &DbSysUser{dbName: dbName, db: db}
	initRoles(sysUser)
	initAdmin(sysUser)

	return sysUser, nil
}

func (d *DbSysUser) Close() error {
	if d != nil && d.db != nil {
		sqlDB, err := d.db.DB()
		if err == nil && sqlDB != nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// 初始化三种角色
func initRoles(sysUser *DbSysUser) error {
	roles, err := sysUser.ListAllRoles()
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		defaultRoles := []SysRole{
			{
				RoleName:   "super_admin",
				IsFullPerm: 1,
			},
			{
				RoleName:   "admin",
				IsFullPerm: 1,
			},
			{
				RoleName:   "operator",
				IsFullPerm: 0,
			},
		}

		for _, role := range defaultRoles {
			if err := sysUser.CreateRole(&role); err != nil {
				l.Log.Printf("failed to create role: %v", err)
				return err
			}
		}
		l.Log.Debug("ok")
	} else {
		l.Log.Debug("role already initialized")
	}
	return nil
}

// 初始化管理员
func initAdmin(sysUser *DbSysUser) error {
	if sysUser == nil || sysUser.db == nil {
		return errors.New("db connection is nil")
	}
	var count int64
	sysUser.db.Model(&SysUser{}).Count(&count)
	if count > 1 {
		sysUser.db.Model(&SysUser{}).Where("user_id = ?", 1).Update("is_changed", true)
		return nil
	}
	if count > 0 {
		return nil
	}

	admin := &SysUser{
		UserName:      "Super Admin",
		NickName:      "Super Admin",
		Password:      "123456",
		RoleId:        1,
		Email:         "",
		Phone:         "",
		InitialPageId: 9999, //设置界面
		Remark:        "super admin",
	}

	if err := sysUser.CreateUser(admin); err != nil {
		return err
	}
	return nil
}

// 创建角色(初始化时使用，因为角色固定为三种)
func (d *DbSysUser) CreateRole(role *SysRole) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Create(role).Error
}

// 获取所有角色
func (d *DbSysUser) ListAllRoles() ([]SysRole, error) {
	if d == nil || d.db == nil {
		return []SysRole{}, errors.New("db connection is nil")
	}
	var roles []SysRole
	result := d.db.Find(&roles)
	return roles, result.Error
}

// 创建用户
func (d *DbSysUser) CreateUser(user *SysUser) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	var err error
	user.Password, err = EncryptPassword(user.Password)
	if err != nil {
		return err
	}

	return d.db.Create(user).Error
}

// 获取用户信息(通过用户名)
func (d *DbSysUser) GetUserByUsername(userName string) (*SysUser, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var user SysUser
	result := d.db.First(&user, "user_name = ?", userName)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// 获取用户信息(通过用户ID)
func (d *DbSysUser) GetUserByUserId(userId int) (*SysUser, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var user SysUser
	result := d.db.First(&user, "user_id = ?", userId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// 获取用户信息(通过RFID卡号)
func (d *DbSysUser) GetUserByRfid(rfid string) (*SysUser, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("db connection is nil")
	}
	var user SysUser
	result := d.db.First(&user, "rfid = ?", rfid)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// 获取多个用户详情
func (d *DbSysUser) GetManyUserByUserIds(userIds []int) ([]SysUser, error) {
	if d == nil || d.db == nil {
		return []SysUser{}, errors.New("db connection is nil")
	}
	var users []SysUser
	result := d.db.Find(&users, "user_id IN ?", userIds)
	return users, result.Error
}

// 更新用户
func (d *DbSysUser) UpdateUser(user *SysUser, pswUpdated bool) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	var existingUser SysUser
	result := d.db.Model(&SysUser{}).Where("user_id = ?", user.UserId).First(&existingUser)
	if result.Error != nil {
		return fmt.Errorf("fail to update user: %w", result.Error)
	}
	updateMap := map[string]interface{}{
		"user_name":       user.UserName,
		"nick_name":       user.NickName,
		"role_id":         user.RoleId,
		"is_enabled":      user.IsEnabled,
		"email":           user.Email,
		"phone":           user.Phone,
		"initial_page_id": user.InitialPageId,
		"remark":          user.Remark,
		"updated_by":      user.UpdatedBy,
		"updated_by_name": user.UpdatedByName,
		"rfid":            user.Rfid,
		"updated_time":    time.Now(),
	}

	if pswUpdated {
		encryptedPassword, err := EncryptPassword(user.Password)
		if err != nil {
			return fmt.Errorf("fail to update user: %w", err)
		}
		updateMap["password"] = encryptedPassword
		updateMap["is_changed"] = true
	}

	result = d.db.Model(&SysUser{}).
		Where("user_id = ?", user.UserId).
		Updates(updateMap)
	if result.Error != nil {
		return fmt.Errorf("fail to update user: %w", result.Error)
	}

	return nil
}

// 删除用户(软删除)
func (d *DbSysUser) DeleteUser(userID []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	for _, id := range userID {
		if id == 1 {
			continue
		}
		if err := d.db.Delete(&SysUser{}, "user_id = ?", id).Error; err != nil {
			return err
		}
	}
	for _, id := range userID {
		if err := d.ClearAllPagePermissions(id); err != nil {
			return err
		}
	}

	return nil
}

// 查询用户列表
func (d *DbSysUser) ListUsers() ([]SysUser, error) {
	if d == nil || d.db == nil {
		return []SysUser{}, errors.New("db connection is nil")
	}
	var users []SysUser
	result := d.db.Find(&users)
	if result.Error != nil {
		return []SysUser{}, result.Error
	}
	return users, nil
}

// 修改密码
func (d *DbSysUser) ChangePassword(userId int, newPassword string) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	var user SysUser
	result := d.db.First(&user, "user_id = ?", userId)
	if result.Error != nil {
		return fmt.Errorf("user not found: %w", result.Error)
	}

	hashedNewPassword, err := EncryptPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt new password: %w", err)
	}

	user.Password = hashedNewPassword
	user.IsChanged = true
	err = d.db.Save(&user).Error
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// 禁用用户
func (d *DbSysUser) DisableUser(userID int, isEnabled bool, updateBy int, updateByName string) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	var user SysUser
	result := d.db.First(&user, "user_id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("user not found: %w", result.Error)
	}

	user.IsEnabled = isEnabled
	user.UpdatedBy = updateBy
	user.UpdatedByName = updateByName
	if err := d.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	return nil
}

// 撤销操作员的页面权限
func (d *DbSysUser) updateUserPermission(userID int, pageIDList []int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}

	d.db.Delete(&SysOperatorPage{}, "user_id = ?", userID)
	for _, pageID := range pageIDList {
		d.db.Create(&SysOperatorPage{
			UserId: userID,
			PageId: pageID,
		})
	}
	return nil
}

// 清空操作员的所有页面权限
func (d *DbSysUser) ClearAllPagePermissions(userID int) error {
	if d == nil || d.db == nil {
		return errors.New("db connection is nil")
	}
	return d.db.Delete(&SysOperatorPage{}, "user_id = ?", userID).Error
}

// 验证用户名和密码
func (d *DbSysUser) Login(userName string, password string) (bool, error) {
	if d == nil || d.db == nil {
		return false, errors.New("db connection is nil")
	}

	var user SysUser
	result := d.db.First(&user, "user_name = ?", userName)
	if result.Error != nil {
		return false, fmt.Errorf("user or password error")
	}
	if !user.IsEnabled {
		return false, fmt.Errorf("user or password error")
	}
	hashedPassword, err := EncryptPassword(password)
	if err != nil {
		return false, err
	}

	if user.Password != hashedPassword {
		return false, fmt.Errorf("user or password error")
	}
	return true, nil
}

type userRolePermission struct {
	UserID        int    `json:"userId"`
	UserName      string `json:"userName"`
	NickName      string `json:"nickName"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	IsEnabled     bool   `json:"isEnabled"`
	RoleID        int    `json:"roleId"`
	RoleName      string `json:"roleName"`
	InitialPageID int    `json:"initialPageId"`
	IsChanged     bool   `json:"isChanged"`
	PageIDList    []int  `json:"pageIdList"`
}

func (d *DbSysUser) GetUserRolePermission(userName string) (userRolePermission, error) {
	if d == nil || d.db == nil {
		return userRolePermission{}, errors.New("db connection is nil")
	}

	var sysUser SysUser
	result := d.db.Model(&SysUser{}).
		Where("user_name = ?", userName).
		First(&sysUser)
	if result.Error != nil {
		return userRolePermission{}, fmt.Errorf("查询用户信息失败: %w", result.Error)
	}

	var sysRole SysRole
	result = d.db.Model(&SysRole{}).
		Where("role_id = ?", sysUser.RoleId).
		First(&sysRole)
	if result.Error != nil {
		return userRolePermission{}, fmt.Errorf("查询角色信息失败: %w", result.Error)
	}

	var pageIDs []int
	err := d.db.Model(&SysOperatorPage{}).
		Where("user_id = ?", sysUser.UserId).
		Pluck("page_id", &pageIDs).Error
	if err != nil {
		return userRolePermission{}, fmt.Errorf("fail to query page ids: %w", err)
	}

	user := userRolePermission{
		UserID:        sysUser.UserId,
		UserName:      sysUser.UserName,
		NickName:      sysUser.NickName,
		Password:      sysUser.Password,
		Email:         sysUser.Email,
		Phone:         sysUser.Phone,
		IsEnabled:     sysUser.IsEnabled,
		RoleID:        sysRole.RoleId,
		RoleName:      sysRole.RoleName,
		InitialPageID: sysUser.InitialPageId,
		IsChanged:     sysUser.IsChanged,
		PageIDList:    pageIDs,
	}

	return user, nil
}

const Seed_TS = "*T-Scale*" // 固定 cost 值

func EncryptPassword(password string) (string, error) {
	pswStr := password + Seed_TS
	hashedPassword := md5.Sum([]byte(pswStr))
	hashedPasswordStr := fmt.Sprintf("%x", hashedPassword)
	return hashedPasswordStr, nil
}
