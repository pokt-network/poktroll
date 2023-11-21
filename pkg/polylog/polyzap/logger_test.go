package polyzap_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzap"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
)

const polyzapEventTypeName = "*polyzap.zapEvent"

var (
	expectedTime                   = time.Now()
	expectedTimestampEventContains = fmt.Sprintf(`"ts":%d.`, expectedTime.Unix())
	expectedTimeEventContains      = fmt.Sprintf(`"Time":%d.`, expectedTime.Unix())
	expectedDuration               = time.Millisecond + (250 * time.Nanosecond) // 1000250
	expectedDurationEventContains  = fmt.Sprintf(`"Dur":%f`, expectedDuration.Seconds())
)

func TestZapLogger_AllLevels_AllEventMethods(t *testing.T) {
	tests := []testpolylog.EventMethodTestCase{
		{
			// Explicitly left empty; no event method should be called.
			EventMethodName:        "",
			Msg:                    "Msg",
			ExpectedOutputContains: "Msg",
		},
		{
			// Explicitly left empty; no event method should be called.
			EventMethodName:        "",
			MsgFmt:                 "%s",
			MsgFmtArgs:             []any{"Msgf"},
			ExpectedOutputContains: "Msgf",
		},
		{
			EventMethodName:        "Str",
			Key:                    "Str",
			Value:                  "str_value",
			ExpectedOutputContains: `"Str":"str_value"`,
		},
		{
			EventMethodName:        "Bool",
			Key:                    "Bool",
			Value:                  true,
			ExpectedOutputContains: `"Bool":true`,
		},
		{
			EventMethodName:        "Int",
			Key:                    "Int",
			Value:                  int(42),
			ExpectedOutputContains: `"Int":42`,
		},
		{
			EventMethodName:        "Int8",
			Key:                    "Int8",
			Value:                  int8(42),
			ExpectedOutputContains: `"Int8":42`,
		},
		{
			EventMethodName:        "Int16",
			Key:                    "Int16",
			Value:                  int16(42),
			ExpectedOutputContains: `"Int16":42`,
		},
		{
			EventMethodName:        "Int32",
			Key:                    "Int32",
			Value:                  int32(42),
			ExpectedOutputContains: `"Int32":42`,
		},
		{
			EventMethodName:        "Int64",
			Key:                    "Int64",
			Value:                  int64(42),
			ExpectedOutputContains: `"Int64":42`,
		},
		{
			EventMethodName:        "Uint",
			Key:                    "Uint",
			Value:                  uint(42),
			ExpectedOutputContains: `"Uint":42`,
		},
		{
			EventMethodName:        "Uint8",
			Key:                    "Uint8",
			Value:                  uint8(42),
			ExpectedOutputContains: `"Uint8":42`,
		},
		{
			EventMethodName:        "Uint16",
			Key:                    "Uint16",
			Value:                  uint16(42),
			ExpectedOutputContains: `"Uint16":42`,
		},
		{
			EventMethodName:        "Uint32",
			Key:                    "Uint32",
			Value:                  uint32(42),
			ExpectedOutputContains: `"Uint32":42`,
		},
		{
			EventMethodName:        "Uint64",
			Key:                    "Uint64",
			Value:                  uint64(42),
			ExpectedOutputContains: `"Uint64":42`,
		},
		{
			EventMethodName:        "Float32",
			Key:                    "Float32",
			Value:                  float32(420.69),
			ExpectedOutputContains: `"Float32":420.69`,
		},
		{
			EventMethodName:        "Float64",
			Key:                    "Float64",
			Value:                  float64(420.69),
			ExpectedOutputContains: `"Float64":420.69`,
		},
		{
			EventMethodName:        "Err",
			Value:                  fmt.Errorf("%d", 42),
			ExpectedOutputContains: `"error":"42"`,
		},
		{
			EventMethodName:        "Timestamp",
			ExpectedOutputContains: expectedTimestampEventContains,
		},
		// TODO_TECHDEBT: figure out why this fails in CI but not locally,
		// (even with `make itest 500 10 ./pkg/polylog/... -- -run=ZapLogger_AllLevels_AllEventMethods`).
		//
		//{
		//  EventMethodName:        "Time",
		//	Key:                    "Time",
		//	Value:                  expectedTime,
		//	ExpectedOutputContains: expectedTimeEventContains,
		//},
		{
			EventMethodName:        "Dur",
			Key:                    "Dur",
			Value:                  expectedDuration,
			ExpectedOutputContains: expectedDurationEventContains,
		},
		{
			EventMethodName: "Fields",
			Value: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			// TODO_IMPROVE: assert on all key/value pairs. Zap doesn't seem to
			// provide any guarantee around the oder of the fields. This requires
			// changing the test and helper structure to support this.
			ExpectedOutputContains: `"key2":42`,
		},
		{
			EventMethodName: "Fields",
			Value:           []any{"key1", "value1", "key2", 42},
			// TODO_IMPROVE: assert on all key/value pairs. Zap doesn't seem to
			// provide any guarantee around the oder of the fields. This requires
			// changing the test and helper structure to support this.
			ExpectedOutputContains: `"key2":42`,
		},
	}

	// TODO_IN_THIS_COMMIT: comment...
	for _, level := range polyzap.Levels() {
		testpolylog.RunEventMethodTests(
			t,
			level,
			tests,
			newTestLogger,
			newTestEventWithLevel,
			getExpectedLevelOutputContains,
		)
	}
}

func TestZapLogger_Levels_Discard(t *testing.T) {
	// Construct a logger with each level. With each logger, log an event at each
	// level and assert that the event is logged if and only if the event level
	// is GTE the logger level.
	for _, loggerLevel := range polyzap.Levels() {
		testDesc := fmt.Sprintf("%s level logger", loggerLevel.String())
		t.Run(testDesc, func(t *testing.T) {
			logger, logOutput := newTestLogger(t, loggerLevel)

			// Log an event for each level.
			for _, eventLevel := range polyzap.Levels() {
				event := newTestEventWithLevel(t, logger, eventLevel)
				// Log the event level string.
				event.Msg(eventLevel.String())

				// If the event level is GTE the logger level, then the event should
				// be logged.
				if eventLevel.Int() >= loggerLevel.Int() {
					require.Truef(t, event.Enabled(), "expected event to be enabled")
					require.Contains(t, logOutput.String(), eventLevel.String())
				} else {
					require.Falsef(t, event.Enabled(), "expected event to be discarded")
					require.NotContains(t, logOutput.String(), eventLevel.String())
				}
			}

			// Print log output for manual inspection.
			t.Log(logOutput.String())
		})
	}
}

func TestZerologLogger_Func_Discard_Enabled(t *testing.T) {
	for _, loggerLevel := range polyzap.Levels() {
		testDesc := fmt.Sprintf("%s loggerLevel logger", loggerLevel.String())
		t.Run(testDesc, func(t *testing.T) {
			var (
				notExpectedOutput = "if you're reading this, the test failed"
				// Construct a spy which implements a #Fn() method which we can use to
				// assert that the function passed to polylog.Event#Func() is called with
				// the expected arg(s).
				logger, logOutput = newTestLogger(t, loggerLevel)
			)

			for _, eventLevel := range polyzap.Levels() {
				funcSpy := testpolylog.EventFuncSpy{}
				funcSpy.On("Fn", mock.AnythingOfType(polyzapEventTypeName)).Return()

				event := newTestEventWithLevel(t, logger, eventLevel)
				expectedEventLevelEnabled := eventLevel.Int() >= loggerLevel.Int()

				require.Equalf(t, expectedEventLevelEnabled, event.Enabled(), "expected event to be initially enabled")

				// If the event level is GTE the logger level, then make additional
				// assertions about #Func(), #Discard(), and #Enabled() behavior.
				if expectedEventLevelEnabled {
					// Assert that #Func() calls `funcSpy#Fn()` method 1 time with
					// an event whose type name matches funcMethodEventTypeName.
					event.Func(funcSpy.Fn)
					funcSpy.AssertCalled(t, "Fn", mock.AnythingOfType(polyzapEventTypeName))
					funcSpy.AssertNumberOfCalls(t, "Fn", 1)

					event.Discard()
					require.Falsef(t, event.Enabled(), "expected event to be disabled after Discard()")

					// Assert that #Func() **does not** call `funcSpy#Fn()` method again.
					event.Func(funcSpy.Fn)
					funcSpy.AssertNumberOfCalls(t, "Fn", 1)

					event.Msg(notExpectedOutput)
					require.NotContains(t, logOutput.String(), notExpectedOutput)
				}

				// NB: this test doesn't produce any log output as all cases
				// exercise discarding.
			}
		})
	}
}

func TestZerologLogger_With(t *testing.T) {
	logger, logOutput := newTestLogger(t, polyzap.DebugLevel)

	logger.Debug().Msg("before")
	require.Contains(t, logOutput.String(), "before")

	logger = logger.With("key", "value")

	logger.Debug().Msg("after")

	require.Contains(t, logOutput.String(), "after")
	require.Contains(t, logOutput.String(), `"key":"value"`)

	// Print log output for manual inspection.
	t.Log(logOutput.String())
}

func TestZerologLogger_WithContext(t *testing.T) {
	var (
		expectedLogger = polyzap.NewLogger()
		ctx            = context.Background()
	)

	// Ensure that no logger is associated with the context.
	existingLogger, ok := ctx.Value(polylog.CtxKey).(polylog.Logger)
	require.False(t, ok)
	require.Nil(t, existingLogger)

	// Retrieve the default logger from the context using polylog and assert
	// that it matches the default context logger.
	defaultLogger := polylog.Ctx(ctx)
	require.Equal(t, polylog.DefaultContextLogger, defaultLogger)

	// Associate a logger with a context.
	ctx = expectedLogger.WithContext(ctx)

	// Retrieve the associated logger from the context using polylog and assert
	// that it matches the one constructed at the beginning of the test.
	actualLogger := polylog.Ctx(ctx)
	require.Equal(t, expectedLogger, actualLogger)
}

// TODO_TECHDEBT/TODO_COMMUNITY: TDD this integration with zap. See `polyzero`
// package for comparison / starting point.
func TestWithTimestampKey(t *testing.T) {
	t.SkipNow()
}

// TODO_TECHDEBT/TODO_COMMUNITY: TDD this integration with zap. See `polyzero`
// package for comparison / starting point.
func TestWithErrorKey(t *testing.T) {
	t.SkipNow()
}

func TestZerologLogger_WithLevel(t *testing.T) {
	logger, logOutput := newTestLogger(t, polyzap.DebugLevel)
	logger.WithLevel(polyzap.DebugLevel).Msg("WithLevel()")

	require.Contains(t, logOutput.String(), "WithLevel()")
}

func TestZerologLogger_Write(t *testing.T) {
	testOutput := "Write()"
	logger, logOutput := newTestLogger(t, polyzap.DebugLevel)

	n, err := logger.Write([]byte(testOutput))
	require.NoError(t, err)
	require.Lenf(t, testOutput, n, "expected %d bytes to be written", len(testOutput))

	require.Contains(t, logOutput.String(), testOutput)
}

func newTestLogger(
	t *testing.T,
	level polylog.Level,
	opts ...polylog.LoggerOption,
) (polylog.Logger, *bytes.Buffer) {
	t.Helper()

	// Redirect standard log output to logOutput buffer.
	logOutput := new(bytes.Buffer)
	opts = append(opts,
		polyzap.WithOutput(logOutput),
		polyzap.WithLevel(polyzap.Level(level.Int())),
	)

	logger := polyzap.NewLogger(opts...)

	return logger, logOutput
}

func newTestEventWithLevel(
	t *testing.T,
	logger polylog.Logger,
	level polylog.Level,
) polylog.Event {
	t.Helper()

	switch level.String() {
	case zap.DebugLevel.String():
		return logger.Debug()
	case zap.InfoLevel.String():
		return logger.Info()
	case zap.WarnLevel.String():
		return logger.Warn()
	case zap.ErrorLevel.String():
		return logger.Error()
	default:
		panic(fmt.Errorf("level not yet supported: %s", level.String()))
	}
}

func getExpectedLevelOutputContains(level polylog.Level) string {
	return fmt.Sprintf(`"level":%q`, level.String())
}
