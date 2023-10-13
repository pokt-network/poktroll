//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
)

// cliPath is the path of the binary installed and is set by the Tiltfile
const cliPath = "/usr/local/bin/pocketd"

var (
	// defaultRPCURL used by targetPod to build commands
	defaultRPCURL string
	// targetDevClientPod is the kube pod that executes calls to the pocket binary under test
	targetDevClientPod = "pocketd-88658b5f8-r9gmv"
	// defaultRPCPort is the default RPC port that poktrolld listens on
	defaultRPCPort = 36657
	// defaultRPCHost is the default RPC host that poktrolld listens on
	defaultRPCHost = "127.0.0.1"
)

func init() {
	defaultRPCURL = fmt.Sprintf("tcp://%s:%d", defaultRPCHost, defaultRPCPort)
}

// commandResult combines the stdout, stderr, and err of an operation
type commandResult struct {
	Stdout string
	Stderr string
	Err    error
}

// PocketClient is a single function interface for interacting with a node
type PocketClient interface {
	RunCommand(...string) (*commandResult, error)
	RunCommandOnHost(string, ...string) (*commandResult, error)
}

// Ensure that Validator fulfills PocketClient
var _ PocketClient = &pocketdPod{}

// pocketdPod holds the connection information to a specific pod in between different instructions during testing
type pocketdPod struct {
	targetPodName string
	result        *commandResult // stores the result of the last command that was run
}

// RunCommand runs a command on a pre-configured kube pod with the given args
func (n *pocketdPod) RunCommand(args ...string) (*commandResult, error) {
	return n.RunCommandOnHost(defaultRPCURL, args...)
}

// RunCommandOnHost runs a command on specified kube pod with the given args
func (n *pocketdPod) RunCommandOnHost(rpcUrl string, args ...string) (*commandResult, error) {
	base := []string{
		"exec", "-i", targetDevClientPod,
		//"--container", "default",
		"--", cliPath,
		//"--node=", defaultRPCURL,
	}
	args = append(base, args...)
	cmd := exec.Command("kubectl", args...)
	r := &commandResult{}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	r.Stdout = string(out)
	n.result = r
	// IMPROVE: make targetPodName configurable
	n.targetPodName = targetDevClientPod
	return r, nil
}
