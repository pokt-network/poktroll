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
	// IsEphemeralChain is a flag that indicates whether the test is expected to be
	// run on localnet or long living chains (i.e. TestNet/DevNet).
	IsEphemeralChain bool                     `yaml:"is_ephemeral_chain"`
	TestNetNode      string                   `yaml:"testnet_node"`
	ServiceId        string                   `yaml:"service_id"`
	Suppliers        []ProvisionedActorConfig `yaml:"suppliers"`
	Gateways         []ProvisionedActorConfig `yaml:"gateways"`
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

	if parsedManifest.IsEphemeralChain {
		return validatedEphemeralChainManifest(&parsedManifest)
	}

	return validatedNonEphemeralChainManifest(&parsedManifest)
}

func validatedEphemeralChainManifest(manifest *LoadTestManifestYAML) (*LoadTestManifestYAML, error) {
	if len(manifest.Gateways) == 0 {
		return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty gateways entry")
	}

	if len(manifest.Suppliers) == 0 {
		return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty suppliers entry")
	}

	if len(manifest.ServiceId) == 0 {
		return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty service id")
	}

	for _, gateway := range manifest.Gateways {
		if len(gateway.KeyName) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty gateway key name")
		}

		if len(gateway.Address) > 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("gateway address forbidden")
		}

		if len(gateway.ExposedUrl) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty gateway server url")
		}

		if _, err := url.Parse(gateway.ExposedUrl); err != nil {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	for _, supplier := range manifest.Suppliers {
		if len(supplier.KeyName) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty supplier key name")
		}

		if len(supplier.ExposedUrl) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty supplier server url")
		}

		if _, err := url.Parse(supplier.ExposedUrl); err != nil {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	return manifest, nil
}

func validatedNonEphemeralChainManifest(manifest *LoadTestManifestYAML) (*LoadTestManifestYAML, error) {
	if len(manifest.Gateways) == 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty gateways entry")
	}

	if len(manifest.Suppliers) > 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("suppliers entry forbidden")
	}

	if len(manifest.TestNetNode) == 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty testnet node url")
	}

	if len(manifest.ServiceId) == 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty service id")
	}

	for _, gateway := range manifest.Gateways {
		if len(gateway.KeyName) > 0 {
			return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("gateway keyName forbidden")
		}

		if len(gateway.Address) == 0 {
			return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty gateway address")
		}

		if len(gateway.ExposedUrl) == 0 {
			return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty gateway server url")
		}

		if _, err := url.Parse(gateway.ExposedUrl); err != nil {
			return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	return manifest, nil
}
