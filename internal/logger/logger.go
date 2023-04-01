package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

const (
	debugLvl = "debug"
	infoLvl  = "info"
	warnLvl  = "warn"
	errorLvl = "error"
)

type Logger interface {
	Debug(msg string, msgArgs ...any)
	Info(msg string, msgArgs ...any)
	Warn(msg string)
	Error(msg string, msgArgs ...any)
}

// ZLBasedLogger - 'Zerolog' based implementation of Logger interface.
type ZLBasedLogger struct {
	logger *zerolog.Logger
}

func NewLogger(lvl string) *ZLBasedLogger {
	var globalLvl zerolog.Level

	switch strings.ToLower(lvl) {
	case errorLvl:
		globalLvl = zerolog.ErrorLevel
	case warnLvl:
		globalLvl = zerolog.WarnLevel
	case infoLvl:
		globalLvl = zerolog.InfoLevel
	case debugLvl:
		globalLvl = zerolog.DebugLevel
	default:
		globalLvl = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(globalLvl)
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		CallerWithSkipFrameCount(3).
		Logger()

	return &ZLBasedLogger{
		logger: &logger,
	}
}

func (l *ZLBasedLogger) Debug(msg string, msgArgs ...any) {
	l.logger.Debug().Msgf(msg, msgArgs...)
}

func (l *ZLBasedLogger) Info(msg string, msgArgs ...any) {
	l.logger.Info().Msgf(msg, msgArgs...)
}

func (l *ZLBasedLogger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

func (l *ZLBasedLogger) Error(msg string, msgArgs ...any) {
	l.logger.Error().Msgf(msg, msgArgs...)
}
