//go:build test

package polyzap

import (
	"go.uber.org/zap"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// GetZapLogger is a helper function which provides direct access to the underlying
// zap logger for testing purposes; e.g. use in assertions. To use this helper,
// ensure that the build tag/constraint "test" is set (e.g. `go build -tags=test`).
// It MUST be defined in this package (as opposed to somewhere in testutils), as
// by definition, it references unexported members of this package.
func GetZapLogger(logger polylog.Logger) *zap.Logger {
	return logger.(*zapLogger).logger
}
