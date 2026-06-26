package logger

import "strings"

const (
	brushPrefix = "\033["
	brushSuffix = "\033[0m"
)

type Level int8

// 日志等级，数值越大优先级越高（与 slog/zap/logrus/zerolog 一致）
const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelPanic
	LevelFatal
)

var levelName = map[Level]string{
	LevelTrace: "TRAC",
	LevelDebug: "DBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERRO",
	LevelPanic: "PANC",
	LevelFatal: "FATL",
}

// 日志级别对应的终端颜色配置
var levelColors = map[Level]string{
	LevelTrace: "1;36", // 青色
	LevelDebug: "1;32", // 绿色
	LevelInfo:  "1;37", // 亮白色
	LevelWarn:  "1;33", // 黄色
	LevelError: "1;31", // 红色
	LevelPanic: "1;41", // 红色底白色字
	LevelFatal: "1;41", // 红色底白色字
}

func SetLevelName(level Level, name string) {
	levelName[level] = name
}

func (l Level) String() string {
	return levelName[l]
}

func (l Level) Brush(text string) string {
	if color, ok := levelColors[l]; !ok {
		return text
	} else {
		return strings.Join([]string{brushPrefix, color, "m", text, brushSuffix}, "")
	}
}
