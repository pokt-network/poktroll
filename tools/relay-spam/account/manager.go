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
	// Import application accounts
	for _, app := range m.config.Applications {
		// Skip if already imported
		_, err := m.keyring.Key(app.Name)
		if err == nil {
			fmt.Printf("Application account %s already imported, skipping\n", app.Name)
			continue
		}

		// Import account
		hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
		_, err = m.keyring.NewAccount(app.Name, app.Mnemonic, "", hdPath, hd.Secp256k1)
		if err != nil {
			return fmt.Errorf("failed to import application account %s: %w", app.Name, err)
		}

		fmt.Printf("Successfully imported application account %s with address %s\n", app.Name, app.Address)
	}

	// Import service accounts
	for _, service := range m.config.Services {
		// Skip if already imported
		_, err := m.keyring.Key(service.Name)
		if err == nil {
			fmt.Printf("Service account %s already imported, skipping\n", service.Name)
			continue
		}

		// Import account
		hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
		_, err = m.keyring.NewAccount(service.Name, service.Mnemonic, "", hdPath, hd.Secp256k1)
		if err != nil {
			return fmt.Errorf("failed to import service account %s: %w", service.Name, err)
		}

		fmt.Printf("Successfully imported service account %s with address %s\n", service.Name, service.Address)
	}

	// Import supplier accounts
	for _, supplier := range m.config.Suppliers {
		// Skip if already imported
		_, err := m.keyring.Key(supplier.Name)
		if err == nil {
			fmt.Printf("Supplier account %s already imported, skipping\n", supplier.Name)
			continue
		}

		// Import account
		hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
		_, err = m.keyring.NewAccount(supplier.Name, supplier.Mnemonic, "", hdPath, hd.Secp256k1)
		if err != nil {
			return fmt.Errorf("failed to import supplier account %s: %w", supplier.Name, err)
		}

		fmt.Printf("Successfully imported supplier account %s with address %s\n", supplier.Name, supplier.Address)
	}

	return nil
}

func (m *Manager) GenerateFundingCommands() ([]string, error) {
	var commands []string

	// Generate funding commands for applications
	appFundAmount := "1000000upokt" // Default value
	if m.config.ApplicationFundGoal != "" {
		appFundAmount = m.config.ApplicationFundGoal
	}

	for _, app := range m.config.Applications {
		cmd := fmt.Sprintf("poktrolld tx bank send faucet %s %s %s",
			app.Address, appFundAmount, m.config.TxFlags)
		commands = append(commands, cmd)
	}

	// Generate funding commands for services
	serviceFundAmount := "1000000upokt" // Default value
	if m.config.ServiceFundGoal != "" {
		serviceFundAmount = m.config.ServiceFundGoal
	}

	for _, service := range m.config.Services {
		cmd := fmt.Sprintf("poktrolld tx bank send faucet %s %s %s",
			service.Address, serviceFundAmount, m.config.TxFlags)
		commands = append(commands, cmd)
	}

	// Generate funding commands for suppliers
	supplierFundAmount := "1000000upokt" // Default value
	if m.config.SupplierStakeGoal != "" {
		supplierFundAmount = m.config.SupplierStakeGoal
	}

	for _, supplier := range m.config.Suppliers {
		cmd := fmt.Sprintf("poktrolld tx bank send faucet %s %s %s",
			supplier.Address, supplierFundAmount, m.config.TxFlags)
		commands = append(commands, cmd)
	}

	return commands, nil
}

func (m *Manager) GenerateStakingCommands() ([]string, error) {
	var commands []string

	// Generate staking commands for applications
	appStakeAmount := "1000000upokt" // Default value
	if m.config.ApplicationStakeGoal != "" {
		appStakeAmount = m.config.ApplicationStakeGoal
	}

	for _, app := range m.config.Applications {
		if app.ServiceIdGoal == "" {
			continue // Skip if no service ID goal
		}
		cmd := fmt.Sprintf("poktrolld tx application stake-application %s %s %s",
			app.ServiceIdGoal, appStakeAmount, m.config.TxFlags)
		commands = append(commands, cmd)
	}

	// Generate staking commands for services
	serviceStakeAmount := "1000000upokt" // Default value
	if m.config.ServiceStakeGoal != "" {
		serviceStakeAmount = m.config.ServiceStakeGoal
	}

	for _, service := range m.config.Services {
		if service.ServiceId == "" {
			continue // Skip if no service ID
		}
		cmd := fmt.Sprintf("poktrolld tx service create-service %s %s %s",
			service.ServiceId, serviceStakeAmount, m.config.TxFlags)
		commands = append(commands, cmd)
	}

	// Generate staking commands for suppliers
	supplierStakeAmount := "1000000upokt" // Default value
	if m.config.SupplierStakeGoal != "" {
		supplierStakeAmount = m.config.SupplierStakeGoal
	}

	for _, supplier := range m.config.Suppliers {
		// Skip if no stake config or services
		if len(supplier.StakeConfig.Services) == 0 {
			continue
		}

		// Create a stake supplier command for each service
		for _, serviceConfig := range supplier.StakeConfig.Services {
			cmd := fmt.Sprintf("poktrolld tx supplier stake-supplier %s %s %s",
				serviceConfig.ServiceId, supplierStakeAmount, m.config.TxFlags)
			commands = append(commands, cmd)
		}
	}

	return commands, nil
}
