//go:build windows

package svc

import (
	"fmt"

	"tinygo.org/x/bluetooth"
)

func parseBTAddress(macStr string) (bluetooth.Address, error) {
	mac, err := bluetooth.ParseMAC(macStr)
	if err != nil {
		return bluetooth.Address{}, fmt.Errorf("MAC地址格式错误: %v", err)
	}
	return bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}, nil
}
