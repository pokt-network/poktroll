package polyzero

import (
	"github.com/rs/zerolog"

	"github.com/pokt-network/pocket/pkg/polylog"
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

// ParseLevel returns the polyzero.Level for the given string. It returns InfoLevel
// if the string is not recognized.
func ParseLevel(level string) polylog.Level {
	switch level {
	case "debug", "Debug", "DEBUG":
		return DebugLevel
	case "info", "Info", "INFO":
		return InfoLevel
	case "warn", "Warn", "WARN":
		return WarnLevel
	case "error", "Error", "ERROR":
		return ErrorLevel
	default:
		return InfoLevel
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
