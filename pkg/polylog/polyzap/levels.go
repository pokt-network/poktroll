package polyzap

import (
	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	// NB: zap log levels use -1 for Debug and 0 for Info.
	//DebugLevel = Level(iota)
	DebugLevel = Level(iota - 1)
	InfoLevel
	WarnLevel
	ErrorLevel
)

var _ polylog.Level = Level(0)

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

func (lvl Level) String() string {
	return zapcore.Level(lvl).String()
}

func (lvl Level) Int() int {
	return int(lvl)
}
