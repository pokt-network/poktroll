package polyzero

import "github.com/pokt-network/poktroll/pkg/polylog"

func init() {
	// Set the default logger to a polyzero logger. This is the logger which will
	// be returned when calling polylog.Ctx() with a context which has no logger
	// associated.
	//
	// This is assigned here to avoid an import cycle. Note that dependency init
	// functions are called before dependents. It is therefore safe to override
	// this default logger assignment in polylog consumer  code, including in
	// consumer package init functions.
	polylog.DefaultContextLogger = NewLogger()
}
