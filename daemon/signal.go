// Package daemon runs a program as a Unix daemon.
package daemon

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	PreSignal = iota
	PostSignal

	DefaultHammerTime = 10
)

func (d *Daemon) handleSignals() {
	d.sigChan = make(chan os.Signal)
	var sig os.Signal

	signal.Notify(
		d.sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	pid := syscall.Getpid()
	for {
		sig = <-d.sigChan
		d.signalHooks(PreSignal, sig)
		switch sig {
		case syscall.SIGHUP:
			d.Log(pid, "Received SIGHUP. reloading.")
			if err := d.Reload(); err != nil {
				d.Log("Reload err: %s", err)
			}
			// 防止多个进程
			go d.hammerTime(DefaultHammerTime * time.Second)
		case syscall.SIGINT:
			d.Log(pid, "Received SIGINT.")
			if d.state == StateRunning {
				go d.hammerTime(DefaultHammerTime * time.Second)
				d.shutdown()
			}
		case syscall.SIGTERM:
			d.Log(pid, "Received SIGTERM.")
			if d.state == StateRunning {
				go d.hammerTime(DefaultHammerTime * time.Second)
				d.shutdown()
			}
		default:
			d.Log("Received %v: nothing i care about...", sig)
		}
		d.signalHooks(PostSignal, sig)
	}
}

func (d *Daemon) signalHooks(flag int, sig os.Signal) {
	if hs, ok := d.SignalHooks[flag][sig]; !ok {
		return
	} else {
		for _, h := range hs {
			h()
		}
	}

	return
}

/* {{{ func hammerTime()
 *
 */
func (d *Daemon) hammerTime(to time.Duration) {
	defer func() {
		if err := recover(); err != nil {
			d.Log("error: %s", err)
		}
	}()

	time.Sleep(to)
	d.Log("[STOP] Forcefully shutting down!")
	//d.shutdown()
	time.Sleep(20 * time.Millisecond)
	os.Exit(0)
}

/* }}} */
