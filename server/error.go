// Package server provides ...
package server

import (
	"fmt"
	"net/http"
)

type (
	ServerError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

// new error
func NewError(code int, msg ...interface{}) *ServerError {
	e := &ServerError{
		Code: code,
	}
	if len(msg) > 0 {
		switch m := msg[0].(type) {
		case string:
			e.Message = m
		case error:
			e.Message = m.Error()
		default:
			e.Message = fmt.Sprint(m)
		}
	} else if e.HTTPStatusCode() > 0 {
		e.Message = http.StatusText(e.HTTPStatusCode())
	}
	return e
}

// new format error
func NewErrorf(code int, format string, a ...interface{}) *ServerError {
	return &ServerError{
		Code:    code,
		Message: fmt.Sprintf(format, a...),
	}
}

// wrap
func WrapError(err error) *ServerError {
	if se, ok := err.(*ServerError); ok {
		return se
	}
	return NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
}

// imp error
func (e *ServerError) Error() string {
	// return fmt.Sprintf("error: code = %d message = %s", e.Code, e.Message)
	return fmt.Sprintf("%s(%d)", e.Message, e.Code)
}

func (e *ServerError) Status() int {
	return e.Code
}

// 利用error生成http的status code
func (e *ServerError) HTTPStatusCode() int {
	sc := e.Code / 1000
	if msg := http.StatusText(sc); msg != "" {
		return sc
	}
	if msg := http.StatusText(e.Code); msg != "" {
		return e.Code
	}
	return 0
}
