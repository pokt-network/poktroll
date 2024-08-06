//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TODO_TECHDEBT(https://github.com/ignite/cli/issues/3737): We're using a combination
// of `pocketd` (legacy) and `poktrolld` (current) because of an issue of how ignite works.
var (
	// defaultRPCURL used by pocketdBin to run remote commands
	defaultRPCURL = os.Getenv("POCKET_NODE")
	// defaultRPCPort is the default RPC port that pocketd listens on
	defaultRPCPort = 26657
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
	RunCurl(rpcUrl, service, path, data string, args ...string) (*commandResult, error)
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

// RunCommandOnHost runs a command on specified host with the given args.
// If rpcUrl is an empty string, the defaultRPCURL is used.
// If rpcUrl is "local", the command is run on the local machine and the `--node` flag is omitted.
func (p *pocketdBin) RunCommandOnHost(rpcUrl string, args ...string) (*commandResult, error) {
	if rpcUrl == "" {
		rpcUrl = defaultRPCURL
	}
	if rpcUrl != "local" {
		args = append(args, "--node", rpcUrl)
	}
	return p.runPocketCmd(args...)
}

// RunCommandOnHostWithRetry is the same as RunCommandOnHost but retries the
// command given the number of retries provided.
func (p *pocketdBin) RunCommandOnHostWithRetry(rpcUrl string, numRetries uint8, args ...string) (*commandResult, error) {
	if numRetries <= 0 {
		return p.RunCommandOnHost(rpcUrl, args...)
	}
	res, err := p.RunCommandOnHost(rpcUrl, args...)
	if err == nil {
		return res, nil
	}
	// TODO_HACK: Figure out a better solution for retries. A parameter? Exponential backoff? What else?
	time.Sleep(5 * time.Second)
	return p.RunCommandOnHostWithRetry(rpcUrl, numRetries-1, args...)
}

// RunCurl runs a curl command on the local machine
func (p *pocketdBin) RunCurl(rpcUrl, service, path, data string, args ...string) (*commandResult, error) {
	if rpcUrl == "" {
		rpcUrl = defaultAppGateServerURL
	}
	return p.runCurlPostCmd(rpcUrl, service, path, data, args...)
}

// RunCurlWithRetry runs a curl command on the local machine with multiple retries.
// It also accounts for an ephemeral error that may occur due to DNS resolution such as "no such host".
func (p *pocketdBin) RunCurlWithRetry(rpcUrl, service, path, data string, numRetries uint8, args ...string) (*commandResult, error) {
	// No more retries left
	if numRetries <= 0 {
		return p.RunCurl(rpcUrl, service, path, data, args...)
	}
	// Run the curl command
	res, err := p.RunCurl(rpcUrl, service, path, data, args...)
	// Retry if there was an error or the response contains "no such host"
	if err != nil || strings.Contains(res.Stdout, "no such host") {
		time.Sleep(10 * time.Millisecond)
		return p.RunCurlWithRetry(rpcUrl, service, path, data, numRetries-1, args...)
	}
	// Return a successful result
	return res, nil
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
func (p *pocketdBin) runCurlPostCmd(rpcUrl, service, path, jsonRpcData string, args ...string) (*commandResult, error) {
	// Ensure that if a path is provided, it starts with a "/".
	// This is required for RESTful APIs that use a path to identify resources.
	// For JSON-RPC APIs, the resource path should be empty, so empty paths are allowed.
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	urlStr := fmt.Sprintf("%s/%s%s", rpcUrl, service, path)
	base := []string{
		"-v",         // verbose output
		"-sS",        // silent with error
		"-X", "POST", // HTTP method
		"-H", "Content-Type: application/json", // HTTP headers
		"--data", jsonRpcData, urlStr, // POST data
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

	if defaultDebugOutput == "true" {
		fmt.Printf("%#v\n", r)
	}

	if err != nil {
		// Include the command executed in the error message for context
		err = fmt.Errorf("error running command [%s]: %v, stderr: %s", commandStr, err, stderrBuf.String())
	}

	return r, err
}
