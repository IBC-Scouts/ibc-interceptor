package mock

import "github.com/ethereum-optimism/optimism/op-service/testutils"

// NewMockExecEngine creates a new mock execution engine
// implements the derive.Engine interface
func NewMockExecEngine() *testutils.MockEngine {
	return &testutils.MockEngine{}
}
