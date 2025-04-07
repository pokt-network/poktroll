package signals

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// ExitCode is a global variable that is intended to be used by CLI commands to
// hold the current exit code and subsequently used in ExitWithCodeIfNonZero.
var ExitCode int

// GoOnExitSignal calls the given callback when the process receives an interrupt
// or terminate signal.
func GoOnExitSignal(onInterrupt func()) {
	go func() {
		// Set up sigCh to receive when this process receives an interrupt or
		// terminate signal.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// Block until we receive an interrupt or kill signal (OS-agnostic)
		<-sigCh

		// Call the onInterrupt callback.
		onInterrupt()
	}()
}

// ExitWithCodeIfNonZero is a helper function that is intended to be used as a
// PostRun function for a cobra command. It checks if the exitCode variable is
// non-zero and exits the program with the exitCode value.
func ExitWithCodeIfNonZero(_ *cobra.Command, _ []string) {
	if ExitCode != 0 {
		os.Exit(ExitCode)
	}
}
