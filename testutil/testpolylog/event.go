package testpolylog

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

// TODO_IN_THIS_COMMIT: comment...
type funcMethodSpy struct{ mock.Mock }

// TODO_IN_THIS_COMMIT: comment...
func (m *funcMethodSpy) Fn(event polylog.Event) {
	m.Called(event)
}

// TODO_IN_THIS_COMMIT: export and move to shared test pkg.
type EventMethodsTest struct {
	Msg                    string
	MsgFmt                 string
	MsgFmtArgs             []any
	Key                    string
	Value                  any
	EventMethodName        string
	ExpectedOutputContains string
}

func RunEventMethodTests(
	t *testing.T,
	levelStr string,
	tests []EventMethodsTest,
	newLoggerAndOutput func() (polylog.Logger, *bytes.Buffer),
) {
	t.Helper()

	for _, tt := range tests {
		// TODO_IN_THIS_COMMIT: comment...
		if tt.EventMethodName == "" {
			tt.EventMethodName = tt.Key
		}

		var (
			methodArgs     = make([]reflect.Value, 0)
			doneMethodName = "Send"
			doneMethodArgs []reflect.Value
		)

		// Set up call args for polylog.Event#Msg() if tt.msg is not emtpy.
		if tt.Msg != "" {
			doneMethodName = "Msg"
			doneMethodArgs = append(doneMethodArgs, reflect.ValueOf(tt.Msg))

			// TODO_IN_THIS_COMMIT: comment...
			if tt.EventMethodName == "" {
				tt.EventMethodName = doneMethodName
			}
		}
		// Set up call args for polylog.Event#Msgf() if tt.msgFmt is not emtpy.
		if tt.MsgFmt != "" {
			doneMethodName = "Msgf"
			doneMethodArgs = append(
				doneMethodArgs,
				reflect.ValueOf(tt.MsgFmt),
				reflect.ValueOf(tt.MsgFmtArgs),
			)

			// TODO_IN_THIS_COMMIT: comment...
			if tt.EventMethodName == "" {
				tt.EventMethodName = doneMethodName
			}
		}

		// TODO_TECHDEBT: `strings.Title()` is deprecated. Follow migration guidance in godocs.
		levelMethodStr := strings.Title(levelStr)
		testDesc := fmt.Sprintf("%s().%s()", levelMethodStr, tt.EventMethodName)

		t.Run(testDesc, func(t *testing.T) {
			logger, logOutput := newLoggerAndOutput()

			// TODO_IN_THIS_COMMIT: comment... use reflect to/because...
			logEvent := newEventWithLevel(t, logger, levelStr)
			logEventValue := reflect.ValueOf(logEvent)

			// Append tt.key to polylog.Event#<level>() call args.
			// TODO_IN_THIS_COMMIT: comment
			//if tt.key != "" {
			if strings.HasPrefix(doneMethodName, "Msg") || tt.Key != "" {
				methodArgs = append(methodArgs, reflect.ValueOf(tt.Key))
			}
			// Append tt.value to polylog.Event#<level>() call args.
			if tt.Value != nil {
				methodArgs = append(methodArgs, reflect.ValueOf(tt.Value))
			}

			// E.g.: logger.Debug().Str("str", "str_value").Send()
			//   or: logger.Debug().Bool("bool", true).Msg("msg")
			//   or: logger.Debug().Msgf("meaning of life: %d", 42)
			if tt.EventMethodName != "" {
				logEventValue.
					MethodByName(tt.EventMethodName).
					Call(methodArgs)
			}

			logEventValue.
				MethodByName(doneMethodName).
				Call(doneMethodArgs)

			// Assert that each line contains the expected prefix.
			expectedLevelOutputContains := fmt.Sprintf(
				`"level":"%s"`,
				levelStr,
			)
			require.Contains(t, logOutput.String(), expectedLevelOutputContains)

			// Assert that the log output contains the expected messages. Split the log
			// output into lines and iterate over them.
			require.Contains(t, logOutput.String(), tt.ExpectedOutputContains)

			// Print log output for manual inspection.
			t.Log(logOutput.String())
		})
	}

	levelMethodStr := strings.Title(levelStr)
	funcTestDesc := fmt.Sprintf("%s().Func()", levelMethodStr)
	t.Run(funcTestDesc, func(t *testing.T) {
		// Redirect standard log output to logOutput buffer.

		// TODO RESUME_HERE: !!!!
		// TODO RESUME_HERE: !!!!
		// TODO RESUME_HERE: !!!!
		// TODO RESUME_HERE: !!!!
		// TODO RESUME_HERE: !!!!

		logOutput := new(bytes.Buffer)
		outputOpt := polyzero.WithOutput(logOutput)

		// TODO_IN_THIS_COMMIT: configuration ... debug level for this test
		logger := polyzero.NewLogger(outputOpt)
		// TODO_IN_THIS_COMMIT: comment...
		funcSpy := funcMethodSpy{}
		funcSpy.On("Fn", mock.AnythingOfType("*polyzero.zerologEvent")).Return()

		logger.Debug().Func(funcSpy.Fn).Send()

		// Assert that the Func field contains the expected value.
		// TODO_IN_THIS_COMMIT: add coverage of an zerologEvent which is disabled,
		// asserting that `Fn` is not called!
		funcSpy.AssertCalled(t, "Fn", mock.AnythingOfType("*polyzero.zerologEvent"))
	})
}

func newEventWithLevel(
	t *testing.T,
	logger polylog.Logger,
	levelStr string,
) polylog.Event {
	t.Helper()

	switch levelStr {
	case zerolog.DebugLevel.String():
		return logger.Debug()
	case zerolog.InfoLevel.String():
		return logger.Info()
	case zerolog.WarnLevel.String():
		return logger.Warn()
	case zerolog.ErrorLevel.String():
		return logger.Error()
	default:
		t.Fatalf("level not yet supported: %s", levelStr)
		return nil
	}
}
