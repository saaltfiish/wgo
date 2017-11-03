package controllers

import (
	"fmt"

	"wgo"
	"wgo_example/middlewares"
)

func init() {
	// defaults, all http server
	//wgo.NotFound(hello)
	//wgo.HTTPServers("wepiao").NotFound(hello)
	wgo.GET("/auth", auth).Use(middlewares.Auth())
	wgo.GET("/hello", hello)
	wgo.Group("/admin").GET("/login", login)

	// single server
	wgo.HTTPServers("wepiao").GET("/bye1", bye)
	wgo.HTTPServers("wepiao").GET("/bye", bye)

	// multiple servers&group
	ss := wgo.HTTPServers("odin", "wepiao")
	ssg := ss.Group("/test")
	ssg.GET("/test1", hello)
	//ssg.GET("/test1", hello).Abandon(middlewares.Access(wgo.Cfg()))
	ssg.GET("/test2", hello)
	ss.GET("/odin1", bye)

}

//func hello(c wgo.Context) error {
func hello(c *wgo.Context) error {
	if c1, err := c.Cookie("hello"); err == nil {
		fmt.Println("c1: ", c1)
	}

	c.SetExt(map[string]string{"hello": "world"})
	return c.String(wgo.StatusOK, c.QueryParam("name")+" hi!")
}

//func bye(c wgo.Context) error {
func bye(c *wgo.Context) error {
	var data = map[string]string{
		"1": "a",
		"2": "b",
		"3": "c",
	}
	return c.JSON(wgo.StatusOK, data)
}

//func login(c wgo.Context) error {
func login(c *wgo.Context) error {
	var data = map[string]string{
		"1": "a",
		"2": "b",
	}
	return c.JSON(wgo.StatusOK, data)
}

func auth(c *wgo.Context) error {
	c.Warn("success")
	return nil
}
