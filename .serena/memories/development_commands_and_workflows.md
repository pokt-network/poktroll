# Development Commands and Workflows

## Core Development Commands

### Code Generation & Build

- `make go_develop` - Generate protos and mocks (run after proto changes)
- `make go_develop_and_test` - Generate code + run all tests
- `make ignite_pocketd_build` - Build pocketd binary
- `make proto_regen` - Regenerate protobuf artifacts

### LocalNet Management

- `make localnet_up` - Start local development network
- `make localnet_down` - Stop local network
- `make localnet_reset` - Reset and restart network
- `make acc_balance_query ACC=<addr>` - Query account balance

### Module Addresses

Key onchain module accounts:

- APPLICATION_MODULE_ADDRESS: `pokt1rl3gjgzexmplmds3tq3r3yk84zlwdl6djzgsvm`
- SUPPLIER_MODULE_ADDRESS: `pokt1j40dzzmn6cn9kxku7a5tjnud6hv37vesr5ccaa`
- GATEWAY_MODULE_ADDRESS: `pokt1f6j7u6875p2cvyrgjr0d2uecyzah0kget9vlpl`
- SERVICE_MODULE_ADDRESS: `pokt1nhmtqf4gcmpxu0p6e53hpgtwj0llmsqpxtumcf`
- GOV_ADDRESS: `pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t`
- PNF_ADDRESS: `pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw`

### Key Environment Variables

- `POCKETD_HOME` - Default: `./localnet/pocketd`
- `POCKET_NODE` - Default: `tcp://127.0.0.1:26657`
- `POCKET_ADDR_PREFIX` - Always: `pokt`
- `PATH_URL` - Default: `http://localhost:3000`

### Protobuf Development Workflow

1. Modify `.proto` files in `/proto/pocket/` directories
2. Run `make proto_regen` to generate Go code
3. Update keeper methods and message handlers in `/x/<module>/keeper/`
4. Add/update tests for new functionality
5. Run `make go_lint` before committing
