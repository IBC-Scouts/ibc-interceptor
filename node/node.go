package node

import (
	"github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ibc-scouts/ibc-interceptor/abci/engine"
	"github.com/ibc-scouts/ibc-interceptor/client/geth"
	"github.com/ibc-scouts/ibc-interceptor/server"
	"github.com/ibc-scouts/ibc-interceptor/server/api"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

type InterceptorNode struct {
	eeServer *server.EERPCServer // RPC server for the Execution Engine
	ethRPC   client.RPC          // RPC client for the Ethereum node

	logger types.CompositeLogger
}

func NewInterceptorNode(config *types.Config) *InterceptorNode {
	logger, err := config.GetLogger("module", "interceptor")
	if err != nil {
		panic(err)
	}

	ethRPC, err := geth.NewRPCClient(config.GethEngineAddr, config.GethAuthSecret, logger.New("client", "op-geth"))
	if err != nil {
		panic(err)
	}

	rpcServerConfig := server.DefaultConfig(config.EngineServerAddr)

	// TODO(jim): Init chain and pass it to the NewEngineServer
	abciEngine := engine.NewEngineServer(nil, logger.With("server", "abci_engine"))

	rpcAPIs := api.GetAPIs(ethRPC, abciEngine, logger.With("server", "exec_engine_api"))
	eeServer := server.NewEeRPCServer(rpcServerConfig, rpcAPIs, logger.With("server", "exec_engine_rpc"))
	return &InterceptorNode{
		eeServer: eeServer,
		ethRPC:   ethRPC,
		logger:   logger,
	}
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

	return nil
}
