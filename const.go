package logger

import "strings"

const (
	brushPrefix = "\033["
	brushSuffix = "\033[0m"
)

type Level int8

// 日志等级，从0-7，日优先级由高到低
const (
	LevelTrace Level = 0 // 用户级基本输出
	LevelDebug       = 1 // 用户级调试
	LevelAlert       = 2
	LevelError       = 70 // 用户级错误
	LevelPanic       = 80 //Panic
	LevelFATAL       = 90 //PANIC
)

// 日志记录等级字段
var levelPrefix = map[Level]string{
	LevelTrace: "TRACE",
	LevelDebug: "DEBUG",
	LevelError: "ERROR",
	LevelAlert: "ALERT",
	LevelPanic: "PANIC",
	LevelFATAL: "FATAL",
}

// 鉴于终端的通常使用习惯，一般白色和黑色字体是不可行的,所以30,37不可用，
var levelColors = map[Level]string{
	LevelTrace: "1;32", //绿色
	LevelDebug: "1;32", //绿色
	LevelAlert: "1;33", //黄色
	LevelError: "1;31", //红色
	LevelPanic: "1;41", //红色底
	LevelFATAL: "1;41", //红色底
}

func (l Level) String() string {
	return levelPrefix[l]
}

func (l Level) Brush(text string) string {
	if color, ok := levelColors[l]; !ok {
		return text
	} else {
		return strings.Join([]string{brushPrefix, color, "m", text, brushSuffix}, "")
	}
}

type Interface interface {
	Fatal(format any, args ...any) //终止程序运行
	Panic(format any, args ...any) //抛出Panic
	Error(format any, args ...any)
	Alert(format any, args ...any)
	Debug(format any, args ...any)
	Trace(format any, args ...any)
}
