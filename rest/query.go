package rest

import (
	"strconv"
	"strings"
	"time"

	"wgo"
	"wgo/utils"
)

//时间段
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// 分页信息
type Pagination struct {
	Page    int
	PerPage int
	Offset  int
}

// get ctype
func getCTypeByPrefix(p byte) int {
	switch p {
	case _PPREFIX_NOT:
		return CTYPE_NOT
	case _PPREFIX_LIKE:
		return CTYPE_LIKE
	case _PPREFIX_GT:
		return CTYPE_GT
	case _PPREFIX_LT:
		return CTYPE_LT
	default:
		return CTYPE_IS
	}
}

// 条件信息
/* {{{ func NewPagation(page, perPage string) (p *Pagination)
 */
func NewPagination(page, perPage string) (p *Pagination) {
	var pageNum, offset, perNum int
	if page == "" {
		pageNum = _DEF_PAGE
	} else {
		pageNum, _ = strconv.Atoi(page)
		if pageNum < 1 {
			pageNum = _DEF_PAGE
		}
	}
	if perPage == "" {
		perNum = _DEF_PER_PAGE
	} else {
		perNum, _ = strconv.Atoi(perPage)
		if perNum > _MAX_PER_PAGE {
			perNum = _MAX_PER_PAGE
		}
	}
	offset = (pageNum - 1) * perNum
	p = &Pagination{
		Page:    pageNum,
		PerPage: perNum,
		Offset:  offset,
	}
	return
}

/* }}} */

/* {{{ func (r *REST) GetParamConds() []*Condition
 * 获取参数条件
 */
func (r *REST) GetParamConds() []*Condition {
	csI := r.GetEnv(ConditionsKey)
	if csI != nil {
		if conds, ok := csI.([]*Condition); ok {
			return conds
		}
	}
	return make([]*Condition, 0)
}

/* }}} */

/* {{{ func (r *REST) SetParamConds(con *Condition) (err error) {
 * 设置参数条件
 */
func (r *REST) SetParamConds(con *Condition) {
	r.Debug("[SetParamConds][key: %s]%v", con.Field, con)
	var conds []*Condition
	csI := r.GetEnv(ConditionsKey)
	if csI != nil {
		if cds, ok := csI.([]*Condition); ok {
			conds = cds
		}
	}
	if conds == nil {
		conds = make([]*Condition, 0)
	}
	set := false
	for _, ec := range conds {
		if ec.Field == con.Field {
			ec.Merge(con)
			set = true
		}
	}
	if !set {
		conds = append(conds, con)
	}
	r.SetEnv(ConditionsKey, conds)
}

/* }}} */

/* {{{ func getTimeRange(s, e string) *TimeRange
 * 根据start/end字符串获得时间段
 */
func getTimeRange(s, e string) *TimeRange {
	tr := new(TimeRange)

	var format string
	var step time.Duration
	switch l := len(s); l {
	case len(_DATE_FORM): // len(_DATE_FORM2) == len(_DATE_FORM)
		if i := strings.Index(s, "-"); i > 0 {
			format = _DATE_FORM
			step = time.Hour*24 - 1 // 精确到纳秒
		} else {
			format = _DATE_FORM2
			step = time.Hour*1 - 1
		}
	case len(_DATE_FORM1):
		format = _DATE_FORM1
		step = time.Hour*24 - 1
	case len(_DATE_FORM3):
		format = _DATE_FORM3
		step = time.Minute*1 - 1
	case len(_DATE_FORM4):
		format = _DATE_FORM4
		step = time.Second*1 - 1
	}
	if ts, err := time.ParseInLocation(format, s, wgo.Env().Location); err == nil {
		//Info("location: %v, ok", wgo.Env().Location)
		tr.Start = ts
		if e != "" {
			tr.End = ts.Add(step) //默认结束时间为开始时间加上步长
			//只有成功获取了start, end才有意义
			if te, err := time.ParseInLocation(format, e, wgo.Env().Location); err == nil && te.After(ts) {
				// end 必须比 start 大
				tr.End = te.Add(step)
			}
		} else {
			// 没有传入end字符串, 默认为今天
			y, m, d := time.Now().In(wgo.Env().Location).Date()
			tr.End = time.Date(y, m, d, 23, 59, 59, 999000000, wgo.Env().Location)
		}
	}

	return tr
}

/* }}} */

/* {{{ func (r *REST) setTimeRangeFromDate(p []string) {
 * 时间段信息
 */
func (r *REST) setTimeRangeFromDate(p []string) {
	var s, e string

	if len(p) > 1 { //有多个,第一个是start, 第二个是end, 其余忽略
		s, e = p[0], p[1]
	} else if len(p) > 0 { //只有一个, 可通过 "{start},{end}"方式传
		pieces := strings.SplitN(p[0], ",", 2)
		s = pieces[0]
		if len(pieces) > 1 {
			e = pieces[1]
		}
	}
	r.SetEnv(TimeRangeKey, getTimeRange(s, e))

	return
}

/* }}} */

/* {{{ func (r *REST) SetUserID(opts ...interface{})
 * 设置UserID, 可以覆盖默认cookie中的user id
 */
func (r *REST) SetUserID(opts ...interface{}) {
	if uid := utils.PrimaryString(opts); uid != "" {
		r.SetEnv(USERID_KEY, uid)
	} else if uid = r.Context().UserID(); uid != "" {
		r.SetEnv(USERID_KEY, uid)
	}
}

/* }}} */

/* {{{ func (r *REST) GetUserID() string
 * 获取user id
 */
func (r *REST) GetUserID() string {
	// 从session中获取
	if id, _ := r.GetEnv(USERID_KEY).(string); id != "" {
		return id
	}
	return "0"
}

/* }}} */

/* {{{ func (r *REST) setTimeRangeFromStartEnd() {
 * 时间段信息
 */
func (r *REST) setTimeRangeFromStartEnd() {
	var s, e string
	c := r.Context()
	if s = c.QueryParam(PARAM_START); s == "" {
		//没有传入start,do nothing
		return
	}

	if e = c.QueryParam(PARAM_END); e == "" {
		// 没有传入end, 默认为当天
		// return
		c.Warn("[setTimeRangeFromStartEnd]not found end date, set to today")
	}

	if s != "" && e != "" && len(s) != len(e) {
		//长度不一致,返回
		return
	}

	r.SetEnv(TimeRangeKey, getTimeRange(s, e))

	return
}

/* }}} */

/* {{{ func (r *REST) setOrderBy(p string) {
 * 时间段信息
 */
func (r *REST) setOrderBy(p []string) {
	ob := new(OrderBy)
	r.SetEnv(OrderByKey, ob)
	if len(p) > 0 { //只有一个, 可通过 "{start},{end}"方式传
		pieces := strings.SplitN(p[0], ",", 2)
		ob.Field = pieces[0]
		ob.Sort = "DESC" //默认降序
		if len(pieces) > 1 && strings.ToUpper(pieces[1]) == "ASC" {
			ob.Sort = "ASC"
		}
		Debug("[orderby][field: %s][sort: %s]", ob.Field, ob.Sort)
	}

	return
}

/* }}} */

/* {{{ func ParseCondition(typ string, con *Condition) *Condition
 *
 */
func ParseCondition(typ string, con *Condition) *Condition {
	switch typ {
	case "*time.Time":
		if con.Is != nil {
			if cv, ok := con.Is.(string); ok {
				if t, err := time.ParseInLocation(_TIME_FORM, cv, wgo.Env().Location); err == nil {
					con.Is = t
				}
			}
		}
		if con.Not != nil {
			if cv, ok := con.Not.(string); ok {
				if t, err := time.ParseInLocation(_TIME_FORM, cv, wgo.Env().Location); err == nil {
					con.Not = t
				}
			}
		}
		if con.Gt != nil {
			if cv, ok := con.Gt.(string); ok {
				if t, err := time.ParseInLocation(_TIME_FORM, cv, wgo.Env().Location); err == nil {
					con.Gt = t
				}
			}
		}
		if con.Lt != nil {
			if cv, ok := con.Lt.(string); ok {
				if t, err := time.ParseInLocation(_TIME_FORM, cv, wgo.Env().Location); err == nil {
					con.Lt = t
				}
			}
		}
		return con
	default:
		return con
	}
}

/* }}} */

// 过滤空参数
func parseParams(v []string) []string {
	rv := make([]string, 0)
	for _, vs := range v {
		if vs != "" {
			rv = append(rv, vs)
		}
	}
	return rv
}
