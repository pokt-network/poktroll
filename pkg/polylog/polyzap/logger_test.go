package polyzap_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzap"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
)

//var (
//	expectedTime     = time.Now()
//	expectedDuration = time.Millisecond + (250 * time.Nanosecond) // 1000250
//	expectedMsgs     = []string{
//		"Msg",
//		"Msgf",
//		`"Str":"str_value"`,
//		`"Bool":true`,
//		`"Int":42`,
//		`"Int8":42`,
//		`"Int16":42`,
//		`"Int32":42`,
//		`"Int64":42`,
//		`"Uint":42`,
//		`"Uint8":42`,
//		`"Uint16":42`,
//		`"Uint32":42`,
//		`"Uint64":42`,
//		`"Float32":420.69`,
//		`"Float64":420.69`,
//		`"error":"42"`,
//		fmt.Sprintf(`"ts":%d.`, expectedTime.Unix()),
//		fmt.Sprintf(`"Time":%d.`, expectedTime.Unix()),
//		fmt.Sprintf(`"Dur":%f`, expectedDuration.Seconds()),
//		//`"Fields":"map[key1:value1 key2:value2]"`,
//		"", // polylog.Event#Func() prints a line with the level only: `{"level":"debug"}`, this is zerolog behavior.
//	}
//	// TODO_THIS_COMMIT: should configurable via an option:
//)

var (
	expectedTime                   = time.Now()
	expectedTimestampEventContains = fmt.Sprintf(`"ts":%d.`, expectedTime.Unix())
	expectedTimeEventContains      = fmt.Sprintf(`"Time":%d.`, expectedTime.Unix())
	expectedDuration               = time.Millisecond + (250 * time.Nanosecond) // 1000250
	expectedDurationEventContains  = fmt.Sprintf(`"Dur":%f`, expectedDuration.Seconds())
)

func TestZapPolyLogger_AllLevels_AllEventMethods(t *testing.T) {
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
	opts := []polylog.LoggerOption{
		polyzap.WithOutput(logOutput),
		polyzap.WithLevel(zapcore.DebugLevel),
	}

	// TODO_IN_THIS_COMMIT: configuration ... debug level for this test
	logger := polyzap.NewLogger(opts...)

	return logger, logOutput
}

// TODO_TEST: that exactly all expected levels log at each level.

// TODO_TEST: #Enabled() and #Discard()
