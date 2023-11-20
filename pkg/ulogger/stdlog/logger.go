package stdlog

import (
	"github.com/pokt-network/poktroll/pkg/ulogger"
)

//var _ ulogger.UniversalLogger = (*stdLogULogger)(nil)

type stdLogULogger struct{}

func NewUniversalLogger() ulogger.UniversalLogger {
	//return &stdLogULogger{}
	return nil
}

func (st *stdLogULogger) Debug() {
	//return newEvent
}

func (st *stdLogULogger) Info() {
	//log.Println("[INFO]", msg, keysAndValues)
}

func (st *stdLogULogger) Warn() {
	//log.Println("[WARN]", msg, keysAndValues)
}

func (st *stdLogULogger) Error() {
	//log.Println("[ERROR]", msg, keysAndValues)
}
