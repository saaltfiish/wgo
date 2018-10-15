package listener

import (
	"net"
	"os"
	"sync"
	"time"
)

type Listener struct {
	// net.Listener
	*net.TCPListener
	wg *sync.WaitGroup
}

func WrapListener(l net.Listener) (el *Listener) {
	return &Listener{
		TCPListener: l.(*net.TCPListener),
		wg:          &sync.WaitGroup{},
	}
}

func New(addr string) (el *Listener) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic("listen failed")
	}

	el = &Listener{
		TCPListener: ln.(*net.TCPListener),
		wg:          &sync.WaitGroup{},
	}

	return

}

// Accept 接受连接
func (l *Listener) Accept() (c net.Conn, err error) {
	tc, err := l.AcceptTCP()
	// tc, err := l.Listener.Accept()
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
	return l.TCPListener.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.TCPListener.Addr()
}

func (l *Listener) File() *os.File {
	return l.File()
}

func (l *Listener) Wait() {
	l.wg.Wait()
}
