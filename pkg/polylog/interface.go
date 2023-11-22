package polylog

import "time"

// TODO_CONSIDERATION: this may be a good candidate package for extraction to
// its own module.

// TODO_INVESTIGATE: check whether the pkg dependency tree includes all logging
// libraries.

// Logger is an interface that exposes methods for each supported log level, each
// of which returns an Event.
type Logger interface {
	Debug() Event
	Info() Event
	Warn() Event
	Error() Event
}

// Event represents a log event. It is instanced by one of the level methods of
// Logger and finalized by the Msg, Msgf, or Send methods. It exposes methods for
// adding fields to the event which will be rendered in an encoding-appropriate
// way in the log output.
//
// TODO_IMPROVE/TODO_COMMUNITY: support #Dict(), #Stack(), #Ctx(), #Any()
// see: https://pkg.go.dev/github.com/rs/zerolog#Event
type Event interface {
	Str(key, value string) Event

	// Bool adds the field key with val as a bool to the *Event context.
	Bool(key string, value bool) Event

	// Int adds the field key with i as a int to the *Event context.
	Int(key string, value int) Event

	// Int8 adds the field key with i as a int8 to the *Event context.
	Int8(key string, value int8) Event

	// Int16 adds the field key with i as a int16 to the *Event context.
	Int16(key string, value int16) Event

	// Int32 adds the field key with i as a int32 to the *Event context.
	Int32(key string, value int32) Event

	// Int64 adds the field key with i as a int64 to the *Event context.
	Int64(key string, value int64) Event

	// Uint adds the field key with i as a uint to the *Event context.
	Uint(key string, value uint) Event

	// Uint8 adds the field key with i as a uint8 to the *Event context.
	Uint8(key string, value uint8) Event

	// Uint16 adds the field key with i as a uint16 to the *Event context.
	Uint16(key string, value uint16) Event

	// Uint32 adds the field key with i as a uint32 to the *Event context.
	Uint32(key string, value uint32) Event

	// Uint64 adds the field key with i as a uint64 to the *Event context.
	Uint64(key string, value uint64) Event

	// Float32 adds the field key with f as a float32 to the *Event context.
	Float32(key string, value float32) Event

	// Float64 adds the field key with f as a float64 to the *Event context.
	Float64(key string, value float64) Event

	// Err adds the field "error" with serialized err to the *Event context.
	// If err is nil, no field is added.
	//
	// TODO_TEST: ensure implementation tests cover this: do not add a field
	// if err is nil.
	//
	// To customize the key name, use the appropriate option when constructing
	// the respective logger.
	Err(err error) Event

	// Timestamp adds the current local time as UNIX timestamp to the *Event context
	// with the "time" key. To customize the key name, use the appropriate option
	// when constructing the respective Logger.
	//
	// NOTE: It won't dedupe the "time" key if the *Event (or *Context) has one
	// already.
	Timestamp() Event

	// Time adds the field key with t formatted as string using a configurable,
	// implementation-specific format.
	//
	// To customize the time format, use the appropriate option when constructing
	// the respective Logger.
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

	// Enabled return false if the *Event is going to be filtered out by
	// log level or sampling.
	Enabled() bool

	// Discard disables the event so Msg(f)/Send won't print it.
	Discard() Event

	// Msg sends the *Event with msg added as the message field if not empty.
	//
	// NOTICE: once this method is called, the *Event should be disposed.
	// Calling Msg twice can have unexpected result.
	Msg(message string)

	// Msgf sends the event with formatted msg added as the message field if not empty.
	//
	// NOTICE: once this method is called, the *Event should be disposed.
	// Calling Msgf twice can have unexpected result.
	Msgf(format string, v ...interface{})

	// Send is equivalent to calling Msg("").
	//
	// NOTICE: once this method is called, the *Event should be disposed.
	Send()
}
