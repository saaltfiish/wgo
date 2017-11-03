package main

import (
	//"runtime/debug"
	"time"
	_ "wgo_example/controllers"
	pb "wgo_example/pb"
	// vendor package
	"wgo"
)

type (
	Config struct {
		Config1 string `json:"config1"`
		Config2 string `json:"config2"`
	}
	Config1 struct {
		Url string `json:"url"`
	}
)

func init() {
	wgo.AddWorker("test", Test)
	// register rpc service
	wgo.RegisterRPCService(pb.RegisterGreeterServer1)
}

func main() {
	//debug.SetTraceback("all")
	wgo.Run()
}

func Test(_ *wgo.WGO) {
	wgo.Info("hi")
	wgo.Warn("warning")
	wgo.Error("error")
	cfg := new(Config)
	if err := wgo.AppConfig(cfg); err == nil {
		wgo.Info("find config: %v", cfg)
	}
	cfg1 := new(Config1)
	if err := wgo.AppConfig(cfg1, "app1"); err == nil {
		wgo.Info("find config1: %v", cfg1)
	}
	time.Sleep(20 * time.Second)
}
