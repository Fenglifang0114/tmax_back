package svc

import (
	"path/filepath"
	"tmaxsrv/comm"
)

type SysUserProvider struct {
	myId   string
	infoPb *DbSysUser
}

func NewSysUserProvider() *SysUserProvider {
	database := filepath.Join(comm.GetSrvDataPath(), "sysuser.db")
	infoPb, _ := NewDbSysUser(database)
	return &SysUserProvider{myId: "SysUserProvider", infoPb: infoPb}
}

// 获取用户详情
func (p *SysUserProvider) GetUserDetail(userName string) (*userRolePermission, error) {
	user, err := p.infoPb.GetUserRolePermission(userName)
	return &user, err
}

// 获取用户详情
func (p *SysUserProvider) GetUserInfoById(userId int) (*SysUser, error) {
	user, err := p.infoPb.GetUserByUserId(userId)
	return user, err
}

// 获取多个用户详情
func (p *SysUserProvider) GetManyUserInfoById(userIds []int) ([]SysUser, error) {
	users, err := p.infoPb.GetManyUserByUserIds(userIds)
	return users, err
}

// 获取用户信息
func (p *SysUserProvider) GetUserInfo(userName string) (*SysUser, error) {
	user, err := p.infoPb.GetUserByUsername(userName)
	if err != nil {
		return nil, err
	}

	return user, err
}

// 通过 RFID 获取用户信息
func (p *SysUserProvider) GetUserByRfid(rfid string) (*SysUser, error) {
	return p.infoPb.GetUserByRfid(rfid)
}

// 新增用户
func (p *SysUserProvider) AddUser(user *SysUser) error {
	return p.infoPb.CreateUser(user)
}

// 删除用户
func (p *SysUserProvider) DeleteUser(userIds []int) error {
	return p.infoPb.DeleteUser(userIds)
}

// 更新用户
func (p *SysUserProvider) UpdateUser(user *SysUser, pswUpdated bool) error {

	return p.infoPb.UpdateUser(user, pswUpdated)

}

// 用户页面权限修改
func (p *SysUserProvider) UpdateUserPageId(userId int, pageIds []int) error {
	return p.infoPb.updateUserPermission(userId, pageIds)
}

// 清除用户所有页面权限
func (p *SysUserProvider) ClearAllPagePermissions(userId int) error {
	return p.infoPb.ClearAllPagePermissions(userId)
}

// 修改密码
func (p *SysUserProvider) UpdateUserPassword(userId int, password string) error {
	return p.infoPb.ChangePassword(userId, password)
}

// 禁用用户
func (p *SysUserProvider) DisableUser(userId int, isEnabled bool, updateBy int, updateByName string) error {
	return p.infoPb.DisableUser(userId, isEnabled, updateBy, updateByName)
}

// 登录
func (p *SysUserProvider) Login(userName string, password string) (bool, error) {
	return p.infoPb.Login(userName, password)
}

// 获取所有用户列表
func (p *SysUserProvider) GetAllUsers() ([]SysUser, error) {
	return p.infoPb.ListUsers()
}
