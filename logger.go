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
	level             Level
	outputs           map[string]Output
	callDepth         int
	filePathFormatter filePathFormatter
	mutex             sync.Mutex
}

func New(depth ...int) *Logger {
	dep := append(depth, 2)[0]
	l := &Logger{}
	l.level = LevelError
	l.outputs = map[string]Output{}
	l.callDepth = dep
	return l
}
func (log *Logger) Close() error {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	var errs []error
	remainingOutputs := map[string]Output{}
	for k, output := range log.outputs {
		if err := output.Close(); err != nil {
			errs = append(errs, err)
			remainingOutputs[k] = output
		}
	}
	log.outputs = remainingOutputs
	if len(errs) > 0 {
		return fmt.Errorf("close logger error: %v", errs)
	}
	return nil
}
func (log *Logger) Write(msg *Message, stack ...string) {
	defer func() {
		_ = recover()
	}()
	if msg.Level < log.level {
		return
	}
	if msg.Time.IsZero() {
		msg.Time = time.Now()
	}
	if log.callDepth > 0 && msg.Path == "" {
		if _, file, lineno, ok := runtime.Caller(log.callDepth); ok {
			msg.Path = log.trimPath(file, lineno)
		}
	}
	if len(stack) > 0 {
		msg.Stack = stack[0]
	}
	for _, output := range log.outputs {
		output.Write(msg)
	}
}

func (log *Logger) Sprint(level Level, content string, stack ...string) {
	log.Write(&Message{Content: content, Level: level}, stack...)
}

func (log *Logger) Fatal(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelFatal, content, string(debug.Stack()))
	os.Exit(1)
}

func (log *Logger) Panic(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelPanic, content, string(debug.Stack()))
	panic(content)
}

// Error Log ERROR level message.
func (log *Logger) Error(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelError, content, string(debug.Stack()))
}
func (log *Logger) Alert(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelAlert, content)
}

// Debug Log DEBUG level message.
func (log *Logger) Debug(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelDebug, content)
}

// Trace Log TRAC level message.
func (log *Logger) Trace(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelTrace, content)
}

// Info Log INFO level message.
func (log *Logger) Info(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelInfo, content)
}

// Warn Log WARN level message.
func (log *Logger) Warn(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelWarn, content)
}

// SetLevel 设置日志输出等级
func (log *Logger) SetLevel(level Level) {
	log.level = level
}

func (log *Logger) SetCallDepth(depth int) {
	log.callDepth = depth
}

// SetFilePathFormatter 设置日志起始路径
func (log *Logger) SetFilePathFormatter(f filePathFormatter) {
	log.filePathFormatter = f
}

func (log *Logger) trimPath(fullPath string, lineno int) (r string) {
	if log.filePathFormatter != nil {
		return log.filePathFormatter(fullPath, lineno)
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
