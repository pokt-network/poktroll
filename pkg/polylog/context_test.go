package polylog_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func TestWithContext_Ctx(t *testing.T) {
	var (
		expectedLogger = polyzero.NewLogger()
		ctx            = context.Background()
	)

	// Ensure that no logger is associated with the context.
	// TODO_TECHDEBT: refactor once Ctx() no longer panics.
	require.Panics(t, func() { polylog.Ctx(ctx) })

	// Associate a logger with a context.
	ctx = expectedLogger.WithContext(ctx)

	// Retrieve the associated logger from the context using polylog and assert
	// that it matches the one constructed at the beginning of the test.
	actualLogger := polylog.Ctx(ctx)
	require.Equal(t, expectedLogger, actualLogger)

	// Retrieve the associated logger from the context using zerolog and assert
	// that it matches the one constructed at the beginning of the test.
	actualZerologLogger := zerolog.Ctx(ctx)
	expectedZerologLogger := polyzero.GetZerologLogger(expectedLogger)
	require.Equal(t, expectedZerologLogger, actualZerologLogger)
}
