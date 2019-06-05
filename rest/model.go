package rest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"

	"wgo"
	"wgo/gorp"
	"wgo/utils"
)

type Model interface {
	Keeper() func(string) (interface{}, error) // 各种检查, 闭包缓存
	KeeperFactory() func(string) (interface{}, error)

	// sql sugar
	Is(string, ...interface{}) Model
	Not(string, interface{}) Model
	Or(string, interface{}) Model
	Like(string, interface{}) Model
	Gt(string, interface{}) Model
	Lt(string, interface{}) Model
	Range(string, interface{}) Model
	Join(string, interface{}, ...interface{}) Model
	Order(string, interface{}) Model
	Raw(string, interface{}) Model

	SetConditions(...*Condition) Model
	Conditions() []*Condition
	SetPagination(p *Pagination) Model
	Pagination() *Pagination
	SetFields(...string) Model
	Fields() []string
	NewList() interface{} // 返回一个空结构列表
	AddTable(...string) Model
	ImportDic(string, ChecklistDic)
	DBConn(string) *gorp.DbMap                         // 数据库连接
	Transaction(...interface{}) (*Transaction, error)  // transaction
	TableName() string                                 // 返回表名称, 默认结构type名字(小写), 有特别的表名称,则自己implement 这个方法
	PKey() (string, string, bool)                      // primary key字段,以及是否auto incr
	Key() (string, string, bool)                       // key字段 name&value
	UnionKeys(...interface{}) map[string]string        // union keys, name&value
	ReadPrepare(...interface{}) (*gorp.Builder, error) // 组条件
	Row(...interface{}) (Model, error)                 //获取单条记录
	Rows(...interface{}) (interface{}, error)          //获取多条记录
	List() (*List, error)                              // 获取多条记录并返回list
	GetRecord(rk ...interface{}) interface{}           // 获取一条记录, 可缓存
	UpdateRecord(...interface{}) error                 // 更新一条记录(包括缓存)
	Write(...interface{}) (Model, error)               // 写记录, 若果不存在创建, 存在则更新
	GetOlder(rk ...string) Model                       //获取旧记录
	GetSum(...string) (*List, error)                   //获取多条记录
	GetCount() (int64, error)                          //获取多条记录
	GetCountNSum() (int64, float64)                    //获取count and sum
	CreateRow() (Model, error)                         //创建单条记录
	UpdateRow(...interface{}) (int64, error)           //更新记录
	DeleteRow(rk string) (int64, error)                //删除记录

	Fill([]byte) error              //填充内容
	Valid(...string) (Model, error) //数据验证, 如果传入opts, 则只验证opts指定的字段
	Filter() (Model, error)         //数据过滤(创建,更新后)
	Protect() (Model, error)        //数据保护(获取数据时过滤字段)
}

type List struct {
	Info ListInfo               `json:"info,omitempty"`
	List interface{}            `json:"list"`
	Ext  map[string]interface{} `json:"ext,omitempty"`
}

type ListInfo struct {
	Page    *int        `json:"page,omitempty"`     //当前页面
	PerPage *int        `json:"per_page,omitempty"` //每页元素个数
	Total   int64       `json:"total"`              // 总数
	Sum     interface{} `json:"sum,omitempty"`      //求和
	Summary interface{} `json:"summary,omitempty"`
}

// build single page list from interface{}
func SinglePageList(i interface{}) *List {
	n := new(int)
	page := new(int)
	*page = 1
	switch reflect.TypeOf(i).Kind() {
	case reflect.Slice:
		*n = reflect.ValueOf(i).Len()
		return &List{
			Info: ListInfo{
				Total:   int64(*n),
				Page:    page,
				PerPage: n,
			},
			List: i,
		}
	default:
		*n = 1
		return &List{
			Info: ListInfo{
				Total:   int64(*n),
				Page:    page,
				PerPage: n,
			},
			List: []interface{}{i},
		}
	}
}

//错误代码
var (
	ErrRequired      = errors.New("field is required")
	ErrNonEditable   = errors.New("field is non-editable")
	ErrNonSearchable = errors.New("field is non-searchable")
	ErrExists        = errors.New("field value exists")
	ErrNoCondition   = errors.New("no condition")
	ErrInvalid       = errors.New("invalid query")
	ErrType          = errors.New("wrong type")
	ErrNoRecord      = errors.New("no record")
	ErrNoModel       = errors.New("no model")
)

type Condition struct {
	Table  string
	Field  string
	Is     interface{}
	Not    interface{}
	Or     interface{}
	Gt     interface{}
	Lt     interface{}
	Like   interface{}
	Join   interface{}
	JoinOn []interface{}
	Range  interface{} //范围条件, btween ? and ?
	Order  interface{}
	Raw    string //原始字符串
}

//order by
type OrderBy struct {
	Field string
	Sort  string
}

func NewCondition(typ int, field string, cs ...interface{}) *Condition {
	if field == "" || len(cs) < 1 { //至少1个元素
		return nil
	}
	con := &Condition{Field: field}
	var v interface{}
	if len(cs) == 1 {
		v = cs[0]
	} else {
		v = cs
	}
	switch typ {
	case CTYPE_IS:
		con.Is = v
	case CTYPE_NOT:
		con.Not = v
	case CTYPE_GT:
		con.Gt = v
	case CTYPE_LT:
		con.Lt = v
	case CTYPE_JOIN:
		con.Join = cs[0]
		if len(cs) >= 3 {
			con.JoinOn = cs[1:]
		}
	case CTYPE_OR:
		con.Or = v
	case CTYPE_LIKE:
		con.Like = v
	case CTYPE_RANGE:
		con.Range = v
	case CTYPE_ORDER:
		con.Order = v
	case CTYPE_RAW:
		con.Raw = v.(string)
	default:
	}
	return con
}

func buildWhereRaw(b *gorp.Builder, tableAlias, field string, con interface{}) {
	if con == nil {
		return
	}
	switch vt := con.(type) {
	case *string:
		b.Where(fmt.Sprintf("%s.`%s` = ?", tableAlias, field), *vt)
	case *int:
		b.Where(fmt.Sprintf("%s.`%s` = ?", tableAlias, field), *vt)
	case *int64:
		b.Where(fmt.Sprintf("%s.`%s` = ?", tableAlias, field), *vt)
	case *float64:
		b.Where(fmt.Sprintf("%s.`%s` = ?", tableAlias, field), *vt)
	case string, int, int64, float64:
		b.Where(fmt.Sprintf("%s.`%s` = ?", tableAlias, field), vt)
	case []string:
		vs := bytes.Buffer{}
		first := true
		vs.WriteString("(")
		for _, vv := range vt {
			if !first {
				vs.WriteString(",")
			}
			vs.WriteString(fmt.Sprintf("'%s'", vv))
			first = false
		}
		vs.WriteString(")")
		b.Where(fmt.Sprintf("%s.`%s` IN %s", tableAlias, field, vs.String()))
	case []int:
		vs := bytes.Buffer{}
		first := true
		vs.WriteString("(")
		for _, vv := range vt {
			if !first {
				vs.WriteString(",")
			}
			vs.WriteString(fmt.Sprintf("'%d'", vv))
			first = false
		}
		vs.WriteString(")")
		b.Where(fmt.Sprintf("%s.`%s` IN %s", tableAlias, field, vs.String()))
	case []interface{}:
		vs := bytes.Buffer{}
		first := true
		vs.WriteString("(")
		for _, vv := range vt {
			if !first {
				vs.WriteString(",")
			}
			vs.WriteString(fmt.Sprint("'", vv, "'"))
			first = false
		}
		vs.WriteString(")")
		b.Where(fmt.Sprintf("%s.`%s` IN %s", tableAlias, field, vs.String()))
	default:
	}
}

/* {{{ func (v *Condition) DoWhere(b *gorp.Builder)
* 只负责生成部分sql, IS/NOT/LIKE/GT/LT
 */
func (v *Condition) DoWhere(b *gorp.Builder) {
	if v.Raw != "" {
		b.Where(fmt.Sprint("(", v.Raw, ")"))
	}
	buildWhereRaw(b, "T", v.Field, v.Is)
	buildWhereRaw(b, "T", v.Field, v.Not)
	if v.Not != nil {
		switch vt := v.Not.(type) {
		case string:
			b.Where(fmt.Sprintf("T.`%s` != ?", v.Field), vt)
		case []string:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(",")
				}
				vs.WriteString(fmt.Sprintf("'%s'", vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(fmt.Sprintf("T.`%s` NOT IN %s", v.Field, vs.String()))
		case []interface{}:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(",")
				}
				vs.WriteString(fmt.Sprintf("'%s'", vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(fmt.Sprintf("T.`%s` NOT IN %s", v.Field, vs.String()))
		default:
		}
	}
	if v.Gt != nil {
		//Debug("[>=][key: %s]%v", v.Field, v)
		switch vt := v.Gt.(type) {
		case string:
			b.Where(fmt.Sprintf("T.`%s` >= ?", v.Field), vt)
		case []string:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(" OR ")
				}
				vs.WriteString(fmt.Sprintf("T.`%s` >= '%s'", v.Field, vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(vs.String())
		case *TimeRange:
			b.Where(fmt.Sprintf("T.`%s` >= ?", v.Field), vt.Start)
		case TimeRange:
			b.Where(fmt.Sprintf("T.`%s` >= ?", v.Field), vt.Start)
		case []interface{}:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(" OR ")
				}
				vs.WriteString(fmt.Sprintf("T.`%s` >= '%s'", v.Field, vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(vs.String())
		default:
		}
	}
	if v.Lt != nil {
		//Debug("[<][key: %s]%v", v.Field, v)
		switch vt := v.Lt.(type) {
		case string:
			b.Where(fmt.Sprintf("T.`%s` < ?", v.Field), vt)
		case []string:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(" OR ")
				}
				vs.WriteString(fmt.Sprintf("T.`%s` < '%s'", v.Field, vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(vs.String())
		case *time.Time:
			b.Where(fmt.Sprintf("T.`%s` < ?", v.Field), vt)
		case time.Time:
			b.Where(fmt.Sprintf("T.`%s` < ?", v.Field), vt)
		case []interface{}:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(" OR ")
				}
				vs.WriteString(fmt.Sprintf("T.`%s` < '%s'", v.Field, vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(vs.String())
		default:
		}
	}
	if v.Like != nil {
		switch vt := v.Like.(type) {
		case string:
			b.Where(fmt.Sprintf("T.`%s` LIKE ?", v.Field), fmt.Sprintf("%%%s%%", vt))
		case []string:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(" OR ")
				}
				vs.WriteString(fmt.Sprintf("T.`%s` LIKE '%%%s%%'", v.Field, vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(vs.String())
		case []interface{}:
			vs := bytes.Buffer{}
			first := true
			vs.WriteString("(")
			for _, vv := range vt {
				if !first {
					vs.WriteString(" OR ")
				}
				vs.WriteString(fmt.Sprintf("T.`%s` LIKE '%%%s%%'", v.Field, vv))
				first = false
			}
			vs.WriteString(")")
			b.Where(vs.String())
		default:
		}
	}
}

/* }}} */

/* {{{ func (con *Condition) Merge(oc *Condition)
* 直接覆盖
 */
func (con *Condition) Merge(oc *Condition) {
	if oc == nil {
		return
	}
	if oc.Is != nil {
		con.Is = oc.Is
	}
	if oc.Or != nil {
		con.Or = oc.Or
	}
	if oc.Not != nil {
		con.Not = oc.Not
	}
	if oc.Gt != nil {
		con.Gt = oc.Gt
	}
	if oc.Lt != nil {
		con.Lt = oc.Lt
	}
	if oc.Like != nil {
		con.Like = oc.Like
	}
	if oc.Range != nil {
		con.Range = oc.Range
	}
	if oc.Order != nil {
		con.Order = oc.Order
	}
	if oc.Join != nil {
		con.Join = oc.Join
		con.JoinOn = oc.JoinOn
	}
	if oc.Raw != "" {
		con.Raw = oc.Raw
	}
}

/* }}} */

// new model from context
func ModelFromContext(c *wgo.Context, i interface{}) Model {
	return GetREST(c).NewModel(i)
}

// 基于类型创建一个全新的model, i会被置为空
func NewModel(i interface{}) Model {
	r := new(REST)
	return r.NewModel(i)
}

// 基于变量创建全新的model,  i的值保留
func SetModel(i interface{}) Model {
	r := new(REST)
	return r.SetModel(i.(Model))
}

// 创建一个跟r有关的model
func (r *REST) NewModel(i interface{}) Model {
	//m := reflect.New(reflect.Indirect(reflect.ValueOf(i)).Type()).Interface().(Model)
	m := reflect.New(reflect.TypeOf(i).Elem()).Interface().(Model)
	return r.SetModel(m)
}

// SetModel
func (r *REST) SetModel(m Model) Model {
	r.model = m
	// 注入m
	r.importTo(m)
	return m
}

// 创造一个全新的model, 并传递context
func (r *REST) GenModel(m Model) Model {
	nr := new(REST)
	nr.setContext(r.Context())
	return nr.NewModel(m)
}

func (r *REST) Modelize(m Model) Model {
	nr := new(REST)
	nr.setContext(r.Context())
	return nr.SetModel(m)
}

// 从rest创建一个全新的model, 不需要传参,因为类型已经知道
// return a new instance
func (r *REST) New() Model {
	if m := r.Model(); m != nil {
		//return reflect.New(reflect.Indirect(reflect.ValueOf(m)).Type()).Interface().(Model)
		return NewModel(m)
	}
	return nil
}

// 把rest注入i
func (r *REST) importTo(i interface{}, fields ...string) {
	field := "REST"
	if len(fields) > 0 {
		field = fields[0]
	}
	if fv := utils.FieldByName(i, field); fv.IsValid() {
		if fv.Kind() == reflect.Ptr {
			fv.Set(reflect.ValueOf(r))
		} else {
			fv.Set(reflect.ValueOf(r).Elem())
		}
	}
}

// Model
func (r *REST) Model() Model {
	if r == nil {
		return nil
	}
	return r.model
}

/* {{{ func GetCondition(cs []*Condition, k string) (con *Condition, err error)
*
 */
func GetCondition(cs []*Condition, k string) (con *Condition, err error) {
	if cs == nil || len(cs) == 0 {
		err = fmt.Errorf("conditions empty")
	} else {
		for _, c := range cs {
			//Debug("field: %s, key: %s", c.Field, k)
			if c != nil && c.Field == k {
				return c, nil
			}
		}
	}
	return nil, fmt.Errorf("cannot found condition: %s", k)
}

/* }}} */

// sugar
func (r *REST) Is(field string, value ...interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_IS, field, value...))
}
func (r *REST) Not(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_NOT, field, value))
}
func (r *REST) Or(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_OR, field, value))
}
func (r *REST) Like(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_LIKE, field, value))
}
func (r *REST) Gt(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_GT, field, value))
}
func (r *REST) Lt(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_LT, field, value))
}
func (r *REST) Range(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_RANGE, field, value))
}
func (r *REST) Join(field string, value interface{}, opts ...interface{}) Model {
	js := strings.SplitN(field, ".", 2)
	// join的field一定是 `table.field`
	if js[0] != "" && js[1] != "" {
		vs := []interface{}{NewCondition(CTYPE_IS, js[1], value)}
		return r.SetConditions(NewCondition(CTYPE_JOIN, js[0], append(vs, opts...)...))
	}
	return r
}
func (r *REST) Order(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_ORDER, field, value))
}
func (r *REST) Raw(field string, value interface{}) Model {
	return r.SetConditions(NewCondition(CTYPE_RAW, field, value))
}

/* {{{ func (r *REST) SetConditions(cs ...*Condition) Model
* 设置条件
 */
func (r *REST) SetConditions(cs ...*Condition) Model {
	if r.conditions == nil {
		r.conditions = make([]*Condition, 0)
	}
	if m := r.Model(); m == nil {
		Warn("[r.SetConditions]: %s", ErrNoModel)
	} else if cols := utils.ReadStructColumns(m, true); cols != nil {
		for _, col := range cols {
			// Debug("[SetConditions][tag: %s][ext: %s][type: %s]", col.Tag, col.ExtTag, col.Type.String())
			// join
			if condition, e := GetCondition(cs, col.ExtTag); e == nil && condition.Join != nil {
				// Debug("[SetConditions][join][table: %s]%v", col.ExtTag, condition)
				r.conditions = append(r.conditions, condition)
			}
			// raw
			if condition, e := GetCondition(cs, col.Tag); e == nil && condition.Raw != "" {
				//Debug("[SetConditions][raw][tag: %s]%v", col.Tag, condition)
				r.conditions = append(r.conditions, condition)
			}
			// time range
			if col.ExtOptions.Contains(TAG_TIMERANGE) {
				if condition, e := GetCondition(cs, col.Tag); e == nil && (condition.Range != nil || condition.Is != nil) {
					// 直接对字段查询
					Debug("[SetConditions]timerange: %+v, %+v, %+v", col.Tag, condition.Is, condition.Range)
					if condition.Range != nil {
						r.conditions = append(r.conditions, condition)
					} else if condition.Is != nil {
						if is, ok := condition.Is.([]string); ok && len(is) > 1 {
							condition.Is = nil
							condition.Range = getTimeRange(is[0], is[1])
							r.conditions = append(r.conditions, condition)
						}
					}
				} else if condition, e := GetCondition(cs, TAG_TIMERANGE); e == nil && condition.Is != nil {
					condition.Field = col.Tag
					r.conditions = append(r.conditions, condition)
				} else {
					//Info("get condition failed: %s", e)
				}
			}
			if col.ExtOptions.Contains(TAG_ORDERBY) {
				if condition, e := GetCondition(cs, TAG_ORDERBY); e == nil && condition.Order != nil {
					//Debug("[SetConditions]order")
					condition.Field = col.Tag
					r.conditions = append(r.conditions, condition)
				} else {
					//Trace("get condition failed: %s", e)
				}
			}
			if col.TagOptions.Contains(DBTAG_PK) || col.TagOptions.Contains(DBTAG_UK) || col.TagOptions.Contains(DBTAG_KEY) || col.ExtOptions.Contains(TAG_CONDITION) { //primary key or union key or conditional
				if condition, e := GetCondition(cs, col.Tag); e == nil && (condition.Is != nil || condition.Not != nil || condition.Gt != nil || condition.Lt != nil || condition.Like != nil || condition.Join != nil || condition.Or != nil) {
					Debug("[SetConditions][tag: %s][type: %s]%v", col.Tag, col.Type.String(), condition)
					r.conditions = append(r.conditions, ParseCondition(col.Type.String(), condition))
				}
			}
		}
	}
	return r
}

/* }}} */

/* {{{ func (r *REST) Conditions() []*Condition
*
 */
func (r *REST) Conditions() []*Condition {
	return r.conditions
}

/* }}} */

/* {{{ func (r *REST) SetPagination(p *Pagination) Model
* 生成条件
 */
func (r *REST) SetPagination(p *Pagination) Model {
	r.pagination = p
	return r
}

/* }}} */

/* {{{ func (r *REST) Pagination() *Pagination
*
 */
func (r *REST) Pagination() *Pagination {
	return r.pagination
}

/* }}} */

/* {{{ func (r *REST) SetFields(fs ...string) Model
* 生成条件
 */
func (r *REST) SetFields(fs ...string) Model {
	if r.fields == nil {
		r.fields = make([]string, 0)
	}
	r.fields = fs
	return r
}

/* }}} */

/* {{{ func (r *REST) Fields() []string
*
 */
func (r *REST) Fields() []string {
	return r.fields
}

/* }}} */

/* {{{ func (r *REST) Keeper() func(string) (interface{}, error)
*
 */
func (r *REST) Keeper() func(string) (interface{}, error) {
	if r.keeper == nil && r.Model() != nil {
		r.keeper = r.Model().KeeperFactory()
	}
	return r.keeper
}

/* }}} */

/* {{{ func (r *REST) NewList() *[]Model
*
 */
func (r *REST) NewList() interface{} {
	if m := r.Model(); m == nil {
		Warn("[NewList]: %s", ErrNoModel)
		return nil
	} else {
		return reflect.New(reflect.SliceOf(reflect.TypeOf(m))).Interface()
	}
}

/* }}} */

/* {{{ func (r *REST) DBConn(tag string) *gorp.DbMap
* 默认数据库连接为admin
 */
func (r *REST) DBConn(tag string) *gorp.DbMap {
	tb := r.TableName()
	if dt, ok := DataAccessor[tb+"::"+tag]; ok && dt != "" {
		return gorp.Using(dt)
	}
	return gorp.Using(DBTAG)
}

/* }}} */

/* {{{ func (r *REST) Transaction(...ineterface{}) (*Transaction, error)
* 获取transaction
 */
func (r *REST) Transaction(opts ...interface{}) (*Transaction, error) {
	if r == nil {
		return nil, fmt.Errorf("not rest model")
	}
	if r.transaction != nil && !r.transaction.Committed() {
		// auto gen savepoint for this sub transaction
		sp := utils.NewShortUUID()
		r.transaction.Savepoint(sp)
		return r.transaction, nil
	}
	// 可以传入一个Transaction来继承
	if trans, ok := utils.NewParams(opts).ItfByIndex(0).(*Transaction); ok && trans != nil && !trans.Committed() {
		sp := utils.NewShortUUID()
		trans.Savepoint(sp)
		r.transaction = trans
		return r.transaction, nil
	}
	trans, err := r.DBConn(WRITETAG).Begin()
	if err != nil {
		return nil, err
	}
	r.transaction = &Transaction{
		Transaction: trans,
		savepoints:  make([]string, 0),
	}
	return r.transaction, nil
}

/* }}} */

/* {{{ func (r *REST) TableName() (n string)
* 获取表名称, 默认为结构名
 */
func (r *REST) TableName() (n string) { //默认, struct的名字就是表名, 如果不是请在各自的model里定义
	if m := r.Model(); m == nil {
		Info("[TableName]error: not found model")
	} else {
		reflectVal := reflect.ValueOf(m)
		mt := reflect.Indirect(reflectVal).Type()
		n = underscore(strings.TrimSuffix(mt.Name(), "Table"))
	}
	return
}

/* }}} */

/* {{{ func (r *REST) PKey() (string, string, bool)
*  通过配置找到pk
 */
func (r *REST) PKey() (f string, v string, ai bool) {
	m := r.Model()
	if m == nil {
		Warn("[PKey]: %s", ErrNoModel)
		return "", "", false
	}
	mv := reflect.ValueOf(m)
	if cols := utils.ReadStructColumns(m, true); cols != nil {
		for _, col := range cols {
			// check required field
			if col.TagOptions.Contains(DBTAG_PK) {
				f = col.Tag
				fv := utils.FieldByIndex(mv, col.Index)
				// Info("field: %s, value: %+v", f, fv)
				v = ""
				if fv.IsValid() && !utils.IsEmptyValue(fv) {
					switch fv.Type().String() {
					case "*string":
						v = fv.Elem().String()
					case "string":
						v = fv.String()
					case "*int":
						v = strconv.Itoa(int(fv.Elem().Int()))
					case "int":
						v = strconv.Itoa(int(fv.Int()))
					case "*int64":
						v = strconv.Itoa(int(fv.Elem().Int()))
					case "int64":
						v = strconv.Itoa(int(fv.Int()))
					default:
						// nothing
					}
				}
				// Info("field: %s, value: %+v, v: %s", f, fv, v)
				if col.TagOptions.Contains(DBTAG_NA) || (col.ExtOptions.Contains(TAG_GENERATE) && col.ExtTag != "") { //服务端生成并且有tag
					ai = false
				} else {
					ai = true
				}
				return
			}
		}
	}
	return
}

/* }}} */

/* {{{ func (r *REST) Key() (string, string, bool)
*  通过配置找到第一个有值的pk or key,  返回field & value & 是否pk
 */
func (r *REST) Key() (f string, v string, isPK bool) {
	m := r.Model()
	if m == nil {
		Warn("[Key]: %s", ErrNoModel)
		return "", "", false
	}
	mv := reflect.ValueOf(m)
	if cols := utils.ReadStructColumns(m, true); cols != nil {
		for _, col := range cols {
			fv := utils.FieldByIndex(mv, col.Index)
			if fv.IsValid() && !utils.IsEmptyValue(fv) {
				if col.TagOptions.Contains(DBTAG_PK) || col.TagOptions.Contains(DBTAG_KEY) {
					f = col.Tag
					// Debug("field: %s, value: %+v", f, fv)
					if col.TagOptions.Contains(DBTAG_PK) {
						isPK = true
					}
					switch fv.Type().String() {
					case "*string":
						v = fv.Elem().String()
					case "string":
						v = fv.String()
					case "*int", "*int64":
						v = strconv.Itoa(int(fv.Elem().Int()))
					case "int", "int64":
						v = strconv.Itoa(int(fv.Int()))
					default:
						// nothing
					}
					return
				}
			}
		}
	}
	return
}

/* }}} */

/* {{{ func (r *REST) UnionKeys(...interface{}) map[string]string
 *  通过配置找到union keys, 返回field => value 的 map
 */
func (r *REST) UnionKeys(opts ...interface{}) (uks map[string]string) {
	m := r.Model()
	if m == nil {
		Warn("[UnionKeys]: %s", ErrNoModel)
		return
	}

	withValue := utils.NewParams(opts).BoolByIndex(0, true) //  是否必须有值, 默认为true

	mv := reflect.ValueOf(m)
	if cols := utils.ReadStructColumns(m, true); cols != nil {
		tmp := make(map[string]string)
		cnt := 0
		for _, col := range cols {
			fv := utils.FieldByIndex(mv, col.Index)
			if col.TagOptions.Contains(DBTAG_UK) {
				f := col.Tag
				if !withValue {
					// 非必须有值, 默认空字符串
					tmp[f] = ""
				}
				cnt++
				if fv.IsValid() && !utils.IsEmptyValue(fv) {
					v := ""
					switch fv.Type().String() {
					case "*string":
						v = fv.Elem().String()
					case "string":
						v = fv.String()
					case "*int", "*int64":
						v = strconv.Itoa(int(fv.Elem().Int()))
					case "int", "int64":
						v = strconv.Itoa(int(fv.Int()))
					default:
						// nothing
					}
					if v != "" {
						tmp[f] = v
					}
				}
			}
		}
		if cnt == len(tmp) {
			return tmp
		}
	}
	return
}

/* }}} */

/* {{{ func (r *REST) KeeperFactory() func(string) (interface{}, error)
 *
 */
func (r *REST) KeeperFactory() func(string) (interface{}, error) {
	return func(tag string) (interface{}, error) {
		return nil, nil
	}
}

/* }}} */

/* {{{ func (r *REST) Filter() (Model, error)
 * 数据过滤
 */
func (r *REST) Filter() (Model, error) {
	if m := r.Model(); m != nil {
		r := r.NewModel(m)
		rv := reflect.ValueOf(r)
		v := reflect.ValueOf(m)
		if cols := utils.ReadStructColumns(m, true); cols != nil {
			for _, col := range cols {
				fv := utils.FieldByIndex(v, col.Index)
				mv := utils.FieldByIndex(rv, col.Index)
				//r.Debug("field:%s; name: %s, kind:%v; type:%s", col.Tag, col.Name, fv.Kind(), fv.Type().String())
				if col.TagOptions.Contains(DBTAG_PK) || col.ExtOptions.Contains(TAG_RETURN) {
					//pk以及定义了返回tag的赋值
					mv.Set(fv)
				}
			}
		}
		return r, nil
	}
	return nil, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) Fill(j []byte) error
 * 填充新对象
 */
func (r *REST) Fill(j []byte) error {
	if r.filled == true {
		return nil
	}
	if m := r.Model(); m == nil {
		return ErrNoModel
	} else if err := json.Unmarshal(j, m); err != nil {
		return err
	} else {
		r.SetModel(m)
		if reflect.ValueOf(m).Kind() == reflect.Ptr {
			// Info("fill to new: %+v", reflect.Indirect(reflect.ValueOf(m)))
			r.new = reflect.Indirect(reflect.ValueOf(m)).Interface()
		} else {
			r.new = m
		}
		r.filled = true
	}
	return nil
}

/* }}} */

/* {{{ func (r *REST) Valid(opts ...string) (Model, error)
 * 验证
 */
func (r *REST) Valid(fields ...string) (Model, error) {
	c := r.Context()
	m := r.Model()
	if m == nil {
		return nil, ErrNoModel
	}
	// fill model
	if rb, err := ioutil.ReadAll(c.RequestBody()); err != nil {
		return nil, err
	} else if err := m.Fill(rb); err != nil {
		return nil, err
	}
	older := m.GetOlder()
	if r.Updating() && older == nil {
		return nil, fmt.Errorf("updating object is not exists")
	}
	keeper := m.Keeper()
	v := reflect.ValueOf(m)
	if cols := utils.ReadStructColumns(m, true); cols != nil {
		cnt := 0
		for _, col := range cols {
			if len(fields) > 0 && !utils.InSlice(col.Tag, fields) { // 如果传了fields, 只验证fields包含的字段
				continue
			}
			fv := utils.FieldByIndex(v, col.Index)
			// server generate,忽略传入的信息
			if fv.IsValid() && !utils.IsEmptyValue(fv) { //传入了内容
				if col.ExtOptions.Contains(TAG_GENERATE) && !col.TagOptions.Contains(DBTAG_PK) { //服务器生成, 忽略传入
					fv.Set(reflect.Zero(fv.Type()))
				} else if r.Updating() && col.ExtOptions.Contains(TAG_DENY) { //尝试编辑不可编辑的字段,要报错
					// 不可编辑字段，数字类型最好是指针，否则数字zero破坏力可强...
					c.Warn("%s is uneditable: %v", col.Tag, fv)
					//return nil, fmt.Errorf("%s is uneditable", col.Tag) //尝试编辑不可编辑的字段,直接报错
					fv.Set(reflect.Zero(fv.Type())) // 不报错, 忽略之
				} else {
					// 可编辑字段
					cnt++
				}
			} else if col.ExtOptions.Contains(TAG_REQUIRED) && r.Creating() { // 创建时必须传入,但是为空
				err := fmt.Errorf("field `%s` required, but empty", col.Tag)
				c.Info(err.Error())
				return nil, err
			}
			switch col.ExtTag { //根据tag, 会对数据进行预处理
			case "sha1":
				if fv.IsValid() && !utils.IsEmptyValue(fv) { //不能为空
					switch fv.Type().String() {
					case "*string":
						sv := fv.Elem().String()
						h := utils.HashSha1(sv)
						fv.Set(reflect.ValueOf(&h))
					case "string":
						sv := fv.String()
						h := utils.HashSha1(sv)
						fv.Set(reflect.ValueOf(h))
					default:
						return nil, fmt.Errorf("field(%s) must be string, not %s", col.Tag, fv.Kind().String())
					}
				}
			case "userid": //替换为userid,如果指定了数值
				if r.Creating() && (!fv.IsValid() || utils.IsEmptyValue(fv)) {
					var userid string
					if uid := r.GetEnv(USERID_KEY); uid == nil {
						// 如果没找到userid, 大可能性是inner请求, 采用传入数值
						if !fv.IsValid() || utils.IsEmptyValue(fv) {
							userid = "0"
						}
					} else {
						userid = uid.(string)
						//c.Debug("userid: %s", userid)
					}
					if err := utils.SetWithProperType(userid, fv); err != nil {
						return nil, fmt.Errorf("field(%s-%s) set value failed: %s", col.Tag, fv.Kind().String(), err)
					}
					// switch fv.Type().String() {
					// case "*string":
					// 	fv.Set(reflect.ValueOf(&userid))
					// case "string":
					// 	fv.Set(reflect.ValueOf(userid))
					// case "int":
					// 	ui, _ := strconv.Atoi(userid)
					// 	fv.Set(reflect.ValueOf(ui))
					// case "*int":
					// 	ui, _ := strconv.Atoi(userid)
					// 	fv.Set(reflect.ValueOf(&ui))
					// case "int64":
					// 	ui64, _ := strconv.ParseInt(userid, 10, 64)
					// 	fv.Set(reflect.ValueOf(ui64))
					// case "*int64":
					// 	ui64, _ := strconv.ParseInt(userid, 10, 64)
					// 	fv.Set(reflect.ValueOf(&ui64))
					// default:
					// 	return nil, fmt.Errorf("field(%s) must be string/int(64), not %s", col.Tag, fv.Kind().String())
					// }
				}
			case "time": //如果没有传值, 就是当前时间
				if r.Creating() && (!fv.IsValid() || utils.IsEmptyValue(fv)) { //创建同时为空
					now := time.Now()
					switch fv.Type().String() {
					case "*time.Time":
						fv.Set(reflect.ValueOf(&now))
					case "time.Time":
						fv.Set(reflect.ValueOf(now))
					default:
						return nil, fmt.Errorf("field(%s) must be time.Time, not %s", col.Tag, fv.Kind().String())
					}
				}
			case "existense": //检查存在性
				//if r.Creating() { //创建时才检查,这里不够安全(将来改)
				if exValue, err := keeper(col.Tag); err != nil {
					c.Debug("%s existense check failed: %s", col.Tag, err)
					return nil, err
				} else if exValue != nil {
					c.Debug("%s existense: %v", col.Tag, exValue)
					fv.Set(reflect.ValueOf(exValue))
				}
				//} else {
				//	c.Warn("not need check existense")
				//}
			case "uuid":
				if r.Creating() {
					switch fv.Type().String() {
					case "*string":
						h := utils.NewShortUUID()
						fv.Set(reflect.ValueOf(&h))
					case "string":
						h := utils.NewShortUUID()
						fv.Set(reflect.ValueOf(h))
					default:
						return nil, fmt.Errorf("field(%s) must be string, not %s", col.Tag, fv.Kind().String())
					}
				}
			case "luuid":
				if r.Creating() {
					switch fv.Type().String() {
					case "*string":
						h := utils.NewUUID()
						fv.Set(reflect.ValueOf(&h))
					case "string":
						h := utils.NewUUID()
						fv.Set(reflect.ValueOf(h))
					default:
						return nil, fmt.Errorf("field(%s) must be string, not %s", col.Tag, fv.Kind().String())
					}
				}
			case "stag":
				if r.Creating() { // 创建时加上内容
					if stag := r.GetEnv(STAG_KEY).(string); stag != "" {
						switch fv.Type().String() {
						case "*string":
							fv.Set(reflect.ValueOf(&stag))
						case "string":
							fv.Set(reflect.ValueOf(stag))
						default:
							return nil, fmt.Errorf("field(%s) must be string, not %s", col.Tag, fv.Kind().String())
						}
					}
				}
			case "forbbiden": //这个字段如果旧记录有值, 则返回错误
				if r.Updating() {
					ov := reflect.ValueOf(older)
					fov := utils.FieldByIndex(ov, col.Index)
					if fov.IsValid() && !utils.IsEmptyValue(fov) {
						return nil, fmt.Errorf("field(%s) has value, can't be updated", col.Tag)
					}
				}
				//default:
				//	//可自定义,初始化时放到tagHooks里面
				//	if col.ExtTag != "" && fv.IsValid() && !utils.IsEmptyValue(fv) { //还必须有值
				//		if hk := DMux.TagHooks.Get(col.ExtTag); hk != nil {
				//			fv.Set(hk.(TagHook)(v))
				//		} else {
				//			c.Info("cannot find hook for tag: %s", col.ExtTag)
				//		}
				//	}
			}
		}
		if r.Updating() && cnt == 0 {
			// 没什么可以编辑的
			return nil, errors.New("nothing to update")
		}
	}
	return m, nil
}

/* }}} */

/* {{{ func (r *REST) Protect() (Model, error)
 * 数据过滤
 */
func (r *REST) Protect() (Model, error) {
	if m := r.Model(); m != nil {
		if cols := utils.ReadStructColumns(m, true); cols != nil {
			v := reflect.ValueOf(m)
			for _, col := range cols {
				if col.ExtOptions.Contains(TAG_SECRET) { //保密,不对外
					fv := utils.FieldByIndex(v, col.Index)
					fv.Set(reflect.Zero(fv.Type()))
				}
			}
		}
		return m, nil
	}
	return nil, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) Row(opts ...interface{}) (Model, error)
 * 根据条件获取一条记录, model为表结构
 */
func (r *REST) Row(opts ...interface{}) (Model, error) {
	m := r.Model()
	if m == nil {
		return nil, ErrNoModel
	}
	params := utils.NewParams(opts)
	//找rowkey
	pf, pv, _ := m.PKey()
	if pv != "" {
		m.SetConditions(NewCondition(CTYPE_IS, pf, pv))
	} else if rk := params.PrimaryStringKey(); rk != "" {
		m.SetConditions(NewCondition(CTYPE_IS, pf, rk))
	} else {
		params.Bind(m)
	}

	if builder, err := m.ReadPrepare(false, true); err != nil {
		//没找到记录
		return nil, err
	} else {
		// builder := bi.(*gorp.Builder)
		ms := m.NewList()
		err := builder.Select(GetDbFields(m)).Limit("1").Find(ms)
		if err != nil && err != sql.ErrNoRows {
			//支持出错
			return nil, err
		} else if ms != nil {
			if resultsValue := reflect.Indirect(reflect.ValueOf(ms)); resultsValue.Len() > 0 {
				return SetModel(resultsValue.Index(0).Interface().(Model)), nil
			}
		}
	}
	return nil, ErrNoRecord
}

/* }}} */

/* {{{ func (r *REST) CreateRow() (Model, error)
 * 根据条件获取一条记录, model为表结构
 */
func (r *REST) CreateRow() (Model, error) {
	if m := r.Model(); m != nil {
		db := r.DBConn(WRITETAG)
		if r.Saved() {
			// 防止重复入库
			return m, nil
		}
		if err := db.Insert(m); err != nil { //Insert会把m换成新的
			return nil, err
		} else {
			return r.Save(m), nil
		}
	}
	return nil, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) Save()
 *
 */
func (r *REST) Save(m Model) Model {
	r.saved = true
	return r.SetModel(m)
}
func (r *REST) Saved() bool {
	return r.saved
}

/* }}} */

/* {{{ func (r *REST) UpdateRow(opts ...interface{}) (affected int64, err error)
 * 更新record
 */
func (r *REST) UpdateRow(opts ...interface{}) (affected int64, err error) {
	if m := r.Model(); m != nil {
		if len(opts) > 0 {
			if id := utils.PrimaryStringKey(opts); id != "" {
				if err = utils.ImportValue(m, map[string]string{DBTAG_PK: id}); err != nil {
					return
				}
			} else {
				Warn("[UpdateRow]not found id: %s, %+v", id, opts)
				return 0, ErrNoRecord
			}
		} else if pf, pv, _ := m.PKey(); pf != "" {
			if pv == "" {
				Warn("[UpdateRow]pk empty: %s, %s", pf, pv)
				return 0, ErrNoRecord
			}
		} else if uks := m.UnionKeys(); len(uks) <= 0 {
			Warn("[UpdateRow]union keys empty")
			return 0, ErrNoRecord
		}
		return r.DBConn(WRITETAG).Update(m)
	}
	err = ErrNoModel
	return
}

/* }}} */

/* {{{ func (r *REST) DeleteRow(id string) (affected int64, err error)
 * 删除记录(逻辑删除)
 */
func (r *REST) DeleteRow(id string) (affected int64, err error) {
	if m := r.Model(); m != nil {
		db := r.DBConn(WRITETAG)
		if err = utils.ImportValue(m, map[string]string{DBTAG_PK: id, DBTAG_LOGIC: "-1"}); err != nil {
			return
		}
		return db.Update(m)
	}
	return 0, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) Rows(...interface{}) (rs interface{}, err error)
 * 获取list, 通用函数
 */
func (r *REST) Rows(opts ...interface{}) (ms interface{}, err error) {
	if m := r.Model(); m != nil {
		params := utils.NewParams(opts)
		// find pagination
		var p *Pagination
		if pp, ok := params.ItfByIndex(0).(*Pagination); ok {
			p = pp
		}
		// read tag
		readTag := true
		if force := params.BoolByIndex(1); force {
			readTag = false
		}

		builder, pe := r.ReadPrepare()
		if pe != nil {
			return nil, pe
		}
		// builder := bi.(*gorp.Builder)

		ms = r.NewList()
		if p != nil {
			err = builder.Select(GetDbFields(m, readTag)).Offset(p.Offset).Limit(p.PerPage).Find(ms)
		} else {
			err = builder.Select(GetDbFields(m, readTag)).Find(ms)
		}
		if err != nil && err != sql.ErrNoRows {
			//支持出错
			return nil, err
		}

		return reflect.ValueOf(ms).Elem().Interface(), nil
	}
	return nil, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) List() (l *List, err error)
 * 获取list, 通用函数
 */
func (r *REST) List() (l *List, err error) {
	if m := r.Model(); m != nil {
		//c := r.Context()
		l = new(List)
		builder, _ := r.ReadPrepare()
		// builder := bi.(*gorp.Builder)
		count, _ := builder.Count() //结果数
		ms := r.NewList()
		if p := r.Pagination(); p != nil {
			l.Info.Page = &p.Page
			l.Info.PerPage = &p.PerPage
			l.Info.Total = count
			err = builder.Select(GetDbFields(m, true)).Offset(p.Offset).Limit(p.PerPage).Find(ms)
			//c.Debug("[offset: %d][per_page: %d]", p.Offset, p.PerPage)
		} else {
			//r.Debug("get fields: %v", GetDbFields(m, true))
			err = builder.Select(GetDbFields(m, true)).Find(ms)
		}
		if err != nil && err != sql.ErrNoRows {
			//支持出错
			return l, err
			// } else if ms == nil {
			// 	//没找到记录
			// 	return l, ErrNoRecord
		}

		l.List = ms

		return l, nil
	}
	return nil, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) GetSum(d ...string) (l *List, err error)
 * 获取list, 通用函数
 */
func (r *REST) GetSum(d ...string) (l *List, err error) {
	if m := r.Model(); m != nil {
		builder, _ := r.ReadPrepare(true)
		// builder := bi.(*gorp.Builder)

		l = new(List)

		group := make([]string, 0)
		ms := r.NewList()
		if err := builder.Select(GetSumFields(m, group...)).Find(ms); err == nil {
			sumValue := reflect.Indirect(reflect.ValueOf(ms))
			if sumValue.Len() > 0 {
				l.Info.Sum = sumValue.Index(0).Interface()
			}
		}

		if len(d) > 0 {
			group = append(group, d...)
		}
		builder.Group(group)

		ms = r.NewList()

		if err = builder.Select(GetSumFields(m, group...)).Find(ms); err != nil {
			return l, err
		} else if ms == nil {
			return l, ErrNoRecord
		}

		listValue := reflect.Indirect(reflect.ValueOf(ms))
		l.Info.Total = int64(listValue.Len())

		l.List = ms

		return
	}
	return nil, ErrNoModel
}

/* }}} */

/* {{{ func (r *REST) GetCount() (cnt int64, err error)
 * 获取list, 通用函数
 */
func (r *REST) GetCount() (cnt int64, err error) {
	if r.Count > 0 {
		return r.Count, nil
	} else {
		builder, _ := r.ReadPrepare()
		// builder := bi.(*gorp.Builder)
		return builder.Count()
	}
}

/* }}} */

/* {{{ func (r *REST) GetCountNSum() (cnt int64, sum float64)
 * 获取计数以及求和, 通用函数
 */
func (r *REST) GetCountNSum() (cnt int64, sum float64) {
	return r.Count, r.Sum
}

/* }}} */

/* {{{ func (r *REST) GetRecord(opts ...interface{}) interface{}
 * get record (cacheable), 注意返回不是指针
 */
func (r *REST) GetRecord(opts ...interface{}) interface{} {
	m := r.Model()
	if m == nil {
		Warn("[GetRecord]: %s", ErrNoModel)
		return nil
	}
	ck := ""
	params := utils.NewParams(opts)
	pk := params.PrimaryStringKey()
	if pk != "" {
		if err := utils.ImportValue(m, map[string]string{DBTAG_PK: pk}); err != nil {
			return nil
		}
		ck = fmt.Sprint(m.TableName(), ":", pk)
	} else if _, pk, _ := m.PKey(); pk != "" {
		ck = fmt.Sprintf("%s:%s", m.TableName(), pk)
	} else {
		// bind参数, bind不上也没关系
		params.Bind(m)
		if kf, v, _ := m.Key(); v != "" {
			ck = fmt.Sprintf("%s:%s:%s", m.TableName(), kf, v)
		} else if uks := m.UnionKeys(); len(uks) > 0 {
			ck = m.TableName()
			for f, v := range uks {
				ck += fmt.Sprintf(":%s:%s", f, v)
			}
		} else {
			// 最后再找所有可能的conditional字段
			cols := utils.ReadStructColumns(m, true)
			for _, col := range cols {
				fv := utils.FieldByIndex(reflect.ValueOf(m), col.Index)
				if (col.ExtOptions.Contains(TAG_CONDITION) || col.TagOptions.Contains(DBTAG_UK) || col.TagOptions.Contains(DBTAG_KEY)) && fv.IsValid() && !utils.IsEmptyValue(fv) {
					if fs := utils.GetRealString(fv); fs != "" { // 多个字段有值, 用AND
						ck += fmt.Sprintf(":%s:%s", col.Tag, fs)
					}
				}
			}
			if ck != "" {
				ck = m.TableName() + ck
			}
		}
	}
	if ck != "" {
		// find var in local cache
		if cvi, err := LocalGet(ck); err == nil {
			// found
			// Debug("hit var in cache: %s, %+v, %s", ck, cvi, utils.ToType(cvi).String())
			Debug("[GetRecord]hit cache: %s", ck)
			if _, ok := cvi.(Model); ok {
				return utils.Pointer(cvi)
			}
		}
		// find in db
		if rec, err := m.Row(); err == nil {
			// found it
			Debug("[GetRecord]found %s in db and save to cache", ck)
			recv := reflect.Indirect(reflect.ValueOf(rec)).Interface()
			LocalSet(ck, recv, CACHE_EXPIRE)
			return rec
		} else {
			Debug("[GetRecord]find %s in db failed: %s", ck, err)
		}
	}
	Debug("[GetRecord]cachekey empty")
	return nil
}

/* }}} */

/* {{{ func (r *REST) UpdateRecord(opts ...interface{}) error
 * 更新record
 */
func (r *REST) UpdateRecord(opts ...interface{}) error {
	m := r.Model()
	if m == nil {
		return ErrNoModel
	}
	ck := ""
	pk := utils.PrimaryStringKey(opts)
	if pk != "" {
		if err := utils.ImportValue(m, map[string]string{DBTAG_PK: pk}); err != nil {
			return err
		}
		ck = fmt.Sprint(m.TableName(), ":", pk)
	} else if _, pk, _ := m.PKey(); pk != "" {
		ck = fmt.Sprintf("%s:%s", m.TableName(), pk)
	} else if uks := m.UnionKeys(); len(uks) > 0 {
		ck = m.TableName()
		for f, v := range uks {
			ck += fmt.Sprintf(":%s:%s", f, v)
		}
	}
	if ck == "" {
		return ErrNoRecord
	}
	Debug("[UpdateRecord]cachekey: %s", ck)
	if _, err := m.UpdateRow(); err != nil {
		Warn("[UpdateRecord]update failed: %s", err)
		return err
	}
	// update local cache
	// 1. if cache exists
	if cvi, err := LocalGet(ck); err == nil {
		if err := utils.Merge(m, cvi); err != nil {
			Warn("[UpdateRecord]merge failed: %s", err)
		}
	}
	LocalSet(ck, reflect.Indirect(reflect.ValueOf(m)).Interface().(Model), CACHE_EXPIRE)
	return nil
}

/* }}} */

/* {{{ func (r *REST) Write(...interface{}) (Model, error)
 * 判断primary key, 记录存在则更新, 不存在则创建
 */
func (r *REST) Write(opts ...interface{}) (Model, error) {
	m := r.Model()
	if m == nil {
		return nil, ErrNoModel
	}
	pf, pk, ai := m.PKey()
	if pf != "" { // 具有primary key
		if pk == "" {
			pk = utils.PrimaryStringKey(opts)
		}
		if pk == "" && !ai { // 具有primary key, 同时不是auto increasement, 并且没找到primary key value, 则返回没找到
			return nil, ErrNoRecord
		}
	} else if uks := m.UnionKeys(); len(uks) > 0 { // 没有primary key, 超找union keys(有多个)
		Debug("[model.Write]union keys: %s", uks)
	} else {
		return nil, ErrNoRecord
	}
	// check if record exists
	if m.GetRecord(pk) != nil {
		// update
		Debug("[model.Write]record exists, update it")
		if err := m.UpdateRecord(pk); err != nil {
			return nil, err
		}
		return m, nil
	} else {
		// create
		Debug("[model.Write]record(%s) not exists and create it", pk)
		return m.CreateRow()
	}
}

/* }}} */

/* {{{ func (r *REST) GetOlder(opts ...string) Model
 * get older record
 */
func (r *REST) GetOlder(opts ...string) Model {
	if r.older == nil {
		if m := r.Model(); m != nil {
			rk := ""
			if len(opts) > 0 && opts[0] != "" {
				// check params
				rk = opts[0]
			} else if _, v, _ := m.PKey(); v != "" {
				// check variable primary key
				rk = v
			} else if c := r.Context(); c != nil {
				rk = c.Param(RowkeyKey)
			}
			// r.Debug("[GetOlder]rowkey: %s", rk)
			if rk != "" {
				if older, err := m.Row(rk); err == nil {
					r.older = older
				}
			}
		}
	}
	return r.older
}

/* }}} */

/* {{{ func (r *REST) AddTable(tags ...string) Model
 * 注册表结构
 */
func (r *REST) AddTable(tags ...string) Model {
	if m := r.Model(); m != nil {
		reflectVal := reflect.ValueOf(m)
		mv := reflect.Indirect(reflectVal).Interface()
		//Debug("table name: %s", r.TableName())
		tb := r.TableName()
		gtm := gorp.AddTableWithName(mv, tb)
		if pf, _, ai := m.PKey(); pf != "" {
			gtm.SetKeys(ai, pf)
		} else if uks := m.UnionKeys(false); len(uks) > 0 {
			// union keys
			// Debug("[AddTable]union keys for %s: %s", tb, uks)
			gtm.SetKeys(false, utils.MapKeys(uks)...)
		}

		//data accessor, 默认都是DBTAG
		DataAccessor[tb+"::"+WRITETAG] = DBTAG
		DataAccessor[tb+"::"+READTAG] = DBTAG
		if len(tags) > 0 {
			writeTag := tags[0]
			if dns := db[writeTag]; dns != "" {
				Info("%s's writer: %s", tb, dns)
				if err := OpenDB(writeTag, dns); err != nil {
					Warn("open db(%s) error: %s", writeTag, err)
				} else {
					DataAccessor[tb+"::"+WRITETAG] = writeTag
				}
			}
		}
		if len(tags) > 1 {
			readTag := tags[1]
			if dns := db[readTag]; dns != "" {
				Info("%s's reader: %s", tb, dns)
				if err := OpenDB(readTag, dns); err != nil {
					Warn("open db(%s) error: %s", readTag, err)
				} else {
					DataAccessor[tb+"::"+READTAG] = readTag
				}
			}
		}
	} else {
		Warn("[AddTable]: %s", ErrNoModel)
	}

	return r
}

/* }}} */

// 注入checklist的字典
func (r *REST) ImportDic(field string, dic ChecklistDic) {
}

/* {{{ func (r *REST) ReadPrepare(opts ...interface{}) (*gorp.Builder, error)
 * 查询准备
 */
func (r *REST) ReadPrepare(opts ...interface{}) (*gorp.Builder, error) {
	disableOrder := false
	if len(opts) > 0 {
		if do, ok := opts[0].(bool); ok && do {
			disableOrder = true
		}
	}
	mustHasCons := false
	if len(opts) > 1 {
		if mh, ok := opts[1].(bool); ok && mh {
			mustHasCons = true
		}
	}

	m := r.Model()
	if m == nil {
		return nil, ErrNoModel
	}
	cols := utils.ReadStructColumns(m, true)
	if cols == nil || len(cols) == 0 {
		return nil, ErrType
	}

	db := r.DBConn(READTAG)
	tb := r.TableName()
	b := gorp.NewBuilder(db).Table(tb)
	cons := r.Conditions()

	// condition
	if len(cons) > 0 {
		//Debug("condition set")
		//range condition,范围查询
		for _, v := range cons {
			//时间范围查询
			if v.Range != nil {
				//Debug("[perpare]timerange")
				switch vt := v.Range.(type) {
				case *TimeRange: //只支持timerange
					b.Where(fmt.Sprintf("T.`%s` BETWEEN ? AND ?", v.Field), vt.Start.Format(_MYSQL_FORM), vt.End.Format(_MYSQL_FORM))
				case TimeRange: //只支持timerange
					b.Where(fmt.Sprintf("T.`%s` BETWEEN ? AND ?", v.Field), vt.Start.Format(_MYSQL_FORM), vt.End.Format(_MYSQL_FORM))
				default:
					//nothing
				}
			}
			//排序
			if v.Order != nil && !disableOrder {
				switch vt := v.Order.(type) {
				case *OrderBy:
					b.Order(fmt.Sprintf("T.`%s` %s", vt.Field, vt.Sort))
				case OrderBy:
					b.Order(fmt.Sprintf("T.`%s` %s", vt.Field, vt.Sort))
				default:
					//nothing
				}
			}
		}
		joinCount := 0
		orCons := make(map[string][]string)
		for _, v := range cons {
			//Debug("[key: %s]%v", v.Field, v)
			v.DoWhere(b) //已经处理了 raw/is/not/like/gt/lt
			if v.Or != nil {
				//Debug("[OR][key: %s]%v", v.Field, v)
				oc := v.Or.(*Condition)
				orKey := oc.Field
				if orCons[orKey] == nil {
					orCons[orKey] = make([]string, 0)
				}
				//Debug("or condition: %s", orKey)
				switch ot := oc.Is.(type) {
				case string:
					//Debug("or condition: %s, field: %s", orKey, v.Field)
					orCons[orKey] = append(orCons[orKey], fmt.Sprintf("T.`%s` = '%s'", v.Field, ot))
				case []string:
					vs := bytes.Buffer{}
					first := true
					vs.WriteString("(")
					for _, vv := range ot {
						if !first {
							vs.WriteString(",")
						}
						vs.WriteString(fmt.Sprintf("'%s'", vv))
						first = false
					}
					vs.WriteString(")")
					orCons[orKey] = append(orCons[orKey], fmt.Sprintf("T.`%s` IN %s", v.Field, vs.String()))
				case []interface{}:
					vs := bytes.Buffer{}
					first := true
					vs.WriteString("(")
					for _, vv := range ot {
						if !first {
							vs.WriteString(",")
						}
						vs.WriteString(fmt.Sprintf("'%s'", vv))
						first = false
					}
					vs.WriteString(")")
					orCons[orKey] = append(orCons[orKey], fmt.Sprintf("T.`%s` IN %s", v.Field, vs.String()))
				default:
				}
			}
			if v.Join != nil { //关联查询
				Debug("join found: %+v", v.Join)
				if vt, ok := v.Join.(*Condition); ok && vt.Is != nil {
					joinTable := v.Field // 字段名就是表名称
					joinField := vt.Field
					Debug("join %s.%s", joinTable, joinField)
					if t, ok := gorp.GetTable(joinTable); ok {
						if fcols := utils.ReadStructColumns(reflect.New(t.Gotype).Interface(), true); fcols != nil {
							for _, col := range fcols {
								if col.Tag == joinField && col.ExtOptions.Contains(TAG_CONDITION) { //可作为条件
									Debug("[match]join %s.%s", joinTable, joinField)
									if v.JoinOn != nil {
										b.Joins(fmt.Sprintf("LEFT JOIN `%s` T%d ON T.`%s` = T%d.`%s`", v.Field, joinCount, v.JoinOn[0], joinCount, v.JoinOn[1]))
									} else {
										b.Joins(fmt.Sprintf("LEFT JOIN `%s` T%d ON T.`%s` = T%d.`id`", joinTable, joinCount, v.Field, joinCount))
									}
									// b.Where(fmt.Sprintf("T%d.`%s`=?", joinCount, joinField), vt.Is.(string))
									buildWhereRaw(b, fmt.Sprintf("T%d", joinCount), joinField, vt.Is)
									joinCount++
									break
								}
							}
						}
					}
				}
			}
		}
		if len(orCons) > 0 {
			for _, css := range orCons {
				b.Where("(" + strings.Join(css, " OR ") + ")")
			}
		}
	} else { // 从自身找， primary key/key
		hasCon := false
		if pf, pk, _ := m.PKey(); pk != "" {
			hasCon = true
			b.Where(fmt.Sprintf("T.`%s` = ?", pf), pk)
		} else if kf, v, _ := m.Key(); v != "" {
			hasCon = true
			b.Where(fmt.Sprintf("T.`%s` = ?", kf), v)
		} else if uks := m.UnionKeys(); len(uks) > 0 {
			hasCon = true
			for f, v := range uks {
				b.Where(fmt.Sprintf("T.`%s` = ?", f), v)
			}
		} else {
			// 最后再找conditional 字段
			for _, col := range cols {
				fv := utils.FieldByIndex(reflect.ValueOf(m), col.Index)
				if (col.ExtOptions.Contains(TAG_CONDITION) || col.TagOptions.Contains(DBTAG_UK) || col.TagOptions.Contains(DBTAG_KEY)) && fv.IsValid() && !utils.IsEmptyValue(fv) {
					//有值
					if fs := utils.GetRealString(fv); fs != "" { // 多个字段有值, 用AND
						hasCon = true
						b.Where(fmt.Sprintf("T.`%s` = ?", col.Tag), fs)
					}
				}
			}
		}
		if !hasCon && mustHasCons {
			// 没有找到任何查询条件，查询失败
			return nil, ErrNoCondition
		}
	}

	if !disableOrder {
		pks := ""
		for _, col := range cols {
			//处理排序问题,如果之前有排序，这里就是二次排序,如果之前无排序,这里是首要排序
			if col.TagOptions.Contains(DBTAG_PK) { // 默认为pk降序
				pks = fmt.Sprintf("T.`%s` DESC", col.Tag)
				if col.ExtOptions.Contains(TAG_AORDERBY) {
					pks = fmt.Sprintf("T.`%s` ASC", col.Tag)
				}
			} else if col.ExtOptions.Contains(TAG_ORDERBY) { // 默认为降序
				b.Order(fmt.Sprintf("T.`%s` DESC", col.Tag))
			} else if col.ExtOptions.Contains(TAG_AORDERBY) { //正排序
				b.Order(fmt.Sprintf("T.`%s` ASC", col.Tag))
			}
			// 处理逻辑删除
			if col.TagOptions.Contains(DBTAG_LOGIC) {
				b.Where(fmt.Sprintf("T.`%s` != -1", col.Tag))
			}
		}
		if pks != "" { //pk排序放到最后
			b.Order(pks)
		}
	}

	return b, nil
}

/* }}} */

/* {{{ func underscore(str string) string
 *
 */
func underscore(str string) string {
	buf := bytes.Buffer{}
	for i, s := range str {
		if s <= 'Z' && s >= 'A' {
			if i > 0 {
				buf.WriteString("_")
			}
			buf.WriteString(string(s + 32))
		} else {
			buf.WriteString(string(s))
		}
	}
	return buf.String()
}

/* }}} */

/* {{{ GetDbFields(i interface{}, ops ...interface{}) (s string)
 * 从struct中解析数据库字段以及字段选项
 */
func GetDbFields(i interface{}, ops ...interface{}) (s []string) {
	var readTag bool
	if len(ops) > 0 {
		if st, ok := ops[0].(bool); ok && st == true {
			readTag = true
		}
	}

	fs := i.(Model).Fields()
	if cols := utils.ReadStructColumns(i, true); cols != nil {
		s = make([]string, 0)
		for _, col := range cols {
			if col.Tag == "-" { //无此字段
				continue
			} else if col.ExtOptions.Contains(TAG_HIDDEN) { //隐藏字段忽略
				continue
			} else if readTag && col.ExtOptions.Contains(TAG_SECRET) { //默认忽略tag
				continue
			} else if len(fs) > 0 && !col.TagOptions.Contains(DBTAG_PK) && !utils.InSlice(col.Tag, fs) {
				continue
			}
			s = append(s, col.Tag)
		}
	}
	return
}

/* }}} */

/* {{{ func GetSumFields(i interface{}, g ...string) (s string)
 * 从struct中解析数据库字段以及字段选项,为了报表
 */
func GetSumFields(i interface{}, g ...string) (s string) {
	if cols := utils.ReadStructColumns(i, true); cols != nil {
		bs := bytes.Buffer{}
		first := true
		for _, col := range cols {
			if !col.ExtOptions.Contains(TAG_REPORT) { //不是报表字段,不对外
				continue
			}
			if col.ExtOptions.Contains(TAG_SECRET) { //保密,不对外
				continue
			}
			if col.ExtOptions.Contains(TAG_CANGROUP) && !utils.InSlice(col.Tag, g) {
				continue
			}
			if !first {
				bs.WriteString(",")
			}
			if col.ExtOptions.Contains(TAG_SUM) {
				bs.WriteString(fmt.Sprintf("SUM(T.`%s`) AS `%s`", col.Tag, col.Tag))
				if col.ExtOptions.Contains(TAG_TSUM) {
					bs.WriteString(fmt.Sprintf(",SUM(T.`%s`) AS `%s`", col.Tag, EXF_SUM))
				}
			} else if col.ExtOptions.Contains(TAG_COUNT) {
				bs.WriteString(fmt.Sprintf("COUNT(T.`%s`) AS `%s`", col.Tag, EXF_COUNT))
			} else {
				bs.WriteString("T.`" + col.Tag + "`")
			}
			first = false
		}
		s = bs.String()
	}
	return
}

/* }}} */

// dig model
func digModel(m Model) Model {
	rt := utils.RealType(m, reflect.TypeOf((*Model)(nil)).Elem())
	//Info("mtype: %v, real type: %v", mt, rt)
	return reflect.New(rt).Interface().(Model)
}
