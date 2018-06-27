// Package rest provides ...
package rest

const (
	// generic action const
	GM_GET = 1 << iota
	GM_LIST
	GM_POST
	GM_DELETE
	GM_PATCH
	GM_PUT
	GM_HEAD
	GM_RPT

	GM_ALL  = GM_GET | GM_LIST | GM_POST | GM_DELETE | GM_PATCH | GM_HEAD | GM_PUT
	GM_NONE = 0

	// action
	ACTION_CREATE = "C"
	ACTION_READ   = "R"
	ACTION_UPDATE = "U"
	ACTION_DELETE = "D"
	ACTION_OTHER  = "O"

	// app env config key
	AECK_REDIS_ADDR      = "session.redis.conn"
	AECK_DB              = "rest.db"
	AECK_ES_ADDR         = "rest.esaddr"
	AECK_REPORTING_INDEX = "rest.reportingindex"
	AECK_LOGS_INDEX      = "rest.logsindex"
	// rest config key
	RCK_ES_ADDR         = "addr"
	RCK_ES_USER         = "user"
	RCK_ES_PWD          = "password"
	RCK_REPORTING_INDEX = "reporting_index"
	RCK_LOGS_INDEX      = "logs_index"
	//env key
	RESTKey           = "_rest_"
	RequestIDKey      = "_reqid_"
	SaveBodyKey       = "_sb_"
	NoLogKey          = "_nl_"
	PaginationKey     = "_pagination_"
	FieldsKey         = "_fields_"
	TimeRangeKey      = "_tr_"
	OrderByKey        = "_ob_"
	ConditionsKey     = "_conditions_"
	LogPrefixKey      = "_prefix_"
	EndpointKey       = "_endpoint_"
	RowkeyKey         = "_rk_"
	RptKey            = "_rpt_"
	SelectorKey       = "_selector_"
	MimeTypeKey       = "_mimetype_"
	DispositionMTKey  = "_dmt_"
	ContentMD5Key     = "_md5_"
	DispositionPrefix = "_dp_"
	DIMENSION_KEY     = "_dimension_"
	SIDE_KEY          = "_sidekey_"
	USERID_KEY        = "_userid_"
	SESSION_KEY       = "_session_"
	APPID_KEY         = "_appid_"
	STAG_KEY          = "_stag_"
	PERMISSION_KEY    = "_perm_"
	EXT_KEY           = "_ext_"
	SKIPAUTH_KEY      = "_skipauth_"
	LimitAccessKey    = "_limitaccess_"
	CustomActionKey   = "_customaction_"
	DescKey           = "_desc_"

	// db tag
	DBTAG_PK    = "pk"
	DBTAG_LOGIC = "logic"

	DBTAG           string = "db"
	READTAG         string = "read"
	WRITETAG        string = "write"
	base            string = "0000-00-00 00:00:00.0000000"
	timeFormat      string = "2006-01-02 15:04:05.999999"
	timeISOFormat29 string = "2006-01-02T15:04:05.999Z07:00" // length: 29
	timeISOFormat25 string = "2006-01-02T15:04:05Z07:00"     // length: 25

	//tag
	TAG_REQUIRED    = "R"     // 必填
	TAG_GENERATE    = "G"     // 服务端生成, 同时不可编辑
	TAG_CONDITION   = "C"     // 可作为查询条件
	TAG_DENY        = "D"     // 不可编辑, 可为空
	TAG_SECRET      = "S"     // 保密,一般不见人
	TAG_HIDDEN      = "H"     // 隐藏
	TAG_DEFAULT     = "DEF"   // 默认
	TAG_TIMERANGE   = "TR"    // 时间范围条件
	TAG_REPORT      = "RPT"   // 报表字段
	TAG_CANGROUP    = "GRP"   // 可以group操作
	TAG_ORDERBY     = "O"     // 可排序(默认DESC)
	TAG_AORDERBY    = "AO"    // 正排序(默认DESC)
	TAG_RETURN      = "RET"   // 返回,创建后需要返回数值
	TAG_SUM         = "SUM"   // 求和
	TAG_TSUM        = "TS"    // 总求和(放到聚合中,只能有一个)
	TAG_COUNT       = "COUNT" // 计数
	TAG_AGGREGATION = "AGG"   // 聚合

	// ext field
	EXF_SUM   = "sum"
	EXF_COUNT = "count"

	_DEF_PAGE     = 1 //1-base
	_DEF_PER_PAGE = 100
	_MAX_PER_PAGE = 1000 //每页最大个数

	//time
	_DATE_FORM  = "2006-01-02"
	_DATE_FORM1 = "20060102"
	_DATE_FORM2 = "2006010215"
	_DATE_FORM3 = "200601021504"
	_DATE_FORM4 = "20060102150405"
	_TIME_FORM  = "20060102150405"
	_MYSQL_FORM = "2006-01-02 15:04:05"

	//固定参数名称
	PARAM_FIELDS  = "fields"
	PARAM_PAGE    = "page"
	PARAM_PERPAGE = "per_page"
	PARAM_DATE    = "date"
	PARAM_START   = "start"
	PARAM_END     = "end"
	PARAM_ORDERBY = "orderby"

	PARAM_RANGEBY = "range_by"
	PARAM_ALLTIME = "all_time"
	PARAM_ALLDATA = "all_data"

	//特殊前缀
	_PPREFIX_NOT  = '!'
	_PPREFIX_LIKE = '~'
	_PPREFIX_GT   = '>'
	_PPREFIX_LT   = '<'

	// 查询类型
	CTYPE_IS = iota
	CTYPE_NOT
	CTYPE_OR
	CTYPE_LIKE
	CTYPE_GT
	CTYPE_LT
	CTYPE_JOIN
	CTYPE_RANGE
	CTYPE_ORDER
	CTYPE_PAGE
	CTYPE_RAW

	// report
	FIELD_TAG = "json"
	RPT_TAG   = "report"

	RPT_NESTED = "nested"
	RPT_TERM   = "term"
	RPT_SUM    = "sum"
	RPT_SEARCH = "search"
	RPT_FILTER = "filter"
	RPT_RANGE  = "range"

	RTKEY_RESULTS = "_results_"
	RTKEY_COUNT   = "_count_"
	RTKEY_NAME    = "_name_"
	RTKEY_HITS    = "_hits_"
	RTKEY_START   = "_start_"
	RTKEY_END     = "_end_"
	RTKEY_DAILY   = "_daily_"
	RTKEY_INTVL   = "_interval_"
	RTKEY_DATE    = "_date_"

	INTVL_HOUR    = "hour"
	INTVL_DAY     = "day"
	INTVL_WEEK    = "week"
	INTVL_MONTH   = "month"
	INTVL_QUARTER = "quarter"
	INTVL_YEAR    = "year"
)
