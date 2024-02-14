// Implement the wrapping calls to the wrapped engines. Forwards the RPC call to op-geth server
// while directly calling the cosmos sdk engine. See https://ethereum.github.io/execution-apis/api-documentation/
package api

// TODO(jim):
// - Document the methods with at least the info in the api-documentation link above.
// - A generic function can be added that just forwards calls in all the wrapping methods.

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/cometbft/cometbft/libs/log"
)

// execEngineServer is the API for the execution engine.
// Implements most of the 'engine_' methods and the currently (guided by op-e2e tests)
// required 'eth_' prefixed methods.
type execEngineServer struct {
	// client dials into op-geth server.
	// Might be best to not embed if we maybe want to add an sdk engine via rpc.
	client.RPC
	logger log.Logger
}

// newExecutionEngineAPI returns a new execEngineAPI.
func newExecutionEngineAPI(engine client.RPC, logger log.Logger) *execEngineServer {
	return &execEngineServer{engine, logger}
}

func GetAPIs(rpcClient client.RPC, logger log.Logger) []rpc.API {
	if rpcClient == nil {
		panic("execEngine is nil")
	}
	if logger == nil {
		panic("logger is nil")
	}

	node := newExecutionEngineAPI(rpcClient, logger)
	apis := []rpc.API{
		{
			Namespace: "engine",
			Service:   node,
		},
		{
			Namespace: "eth",
			Service:   node,
		},
		{
			Namespace: "cosmos",
			Service:   node,
		},
	}

	return apis
}

/* 'engine_' prefixed server methods, only required ones. */

func (e *execEngineServer) ForkchoiceUpdatedV1(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV1, forwarding to geth", "fcs", fcs, "pa", pa)

	var result eth.ForkchoiceUpdatedResult
	err := e.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV1", fcs, pa)

	e.logger.Info("completed: ForkchoiceUpdatedV1", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineServer) ForkchoiceUpdatedV2(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV2", "fcs", fcs, "pa", pa)

	var result eth.ForkchoiceUpdatedResult
	err := e.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV2", fcs, pa)

	e.logger.Info("completed: ForkchoiceUpdatedV2", "error", err, "result", result)

	return &result, err
}

func (e *execEngineServer) ForkchoiceUpdatedV3(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV3", "fcs", fcs, "pa", pa)

	var result eth.ForkchoiceUpdatedResult
	err := e.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV3", fcs, pa)

	e.logger.Info("completed: ForkchoiceUpdatedV3", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineServer) GetPayloadV1(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Info("GetPayloadV1", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.CallContext(context.TODO(), &result, "engine_getPayloadV1", payloadID)

	e.logger.Info("completed: GetPayloadV1", "error", err, "result", &result)

	return result.ExecutionPayload, err
}

func (e *execEngineServer) GetPayloadV2(payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	e.logger.Info("GetPayloadV2", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.CallContext(context.TODO(), &result, "engine_getPayloadV2", payloadID)

	e.logger.Info("completed: GetPayloadV2", "error", err, "result", result.ExecutionPayload)

	return &result, err
}

func (e *execEngineServer) GetPayloadV3(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Info("GetPayloadV3", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.CallContext(context.TODO(), &result, "engine_getPayloadV3", payloadID)

	e.logger.Info("completed: GetPayloadV3", "error", err, "result", &result)

	return result.ExecutionPayload, err
}

func (e *execEngineServer) NewPayloadV1(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.CallContext(context.TODO(), &result, "engine_newPayloadV1", payload)

	e.logger.Info("completed: NewPayloadV1", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineServer) NewPayloadV2(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.CallContext(context.TODO(), &result, "engine_newPayloadV2", payload)

	e.logger.Info("completed: NewPayloadV2", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineServer) NewPayloadV3(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV3", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.CallContext(context.TODO(), &result, "engine_newPayloadV3", payload)

	e.logger.Info("completed: NewPayloadV3", "error", err, "result", &result)

	return &result, err
}

/* 'eth_' prefixed server methods, only required ones. */

func (e *execEngineServer) ChainId() (hexutil.Big, error) { // nolint: revive, stylecheck
	e.logger.Info("trying: ChainID")

	var id hexutil.Big
	err := e.CallContext(context.TODO(), &id, "eth_chainId")

	return id, err
}

// Docu yanked from go-eth for fullTx.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (e *execEngineServer) GetBlockByNumber(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByNumber", "id", id)

	var result map[string]any
	err := e.CallContext(context.TODO(), &result, "eth_getBlockByNumber", id, fullTx)

	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *execEngineServer) GetBlockByHash(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByHash", "id", id)

	var result map[string]any
	err := e.CallContext(context.TODO(), &result, "eth_getBlockByHash", id, fullTx)

	e.logger.Info("completed: GetBlockByHash", "result", result)

	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *execEngineServer) GetProof(address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (map[string]any, error) {
	e.logger.Info("trying: GetProof")

	var result map[string]any
	err := e.CallContext(context.TODO(), &result, "eth_getProof", address, storageKeys, blockNrOrHash)

	e.logger.Info("completed: GetBlockByHash", "result", result)
	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (e *execEngineServer) GetTransactionReceipt(txHash common.Hash) (map[string]any, error) {
	e.logger.Info("trying: GetTransactionReceipt")
	var result map[string]any
	err := e.CallContext(context.TODO(), &result, "eth_getTransactionReceipt", txHash)

	e.logger.Info("completed: GetTransactionReceipt", "error", err, "result", result)
	return result, err
}

/* 'cosmos_' Namespace server methods:

Basically for any information we might want to send over from our e2es. */

// SendCosmosTx receives an opaque tx byte slice and adds it to the mempool.
func (e *execEngineServer) SendTransaction(tx []byte) (SendCosmosTxResult, error) {
	e.logger.Info("trying: SendTransaction", "tx", tx)

	// TODO(jim): Add it to our dummy mempool.

	e.logger.Info("completed: SendTransaction")
	return SendCosmosTxResult{}, nil
}
