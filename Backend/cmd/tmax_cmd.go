package cmd

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	m "tmaxsrv/comm"
	l "tmaxsrv/log"
	"tmaxsrv/util"
)

var (
	EN_ENG_CMD_TMAX           []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf1, 0x00, 0xf9, 0x16, 0x57, 0x5e, 0xa5, 0x5a}
	GET_RANDOM_DATA_CMD_TMAX  []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf7, 0x00, 0x1C, 0xC0, 0xE8, 0xF8, 0xa5, 0x5a}
	DIS_ENG_CMD_TMAX          []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf2, 0x00, 0x8b, 0xfd, 0x08, 0x8d, 0xa5, 0x5a}
	ZERO_CMD_TMAX             []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe1, 0x03, 0x00, 0x0c, 0xD6, 0x90, 0x0C, 0xa5, 0x5a}
	TARE_CMD_TMAX             []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0xE1, 0x05, 0x00, 0xE9, 0x00, 0x2F, 0xAA, 0xA5, 0x5A}
	ZERO_UNSTABLE_CMD_TMAX    []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe4, 0x01, 0x00, 0xaa, 0x9e, 0x10, 0x98, 0xa5, 0x5a}
	TARE_UNSTABLE_CMD_TMAX    []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0xE4, 0x02, 0x00, 0xd8, 0x75, 0x4f, 0x4b, 0xA5, 0x5a}
	REBOOT_CMD_TMAX           []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0x05, 0x55, 0x00, 0x1E, 0x1D, 0x98, 0x49, 0xA5, 0x5A}
	READ_WEIGHT_CMD_TMAX      []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe1, 0x05, 0x00, 0xac, 0x24, 0x0e, 0x03, 0xa5, 0x5a}
	EN_CONT_MODE_CMD_TMAX     []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe1, 0x07, 0x00, 0x49, 0xf2, 0xb1, 0xa5, 0xa5, 0x5a}
	DIS_CONT_MODE_CMD_TMAX    []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe1, 0x08, 0x00, 0xf4, 0x75, 0x8c, 0x8d, 0xa5, 0x5a}
	EN_PASSTH_MODE_CMD_TMAX   []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf3, 0x00, 0x59, 0xE4, 0xC9, 0x51, 0xa5, 0x5a}
	DIS_PASSTH_MODE_CMD_TMAX  []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf4, 0x00, 0x6E, 0x2B, 0xB7, 0x2B, 0xa5, 0x5a}
	GET_BUILD_INFO_CMD_TMAX   []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0x56, 0x00, 0x6C, 0xF6, 0xC7, 0x9A, 0xa5, 0x5a} //20230926@FLF
	GET_SCALE_INFO_CMD_TMAX   []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf0, 0x00, 0x2B, 0x0F, 0x96, 0x82, 0xa5, 0x5a} //20231101@FLF
	GET_INSERT_PLU_ADDR_TMAX  []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0xf3, 0x02, 0x00, 0xc0, 0xf4, 0xc0, 0xae, 0xA5, 0x5A}
	GET_PLU_HEAD_TMAX         []byte = []byte{0x5a, 0xa5, 0x00, 0x11, 0xf1, 0x01, 0x00, 0x20, 0x06, 0xa0, 0x00, 0x00, 0x64, 0x6c, 0xb0, 0xb0, 0x36, 0xa5, 0x5a}
	ERASE_INSERT_PLU_TMAX     []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0xf3, 0x03, 0x00, 0x12, 0xED, 0x01, 0x72, 0xA5, 0x5A}
	GET_SCALE_TIME_CMD_TMAX   []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xF4, 0x02, 0x00, 0xC5, 0xFF, 0x87, 0x3B, 0xa5, 0x5a}                                     //20230112@FLF
	READ_EEPROM_256_CMD_TMAX  []byte = []byte{0x5a, 0xa5, 0x00, 0x11, 0xf1, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x97, 0xab, 0x7f, 0x0d, 0xa5, 0x5a} //20240124@FLF
	READ_EEPROM_512_CMD_TMAX  []byte = []byte{0x5a, 0xa5, 0x00, 0x11, 0xf1, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x4B, 0xC6, 0xE5, 0xBA, 0xa5, 0x5a} //20240129@FLF
	GET_FACTORY_INFO_CMD_TMAX []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf6, 0x00, 0xCE, 0xD9, 0x29, 0x24, 0xa5, 0x5a}
	GET_BASIC_DATA_CMD_TMAX   []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xf8, 0x00, 0xA1, 0x47, 0xD5, 0xD0, 0xa5, 0x5a}
	PAY_BILL_ON_CMD_TMAX      []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe1, 0x09, 0x01, 0x22, 0xad, 0x50, 0xe6, 0xa5, 0x5a} //20240829@FLF结账发送开启
	ANSWER_ALIVE_CMD_TMAX     []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0x05, 0xFF, 0x00, 0x96, 0x88, 0xAB, 0xAA, 0xa5, 0x5a}
	CAL_HEART_CMD_TMAX        []byte = []byte{0x5a, 0xa5, 0x00, 0x0b, 0xe1, 0x34, 0x00, 0x0b, 0xeb, 0x43, 0x43, 0xa5, 0x5a}

	GET_WIRED_IP_CMD_TMAX   []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0x05, 0x6A, 0x00, 0x93, 0x68, 0x08, 0x54, 0xA5, 0x5A}
	GET_WIRED_DHCP_CMD_TMAX []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0x05, 0x6D, 0x00, 0xA4, 0xA7, 0x76, 0x2E, 0xA5, 0x5A}

	EN_CODE_CMD_TMAX         []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0xE1, 0x13, 0x00, 0x1C, 0x87, 0x0B, 0x1F, 0xA5, 0x5A} //开启内码
	DIS_CODE_CMD_TMAX        []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0xE1, 0x14, 0x00, 0x2B, 0x48, 0x75, 0x65, 0xA5, 0x5A} //关闭内码
	ASK_ROM_VERSION_CMD_TMAX []byte = []byte{0x5A, 0xA5, 0x00, 0x0B, 0x05, 0x59, 0x00, 0xD1, 0x71, 0xFA, 0xB2, 0xA5, 0x5A} //查询eeprom的版本号

)

func NewComposerTMAX() *m.CmdComposer {

	composer := m.CmdComposer{}
	composer.ScaleCat = m.SCALE_TMAX
	composer.ComposeCmd = ComposeCmdTMAX
	return &composer
}

func ComposeCmdTMAX(composer *m.CmdComposer, cmd m.CmdType, cmdData m.CmdData) ([]byte, int, error) {
	if composer.ScaleCat != m.SCALE_TMAX {
		return nil, CMD_TIMEOUT_IMMEDIATE, fmt.Errorf("ScaleCat is not SCALE_TMAX")
	}

	switch cmd {
	case m.CMD_CHECK_FAC_MODE:
		return EN_ENG_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_DIS_FAC_MODE:
		return DIS_ENG_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_ZERO:
		return ZERO_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_TARE:
		return TARE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_ZERO_UNSTABLE:
		return ZERO_UNSTABLE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_TARE_UNSTABLE:
		return TARE_UNSTABLE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_READ_WEIGHT:
		return READ_WEIGHT_CMD_TMAX, CMD_TIMEOUT_SHORT_1500_MS, nil
	case m.CMD_EN_CONTINUE_MODE:
		return EN_CONT_MODE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_DIS_CONTINUE_MODE:
		return DIS_CONT_MODE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_REBOOT:
		return REBOOT_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_EN_PASSTH:
		return EN_PASSTH_MODE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_DIS_PASSTH:
		return DIS_PASSTH_MODE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_BUILD_INFO:
		return GET_BUILD_INFO_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil //20230926@FLF
	case m.CMD_GET_SCALE_TIME:
		return GET_SCALE_TIME_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil //20230926@FLF
	case m.CMD_READ_EEPROM_256:
		return READ_EEPROM_256_CMD_TMAX, CMD_TIMEOUT_MEDIUM_2000_MS, nil //20240125@FLF
	case m.CMD_READ_EEPROM_8:
		addr := parseReadAddrTMAX(cmdData.Data.(string))
		return readDataCmdTMAX(uint32(addr)), CMD_TIMEOUT_MEDIUM_2000_MS, nil //20240703@FLF
	case m.CMD_READ_EEPROM_512:
		return READ_EEPROM_512_CMD_TMAX, CMD_TIMEOUT_MEDIUM_2000_MS, nil //20240129@FLF
	case m.CMD_GET_SCALE_INFO:
		return GET_SCALE_INFO_CMD_TMAX, CMD_TIMEOUT_SHORT_1500_MS, nil //20231101@FLF
	case m.CMD_GET_FACTORY_INFO:
		return GET_FACTORY_INFO_CMD_TMAX, CMD_TIMEOUT_SHORT_1500_MS, nil //20240417@FLF
	case m.CMD_INSERT_PLU_ADDR:
		return GET_INSERT_PLU_ADDR_TMAX, CMD_TIMEOUT_SHORT_1500_MS, nil //20231101@FLF
	case m.CMD_GET_PLU_HEAD:
		return GET_PLU_HEAD_TMAX, CMD_TIMEOUT_SHORT_1500_MS, nil //20230103@FLF
	case m.CMD_ERASE_INSERT_PLU:
		return ERASE_INSERT_PLU_TMAX, CMD_TIMEOUT_SHORT_1500_MS, nil //20230109@FLF
	case m.CMD_GET_WEIGHT_ERR:
		addr, data := parseReadFlashTMAX(cmdData.Data.(string))
		return readFlashCmdTMAX(uint32(addr), data), CMD_TIMEOUT_SHORT_1500_MS, nil
	case m.CMD_ERASE_FLASH:
		addr := cmdData.Data.(int)
		return eraseCmdTMAX(uint32(addr)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_ERASE_FLASH_512:
		addr := cmdData.Data.(int)
		return eraseCmdTMAX_512(uint32(addr)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WRITE_FLASH_256:
		addr, data := parseWrDataTMAX(cmdData.Data.(string))
		return wrDataCmdTMAX(uint32(addr), data, FILE_CHUNK_SIZE_256_TMAX), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WRITE_FLASH_8:
		addr, data := parseWrDataTMAX(cmdData.Data.(string))
		return wrDataCmdTMAX(uint32(addr), data, FILE_CHUNK_SIZE_28_TMAX), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WRITE_FLASH_512:
		addr, data := parseWrDataTMAX(cmdData.Data.(string))
		return wrDataCmdTMAX(uint32(addr), data, FILE_CHUNK_SIZE_512_TMAX), CMD_TIMEOUT_MED_LONG_4000_MS, nil
	case m.CMD_WIFI_DATA_PASSTH:
		return sendDataToWifiCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_GET_AP_LIST:
		return getApListCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_AT_VERSION:
		return getAtVersionCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_AT_MODE:
		return getAtModeCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil

	//初始化WiFi模块命令

	case m.CMD_WIFI_CLOSE_SERVER_CMD:
		return closeServerCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil

	case m.CMD_WIFI_DIS_BT_CMD:
		return disBtCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_EN_AUTO_CONN_CMD:
		return enAutoConnCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_WIFI_STATION_MODE_CMD:
		return setWifiStationModeCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_MULTI_CONN_CMD:
		return setMultiConnCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_DIS_RECONN_CMD:
		return disAutoReconnCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_DIS_IP_PORT_INFO_CMD:
		return disIpPortInfoCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_SINGLE_CONN_CMD:
		return setSingleConnCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_CONN_PORT_CMD:
		return setConnPortCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_TIME_OUT_CMD:
		return setTimeOutCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_PASSTH_MODE_CMD:
		return setPassthModeCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_SCAN_AP_PARAM_CMD:
		return setScanApParamCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil

	case m.CMD_WIFI_EN_DHCP:
		return EnWifiDhcpCmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_EN_DHCP_32:
		return EnWifiDhcp32CmdTMAX(), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_STATIC_IP:
		fields := strings.Split(cmdData.Data.(string), ",")
		return setWifiStaticIpCmdTMAX(fields[0], fields[1], fields[2]), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_SET_STATIC_IP_32:
		fields := strings.Split(cmdData.Data.(string), ",")
		return setWifiStaticIp32CmdTMAX(fields[0], fields[1], fields[2]), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WIFI_GET_IP_INFO:
		return getIpInfoCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_GET_IP_INFO_32:
		return getIp32InfoCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_CHANGE_WIFI_MODE:
		return changeWifiModeCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_GET_AP_INFO:
		return getApInfoCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_GET_AP_INFO_32:
		return getAp32InfoCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_GET_IP_MODE:
		return getIpModCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_GET_IP_MODE_32:
		return getIp32ModCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_WIFI_CONN_AP:
		fields := strings.Split(cmdData.Data.(string), ",")
		return connectWifiApCmdTMAX(fields[0], fields[1], fields[2]), CMD_TIMEOUT_LONG_20000_MS, nil //ESP8266 默认超时15秒
	case m.CMD_WIFI_CONN_AP32:
		fields := strings.Split(cmdData.Data.(string), ",")
		return connectWifiAp32CmdTMAX(fields[0], fields[1], fields[2]), CMD_TIMEOUT_LONG_20000_MS, nil //ESP8266 默认超时15秒
	case m.CMD_WIFI_CONN_AP_ONE_KEY:
		fields := strings.Split(cmdData.Data.(string), ",")
		return connectWifiApCmdTMAX(fields[0], fields[1], fields[2]), CMD_TIMEOUT_MEDIUM_2000_MS, nil //ESP8266
	case m.CMD_WIFI_DISCONN_AP:
		return disconnectWifiApCmdTMAX(), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_BT_DATA_PASSTH:
		return sendDataToBTCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_MODIFY_BT_NAME:
		return modifyBTNameCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_DEL_PLU:
		return getDelPluCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_SET_DECIMAl_VALUE:
		return getSetDecimalValueCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_SET_GADUATION1_VALUE:
		return getSetGaduationValueCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_WRITE_EEPROM:
		addr, data := parseWrDataTMAX(cmdData.Data.(string))
		return getModifyEepromCmdTMAX(uint32(addr), data), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_SET_SCALE_TIME:
		return getSetScaleTimeCmdTMAX(cmdData.Data.(string)), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_MODIFY_VAR_VALUE:
		data, _ := hex.DecodeString(cmdData.Data.(string))
		return getModifyVarValueCmdTMAX(data), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_EN_FACTORY_MODE:
		data, _ := hex.DecodeString(cmdData.Data.(string))
		return enFactoryModeCmdTMAX(data), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_RANDOM_DATA:
		return GET_RANDOM_DATA_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_BASIC_DATA:
		return GET_BASIC_DATA_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_LIMIT_TO_SCALE:
		data, _ := hex.DecodeString(cmdData.Data.(string))
		return getSetLimitToScaleCmdTMAX(data), CMD_TIMEOUT_MEDIUM_2000_MS, nil
	case m.CMD_OPEN_BILL_SEND:
		return PAY_BILL_ON_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_CAl_WGT:
		dataInt := cmdData.Data.(int)
		return setSw15CmdTax_4Byte(CMDID_SET_CAL_WGT_TMAX, uint32(dataInt)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SEND_CAL_HEART_BEAT:
		return CAL_HEART_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_DECIMAL_VALUE:
		return composeCmd(CMDID_GET_DECIMAL_VALUE_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_MAX_RANGE1:
		return composeCmd(CMDID_GET_MAX_RANGE1_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_MAX_RANGE2:
		return composeCmd(CMDID_GET_MAX_RANGE2_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_MAX_RANGE1:
		dataInt := cmdData.Data.(int)
		return setSw15CmdTax_4Byte(CMDID_SET_MAX_RANGE1_TMAX, uint32(dataInt)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_MAX_RANGE2:
		dataInt := cmdData.Data.(int)
		return setSw15CmdTax_4Byte(CMDID_SET_MAX_RANGE2_TMAX, uint32(dataInt)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_GADUATION1_VALUE:
		return composeCmd(CMDID_GET_GADUATION_VALUE1_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_GADUATION2_VALUE:
		return composeCmd(CMDID_GET_GADUATION_VALUE2_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_GADUATION2_VALUE:
		return getSetSw15ParameterCmdTMAX(CMDID_SET_GADUATION_VALUE2_TMAX, cmdData.Data.(string)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_ZERO_TRACKING:
		return composeCmd(CMDID_GET_ZERO_TRACK_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_ZERO_TRACKING:
		return getSetSw15ParameterCmdTMAX(CMDID_SET_ZERO_TRACK_TMAX, cmdData.Data.(string)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_WEIGHT_UNIT:
		return composeCmd(CMDID_GET_WGT_UNIT_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_WEIGHT_UNIT:
		return getSetSw15ParameterCmdTMAX(CMDID_SET_WGT_UNIT_TMAX, cmdData.Data.(string)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_GRAV_ACC:
		return composeCmd(CMDID_GET_GRAVITY_ACCEL_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_FORCE_UNTARE:
		return composeCmd(CMDID_SET_FORCE_UNTARE_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_GRAV_ACC:
		dataInt := cmdData.Data.(int)
		return setSw15CmdTax_4Byte(CMDID_SET_GRAVITY_ACCEL_TMAX, uint32(dataInt)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_MANUAL_ZERO:
		return getSetSw15ParameterCmdTMAX(CMDID_SET_MANUAL_ZERO_TMAX, cmdData.Data.(string)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_MANUAL_ZERO:
		return composeCmd(CMDID_GET_MANUAL_ZERO_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_INITIAL_ZERO:
		return getSetSw15ParameterCmdTMAX(CMDID_SET_INIT_ZERO_TMAX, cmdData.Data.(string)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_INITIAL_ZERO:
		return composeCmd(CMDID_GET_INIT_ZERO_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_WIRED_IP:
		return GET_WIRED_IP_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_GET_WIRED_DHCP:
		return GET_WIRED_DHCP_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_WIRED_IP:

		return setWiredIpCmdTMAX(CMDID_SET_WIRED_IP_TMAX, cmdData.Data.(string)), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_WIRED_DHCP:
		dataInt := cmdData.Data.(int)
		return composeCmd(CMDID_SET_WIRED_DHCP_TMAX, 0, []byte{byte(dataInt)}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil

	case m.CMD_GET_SEAL_STATUS:
		return composeCmd(CMDID_GET_SEAL_STATUS_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil

	case m.CMD_GET_MODEL:
		return composeCmd(CMDID_GET_MODEL_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil

	case m.CMD_SET_SOFT_SEAL:
		data, _ := hex.DecodeString(cmdData.Data.(string))
		return composeCmd(CMDID_SET_SOFT_SEAL_TMAX, 0, data), CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_REMOVE_SOFT_SEAL:
		data, _ := hex.DecodeString(cmdData.Data.(string))
		return composeCmd(CMDID_REMOVE_SOFT_SEAL_TMAX, 0, data), CMD_TIMEOUT_VERY_SHORT_200_MS, nil

	case m.CMD_REMOVE_SOFT_SEAL_ONCE:
		return composeCmd(CMDID_REMOVE_SOFT_SEAL_ONCE_TMX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil

	case m.CMD_EN_CODE:
		return EN_CODE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_DIS_CODE:
		return DIS_CODE_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_ASK_ROM_VERSION:
		return ASK_ROM_VERSION_CMD_TMAX, CMD_TIMEOUT_VERY_SHORT_200_MS, nil
	case m.CMD_SET_SERIAL_PORT:
		dataStr := cmdData.Data.(string)
		return composeCmd(CMDID_SET_SERIAL_PORT_TMAX, 0, []byte(dataStr)), CMD_TIMEOUT_LONG_20000_MS, nil
	case m.CMD_GET_SERIAL_PORT:
		return composeCmd(CMDID_GET_SERIAL_PORT_TMAX, 0, []byte{}), CMD_TIMEOUT_VERY_SHORT_200_MS, nil

	}

	return nil, CMD_TIMEOUT_IMMEDIATE, nil

}

var GET_AT_VERSION_CMD []byte = []byte("AT+GMR\r\n") //查看wifi 模块的版本信息会包含 ESP32 等信息
// var GET_AT_VERSION_CMD []byte = []byte("AT+BLENAME?\r\n") //查看wifi 模块的版本信息会包含 ESP32 等信息

var GET_AP_LIST_CMD []byte = []byte("AT+CWLAP\r\n")
var CONNECT_AP_CMD []byte = []byte("AT+CWJAP_DEF=\"%s\",\"%s\"\r\n")         // ssid, password, bssid
var CONNECT_AP_CMD_32 []byte = []byte("AT+CWJAP=\"%s\",\"%s\"\r\n")          // ssid, password, bssid
var CONNECT_AP_CMD_32_B []byte = []byte("AT+CWJAP=\"%s\",\"%s\",\"%s\"\r\n") // ssid, password, bssid
var CONNECT_AP_CMD_B []byte = []byte("AT+CWJAP=\"%s\",\"%s\"\r\n")           // ssid, password, bssid
var DISCONNECT_AP_CMD []byte = []byte("AT+CWQAP\r\n")                        // ssid, password, bssid
var GET_AP_INFO_CMD []byte = []byte("AT+CWJAP_DEF?\r\n")                     // ssid, password, bssid
var GET_AP_INFO_CMD_32 []byte = []byte("AT+CWJAP?\r\n")                      // ssid, password, bssid
var GET_IP_INFO_CMD []byte = []byte("AT+CIPSTA_CUR?\r\n")                    // ssid, password, bssid
var GET_IP_INFO_CMD_32 []byte = []byte("AT+CIPSTA?\r\n")                     // ssid, password, bssid
var GET_IP_MODE_CMD []byte = []byte("AT+CWDHCP_CUR?\r\n")                    //改为CWDHCP
var GET_IP_MODE_CMD_32 []byte = []byte("AT+CWDHCP?\r\n")                     //改为CWDHCP

var EN_DHCP_DEF_CMD []byte = []byte("AT+CWDHCP_DEF=1,1\r\n")
var EN_DHCP_DEF_CMD_32 []byte = []byte("AT+CWDHCP=1,1\r\n")

var SET_WIFI_STATIC_IP_DEF_CMD []byte = []byte("AT+CIPSTA_DEF=\"%s\",\"%s\",\"%s\"\r\n") // ip, gateway, netmask
var SET_WIFI_STATIC_IP_DEF_CMD_32 []byte = []byte("AT+CIPSTA=\"%s\",\"%s\",\"%s\"\r\n")  // ip, gateway, netmask
var CHANGE_WIFI_MODE_CMD []byte = []byte("AT+CWMODE=1\r\n")

// var GET_AT_MODE_CMD []byte = []byte("AT+CWMODE?\r\n")
var GET_AT_MODE_CMD []byte = []byte("AT+CWAUTOCONN?\r\n")

// 初始化WiFi模块命令
var CLOSE_SERVER_CMD []byte = []byte("AT+CIPSERVER=0\r\n") //关闭TCP服务器

var DIS_BT_CMD []byte = []byte("AT+BLEINIT=0\r\n")                 //关闭蓝牙
var EN_AUTO_CONN_CMD []byte = []byte("AT+CWAUTOCONN=1\r\n")        //打开自动连接
var SET_WIFI_STATION_MODE_CMD []byte = []byte("AT+CWMODE=1,1\r\n") //设置Wi-Fi模式为station

// 连上IP地址后，设置下面的参数
var SET_MULTI_CONN_CMD []byte = []byte("AT+CIPMUX=1\r\n")                 //设置多连接
var SET_MAX_CONN_CMD []byte = []byte("AT+CIPSERVERMAXCONN=1\r\n")         //设置最大连接数1
var DIS_RECONN_CMD []byte = []byte("AT+CWRECONNCFG=20,0\r\n")             //断开重连
var DIS_IP_PORT_INFO_CMD []byte = []byte("AT+CIPDINFO=0\r\n")             //不提示对端IP及端口号
var SET_TCP_SERVER_CMD []byte = []byte("AT+CIPSERVER=1,8580,\"TCP\"\r\n") //设置TCP服务器，端口8580
var SET_TIME_OUT_CMD []byte = []byte("AT+CIPSTO=0\r\n")                   //设置本地TCP服务器超时
var SET_PASSTH_MODE_CMD []byte = []byte("AT+CIPMODE=0\r\n")               //设置传输模式 0-普通 1-透传
var SET_SCAN_AP_PARAM_CMD []byte = []byte("AT+CWLAPOPT=1,14\r\n")         //设置扫描AP的参数

//初始化WiFi模块命令

const (
	PACKET_HEAD_TMAX         = 0x5AA5
	PACKET_TAIL_TMAX         = 0xA55A
	FILE_CHUNK_SIZE_256_TMAX = 275 //FLF
	FILE_CHUNK_SIZE_28_TMAX  = 27  //FLF
	FILE_CHUNK_SIZE_512_TMAX = 531 //@FLF20240110
	EARSE_CHUNK_SIZE_TMAX    = 0x13
	CAL_VALUE_SIZE_TMAX      = 0x11
	ERASE_SIZE_TMAX          = 0x800
	ERASE_SIZE_TMAX_512      = 0x200
	READ_EEPROM_TMAX_8       = 0x08
)

const (
	CMDID_READ_WEIGHT_TMAX         = 0xE101
	CMDID_READ_STABLE_WEIGHT_TMAX  = 0xE102
	CMDID_PREF_ZERO_TMAX           = 0xE103
	CMDID_PREF_ZERO_ON_STABLE_TMAX = 0xE104
	CMDID_PREF_TARE_TMAX           = 0xE105
	CMDID_PREF_TARE_ON_STABLE_TMAX = 0xE106
	CMDID_EN_CONT_WEIGHT_TMAX      = 0xE107
	CMDID_DIS_CONT_WEIGHT_TMAX     = 0xE108
	// CMDID_SET_1ST_CAP_TMAX         = 0xE109
	// CMDID_SET_2ND_CAP_TMAX         = 0xE10A
	CMDID_PAY_BILL_ON_TMAX    = 0xE109 // 开结账开关
	CMDID_PAY_BILL_OFF_TMAX   = 0xE10A // 关结账
	CMDID_SET_PRICE_MODE_TMAX = 0xE10B // 设置计价秤
	CMDID_SET_WGT_MODE_TMAX   = 0xE10C // 设置计重秤
	CMDID_CLARE_LIMIT_TMAX    = 0xE10D // 清除上下限
	CMDID_SET_LIMIT_TMAX      = 0xE10E // 设置上下限
	CMDID_SWITCH_LIMIT_TMAX   = 0xE10F // 切换上下限
	CMDID_PAY_BILL_HEAD_TMAX  = 0xE110 // 结账头
	CMDID_PAY_BILL_MID_TMAX   = 0xE111 // 结账中
	CMDID_PAY_BILL_TAIL_TMAX  = 0xE112 // 结账尾

	CMDID_CONT_CODE_TMAX = 0xE115 // 连续发送内码

	CMDID_SET_MAX_RANGE1_TMAX = 0xE300 //设置量程1
	CMDID_GET_MAX_RANGE1_TMAX = 0xE301 //读取量程1
	CMDID_SET_MAX_RANGE2_TMAX = 0xE305 //设置量程2
	CMDID_GET_MAX_RANGE2_TMAX = 0xE306 //读取量程2

	CMDID_SET_CAL_WGT_TMAX          = 0xE30A //设置标定重量
	CMDID_SET_DECIMAL_VALUE_TMAX    = 0xE30F //设置小数点
	CMDID_GET_DECIMAL_VALUE_TMAX    = 0xE310 //读取小数点
	CMDID_SET_GADUATION_VALUE1_TMAX = 0xE314 //设置分度值1
	CMDID_GET_GADUATION_VALUE1_TMAX = 0xE315 //读取分度值1
	CMDID_SET_GADUATION_VALUE2_TMAX = 0xE319 //设置分度值2
	CMDID_GET_GADUATION_VALUE2_TMAX = 0xE31A //读取分度值2
	CMDID_SET_WGT_UNIT_TMAX         = 0xE31E //设置称重单位
	CMDID_GET_WGT_UNIT_TMAX         = 0xE31F //读取称重单位
	CMDID_SET_MANUAL_ZERO_TMAX      = 0xE323 //设置手动归零
	CMDID_GET_MANUAL_ZERO_TMAX      = 0xE324 //读取手动归零
	CMDID_SET_ZERO_TRACK_TMAX       = 0xE328 //设置零点追踪
	CMDID_GET_ZERO_TRACK_TMAX       = 0xE329 //读取零点追踪
	CMDID_SET_INIT_ZERO_TMAX        = 0xE32D //设置初始置零
	CMDID_GET_INIT_ZERO_TMAX        = 0xE32E //读取初始置零
	CMDID_SET_GRAVITY_ACCEL_TMAX    = 0xE332 //设置重力加速度
	CMDID_GET_GRAVITY_ACCEL_TMAX    = 0xE333 //读取重力加速度
	CMDID_SET_FORCE_UNTARE_TMAX     = 0xE33A //强制解除扣重
	CMDID_SET_SERIAL_PORT_TMAX      = 0xE340 //设置串口
	CMDID_GET_SERIAL_PORT_TMAX      = 0xE341 //读取串口

)

const (
	CMDID_SEND_DATA_TO_BT_TMAX   = 0xF201
	CMDID_SEND_DATA_TO_WIFI_TMAX = 0xF202
	CMDID_SEND_DATA_TO_PRN_TMAX  = 0xF203
)

const (
	CMDID_REBOOT_TMAX            = 0x0555
	CMDID_READ_SCALE_INFO_TMAX   = 0x05F0
	CMDID_EN_FAC_TMAX            = 0x05F1
	CMDID_DIS_FAC_TMAX           = 0x05F2
	CMDID_EN_PASSTH_TMAX         = 0x05F3
	CMDID_DIS_PASSTH_TMAX        = 0x05F4
	CMDID_GET_MAX_PACK_SIZE_TMAX = 0x05F5
	CMDID_GET_FACTORY_INFO_TMAX  = 0x05F6
	CMDID_GET_RANDOM_DATA_TMAX   = 0x05F7
	CMDID_GET_BASIC_DATA_TMAX    = 0x05F8
	CMDID_ANSWER_ALIVE_TMAX      = 0x05FF //秤会问是否活着

	CMDID_GET_WIRED_IP_TMAX   = 0x056A
	CMDID_SET_WIRED_IP_TMAX   = 0x056B
	CMDID_SET_WIRED_DHCP_TMAX = 0x056C
	CMDID_GET_WIRED_DHCP_TMAX = 0x056D

	CMDID_GET_MODEL_TMAX = 0x0558 //读取型号 T-MAX

	CMDID_GET_SEAL_STATUS_TMAX      = 0x0570
	CMDID_SET_SOFT_SEAL_TMAX        = 0x0571
	CMDID_REMOVE_SOFT_SEAL_TMAX     = 0x0572
	CMDID_REMOVE_SOFT_SEAL_ONCE_TMX = 0x0573
)
const (
	CMDID_READ_FLASH_TMAX        = 0xF101
	CMDID_WRITE_FLASH_TMAX       = 0xF102 //写flash和写rom一样的，秤会根据地址自己偏移。命令不区分
	CMDID_ERASE_FLASH_TMAX       = 0xF103
	CMDID_READ_EEPROM_TMAX       = 0xF104
	CMDID_WRITE_EEPROM_TMAX      = 0xF105
	CMDID_ERASE_EEPROM_TMAX      = 0xF106
	CMDID_READ_ROM_TMAX          = 0xF107
	CMDID_WRITE_ROM_TMAX         = 0xF108
	CMDID_ERASE_ROM_TMAX         = 0xF109
	CMDID_DEL_PLU_TMAX           = 0xF301
	CMDID_INSERT_PLU_TMAX        = 0xF302
	CMDID_ERASE_INSERT_PLU_TMAX  = 0xF303
	CMDID_SET_SCALE_TIME_TMAX    = 0xF401
	CMDID_GET_SCALE_TIME_TMAX    = 0xF402
	CMDID_SCALE_PASSTH_DATA_TMAX = 0xFF23 // virtual command ID
	CMDID_DOWN_PLU_TMAX          = 0xFF24
	CMDID_MODIFY_VAR_TMAX        = 0xF501
	CMDID_EN_FACTORY_MODE        = 0xF601
	CMDID_GET_AT_VERSION_TMAX    = 0xFF25
)

const (
	RESP_RESULT_OK   = 0x06
	RESP_RESULT_FAIL = 0x15
)

const (
	CMD_IDENTIFY_IDX = 2
	CMD_TYPE_IDX     = 3
)

const PACK_LEN_WITHOUT_DATA_TMAX = 13 // for TMAX protocol version, 13: Head: 2 + len:2 + CmdId: 1 + SubCmdId: 1 + SeqNo:1 + CRC:4 + Tail:2
func composeCmd(cmdID uint16, seqNo byte, data []byte) []byte {
	// Calculate packet length
	var packLen uint16 = uint16(PACK_LEN_WITHOUT_DATA_TMAX + len(data))
	cmd := make([]byte, packLen)
	binary.BigEndian.PutUint16(cmd[0:2], PACKET_HEAD_TMAX)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(cmd[2:4], packLen-2) // packet length without header
	binary.BigEndian.PutUint16(cmd[4:6], cmdID)
	cmd[6] = seqNo
	if len(data) > 0 {
		copy(cmd[7:], data)
	}
	// 计算与添加校验码
	checksum := util.Crc32MPEG2(cmd[2 : packLen-6])
	binary.BigEndian.PutUint32(cmd[packLen-6:], checksum)
	// 添加包尾
	binary.BigEndian.PutUint16(cmd[packLen-2:], PACKET_TAIL_TMAX)
	fmt.Printf("%X\n", cmd)
	return cmd
}

const PACK_LEN_HEADER_TMAX = 13 //9 = 2个头+2个f501+1个00+2个长度+4个校验位+2个尾巴
// 构建命令   //FLF
func composeModifyVarCmd(cmdID uint16, data []byte) []byte {
	// Calculate packet length
	var packLen uint16 = uint16(PACK_LEN_HEADER_TMAX + len(data))
	cmd := make([]byte, packLen)
	binary.BigEndian.PutUint16(cmd[0:2], PACKET_HEAD_TMAX)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(cmd[2:4], packLen-2) // packet length without header
	binary.BigEndian.PutUint16(cmd[4:6], cmdID)
	cmd[6] = 0x00
	if len(data) > 0 {
		copy(cmd[7:], data)
	}
	// 计算与添加校验码
	checksum := util.Crc32MPEG2(cmd[2 : packLen-6])
	binary.BigEndian.PutUint32(cmd[packLen-6:], checksum)
	// 添加包尾
	binary.BigEndian.PutUint16(cmd[packLen-2:], PACKET_TAIL_TMAX)
	fmt.Printf("%x", cmd)

	return cmd
}

const PACK_LEN_FACTORY_TMAX = 17 // = 2个头+2个长度+2个f5+1个00+4个数据+4个校验位+2个尾巴
// 构建命令   //FLF
func composeFactoryCmd(cmdID uint16, data []byte) []byte {
	// Calculate packet length
	var packLen uint16 = uint16(PACK_LEN_FACTORY_TMAX)
	cmd := make([]byte, packLen)
	binary.BigEndian.PutUint16(cmd[0:2], PACKET_HEAD_TMAX)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(cmd[2:4], packLen-2) // packet length without header
	binary.BigEndian.PutUint16(cmd[4:6], cmdID)
	cmd[6] = 0x00
	//data 有6个字节，data[0]data[1] 随机数 data[2]data[3]data[4]data[5] md5
	//data[0]和data[2]做XOR  得到的数据x1
	//data[1]和data[5]做XOR  得到的数据x2
	//x1 data[3] data[4] x2  4个字节做CRC得到此命令的数据部分
	data[2] = data[0] ^ data[2]
	data[5] = data[1] ^ data[5]
	dataCrc := util.Crc32MPEG2(data[2:]) //此处是数据，只不过数据值是CRC的校验值，后面还是有校验的
	binary.BigEndian.PutUint32(cmd[7:11], dataCrc)
	// 计算与添加校验码
	checksum := util.Crc32MPEG2(cmd[2 : packLen-6])
	binary.BigEndian.PutUint32(cmd[packLen-6:], checksum)
	// 添加包尾
	binary.BigEndian.PutUint16(cmd[packLen-2:], PACKET_TAIL_TMAX)
	fmt.Printf("%x", cmd)

	return cmd
}

// 构建一个数据包   //FLF
// 0x5a, 0xa5, 0x00, 0x11, 0xf1, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x97, 0xab, 0x7f, 0x0d, 0xa5, 0x5a
// 0x5a, 0xa5, 0x00, 0x11, 0xf1, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x4B, 0xC6, 0xE5, 0xBA, 0xa5, 0x5a
const PACK_LEN_READ_EEPROM_TMAX = 19

func readDataCmdTMAX(addr uint32) []byte {
	// 构建包头
	var packLen uint16 = uint16(PACK_LEN_READ_EEPROM_TMAX)
	packet := make([]byte, packLen)
	binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD_TMAX)
	//构建数据长度  总长度-包头
	binary.BigEndian.PutUint16(packet[2:4], packLen-2)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(packet[4:6], uint16(CMDID_READ_FLASH_TMAX))
	//构建保留数据 00
	binary.BigEndian.PutUint16(packet[6:8], 0x00)
	// 构建地址
	binary.BigEndian.PutUint32(packet[7:11], addr)

	// 构建数据长度
	binary.BigEndian.PutUint16(packet[11:13], uint16(READ_EEPROM_TMAX_8))

	// 计算与添加校验码
	checksum := util.Crc32MPEG2(packet[2 : packLen-6])
	binary.BigEndian.PutUint32(packet[packLen-6:], checksum)

	// 添加包尾
	binary.BigEndian.PutUint16(packet[packLen-2:], PACKET_TAIL_TMAX)

	return packet
}

// 构建一个数据包   //FLF
func wrDataCmdTMAX(addr uint32, data []byte, packetLen uint16) []byte {
	// 构建包头
	packet := make([]byte, packetLen)
	binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD_TMAX)
	//构建数据长度  总长度-包头
	binary.BigEndian.PutUint16(packet[2:4], packetLen-2)

	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(packet[4:6], uint16(CMDID_WRITE_FLASH_TMAX))
	//构建保留数据 00
	binary.BigEndian.PutUint16(packet[6:8], 0x00)
	// 构建地址
	binary.BigEndian.PutUint32(packet[7:11], addr)

	// 构建数据长度
	binary.BigEndian.PutUint16(packet[11:13], uint16(len(data)))

	// 复制数据
	copy(packet[13:], data)

	// 计算与添加校验码
	checksum := util.Crc32MPEG2(packet[2 : packetLen-6])
	binary.BigEndian.PutUint32(packet[packetLen-6:], checksum)

	// 添加包尾
	binary.BigEndian.PutUint16(packet[packetLen-2:], PACKET_TAIL_TMAX)

	return packet
}

// CMD:  5a a5 00 11 f1 01 00 20 04 C0 00 00 08 5C F9 F4 7B a5 5a   //FLF
// 读取flash组命令
func readFlashCmdTMAX(addr uint32, data []byte) []byte { // erase size will 2K
	// 构建包头
	packet := make([]byte, EARSE_CHUNK_SIZE_TMAX)
	binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD_TMAX)
	//构建数据长度  总长度-包头
	binary.BigEndian.PutUint16(packet[2:4], EARSE_CHUNK_SIZE_TMAX-2)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(packet[4:6], uint16(CMDID_READ_FLASH_TMAX))
	//构建保留数据 00
	binary.BigEndian.PutUint16(packet[6:8], 0x00)
	// 构建地址
	binary.BigEndian.PutUint32(packet[7:11], addr)
	// 读取长度
	copy(packet[11:13], data)
	// 计算与添加校验码
	checksum := util.Crc32MPEG2(packet[2 : EARSE_CHUNK_SIZE_TMAX-6])
	binary.BigEndian.PutUint32(packet[EARSE_CHUNK_SIZE_TMAX-6:], checksum)

	// 添加包尾
	binary.BigEndian.PutUint16(packet[EARSE_CHUNK_SIZE_TMAX-2:], PACKET_TAIL_TMAX)

	return packet
}

// CMD:  5a a5 00 11 f1 03 00 08 01 e0 00 08 00 75 E8 A3 E3 a5 5a   //FLF
// 擦除原本秤上的打印格式
func eraseCmdTMAX(addr uint32) []byte { // erase size will 2K
	// 构建包头
	packet := make([]byte, EARSE_CHUNK_SIZE_TMAX)
	binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD_TMAX)
	//构建数据长度  总长度-包头
	binary.BigEndian.PutUint16(packet[2:4], EARSE_CHUNK_SIZE_TMAX-2)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(packet[4:6], uint16(CMDID_ERASE_FLASH_TMAX))
	//构建保留数据 00
	binary.BigEndian.PutUint16(packet[6:8], 0x00)
	// 构建地址
	binary.BigEndian.PutUint32(packet[7:11], addr)
	// 擦除长度
	binary.BigEndian.PutUint16(packet[11:13], ERASE_SIZE_TMAX)

	// 计算与添加校验码
	checksum := util.Crc32MPEG2(packet[2 : EARSE_CHUNK_SIZE_TMAX-6])
	binary.BigEndian.PutUint32(packet[EARSE_CHUNK_SIZE_TMAX-6:], checksum)

	// 添加包尾
	binary.BigEndian.PutUint16(packet[EARSE_CHUNK_SIZE_TMAX-2:], PACKET_TAIL_TMAX)

	return packet
}

// CMD:  5a a5 00 11 f1 03 00 08 01 e0 00 08 00 75 E8 A3 E3 a5 5a   //FLF
// 擦除原本秤上的默认参数备份  一次512
func eraseCmdTMAX_512(addr uint32) []byte { // erase size will 512
	// 构建包头
	packet := make([]byte, EARSE_CHUNK_SIZE_TMAX)
	binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD_TMAX)
	//构建数据长度  总长度-包头
	binary.BigEndian.PutUint16(packet[2:4], EARSE_CHUNK_SIZE_TMAX-2)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(packet[4:6], uint16(CMDID_ERASE_FLASH_TMAX))
	//构建保留数据 00
	binary.BigEndian.PutUint16(packet[6:8], 0x00)
	// 构建地址
	binary.BigEndian.PutUint32(packet[7:11], addr)
	// 擦除长度
	binary.BigEndian.PutUint16(packet[11:13], ERASE_SIZE_TMAX_512)

	// 计算与添加校验码
	checksum := util.Crc32MPEG2(packet[2 : EARSE_CHUNK_SIZE_TMAX-6])
	binary.BigEndian.PutUint32(packet[EARSE_CHUNK_SIZE_TMAX-6:], checksum)

	// 添加包尾
	binary.BigEndian.PutUint16(packet[EARSE_CHUNK_SIZE_TMAX-2:], PACKET_TAIL_TMAX)

	return packet
}

// 设置SW15 的参数 4个字节的
func setSw15CmdTax_4Byte(cmdId uint16, value uint32) []byte { // erase size will 2K
	// 构建包头
	packet := make([]byte, CAL_VALUE_SIZE_TMAX)
	binary.BigEndian.PutUint16(packet[0:2], PACKET_HEAD_TMAX)
	//构建数据长度  总长度-包头
	binary.BigEndian.PutUint16(packet[2:4], CAL_VALUE_SIZE_TMAX-2)
	// 构建命令ID与命令类型
	binary.BigEndian.PutUint16(packet[4:6], uint16(cmdId))
	//构建保留数据 00
	binary.BigEndian.PutUint16(packet[6:8], 0x00)
	// 构建标定值
	binary.BigEndian.PutUint32(packet[7:11], value)

	// 计算与添加校验码
	checksum := util.Crc32MPEG2(packet[2 : CAL_VALUE_SIZE_TMAX-6])
	binary.BigEndian.PutUint32(packet[CAL_VALUE_SIZE_TMAX-6:], checksum)

	// 添加包尾
	binary.BigEndian.PutUint16(packet[CAL_VALUE_SIZE_TMAX-2:], PACKET_TAIL_TMAX)

	return packet
}

func MyStringToBytes(str string) []byte {
	byteSlice := make([]byte, len(str))
	copy(byteSlice, str)
	return byteSlice
}

// 修改蓝牙名称
func modifyBTNameCmdTMAX(name string) []byte {
	l.Log.Debug("compose modify BT name cmd")
	// MODIFY_BT_NAME_CHUNK_SIZE should include all data except BT name
	// data := "TTM:REN-" + name + "\r\n\x00"//旧的蓝牙
	data := "AT+BLENAME=" + "\"" + name + "\"" + "\r\n\x00"

	bytes := MyStringToBytes(data)
	return composeCmd(0xf201, 0, bytes)
}

// Get AP list
func getApListCmdTMAX() []byte {
	l.Log.Debug("compose Get AP list cmd")
	return composeCmd(0xf202, 0, GET_AP_LIST_CMD)
}

// Get at version
func getAtVersionCmdTMAX() []byte {
	l.Log.Debug("compose Get AP list cmd")
	return composeCmd(0xf202, 0, GET_AT_VERSION_CMD)
}

// Get at version
func getAtModeCmdTMAX() []byte {
	l.Log.Debug("compose Get AP list cmd")
	return composeCmd(0xf202, 0, GET_AT_MODE_CMD)
}

// 关闭TCP服务器
func closeServerCmdTMAX() []byte {
	l.Log.Debug("compose close server cmd")
	return composeCmd(0xf202, 0, CLOSE_SERVER_CMD)
}

func disBtCmdTMAX() []byte {
	l.Log.Debug("compose disable BT cmd")
	return composeCmd(0xf202, 0, DIS_BT_CMD)
}

func enAutoConnCmdTMAX() []byte {
	l.Log.Debug("compose enable auto connect cmd")
	return composeCmd(0xf202, 0, EN_AUTO_CONN_CMD)
}

func setWifiStationModeCmdTMAX() []byte {
	l.Log.Debug("compose set wifi station mode cmd")
	return composeCmd(0xf202, 0, SET_WIFI_STATION_MODE_CMD)
}

func setMultiConnCmdTMAX() []byte {
	l.Log.Debug("compose set multi conn cmd")
	return composeCmd(0xf202, 0, SET_MULTI_CONN_CMD)
}

func disAutoReconnCmdTMAX() []byte {
	l.Log.Debug("compose disable auto reconnect cmd")
	return composeCmd(0xf202, 0, DIS_RECONN_CMD)
}

func disIpPortInfoCmdTMAX() []byte {
	l.Log.Debug("compose disable IP port info cmd")
	return composeCmd(0xf202, 0, DIS_IP_PORT_INFO_CMD)
}

func setSingleConnCmdTMAX() []byte {
	l.Log.Debug("compose set single conn cmd")
	return composeCmd(0xf202, 0, SET_MAX_CONN_CMD)
}

func setConnPortCmdTMAX() []byte {
	l.Log.Debug("compose get conn port cmd")
	return composeCmd(0xf202, 0, SET_TCP_SERVER_CMD)
}

func setTimeOutCmdTMAX() []byte {
	l.Log.Debug("compose set time out cmd")
	return composeCmd(0xf202, 0, SET_TIME_OUT_CMD)
}

func setPassthModeCmdTMAX() []byte {
	l.Log.Debug("compose set passth mode cmd")
	return composeCmd(0xf202, 0, SET_PASSTH_MODE_CMD)
}

func setScanApParamCmdTMAX() []byte {
	l.Log.Debug("compose set scan ap param cmd")
	return composeCmd(0xf202, 0, SET_SCAN_AP_PARAM_CMD)
}

// Change Wifi Mode
func changeWifiModeCmdTMAX() []byte {
	l.Log.Debug("compose change wifi mode cmd")
	return composeCmd(0xf202, 0, CHANGE_WIFI_MODE_CMD)
}

// Send data to BT
func sendDataToBTCmdTMAX(data string) []byte {
	l.Log.Debug("compose send data to BT cmd")
	return composeCmd(0xf201, 0, []byte(data))
}

// Send data to Wifi
func sendDataToWifiCmdTMAX(data string) []byte {
	l.Log.Debug("compose send data to Wifi cmd")
	return composeCmd(0xf202, 0, []byte(data))
}

func EnWifiDhcpCmdTMAX() []byte {
	l.Log.Debug("compose set wifi dynamic IP cmd")
	return composeCmd(0xf202, 0, EN_DHCP_DEF_CMD)
}

func EnWifiDhcp32CmdTMAX() []byte {
	l.Log.Debug("compose set wifi dynamic IP cmd")
	return composeCmd(0xf202, 0, EN_DHCP_DEF_CMD_32)
}

func setWifiStaticIpCmdTMAX(ip string, gateway string, netmask string) []byte {
	l.Log.Debug("compose set wifi to static IP cmd")
	return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(SET_WIFI_STATIC_IP_DEF_CMD), ip, gateway, netmask)))
}

// 连接静态IP ESP32
func setWifiStaticIp32CmdTMAX(ip string, gateway string, netmask string) []byte {
	l.Log.Debug("compose set wifi to static IP cmd")
	return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(SET_WIFI_STATIC_IP_DEF_CMD_32), ip, gateway, netmask)))
}

// Connect to specifi AP
func connectWifiApCmdTMAX(ssid string, passwd string, bssid string) []byte {
	l.Log.Debug("compose connect to Wifi AP cmd")
	if bssid == "" {
		return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(CONNECT_AP_CMD), ssid, passwd)))

	}
	return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(CONNECT_AP_CMD_B), ssid, passwd, bssid)))
}

func connectWifiAp32CmdTMAX(ssid string, passwd string, bssid string) []byte {
	l.Log.Debug("compose connect to Wifi AP cmd")
	if bssid == "" {
		return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(CONNECT_AP_CMD_32), ssid, passwd)))
	}
	return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(CONNECT_AP_CMD_32), ssid, passwd))) //测试可用
	// return composeCmd(0xf202, 0, []byte(fmt.Sprintf(string(CONNECT_AP_CMD_32_B), ssid, passwd, bssid)))
}

// Connect to specifi AP
func disconnectWifiApCmdTMAX() []byte {
	l.Log.Debug("compose disconnect to Wifi AP cmd")
	return composeCmd(0xf202, 0, []byte{})
}

// Get wifi AP info from scale
func getApInfoCmdTMAX() []byte {
	l.Log.Debug("compose get IP info cmd")
	return composeCmd(0xf202, 0, GET_AP_INFO_CMD)
}

// Get wifi AP info from scale 32
func getAp32InfoCmdTMAX() []byte {
	l.Log.Debug("compose get IP info cmd")
	return composeCmd(0xf202, 0, GET_AP_INFO_CMD_32)
}

// Get IP info from scale
func getIpInfoCmdTMAX() []byte {
	l.Log.Debug("compose get IP info cmd")
	return composeCmd(0xf202, 0, GET_IP_INFO_CMD)
}

// Get IP info from scale 32
func getIp32InfoCmdTMAX() []byte {
	l.Log.Debug("compose get IP info cmd")
	return composeCmd(0xf202, 0, GET_IP_INFO_CMD_32)
}

func getIpModCmdTMAX() []byte {
	l.Log.Debug("compose get IP mode cmd")
	return composeCmd(0xf202, 0, GET_IP_MODE_CMD)
}

func getIp32ModCmdTMAX() []byte {
	l.Log.Debug("compose get IP mode cmd")
	return composeCmd(0xf202, 0, GET_IP_MODE_CMD_32)

}

func getDelPluCmdTMAX(data string) []byte {
	l.Log.Debug("compose get IP mode cmd")
	return composeCmd(0xf301, 0, []byte(data))
}

func getSetDecimalValueCmdTMAX(data string) []byte {
	l.Log.Debug("compose set decimal value cmd")
	num, _ := strconv.Atoi(data)

	// 转换为单字节数据
	data1 := []byte{byte(num)}
	return composeCmd(0xE30F, 0, []byte(data1))
}

func getSetSw15ParameterCmdTMAX(cmdID uint16, data string) []byte {
	l.Log.Debug("compose set sw15 parameter cmd")
	num, _ := strconv.Atoi(data)

	// 转换为单字节数据
	data1 := []byte{byte(num)}
	return composeCmd(cmdID, 0, []byte(data1))
}

func setWiredIpCmdTMAX(cmdID uint16, data string) []byte {
	l.Log.Debug("compose set sw15 parameter cmd")
	//将16进制的字符串转换为字节数组

	dataByte, _ := hex.DecodeString(data)
	return composeCmd(cmdID, 0, dataByte)
}

func getSetGaduationValueCmdTMAX(data string) []byte {
	l.Log.Debug("compose set gaduation value cmd")
	num, _ := strconv.Atoi(data)
	// 转换为单字节数据
	data1 := []byte{byte(num)}
	return composeCmd(0xE314, 0, []byte(data1))
}

func getModifyEepromCmdTMAX(addr uint32, data []byte) []byte {
	l.Log.Debug("compose modify eeprom info cmd")
	var dataLen = 19 + len(data)
	return wrDataCmdTMAX(addr, data, uint16(dataLen))
}

func getModifyVarValueCmdTMAX(data []byte) []byte {
	l.Log.Debug("compose modify var value cmd")
	return composeModifyVarCmd(0xf501, data)
}

func enFactoryModeCmdTMAX(data []byte) []byte {
	l.Log.Debug("compose en factory mode cmd")
	return composeFactoryCmd(0xf601, data)
}

func getSetScaleTimeCmdTMAX(data string) []byte {
	l.Log.Debug("compose set scale Time cmd")
	return composeCmd(0xf401, 0, []byte(data))
}

func getSetLimitToScaleCmdTMAX(data []byte) []byte {
	l.Log.Debug("compose set limit to cmd")
	return composeCmd(CMDID_SET_LIMIT_TMAX, 0, data)
}
func parseReadAddrTMAX(inData string) (addr int64) { // inData is hex ascii
	addr, err := strconv.ParseInt(inData[0:8], 16, 64)
	if err != nil {
		fmt.Println("Invalid offset number")
		return -1
	}

	return
}

func parseWrDataTMAX(inData string) (addr int64, data []byte) { // inData is hex ascii
	addr, err := strconv.ParseInt(inData[0:8], 16, 64)
	if err != nil {
		fmt.Println("Invalid offset number")
		return -1, nil
	}

	// Extract byte array
	data, err = hex.DecodeString(inData[9:])
	if err != nil {
		fmt.Println("Invalid hex string")
		return -1, nil
	}

	return
}

func parseReadFlashTMAX(inData string) (addr int64, data []byte) { // inData is hex ascii
	addr, err := strconv.ParseInt(inData[0:8], 16, 64)
	if err != nil {
		fmt.Println("Invalid offset number")
		return -1, nil
	}

	// Extract byte array
	data, err = hex.DecodeString(inData[9:])
	if err != nil {
		fmt.Println("Invalid hex string")
		return -1, nil
	}

	return
}
