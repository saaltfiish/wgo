//
// client.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package rest

import (
	"encoding/json"
	"fmt"
	"strings"

	"wgo"
	"wgo/resty"
	"wgo/server"
	"wgo/utils"
	"wgo/whttp"
)

// for inner request
type Client struct {
	host string
	path string
	app  string

	req *resty.Request
}

type Response struct {
	code    int
	message string
	parsed  bool
	data    *utils.Json

	*resty.Response
}

// new client, can pass appid
func NewClient(service string, opts ...interface{}) *Client {
	host := ""
	if len(service) > 5 && strings.ToLower(service[0:4]) == "http" {
		// 兼容旧版本, 旧版直接传入服务地址
		host = service
	} else {
		host = GetService(service)
	}
	client := &Client{
		host: host,
		req:  resty.New().R(),
	}
	if len(opts) > 0 {
		if app, ok := opts[0].(string); ok {
			client.app = app
		}
		// todo, get app info & sign by appid
	}
	return client
}

// new inner client
func NewInnerClient(service string) (*Client, error) {
	// todo: not hardcore inner appid
	if host := GetService(service); host != "" {
		return NewClient(host, "gxfstpp"), nil
	}
	return nil, fmt.Errorf("Unknown service: %s", service)
}

func (client *Client) SetJson(data interface{}) *Client {
	jb, _ := json.Marshal(data)
	client.req.SetBody(jb)
	return client
}

func (client *Client) SetForm(fd map[string]string) *Client {
	client.req.SetFormData(fd)
	return client
}

func (client *Client) SetParams(fd map[string]string) *Client {
	client.req.SetQueryParams(fd)
	return client
}

// sendAndRecv
func (client *Client) sendAndRecv(cat string, opts ...interface{}) (resp *Response, err error) {
	var restyResp *resty.Response
	if client.app != "" {
		client.req.SetHeader(whttp.HeaderXAppId, client.app)
	}
	if len(opts) > 0 {
		if path, ok := opts[0].(string); ok {
			client.path = path
		}
	}
	url := client.host + client.path
	switch strings.ToLower(cat) {
	case "get":
		restyResp, err = client.req.Get(url)
	case "patch":
		restyResp, err = client.req.Patch(url)
	case "head":
		restyResp, err = client.req.Head(url)
	case "put":
		restyResp, err = client.req.Put(url)
	case "options":
		restyResp, err = client.req.Options(url)
	default:
		restyResp, err = client.req.Post(url)
	}
	if err != nil {
		return nil, err
	}
	resp = &Response{
		Response: restyResp,
	}
	resp.Parse()
	// wgo.Info("[sendAndRecv]code: %d, message: %s", resp.Code(), resp.Message())

	// renew req
	client.req = resty.New().R()

	return resp, nil
}

// response methods
func (resp *Response) Parse() (err error) {
	if resp.parsed {
		return
	}
	rc := resp.StatusCode()
	if rc >= 400 {
		resp.message = resp.Status()
	}
	if len(resp.Body()) > 0 {
		resp.data, err = utils.NewJson(resp.Body())
		if err != nil {
			wgo.Warn("[Parse]unmarshal response error: %s", err)
			if rc >= 400 {
				// 如果解释为json失败并且httpcode大于400, 直接把返回当做出错信息
				resp.code = rc
				resp.message = string(resp.Body())
			}
		} else if code := resp.data.Get("code").MustInt(); code > 0 {
			resp.code = code
			resp.message = resp.data.Get("message").MustString()
		}
	}
	resp.parsed = true
	return
}
func (resp *Response) Code() int {
	if !resp.parsed {
		resp.Parse()
	}
	return resp.code
}
func (resp *Response) Message() string {
	if !resp.parsed {
		resp.Parse()
	}
	return resp.message
}
func (resp *Response) Data() *utils.Json {
	return resp.data
}

// base methods
func (client *Client) Post(path string) (*Response, error) {
	return client.sendAndRecv("post", path)
}
func (client *Client) Get(path string) (*Response, error) {
	return client.sendAndRecv("get", path)
}
func (client *Client) Patch(path string) (*Response, error) {
	return client.sendAndRecv("patch", path)
}

func (client *Client) Query(service, path, method string, obj interface{}) (*utils.Json, error) {
	request, err := NewInnerClient(service)
	if err != nil {
		return nil, err
	}
	var resp *Response
	var re error
	switch strings.ToLower(method) {
	case "get":
		resp, re = request.Get(path)
	case "patch":
		resp, re = request.Patch(path)
	default:
		resp, re = request.Post(path)
	}
	if re != nil {
		return nil, re
	}
	if resp.Code() != 0 {
		return nil, server.NewError(resp.Code(), resp.Message())
	}
	return resp.Data(), nil
}
