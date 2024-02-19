package node

import (
	"sync"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	nodeclient "github.com/ibc-scouts/ibc-interceptor/node/client"
	"github.com/ibc-scouts/ibc-interceptor/node/server"
	"github.com/ibc-scouts/ibc-interceptor/node/server/api"
	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

const (
	DefaultGasLimit  = 30_000_000
	AppStateDbName   = "appstate"
	BlockStoreDbName = "blockstore"
	TxStoreDbName    = "txstore"
)

type InterceptorNode struct {
	// eeServer is the RPC server for the Execution Engine
	eeServer *server.EERPCServer
	// ethRPC is the RPC client for the Ethereum node
	ethRPC client.RPC
	// peptideRPC is the RPC client for the Peptide node
	peptideRPC client.RPC

	logger types.CompositeLogger
	lock   sync.RWMutex
	ps     eetypes.PayloadStore
	// txMempool is a basic Mempool to be used in OpApp.
	txMempool cmttypes.Txs
	bs        eetypes.BlockStore
	// latest mined block that may not be canonical
	latestBlock *eetypes.Block
}

func NewInterceptorNode(config *types.Config) *InterceptorNode {
	logger, err := config.GetLogger("module", "interceptor")
	if err != nil {
		panic(err)
	}

	// create the geth client based on address passed in via command line.
	ethRPC, err := nodeclient.NewRPCClient(config.GethEngineAddr, config.GethAuthSecret, logger.New("client", "op-geth"))
	if err != nil {
		panic(err)
	}
	peptideRPC, err := nodeclient.NewRPCClient(config.PeptideEngineAddr, nil, logger.New("client", "peptide"))
	if err != nil {
		panic(err)
	}

	rpcServerConfig := server.DefaultConfig(config.EngineServerAddr)

	node := &InterceptorNode{
		logger:     logger,
		ethRPC:     ethRPC,
		peptideRPC: peptideRPC,
	}

	rpcAPIs := api.GetAPIs(ethRPC, peptideRPC, logger.With("server", "exec_engine_api"))
	eeServer := server.NewEeRPCServer(rpcServerConfig, rpcAPIs, logger.With("server", "exec_engine_rpc"))

	node.eeServer = eeServer

	return node
}

func (n *InterceptorNode) Start() error {
	if err := n.eeServer.Start(); err != nil {
		return err
	}

	return nil
}

func (n *InterceptorNode) Stop() error {
	if err := n.eeServer.Stop(); err != nil {
		return err
	}

	n.ethRPC.Close()
	n.peptideRPC.Close()

	return nil
}

// AddTxToMempool adds txs to the mempool.
func (n *InterceptorNode) AddTxToMempool(tx cmttypes.Tx) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.txMempool = append(n.txMempool, tx)
}

/*

// LastBlockHeight implements the Node interface.
func (n *InterceptorNode) LastBlockHeight() int64 {
	return n.app.LastBlockHeight()
}

// GetBlock implements the Node interface.
func (n *InterceptorNode) GetBlock(id any) (*eetypes.Block, error) {
	n.logger.Info("trying: OpApp.GetBlock", "id", id)
	n.lock.RLock()
	defer n.lock.RUnlock()
	n.logger.Info("OpApp.GetBlock", "id", id)

	block, err := func() (eetypes.BlockData, error) {
		switch v := id.(type) {
		case nil:
			return n.bs.BlockByLabel(eth.Unsafe), nil
		case []byte:
			return n.bs.BlockByHash(eetypes.Hash(v)), nil
		case int64:
			return n.getBlockByNumber(v), nil
		// sometimes int values are weirdly converted to float?
		case float64:
			return n.getBlockByNumber(int64(v)), nil
		case string:
			return n.getBlockByString(v), nil
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
	return eetypes.MustInferBlock(block), nil
}

func (n *InterceptorNode) getBlockByString(str string) eetypes.BlockData {
	// use base 0 so it's autodetected
	number, err := strconv.ParseInt(str, 0, 64)
	if err == nil {
		return n.getBlockByNumber(number)
	}
	// When block number is ethrpc.PendingBlockNumber, optimsim expects the latest block.
	// See https://github.com/ethereum-optimism/optimism/blob/v1.2.0/op-e2e/system_test.go#L1353
	// The ethclient converts negative int64 numbers to their respective labels and that's what
	// the server (us) gets. i.e. ethrpc.PendingBlockNumber (-1) is converted to "pending"
	// See https://github.com/ethereum-optimism/op-geth/blob/v1.101304.1/rpc/types.go
	// Since "pending" is no a label we use elsewhere, we need to check for it here
	// and returna the latest (unsafe) block
	if str == "pending" {
		return n.bs.BlockByLabel(eth.Unsafe)
	}
	return n.bs.BlockByLabel(eth.BlockLabel(str))
}

func (n *InterceptorNode) getBlockByNumber(number int64) eetypes.BlockData {
	switch ethrpc.BlockNumber(number) {
	// optimism expects these two to be the same
	case ethrpc.PendingBlockNumber, ethrpc.LatestBlockNumber:
		return n.bs.BlockByLabel(eth.Unsafe)
	case ethrpc.SafeBlockNumber:
		return n.bs.BlockByLabel(eth.Safe)
	case ethrpc.FinalizedBlockNumber:
		return n.bs.BlockByLabel(eth.Finalized)
	case ethrpc.EarliestBlockNumber:
		return n.bs.BlockByHash(n.genesis.GenesisBlock.Hash)
	default:
		return n.bs.BlockByNumber(number)
	}
}

// GetChainID implements the Node interface.
func (n *InterceptorNode) GetChainID() string {
	return n.genesis.ChainID
}

// HeadBlockHash implements the Node interface.
func (n *InterceptorNode) HeadBlockHash() eetypes.Hash {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.latestBlock.Hash()
}

// CommitBlock implements the Node interface.
func (oa *InterceptorNode) CommitBlock() error {
	oa.lock.Lock()
	defer oa.lock.Unlock()
	oa.commitBlockAndUpdateNodeInfo()
	return nil
}

// UpdateLabel implements the Node interface.
func (oa *InterceptorNode) UpdateLabel(label eth.BlockLabel, hash eetypes.Hash) error {
	oa.logger.Debug("trying: OpApp.UpdateLabel", "label", label, "hash", hash)
	oa.lock.Lock()
	defer oa.lock.Unlock()
	oa.logger.Debug("OpApp.UpdateLabel", "label", label, "hash", hash)

	return oa.bs.UpdateLabel(label, hash)
}

// SavePayload implements the Node interface.
// It saves the payload by its ID if it's not already in payload cache.
// Also update the latest Payload if this is a new payload
//
// payload must be valid
func (oa *InterceptorNode) SavePayload(payload *eetypes.Payload) {
	_ = oa.ps.Add(payload)
}

// GetPayload implements the Node interface.
func (oa *InterceptorNode) GetPayload(payloadID eetypes.PayloadID) (*eetypes.Payload, bool) {
	return oa.ps.Get(payloadID)
}

// CurrentPayload implements the Node interface.
func (oa *InterceptorNode) CurrentPayload() *eetypes.Payload {
	return oa.ps.Current()
}

// Rollback implements the Node interface.
func (*InterceptorNode) Rollback(head, safe, finalized *eetypes.Block) error {
	return fmt.Errorf("rollback not implemented")
}

// commitBlockAndUpdateNodeInfo simulates committing current block and updates node info.
//
// Need to combine startBuildingBlock and CommitAndBeginNextBlock
// https://github.com/cosmos/ibc-go/blob/main/testing/chain.go#L356
// https://github.com/cosmos/ibc-go/blob/main/testing/simapp/test_helpers.go#L130
func (oa *InterceptorNode) commitBlockAndUpdateNodeInfo() {
	block := oa.startBuildingBlock()

	oa.app.CommitAndBeginNextBlock(oa.ps.Current().Attrs.Timestamp)
	block = oa.sealBlock(block)

	oa.bs.AddBlock(block)
	oa.latestBlock = block
}

// startBuildingBlock starts building a new block for App is committed.
func (oa *InterceptorNode) startBuildingBlock() *eetypes.Block {
	// fill in block fields with L1 data
	block := oa.fillBlockWithL1Data(&eetypes.Block{})
	oa.applyL1Txs(block)
	oa.applyBlockL2Txs(block)
	return block
}

// fillBlockWithL1Data fills in block fields with L1 data.
func (oa *InterceptorNode) fillBlockWithL1Data(block *eetypes.Block) *eetypes.Block {
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
func (*InterceptorNode) applyL1Txs(block *eetypes.Block) {
	// TODO: we don't need to apply L1 txs in the proof-of-concept
}

// applyBlockL2Txs applies L2 txs to the block that's currently being built.
func (oa *InterceptorNode) applyBlockL2Txs(block *eetypes.Block) {
	block.Txs = oa.txMempool
	oa.txMempool = nil

	_, _ = oa.app.FinalizeBlock(&abcitypes.RequestFinalizeBlock{
		Height:             oa.LastBlockHeight() + 1,
		Time:               time.Unix(int64(block.Header.Time), 0),
		NextValidatorsHash: block.Header.NextValidatorsHash,
		Txs:                block.Txs.ToSliceOfBytes(),
	})
}

// sealBlock finishes building current L2 block from currentHeader, L2 txs in mempool, and L1 txs from
// payloadAttributes.
//
// sealBlock should be called after chainApp's committed. So chainApp.LastBlockHeight is the sealed block's height
func (oa *InterceptorNode) sealBlock(block *eetypes.Block) *eetypes.Block {
	oa.logger.Info("seal block", "height", oa.LastBlockHeight())

	// finalize block fields
	header := eetypes.Header{}
	block.Header = header.Populate(oa.app.LastHeader())

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

func (oa *InterceptorNode) findParentHash() eetypes.Hash {
	lastBlock := oa.bs.BlockByNumber(oa.LastBlockHeight() - 1)
	if lastBlock != nil {
		return lastBlock.Hash()
	}
	// TODO: handle cases where non-genesis block is missing
	return eetypes.ZeroHash
}

*/
