package polystd

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	errorFieldKey = "error"
	// TODO_IMPROVE: Support configurable timestamp format via an options.
	defaultTimeLayout = time.RFC3339
)

var _ polylog.Event = (*stdLogEvent)(nil)

type stdLogEvent struct {
	levelString string
	fieldsMu    sync.Mutex
	fields      stdLogFields
	discardedMu sync.Mutex
	discarded   bool
}

type stdLogFields map[string]any

func newEvent(level Level) polylog.Event {
	return &stdLogEvent{
		levelString: getLevelLabel(level),
		fields:      make(stdLogFields),
	}
}

func (sle *stdLogEvent) Str(key, value string) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Bool(key string, value bool) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Int(key string, value int) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Int8(key string, value int8) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Int16(key string, value int16) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Int32(key string, value int32) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Int64(key string, value int64) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Uint(key string, value uint) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Uint8(key string, value uint8) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Uint16(key string, value uint16) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Uint32(key string, value uint32) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Uint64(key string, value uint64) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Float32(key string, value float32) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Float64(key string, value float64) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value
	return sle
}

func (sle *stdLogEvent) Err(err error) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[errorFieldKey] = err
	return sle
}

func (sle *stdLogEvent) Timestamp() polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	// TODO_IMPROVE: this key should be configurable via an option.
	sle.fields["time"] = time.Now().Format(defaultTimeLayout)
	return sle
}

func (sle *stdLogEvent) Time(key string, value time.Time) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value.Format(defaultTimeLayout)
	return sle
}

func (sle *stdLogEvent) Dur(key string, value time.Duration) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	sle.fields[key] = value.String()
	return sle
}

func (sle *stdLogEvent) Fields(fields any) polylog.Event {
	sle.fieldsMu.Lock()
	defer sle.fieldsMu.Unlock()

	switch fieldsVal := fields.(type) {
	case map[string]any:
		for key, value := range fieldsVal {
			sle.fields[key] = value
		}
	case []any:
		var nextKey string
		for fieldIdx, value := range fieldsVal {
			if fieldIdx%2 == 0 {
				nextKey = value.(string)
			} else {
				sle.fields[nextKey] = value
			}
		}
	}
	return sle
}

func (sle *stdLogEvent) Func(fn func(polylog.Event)) polylog.Event {
	if sle.Enabled() {
		fn(sle)
	}
	return sle
}

func (sle *stdLogEvent) Enabled() bool {
	sle.discardedMu.Lock()
	defer sle.discardedMu.Unlock()

	return !sle.discarded
}

func (sle *stdLogEvent) Discard() polylog.Event {
	sle.discardedMu.Lock()
	defer sle.discardedMu.Unlock()

	sle.discarded = true
	return sle
}

func (sle *stdLogEvent) Msg(msg string) {
	log.Println(sle.levelString, sle.fields.String(), msg)
}

func (sle *stdLogEvent) Msgf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Println(sle.levelString, sle.fields.String(), msg)
}

func (sle *stdLogEvent) Send() {
	log.Println(sle.levelString, sle.fields.String())
}

// TODO_IMPROVE: Support configurable key/value and field delimiters via options.
func (stf stdLogFields) String() string {
	var fieldLines []string
	for key, value := range stf {
		var line string
		switch concreteVal := value.(type) {
		case string:
			line = fmt.Sprintf("%q:%q", key, value)
		case error:
			line = fmt.Sprintf("%q:%q", key, concreteVal.Error())
		default:
			line = fmt.Sprintf("%q:%v", key, value)
		}
		fieldLines = append(fieldLines, line)
	}
	return strings.Join(fieldLines, ",")
}

func getLevelLabel(level Level) string {
	switch level {
	case DebugLevel:
		return "[DEBUG]"
	case InfoLevel:
		return "[INFO]"
	case WarnLevel:
		return "[WARN]"
	case ErrorLevel:
		return "[ERROR]"
	default:
		return "[UNKNOWN]"
	}
}
