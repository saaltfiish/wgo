package wlog

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"wgo/wlog/client"
)

// SocketLogWriter socket log writer
type SysLogWriter struct {
	format string
	fbs    [][]byte
	record chan *LogRecord
}

// LogWrite 实现write接口
func (w SysLogWriter) LogWrite(rec *LogRecord) {
	w.record <- rec
}

// Close 实现close接口
func (w SysLogWriter) Close() {
	close(w.record)
}

// NewSysLogWriter 创建呗
func NewSysLogWriter(format, network, raddr string, threadNum int, tag string) *SysLogWriter {
	w := &SysLogWriter{
		format: format,
		fbs:    bytes.Split([]byte(format), []byte{'%'}),
		record: make(chan *LogRecord, LogBufferLength),
	}
	for i := 0; i < threadNum; i++ {
		go w.run(network, raddr, tag)
	}
	return w
}

func (w *SysLogWriter) run(network, raddr, tag string) {
	logger, err := client.Dial(network, raddr, client.LOG_INFO, tag)
	if err != nil {
		panic("SysLog Connect Failed......")
	}

	for {
		rec, ok := <-w.record
		if !ok {
			fmt.Fprintf(os.Stderr, "SysLogWriter: channel close")
			return
		}
		err := logger.Info(FormatLogRecord(w.fbs, rec))
		if err != nil {
			fmt.Fprintf(os.Stderr, "SysLogWriter: %s\n", err)

			w.record <- rec
			time.Sleep(10 * time.Second)
		}
	}

}
