package log

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/common"
)

type logFormat int

type LogOptions struct {
	AddSource  bool
	Level      slog.Level
	LogFormat  logFormat
	TimeFormat string
	MaskFields []string
}

type logInstance struct {
	slog *slog.Logger
}

const (
	DebugLevel = slog.LevelDebug
	InfoLevel  = slog.LevelInfo
	WarnLevel  = slog.LevelWarn
	ErrorLevel = slog.LevelError
	FatalLevel = slog.Level(12)
)

const (
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
var singleInstance common.Logger
var once sync.Once

func loadLogOptions(opts *LogOptions) *LogOptions {
	if opts == nil {
		opts = &LogOptions{}
	}

	if opts.LogFormat == 0 {
		opts.LogFormat = PrettyFormat
	}

	if opts.TimeFormat == "" {
		opts.TimeFormat = time.DateTime
	}

	return opts
}

func NewLog(opts *LogOptions) common.Logger {
	once.Do(func() {
		logOpts := loadLogOptions(opts)

		slogOptions := slog.HandlerOptions{
			AddSource:   logOpts.AddSource,
			Level:       logOpts.Level,
			ReplaceAttr: replaceLevelLabel,
		}

		var handler slog.Handler
		switch logOpts.LogFormat {
		case TextFormat:
			handler = slog.NewTextHandler(os.Stdout, &slogOptions)
		case JSONFormat:
			handler = slog.NewJSONHandler(os.Stdout, &slogOptions)
		default:
			handler = newPrettyHandler(os.Stdout, &prettyHandlerOptions{
				TimeFormat:     logOpts.TimeFormat,
				HandlerOptions: slogOptions,
			})
		}

		base := &logInstance{slog: slog.New(handler)}
		singleInstance = WrapLogger(base, logOpts.MaskFields)
	})

	return singleInstance
}

func replaceLevelLabel(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) > 0 || attr.Key != slog.LevelKey {
		return attr
	}
	level, ok := attr.Value.Any().(slog.Level)
	if !ok {
		return attr
	}
	label, ok := levelLabel[level]
	if !ok {
		label = level.String()
	}
	attr.Value = slog.StringValue(label)
	return attr
}

func (instance *logInstance) enabled(level slog.Level) bool {
	return instance.slog.Enabled(context.Background(), level)
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
