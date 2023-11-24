package polylog

import (
	"context"
	"time"
)

// TODO_CONSIDERATION: this may be a good candidate package for extraction to
// its own module.

// TODO_INVESTIGATE: check whether the pkg dependency tree includes all logging
// libraries.

// Logger is an interface that exposes methods for each supported log level, each
// of which returns an Event.
type Logger interface {
	// Debug starts a new log message (event) with debug level.
	//
	// You must call Msg on the returned event in order to send the event.
	Debug() Event

	// Info starts a new log message (event) with info level.
	//
	// You must call Msg on the returned event in order to send the event.
	Info() Event

	// Warn starts a new log message (event) with warn level.
	//
	// You must call Msg on the returned event in order to send the event.
	Warn() Event

	// Error starts a new log message (event) with error level.
	//
	// You must call Msg on the returned event in order to send the event.
	Error() Event

	// With creates a child logger with the fields constructed from keyVals added
	// to its context.
	With(keyVals ...any) Logger

	// WithContext returns a copy of ctx with the receiver attached. The Logger
	// attached to the provided Context (if any) will not be effected.  If the
	// receiver's log level is Disabled it will only be attached to the returned
	// Context if the provided Context has a previously attached Logger. If the
	// provided Context has no attached Logger, a Disabled Logger will not be
	// attached.
	//
	// TODO_IMPROVE/TODO_COMMUNITY: support #UpdateContext() and  update this
	// godoc to include #UpdateContext() usage example.
	//
	// See: https://pkg.go.dev/github.com/rs/zerolog#Logger.UpdateContext
	WithContext(ctx context.Context) context.Context

	// WithLevel starts a new message (event) with level.
	//
	// You must call Msg on the returned event in order to send the event.
	WithLevel(level int) Event

	// Write implements the io.Writer interface. This is useful to set as a writer
	// for the standard library log.
	Write(p []byte) (n int, err error)
}

// Event represents a log event. It is instanced by one of the level methods of
// Logger and finalized by the Msg, Msgf, or Send methods. It exposes methods for
// adding fields to the event which will be rendered in an encoding-appropriate
// way in the log output.
//
// TODO_IMPROVE/TODO_COMMUNITY: support #Dict(), #Stack(), #Any() fields.
// See: https://pkg.go.dev/github.com/rs/zerolog#Event
//
// TODO_IMPROVE/TODO_COMMUNITY: support #UpdateContext() and  update #Ctx() godoc.
// See: https://pkg.go.dev/github.com/rs/zerolog#Logger.UpdateContext
type Event interface {
	// Str adds the field key with value as a string to the Event context.
	Str(key, value string) Event

	// Bool adds the field key with value as a bool to the Event context.
	Bool(key string, value bool) Event

	// Int adds the field key with value as an int to the Event context.
	Int(key string, value int) Event

	// Int8 adds the field key with value as an int8 to the Event context.
	Int8(key string, value int8) Event

	// Int16 adds the field key with value as an int16 to the Event context.
	Int16(key string, value int16) Event

	// Int32 adds the field key with value as an int32 to the Event context.
	Int32(key string, value int32) Event

	// Int64 adds the field key with value as an int64 to the Event context.
	Int64(key string, value int64) Event

	// Uint adds the field key with value as a uint to the Event context.
	Uint(key string, value uint) Event

	// Uint8 adds the field key with value as a uint8 to the Event context.
	Uint8(key string, value uint8) Event

	// Uint16 adds the field key with value as a uint16 to the Event context.
	Uint16(key string, value uint16) Event

	// Uint32 adds the field key with value as a uint32 to the Event context.
	Uint32(key string, value uint32) Event

	// Uint64 adds the field key with value as a uint64 to the Event context.
	Uint64(key string, value uint64) Event

	// Float32 adds the field key with value as a float32 to the Event context.
	Float32(key string, value float32) Event

	// Float64 adds the field key with value as a float64 to the Event context.
	Float64(key string, value float64) Event

	// Err adds the field "error" with serialized err to the Event context.
	// If err is nil, no field is added.
	//
	// TODO_TEST: ensure implementation tests cover this: do not add a field
	// if err is nil.
	//
	// To customize the key name, use the appropriate option from the respective
	// package when constructing a logger.
	//
	// TODO_UPNEXT(@bryanchriswhite): ensure implementations' godoc examples cover
	// options.
	Err(err error) Event

	// Timestamp adds the current local time as UNIX timestamp to the Event context
	// with the "time" key. To customize the key name, use the appropriate option
	// from the respective package when constructing a Logger.
	//
	// TODO_UPNEXT(@bryanchriswhite): ensure implementations' godoc examples cover
	// options.
	//
	// NOTE: It won't dedupe the "time" key if the Event (or *Context) has one
	// already.
	Timestamp() Event

	// Time adds the field key with value formatted as string using a configurable,
	// implementation-specific format.
	//
	// To customize the time format, use the appropriate option from the respective
	// package when constructing a Logger.
	Time(key string, value time.Time) Event

	// Dur adds the field key with duration with a configurable,
	// implementation-specific value format.
	Dur(key string, value time.Duration) Event

	// Fields is a helper function to use a map or slice to set fields using type assertion.
	// Only map[string]interface{} and []interface{} are accepted. []interface{} must
	// alternate string keys and arbitrary values, and extraneous ones are ignored.
	Fields(fields any) Event

	// Func allows an anonymous func to run only if the event is enabled.
	Func(func(Event)) Event

	// Enabled return false if the Event is going to be filtered out by
	// log level or sampling.
	Enabled() bool

	// Discard disables the event so Msg(f)/Send won't print it.
	Discard() Event

	// Msg sends the Event with msg added as the message field if not empty.
	//
	// NOTICE: once this method is called, the Event should be disposed.
	// Calling Msg twice can have unexpected result.
	Msg(message string)

	// Msgf sends the event with formatted msg added as the message field if not empty.
	//
	// NOTICE: once this method is called, the Event should be disposed.
	// Calling Msgf twice can have unexpected result.
	Msgf(format string, keyVals ...interface{})

	// Send is equivalent to calling Msg("").
	//
	// NOTICE: once this method is called, the Event should be disposed.
	Send()
}
