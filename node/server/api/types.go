// types.go holds any additional required type definitions for the server implementations.
package api

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
)

type (
	Hash  = eetypes.Hash
	Block = eetypes.Block
)

type Interceptor interface {
	MempoolNode
	BlockStore
	PayloadStore
}

// MempoolNode allows accessing/modifying/inspecting the mempool.
type MempoolNode interface {
	// HasMsgs returns true if the mempool has messages.
	HasMsgs() bool
	// GetMsgs returns all messages in the mempool.
	GetMsgs() [][]byte
	// ClearMsgs clears all messages from the mempool.
	ClearMsgs()
	// AddMsgToMempool adds a message to the mempool.
	AddMsgToMempool(bz []byte)
}

// BlockStore allows accessing/modifying/inspecting the compose blocks.
type BlockStore interface {
	GetCompositeBlock(common.Hash) eetypes.CompositeBlock
	SaveCompositeBlock(eetypes.CompositeBlock)
}

type PayloadStore interface {
	GetCompositePayload(eth.PayloadID) eetypes.CompositePayload
	SaveCompositePayload(eetypes.CompositePayload)
}

// TODO(jim): Ethereum JSON/RPC dictates responses should either return 0, 1 (response or error) or 2 (response and error).
// For now, we return 2 just to keep separated.
type SendCosmosTxResult struct{}
