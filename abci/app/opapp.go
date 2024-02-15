package app

import (
	"sync"

	"github.com/cometbft/cometbft/libs/log"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/ibc-scouts/ibc-interceptor/abci/types"
)

// OpApp is a wrapper around the SimApp that provides required implementations for
// compatibility with optimism.
//
//nolint:unused // TODO: remove this once the OpApp is fully implemented.
type OpApp struct {
	*SimApp

	genesis types.OpGenesis

	logger log.Logger

	lock sync.RWMutex

	ps types.PayloadStore
	// txMempool is a basic Mempool to be used in OpApp.
	txMempool cmttypes.Txs
}

// AddTxToMempool adds txs to the mempool.
func (oa *OpApp) AddTxToMempool(tx cmttypes.Tx) {
	oa.lock.Lock()
	defer oa.lock.Unlock()

	oa.txMempool = append(oa.txMempool, tx)
}
