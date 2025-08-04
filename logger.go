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

type filePathFormatter func(string, int) string

type Logger struct {
	mutex sync.Mutex
	level Level
	//usePath   []string
	outputs           map[string]Output
	callDepth         int
	filePathFormatter filePathFormatter
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
	defer func() {
		_ = recover()
	}()
	if msg.Level < this.level {
		return
	}
	if msg.Time.IsZero() {
		msg.Time = time.Now()
	}
	if this.callDepth > 0 && msg.Path == "" {
		if _, file, lineno, ok := runtime.Caller(this.callDepth); ok {
			msg.Path = this.trimPath(file, lineno)
		}
	}
	if msg.Level >= LevelError && msg.Stack == "" {
		msg.Stack = string(debug.Stack())
	}
	for _, output := range this.outputs {
		output.Write(msg)
	}
}

func (this *Logger) Sprint(level Level, format any, args ...any) {
	this.Write(&Message{Content: Sprintf(format, args...), Level: level})
}

func (this *Logger) Fatal(format any, args ...any) {
	this.Sprint(LevelFATAL, format, args...)
	os.Exit(1)
}

func (this *Logger) Panic(format any, args ...any) {
	this.Sprint(LevelPanic, format, args...)
	panic(Sprintf(format, args...))
}

// Error Log ERROR level message.
func (this *Logger) Error(format interface{}, v ...interface{}) {
	this.Sprint(LevelError, format, v...)
}
func (this *Logger) Alert(format interface{}, args ...interface{}) {
	this.Sprint(LevelAlert, format, args...)
}

// Debug Log DEBUG level message.
func (this *Logger) Debug(format interface{}, v ...interface{}) {
	this.Sprint(LevelDebug, format, v...)
}

// Trace Log TRAC level message.
func (this *Logger) Trace(format interface{}, v ...interface{}) {
	this.Sprint(LevelTrace, format, v...)
}

// SetLevel 设置日志输出等级
func (this *Logger) SetLevel(level Level) {
	this.level = level
}

func (this *Logger) SetCallDepth(depth int) {
	this.callDepth = depth
}

// SetFilePathFormatter 设置日志起始路径
func (this *Logger) SetFilePathFormatter(f filePathFormatter) {
	this.filePathFormatter = f
}

func (this *Logger) trimPath(fullPath string, lineno int) (r string) {
	if this.filePathFormatter != nil {
		return this.filePathFormatter(fullPath, lineno)
	}
	var filePath string
	if i := strings.LastIndex(fullPath, ".com/"); i >= 0 {
		filePath = fmt.Sprintf("%s:%d", fullPath[i+5:], lineno)
	} else {
		dir := filepath.Dir(fullPath)
		pkg := filepath.Base(dir) // 包名
		file := filepath.Base(fullPath)
		filePath = fmt.Sprintf("%s/%s:%d", pkg, file, lineno)
	}

	return strings.Replace(filePath, "%2e", ".", -1)
}
