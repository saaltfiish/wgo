// Package middlewares provides ...
package middlewares

import (
	//"errors"

	"wgo"
)

func Auth() wgo.MiddlewareFunc {
	return func(next wgo.HandlerFunc) wgo.HandlerFunc {
		return func(c *wgo.Context) (err error) {
			if auth := APIAuthorization(c); !auth {
				c.Error("auth failed!")
				//return errors.New("auth failed!")
				return c.NewError(401, "auth failed!")
			}
			return next(c)
		}
	}
}
func APIAuthorization(c *wgo.Context) bool {
	return false
}
