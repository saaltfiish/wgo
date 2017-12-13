package rest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"gorp"
	//"wgo"
	"wgo/utils"
)

type Model interface {
	Keeper() Keeper // 各种检查
	KeeperFactory() Keeper

	SetConditions(...*Condition) Model
	Conditions() []*Condition
	SetPagination(p *Pagination) Model
	Pagination() *Pagination
	SetFields(...string) Model
	Fields() []string
	NewList() interface{} // 返回一个空结构列表
	AddTable(...string)
	DBConn(string) *gorp.DbMap                   // 数据库连接
	TableName() string                           // 返回表名称, 默认结构type名字(小写), 有特别的表名称,则自己implement 这个方法
	PKey() (string, string, bool)                // key字段,以及是否auto incr
	ReadPrepare() (interface{}, error)           // 组条件
	Row(...interface{}) (Model, error)           //获取单条记录
	Rows() (*List, error)                        //获取多条记录
	GetOlder(rk ...string) Model                 //获取旧记录
	GetSum(...string) (*List, error)             //获取多条记录
	GetCount() (int64, error)                    //获取多条记录
	GetCountNSum() (int64, float64)              //获取count and sum
	CreateRow() (Model, error)                   //创建单条记录
	UpdateRow(ext ...interface{}) (int64, error) //更新记录
	DeleteRow(rk string) (int64, error)          //更新记录

	Fill([]byte) error              //填充内容
	Valid(...string) (Model, error) //数据验证, 如果传入opts, 则只验证opts指定的字段
	Filter() (Model, error)         //数据过滤(创建,更新后)
	Protect() (Model, error)        //数据保护(获取数据时过滤字段)
}

type Keeper func(string) (interface{}, error)

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
	ErrInvalid       = errors.New("invalid query")
	ErrNoRecord      = errors.New("no record")
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
	Page   interface{}
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
	case CTYPE_PAGE:
		con.Page = v
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
	case string:
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
			b.Where(fmt.Sprintf("T.`%s` >= '%s'", v.Field, vt))
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
			b.Where(fmt.Sprintf("T.`%s` < '%s'", v.Field, vt))
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
			b.Where(fmt.Sprintf("T.`%s` LIKE '%%%s%%'", v.Field, vt))
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
	if oc.Page != nil {
		con.Page = oc.Page
	}
	if oc.Raw != "" {
		con.Raw = oc.Raw
	}
}

/* }}} */

// 创建一个全新的model
func NewModel(i interface{}) Model {
	rest := new(REST)
	return rest.NewModel(i)
}

// 创建一个跟rest有关的model
func (rest *REST) NewModel(i interface{}) Model {
	//m := reflect.New(reflect.Indirect(reflect.ValueOf(i)).Type()).Interface().(Model)
	m := reflect.New(reflect.TypeOf(i).Elem()).Interface().(Model)
	return rest.SetModel(m)
}

// 从rest创建一个全新的model, 不需要传参,因为类型已经知道
// return a new instance
func (rest *REST) New() Model {
	if m := rest.Model(); m != nil {
		//return reflect.New(reflect.Indirect(reflect.ValueOf(m)).Type()).Interface().(Model)
		return NewModel(m)
	}
	return nil
}

/* {{{ func GetCondition(cs []*Condition, k string) (con *Condition, err error)
 *
 */
func GetCondition(cs []*Condition, k string) (con *Condition, err error) {
	if cs == nil {
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

/* {{{ func (rest *REST) SetConditions(cs ...*Condition) Model
 * 设置条件
 */
func (rest *REST) SetConditions(cs ...*Condition) Model {
	if rest.conditions == nil {
		rest.conditions = make([]*Condition, 0)
	}
	if m := rest.Model(); m == nil {
		Warn("[SetConditions]: not found model")
	} else if cols := utils.ReadStructColumns(m, true); cols != nil {
		for _, col := range cols {
			// Debug("[SetConditions][tag: %s][ext: %s][type: %s]", col.Tag, col.ExtTag, col.Type.String())
			// join
			if condition, e := GetCondition(cs, col.ExtTag); e == nil && condition.Join != nil {
				// Debug("[SetConditions][join][table: %s]%v", col.ExtTag, condition)
				rest.conditions = append(rest.conditions, condition)
			}
			// raw
			if condition, e := GetCondition(cs, col.Tag); e == nil && condition.Raw != "" {
				//Debug("[SetConditions][raw][tag: %s]%v", col.Tag, condition)
				rest.conditions = append(rest.conditions, condition)
			}
			// time range
			if col.ExtOptions.Contains(TAG_TIMERANGE) {
				if condition, e := GetCondition(cs, TAG_TIMERANGE); e == nil && condition.Range != nil {
					//Info("[SetConditions]timerange")
					condition.Field = col.Tag
					rest.conditions = append(rest.conditions, condition)
				} else {
					//Info("get condition failed: %s", e)
				}
			}
			if col.ExtOptions.Contains(TAG_ORDERBY) {
				if condition, e := GetCondition(cs, TAG_ORDERBY); e == nil && condition.Order != nil {
					//Debug("[SetConditions]order")
					condition.Field = col.Tag
					rest.conditions = append(rest.conditions, condition)
				} else {
					//Trace("get condition failed: %s", e)
				}
			}
			if col.TagOptions.Contains(DBTAG_PK) || col.ExtOptions.Contains(TAG_CONDITION) { //primary key or conditional
				if condition, e := GetCondition(cs, col.Tag); e == nil && (condition.Is != nil || condition.Not != nil || condition.Gt != nil || condition.Lt != nil || condition.Like != nil || condition.Join != nil || condition.Or != nil) {
					//Debug("[SetConditions][tag: %s][type: %s]%v", col.Tag, col.Type.String(), condition)
					rest.conditions = append(rest.conditions, ParseCondition(col.Type.String(), condition))
				}
			}
		}
	}
	return rest
}

/* }}} */

/* {{{ func (rest *REST) Conditions() []*Condition
 *
 */
func (rest *REST) Conditions() []*Condition {
	return rest.conditions
}

/* }}} */

/* {{{ func (rest *REST) SetPagination(p *Pagination) Model
 * 生成条件
 */
func (rest *REST) SetPagination(p *Pagination) Model {
	if rest.conditions == nil {
		rest.pagination = new(Pagination)
	}
	rest.pagination = p
	return rest
}

/* }}} */

/* {{{ func (rest *REST) Pagination() *Pagination
 *
 */
func (rest *REST) Pagination() *Pagination {
	return rest.pagination
}

/* }}} */

/* {{{ func (rest *REST) SetFields(fs ...string) Model
 * 生成条件
 */
func (rest *REST) SetFields(fs ...string) Model {
	if rest.fields == nil {
		rest.fields = make([]string, 0)
	}
	rest.fields = fs
	return rest
}

/* }}} */

/* {{{ func (rest *REST) Fields() []string
 *
 */
func (rest *REST) Fields() []string {
	return rest.fields
}

/* }}} */

/* {{{ func (rest *REST) Keeper() Keeper
 *
 */
func (rest *REST) Keeper() Keeper {
	if rest.keeper == nil && rest.Model() != nil {
		rest.keeper = rest.Model().KeeperFactory()
	}
	return rest.keeper
}

/* }}} */

/* {{{ func (rest *REST) NewList() *[]Model
 *
 */
func (rest *REST) NewList() interface{} {
	if m := rest.Model(); m == nil {
		Info("[NewList]: not found model")
		return nil
	} else {
		return reflect.New(reflect.SliceOf(reflect.TypeOf(m))).Interface()
	}
}

/* }}} */

/* {{{ func (rest *REST) DBConn(tag string) *gorp.DbMap
 * 默认数据库连接为admin
 */
func (rest *REST) DBConn(tag string) *gorp.DbMap {
	tb := rest.TableName()
	if dt, ok := DataAccessor[tb+"::"+tag]; ok && dt != "" {
		return gorp.Using(dt)
	}
	return gorp.Using(DBTAG)
}

/* }}} */

/* {{{ func (rest *REST) TableName() (n string)
 * 获取表名称, 默认为结构名
 */
func (rest *REST) TableName() (n string) { //默认, struct的名字就是表名, 如果不是请在各自的model里定义
	if m := rest.Model(); m == nil {
		Info("[TableName]error: not found model")
	} else {
		reflectVal := reflect.ValueOf(m)
		mt := reflect.Indirect(reflectVal).Type()
		n = underscore(strings.TrimSuffix(mt.Name(), "Table"))
	}
	return
}

/* }}} */

/* {{{ func (rest *REST) PKey() (string, string, bool)
 *  通过配置找到pk
 */
func (rest *REST) PKey() (f string, v string, ai bool) {
	var m Model
	if m = rest.Model(); m == nil {
		Info("[PKey]: not found model")
		return "", "", false
	}
	mv := reflect.ValueOf(m)
	if cols := utils.ReadStructColumns(m, true); cols != nil {
		for _, col := range cols {
			// check required field
			if col.TagOptions.Contains(DBTAG_PK) {
				f = col.Tag
				fv := utils.FieldByIndex(mv, col.Index)
				v = ""
				if fv.IsValid() && !utils.IsEmptyValue(fv) {
					switch fv.Type().String() {
					case "*string":
						v = fv.Elem().String()
					case "string":
						v = fv.String()
					default:
						// nothing
					}
				}
				if col.ExtOptions.Contains(TAG_GENERATE) && col.ExtTag != "" { //服务端生成并且有tag
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

/* {{{ func (rest *REST) KeeperFactory() Keeper
 *
 */
func (rest *REST) KeeperFactory() Keeper {
	return func(tag string) (interface{}, error) {
		return nil, nil
	}
}

/* }}} */

/* {{{ func (rest *REST) Filter() (Model, error)
 * 数据过滤
 */
func (rest *REST) Filter() (Model, error) {
	if m := rest.Model(); m != nil {
		r := rest.NewModel(m)
		rv := reflect.ValueOf(r)
		v := reflect.ValueOf(m)
		if cols := utils.ReadStructColumns(m, true); cols != nil {
			for _, col := range cols {
				fv := utils.FieldByIndex(v, col.Index)
				mv := utils.FieldByIndex(rv, col.Index)
				//rest.Debug("field:%s; name: %s, kind:%v; type:%s", col.Tag, col.Name, fv.Kind(), fv.Type().String())
				if col.TagOptions.Contains(DBTAG_PK) || col.ExtOptions.Contains(TAG_RETURN) {
					//pk以及定义了返回tag的赋值
					mv.Set(fv)
				}
			}
		}
		return r, nil
	} else {
		err := fmt.Errorf("[Filter] not found model")
		//Info("error: %s", err)
		return nil, err
	}
}

/* }}} */

/* {{{ func (rest *REST) Fill(j []byte) error
 * 根据条件获取一条记录, model为表结构
 */
func (rest *REST) Fill(j []byte) error {
	if rest.filled == true {
		return nil
	}
	if m := rest.Model(); m == nil {
		return fmt.Errorf("[Fill] not found model")
	} else if err := json.Unmarshal(j, m); err != nil {
		return err
	} else {
		rest.SetModel(m)
		rest.filled = true
	}
	return nil
}

/* }}} */

/* {{{ func (rest *REST) Valid(opts ...string) (Model, error)
 * 验证
 */
func (rest *REST) Valid(fields ...string) (Model, error) {
	c := rest.Context()
	m := rest.Model()
	if m == nil {
		return nil, fmt.Errorf("rest model() failed")
	}
	// fill model
	if rb, err := ioutil.ReadAll(c.RequestBody()); err != nil {
		return nil, err
	} else if err := m.Fill(rb); err != nil {
		return nil, err
	}
	older := m.GetOlder()
	if rest.Updating() && older == nil {
		return nil, fmt.Errorf("updating object is not exists")
	}
	keeper := m.Keeper()
	v := reflect.ValueOf(m)
	if cols := utils.ReadStructColumns(m, true); cols != nil {
		for _, col := range cols {
			if len(fields) > 0 && !utils.InSlice(col.Tag, fields) { // 如果传了fields, 只验证fields包含的字段
				continue
			}
			fv := utils.FieldByIndex(v, col.Index)
			// server generate,忽略传入的信息
			if fv.IsValid() && !utils.IsEmptyValue(fv) { //传入了内容
				if col.ExtOptions.Contains(TAG_GENERATE) && !col.TagOptions.Contains(DBTAG_PK) { //服务器生成, 忽略传入
					fv.Set(reflect.Zero(fv.Type()))
				} else if rest.Updating() && col.ExtOptions.Contains(TAG_DENY) { //尝试编辑不可编辑的字段,要报错
					// 注意不可编辑字段，数字类型最好是指针，否则数字zero破坏力可强...
					c.Info("%s is uneditable: %v", col.Tag, fv)
					//return nil, fmt.Errorf("%s is uneditable", col.Tag) //尝试编辑不可编辑的字段,直接报错
					fv.Set(reflect.Zero(fv.Type())) // 不报错, 忽略之
				}
			} else if col.ExtOptions.Contains(TAG_REQUIRED) && rest.Creating() { // 创建时必须传入,但是为空
				err := fmt.Errorf("field %s required, but empty", col.Tag)
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
				if rest.Creating() && (!fv.IsValid() || utils.IsEmptyValue(fv)) {
					var userid string
					if uid := rest.GetEnv(USERID_KEY); uid == nil {
						userid = "0"
						//c.Debug("userid not exists")
					} else {
						userid = uid.(string)
						//c.Debug("userid: %s", userid)
					}
					switch fv.Type().String() {
					case "*string":
						fv.Set(reflect.ValueOf(&userid))
					case "string":
						fv.Set(reflect.ValueOf(userid))
					default:
						return nil, fmt.Errorf("field(%s) must be string, not %s", col.Tag, fv.Kind().String())
					}
				}
			case "time": //如果没有传值, 就是当前时间
				if rest.Creating() && (!fv.IsValid() || utils.IsEmptyValue(fv)) { //创建同时为空
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
				//if rest.Creating() { //创建时才检查,这里不够安全(将来改)
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
				if rest.Creating() {
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
				if rest.Creating() {
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
				if rest.Creating() { // 创建时加上内容
					if stag := rest.GetEnv(STAG_KEY).(string); stag != "" {
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
				if rest.Updating() {
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
	}
	return m, nil
}

/* }}} */

/* {{{ func (rest *REST) Protect() (Model, error)
 * 数据过滤
 */
func (rest *REST) Protect() (Model, error) {
	if m := rest.Model(); m != nil {
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
	} else {
		err := fmt.Errorf("not found model")
		//Info("error: %s", err)
		return nil, err
	}
}

/* }}} */

/* {{{ func (rest *REST) Row(ext ...interface{}) (Model, error)
 * 根据条件获取一条记录, model为表结构
 */
func (rest *REST) Row(ext ...interface{}) (Model, error) {
	var m Model
	if m = rest.Model(); m == nil {
		err := fmt.Errorf("not found model")
		Info("error: %s", err)
		return nil, err
	}
	//找rowkey
	if pf, pv, _ := m.PKey(); pv != "" {
		//Info("pk: %s", pv)
		m.SetConditions(NewCondition(CTYPE_IS, pf, pv))
	} else if len(ext) > 0 {
		if id, ok := ext[0].(string); ok && id != "" {
			m.SetConditions(NewCondition(CTYPE_IS, pf, id))
		}
	}
	bi, _ := m.ReadPrepare()
	builder := bi.(*gorp.Builder)
	ms := m.NewList()
	var err error
	err = builder.Select(GetDbFields(m)).Limit("1").Find(ms)
	if err != nil && err != sql.ErrNoRows {
		//支持出错
		return nil, err
	} else if ms == nil {
		//没找到记录
		return nil, ErrNoRecord
	}

	resultsValue := reflect.Indirect(reflect.ValueOf(ms))
	if resultsValue.Len() <= 0 {
		return nil, ErrNoRecord
	}
	//return rest.SetModel(resultsValue.Index(0).Interface().(Model)), nil
	return resultsValue.Index(0).Interface().(Model), nil
}

/* }}} */

/* {{{ func (rest *REST) CreateRow() (Model, error)
 * 根据条件获取一条记录, model为表结构
 */
func (rest *REST) CreateRow() (Model, error) {
	if m := rest.Model(); m != nil {
		db := rest.DBConn(WRITETAG)
		if err := db.Insert(m); err != nil { //Insert会把m换成新的
			return nil, err
		} else {
			rest.SetModel(m)
			return m, nil
		}
	} else {
		err := fmt.Errorf("not found model")
		//Info("error: %s", err)
		return nil, err
	}
}

/* }}} */

/* {{{ func (rest *REST) UpdateRow(ext ...interface{}) (affected int64, err error)
 * 更新record
 */
func (rest *REST) UpdateRow(ext ...interface{}) (affected int64, err error) {
	if m := rest.Model(); m != nil {
		id := ""
		if len(ext) > 0 {
			if rk, ok := ext[0].(string); ok && rk != "" {
				id = rk
			}
		} else if _, pv, _ := m.PKey(); pv != "" {
			id = pv
		}
		if id == "" {
			rest.Info("not found id")
			return 0, ErrNoRecord
		}
		db := rest.DBConn(WRITETAG)
		if id != "" {
			if err = utils.ImportValue(m, map[string]string{DBTAG_PK: id}); err != nil {
				return
			}
		} else {
			//Info("not_found_row")
			err = fmt.Errorf("not_found_row_to_update")
			return
		}
		return db.Update(m)
	} else {
		err = fmt.Errorf("not_found_model")
		return
	}
}

/* }}} */

/* {{{ func (rest *REST) DeleteRow(id string) (affected int64, err error)
 * 删除记录(逻辑删除)
 */
func (rest *REST) DeleteRow(id string) (affected int64, err error) {
	if m := rest.Model(); m != nil {
		db := rest.DBConn(WRITETAG)
		if err = utils.ImportValue(m, map[string]string{DBTAG_PK: id, DBTAG_LOGIC: "-1"}); err != nil {
			return
		}
		return db.Update(m)
	} else {
		err := fmt.Errorf("not found model")
		//Info("error: %s", err)
		return 0, err
	}
}

/* }}} */

/* {{{ func (rest *REST) Rows() (l *List, err error)
 * 获取list, 通用函数
 */
func (rest *REST) Rows() (l *List, err error) {
	if m := rest.Model(); m != nil {
		//c := rest.Context()
		l = new(List)
		bi, _ := rest.ReadPrepare()
		builder := bi.(*gorp.Builder)
		count, _ := builder.Count() //结果数
		ms := rest.NewList()
		if p := rest.Pagination(); p != nil {
			l.Info.Page = &p.Page
			l.Info.PerPage = &p.PerPage
			l.Info.Total = count
			err = builder.Select(GetDbFields(m, true)).Offset(p.Offset).Limit(p.PerPage).Find(ms)
			//c.Debug("[offset: %d][per_page: %d]", p.Offset, p.PerPage)
		} else {
			//rest.Debug("get fields: %v", GetDbFields(m, true))
			err = builder.Select(GetDbFields(m, true)).Find(ms)
		}
		if err != nil && err != sql.ErrNoRows {
			//支持出错
			return l, err
		} else if ms == nil {
			//没找到记录
			return l, ErrNoRecord
		}

		l.List = ms

		return l, nil
	} else {
		err := fmt.Errorf("not found model")
		//Info("error: %s", err)
		return nil, err
	}
}

/* }}} */

/* {{{ func (rest *REST) GetSum(d ...string) (l *List, err error)
 * 获取list, 通用函数
 */
func (rest *REST) GetSum(d ...string) (l *List, err error) {
	//c := m.GetCtx()
	if m := rest.Model(); m != nil {
		bi, _ := rest.ReadPrepare()
		builder := bi.(*gorp.Builder)

		l = new(List)

		group := make([]string, 0)
		ms := rest.NewList()
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

		ms = rest.NewList()

		if err = builder.Select(GetSumFields(m, group...)).Find(ms); err != nil {
			return l, err
		} else if ms == nil {
			return l, ErrNoRecord
		}

		listValue := reflect.Indirect(reflect.ValueOf(ms))
		l.Info.Total = int64(listValue.Len())

		l.List = ms

		return
	} else {
		err := fmt.Errorf("not found model")
		//Info("error: %s", err)
		return nil, err
	}
}

/* }}} */

/* {{{ func (rest *REST) GetCount() (cnt int64, err error)
 * 获取list, 通用函数
 */
func (rest *REST) GetCount() (cnt int64, err error) {
	if rest.Count > 0 {
		return rest.Count, nil
	} else {
		bi, _ := rest.ReadPrepare()
		builder := bi.(*gorp.Builder)
		return builder.Count()
	}
}

/* }}} */

/* {{{ func (rest *REST) GetCountNSum() (cnt int64, sum float64)
 * 获取计数以及求和, 通用函数
 */
func (rest *REST) GetCountNSum() (cnt int64, sum float64) {
	return rest.Count, rest.Sum
}

/* }}} */

/* {{{ func (rest *REST) GetOlder(opts ...string) Model
 * 获取旧记录
 */
func (rest *REST) GetOlder(opts ...string) Model {
	if rest.older == nil {
		if m := rest.Model(); m != nil {
			rk := ""
			if len(opts) > 0 && opts[0] != "" {
				rk = opts[0]
			} else if c := rest.Context(); c != nil {
				rk = c.Param(RowkeyKey)
			}
			if rk != "" {
				if older, err := m.Row(rk); err == nil {
					rest.older = older
				}
			}
		}
	}
	return rest.older
}

/* }}} */

/* {{{ func (rest *REST) AddTable(tags ...string)
 * 注册表结构
 */
func (rest *REST) AddTable(tags ...string) {
	if m := rest.Model(); m != nil {
		reflectVal := reflect.ValueOf(m)
		mv := reflect.Indirect(reflectVal).Interface()
		//Debug("table name: %s", rest.TableName())
		tb := rest.TableName()
		pf, _, ai := m.PKey()
		if !ai {
			//Debug("[pk not auto incr: %s]", pf)
		} else {
			//Debug("[pk auto incr: %s]", pf)
		}
		//Debug("table: %s", tb)
		gorp.AddTableWithName(mv, tb).SetKeys(ai, pf)

		//data accessor, 默认都是DBTAG
		DataAccessor[tb+"::"+WRITETAG] = DBTAG
		DataAccessor[tb+"::"+READTAG] = DBTAG
		if len(tags) > 0 {
			writeTag := tags[0]
			if dns := config.DB[writeTag]; dns != "" {
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
			if dns := config.DB[readTag]; dns != "" {
				Info("%s's reader: %s", tb, dns)
				if err := OpenDB(readTag, dns); err != nil {
					Warn("open db(%s) error: %s", readTag, err)
				} else {
					DataAccessor[tb+"::"+READTAG] = readTag
				}
			}
		}
	} else {
		err := fmt.Errorf("not found model")
		Info("[AddTable]: %s", err)
	}
}

/* }}} */

/* {{{ func (rest *REST) ReadPrepare() (interface{}, error)
 * 查询准备
 */
func (rest *REST) ReadPrepare() (interface{}, error) {
	var m Model
	if m = rest.Model(); m == nil {
		err := fmt.Errorf("not found model")
		Info("error: %s", err)
		return nil, err
	}
	db := rest.DBConn(READTAG)
	tb := rest.TableName()
	b := gorp.NewBuilder(db).Table(tb)
	cons := rest.Conditions()

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
					b.Where(fmt.Sprintf("T.`%s` BETWEEN ? AND ?", v.Field), vt.Start, vt.End)
				case TimeRange: //只支持timerange
					b.Where(fmt.Sprintf("T.`%s` BETWEEN ? AND ?", v.Field), vt.Start, vt.End)
				default:
					//nothing
				}
			}
			//排序
			if v.Order != nil {
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
				if vt, ok := v.Join.(*Condition); ok && vt.Is != nil {
					joinTable := v.Field // 字段名就是表名称
					joinField := vt.Field
					Debug("join %s.%s", joinTable, joinField)
					if t, ok := gorp.GetTable(joinTable); ok {
						if cols := utils.ReadStructColumns(reflect.New(t.Gotype).Interface(), true); cols != nil {
							for _, col := range cols {
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
	} else { //没有条件从自身找
		//Debug("find condition from struct")
		if cols := utils.ReadStructColumns(m, true); cols != nil {
			v := reflect.ValueOf(m)
			for _, col := range cols {
				fv := utils.FieldByIndex(v, col.Index)
				if (col.TagOptions.Contains(DBTAG_PK) || col.ExtOptions.Contains(TAG_CONDITION)) && fv.IsValid() && !utils.IsEmptyValue(fv) { //有值
					if fs := utils.GetRealString(fv); fs != "" {
						//Info("field: %s, value: %s", col.Tag, fs)
						// 多个字段有值, 用AND
						b.Where(fmt.Sprintf("T.`%s` = ?", col.Tag), fs)
					}
				}
			}
		}
	}

	if cols := utils.ReadStructColumns(m, true); cols != nil {
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

// 挖掘 model
func digModel(m Model) Model {
	rt := utils.RealType(m, reflect.TypeOf((*Model)(nil)).Elem())
	//Info("mtype: %v, real type: %v", mt, rt)
	return reflect.New(rt).Interface().(Model)
}
