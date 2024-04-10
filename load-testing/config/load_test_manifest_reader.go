package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

type ProvisionedActorConfig struct {
	KeyName    string `yaml:"key_name"`
	ExposedUrl string `yaml:"exposed_url"`
}

type LoadTestManifestYAML struct {
	Suppliers []ProvisionedActorConfig `yaml:"suppliers"`
	Gateways  []ProvisionedActorConfig `yaml:"gateways"`
}

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
