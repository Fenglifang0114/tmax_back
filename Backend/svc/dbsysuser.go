package svc

import (
	"crypto/md5"
	"fmt"
	"time"
	l "tmaxsrv/log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//系统用户表

type DbSysUser struct {
	dbName string
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
	var err error
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
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
	if err = db.AutoMigrate(
		&SysRole{},
		&SysUser{},
		&SysOperatorPage{},
	); err != nil {
		l.Log.Debug("failed to migrate database of scale connection")
	}
	sysUser := &DbSysUser{dbName: dbName}
	initRoles(sysUser)
	// 初始化管理员账号
	initAdmin(sysUser)

	return &DbSysUser{dbName: dbName}, nil
}

// 初始化三种角色
func initRoles(sysUser *DbSysUser) error {
	var roleCount int64
	roles, err := sysUser.ListAllRoles()
	if err != nil {
		return err
	}
	roleCount = int64(len(roles))
	if roleCount == 0 {
		// 初始化三种角色
		roles := []SysRole{
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

		for _, role := range roles {
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
	// 检查管理员账号是否已存在
	var count int64
	db, err := gorm.Open(sqlite.Open(sysUser.dbName), &gorm.Config{})
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
	db.Model(&SysUser{}).Count(&count)
	if count > 1 {
		//修改管理员的isChanged为true
		db.Model(&SysUser{}).Where("user_id = ?", 1).Update("is_changed", true)
		return nil
	}
	if count > 0 {
		return nil
	}

	// 初始化管理员账号
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

	// 初始化管理员账号
	if err := sysUser.CreateUser(admin); err != nil {
		return err
	}
	return nil
}

// 创建角色(初始化时使用，因为角色固定为三种)
func (d *DbSysUser) CreateRole(role *SysRole) error {
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
	if err = db.Create(role).Error; err != nil {
		return err
	}
	return nil
}

// 获取所有角色
func (d *DbSysUser) ListAllRoles() ([]SysRole, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []SysRole{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []SysRole{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var roles []SysRole
	result := db.Find(&roles)
	return roles, result.Error
}

// 创建用户
func (d *DbSysUser) CreateUser(user *SysUser) error {
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
	// 密码加密

	user.Password, err = EncryptPassword(user.Password)
	if err != nil {
		return err
	}

	if err = db.Create(user).Error; err != nil {
		return err
	}
	return nil
}

// 获取用户信息(通过用户名)
func (d *DbSysUser) GetUserByUsername(userName string) (*SysUser, error) {
	var err error
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
	var user SysUser
	result := db.First(&user, "user_name = ?", userName)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// 获取用户信息(通过用户名)
func (d *DbSysUser) GetUserByUserId(userId int) (*SysUser, error) {
	var err error
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
	var user SysUser
	result := db.First(&user, "user_id = ?", userId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// 获取用户信息(通过RFID卡号)
func (d *DbSysUser) GetUserByRfid(rfid string) (*SysUser, error) {
	var err error
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
	var user SysUser
	result := db.First(&user, "rfid = ?", rfid)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// 获取多个用户详情
func (d *DbSysUser) GetManyUserByUserIds(userIds []int) ([]SysUser, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []SysUser{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []SysUser{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var users []SysUser
	result := db.Find(&users, "user_id IN ?", userIds)
	return users, result.Error
}

// 更新用户

func (d *DbSysUser) UpdateUser(user *SysUser, pswUpdated bool) error {

	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("fail to update user: %w", err)

	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("fail to update user: %w", err)
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	// 1. 验证用户是否存在
	var existingUser SysUser
	result := db.Model(&SysUser{}).Where("user_id = ?", user.UserId).First(&existingUser)
	if result.Error != nil {
		return fmt.Errorf("fail to update user: %w", result.Error)
	}
	// 1. 密码加密
	if pswUpdated {
		encryptedPassword, err := EncryptPassword(user.Password)
		if err != nil {
			return fmt.Errorf("fail to update user: %w", err)
		}
		user.Password = encryptedPassword
		user.IsChanged = true
	}

	// 2. 执行更新操作（仅更新提供的字段）
	result = db.Model(&SysUser{}).
		Where("user_id = ?", user.UserId). // 明确指定更新条件
		Updates(user)                      // 使用map传递需要更新的字段
	if result.Error != nil {
		return fmt.Errorf("fail to update user: %w", result.Error)
	}

	return nil
}

// 删除用户(软删除)
func (d *DbSysUser) DeleteUser(userID []int) error {
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
	for _, id := range userID {
		if id == 1 {
			continue
		}
		if err = db.Delete(&SysUser{}, "user_id = ?", id).Error; err != nil {
			return err
		}
	}
	// 删除用户页面权限
	for _, id := range userID {
		if err := d.ClearAllPagePermissions(id); err != nil {
			return err
		}
	}

	return nil
}

// 查询用户列表
func (d *DbSysUser) ListUsers() ([]SysUser, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return []SysUser{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []SysUser{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	var users []SysUser
	result := db.Find(&users)
	if result.Error != nil {
		return []SysUser{}, err
	}
	if result.RowsAffected == 0 {
		return []SysUser{}, err
	}
	return users, result.Error
}

// 修改密码
func (d *DbSysUser) ChangePassword(userId int, newPassword string) error {
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 查询用户信息
	var user SysUser
	result := db.First(&user, "user_id = ?", userId)
	if result.Error != nil {
		return fmt.Errorf("user not found: %w", result.Error)
	}

	// 加密新密码
	hashedNewPassword, err := EncryptPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt new password: %w", err)
	}

	// 更新密码
	user.Password = hashedNewPassword
	user.IsChanged = true
	err = db.Save(&user).Error
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// 禁用用户

func (d *DbSysUser) DisableUser(userID int, isEnabled bool, updateBy int, updateByName string) error {

	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var user SysUser
	result := db.First(&user, "user_id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("user not found: %w", result.Error)
	}

	user.IsEnabled = isEnabled
	user.UpdatedBy = updateBy
	user.UpdatedByName = updateByName
	if err := db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	return nil
}

// 撤销操作员的页面权限
func (d *DbSysUser) updateUserPermission(userID int, pageIDList []int) error {
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

	//删除用户的所有页面权限
	db.Delete(&SysOperatorPage{}, "user_id = ?", userID)
	//添加用户的页面权限
	for _, pageID := range pageIDList {
		db.Create(&SysOperatorPage{
			UserId: userID,
			PageId: pageID,
		})
	}
	return nil
}

// 清空操作员的所有页面权限
func (d *DbSysUser) ClearAllPagePermissions(userID int) error {
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
	return db.Delete(&SysOperatorPage{}, "user_id = ?", userID).Error
}

// 验证用户名和密码
func (d *DbSysUser) Login(userName string, password string) (bool, error) {
	var err error
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
	var user SysUser
	result := db.First(&user, "user_name = ?", userName)
	if result.Error != nil {
		return false, fmt.Errorf("user or password error")
	}
	//验证用户是否可用
	if !user.IsEnabled {
		return false, fmt.Errorf("user or password error")
	}
	hashedPassword, err := EncryptPassword(password)
	if err != nil {
		return false, err
	}

	// 比较加密后的密码
	if user.Password != hashedPassword {
		return false, fmt.Errorf("user or password error")
	}
	return true, nil
}

// 查询一个用用户的信息和角色信息，还有权限信息融合在一个信息里面
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
	var err error
	db, err := gorm.Open(sqlite.Open(d.dbName), &gorm.Config{})
	if err != nil {
		return userRolePermission{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return userRolePermission{}, err
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// 1. 查询用户基本信息
	var sysUser SysUser
	result := db.Model(&SysUser{}).
		Where("user_name = ?", userName).
		First(&sysUser)
	if result.Error != nil {
		return userRolePermission{}, fmt.Errorf("查询用户信息失败: %w", result.Error)
	}

	// 2. 根据用户的role_id查询角色信息
	var sysRole SysRole
	result = db.Model(&SysRole{}).
		Where("role_id = ?", sysUser.RoleId).
		First(&sysRole)
	if result.Error != nil {
		return userRolePermission{}, fmt.Errorf("查询角色信息失败: %w", result.Error)
	}

	// 查询用户可访问的页面 ID 列表
	var pageIDs []int
	err = db.Model(&SysOperatorPage{}).
		Where("user_id = ?", sysUser.UserId).
		Pluck("page_id", &pageIDs).Error
	if err != nil {
		return userRolePermission{}, fmt.Errorf("fail to query page ids: %w", err)
	}

	// 3. 合并结果
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

// 加密过程
const Seed_TS = "*T-Scale*" // 固定 cost 值

func EncryptPassword(password string) (string, error) {
	pswStr := password + Seed_TS
	hashedPassword := md5.Sum([]byte(pswStr))

	// 将MD5字节数组转换为32位十六进制字符串
	hashedPasswordStr := fmt.Sprintf("%x", hashedPassword)
	return hashedPasswordStr, nil
}
