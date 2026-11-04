package svc

import (
	"path/filepath"
	"tmaxsrv/comm"
)

type ModeSettingProvider struct {
	myId      string
	settingPb *DbModeSetting
}

var (
	MODE_SETTING_DB_FILE = filepath.Join(comm.GetSrvDataPath(), "modesetting.db")
)

func NewModeSettingProvider() *ModeSettingProvider {
	settingPb, _ := NewDbModeSetting(MODE_SETTING_DB_FILE)
	if settingPb != nil {
		for mode := uint(0); mode <= 3; mode++ {
			_, _ = settingPb.GetModeSetting(mode)
		}
	}
	return &ModeSettingProvider{myId: "ModeSettingProvider", settingPb: settingPb}
}

func (p *ModeSettingProvider) GetModeSetting(scaleMode uint) ([]ModeSetting, error) {
	setting, err := p.settingPb.GetModeSetting(scaleMode)
	return setting, err
}

func (p *ModeSettingProvider) ModifyModeSe(modeSetting ModeSetting) error {
	return p.settingPb.UpdateModeSetting(modeSetting)
}
