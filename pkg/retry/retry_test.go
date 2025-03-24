package retry_test

/* TODO_TECHDEBT: improve this test:
- fix race condition around the logOutput buffer
- factor our common setup and assertion code
- drive out flakiness
- improve comments
*/

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/pkg/polylog/polyzero"
	_ "github.com/pokt-network/pocket/pkg/polylog/polyzero"
	"github.com/pokt-network/pocket/pkg/retry"
)

var testErr = fmt.Errorf("test error")

// TestOnError verifies the behavior of the OnError function in the retry package.
// It ensures that the function correctly retries a failing operation for a specified
// number of times with the expected delay between retries.
func TestOnError(t *testing.T) {
	t.Skip("TODO_TECHDEBT: this test should pass but contains a race condition around the logOutput buffer")

	// Setting up the test variables.
	var (
		// logOutput captures the log output for verification of logged messages.
		logOutput = new(bytes.Buffer)
		// expectedRetryDelay is the duration we expect between retries.
		expectedRetryDelay = time.Millisecond
		// expectedRetryLimit is the maximum number of retries the test expects.
		expectedRetryLimit = 5
		// retryResetTimeout is the duration after which the retry count should reset.
		retryResetTimeout = time.Second
		// testFnCallCount keeps track of how many times the test function is called.
		testFnCallCount int32
		// testFnCallTimeCh is a channel receives a time.Time each when the test
		// function is called.
		testFnCallTimeCh = make(chan time.Time, expectedRetryLimit)
		ctx              = context.Background()
	)

	// Redirect the log output for verification later
	logOpt := polyzero.WithOutput(logOutput)
	// Construct a new polylog logger & attach it to the context.
	ctx = polyzero.NewLogger(logOpt).WithContext(ctx)

	// Define testFn, a function that simulates a failing operation and logs its invocation times.
	testFn := func() chan error {
		// Record the current time to track the delay between retries.
		testFnCallTimeCh <- time.Now()

		// Create a channel to return an error, simulating a failing operation.
		errCh := make(chan error, 1)
		errCh <- testErr

		// Increment the call count safely across goroutine boundaries.
		atomic.AddInt32(&testFnCallCount, 1)

		return errCh
	}

	// Create a channel to receive the error result from the OnError function.
	retryOnErrorErrCh := make(chan error, 1)

	// Start the OnError function in a separate goroutine, simulating concurrent operation.
	go func() {
		// Call the OnError function with the test parameters and function.
		retryOnErrorErrCh <- retry.OnError(
			ctx,
			expectedRetryLimit,
			expectedRetryDelay,
			retryResetTimeout,
			"TestOnError",
			testFn,
		)
	}()

	// Calculate the total expected time for all retries to complete.
	totalExpectedDelay := expectedRetryDelay * time.Duration(expectedRetryLimit)
	// Wait for the OnError function to execute and retry the expected number of times.
	time.Sleep(totalExpectedDelay + 100*time.Millisecond)

	// Verify that the test function was called the expected number of times.
	require.Equal(t, expectedRetryLimit, int(testFnCallCount), "Test function was not called the expected number of times")

	// Verify the delay between retries of the test function.
	var prevCallTime time.Time
	for i := 0; i < expectedRetryLimit; i++ {
		// Retrieve the next function call time from the channel.
		nextCallTime, ok := <-testFnCallTimeCh
		if !ok {
			t.Fatalf("expected %d calls to testFn, but channel closed after %d", expectedRetryLimit, i)
		}

		// For all calls after the first, check that the delay since the previous call meets expectations.
		if i != 0 {
			actualRetryDelay := nextCallTime.Sub(prevCallTime)
			require.GreaterOrEqual(t, actualRetryDelay, expectedRetryDelay, "Retry delay was less than expected")
		}

		// Update prevCallTime for the next iteration.
		prevCallTime = nextCallTime
	}

	// Verify that the OnError function returned the expected error.
	select {
	case err := <-retryOnErrorErrCh:
		require.ErrorIs(t, err, testErr, "OnError did not return the expected error")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected error from OnError, but none received")
	}

	// Verify the error messages logged during the retries.
	expectedErrLine := `"error":"test error"`
	trimmedLogOutput := strings.Trim(logOutput.String(), "\n")
	logOutputLines := strings.Split(trimmedLogOutput, "\n")
	require.Lenf(t, logOutputLines, expectedRetryLimit, "unexpected number of log lines")
	for _, line := range logOutputLines {
		require.Contains(t, line, expectedErrLine, "log line does not contain the expected prefix")
	}
}

// TODO_TECHDEBT: assert that the retry loop exits when the context is closed.
func TestOnError_ExitsWhenCtxCloses(t *testing.T) {
	t.SkipNow()
}

func TestOnError_ExitsWhenErrChCloses(t *testing.T) {
	t.Skip("TODO_TECHDEBT: this test should pass but contains a race condition around the logOutput buffer")

	// Setup test variables and log capture
	var (
		logOutput          = new(bytes.Buffer)
		testFnCallCount    int32
		expectedRetryDelay = time.Millisecond
		expectedRetryLimit = 3
		retryLimit         = 5
		retryResetTimeout  = time.Second
		testFnCallTimeCh   = make(chan time.Time, expectedRetryLimit)
		ctx                = context.Background()
	)

	// Redirect the log output for verification later
	logOpt := polyzero.WithOutput(logOutput)
	// Construct a new polylog logger & attach it to the context.
	ctx = polyzero.NewLogger(logOpt).WithContext(ctx)

	// Define the test function that simulates an error and counts its invocations
	testFn := func() chan error {
		atomic.AddInt32(&testFnCallCount, 1) // Increment the invocation count atomically
		testFnCallTimeCh <- time.Now()       // Track the invocation time

		errCh := make(chan error, 1)
		if atomic.LoadInt32(&testFnCallCount) >= int32(expectedRetryLimit) {
			close(errCh)
			return errCh
		}

		errCh <- testErr
		return errCh
	}

	retryOnErrorErrCh := make(chan error, 1)
	// Spawn a goroutine to test the OnError function
	go func() {
		retryOnErrorErrCh <- retry.OnError(
			ctx,
			retryLimit,
			expectedRetryDelay,
			retryResetTimeout,
			"TestOnError_ExitsWhenErrChCloses",
			testFn,
		)
	}()

	// Wait for the OnError function to execute and retry the expected number of times
	totalExpectedDelay := expectedRetryDelay * time.Duration(expectedRetryLimit)
	time.Sleep(totalExpectedDelay + 100*time.Millisecond)

	// Assert that the test function was called the expected number of times
	require.Equal(t, expectedRetryLimit, int(testFnCallCount))

	// Assert that the retry delay between function calls matches the expected delay
	var prevCallTime = new(time.Time)
	for i := 0; i < expectedRetryLimit; i++ {
		select {
		case nextCallTime := <-testFnCallTimeCh:
			if i != 0 {
				actualRetryDelay := nextCallTime.Sub(*prevCallTime)
				require.GreaterOrEqual(t, actualRetryDelay, expectedRetryDelay)
			}

			*prevCallTime = nextCallTime
		default:
			t.Fatalf(
				"expected %d calls to testFn, but only received %d",
				expectedRetryLimit, i+1,
			)
		}
	}

	select {
	case err := <-retryOnErrorErrCh:
		require.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected error from OnError, but none received")
	}

	// Verify the logged error messages
	var (
		logOutputLines  = strings.Split(strings.Trim(logOutput.String(), "\n"), "\n")
		errorLines      = logOutputLines[:len(logOutputLines)-1]
		warnLine        = logOutputLines[len(logOutputLines)-1]
		expectedWarnMsg = "error channel closed, will no longer retry on error"
		expectedErrMsg  = `"error":"test error"`
	)

	require.Lenf(
		t, logOutputLines,
		expectedRetryLimit,
		"expected %d log lines, got %d",
		expectedRetryLimit, len(logOutputLines),
	)
	for _, line := range errorLines {
		require.Contains(t, line, expectedErrMsg)
	}
	require.Contains(t, warnLine, expectedWarnMsg)
}

// assert that retryCount resets on success
func TestOnError_RetryCountResetTimeout(t *testing.T) {
	t.Skip("TODO_TECHDEBT: this test should pass but contains a race condition around the logOutput buffer")

	// Setup test variables and log capture
	var (
		logOutput          = new(bytes.Buffer)
		testFnCallCount    int32
		expectedRetryDelay = time.Millisecond
		expectedRetryLimit = 9
		retryLimit         = 5
		retryResetTimeout  = 3 * time.Millisecond
		testFnCallTimeCh   = make(chan time.Time, expectedRetryLimit)
		ctx                = context.Background()
	)

	// Redirect the log output for verification later
	logOpt := polyzero.WithOutput(logOutput)
	// Construct a new polylog logger & attach it to the context.
	ctx = polyzero.NewLogger(logOpt).WithContext(ctx)

	// Define the test function that simulates an error and counts its invocations
	testFn := func() chan error {
		// Track the invocation time
		testFnCallTimeCh <- time.Now()

		errCh := make(chan error, 1)

		count := atomic.LoadInt32(&testFnCallCount)
		if count == int32(retryLimit) {
			go func() {
				time.Sleep(retryResetTimeout)
				errCh <- testErr
			}()
		} else {
			errCh <- testErr
		}

		// Increment the invocation count atomically
		atomic.AddInt32(&testFnCallCount, 1)
		return errCh
	}

	retryOnErrorErrCh := make(chan error, 1)
	// Spawn a goroutine to test the OnError function
	go func() {
		retryOnErrorErrCh <- retry.OnError(
			ctx,
			retryLimit,
			expectedRetryDelay,
			retryResetTimeout,
			"TestOnError",
			testFn,
		)
	}()

	// Wait for the OnError function to execute and retry the expected number of times
	totalExpectedDelay := expectedRetryDelay * time.Duration(expectedRetryLimit)
	time.Sleep(totalExpectedDelay + 100*time.Millisecond)

	// Assert that the test function was called the expected number of times
	require.Equal(t, expectedRetryLimit, int(testFnCallCount))

	// Assert that the retry delay between function calls matches the expected delay
	var prevCallTime = new(time.Time)
	for i := 0; i < expectedRetryLimit; i++ {
		select {
		case nextCallTime := <-testFnCallTimeCh:
			if i != 0 {
				actualRetryDelay := nextCallTime.Sub(*prevCallTime)
				require.GreaterOrEqual(t, actualRetryDelay, expectedRetryDelay)
			}

			*prevCallTime = nextCallTime
		default:
			t.Fatalf(
				"expected %d calls to testFn, but only received %d",
				expectedRetryLimit, i+1,
			)
		}
	}

	// Verify the logged error messages
	var (
		logOutputLines = strings.Split(strings.Trim(logOutput.String(), "\n"), "\n")
		expectedErrMsg = `"error":"test error"`
	)

	select {
	case err := <-retryOnErrorErrCh:
		require.ErrorIs(t, err, testErr)
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected error from OnError, but none received")
	}

	require.Lenf(
		t, logOutputLines,
		expectedRetryLimit-1,
		"expected %d log lines, got %d",
		expectedRetryLimit-1, len(logOutputLines),
	)
	for _, line := range logOutputLines {
		require.Contains(t, line, expectedErrMsg)
	}
}

// assert that a negative retry limit continually calls workFn
func TestOnError_NegativeRetryLimit(t *testing.T) {
	t.Skip("TODO_TECHDEBT(@bryanchriswhite): this test should pass but contains a race condition around the logOutput buffer")

	// Setup test variables and log capture
	var (
		logOutput          = new(bytes.Buffer)
		testFnCallCount    int32
		minimumCallCount   = 90
		expectedRetryDelay = time.Millisecond
		retryLimit         = -1
		retryResetTimeout  = 3 * time.Millisecond
		testFnCallTimeCh   = make(chan time.Time, minimumCallCount)
		ctx                = context.Background()
	)

	// Redirect the log output for verification later
	logOpt := polyzero.WithOutput(logOutput)
	// Construct a new polylog logger & attach it to the context.
	ctx = polyzero.NewLogger(logOpt).WithContext(ctx)

	// Define the test function that simulates an error and counts its invocations
	testFn := func() chan error {
		// Track the invocation time
		testFnCallTimeCh <- time.Now()

		errCh := make(chan error, 1)

		// Increment the invocation count atomically
		count := atomic.AddInt32(&testFnCallCount, 1) - 1
		if count == int32(retryLimit) {
			go func() {
				time.Sleep(retryResetTimeout)
				errCh <- testErr
			}()
		} else {
			errCh <- testErr
		}
		return errCh
	}

	retryOnErrorErrCh := make(chan error, 1)
	// Spawn a goroutine to test the OnError function
	go func() {
		retryOnErrorErrCh <- retry.OnError(
			ctx,
			retryLimit,
			expectedRetryDelay,
			retryResetTimeout,
			"TestOnError",
			testFn,
		)
	}()

	// Wait for the OnError function to execute and retry the expected number of times
	totalExpectedDelay := expectedRetryDelay * time.Duration(minimumCallCount)
	time.Sleep(totalExpectedDelay + 100*time.Millisecond)

	// Assert that the test function was called the expected number of times
	require.GreaterOrEqual(t, minimumCallCount, int(testFnCallCount))

	// Assert that the retry delay between function calls matches the expected delay
	var prevCallTime = new(time.Time)
	for i := 0; i < minimumCallCount; i++ {
		select {
		case nextCallTime := <-testFnCallTimeCh:
			if i != 0 {
				actualRetryDelay := nextCallTime.Sub(*prevCallTime)
				require.GreaterOrEqual(t, actualRetryDelay, expectedRetryDelay)
			}

			*prevCallTime = nextCallTime
		default:
			t.Fatalf(
				"expected %d calls to testFn, but only received %d",
				minimumCallCount, i+1,
			)
		}
	}

	// Verify the logged error messages
	var (
		logOutputLines = strings.Split(strings.Trim(logOutput.String(), "\n"), "\n")
		expectedErrMsg = `"error":"test error"`
	)

	require.Lenf(
		t, logOutputLines,
		minimumCallCount-1,
		"expected %d log lines, got %d",
		minimumCallCount-1, len(logOutputLines),
	)
	for _, line := range logOutputLines {
		require.Contains(t, line, expectedErrMsg)
	}
}
