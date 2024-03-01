package node

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	nodeclient "github.com/ibc-scouts/ibc-interceptor/node/client"
	"github.com/ibc-scouts/ibc-interceptor/node/server"
	"github.com/ibc-scouts/ibc-interceptor/node/server/api"
	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

const IBCCrossDomainMessenger = "0x42000000000000000000000000000000000000E0"

// InterceptorNode is the main struct for the Interceptor node that facilitates communication
// between the op-node on one side and the ethereum and sdk engines on the other. It holds
// rpc clients for boths and intercepts all engine API calls performed by op-node.
type InterceptorNode struct {
	// eeServer is the RPC server for the Execution Engine
	eeServer *server.EERPCServer
	// ethRPC is the RPC client for the Ethereum node
	ethRPC client.RPC
	// peptideRPC is the RPC client for the Peptide node
	peptideRPC client.RPC

	// msgMempool is a basic Mempool to be used in OpApp.
	// TODO(jim): Might need to make into a full fledged type to support more complex mempool operations.
	msgMempool   [][]byte
	blockStore   map[common.Hash]eetypes.CompositeBlock
	payloadStore map[eth.PayloadID]eetypes.CompositePayload

	logger types.CompositeLogger
	lock   sync.RWMutex

	// eventCh is used to receive events from the Ethereum node
	eventCh chan<- []*ethtypes.Log
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
	// create the peptide client based on address passed in via command line.
	peptideRPC, err := nodeclient.NewRPCClient(config.PeptideEngineAddr, nil, logger.New("client", "peptide"))
	if err != nil {
		panic(err)
	}

	node := &InterceptorNode{
		logger:       logger,
		ethRPC:       ethRPC,
		peptideRPC:   peptideRPC,
		blockStore:   make(map[common.Hash]eetypes.CompositeBlock),
		payloadStore: make(map[eth.PayloadID]eetypes.CompositePayload),
		eventCh:      make(chan []*ethtypes.Log),
	}

	arg := map[string]interface{}{
		"address": []common.Address{common.HexToAddress(IBCCrossDomainMessenger)},
		// "topics":  q.Topics,
	}

	// subscribe based on a modification of:
	// https://github.com/ethereum-optimism/op-geth/blob/0402d543c3d0cff3a3d344c0f4f83809edb44f10/ethclient/ethclient.go#L444
	_, err = ethRPC.EthSubscribe(context.Background(), node.eventCh, "logs", arg)
	if err != nil {
		panic(err)
	}

	// Add APIs to the RPC server
	rpcAPIs := api.GetEngineAPI(node, ethRPC, peptideRPC, logger.With("server", "exec_engine_api"))
	rpcAPIs = append(
		rpcAPIs,
		// Add eth and cosmos APIs
		api.GetEthAPI(node, ethRPC, peptideRPC, logger.With("server", "eth_api")),
		api.GetCosmosAPI(node, peptideRPC, logger.With("server", "cosmos_api")),
	)

	// Create config for the RPC server (address to bind to)
	rpcServerConfig := server.DefaultConfig(config.EngineServerAddr)

	node.eeServer = server.NewEeRPCServer(rpcServerConfig, rpcAPIs, logger.With("server", "exec_engine_rpc"))
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

// -- MempoolNode interface --

// AddTxToMempool add a tx to the mempool.
func (n *InterceptorNode) AddMsgToMempool(bz []byte) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.logger.Info("AddMsgToMempool", "msg", bz)
	n.msgMempool = append(n.msgMempool, bz)
}

// HasMsgs returns true if the mempool has messages.
func (n *InterceptorNode) HasMsgs() bool {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return len(n.msgMempool) > 0
}

// GetMsgs returns all messages in the mempool.
func (n *InterceptorNode) GetMsgs() [][]byte {
	n.lock.Lock()
	defer n.lock.Unlock()

	return n.msgMempool
}

// ClearMsgs clears all messages from the mempool.
func (n *InterceptorNode) ClearMsgs() {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.msgMempool = nil
}

// -- BlockStore interface --

// GetCompositeBlock returns a composite block given the combined block hash
func (n *InterceptorNode) GetCompositeBlock(blockHash common.Hash) eetypes.CompositeBlock {
	return n.blockStore[blockHash]
}

func (n *InterceptorNode) SaveCompositeBlock(compositeBlock eetypes.CompositeBlock) {
	n.blockStore[compositeBlock.Hash()] = compositeBlock
}

// -- PayloadStore interface --

// GetCompositePayload returns a composite payload given the combined payload hash
func (n *InterceptorNode) GetCompositePayload(compositePayload eth.PayloadID) eetypes.CompositePayload {
	return n.payloadStore[compositePayload]
}

func (n *InterceptorNode) SaveCompositePayload(compositePayload eetypes.CompositePayload) {
	n.payloadStore[*compositePayload.Payload()] = compositePayload
}
