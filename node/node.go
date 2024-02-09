package node

import (
	"github.com/ethereum-optimism/optimism/op-service/sources"

	"github.com/ibc-scouts/ibc-interceptor/client/geth"
	"github.com/ibc-scouts/ibc-interceptor/server"
	"github.com/ibc-scouts/ibc-interceptor/server/api"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

type InterceptorNode struct {
	eeServer *server.EERPCServer   // RPC server for the Execution Engine
	geth     *sources.EngineClient // geth Execution Engine RPC bindings

	logger types.CompositeLogger
}

func NewInterceptorNode(config *types.Config) *InterceptorNode {
	logger, err := config.GetLogger("module", "interceptor")
	if err != nil {
		panic(err)
	}

	gethClient, err := geth.NewGethEngineClient(config.GethEngineAddr, config.GethAuthSecret, logger.New("client", "geth"))
	if err != nil {
		panic(err)
	}

	rpcServerConfig := server.DefaultConfig(config.EngineServerAddr)
	eeServer := server.NewEeRPCServer(rpcServerConfig, api.GetExecutionEngineAPIs(gethClient, logger.With("server", "exec_engine_api")), logger.With("server", "exec_engine_rpc"))
	return &InterceptorNode{
		eeServer: eeServer,
		geth:     gethClient,

		logger: logger,
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

	n.geth.Close()

	return nil
}
