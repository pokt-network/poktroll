//go:build e2e

package e2e

import (
	"fmt"
	"strings"

	"github.com/stretchr/testify/require"
)

// keyExistsInKeyring checks if a key with the given name exists in the keyring
// using the `pocketd keys show <key_name>` CLI subcommand.
func (s *suite) keyExistsInKeyring(keyName string) bool {
	s.Helper()

	// pocketd keys show <key_name> --keyring-backend test
	argsAndFlags := []string{
		"keys",
		"show",
		keyName,
		keyRingFlag,
	}
	res, err := s.pocketd.RunCommand(argsAndFlags...)
	if err == nil {
		// `keys show <key_name>` will not error if a key with the given name exists.
		return true
	}

	keyNotExistErrFmt := "%s is not a valid name or address"
	if strings.Contains(res.Stderr, fmt.Sprintf(keyNotExistErrFmt, keyName)) {
		return false
	}

	// If the key doesn't exist & the error message is unexpected, fail the test.
	s.Fatal(err)
	return false
}

// addKeyToKeyring adds a new key, with the given name, to the keyring
// using the `pocketd keys add <key_name>` CLI subcommand.
func (s *suite) addKeyToKeyring(keyName string) {
	s.Helper()

	// pocketd keys add <key_name> --keyring-backend test
	argsAndFlags := []string{
		"keys",
		"add",
		keyName,
		keyRingFlag,
	}
	_, err := s.pocketd.RunCommand(argsAndFlags...)
	require.NoError(s, err)
}
