package logger

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// This is used externally only for middleware initialization.
func GlobalLogger() zerolog.Logger {
	return log.Logger
}

func Trace(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Trace()
}

func Debug(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Debug()
}

func Info(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Info()
}

func Warn(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Warn()
}

func Error(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Error()
}

// Fatal logs the message and then calls os.Exit(1).
func Fatal(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Fatal()
}

// Panic logs the message and then calls panic().
func Panic(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Panic()
}
