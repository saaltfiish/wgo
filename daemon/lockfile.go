// +build darwin freebsd linux

// Package daemon runs a program as a Unix daemon.
package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

/* {{{ func LockFile(file string) error
 * only support linux/unix
 */
func LockFile(file string) (f *os.File, err error) {
	if f, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0755); err != nil {
		//fmt.Printf("open failed: %s\n", err)
		return
	} else if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		//fmt.Printf("lock failed: %s\n", err)
		return
	}
	return
}

/* }}} */

/* {{{ func GetPidFromFile(file string) (int, error)
 *
 */
func GetPidFromFile(file string) (int, error) {
	if data, err := ioutil.ReadFile(file); err != nil {
		return 0, err
	} else if pid, err := strconv.ParseInt(strings.TrimSpace(string(data)), 0, 32); err != nil {
		return 0, err
	} else {
		return int(pid), nil
	}
}

/* }}} */

/* {{{ func LockPidFile(file string) (*os.File, error)
 *
 */
func LockPidFile(file string) (*os.File, error) {
	if f, err := LockFile(file); err != nil {
		return nil, err
	} else if _, err := f.WriteString(fmt.Sprint(os.Getpid())); err != nil {
		return nil, err
	} else if err := f.Truncate(int64(len(fmt.Sprint(os.Getpid())))); err != nil {
		return nil, err
	} else {
		return f, nil
	}
}

/* }}} */

/* {{{ func CheckStatus(file string) (running bool, pid int)
 *
 */
func CheckStatus(file string) (running bool, pid int) {
	if _, err := LockFile(file); err == nil { // lock成功
		return
	}
	running = true
	pid, _ = GetPidFromFile(file)
	return
}

/* }}} */
