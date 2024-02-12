package api

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ibc-scouts/ibc-interceptor/client/geth"

	"github.com/cometbft/cometbft/libs/log"
)

// The public rpc methods are prefixed by the namespace (lower case) followed by all exported
// methods of the "service" in camelcase
func GetExecutionEngineAPIs(execEngine *geth.GethWrappedClient, logger log.Logger) []rpc.API {
	if execEngine == nil {
		panic("execEngine is nil")
	}
	if logger == nil {
		panic("logger is nil")
	}

	node := newExecutionEngineAPI(execEngine, logger)
	apis := []rpc.API{
		{
			Namespace: "engine",
			Service:   node,
		},
		{
			Namespace: "eth",
			Service:   node,
		},
	}

	return apis
}

// execEngineAPI is the API for the execution engine.
// Implements the methods prefixed with "engine_" defined in
// https://ethereum.github.io/execution-apis/api-documentation/
type execEngineAPI struct {
	gethClient *geth.GethWrappedClient
	logger     log.Logger
	// lock   sync.RWMutex
}

// newExecutionEngineAPI returns a new execEngineAPI.
func newExecutionEngineAPI(engine *geth.GethWrappedClient, logger log.Logger) *execEngineAPI {
	return &execEngineAPI{
		gethClient: engine,
		logger:     logger,
	}
}

func (e *execEngineAPI) ForkchoiceUpdatedV1(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV1, forwarding to geth", "fcs", fcs, "pa", pa)
	// TODO: may add timeout to the context or if anything else is needed
	ctx := context.TODO()

	var result eth.ForkchoiceUpdatedResult
	err := e.gethClient.Client.CallContext(ctx, &result, "engine_forkchoiceUpdatedV1", fcs, pa)

	e.logger.Info("completed: ForkchoiceUpdatedV1", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineAPI) ForkchoiceUpdatedV2(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV2", "fcs", fcs, "pa", pa)
	// TODO: may add timeout to the context or if anything else is needed
	ctx := context.TODO()

	var result eth.ForkchoiceUpdatedResult
	err := e.gethClient.Client.CallContext(ctx, &result, "engine_forkchoiceUpdatedV2", fcs, pa)

	e.logger.Info("completed: ForkchoiceUpdatedV2", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineAPI) ForkchoiceUpdatedV3(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Info("trying: ForkchoiceUpdatedV3", "fcs", fcs, "pa", pa)
	// TODO: may add timeout to the context or if anything else is needed
	ctx := context.TODO()

	var result eth.ForkchoiceUpdatedResult
	err := e.gethClient.Client.CallContext(ctx, &result, "engine_forkchoiceUpdatedV3", fcs, pa)

	e.logger.Info("completed: ForkchoiceUpdatedV3", "error", err, "result", &result)

	return &result, err

	// return e.gethClient.ForkchoiceUpdate(ctx, &fcs, &pa)
}

func (e *execEngineAPI) GetPayloadV1(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Info("GetPayloadV1", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "engine_getPayloadV1", payloadID)

	e.logger.Info("completed: GetPayloadV1", "error", err, "result", &result)

	return result.ExecutionPayload, err
}

func (e *execEngineAPI) GetPayloadV2(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Info("GetPayloadV2", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "engine_getPayloadV2", payloadID)

	e.logger.Info("completed: GetPayloadV2", "error", err, "result", &result)

	return result.ExecutionPayload, err
}

func (e *execEngineAPI) GetPayloadV3(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Info("GetPayloadV3", "payload_id", payloadID)

	var result eth.ExecutionPayloadEnvelope
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "engine_getPayloadV3", payloadID)

	e.logger.Info("completed: GetPayloadV3", "error", err, "result", &result)

	return result.ExecutionPayload, err

	// return e.gethClient.GetPayload(context.TODO(), payloadID)
}

func (e *execEngineAPI) NewPayloadV1(payload eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "engine_newPayloadV1", payload)

	e.logger.Info("completed: NewPayloadV1", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineAPI) NewPayloadV2(payload eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "engine_newPayloadV2", payload)

	e.logger.Info("completed: NewPayloadV2", "error", err, "result", &result)

	return &result, err
}

func (e *execEngineAPI) NewPayloadV3(payload eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Info("trying: NewPayloadV3", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())

	var result eth.PayloadStatusV1
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "engine_newPayloadV3", payload)

	e.logger.Info("completed: NewPayloadV3", "error", err, "result", &result)

	return &result, err

	// return e.gethClient.NewPayload(context.TODO(), &payload)
}

func (e *execEngineAPI) ChainId() (hexutil.Big, error) {
	e.logger.Info("trying: ChainID")

	var id hexutil.Big
	err := e.gethClient.Client.CallContext(context.TODO(), &id, "eth_chainId")

	return id, err
}

// TODO(jim): Change the name from 'something' :D (look up eth_getBlockByNumber in rpc docs)
func (e *execEngineAPI) GetBlockByNumber(id any, something any) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByNumber", "id", id)

	var result map[string]any
	err := e.gethClient.Client.CallContext(context.TODO(), &result, "eth_getBlockByNumber", id, something)

	return result, err
}
