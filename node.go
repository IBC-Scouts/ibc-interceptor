package ibcinterceptor

import (
	"os"

	ethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/sources"

	cometlog "github.com/cometbft/cometbft/libs/log"

	"github.com/ibc-scouts/ibc-interceptor/client/geth"
	rpcee "github.com/ibc-scouts/ibc-interceptor/rpc_ee"
	"github.com/ibc-scouts/ibc-interceptor/server/engine"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

type InterceptorNode struct {
	eeServer *rpcee.EERPCServer    // RPC server for the Execution Engine
	geth     *sources.EngineClient // geth Execution Engine RPC bindings

	cometLogger cometlog.Logger
	ethLogger   ethlog.Logger
}

func NewInterceptorNode(config *types.Config) *InterceptorNode {
	ethLogger := ethlog.New("module", "interceptor")
	cometLogger := cometlog.NewTMLogger(cometlog.NewSyncWriter(os.Stdout)).With("module", "interceptor")

	cometLogger.Info("Interceptor Node starting", "config", config)

	gethClient, err := geth.NewGethEngineClient(config.GethEngineAddr, config.GethAuthSecret)
	if err != nil {
		panic(err)
	}

	rpcServerConfig := rpcee.DefaultConfig(config.EngineServerAddr)
	eeServer := rpcee.NewEeRPCServer(rpcServerConfig, engine.GetExecutionEngineAPIs(gethClient, ethLogger), cometLogger)
	return &InterceptorNode{
		eeServer: eeServer,
		geth:     gethClient,

		cometLogger: cometLogger,
		ethLogger:   ethLogger,
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
