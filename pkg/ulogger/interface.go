package ulogger

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
	//Int8(key string, value int) Event
	//Int16(key string, value int) Event
	//Int32(key string, value int) Event
	//Int64(key string, value int) Event
	//Uint(key string, value int) Event
	//Uint8(key string, value int) Event
	//Uint16(key string, value int) Event
	//Uint32(key string, value int) Event
	//Uint64(key string, value int) Event
	//Float32(key string, value float64) Event
	//Float64(key string, value float64) Event
	//Err(err error) Event
	//Func(func(Event)) Event
	//Timestamp() Event
	//Time() Event
	//Dur() Event

	Fields(fields any) Event

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
