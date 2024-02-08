package ibcinterceptor

import (
	"github.com/ethereum-optimism/optimism/op-service/sources"

	rpcee "github.com/ibc-scouts/ibc-interceptor/rpc_ee"
)

type InterceptorNode struct {
	eeServer *rpcee.EERPCServer    // RPC server for the Execution Engine
	geth     *sources.EngineClient // geth Execution Engine RPC bindings
}

func NewInterceptorNode(eeServer *rpcee.EERPCServer, geth *sources.EngineClient) *InterceptorNode {
	return &InterceptorNode{
		eeServer: eeServer,
		geth:     geth,
	}
}
