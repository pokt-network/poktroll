package signals

import (
	"os"
	"os/signal"
	"syscall"
)

// GoOnExitSignal calls the given callback when the process receives an interrupt
// or kill signal.
func GoOnExitSignal(onInterrupt func()) {
	go func() {
		// Set up sigCh to receive when this process receives an interrupt or
		// kill signal.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// Block until we receive an interrupt or kill signal (OS-agnostic)
		<-sigCh

		// Call the onInterrupt callback.
		onInterrupt()
	}()
}
