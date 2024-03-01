package api

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const ibcBridgeAddress = "0x42000000000000000000000000000000000000E1"

// EngineForkStates takes in the interceptor fork state and the blockstore and creates two states for abci and
// geth.
func EngineForkStates(blockStore BlockStore, interceptorForkState eth.ForkchoiceState) (eth.ForkchoiceState, eth.ForkchoiceState) {
	head, safe, finalized := interceptorForkState.HeadBlockHash, interceptorForkState.SafeBlockHash, interceptorForkState.FinalizedBlockHash

	abci := eth.ForkchoiceState{
		HeadBlockHash:      blockStore.GetCompositeBlock(head).ABCIHash,
		SafeBlockHash:      blockStore.GetCompositeBlock(safe).ABCIHash,
		FinalizedBlockHash: blockStore.GetCompositeBlock(finalized).ABCIHash,
	}

	geth := eth.ForkchoiceState{
		HeadBlockHash:      blockStore.GetCompositeBlock(head).GethHash,
		SafeBlockHash:      blockStore.GetCompositeBlock(safe).GethHash,
		FinalizedBlockHash: blockStore.GetCompositeBlock(finalized).GethHash,
	}

	return abci, geth
}

// IsIBCBridgeTx returns true if the transaction is a transaction sent to the IBCStandardBridge.
// tx is a hex encoded marshalled types.Transaction (op-geth/core/types/transaction.go)
func IsIBCBridgeTx(data hexutil.Bytes) bool {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(data); err != nil {
		// This will fail in op-geth if we can't unmarshal so, just
		// return false here.
		return false
	}

	return tx.To().Hex() == ibcBridgeAddress
}
