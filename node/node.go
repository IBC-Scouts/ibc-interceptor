package node

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ibc-scouts/ibc-interceptor/client/geth"
	"github.com/ibc-scouts/ibc-interceptor/server"
	"github.com/ibc-scouts/ibc-interceptor/server/api"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

type InterceptorNode struct {
	eeServer *server.EERPCServer // RPC server for the Execution Engine
	client   client.RPC          // Client for calling into op-geth RPC server.

	logger types.CompositeLogger
}

func NewInterceptorNode(config *types.Config) *InterceptorNode {
	logger, err := config.GetLogger("module", "interceptor")
	if err != nil {
		panic(err)
	}

	rpcClient, err := geth.NewRPCClient(config.GethEngineAddr, config.GethAuthSecret, logger.New("client", "op-geth"))
	if err != nil {
		panic(err)
	}

	rpcServerConfig := server.DefaultConfig(config.EngineServerAddr)
	eeServer := server.NewEeRPCServer(rpcServerConfig, api.GetAPIs(rpcClient, logger.With("server", "exec_engine_api")), logger.With("server", "exec_engine_rpc"))
	return &InterceptorNode{
		eeServer: eeServer,
		client:   rpcClient,
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

	n.client.Close()

	return nil
}
