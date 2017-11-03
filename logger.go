package wgo

import (
	"fmt"
	"log"
	"strings"
)

type (
	logger interface {
		AddConsole()
		DenyConsole()
		Debug(arg0 interface{}, args ...interface{})
		Info(arg0 interface{}, args ...interface{})
		Warn(arg0 interface{}, args ...interface{})
		Error(arg0 interface{}, args ...interface{})
		Log(arg0 interface{}, args ...interface{})
		Fatal(arg0 interface{}, args ...interface{})
	}
)

/* {{{ func nlog(arg0 interface{}, args ...interface{})
 * native log
 */
func nlog(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		// Use the first string as a format string
		log.Printf(first, args...)
	default:
		log.Printf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
}

/* }}} */

// wgo logging
/* {{{ func Debug()
 *
 */
func Debug(arg0 interface{}, args ...interface{}) { wgo.Debug(arg0, args...) }
func (w *WGO) Debug(arg0 interface{}, args ...interface{}) {
	if w != nil {
		w.Logger().Debug(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Info()
 *
 */
func Info(arg0 interface{}, args ...interface{}) { wgo.Info(arg0, args...) }
func (w *WGO) Info(arg0 interface{}, args ...interface{}) {
	if w != nil {
		w.Logger().Info(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}
func (w *WGO) Printf(format string, args ...interface{}) {
	if w != nil {
		w.Logger().Info(format, args...)
	} else {
		nlog(format, args...)
	}
}

/* }}} */

/* {{{ func Warn()
 *
 */
func Warn(arg0 interface{}, args ...interface{}) { wgo.Warn(arg0, args...) }
func (w *WGO) Warn(arg0 interface{}, args ...interface{}) {
	if w != nil {
		w.Logger().Warn(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Error()
 *
 */
func Error(arg0 interface{}, args ...interface{}) { wgo.Error(arg0, args...) }
func (w *WGO) Error(arg0 interface{}, args ...interface{}) {
	if w != nil {
		w.Logger().Error(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Log()
 *
 */
func (w *WGO) Log(arg0 interface{}, args ...interface{}) {
	if w != nil {
		w.Logger().Log(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Fatal()
 *
 */
func Fatal(arg0 interface{}, args ...interface{}) { wgo.Fatal(arg0, args...) }
func (w *WGO) Fatal(arg0 interface{}, args ...interface{}) {
	if w != nil {
		w.Logger().Fatal(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

// context logging
// Debug
func (c *Context) Debug(arg0 interface{}, args ...interface{}) {
	if rid := c.RequestID(); rid != "" {
		switch arg0.(type) {
		case string:
			arg0 = fmt.Sprintf("%s [%s]", arg0, rid)
		default:
			args = append(args, fmt.Sprint("[", rid, "]"))
		}
	}
	c.logger.Debug(arg0, args...)
}

// Info
func (c *Context) Info(arg0 interface{}, args ...interface{}) {
	if rid := c.RequestID(); rid != "" {
		switch arg0.(type) {
		case string:
			arg0 = fmt.Sprintf("%s [%s]", arg0, rid)
		default:
			args = append(args, fmt.Sprint("[", rid, "]"))
		}
	}

	c.logger.Info(arg0, args...)
}

// Warn
func (c *Context) Warn(arg0 interface{}, args ...interface{}) {
	if rid := c.RequestID(); rid != "" {
		switch arg0.(type) {
		case string:
			arg0 = fmt.Sprintf("%s [%s]", arg0, rid)
		default:
			args = append(args, fmt.Sprint("[", rid, "]"))
		}
	}
	c.logger.Warn(arg0, args...)
}

// Error
func (c *Context) Error(arg0 interface{}, args ...interface{}) {
	if rid := c.RequestID(); rid != "" {
		switch arg0.(type) {
		case string:
			arg0 = fmt.Sprintf("%s [%s]", arg0, rid)
		default:
			args = append(args, fmt.Sprint("[", rid, "]"))
		}
	}
	c.logger.Error(arg0, args...)
}
