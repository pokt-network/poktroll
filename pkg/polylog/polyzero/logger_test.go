package polyzero_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
)

const polyzeroEventTypeName = "*polyzero.zerologEvent"

var (
	expectedErr  = fmt.Errorf("%d", 42)
	expectedTime = time.Now()
	// expectedTimestampDayPrecisionLayout is a "layout" which is described using
	// the "reference time", as per the time package usage convention.
	// See: https://golang.org/pkg/time/#pkg-constants for more details on "layouts" and the "reference time".
	//
	// NB: #Timestamp() uses time.Now() internally. If the test is run around the
	// rollover of a second, minute, or hour, the expected timestamp time may not
	// match the actual time precisely enough. While this is still a possibility
	// near the rollover of a day, this window occurs less frequently and is many
	// multiples of the time it takes CI to run.
	//
	// TODO_CONSIDERATION: redesign the test helper to support regular expressions
	// for the output expectation.
	expectedTimestampDayPrecisionLayout = "2006-01-02T"
	// TODO_TECHDEBT: see TODO_TECHDEBT in #Time() test case.
	// expectedTimeLayout                  = "2006-01-02T15:04:05-07:00"
	// expectedTimeEventContains           = fmt.Sprintf(`"Time":"%s`, expectedTime.Format(expectedTimeLayout))
	expectedTimestampEventContains = fmt.Sprintf(`"time":"%s`, expectedTime.Format(expectedTimestampDayPrecisionLayout))
	expectedDuration               = time.Millisecond + (250 * time.Nanosecond)                   // 1000250
	expectedDurationString         = expectedDuration.String()[:len(expectedDuration.String())-2] // 1.00025
	expectedDurationEventContains  = fmt.Sprintf(`"Dur":%s`, expectedDurationString)
)

func TestZerologLogger_AllLevels_AllEventTypeMethods(t *testing.T) {
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
			Key:                    "Str",
			Value:                  "str_value",
			EventMethodName:        "Str",
			ExpectedOutputContains: `"Str":"str_value"`,
		},
		{
			Key:                    "Bool",
			Value:                  true,
			EventMethodName:        "Bool",
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
			Key:                    "Uint16",
			ExpectedOutputContains: `"Uint16":42`,
			Value:                  uint16(42),
			EventMethodName:        "Uint16",
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
			Value:                  expectedErr,
			ExpectedOutputContains: `"error":"42"`,
		},
		{
			EventMethodName:        "Timestamp",
			ExpectedOutputContains: expectedTimestampEventContains,
		},
		// TODO_TECHDEBT: figure out why this fails in CI but not locally,
		// (even with `make itest 500 10 ./pkg/polylog/... -- -run=ZeroLogger_AllLevels_AllEventTypeMethods`).
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
			ExpectedOutputContains: `"key1":"value1","key2":42`,
		},
		{
			EventMethodName:        "Fields",
			Value:                  []any{"key1", "value1", "key2", 42},
			ExpectedOutputContains: `"key1":"value1","key2":42`,
		},
	}

	for _, level := range polyzero.Levels() {
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
	for _, loggerLevel := range polyzero.Levels() {
		testDesc := fmt.Sprintf("%s level logger", loggerLevel.String())
		t.Run(testDesc, func(t *testing.T) {
			logger, logOutput := newTestLogger(t, loggerLevel)

			// Log an event for each level.
			for _, eventLevel := range polyzero.Levels() {
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
	for _, loggerLevel := range polyzero.Levels() {
		testDesc := fmt.Sprintf("%s loggerLevel logger", loggerLevel.String())
		t.Run(testDesc, func(t *testing.T) {
			var (
				notExpectedOutput = "if you're reading this, the test failed"
				// Construct a spy which implements a #Fn() method which we can use to
				// assert that the function passed to polylog.Event#Func() is called with
				// the expected arg(s).
				logger, logOutput = newTestLogger(t, loggerLevel)
			)

			for _, eventLevel := range polyzero.Levels() {
				funcSpy := testpolylog.EventFuncSpy{}
				funcSpy.On("Fn", mock.AnythingOfType(polyzeroEventTypeName)).Return()

				event := newTestEventWithLevel(t, logger, eventLevel)
				expectedEventLevelEnabled := eventLevel.Int() >= loggerLevel.Int()

				require.Equalf(t, expectedEventLevelEnabled, event.Enabled(), "expected event to be initially enabled")

				// If the event level is GTE the logger level, then make additional
				// assertions about #Func(), #Discard(), and #Enabled() behavior.
				if expectedEventLevelEnabled {
					// Assert that #Func() calls `funcSpy#Fn()` method 1 time with
					// an event whose type name matches funcMethodEventTypeName.
					event.Func(funcSpy.Fn)
					funcSpy.AssertCalled(t, "Fn", mock.AnythingOfType(polyzeroEventTypeName))
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
	logger, logOutput := newTestLogger(t, polyzero.DebugLevel)

	logger.Debug().Msg("before")
	require.Contains(t, logOutput.String(), "before")

	logger = logger.With("key", "value")

	logger.Debug().Msg("after")
	require.Contains(t, logOutput.String(), "after")
	require.Contains(t, logOutput.String(), `"key":"value"`)
}

func TestZerologLogger_WithLevel(t *testing.T) {
	logger, logOutput := newTestLogger(t, polyzero.DebugLevel)
	logger.WithLevel(polyzero.DebugLevel).Msg("WithLevel()")

	require.Contains(t, logOutput.String(), "WithLevel()")
}

func TestZerologLogger_Write(t *testing.T) {
	testOutput := "Write()"
	logger, logOutput := newTestLogger(t, polyzero.DebugLevel)

	n, err := logger.Write([]byte(testOutput))
	require.NoError(t, err)
	require.Lenf(t, testOutput, n, "expected %d bytes to be written", len(testOutput))

	require.Contains(t, logOutput.String(), testOutput)
}

func TestWithTimestampKey(t *testing.T) {
	expectedTimestampKey := "custom-timestamp-key"

	timestampKeyOpt := polyzero.WithTimestampKey(expectedTimestampKey)
	// Reset zerolog timestamp key to default value after test.
	t.Cleanup(func() {
		zerolog.TimestampFieldName = "time"
	})
	logger, logOutput := newTestLogger(t, polyzero.DebugLevel, timestampKeyOpt)

	logger.Debug().Timestamp().Send()

	expectedCustomTimestampEventContains := fmt.Sprintf(
		`"%s":"%s`,
		expectedTimestampKey,
		expectedTime.Format(expectedTimestampDayPrecisionLayout),
	)
	require.Contains(t, logOutput.String(), expectedCustomTimestampEventContains)

	// Print log output for manual inspection.
	t.Log(logOutput)
}

func TestWithErrorKey(t *testing.T) {
	expectedErrKey := "custom-error-key"

	errorKeyOpt := polyzero.WithErrKey(expectedErrKey)
	// Reset zerolog error key to default value after test.
	t.Cleanup(func() {
		zerolog.ErrorFieldName = "error"
	})
	logger, logOutput := newTestLogger(t, polyzero.DebugLevel, errorKeyOpt)

	logger.Debug().Err(expectedErr).Send()

	require.Contains(t, logOutput.String(), expectedErr.Error())
	require.Contains(t, logOutput.String(), expectedErrKey)

	// Print log output for manual inspection.
	t.Log(logOutput)
}

func TestZerologLogger_With_Deduplication(t *testing.T) {
	logger, logOutput := newTestLogger(t, polyzero.DebugLevel)

	// Step 1: Add a field "key" = "first_value"
	logger = logger.With("key", "first_value")
	logger.Debug().Msg("step1")
	require.Contains(t, logOutput.String(), `"key":"first_value"`, "expected field in log output after first With() call")

	// Clear buffer to isolate next output
	logOutput.Reset()

	// Step 2: Call With("key", "second_value") again
	logger = logger.With("key", "second_value")
	logger.Debug().Msg("step2")

	// The final JSON should only have "key" once, with "second_value"
	// If deduplication is broken, or fields are stacked, we might see "key":"first_value" plus "key":"second_value".
	require.Contains(t, logOutput.String(), `"key":"second_value"`, "expected deduplicated field in final output")
	require.NotContains(t, logOutput.String(), `"key":"first_value"`, "should not still contain the old value")
}

func newTestLogger(
	t *testing.T,
	level polylog.Level,
	opts ...polylog.LoggerOption,
) (polylog.Logger, *bytes.Buffer) {
	t.Helper()

	// Redirect standard log output to logOutput buffer.
	logOutput := new(bytes.Buffer)
	opts = append(
		opts,
		polyzero.WithOutput(logOutput),
		// NB: typically consumers would pass zerolog.<some>Level directly instead.
		polyzero.WithLevel(level),
	)

	logger := polyzero.NewLogger(opts...)

	return logger, logOutput
}

func newTestEventWithLevel(
	t *testing.T,
	logger polylog.Logger,
	level polylog.Level,
) polylog.Event {
	t.Helper()

	// Match on level string to determine which level method to call.
	switch level.String() {
	case zerolog.DebugLevel.String():
		return logger.Debug()
	case zerolog.InfoLevel.String():
		return logger.Info()
	case zerolog.WarnLevel.String():
		return logger.Warn()
	case zerolog.ErrorLevel.String():
		return logger.Error()
	default:
		panic(fmt.Errorf("level not yet supported: %s", level.String()))
	}
}

func getExpectedLevelOutputContains(level polylog.Level) string {
	return fmt.Sprintf(`"level":%q`, level.String())
}
