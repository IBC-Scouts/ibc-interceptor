// Implement the wrapping calls to the wrapped engines. Forwards the RPC call to op-geth server
// while directly calling the cosmos sdk engine. See https://ethereum.github.io/execution-apis/api-documentation/
package api

// TODO(jim):
// - Document the methods with at least the info in the api-documentation link above.
// - A generic function can be added that just forwards calls in all the wrapping methods.

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/cometbft/cometbft/libs/log"

	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
)

// TODO(jim): passed by lock.
func GetEngineAPI(interceptor Interceptor, ethPRC, peptideRPC client.RPC, logger log.Logger) []rpc.API {
	return []rpc.API{
		{
			Namespace: "engine",
			Service:   newEngineAPI(interceptor, ethPRC, peptideRPC, logger),
		},
	}
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

func (e *engineServer) NewPayloadV2(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	compositeBlockHash := e.interceptor.GetCompositeBlock(payload.BlockHash)
	compositeParentHash := e.interceptor.GetCompositeBlock(payload.ParentHash)

	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	payload.BlockHash = compositeBlockHash.GethHash
	payload.ParentHash = compositeParentHash.GethHash

	var gethResult eth.PayloadStatusV1
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "engine_newPayloadV2", payload)
	if err != nil {
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
