package signals

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const shutDownTimeout = 30 * time.Second

// ExitCode is a global variable that is intended to be used by CLI commands to
// hold the current exit code and subsequently used in ExitWithCodeIfNonZero.
var ExitCode int

// ExitWithCodeIfNonZero is a helper function that is intended to be used as a PostRun function for a cobra command.
// It checks if the exitCode variable is non-zero and exits the program with the global ExitCode value.
func ExitWithCodeIfNonZero(_ *cobra.Command, _ []string) {
	if ExitCode != 0 {
		os.Exit(ExitCode)
	}
}

// GoOnExitSignal calls the given callback when the process receives an interrupt or terminate signal.
// It sets up a goroutine that listens for OS signals and invokes the callback
func GoOnExitSignal(logger polylog.Logger, onInterrupt func()) {
	go func() {
		// Set up sigCh to receive when this process receives an interrupt or
		// terminate signal.
		// Use a buffered channel with large capacity to prevent signal loss
		sigCh := make(chan os.Signal, 5)

		// Register the signals we want to listen for.
		// DEV_NOTE: SIGKILL cannot be trapped, so we don't listen for it.
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGABRT)

		// Block until we receive an interrupt or kill signal (OS-agnostic)
		sig := <-sigCh
		logger.Info().Msgf("🔚 Received signal %s, starting graceful shutdown...", sig)

		// Create a channel to track shutdown completion
		done := make(chan struct{})

		// Start the graceful shutdown in a goroutine
		go func() {
			// Call the onInterrupt callback.
			onInterrupt()
			close(done)
		}()

		// Wait for either completion or another signal or timeout
		select {
		case <-done:
			logger.Info().Msg("✅ Graceful shutdown completed successfully.")
			return
		case sig := <-sigCh:
			logger.Warn().Msgf("⚠️ Received another signal %s during shutdown, 🗡️ exiting immediately.", sig)
			// Exit immediately if another signal is received during shutdown
			os.Exit(130) // UNIX convention, use 128 + 2 to indicate a double interrupt (SIGINT)
		case <-time.After(shutDownTimeout):
			logger.Warn().Msgf("⌛ Graceful shutdown timed out after %s, 🗡️ exiting immediately.", shutDownTimeout)
			os.Exit(1) // Exit immediately if the shutdown takes too long
		}
	}()
}
