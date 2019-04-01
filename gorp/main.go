package gorp

import (
	"bytes"
	"database/sql"
	"fmt"
	"regexp"
	//"strings"
	"sync"
)

var (
	dbMap   *DbMap             = &DbMap{Dialect: MySQLDialect{}, tables: map[string]*TableMap{}}
	dbcache map[string]*sql.DB = map[string]*sql.DB{}
	m       *sync.Mutex        = &sync.Mutex{}
)

func Open(alias, driver, datasource string, param ...int) error {
	if _, ok := dbcache[alias]; ok {
		//panic(fmt.Errorf("DbMap with alias `%s` already opened!"))
		return fmt.Errorf("DbMap with alias `%s` already opened!")
	}
	db, err := sql.Open(driver, datasource)
	if err != nil {
		return err
	}
	switch len(param) {
	case 1:
		db.SetMaxIdleConns(param[0])
	case 2:
		db.SetMaxIdleConns(param[0])
		db.SetMaxOpenConns(param[1])
	default:
		if len(param) > 2 {
			db.SetMaxIdleConns(param[0])
			db.SetMaxOpenConns(param[1])
		}
	}

	dbcache[alias] = db

	return nil
}

func Using(alias string) *DbMap {
	m.Lock()
	defer m.Unlock()
	db, ok := dbcache[alias]
	if !ok {
		panic(fmt.Errorf("cannot find DbMap with alias `%s`", alias))
	}
	if dbMap.Db != db {
		return &DbMap{Db: db, Dialect: dbMap.Dialect, tables: dbMap.tables, logger: dbMap.logger, TypeConverter: dbMap.TypeConverter, logPrefix: dbMap.logPrefix}
	}
	dbMap.Db = db
	return dbMap
}

func AddTable(i interface{}) *TableMap {
	return AddTableWithName(i, "")
}

func AddTableWithName(i interface{}, name string) *TableMap {
	return AddTableWithNameAndSchema(i, "", name)
}

func AddTableWithNameAndSchema(i interface{}, schema string, name string) *TableMap {
	return dbMap.AddTableWithNameAndSchema(i, schema, name)
}

func SetTypeConvert(t TypeConverter) {
	dbMap.TypeConverter = t
}

func TraceOn(prefix string, logger GorpLogger) {
	dbMap.TraceOn(prefix, logger)
}

func TraceOff() {
	dbMap.TraceOff()
}

func GetTable(name string) (*TableMap, bool) {
	t, ok := dbMap.tables[name]
	return t, ok
}

var UNDERSCORE_PATTERN_1 = regexp.MustCompile("([A-Z]+)([A-Z][a-z])")
var UNDERSCORE_PATTERN_2 = regexp.MustCompile("([a-z\\d])([A-Z])")

//func underscore(camelCaseWord string) string {
//	underscoreWord := UNDERSCORE_PATTERN_1.ReplaceAllString(camelCaseWord, "${1}_${2}")
//	underscoreWord = UNDERSCORE_PATTERN_2.ReplaceAllString(underscoreWord, "${1}_${2}")
//	underscoreWord = strings.Replace(underscoreWord, "-", "_", 0)
//	underscoreWord = strings.ToLower(underscoreWord)
//	return underscoreWord
//}
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
