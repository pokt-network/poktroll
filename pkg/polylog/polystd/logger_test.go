package polystd_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polystd"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
)

const polystdEventTypeName = "*polystd.stdLogEvent"

var (
	expectedTime                   = time.Now()
	expectedTimestampLayout        = "2006-01-02T15:04:05-07:00"
	expectedTimestampEventContains = fmt.Sprintf(`"time":"%s"`, expectedTime.Format(expectedTimestampLayout))
	expectedTimeEventContains      = fmt.Sprintf(`"Time":"%s"`, expectedTime.Format(expectedTimestampLayout))
	expectedDuration               = time.Millisecond + (250 * time.Nanosecond) // 1000250

	expectedDurationEventContains = fmt.Sprintf(`"Dur":%q`, expectedDuration.String()) // 1.00025ms
)

func TestStdLogger_AllLevels_AllEventMethods(t *testing.T) {
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
		// (even with `make itest 500 10 ./pkg/polylog/... -- -run=StdLogger_AllLevels_AllEventTypeMethods`).
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
			// TODO_IMPROVE: assert on all key/value pairs. go doesn't guarantee
			// iteration oder of map key/value pairs. This requires promoting this
			// case to its own test or refactoring and/or restructuring test and
			// helper to support this.
			ExpectedOutputContains: `"key2":42`,
		},
		{
			EventMethodName: "Fields",
			Value:           []any{"key1", "value1", "key2", 42},
			// TODO_IMPROVE: assert on all key/value pairs. go doesn't guarantee
			// iteration oder of the slice (?). This requires promoting this
			// case to its own test or refactoring and/or restructuring test and
			// helper to support this.
			ExpectedOutputContains: `"key2":42`,
		},
	}

	levels := []polystd.Level{
		polystd.DebugLevel,
		polystd.InfoLevel,
		polystd.WarnLevel,
		polystd.ErrorLevel,
	}

	// TODO_IN_THIS_COMMIT: comment...
	for _, level := range levels {
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

func TestZerologLogger_Levels_Discard(t *testing.T) {
	// Construct a logger with each level. With each logger, log an event at each
	// level and assert that the event is logged if and only if the event level
	// is GTE the logger level.
	for _, loggerLevel := range polystd.Levels() {
		testDesc := fmt.Sprintf("%s level logger", loggerLevel.String())
		t.Run(testDesc, func(t *testing.T) {
			logger, logOutput := newTestLogger(t, loggerLevel)

			// Log an event for each level.
			for _, eventLevel := range polystd.Levels() {
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
	for _, loggerLevel := range polystd.Levels() {
		testDesc := fmt.Sprintf("%s loggerLevel logger", loggerLevel.String())
		t.Run(testDesc, func(t *testing.T) {
			var (
				notExpectedOutput = "if you're reading this, the test failed"
				// Construct a spy which implements a #Fn() method which we can use to
				// assert that the function passed to polylog.Event#Func() is called with
				// the expected arg(s).
				logger, logOutput = newTestLogger(t, loggerLevel)
			)

			for _, eventLevel := range polystd.Levels() {
				funcSpy := testpolylog.EventFuncSpy{}
				funcSpy.On("Fn", mock.AnythingOfType(polystdEventTypeName)).Return()

				event := newTestEventWithLevel(t, logger, eventLevel)
				expectedEventLevelEnabled := eventLevel.Int() >= loggerLevel.Int()

				require.Equalf(t, expectedEventLevelEnabled, event.Enabled(), "expected event to be initially enabled")

				// If the event level is GTE the logger level, then make additional
				// assertions about #Func(), #Discard(), and #Enabled() behavior.
				if expectedEventLevelEnabled {
					// Assert that #Func() calls `funcSpy#Fn()` method 1 time with
					// an event whose type name matches funcMethodEventTypeName.
					event.Func(funcSpy.Fn)
					funcSpy.AssertCalled(t, "Fn", mock.AnythingOfType(polystdEventTypeName))
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
	logger, logOutput := newTestLogger(t, polystd.DebugLevel)

	logger.Debug().Msg("before")
	require.Contains(t, logOutput.String(), "before")

	logger = logger.With("key", "value")

	logger.Debug().Msg("after")
	require.Contains(t, logOutput.String(), "after")
	require.Contains(t, logOutput.String(), `"key":"value"`)
}

// TODO_TEST/TODO_COMMUNITY: test-drive (TDD) out `polystd.Logger#WithContext()`.
func TestZerologLogger_WithContext(t *testing.T) {
	t.SkipNow()
}

func TestZerologLogger_WithLevel(t *testing.T) {
	logger, logOutput := newTestLogger(t, polystd.DebugLevel)
	logger.WithLevel(polystd.DebugLevel).Msg("WithLevel()")

	require.Contains(t, logOutput.String(), "WithLevel()")
}

func TestZerologLogger_Write(t *testing.T) {
	testOutput := "Write()"
	logger, logOutput := newTestLogger(t, polystd.DebugLevel)

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
		polystd.WithOutput(logOutput),
		// NB: typically consumers would use polystd.<some>Level directly instead
		// of casting like this.
		polystd.WithLevel(polystd.Level(level.Int())),
	)

	// TODO_IN_THIS_COMMIT: configuration ... debug level for this test
	logger := polystd.NewLogger(opts...)

	return logger, logOutput
}

// TODO_TEST: that exactly all expected levels log at each level.

// TODO_TEST: #Enabled() and #Discard()

func newTestEventWithLevel(
	t *testing.T,
	logger polylog.Logger,
	level polylog.Level,
) polylog.Event {
	t.Helper()

	// Match on level string to determine which method to call on the logger.
	switch level.String() {
	case polystd.DebugLevel.String():
		return logger.Debug()
	case polystd.InfoLevel.String():
		return logger.Info()
	case polystd.WarnLevel.String():
		return logger.Warn()
	case polystd.ErrorLevel.String():
		return logger.Error()
	default:
		panic(fmt.Errorf("level not yet supported: %s", level.String()))
	}
}

func getExpectedLevelOutputContains(level polylog.Level) string {
	return fmt.Sprintf(`[%s]`, strings.ToUpper(level.String()))
}
