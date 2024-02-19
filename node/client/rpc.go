package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
)

// NewRPCClient creates a new eth rpc client.
func NewRPCClient(address string, authSecret []byte, logger log.Logger) (client.RPC, error) {
	if strings.TrimSpace(address) == "" {
		return nil, fmt.Errorf("geth execution engine address must be non-empty")
	}

	var secret [32]byte
	if len(authSecret) == 0 {
		secret = [32]byte{123}
	} else {
		copy(secret[:], authSecret[:min(len(authSecret), 32)])
	}

	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(secret))
	opts := []client.RPCOption{
		client.WithGethRPCOptions(auth),
		client.WithDialBackoff(10),
	}
	rpcClient, err := client.NewRPC(context.TODO(), logger, address, opts...)
	if err != nil {
		return nil, err
	}

	return rpcClient, nil
}
