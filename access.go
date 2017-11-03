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
		Ts      string   `json:"ts"`   // timestamp
		Ver     string   `json:"ver"`  // server version
		Host    string   `json:"host"` // server host
		Message *Message `json:"message"`
	}

	Message struct {
		Service string      `json:"service"` // 服务ID(服务发现管理)
		SName   string      `json:"sname"`   // 服务名(服务发现管理)
		Env     string      `json:"env"`     // 服务环境(testing,production等)
		Proto   string      `json:"proto"`   // 协议 `[ "http", "rpc" ]`
		ReqID   string      `json:"reqid"`   // request-id, 首次访问由服务端生成, 各端传播
		Dura    float64     `json:"dura"`    // 持续时间, 单位毫秒
		Err     int         `json:"err"`     // 错误码(成功为0)
		Msg     string      `json:"msg"`     // 错误信息
		CIP     string      `json:"cip"`     // 客户端IP
		App     *App        `json:"app"`     // 应用程序信息
		User    *User       `json:"user"`    // 客户信息
		Call    *Call       `json:"call"`    // 调用信息
		Ext     interface{} `json:"ext"`     // 额外信息
	}

	App struct {
		Query    string `json:"query"`    // http querystring | grpc method
		Params   string `json:"params"`   // 参数信息
		Host     string `json:"host"`     // 域名
		Status   int    `json:"status"`   // 状态码
		ReqLen   int64  `json:"req_len"`  // 请求长度
		RespLen  int64  `json:"resp_len"` // 返回长度
		UA       string `json:"ua"`       // user-agent(最长256字节)
		Referer  string `json:"referer"`  // referer header(最长128字节)
		Ct       string `json:"ct"`       // content-type
		Encoding string `json:"enc"`      // 压缩编码
	}
	User struct {
		IP    string `json:"ip"`    // 用户IP
		Id    string `json:"id"`    // 用户ID
		ExtId string `json:"extid"` // 第三方id,openid等
		Sid   string `json:"sid"`   // session-id
	}
	Call struct {
		Depth uint64 `json:"depth"` // 调用深度, 收到请求后+1
		From  string `json:"from"`  // 调用端服务ID(服务发现管理), 可为空
		To    string `json:"to"`    // 向下调用服务ID, 如果有多个, 用逗号分隔
	}
)

func NewAccessLog() *AccessLog {
	ac := &AccessLog{
		Ver:  getVersion(),
		Host: Env().Hostname,
		Message: &Message{
			SName:   Env().ServiceName, // 服务名, 这个代码应该'自知'
			Service: Env().ServiceId,   // 服务id, 这个应该从配置中心拿到
			Env:     Env().ServiceEnv,  // 服务环境, 这个应该从配置中心拿到
			App:     &App{},
			User:    &User{},
			Call:    &Call{},
		},
	}
	return ac
}

// access reset
func (ac *AccessLog) Reset(t time.Time) {
	ac.Ts = t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	ac.Message.ReqID = ""
	ac.Message.Dura = 0
	ac.Message.Err = 0
	ac.Message.Msg = ""
	ac.Message.CIP = ""
	ac.Message.Proto = ""
	ac.Message.App.Query = ""
	ac.Message.App.Params = ""
	ac.Message.App.Host = ""
	ac.Message.App.Status = 0
	ac.Message.App.ReqLen = 0
	ac.Message.App.RespLen = 0
	ac.Message.App.UA = ""
	ac.Message.App.Referer = ""
	ac.Message.User.IP = ""
	ac.Message.User.Id = ""
	ac.Message.User.ExtId = ""
	ac.Message.User.Sid = ""
	ac.Message.Call.Depth = 0
	ac.Message.Call.From = ""
	ac.Message.Call.To = ""
}

// 获取版本号
func getVersion() (ver string) {
	if Env().ServiceVer != "" {
		return Env().ServiceVer
	}
	return "nover"
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
					ac.Message.Err = se.Status()
					ac.Message.Msg = se.Error()
				} else {
					ac.Message.Err = -1
					ac.Message.Msg = err.Error()
				}
			}
			// server mode
			ac.Message.Proto = c.ServerMode()
			// client ip
			ac.Message.CIP = c.ClientIP()
			// request id
			ac.Message.ReqID = c.RequestID()
			// dura
			ac.Message.Dura = utils.Round(c.Sub().Seconds()*1000, 3)
			// ext
			ac.Message.Ext = c.Ext()
			// app
			ac.Message.App.Query = c.Query()
			ac.Message.App.Params = c.Params()
			ac.Message.App.Host = c.Host()
			ac.Message.App.Status = c.Status()
			ac.Message.App.ReqLen = c.ReqLen()
			ac.Message.App.RespLen = c.RespLen()
			ac.Message.App.UA = c.UserAgent()
			ac.Message.App.Referer = c.Referer()
			ac.Message.App.Ct = c.ContentType()
			ac.Message.App.Encoding = c.ContentEncoding()
			// user
			ac.Message.User.IP = c.UserIP()
			ac.Message.User.Id = c.UserID()
			// call
			ac.Message.Call.Depth = c.Depth()
			ac.Message.Call.From = c.From()

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
