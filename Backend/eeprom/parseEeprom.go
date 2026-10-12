package eeprom

import (
	"fmt"
	"strconv"
	"strings"
	"tmaxsrv/comm"

	"github.com/xuri/excelize/v2"
)

type EepromStruct struct {
	EepromStruct []EepromField
}
type EepromField struct {
	FieldName   string
	Size        int
	Addr        int
	Type        string
	SubType     string
	Permission  int
	Values      string
	Comments    string
	Description string
	Category    string
}

const (
	TMAX_EEPROM_EXCEL = "T-MAXeeprom.xlsx"
)

func getExcelPath() string {
	currentPath := comm.GetStaticFilePath(TMAX_EEPROM_EXCEL)
	fmt.Println(currentPath)
	return currentPath
}

// 读取excel文件
func GetExcelData() (EepromStruct, bool) {

	var fieldData EepromStruct
	var filename = getExcelPath()
	f, err := excelize.OpenFile(filename)
	if err != nil {
		fmt.Println("read excel file error", err.Error())
		return fieldData, false
	}
	defer f.Close()
	sheet1 := f.GetSheetName(0)
	rows, err := f.GetRows(sheet1)
	if err != nil {
		fmt.Println("read excel file error", err.Error())
		return fieldData, false
	}
	for i, row := range rows {
		if i > 1 && len(row) >= 14 && strings.Contains(row[4], "M") {
			var tempEepromField EepromField
			tempEepromField.FieldName = row[4]
			size, err := strconv.Atoi(row[5])
			if err != nil {
				size = 0
			}
			tempEepromField.Size = size
			tempEepromField.Comments = row[8]
			addr, err := strconv.Atoi(row[9])
			if err != nil {
				addr = 0
			}
			tempEepromField.Addr = addr
			tempEepromField.Type = row[10]
			tempEepromField.SubType = row[11]
			permission, err := strconv.Atoi(row[12])
			if err != nil {
				permission = 0
			}
			tempEepromField.Permission = permission
			tempEepromField.Values = row[13]
			tempEepromField.Description = row[14]
			tempEepromField.Category = row[15]

			if tempEepromField.Size != 0 {
				fieldData.EepromStruct = append(fieldData.EepromStruct, tempEepromField)
			}
		}
	}
	return fieldData, true
}

func GetFuncAddrSize(fieldName string) (addr int, size int) {

	excelData, res := GetExcelData()
	if !res {
		return 0, 0
	}

	for _, data := range excelData.EepromStruct {
		// 找到满足条件的数据
		if data.FieldName == fieldName {
			return data.Addr, data.Size
		}
	}

	return 0, 0

}
