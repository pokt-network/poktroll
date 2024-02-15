package polyzero

import (
	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// TODO_TECHDEBT: support a Disabled level.
const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = Level(iota)
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
)

var _ polylog.Level = Level(0)

// Level implements the polylog.Level interface for zerolog levels.
type Level int

// Levels is a convenience function to return all supported levels.
func Levels() []Level {
	return []Level{
		DebugLevel,
		InfoLevel,
		WarnLevel,
		ErrorLevel,
	}
}

// String implements polylog.Level#String().
func (lvl Level) String() string {
	return zerolog.Level(lvl).String()
}

// Int implements polylog.Level#Int().
func (lvl Level) Int() int {
	return int(lvl)
}
