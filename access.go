package wgo

import (
	"encoding/json"
	"time"

	"wgo/environ"
	"wgo/server"
	"wgo/utils"
	"wgo/wlog"
)

type (
	AccessConfig struct {
		Path    string `mapstructure:"path"`    // 日志文件, 对应wlog type="file"
		Addr    string `mapstructure:"addr"`    // 日志收集地址, 对应wlog type="syslog"
		Topic   string `mapstructure:"topic"`   // 由大数据分配的topic
		Thread  int    `mapstructure:"thread"`  // type="syslog"时，运行的goroutine数
		Network string `mapstructure:"network"` // type="syslog"时，network (tcp)
		Prefix  string `mapstructure:"prefix"`  // type="syslog"时，prefix ("")
	}
	// access log
	AccessLog struct {
		Ts      string  `json:"t,omitempty"`       // timestamp
		Ver     string  `json:"v,omitempty"`       // server version
		Host    string  `json:"h,omitempty"`       // server host
		SId     string  `json:"s,omitempty"`       // 服务ID(服务发现管理)
		SName   string  `json:"n,omitempty"`       // 服务名(服务发现管理)
		Dura    float64 `json:"d"`                 // 持续时间, 单位毫秒
		ReqID   string  `json:"rid,omitempty"`     // request-id, 首次访问由服务端生成, 各端传播
		Env     string  `json:"env,omitempty"`     // 服务环境(testing,production等)
		Err     int     `json:"err"`               // 错误码(成功为0)
		Msg     string  `json:"msg,omitempty"`     // 错误信息
		CIP     string  `json:"cip,omitempty"`     // 客户端IP
		Proto   string  `json:"proto,omitempty"`   // 协议 `[ "http", "rpc" ]`
		Call    Call    `json:"call,omitempty"`    // 调用信息
		App     App     `json:"app,omitempty"`     // 应用程序信息
		Service Service `json:"service,omitempty"` // 服务信息
	}

	App struct {
		Query    string      `json:"query,omitempty"`   // http querystring | grpc method
		Params   string      `json:"params,omitempty"`  // 参数信息
		Host     string      `json:"host,omitempty"`    // 域名
		Origin   string      `json:"origin,omitempty"`  // from where
		Status   int         `json:"status"`            // 状态码
		ReqLen   int64       `json:"req_len"`           // 请求长度
		RespLen  int64       `json:"resp_len"`          // 返回长度
		UA       string      `json:"ua,omitempty"`      // user-agent(最长256字节)
		Referer  string      `json:"referer,omitempty"` // referer header(最长128字节)
		Ct       string      `json:"ct,omitempty"`      // content-type
		Encoding string      `json:"enc,omitempty"`     // 压缩编码
		Ext      interface{} `json:"ext,omitempty"`     // 额外信息
	}
	User struct {
		IP    string `json:"ip,omitempty"`     // 用户IP
		Id    string `json:"id,omitempty"`     // 用户ID
		ExtId string `json:"ext_id,omitempty"` // 第三方id,openid等
		Sid   string `json:"sid,omitempty"`    // session-id
	}
	Call struct {
		Depth uint64 `json:"depth"`          // 调用深度, 收到请求后+1
		From  string `json:"from,omitempty"` // 调用端服务ID(服务发现管理), 可为空
		To    string `json:"to,omitempty"`   // 向下调用服务ID, 如果有多个, 用逗号分隔
	}

	// service
	Service struct {
		Endpoint string `json:"ep,omitempty"`
		Action   string `json:"act,omitempty"`
		Desc     string `json:"desc,omitempty"`
		RowKey   string `json:"rk,omitempty"`
		User     User   `json:"user,omitempty"` // 客户信息
		Old      string `json:"old,omitempty"`
		New      string `json:"new,omitempty"`
	}
)

func NewAccessLog() *AccessLog {
	ac := &AccessLog{
		Ver:   VERSION,
		Host:  Env().Hostname,
		SId:   Env().ServiceId,   // 服务id, 这个应该从配置中心拿到
		SName: Env().ServiceName, // 服务名, 这个代码应该'自知'
		Env:   Env().ServiceEnv,  // 服务环境, 这个应该从配置中心拿到
		Call:  Call{},
		App:   App{},
		Service: Service{
			User: User{},
		},
	}
	return ac
}

// access reset
func (ac *AccessLog) Reset(t time.Time) {
	ac.Ts = t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	ac.ReqID = ""
	ac.Dura = 0
	ac.Err = 0
	ac.Msg = ""
	ac.CIP = ""
	ac.Proto = ""
	ac.Call.Depth = 0
	ac.Call.From = ""
	ac.Call.To = ""
	ac.App.Query = ""
	ac.App.Params = ""
	ac.App.Host = ""
	ac.App.Origin = ""
	ac.App.Status = 0
	ac.App.ReqLen = 0
	ac.App.RespLen = 0
	ac.App.UA = ""
	ac.App.Referer = ""
	ac.Service.Endpoint = ""
	ac.Service.Action = ""
	ac.Service.Desc = ""
	ac.Service.RowKey = ""
	ac.Service.New = ""
	ac.Service.Old = ""
	ac.Service.User.IP = ""
	ac.Service.User.Id = ""
	ac.Service.User.ExtId = ""
	ac.Service.User.Sid = ""
}

// 记录access日志
// access应该是最外层的middleware
func Access() MiddlewareFunc {
	accessor := Accessor()
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			if accessor == nil { // 没有定义则跳过
				return next(c)
			}

			if err = next(c); err != nil {
				c.ERROR(err) // 这里必须处理error, 否则access抓不到
			}

			c.Flush() // 主要为了standard http

			ac := c.Access()

			// error
			if err != nil {
				if se, ok := err.(*server.ServerError); ok {
					ac.Err = se.Status()
					ac.Msg = se.Error()
				} else {
					ac.Err = -1
					ac.Msg = err.Error()
				}
			}
			// server mode
			ac.Proto = c.ServerMode()
			// client ip
			ac.CIP = c.ClientIP()
			// request id
			ac.ReqID = c.RequestID()
			// dura
			ac.Dura = utils.Round(c.Sub().Seconds()*1000, 3)
			// call
			ac.Call.Depth = c.Depth()
			ac.Call.From = c.From()
			// ext
			ac.App.Ext = c.Ext()
			// app
			ac.App.Query = c.Query()
			ac.App.Params = c.Params()
			ac.App.Host = c.Host()
			ac.App.Origin = c.Origin()
			ac.App.Status = c.Status()
			ac.App.ReqLen = c.ReqLen()
			ac.App.RespLen = c.RespLen()
			ac.App.UA = c.UserAgent()
			ac.App.Referer = c.Referer()
			ac.App.Ct = c.ContentType()
			ac.App.Encoding = c.ContentEncoding()
			// user
			ac.Service.User.IP = c.UserIP()
			if ac.Service.User.Id == "" {
				ac.Service.User.Id = c.UserID()
			}

			if sa, err := json.Marshal(ac); err != nil {
				c.Logger().Error("serialize access data failed: %s", err)
			} else {
				accessor.Access(string(sa))
			}

			return nil
		}
	}
}

// get access logger(accessor)
func Accessor() wlog.Logger {
	ac := &AccessConfig{}
	if err := wgo.Cfg().UnmarshalKey(environ.CFG_KEY_ACCESS, ac); err == nil { // 有配置
		accessor := make(wlog.Logger)
		if ac.Path != "" { // file
			accessor.Start(wlog.LogConfig{
				Type:    "file",
				Tag:     "_AC_",
				Format:  "%M",
				Level:   "ACCESS",
				Path:    ac.Path,
				Maxsize: 1 << 28,
				Daily:   true,
				MaxDays: 30,
				Mkdir:   true,
			})
			return accessor
		} else if ac.Addr != "" { // syslog
			if ac.Thread < 1 {
				ac.Thread = 10 //default
			}

			accessor.Start(wlog.LogConfig{
				Type:    "syslog",
				Tag:     "_AC_",
				Format:  "%M",
				Level:   "ACCESS",
				Network: ac.Network,
				Addr:    ac.Addr,
				Thread:  ac.Thread,
				Prefix:  ac.Prefix,
			})
			return accessor
		}
	}
	return nil
}
