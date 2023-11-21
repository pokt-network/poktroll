package ulog

import (
	"fmt"
	"log"
	"strings"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

const (
	errorFieldKey  = "error"
	fieldsFieldKey = "fields"
)

var _ ulogger.Event = (*stdLogEvent)(nil)

type stdLogEvent struct {
	levelString string
	fields      stdLogFields
}

type stdLogFields map[string]any

func newEvent(level ulogger.Level) ulogger.Event {
	return &stdLogEvent{
		levelString: getLevelString(level),
		fields:      make(stdLogFields),
	}
}

func (st *stdLogEvent) Str(key, value string) ulogger.Event {
	st.fields[key] = value
	return st
}

func (st *stdLogEvent) Bool(key string, value bool) ulogger.Event {
	st.fields[key] = value
	return st
}

func (st *stdLogEvent) Int(key string, value int) ulogger.Event {
	st.fields[key] = value
	return st
}

func (st *stdLogEvent) Err(err error) ulogger.Event {
	st.fields[errorFieldKey] = err
	return st
}

func (st *stdLogEvent) Fields(fields any) ulogger.Event {
	st.fields[fieldsFieldKey] = fields
	return st
}

func (st *stdLogEvent) Msg(msg string) {
	log.Println(st.levelString, st.fields.String(), msg)
}

func (st *stdLogEvent) Msgf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Println(st.levelString, st.fields.String(), msg)
}

func (st *stdLogEvent) Send() {
	log.Println(st.levelString, st.fields.String())
}

func (stf stdLogFields) String() string {
	var fieldLines []string
	for key, value := range stf {
		line := fmt.Sprintf("%s=%v", key, value)
		fieldLines = append(fieldLines, line)
	}
	return strings.Join(fieldLines, " ")
}

func getLevelString(level ulogger.Level) string {
	switch level {
	case ulogger.LevelDebug:
		return "[DEBUG]"
	case ulogger.LevelInfo:
		return "[INFO]"
	case ulogger.LevelWarn:
		return "[WARN]"
	case ulogger.LevelError:
		return "[ERROR]"
	default:
		return "[UNKNOWN]"
	}
}
