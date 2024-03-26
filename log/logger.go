package log

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

var DefaultLogger *Logger = NewLogger()

type Logger struct {
	minLevel    Level
	out         *os.File
	timeFormat  string
	exitOnFatal bool
}

type LoggerOpt func(l *Logger)

func NewLogger(opts ...LoggerOpt) *Logger {
	l := &Logger{
		minLevel:    DebugLevel,
		out:         os.Stderr,
		timeFormat:  "06/01/02 15:04:05.000",
		exitOnFatal: true,
	}
	l.SetOptions(opts...)
	return l
}

func (l *Logger) SetOptions(opts ...LoggerOpt) {
	for _, opt := range opts {
		opt(l)
	}
}

func WithMinLevel(level Level) LoggerOpt {
	return func(l *Logger) {
		l.minLevel = level
	}
}

func WithOut(out *os.File) LoggerOpt {
	return func(l *Logger) {
		l.out = out
	}
}

func WithTimeFormat(format string) LoggerOpt {
	return func(l *Logger) {
		l.timeFormat = format
	}
}

func WithExitOnFatal(exitOnFatal bool) LoggerOpt {
	return func(l *Logger) {
		l.exitOnFatal = exitOnFatal
	}
}

func (l *Logger) printf(level Level, ctx string, format string, args ...any) {
	if level >= l.minLevel {
		if l.out != nil {
			// 15:04:05.000 DEBU [context] message
			joinedFormat := "%s %s "
			joinedArgs := []any{
				color.HiBlackString(time.Now().Format(l.timeFormat)),
				level.String(),
			}
			if len(ctx) > 0 {
				joinedFormat += "%s "
				joinedArgs = append(joinedArgs, color.BlueString(ctx))
			}
			joinedFormat += format
			joinedArgs = append(joinedArgs, args...)
			fmt.Fprintf(l.out, joinedFormat+"\n", joinedArgs...)
		}
	}
	if level == FatalLevel && l.exitOnFatal {
		os.Exit(1)
	}
}

func (l *Logger) println(level Level, ctx string, args ...any) {
	if level >= l.minLevel {
		if l.out != nil {
			// 15:04:05.000 DEBU [context] message
			joined := []any{
				color.HiBlackString(time.Now().Format(l.timeFormat)),
				level.String(),
			}
			if len(ctx) > 0 {
				joined = append(joined, color.BlueString(ctx))
			}
			joined = append(joined, args...)
			fmt.Fprintln(l.out, joined...)
		}
	}
	if level == FatalLevel && l.exitOnFatal {
		os.Exit(1)
	}
}

func (l *Logger) Debugf(format string, args ...any) {
	l.printf(DebugLevel, "", format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.printf(InfoLevel, "", format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.printf(WarnLevel, "", format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.printf(ErrorLevel, "", format, args...)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.printf(FatalLevel, "", format, args...)
}

func (l *Logger) Debugln(args ...any) {
	l.println(DebugLevel, "", args...)
}

func (l *Logger) Infoln(args ...any) {
	l.println(InfoLevel, "", args...)
}

func (l *Logger) Warnln(args ...any) {
	l.println(WarnLevel, "", args...)
}

func (l *Logger) Errorln(args ...any) {
	l.println(ErrorLevel, "", args...)
}

func (l *Logger) Fatalln(args ...any) {
	l.println(FatalLevel, "", args...)
}
