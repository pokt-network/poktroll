package polyzero_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
)

var (
	expectedTime                   = time.Now()
	expectedTimestampLayout        = "2006-01-02T15:04:05-07:00"
	expectedTimestampEventContains = fmt.Sprintf(`"time":"%s"`, expectedTime.Format(expectedTimestampLayout))
	expectedTimeEventContains      = fmt.Sprintf(`"Time":"%s"`, expectedTime.Format(expectedTimestampLayout))
	expectedDuration               = time.Millisecond + (250 * time.Nanosecond)                   // 1000250
	expectedDurationString         = expectedDuration.String()[:len(expectedDuration.String())-2] // 1.00025
	expectedDurationEventContains  = fmt.Sprintf(`"Dur":%s`, expectedDurationString)
)

func TestZerologPolyLogger_AllLevels_AllEventMethods(t *testing.T) {
	tests := []testpolylog.EventMethodsTest{
		{
			Msg:                    "Msg",
			ExpectedOutputContains: "Msg",
		},
		{
			MsgFmt:                 "%s",
			MsgFmtArgs:             []any{"Msgf"},
			ExpectedOutputContains: "Msgf",
		},
		{
			Key:                    "Str",
			Value:                  "str_value",
			ExpectedOutputContains: `"Str":"str_value"`,
		},
		{
			Key:                    "Bool",
			Value:                  true,
			ExpectedOutputContains: `"Bool":true`,
		},
		{
			Key:                    "Int",
			Value:                  int(42),
			ExpectedOutputContains: `"Int":42`,
		},
		{
			Key:                    "Int8",
			Value:                  int8(42),
			ExpectedOutputContains: `"Int8":42`,
		},
		{
			Key:                    "Int16",
			Value:                  int16(42),
			ExpectedOutputContains: `"Int16":42`,
		},
		{
			Key:                    "Int32",
			Value:                  int32(42),
			ExpectedOutputContains: `"Int32":42`,
		},
		{
			Key:                    "Int64",
			Value:                  int64(42),
			ExpectedOutputContains: `"Int64":42`,
		},
		{
			Key:                    "Uint",
			Value:                  uint(42),
			ExpectedOutputContains: `"Uint":42`,
		},
		{
			Key:                    "Uint8",
			Value:                  uint8(42),
			ExpectedOutputContains: `"Uint8":42`,
		},
		{
			Key:                    "Uint16",
			Value:                  uint16(42),
			ExpectedOutputContains: `"Uint16":42`,
		},
		{
			Key:                    "Uint32",
			Value:                  uint32(42),
			ExpectedOutputContains: `"Uint32":42`,
		},
		{
			Key:                    "Uint64",
			Value:                  uint64(42),
			ExpectedOutputContains: `"Uint64":42`,
		},
		{
			Key:                    "Float32",
			Value:                  float32(420.69),
			ExpectedOutputContains: `"Float32":420.69`,
		},
		{
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
		{
			Key:                    "Time",
			Value:                  expectedTime,
			ExpectedOutputContains: expectedTimeEventContains,
		},
		{
			Key:                    "Dur",
			Value:                  expectedDuration,
			ExpectedOutputContains: expectedDurationEventContains,
		},
	}

	levels := []zerolog.Level{
		zerolog.DebugLevel,
		zerolog.InfoLevel,
		zerolog.WarnLevel,
		zerolog.ErrorLevel,
	}

	// TODO_IN_THIS_COMMIT: comment...
	for _, level := range levels {
		testpolylog.RunEventMethodTests(t, level.String(), tests, newLoggerFn)
	}
}

func newLoggerFn() (polylog.Logger, *bytes.Buffer) {
	// Redirect standard log output to logOutput buffer.
	logOutput := new(bytes.Buffer)
	outputOpt := polyzero.WithOutput(logOutput)

	// TODO_IN_THIS_COMMIT: configuration ... debug level for this test
	logger := polyzero.NewLogger(outputOpt)

	return logger, logOutput
}

// TODO_TEST: that exactly all expected levels log at each level.

// TODO_TEST: #Enabled() and #Discard()
