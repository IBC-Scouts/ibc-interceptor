package geth

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// NewGethEngineClient creates a new geth EngineClient
func NewGethEngineClient(gethEngineAddr string, gethAuthSecret []byte) (*sources.EngineClient, error) {
	// necessary setup args
	ctx, m, logger := context.Background(), metrics.NewMetrics(""), log.New()

	if strings.TrimSpace(gethEngineAddr) == "" {
		return nil, fmt.Errorf("geth execution engine address must be non-empty")
	}

	var authSecret [32]byte
	if len(gethAuthSecret) == 0 {
		authSecret = [32]byte{123}
	} else {
		copy(authSecret[:], gethAuthSecret[:min(len(gethAuthSecret), 32)])
	}

	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(authSecret))
	opts := []client.RPCOption{
		client.WithGethRPCOptions(auth),
		client.WithDialBackoff(10),
	}
	rpcClient, err := client.NewRPC(ctx, logger, gethEngineAddr, opts...)
	if err != nil {
		return nil, err
	}

	// TODO(colin): figure out how to populate rollupConfig
	rollupCfg := &rollup.Config{}
	rpcCfg := sources.EngineClientDefaultConfig(rollupCfg)

	engineClient, err := sources.NewEngineClient(
		client.NewInstrumentedRPC(rpcClient, m), logger, m.L2SourceCache, rpcCfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Engine client: %w", err)
	}

	return engineClient, nil
}
