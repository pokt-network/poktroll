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
	// Address is the address of the actor, which is used to identify already staked
	// actors in the network in persistent chains.
	Address string `yaml:"address"`
	// ExposedUrl is the URL where the actor is expected to be reachable.
	ExposedUrl string `yaml:"exposed_url"`
}

// LoadTestManifestYAML is the struct which the load test manifest is deserialized into.
// It contains the list of suppliers and gateways that the load test expects to be pre-provisioned.
type LoadTestManifestYAML struct {
	// PersistentChain is a flag that indicates whether the test is expected to be
	// run on localnet or long living chains (i.e. TestNet/DevNet).
	PersistentChain bool                     `yaml:"persistent_chain"`
	TestNetNode     string                   `yaml:"testnet_node"`
	ServiceId       string                   `yaml:"service_id"`
	Suppliers       []ProvisionedActorConfig `yaml:"suppliers"`
	Gateways        []ProvisionedActorConfig `yaml:"gateways"`
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

	if len(parsedManifest.Suppliers) == 0 && !parsedManifest.PersistentChain {
		return nil, ErrLoadTestInvalidManifest.Wrap("empty suppliers entry")
	}

	if parsedManifest.TestNetNode == "" && parsedManifest.PersistentChain {
		return nil, ErrLoadTestInvalidManifest.Wrap("empty testnet node url")
	}

	if parsedManifest.ServiceId == "" {
		return nil, ErrLoadTestInvalidManifest.Wrap("empty service id")
	}

	for _, gateway := range parsedManifest.Gateways {
		if gateway.KeyName == "" && !parsedManifest.PersistentChain {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty gateway key name")
		}

		if gateway.Address == "" && parsedManifest.PersistentChain {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty gateway address")
		}

		if gateway.ExposedUrl == "" {
			return nil, ErrLoadTestInvalidManifest.Wrap("empty gateway server url")
		}

		if _, err := url.Parse(gateway.ExposedUrl); err != nil {
			return nil, ErrLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	if parsedManifest.PersistentChain {
		return &parsedManifest, nil
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

	return &parsedManifest, nil
}
