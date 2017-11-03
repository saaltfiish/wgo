package wlog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

// ConsoleLogWriter 终端输出
type ConsoleLogWriter struct {
	format string
	fbs    [][]byte // format byte slice
	record chan *LogRecord
}

// NewConsoleLogWriter 创建
func NewConsoleLogWriter(format string) *ConsoleLogWriter {
	w := &ConsoleLogWriter{
		format: format,
		fbs:    bytes.Split([]byte(format), []byte{'%'}),
		record: make(chan *LogRecord, LogBufferLength),
	}
	go w.run(os.Stdout)
	return w
}

func (c *ConsoleLogWriter) run(out io.Writer) {
	for rec := range c.record {
		fmt.Fprint(out, FormatLogRecord(c.fbs, rec))
	}
}

// LogWrite 接口
func (c *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	c.record <- rec
}

// Close 接口
func (c *ConsoleLogWriter) Close() {
	close(c.record)
	time.Sleep(50 * time.Millisecond) // Try to give console I/O time to complete
}
