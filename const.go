package logger

import "strings"

const (
	brushPrefix = "\033["
	brushSuffix = "\033[0m"
)

type Level int8

// 日志等级，数值越大优先级越高
const (
	LevelDebug Level = 0 // 调试信息，最详细的日志
	LevelTrace Level = 1 // 追踪信息, 例如: 函数调用, 变量值
	LevelInfo  Level = 2 // 一般信息，正常运行状态, 例如: 服务启动, 数据库连接成功
	LevelWarn  Level = 3 // 警告信息，可能的问题, 例如: 配置错误, 资源不足
	LevelAlert Level = 4 // 警报信息，需要关注的问题, 例如: 数据库连接失败, 内存泄漏
	LevelError Level = 5 // 错误信息，发生错误但不影响程序运行, 例如: 文件读写错误, 网络连接错误
	LevelPanic Level = 6 // 严重错误，抛出panic但不终止程序
	LevelFatal Level = 7 // 致命错误，记录日志后终止程序运行
)

// 日志记录等级字段（所有级别统一为5个字符长度，便于日志对齐）
var levelPrefix = map[Level]string{
	LevelDebug: "DEBUG",
	LevelTrace: "TRACE",
	LevelInfo:  "INFO+",
	LevelWarn:  "WARN+",
	LevelAlert: "ALERT",
	LevelError: "ERROR",
	LevelPanic: "PANIC",
	LevelFatal: "FATAL",
}

// 日志级别对应的终端颜色配置
var levelColors = map[Level]string{
	LevelDebug: "1;32", // 绿色
	LevelTrace: "1;36", // 青色
	LevelInfo:  "1;37", // 亮白色
	LevelWarn:  "1;33", // 黄色
	LevelAlert: "1;35", // 洋红色
	LevelError: "1;31", // 红色
	LevelPanic: "1;41", // 红色底白色字
	LevelFatal: "1;41", // 红色底白色字
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
