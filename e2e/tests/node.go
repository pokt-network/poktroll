//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"net/url"
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
	// defaultPathURL used by curl commands to send relay requests
	defaultPathURL = os.Getenv("PATH_URL")
	// defaultDebugOutput provides verbose output on manipulations with binaries (cli command, stdout, stderr)
	defaultDebugOutput = os.Getenv("E2E_DEBUG_OUTPUT")
)

func isVerbose() bool {
	return defaultDebugOutput == "true"
}

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
	RunCurl(rpcUrl, service, method, path, appAddr, data string, args ...string) (*commandResult, error)
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
	// DEV_NOTE: Intentionally keeping a print statement here so errors are
	// very visible even though the output may be noisy.
	fmt.Printf(`
----------------------------------------
Retrying command due to error:
	- RPC URL:      %s
	- Arguments:    %v
	- Response:     %v
	- Error:        %v
----------------------------------------
`, rpcUrl, args, res, err)
	// TODO_TECHDEBT(@bryanchriswhite): Figure out a better solution for retries. A parameter? Exponential backoff? What else?
	time.Sleep(5 * time.Second)
	return p.RunCommandOnHostWithRetry(rpcUrl, numRetries-1, args...)
}

// RunCurl runs a curl command on the local machine
func (p *pocketdBin) RunCurl(rpcUrl, service, method, path, appAddr, data string, args ...string) (*commandResult, error) {
	if rpcUrl == "" {
		rpcUrl = defaultPathURL
	}
	return p.runCurlCmd(rpcUrl, service, method, path, appAddr, data, args...)
}

// RunCurlWithRetry runs a curl command on the local machine with multiple retries.
// It also accounts for an ephemeral error that may occur due to DNS resolution such as "no such host".
func (p *pocketdBin) RunCurlWithRetry(rpcUrl, service, method, path, appAddr, data string, numRetries uint8, args ...string) (*commandResult, error) {
	if service == "" {
		err := fmt.Errorf("Missing service name for curl request with url: %s", rpcUrl)
		return nil, err
	}

	// No more retries left
	if numRetries <= 0 {
		return p.RunCurl(rpcUrl, service, method, path, appAddr, data, args...)
	}
	// Run the curl command
	res, err := p.RunCurl(rpcUrl, service, method, path, appAddr, data, args...)
	if err != nil {
		return p.RunCurlWithRetry(rpcUrl, service, method, path, appAddr, data, numRetries-1, args...)
	}

	// TODO_HACK: This is a list of common flaky / ephemeral errors that can occur
	// during end-to-end tests. If any of them are hit, we retry the command.
	ephemeralEndToEndErrors := []string{
		"no such host",
		"internal error: upstream error",
	}
	for _, ephemeralError := range ephemeralEndToEndErrors {
		if strings.Contains(res.Stdout, ephemeralError) {
			if isVerbose() {
				fmt.Println("Retrying due to ephemeral error:", res.Stdout)
			}
			time.Sleep(10 * time.Millisecond)
			return p.RunCurlWithRetry(rpcUrl, service, method, path, appAddr, data, numRetries-1, args...)
		}
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

// runCurlCmd is a helper to run a command using the local pocketd binary with the flags provided
func (p *pocketdBin) runCurlCmd(rpcBaseURL, service, method, path, appAddr, data string, args ...string) (*commandResult, error) {
	rpcUrl, err := url.Parse(rpcBaseURL)
	if err != nil {
		return nil, err
	}

	// Ensure that if a path is provided, it starts with a "/".
	// This is required for RESTful APIs that use a path to identify resources.
	// For JSON-RPC APIs, the resource path should be empty, so empty paths are allowed.
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	rpcUrl.Path = rpcUrl.Path + path

	// Ensure that the path also ends with a "/" if it only contains the version.
	// This is required because the server responds with a 301 redirect for "/v1"
	// and curl binaries on some platforms MAY NOT support re-sending POST data
	// while following a redirect (`-L` flag).
	if strings.HasSuffix(rpcUrl.Path, "/v1") {
		rpcUrl.Path = rpcUrl.Path + "/"
	}

	base := []string{
		"-v",                                   // verbose output
		"-sS",                                  // silent with error
		"-H", `Content-Type: application/json`, // HTTP headers
		"-H", fmt.Sprintf("Host: %s", rpcUrl.Host), // Add virtual host header
		"-H", fmt.Sprintf("App-Address: %s", appAddr),
		"-H", fmt.Sprintf("Target-Service-Id: %s", service),
		rpcUrl.String(),
	}

	if method == "POST" {
		base = append(base, "--data", data)
	} else if len(data) > 0 {
		fmt.Printf("WARN: data provided but not being included in the %s request because it is not of type POST", method)
	}
	args = append(base, args...)
	commandStr := "curl " + strings.Join(args, " ") // Create a string representation of the command
	cmd := exec.Command("curl", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
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
