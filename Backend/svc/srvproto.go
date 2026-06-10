package svc

import (
	"time"

	"go.bug.st/serial"

	"tmaxsrv/comm"
)

// ********** Request for scale manager **********
type Request struct {
	Req     ReqType
	ReqData string // should be json encoded string of one of the following structures, i.e. ReqAddScale, ReqDelScale, ReqModifyScale
}

type ReqType string

const (
	REQ_GET_PORT_LIST    ReqType = "get_port_list"    // without parameter
	REQ_GET_BT_LIST      ReqType = "get_bt_list"      // without parameter
	REQ_GET_SCALE_LIST   ReqType = "get_scale_list"   // without parameter
	REQ_GET_PRODUCT_LIST ReqType = "get_product_list" // without parameter

	REQ_DOWN_ALL_PLU    ReqType = "down_all_plu"    // without parameter
	REQ_SET_PLU         ReqType = "set_plu_fields"  // with ReqPluSetting parameter
	REQ_GET_PLU_SETTING ReqType = "get_plu_setting" // without parameter

	REQ_GET_PLU_BY_PAGE   ReqType = "get_plu_by_page"   // without parameter
	REQ_GET_USER_LIST     ReqType = "get_user_list"     // without parameter
	REQ_GET_WIFI_PWD_LIST ReqType = "get_wifi_pwd_list" // without parameter

	REQ_ADD_SCALE         ReqType = "add_scale"         // with ReqAddScale parameter
	REQ_DEL_SCALE         ReqType = "del_scale"         // with ReqDelScale parameter
	REQ_MODIFY_SCALE      ReqType = "modify_scale"      // with ReqModifyScale parameter
	REQ_MODIFY_SCALE_NAME ReqType = "modify_scale_name" // with ReqModifyScale parameter

	REQ_EXPORT_PRODUCT  ReqType = "export_product"  // without parameter
	REQ_ADD_PRODUCT     ReqType = "add_product"     // with ReqAddProduct parameter
	REQ_ADD_ONE_PRODUCT ReqType = "add_one_product" // with ReqAddOneProduct parameter
	REQ_CLEAR_PRODUCT   ReqType = "clear_product"   // without parameter
	check_plu_exist
	REQ_CHECK_PLU_EXIST ReqType = "check_plu_exist" // with ReqCheckPluExist parameter

	REQ_DEL_PRODUCT          ReqType = "del_product"          // with ReqDelScale parameter
	REQ_DEL_ALL_PRODUCT      ReqType = "del_all_product"      // with ReqDelScale parameter
	REQ_GET_LAST_PRODUCT_REC ReqType = "get_last_product_rec" // without parameter
	REQ_UPDATE_ENABLED_PLU   ReqType = "update_enabled_plu"   // with ReqUpdateEnabledPlu parameter
	REQ_MODIFY_PRODUCT       ReqType = "modify_product"       // with ReqModifyScale parameter

	REQ_ADD_USER    ReqType = "add_user"    // with ReqAddScale parameter
	REQ_DEL_USER    ReqType = "del_user"    // with ReqDelScale parameter
	REQ_MODIFY_USER ReqType = "modify_user" // with ReqModifyScale parameter

	REQ_QUIT_APPLICATION  ReqType = "quit_application"  // without parameter
	REQ_GET_UI_CONF       ReqType = "get_ui_conf"       // without parameter
	REQ_UPDATE_UI_CONF    ReqType = "update_ui_conf"    // without parameter
	REQ_GET_LICENSE       ReqType = "get_license"       // without parameter
	REQ_CHECK_LICENSE_KEY ReqType = "check_license_key" // with parameter
	REQ_UPDATE_LICENSE    ReqType = "update_license"    // with parameter

	REQ_GET_DETAIL_LIST ReqType = "get_detail_list" // without parameter 20240903

	REQ_WIFI_PWD ReqType = "add_wifi_pwd" // with ReqAddWifiPwd parameter
	//下面添加给直接转给小服务的
	REQ_SEND_TO_SRV1          ReqType = "send_to_srv1" // ui给服务1发送的数据
	REQ_SEND_TO_UI            ReqType = "send_to_ui"
	REQ_GET_SCALE_SRV_LIST    ReqType = "get_scale_srv_list"
	REQ_SET_SCALE_SRV_VAL     ReqType = "set_scale_srv_val"
	REQ_SET_DO_SERVICE_ACTION ReqType = "do_service_action"

	// Modbus 服务网关相关
	REQ_GET_MODBUS_SERVICES ReqType = "get_modbus_services"
	REQ_ADD_MODBUS_SERVICE  ReqType = "add_modbus_service"
	REQ_EDIT_MODBUS_SERVICE ReqType = "edit_modbus_service"
	REQ_DEL_MODBUS_SERVICE  ReqType = "del_modbus_service"

	///////////
	REQ_ADD_RAW_TYPE             ReqType = "add_raw_type"             //添加原料类型
	REQ_DEL_UNUSED_FMA_TYPE      ReqType = "del_unused_fma_type"      //删除未使用的配方类型
	REQ_DEL_UNUSED_RAW_TYPE      ReqType = "del_unused_raw_type"      //删除未使用的原料类型
	REQ_DEL_RAW_TYPE             ReqType = "del_raw_type"             //删除原料类型
	REQ_EDIT_RAW_TYPE            ReqType = "edit_raw_type"            //修改原料类型
	REQ_GET_RAW_TYPE_LIST        ReqType = "get_raw_type_list"        //获取原料类型列表
	REQ_ADD_FORMULA_TYPE         ReqType = "add_formula_type"         //添加配方类型
	REQ_DEL_FORMULA_TYPE         ReqType = "del_formula_type"         //删除配方类型
	REQ_EDIT_FORMULA_TYPE        ReqType = "edit_formula_type"        //修改配方类型
	REQ_GET_FORMULA_TYPE_LIST    ReqType = "get_formula_type_list"    //获取配方类型列表
	REQ_IMPORT_RAW_LIST          ReqType = "import_raw_list"          //导入原料列表
	REQ_IMPORT_FMA_LIST          ReqType = "import_fma_list"          //导入配方列表
	REQ_ADD_RAW_DATA             ReqType = "add_raw_data"             //添加原始数据
	REQ_DEL_RAW_DATA             ReqType = "del_raw_data"             //删除原始数据
	REQ_EDIT_RAW_DATA            ReqType = "edit_raw_data"            //修改原始数据
	REQ_GET_RAW_DATA_LIST        ReqType = "get_raw_data_list"        //获取原始数据列表
	REQ_DELETE_RAW_DATA          ReqType = "delete_raw_data"          //删除原料数据
	REQ_ADD_FORMULA_DATA         ReqType = "add_formula_data"         //新增配方信息
	REQ_EDIT_FORMULA_DATA        ReqType = "edit_formula_data"        //修改配方信息
	REQ_GET_FORMULA_LIST         ReqType = "get_formula_list"         //获取配方信息列表
	REQ_GET_FORMULA_DATA         ReqType = "get_fma_data"             //获取配方数据
	REQ_GET_RAW_DATA             ReqType = "get_raw_data"             //获取原料数据
	REQ_DELETE_FORMULA_DATA      ReqType = "delete_formula_data"      //删除配方信息
	REQ_DELETE_MANY_FORMULA      ReqType = "del_many_fma"             //删除所有配方
	REQ_DEL_MANY_RAW_DATA        ReqType = "del_many_raw"             //删除所有原料数据
	REQ_DEL_MANY_DRAFT_FMA       ReqType = "del_many_draft_fma"       //删除所有暂存配方
	REQ_ADD_FORMULA_REC          ReqType = "add_formula_rec"          //新增配方称重记录
	REQ_GET_FORMULA_REC_LIST     ReqType = "get_formula_rec_list"     //获取配方称重记录列表
	REQ_GET_ONE_FORMULA_REC_LIST ReqType = "get_fma_rec_by_id"        //获取配方称重记录列表
	REQ_GET_FMA_REC_BY_ORDER     ReqType = "get_fma_rec_by_order"     //根据订单号获取配方称重记录
	REQ_ADD_FLOW_RATE            ReqType = "add_flow_rate"            //新增流速
	REQ_GET_FLOW_RATE_LIST       ReqType = "get_flow_rate_list"       //获取流速列表
	REQ_GET_RAW_OUTPUT_BY_FMA_ID ReqType = "get_raw_output_by_fma_id" //根据配方ID获取原料输出端口

	//获取称重记录，不按秤来，总体的记录
	REQ_GET_ALL_WGT_REC_LIST ReqType = "get_all_wgt_rec_list" //获取称重记录列表  要分类型(重量收集，检重，加法，减法)
	REQ_GET_SEARCH_REC_LIST  ReqType = "get_search_rec_list"  //获取搜索记录列表  (搜索字段)
	REQ_ADD_WGT_REC          ReqType = "add_wgt_rec"          //添加称重记录(汇总的称重记录)
	REQ_DEL_WGT_REC          ReqType = "del_wgt_rec"          //删除称重记录  根据称重类型删除所有的数据
	REQ_DEL_WGT_REC_BY_ID    ReqType = "del_wgt_rec_by_id"    //根据称重记录recID 删除称重记录
	REQ_EXPORT_ALL_RECS      ReqType = "export_all_recs"      //导出所有记录
	REQ_KILL_BOOT_COMMANDER  ReqType = "kill_boot_commander"  //杀掉boot_commander进程
	REQ_GET_AUTO_NEXT        ReqType = "get_auto_next"        //获取自动下一步设置
	REQ_UPDATE_AUTO_NEXT     ReqType = "update_auto_next"     //更新自动下一步设置
	REQ_GET_OUTPUT_PORT      ReqType = "get_output_port"      //获取输出端口状态
	REQ_UPDATE_OUTPUT_PORT   ReqType = "update_output_port"   //更新输出端口状态

	REQ_GET_INPUT_PORT    ReqType = "get_input_port"    //获取输入端口状态
	REQ_UPDATE_INPUT_PORT ReqType = "update_input_port" //更新输入端口状态

	REQ_GET_UNSTABLE_ZERO_TARE    ReqType = "get_unstable_zero_tare"    //获取不稳定归零扣重开关
	REQ_UPDATE_UNSTABLE_ZERO_TARE ReqType = "update_unstable_zero_tare" //更新不稳定归零扣重开关

	REQ_GET_FORMULA_BY_BARCODE   ReqType = "get_formula_by_barcode"   //根据条码获取配方信息
	REQ_CHECK_FMA_ID_AND_BARCODE ReqType = "check_fma_id_and_barcode" //检查配方ID和条码是否匹配

	REQ_GET_REPORT_PRINT_SETTING    ReqType = "get_report_print_setting"    //获取报表打印设置
	REQ_UPDATE_REPORT_PRINT_SETTING ReqType = "update_report_print_setting" //更新报表打印设置

	REQ_UPLOAD_SERVER_GET  ReqType = "upload_server_get"  //获取上传服务器设置
	REQ_UPLOAD_SERVER_EDIT ReqType = "upload_server_edit" //编辑上传服务器设置

	REQ_GET_ALL_SEAL_LOG     ReqType = "get_all_seal_log"     //获取所有铅封日志记录
	REQ_UNSEAL_BY_MASTER_KEY ReqType = "unseal_by_master_key" //万能钥匙解封

	//暂存配方称重记录
	REQ_GET_DRAFT_FMA_WGT_REC_LIST ReqType = "get_draft_fma_wgt_rec_list" //获取草稿配方称重记录列表
	REQ_UPDATE_DRAFT_FMA_WGT_REC   ReqType = "update_draft_fma_wgt_rec"   //更新草稿配方称重记录
	REQ_DELETE_DRAFT_FMA_WGT_REC   ReqType = "delete_draft_fma_wgt_rec"   //删除草稿配方称重记录
	REQ_CREATE_DRAFT_FMA_WGT_REC   ReqType = "create_draft_fma_wgt_rec"   //创建草稿配方称重记录

	REQ_ADD_SYS_USER     ReqType = "add_sys_user"     //新增系统用户
	REQ_DELETE_SYS_USER  ReqType = "delete_sys_user"  //删除系统用户
	REQ_UPDATE_SYS_USER  ReqType = "update_sys_user"  //更新系统用户
	REQ_DISABLE_SYS_USER ReqType = "disable_sys_user" //禁用系统用户
	REQ_CHANGE_PASSWORD  ReqType = "change_password"  //修改密码
	REQ_LOGIN            ReqType = "login"            //登录、
	REQ_LOGOUT           ReqType = "logout"           //登出
	REQ_GET_ALL_USERS    ReqType = "get_all_users"    //获取所有用户列表
	REQ_GET_USER_DETAIL  ReqType = "get_user_detail"  //获取用户详情

	REQ_ADD_SYS_LOG       ReqType = "add_sys_log"       //新增系统日志记录
	REQ_ADD_CAL_LOG       ReqType = "add_cal_log"       //新增校准日志记录
	REQ_ADD_SCALE_LOG     ReqType = "add_scale_log"     //新增称重日志记录
	REQ_DEL_SYS_LOG       ReqType = "del_sys_log"       //删除系统日志记录
	REQ_DEL_CAL_LOG       ReqType = "del_cal_log"       //删除校准日志记录
	REQ_DEL_SCALE_LOG     ReqType = "del_scale_log"     //删除称重日志记录
	REQ_DEL_ALL_SYS_LOG   ReqType = "del_all_sys_log"   //删除所有系统日志记录
	REQ_DEL_ALL_CAL_LOG   ReqType = "del_all_cal_log"   //删除所有校准日志记录
	REQ_DEL_ALL_SCALE_LOG ReqType = "del_all_scale_log" //删除所有称重日志记录
	REQ_EXPORT_SYS_LOG    ReqType = "export_sys_log"    //导出系统日志记录
	REQ_EXPORT_CAL_LOG    ReqType = "export_cal_log"    //导出校准日志记录
	REQ_EXPORT_SCALE_LOG  ReqType = "export_scale_log"  //导出称重日志记录
	REQ_GET_SYS_LOG       ReqType = "get_sys_log"       //获取系统日志记录
	REQ_GET_CAL_LOG       ReqType = "get_cal_log"       //获取校准日志记录
	REQ_GET_SCALE_LOG     ReqType = "get_scale_log"     //获取称重日志记录

	REQ_OPEN_OUTPUT_PORT ReqType = "open_output_port" // with parameter of PortInfo	//写modbus线圈
	REQ_READ_OUTPUT_PORT ReqType = "read_output_port" // with parameter of PortInfo	//读modbus线圈

)

type ReqAddScale struct {
	ScaleModel string
	ScaleSn    string
	// MediaType  MediaType
	MediaConf MediaConf // will be ComInfo/NetInfo/BtInfo according to the media type
	ModbusId  int       // 关联的 Modbus 从机站号
}

type ReqDelScale struct {
	ScaleId int64
}

type ReqModifyScale struct {
	ScaleId int64
	// MediaType MediaType
	MediaConf  MediaConf
	ScaleModel string
	ModbusId   int // 关联的 Modbus 从机站号
}

type ReqModifyScaleName struct {
	ScaleId   int64
	ScaleName string
}

type ReqModifyScaleSn struct {
	ScaleId    int64
	Sn         string
	ScaleModel string
}

type ReqAddPlu struct {
	PluList []AddProduct
	Total   int
	Index   int
}

type ReqPluSetting struct {
	Plu []string `json:"plu"`
}

type ReqExportProduct struct {
	Path        string
	Translation map[string]string
	SearchPlu   ProductQuery
}

type ReqGetPluByPage struct {
	Page      int
	PageSize  int
	FieldName string
	Direction string
	Search    ProductQuery
}

type ProductQuery struct {
	Plu        string
	PluName    string
	Category   string
	Enabled    bool
	SetEnabled bool // 是否设置了Enabled条件
}

type AddProduct struct {
	RecId       int
	Plu         string
	ProductCode string
	ItemCode    string
	Category    string
	ProductName string
	GeneralUnit string
	TaxType     string
	Price       string
	UnitWeight  string
	Pretare     string
	LimitHigh   string
	LimitLow    string
	CreateBy    int
	UpdateBy    int
	CreateUser  string
	UpdateUser  string
}

type ReqDelProduct struct {
	RecId []int
}

type ReqUpdateEnabledPlu struct {
	PluList  []int
	Enabled  bool
	UpdateBy string
}

type ReqModifyProduct struct {
	RecId       int64
	Id          string
	Product     string
	WithPretare bool
	Pretare     string
	Remarks     string
}

type ReqAddUser struct {
	Id       string
	Name     string
	IsFemale bool
	Phone    string
	Remarks  string
}

type ReqAddWifi struct {
	Ssid string
	Pwd  string
}

type ReqDoServiceAction struct {
	ServiceId int64
	Action    string
}

type ReqDelUser struct {
	RecId int64
}

type ReqModifyUser struct {
	RecId    int64
	Id       string
	Name     string
	IsFemale bool
	Phone    string
	Remarks  string
}

type ReqAddModbusService struct {
	TargetModbusId int
	Protocol       string // "RTU" or "TCP"
	Port           string // "COM2" or "502"
	BaudRate       int    // 9600
}

type ReqEditModbusService struct {
	Id             uint
	TargetModbusId int
	Protocol       string
	Port           string
	BaudRate       int
}

type ReqDelModbusService struct {
	Id uint
}

type ReqAddRawType struct {
	Name string
}

type ReqEditRawType struct {
	Id   int
	Name string
}

type ReqAddFormulaType struct {
	Name string
}

type ReqUpdateAutoNext struct {
	AutoNext   bool
	StableTime int
	AutoTare   bool
	CheckCode  bool
}

type ReqUpdateOutputPort struct {
	Port      int
	Status    bool
	StartTime int
	EndValue  float64
	Remark    string
}

type ReqUpdateUnstableZeroTare struct {
	Enable bool
}

type ReqUpdateInputPort struct {
	Port int
	Btn  string
}

type ReqDeleteDraftFmaWgtRec struct {
	OrderId string
}

type ReqDeleteAllDraftFmaWgtRec struct {
	OrderId []string
}

type ReqGetRawOutputByFmaId struct {
	FormulaId string
}

type ReqAddRawData struct {
	MaterialID   string
	MaterialName string
	CategoryID   int
	Ingredient   string
	CreatedBy    string
	UpdatedBy    string
	Remark       string
	Remark1      string
	ScaleId      int
	CheckCode    string
	Output       int
}

type ReqImportRawList struct {
	RawInfo   []ReqAddRawInfo
	CreatedBy string
}

type ReqAddRawInfo struct {
	MaterialID   string
	MaterialName string
	CategoryName string
	Ingredient   string
	ScaleName    string
	CategoryId   int
	ScaleId      int
	CheckCode    string
}

type ReqImportFmaList struct {
	FmaInfo  []ReqAddFmaInfo
	CreateBy string
}
type ReqAddFmaInfo struct {
	FormulaId      string
	FormulaName    string
	Mode           string
	WeightUnit     string
	Category       string
	IsConfidential bool
	NeedContainer  bool
	Ingredients    []ReqAddFmaIngredient
	Notes          string
	Barcode        string
}
type ReqAddFmaIngredient struct {
	IngredientNo    int
	IngredientId    string
	IngredientName  string
	WeightOrPercent float64
	AllowError      float64
}

type ReqEditRawData struct {
	RecId        int
	MaterialID   string
	MaterialName string
	CategoryID   int
	Ingredient   string
	CreatedBy    string
	UpdatedBy    string
	Remark       string
	Remark1      string
	ScaleId      int
	CheckCode    string
	Output       int
}
type ReqDelRawData struct {
	RecId int
}

type ReqDelAllRawData struct {
	RecId []int
}

type ReqDelFmaData struct {
	RecId int
}

type ReqDelAllFmaData struct {
	RecID []int
}

type ReqGetFormulaByBarcode struct {
	Barcode string
}

type ReqCheckFmaIdAndBarcode struct {
	RecId          int
	FormulaID      string
	FormulaBarcode string
}

type ReqAddFormulaData struct {
	Header ReqAddFormulaHeader
	Detail []ReqAddFormulaDetail
}

type ReqAddSysUser struct {
	RoleId        int
	Username      string
	NickName      string
	Password      string
	IsEnabled     bool
	Email         string
	Phone         string
	InitialPageId int
	Remark        string
	CreatedBy     int
	UpdatedBy     int
	PagesId       []int
}

type ReqUpdateSysUser struct {
	UpdateUser UpdateUser
	PagesId    []int
}
type UpdateUser struct {
	UserId        int    `gorm:"primaryKey;not null;autoincrement;"`
	UserName      string `gorm:"not null;unique"`         //账户名 用于登录
	NickName      string `gorm:"not null;default:'name'"` // 昵称 或者姓名
	RoleId        int    `gorm:"not null;"`
	Password      string `gorm:"not null"`
	IsEnabled     bool   `gorm:"not null;default:true;"`
	Email         string
	Phone         string
	InitialPageId int
	Remark        string
	CreatedBy     int
	UpdatedBy     int
}
type ReqEnabledSysUserId struct {
	UserId    int
	IsEnabled bool
}
type ReqChangePassword struct {
	UserId      int
	NewPassword string
}

// 登录
type ReqLogin struct {
	UserName  string
	Password  string
	AutoLogin bool
}

// 用户
type ReqSysUserName struct {
	UserName string
}

// 用户Id
type ReqSysUserId struct {
	UserId int
}

// 用户Id
type ReqSysUserIdList struct {
	UserIds []int
}

// 新增系统日志记录
type ReqAddSysLog struct {
	RecId         int
	Operator      string
	RoleId        int
	Module        string
	FuncName      string
	OperationType string
	Operation     string
	Remarks       string
	CreateTime    time.Time
}

// 标定日志

// 称重日志
type ReqAddScaleLog struct {
	RecId      int
	Operator   string
	RoleId     int
	Module     string
	ScaleId    int
	ScaleName  string
	ModelName  string
	Sn         string
	Weight     string
	Unit       string
	Remarks    string
	CreateTime time.Time
}

type ReqDelLogs struct {
	RecId []int
}

type ReqExportLog struct {
	FilePath    string
	FieldName   string
	Direction   string
	Search      LogQuery
	Translation map[string]string
	Headers     []string
}

type ReqGetLog struct {
	Page      int
	PageSize  int
	FieldName string
	Direction string
	Search    LogQuery
}

type ReqPortInfo struct {
	PortId int
	Status bool
}

type GetSealLogReq struct {
	Model string
	Sn    string
}

type LogQuery struct {
	Operator  string // 操作员
	Module    string
	RoleId    int    // 角色ID
	StartTime string // 创建时间起始
	EndTime   string // 创建时间结束
}

// 配方头表
type ReqAddFormulaHeader struct {
	RecId      int
	FormulaKey int `gorm:"not null"`
	// 配方编号（主键）
	FormulaID string `gorm:"not null"`
	// 配方名称
	FormulaName string `gorm:"not null"`
	// 配方类别
	CategoryID int `gorm:"not null"`
	// 配方模式
	FormulaMode string
	// 配方单位
	FormulaUnit string
	// 配方总重量
	TotalWeight float64
	// 原料数量
	MaterialCount int
	// 是否加密
	IsEncrypted bool
	//是否需要容器
	NeedContainer bool
	// 配方创建人
	CreatedBy string
	// 配方修改人
	UpdatedBy string
	// 备注
	Remark string
	//配方条码
	FormulaBarcode string
}

// 配方明细表
type ReqAddFormulaDetail struct {
	// 配方编号（主键）
	FormulaRecID int `gorm:"not null"`
	// 原料编号（主键）
	MaterialID string `gorm:"not null"`
	// 原料重量
	MaterialWeight float64
	// 原料百分比
	MaterialPercentage float64
	// 序号
	Sequence int
	// 允许误差
	AllowableError float64
	// 备注
	Remark string
}

// 配方称重记录
type ReqFormulaWgtRec struct {
	RecHeader ReqFormulaWgtRecHeader
	RecDetail []ReqFormulaWgtRecDetail
}

// FormulaWgtRecHeader 配方称重记录头表
type ReqFormulaWgtRecHeader struct {
	// 记录编号（主键）
	RecordID string `gorm:"not null"`
	// 记录操作员
	Operator string
	// 配方编号
	FormulaID string
	// 配方类别名称
	FormulaTypeName string
	// 配方总重量
	TotalWeight float64
	//实际总重量
	ActualTotalWeight float64
	// 总重量单位
	TotalWeightUnit string
	// 原料总重量
	TotalMaterialWeight float64
	// 原料总重量单位
	TotalMaterialWeightUnit string
	// 是否达标
	IsQualified string
	//配方实际需要的重量
	ActualFmaTotalWgt float64
	ScaleId           int
	ScaleName         string
	ScaleModel        string
	ScaleSn           string
	FormulaBarcode    string
}

// FormulaWgtRecDetail 配方称重记录详情表
type ReqFormulaWgtRecDetail struct {
	RecId int `gorm:"primaryKey;autoincrement;not null"`
	// 记录编号（主键）
	RecordID string `gorm:"not null"`
	// 原料编号（主键）
	MaterialID     string `gorm:"not null"`
	MaterialTypeId string
	// 原料类别名称
	MaterialTypeName string
	// 序号
	Sequence int
	//目标重量
	TargetWgt float64
	// 允许误差
	AllowableError float64
	// 实际重量
	ActualWeight float64
	// 实际重量单位
	ActualWeightUnit string
	// 实际百分比
	ActualPercentage float64
	// 实际误差重量
	ActualErrorWgt float64
	// 实际误差百分比
	ActualErrorPct float64
	// 达标情况
	IsQualified string
	//记录秤号
	ScaleId    int
	ScaleName  string
	ScaleModel string
	ScaleSn    string
	CheckCode  string
}

type ReqFlowRateRec struct {
	RecHeader ReqFlowRateHeader
	RecDetail []ReqFlowRateDetail
}

// FlowRateHeader 流速头表格
type ReqFlowRateHeader struct {
	TotalWeight     float64
	TotalTime       float64
	AverageFlowRate float64
	MinFlowRate     float64
	MaxFlowRate     float64
	WgtUnit         string
}

// FlowRateDetail 流速明细表
type ReqFlowRateDetail struct {
	Id   int
	Rate float64
	Time float64
}

// 公共接口去拿所有称重的数据（分称重类型）
type ReqGetAllWgtRecList struct {
	Mode       int //按称重模式 重量收集:0  加法秤:，减法秤: ，检重秤:
	Page       int
	PageSize   int
	ColumnName string //排序字段
	Direction  string // 排序的方向

}

// 按条件拿数据
type ReqGetSearchRecList struct {
	Mode        int //按称重模式 重量收集:0  加法秤:，减法秤: ，检重秤:
	ScaleModel  string
	ScaleSn     string
	Plu         string
	ProductCode string
	ItemCode    string
	Category    string
	ProductName string
	GeneralUnit string
	TaxType     string
	UnitWeight  string
	WeightUnit  string
	UserNo      string
	UserName    string
	ScaleMode   string //0 =DC500 1=check Weigher 2=take in  3=take out
	ScaleName   string
	Page        int
	PageSize    int
	ColumnName  string //排序字段
	Direction   string // 排序的方向
}

type ReqAddWgtRec struct {
	Mode      int
	HeadRec   ScaleRec
	DetailRec []ScaleRecDetail
}

type ReqDelWgtRec struct {
	Mode uint
}

type ReqDelWgtRecById struct {
	Mode  uint
	RecId uint
}

type ReqExportAllRecs struct {
	Mode        uint
	Path        string
	Translation map[string]string
	FieldName   []string
}

// ********** Response of scale manager **********
type ScaleMgrRespMsg struct {
	MsgType ScaleMgrRespMsgType
	MsgBody interface{} // MsgBody [T PortsListMsg|ScalesListMsg|MgrRespMsg|string] []T
}

// 小服务的数据格式
type SrvMgrRespMsg struct {
	MsgType ScaleMgrRespMsgType
	MsgBody interface{} // MsgBody [T PortsListMsg|ScalesListMsg|MgrRespMsg|string] []T
	ScaleId int64
}

type ModbusServiceInfo struct {
	Id             uint   `json:"Id" gorm:"primaryKey;autoincrement"`
	TargetModbusId int    `json:"TargetModbusId"`
	Protocol       string `json:"Protocol"` // "RTU" or "TCP"
	Port           string `json:"Port"`     // "COM2" or "502"
	BaudRate       int    `json:"BaudRate"` // 9600
}

type ScaleMgrRespMsgType string

// 处理公用的回应
const (
	SCALE_MGR_RESP_PORTS_LIST    ScaleMgrRespMsgType = "resp_ports_list"   // with response of PortsListMsg
	SCALE_MGR_RESP_BT_LIST       ScaleMgrRespMsgType = "resp_bt_list"      // with response of ScalesListMsg
	SCALE_MGR_RESP_SCALES_LIST   ScaleMgrRespMsgType = "resp_scales_list"  // with response of ScalesListMsg
	SCALE_MGR_RESP_SCALE_ADD     ScaleMgrRespMsgType = "resp_scale_add"    // with response of MgrRespMsg to indicate that status coreponding request procsssed
	SCALE_MGR_RESP_SCALE_DEL     ScaleMgrRespMsgType = "resp_scale_del"    // same as SCALE_MGR_RESP_SCALE_Add
	SCALE_MGR_RESP_SCALE_MODIFY  ScaleMgrRespMsgType = "resp_scale_modify" // same as SCALE_MGR_RESP_SCALE_Add
	SCALE_MGR_RESP_PRODUCTS_LIST ScaleMgrRespMsgType = "resp_product_list" // with response of ScalesListMsg

	SCALE_MGR_RESP_CHECK_PLU_EXIST ScaleMgrRespMsgType = "resp_check_plu_exist" // with response of string "ok" or "not exist"
	SCALE_MGR_RESP_EXPORT_PLU_LIST ScaleMgrRespMsgType = "resp_export_plu_list" // with response of ScalesListMsg

	SCALE_MGR_RESP_PLU_LIST        ScaleMgrRespMsgType = "resp_plu_list"        // with response of ScalesListMsg
	SCALE_MGR_RESP_PRODUCT_ADD     ScaleMgrRespMsgType = "resp_product_add"     // with response of MgrRespMsg to indicate that status coreponding request procsssed
	SCALE_MGR_RESP_PRODUCT_ADD_ONE ScaleMgrRespMsgType = "resp_product_add_one" // with response of MgrRespMsg to indicate that status coreponding request procsssed
	SCALE_MGR_RESP_PRODUCT_DEL     ScaleMgrRespMsgType = "resp_product_del"     // same as SCALE_MGR_RESP_SCALE_Add
	SCALE_MGR_RESP_PLU_SETTING     ScaleMgrRespMsgType = "resp_plu_setting"     // with response of MgrRespMsg to indicate that status coreponding request procsssed

	SCALE_MGR_RESP_DOWN_ALL_PLU         ScaleMgrRespMsgType = "resp_down_all_plu"         // with response of MgrRespMsg to indicate that status coreponding request procsssed
	SCALE_MGR_RESP_UPDATE_ENABLED_PLU   ScaleMgrRespMsgType = "resp_update_enabled_plu"   // with response of MgrRespMsg to indicate that status coreponding request procsssed
	SCALE_MGR_RESP_GET_LAST_PRODUCT_REC ScaleMgrRespMsgType = "resp_get_last_product_rec" // with response of ScalesListMsg
	SCALE_MGR_RESP_PRODUCT_MODIFY       ScaleMgrRespMsgType = "resp_product_modify"       // same as SCALE_MGR_RESP_SCALE_Add
	SCALE_MGR_RESP_USERS_LIST           ScaleMgrRespMsgType = "resp_user_list"            // with response of ScalesListMsg
	SCALE_MGR_RESP_USER_ADD             ScaleMgrRespMsgType = "resp_user_add"             // with response of MgrRespMsg to indicate that status coreponding request procsssed
	SCALE_MGR_RESP_USER_DEL             ScaleMgrRespMsgType = "resp_user_del"             // same as SCALE_MGR_RESP_SCALE_Add
	SCALE_MGR_RESP_USER_MODIFY          ScaleMgrRespMsgType = "resp_user_modify"          // same as SCALE_MGR_RESP_SCALE_Add
	SCALE_MGR_RESP_QUIT_APPLICATION     ScaleMgrRespMsgType = "resp_quit_application"     // without data
	// SCALE_MGR_RESP_GET_UI_CONFIG     ScaleMgrRespMsgType = "resp_get_ui_config"     // with response of UI configuration
	// SCALE_MGR_RESP_UPDATE_UI_CONFIG  ScaleMgrRespMsgType = "resp_update_ui_config"  // without parameter
	SCALE_MGR_RESP_GET_LICENSE       ScaleMgrRespMsgType = "resp_get_license"       // with response of true or false
	SCALE_MGR_RESP_CHECK_LICENSE_KEY ScaleMgrRespMsgType = "resp_check_license_key" // with response of true or false
	SCALE_MGR_RESP_UPDATE_LICENSE    ScaleMgrRespMsgType = "resp_update_license"    // with response of true or false
	SCALE_MGR_RESP_DETAIL_LIST       ScaleMgrRespMsgType = "resp_detail_list"       // with response of ScalesListMsg
	SCALE_MGR_RESP_WIFI_PWD_LIST     ScaleMgrRespMsgType = "resp_wifi_pwd_list"     // with response of ScalesListMsg
	SCALE_MGR_RESP_WIFI_PWD_ADD      ScaleMgrRespMsgType = "resp_wifi_pwd_add"      // with response
	SCALE_MGR_RESP_SCALE_ONLINE      ScaleMgrRespMsgType = "resp_scale_online"      //回复秤的状态
	SCALE_MGR_RESP_SCALE_INPUT       ScaleMgrRespMsgType = "resp_scale_input"       //回复秤输入口按下的状态
	//下面是添加给小服务的
	SCALE_MGR_RESP_SNED_TO_SRV1       ScaleMgrRespMsgType = "resp_send_to_srv1"       // with response
	SCALE_MGR_RESP_GET_SCALE_SRV_LIST ScaleMgrRespMsgType = "resp_get_scale_srv_list" // with response
	SCALE_MGR_RESP_SET_SCALE_SRV_VAL  ScaleMgrRespMsgType = "resp_set_scale_srv_val"  // with response
	SCALE_MGR_RESP_DO_SERVICE_ACTION  ScaleMgrRespMsgType = "resp_do_service_action"  // with response

	//配方秤
	SCALE_MGR_RESP_RAW_TYPE_ADD             ScaleMgrRespMsgType = "resp_raw_type_add"
	SCALE_MGR_RESP_RAW_TYPE_EDIT            ScaleMgrRespMsgType = "resp_raw_type_edit"
	SCALE_MGR_RESP_RAW_TYPE_DELETE          ScaleMgrRespMsgType = "resp_raw_type_delete"
	SCALE_MGR_RESP_FMA_TYPE_EDIT            ScaleMgrRespMsgType = "resp_fma_type_edit"
	SCALE_MGR_RESP_FMA_TYPE_DELETE          ScaleMgrRespMsgType = "resp_fma_type_delete"
	SCALE_MGR_RESP_FORMULA_TYPE_ADD         ScaleMgrRespMsgType = "resp_formula_type_add"
	SCALE_MGR_RESP_RAW_TYPE_DEL_UNUSED      ScaleMgrRespMsgType = "resp_raw_type_unused_del"
	SCALE_MGR_RESP_FORMULA_TYPE_DEL_UNUSED  ScaleMgrRespMsgType = "resp_fma_type_unused_del"
	SCALE_MGR_RESP_FORMULA_TYPE_LIST        ScaleMgrRespMsgType = "resp_formula_type_list"
	SCALE_MGR_RESP_RAW_TYPE_LIST            ScaleMgrRespMsgType = "resp_raw_type_list"
	SCALE_MGR_RESP_RAW_LIST                 ScaleMgrRespMsgType = "resp_raw_list"
	SCALE_MGR_RESP_RAW_DATA_EDIT            ScaleMgrRespMsgType = "resp_raw_data_edit"
	SCALE_MGR_RESP_RAW_DATA_DELETE          ScaleMgrRespMsgType = "resp_raw_data_delete"
	SCALE_MGR_RESP_RAW_DATA_ADD             ScaleMgrRespMsgType = "resp_raw_data_add"
	SCALE_MGR_RESP_RAW_OUTPUT_BY_FMA_ID     ScaleMgrRespMsgType = "resp_raw_output_by_fma_id"
	SCALE_MGR_RESP_RAW_LIST_IMPORT          ScaleMgrRespMsgType = "resp_raw_list_import"
	SCALE_MGR_RESP_FMA_LIST_IMPORT          ScaleMgrRespMsgType = "resp_fma_list_import"
	SCALE_MGR_RESP_FORMULA_ADD              ScaleMgrRespMsgType = "resp_formula_add"
	SCALE_MGR_RESP_FORMULA_UPDATE           ScaleMgrRespMsgType = "resp_formula_update"
	SCALE_MGR_RESP_FORMULA_LIST             ScaleMgrRespMsgType = "resp_formula_list"
	SCALE_MGR_RESP_FORMULA_LIST_BY_BARCODE  ScaleMgrRespMsgType = "resp_formula_list_by_barcode"
	SCALE_MGR_RESP_CHECK_FMA_ID_AND_BARCODE ScaleMgrRespMsgType = "resp_check_fma_id_and_barcode"

	SCALE_MGR_RESP_FORMULA                       ScaleMgrRespMsgType = "resp_formula_data"
	SCALE_MGR_RESP_RAW                           ScaleMgrRespMsgType = "resp_raw_data"
	SCALE_MGR_RESP_FORMULA_REC_ADD               ScaleMgrRespMsgType = "resp_formula_rec_add"
	SCALE_MGR_RESP_FORMULA_REC_LIST              ScaleMgrRespMsgType = "resp_formula_rec_list"
	SCALE_MGR_RESP_ONE_FORMULA_REC_LIST          ScaleMgrRespMsgType = "resp_one_fma_rec_list"
	SCALE_MGR_RESP_FORMULA_REC_BY_ORDER          ScaleMgrRespMsgType = "resp_formula_rec_by_order"
	SCALE_MGR_RESP_FORMULA_DELETE                ScaleMgrRespMsgType = "resp_formula_delete"
	SCALE_MGR_RESP_MANY_FMA_DELETE               ScaleMgrRespMsgType = "resp_many_fma_del"
	SCALE_MGR_RESP_MANY_RAW_DELETE               ScaleMgrRespMsgType = "resp_many_raw_del"
	SCALE_MGR_RESP_MANY_DRAFT_FMA_WGT_REC_DELETE ScaleMgrRespMsgType = "resp_many_draft_fma_del"
	SCALE_MGR_RESP_FLOW_RATE_ADD                 ScaleMgrRespMsgType = "resp_flow_rate_add"
	SCALE_MGR_RESP_FLOW_RATE_LIST                ScaleMgrRespMsgType = "resp_flow_rate_list"
	SCALE_MGR_RESP_GET_ALL_WGT_REC_LIST          ScaleMgrRespMsgType = "resp_get_all_wgt_rec_list"
	SCALE_MGR_RESP_GET_SEARCH_REC_LIST           ScaleMgrRespMsgType = "resp_get_search_rec_list"
	SCALE_MGR_RESP_ADD_WGT_REC                   ScaleMgrRespMsgType = "resp_add_wgt_rec"
	SCALE_MGR_RESP_DEL_WGT_REC                   ScaleMgrRespMsgType = "resp_del_wgt_rec"
	SCALE_MGR_RESP_DEL_WGT_REC_BY_ID             ScaleMgrRespMsgType = "resp_del_wgt_rec_by_id"
	SCALE_MGR_RESP_GET_UI_CONFIG                 ScaleMgrRespMsgType = "resp_get_ui_config"       // with response of UI configuration
	SCALE_MGR_RESP_UPDATE_UI_CONFIG              ScaleMgrRespMsgType = "resp_update_ui_config"    // without parameter
	SCALE_MGR_RESP_EXPORT_ALL_RECS               ScaleMgrRespMsgType = "resp_export_all_recs"     // without parameter
	SCALE_MGR_RESP_KILL_BOOT_COMMANDER           ScaleMgrRespMsgType = "resp_kill_boot_commander" // without parameter
	SCALE_MGR_RESP_GET_AUTO_NEXT                 ScaleMgrRespMsgType = "resp_get_auto_next"
	SCALE_MGR_RESP_GET_UNSTABLE_ZERO_TARE        ScaleMgrRespMsgType = "resp_get_unstable_zero_tare"
	SCALE_MGR_RESP_UPDATE_UNSTABLE_ZERO_TARE     ScaleMgrRespMsgType = "resp_update_unstable_zero_tare"
	SCALE_MGR_RESP_UPDATE_AUTO_NEXT              ScaleMgrRespMsgType = "resp_update_auto_next"
	SCALE_MGR_RESP_GET_OUTPUT_PORT               ScaleMgrRespMsgType = "resp_get_output_port"    //获取输出端口状态
	SCALE_MGR_RESP_UPDATE_OUTPUT_PORT            ScaleMgrRespMsgType = "resp_update_output_port" //更新输出端口状态

	SCALE_MGR_RESP_UPDATE_INPUT_PORT ScaleMgrRespMsgType = "resp_update_input_port" //更新输入端口状态
	SCALE_MGR_RESP_GET_INPUT_PORT    ScaleMgrRespMsgType = "resp_get_input_port"    //获取输入端口状态

	SCALE_MGR_RESP_UPDATE_SET_REPORT_PRINT ScaleMgrRespMsgType = "resp_update_set_report_print" //更新报表打印设置
	SCALE_MGR_RESP_GET_SET_REPORT_PRINT    ScaleMgrRespMsgType = "resp_get_set_report_print"

	SCALE_MGR_RESP_UPLOAD_SERVER_EDIT ScaleMgrRespMsgType = "resp_upload_server_edit" //编辑上传配方称重记录服务器
	SCALE_MGR_RESP_UPLOAD_SERVER_GET  ScaleMgrRespMsgType = "resp_upload_server_get"  //获取上传配方称重记录服务器

	SCALE_MGR_RESP_GET_DRAFT_FMA_WGT_REC_LIST ScaleMgrRespMsgType = "resp_get_draft_fma_wgt_rec_list"    //获取草稿配方称重记录列表
	SCALE_MGR_RESP_UPDATE_DRAFT_FMA_WGT_REC   ScaleMgrRespMsgType = "resp_update_draft_fma_wgt_rec_list" //更新草稿配方称重记录列表
	SCALE_MGR_RESP_DELETE_DRAFT_FMA_WGT_REC   ScaleMgrRespMsgType = "resp_delete_draft_fma_wgt_rec_list" //删除草稿配方称重记录列表
	SCALE_MGR_RESP_CREATE_DRAFT_FMA_WGT_REC   ScaleMgrRespMsgType = "resp_create_draft_fma_wgt_rec_list" //创建草稿配方称重记录列表

	SCALE_MGR_RESP_ADD_SYS_USER     ScaleMgrRespMsgType = "resp_add_sys_user"     //新增系统用户
	SCALE_MGR_RESP_DELETE_SYS_USER  ScaleMgrRespMsgType = "resp_delete_sys_user"  //删除系统用户
	SCALE_MGR_RESP_UPDATE_SYS_USER  ScaleMgrRespMsgType = "resp_update_sys_user"  //更新系统用户
	SCALE_MGR_RESP_DISABLE_SYS_USER ScaleMgrRespMsgType = "resp_disable_sys_user" //禁用系统用户
	SCALE_MGR_RESP_CHANGE_PASSWORD  ScaleMgrRespMsgType = "resp_change_password"  //修改密码
	SCALE_MGR_RESP_LOGIN            ScaleMgrRespMsgType = "resp_login"            //登录
	SCALE_MGR_RESP_GET_ALL_USERS    ScaleMgrRespMsgType = "resp_get_all_users"    //获取所有用户列表
	SCALE_MGR_RESP_GET_USER_DETAIL  ScaleMgrRespMsgType = "resp_get_user_detail"  //获取用户详情

	SCALE_MGR_RESP_SYS_LOG_ADD   ScaleMgrRespMsgType = "resp_sys_log_add"   //新增系统日志记录
	SCALE_MGR_RESP_CAL_LOG_ADD   ScaleMgrRespMsgType = "resp_cal_log_add"   //新增校准日志记录
	SCALE_MGR_RESP_SCALE_LOG_ADD ScaleMgrRespMsgType = "resp_scale_log_add" //新增称重日志记录

	SCALE_MGR_RESP_ADD_RAW_TYPE    ScaleMgrRespMsgType = "resp_add_raw_type"
	SCALE_MGR_RESP_DEL_RAW_TYPE    ScaleMgrRespMsgType = "resp_del_raw_type"
	SCALE_MGR_RESP_EDIT_RAW_TYPE   ScaleMgrRespMsgType = "resp_edit_raw_type"
	SCALE_MGR_RESP_GET_RAW_TYPE    ScaleMgrRespMsgType = "resp_get_raw_type"

	// Modbus
	SCALE_MGR_RESP_MODBUS_SERVICES ScaleMgrRespMsgType = "resp_modbus_services"
	SCALE_MGR_RESP_MODBUS_ADD      ScaleMgrRespMsgType = "resp_modbus_add"
	SCALE_MGR_RESP_MODBUS_EDIT     ScaleMgrRespMsgType = "resp_modbus_edit"
	SCALE_MGR_RESP_MODBUS_DEL      ScaleMgrRespMsgType = "resp_modbus_del"

	SCALE_MGR_RESP_DEL_SYS_LOG   ScaleMgrRespMsgType = "resp_del_sys_log"   //删除系统日志记录
	SCALE_MGR_RESP_DEL_CAL_LOG   ScaleMgrRespMsgType = "resp_del_cal_log"   //删除校准日志记录
	SCALE_MGR_RESP_DEL_SCALE_LOG ScaleMgrRespMsgType = "resp_del_scale_log" //删除称重日志记录

	SCALE_MGR_RESP_DEL_ALL_SYS_LOG   ScaleMgrRespMsgType = "resp_del_all_sys_log"   //删除所有系统日志记录
	SCALE_MGR_RESP_DEL_ALL_CAL_LOG   ScaleMgrRespMsgType = "resp_del_all_cal_log"   //删除所有校准日志记录
	SCALE_MGR_RESP_DEL_ALL_SCALE_LOG ScaleMgrRespMsgType = "resp_del_all_scale_log" //删除所有称重日志记录

	SCALE_MGR_RESP_GET_SYS_LOG_LIST   ScaleMgrRespMsgType = "resp_get_sys_log_list"   //获取系统日志记录列表
	SCALE_MGR_RESP_GET_CAL_LOG_LIST   ScaleMgrRespMsgType = "resp_get_cal_log_list"   //获取校准日志记录列表
	SCALE_MGR_RESP_GET_SCALE_LOG_LIST ScaleMgrRespMsgType = "resp_get_scale_log_list" //获取称重日志记录列表

	SCALE_MGR_RESP_EXPORT_SYS_LOG   ScaleMgrRespMsgType = "resp_export_sys_log"   //导出系统日志记录列表
	SCALE_MGR_RESP_EXPORT_CAL_LOG   ScaleMgrRespMsgType = "resp_export_cal_log"   //导出校准日志记录列表
	SCALE_MGR_RESP_EXPORT_SCALE_LOG ScaleMgrRespMsgType = "resp_export_scale_log" //导出称重日志记录列表

	SCALE_MGR_RESP_GET_ALL_SEAL_LOG     ScaleMgrRespMsgType = "resp_get_all_seal_log"     //获取所有铅封日志记录
	SCALE_MGR_RESP_UNSEAL_BY_MASTER_KEY ScaleMgrRespMsgType = "resp_unseal_by_master_key" //使用主密钥解封

)

type PortsListMsg struct {
	PortsList []serial.Port
}

type ScalesListMsg struct {
	ScaleList []ScaleConnMedia
}

type ProductsListMsg struct {
	ProductList []ProductRec
}

type UsersListMsg struct {
	UserList []UserRec
}

type ScaleConnMedia struct { // connection information will be stored in database
	IsOnline   bool
	ScaleModel string //默认与CustomModel 相同 但是，如果能从秤上读取到机种名，就存储秤上的机种名，并不显示给客户
	ScaleCat   comm.ScaleCat
	ScaleSn    string
	ScaleId    int64
	TMedia     MediaType
	MediaConf  MediaConf `gorm:"embedded;embeddedPrefix:mediainfo_"`
	scale      *Scale    `gorm:"-"` // should not be stored in database
	IsDefault  bool      //是否默认的连接方式   新增的秤连接方式都视为默认的，sn和model name 一样的连上后，将isdefault改为仅一个默认
	ScaleName  string
	// 为小服务新增字段
	SendService bool // 是否发送数据给小服务
	//新增字段 2025/12/1
	CustomModel  string //客户机种名  显示的话，一直显示这个机种名
	InnerModel   string //内部机种名
	ProtocolName string //协议名

	ModbusId int // 关联的 Modbus 从机站号 (0 表示未分配)
}

// 服务与秤的关系，哪些服务管理哪些秤
type SrvScaleRel struct {
	ScaleId int64
	SrvId   int64
	IsUsed  bool
}

type MediaConf struct {
	Type          MediaType
	MediaInfoJson string // will be unmarshaled json of ComInfo, NetInfo and BtInfo
}
type MediaType int

const (
	MEDIA_COM MediaType = iota
	MEDIA_NET
	MEDIA_BT
)

type ComInfo struct {
	DevPath  string // device path of Com port, e.g. COM3
	Baud     int    // e.x. 9600
	DataBits int    // value: 7,8,9
	StopBits int    // 0: 1 stop bit, 1: 1.5 stop bits, 2: 2 stop bits
	Parity   int    // 0: no parity, 1: odd, 2: even
}
type NetInfo struct {
	Ip   string // format should be xxx.xxx.xxx.xxx
	Port int    // value should be 1-65535
}
type BtInfo struct {
	Mac  string
	Name string
}

type MgrRespMsg struct {
	IsAck   bool
	AckData string
}

type SRequest struct { // request for a scale or scale manager for records
	Req     SReqType
	ReqData string // should be json encoded string of one of the following structures, i.e. ReqScaleRec, ReqAddScaleRec, ReqDelScaleRec
}

type SReqType string

const (
	SREQ_ZERO                     SReqType = "zero"
	SREQ_TARE                     SReqType = "tare"
	SREQ_ZERO_UNSTABLE            SReqType = "zero_unstable"
	SREQ_TARE_UNSTABLE            SReqType = "tare_unstable"
	SREQ_GET_WEIGHT               SReqType = "get_weight"
	SREQ_SEND_WT_CONT             SReqType = "send_wt_cont"
	SREQ_STOP_SEND_WT             SReqType = "stop_send_wt"
	SREQ_REG_WEIGHT_DATA          SReqType = "reg_weight_data"
	SREQ_UNREG_WEIGHT_DATA        SReqType = "unreg_weight_data"
	SREQ_GET_RECS                 SReqType = "get_recs"                   // with parameter ReqScaleRec
	SREQ_ADD_REC                  SReqType = "add_rec"                    // with parameter ReqAddScaleRec
	SREQ_DEL_REC                  SReqType = "del_rec"                    // with parameter ReqDelScaleRec
	SREQ_DOWN_PRN_FMT             SReqType = "down_print_format_to_scale" // with parameter csv formatted string
	SREQ_GET_AP_LIST              SReqType = "get_ap_list"
	SREQ_RESCAN_AP_LIST           SReqType = "rescan_ap_list"
	SREQ_CONNECT_AP               SReqType = "connect_ap"
	SREQ_CONNECT_AP_ONE_KEY       SReqType = "connect_ap_one_key"
	SREQ_SET_WIFI_DYNAMIC_IP      SReqType = "set_wifi_dynamic_ip"
	SREQ_SET_WIFI_STATIC_IP       SReqType = "set_wifi_static_ip"
	SREQ_GET_IP_INFO              SReqType = "get_ip_info"
	SREQ_MODIFY_BT_NAME           SReqType = "modify_bt_name"
	SREQ_SEND_DATA_TO_BT          SReqType = "send_data_to_bt"
	SREQ_SEND_DATA_TO_WIFI        SReqType = "send_data_to_wifi"
	SREQ_GET_IP_MODE              SReqType = "get_ip_mode"
	SREQ_GET_WIFI_INFO            SReqType = "get_wifi_info"
	SREQ_UPDATE_FIRMWARE          SReqType = "update_firmware"
	SREQ_DOWN_FIRMWARE_WIFI       SReqType = "update_firmware_wifi"
	SREQ_CHECK_SERIAL_PORT        SReqType = "check_serial_port"
	SREQ_GET_BUILD_INFO           SReqType = "get_build_info"      //20230926@FLF
	SREQ_GET_SCALE_TIME           SReqType = "get_scale_time"      //20240112@FLF
	SREQ_SET_SCALE_TIME           SReqType = "set_scale_time"      //20240112@FLF
	SREQ_GET_ONE_EEPROM_INFO      SReqType = "get_one_eeprom_info" //20240125@FLF
	SREQ_GET_ALL_EEPROM_INFO      SReqType = "get_all_eeprom_info" //20240129@FLF
	SREQ_SET_OUTPUT_FMT           SReqType = "set_output_format"
	SREQ_OPNE_SCALE_PASSTHROUGH   SReqType = "open_scale_passthrough" //20231023@FLF
	SREQ_CLOSE_SCALE_PASSTHROUGH  SReqType = "close_scale_passthrough"
	SREQ_CHANGE_SCALE_PASSTH_MODE SReqType = "change_scale_passth_mode"
	SREQ_GET_SCALE_INFO           SReqType = "get_scale_info"   //20231101@FLF
	SREQ_GET_FACTORY_INFO         SReqType = "get_factory_info" //20240417@FLF
	SREQ_GET_WEIGHT_ERR           SReqType = "get_weight_err"   //20240111@FLF
	SREQ_DOWN_PLU                 SReqType = "down_plu_to_scale"
	SREQ_DEL_PLU                  SReqType = "del_plu_from_scale"
	SREQ_INSERT_PLU               SReqType = "insert_plu_to_scale"
	SREQ_GET_UI_CONF              SReqType = "get_ui_conf"
	SREQ_UPDATE_UI_CONF           SReqType = "update_ui_conf"
	SREQ_CHANGE_WIFI_MODE         SReqType = "change_wifi_mode"
	SREQ_MODIFY_EEPROM_INFO       SReqType = "modify_eeprom_info"
	SREQ_DOWN_EEPROM_INFO         SReqType = "down_eeprom_info"
	SREQ_DOWN_FACTORY_INFO_FC     SReqType = "down_factory_info"
	SREQ_DOWN_FACTORY_INFO        SReqType = "down_factory_info_tmax"
	SREQ_MODIFY_VAR_VALUE         SReqType = "modify_var_value"
	SREQ_SET_SERVER_IP            SReqType = "set_server_ip"
	SREQ_EN_FACTORY_MODE          SReqType = "en_factory_mode"
	SREQ_DOWN_DEFAULT_PRN_FMT     SReqType = "down_def_print_format"
	SREQ_BACKUP_DEF_SETTING       SReqType = "backup_def_setting"
	SREQ_GET_EEPROM_TO_BIN        SReqType = "get_eeprom_to_bin"
	SREQ_SET_EEPROM_FROM_BIN      SReqType = "set_eeprom_from_bin"
	SREQ_SET_EEPROM_FROM_BIN_FC   SReqType = "set_eeprom_from_bin_fc"
	SREQ_GET_BASIC_DATA           SReqType = "get_basic_data"     //20240820@FLF
	SREQ_SET_LIMIT_TO_SCALE       SReqType = "set_limit_to_scale" //20240829@FLF
	SREQ_OPEN_BILL_SEND           SReqType = "open_bill_send"     //20240903@FLF
	SREQ_DIS_PASSTH_MODE          SReqType = "dis_passth_mode"    //20240914@FLF
	SREQ_CLOSE_SERIAL_PORT        SReqType = "close_serial_port"  //20241209@FLF  关闭串口
	SREQ_OPEN_SERIAL_PORT         SReqType = "open_serial_port"   //20241209@FLF  打开串口
	SREQ_EXPORT_RECS              SReqType = "export_recs"        //20250218        // with parameter ReqScaleRec
	SREQ_SEND_SCALE_ALIVE         SReqType = "send_scale_alive"   //20250227

	SREQ_CAL_WGT             SReqType = "cal_weight"          //20250529
	SREQ_SEND_CAL_HEART_BEAT SReqType = "send_cal_heart_beat" //20250529
	SREQ_SET_DECIMAL_VALUE   SReqType = "set_decimal_value"   //20250529

	SREQ_SET_MAX_RANGE1       SReqType = "set_max_range1"       //20250716
	SREQ_SET_MAX_RANGE2       SReqType = "set_max_range2"       //20250716
	SREQ_GET_MAX_RANGE1       SReqType = "get_max_range1"       //20250716
	SREQ_GET_MAX_RANGE2       SReqType = "get_max_range2"       //20250716
	SREQ_SET_GADUATION1_VALUE SReqType = "set_gaduation1_value" //20250716
	SREQ_SET_GADUATION2_VALUE SReqType = "set_gaduation2_value" //20250716
	SREQ_GET_GADUATION1_VALUE SReqType = "get_gaduation1_value" //20250716
	SREQ_GET_GADUATION2_VALUE SReqType = "get_gaduation2_value" //20250716
	SREQ_GET_DECIMAL_VALUE    SReqType = "get_decimal_value"    //20250716
	SREQ_SET_SERIAL_PORT      SReqType = "set_serial_port"
	SREQ_GET_SERIAL_PORT      SReqType = "get_serial_port"
	SREQ_SET_WEIGHT_UNIT      SReqType = "set_weight_unit"      //20250716
	SREQ_GET_WEIGHT_UNIT      SReqType = "get_weight_unit"      //20250716
	SREQ_SET_INITIAL_ZERO     SReqType = "set_initial_zero"     //20250716
	SREQ_GET_INITIAL_ZERO     SReqType = "get_initial_zero"     //20250716
	SREQ_SET_MANUAL_ZERO      SReqType = "set_manual_zero"      //20250716
	SREQ_GET_MANUAL_ZERO      SReqType = "get_manual_zero"      //20250716
	SREQ_SET_ZERO_TRACKING    SReqType = "set_zero_tracking"    //20250716
	SREQ_GET_ZERO_TRACKING    SReqType = "get_zero_tracking"    //20250716
	SREQ_SET_GRAV_ACC         SReqType = "set_grav_acc"         //20250716
	SREQ_GET_GRAV_ACC         SReqType = "get_grav_acc"         //20250716
	SREQ_SET_FORCE_UNTARE     SReqType = "force_untare"         //20251104

	SREQ_GET_MODEL       SReqType = "get_model"
	SREQ_EN_CODE         SReqType = "en_code"
	SREQ_DIS_CODE        SReqType = "dis_code"
	SREQ_ASK_ROM_VERSION SReqType = "ask_rom_version"

	SREQ_GET_SEAL_STATUS       SReqType = "get_seal_status"
	SREQ_SOFT_SEAL             SReqType = "soft_seal"
	SREQ_REMOVE_SOFT_SEAL      SReqType = "remove_soft_seal"
	SREQ_REMOVE_SOFT_SEAL_ONCE SReqType = "remove_soft_seal_once"

	SREQ_GET_WIRED_IP   SReqType = "get_wired_ip"
	SREQ_SET_WIRED_IP   SReqType = "set_wired_ip"
	SREQ_SET_WIRED_DHCP SReqType = "set_wired_dhcp"
	SREQ_GET_WIRED_DHCP SReqType = "get_wired_dhcp"

	SREQ_INIT_WIFI SReqType = "init_wifi"
)

type ReqScaleRec struct {
	ScaleId int64
}

type ReqAddScaleRec struct {
	ScaleId int64
	Product string
	Weight  string
	Price   string
}

type ReqDelScaleRec struct {
	RecId uint
}

type ReqPrnData struct {
	ScaleModel   string   `json:"ScaleModel"`
	PrinterModel string   `json:"PrinterModel"`
	FilePaths    []string `json:"FilePaths"`
}

type ReqDefaultPrnData struct {
	ScaleModel   string   `json:"ScaleModel"`
	PrinterModel string   `json:"PrinterModel"`
	FilePaths    []string `json:"FilePath"`
}

type ReqDelPLuData struct {
	ScaleModel string   `json:"ScaleModel"`
	PluId      []string `json:"PluId"`
}

type OlUlInfo struct {
	OlCnt  int
	OlTime int
	UlCnt  int
	UlTime int
}

type ReqPluData struct {
	ScaleModel string `json:"ScaleModel"`
	FilePath   string `json:"FilePath"`
	NameMaxLen int    `json:"NameMaxLen"`
}

type ReqFirmwareInfo struct {
	ModelName         string `json:"modelName"`
	BinKey            string `json:"binKey"`
	SrecKey           string `json:"srecKey"`
	Version           string `json:"version"`
	BootloaderVersion string `json:"bootloaderVersion"`
}

type ReqSerialFileList struct {
	Paths []string `json:"FilePath"`
}

type ScaleRespMsg struct { // including response and unsolicited messages
	MsgType comm.RespMsgType
	MsgBody interface{} // MsgBody [T RespMsg|string]
	ScaleId int64
}

type RespMsg struct {
	IsAck   bool
	AckData string
}

type WeightMsg struct {
	IsZero     bool
	IsStable   bool
	IsNet      bool
	WeightVal  string
	WeightUnit string
}

type RespRecs struct {
	ScaleId int64
	Recs    []ScaleRec
}

type CodeMsg struct {
	IsStable bool
	CodeVal  int64
}
