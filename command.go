package wgo

import (
	"fmt"
	"os"
	"syscall"
	"time"

	// project
	"wgo/daemon"
)

func interceptCmd(tag string) {
	if running, pid := daemon.CheckStatus(wgo.Daemon.PidFile); running {
		fmt.Printf("%s is running at %d\n", wgo.Daemon.ProcName, pid)
		switch tag {
		case "status":
			// do nothing
		case "stop":
			if pid > 0 {
				if proc, err := os.FindProcess(pid); err != nil {
					fmt.Printf("can't find proc by pid(%d): %s", pid, err)
				} else if err := proc.Signal(syscall.SIGINT); err != nil {
					fmt.Printf("send sig(%s) to %d error: %s", syscall.SIGINT, pid, err)
				}
				fmt.Printf("%s is stopping \n", wgo.Daemon.ProcName)
			} else {
				fmt.Printf("%s's pid lost\n", wgo.Daemon.ProcName)
			}
		case "reload":
			if pid > 0 {
				if proc, err := os.FindProcess(pid); err != nil {
					fmt.Printf("can't find proc by pid(%d): %s", pid, err)
				} else if err := proc.Signal(syscall.SIGHUP); err != nil {
					fmt.Printf("send sig(%s) to %d error: %s", syscall.SIGHUP, pid, err)
				}
				fmt.Printf("%s is reloading\n", wgo.Daemon.ProcName)
			} else {
				fmt.Printf("%s's pid lost\n", wgo.Daemon.ProcName)
			}
		case "help", "h":
			fmt.Println("working")
		default:
			fmt.Println("invalid command")
		}
	} else {
		fmt.Printf("%s not running\n", wgo.Daemon.ProcName)
	}
	time.Sleep(20 * time.Millisecond)
	os.Exit(0)
}
