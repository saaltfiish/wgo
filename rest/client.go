//
// client.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package rest

import (
	"encoding/json"
	"strings"
	"wgo"
	"wgo/resty"
	"wgo/utils"
	"wgo/whttp"
)

// for inner request
type Client struct {
	req *resty.Request
	url string
	app string
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    *utils.Json `json:"data"`
}

func NewClient(url string, opts ...interface{}) *Client {
	client := &Client{
		req: resty.NewRequest(),
		url: url,
	}
	if len(opts) > 0 {
		if app, ok := opts[0].(string); ok {
			client.app = app
		}
	}
	return client
}

func (client *Client) SetJson(data map[string]interface{}) *Client {
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

func (client *Client) sendAndRecv(cat string, opts ...interface{}) (resp *Response, err error) {
	var restyResp *resty.Response
	if client.app != "" {
		client.req.SetHeader(whttp.HeaderXAppId, client.app)
	}
	switch strings.ToLower(cat) {
	case "get":
		restyResp, err = client.req.Get(client.url)
	case "patch":
		restyResp, err = client.req.Patch(client.url)
	case "head":
		restyResp, err = client.req.Head(client.url)
	case "put":
		restyResp, err = client.req.Put(client.url)
	case "options":
		restyResp, err = client.req.Options(client.url)
	default:
		restyResp, err = client.req.Post(client.url)
	}
	if err != nil {
		return nil, err
	}
	resp = &Response{
		Code:    0, // 默认为0
		Message: restyResp.Status(),
	}
	respCode := restyResp.StatusCode()
	var respData *utils.Json
	if len(restyResp.Body()) > 0 {
		respData, err = utils.NewJson(restyResp.Body())
		if err != nil {
			wgo.Error("[sendAndRecv]unmarshal response error: %s", err)
		}
	}
	if respCode >= 200 && respCode < 400 {
		// success
		resp.Data = respData
	} else if respCode >= 400 {
		resp.Code = respCode
		if respData != nil && respData.Get("code").MustInt() > 0 {
			resp.Code = respData.Get("code").MustInt()
			resp.Message = respData.Get("message").MustString()
		}
	}
	wgo.Info("[sendAndRecv]code: %d, message: %s", resp.Code, resp.Message)
	if len(restyResp.Body()) > 0 {
		resp.Data, err = utils.NewJson(restyResp.Body())
		if err != nil {
			wgo.Error("[sendAndRecv]unmarshal response error: %s", err)
		}
	}

	// renew req
	client.req = resty.NewRequest()

	return resp, nil
}

// base methods
func (client *Client) Post() (*Response, error) {
	return client.sendAndRecv("post")
}
func (client *Client) Get() (*Response, error) {
	return client.sendAndRecv("get")
}
