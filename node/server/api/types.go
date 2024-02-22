// types.go holds any additional required type definitions for the server implementations.
package api

import eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"

type (
	Hash  = eetypes.Hash
	Block = eetypes.Block
)

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

// TODO(jim): Ethereum JSON/RPC dictates responses should either return 0, 1 (response or error) or 2 (response and error).
// For now, we return 2 just to keep separated.
type SendCosmosTxResult struct{}
