package svc

import (
	"path/filepath"
	"tmaxsrv/comm"
)

type ModbusServiceProvider struct {
	myId   string
	connPb *DbModbusServices
}

func NewModbusServiceProvider() *ModbusServiceProvider {
	database := filepath.Join(comm.GetSrvDataPath(), "modbus_services.db")
	connPb, _ := NewDbModbusServices(database)
	return &ModbusServiceProvider{myId: "ModbusServiceProvider", connPb: connPb}
}

func (p *ModbusServiceProvider) GetModbusServiceList() ([]ModbusServiceInfo, error) {
	return p.connPb.GetModbusServiceList()
}

func (p *ModbusServiceProvider) InsertModbusService(service *ModbusServiceInfo) error {
	return p.connPb.InsertModbusService(service)
}

func (p *ModbusServiceProvider) UpdateModbusService(service ModbusServiceInfo) error {
	return p.connPb.UpdateModbusService(service)
}

func (p *ModbusServiceProvider) DeleteModbusService(id uint) error {
	return p.connPb.DeleteModbusService(id)
}
