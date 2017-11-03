package wlog

import (
	"bytes"
	"fmt"
)

const (
	// colors
	nocolor = 0
	red     = 41
	green   = 42
	yellow  = 43
	blue    = 44
	magenta = 45
	gray    = 47
)

// Known format codes:
// %T - Time (2006-01-02 15:04:05.000)
// %L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
// %C - colored Level, for console logging
// %S - timestamp
// %M - Message
// %E - Environ
// Ignores unknown formats
// Recommended: "%T%E[%L] %M"
func FormatLogRecord(fbs [][]byte, rec *LogRecord) string {
	if rec == nil {
		return "<nil>"
	}

	out := bytes.NewBuffer(make([]byte, 0, 64))

	// Iterate over the fbs, replacing known formats
	for i, piece := range fbs {
		if i > 0 && len(piece) > 0 {
			switch piece[0] {
			case 'T':
				//out.WriteString(rec.Created.Format("2006-01-02 15:04:05.000"))
				out.WriteString(rec.Created.Format("2006-01-02T15:04:05.000Z07:00"))
			case 'L':
				out.WriteString(rec.Level.desc[0:4])
			case 'C':
				color := nocolor
				switch rec.Level.offset {
				case DEBUG:
					color = gray
				case TRACE:
					color = magenta
				case INFO:
					color = blue
				case WARNING:
					color = green
				case ERROR:
					color = yellow
				case CRITICAL, FATAL:
					color = red
				}
				//out.WriteString(fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, rec.Level.desc[0:4]))
				out.WriteString(fmt.Sprintf("\x1b[30;%dm%s\x1b[0m", color, rec.Level.desc[0:4]))
			case 'M':
				out.WriteString(rec.Message)
			case 'S':
				out.WriteString(fmt.Sprint(rec.Created.Unix()))
			case 'E':
				out.WriteString(fmt.Sprintf("[%d]%s", pid, rec.Tag))
			}
			if len(piece) > 1 {
				out.Write(piece[1:])
			}
		} else if len(piece) > 0 {
			out.Write(piece)
		}
	}
	out.WriteByte('\n')

	return out.String()
}
