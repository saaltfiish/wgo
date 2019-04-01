package gorp

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func (b *Builder) primaryCondiation(value interface{}) string {
	return fmt.Sprintf("(%v = %v)", b.Quote(b.PrimaryKey), value)
}

func (b *Builder) buildWhereCondition(clause map[string]interface{}) (str string) {
	switch value := clause["query"].(type) {
	case string:
		// if string is number
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return b.primaryCondiation(b.AddToVars(id))
		} else {
			str = value
		}
	case int, int64, int32:
		return b.primaryCondiation(b.AddToVars(value))
	case sql.NullInt64:
		return b.primaryCondiation(b.AddToVars(value.Int64))
	case []int64, []int, []int32, []string:
		str = fmt.Sprintf("(%v in (?))", b.Quote(b.PrimaryKey))
		clause["args"] = []interface{}{value}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v = %v)", b.Quote(key), b.AddToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			values := reflect.ValueOf(arg)
			var tempMarks []string
			for i := 0; i < values.Len(); i++ {
				tempMarks = append(tempMarks, b.AddToVars(values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
		default:
			if valuer, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = valuer.Value()
			}

			str = strings.Replace(str, "?", b.AddToVars(arg), 1)
		}
	}
	return
}

func (b *Builder) buildNotCondition(clause map[string]interface{}) (str string) {
	var notEqualSql string

	switch value := clause["query"].(type) {
	case string:
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", b.Quote(b.PrimaryKey), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			notEqualSql = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v NOT IN (?))", b.Quote(value))
			notEqualSql = fmt.Sprintf("(%v <> ?)", b.Quote(value))
		}
	case int, int64, int32:
		return fmt.Sprintf("(%v <> %v)", b.Quote(b.PrimaryKey), value)
	case []int64, []int, []int32, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v not in (?))", b.Quote(b.PrimaryKey))
			clause["args"] = []interface{}{value}
		} else {
			return ""
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v <> %v)", b.Quote(key), b.AddToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			values := reflect.ValueOf(arg)
			var tempMarks []string
			for i := 0; i < values.Len(); i++ {
				tempMarks = append(tempMarks, b.AddToVars(values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
		default:
			if scanner, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = scanner.Value()
			}
			str = strings.Replace(notEqualSql, "?", b.AddToVars(arg), 1)
		}
	}
	return
}

func (b *Builder) where(where ...interface{}) {
	if len(where) > 0 {
		b.search = b.search.where(where[0], where[1:]...)
	}
}

func (b *Builder) whereSql() (sql string) {
	var andConditions, orConditions []string

	for _, clause := range b.search.WhereConditions {
		andConditions = append(andConditions, b.buildWhereCondition(clause))
	}

	for _, clause := range b.search.OrConditions {
		orConditions = append(orConditions, b.buildWhereCondition(clause))
	}

	for _, clause := range b.search.NotConditions {
		andConditions = append(andConditions, b.buildNotCondition(clause))
	}

	orSql := strings.Join(orConditions, " OR ")
	combinedSql := strings.Join(andConditions, " AND ")
	if len(combinedSql) > 0 {
		if len(orSql) > 0 {
			combinedSql = combinedSql + " OR " + orSql
		}
		sql = "WHERE " + combinedSql
	} else if len(orSql) > 0 {
		sql = "WHERE " + orSql
	}
	return
}

func (b *Builder) selectSql() string {
	if len(b.search.Select) == 0 {
		return "*"
	} else {
		return b.search.Select
	}
}

func (b *Builder) orderSql() string {
	if len(b.search.Orders) == 0 {
		return ""
	} else {
		return " ORDER BY " + strings.Join(b.search.Orders, ",")
	}
}

func (b *Builder) limitSql() string {
	if len(b.search.Limit) == 0 {
		return ""
	} else {
		return " LIMIT " + b.search.Limit
	}
}

func (b *Builder) offsetSql() string {
	if len(b.search.Offset) == 0 {
		return ""
	} else {
		return " OFFSET " + b.search.Offset
	}
}

func (b *Builder) groupSql() string {
	if len(b.search.Group) == 0 {
		return ""
	} else {
		return " GROUP BY " + b.search.Group
	}
}

func (b *Builder) havingSql() string {
	if b.search.HavingCondition == nil {
		return ""
	} else {
		return " HAVING " + b.buildWhereCondition(b.search.HavingCondition)
	}
}

func (b *Builder) joinsSql() string {
	return b.search.Joins + " "
}

func (b *Builder) prepareQuerySql() {
	if b.search.Raw {
		b.Sql = strings.Replace(strings.TrimLeft(b.CombinedConditionSql(), "WHERE "), "$$", "?", -1)
	} else {
		b.Sql = strings.Replace(fmt.Sprintf("SELECT %v %v FROM %v %v", "", b.selectSql(), b.QuotedTableName(), b.CombinedConditionSql()), "$$", "?", -1)
	}
	return
}

func (b *Builder) inlineCondition(values ...interface{}) *Builder {
	if len(values) > 0 {
		b.search = b.search.where(values[0], values[1:]...)
	}
	return b
}

func (b *Builder) count() *Builder {
	b.search = b.search.selects("count(*)")
	b.prepareQuerySql()
	return b
}
