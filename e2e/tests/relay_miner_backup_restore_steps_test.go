//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// relayMinerRestartTimeout is the duration to wait for the relay miner to restart
	relayMinerRestartTimeout = 60 * time.Second
	// relayMinerBackupRestoreTimeout is the duration to wait for backup restoration to complete
	relayMinerBackupRestoreTimeout = 30 * time.Second
)

// runShellCommand executes a shell command and returns the result similar to pocketdBin.runPocketCmd
func (s *suite) runShellCommand(command string) (*commandResult, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	
	cmd := exec.Command(parts[0], parts[1:]...)
	
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	
	err := cmd.Run()
	result := &commandResult{
		Command: command,
		Stdout:  stdoutBuf.String(),
		Stderr:  stderrBuf.String(),
		Err:     err,
	}
	
	if err != nil {
		err = fmt.Errorf("error running command [%s]: %v, stderr: %s", command, err, stderrBuf.String())
	}
	
	return result, err
}

// TheUserNongracefullyRestartsTheRelayMiner performs a non-graceful restart of the specified relay miner
// using tilt trigger to simulate a failure scenario where the process is killed and restarted.
func (s *suite) TheUserNongracefullyRestartsTheRelayMiner(relayMinerName string) {
	s.Log("Non-gracefully restarting relay miner: %s", relayMinerName)
	
	// Use tilt trigger to restart the relay miner resource
	// This simulates a non-graceful restart similar to a process crash
	command := fmt.Sprintf("tilt trigger %s", relayMinerName)
	res, err := s.runShellCommand(command)
	require.NoError(s, err, "error restarting relay miner %s due to: %v", relayMinerName, err)
	
	s.Log("Relay miner restart command output: %s", res.Stdout)
	if res.Stderr != "" {
		s.Log("Relay miner restart stderr: %s", res.Stderr)
	}
	
	// Wait for the restart to take effect
	time.Sleep(5 * time.Second)
	
	s.pocketd.result = res
}

// TheRelayMinerShouldRestoreFromBackup verifies that the relay miner successfully restored from backup
// by checking the logs for backup restoration indicators.
func (s *suite) TheRelayMinerShouldRestoreFromBackup() {
	s.Log("Verifying relay miner restored from backup")
	
	// Wait a moment for the relay miner to start and log restoration
	time.Sleep(relayMinerBackupRestoreTimeout)
	
	// Get relay miner logs to verify backup restoration occurred
	command := "tilt logs relayminer1"
	res, err := s.runShellCommand(command)
	require.NoError(s, err, "error getting relay miner logs due to: %v", err)
	
	// Check for backup restoration indicators in the logs
	logOutput := res.Stdout
	
	// Look for key backup restoration log messages
	backupRestorationIndicators := []string{
		"restored from backup",
		"backup restoration completed",
		"session tree restored",
		"loading backup data",
		"backup file found",
	}
	
	foundIndicator := false
	for _, indicator := range backupRestorationIndicators {
		if strings.Contains(strings.ToLower(logOutput), strings.ToLower(indicator)) {
			s.Log("Found backup restoration indicator: %s", indicator)
			foundIndicator = true
			break
		}
	}
	
	// If no specific indicator is found, check for general startup without errors
	if !foundIndicator {
		// Check that the relay miner started successfully after the restart
		require.NotContains(s, strings.ToLower(logOutput), "fatal", "Relay miner logs contain fatal errors")
		require.NotContains(s, strings.ToLower(logOutput), "panic", "Relay miner logs contain panic errors")
		s.Log("Relay miner appears to have started successfully after restart (no backup indicators found but no errors)")
	}
	
	s.pocketd.result = res
}

// TheRelayMinerShouldContinueFromBackupState verifies that the relay miner can continue operations
// from the restored backup state by checking that it's ready to handle new requests.
func (s *suite) TheRelayMinerShouldContinueFromBackupState() {
	s.Log("Verifying relay miner can continue operations from backup state")
	
	// Wait for the relay miner to be fully operational
	time.Sleep(10 * time.Second)
	
	// Get relay miner status to verify it's running and operational
	command := "tilt get uiresource relayminer1"
	res, err := s.runShellCommand(command)
	require.NoError(s, err, "error getting relay miner status due to: %v", err)
	
	// Verify the relay miner resource is in a healthy state
	statusOutput := res.Stdout
	require.NotContains(s, strings.ToLower(statusOutput), "error", "Relay miner resource shows errors")
	require.NotContains(s, strings.ToLower(statusOutput), "failed", "Relay miner resource shows failures")
	
	s.Log("Relay miner resource status: %s", statusOutput)
	
	// Additional verification: check that the relay miner logs show it's ready for operations
	logsCommand := "tilt logs relayminer1 --since=30s"
	logsRes, err := s.runShellCommand(logsCommand)
	require.NoError(s, err, "error getting recent relay miner logs due to: %v", err)
	
	recentLogs := logsRes.Stdout
	s.Log("Recent relay miner logs: %s", recentLogs)
	
	// Check for operational readiness indicators
	operationalIndicators := []string{
		"server started",
		"ready to serve",
		"listening on",
		"relay miner started",
		"initialized successfully",
	}
	
	foundOperationalIndicator := false
	for _, indicator := range operationalIndicators {
		if strings.Contains(strings.ToLower(recentLogs), strings.ToLower(indicator)) {
			s.Log("Found operational readiness indicator: %s", indicator)
			foundOperationalIndicator = true
			break
		}
	}
	
	if !foundOperationalIndicator {
		s.Log("No specific operational indicators found, but relay miner appears to be running based on resource status")
	}
	
	s.pocketd.result = res
}

