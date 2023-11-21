package polyzero_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	expectedTime           = time.Now()
	expectedDuration       = time.Millisecond + (250 * time.Nanosecond)                   // 1000250
	expectedDurationString = expectedDuration.String()[:len(expectedDuration.String())-2] // 1.00025
	expectedMsgs           = []string{
		"Msg",
		"Msgf",
		`"Str":"str_value"`,
		`"Bool":true`,
		`"Int":42`,
		`"Int8":42`,
		`"Int16":42`,
		`"Int32":42`,
		`"Int64":42`,
		`"Uint":42`,
		`"Uint8":42`,
		`"Uint16":42`,
		`"Uint32":42`,
		`"Uint64":42`,
		`"Float32":420.69`,
		`"Float64":420.69`,
		`"error":"42"`,
		//`"Func":"0x"`,
		fmt.Sprintf(`"time":"%s"`, expectedTime.Format(expectedTimestampLayout)),
		fmt.Sprintf(`"Time":"%s"`, expectedTime.Format(expectedTimestampLayout)),
		fmt.Sprintf(`"Dur":%s`, expectedDurationString),
		//`"Fields":"map[key1:value1 key2:value2]"`,
		"", // polylog.Event#Func() prints a line with the level only: `{"level":"debug"}`, this is zerolog behavior.
	}
	expectedTimestampLayout = "2006-01-02T15:04:05-07:00"
)

// TODO_IN_THIS_COMMIT: comment...
type funcMethodSpy struct{ mock.Mock }

// TODO_IN_THIS_COMMIT: comment...
func (m *funcMethodSpy) Fn(event polylog.Event) {
	m.Called(event)
}

func TestZerologULogger(t *testing.T) {
	// Redirect standard log output to logOutput buffer.
	logOutput := new(bytes.Buffer)
	outputOpt := polyzero.WithOutput(logOutput)

	// TODO_IN_THIS_COMMIT: configuration ... debug level for this test
	logger := polyzero.NewUniversalLogger(outputOpt)

	logger.Debug().Msg("Msg")
	logger.Debug().Msgf("%s", "Msgf")
	logger.Debug().Str("Str", "str_value").Send()
	logger.Debug().Bool("Bool", true).Send()
	logger.Debug().Int("Int", 42).Send()
	logger.Debug().Int8("Int8", 42).Send()
	logger.Debug().Int16("Int16", 42).Send()
	logger.Debug().Int32("Int32", 42).Send()
	logger.Debug().Int64("Int64", 42).Send()
	logger.Debug().Uint("Uint", 42).Send()
	logger.Debug().Uint8("Uint8", 42).Send()
	logger.Debug().Uint16("Uint16", 42).Send()
	logger.Debug().Uint32("Uint32", 42).Send()
	logger.Debug().Uint64("Uint64", 42).Send()
	logger.Debug().Float32("Float32", 420.69).Send()
	logger.Debug().Float64("Float64", 420.69).Send()
	logger.Debug().Err(fmt.Errorf("%d", 42)).Send()
	logger.Debug().Timestamp().Send()
	logger.Debug().Time("Time", expectedTime).Send()
	logger.Debug().Dur("Dur", expectedDuration).Send()
	//logger.Debug().Fields(map[string]string{
	//	"key1": "value1",
	//	"key2": "value2",
	//}).Send()

	// TODO_IN_THIS_COMMIT: comment...
	funcSpy := funcMethodSpy{}
	funcSpy.On("Fn", mock.AnythingOfType("*polyzero.zerologEvent")).Return()

	logger.Debug().Func(funcSpy.Fn).Send()

	// TODO:
	// .Enabled()
	// .Discard()

	// Assert that the log output contains the expected messages. Split the log
	// output into lines and iterate over them.
	lines := strings.Split(logOutput.String(), "\n")
	lines = lines[:len(lines)-1] // Remove last empty line.
	// Assert that the log output contains the expected number of lines.
	// Intentionally not using `require` to provide additional error context.
	assert.Lenf(
		t, lines,
		len(expectedMsgs),
		"log output should contain %d lines, got: %d",
		len(expectedMsgs), len(lines),
	)

	for lineIdx, line := range lines {
		// Assert that each line contains the expected prefix.
		require.Contains(t, line, `"level":"debug"`)

		expectedMsg := expectedMsgs[lineIdx]
		require.Contains(t, line, expectedMsg)
	}

	// Assert that the Func field contains the expected value.
	// TODO_IN_THIS_COMMIT: add coverage of an zerologEvent which is disabled,
	// asserting that `Fn` is not called!
	funcSpy.AssertCalled(t, "Fn", mock.AnythingOfType("*polyzero.zerologEvent"))
}
