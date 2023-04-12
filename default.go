package logger

const DefaultConsoleName = "_defaultConsoleName"

var defaultLogger *Logger

func init() {
	defaultLogger = New(3)
	_ = defaultLogger.SetOutput(DefaultConsoleName, NewConsole())
}

func Default() *Logger {
	return defaultLogger
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

// Warn 废弃
func Warn(f any, v ...any) {
	defaultLogger.Alert(f, v...)
}

// Info 废弃
func Info(f any, v ...any) {
	defaultLogger.Alert(f, v...)
}

func Debug(f any, v ...any) {
	defaultLogger.Debug(f, v...)
}

func Trace(f any, v ...any) {
	defaultLogger.Trace(f, v...)
}
