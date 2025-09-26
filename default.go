package logger

import (
	"fmt"
)

var defaultLogger *Logger

func init() {
	defaultLogger = New(3)
	_ = defaultLogger.SetOutput(Console.Name(), Console)
}

func Write(msg *Message, stack ...string) {
	defaultLogger.Write(msg, stack...)
}
func Sprint(level Level, content string, stack ...string) {
	defaultLogger.Sprint(level, content, stack...)
}
func Fatal(f any, v ...any) {
	defaultLogger.Fatal(f, v...)
}
func Panic(f any, v ...any) {
	defaultLogger.Panic(f, v...)
}
func Error(f any, v ...any) {
	defaultLogger.Error(f, v...)
}

func Alert(f any, v ...any) {
	defaultLogger.Alert(f, v...)
}

func Debug(f any, v ...any) {
	defaultLogger.Debug(f, v...)
}

func Trace(f any, v ...any) {
	defaultLogger.Trace(f, v...)
}

func Info(f any, v ...any) {
	defaultLogger.Info(f, v...)
}

func Warn(f any, v ...any) {
	defaultLogger.Warn(f, v...)
}

// SetLevel 设置日志输出等级
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetFilePathFormatter 设置日志起始路径
func SetFilePathFormatter(f filePathFormatter) {
	defaultLogger.SetFilePathFormatter(f)
}

func SetCallDepth(depth int) {
	defaultLogger.SetCallDepth(depth)
}

func SetOutput(name string, output Output) error {
	return defaultLogger.SetOutput(name, output)
}
func DelOutput(name string) {
	defaultLogger.DelOutput(name)
}
func Format(format any, args ...any) (text string) {
	switch v := format.(type) {
	case string:
		text = v
	case error:
		text = v.Error()
	default:
		text = fmt.Sprintf("%v", format)
	}
	if len(args) > 0 {
		text = fmt.Sprintf(text, args...)
	}
	return
}
