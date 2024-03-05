package log

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

var DefaultLogger *Logger = NewLogger()

type Logger struct {
	Level      Level
	Out        *os.File
	TimeFormat string
}

type LoggerOpt func(l *Logger)

func NewLogger(opts ...LoggerOpt) *Logger {
	l := &Logger{
		Level:      DebugLevel,
		Out:        os.Stderr,
		TimeFormat: "15:04:05.000",
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func WithMinLevel(level Level) LoggerOpt {
	return func(l *Logger) {
		l.Level = level
	}
}

func WithOut(out *os.File) LoggerOpt {
	return func(l *Logger) {
		l.Out = out
	}
}

func WithTimeFormat(format string) LoggerOpt {
	return func(l *Logger) {
		l.TimeFormat = format
	}
}

func (l *Logger) printf(level Level, format string, args []any) {
	if l.Out != nil {
		// 15:04:05.000 DEBU message
		args = append([]any{
			color.HiBlackString(time.Now().Format(l.TimeFormat)),
			level.String(),
		}, args...)
		fmt.Fprintf(l.Out, "%s %s "+format+"\n", args...)
	}
}

func (l *Logger) Debug(format string, args ...any) {
	l.printf(DebugLevel, format, args)
}

func (l *Logger) Info(format string, args ...any) {
	l.printf(InfoLevel, format, args)
}

func (l *Logger) Warn(format string, args ...any) {
	l.printf(WarnLevel, format, args)
}

func (l *Logger) Error(format string, args ...any) {
	l.printf(ErrorLevel, format, args)
}

func (l *Logger) Fatal(format string, args ...any) {
	l.printf(FatalLevel, format, args)
	os.Exit(1)
}
