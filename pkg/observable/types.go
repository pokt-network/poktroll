package observable

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

type (
	Error       = Observable[error]
	Relay       = Observable[*servicetypes.Relay]
	SessionTree = Observable[relayer.SessionTree]
)
