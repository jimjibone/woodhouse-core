package log

type Context struct {
	logger   *Logger
	name     string
	minLevel Level
}

func NewContext(logger *Logger, name string, minLevel Level) *Context {
	return &Context{
		logger:   logger,
		name:     name,
		minLevel: minLevel,
	}
}

func (cl *Context) printf(level Level, format string, args ...any) {
	if level >= cl.minLevel {
		cl.logger.printf(level, cl.name, format, args...)
	}
}

func (cl *Context) println(level Level, args ...any) {
	if level >= cl.minLevel {
		cl.logger.println(level, cl.name, args...)
	}
}

func (cl *Context) Debugf(format string, args ...any) {
	cl.printf(DebugLevel, format, args...)
}

func (cl *Context) Infof(format string, args ...any) {
	cl.printf(InfoLevel, format, args...)
}

func (cl *Context) Warnf(format string, args ...any) {
	cl.printf(WarnLevel, format, args...)
}

func (cl *Context) Errorf(format string, args ...any) {
	cl.printf(ErrorLevel, format, args...)
}

func (cl *Context) Fatalf(format string, args ...any) {
	cl.printf(FatalLevel, format, args...)
}

func (cl *Context) Debugln(args ...any) {
	cl.println(DebugLevel, args...)
}

func (cl *Context) Infoln(args ...any) {
	cl.println(InfoLevel, args...)
}

func (cl *Context) Warnln(args ...any) {
	cl.println(WarnLevel, args...)
}

func (cl *Context) Errorln(args ...any) {
	cl.println(ErrorLevel, args...)
}

func (cl *Context) Fatalln(args ...any) {
	cl.println(FatalLevel, args...)
}
