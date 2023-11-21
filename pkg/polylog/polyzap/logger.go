package polyzap

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ polylog.Logger = (*zapLogger)(nil)

type zapLogger struct {
	// NB: Default (0) is Info.
	level         zapcore.Level
	writeSyncer   zapcore.WriteSyncer
	encoderConfig zapcore.EncoderConfig
	encoder       zapcore.Encoder
	core          zapcore.Core
	logger        *zap.Logger
}

func NewLogger(
	opts ...polylog.LoggerOption,
) polylog.Logger {
	ze := &zapLogger{}

	for _, opt := range opts {
		opt(ze)
	}

	ze.buildLoggerAndSetDefaults()

	return ze
}

func (za *zapLogger) Debug() polylog.Event {
	return newEvent(za.logger, zapcore.DebugLevel)
}

func (za *zapLogger) Info() polylog.Event {
	return newEvent(za.logger, zapcore.InfoLevel)
}

func (za *zapLogger) Warn() polylog.Event {
	return newEvent(za.logger, zapcore.WarnLevel)
}

func (za *zapLogger) Error() polylog.Event {
	return newEvent(za.logger, zapcore.ErrorLevel)
}

func (za *zapLogger) With(keyVals ...any) polylog.Logger {
	var (
		fields  []zap.Field
		nextKey any
	)
	for keyValIdx, keyVal := range keyVals {
		if keyValIdx%2 == 0 {
			nextKey = keyVal
			continue
		}
		nextKeyStr := fmt.Sprintf("%s", nextKey)
		fields = append(fields, zap.Any(nextKeyStr, keyVal))
	}

	return &zapLogger{
		level:  za.level,
		logger: za.logger.With(fields...),
	}
}

// WithContext returns a copy of ctx with the receiver attached. The Logger
// attached to the provided Context (if any) will not be effected.  If the
// receiver's log level is Disabled it will only be attached to the returned
// Context if the provided Context has a previously attached Logger. If the
// provided Context has no attached Logger, a Disabled Logger will not be
// attached.
//
// TODO_TECHDEBT/TODO_COMMUNITY: implement with behavior analogous to that
// of `polyzero.Logger`'s.
//
// TODO_IMPROVE/TODO_COMMUNITY: support #UpdateContext() and  update this
// godoc to include #UpdateContext() usage example.
// See: https://pkg.go.dev/github.com/rs/zerolog#Logger.UpdateContext.
func (za *zapLogger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, polylog.CtxKey, za)
}

func (za *zapLogger) WithLevel(level polylog.Level) polylog.Event {
	// TODO_IN_THIS_COMMIT: consider comparing level.String() & zapcore.Level.String().
	return newEvent(za.logger, zapcore.Level(level.Int()))
}

func (za *zapLogger) Write(p []byte) (n int, err error) {
	za.logger.Log(
		za.level,
		string(p),
	)
	return len(p), nil
}

func (za *zapLogger) buildLoggerAndSetDefaults() {
	if za.writeSyncer == nil {
		za.writeSyncer = zapcore.AddSync(os.Stderr)
	}

	if za.logger == nil {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoder := zapcore.NewJSONEncoder(encoderConfig)
		core := zapcore.NewCore(encoder, za.writeSyncer, za.level)

		za.logger = zap.New(core)
	}

}
