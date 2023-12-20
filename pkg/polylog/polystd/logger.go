package polystd

import (
	"context"
	"fmt"
	"log"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ polylog.Logger = (*stdLogLogger)(nil)

type stdLogLogger struct {
	level Level
}

func NewLogger(opts ...polylog.LoggerOption) polylog.Logger {
	logger := &stdLogLogger{}

	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

func (st *stdLogLogger) Debug() polylog.Event {
	return newEvent(DebugLevel)
}

func (st *stdLogLogger) Info() polylog.Event {
	return newEvent(InfoLevel)
}

func (st *stdLogLogger) Warn() polylog.Event {
	return newEvent(WarnLevel)
}

func (st *stdLogLogger) Error() polylog.Event {
	return newEvent(ErrorLevel)
}

// WithContext ...
//
// TODO_TEST/TODO_COMMUNITY: test-drive (TDD) out `polystd.Logger#WithContext()`.
func (st *stdLogLogger) WithContext(ctx context.Context) context.Context {
	panic("not yet implemented")
}

func (st *stdLogLogger) With(keyVals ...any) polylog.Logger {
	// TODO_TECHDEBT:TODO_COMMUNITY: implement this to have analogous behavior
	// to that of `polyzero.Logger`'s. Investigate `log.SetPrefix()` and consider
	// combining the level label with formatted keyVals as a prefix.
	panic("not yet implemented")
}

func (st *stdLogLogger) WithLevel(level polylog.Level) polylog.Event {
	switch level.String() {
	case DebugLevel.String():
		return st.Debug()
	case InfoLevel.String():
		return st.Info()
	case WarnLevel.String():
		return st.Warn()
	case ErrorLevel.String():
		return st.Error()
	default:
		panic(fmt.Sprintf("level not supported: %s", level.String()))
	}
}

func (st *stdLogLogger) Write(p []byte) (n int, err error) {
	return log.Writer().Write(p)
}
