//nolint:revive
package main

/*
This file is used to import all the necessary dependencies for the rollup package
so that the go mod tidy command does not remove them from the go.mod file.
This can be removed once the rollup package has its own dependencies.
*/

import (
	_ "github.com/cometbft/cometbft/abci/types"
	_ "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/ibc-go/v8/modules/core/types"
	_ "github.com/ethereum-optimism/optimism/op-node/rollup"
	_ "github.com/ethereum/go-ethereum/common"
)
