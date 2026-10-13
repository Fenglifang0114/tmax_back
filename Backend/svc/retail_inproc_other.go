//go:build !darwin

package svc

func RetailInProcEnabled() bool {
	return false
}

func RetailInProcOnDetail(scaleId int64, list DetailList) {}

func RetailInProcHandleUiReq(jsonStr string) bool {
	return false
}

func RetailInProcServiceAction(serviceId int64) (string, bool) {
	return "", false
}
