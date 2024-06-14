//go:build load

package tests

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	defaultManifestPath  = filepath.Join("load-testing", "loadtest_manifest_localnet.yaml")
	flagLogLevel         string
	flagManifestFilePath string
	logger               polylog.Logger
)

func init() {
	flag.StringVar(&flagLogLevel, "log-level", "", "Specifies the log level for the runner")
	flag.StringVar(
		&flagManifestFilePath,
		"manifest",
		defaultManifestPath,
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
