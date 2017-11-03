// Package wrpc provides ...
package wrpc

import (
	"wgo/server"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Err(err error) error {
	if se, ok := err.(*server.ServerError); ok {
		return status.Error(codes.Code(se.Code), se.Message)
	}
	return status.Error(codes.Unknown, err.Error())
}

// Error returns an error representing c and msg.  If c is OK, returns nil.
func NewError(c codes.Code, msg string) error {
	return status.New(c, msg).Err()
}
