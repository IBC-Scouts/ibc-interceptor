package app

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ibc-scouts/ibc-interceptor/abci/engine"
	"github.com/ibc-scouts/ibc-interceptor/abci/types"
)

var _ engine.Node = (*OpApp)(nil)

// OpApp is a wrapper around the SimApp that provides required implementations for
// compatibility with optimism.
//
//nolint:unused // TODO: remove this once the OpApp is fully implemented.
type OpApp struct {
	*SimApp

	genesis types.OpGenesis

	logger cmtlog.Logger

	lock sync.RWMutex

	ps types.PayloadStore
	// txMempool is a basic Mempool to be used in OpApp.
	txMempool cmttypes.Txs

	bs          types.BlockStore
	latestBlock *types.Block // latest mined block that may not be canonical

	ValSet               *tmtypes.ValidatorSet
	lastHeader           *tmproto.Header
	currentHeader        *tmproto.Header
}

// AddTxToMempool adds txs to the mempool.
func (oa *OpApp) AddTxToMempool(tx cmttypes.Tx) {
	oa.lock.Lock()
	defer oa.lock.Unlock()

	oa.txMempool = append(oa.txMempool, tx)
}

// LastBlockHeight implements the Node interface.
func (oa *OpApp) LastBlockHeight() int64 {
	return oa.SimApp.LastBlockHeight()
}

// GetBlock implements the Node interface.
func (oa *OpApp) GetBlock(id any) (*types.Block, error) {
	oa.logger.Info("trying: OpApp.GetBlock", "id", id)
	oa.lock.RLock()
	defer oa.lock.RUnlock()
	oa.logger.Info("OpApp.GetBlock", "id", id)

	block, err := func() (types.BlockData, error) {
		switch v := id.(type) {
		case nil:
			return oa.bs.BlockByLabel(eth.Unsafe), nil
		case []byte:
			return oa.bs.BlockByHash(types.Hash(v)), nil
		case int64:
			return oa.getBlockByNumber(v), nil
		// sometimes int values are weirdly converted to float?
		case float64:
			return oa.getBlockByNumber(int64(v)), nil
		case string:
			return oa.getBlockByString(v), nil
		default:
			return nil, fmt.Errorf("cannot query block by value %v (%T)", v, id)
		}
	}()
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, ethereum.NotFound
	}
	return types.MustInferBlock(block), nil
}

func (oa *OpApp) getBlockByString(str string) types.BlockData {
	// use base 0 so it's autodetected
	number, err := strconv.ParseInt(str, 0, 64)
	if err == nil {
		return oa.getBlockByNumber(number)
	}
	// When block number is ethrpc.PendingBlockNumber, optimsim expects the latest block.
	// See https://github.com/ethereum-optimism/optimism/blob/v1.2.0/op-e2e/system_test.go#L1353
	// The ethclient converts negative int64 numbers to their respective labels and that's what
	// the server (us) gets. i.e. ethrpc.PendingBlockNumber (-1) is converted to "pending"
	// See https://github.com/ethereum-optimism/op-geth/blob/v1.101304.1/rpc/types.go
	// Since "pending" is no a label we use elsewhere, we need to check for it here
	// and returna the latest (unsafe) block
	if str == "pending" {
		return oa.bs.BlockByLabel(eth.Unsafe)
	}
	return oa.bs.BlockByLabel(eth.BlockLabel(str))
}

func (oa *OpApp) getBlockByNumber(number int64) types.BlockData {
	switch ethrpc.BlockNumber(number) {
	// optimism expects these two to be the same
	case ethrpc.PendingBlockNumber, ethrpc.LatestBlockNumber:
		return oa.bs.BlockByLabel(eth.Unsafe)
	case ethrpc.SafeBlockNumber:
		return oa.bs.BlockByLabel(eth.Safe)
	case ethrpc.FinalizedBlockNumber:
		return oa.bs.BlockByLabel(eth.Finalized)
	case ethrpc.EarliestBlockNumber:
		return oa.bs.BlockByHash(oa.genesis.GenesisBlock.Hash)
	default:
		return oa.bs.BlockByNumber(number)
	}
}

// GetChainID implements the Node interface.
func (oa *OpApp) GetChainID() string {
	return oa.genesis.ChainID
}

// HeadBlockHash implements the Node interface.
func (oa *OpApp) HeadBlockHash() types.Hash {
	oa.lock.RLock()
	defer oa.lock.RUnlock()
	return oa.latestBlock.Hash()
}

// CommitBlock implements the Node interface.
func (oa *OpApp) CommitBlock() error {
	oa.lock.Lock()
	defer oa.lock.Unlock()
	oa.commitBlockAndUpdateNodeInfo()
	return nil
}

// commitBlockAndUpdateNodeInfo simulates committing current block and updates node info.
//
// Need to combine startBuildingBlock and CommitAndBeginNextBlock
// https://github.com/cosmos/ibc-go/blob/main/testing/chain.go#L356
// https://github.com/cosmos/ibc-go/blob/main/testing/simapp/test_helpers.go#L130
func (oa *OpApp) commitBlockAndUpdateNodeInfo() {
	block := oa.startBuildingBlock()

	oa.CommitAndBeginNextBlock(oa.ps.Current().Attrs.Timestamp)
	block = oa.sealBlock(block)

	oa.bs.AddBlock(block)
	oa.latestBlock = block
}

// Commit pending changes to chain state and start a new block.
// Will error if there is no deliverState, eg. InitChain is not called before first block.
func (oa *OpApp) CommitAndBeginNextBlock(timestamp eth.Uint64Quantity) {
	oa.Commit()
	oa.OnCommit(timestamp)
}

// startBuildingBlock starts building a new block for App is committed.
func (oa *OpApp) startBuildingBlock() *types.Block {
	// fill in block fields with L1 data
	block := oa.fillBlockWithL1Data(&types.Block{})
	oa.applyL1Txs(block)
	oa.applyBlockL2Txs(block)
	return block
}

// fillBlockWithL1Data fills in block fields with L1 data.
func (oa *OpApp) fillBlockWithL1Data(block *types.Block) *types.Block {
	if oa.ps.Current() != nil {
		// must include L1Txs for L2 block's L1Origin
		block.L1Txs = oa.ps.Current().Attrs.Transactions
		block.Withdrawals = oa.ps.Current().Attrs.Withdrawals
	} else {
		oa.logger.Error("currentPayload is nil for non-genesis block", "blockHeight", block.Height())
		log.Panicf("currentPayload is nil for non-genesis block with height %d", block.Height())
	}

	return block
}

// applyL1Txs applies L1 txs to the block that's currently being built.
func (oa *OpApp) applyL1Txs(block *types.Block) {
	// TODO: we don't need to apply L1 txs in the proof-of-concept
}

// applyBlockL2Txs applies L2 txs to the block that's currently being built.
func (oa *OpApp) applyBlockL2Txs(block *types.Block) {
	block.Txs = oa.txMempool
	oa.txMempool = nil

	oa.FinalizeBlock(&abcitypes.RequestFinalizeBlock{
		Height:             oa.LastBlockHeight() + 1,
		Time:               time.Unix(int64(block.Header.Time), 0),
		NextValidatorsHash: block.Header.NextValidatorsHash,
		Txs:                block.Txs.ToSliceOfBytes(),
	})
}

// OnCommit updates the last header and current header after App Commit or InitChain
func (oa *OpApp) OnCommit(timestamp eth.Uint64Quantity) {
	// TODO: don't know if we need to track the headers like polymer
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
		ChainID:            oa.GetChainID(),
		Time:               time.Unix(int64(timestamp), 0),
	}
}

// sealBlock finishes building current L2 block from currentHeader, L2 txs in mempool, and L1 txs from
// payloadAttributes.
//
// sealBlock should be called after chainApp's committed. So chainApp.LastBlockHeight is the sealed block's height
func (oa *OpApp) sealBlock(block *types.Block) *types.Block {
	oa.logger.Info("seal block", "height", oa.LastBlockHeight())

	// finalize block fields
	header := types.Header{}
	block.Header = header.Populate(oa.lastHeader)

	payload := oa.ps.Current()
	if payload != nil {
		block.GasLimit = *payload.Attrs.GasLimit
		block.Header.Time = uint64(payload.Attrs.Timestamp)
		block.PrevRandao = payload.Attrs.PrevRandao
		block.Withdrawals = payload.Attrs.Withdrawals
	}

	block.ParentBlockHash = oa.findParentHash()

	// if err := oa.indexAndPublishAllTxs(block); err != nil {
	// 	oa.logger.Error("failed to index and publish txs", "err", err)
	// }
	// // reset txResults
	// oa.txResults = nil

	return block
}

func (oa *OpApp) findParentHash() types.Hash {
	lastBlock := oa.bs.BlockByNumber(oa.LastBlockHeight() - 1)
	if lastBlock != nil {
		return lastBlock.Hash()
	}
	// TODO: handle cases where non-genesis block is missing
	return types.ZeroHash
}
