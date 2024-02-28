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

	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
)

// TODO(jim): passed by lock.
func GetAPIs(interceptor Interceptor, ethPRC, peptideRPC client.RPC, logger log.Logger) []rpc.API {
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
			Service:   newEngineAPI(interceptor, ethPRC, peptideRPC, logger),
		},
		{
			Namespace: "eth",
			Service:   newEthAPI(interceptor, ethPRC, peptideRPC, logger),
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
	interceptor Interceptor
	// ethRPC is an RPC client for calling into op-geth RPC server.
	ethRPC client.RPC
	// peptideRPC is an RPC client for calling into the peptide RPC server (sdk engine).
	peptideRPC client.RPC

	logger log.Logger
}

// newExecutionEngineAPI returns a new execEngineAPI.
func newEngineAPI(interceptor Interceptor, ethRPC, peptideRPC client.RPC, logger log.Logger) *engineServer {
	return &engineServer{interceptor, ethRPC, peptideRPC, logger}
}

func (e *engineServer) ForkchoiceUpdatedV2(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	abciFcs, gethFcs := EngineForkStates(e.interceptor, fcs)
	e.logger.Info("trying: ForkchoiceUpdatedV2", "abciFcs", abciFcs, "gethFcs", gethFcs, "pa", pa)

	var gethResult eth.ForkchoiceUpdatedResult
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "engine_forkchoiceUpdatedV2", gethFcs, pa)
	if err != nil {
		// TODO(jim): What do we do if geth for some reason errs and we dont?
		e.logger.Error("failed to forward ForkchoiceUpdatedV2 to geth engine", "error", err)
		return nil, err
	}
	e.logger.Info("success in forwarding ForkchoiceUpdatedV2 to geth engine", "result", gethResult)

	// Forward to the abci engine.
	e.logger.Info("forwarding ForkchoiceUpdatedV2 to abci engine")

	var peptideResult eth.ForkchoiceUpdatedResult
	err = e.peptideRPC.CallContext(context.TODO(), &peptideResult, "engine_forkchoiceUpdatedV2", abciFcs, pa)
	if err != nil {
		e.logger.Error("failed to forward ForkchoiceUpdatedV2 to abci engine", "error", err)
	}
	e.logger.Info("success in forwarding ForkchoiceUpdatedV2 to abci engine", "result", peptideResult)

	// TODO(jim): Crude at this point.
	e.logger.Info("message mempool status: ", "hasMsgs", e.interceptor.HasMsgs())
	if e.interceptor.HasMsgs() {
		msgs := e.interceptor.GetMsgs()

		for _, msg := range msgs {
			e.logger.Info("forwarding a message to abci mempool", "msg", msg)
			err = e.peptideRPC.CallContext(context.TODO(), nil, "intercept_addMsgToTxMempool", msg)
			if err != nil {
				e.logger.Error("failed to forward message to abci mempool", "error", err)
			}
		}
	}

	// Combine payload ids and save them.
	compositePayload := eetypes.NewCompositePayload(gethResult.PayloadID, peptideResult.PayloadID)
	e.interceptor.SaveCompositePayload(compositePayload)
	gethResult.PayloadID = compositePayload.Payload()

	// LatestValidHash of the Payload status should be our composite hash.
	compositeLatestValidHash := eetypes.NewCompositeBlock(*gethResult.PayloadStatus.LatestValidHash, *peptideResult.PayloadStatus.LatestValidHash)
	e.interceptor.SaveCompositeBlock(compositeLatestValidHash)
	compositeHash := compositeLatestValidHash.Hash()
	gethResult.PayloadStatus.LatestValidHash = &compositeHash

	e.logger.Info("completed: ForkchoiceUpdatedV2", "error", err, "result", gethResult)
	return &gethResult, err
}

func (e *engineServer) GetPayloadV2(payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	// Get payload for each of the engines.
	compositePayload := e.interceptor.GetCompositePayload(payloadID)
	abciPayload, gethPayload := compositePayload.ABCIPayload, compositePayload.GethPayload
	e.logger.Info("GetPayloadV2", "payload_id", payloadID, "abciPayload", abciPayload, "gethPayload", gethPayload)

	var gethResult eth.ExecutionPayloadEnvelope
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "engine_getPayloadV2", gethPayload)
	if err != nil {
		// TODO(jim): What do we do if geth for some reason errs and we dont?
		e.logger.Error("failed to forward GetPayloadV2 to geth engine", "error", err)
		return nil, err
	}
	e.logger.Info("success in forwarding GetPayloadV2 to geth engine", "result", gethResult)

	// Forward to the abci engine.
	e.logger.Info("forwarding GetPayloadV2 to abci engine")

	var abciResult eth.ExecutionPayloadEnvelope
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "engine_getPayloadV2", abciPayload)
	if err != nil {
		e.logger.Error("failed to forward GetPayloadV2 to abci engine", "error", err)
	}
	e.logger.Info("success in forwarding GetPayloadV2 to abci engine", "result", abciResult)

	compositeBlock := eetypes.NewCompositeBlock(gethResult.ExecutionPayload.BlockHash, abciResult.ExecutionPayload.BlockHash)
	e.interceptor.SaveCompositeBlock(compositeBlock)
	gethResult.ExecutionPayload.BlockHash = compositeBlock.Hash()
	e.logger.Info("created composite block:", "combined hash", compositeBlock.Hash(), "gethHash", gethResult.ExecutionPayload.BlockHash, "abciHash", abciResult.ExecutionPayload.BlockHash)

	compositeParent := eetypes.NewCompositeBlock(gethResult.ExecutionPayload.ParentHash, abciResult.ExecutionPayload.ParentHash)
	e.interceptor.SaveCompositeBlock(compositeParent)
	gethResult.ExecutionPayload.ParentHash = compositeParent.Hash()
	e.logger.Info("created composite parent:", "combined hash", compositeParent.Hash(), "gethHash", gethResult.ExecutionPayload.ParentHash, "abciHash", abciResult.ExecutionPayload.ParentHash)

	e.logger.Info("completed: GetPayloadV2", "error", err, "result", gethResult.ExecutionPayload)
	return &gethResult, err
}

// TODO(jim): Decompose the blockHash + parentHash as set in the result for GetPayloadV2
func (e *engineServer) NewPayloadV2(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	compositeBlockHash := e.interceptor.GetCompositeBlock(payload.BlockHash)
	compositeParentHash := e.interceptor.GetCompositeBlock(payload.ParentHash)

	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	payload.BlockHash = compositeBlockHash.GethHash
	payload.ParentHash = compositeParentHash.GethHash

	var gethResult eth.PayloadStatusV1
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "engine_newPayloadV2", payload)
	if err != nil {
		// TODO(jim): What do we do if geth for some reason errs and we dont?
		e.logger.Error("failed to forward NewPayloadV2 to geth engine", "error", err)
		return nil, err
	}

	e.logger.Info("forwarding NewPayloadV2 to abci engine")

	// TODO(jim): Is this re-use safe?
	payload.BlockHash = compositeBlockHash.ABCIHash
	payload.ParentHash = compositeParentHash.ABCIHash
	var abciResult eth.PayloadStatusV1
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "engine_newPayloadV2", payload)
	if err != nil {
		e.logger.Error("failed to forward NewPayloadV2 to abci engine", "error", err)
	}

	// Combine latestValidHash and save it.
	compositeLatestValidHash := eetypes.NewCompositeBlock(*gethResult.LatestValidHash, *abciResult.LatestValidHash)
	e.interceptor.SaveCompositeBlock(compositeLatestValidHash)
	compositeHash := compositeLatestValidHash.Hash()
	gethResult.LatestValidHash = &compositeHash

	e.logger.Info("completed: NewPayloadV2", "error", err, "result", &gethResult)
	return &gethResult, err
}

/* 'eth_' prefixed server methods, only required ones. */

// ethServer is the API for the eth like server.
// Implements required 'eth_' prefixed methods.
type ethServer struct {
	// client dials into op-geth server.
	// Might be best to not embed if we maybe want to add an sdk engine via rpc.
	blockStore BlockStore
	ethRPC     client.RPC
	peptideRPC client.RPC
	logger     log.Logger
}

// newEthAPI returns a new execEngineAPI.
func newEthAPI(blockStore BlockStore, ethRPC, peptideRPC client.RPC, logger log.Logger) *ethServer {
	return &ethServer{blockStore, ethRPC, peptideRPC, logger}
}

func (e *ethServer) ChainId() (hexutil.Big, error) { // nolint: revive, stylecheck
	e.logger.Info("trying: ChainID")

	var id hexutil.Big
	err := e.ethRPC.CallContext(context.TODO(), &id, "eth_chainId")

	e.logger.Info("completed: ChainID", "id", id, "error", err)
	return id, err
}

// Docu yanked from go-eth for fullTx.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (e *ethServer) GetBlockByNumber(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByNumber", "id", id)

	var gethResult map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "eth_getBlockByNumber", id, fullTx)
	if err != nil {
		e.logger.Error("failed to call geth", "error", err)
		// TODO(jim): What do we do if geth for some reason errs and we dont? This happens when
		// GetBlockByNumber is called with a label of 'finalized'. For some reason ABCI engine
		// does _not_ return an error.
		return nil, err
	}

	var abciResult map[string]any
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "eth_getBlockByNumber", id, fullTx)
	if err != nil {
		e.logger.Error("failed to call abci", "error", err)
	}

	// Combine the hashes and store the composite block, return the composite hash as the geth["hash"] field.
	// See monomers ToEthBlock for fields populated in the abci call.
	gethHash := common.HexToHash(gethResult["hash"].(string))
	abciHash := common.HexToHash(abciResult["hash"].(string))
	compositeBlock := eetypes.NewCompositeBlock(gethHash, abciHash)
	e.blockStore.SaveCompositeBlock(compositeBlock)

	gethResult["hash"] = compositeBlock.Hash()

	e.logger.Info("composite block", "compositeHash", compositeBlock.Hash().Hex())
	e.logger.Info("completed: GetBlockByNumber", "result", gethResult)
	return gethResult, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *ethServer) GetBlockByHash(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByHash", "id", id)

	hash := common.Hash{}
	switch id := id.(type) {
	case string:
		hash = common.HexToHash(id)
	case []byte:
		hash = common.BytesToHash(id)
	default:
		e.logger.Error("invalid type for id", "id", id)
	}
	compositeBlock := e.blockStore.GetCompositeBlock(hash)

	var gethResult map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "eth_getBlockByHash", compositeBlock.GethHash, fullTx)
	if err != nil {
		e.logger.Error("failed to call geth", "error", err)
		return nil, err
	}

	// NOTE: Do we even need to do forwarding? We don't use this block currently.
	var abciResult map[string]any
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "eth_getBlockByHash", compositeBlock.ABCIHash, fullTx)
	if err != nil {
		e.logger.Error("failed to call abci", "error", err)
		return nil, err
	}

	gethResult["hash"] = compositeBlock.Hash()

	e.logger.Info("completed: GetBlockByHash", "result", gethResult)
	return gethResult, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *ethServer) GetProof(address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (map[string]any, error) {
	e.logger.Info("trying: GetProof")

	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getProof", address, storageKeys, blockNrOrHash)

	e.logger.Info("completed: GetProof", "result", result)
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

// Added to be able to intercept and forward eth transactions.
func (e *ethServer) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	e.logger.Info("trying: SendRawTransaction")

	var result common.Hash
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_sendRawTransaction", data)

	e.logger.Info("completed: SendRawTransaction", "error", err, "result", result)
	return result, err
}

func (e *ethServer) MaxPriorityFeePerGas() (hexutil.Big, error) {
	e.logger.Info("trying: MaxPriorityFeePerGas")

	var result hexutil.Big
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_maxPriorityFeePerGas")

	e.logger.Info("completed: MaxPriorityFeePerGas", "result", result, "error", err)
	return result, err
}

func (e *ethServer) GetCode(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	e.logger.Info("trying: GetCode")

	var result hexutil.Bytes
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getCode", address, blockNrOrHash)

	e.logger.Info("completed: GetCode", "result", result, "error", err)
	return result, err
}

func (e *ethServer) EstimateGas(arg1 any) (hexutil.Uint64, error) {
	e.logger.Info("trying: EstimateGas")

	var result hexutil.Uint64
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_estimateGas", arg1)
	if err != nil {
		return 0, err
	}

	e.logger.Info("completed: EstimateGas", "result", result, "error", err)
	return result, nil
}

func (e *ethServer) GetTransactionCount(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	e.logger.Info("trying: GetTransactionCount")

	var result hexutil.Uint64
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getTransactionCount", address, blockNrOrHash)

	e.logger.Info("completed: GetTransactionCount", "result", result, "error", err)
	return result, err
}

func (e *ethServer) Call(msg any, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	e.logger.Info("trying: Call")

	var result hexutil.Bytes
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_call", msg, blockNrOrHash)

	e.logger.Info("completed: Call", "result", result, "error", err)
	return result, err
}
