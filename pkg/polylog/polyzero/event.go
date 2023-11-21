package polyzero

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ polylog.Event = (*zerologEvent)(nil)

type zerologEvent struct {
	event *zerolog.Event
}

func newEvent(event *zerolog.Event) polylog.Event {
	return &zerologEvent{
		event: event,
	}
}

func (zle *zerologEvent) Str(key, value string) polylog.Event {
	zle.event.Str(key, value)
	return zle
}

func (zle *zerologEvent) Bool(key string, value bool) polylog.Event {
	zle.event.Bool(key, value)
	return zle
}

func (zle *zerologEvent) Int(key string, value int) polylog.Event {
	zle.event.Int(key, value)
	return zle
}

func (zle *zerologEvent) Int8(key string, value int8) polylog.Event {
	zle.event.Int8(key, value)
	return zle
}

func (zle *zerologEvent) Int16(key string, value int16) polylog.Event {
	zle.event.Int16(key, value)
	return zle
}

func (zle *zerologEvent) Int32(key string, value int32) polylog.Event {
	zle.event.Int32(key, value)
	return zle
}

func (zle *zerologEvent) Int64(key string, value int64) polylog.Event {
	zle.event.Int64(key, value)
	return zle
}

func (zle *zerologEvent) Uint(key string, value uint) polylog.Event {
	zle.event.Uint(key, value)
	return zle
}

func (zle *zerologEvent) Uint8(key string, value uint8) polylog.Event {
	zle.event.Uint8(key, value)
	return zle
}

func (zle *zerologEvent) Uint16(key string, value uint16) polylog.Event {
	zle.event.Uint16(key, value)
	return zle
}

func (zle *zerologEvent) Uint32(key string, value uint32) polylog.Event {
	zle.event.Uint32(key, value)
	return zle
}

func (zle *zerologEvent) Uint64(key string, value uint64) polylog.Event {
	zle.event.Uint64(key, value)
	return zle
}

func (zle *zerologEvent) Float32(key string, value float32) polylog.Event {
	zle.event.Float32(key, value)
	return zle
}

func (zle *zerologEvent) Float64(key string, value float64) polylog.Event {
	zle.event.Float64(key, value)
	return zle
}

func (zle *zerologEvent) Err(err error) polylog.Event {
	zle.event.Err(err)
	return zle
}

func (zle *zerologEvent) Timestamp() polylog.Event {
	// TODO_IMPROVE: the key for this can be configured by changing the value of
	// the zerolog.TimestampFieldName variable.
	zle.event.Timestamp()
	return zle
}

func (zle *zerologEvent) Time(key string, value time.Time) polylog.Event {
	zle.event.Time(key, value)
	return zle
}

func (zle *zerologEvent) Dur(key string, value time.Duration) polylog.Event {
	zle.event.Dur(key, value)
	return zle
}

func (zle *zerologEvent) Func(fn func(polylog.Event)) polylog.Event {
	zle.event = zle.event.Func(
		func(event *zerolog.Event) {
			fn(newEvent(event))
		},
	)
	return zle
}

func (zle *zerologEvent) Fields(fields any) polylog.Event {
	zle.event.Fields(fields)
	return zle
}

func (zle *zerologEvent) Enabled() bool {
	return zle.event.Enabled()
}

func (zle *zerologEvent) Discard() polylog.Event {
	zle.event.Discard()
	return zle
}

func (zle *zerologEvent) Msg(msg string) {
	zle.event.Msg(msg)
}

func (zle *zerologEvent) Msgf(format string, args ...interface{}) {
	zle.event.Msgf(format, args...)
}

func (zle *zerologEvent) Send() {
	zle.event.Send()
}
