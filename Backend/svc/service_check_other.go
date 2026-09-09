//go:build !windows

package svc

func IsServiceInstalled(serviceName string) bool {
	return true
}

func IsServiceRunning(serviceName string) bool {
	return true
}
