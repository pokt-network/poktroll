package polystd

import (
	"fmt"
	"log"
	"strings"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	errorFieldKey  = "error"
	fieldsFieldKey = "fields"
)

var _ polylog.Event = (*stdLogEvent)(nil)

type stdLogEvent struct {
	levelString string
	fields      stdLogFields
}

type stdLogFields map[string]any

func newEvent(level polylog.Level) polylog.Event {
	return &stdLogEvent{
		levelString: getLevelString(level),
		fields:      make(stdLogFields),
	}
}

func (st *stdLogEvent) Str(key, value string) polylog.Event {
	st.fields[key] = value
	return st
}

func (st *stdLogEvent) Bool(key string, value bool) polylog.Event {
	st.fields[key] = value
	return st
}

func (st *stdLogEvent) Int(key string, value int) polylog.Event {
	st.fields[key] = value
	return st
}

func (st *stdLogEvent) Err(err error) polylog.Event {
	st.fields[errorFieldKey] = err
	return st
}

func (st *stdLogEvent) Fields(fields any) polylog.Event {
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

func getLevelString(level polylog.Level) string {
	switch level {
	case polylog.LevelDebug:
		return "[DEBUG]"
	case polylog.LevelInfo:
		return "[INFO]"
	case polylog.LevelWarn:
		return "[WARN]"
	case polylog.LevelError:
		return "[ERROR]"
	default:
		return "[UNKNOWN]"
	}
}
