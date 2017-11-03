package wlog

// SocketLogWriter socket log writer
type SocketLogWriter chan *LogRecord

// LogWrite 实现write接口
func (w SocketLogWriter) LogWrite(rec *LogRecord) {
	w <- rec
}

// Close 实现close接口
func (w SocketLogWriter) Close() {
	close(w)
}

// NewSocketLogWriter 创建呗
func NewSocketLogWriter(format, proto, hostport string) SocketLogWriter {

	w := SocketLogWriter(make(chan *LogRecord, LogBufferLength))

	return w
}
