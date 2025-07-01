# Suggested Shell Commands for Poktroll Development

## Essential Development Commands

### Quick Start Development Workflow
```bash
# Generate code and run all tests  
make go_develop_and_test

# Start LocalNet for E2E testing
make localnet_up

# Run comprehensive linting (always before commits)
make go_lint

# Reset LocalNet when needed
make localnet_reset
```

### Protocol Buffer Workflow
```bash
# Regenerate protobuf artifacts after .proto changes
make proto_regen

# Build the pocketd binary
make ignite_pocketd_build
```

### Testing Commands
```bash
# Run all tests (fastest, default)
make test_all

# Run tests with integration
make test_all_with_integration

# Run E2E tests (requires LocalNet)
make test_e2e

# Run specific E2E test suites
make test_e2e_relay
make test_e2e_app  
make test_e2e_supplier
```

### Development Tools Installation
```bash
# Install development dependencies
make install_ci_deps

# Install cosmovisor for upgrades
make install_cosmovisor

# Verify gopls is available (required for Serena)
gopls version || go install golang.org/x/tools/gopls@latest
```

### LocalNet Management
```bash
# Query account balance
make acc_balance_query ACC=<address>

# View LocalNet logs
kubectl logs -f <pod-name> -n default

# Check LocalNet status
kubectl get pods -n default
```

### Repository Exploration
```bash
# List all Go modules
find . -name "go.mod" -not -path "./vendor/*"

# Find protocol buffer files  
find proto/ -name "*.proto" | head -20

# List all test files
find . -name "*_test.go" | head -20

# Check module structure
ls -la x/
```

### Git and Code Quality
```bash
# Check git status
git status

# View recent commits
git log --oneline -10

# Run linting (mandatory before commits)
make go_lint

# Check for todos and fixmes
grep -r "TODO\|FIXME" --include="*.go" . | head -10
```

### Environment Setup Verification
```bash
# Check Go version (should be 1.24.3+)
go version

# Verify required tools
which golangci-lint
which goimports  
which yq

# Check environment variables
echo $GOPATH
echo $PATH | tr ':' '\n' | grep go
```

## Safety Commands (Use with Caution)

### LocalNet Reset and Cleanup
```bash
# Complete LocalNet reset (destructive)
make localnet_down && make localnet_reset

# Clean build artifacts
make clean  # if available

# Reset go modules (extreme case)
go mod tidy && go mod download
```

## Debugging and Investigation Commands

### Log Analysis
```bash
# View E2E test output with debug
E2E_DEBUG_OUTPUT=true make test_e2e_verbose

# Check for recent errors in logs
grep -i error /path/to/logs/*.log | tail -20
```

### Code Analysis  
```bash
# Find symbol definitions
grep -r "type.*struct" x/ --include="*.go" | head -10

# Search for specific patterns
grep -r "keeper.*Query" x/ --include="*.go"

# Check imports for specific packages
grep -r "pkg/polylog" --include="*.go" . | head -5
```

## Notes

- **Always run `make go_lint` before commits** - This is mandatory
- **Use `make localnet_reset`** when LocalNet state becomes inconsistent  
- **Run `make proto_regen`** after any `.proto` file changes
- **Check `make help`** for complete list of available targets
- **GOPATH/bin must be in PATH** for tools like gopls to work properly