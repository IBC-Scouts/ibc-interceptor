package api

import (
	"fmt"
	"log"
	"sync"

	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
)

type SdkEngine struct {
	logger cmtlog.Logger
	node   Node
	lock   sync.RWMutex
}

// TODO(jim): Lock! Also, warning re passing lock by value in api.go
func NewSdkEngine(node Node, logger cmtlog.Logger) SdkEngine {
	return SdkEngine{
		logger: logger,
		node:   node,
	}
}

func (e *SdkEngine) rollback(head *eetypes.Block, safeHash, finalizedHash eetypes.Hash) error {
	e.logger.Debug("engineAPIserver.rollback", "head", head.Height(), "safe", safeHash, "finalized", finalizedHash)

	getID := func(label string, hash eetypes.Hash) any {
		if hash != eetypes.ZeroHash {
			return hash.Bytes()
		}
		return label
	}
	safe, err := e.node.GetBlock(getID(eth.Safe, safeHash))
	if err != nil {
		return err
	}
	finalized, err := e.node.GetBlock(getID(eth.Finalized, finalizedHash))
	if err != nil {
		return err
	}
	return e.node.Rollback(head, safe, finalized)
}

func (e *SdkEngine) ForkchoiceUpdatedV1(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Debug("trying: ForkchoiceUpdatedV1")
	return e.ForkchoiceUpdatedV3(fcs, pa)
}

func (e *SdkEngine) ForkchoiceUpdatedV2(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Debug("trying: ForkchoiceUpdatedV2")
	return e.ForkchoiceUpdatedV3(fcs, pa)
}

func (e *SdkEngine) ForkchoiceUpdatedV3(
	fcs eth.ForkchoiceState,
	pa *eth.PayloadAttributes,
) (*eth.ForkchoiceUpdatedResult, error) {
	e.logger.Debug("trying: ForkchoiceUpdatedV3",
		"appHeight", e.node.LastBlockHeight()+1,
		"unsafe", fcs.HeadBlockHash.Hex(),
		"safe", fcs.SafeBlockHash.Hex(),
		"finalized", fcs.FinalizedBlockHash.Hex(),
		"attr", eetypes.HasPayloadAttributes(pa),
	)
	e.lock.Lock()
	defer e.lock.Unlock()

	headBlock, err := e.node.GetBlock(fcs.HeadBlockHash.Bytes())
	if err != nil {
		e.logger.Error("failed to get headBlock", "headBlockHash", fcs.HeadBlockHash.Hex(), "err", err)
		return nil, engine.InvalidForkChoiceState.With(err)
	}

	e.logger.Debug("ForkchoiceUpdatedV3",
		"appHeight", e.node.LastBlockHeight()+1,
		"fcu.unsafe.height", headBlock.Height(),
	)

	defer telemetry.IncrCounter(1, "query", "ForkchoiceUpdated")

	if eetypes.IsForkchoiceStateEmpty(&fcs) {
		return nil, engine.InvalidForkChoiceState.With(fmt.Errorf("forkchoice state is empty"))
	}

	// update labeled blocks

	reorg := false
	// When OpNode issues a FCU with a head block that's different than App's view, it means a reorg happened.
	// In this case, we need to rollback App and BlockStore to the head block's height-1.
	if headBlock.Height() != e.node.LastBlockHeight() {
		e.logger.Info("block head does not match the last sealed block [reorg from OpNode]", "head_height", headBlock.Height(), "app_height", e.node.LastBlockHeight())
		if err := e.rollback(headBlock, fcs.SafeBlockHash, fcs.FinalizedBlockHash); err != nil {
			e.logger.Error("rollback failed: %w", err)
			return nil, engine.InvalidForkChoiceState.With(err)
		}
		e.logger.Info("rollback succeeded", "head_height", headBlock.Height(), "app_height", e.node.LastBlockHeight())
		reorg = true
	}

	// update canonical block head
	e.logger.Info("updating unsafe/latest block", "hash", fcs.SafeBlockHash, "height", headBlock.Height())

	err = e.node.UpdateLabel(eth.Unsafe, fcs.HeadBlockHash)
	if err != nil {
		e.logger.Error("error updating unsafe block", "err", err)
	}

	if fcs.SafeBlockHash != eetypes.ZeroHash {
		e.logger.Info("updating safe block", "hash", fcs.SafeBlockHash)
		if err := e.node.UpdateLabel(eth.Safe, fcs.SafeBlockHash); err != nil {
			e.logger.Error("invalid safe head", "err", err)
			return nil, engine.InvalidForkChoiceState.With(err)
		}
	}

	// update finalized block head
	if fcs.FinalizedBlockHash != eetypes.ZeroHash {
		e.logger.Info("updating finalized block", "hash", fcs.FinalizedBlockHash)
		if err := e.node.UpdateLabel(eth.Finalized, fcs.FinalizedBlockHash); err != nil {
			e.logger.Error("invalid finalized head", "err", err)
			return nil, engine.InvalidForkChoiceState.With(err)
		}
	}

	// OpNode providing a new payload with reorg
	if reorg {
		payload := eetypes.NewPayload(pa, fcs.HeadBlockHash, e.node.LastBlockHeight()+1)
		payloadID, err := payload.GetPayloadID()
		if err != nil {
			return nil, engine.InvalidPayloadAttributes.With(err)
		}
		e.node.SavePayload(payload)
		e.logger.Info("engine reorg payload", "payload_id", payloadID, "payload_head_block_hash", fcs.HeadBlockHash, "store_head_block_hash", e.node.HeadBlockHash())
		// TODO: use one method for both cases: payload.Valid()
		return eetypes.ValidForkchoiceUpdateResult(&fcs.HeadBlockHash, payloadID), nil
	}

	// start new payload mode
	if eetypes.HasPayloadAttributes(pa) {
		// TODO check for invalid txs in pa
		payload := eetypes.NewPayload(pa, fcs.HeadBlockHash, e.node.LastBlockHeight()+1)
		payloadID, err := payload.GetPayloadID()
		if err != nil {
			return nil, engine.InvalidPayloadAttributes.With(err)
		}
		e.node.SavePayload(payload)
		e.logger.Info("engine saving new payload", "payload_id", payloadID, "payload_head_block_hash", fcs.HeadBlockHash, "store_head_block_hash", e.node.HeadBlockHash(), "headBlockHeight", headBlock.Height())
		return payload.Valid(payloadID), nil
	}

	// OpNode providing an existing payload, which only updates the head latest/unsafe block pointer
	// after reboot, in-mem payload cache is lost, causing OpNode failed to find Payload
	e.logger.Info("engine updating head block with existing payload", "hash", fcs.HeadBlockHash, "headBlockHeight", headBlock.Height())
	return eetypes.ValidForkchoiceUpdateResult(&fcs.HeadBlockHash, nil), nil
}

func (e *SdkEngine) GetPayloadV1(payloadID eetypes.PayloadID) (*eth.ExecutionPayload, error) {
	e.logger.Debug("GetPayloadV1", "payload_id", payloadID)
	pe, err := e.GetPayloadV3(payloadID)
	if err != nil {
		return nil, err
	}

	return pe.ExecutionPayload, nil
}

func (e *SdkEngine) GetPayloadV2(payloadID eetypes.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	e.logger.Debug("GetPayloadV2", "payload_id", payloadID)
	return e.GetPayloadV3(payloadID)
}

// OpNode sequencer calls this API to seal a new block
func (e *SdkEngine) GetPayloadV3(payloadID eetypes.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	newBlockHeight := e.node.LastBlockHeight() + 1
	e.logger.Debug("GetPayloadV3", "payload_id", payloadID, "newBlockHeight", newBlockHeight)

	defer telemetry.IncrCounter(1, "query", "GetPayload")

	payload, ok := e.node.GetPayload(payloadID)
	if !ok {
		return nil, eetypes.UnknownPayload
	}
	if payload != e.node.CurrentPayload() {
		e.logger.Error("payload is not current", "payload_id", payloadID, "newBlockHeight", newBlockHeight)
		return nil, engine.InvalidParams.With(fmt.Errorf("payload is not current"))
	}

	// e.mutex.Lock()
	// defer e.mutex.Unlock()

	// e.debugL1UserTxs(payload.Attrs.Transactions, "EngineGetPayload")

	// TODO: handle time slot based block production
	// for now assume block is sealed by this call
	err := e.node.CommitBlock()
	// TODO error handling
	if err != nil {
		e.logger.Error("failed to commit block", "err", err)
		log.Panicf("failed to commit block: %v", err)
	}

	return payload.ToExecutionPayloadEnvelope(e.node.HeadBlockHash()), nil
}

func (e *SdkEngine) NewPayloadV1(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Debug("trying: NewPayloadV1", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex(), "height", e.node.LastBlockHeight()+1)
	return e.NewPayloadV3(payload)
}

func (e *SdkEngine) NewPayloadV2(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Debug("trying: NewPayloadV2", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex(), "height", e.node.LastBlockHeight()+1)
	return e.NewPayloadV3(payload)
}

func (e *SdkEngine) NewPayloadV3(payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.logger.Debug("trying: NewPayloadV3", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex(), "height", e.node.LastBlockHeight()+1)
	e.lock.Lock()
	defer e.lock.Unlock()
	defer telemetry.IncrCounter(1, "query", "NewPayload")

	e.logger.Debug("NewPayloadV3", "payload.ID", payload.ID(), "blockHash", payload.BlockHash.Hex(), "height", e.node.LastBlockHeight()+1)

	if _, err := e.node.GetBlock(payload.BlockHash.Bytes()); err != nil {
		e.logger.Error("Engine.NewPayload: failed to get block", "blockHash", payload.BlockHash.Hex(), "err", err)
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalidBlockHash},
			engine.InvalidParams.With(err)
	}
	headBlockHash := e.node.HeadBlockHash()
	return &eth.PayloadStatusV1{
		Status:          eth.ExecutionValid,
		LatestValidHash: &headBlockHash,
	}, nil
}
