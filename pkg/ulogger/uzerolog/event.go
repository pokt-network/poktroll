package uzerolog

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

var _ ulogger.Event = (*zerologEvent)(nil)

type zerologEvent struct {
	event *zerolog.Event
}

func newEvent(event *zerolog.Event) ulogger.Event {
	return zerologEvent{
		event: event,
	}
}

func (zle zerologEvent) Str(key, value string) ulogger.Event {
	zle.event.Str(key, value)
	return zle
}

func (zle zerologEvent) Bool(key string, value bool) ulogger.Event {
	zle.event.Bool(key, value)
	return zle
}

func (zle zerologEvent) Int(key string, value int) ulogger.Event {
	zle.event.Int(key, value)
	return zle
}

func (zle zerologEvent) Int8(key string, value int8) ulogger.Event {
	zle.event.Int8(key, value)
	return zle
}

func (zle zerologEvent) Int16(key string, value int16) ulogger.Event {
	zle.event.Int16(key, value)
	return zle
}

func (zle zerologEvent) Int32(key string, value int32) ulogger.Event {
	zle.event.Int32(key, value)
	return zle
}

func (zle zerologEvent) Int64(key string, value int64) ulogger.Event {
	zle.event.Int64(key, value)
	return zle
}

func (zle zerologEvent) Uint(key string, value uint) ulogger.Event {
	zle.event.Uint(key, value)
	return zle
}

func (zle zerologEvent) Uint8(key string, value uint8) ulogger.Event {
	zle.event.Uint8(key, value)
	return zle
}

func (zle zerologEvent) Uint16(key string, value uint16) ulogger.Event {
	zle.event.Uint16(key, value)
	return zle
}

func (zle zerologEvent) Uint32(key string, value uint32) ulogger.Event {
	zle.event.Uint32(key, value)
	return zle
}

func (zle zerologEvent) Uint64(key string, value uint64) ulogger.Event {
	zle.event.Uint64(key, value)
	return zle
}

func (zle zerologEvent) Float32(key string, value float32) ulogger.Event {
	zle.event.Float32(key, value)
	return zle
}

func (zle zerologEvent) Float64(key string, value float64) ulogger.Event {
	zle.event.Float64(key, value)
	return zle
}

func (zle zerologEvent) Err(err error) ulogger.Event {
	zle.event.Err(err)
	return zle
}

func (zle zerologEvent) Timestamp() ulogger.Event {
	// TODO_IMPROVE: the key for this can be configured by changing the value of
	// the zerolog.TimestampFieldName variable.
	zle.event.Timestamp()
	return zle
}

func (zle zerologEvent) Time(key string, value time.Time) ulogger.Event {
	zle.event.Time(key, value)
	return zle
}

func (zle zerologEvent) Dur(key string, value time.Duration) ulogger.Event {
	zle.event.Dur(key, value)
	return zle
}

func (zle zerologEvent) Fields(fields any) ulogger.Event {
	zle.event.Fields(fields)
	return zle
}

func (zle zerologEvent) Msg(msg string) {
	zle.event.Msg(msg)
}

func (zle zerologEvent) Msgf(format string, args ...interface{}) {
	zle.event.Msgf(format, args...)
}

func (zle zerologEvent) Send() {
	zle.event.Send()
}
