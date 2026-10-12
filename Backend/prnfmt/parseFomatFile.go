package prnfmt

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"tmaxsrv/comm"

	"github.com/xuri/excelize/v2"
)

//go:embed varTable.json
var defaultVarTableBytes []byte

type VarStruct struct {
	id       uint16
	startPos uint16
	endPos   uint16
	align    uint16
	maxlen   uint16
}

type RptVarStruct struct {
	id       uint8
	startPos uint16
	align    uint8
	maxLen   uint8
}

type ScaleVarOrder struct {
	ScaleVarTable []ScaleVar `json:"var"`
}
type ScaleVar struct {
	ValueVame string `json:"valuename"`
	Id        int    `json:"id"`
	Comment   string `json:"comment"`
}

var (
	VarList    []VarStruct
	RptVarList []RptVarStruct
	VarTable   ScaleVarOrder
)

func GetFormatLines(fileNames string) ([]string, bool) {
	var formatArray []string
	res := false
	lineIdx := 0

	f, err := os.Open(fileNames)
	if err != nil {
		log.Println("open file error")
	}
	defer f.Close() // 打开文件出错处理
	if nil == err {
		buff := bufio.NewReader(f) // 读入缓存
		for {

			line, err := buff.ReadString('\n') // 以'\n'为结束符读入一行
			if err != nil || io.EOF == err {
				break
			}
			if lineIdx == 0 && strings.Contains(line, "\ufeff") {
				line = strings.Replace(line, "\ufeff", "", 1)
			}
			formatArray = append(formatArray, line)
		}

	}
	if len(formatArray) > 1 {
		res = true
	}
	return formatArray, res
}

func ReadTableFromFile(tableFilePath string) ScaleVarOrder {
	var tempTable ScaleVarOrder

	// 1. 尝试从指定路径或候选路径读取文件
	candidatePaths := []string{
		tableFilePath,
		comm.GetStaticFilePath("varTable.json"),
		filepath.Join(comm.GetSrvDataPath(), "varTable.json"),
		filepath.Join(comm.GetExePath(), "srvdata", "varTable.json"),
		filepath.Join(comm.GetExePath(), "varTable.json"),
		filepath.Join("srvdata", "varTable.json"),
		filepath.Join("..", "srvdata", "varTable.json"),
	}

	for _, p := range candidatePaths {
		if p == "" {
			continue
		}
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			if err := json.Unmarshal(data, &tempTable); err == nil && len(tempTable.ScaleVarTable) > 0 {
				return tempTable
			}
		}
	}

	// 2. 如果磁盘文件均无法加载，使用编译期嵌入的内置默认变量表兜底，保证服务永不因缺文件而崩溃
	if len(defaultVarTableBytes) > 0 {
		if err := json.Unmarshal(defaultVarTableBytes, &tempTable); err == nil {
			log.Printf("[prnfmt] Notice: varTable.json not found on disk, using embedded default varTable")
			return tempTable
		}
	}

	log.Printf("[prnfmt] Warning: Failed to load varTable from disk and embedded fallback (candidate paths: %v)", candidatePaths)
	return tempTable
}

// 读取excel文件，返回行列数组
func getBarCodeTypeId(filename string, tempLan string, tempType string) string {
	ret := ""
	f, err := excelize.OpenFile(filename)
	if err != nil {
		baseName := filepath.Base(filename)
		altPath := comm.GetStaticFilePath(baseName)
		if altPath != "" && altPath != filename {
			f, err = excelize.OpenFile(altPath)
		}
	}
	if err != nil {
		fmt.Println("读取excel文件出错", err.Error())
		return ret
	}
	defer f.Close()
	sheet1 := f.GetSheetName(0)
	rows, err := f.GetRows(sheet1)
	if err != nil {
		fmt.Println("读取excel文件出错", err.Error())
		return ret
	}
	cols := []string{}
	isOk := false

	for i, row := range rows {
		if i == 0 { // 取得第一行的所有数据---execel表头
			cols = append(cols, row...)
			isOk = true
			// fmt.Println("列信息", cols)

		} else {
			if isOk {
				isOk = false
				break
			}
		}
	}
	colId := findRowColId(cols, tempLan)

	row1 := []string{}
	rowDes := []string{}
	isDesOk := false
	if colId > 0 {
		cols1, err := f.GetCols(sheet1)
		if err != nil {
			fmt.Println("读取excel文件出错", err.Error())
			return ret
		}

		for i, col := range cols1 {
			if i == 1 { // 取得第一行的所有数据---execel表头
				row1 = append(row1, col...)
				isOk = true
				// fmt.Println("行信息", row1)

			} else if i == colId {
				rowDes = append(rowDes, col...)
				isDesOk = true
				// fmt.Println("行信息", rowDes)

			} else {
				if isOk && isDesOk {
					break
				}
			}
		}
	}

	rowId := findRowColId(row1, tempType)
	if rowId > 0 {
		ret = rowDes[rowId]
	}
	return ret
}

func findRowColId(tempArr []string, tempVar string) int {
	id := 0
	for i := 0; i < len(tempArr); i++ {
		if tempArr[i] == tempVar {
			id = i
			break
		}
	}
	return id
}
