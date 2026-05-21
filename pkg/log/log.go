package log

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync/atomic"
)

type Log struct {
	scope string
	mode  atomic.Uint32
}

type Mode uint32

const (
	ModeDebug Mode = 1 << iota
	ModeInfo
	ModeWarn
	ModeError

	ModeAllDebug   = ModeDebug | ModeInfo | ModeWarn | ModeError
	ModeAllRelease = ModeInfo | ModeWarn | ModeError

	ModeNone Mode = 0
)

func New(m Mode, scope ...string) *Log {
	var s string
	if len(scope) == 0 {
		s = ""
	} else {
		s = strings.Join(scope, "::")
	}

	l := &Log{scope: s}
	l.Set(m)

	return l
}
func (l *Log) Set(m Mode)      { l.mode.Store(uint32(m)) }
func (l *Log) Has(m Mode) bool { return Mode(l.mode.Load())&m != 0 }
func (l *Log) Add(m Mode)      { l.mode.Store(uint32(Mode(l.mode.Load()) | m)) }
func (l *Log) Remove(m Mode)   { l.mode.Store(uint32(Mode(l.mode.Load()) &^ m)) }
func (l *Log) Flag(m Mode, flag bool) {
	if flag {
		l.Add(m)
	} else {
		l.Remove(m)
	}
}

func (l *Log) Debug(w io.Writer, a ...any)              { l.log(w, ModeDebug, "debug", a...) }
func (l *Log) Debugf(w io.Writer, fmt string, a ...any) { l.logf(w, ModeDebug, "debug", fmt, a...) }
func (l *Log) Warn(w io.Writer, a ...any)               { l.log(w, ModeWarn, "warn", a...) }
func (l *Log) Warnf(w io.Writer, fmt string, a ...any)  { l.logf(w, ModeWarn, "warn", fmt, a...) }
func (l *Log) Error(w io.Writer, a ...any)              { l.log(w, ModeError, "error", a...) }
func (l *Log) Errorf(w io.Writer, fmt string, a ...any) { l.logf(w, ModeError, "error", fmt, a...) }
func (l *Log) Info(w io.Writer, a ...any)               { l.log(w, ModeInfo, "info", a...) }
func (l *Log) Infof(w io.Writer, fmt string, a ...any)  { l.logf(w, ModeInfo, "info", fmt, a...) }

func (l *Log) log(w io.Writer, mode Mode, level string, a ...any) {
	if l.Has(mode) {
		fmt.Fprint(w, l.scopeStr(level))
		fmt.Fprint(w, a...)
	}
}

func (l *Log) logf(w io.Writer, mode Mode, level, format string, a ...any) {
	if l.Has(mode) {
		fmt.Fprintf(w, l.scopeStr(level)+format, a...)
	}
}

func (l *Log) scopeStr(level string) string {
	scope := l.scope

	if l.scope == "" {
		_, file, line, ok := runtime.Caller(2)
		if !ok {
			scope = "package"
		} else {
			scope = fmt.Sprintf("%s@L%d", file, line)
		}
	}

	return fmt.Sprintf("%s(%s): ", level, scope)
}
