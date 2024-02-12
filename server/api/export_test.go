package api

import (
	"github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/cometbft/cometbft/libs/log"
)

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// NewExectionEngineAPI is a wrapper around newExecutionEngineAPI that allows for testing.
//
//nolint:revive
func NewExectionEngineAPI(client client.RPC, logger log.Logger) *execEngineServer {
	return newExecutionEngineAPI(client, logger)
}
