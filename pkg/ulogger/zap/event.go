package zap

import (
	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

type zerologEvent struct {
	event *zerolog.Event
}

func newEvent(event *zerolog.Event) ulogger.Event {
	//return zerologEvent{
	//	event: event,
	//}
	return nil
}

func (zae zerologEvent) Str(key, value string) ulogger.Event {
	//zae.zae.Str(key, value)
	//return zae
	return nil
}

func (zae zerologEvent) Int(key string, value int) ulogger.Event {
	//zae.zae.Int(key, value)
	//return zae
	return nil
}
