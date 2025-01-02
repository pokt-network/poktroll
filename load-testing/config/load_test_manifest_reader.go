package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// ProvisionedActorConfig is used to represent the signing key used & URL exposed
// by the pre-provisioned gateway & supplier actors that the load test expects.
type ProvisionedActorConfig struct {
	// The address used to identify the actor. In an ephemeral chain, the corresponding
	// account must be present in the keyring to be able to stake.
	Address string `yaml:"address"`
	// ExposedUrl is the URL where the actor is expected to be reachable.
	ExposedUrl string `yaml:"exposed_url"`
}

// LoadTestManifestYAML is the struct which the load test manifest is deserialized into.
// It contains the list of suppliers and gateways that the load test expects to be pre-provisioned.
type LoadTestManifestYAML struct {
	// IsEphemeralChain is a flag that indicates whether the test is expected to be
	// run on LocalNet or long-living remote chain (i.e. TestNet/DevNet).
	IsEphemeralChain      bool                     `yaml:"is_ephemeral_chain"`
	PRCNode               string                   `yaml:"rpc_node"`
	ServiceId             string                   `yaml:"service_id"`
	Suppliers             []ProvisionedActorConfig `yaml:"suppliers"`
	Gateways              []ProvisionedActorConfig `yaml:"gateways"`
	FundingAccountAddress string                   `yaml:"funding_account_address"`
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

	if len(manifest.FundingAccountAddress) == 0 {
		return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty funding account address")
	}

	if len(manifest.PRCNode) == 0 {
		return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty rpc node url")
	}

	for _, gateway := range manifest.Gateways {
		if len(gateway.Address) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty gateway address")
		}

		if len(gateway.ExposedUrl) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty gateway server url")
		}

		if _, err := url.Parse(gateway.ExposedUrl); err != nil {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrapf("invalid supplier server url: %s", err)
		}
	}

	for _, supplier := range manifest.Suppliers {
		if len(supplier.Address) == 0 {
			return nil, ErrEphemeralChainLoadTestInvalidManifest.Wrap("empty supplier operator address")
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

	if len(manifest.PRCNode) == 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty rpc node url")
	}

	if len(manifest.ServiceId) == 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty service id")
	}

	if len(manifest.FundingAccountAddress) == 0 {
		return nil, ErrNonEphemeralChainLoadTestInvalidManifest.Wrap("empty funding account address")
	}

	for _, gateway := range manifest.Gateways {
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
