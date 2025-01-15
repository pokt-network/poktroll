package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	comettypes "github.com/cometbft/cometbft/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// TODO_IN_THIS_COMMIT: godoc...
type CometBFTMethod string

// TODO_IN_THIS_COMMIT: godoc...
type ServiceMethodUri string

const (
	abciQueryMethod         = CometBFTMethod("abci_query")
	broadcastTxSyncMethod   = CometBFTMethod("broadcast_tx_sync")
	broadcastTxAsyncMethod  = CometBFTMethod("broadcast_tx_async")
	broadcastTxCommitMethod = CometBFTMethod("broadcast_tx_commit")
	blockMethod             = CometBFTMethod("block")

	authAccountQueryUri = ServiceMethodUri("/cosmos.auth.v1beta1.Query/Account")
)

// handleABCIQuery handles the actual ABCI query logic
func newPostHandler(
	ctx context.Context,
	client gogogrpc.ClientConn,
	app *E2EApp,
) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		// DEV_NOTE: http.Error() automatically sets the Content-Type header to "text/plain".
		w.Header().Set("Content-Type", "application/json")

		// Read and log request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse JSON-RPC request
		var req rpctypes.RPCRequest
		if err = json.Unmarshal(body, &req); err != nil {
			writeErrorResponseFromErr(w, req, err)
			return
		}

		params := make(map[string]json.RawMessage)
		if err = json.Unmarshal(req.Params, &params); err != nil {
			writeErrorResponseFromErr(w, req, err)
			return
		}

		response := new(rpctypes.RPCResponse)
		switch CometBFTMethod(req.Method) {
		// TODO_IN_THIS_COMMIT: extract...
		case abciQueryMethod:
			response, err = app.abciQuery(ctx, client, req, params)
			if err != nil {
				*response = rpctypes.NewRPCErrorResponse(req.ID, 500, err.Error(), "")
			}
		case broadcastTxSyncMethod, broadcastTxAsyncMethod, broadcastTxCommitMethod:
			response, err = app.broadcastTx(req, params)
			if err != nil {
				*response = rpctypes.NewRPCErrorResponse(req.ID, 500, err.Error(), "")
			}
		case blockMethod:
			response, err = app.block(ctx, client, req, params)
			if err != nil {
				*response = rpctypes.NewRPCErrorResponse(req.ID, 500, err.Error(), "")
			}
		default:
			*response = rpctypes.NewRPCErrorResponse(req.ID, 500, "unsupported method", string(req.Params))
		}

		if err = json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func (app *E2EApp) abciQuery(
	ctx context.Context,
	client gogogrpc.ClientConn,
	req rpctypes.RPCRequest,
	params map[string]json.RawMessage,
) (*rpctypes.RPCResponse, error) {
	var (
		resData []byte
		height  int64
	)

	pathRaw, hasPath := params["path"]
	if !hasPath {
		return nil, fmt.Errorf("missing path param: %s", string(req.Params))
	}

	var path string
	if err := json.Unmarshal(pathRaw, &path); err != nil {
		return nil, err
	}

	switch ServiceMethodUri(path) {
	case authAccountQueryUri:
		dataRaw, hasData := params["data"]
		if !hasData {
			return nil, fmt.Errorf("missing data param: %s", string(req.Params))
		}

		data, err := hex.DecodeString(string(bytes.Trim(dataRaw, `"`)))
		if err != nil {
			return nil, err
		}

		queryReq := new(authtypes.QueryAccountRequest)
		if err = queryReq.Unmarshal(data); err != nil {
			return nil, err
		}

		var height int64
		heightRaw, hasHeight := params["height"]
		if hasHeight {
			if err = json.Unmarshal(bytes.Trim(heightRaw, `"`), &height); err != nil {
				return nil, err
			}
		}

		queryRes := new(authtypes.QueryAccountResponse)
		if err = client.Invoke(ctx, path, queryReq, queryRes); err != nil {
			return nil, err
		}

		resData, err = queryRes.Marshal()
		if err != nil {
			return nil, err
		}
	}

	abciQueryRes := coretypes.ResultABCIQuery{
		Response: types.ResponseQuery{
			//Code:      0,
			//Index:     0,
			//Key:       nil,
			Value:  resData,
			Height: height,
		},
	}

	res := rpctypes.NewRPCSuccessResponse(req.ID, abciQueryRes)
	return &res, nil
}

// TODO_IN_THIS_COMMIT: godoc...
func (app *E2EApp) broadcastTx(
	req rpctypes.RPCRequest,
	params map[string]json.RawMessage,
) (*rpctypes.RPCResponse, error) {
	var txBz []byte
	txRaw, hasTx := params["tx"]
	if !hasTx {
		return nil, fmt.Errorf("missing tx param: %s", string(req.Params))
	}
	if err := json.Unmarshal(txRaw, &txBz); err != nil {
		return nil, err
	}

	// TODO_CONSIDERATION: more correct implementation of the different
	// broadcast_tx methods (i.e. sync, async, commit) is a matter of
	// the sequencing of the following:
	// - calling the finalize block ABCI method
	// - returning the JSON-RPC response
	// - emitting websocket event

	_, finalizeBlockRes, err := app.RunTx(nil, txBz)
	if err != nil {
		return nil, err
	}

	// TODO_IN_THIS_COMMIT: something better...
	go func() {
		// Simulate 1 second block production delay.
		time.Sleep(time.Second * 1)

		//fmt.Println(">>> emitting ws events")
		//app.EmitWSEvents(app.GetSdkCtx().EventManager().Events())

		// TODO_IMPROVE: If we want/need to support multiple txs per
		// block in the future, this will have to be refactored.
		app.EmitWSEvents(finalizeBlockRes, txBz)
	}()

	// DEV_NOTE: There SHOULD ALWAYS be exactly one tx result so long as
	// we're finalizing one tx at a time (single tx blocks).
	txRes := finalizeBlockRes.GetTxResults()[0]

	bcastTxRes := coretypes.ResultBroadcastTx{
		Code:      txRes.GetCode(),
		Data:      txRes.GetData(),
		Log:       txRes.GetLog(),
		Codespace: txRes.GetCodespace(),
		Hash:      comettypes.Tx(txBz).Hash(),
	}

	res := rpctypes.NewRPCSuccessResponse(req.ID, bcastTxRes)
	return &res, nil
}

// TODO_IN_THIS_COMMIT: godoc...
func (app *E2EApp) block(
	ctx context.Context,
	client gogogrpc.ClientConn,
	req rpctypes.RPCRequest,
	params map[string]json.RawMessage,
) (*rpctypes.RPCResponse, error) {
	resultBlock := coretypes.ResultBlock{
		BlockID: app.GetCometBlockID(),
		Block: &comettypes.Block{
			Header: comettypes.Header{
				//Version:            version.Consensus{},
				ChainID:     "poktroll-test",
				Height:      app.GetSdkCtx().BlockHeight(),
				Time:        time.Now(),
				LastBlockID: app.GetCometBlockID(),
				//LastCommitHash:     nil,
				//DataHash:           nil,
				//ValidatorsHash:     nil,
				//NextValidatorsHash: nil,
				//ConsensusHash:      nil,
				//AppHash:            nil,
				//LastResultsHash:    nil,
				//EvidenceHash:       nil,
				//ProposerAddress:    nil,
			},
			//Data:       comettypes.Data{},
			//Evidence:   comettypes.EvidenceData{},
			//LastCommit: nil,
		},
	}
	res := rpctypes.NewRPCSuccessResponse(req.ID, resultBlock)
	return &res, nil
}

// TODO_IN_THIS_COMMIT: godoc...
func writeErrorResponseFromErr(w http.ResponseWriter, req rpctypes.RPCRequest, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	writeErrorResponse(w, req, errMsg, "")
}

// TODO_IN_THIS_COMMIT: godoc...
func writeErrorResponse(w http.ResponseWriter, req rpctypes.RPCRequest, msg, data string) {
	errRes := rpctypes.NewRPCErrorResponse(req.ID, 500, msg, data)
	if err := json.NewEncoder(w).Encode(errRes); err != nil {
		// TODO_IN_THIS_COMMIT: log error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
