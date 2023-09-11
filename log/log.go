package log

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

type logFormat int

type LogTransport struct {
	Filename string
	Level    slog.Level
}

type LogOptions struct {
	AddSource  bool
	Level      slog.Level
	LogFormat  logFormat
	TimeFormat string
	Transports []*LogTransport
}

type logInstance struct {
	slog *slog.Logger
}

const (
	labelDebug = "DEBUG"
	labelInfo  = "INFO"
	labelWarn  = "WARN"
	labelError = "ERROR"
	labelFatal = "FATAL"

	DebugLevel = slog.LevelDebug
	InfoLevel  = slog.LevelInfo
	WarnLevel  = slog.LevelWarn
	ErrorLevel = slog.LevelError
	FatalLevel = slog.Level(12)

	TextFormat logFormat = iota + 1
	JSONFormat
	PrettyFormat
)

var levelLabel = map[slog.Level]string{
	DebugLevel: labelDebug,
	InfoLevel:  labelInfo,
	WarnLevel:  labelWarn,
	ErrorLevel: labelError,
	FatalLevel: labelFatal,
}
var singleInstance *logInstance
var once sync.Once

func loadLogOptions(opts *LogOptions) *LogOptions {
	if opts == nil {
		opts = &LogOptions{}
	}

	if opts.Level == -1 {
		opts.Level = InfoLevel
	}

	if opts.LogFormat == 0 {
		opts.LogFormat = PrettyFormat
	}

	if opts.TimeFormat == "" {
		opts.TimeFormat = time.DateTime
	}

	return opts
}

func NewLog(opts *LogOptions) *logInstance {
	once.Do(func() {
		logOpts := loadLogOptions(opts)

		slogOptions := slog.HandlerOptions{
			AddSource: logOpts.AddSource,
			Level:     logOpts.Level,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if attr.Key == slog.LevelKey {
					level := attr.Value.Any().(slog.Level)
					label, ok := levelLabel[level]
					if !ok {
						label = level.String()
					}
					attr.Value = slog.StringValue(label)
				}

				return attr
			},
		}

		switch logOpts.LogFormat {
		case TextFormat:
			logHandler := slog.NewTextHandler(os.Stdout, &slogOptions)
			singleInstance = &logInstance{
				slog: slog.New(logHandler),
			}

		case JSONFormat:
			logHandler := slog.NewJSONHandler(os.Stdout, &slogOptions)
			singleInstance = &logInstance{
				slog: slog.New(logHandler),
			}

		default:
			logHandler := NewPrettyHandler(os.Stdout, &PrettyHandlerOptions{
				TimeFormat:     logOpts.TimeFormat,
				HandlerOptions: slogOptions,
			})
			singleInstance = &logInstance{
				slog: slog.New(logHandler),
			}
		}
	})

	return singleInstance
}

func (instance *logInstance) Debug(msg string, args ...any) {
	instance.slog.Debug(msg, args...)
}

func (instance *logInstance) Info(msg string, args ...any) {
	instance.slog.Info(msg, args...)
}

func (instance *logInstance) Warn(msg string, args ...any) {
	instance.slog.Warn(msg, args...)
}

func (instance *logInstance) Error(msg string, args ...any) {
	instance.slog.Error(msg, args...)
}

func (instance *logInstance) Fatal(msg string, args ...any) {
	instance.slog.Log(context.Background(), FatalLevel, msg, args...)
	os.Exit(1)
}
