//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TODO_TECHDEBT(https://github.com/ignite/cli/issues/3737): We're using a combination
// of `pocketd` (legacy) and `poktrolld` (current) because of an issue of how ignite works.
var (
	// defaultRPCURL used by pocketdBin to run remote commands
	defaultRPCURL = os.Getenv("POCKET_NODE")
	// defaultRPCPort is the default RPC port that pocketd listens on
	defaultRPCPort = 36657
	// defaultRPCHost is the default RPC host that pocketd listens on
	defaultRPCHost = "127.0.0.1"
	// defaultHome is the default home directory for pocketd
	defaultHome = os.Getenv("POKTROLLD_HOME")
	// defaultAppGateServerURL used by curl commands to send relay requests
	defaultAppGateServerURL = os.Getenv("APPGATE_SERVER")
	// defaultDebugOutput provides verbose output on manipulations with binaries (cli command, stdout, stderr)
	defaultDebugOutput = os.Getenv("E2E_DEBUG_OUTPUT")
)

func init() {
	if defaultRPCURL == "" {
		defaultRPCURL = fmt.Sprintf("tcp://%s:%d", defaultRPCHost, defaultRPCPort)
	}
	if defaultHome == "" {
		defaultHome = "../../localnet/poktrolld"
	}
}

// commandResult combines the stdout, stderr, and err of an operation
type commandResult struct {
	Command string // the command that was executed
	Stdout  string // standard output
	Stderr  string // standard error
	Err     error  // execution error, if any
}

// PocketClient is a single function interface for interacting with a node
type PocketClient interface {
	RunCommand(args ...string) (*commandResult, error)
	RunCommandOnHost(rpcUrl string, args ...string) (*commandResult, error)
	RunCurl(rpcUrl, service, data string, args ...string) (*commandResult, error)
}

// Ensure that pocketdBin struct fulfills PocketClient
var _ PocketClient = (*pocketdBin)(nil)

// pocketdBin holds the reults of the last command that was run
type pocketdBin struct {
	result *commandResult // stores the result of the last command that was run
}

// RunCommand runs a command on the local machine using the pocketd binary
func (p *pocketdBin) RunCommand(args ...string) (*commandResult, error) {
	return p.runPocketCmd(args...)
}

// RunCommandOnHost runs a command on specified host with the given args
func (p *pocketdBin) RunCommandOnHost(rpcUrl string, args ...string) (*commandResult, error) {
	if rpcUrl == "" {
		rpcUrl = defaultRPCURL
	}
	args = append(args, "--node", rpcUrl)
	return p.runPocketCmd(args...)
}

// RunCurl runs a curl command on the local machine
func (p *pocketdBin) RunCurl(rpcUrl, service, data string, args ...string) (*commandResult, error) {
	if rpcUrl == "" {
		rpcUrl = defaultAppGateServerURL
	}
	return p.runCurlPostCmd(rpcUrl, service, data, args...)
}

// runPocketCmd is a helper to run a command using the local pocketd binary with the flags provided
func (p *pocketdBin) runPocketCmd(args ...string) (*commandResult, error) {
	base := []string{"--home", defaultHome}
	args = append(base, args...)
	commandStr := "poktrolld " + strings.Join(args, " ") // Create a string representation of the command
	cmd := exec.Command("poktrolld", args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	r := &commandResult{
		Command: commandStr, // Set the command string
		Stdout:  stdoutBuf.String(),
		Stderr:  stderrBuf.String(),
		Err:     err,
	}
	p.result = r

	if err != nil {
		// Include the command executed in the error message for context
		err = fmt.Errorf("error running command [%s]: %v, stderr: %s", commandStr, err, stderrBuf.String())
	}

	if defaultDebugOutput == "true" {
		fmt.Printf("%#v\n", r)
	}

	return r, err
}

// runCurlPostCmd is a helper to run a command using the local pocketd binary with the flags provided
func (p *pocketdBin) runCurlPostCmd(rpcUrl string, service string, data string, args ...string) (*commandResult, error) {
	dataStr := fmt.Sprintf("%s", data)
	urlStr := fmt.Sprintf("%s/%s", rpcUrl, service)
	base := []string{
		"-v",         // verbose output
		"-sS",        // silent with error
		"-X", "POST", // HTTP method
		"-H", "Content-Type: application/json", // HTTP headers
		"--data", dataStr, urlStr, // POST data
	}
	args = append(base, args...)
	commandStr := "curl " + strings.Join(args, " ") // Create a string representation of the command
	cmd := exec.Command("curl", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	r := &commandResult{
		Command: commandStr, // Set the command string
		Stdout:  stdoutBuf.String(),
		Stderr:  stderrBuf.String(),
		Err:     err,
	}
	p.result = r
	fmt.Println("OLSHIT", cmd, r)

	if defaultDebugOutput == "true" {
		fmt.Printf("%#v\n", r)
	}

	if err != nil {
		// Include the command executed in the error message for context
		err = fmt.Errorf("error running command [%s]: %v, stderr: %s", commandStr, err, stderrBuf.String())
	}

	return r, err
}
