package testpolylog

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/pokt-network/pocket/pkg/polylog"
)

type (
	// NewLoggerAndOutputFn is called in the test helper to create a new logger
	// configured with the given level and options. It returns the logger and
	// the buffer to which the logger writes. It is useful for decoupling test
	// helpers from a specific logger implementation and reducing boilerplate
	// code in various tests.
	NewLoggerAndOutputFn func(
		*testing.T,
		polylog.Level,
		...polylog.LoggerOption,
	) (polylog.Logger, *bytes.Buffer)

	// NewEventWithLevelFn is called in the test helper to create a new event
	// at the given level from the given logger. It is useful for decoupling
	// test helpers from a specific logger implementation so that we can
	// call `logger.<LevelMethod>() without knowing the concrete logger
	// type nor the level.
	NewEventWithLevelFn func(
		*testing.T,
		polylog.Logger,
		polylog.Level,
	) polylog.Event
)

// EventFuncSpy is a mock which implements a #Fn() method that is intended to be
// used in tests to assert that the function passed to polylog.Event#Func() is
// called with the expected arg(s).
type EventFuncSpy struct{ mock.Mock }

// Fn is a mock method which can be asserted on via the mock.Mock API.
// See: https://pkg.go.dev/github.com/stretchr/testify@v1.8.4/mock#Mock.
func (m *EventFuncSpy) Fn(event polylog.Event) {
	m.Called(event)
}

// EventMethodTestCase is a test case for expressing and exercising polylog.Event
// methods in a concise way.
type EventMethodTestCase struct {
	// Msg is the string to pass to polylog.Event#Msg(), which will be called
	// after the event method under test. Usage of Msg is mutually exclusive
	// with MsgFmt. If neither are provided, then polylog.Event#Send() is called
	// after the event method instead.
	Msg string

	// MsgFmt is the format string to pass to polylog.Event#Msgf(), which will
	// be called on the event returned from the event method under test. Usage
	// of MsgFmt is mutually exclusive with Msg. If neither are provided, then
	// polylog.Event#Send() is called after the event method instead.
	MsgFmt string

	// MsgFmtArgs are the args to pass to polylog.Event#Msgf(). It is an error
	// to provide MsgFmtArgs without also providing MsgFmt or while providing
	// Msg.
	MsgFmtArgs []any

	// Key is the key to pass to the event method under test.
	Key string

	// Value is the value to pass to the event method under test.
	Value any

	// EventMethodName is the name of the event method to call on the logger.
	EventMethodName string

	// ExpectedOutputContains is the string that is expected to be contained
	// in the log output.
	ExpectedOutputContains string
}

// RunEventMethodTests runs a set of tests for a given level.
func RunEventMethodTests(
	t *testing.T,
	level polylog.Level,
	tests []EventMethodTestCase,
	newLoggerAndOutput NewLoggerAndOutputFn,
	newEventWithLevel NewEventWithLevelFn,
	getExpectedLevelOutputContains func(level polylog.Level) string,
) {
	t.Helper()

	// Title-case level string so that it can be used as the name of the
	// method to call on the logger using reflect and for the sub-test
	// descriptions.
	levelMethodName := cases.Title(language.Und).String(level.String())

	for _, tt := range tests {
		var (
			eventMethodArgs []reflect.Value
			doneMethodName  string
			doneMethodArgs  []reflect.Value
		)

		// Ensure that calls to #Msg(), #Msgf(), and #Send() are mutually exclusive.
		switch {
		case tt.Msg != "":
			require.Emptyf(t, tt.MsgFmt, "Msg and MsgFmt are mutually exclusive but MsgFmt was not empty: %s", tt.MsgFmt)
			require.Emptyf(t, tt.MsgFmtArgs, "Msg and MsgFmt are mutually exclusive but MsgFmtArgs was not empty: %v", tt.MsgFmtArgs)

			// Set up call args for polylog.Event#Msg() if tt.msg is not empty.
			doneMethodName = "Msg"
			doneMethodArgs = append(doneMethodArgs, reflect.ValueOf(tt.Msg))
		case tt.MsgFmt != "":
			// Set up call args for polylog.Event#Msgf() if tt.msgFmt is not empty.
			doneMethodName = "Msgf"
			doneMethodArgs = append(
				doneMethodArgs,
				reflect.ValueOf(tt.MsgFmt),
				reflect.ValueOf(tt.MsgFmtArgs),
			)
		default:
			// Default to calling polylog.Event#Send() if tt.msg and tt.msgFmt are
			// both empty.
			doneMethodName = "Send"
		}

		// Test description for this sub-test is interpolated based on the logger
		// level, event, and "done" method names (e.g. `Debug().Msg()` or `Info().Str()`).
		// If the event method name is not empty, the done method name is omitted.
		// This is done for brevity as not every permutation of event method and done
		// method is exercised (nor need they be).
		// If the event method name is empty, then the test description is interpolated
		// using the level method name and the "done" method name (e.g. `Error().Msg()`
		// or `Warn().Send()`).
		descMethodName := tt.EventMethodName
		if tt.EventMethodName == "" {
			descMethodName = doneMethodName
		}
		testDesc := fmt.Sprintf("%s().%s()", levelMethodName, descMethodName)

		// Run this sub-test for the current level.
		t.Run(testDesc, func(t *testing.T) {
			logger, logOutput := newLoggerAndOutput(t, level)

			// Need to use reflection in order to minimize the test code necessary
			// to exercise all the permutations of logger level and event type methods.
			logEvent := newEventWithLevel(t, logger, level)
			logEventValue := reflect.ValueOf(logEvent)

			// If tt.EventMethodName is not empty, build the args and call it.
			if tt.EventMethodName != "" {
				// Append tt.key to polylog.Event#<level>() call args.
				if tt.Key != "" {
					eventMethodArgs = append(eventMethodArgs, reflect.ValueOf(tt.Key))
				}
				// Append tt.value to polylog.Event#<level>() call args.
				if tt.Value != nil {
					eventMethodArgs = append(eventMethodArgs, reflect.ValueOf(tt.Value))
				}

				// E.g.: logEvent := logger.Debug().Str("str", "str_value")
				//   or: logEvent := logger.Debug().Bool("bool", true)
				logEventValue.
					MethodByName(tt.EventMethodName).
					Call(eventMethodArgs)
			}

			// E.g.: logEvent.Send()
			//   or: logEvent.Msg("msg")
			//   or: logEvent.Msgf("meaning of life: %d", 42)
			logEventValue.
				MethodByName(doneMethodName).
				Call(doneMethodArgs)

			// Assert that each line contains the expected prefix.
			require.Contains(t, logOutput.String(), getExpectedLevelOutputContains(level))

			// Assert that the log output contains the expected messages. Split the log
			// output into lines and iterate over them.
			require.Contains(t, logOutput.String(), tt.ExpectedOutputContains)

			// Print log output for manual inspection.
			t.Log(logOutput.String())
		})
	}
}
