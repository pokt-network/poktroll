package tests

import (
	"flag"
	"testing"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	flagFeaturesPath string
	flagLogLevel     string
	logger           polylog.Logger
)

func init() {
	flag.StringVar(&flagFeaturesPath, "features-path", "*.feature", "Specifies glob paths for the runner to look up .feature files")
	flag.StringVar(&flagLogLevel, "log-level", "", "Specifies the log level for the runner")
}

func TestMain(m *testing.M) {
	// TODO: parse any flags here (e.g. feature(s) path)
	flag.Parse()

	configureLogger()

	m.Run()
}

//func TestLoadFeatures(t *testing.T) {
//	logger.Info().Msgf("features path: %s", flagFeaturesPath)
//	gocuke.NewRunner(t, &loadSuite{}).Path(flagFeaturesPath).Run()
//}

func configureLogger() {
	// Match log level flag to available level object.
	var logLevel polylog.Level
	for _, level := range polyzero.Levels() {
		if level.String() == flagLogLevel {
			logLevel = level
			break
		}
	}

	// Default to info level if flag is not provided.
	if logLevel == nil {
		logLevel = polyzero.InfoLevel
	}

	// Set the suite logger.
	zerologLevel := zerolog.Level(logLevel.Int())
	logger = polyzero.NewLogger(polyzero.WithLevel(zerologLevel))
}
