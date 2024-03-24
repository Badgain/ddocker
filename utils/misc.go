package utils

import (
	"context"
	"log/slog"
)

var (
	loggerKey struct{}
)

func Exists[T comparable](arr []T, val T) bool {
	var found bool

	for _, v := range arr {
		if v == val {
			found = true
			break
		}
	}

	return found
}

func ContextWithLogger(ctx context.Context, lg *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, lg)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if val := ctx.Value(loggerKey); val != nil {
		lg, ok := val.(*slog.Logger)
		if !ok {
			return slog.Default()
		}
		return lg
	}
	return slog.Default()
}
