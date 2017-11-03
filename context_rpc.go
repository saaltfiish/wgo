package wgo

import (
	"time"

	"wgo/wrpc"
)

// reset rpc context
func (c *Context) RPCReset(req *wrpc.Request, res *wrpc.Response) {
	c.context = req.Context()
	c.request = req
	c.response = res
	c.mode = "rpc"
	c.start = time.Now()
	c.access.Reset(c.start)
	c.auth = false
	c.encoding = ""
	c.node = nil
	c.reqID = ""
	c.noCache = false
	c.ext = nil
}

// rpc response
func (c *Context) RPC(out interface{}) {
	c.Response().(*wrpc.Response).Body = out
}

// rpc requesrt message
func (c *Context) ReqMsg() interface{} {
	return c.Request().(*wrpc.Request).Body
}

// Decode
//func (c *Context) Decode(in interface{}) error {
//	return c.Request().(*wrpc.Request).Decode(in)
//}
