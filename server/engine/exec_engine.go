package engine

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"

	eth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"

	"github.com/cometbft/cometbft/libs/log"
)

// The public rpc methods are prefixed by the namespace (lower case) followed by all exported
// methods of the "service" in camelcase
func GetExecutionEngineAPIs(gethClient *sources.EngineClient, logger log.Logger) []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "engine",
			Service:   &execEngineAPI{gethClient: gethClient, logger: logger},
		},
		// {
		// 	Namespace: "eth",
		// 	Service:   &ethLikeServer{node: node, logger: logger.With("module", "eth")},
		// },
		// {
		// 	Namespace: "pep",
		// 	Service:   &peptideServer{node: node, logger: logger.With("module", "peptide")},
		// },
	}
	// if enabledApis.IsAdminApiEnabled() {
	// 	apis = append(apis, rpc.API{
	// 		Namespace: "admin",
	// 		Service:   &adminServer{node: node, logger: logger},
	// 	})
	// }
	return apis
}

// execEngineAPI is the API for the execution engine.
// Implements the methods prefixed with "engine_" defined in
// https://ethereum.github.io/execution-apis/api-documentation/
type execEngineAPI struct {
	gethClient *sources.EngineClient
	logger     log.Logger
	// lock   sync.RWMutex
}

func (e *execEngineAPI) ForkchoiceUpdatedV1(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Debug("trying: ForkchoiceUpdatedV1, forwarding to V3")
	return e.ForkchoiceUpdatedV3(fcs, pa)
}

func (e *execEngineAPI) ForkchoiceUpdatedV2(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Debug("trying: ForkchoiceUpdatedV2, forwarding to V3")
	return e.ForkchoiceUpdatedV3(fcs, pa)
}

func (e *execEngineAPI) ForkchoiceUpdatedV3(
	fcs eth.ForkchoiceState,
	pa eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Debug("trying: ForkchoiceUpdatedV3, forwarding to geth")
	// TODO: may add timeout to the context or if anything else is needed
	ctx := context.TODO()
	return e.gethClient.ForkchoiceUpdate(ctx, &fcs, &pa)
}

func (e *execEngineAPI) GetPayloadV1(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Debug("trying: GetPayloadV1, forwarding to V3")
	return e.GetPayloadV3(payloadID)
}

func (e *execEngineAPI) GetPayloadV2(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Debug("trying: GetPayloadV2, forwarding to V3")
	return e.GetPayloadV3(payloadID)
}

func (e *execEngineAPI) GetPayloadV3(payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Debug("GetPayloadV3", "payload_id", payloadID)
	return e.gethClient.GetPayload(context.TODO(), payloadID)
}

func (e *execEngineAPI) NewPayloadV1(payload eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Debug("trying: NewPayloadV1, forwarding to V3")
	return e.NewPayloadV3(payload)
}

func (e *execEngineAPI) NewPayloadV2(payload eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Debug("trying: NewPayloadV2, forwarding to V3")
	return e.NewPayloadV3(payload)
}

func (e *execEngineAPI) NewPayloadV3(payload eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Debug("trying: NewPayloadV3", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex())
	return e.gethClient.NewPayload(context.TODO(), &payload)
}
