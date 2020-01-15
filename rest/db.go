package rest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"wgo"
	"wgo/gorp"
	"wgo/utils"
)

var (
	DataAccessor = make(map[string]string) // tablename::{read/write} => tag
)

//出/入库转换器
type SelfConverter interface {
	ToDb() (interface{}, error)                                             //入库
	FromDb(interface{}) (interface{}, func(interface{}, interface{}) error) //出库
}

type BaseConverter struct{}

/* {{{ func OpenDB(tag,dns string) error
 *
 */
func OpenDB(tag, dns string) (err error) {
	//Debug("open mysql: %s,%s", tag, dns)
	if wgo.Env().DebugMode {
		Debug("gorp debug on")
		gorp.TraceOn("gorp|", logger)
	}
	gorp.SetTypeConvert(BaseConverter{})
	if err = gorp.Open(tag, "mysql", dns); err != nil {
		//Debug("open error: %s", err)
	}
	return
}

/* }}} */

/* {{{ func (_ BaseConverter) ToDb(val interface{}) (interface{}, error)
 *
 */
func (_ BaseConverter) ToDb(val interface{}) (interface{}, error) {
	switch t := val.(type) {
	case *[]string, []string, *[]int, []int, map[string]string, *map[string]string, map[string]interface{}, *map[string]interface{}, map[interface{}]interface{}, []interface{}, *Array, Array: //转为字符串
		c, _ := json.Marshal(t)
		return string(c), nil
	case bool: // 转为数字
		if t == true {
			return 1, nil
		}
		return 0, nil
	case time.Time, *time.Time:
		// nothing
	default:
		// 自定义的类型,如果实现了SelfConverter接口,则这里自动执行
		// Info("not known val: %v, %v", reflect.TypeOf(t), val)
		if _, ok := val.(SelfConverter); ok {
			// Debug("custom ToDb, %s", reflect.TypeOf(t))
			return val.(SelfConverter).ToDb()
		} else if reflect.ValueOf(val).IsValid() {
			if _, ok := reflect.Indirect(reflect.ValueOf(val)).Interface().(SelfConverter); ok { // 如果采用了指针
				// Debug("prt selfconvert todb, %+v", reflect.TypeOf(t))
				return val.(SelfConverter).ToDb()
			}
		}
		// rest.Model, auto find primary key
		if m, ok := val.(Model); ok {
			// model type
			// Info("[ToDb]model value: %+v", m)
			if f, v, _ := primaryKey(m); f != "" && v != "" {
				// Debug("[ToDb]save primary key(%s: %s) to db", f, v)
				return v, nil
			} else {
				Info("[ToDb]not found model primary key(%s: %s)", f, v)
			}
		} else if typ := utils.ToType(val); typ.Kind() == reflect.Struct {
			// struct尝试marshal
			// Info("[ToDb]save struct(%s) to db", typ)
			c, _ := json.Marshal(t)
			return string(c), nil
		}
	}
	return val, nil
}

/* }}} */

/* {{{ func (_ BaseConverter) FromDb(target interface{}) (gorp.CustomScanner, bool)
 * 类型转换, 主要处理标准类型; 自定义类型通过SelfConverter实现
 */
func (_ BaseConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch t := target.(type) {
	case **time.Time:
		binder := func(holder, target interface{}) error {
			var err error
			if holder.(*sql.NullString).Valid {
				var dt time.Time
				str := holder.(*sql.NullString).String
				switch len(str) {
				case 10, 19, 21, 22, 23, 24, 25, 26: // up to "YYYY-MM-DD HH:MM:SS.MMMMMM"
					if str == base[:len(str)] {
						return nil
					}
					// Info("format: %s, str: %s, location: %s", timeFormat[:len(str)], str, wgo.Env().Location.String())
					dt, err = time.ParseInLocation(timeFormat[:len(str)], str, wgo.Env().Location)
				default:
					err = fmt.Errorf("Invalid Time-String: %s", str)
					return err
				}
				if err != nil {
					return err
				}
				//dt = dt.UTC()
				// dt = dt.Local()
				// dt = dt.In(wgo.Env().Location)
				*(target.(**time.Time)) = &dt
				return nil
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *time.Time:
		binder := func(holder, target interface{}) error {
			var err error
			if holder.(*sql.NullString).Valid {
				var dt time.Time
				str := holder.(*sql.NullString).String
				switch len(str) {
				case 10, 19, 21, 22, 23, 24, 25, 26: // up to "YYYY-MM-DD HH:MM:SS.MMMMMM"
					if str == base[:len(str)] {
						return nil
					}
					dt, err = time.ParseInLocation(timeFormat[:len(str)], str, wgo.Env().Location)
				default:
					err = fmt.Errorf("Invalid Time-String: %s", str)
					return err
				}
				if err != nil {
					return err
				}
				//dt = dt.UTC()
				// dt = dt.Local()
				// dt = dt.In(wgo.Env().Location)
				*(target.(*time.Time)) = dt
				return nil
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case **[]string:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st []string
				if str := holder.(*sql.NullString).String; str != "" {
					//Debug("str: %s", str)
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						//return err
						// unmarshal失败, 直接使用字符串
						st = []string{str}
					}
					*(target.(**[]string)) = &st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *[]string:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st []string
				if str := holder.(*sql.NullString).String; str != "" {
					//Debug("str: %s", str)
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						//return err
						st = []string{str}
					}
					*(target.(*[]string)) = st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *[]interface{}:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st []interface{}
				if str := holder.(*sql.NullString).String; str != "" {
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						// unmarshal失败, 直接使用字符串
						st = []interface{}{str}
					}
					*(target.(*[]interface{})) = st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case **Array:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st Array
				if str := holder.(*sql.NullString).String; str != "" {
					//Debug("str: %s", str)
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						//return err
						// unmarshal失败, 直接使用字符串
						st = Array{str}
					}
					*(target.(**Array)) = &st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *Array:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st Array
				if str := holder.(*sql.NullString).String; str != "" {
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						// unmarshal失败, 直接使用字符串
						st = Array{str}
					}
					*(target.(*Array)) = st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case **map[string]string:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st map[string]string
				if str := holder.(*sql.NullString).String; str != "" {
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						return err
					}
					*(target.(**map[string]string)) = &st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *map[string]string:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st map[string]string
				if str := holder.(*sql.NullString).String; str != "" {
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						return err
					}
					*(target.(*map[string]string)) = st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case **map[string]interface{}:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st map[string]interface{}
				if str := holder.(*sql.NullString).String; str != "" {
					//Debug("str: %s", str)
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						return err
					}
					*(target.(**map[string]interface{})) = &st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *map[string]interface{}:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st map[string]interface{}
				if str := holder.(*sql.NullString).String; str != "" {
					//Debug("str: %s", str)
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						return err
					}
					*(target.(*map[string]interface{})) = st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case *map[interface{}]interface{}:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				var st map[interface{}]interface{}
				if str := holder.(*sql.NullString).String; str != "" {
					//Debug("str: %s", str)
					if err := json.Unmarshal([]byte(str), &st); err != nil {
						return err
					}
					*(target.(*map[interface{}]interface{})) = st
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case **string:
		binder := func(holder, target interface{}) error {
			*t = &holder.(*sql.NullString).String
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	case **float64:
		binder := func(holder, target interface{}) error {
			*t = &holder.(*sql.NullFloat64).Float64
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullFloat64), Target: target, Binder: binder}, true
	case **int64:
		binder := func(holder, target interface{}) error {
			*t = &holder.(*sql.NullInt64).Int64
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullInt64), Target: target, Binder: binder}, true
	case *bool:
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullInt64).Valid {
				*(target.(*bool)) = false
				if v := holder.(*sql.NullInt64).Int64; v == 1 {
					*(target.(*bool)) = true
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullInt64), Target: target, Binder: binder}, true
	case *interface{}:
		// Info("here interface")
		binder := func(holder, target interface{}) error {
			if holder.(*sql.NullString).Valid {
				if str := holder.(*sql.NullString).String; str != "" {
					// 先尝试数组
					var st0 []interface{}
					// Info("interface str: %s", str)
					if err := json.Unmarshal([]byte(str), &st0); err != nil {
						// Debug("not array: %s", err)
						// 再尝试object
						var st1 map[string]interface{}
						if err := json.Unmarshal([]byte(str), &st1); err != nil {
							// Debug("not object: %s", err, str)
							*t = &holder.(*sql.NullString).String
						} else {
							*(target.(*interface{})) = st1
						}
					} else {
						*(target.(*interface{})) = st0
					}
				}
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
	default:
		// 自定义的类型,如果实现了SelfConverter接口,则这里自动执行
		if t, ok := target.(SelfConverter); ok {
			//Debug("selfconvert begin(value)")
			holder, binder := t.FromDb(target)
			return gorp.CustomScanner{Holder: holder, Target: target, Binder: binder}, true
		} else if t, ok := reflect.Indirect(reflect.ValueOf(target)).Interface().(SelfConverter); ok { //如果采用了指针, 则到这里
			//Trace("ptr converter: %s", target)
			holder, binder := t.FromDb(target)
			return gorp.CustomScanner{Holder: holder, Target: target, Binder: binder}, true
		}
		// check if model
		if reflect.TypeOf(target).Implements(modelType) || reflect.TypeOf(target).Elem().Implements(modelType) {
			// model type
			// Debug("[FromDb]found model: %+v, auto fetch primary key", reflect.TypeOf(target))
			binder := func(holder, target interface{}) error {
				var s string
				if holder.(*sql.NullString).Valid {
					s = holder.(*sql.NullString).String
				}
				if s != "" && s != "0" {
					nm := digModel(target)
					if err := utils.ImportValue(nm, map[string]string{DBTAG_PK: s}); err != nil {
						Warn("[FromDb]import value failed: %s", err)
					}
					// Info("[FromDb]new model, primary key: %s, value: %+v", s, nm)
					// if reflect.TypeOf(target).Elem().Implements(modelType) {
					if reflect.TypeOf(target).Implements(modelType) {
						// 非指针Model
						reflect.Indirect(reflect.ValueOf(target)).Set(reflect.Indirect(reflect.ValueOf(nm)))
						// reflect.Indirect(reflect.ValueOf(target)).Set(reflect.ValueOf(nm))
					} else {
						// 指针Model
						reflect.Indirect(reflect.ValueOf(target)).Set(reflect.ValueOf(nm))
					}
				}
				return nil
			}
			return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
		} else if typ := utils.ToType(target); typ.Kind() == reflect.Struct {
			binder := func(holder, target interface{}) error {
				if holder.(*sql.NullString).Valid {
					if str := holder.(*sql.NullString).String; str != "" {
						// 再尝试object
						st := reflect.New(typ).Interface()
						if err := json.Unmarshal([]byte(str), st); err == nil {
							if reflect.TypeOf(target).Elem().Kind() == reflect.Ptr {
								// 指针型struct
								reflect.Indirect(reflect.ValueOf(target)).Set(reflect.ValueOf(st))
							} else {
								reflect.Indirect(reflect.ValueOf(target)).Set(reflect.Indirect(reflect.ValueOf(st)))
							}
						} else {
							Info("[FromDb]fetch struct(%s) from db failed: %s", typ, err)
						}
					}
				}
				return nil
			}
			return gorp.CustomScanner{Holder: new(sql.NullString), Target: target, Binder: binder}, true
		}
	}
	return gorp.CustomScanner{}, false
}

/* }}} */

// transaction
type Transaction struct {
	savepoints []string
	committed  bool

	*gorp.Transaction
}

func (t *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.Transaction.Exec(query, args...)
}

// commit当前的savepoint, 如果没有savepoint, 则直接commit整个transaction
func (t *Transaction) Commit() error {
	if len(t.savepoints) > 0 {
		// release current savepoint
		return t.ReleaseSavepoint()
	}
	return t.CommitAll()
}

// 全面commit
func (t *Transaction) CommitAll() error {
	t.committed = true
	return t.Transaction.Commit()
}

func (t *Transaction) Committed() bool {
	return t.committed
}

func (t *Transaction) RollbackAll() error {
	t.committed = true
	return t.Transaction.Rollback()
}

func (t *Transaction) Rollback() error {
	if len(t.savepoints) > 0 {
		// release current savepoint
		sp := t.savepoints[len(t.savepoints)-1]
		t.savepoints = t.savepoints[:len(t.savepoints)-1]
		return t.Transaction.RollbackToSavepoint(sp)
	}
	return t.RollbackAll()
}

func (t *Transaction) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return t.Transaction.Get(i, keys...)
}

func (t *Transaction) Savepoint(name string) error {
	t.savepoints = append(t.savepoints, name)
	return t.Transaction.Savepoint(name)
}

func (t *Transaction) ReleaseSavepoint() error {
	if len(t.savepoints) > 0 {
		sp := t.savepoints[len(t.savepoints)-1]
		t.savepoints = t.savepoints[:len(t.savepoints)-1]
		return t.Transaction.ReleaseSavepoint(sp)
	}
	return fmt.Errorf("not found savepoint")
}

func (t *Transaction) SelectInt(query string, args ...interface{}) (int64, error) {
	return t.Transaction.SelectInt(query, args...)
}

func (t *Transaction) Insert(list ...interface{}) error {
	return t.Transaction.Insert(list...)
}

func (t *Transaction) Update(list ...interface{}) (int64, error) {
	return t.Transaction.Update(list...)
}
