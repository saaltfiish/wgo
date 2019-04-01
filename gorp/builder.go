package gorp

import "time"
import "fmt"

// NowFunc returns current time, this function is exported in order to be able
// to give the flexiblity to the developer to costumize it accoring to their
// needs
//
//   e.g: return time.Now().UTC()
//
var NowFunc = func() time.Time {
	return time.Now()
}

type Builder struct {
	search     *search
	DbMap      *DbMap
	PrimaryKey string
	Sql        string
	SqlVars    []interface{}
	err        error
}

func (b *Builder) Err(e error) {
	b.err = e
}

func NewBuilder(dbMap *DbMap) *Builder {
	b := &Builder{}
	b.DbMap = dbMap
	b.search = &search{builder: b}
	return b
}

func (b *Builder) clone() *Builder {
	return &Builder{search: b.search.clone(), DbMap: b.DbMap, PrimaryKey: b.PrimaryKey}
}

func (b *Builder) AddToVars(value interface{}) string {
	b.SqlVars = append(b.SqlVars, value)
	return "$$"
}

func (b *Builder) Where(query interface{}, args ...interface{}) *Builder {
	return b.search.where(query, args...).builder
}

func (b *Builder) Or(query interface{}, args ...interface{}) *Builder {
	return b.search.or(query, args...).builder
}

func (b *Builder) Not(query interface{}, args ...interface{}) *Builder {
	return b.search.not(query, args...).builder
}

func (b *Builder) Limit(value interface{}) *Builder {
	return b.search.limit(value).builder
}

func (b *Builder) Offset(value interface{}) *Builder {
	return b.search.offset(value).builder
}

func (b *Builder) Order(value string, reorder ...bool) *Builder {
	return b.search.order(value, reorder...).builder
}

func (b *Builder) Select(value interface{}) *Builder {
	return b.search.selects(value).builder
}

//func (b *Builder) Group(query string) *Builder {
func (b *Builder) Group(value interface{}) *Builder {
	return b.search.group(value).builder
}

func (b *Builder) Having(query string, values ...interface{}) *Builder {
	return b.search.having(query, values...).builder
}

func (b *Builder) Joins(query string) *Builder {
	return b.search.joins(query).builder
}

func (b *Builder) Includes(value interface{}) *Builder {
	return b.search.includes(value).builder
}

func (b *Builder) Unscoped() *Builder {
	return b.search.unscoped().builder
}

func (b *Builder) Raw(sql string, values ...interface{}) *Builder {
	return b.search.raw(true).where(sql, values...).builder
}

func (b *Builder) Count() (int64, error) {
	x := b.clone()
	x.count()
	return b.DbMap.SelectInt(x.Sql, x.SqlVars...)
}

func (b *Builder) Find(i interface{}) error {
	x := b.clone()
	x.prepareQuerySql()
	_, err := b.DbMap.Select(i, x.Sql, x.SqlVars...)
	return err
}

func (b *Builder) Table(name string) *Builder {
	if t, ok := GetTable(name); ok {
		if len(t.keys) > 0 {
			b.PrimaryKey = t.keys[0].ColumnName
		}
		return b.search.table(name).builder
	}
	panic(fmt.Errorf("unknow table `%s`", name))
}

func (b *Builder) QuotedTableName() string {
	//return b.search.TableName
	return b.Quote(b.search.TableName)
}

func (b *Builder) Quote(str string) string {
	return fmt.Sprintf("`%s` T", str)
}

func (b *Builder) CombinedConditionSql() string {
	return b.joinsSql() + b.whereSql() + b.groupSql() +
		b.havingSql() + b.orderSql() + b.limitSql() + b.offsetSql()
}
