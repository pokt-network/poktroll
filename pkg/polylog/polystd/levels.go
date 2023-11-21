package polystd

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	// TODO_IN_THIS_COMMIT: consider fatal and panic levels.
)

type Level int

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}
