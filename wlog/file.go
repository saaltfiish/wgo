package wlog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 不建议使用文件作为日志存储介质,慢慢慢,慢的不行不行的,而且只能一个协程同步写入,建议用 tcp 或者udp
type FileLogWriter struct {
	Mkdir    bool
	Daily    bool
	Maxsize  int
	MaxDays  int64
	curSize  int
	openTime time.Time
	openDay  int

	rec      chan *LogRecord
	filename string
	file     *os.File
	format   string
	fbs      [][]byte
}

// This is the FileLogWriter's output method
func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *FileLogWriter) Close() {
	close(w.rec)
	w.file.Sync()
}

func NewFileLogWriter(format, fname string, mkdir, daily bool, maxsize, maxdays int) *FileLogWriter {
	w := &FileLogWriter{
		Mkdir:    mkdir,
		Daily:    daily,
		Maxsize:  maxsize,
		MaxDays:  int64(maxdays),
		rec:      make(chan *LogRecord, LogBufferLength),
		filename: fname,
		format:   format,
		fbs:      bytes.Split([]byte(format), []byte{'%'}),
	}
	w.run()
	return w
}

// open file
func (w *FileLogWriter) createLogFile() (*os.File, error) {
	if w.file != nil && int(w.file.Fd()) > 0 {
		return w.file, nil
	}
	if w.Mkdir { //自动创建日志目录
		dir := filepath.Dir(w.filename)
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				//mkdir
				if err := os.Mkdir(dir, 0755); err != nil {
					return nil, err
				}
			}
		}
	}
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err == nil {
		now := time.Now()
		w.file = fd
		w.openTime = now
		w.openDay = now.Day()
		//curSize
		fi, err := fd.Stat()
		if err != nil {
			return nil, fmt.Errorf("get stat err: %s", err)
		}
		w.curSize = int(fi.Size())
	}
	return fd, err
}

func (w *FileLogWriter) run() {
	if _, err := w.createLogFile(); err != nil {
		panic(err)
	}

	go func() {
		for {
			rec, ok := <-w.rec
			if !ok {
				return
			}

			// check
			if err := w.check(rec); err != nil {
				fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
				return
			}

			_, err := fmt.Fprint(w.file, FormatLogRecord(w.fbs, rec))
			if err != nil {
				fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
				return
			}
		}
	}()

}

// 写消息前check
func (w *FileLogWriter) check(rec *LogRecord) error {
	now := time.Now()
	mLen := len(rec.Message)
	if (w.Maxsize > 0 && (w.curSize+mLen) > w.Maxsize) || (w.Daily && w.openDay != now.Day()) {
		return w.rotate()
	}
	w.curSize += mLen
	return nil
}

// rotate
func (w *FileLogWriter) rotate() error {
	_, err := w.file.Stat()
	if err == nil { // file exists
		// Find the next available number
		num := 1
		fname := ""
		for ; err == nil && num <= 999; num++ { //最多999个,超过就覆盖(unsafe)
			fname = w.filename + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), num)
			_, err = os.Lstat(fname)
		}

		// rotate log file
		w.file.Close()
		err = os.Rename(w.filename, fname)
		if err != nil {
			return fmt.Errorf("Rotate failed: %s\n", err)
		}

		// create new log file
		if _, err := w.createLogFile(); err != nil {
			return fmt.Errorf("Rotate failed: %s\n", err)
		}

		// delete old log files
		go w.deleteOldFiles()
	}
	return nil
}

func (w *FileLogWriter) deleteOldFiles() {
	dir := filepath.Dir(w.filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		fmt.Printf("old file(%s) info: %v", path, info)
		if !info.IsDir() && strings.HasPrefix(filepath.Base(path), filepath.Base(w.filename)) && info.ModTime().Unix() < (time.Now().Unix()-60*60*24*w.MaxDays) {
			os.Remove(path)
		}
		return nil
	})
}
