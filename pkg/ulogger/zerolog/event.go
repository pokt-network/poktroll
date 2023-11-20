package zerolog

import (
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

func (zle zerologEvent) Err(err error) ulogger.Event {
	zle.event.Err(err)
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
