//go:build e2e

package e2e

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"
)

var addrRe *regexp.Regexp

func init() {
	addrRe = regexp.MustCompile(`address:\s+(pokt1\w+)`)
}

type suite struct {
	gocuke.TestingT
	pocketd *pocketdBin
}

func (s *suite) Before() {
	s.pocketd = new(pocketdBin)
}

// TestFeatures runs the e2e tests specified in any .features files in this directory
// * This test suite assumes that a LocalNet is running
func TestFeatures(t *testing.T) {
	gocuke.NewRunner(t, &suite{}).Path("*.feature").Run()
}

func (s *suite) TheUserHasThePocketdBinaryInstalled() {
	s.TheUserRunsTheCommand("help")
}

func (s *suite) ThePocketdBinaryShouldExitWithoutError() {
	require.NoError(s, s.pocketd.result.Err)
}

func (s *suite) TheUserRunsTheCommand(cmd string) {
	cmds := strings.Split(cmd, " ")
	res, err := s.pocketd.RunCommand(cmds...)
	s.pocketd.result = res
	if err != nil {
		s.Fatalf("error running command %s: %s", cmd, err)
	}
}

func (s *suite) TheUserShouldBeAbleToSeeStandardOutputContaining(arg1 string) {
	if !strings.Contains(s.pocketd.result.Stdout, arg1) {
		s.Fatalf("stdout must contain %s", arg1)
	}
}

func (s *suite) TheUserSendsUpoktToAnotherAddress(amount int64) {
	addrs := s.getAddresses()
	args := []string{
		"tx",
		"bank",
		"send",
		addrs[0],
		addrs[1],
		fmt.Sprintf("%dupokt", amount),
		"--keyring-backend",
		"test",
		"-y",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	if err != nil {
		s.Fatalf("error sending upokt: %s", err)
	}
	s.pocketd.result = res
}

func (s *suite) getAddresses() [2]string {
	var strs [2]string
	res, err := s.pocketd.RunCommand(
		"keys", "list", "--keyring-backend", "test",
	)
	if err != nil {
		s.Fatalf("error getting keys: %s", err)
	}
	matches := addrRe.FindAllStringSubmatch(res.Stdout, -1)
	if len(matches) >= 2 {
		strs[0] = matches[0][1]
		strs[1] = matches[len(matches)-1][1]
	} else {
		s.Fatalf("could not find two addresses in output: %s", res.Stdout)
	}
	return strs
}
