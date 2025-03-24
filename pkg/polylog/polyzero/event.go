package polyzero

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/pokt-network/pocket/pkg/polylog"
)

var _ polylog.Event = (*zerologEvent)(nil)

// zerologEvent is a thin wrapper around a zerolog event.
type zerologEvent struct {
	event *zerolog.Event
}

// Str adds the field key with value as a string to the Event context.
func (zle *zerologEvent) Str(key, value string) polylog.Event {
	zle.event.Str(key, value)
	return zle
}

// Bool adds the field key with value as a bool to the Event context.
func (zle *zerologEvent) Bool(key string, value bool) polylog.Event {
	zle.event.Bool(key, value)
	return zle
}

// Int adds the field key with value as an int to the Event context.
func (zle *zerologEvent) Int(key string, value int) polylog.Event {
	zle.event.Int(key, value)
	return zle
}

// Int8 adds the field key with value as an int8 to the Event context.
func (zle *zerologEvent) Int8(key string, value int8) polylog.Event {
	zle.event.Int8(key, value)
	return zle
}

// Int16 adds the field key with value as an int16 to the Event context.
func (zle *zerologEvent) Int16(key string, value int16) polylog.Event {
	zle.event.Int16(key, value)
	return zle
}

// Int32 adds the field key with value as an int32 to the Event context.
func (zle *zerologEvent) Int32(key string, value int32) polylog.Event {
	zle.event.Int32(key, value)
	return zle
}

// Int64 adds the field key with value as an int64 to the Event context.
func (zle *zerologEvent) Int64(key string, value int64) polylog.Event {
	zle.event.Int64(key, value)
	return zle
}

// Uint adds the field key with value as an uint to the Event context.
func (zle *zerologEvent) Uint(key string, value uint) polylog.Event {
	zle.event.Uint(key, value)
	return zle
}

// Uint8 adds the field key with value as an uint8 to the Event context.
func (zle *zerologEvent) Uint8(key string, value uint8) polylog.Event {
	zle.event.Uint8(key, value)
	return zle
}

// Uint16 adds the field key with value as an uint16 to the Event context.
func (zle *zerologEvent) Uint16(key string, value uint16) polylog.Event {
	zle.event.Uint16(key, value)
	return zle
}

// Uint32 adds the field key with value as an uint32 to the Event context.
func (zle *zerologEvent) Uint32(key string, value uint32) polylog.Event {
	zle.event.Uint32(key, value)
	return zle
}

// Uint64 adds the field key with value as an uint64 to the Event context.
func (zle *zerologEvent) Uint64(key string, value uint64) polylog.Event {
	zle.event.Uint64(key, value)
	return zle
}

// Float32 adds the field key with value as a float32 to the Event context.
func (zle *zerologEvent) Float32(key string, value float32) polylog.Event {
	zle.event.Float32(key, value)
	return zle
}

// Float64 adds the field key with value as a float64 to the Event context.
func (zle *zerologEvent) Float64(key string, value float64) polylog.Event {
	zle.event.Float64(key, value)
	return zle
}

// Err adds the field "error" with serialized err to the Event context.
// If err is nil, no field is added.
//
// To customize the key name, change zerolog.ErrorFieldName. This can be done
// directly or by using the WithErrKey() option when constructing the logger.
//
// If Stack() has been called before and zerolog.ErrorStackMarshaler is defined,
// the err is passed to ErrorStackMarshaler and the result is appended to the
// zerolog.ErrorStackFieldName.
func (zle *zerologEvent) Err(err error) polylog.Event {
	zle.event.Err(err)
	return zle
}

// Timestamp adds the current local time as UNIX timestamp to the Event context
// with the "time" key.
// To customize the key name, change zerolog.TimestampFieldName. This can be done directly or via
// the WithTimestampKey() option when constructing the logger.
//
// NOTE: It won't dedupe the "time" key if the Event (or *Context) has one
// already.
func (zle *zerologEvent) Timestamp() polylog.Event {
	zle.event.Timestamp()
	return zle
}

// Time adds the field key with value formatted as string using zerolog.TimeFieldFormat.
func (zle *zerologEvent) Time(key string, value time.Time) polylog.Event {
	zle.event.Time(key, value)
	return zle
}

// Dur adds the field key with duration value stored as zerolog.DurationFieldUnit.
// If zerolog.DurationFieldInteger is true, durations are rendered as integer
// instead of float.
func (zle *zerologEvent) Dur(key string, value time.Duration) polylog.Event {
	zle.event.Dur(key, value)
	return zle
}

// Func allows an anonymous func to run only if the event is enabled.
func (zle *zerologEvent) Func(fn func(polylog.Event)) polylog.Event {
	// NB: no need to call #Enabled() here because the underlying zerolog.Event
	// will do that for us.
	if zle.Enabled() {
		zle.event.Func(
			func(event *zerolog.Event) {
				fn(newEvent(event))
			},
		)
	}
	return zle
}

// Fields is a helper function to use a map or slice to set fields using type
// assertion. Only map[string]any and []any are accepted. any must alternate
// string keys and arbitrary values, and extraneous ones are ignored.
func (zle *zerologEvent) Fields(fields any) polylog.Event {
	zle.event.Fields(fields)
	return zle
}

// Enabled return false if the Event is going to be filtered out by
// log level or sampling.
func (zle *zerologEvent) Enabled() bool {
	return zle.event.Enabled()
}

// Discard disables the event so Msg(f)/Send won't print it.
func (zle *zerologEvent) Discard() polylog.Event {
	zle.event.Discard()
	return zle
}

// Msg sends the Event with msg added as the message field if not empty.
//
// NOTICE: once this method is called, the Event should be disposed.
// Calling Msg twice can have unexpected result.
func (zle *zerologEvent) Msg(msg string) {
	zle.event.Msg(msg)
}

// Msgf sends the event with formatted msg added as the message field if not empty.
//
// NOTICE: once this method is called, the Event should be disposed.
// Calling Msgf twice can have unexpected result.
func (zle *zerologEvent) Msgf(format string, args ...any) {
	zle.event.Msgf(format, args...)
}

// Send is equivalent to calling Msg(""). It can be thought of as a Flush.
//
// NOTICE: once this method is called, the Event should be disposed.
func (zle *zerologEvent) Send() {
	zle.event.Send()
}

// newEvent takes a zerolog event pointer and wraps it in a polylog.zerologEvent
// struct.
func newEvent(event *zerolog.Event) polylog.Event {
	return &zerologEvent{
		event: event,
	}
}
