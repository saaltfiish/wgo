package wgo

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"wgo/cluster"
	"wgo/daemon"
	"wgo/environ"
	"wgo/server"
	"wgo/storage"
	"wgo/whttp"
	"wgo/wrpc"
)

type (
	// WGO options
	WGO struct {
		lock    sync.Mutex
		cfg     *environ.Config  // 配置参数
		env     *environ.Environ // 环境参数
		logger  logger           // 日志
		storage *storage.Storage
		works   []*WorkerPool
		servers Servers

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
	// init env(include read configuration, init logger)
	env := environ.New(AppLevel).WithConfig()

	// build wgo with env, and register
	New(env).Register()

	// packages logging
	whttp.SetLogger(Logger())
	server.SetLogger(Logger())
	wrpc.SetLogger(Logger())
	storage.SetLogger(Logger())
	cluster.SetLogger(Logger())

	// 处理命令
	if tag := environ.CommandTag(); tag != "" {
		Info("iterrupt command tag: %s", tag)
		interceptCmd(tag)
	}

	// init storage
	var sn string
	var nodes []string
	if rns := os.Getenv("storage.redis.nodes"); rns != "" {
		sn = "redis"
		nodes = strings.Split(rns, ",")
	} else if scfg := SubConfig(environ.CFG_KEY_STORAGE); scfg != nil {
		// nodes可以通过env传递
		sn = scfg.String("name")
		nodes = scfg.StringSlice("nodes")
	}
	if sn != "" && len(nodes) > 0 {
		// 配置了storage再进行初始化
		css := make([]string, 0)
		for _, node := range nodes {
			Info("open %s storage, %s", sn, node)
			split := strings.Split(node, ":")
			if len(split) > 0 && split[0] != "" {
				host := split[0]
				port := "6379"
				db := "0"
				if len(split) >= 2 && split[1] != "" {
					port = split[1]
				}
				if len(split) >= 3 && split[2] != "" {
					db = split[2]
				}
				css = append(css, fmt.Sprintf("{\"conn\":\"%s:%s\",\"dbNum\":\"%s\"}", host, port, db))
			}
		}
		s, err := storage.New(sn, css...)
		if err != nil {
			Fatal("open storage failed: %s", err)
		}
		SetStorage(s)
	}

	// add servers
	if scs := environ.ServersConfig(Cfg()); len(scs) > 0 {
		for _, sc := range scs {
			// sc.Name = fmt.Sprintf("%s %s", Env().ProcName, Version())
			AddServer(sc)
		}
	}

	// default middleware
	Use(Access())
	Use(Recover())
	Use(Prepare())
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

/* {{{ Storage() logger
 * get logger
 */
func Storage() *storage.Storage { return wgo.Storage() }
func (w *WGO) Storage() *storage.Storage {
	return w.storage
}

/* }}} */

// set storage
func SetStorage(s *storage.Storage) {
	wgo.SetStorage(s)
}
func (w *WGO) SetStorage(s *storage.Storage) {
	if w != nil {
		w.storage = s
	}
}

/* {{{ Logger() logger
 * get logger
 */
func Logger() logger { return wgo.Logger() }
func (w *WGO) Logger() logger {
	if w.logger == nil { // 这里env就是logger, 只要实现了logger接口的都可set
		w.SetLogger(w.Env())
	}
	return w.logger
}

/* }}} */

/* {{{ func SetLogger(l logger)
 *
 */
func SetLogger(l logger) {
	wgo.SetLogger(l)
}
func (w *WGO) SetLogger(l logger) {
	if w != nil {
		w.logger = l
	}
}

/* }}} */

/* {{{ func Run()
 *
 */
func Run() { wgo.Run() }
func (w *WGO) Run() {
	defer func() {
		if err := recover(); err != nil {
			Error("WGO crashed with error: ", err)
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				Error(file, line)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// daemonize
	w.daemonize()

	// serve
	w.serve()

	// never end
	//Info("run to end")
	ch := make(chan os.Signal)
	<-ch
}

/* }}} */

// add work
func AddWork(label string, max int, jf JobHandler) *WorkerPool { return wgo.AddWork(label, max, jf) }
func (w *WGO) AddWork(label string, max int, jf JobHandler) *WorkerPool {
	if len(w.works) <= 0 {
		w.works = make([]*WorkerPool, 0)
	}

	wp := NewWorkerPool(label, max, jf)
	w.works = append(w.works, wp)
	return wp
}

/* {{{ func AddServer(sc environ.Server)
 *
 */
// factory
func Factory(s *server.Server) *server.Server {
	switch s.Mode() {
	case MODE_RPC, MODE_GRPC, MODE_WRPC:
		return wrpc.SetFactory(s, NewContext, mixWrpcMiddlewares).NewEngine()
	default: // 默认为http
		return whttp.SetFactory(s, NewContext, mixWhttpMiddlewares).NewEngine()
	}
}
func AddServer(sc server.Config) { wgo.AddServer(sc) }
func (w *WGO) AddServer(sc server.Config) {
	// 新建server
	s := server.NewServer(sc)
	// 装入
	w.push(Factory(s))
	Info("Added server: %s(%s<%s>), %d", s.Name(), s.Mode(), s.Engine().Name(), len(w.servers))
}

/* }}} */

/* {{{ func (w *WGO) serve()
 *
 */
func (w *WGO) serve() {
	// works
	if len(w.works) > 0 {
		for i, worker := range w.works {
			if i == 0 {
				// 注册第一个为默认
				Info("start work(default): %s", worker.Name())
				worker.Start().Register()
			} else {
				Info("start work(%d): %s", i, worker.Name())
				worker.Start()
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

			s.ListenAndServe(w.Daemon)
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
