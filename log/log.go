package log

func SetOptions(opts ...LoggerOpt) {
	DefaultLogger.SetOptions(opts...)
}

func Debugf(format string, args ...any) {
	DefaultLogger.Debugf(format, args...)
}

func Infof(format string, args ...any) {
	DefaultLogger.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	DefaultLogger.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	DefaultLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	DefaultLogger.Fatalf(format, args...)
}

func Debugln(args ...any) {
	DefaultLogger.Debugln(args...)
}

func Infoln(args ...any) {
	DefaultLogger.Infoln(args...)
}

func Warnln(args ...any) {
	DefaultLogger.Warnln(args...)
}

func Errorln(args ...any) {
	DefaultLogger.Errorln(args...)
}

func Fatalln(args ...any) {
	DefaultLogger.Fatalln(args...)
}
