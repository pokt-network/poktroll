# Comprehensive Plan for Go Relay Spam Tool

## 1. Overall Architecture

The Go implementation will follow a modular, clean architecture with the following components:

1. **CLI Interface** - Using Viper and Cobra for command-line parsing and configuration
2. **Configuration** - YAML-based configuration similar to the Ruby version
3. **Account Management** - Handling account creation, import, and funding
4. **Application Staking** - Managing application staking and delegation
5. **Relay Spam Engine** - Core functionality for sending relay requests
6. **Metrics Collection** - Tracking success rates, request counts, etc.

## 2. Project Structure

```
tools/relay-spam/
├── cmd/
│   └── root.go           # Main CLI entry point
│   └── populate.go       # Command to populate config accounts
│   └── import.go         # Command to import accounts
│   └── fund.go           # Command to fund accounts
│   └── stake.go          # Command to stake applications
│   └── run.go            # Command to run relay spam
├── config/
│   └── config.go         # Configuration handling
├── account/
│   └── manager.go        # Account management functionality
├── application/
│   └── staker.go         # Application staking functionality
├── relay/
│   └── spammer.go        # Core relay spam functionality
│   └── request.go        # Request handling
├── metrics/
│   └── collector.go      # Metrics collection
├── util/
│   └── helpers.go        # Utility functions
├── main.go               # Entry point
└── config.yml.example    # Example configuration file
```

## 3. Detailed Component Descriptions

### 3.1 CLI Interface (using Cobra and Viper)

```go
// cmd/root.go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    configFile    string
    numRequests   int
    concurrency   int
    numAccounts   int
    rateLimit     float64
)

var rootCmd = &cobra.Command{
    Use:   "relay-spam",
    Short: "A tool for stress testing Pocket Network with relay requests",
    Long:  `Relay Spam is a comprehensive tool for testing Pocket Network's relay capabilities by generating high volumes of relay requests from multiple accounts.`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yml", "Config file")
    rootCmd.PersistentFlags().IntVarP(&numRequests, "num-requests", "n", 10, "Number of requests per application-gateway pair")
    rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "p", 10, "Concurrent requests")
    rootCmd.PersistentFlags().IntVarP(&numAccounts, "num-accounts", "a", 10, "Number of accounts to create")
    rootCmd.PersistentFlags().Float64VarP(&rateLimit, "rate-limit", "r", 0, "Rate limit in requests per second (0 for no limit)")
    
    viper.BindPFlag("num_requests", rootCmd.PersistentFlags().Lookup("num-requests"))
    viper.BindPFlag("concurrency", rootCmd.PersistentFlags().Lookup("concurrency"))
    viper.BindPFlag("num_accounts", rootCmd.PersistentFlags().Lookup("num-accounts"))
    viper.BindPFlag("rate_limit", rootCmd.PersistentFlags().Lookup("rate-limit"))
}

func initConfig() {
    if configFile != "" {
        viper.SetConfigFile(configFile)
    } else {
        viper.AddConfigPath(".")
        viper.SetConfigName("config")
        viper.SetConfigType("yml")
    }
    
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err == nil {
        fmt.Println("Using config file:", viper.ConfigFileUsed())
    }
}
```

### 3.2 Configuration Structure

```go
// config/config.go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Home               string                 `mapstructure:"home"`
    TxFlags            string                 `mapstructure:"tx_flags"`
    TxFlagsTemplate    map[string]string      `mapstructure:"tx_flags_template"`
    Applications       []Application          `mapstructure:"applications"`
    ApplicationDefaults ApplicationDefaults   `mapstructure:"application_defaults"`
}

type Application struct {
    Name      string   `mapstructure:"name"`
    Address   string   `mapstructure:"address"`
    Mnemonic  string   `mapstructure:"mnemonic"`
    Gateways  []string `mapstructure:"gateways"`
    Staked    bool     `mapstructure:"staked"`
    Delegated bool     `mapstructure:"delegated"`
}

type ApplicationDefaults struct {
    Stake      string   `mapstructure:"stake"`
    ServiceID  string   `mapstructure:"service_id"`
    Gateways   []string `mapstructure:"gateways"`
}

func LoadConfig() (*Config, error) {
    var config Config
    err := viper.Unmarshal(&config)
    if err != nil {
        return nil, err
    }
    
    // Process config (expand templates, etc.)
    if config.TxFlagsTemplate != nil {
        config.TxFlags = expandTxFlagsTemplate(config.TxFlagsTemplate)
    }
    
    // Set default home directory if not specified
    if config.Home == "" {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return nil, err
        }
        config.Home = filepath.Join(homeDir, ".poktroll")
    }
    
    return &config, nil
}

func expandTxFlagsTemplate(template map[string]string) string {
    var flags []string
    for k, v := range template {
        flags = append(flags, fmt.Sprintf("--%s=%s", k, v))
    }
    return strings.Join(flags, " ")
}
```

### 3.3 Account Management

```go
// account/manager.go
package account

import (
    "github.com/cosmos/cosmos-sdk/crypto/hd"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    sdktypes "github.com/cosmos/cosmos-sdk/types"
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
    for _, app := range m.config.Applications {
        // Extract index from name (relay_spam_app_X)
        // Update startIndex if higher
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
        hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
        record, err := m.keyring.NewAccount(name, mnemonic, "", hdPath, hd.Secp256k1)
        if err != nil {
            return nil, err
        }
        
        address, err := record.GetAddress()
        if err != nil {
            return nil, err
        }
        
        // Create application config
        app := config.Application{
            Name:      name,
            Address:   address.String(),
            Mnemonic:  mnemonic,
            Gateways:  m.config.ApplicationDefaults.Gateways,
            Staked:    false,
            Delegated: false,
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
            continue
        }
        
        // Import account
        hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
        _, err = m.keyring.NewAccount(app.Name, app.Mnemonic, "", hdPath, hd.Secp256k1)
        if err != nil {
            return err
        }
    }
    
    return nil
}

func (m *Manager) GenerateFundingCommands() ([]string, error) {
    var commands []string
    
    for _, app := range m.config.Applications {
        cmd := fmt.Sprintf("poktrolld tx bank send faucet %s 1000000upokt %s", 
            app.Address, m.config.TxFlags)
        commands = append(commands, cmd)
    }
    
    return commands, nil
}
```

### 3.4 Application Staking

```go
// application/staker.go
package application

import (
    "context"
    "time"
    
    "github.com/cosmos/cosmos-sdk/client"
    "github.com/cosmos/cosmos-sdk/client/tx"
    sdktypes "github.com/cosmos/cosmos-sdk/types"
    "github.com/pokt-network/poktroll/x/application/types"
)

type Staker struct {
    clientCtx client.Context
    config    *config.Config
}

func NewStaker(clientCtx client.Context, cfg *config.Config) *Staker {
    return &Staker{
        clientCtx: clientCtx,
        config:    cfg,
    }
}

func (s *Staker) StakeApplications() error {
    for i, app := range s.config.Applications {
        if app.Staked {
            continue
        }
        
        // Get account from keyring
        key, err := s.clientCtx.Keyring.Key(app.Name)
        if err != nil {
            return err
        }
        
        // Create stake message
        stakeAmount, err := sdktypes.ParseCoinNormalized(s.config.ApplicationDefaults.Stake)
        if err != nil {
            return err
        }
        
        msg := types.NewMsgStakeApplication(
            key.GetAddress().String(),
            stakeAmount,
            s.config.ApplicationDefaults.ServiceID,
        )
        
        // Build and sign transaction
        txBuilder := s.clientCtx.TxConfig.NewTxBuilder()
        txBuilder.SetMsgs(msg)
        // Set gas, fees, etc.
        
        // Sign transaction
        signerData := tx.SignerData{
            ChainID:       s.clientCtx.ChainID,
            AccountNumber: s.clientCtx.AccountNumber,
            Sequence:      s.clientCtx.Sequence,
        }
        
        // Sign and broadcast transaction
        // ...
        
        // Update config
        s.config.Applications[i].Staked = true
        
        // Sleep to avoid sequence issues
        time.Sleep(time.Second)
    }
    
    return nil
}

func (s *Staker) DelegateToGateways() error {
    for i, app := range s.config.Applications {
        if !app.Staked || app.Delegated {
            continue
        }
        
        for _, gatewayAddr := range app.Gateways {
            // Create delegation message
            // ...
            
            // Build, sign and broadcast transaction
            // ...
        }
        
        // Update config
        s.config.Applications[i].Delegated = true
    }
    
    return nil
}
```

### 3.5 Relay Spam Engine

```go
// relay/spammer.go
package relay

import (
    "context"
    "sync"
    "sync/atomic"
    "time"
)

type Spammer struct {
    config      *config.Config
    numRequests int
    concurrency int
    rateLimit   float64
}

func NewSpammer(cfg *config.Config, numRequests, concurrency int, rateLimit float64) *Spammer {
    return &Spammer{
        config:      cfg,
        numRequests: numRequests,
        concurrency: concurrency,
        rateLimit:   rateLimit,
    }
}

func (s *Spammer) Run(ctx context.Context) (*Metrics, error) {
    metrics := &Metrics{
        TotalRequests:    0,
        SuccessfulRequests: 0,
        FailedRequests:   0,
        StartTime:        time.Now(),
    }
    
    // Create work items
    var workItems []WorkItem
    for _, app := range s.config.Applications {
        for _, gatewayURL := range app.Gateways {
            for i := 0; i < s.numRequests; i++ {
                workItems = append(workItems, WorkItem{
                    AppAddress: app.Address,
                    GatewayURL: gatewayURL,
                })
            }
        }
    }
    
    metrics.TotalRequests = int64(len(workItems))
    
    // Create worker pool
    var wg sync.WaitGroup
    workCh := make(chan WorkItem)
    
    // Start workers
    for i := 0; i < s.concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for work := range workCh {
                result := s.makeRequest(work.GatewayURL, work.AppAddress)
                if result.Success {
                    atomic.AddInt64(&metrics.SuccessfulRequests, 1)
                } else {
                    atomic.AddInt64(&metrics.FailedRequests, 1)
                }
            }
        }()
    }
    
    // Rate limiting
    startTime := time.Now()
    for i, work := range workItems {
        if s.rateLimit > 0 {
            expectedTime := float64(i) / s.rateLimit
            elapsed := time.Since(startTime).Seconds()
            if elapsed < expectedTime {
                time.Sleep(time.Duration((expectedTime - elapsed) * float64(time.Second)))
            }
        }
        
        select {
        case workCh <- work:
            // Work sent
        case <-ctx.Done():
            close(workCh)
            return metrics, ctx.Err()
        }
    }
    
    close(workCh)
    wg.Wait()
    
    metrics.EndTime = time.Now()
    return metrics, nil
}

func (s *Spammer) makeRequest(gatewayURL, appAddress string) RequestResult {
    client := &http.Client{
        Timeout: 10 * time.Second,
    }
    
    payload := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "eth_blockNumber",
        "params":  []interface{}{},
        "id":      1,
    }
    
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return RequestResult{Success: false, Error: err.Error()}
    }
    
    req, err := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(payloadBytes))
    if err != nil {
        return RequestResult{Success: false, Error: err.Error()}
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-App-Address", appAddress)
    req.Header.Set("Target-Service-ID", s.config.ApplicationDefaults.ServiceID)
    
    resp, err := client.Do(req)
    if err != nil {
        return RequestResult{Success: false, Error: err.Error()}
    }
    defer resp.Body.Close()
    
    return RequestResult{
        Success: resp.StatusCode == 200,
        Error:   resp.StatusCode != 200 ? fmt.Sprintf("HTTP %d", resp.StatusCode) : "",
    }
}
```

### 3.6 Metrics Collection

```go
// metrics/collector.go
package metrics

import (
    "fmt"
    "time"
)

type Metrics struct {
    TotalRequests      int64
    SuccessfulRequests int64
    FailedRequests     int64
    StartTime          time.Time
    EndTime            time.Time
}

func (m *Metrics) Print() {
    duration := m.EndTime.Sub(m.StartTime)
    successRate := float64(m.SuccessfulRequests) / float64(m.TotalRequests) * 100
    requestsPerSecond := float64(m.TotalRequests) / duration.Seconds()
    
    fmt.Println("=== Relay Spam Results ===")
    fmt.Printf("Total Requests:      %d\n", m.TotalRequests)
    fmt.Printf("Successful Requests: %d (%.2f%%)\n", m.SuccessfulRequests, successRate)
    fmt.Printf("Failed Requests:     %d (%.2f%%)\n", m.FailedRequests, 100-successRate)
    fmt.Printf("Duration:            %.2f seconds\n", duration.Seconds())
    fmt.Printf("Requests Per Second: %.2f\n", requestsPerSecond)
}
```

### 3.7 Main Entry Point

```go
// main.go
package main

import (
    "fmt"
    "os"
    
    "github.com/cosmos/cosmos-sdk/codec"
    "github.com/cosmos/cosmos-sdk/codec/types"
    cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
    sdktypes "github.com/cosmos/cosmos-sdk/types"
    
    "github.com/pokt-network/poktroll/tools/relay-spam/cmd"
)

func init() {
    // Set the address prefixes for Pocket Network
    config := sdktypes.GetConfig()
    config.SetBech32PrefixForAccount("pokt", "poktpub")
    config.SetBech32PrefixForValidator("poktvaloper", "poktvaloperpub")
    config.SetBech32PrefixForConsensusNode("poktvalcons", "poktvalconspub")
    config.Seal()
}

func main() {
    cmd.Execute()
}
```

## 4. Configuration File Structure

```yaml
# Example config.yml
home: ~/.poktroll

tx_flags_template:
  chain-id: poktroll
  gas: auto
  gas-adjustment: 1.5
  gas-prices: 0.01upokt
  broadcast-mode: sync
  yes: true
  keyring-backend: test

application_defaults:
  stake: 1000000upokt
  service_id: svc1qjpxsjkz0kujcvdlxm2wkjv5m4g0p9k
  gateways:
    - http://localhost:8081
    - http://localhost:8082

applications:
  - name: relay_spam_app_0
    address: pokt1abc123...
    mnemonic: word1 word2 word3...
    staked: false
    delegated: false
```

## 5. Implementation Strategy

1. **Phase 1: Basic Structure and Configuration**
   - Set up project structure
   - Implement configuration loading with Viper
   - Create CLI commands with Cobra

2. **Phase 2: Account Management**
   - Implement account creation and import
   - Implement funding command generation

3. **Phase 3: Application Staking**
   - Implement application staking
   - Implement gateway delegation

4. **Phase 4: Relay Spam Engine**
   - Implement request generation
   - Implement rate limiting
   - Implement concurrency control

5. **Phase 5: Metrics and Reporting**
   - Implement metrics collection
   - Implement results reporting

6. **Phase 6: Testing and Optimization**
   - Test with different configurations
   - Optimize performance
   - Add error handling and recovery

## 6. Advantages Over Ruby Implementation

1. **Performance**: Go's concurrency model with goroutines will be more efficient than Ruby's threads/ractors
2. **Memory Efficiency**: Go's static typing and memory management will use less memory for large workloads
3. **Native Integration**: Direct integration with Cosmos SDK libraries instead of shell commands
4. **Maintainability**: Strongly typed code with clear module boundaries
5. **Deployment**: Single binary deployment without dependencies

## 7. Potential Challenges

1. **Transaction Signing**: Ensuring proper transaction signing and broadcasting
2. **Rate Limiting**: Implementing precise rate limiting across goroutines
3. **Error Handling**: Robust error handling for network issues and transaction failures
4. **Configuration Management**: Maintaining and updating the configuration file

This plan provides a comprehensive roadmap for rewriting the Ruby relay spam tool in Go, leveraging Go's strengths while maintaining the functionality of the original tool.
