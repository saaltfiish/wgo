//
// example.go
// Copyright (C) 2021 Odin <odinmanlee@gmail.com>
//
// Distributed under terms of the MIT license.
//

package api

import (
	"fmt"
	"wgo"
)

func init() {
	wgo.GET("/examples", hello)
	wgo.GET("/example1", hello1)
}

func hello(c *wgo.Context) error {
	if c1, err := c.Cookie("hello"); err == nil {
		fmt.Println("c1: ", c1)
	}

	c.SetExt(map[string]string{"hello": "world"})
	return c.String(wgo.StatusOK, "hello world! "+c.QueryParam("name")+" hi!")
}
func hello1(c *wgo.Context) error {
	if c1, err := c.Cookie("hello"); err == nil {
		fmt.Println("c1: ", c1)
	}

	c.SetExt(map[string]string{"hello": "world"})
	return c.String(wgo.StatusOK, "hello world 1! "+c.QueryParam("name")+" hi!")
}
