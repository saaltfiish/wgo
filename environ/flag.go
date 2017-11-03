package environ

import (
	goflags "flag"
	"fmt"
	"os"
)

type (
	flag struct {
		ns []string // flag names
		d  interface{}
		u  string
		v  interface{}
	}
	Flags map[string]*flag
)

const (
	FLAG_KEY_CONFIGFILE = "config_file"
	FLAG_KEY_CONFIGTYPE = "config_type"
	FLAG_KEY_CMDTAG     = "cmd_tag"

	FLAG_CMD_STATUS = "status"
	FLAG_CMD_STOP   = "stop"
	FLAG_CMD_RELOAD = "reload"
)

var (
	flagset *goflags.FlagSet
	flags   = Flags{ // 这里定义flag
		FLAG_KEY_CONFIGFILE: &flag{
			ns: []string{"config_file", "config", "c"},
			u:  "configuration file",
		},
		FLAG_KEY_CONFIGTYPE: &flag{
			ns: []string{"config_type", "ct"},
			d:  "json",
			u:  "configuration type",
		},
		FLAG_KEY_CMDTAG: &flag{
			u: "command tag",
		},
	}
)

func init() {
	// 解析参数
	parseFlags()
}

/* {{{ func parseFlags()
 *
 */
func parseFlags() {
	flagset = goflags.NewFlagSet("_ENV_", goflags.ContinueOnError)
	// command tag
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case FLAG_CMD_STATUS:
			flags[FLAG_KEY_CMDTAG].v = FLAG_CMD_STATUS
		case FLAG_CMD_RELOAD:
			flags[FLAG_KEY_CMDTAG].v = FLAG_CMD_RELOAD
		case FLAG_CMD_STOP:
			flags[FLAG_KEY_CMDTAG].v = FLAG_CMD_STOP
		}
	}
	if len(flags) > 0 {
		for _, flag := range flags {
			if len(flag.ns) > 0 {
				if flag.d == nil {
					flag.d = ""
				}
				for _, n := range flag.ns {
					switch flag.d.(type) { //根据默认值判断类型
					case int:
						flag.v = flagset.Int(n, flag.d.(int), flag.u)
					case bool:
						flag.v = flagset.Bool(n, flag.d.(bool), flag.u)
					default:
						flag.v = flagset.String(n, flag.d.(string), flag.u)
					}
				}
			}
		}
	}
	if err := flagset.Parse(os.Args[1:]); err != nil {
		fmt.Println("[Error] ", err)
	}
}

/* }}} */

/* {{{ func (fs Flags) get(name string) *flag
 *
 */
func (fs Flags) get(name string) *flag {
	if f, ok := fs[name]; ok {
		return f
	}

	return nil
}

/* }}} */

/* {{{ func StringFlag(name string) string
 *
 */
func StringFlag(name string) string { return flags.String(name) }
func (fs Flags) String(name string) string {
	if f := fs.get(name); f.v != nil {
		switch f.v.(type) {
		case string:
			return f.v.(string)
		case *string:
			return *f.v.(*string)
		}
	}
	return ""
}

/* }}} */

/* {{{ func CommandTag() string
 *
 */
func CommandTag() string { return flags.String(FLAG_KEY_CMDTAG) }

/* }}} */

/* {{{ func BoolFlag(name string) bool
 *
 */
func BoolFlag(name string) bool { return flags.Bool(name) }
func (fs Flags) Bool(name string) bool {
	if f := fs.get(name); f.v != nil {
		return f.v.(bool)
	} else {
		return false
	}
}

/* }}} */

/* {{{ func IntFlag(name string) int
 *
 */
func IntFlag(name string) int { return flags.Int(name) }
func (fs Flags) Int(name string) int {
	if f := fs.get(name); f.v != nil {
		return *f.v.(*int)
	} else {
		return -1
	}
}

/* }}} */
