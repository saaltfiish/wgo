package environ

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"wgo/daemon"
	"wgo/environ/uranus"
)

const (
	// level
	LVL_DEV        = "dev"
	LVL_TESTING    = "testing"
	LVL_PRODUCTION = "production"

	defaultLoc         = "Asia/Shanghai"
	defaultAppLevel    = LVL_DEV
	defaultDaemonize   = false
	defaultEnableCache = true
	defaultDockerize   = false
	defaultDebugMode   = false
)

var (
	environ *Environ

	hostname    string
	level       string = defaultAppLevel
	executePath string
	executeDir  string
	executeName string

	defaultWorkDir string
	defaultPidFile string

	defaultLocation *time.Location
)

type (
	Environ struct {
		// service discovery
		ServiceId   string // service id
		ServiceName string // service name
		ServiceVer  string // service version
		ServiceEnv  string // service env

		// app info
		Hostname    string // hostname
		ExecPath    string // execute path
		ProcName    string // proc name
		Daemonize   bool   // daemonize or not
		Dockerize   bool   // dockerize or not
		EnableCache bool   // enable cache or not
		DebugMode   bool   // debug mode
		WorkDir     string // working dir(abs)
		AppDir      string // application dir
		PidFile     string // pidfile abs path

		// options
		Location *time.Location // location

		// origin config
		cfg *Config

		// logger
		logger *logger
	}
)

func init() {
	hostname, _ = os.Hostname()
	if ep, err := daemon.GetExecutablePath(); err != nil {
		panic("[PANIC] can't determine full path to executable")
	} else {
		executePath = ep
	}
	executeName = filepath.Base(executePath)
	executeDir, _ = filepath.Abs(filepath.Dir(executePath))

	defaultWorkDir = executeDir
	defaultPidFile = filepath.Join(defaultWorkDir, "run", executeName+".pid")

	// location
	var err error
	defaultLocation, err = time.LoadLocation(defaultLoc)
	if err != nil {
		panic(fmt.Sprintf("[PANIC] load location failed: %s", err))
	}
}

/* {{{ func New(opts ...interface{}) *Environ
 *
 */
func New(opts ...interface{}) *Environ {
	if len(opts) > 0 {
		if lvl, ok := opts[0].(string); ok && lvl != "" {
			level = lvl
		}
	}
	w := new(Environ).WithDefaults()
	return w
}

/* }}} */

/* {{{ func (env *Environ) WithDefaults() *Environ
 * wgo default env
 */
func (env *Environ) WithDefaults() *Environ {
	env.ExecPath = executePath
	env.Hostname = hostname
	env.ProcName = executeName
	env.Daemonize = defaultDaemonize
	env.Dockerize = defaultDockerize
	env.EnableCache = defaultEnableCache
	env.DebugMode = defaultDebugMode
	if level == LVL_DEV {
		// 生产环境默认不debug
		env.DebugMode = true
	}
	env.WorkDir = defaultWorkDir
	env.AppDir = executeDir
	env.PidFile = defaultPidFile
	env.Location = defaultLocation

	return env
}

/* }}} */

/* {{{ func (env *Environ) Register() *Environ
 *
 */
func (env *Environ) Register() *Environ {
	environ = env
	return env
}

/* }}} */

/* {{{ func (env *Environ) WithConfig() *Environ
 * configured env
 */
func WithConfig() *Environ { return environ.WithConfig() }
func (env *Environ) WithConfig() *Environ {
	cfg := env.Cfg()

	if pn := cfg.String(CFG_KEY_PROCNAME); pn != "" {
		env.ProcName = pn
	}
	if d := cfg.Bool(CFG_KEY_DAEMONIZE); d == true {
		env.Daemonize = d
	}
	if d := cfg.Bool(CFG_KEY_DOCKERIZE); d == true {
		env.Dockerize = d
	}
	if ec := cfg.Bool(CFG_KEY_ENABLECACHE); ec == true {
		env.EnableCache = ec
	}
	if dbg := cfg.Bool(CFG_KEY_DEBUG); dbg == true {
		env.DebugMode = dbg
	}
	if ad := cfg.String(CFG_KEY_APPDIR); ad != "" {
		env.AppDir = cfg.String(CFG_KEY_APPDIR)
	}
	if wd := cfg.String(CFG_KEY_WORKDIR); wd != "" {
		env.WorkDir = cfg.String(CFG_KEY_WORKDIR)
	}
	if pf := cfg.String(CFG_KEY_PIDFILE); pf != "" {
		env.PidFile = cfg.String(CFG_KEY_PIDFILE)
	} else {
		env.PidFile = filepath.Join(env.WorkDir, "run", env.ProcName+".pid")
	}

	if etz := os.Getenv(CFG_KEY_TIMEZONE); etz != "" {
		if loc, err := time.LoadLocation(etz); err == nil {
			env.Location = loc
		} else {
			fmt.Printf("load zone(%s) from env error: %s\n", etz, err)
		}
	} else if tz := cfg.String(CFG_KEY_TIMEZONE); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			env.Location = loc
		} else {
			fmt.Printf("load zone(%s) from cfg error: %s\n", tz, err)
		}
	}
	if env.Location == nil {
		env.Location, _ = time.LoadLocation(defaultLoc)
	}
	// init logger
	env.Logger().Init(cfg)

	env.cfg = cfg

	return env
}

/* }}} */

/* {{{ func (env *Environ) Cfg() *Config
 *
 */
func Cfg() *Config { return environ.Cfg() }
func (env *Environ) Cfg() *Config {

	// default config
	if env.cfg == nil { //如果为nil, 则采用默认的方式读取配置
		env.cfg = env.CustomConfig()
	}
	return env.cfg
}

/* }}} */

/* {{{ func CustomConfig() *Config
 *
 */
func (env *Environ) CustomConfig() *Config {
	cfg := NewConfig()

	// uranus
	uaddr := os.Getenv("__uranus")
	sn := os.Getenv("__sname")
	if sn == "" {
		sn = executeName // 如果没有传入服务名, 用程序名尝试
	}
	env.ServiceName = sn
	ver := os.Getenv("__ver")
	envtag := os.Getenv("__env")
	if uaddr != "" && sn != "" && ver != "" && envtag != "" { // 相关信息都齐了
		if sid, cc, err := uranus.GetConfigContent(uaddr, sn, ver, envtag); err == nil {
			// 写入service信息
			env.ServiceId = sid
			env.ServiceEnv = envtag
			env.ServiceVer = ver
			return cfg.ReadConfig(cc, "json")
		} else {
			// TODO, 读取之前的配置缓存
			panic(fmt.Sprintf("[PANIC] read from uranus failed: %s", err))
		}
	}

	return DefaultConfig()
}

/* }}} */
