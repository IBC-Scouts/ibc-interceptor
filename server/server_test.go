package server_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	ethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	cmtlog "github.com/cometbft/cometbft/libs/log"

	"github.com/ibc-scouts/ibc-interceptor/client/geth"
	"github.com/ibc-scouts/ibc-interceptor/server"
	"github.com/ibc-scouts/ibc-interceptor/server/api"
	"github.com/ibc-scouts/ibc-interceptor/testing/mock"
)

func TestDefaultConfig(t *testing.T) {
	config := server.DefaultConfig("localhost:0")
	require.Equal(t, 0, config.HTTPPort)
	require.Equal(t, "localhost", config.HTTPHost)
	require.Equal(t, "/", config.HTTPPathPrefix)
	require.Equal(t, "Execution-Engine", config.Name)
	require.Equal(t, "/websocket", config.WSPathPrefix)
}

func TestServer(t *testing.T) {
	mockEngine := mock.NewMockExecEngine()
	nopLogger := cmtlog.NewNopLogger()

	config := server.DefaultConfig("localhost:0")

	mockGethServer := server.NewEeRPCServer(config, api.GetExecutionEngineAPIs(mockEngine, nopLogger), nopLogger)
	require.NotNil(t, mockGethServer)

	err := mockGethServer.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = mockGethServer.Stop()
		mockGethServer.Quit()
	})

	require.Equal(t, "tcp", mockGethServer.Address().Network())
	host, portstr, err := net.SplitHostPort(mockGethServer.Address().String())
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host)
	require.NotEqual(t, "0", portstr)

	// Test that a this can be connected to as a geth client
	client, err := geth.NewGethEngineClient("http://"+mockGethServer.Address().String(), nil, ethlog.New())
	require.NoError(t, err)
	require.NotNil(t, client)

	expResult := &eth.ForkchoiceUpdatedResult{PayloadID: &eth.PayloadID{12}}
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{}, &eth.PayloadAttributes{}, expResult, nil)

	ctx := context.Background()
	result, err := client.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{}, &eth.PayloadAttributes{})
	require.NoError(t, err)
	require.Equal(t, expResult, result)
}
