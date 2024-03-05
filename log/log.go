package log

func Debug(format string, args ...any) {
	DefaultLogger.Debug(format, args...)
}

func Info(format string, args ...any) {
	DefaultLogger.Info(format, args...)
}

func Warn(format string, args ...any) {
	DefaultLogger.Warn(format, args...)
}

func Error(format string, args ...any) {
	DefaultLogger.Error(format, args...)
}

func Fatal(format string, args ...any) {
	DefaultLogger.Fatal(format, args...)
}
