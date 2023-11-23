package testpolylog

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

type (
	// NewLoggerAndOutputFn is called in the test helper to create a new logger
	// configured with the given level and options. It returns the logger and
	// the buffer to which the logger writes. It is useful for decoupling test
	// helpers from a specific logger implementation.
	NewLoggerAndOutputFn func(
		*testing.T,
		polylog.Level,
		...polylog.LoggerOption,
	) (polylog.Logger, *bytes.Buffer)

	// NewEventWithLevelFn is called in the test helper to create a new event
	// at the given level from the given logger. It is useful for decoupling
	// test helpers from a specific logger implementation.
	NewEventWithLevelFn func(
		*testing.T,
		polylog.Logger,
		polylog.Level,
	) polylog.Event
)

// FnMethodSpy is a mock which implements a #Fn() method that is intended to be
// used in tests to assert that the function passed to polylog.Event#Func() is
// called with the expected arg(s).
type FnMethodSpy struct{ mock.Mock }

// Fn is a mock method which can be asserted on via the mock.Mock API.
// See: https://pkg.go.dev/github.com/stretchr/testify@v1.8.4/mock#Mock.
func (m *FnMethodSpy) Fn(event polylog.Event) {
	m.Called(event)
}

// EventMethodsTest is a test case for expressing and exercising polylog.Event
// methods in a concise way.
type EventMethodsTest struct {
	Msg                    string
	MsgFmt                 string
	MsgFmtArgs             []any
	Key                    string
	Value                  any
	EventMethodName        string
	ExpectedOutputContains string
}

// RunEventMethodTests runs a set of tests for a given level. It also includes a
// test for polylog.Event#Func().
func RunEventMethodTests(
	t *testing.T,
	level polylog.Level,
	tests []EventMethodsTest,
	newLoggerAndOutput NewLoggerAndOutputFn,
	newEventWithLevel NewEventWithLevelFn,
	funcMethodEventTypeName string,
	getExpectedLevelOutputContains func(level polylog.Level) string,
) {
	t.Helper()

	// Title-case level string so that it can be used as the name of the
	// method to call on the logger using reflect and for the sub-test
	// descriptions.
	//
	// TODO_TECHDEBT/TODO_COMMUNITY: `strings.Title()` is deprecated. Follow
	// migration guidance in godocs: https://pkg.go.dev/strings@go1.21.4#Title.
	levelMethodStr := strings.Title(level.String())

	for _, tt := range tests {
		// If the test case does not specify an event method name, use the test
		// case's key as the event method name. This is done for convenience only.
		if tt.EventMethodName == "" {
			tt.EventMethodName = tt.Key
		}

		var (
			eventMethodArgs []reflect.Value
			doneMethodName  string
			doneMethodArgs  []reflect.Value
		)

		// Ensure that calls to #Msg(), #Msgf(), and #Send() are mutually exclusive.
		switch {
		case tt.Msg != "":
			// Set up call args for polylog.Event#Msg() if tt.msg is not emtpy.
			doneMethodName = "Msg"
			doneMethodArgs = append(doneMethodArgs, reflect.ValueOf(tt.Msg))
		case tt.MsgFmt != "":
			// Set up call args for polylog.Event#Msgf() if tt.msgFmt is not emtpy.
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

		// If tt.EventMethodName and tt.Key are both empty, then use the done
		// method name for the test description instead of the event method name.
		descMethodName := tt.EventMethodName
		if tt.EventMethodName == "" {
			descMethodName = doneMethodName
		}
		testDesc := fmt.Sprintf("%s().%s()", levelMethodStr, descMethodName)

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

	// Assert that #Func() works for each test at each level.
	funcTestDesc := fmt.Sprintf("%s().Func()", levelMethodStr)
	t.Run(funcTestDesc, func(t *testing.T) {
		logger, _ := newLoggerAndOutput(t, level)

		// Construct a spy which implements a #Fn() method which we can use to
		// assert that the function passed to polylog.Event#Func() is called with
		// the expected arg(s).
		funcSpy := FnMethodSpy{}
		funcSpy.On("Fn", mock.AnythingOfType(funcMethodEventTypeName)).Return()

		logger.Debug().Func(funcSpy.Fn).Send()

		// Assert that `funcSpy#Fn()` method is called with an event whose type
		// name matches funcMethodEventTypeName.
		funcSpy.AssertCalled(t, "Fn", mock.AnythingOfType(funcMethodEventTypeName))
	})
}
