// +build darwin freebsd linux

// Package daemon runs a program as a Unix daemon.
package daemon

// Copyright (c) 2013-2015 VividCortex, Inc. All rights reserved.
// Please see the LICENSE file for applicable license terms.

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Environment variables to support this process
const (
	stageVar         = "__DAEMON_STAGE"
	reloadVar        = "__DAEMON_RELOAD"
	listenerCountVar = "__DAEMON_LC"
	fdVarPrefix      = "__DAEMON_FD_"

	StateInit = iota
	StateRunning
	StateShuttingDown
	StateTerminate
)

// Daemon describes the options that apply to daemonization
type (
	Daemon struct {
		Daemonize   bool        // daemonize or not
		Dockerize   bool        // dockerize or not
		ExecPath    string      // exec path
		ProcName    string      // child's os.Args[0]; copied from parent if empty
		WorkDir     string      // work dir
		PidFile     string      // pid file path
		Files       []**os.File // 文件
		SignalHooks map[int]map[os.Signal][]func()

		logger   logger // Logger for this package
		state    uint8
		sigChan  chan os.Signal
		lp       []net.Listener
		pidlock  *os.File
		shutdown func() // shutdown function
	}
)

// make a daemon var
var daemon *Daemon

// DaemonStage tells in what stage in the process we are. See Stage().
type DaemonStage int

// Stages in the daemonizing process.
const (
	StageParent = DaemonStage(iota) // Original process
	StageChild                      // Spawn() called once: first child
	StageDaemon                     // Spawn() run twice: final daemon

	stageUnknown = DaemonStage(-1)
)

func saveFileName(fd int, name string) {
	// We encode in hex to avoid issues with filename encoding, and to be able
	// to separate it from the original variable value (if set) that we want to
	// keep. Otherwise, all non-zero characters are valid in the name, and we
	// can't insert a zero in the var as a separator.
	fdVar := fdVarPrefix + fmt.Sprint(fd)
	value := fmt.Sprintf("%s:%s",
		hex.EncodeToString([]byte(name)), os.Getenv(fdVar))

	if err := os.Setenv(fdVar, value); err != nil {
		fmt.Fprintf(os.Stderr, "can't set %s: %s\n", fdVar, err)
		os.Exit(1)
	}
}

func getFileName(fd int) string {
	fdVar := fdVarPrefix + fmt.Sprint(fd)
	value := os.Getenv(fdVar)
	sep := bytes.IndexByte([]byte(value), ':')

	if sep < 0 {
		fmt.Fprintf(os.Stderr, "bad fd var %s\n", fdVar)
		os.Exit(1)
	}
	name, err := hex.DecodeString(value[:sep])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decoding %s\n", fdVar)
		os.Exit(1)
	}
	return string(name)
}

func resetFileName(fd int) {
	fdVar := fdVarPrefix + fmt.Sprint(fd)
	value := os.Getenv(fdVar)
	sep := bytes.IndexByte([]byte(value), ':')

	if sep < 0 {
		fmt.Fprintf(os.Stderr, "bad fd var %s\n", fdVar)
		os.Exit(1)
	}
	if err := os.Setenv(fdVar, value[sep+1:]); err != nil {
		fmt.Fprintf(os.Stderr, "can't reset %s\n", fdVar)
		os.Exit(1)
	}
}

// currStage keeps the current stage. This is used only as a cache for Stage(),
// in order to extend a valid result after Spawn() has returned, where the
// environment variable would have already been reset. (Also, this is faster
// than repetitive calls to getStage().) Note that this approach is valid cause
// the stage doesn't change throughout any single process execution. It does
// only for the next process after the Spawn() call.
var currStage = stageUnknown

// Stage returns the "stage of daemonizing", i.e., it allows you to know whether
// you're currently working in the parent, first child, or the final daemon.
// This is useless after the call to Spawn(), cause that call will only
// return for the daemon stage. However, you can still use Stage() to tell
// whether you've daemonized or not, in case you have a running path that may
// exclude the call to Spawn().
func Stage() DaemonStage {
	if currStage == stageUnknown {
		s, _, _ := getStage()
		currStage = DaemonStage(s)
	}
	return currStage
}

// String returns a humanly readable daemonization stage.
func (s DaemonStage) String() string {
	switch s {
	case StageParent:
		return "parent"
	case StageChild:
		return "first child"
	case StageDaemon:
		return "daemon"
	default:
		return "unknown"
	}
}

// Returns the current stage in the "daemonization process", that's kept in
// an environment variable. The variable is instrumented with a digital
// signature, to avoid misbehavior if it was present in the user's
// environment. The original value is restored after the last stage, so that
// there's no final effect on the environment the application receives.
func getStage() (stage int, advanceStage func() error, resetEnv func() error) {
	var origValue string
	stage = 0

	daemonStage := os.Getenv(stageVar)
	stageTag := strings.SplitN(daemonStage, ":", 2)
	stageInfo := strings.SplitN(stageTag[0], "/", 3)

	if len(stageInfo) == 3 {
		stageStr, tm, check := stageInfo[0], stageInfo[1], stageInfo[2]

		hash := sha1.New()
		hash.Write([]byte(stageStr + "/" + tm + "/"))

		if check != hex.EncodeToString(hash.Sum([]byte{})) {
			// This whole chunk is original data
			origValue = daemonStage
		} else {
			stage, _ = strconv.Atoi(stageStr)

			if len(stageTag) == 2 {
				origValue = stageTag[1]
			}
		}
	} else {
		origValue = daemonStage
	}

	advanceStage = func() error {
		base := fmt.Sprintf("%d/%09d/", stage+1, time.Now().Nanosecond())
		hash := sha1.New()
		hash.Write([]byte(base))
		tag := base + hex.EncodeToString(hash.Sum([]byte{}))

		if err := os.Setenv(stageVar, tag+":"+origValue); err != nil {
			return fmt.Errorf("can't set %s: %s", stageVar, err)
		}
		return nil
	}
	resetEnv = func() error {
		os.Setenv(reloadVar, "")
		os.Setenv(listenerCountVar, "")
		return os.Setenv(stageVar, origValue)
	}

	return stage, advanceStage, resetEnv
}

/* {{{ func New() *Daemon
 * Odin style: 所有lib都用New()方法返回一个主类型
 */
func New() *Daemon {
	return &Daemon{}
}

/* }}} */

/* {{{ func (d *Daemon) WithDefaults() *Daemon
 * Odin sytle: WithDefaults赋予默认值
 */
func (d *Daemon) WithDefaults() *Daemon {
	d.shutdown = func() { // 默认shutdown函数
		time.Sleep(20 * time.Millisecond)
		os.Exit(0)
	}
	return d
}

/* }}} */

/* {{{ func (d *Daemon) Register(l logger) *Daemon
 * Odin sytle: 所有lib都用Register方法给lib内置变量赋值
 */
func (d *Daemon) Register(l logger) *Daemon {
	d.SetLogger(l)
	daemon = d
	return daemon
}

/* }}} */

/* {{{ func (d *Daemon) MakeDaemon() error
 * modified by Odin from MakeDaemon() 2016-09-19 10:29:12
 */
func MakeDaemon() error {
	if daemon != nil {
		return daemon.MakeDaemon()
	}
	return fmt.Errorf("[MakeDaemon] daemon is not registered")
}
func (d *Daemon) MakeDaemon() error {
	if !d.Dockerize { // docker模式不需要spawn
		if _, _, err := d.Spawn(); err != nil {
			return fmt.Errorf("spawn %s failed: %s", d.ProcName, err)
		}
	}

	//check&write pidfile
	dir := filepath.Dir(d.PidFile)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			//mkdir
			if err := os.Mkdir(dir, 0755); err != nil {
				return fmt.Errorf("create pid dir(%s) failed: %s", dir, err)
			}
		} else {
			return fmt.Errorf("pid dir(%s) wrong: %s", dir, err)
		}
	}
	if f, err := LockPidFile(d.PidFile); err != nil {
		return fmt.Errorf("Already running: %s", err)
	} else {
		d.pidlock = f
	}

	// handle signals
	go d.handleSignals()

	// running state
	d.state = StateRunning

	return nil
}

/* }}} */

/* {{{ func (d *Daemon) Spawn() (io.Reader, io.Reader, error)
 *
 */
func Spawn() (io.Reader, io.Reader, error) {
	if daemon != nil {
		return daemon.Spawn()
	}
	return nil, nil, fmt.Errorf("[Spawn] daemon is not registered")
}
func (d *Daemon) Spawn() (io.Reader, io.Reader, error) {
	if reloaded := os.Getenv(reloadVar); reloaded != "" {
		return d.reloaded()
	}
	stage, advanceStage, resetEnv := getStage()

	// This is a handy wrapper to do the proper thing in case of fareloadVartal
	// conditions. For the first stage you may want to recover, so it will
	// return the error. Otherwise it will exit the process, cause you'll be
	// half-way with some descriptors already changed. There's no chance to
	// write to stdout or stderr in the later case; they'll be already closed.
	fatal := func(err error) (io.Reader, io.Reader, error) {
		if stage > 0 {
			os.Exit(1)
		}
		resetEnv()
		return nil, nil, err
	}

	fileCount := 3 + len(d.Files)
	files := make([]*os.File, fileCount, fileCount+2)

	if stage == 0 {
		if d.Daemonize {
			// Descriptors 0, 1 and 2 are fixed in the "os" package. If we close
			// them, the process may choose to open something else there, with bad
			// consequences if some write to os.Stdout or os.Stderr follows (even
			// from Go's library itself, through the default log package). We thus
			// reserve these descriptors to avoid that.
			nullDev, err := os.OpenFile("/dev/null", 0, 0)
			if err != nil {
				return fatal(err)
			}
			files[0], files[1], files[2] = nullDev, nullDev, nullDev
		} else {
			files[0], files[1], files[2] = os.Stdin, os.Stdout, os.Stderr
		}

		fd := 3
		for _, fPtr := range d.Files {
			files[fd] = *fPtr
			saveFileName(fd, (*fPtr).Name())
			fd++
		}
	} else {
		files[0], files[1], files[2] = os.Stdin, os.Stdout, os.Stderr

		fd := 3
		for _, fPtr := range d.Files {
			*fPtr = os.NewFile(uintptr(fd), getFileName(fd))
			syscall.CloseOnExec(fd)
			files[fd] = *fPtr
			fd++
		}
	}

	if (stage < 1 && !d.Daemonize) || (stage < 2 && d.Daemonize) {
		// getExecutablePath() is OS-specific.
		execPath := d.ExecPath
		if execPath == "" {
			if ep, err := GetExecutablePath(); err != nil {
				return fatal(fmt.Errorf("can't determine full path to executable"))
			} else {
				execPath = ep
			}
		}

		// If getExecutablePath() returns "" but no error, determinating the
		// executable path is not implemented on the host OS, so daemonization
		// is not supported.
		if len(execPath) == 0 {
			return fatal(fmt.Errorf("can't determine full path to executable"))
		}

		if stage == 1 && d.Daemonize {
			files = files[:fileCount+2]

			var fe error
			// stdout: write at fd:1, read at fd:fileCount
			if files[fileCount], files[1], fe = os.Pipe(); fe != nil {
				return fatal(fe)
			}
			// stderr: write at fd:2, read at fd:fileCount+1
			if files[fileCount+1], files[2], fe = os.Pipe(); fe != nil {
				return fatal(fe)
			}
		}

		if err := advanceStage(); err != nil {
			return fatal(err)
		}
		dir, _ := os.Getwd()
		osAttrs := os.ProcAttr{Dir: dir, Env: os.Environ(), Files: files}

		if stage == 0 && d.Daemonize {
			sysattrs := syscall.SysProcAttr{Setsid: true}
			osAttrs.Sys = &sysattrs
		}

		procName := d.ProcName
		if len(procName) == 0 {
			procName = os.Args[0]
		}
		args := append([]string{procName}, os.Args[1:]...)
		proc, err := os.StartProcess(execPath, args, &osAttrs)
		if err != nil {
			return fatal(fmt.Errorf("can't create process %s: %s", procName, err))
		}
		proc.Release()
		os.Exit(0)
	}

	if d.WorkDir == "" {
		d.WorkDir = "/"
	}
	if err := os.Chdir(d.WorkDir); err != nil {
		return fatal(fmt.Errorf("can't change workdir %s: %s", d.WorkDir, err))
	}
	syscall.Umask(0)
	resetEnv()

	for fd := 3; fd < fileCount; fd++ {
		resetFileName(fd)
	}
	currStage = DaemonStage(stage)

	var stdout, stderr *os.File
	if d.Daemonize {
		stdout = os.NewFile(uintptr(fileCount), "stdout")
		stderr = os.NewFile(uintptr(fileCount+1), "stderr")
		if d.logger != nil {
			go func(reader io.Reader) {
				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					d.logger.Log("%s [stdout]", scanner.Text())
				}
			}(stdout)
			go func(reader io.Reader) {
				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					d.logger.Log("%s [stderr]", scanner.Text())
				}
			}(stderr)
		}
	}
	return stdout, stderr, nil
}

/* }}} */

/* {{{ func (d *Daemon) reloaded() (io.Reader, io.Reader, error)
 *
 */
func (d *Daemon) reloaded() (io.Reader, io.Reader, error) {
	lc, _ := strconv.Atoi(os.Getenv(listenerCountVar))

	fatal := func(err error) (io.Reader, io.Reader, error) {
		os.Setenv(reloadVar, "")
		os.Setenv(listenerCountVar, "")
		return nil, nil, err
	}

	fileCount := 3 + lc
	d.Log("listener count: %d, fileCount: %d", lc, fileCount)

	files := make([]*os.File, fileCount, fileCount+2)

	files[0], files[1], files[2] = os.Stdin, os.Stdout, os.Stderr

	fd := 3
	for _, fPtr := range d.Files {
		*fPtr = os.NewFile(uintptr(fd), getFileName(fd))
		syscall.CloseOnExec(fd)
		files[fd] = *fPtr
		fd++
	}
	if lc > 0 {
		d.Log("reload listener from %d files", lc)
		for i := 0; i < lc; i++ {
			f := os.NewFile(uintptr(fd), getFileName(fd))
			if nl, err := net.FileListener(f); err != nil {
				return fatal(err)
			} else {
				d.Log("find listener: %s", nl.Addr().String())
				d.AddListener(nl)
			}
			fd++
		}
	}

	if d.WorkDir == "" {
		d.WorkDir = "/"
	}
	if err := os.Chdir(d.WorkDir); err != nil {
		return fatal(fmt.Errorf("can't change workdir %s: %s", d.WorkDir, err))
	}
	syscall.Umask(0)
	os.Setenv(reloadVar, "")
	os.Setenv(listenerCountVar, "")

	for fd := 3; fd < fileCount; fd++ {
		resetFileName(fd)
	}

	var stdout, stderr *os.File
	if d.Daemonize {
		stdout = os.NewFile(uintptr(fileCount), "stdout")
		stderr = os.NewFile(uintptr(fileCount+1), "stderr")

		go func(reader io.Reader) {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				d.Log("%s [stdout]", scanner.Text())
			}
		}(stdout)
		go func(reader io.Reader) {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				d.Log("%s [stderr]", scanner.Text())
			}
		}(stderr)
	}

	// signal to parent
	ppid := os.Getppid()
	time.Sleep(5 * time.Millisecond)
	if pproc, err := os.FindProcess(ppid); err != nil {
		d.Log("find proc(%d) error: %s", ppid, err)
		return fatal(err)
	} else {
		if err := pproc.Signal(syscall.SIGINT); err != nil {
			d.Log("send sig(%d) to %d error: %s", syscall.SIGINT, ppid, err)
			return fatal(err)
		} else {
			d.Log("send pid(%d) sig(%d)", ppid, syscall.SIGINT)
		}
	}

	return stdout, stderr, nil
}

/* }}} */

/* {{{ func (d *Daemon) Reload() error
 * 重启自己,解锁pidfile
 */
func Reload() error {
	if daemon != nil {
		return daemon.Reload()
	}
	return fmt.Errorf("[Reload] daemon is not registered")
}
func (d *Daemon) Reload() error {
	defer func() {
		if err := recover(); err != nil {
			d.Log("Crashed with error: ", err)
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				d.Log(file, line)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}()

	files := []*os.File{os.Stdin, os.Stdout, os.Stderr}

	for _, fPtr := range d.Files {
		saveFileName(len(files), (*fPtr).Name())
		files = append(files, *fPtr)
	}
	// listener pool, reload时候需要保持
	for _, l := range d.lp {
		if f, err := l.(*net.TCPListener).File(); err == nil {
			saveFileName(len(files), f.Name())
			d.Log("save listener to file, fd: %d, addr: %s, file: %s", len(files), l.Addr().String(), f.Name())
			files = append(files, f)
		} else {
			d.Log("save listener to file failed, fd: %d, addr: %s", len(files), l.Addr().String())
		}
	}
	os.Setenv(listenerCountVar, fmt.Sprint(len(d.lp)))

	execPath := d.ExecPath
	if execPath == "" {
		d.Log("can't find execPath")
		return fmt.Errorf("can't determine full path to executable")
	} else {
		d.Log("find execPath: %s", execPath)
	}

	// If getExecutablePath() returns "" but no error, determinating the
	// executable path is not implemented on the host OS, so daemonization
	// is not supported.
	if len(execPath) == 0 {
		return fmt.Errorf("can't determine full path to executable")
	}

	if d.Daemonize { // wgo的daemonize需要capture stdout, stdin

		var fe error
		// stdout: write at fd:1, read at fd:fileCount
		var fro *os.File
		if fro, files[1], fe = os.Pipe(); fe != nil {
			return fe
		} else {
			files = append(files, fro)
		}
		// stderr: write at fd:2, read at fd:fileCount+1
		var fre *os.File
		if fre, files[2], fe = os.Pipe(); fe != nil {
			return fe
		} else {
			files = append(files, fre)
		}
	}

	os.Setenv(reloadVar, "reloading")
	dir, _ := os.Getwd()
	osAttrs := os.ProcAttr{Dir: dir, Env: os.Environ(), Files: files}

	// unlock pidfile, ignore errors
	if d.PidFile != "" && d.pidlock != nil {
		//if _, err := UnlockFile(d.PidFile); err != nil {
		if err := d.pidlock.Close(); err != nil {
			d.Log("unlock pidfile error: %s", err)
		} else {
			//fmt.Printf("unlock pidfile good: %s\n", d.PidFile)
			d.Log("unlock pidfile good: %s", d.PidFile)
		}
	} else {
		d.Log("can't unlock: %s", d.PidFile)
	}

	procName := d.ProcName
	if len(procName) == 0 {
		procName = os.Args[0]
	}
	procName = fmt.Sprintf("%s[reloaded]", procName)
	args := append([]string{procName}, os.Args[1:]...)
	if _, err := os.StartProcess(execPath, args, &osAttrs); err != nil {
		return fmt.Errorf("can't create process %s: %s", execPath, err)
	}

	return nil
}

/* }}} */

/* {{{ func (d *Daemon) AddListener() error
 * listener加入pool
 */
func AddListener(l net.Listener) error {
	if daemon != nil {
		return daemon.AddListener(l)
	}
	return fmt.Errorf("[AddListener] daemon is not registered")
}
func (d *Daemon) AddListener(l net.Listener) error {
	if len(d.lp) <= 0 {
		d.lp = make([]net.Listener, 0)
	} else if l != nil {
		for i, ol := range d.lp {
			if ol == nil {
				d.lp[i] = l
				return nil
			}
		}
	}
	d.lp = append(d.lp, l)
	return nil
}

/* }}} */

/* {{{ func (d *Daemon) GetListener(as string) (net.Listener, error)
 * listener加入pool
 */
func GetListener(as string) (net.Listener, error) {
	if daemon != nil {
		return daemon.GetListener(as)
	}
	return nil, fmt.Errorf("[GetListener] daemon is not registered")
}
func (d *Daemon) GetListener(as string) (net.Listener, error) {
	if len(d.lp) <= 0 {
		return nil, fmt.Errorf("no listener")
	}
	for _, l := range d.lp {
		if l != nil && l.Addr().String() == as {
			return l, nil
		}
	}
	return nil, fmt.Errorf("%s not found listener", as)
}

/* }}} */

/* {{{ func (d *Daemon) RegisterShutdown(f func())
 * 注册shutdown函数
 */
func RegisterShutdown(f func()) {
	if daemon != nil {
		daemon.RegisterShutdown(f)
		return
	}
	Log("[%d][RegisterShutdown] daemon is not registered", os.Getpid())
}
func (d *Daemon) RegisterShutdown(f func()) {
	if f != nil {
		d.shutdown = f
	} else {
		d.shutdown = func() {
			time.Sleep(10 * time.Millisecond)
			os.Exit(0)
		}
	}
}

/* }}} */
