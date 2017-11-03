package wgo

import (
	"os"
	"sync"
	"time"

	s "wgo/server"
)

func (w *WGO) shutdown() {
	wg := new(sync.WaitGroup)
	for _, server := range w.servers {
		wg.Add(1)
		//w.Info("Closing Server(%s,%s)", server.Name(), server.Addr())
		go func(server *s.Server) {
			defer wg.Done()
			if err := server.Close(); err != nil {
				w.Error("Close server(%s) error: %s", server.Addr(), err)
			}
			//w.Info("Shutting down, waiting server(%s, %s) to idle...", server.Name(), server.Addr())
			if server.IsIdle() {
				w.Info("bye server(%s, %s)", server.Name(), server.Addr())
			}
		}(server)
	}
	wg.Wait()
	time.Sleep(20 * time.Millisecond)
	os.Exit(0)
}

/* {{{ func (w *WGO) daemonize()
 *
 */
func (w *WGO) daemonize() {
	if err := w.Daemon.MakeDaemon(); err != nil {
		Fatal("%s daemonize failed: %s", w.Daemon.ProcName, err)
	} else if w.Daemon.Daemonize {
		// deny console logging, and capture stdout
		w.Logger().DenyConsole()
	} else {
		w.Logger().AddConsole()
	}
}

/* }}} */
