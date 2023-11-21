package polylog

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	// TODO_IN_THIS_COMMIT: consider fatal and panic levels.
)

type Level int

type LoggerOption func(logger PolyLogger)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}
