package types

import "github.com/ethereum-optimism/optimism/op-service/eth"

// EngineServer is the API for the execution engine.
// Based on https://github.com/ethereum-optimism/op-geth/blob/f33198a04ad51e89570d646cf3802a493c93e0b2/eth/catalyst/api.go#L40
type EngineServer interface {
	ForkchoiceUpdatedV1(eth.ForkchoiceState, *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedV2(eth.ForkchoiceState, *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedV3(eth.ForkchoiceState, *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	GetPayloadV1(eth.PayloadID) (*eth.ExecutionPayload, error)
	GetPayloadV2(eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV3(eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	NewPayloadV1(*eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	NewPayloadV2(*eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	NewPayloadV3(*eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
}
