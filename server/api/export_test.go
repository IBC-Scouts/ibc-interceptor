package api

import (
	"github.com/ibc-scouts/ibc-interceptor/client/geth"

	"github.com/cometbft/cometbft/libs/log"
)

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// NewExectionEngineAPI is a wrapper around newExecutionEngineAPI that allows for testing.
//
//nolint:revive
func NewExectionEngineAPI(engine *geth.GethWrappedClient, logger log.Logger) *execEngineAPI {
	return newExecutionEngineAPI(engine, logger)
}
