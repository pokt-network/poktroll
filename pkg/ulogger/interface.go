package ulogger

import "time"

// TODO_CONSIDERATION: this may be a good candidate package for extraction to
// its own module.

// TODO_INVESTIGATE: check whether the pkg dependency tree includes all logging
// libraries.

type UniversalLogger interface {
	Debug() Event
	Info() Event
	Warn() Event
	Error() Event
}

type Event interface {
	Str(key, value string) Event
	Bool(key string, value bool) Event
	Int(key string, value int) Event
	Int8(key string, value int8) Event
	Int16(key string, value int16) Event
	Int32(key string, value int32) Event
	Int64(key string, value int64) Event
	Uint(key string, value uint) Event
	Uint8(key string, value uint8) Event
	Uint16(key string, value uint16) Event
	Uint32(key string, value uint32) Event
	Uint64(key string, value uint64) Event
	Float32(key string, value float32) Event
	Float64(key string, value float64) Event
	Err(err error) Event
	Timestamp() Event
	Time(key string, value time.Time) Event
	Dur(key string, value time.Duration) Event

	//Fields(fields any) Event
	//Func(func(Event)) Event

	// Enabled return false if the *Event is going to be filtered out by log
	// level or sampling.
	//Enabled() bool
	// Discard disables the event so it won't print
	//Discard() Event
	// Fields ... fields is expected to either be a map or a slice.

	Msg(message string)
	Msgf(format string, v ...interface{})
	Send()
}
