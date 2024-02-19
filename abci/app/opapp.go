package app

import (
	"sync"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	db "github.com/cosmos/cosmos-db"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// OpApp is a wrapper around the SimApp that provides required implementations for
// compatibility with optimism.
//
//nolint:unused // TODO: remove this once the OpApp is fully implemented.
type OpApp struct {
	*SimApp

	logger log.Logger

	lock sync.RWMutex

	ValSet        *cmttypes.ValidatorSet
	lastHeader    *tmproto.Header
	currentHeader *tmproto.Header
	ChainId       string
}

func NewOpApp(chainID string, dir string, db db.DB, logger log.Logger) *OpApp {
	return &OpApp{
		logger:  logger, // TODO(jim): create a logger
		SimApp:  nil,    // TODO(jim): Create new simapp.
		ChainId: chainID,
	}
}

// This is what initiates the chain app initialisation. It's only meant to be called when the genesis is
// being sealed so the genesis block can be produced.
// - It triggers a call into the base app's InitChain()
// - Commits the app state to disk so it can be persisted across executions
// - Returns a "genesis header" with the genesis block height and app state hash
func (oa *OpApp) Init(appState []byte, initialHeight int64, genesisTime time.Time) *tmproto.Header {
	response, err := oa.InitChain(&abci.RequestInitChain{
		ChainId:         oa.ChainId,
		ConsensusParams: DefaultConsensusParams,
		AppStateBytes:   appState,
		Time:            genesisTime,
		InitialHeight:   initialHeight,
	})
	if err != nil {
		panic(err)
	}

	// this will store the app state into disk. Failing to call this will result in missing data the next
	// time the app is called
	oa.Commit()

	// use LastBlockHeight() since it might not be the same as InitialHeight.
	return &tmproto.Header{
		Height:             oa.LastBlockHeight(),
		ValidatorsHash:     oa.ValSet.Hash(),
		NextValidatorsHash: oa.ValSet.Hash(),
		ChainID:            oa.ChainId,
		Time:               genesisTime,
		AppHash:            response.AppHash,
	}
}

// LastBlockHeight implements the Node interface.
func (oa *OpApp) LastBlockHeight() int64 {
	return oa.SimApp.LastBlockHeight()
}

// Last header is header that was commited, either as a genesis block header or the latest commited block header
func (oa *OpApp) LastHeader() *tmproto.Header {
	return oa.lastHeader
}

// Commit pending changes to chain state and start a new block.
// Will error if there is no deliverState, eg. InitChain is not called before first block.
func (oa *OpApp) CommitAndBeginNextBlock(timestamp eth.Uint64Quantity) {
	_, _ = oa.Commit()
	oa.OnCommit(timestamp)
}

// OnCommit updates the last header and current header after App Commit or InitChain
func (oa *OpApp) OnCommit(timestamp eth.Uint64Quantity) {
	// update last header to the committed time and app hash
	lastHeader := oa.currentHeader
	lastHeader.Time = time.Unix(int64(timestamp), 0)
	lastHeader.AppHash = oa.LastCommitID().Hash
	oa.lastHeader = lastHeader

	// start a new partial header for next round
	oa.currentHeader = &tmproto.Header{
		Height:             oa.LastBlockHeight() + 1,
		ValidatorsHash:     oa.ValSet.Hash(),
		NextValidatorsHash: oa.ValSet.Hash(),
		ChainID:            oa.ChainId,
		Time:               time.Unix(int64(timestamp), 0),
	}
}
