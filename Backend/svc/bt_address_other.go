//go:build !windows

package svc

import (
	"fmt"
	"strings"

	"tinygo.org/x/bluetooth"
)

func parseBTAddress(macStr string) (bluetooth.Address, error) {
	macStr = strings.TrimSpace(macStr)
	if macStr == "" {
		return bluetooth.Address{}, fmt.Errorf("蓝牙设备地址不能为空")
	}
	var addr bluetooth.Address
	addr.Set(macStr)
	if addr.String() == "00000000-0000-0000-0000-000000000000" && macStr != "00000000-0000-0000-0000-000000000000" {
		return addr, fmt.Errorf("无效的蓝牙设备地址/UUID格式: %s", macStr)
	}
	return addr, nil
}

