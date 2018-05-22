// Package utils provides ...
package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Tag struct {
	Name    string
	Options TagOptions
}

type Prefix struct {
	Tag   string // 针对某个tag的prefix, 如果为空则为全部
	Value string // prefix内容
}

type StructField struct {
	Name      string
	Type      reflect.Type
	Index     []int
	Path      string
	Tags      map[string]Tag
	SubFields StructFields
}
type StructFields []StructField

type StructColumn struct {
	Tag        string
	Name       string
	TagOptions TagOptions
	ExtTag     string
	ExtOptions TagOptions
	Type       reflect.Type
	Index      []int
}

/* {{{ func ReadStructFields(i interface{}, tags ...string) (fields []string)
 * 从struct type中读取字段名
 */
func ReadStructFields(i interface{}, def bool, tags ...string) (fields StructFields) {
	if t := toType(i); t.Kind() != reflect.Struct {
		return
	} else {
		return typeStructFields(t, def, []int{}, "", tags...)
	}
}

/* }}} */

/* {{{ func typeStructFields(t reflect.Type, def bool, idx []int, prefix string, tags ...string) (fields StructFields)
 * 从struct中读取字段名
 */
func typeStructFields(t reflect.Type, def bool, idx []int, prefix string, tags ...string) (fields StructFields) {
	n := t.NumField()
	for i := 0; i < n; i++ {
		index := append(idx, i)
		f := t.Field(i)
		if fc := f.Name[0]; fc < 'A' || fc > 'Z' {
			// 非大写, 略过
			continue
		}
		//fmt.Printf("type: %v, org: %v\n", f.Type.Elem().Kind(), f.Type)
		// prefix逐级递增继承, 除非设置了prefix:",skip"
		pf := prefix
		npf := "" // next prefix
		if pfs := f.Tag.Get("prefix"); pfs != "" {
			p, po := ParseTag(pfs)
			if po.Contains("skip") { // 忽略上级prefix
				pf = ""
			} else {
				npf = pf + p
			}
		}
		if f.Anonymous && f.Type.Kind() == reflect.Struct { //匿名struct, 层级为当前
			// Recursively add nested fields in embedded structs.
			nestedFields := typeStructFields(f.Type, def, index, npf, tags...)
			//field.SubFields = subfields
			fields = append(fields, nestedFields...)
		} else {
			field := StructField{
				Type:  f.Type,
				Index: index,
			}
			// parse tag
			ts := make(map[string]Tag)
			if len(tags) > 0 {
				for _, key := range tags {
					if tagString := f.Tag.Get(key); tagString != "" {
						tn, tops := ParseTag(tagString)
						if key == "json" && tn != "" { // json 忽略prefix
						} else if tn == "" && def { // tag name没有指定并且def ==true, 则用字段名转下划线方式为tag name
							tn = pf + Underscore(f.Name)
						} else {
							tn = pf + tn
						}
						ts[key] = Tag{Name: tn, Options: tops}
					} else if def { // 给个默认
						ts[key] = Tag{Name: pf + Underscore(f.Name)}
					} else {
						ts[key] = Tag{}
					}
				}
			}
			// struct field
			field.Name = f.Name
			field.Tags = ts
			if f.Type.String() != "*time.Time" && f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
				//fmt.Printf("type: %v\n", f.Type.Elem())
				field.SubFields = typeStructFields(f.Type.Elem(), def, index, npf, tags...)
			} else if f.Type.String() != "time.Time" && f.Type.Kind() == reflect.Struct {
				field.SubFields = typeStructFields(f.Type, def, index, npf, tags...)
			} else if f.Type.Kind() == reflect.Slice {
				mt := f.Type.Elem() // member type
				if mt.String() != "*time.Time" && mt.Kind() == reflect.Ptr && mt.Elem().Kind() == reflect.Struct {
					field.SubFields = typeStructFields(mt.Elem(), def, index, npf, tags...)
				} else if mt.String() != "time.Time" && mt.Kind() == reflect.Struct {
					field.SubFields = typeStructFields(mt, def, index, npf, tags...)
				}
			}
			fields = append(fields, field)
		}
	}
	return
}

/* }}} */

// ScanStructFields
func ScanStructFields(fs StructFields, tag, prefix, path string) (fields StructFields) {

	if prefix != "" {
		prefix = prefix + "."
	}
	for _, field := range fs {
		if path != "" {
			field.Path = path
		}
		tv := field.Tags[tag]
		tv.Name = prefix + tv.Name
		field.Tags[tag] = tv
		fields = append(fields, field)
		if len(field.SubFields) > 0 {
			newPath := path
			if field.Type.Kind() == reflect.Slice {
				// new path
				newPath = tv.Name
			}
			fields = append(fields, ScanStructFields(field.SubFields, tag, tv.Name, newPath)...)
		}
	}

	return
}

/* {{{ func ReadStructColumns(i interface{}, underscore bool, tags ...string) (cols []string)
 * 从struct type中读取字段名
 * 默认从struct的FieldName读取, 如果tag里有db, 则以db为准
 */
func ReadStructColumns(i interface{}, underscore bool, tags ...string) (cols []StructColumn) {
	if t := toType(i); t.Kind() != reflect.Struct {
		return
	} else {
		return typeStructColumns(t, underscore, tags...)
	}
}

/* }}} */

/* {{{ func FieldByIndex(v reflect.Value, index []int) reflect.Value
 * 通过索引返回field
 */
func FieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

/* }}} */

/* {{{ func FieldByName(i interface{}, field string) reflect.Value
 * 找到第一个符合名字的字段
 */
func FieldByName(i interface{}, field string) reflect.Value {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.FieldByName(field)
}

/* }}} */

/* {{{ func FieldByType(v reflect.Value, field string) reflect.Value
 * 找到实现了interface的真正类型(匿名字段)
 */
func RealType(i interface{}, typ reflect.Type) reflect.Type {
	//v := reflect.ValueOf(i)
	t := toType(i)
	if t.Kind() == reflect.Struct {
		n := t.NumField()
		for i := 0; i < n; i++ {
			f := t.Field(i)
			//if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Type.Implements(typ) { //匿名struct , 也就是嵌套
			if f.Anonymous && (f.Type.Implements(typ) || reflect.PtrTo(f.Type).Implements(typ)) {
				return f.Type
			}
		}
	}
	return t
}

/* }}} */

/* {{{ func ImportVal(i interface{}, import map[string]string) (err error)
 * 将tag匹配的值导入结构
 */
func ImportValue(i interface{}, is map[string]string) (err error) {
	v := reflect.ValueOf(i)
	if cols := ReadStructColumns(i, true); cols != nil {
		for _, col := range cols {
			for tag, iv := range is {
				if col.TagOptions.Contains(tag) {
					fv := FieldByIndex(v, col.Index)
					switch fv.Type().String() {
					case "*string":
						fv.Set(reflect.ValueOf(&iv))
					case "string":
						fv.Set(reflect.ValueOf(iv))
					case "*int64":
						pv, _ := strconv.ParseInt(iv, 10, 64)
						fv.Set(reflect.ValueOf(&pv))
					case "int64":
						pv, _ := strconv.ParseInt(iv, 10, 64)
						fv.Set(reflect.ValueOf(pv))
					case "*int":
						tv, _ := strconv.ParseInt(iv, 10, 0)
						pv := int(tv)
						fv.Set(reflect.ValueOf(&pv))
					case "int":
						tv, _ := strconv.ParseInt(iv, 10, 0)
						pv := int(tv)
						fv.Set(reflect.ValueOf(pv))
					default:
						err = fmt.Errorf("field(%s) not support %s", col.Tag, fv.Kind().String())
					}
				}
			}
		}
	}
	return
}

/* }}} */

/* {{{ func typeStructColumns(t reflect.Type, underscore bool, tags ...string) (cols []StructColumn)
 * 从struct中读取字段名
 * 默认从struct的FieldName读取, 如果tag里有db, 则以db为准
 */
func typeStructColumns(t reflect.Type, underscore bool, tags ...string) (cols []StructColumn) {
	tag := "db"        // 默认tag是"db"
	extTag := "filter" // 默认扩展tag是filter
	if len(tags) > 0 {
		tag = tags[0]
	}
	if len(tags) > 1 {
		extTag = tags[1]
	}
	n := t.NumField()
	for i := 0; i < n; i++ {
		index := make([]int, 0)
		f := t.Field(i)
		index = append(index, i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct { //匿名struct , 也就是嵌套
			// Recursively add nested fields in embedded structs.
			subcols := typeStructColumns(f.Type, underscore, tags...)
			// 如果重名则不append, drop
			for _, subcol := range subcols {
				shouldAppend := true
				for _, col := range cols {
					if subcol.Tag == col.Tag {
						shouldAppend = false
						break
					}
				}
				if shouldAppend {
					for _, ii := range subcol.Index {
						subcol.Index = append(index, ii)
					}
					cols = append(cols, subcol)
				}
			}
		} else {
			// parse tag
			ts, tops := ParseTag(f.Tag.Get(tag))
			if ts == "" { //为空,则取字段名
				if underscore {
					ts = Underscore(f.Name)
				} else {
					ts = f.Name
				}
			}
			// parse exttag
			extTs, extTops := ParseTag(f.Tag.Get(extTag))
			// struct col
			sc := StructColumn{
				Tag:        ts,
				Name:       f.Name,
				TagOptions: tops,
				ExtTag:     extTs,
				ExtOptions: extTops,
				Type:       f.Type,
				Index:      index,
			}
			//检查同名,有则覆盖
			shouldAppend := true
			for index, col := range cols {
				if col.Tag == sc.Tag {
					cols[index] = sc
					shouldAppend = false
					break
				}
			}
			if shouldAppend {
				cols = append(cols, sc)
			}
		}
	}
	return
}

/* }}} */

/* {{{ func toType(i interface{}) reflect.Type
 * 如果是指针, 则调用Elem()至Type为止, 如果Type不是struct, 报错
 */
func toType(i interface{}) reflect.Type {
	var t reflect.Type
	if tt, ok := i.(reflect.Type); ok {
		t = tt
	} else {
		t = reflect.TypeOf(i)
	}

	// If a Pointer to a type, follow
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	//if t.Kind() != reflect.Struct {
	//	return nil, fmt.Errorf("utils: Cannot SELECT into this type: %v", reflect.TypeOf(i))
	//}
	return t
}

/* }}} */

/* {{{ Underscore
 * 小程序, 把驼峰式转化为匈牙利式
 */
func Underscore(camelCaseWord string) string {
	underscoreWord := regexp.MustCompile("([A-Z]+)([A-Z][a-z])").ReplaceAllString(camelCaseWord, "${1}_${2}")
	underscoreWord = regexp.MustCompile("([a-z\\d])([A-Z])").ReplaceAllString(underscoreWord, "${1}_${2}")
	underscoreWord = strings.Replace(underscoreWord, "-", "_", 0)
	underscoreWord = strings.ToLower(underscoreWord)
	return underscoreWord
}

/* }}} */

/* {{{ func IsEmptyValue(v reflect.Value) bool
 *
 */
func IsEmptyValue(v reflect.Value) bool {
	const deref = false
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	//case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	//	return v.Int() == 0
	//case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
	//	return v.Uint() == 0
	//case reflect.Float32, reflect.Float64:
	//	return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		if deref {
			if v.IsNil() {
				return true
			}
			return IsEmptyValue(v.Elem())
		} else {
			return v.IsNil()
		}
	case reflect.Struct:
		// return true if all fields are empty. else return false.
		return v.Interface() == reflect.Zero(v.Type()).Interface()
		// for i, n := 0, v.NumField(); i < n; i++ {
		// 	if !isEmptyValue(v.Field(i), deref) {
		// 		return false
		// 	}
		// }
		// return true
	}
	return false
}

/* }}} */

/* {{{ func GetRealString(v reflect.Value) string
 *
 */
func GetRealString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', 2, 64)
	case reflect.Ptr:
		if v.IsNil() {
			return ""
		}
		return GetRealString(v.Elem())
	default:
		//nothing
	}
	return ""
}

/* }}} */

/* {{{ func FieldExists(i interface{},f string) bool
 * 判断一个结构变量是否有某个字段
 */
func FieldExists(i interface{}, f string) bool {
	r := reflect.ValueOf(i)
	fv := reflect.Indirect(r).FieldByName(f)
	if !fv.IsValid() {
		return false
	} else {
		return true
	}
}

/* }}} */

// 确保一个struct指针不为空
func Instance(ob interface{}) interface{} {
	if IsEmptyValue(reflect.ValueOf(ob)) {
		if reflect.TypeOf(ob).Kind() == reflect.Ptr {
			return reflect.New(reflect.TypeOf(ob).Elem()).Interface()
		} else {
			return reflect.New(reflect.TypeOf(ob)).Interface()
		}
	}
	return ob
}

// StringMap, turn a struct to map[string]string
func StringMap(i interface{}, opts ...string) map[string]string {
	tag := "json"
	must := ""
	if len(opts) > 0 && opts[0] != "" {
		tag = opts[0]
	}
	if len(opts) > 1 && opts[1] != "" {
		must = opts[1]
	}
	sm := make(map[string]string)
	if fields := ReadStructFields(i, true, tag); fields != nil {
		v := reflect.ValueOf(i)
		for _, field := range fields {
			fn := field.Tags[tag].Name
			fv := FieldByIndex(v, field.Index)
			options := field.Tags[tag].Options
			fts := fv.Type().String()
			// fmt.Printf("field name: %s, value: %+v, type: %s\n", fn, fv, fts)
			if (must == "" || options.Contains(must)) && (!options.Contains("omitempty") || !IsEmptyValue(fv)) {
				// 不为空 or 没有设置`omitempty`
				switch fts {
				case "string":
					sm[fn] = fv.String()
				case "*string":
					sm[fn] = fv.Elem().String()
				case "int", "int64":
					fvi := fv.Int()
					if !options.Contains("omitempty") || fvi > 0 { // IsEmptyValue没办法判断int, int64
						sm[fn] = strconv.FormatInt(fvi, 10)
					}
				case "*int", "*int64":
					fvi := fv.Elem().Int()
					sm[fn] = strconv.FormatInt(fvi, 10)
				default: // json encode
					fvb, _ := json.Marshal(fv.Interface())
					sm[fn] = string(fvb)
				}
			}
		}
	}
	return sm
}

// convert interface{} to struct
func Convert(i interface{}, o interface{}) (interface{}, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	fmt.Printf("converting: %s\n", string(b))
	err = json.Unmarshal(b, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}
