package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/ibc-scouts/ibc-interceptor/server/api"
	"github.com/ibc-scouts/ibc-interceptor/testing/mock"
)

func TestGetExecutionEngineAPIs(t *testing.T) {
	testCases := []struct {
		name     string
		engine   derive.Engine
		logger   log.Logger
		expPanic bool
	}{
		{
			"success",
			mock.NewMockExecEngine(),
			log.NewNopLogger(),
			false,
		},
		{
			"failure: nil engine",
			nil,
			log.NewNopLogger(),
			true,
		},
		{
			"failure: nil log",
			mock.NewMockExecEngine(),
			nil,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := []rpc.API(nil)
			f := func() {
				result = api.GetExecutionEngineAPIs(tc.engine, tc.logger)
			}

			if !tc.expPanic {
				f()
				require.NotNil(t, result)
			} else {
				require.Panics(t, f)
			}
		})
	}
}

func TestExecutionEngineAPI(t *testing.T) {
	mockEngine := mock.NewMockExecEngine()
	eeAPI := api.NewExectionEngineAPI(mockEngine, log.NewNopLogger())

	expResult := &eth.ForkchoiceUpdatedResult{PayloadID: &engine.PayloadID{1}}
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{}, &eth.PayloadAttributes{}, expResult, nil)

	res, err := eeAPI.ForkchoiceUpdatedV3(eth.ForkchoiceState{}, eth.PayloadAttributes{})
	require.NoError(t, err)
	require.Equal(t, expResult, res)
}
