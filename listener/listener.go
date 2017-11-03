package listener

import (
	"net"
	"os"
	"sync"
	"time"
)

type Listener struct {
	net.Listener
	wg *sync.WaitGroup
}

func WrapListener(l net.Listener) (el *Listener) {
	return &Listener{
		Listener: l,
		wg:       &sync.WaitGroup{},
	}
}

func New(addr string) (el *Listener) {
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		panic("listen failed")
	}

	el = &Listener{
		Listener: ln,
		wg:       &sync.WaitGroup{},
	}

	return

}

// Accept 接受连接
func (l *Listener) Accept() (c net.Conn, err error) {
	tc, err := l.Listener.(*net.TCPListener).AcceptTCP()
	if err != nil {
		return nil, err
	}

	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlivePeriod(30 * time.Second)
	if err != nil {
		return nil, err
	}

	// wait group
	l.wg.Add(1)

	return &Conn{Conn: tc, wg: l.wg}, nil
}

func (l *Listener) Close() error {
	return l.Listener.Close()
}

func (l *Listener) File() *os.File {
	tl := l.Listener.(*net.TCPListener)
	fl, _ := tl.File()
	return fl
}

func (l *Listener) Wait() {
	l.wg.Wait()
}
