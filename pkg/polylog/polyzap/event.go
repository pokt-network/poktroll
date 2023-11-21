package polyzap

import (
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ polylog.Event = (*zapEvent)(nil)

type zapEvent struct {
	logger    *zap.Logger
	level     zapcore.Level
	fields    []zapcore.Field
	discarded atomic.Bool
}

func newEvent(logger *zap.Logger, level zapcore.Level) polylog.Event {
	discarded := atomic.Bool{}
	if level < logger.Level() {
		discarded.Store(true)
	}

	return &zapEvent{
		logger:    logger,
		level:     level,
		discarded: discarded,
	}
}

func (zae *zapEvent) Str(key, value string) polylog.Event {
	zae.fields = append(zae.fields, zap.String(key, value))
	return zae
}

func (zae *zapEvent) Bool(key string, value bool) polylog.Event {
	zae.fields = append(zae.fields, zap.Bool(key, value))
	return zae
}

func (zae *zapEvent) Int(key string, value int) polylog.Event {
	zae.fields = append(zae.fields, zap.Int(key, value))
	return zae
}

func (zae *zapEvent) Int8(key string, value int8) polylog.Event {
	zae.fields = append(zae.fields, zap.Int8(key, value))
	return zae
}

func (zae *zapEvent) Int16(key string, value int16) polylog.Event {
	zae.fields = append(zae.fields, zap.Int16(key, value))
	return zae
}

func (zae *zapEvent) Int32(key string, value int32) polylog.Event {
	zae.fields = append(zae.fields, zap.Int32(key, value))
	return zae
}

func (zae *zapEvent) Int64(key string, value int64) polylog.Event {
	zae.fields = append(zae.fields, zap.Int64(key, value))
	return zae
}

func (zae *zapEvent) Uint(key string, value uint) polylog.Event {
	zae.fields = append(zae.fields, zap.Uint(key, value))
	return zae
}

func (zae *zapEvent) Uint8(key string, value uint8) polylog.Event {
	zae.fields = append(zae.fields, zap.Uint8(key, value))
	return zae
}

func (zae *zapEvent) Uint16(key string, value uint16) polylog.Event {
	zae.fields = append(zae.fields, zap.Uint16(key, value))
	return zae
}

func (zae *zapEvent) Uint32(key string, value uint32) polylog.Event {
	zae.fields = append(zae.fields, zap.Uint32(key, value))
	return zae
}

func (zae *zapEvent) Uint64(key string, value uint64) polylog.Event {
	zae.fields = append(zae.fields, zap.Uint64(key, value))
	return zae
}

func (zae *zapEvent) Float32(key string, value float32) polylog.Event {
	zae.fields = append(zae.fields, zap.Float32(key, value))
	return zae
}

func (zae *zapEvent) Float64(key string, value float64) polylog.Event {
	zae.fields = append(zae.fields, zap.Float64(key, value))
	return zae
}

func (zae *zapEvent) Err(err error) polylog.Event {
	zae.fields = append(zae.fields, zap.Error(err))
	return zae
}

func (zae *zapEvent) Timestamp() polylog.Event {
	// TODO_IMPROVE: the key should be configurable via an option.
	zae.fields = append(zae.fields, zap.Time("timestamp", time.Now()))
	return zae
}

func (zae *zapEvent) Time(key string, value time.Time) polylog.Event {
	zae.fields = append(zae.fields, zap.Time(key, value))
	return zae
}

func (zae *zapEvent) Dur(key string, value time.Duration) polylog.Event {
	zae.fields = append(zae.fields, zap.Duration(key, value))
	return zae
}

func (zae *zapEvent) Func(fn func(polylog.Event)) polylog.Event {
	if zae.Enabled() {
		fn(zae)
	}
	return zae
}

// TODO_IN_THIS_COMMIT: not like this...
func (zae *zapEvent) Fields(fields any) polylog.Event {
	// TODO_IMPROVE/TODO_INVESTIGATE: look into whether zapcore.ArrayMarshaler is
	// applicable and useful here.
	switch fieldsVal := fields.(type) {
	case map[string]any:
		for key, value := range fieldsVal {
			zae.fields = append(zae.fields, zap.Any(key, value))
		}
	case []any:
		var nextFieldKey string
		for fieldIdx, value := range fieldsVal {
			if fieldIdx%2 == 0 {
				nextFieldKey = fmt.Sprintf("%v", value)
				continue
			}

			zae.fields = append(zae.fields, zap.Any(nextFieldKey, value))
		}
	}
	return zae
}

func (zae *zapEvent) Enabled() bool {
	return !zae.discarded.Load()
}

func (zae *zapEvent) Discard() polylog.Event {
	// Set zae.discarded to true (only if not already).
	zae.discarded.CompareAndSwap(false, true)
	return zae
}

func (zae *zapEvent) Msg(msg string) {
	if !zae.Enabled() {
		return
	}

	zae.log(msg, zae.fields...)
}

func (zae *zapEvent) Msgf(format string, args ...any) {
	if !zae.Enabled() {
		return
	}

	zae.log(fmt.Sprintf(format, args...))
}

func (zae *zapEvent) Send() {
	if !zae.Enabled() {
		return
	}

	zae.log("", zae.fields...)
}

func (zae *zapEvent) log(msg string, fields ...zapcore.Field) {
	zae.logger.Check(zae.level, msg).Write(fields...)
}
