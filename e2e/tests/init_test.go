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

var (
	addrRe        *regexp.Regexp
	accNameToAddr = make(map[string]string)
	keyRingFlag   = "--keyring-backend=test"
)

func init() {
	addrRe = regexp.MustCompile(`address: (\S+)\s+name: (\S+)`)
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
	runner := gocuke.NewRunner(t, &suite{}).Path("*.feature")
	runner = runner.Step(`^the user sends (\d+) uPOKT from account (\w+) to account (\w+)$`, (*suite).TheUserSendsUpoktFromAccountToAccount)
	runner.Run()
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

func (s *suite) TheUserSendsUpoktFromAccountToAccount(amount int64, acc1, acc2 string) {
	s.buildAddrMap()
	args := []string{
		"tx",
		"bank",
		"send",
		accNameToAddr[acc1],
		accNameToAddr[acc2],
		fmt.Sprintf("%dupokt", amount),
		keyRingFlag,
		"-y",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	if err != nil {
		s.Fatalf("error sending upokt: %s", err)
	}
	s.pocketd.result = res
}

func (s *suite) buildAddrMap() {
	res, err := s.pocketd.RunCommand(
		"keys", "list", keyRingFlag,
	)
	if err != nil {
		s.Fatalf("error getting keys: %s", err)
	}
	matches := addrRe.FindAllStringSubmatch(res.Stdout, -1)
	for _, match := range matches {
		name := match[2]
		address := match[1]
		accNameToAddr[name] = address
	}
}
