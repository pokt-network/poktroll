package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// ProvisionedActorConfig is used to represent the signing key used & URL exposed
// by the pre-provisioned gateway & supplier actors that the load test expects.
type ProvisionedActorConfig struct {
	// KeyName is the **name** of the key in the keyring to be used by the given actor.
	KeyName string `yaml:"key_name"`
	// ExposedUrl is the URL where the actor is expected to be reachable.
	ExposedUrl string `yaml:"exposed_url"`
}

// LoadTestManifestYAML is the struct which the load test manifest is deserialized into.
// It contains the list of suppliers and gateways that the load test expects to be pre-provisioned.
type LoadTestManifestYAML struct {
	Suppliers []ProvisionedActorConfig `yaml:"suppliers"`
	Gateways  []ProvisionedActorConfig `yaml:"gateways"`
}

// ParseLoadTestManifest reads the load test manifest from the given byte slice
// and returns the parsed LoadTestManifestYAML struct.
// It returns an error if the manifest is empty, or if it fails to unmarshal.
func ParseLoadTestManifest(manifestContent []byte) (*LoadTestManifestYAML, error) {
	var parsedManifest LoadTestManifestYAML

	if len(manifestContent) == 0 {
		return nil, ErrLoadTestManifestEmpty
	}

	if err := yaml.Unmarshal(manifestContent, &parsedManifest); err != nil {
		return nil, ErrLoadTestManifestUnmarshalYAML.Wrapf("%s", err)
	}

	if len(parsedManifest.Gateways) == 0 {
		return nil, ErrLoadTestInvalidManifest.Wrap("empty gateways entry")
	}

	if len(parsedManifest.Suppliers) == 0 {
		return nil, ErrLoadTestInvalidManifest.Wrap("empty suppliers entry")
	}

	for _, supplier := range parsedManifest.Suppliers {
		if supplier.KeyName == "" {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty supplier key name")
		}

		if supplier.ExposedUrl == "" {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty supplier server url")
		}

		if _, err := url.Parse(supplier.ExposedUrl); err != nil {
			return nil, ErrLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	for _, gateway := range parsedManifest.Gateways {
		if gateway.KeyName == "" {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty gateway key name")
		}

		if gateway.ExposedUrl == "" {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty gateway server url")
		}

		if _, err := url.Parse(gateway.ExposedUrl); err != nil {
			return nil, ErrLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	return &parsedManifest, nil
}
