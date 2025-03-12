package account

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/tyler-smith/go-bip39"

	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

type Manager struct {
	keyring keyring.Keyring
	config  *config.Config
}

func NewManager(kr keyring.Keyring, cfg *config.Config) *Manager {
	return &Manager{
		keyring: kr,
		config:  cfg,
	}
}

func (m *Manager) CreateAccounts(numAccounts int) ([]config.Application, error) {
	var applications []config.Application

	// Find the highest existing index
	startIndex := 0
	re := regexp.MustCompile(`relay_spam_app_(\d+)`)

	for _, app := range m.config.Applications {
		matches := re.FindStringSubmatch(app.Name)
		if len(matches) > 1 {
			index, err := strconv.Atoi(matches[1])
			if err == nil && index >= startIndex {
				startIndex = index + 1
			}
		}
	}

	for i := 0; i < numAccounts; i++ {
		index := startIndex + i
		name := fmt.Sprintf("relay_spam_app_%d", index)

		// Generate mnemonic
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			return nil, err
		}
		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			return nil, err
		}

		// Create account
		record, err := m.keyring.NewAccount(name, mnemonic, "", "m/44'/118'/0'/0/0", hd.Secp256k1)
		if err != nil {
			return nil, err
		}

		address, err := record.GetAddress()
		if err != nil {
			return nil, err
		}

		// Create application config
		app := config.Application{
			Name:           name,
			Address:        address.String(),
			Mnemonic:       mnemonic,
			ServiceIdGoal:  "",         // Default service ID
			DelegateesGoal: []string{}, // Empty delegatees by default
		}

		applications = append(applications, app)
	}

	return applications, nil
}

func (m *Manager) ImportAccounts() error {
	for _, app := range m.config.Applications {
		// Skip if already imported
		_, err := m.keyring.Key(app.Name)
		if err == nil {
			fmt.Printf("Account %s already imported, skipping\n", app.Name)
			continue
		}

		// Import account
		hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
		_, err = m.keyring.NewAccount(app.Name, app.Mnemonic, "", hdPath, hd.Secp256k1)
		if err != nil {
			return fmt.Errorf("failed to import account %s: %w", app.Name, err)
		}

		fmt.Printf("Successfully imported account %s with address %s\n", app.Name, app.Address)
	}

	return nil
}

func (m *Manager) GenerateFundingCommands() ([]string, error) {
	var commands []string

	// Use the global ApplicationFundGoal if available, otherwise use a default value
	fundAmount := "1000000upokt" // Default value
	if m.config.ApplicationFundGoal != "" {
		fundAmount = m.config.ApplicationFundGoal
	}

	for _, app := range m.config.Applications {
		cmd := fmt.Sprintf("poktrolld tx bank send faucet %s %s %s",
			app.Address, fundAmount, m.config.TxFlags)
		commands = append(commands, cmd)
	}

	return commands, nil
}
