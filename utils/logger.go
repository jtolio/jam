package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type LogLevel int

var (
	Debug  LogLevel = 1
	Normal LogLevel = 2
	Urgent LogLevel = 3
	None   LogLevel = 4
)

func (l LogLevel) String() string {
	switch l {
	case Debug:
		return "debug"
	case Normal:
		return "normal"
	case Urgent:
		return "urgent"
	case None:
		return "none"
	default:
		return fmt.Sprintf("unknown(%v)", l)
	}
}

func ParseLogLevel(val string) (LogLevel, error) {
	switch strings.ToLower(val) {
	case "debug":
		return Debug, nil
	case "normal":
		return Normal, nil
	case "urgent":
		return Urgent, nil
	case "none":
		return None, nil
	default:
		return None, fmt.Errorf("unknown log level: %q", val)
	}
}

type Logger interface {
	Printf(level LogLevel, format string, v ...interface{})
}

type LoggerFunc func(level LogLevel, format string, v ...interface{})

func (f LoggerFunc) Printf(level LogLevel, format string, v ...interface{}) {
	f(level, format, v...)
}

func StandardLogger(minLevel LogLevel) Logger {
	if minLevel >= None {
		return LoggerFunc(func(level LogLevel, format string, v ...interface{}) {})
	}
	return LoggerFunc(func(level LogLevel, format string, v ...interface{}) {
		if level >= minLevel {
			fmt.Fprintf(os.Stderr,
				strings.TrimRightFunc(format, unicode.IsSpace)+"\n",
				v...)
		}
	})
}

type ctxKey int

var (
	loggerKey ctxKey = 1
)

func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func L(ctx context.Context) LogWrapper {
	if logger, ok := ctx.Value(loggerKey).(Logger); ok {
		return LogWrapper{Logger: logger}
	}
	return LogWrapper{Logger: StandardLogger(None)}
}

type LogWrapper struct {
	Logger
}

func (l LogWrapper) Debugf(format string, v ...interface{}) {
	l.Printf(Debug, format, v...)
}

func (l LogWrapper) Normalf(format string, v ...interface{}) {
	l.Printf(Normal, format, v...)
}

func (l LogWrapper) Urgentf(format string, v ...interface{}) {
	l.Printf(Urgent, format, v...)
}
