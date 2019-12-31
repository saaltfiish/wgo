// Package utils provides ...
package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNilArguments                 = errors.New("src and dst must not be nil")
	ErrDifferentArgumentsTypes      = errors.New("src and dst must be of same type")
	ErrNotSupported                 = errors.New("only structs and maps are supported")
	ErrExpectedMapAsDestination     = errors.New("dst was expected to be a map")
	ErrExpectedPointerAsDestination = errors.New("dst was expected to be a pointer")
	ErrExpectedStructAsDestination  = errors.New("dst was expected to be a struct")
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
	if t := ToType(i); t.Kind() != reflect.Struct {
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

func hasExportedField(dst reflect.Value) (exported bool) {
	for i, n := 0, dst.NumField(); i < n; i++ {
		field := dst.Type().Field(i)
		if field.Anonymous && dst.Field(i).Kind() == reflect.Struct {
			exported = exported || hasExportedField(dst.Field(i))
		} else {
			exported = exported || len(field.PkgPath) == 0
		}
	}
	return
}

// During deepMerge, must keep track of checks that are
// in progress.  The comparison algorithm assumes that all
// checks in progress are true when it reencounters them.
// Visited are stored in a map indexed by 17 * a1 + a2;
type visit struct {
	ptr  uintptr
	typ  reflect.Type
	next *visit
}

// Traverses recursively both values, assigning src's fields values to dst.
// The map argument tracks comparisons that have already been seen, which allows
// short circuiting on recursive types.
func deepMerge(dst, src reflect.Value, visited map[uintptr]*visit, depth int) (err error) {
	overwrite := false
	overwriteWithEmptySrc := false

	if !src.IsValid() {
		return
	}
	if dst.CanAddr() {
		addr := dst.UnsafeAddr()
		h := 17 * addr
		seen := visited[h]
		typ := dst.Type()
		for p := seen; p != nil; p = p.next {
			if p.ptr == addr && p.typ == typ {
				return nil
			}
		}
		// Remember, remember...
		visited[h] = &visit{addr, typ, seen}
	}

	switch dst.Kind() {
	case reflect.Struct:
		if hasExportedField(dst) {
			for i, n := 0, dst.NumField(); i < n; i++ {
				if err = deepMerge(dst.Field(i), src.Field(i), visited, depth+1); err != nil {
					return
				}
			}
		} else {
			if dst.CanSet() && (!IsEmptyValue(src) || overwriteWithEmptySrc) && (overwrite || IsEmptyValue(dst)) {
				dst.Set(src)
			}
		}
	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
		if src.IsNil() {
			break
		}

		if dst.Kind() != reflect.Ptr && src.Type().AssignableTo(dst.Type()) {
			if dst.IsNil() && dst.CanSet() {
				dst.Set(src)
			}
			break
		}

		if src.Kind() != reflect.Interface {
			if dst.IsNil() && dst.CanSet() {
				dst.Set(src)
			} else if src.Kind() == reflect.Ptr {
				if err = deepMerge(dst.Elem(), src.Elem(), visited, depth+1); err != nil {
					return
				}
			} else if dst.Elem().Type() == src.Type() {
				if err = deepMerge(dst.Elem(), src, visited, depth+1); err != nil {
					return
				}
			} else {
				return ErrDifferentArgumentsTypes
			}
			break
		}
		if dst.IsNil() && dst.CanSet() {
			dst.Set(src)
		} else if err = deepMerge(dst.Elem(), src.Elem(), visited, depth+1); err != nil {
			return
		}
	default:
		if dst.CanSet() && (!IsEmptyValue(src) || overwriteWithEmptySrc) && (overwrite || IsEmptyValue(dst)) {
			dst.Set(src)
		}
	}
	return
}

func resolveValues(dst, src interface{}) (vDst, vSrc reflect.Value, err error) {
	if dst == nil || src == nil {
		err = ErrNilArguments
		return
	}
	vDst = reflect.ValueOf(dst)
	if vDst.Kind() != reflect.Ptr {
		err = ErrExpectedPointerAsDestination
		return
	}
	vDst = vDst.Elem()
	if vDst.Kind() != reflect.Struct {
		err = ErrNotSupported
		return
	}
	vSrc = reflect.ValueOf(src)
	// We check if vSrc is a pointer to dereference it.
	if vSrc.Kind() == reflect.Ptr {
		vSrc = vSrc.Elem()
	}
	return
}

// merge struct
func Merge(dst, src interface{}) error {
	var (
		vDst, vSrc reflect.Value
		err        error
	)

	if vDst, vSrc, err = resolveValues(dst, src); err != nil {
		return err
	}
	if vDst.Type() != vSrc.Type() {
		return ErrDifferentArgumentsTypes
	}
	return deepMerge(vDst, vSrc, make(map[uintptr]*visit), 0)
}

/* {{{ func ReadStructColumns(i interface{}, underscore bool, tags ...string) (cols []string)
 * 从struct type中读取字段名
 * 默认从struct的FieldName读取, 如果tag里有db, 则以db为准
 */
func ReadStructColumns(i interface{}, underscore bool, tags ...string) (cols []StructColumn) {
	if t := ToType(i); t.Kind() != reflect.Struct {
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
	if v.Kind() == reflect.Struct {
		return v.FieldByName(field)
	}
	return reflect.Value{}
}

/* }}} */

/* {{{ func ImportByField(target)
 * 将值注入struct字段
 */
func ImportByField(i interface{}, v interface{}, field string) error {
	if field == "" {
		return errors.New("invalid field")
	}
	if fv := FieldByName(i, field); fv.IsValid() && fv.CanSet() {
		ft := ToType(fv.Type())
		vt := ToType(v)
		if ft != vt {
			return errors.New("type mismatch")
		}
		vv := reflect.ValueOf(v)
		switch {
		case fv.Kind() == vv.Kind():
			fv.Set(vv)
		case fv.Kind() == reflect.Ptr:
			fv.Set(vv.Addr())
		default:
			fv.Set(vv.Elem())
		}
	}
	return nil
}

/* }}} */

/* {{{ func FieldByType(v reflect.Value, field string) reflect.Value
 * 找到实现了interface的真正类型(唯一匿名字段)
 */
func RealType(i interface{}, typ reflect.Type) reflect.Type {
	//v := reflect.ValueOf(i)
	t := ToType(i)
	if t.Kind() == reflect.Struct {
		if n := t.NumField(); n == 1 {
			f := t.Field(0)
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
					// fv := FieldByIndex(v, col.Index)
					// switch fv.Type().String() {
					// case "*string":
					// 	fv.Set(reflect.ValueOf(&iv))
					// case "string":
					// 	fv.Set(reflect.ValueOf(iv))
					// case "*int64":
					// 	pv, _ := strconv.ParseInt(iv, 10, 64)
					// 	fv.Set(reflect.ValueOf(&pv))
					// case "int64":
					// 	pv, _ := strconv.ParseInt(iv, 10, 64)
					// 	fv.Set(reflect.ValueOf(pv))
					// case "*int":
					// 	tv, _ := strconv.ParseInt(iv, 10, 0)
					// 	pv := int(tv)
					// 	fv.Set(reflect.ValueOf(&pv))
					// case "int":
					// 	tv, _ := strconv.ParseInt(iv, 10, 0)
					// 	pv := int(tv)
					// 	fv.Set(reflect.ValueOf(pv))
					// default:
					// 	err = fmt.Errorf("field(%s) not support %s", col.Tag, fv.Kind().String())
					// }
					err = SetWithProperType(iv, FieldByIndex(v, col.Index))
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
		// if f.Anonymous && f.Type.Kind() == reflect.Struct { //匿名struct , 也就是嵌套
		if f.Anonymous && (f.Type.Kind() == reflect.Struct || f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct) {
			ft := f.Type
			if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
				ft = f.Type.Elem()
			}
			// Recursively add nested fields in embedded structs.
			subcols := typeStructColumns(ft, underscore, tags...)
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
	//case reflect.Bool:
	//	return !v.Bool()
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

/* {{{ func IsZero(v reflect.Value) bool
 *
 */
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr:
		// fmt.Println("ptr")
		if v.IsNil() {
			return true
		}
		return IsZero(v.Elem())
	case reflect.Interface:
		// fmt.Printf("interface: %+v, %+v, %+v\n", v, v.IsNil(), v.Type())
		return v.IsNil()
	case reflect.Struct:
		// fmt.Println("struct")
		// return true if all fields are empty. else return false.
		return v.Interface() == reflect.Zero(v.Type()).Interface()
		// for i, n := 0, v.NumField(); i < n; i++ {
		// 	if !isEmptyValue(v.Field(i), deref) {
		// 		return false
		// 	}
		// }
		// return true
	default:
		return true
	}
}

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

// get fields slice
func Fields(i interface{}, opts ...interface{}) []string {
	tag := "json"
	must := ""
	skip := []interface{}{}
	skipTag := "sf" // 默认`sf`(=skip filed)就是忽略关键词
	params := NewParams(opts)
	if ot := params.StringByIndex(0); ot != "" {
		tag = ot
	}
	if om := params.StringByIndex(1); om != "" {
		must = om
	}
	if oskip := params.ArrayByIndex(2); len(oskip) > 0 {
		skip = oskip
	} else if ost := params.StringByIndex(2); ost != "" {
		skipTag = ost
	}
	fs := make([]string, 0)
	if fields := ReadStructFields(i, true, tag); fields != nil {
		for _, field := range fields {
			// if field.SubFields == nil && field.Tags[tag].Name != "" {
			if field.Tags[tag].Name != "" {
				options := field.Tags[tag].Options
				if (must == "" || options.Contains(must)) &&
					(len(skip) == 0 || !InSliceIface(field.Tags[tag].Name, skip)) &&
					(skipTag == "" || !options.Contains(skipTag)) {
					fs = append(fs, field.Tags[tag].Name)
				}
			}
		}
	}
	return fs
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
			// fmt.Printf("filed: %s, value: %+v", fn, fv)
			fts := fv.Type().String()
			// fmt.Printf("field name: %s, value: %+v, type: %s\n", fn, fv, fts)
			if (must == "" || options.Contains(must)) && (!options.Contains("omitempty") || !IsZero(fv)) {
				// 不为空 or 没有设置`omitempty`
				switch fts {
				case "string":
					sm[fn] = fv.String()
				case "*string":
					sm[fn] = fv.Elem().String()
				case "int", "int64":
					fvi := fv.Int()
					sm[fn] = strconv.FormatInt(fvi, 10)
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
	// fmt.Printf("converting: %s\n", string(b))
	// https://stackoverflow.com/questions/22343083/json-marshaling-with-long-numbers-in-golang-gives-floating-point-number
	// Unmarshal会存在数字问题
	// err = json.Unmarshal(b, o)
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	err = d.Decode(o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func Bind(ptr interface{}, data map[string]interface{}) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			// fmt.Printf("can not set: %s, %s\n", typeField.Name, structField.Kind().String())
			// not bind struct field, for now
			continue
		}
		if structField.Kind() == reflect.Struct {
			// not bind struct field, for now
			continue
		}
		//if structField.Kind() == reflect.Struct {
		//	err := Bind(structField.Addr().Interface(), data)
		//	if err != nil {
		//		return err
		//	}
		//	continue
		//}
		// first check field name
		inputValue, exists := data[typeField.Name]
		// second check field name(ignorecase)
		if !exists {
			inputValue, exists = data[strings.ToLower(typeField.Name)]
		}
		// third check field name(Underscore)
		if !exists {
			inputValue, exists = data[Underscore(typeField.Name)]
		}
		// forth check tag `json` name
		if !exists {
			inputFieldName := typeField.Tag.Get("json")
			inputValue, exists = data[inputFieldName]
		}
		if !exists {
			inputFieldName := typeField.Tag.Get("json")
			inputValue, exists = data[inputFieldName]
		}
		// not found field to set
		if !exists {
			continue
		}
		// fmt.Printf("will set: %s, %s, %s\n", typeField.Name, structField.Kind().String(), structField.Type().String())

		if err := SetWithProperType(inputValue, structField); err != nil {
			fmt.Printf("set failed: %s\n", err)
			return err
		}
	}
	return nil
}

func SetWithProperType(vi interface{}, structField reflect.Value) error {
	val := ""
	switch pv := vi.(type) {
	case string:
		val = pv
	case *string:
		val = *pv
	case int64:
		val = strconv.FormatInt(pv, 10)
	case int:
		val = strconv.FormatInt(int64(pv), 10)
	case *int64:
		val = strconv.FormatInt(*pv, 10)
	case *int:
		val = strconv.FormatInt(int64(*pv), 10)
	default:
		return errors.New("unknown support type")
	}
	switch structField.Type().String() {
	case "int":
		return setIntField(val, 0, structField)
	case "*int":
		return setIntPtrField(val, 0, structField)
	case "int8":
		return setIntField(val, 8, structField)
	case "*int8":
		return setIntPtrField(val, 8, structField)
	case "int16":
		return setIntField(val, 16, structField)
	case "*int16":
		return setIntPtrField(val, 16, structField)
	case "int32":
		return setIntField(val, 32, structField)
	case "*int32":
		return setIntPtrField(val, 32, structField)
	case "int64":
		return setIntField(val, 64, structField)
	case "*int64":
		return setIntPtrField(val, 64, structField)
	case "uint":
		return setUintField(val, 0, structField)
	case "*uint":
		return setUintPtrField(val, 0, structField)
	case "uint8":
		return setUintField(val, 8, structField)
	case "*uint8":
		return setUintPtrField(val, 8, structField)
	case "uint16":
		return setUintField(val, 16, structField)
	case "*uint16":
		return setUintPtrField(val, 16, structField)
	case "uint32":
		return setUintField(val, 32, structField)
	case "*uint32":
		return setUintPtrField(val, 32, structField)
	case "uint64":
		return setUintField(val, 64, structField)
	case "*uint64":
		return setUintPtrField(val, 64, structField)
	case "bool":
		return setBoolField(val, structField)
	case "*bool":
		return setBoolPtrField(val, structField)
	case "float32":
		return setFloatField(val, 32, structField)
	case "*float32":
		return setFloatPtrField(val, 32, structField)
	case "float64":
		return setFloatField(val, 64, structField)
	case "*float64":
		return setFloatPtrField(val, 64, structField)
	case "string":
		structField.SetString(val)
	case "*string":
		return setStringPtrField(val, structField)
	default:
		return fmt.Errorf("unknown type: %s", structField.Type().String())
	}
	return nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setIntPtrField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		switch bitSize {
		case 0: // *int
			val := new(int)
			*val = int(intVal)
			field.Set(reflect.ValueOf(val))
		case 8:
			val := new(int8)
			*val = int8(intVal)
			field.Set(reflect.ValueOf(val))
		case 16:
			val := new(int16)
			*val = int16(intVal)
			field.Set(reflect.ValueOf(val))
		case 32:
			val := new(int32)
			*val = int32(intVal)
			field.Set(reflect.ValueOf(val))
		default: // default 64
			field.Set(reflect.ValueOf(&intVal))
		}
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setUintPtrField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		switch bitSize {
		case 0: // *uint
			uval := new(uint)
			*uval = uint(uintVal)
			field.Set(reflect.ValueOf(uval))
		case 8:
			uval := new(uint8)
			*uval = uint8(uintVal)
			field.Set(reflect.ValueOf(uval))
		case 16:
			uval := new(uint16)
			*uval = uint16(uintVal)
			field.Set(reflect.ValueOf(uval))
		case 32:
			uval := new(uint32)
			*uval = uint32(uintVal)
			field.Set(reflect.ValueOf(uval))
		default: // default *uint64
			field.Set(reflect.ValueOf(&uintVal))
		}
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setBoolPtrField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.Set(reflect.ValueOf(&boolVal))
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setFloatPtrField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.Set(reflect.ValueOf(&floatVal))
	}
	return err
}

func setStringPtrField(value string, field reflect.Value) error {
	if value == "" {
		return nil
	}
	field.Set(reflect.ValueOf(&value))
	return nil
}
