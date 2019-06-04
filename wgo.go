package wgo

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"wgo/cron"
	"wgo/daemon"
	"wgo/environ"
	"wgo/server"
	"wgo/storage"
	"wgo/whttp"
	"wgo/wlog"
	"wgo/wrpc"
)

type (
	// WGO options
	WGO struct {
		lock     sync.Mutex
		cfg      *environ.Config  // 配置参数
		env      *environ.Environ // 环境参数
		logger   logger           // 日志
		accessor wlog.Logger
		storage  *storage.Storage
		cron     *cron.Cron
		works    []*WorkerPool
		servers  Servers

		Daemon *daemon.Daemon // 守护进程
	}
)

var (
	debug bool
	wgo   *WGO
	wp    *WorkerPool
)

func init() {
	Init()
}

/* {{{ func New(cfg *Config) *WGO
 * return a new wgo
 */
func New(env *environ.Environ) *WGO {

	w := new(WGO)

	w.env = env

	// daemon
	w.Daemon = (&daemon.Daemon{
		Daemonize: w.env.Daemonize, // 是否守护进程
		Dockerize: w.env.Dockerize, // 是否dockerize
		WorkDir:   w.env.WorkDir,   // 工作目录
		ExecPath:  w.env.ExecPath,  // 执行文件
		ProcName:  w.env.ProcName,  // 进程表中的名称
		PidFile:   w.env.PidFile,   // pidfile, reload时候需要unlock
	}).Register(w) // 注册到daemon包, 效果是调用daemnon.xxx 等效于 w.Daemon.xxx
	daemon.RegisterShutdown(w.shutdown)

	// debug mode
	debug = w.env.DebugMode

	return w
}

/* }}} */

/* {{{ func Register() *WGO
 *
 */
func (w *WGO) Register() *WGO {
	wgo = w
	return wgo
}

/* }}} */

/* {{{ func Init()
 * wgo initialization
 */
func Init() {
	defer func() {
		if err := recover(); err != nil {
			Error("[WGO.Init]crashed with error: %s", err)
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				Error("[WGO.Init]%s:%d", file, line)
			}
			time.Sleep(10 * time.Millisecond)
			panic(err)
		}
	}()
	// init env(include read configuration, init logger)
	env := environ.New(AppLevel).WithConfig()

	// build wgo with env, and register
	New(env).Register()

	// packages logging
	whttp.SetLogger(Logger())
	server.SetLogger(Logger())
	wrpc.SetLogger(Logger())
	storage.SetLogger(Logger())
	cron.SetLogger(Logger())

	// 处理命令
	if tag := environ.CommandTag(); tag != "" {
		Info("iterrupt command tag: %s", tag)
		interceptCmd(tag)
	}

	// init storage
	initStorage()

	// init cron
	initCron()

	// add servers
	if scs := environ.ServersConfig(Cfg()); len(scs) > 0 {
		for _, sc := range scs {
			// sc.Name = fmt.Sprintf("%s %s", Env().ProcName, Version())
			AddServer(sc)
		}
	}

	// default middleware
	Use(Recover())
	Use(Prepare())
	Use(Access())
	if env.EnableCache {
		Use(Cache())
	}
}

/* }}} */

/* {{{ func Self() *WGO
 *
 */
func Self() *WGO {
	return wgo
}

/* }}} */

/* {{{ Cfg() *environ.Config
 * get config info
 */
func Cfg() *environ.Config { return wgo.Cfg() }
func (w *WGO) Cfg() *environ.Config {
	return w.Env().Cfg()
}

/* }}} */

/* {{{ Env() *environ.Environ
 * get env info
 */
func Env() *environ.Environ { return wgo.Env() }
func (w *WGO) Env() *environ.Environ {
	if w.env == nil { // init env
		panic("[PANIC] not found env")
	}
	return w.env
}

/* }}} */

/* {{{ func Run(ces ...server.Engine)
 * 可传入自定义的engine `ces = custom engines`
 */
func Run(ces ...server.Engine) { wgo.Run(ces...) }
func (w *WGO) Run(ces ...server.Engine) {
	defer func() {
		if err := recover(); err != nil {
			Error("[WGO.Run]crashed with error: %s", err)
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				Error("[WGO.Run]%s:%d", file, line)
			}
			time.Sleep(10 * time.Millisecond)
			panic(err)
		}
	}()

	// daemonize
	w.daemonize()

	// serve
	w.serve(ces...)

	// never end
	//Info("run to end")
	ch := make(chan os.Signal)
	<-ch
}

/* }}} */

// add work
// func AddWork(label string, max int, jf HandlerFunc) *WorkerPool {
func AddWork(label string, opts ...interface{}) *WorkerPool {
	// max workers, default 10
	max := 10
	if len(opts) > 0 {
		if im, ok := opts[0].(int); ok {
			max = im
		}
	}
	// default job handler
	dh := func(c *Context) error {
		return fmt.Errorf("Unknown method!")
	}
	if len(opts) > 1 {
		if idh, ok := opts[1].(func(c *Context) error); ok {
			dh = idh
		}
	}
	return wgo.AddWork(label, max, dh)
}
func (w *WGO) AddWork(label string, max int, jf HandlerFunc) *WorkerPool {
	if len(w.works) <= 0 {
		w.works = make([]*WorkerPool, 0)
	}

	wp := NewWorkerPool(label, max, jf)
	w.works = append(w.works, wp)
	return wp
}

// factory
func Factory(s *server.Server) *server.Server {
	switch s.Mode() {
	case server.MODE_RPC, server.MODE_GRPC, server.MODE_WRPC: // all is grpc
		return wrpc.Factory(s, NewContext, mixWrpcMiddlewares).BuildEngine()
	case server.MODE_HTTP, server.MODE_HTTPS: // http/https
		return whttp.Factory(s, NewContext, mixWhttpMiddlewares).BuildEngine()
	default: // 直接return s, 将使用自定义的server.Engine(在Run的时候使用...)
		Debug("[wgo.Factory]custom mode: %s", s.Mode())
		return s
	}
}

/* {{{ func AddServer(sc environ.Server)
 *
 */
func AddServer(sc server.Config) { wgo.AddServer(sc) }
func (w *WGO) AddServer(sc server.Config) {
	// 新建server
	s := server.NewServer(sc)
	// 装入
	// Debug("[AddServer]mode: %s, engine: %s", s.Mode(), s.EngineName())
	w.push(Factory(s))
	Debug("Added server: %s(%s<%s>)", s.Name(), s.Mode(), s.EngineName())
}

/* }}} */

/* {{{ func (w *WGO) serve(ces ...server.Engine)
 *
 */
func (w *WGO) serve(ces ...server.Engine) {
	// works
	if len(w.works) > 0 {
		for i, worker := range w.works {
			if i == 0 {
				// 注册第一个为默认
				Debug("start work(default): %s", worker.Name())
				worker.Start().Register()
			} else {
				Debug("start work(%d): %s", i, worker.Name())
				worker.Start()
			}
		}
	}

	// checking custom engines
	for _, s := range w.servers {
		if s.Engine() == nil {
			for _, ce := range ces {
				if s.Mode() == ce.Name() { // engine名称与mode需要匹配
					Debug("[wgo.Run]found custom server engine: %s", ce.Name())
					s.SetEngine(ce)
				}
			}
		}
	}

	// start servers
	wg := new(sync.WaitGroup)
	for _, s := range w.servers {
		wg.Add(1)
		go func(s *server.Server) {
			defer wg.Done()

			// prepare, build routes, etc...
			s.Prepare()

			if err := s.ListenAndServe(w.Daemon); err != nil {
				panic(err)
			}
		}(s)
	}

	wg.Wait()
}

/* }}} */

// 获取配置
func AppConfig(rawVal interface{}, opts ...interface{}) error {
	return Cfg().AppConfig(rawVal, opts...)
}
func SubConfig(opts ...interface{}) *environ.Config {
	if len(opts) > 0 {
		if k, ok := opts[0].(string); ok {
			return Cfg().Sub(k)
		}
	}
	return nil
}
