package uzap

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

type zapEvent struct {
	logger *zap.Logger
	level  zapcore.Level
	fields []zapcore.Field
}

func newEvent(logger *zap.Logger, level zapcore.Level) ulogger.Event {
	return &zapEvent{
		logger: logger,
		level:  level,
	}
}

func (zae zapEvent) Str(key, value string) ulogger.Event {
	zae.fields = append(zae.fields, zap.String(key, value))
	return zae
}

func (zae zapEvent) Bool(key string, value bool) ulogger.Event {
	zae.fields = append(zae.fields, zap.Bool(key, value))
	return zae
}

func (zae zapEvent) Int(key string, value int) ulogger.Event {
	zae.fields = append(zae.fields, zap.Int(key, value))
	return zae
}

func (zae zapEvent) Err(err error) ulogger.Event {
	zae.fields = append(zae.fields, zap.Error(err))
	return zae
}

// TODO_IN_THIS_COMMIT: not like this...
//func (zae zapEvent) Fields(fields any) ulogger.Event {
//	zae.fields = append(zae.fields, zap.Any(fields))
//	return zae
//}

func (zae zapEvent) Msg(msg string) {
	zae.log(msg, zae.fields...)
}

func (zae zapEvent) Msgf(format string, args ...any) {
	zae.log(fmt.Sprintf(format, args...))
}

func (zae zapEvent) Send() {
	zae.log("", zae.fields...)
}

func (zae zapEvent) log(msg string, fields ...zapcore.Field) {
	zae.logger.Check(zae.level, msg).Write(fields...)
}
