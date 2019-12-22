package listener

import (
	"net"
	"sync"
)

type Conn struct {
	net.Conn
	wg *sync.WaitGroup
}

func (c *Conn) Close() error {
	// defer func() {
	// 	if c.wg != nil {
	// 		c.wg.Done()
	// 	}
	// }()

	// if err := c.Conn.Close(); err != nil {
	// 	return err
	// }
	// return nil
	return c.Conn.Close()
}
