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

// TODO(jim): passed by lock.
func GetAPIs(mempoolNode MempoolNode, ethPRC, peptideRPC client.RPC, logger log.Logger) []rpc.API {
	if ethPRC == nil {
		panic("eth client is nil")
	}
	if peptideRPC == nil {
		panic("peptide client is nil")
	}

	if logger == nil {
		panic("logger is nil")
	}

	// TODO(jim): Move eth to its own file?
	apis := []rpc.API{
		{
			Namespace: "engine",
			Service:   newEngineAPI(mempoolNode, ethPRC, peptideRPC, logger),
		},
		{
			Namespace: "eth",
			Service:   newEthAPI(ethPRC, peptideRPC, logger),
		},
	}

	return apis
}

/* 'engine_' prefixed server methods, only required ones. */

// engineServer is the API for the execution engine.
// Implements most of the 'engine_' methods and the currently (guided by op-e2e tests)
// required 'eth_' prefixed methods.
type engineServer struct {
	// mempoolNode contains a reference to the mempool.
	mempoolNode MempoolNode

	// ethRPC is an RPC client for calling into op-geth RPC server.
	ethRPC client.RPC
	// peptideRPC is an RPC client for calling into the peptide RPC server (sdk engine).
	peptideRPC client.RPC

	logger log.Logger
}

// newExecutionEngineAPI returns a new execEngineAPI.
func newEngineAPI(mempoolNode MempoolNode, ethRPC, peptideRPC client.RPC, logger log.Logger) *engineServer {
	return &engineServer{mempoolNode, ethRPC, peptideRPC, logger}
}

// TODO(jim): Is not called, forward to V2 which is?
func (e *engineServer) ForkchoiceUpdatedV1(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV1, forwarding to geth", "fcs", fcs, "pa", pa)

	var result eth.ForkchoiceUpdatedResult
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV1", fcs, pa)

	if false {
		// Forward to the abci engine.
		e.logger.Info("forwarding ForkchoiceUpdatedV1 to abci engine")
		err = e.peptideRPC.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV1", fcs, pa)

		if err != nil {
			e.logger.Error("failed to forward ForkchoiceUpdatedV1 to abci engine", "error", err)
		} else {
			e.logger.Info("completed: forwarding ForkchoiceUpdatedV1 to abci engine")
		}
	}

	e.logger.Info("completed: forwarding ForkchoiceUpdatedV1 to geth", "error", err, "result", &result)

	return &result, err
}

func (e *engineServer) ForkchoiceUpdatedV2(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV2", "fcs", fcs, "pa", pa)

	var result eth.ForkchoiceUpdatedResult
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV2", fcs, pa)

	e.logger.Info("message mempool status: ", "hasMsgs", e.mempoolNode.HasMsgs())

	//if false {
	// Forward to the abci engine.
	e.logger.Info("forwarding ForkchoiceUpdatedV2 to abci engine")

	var peptideResult eth.ForkchoiceUpdatedResult
	err = e.peptideRPC.CallContext(context.TODO(), &peptideResult, "engine_forkchoiceUpdatedV2", fcs, pa)
	if err != nil {
		e.logger.Error("failed to forward ForkchoiceUpdatedV2 to abci engine", "error", err)
	} else {
		e.logger.Info("completed: forwarding ForkchoiceUpdatedV2 to abci engine")
	}

	//}

	// TODO(jim): Crude at this point.
	if e.mempoolNode.HasMsgs() {
		msgs := e.mempoolNode.GetMsgs()

		for _, msg := range msgs {
			e.logger.Info("forwarding a message to abci mempool", "msg", msg)
			e.peptideRPC.CallContext(context.TODO(), nil, "intercept_addMsgToTxMempool", msg)
		}
	}

	e.logger.Info("completed: ForkchoiceUpdatedV2", "error", err, "result", result)

	return &result, err
}

// TODO(jim): Isn't called in current version of op-node we depend on but _is_ called in version
// peptide was developed against.
func (e *engineServer) ForkchoiceUpdatedV3(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV3", "fcs", fcs, "pa", pa)

	var result eth.ForkchoiceUpdatedResult
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV3", fcs, pa)

	if false {
		// Forward to the abci engine.
		e.logger.Info("forwarding ForkchoiceUpdatedV3 to abci engine")

		err = e.peptideRPC.CallContext(context.TODO(), &result, "engine_forkchoiceUpdatedV3", fcs, pa)
		if err != nil {
			e.logger.Error("failed to forward ForkchoiceUpdatedV3 to abci engine", "error", err)
		} else {
			e.logger.Info("completed: forwarding ForkchoiceUpdatedV3 to abci engine")
		}
	}

	e.logger.Info("completed: ForkchoiceUpdatedV3", "error", err, "result", &result)
	return &result, err
}

func (e *engineServer) GetPayloadV1(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Info("trying: GetPayloadV1", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_getPayloadV1", payloadID)

	if false {
		// Forward to the abci engine.
		e.logger.Info("forwarding GetPayloadV1 to abci engine")

		err = e.peptideRPC.CallContext(context.TODO(), &result, "engine_getPayloadV1", payloadID)
		if err != nil {
			e.logger.Error("failed to forward GetPayloadV1 to abci engine", "error", err)
		} else {
			e.logger.Info("completed: forwarding GetPayloadV1 to abci engine")
		}
	}

	e.logger.Info("completed: GetPayloadV1", "error", err, "result", &result)
	return result.ExecutionPayload, err
}

func (e *engineServer) GetPayloadV2(payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	e.logger.Info("GetPayloadV2", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_getPayloadV2", payloadID)

	// Forward to the abci engine.
	e.logger.Info("forwarding GetPayloadV2 to abci engine")

	var abciResult eth.ExecutionPayloadEnvelope
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "engine_getPayloadV2", payloadID)
	if err != nil {
		e.logger.Error("failed to forward GetPayloadV2 to abci engine", "error", err)
	} else {
		e.logger.Info("completed: forwarding GetPayloadV2 to abci engine")
	}

	e.logger.Info("completed: GetPayloadV2", "error", err, "result", result.ExecutionPayload)
	return &result, err
}

func (e *engineServer) GetPayloadV3(payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	e.logger.Info("GetPayloadV3", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_getPayloadV3", payloadID)

	if false {
		// Forward to the abci engine.
		e.logger.Info("forwarding GetPayloadV3 to abci engine")

		err = e.peptideRPC.CallContext(context.TODO(), &result, "engine_getPayloadV3", payloadID)
		if err != nil {
			e.logger.Error("failed to forward GetPayloadV3 to abci engine", "error", err)
		} else {
			e.logger.Info("completed: forwarding GetPayloadV3 to abci engine")
		}
	}

	e.logger.Info("completed: GetPayloadV3", "error", err, "result", &result)
	return &result, err
}

func (e *engineServer) NewPayloadV1(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_newPayloadV1", payload)

	if false {
		// Forward to the abci engine.
		e.logger.Info("forwarding NewPayloadV1 to abci engine")

		err = e.peptideRPC.CallContext(context.TODO(), &result, "engine_newPayloadV1", payload)
		if err != nil {
			e.logger.Error("failed to forward NewPayloadV1 to abci engine", "error", err)
		} else {
			e.logger.Info("completed: forwarding NewPayloadV1 to abci engine")
		}
	}

	e.logger.Info("completed: NewPayloadV1", "error", err, "result", &result)
	return &result, err
}

func (e *engineServer) NewPayloadV2(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_newPayloadV2", payload)

	// Forward to the abci engine.
	e.logger.Info("forwarding NewPayloadV2 to abci engine")

	var abciResult eth.PayloadStatusV1
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "engine_newPayloadV2", payload)
	if err != nil {
		e.logger.Error("failed to forward NewPayloadV2 to abci engine", "error", err)
	} else {
		e.logger.Info("completed: forwarding NewPayloadV2 to abci engine")
	}

	e.logger.Info("completed: NewPayloadV2", "error", err, "result", &result)
	return &result, err
}

func (e *engineServer) NewPayloadV3(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV3", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.ethRPC.CallContext(context.TODO(), &result, "engine_newPayloadV3", payload)

	if false {
		// Forward to the abci engine.
		e.logger.Info("forwarding NewPayloadV3 to abci engine")

		err = e.peptideRPC.CallContext(context.TODO(), &result, "engine_newPayloadV3", payload)
		if err != nil {
			e.logger.Error("failed to forward NewPayloadV3 to abci engine", "error", err)
		} else {
			e.logger.Info("completed: forwarding NewPayloadV3 to abci engine")
		}
	}

	e.logger.Info("completed: NewPayloadV3", "error", err, "result", &result)
	return &result, err
}

/* 'eth_' prefixed server methods, only required ones. */

// ethServer is the API for the eth like server.
// Implements required 'eth_' prefixed methods.
type ethServer struct {
	// client dials into op-geth server.
	// Might be best to not embed if we maybe want to add an sdk engine via rpc.
	ethRPC     client.RPC
	peptideRPC client.RPC
	logger     log.Logger
}

// newEthAPI returns a new execEngineAPI.
func newEthAPI(ethRPC, peptideRPC client.RPC, logger log.Logger) *ethServer {
	return &ethServer{ethRPC, peptideRPC, logger}
}

func (e *ethServer) ChainId() (hexutil.Big, error) { // nolint: revive, stylecheck
	e.logger.Info("trying: ChainID")

	var id hexutil.Big
	err := e.ethRPC.CallContext(context.TODO(), &id, "eth_chainId")

	return id, err
}

// Docu yanked from go-eth for fullTx.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (e *ethServer) GetBlockByNumber(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByNumber", "id", id)

	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getBlockByNumber", id, fullTx)

	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *ethServer) GetBlockByHash(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByHash", "id", id)

	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getBlockByHash", id, fullTx)

	e.logger.Info("completed: GetBlockByHash", "result", result)

	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *ethServer) GetProof(address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (map[string]any, error) {
	e.logger.Info("trying: GetProof")

	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getProof", address, storageKeys, blockNrOrHash)

	e.logger.Info("completed: GetBlockByHash", "result", result)
	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (e *ethServer) GetTransactionReceipt(txHash common.Hash) (map[string]any, error) {
	e.logger.Info("trying: GetTransactionReceipt")
	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getTransactionReceipt", txHash)

	e.logger.Info("completed: GetTransactionReceipt", "error", err, "result", result)
	return result, err
}
