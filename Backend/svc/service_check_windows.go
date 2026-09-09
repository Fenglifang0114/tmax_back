//go:build windows

package svc

import (
	"log"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func IsServiceInstalled(serviceName string) bool {
	m, err := mgr.Connect()
	if err != nil {
		log.Printf("%v", err)
	}
	defer m.Disconnect()

	services, err := m.ListServices()
	if err != nil {
		log.Printf("%v", err)
	}

	for _, s := range services {
		if s == serviceName {
			return true
		}
	}
	return false
}

func IsServiceRunning(serviceName string) bool {
	m, err := mgr.Connect()
	if err != nil {
		log.Printf("mgr.Connect error: %v", err)
		return false
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		log.Printf("OpenService error: %v", err)
		return false
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		log.Printf("Query error: %v", err)
		return false
	}

	return status.State == svc.Running
}
