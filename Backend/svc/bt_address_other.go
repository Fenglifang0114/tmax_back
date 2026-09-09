//go:build !windows

package svc

import (
	"tinygo.org/x/bluetooth"
)

func parseBTAddress(macStr string) (bluetooth.Address, error) {
	var addr bluetooth.Address
	addr.Set(macStr)
	return addr, nil
}
