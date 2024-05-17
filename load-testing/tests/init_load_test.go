//go:build load

package tests

import (
	"flag"
	"testing"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	flagLogLevel         string
	flagManifestFilePath string = "../../loadtest_manifest.yaml"
	logger               polylog.Logger
)

func init() {
	flag.StringVar(&flagLogLevel, "log-level", "", "Specifies the log level for the runner")
	flag.StringVar(
		&flagManifestFilePath,
		"manifest",
		"../../loadtest_manifest.yaml",
		"Specifies the path to the load test manifest file",
	)
}

func TestMain(m *testing.M) {
	flag.Parse()

	// Initialize the global logger with the log level specified by the corresponding flag.
	configureLogger()

	m.Run()
}

func configureLogger() {
	logLevel := polyzero.ParseLevel(flagLogLevel)

	// Set the suite logger.
	logger = polyzero.NewLogger(polyzero.WithLevel(logLevel))
}
