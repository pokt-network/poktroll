//go:build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

var (
	// pocketd holds command results between runs and reports errors to the test suite
	pocketd = &pocketdPod{}
)

// TestFeatures runs the e2e tests specified in any .features files in this directory
// * This test suite assumes that a LocalNet is running that can be accessed by `kubectl`
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"./"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

// InitializeScenario registers step regexes to function handlers
func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^the user runs the command "([^"]*)"$`, theUserRunsTheCommand)
	ctx.Step(`^the user should be able to see standard output containing "([^"]*)"$`, theUserShouldBeAbleToSeeStandardOutputContaining)
	ctx.Step(`^the user has the pocketd binary installed$`, theUserHasPocketd)
	ctx.Step(`^the pocketd binary should exit without error$`, thePocketdShouldHaveExitedWithoutError)
}

func theUserHasPocketd() error {
	res, err := pocketd.RunCommand("help")
	pocketd.result = res
	if err != nil {
		return err
	}
	return nil
}

func thePocketdShouldHaveExitedWithoutError() error {
	return pocketd.result.Err
}

func theUserRunsTheCommand(cmd string) error {
	cmds := strings.Split(cmd, " ")
	res, err := pocketd.RunCommand(cmds...)
	pocketd.result = res
	if err != nil {
		return err
	}
	return nil
}

func theUserShouldBeAbleToSeeStandardOutputContaining(arg1 string) error {
	if !strings.Contains(pocketd.result.Stdout, arg1) {
		return fmt.Errorf("stdout must contain %s", arg1)
	}
	return nil
}
