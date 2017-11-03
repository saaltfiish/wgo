package controllers

import (
	"wgo"

	pb "wgo_example/pb"
)

func init() {
	wgo.RPCServers()
	wgo.GET("/rest", rest)
	// wrpc, method name + wgo.HandlerFunc, 注册rpc路由
	//wgo.AddRPC("SayHello", rpc)
	wgo.GET("/both/@test", both)
	wgo.GET("/both/:id", both).Cache(10, []string{"name"}, []string{"X-WG-UserId"})
	//wgo.GET("/both", both)
	wgo.AddRPC("SayHello", both)
}

func rpc(c *wgo.Context) error {
	wgo.Info("hi wrpc, reqid: %s", c.RequestID())
	//in := new(pb.HelloRequest)
	//if err := c.Decode(in); err != nil {
	//	wgo.Info("decode failed")
	//} else {
	//	wgo.Info("in: %v", in)
	//	out := new(pb.HelloReply)
	//	out.Message = "odintest1"
	//	c.RPC(out)
	//}
	wgo.Info("in: %v", c.Request())
	out := new(pb.HelloReply)
	out.Message = "odintest1"
	c.RPC(out)
	return nil
}

func rest(c *wgo.Context) error {
	wgo.Info("hi rest, reqid: %s", c.RequestID())
	out := new(pb.HelloReply)
	out.Message = "odintest1"
	//c.ERROR(fmt.Errorf("this error!"))
	//return fmt.Errorf("this error!")
	return c.NewError(400001, "just for test")
}

func both(c *wgo.Context) error {
	//c.Info("id: %s, mode: %s", c.Param("id"), c.ServerMode())
	//wgo.Warn("id: %s, mode: %s", c.Param("id"), c.ServerMode())
	switch c.ServerMode() {
	case "http":
		out := new(pb.HelloReply)
		out.Message = "odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1odintest1"
		c.JSON(200, out)
	case "rpc":
		//in := new(pb.HelloRequest)
		//if err := c.Decode(in); err != nil {
		//	wgo.Info("decode failed")
		//} else {
		//	wgo.Info("in: %v", in)
		//	out := new(pb.HelloReply)
		//	out.Message = "odintest1"
		//	c.RPC(out)
		//}
		c.Info("name: %s", c.ReqMsg().(*pb.HelloRequest).Name)
		out := new(pb.HelloReply)
		out.Message = "odintest1"
		c.RPC(out)
	}
	return nil
}
