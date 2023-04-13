package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mutex     sync.Mutex
	level     Level
	usePath   string
	outputs   map[string]Output
	callDepth int
}

func New(depth ...int) *Logger {
	dep := append(depth, 2)[0]
	l := &Logger{}
	l.level = LevelError
	l.outputs = map[string]Output{}
	l.callDepth = dep
	return l
}

func (this *Logger) Write(msg *Message) {
	if msg.Time.IsZero() {
		msg.Time = time.Now()
	}
	if this.callDepth > 0 && msg.Path == "" {
		_, file, lineno, ok := runtime.Caller(this.callDepth)
		if ok {
			if this.usePath != "" {
				file = stringTrim(file, this.usePath)
			}
			msg.Path = strings.Replace(fmt.Sprintf("%s:%d", file, lineno), "%2e", ".", -1)
		}
	}
	if msg.Level >= LevelError && msg.Stack == "" {
		msg.Stack = string(debug.Stack())
	}
	for name, output := range this.outputs {
		if err := output.Write(msg); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "unable to WriteMsg to adapter:%v,error:%v\n", name, err)
		}
	}
}

func (this *Logger) writeMsg(level Level, format any, args ...any) {
	if level < this.level {
		return
	}
	this.Write(&Message{Content: Sprintf(format, args...), Level: level})
}

func (this *Logger) Fatal(format any, args ...any) {
	this.writeMsg(LevelFATAL, format, args...)
	os.Exit(1)
}

func (this *Logger) Panic(format any, args ...any) {
	this.writeMsg(LevelPanic, format, args...)
	panic(Sprintf(format, args...))
}

// Error Log ERROR level message.
func (this *Logger) Error(format interface{}, v ...interface{}) {
	this.writeMsg(LevelError, format, v...)
}
func (this *Logger) Alert(format interface{}, args ...interface{}) {
	this.writeMsg(LevelAlert, format, args...)
}

// Debug Log DEBUG level message.
func (this *Logger) Debug(format interface{}, v ...interface{}) {
	this.writeMsg(LevelDebug, format, v...)
}

// Trace Log TRAC level message.
func (this *Logger) Trace(format interface{}, v ...interface{}) {
	this.writeMsg(LevelTrace, format, v...)
}

// SetLevel 设置日志输出等级
func (this *Logger) SetLevel(level Level) {
	this.level = level
}

// SetPathTrim 设置日志起始路径
func (this *Logger) SetPathTrim(trimPath string) {
	this.usePath = filepath.ToSlash(trimPath)
}

func (this *Logger) SetCallDepth(depth int) {
	this.callDepth = depth
}

func stringTrim(s string, cut string) string {
	ss := strings.SplitN(s, cut, 2)
	if 1 == len(ss) {
		return ss[0]
	}
	return ss[1]
}
