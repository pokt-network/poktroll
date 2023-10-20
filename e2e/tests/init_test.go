//go:build e2e

package e2e

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"
)

var (
	addrRe        *regexp.Regexp
	amountRe      *regexp.Regexp
	accNameToAddr = make(map[string]string)
	keyRingFlag   = "--keyring-backend=test"
)

func init() {
	addrRe = regexp.MustCompile(`address: (\S+)\s+name: (\S+)`)
	amountRe = regexp.MustCompile(`amount: "(.+?)"\s+denom: upokt`)
}

type suite struct {
	gocuke.TestingT
	pocketd   *pocketdBin
	tempState map[string]any // temporary state for each scenario
}

func (s *suite) Before() {
	s.pocketd = new(pocketdBin)
	s.tempState = make(map[string]any)
}

// TestFeatures runs the e2e tests specified in any .features files in this directory
// * This test suite assumes that a LocalNet is running
func TestFeatures(t *testing.T) {
	runner := gocuke.NewRunner(t, &suite{}).Path("*.feature")
	runner = registerSteps(runner)
	runner.Run()
}

func registerSteps(runner *gocuke.Runner) *gocuke.Runner {
	return runner.
		Step(`^the user sends (\d+) uPOKT from account (\w+) to account (\w+)$`, (*suite).TheUserSendsUpoktFromAccountToAccount).
		Step(`^the account (\w+) has a balance greater than (\d+) uPOKT$`, (*suite).TheAccountHasABalanceGreaterThanUpokt).
		Step(`^the account balance is known for (\w+)$`, (*suite).TheAccountBalanceIsKnownFor).
		Step(`^the account balance of (\w+) should be (\d+) uPOKT (\w+) than before$`, (*suite).TheAccountBalanceOfShouldBeUpoktThanBefore)
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

func (s *suite) TheAccountHasABalanceGreaterThanUpokt(acc string, amount int64) {
	s.buildAddrMap()
	args := []string{
		"query",
		"bank",
		"balances",
		accNameToAddr[acc],
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	if err != nil {
		s.Fatalf("error getting balance: %s", err)
	}
	s.pocketd.result = res
	match := amountRe.FindStringSubmatch(res.Stdout)
	if len(match) < 2 {
		s.Fatalf("no balance found for %s", acc)
	}
	found, err := strconv.Atoi(match[1])
	require.NoError(s, err)
	if int64(found) < amount {
		s.Fatalf("account %s does not have enough upokt: %d < %d", acc, found, amount)
	}
	s.tempState[acc] = found // save the balance for later
}

func (s *suite) TheAccountBalanceIsKnownFor(acc string) {
	s.buildAddrMap()
	args := []string{
		"query",
		"bank",
		"balances",
		accNameToAddr[acc],
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	if err != nil {
		s.Fatalf("error getting balance: %s", err)
	}
	s.pocketd.result = res
	match := amountRe.FindStringSubmatch(res.Stdout)
	if len(match) < 2 {
		s.Fatalf("no balance found for %s", acc)
	}
	found, err := strconv.Atoi(match[1])
	require.NoError(s, err)
	s.tempState[acc] = found // save the balance for later
}

func (s *suite) TheAccountBalanceOfShouldBeUpoktThanBefore(acc string, amount int64, condition string) {
	s.buildAddrMap()
	prev, ok := s.tempState[acc]
	if !ok {
		s.Fatalf("no previous balance found for %s", acc)
	}
	args := []string{
		"query",
		"bank",
		"balances",
		accNameToAddr[acc],
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	if err != nil {
		s.Fatalf("error getting balance: %s", err)
	}
	s.pocketd.result = res
	match := amountRe.FindStringSubmatch(res.Stdout)
	if len(match) < 2 {
		s.Fatalf("no balance found for %s", acc)
	}
	found, err := strconv.Atoi(match[1])
	require.NoError(s, err)
	switch condition {
	case "more":
		if found <= prev.(int) {
			s.Fatalf("account %s does not have more upokt: %d <= %d", acc, found, prev)
		}
	case "less":
		if found >= prev.(int) {
			s.Fatalf("account %s does not have less upokt: %d >= %d", acc, found, prev)
		}
	default:
		s.Fatalf("unknown condition %s", condition)
	}
}

func (s *suite) TheUserShouldWaitForSeconds(dur int64) {
	time.Sleep(time.Duration(dur) * time.Second)
}

func (s *suite) buildAddrMap() {
	if len(accNameToAddr) > 0 {
		return
	}
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
