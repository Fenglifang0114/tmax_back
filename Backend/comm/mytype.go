package comm

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ScaleCat int

const (
	SCALE_C51 ScaleCat = iota
	SCALE_T2200
	SCALE_T2200_PASSTH
	SCALE_JWP
	SCALE_JWP_PASSTH
	SCALE_TMAX
	SCALE_TMAX_PASSTH
)

type CmdType int

const (
	CMD_CHECK_FAC_MODE CmdType = iota
	CMD_DIS_FAC_MODE

	CMD_ZERO
	CMD_TARE
	CMD_READ_WEIGHT

	CMD_ZERO_UNSTABLE
	CMD_TARE_UNSTABLE

	CMD_EN_CONTINUE_MODE
	CMD_DIS_CONTINUE_MODE

	CMD_EN_USR_CONT_MODE

	CMD_EN_PASSTH
	CMD_DIS_PASSTH

	CMD_GET_BUILD_INFO
	CMD_GET_SCALE_INFO
	CMD_GET_FACTORY_INFO

	CMD_REBOOT

	CMD_WIFI_DATA_PASSTH
	CMD_WIFI_GET_AP_LIST
	CMD_WIFI_EN_DHCP
	CMD_WIFI_EN_DHCP_32
	CMD_WIFI_SET_STATIC_IP
	CMD_WIFI_SET_STATIC_IP_32
	CMD_WIFI_GET_AP_INFO
	CMD_WIFI_GET_AP_INFO_32
	CMD_WIFI_GET_IP_INFO
	CMD_WIFI_GET_IP_INFO_32
	CMD_WIFI_GET_IP_MODE
	CMD_WIFI_GET_IP_MODE_32
	CMD_WIFI_CONN_AP
	CMD_WIFI_CONN_AP32
	CMD_WIFI_CONN_AP_ONE_KEY
	CMD_WIFI_DISCONN_AP
	CMD_CHANGE_WIFI_MODE
	CMD_WIFI_AT_VERSION
	CMD_WIFI_AT_MODE

	CMD_WIFI_CLOSE_SERVER_CMD
	CMD_WIFI_DIS_BT_CMD
	CMD_WIFI_EN_AUTO_CONN_CMD
	CMD_WIFI_SET_WIFI_STATION_MODE_CMD
	CMD_WIFI_SET_MULTI_CONN_CMD
	CMD_WIFI_DIS_RECONN_CMD
	CMD_WIFI_DIS_IP_PORT_INFO_CMD
	CMD_WIFI_SET_SINGLE_CONN_CMD
	CMD_WIFI_SET_CONN_PORT_CMD
	CMD_WIFI_SET_TIME_OUT_CMD
	CMD_WIFI_SET_PASSTH_MODE_CMD
	CMD_WIFI_SET_SCAN_AP_PARAM_CMD

	CMD_BT_DATA_PASSTH
	CMD_MODIFY_BT_NAME
	CMD_READ_EEPROM_8
	CMD_READ_EEPROM_256
	CMD_READ_EEPROM_512
	CMD_WRITE_EEPROM
	CMD_ERASE_FLASH
	CMD_ERASE_FLASH_512
	CMD_READ_FLASH
	CMD_WRITE_FLASH_8
	CMD_WRITE_FLASH_256
	CMD_WRITE_FLASH_512
	CMD_DEL_PLU
	CMD_INSERT_PLU_ADDR
	CMD_GET_PLU_HEAD
	CMD_ERASE_INSERT_PLU
	CMD_GET_WEIGHT_ERR
	CMD_GET_SCALE_TIME
	CMD_SET_SCALE_TIME
	CMD_MODIFY_VAR_VALUE
	CMD_EN_FACTORY_MODE
	CMD_GET_RANDOM_DATA
	CMD_GET_BASIC_DATA
	CMD_SET_LIMIT_TO_SCALE
	CMD_OPEN_BILL_SEND
	CMD_CAl_WGT
	CMD_SEND_CAL_HEART_BEAT
	CMD_SET_DECIMAl_VALUE
	CMD_GET_DECIMAL_VALUE
	CMD_SET_MAX_RANGE1
	CMD_SET_MAX_RANGE2
	CMD_GET_MAX_RANGE1
	CMD_GET_MAX_RANGE2
	CMD_SET_GADUATION1_VALUE
	CMD_GET_GADUATION1_VALUE
	CMD_GET_GADUATION2_VALUE
	CMD_SET_GADUATION2_VALUE
	CMD_SET_WEIGHT_UNIT
	CMD_GET_WEIGHT_UNIT
	CMD_GET_GRAV_ACC
	CMD_SET_GRAV_ACC
	CMD_SET_MANUAL_ZERO
	CMD_GET_MANUAL_ZERO
	CMD_SET_INITIAL_ZERO
	CMD_GET_INITIAL_ZERO
	CMD_GET_ZERO_TRACKING
	CMD_SET_ZERO_TRACKING
	CMD_SET_FORCE_UNTARE
	CMD_GET_WIRED_IP
	CMD_GET_WIRED_DHCP
	CMD_SET_WIRED_IP
	CMD_SET_WIRED_DHCP
	CMD_GET_SEAL_STATUS
	CMD_SET_SOFT_SEAL
	CMD_REMOVE_SOFT_SEAL
	CMD_REMOVE_SOFT_SEAL_ONCE
	CMD_GET_MODEL
	CMD_EN_CODE
	CMD_DIS_CODE
	CMD_ASK_ROM_VERSION
	CMD_SET_SERIAL_PORT
	CMD_GET_SERIAL_PORT
	CMD_GET_GROSS_WEIGHT
	CMD_GET_NET_WEIGHT
	CMD_GET_TARE_WEIGHT
	CMD_GET_PRE_TARE_WEIGHT
	CMD_SET_PRE_TARE_S15
)

type DataType int

const (
	DATA_TYPE_INT = iota
	DATA_TYPE_BYTE_ARR
	DATA_TYPE_STR
)

type CmdData struct {
	Type DataType
	Data interface{}
}

var (
	LICENSE_FILE   = "tmaxlic.txt"
	SRV_DATA_PATH  = "srvdata"
	COMM_DATA_BASE = "database"
	COMM_SERVICE   = "service"
)

var LogFile *os.File

const (
	AT_VERSION     = "ESP32C3"
	IS_ESP32       = true
	PLU_BACK_PATH  = "plufiles"
	AT_VERSION8266 = "8266"
)

type CmdComposer struct {
	ScaleCat   ScaleCat
	ComposeCmd func(composer *CmdComposer, cmd CmdType, cmdData CmdData) ([]byte, int, error)
}

type Packet struct {
	PayloadLen uint16
	CmdID      uint8
	CmdSubId   uint8
	SeqNum     uint8
	Payload    []byte
}

type RespMsgType string

// 处理scale回应
const (
	WEIGHT_DATA                   RespMsgType = "weight_data"
	ZERO_UNSTABLE_CMD_RESP        RespMsgType = "resp_zero_unstable_cmd"
	TARE_UNSTABLE_CMD_RESP        RespMsgType = "resp_tare_unstable_cmd"
	ZERO_CMD_RESP                 RespMsgType = "resp_zero_cmd"
	TARE_CMD_RESP                 RespMsgType = "resp_tare_cmd"
	WEIGHT_DATA_RESP              RespMsgType = "resp_weight_data"
	REG_WEIGHT_RESP               RespMsgType = "resp_reg_weight"
	UNREG_WEIGHT_RESP             RespMsgType = "resp_unreg_weight"
	GET_RECS_RESP                 RespMsgType = "resp_get_recs"
	ADD_REC_RESP                  RespMsgType = "resp_add_rec"
	DEL_REC_RESP                  RespMsgType = "resp_del_rec"
	EN_FAC_MODE_RESP              RespMsgType = "resp_en_fac_mode"
	DIS_FAC_MODE_RESP             RespMsgType = "resp_dis_fac_mode"
	EN_PASSTH_MODE_RESP           RespMsgType = "resp_en_passth_mode"
	DIS_PASSTH_MODE_RESP          RespMsgType = "resp_dis_passth_mode"
	ERASE_FLASH_RESP              RespMsgType = "resp_erase_flash"
	WRITE_DATA_FLASH_RESP         RespMsgType = "resp_write_data_flash"
	DOWN_PRN_FMT_RESP             RespMsgType = "resp_down_prn_fmt"
	ERR_SERIAL_RESP               RespMsgType = "resp_err_serial"
	GET_AP_LIST_RESP              RespMsgType = "resp_get_ap_list"
	RESCAN_AP_LIST_RESP           RespMsgType = "resp_rescan_ap_list"
	CONNECT_AP_RESP               RespMsgType = "resp_connect_ap"
	CONNECT_AP_ONE_KEY_RESP       RespMsgType = "resp_connect_ap_one_key"
	SET_WIFI_DYNAMIC_IP_RESP      RespMsgType = "resp_set_wifi_dynamic_ip"
	SET_WIFI_STATIC_IP_RESP       RespMsgType = "resp_set_wifi_static_ip"
	GET_AT_VERSION_RESP           RespMsgType = "resp_get_at_version"
	GET_AT_MODE_RESP              RespMsgType = "resp_get_at_mode"
	GET_WIFI_AP_INFO_RESP         RespMsgType = "resp_get_wifi_ap_info"
	GET_IP_INFO_RESP              RespMsgType = "resp_get_ip_info"
	GET_IP_MODE_RESP              RespMsgType = "resp_get_ip_mode"
	MODIFY_BT_NAME_RESP           RespMsgType = "resp_modify_bt_name"
	NO_RESP                       RespMsgType = "resp_no_response"
	BT_PASSTH_DATA_RESP           RespMsgType = "resp_bt_passth_data"
	WIFI_PASSTH_DATA_RESP         RespMsgType = "resp_wifi_passth_data"
	PRT_PASSTH_DATA_RESP          RespMsgType = "resp_prt_passth_data"
	SEND_DATA_TO_BT_RESP          RespMsgType = "resp_bt_passth_data"
	SEND_DATA_TO_WIFI_RESP        RespMsgType = "resp_send_data_to_wifi"
	UPDATE_FIRMWARE_RESP          RespMsgType = "resp_update_firmware"
	UPDATE_FIRMWARE_PROGRESS      RespMsgType = "resp_update_firmware_progress"
	DOWN_FIRMWARE_WIFI_RESP       RespMsgType = "resp_update_firmware_wifi"
	CHECK_SERIAL_PORT_RESP        RespMsgType = "resp_check_serial_port"
	GET_BUILD_INFO_RESP           RespMsgType = "resp_get_build_info"
	GET_SCALE_TIME_RESP           RespMsgType = "resp_get_scale_time"      //20240112@FLF
	SET_SCALE_TIME_RESP           RespMsgType = "resp_set_scale_time"      //20240112@FLF
	GET_ONE_EEPROM_INFO_RESP      RespMsgType = "resp_get_one_eeprom_info" //20240125@FLF
	GET_ALL_EEPROM_INFO_RESP      RespMsgType = "resp_get_all_eeprom_info" //20240125@FLF
	SET_OUTPUT_FMT_RESP           RespMsgType = "resp_set_output_fmt"
	OPEN_SCALE_PASSTHROUGH_RESP   RespMsgType = "resp_open_scale_passthrough" //20231023@FLF 连续发送的透传
	CLOSE_SCALE_PASSTHROUGH_RESP  RespMsgType = "resp_close_scale_passthrough"
	SCALE_PASSTH_DATA             RespMsgType = "scale_passth_data"
	CHANGE_SCALE_PASSTH_MODE_RESP RespMsgType = "resp_change_scale_passth_mode"
	GET_SCALE_INFO_RESP           RespMsgType = "resp_get_scale_info"
	GET_FACTORY_INFO_RESP         RespMsgType = "resp_get_factory_info"
	GET_WEIGHT_ERR_RESP           RespMsgType = "resp_get_weight_err"
	DOWN_PLU_RESP                 RespMsgType = "resp_down_plu"
	DEL_PLU_RESP                  RespMsgType = "resp_del_plu"
	INSERT_PLU_RESP               RespMsgType = "resp_insert_plu"
	GET_UI_CONF_RESP              RespMsgType = "resp_get_ui_conf"
	UPDATE_UI_CONF_RESP           RespMsgType = "resp_update_ui_conf"
	CHANGE_WIFI_MODE_RESP         RespMsgType = "resp_change_wifi_mode"
	INSERT_PLU_ADDR_RESP          RespMsgType = "resp_insert_plu_addr"
	READ_FLASH_DATA_RESP          RespMsgType = "resp_read_flash_data"
	ERASE_INSERT_PLU_RESP         RespMsgType = "resp_erase_insert_plu"
	REBOOT_RESP                   RespMsgType = "resp_reboot"
	MODIFY_EEPROM_INFO_RESP       RespMsgType = "resp_modify_eeprom_info"
	DOWN_EEPROM_INFO_RESP         RespMsgType = "resp_down_eeprom_info"
	DOWN_FACTORY_INFO_FC_RESP     RespMsgType = "resp_down_factory_info"
	DOWN_FACTORY_INFO_RESP        RespMsgType = "resp_down_factory_info_tmax"
	MODIFY_VAR_RESP               RespMsgType = "resp_modify_var_value"
	SET_SERVER_IP_RESP            RespMsgType = "resp_set_server_ip"
	GET_RANDOM_DATA_RESP          RespMsgType = "resp_get_random_data"
	EN_FACTORY_MODE_RESP          RespMsgType = "resp_en_factory_mode"
	DOWN_DEFAULT_PRN_FMT_RESP     RespMsgType = "resp_down_def_prn_fmt"
	BACKUP_DEF_SETTING_RESP       RespMsgType = "resp_backup_def_setting"
	GET_EEPROM_TO_BIN_RESP        RespMsgType = "resp_get_eeprom_to_bin"
	GET_EEPROM_BIN_DATA_RESP      RespMsgType = "resp_get_eeprom_bin_data"
	SET_EEPROM_FROM_BIN_RESP      RespMsgType = "resp_set_eeprom_from_bin"
	GET_BASIC_DATA_RESP           RespMsgType = "resp_get_basic_data"
	SET_LIMIT_TO_SCALE_RESP       RespMsgType = "resp_set_limit_to_scale"
	SWITCH_LIMIT_RESP             RespMsgType = "resp_switch_limit_from_scale"
	REV_DETAIl_HEAD_RESP          RespMsgType = "resp_rev_detail_head"
	REV_DETAIl_MID_RESP           RespMsgType = "resp_rev_detail_mid"
	REV_DETAIl_TAIL_RESP          RespMsgType = "resp_rev_detail_tail"
	OPEN_BILL_SEND_RESP           RespMsgType = "resp_open_bill_send"
	CLOSE_SERIAL_PORT_RESP        RespMsgType = "resp_close_serial_port"
	OPEN_SERIAL_PORT_RESP         RespMsgType = "resp_open_serial_port"
	ANSWER_ALIVE_RESP             RespMsgType = "resp_answer_alive"
	EXPORT_RECS_RESP              RespMsgType = "resp_export_recs"
	SET_CAL_WGT_RESP              RespMsgType = "resp_cal_weight"
	SET_DECIMAL_VALUE_RESP        RespMsgType = "resp_set_decimal_value"
	GET_DECIMAL_VALUE_RESP        RespMsgType = "resp_get_decimal_value"
	SET_MAX_RANGE1_RESP           RespMsgType = "resp_set_max_range1"
	GET_MAX_RANGE1_RESP           RespMsgType = "resp_get_max_range1"
	SET_MAX_RANGE2_RESP           RespMsgType = "resp_set_max_range2"
	GET_MAX_RANGE2_RESP           RespMsgType = "resp_get_max_range2"

	SET_GADUATION1_VALUE_RESP RespMsgType = "resp_set_gaduation1_value"
	GET_GADUATION1_VALUE_RESP RespMsgType = "resp_get_gaduation1_value"
	SET_GADUATION2_VALUE_RESP RespMsgType = "resp_set_gaduation2_value"
	GET_GADUATION2_VALUE_RESP RespMsgType = "resp_get_gaduation2_value"

	SET_WEIGHT_UNIT_RESP RespMsgType = "resp_set_weight_unit"
	GET_WEIGHT_UNIT_RESP RespMsgType = "resp_get_weight_unit"

	SET_INITIAL_ZERO_RESP  RespMsgType = "resp_set_initial_zero"
	SET_MANUAL_ZERO_RESP   RespMsgType = "resp_set_manual_zero"
	SET_ZERO_TRACKING_RESP RespMsgType = "resp_set_zero_tracking"
	SET_GRAV_ACC_RESP      RespMsgType = "resp_set_grav_acc"

	GET_INITIAL_ZERO_RESP  RespMsgType = "resp_get_initial_zero"
	GET_MANUAL_ZERO_RESP   RespMsgType = "resp_get_manual_zero"
	GET_ZERO_TRACKING_RESP RespMsgType = "resp_get_zero_tracking"
	GET_GRAV_ACC_RESP      RespMsgType = "resp_get_grav_acc"
	SET_FORCE_UNTARE_RESP  RespMsgType = "resp_force_untare"

	GET_MODEL_RESP       RespMsgType = "resp_get_model"       //获取秤的内部型号 T-MAX
	EN_CODE_RESP         RespMsgType = "resp_en_code"         //开启内码
	DIS_CODE_RESP        RespMsgType = "resp_dis_code"        //关闭内码
	ASK_ROM_VERSION_RESP RespMsgType = "resp_ask_rom_version" //询问Tmax rom 版本号
	CONT_CODE_RESP       RespMsgType = "resp_cont_code"       //连续发送内码

	GET_SEAL_STATUS_RESP  RespMsgType = "resp_get_seal_status"
	SOFT_SEAL_RESP        RespMsgType = "resp_soft_seal"
	REMOVE_SOFT_SEAL_RESP RespMsgType = "resp_remove_soft_seal"

	GET_WIRED_IP_RESP          RespMsgType = "resp_get_wired_ip"
	SET_WIRED_IP_RESP          RespMsgType = "resp_set_wired_ip"
	SET_WIRED_DHCP_RESP        RespMsgType = "resp_set_wired_dhcp"
	GET_WIRED_DHCP_RESP        RespMsgType = "resp_get_wired_dhcp"
	REMOVE_SOFT_SEAL_ONCE_RESP RespMsgType = "resp_remove_soft_seal_once"
	SET_SERIAL_PORT_RESP       RespMsgType = "resp_set_serial_port"
	GET_SERIAL_PORT_RESP       RespMsgType = "resp_get_serial_port"
	GET_GROSS_WEIGHT_RESP      RespMsgType = "resp_get_gross_weight"
	GET_NET_WEIGHT_RESP        RespMsgType = "resp_get_net_weight"
	GET_TARE_WEIGHT_RESP       RespMsgType = "resp_get_tare_weight"
	GET_PRE_TARE_WEIGHT_RESP   RespMsgType = "resp_get_pre_tare_weight"
	SET_PRE_TARE_S15_RESP      RespMsgType = "resp_set_pre_tare_s15"

	CLOSE_SERVER_CMD_RESP          RespMsgType = "resp_close_server_cmd"
	DIS_BT_CMD_RESP                RespMsgType = "resp_dis_bt_cmd"
	EN_AUTO_CONN_CMD_RESP          RespMsgType = "resp_en_auto_conn_cmd"
	SET_WIFI_STATION_MODE_CMD_RESP RespMsgType = "resp_set_wifi_station_mode_cmd"
	SET_MULTI_CONN_CMD_RESP        RespMsgType = "resp_set_multi_conn_cmd"
	DIS_RECONN_CMD_RESP            RespMsgType = "resp_dis_auto_reconn_cmd"
	DIS_IP_PORT_INFO_CMD_RESP      RespMsgType = "resp_dis_ip_port_info_cmd"
	SET_SINGLE_CONN_CMD_RESP       RespMsgType = "resp_set_single_conn_cmd"
	SET_TCP_SERVER_CMD_RESP        RespMsgType = "resp_set_tcp_server_cmd"
	SET_TIME_OUT_CMD_RESP          RespMsgType = "resp_set_time_out_cmd"
	SET_PASSTH_MODE_CMD_RESP       RespMsgType = "resp_set_passth_mode_cmd"
	SET_SCAN_AP_PARAM_CMD_RESP     RespMsgType = "resp_set_scan_ap_param_cmd"
	INIT_WIFI_RESP                 RespMsgType = "resp_init_wifi"
	SET_TCP_SERVER_RESP            RespMsgType = "resp_set_tcp_server"

	UNKNOWN_DATA RespMsgType = "unknown_data"
)

func GetServicePath() string {
	myPath := GetSrvDataPath()
	return filepath.Join(myPath, COMM_SERVICE)
}

func GetSrvDataPath() string {
	myPath := GetExePath()
	p := filepath.Join(myPath, SRV_DATA_PATH)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if _, err := os.Stat(SRV_DATA_PATH); err == nil {
			return SRV_DATA_PATH
		}
		if _, err := os.Stat(filepath.Join("..", SRV_DATA_PATH)); err == nil {
			return filepath.Join("..", SRV_DATA_PATH)
		}
	}
	return p
}

func GetExePath() string {
	myPath, _ := getCurrentPath()
	return myPath
}

func getParentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	parentPath := path[:i]
	j := strings.LastIndex(parentPath, "/")
	if j < 0 {
		j = strings.LastIndex(parentPath, "\\")
	}
	if j < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return parentPath[:j+1], nil
}

func GetCommDataBasePath() string {

	myParentPath, _ := getParentPath()
	return filepath.Join(myParentPath, COMM_DATA_BASE)
}

func getCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}
