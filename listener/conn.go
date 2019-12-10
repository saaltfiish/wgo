package listener

import (
	"errors"
	"net"
	"sync"
)

type Conn struct {
	net.Conn
	wg *sync.WaitGroup
}

func (c *Conn) Close() (err error) {
	defer func() {
		// log.Println("[Odin]conn close!!")
		c.wg.Done()
		if r := recover(); r != nil {
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()

	if err = c.Conn.Close(); err != nil {
		return
	}

	return
}
