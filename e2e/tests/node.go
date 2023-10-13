//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
)

var (
	// defaultRPCURL used by targetPod to build commands
	defaultRPCURL string
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

// Ensure that pocketdBin struct fulfills PocketClient
var _ PocketClient = &pocketdBin{}

// pocketdBin holds the reults of the last command that was run
type pocketdBin struct {
	result *commandResult // stores the result of the last command that was run
}

// RunCommand runs a command on a pre-configured kube pod with the given args
func (n *pocketdBin) RunCommand(args ...string) (*commandResult, error) {
	return n.RunCommandOnHost(defaultRPCURL, args...)
}

// RunCommandOnHost runs a command on specified host with the given args
func (n *pocketdBin) RunCommandOnHost(rpcUrl string, args ...string) (*commandResult, error) {
	base := []string{
		//"--node", defaultRPCURL,
	}
	args = append(base, args...)
	cmd := exec.Command("pocketd", args...)
	r := &commandResult{}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	r.Stdout = string(out)
	n.result = r
	return r, nil
}
