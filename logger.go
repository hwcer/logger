package logger

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	level     Level
	outputs   map[string]Output
	callDepth int
	mutex     sync.Mutex //仅串行化 SetOutput/RemoveOutput/Close 的读-改-写(写侧 COW,换引用不改底表)
}

func New(depth ...int) *Logger {
	dep := 2
	if len(depth) > 0 {
		dep = depth[0]
	}
	l := &Logger{}
	l.level = LevelTrace
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
	return errors.Join(errs...)
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
	if len(stack) > 0 {
		msg.Stack = stack[0]
	}
	// outputs/level 仅初始化期配置,运行期 SetOutput/SetLevel 与此读的竞争
	// 后果可接受:写侧 COW 保证 map 不损坏,最坏向已 Close 的输出写一条
	// 被 recover 吞掉——日志库不得为配置期写入拖累打日志的热路径
	for _, output := range log.outputs {
		output.Write(msg)
	}
}

func (log *Logger) Sprint(level Level, content string, stack ...string) {
	log.Write(&Message{Content: content, Level: level}, stack...)
}

func (log *Logger) Fatal(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelFatal, content, log.callerStack())
	os.Exit(1)
}

func (log *Logger) Panic(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelPanic, content, log.callerStack())
	panic(content)
}

func (log *Logger) Error(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelError, content, log.callerStack())
}

func (log *Logger) Warn(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelWarn, content)
}

func (log *Logger) Info(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelInfo, content)
}

func (log *Logger) Debug(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelDebug, content)
}

func (log *Logger) Trace(format any, args ...any) {
	content := Format(format, args...)
	log.Sprint(LevelTrace, content)
}

func (log *Logger) SetLevel(level Level) {
	log.level = level
}
func (log *Logger) GetLevel() Level {
	return log.level
}
func (log *Logger) SetCallDepth(depth int) {
	log.callDepth = depth
}

func (log *Logger) callerStack() string {
	var pcs [32]uintptr
	n := runtime.Callers(log.callDepth+1, pcs[:])
	if n == 0 {
		return ""
	}
	var b strings.Builder
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		_, _ = fmt.Fprintf(&b, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return b.String()
}
