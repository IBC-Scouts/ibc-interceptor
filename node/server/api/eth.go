package api

import (
	"context"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	eetypes "github.com/ibc-scouts/ibc-interceptor/node/types"
)

/* 'eth_' prefixed server methods, only required ones. */

// ethServer is the API for the eth like server.
// Implements required 'eth_' prefixed methods.
type ethServer struct {
	// client dials into op-geth server.
	// Might be best to not embed if we maybe want to add an sdk engine via rpc.
	blockStore BlockStore
	ethRPC     client.RPC
	peptideRPC client.RPC
	logger     log.Logger

	engineServer *engineServer
}

// newEthAPI returns a new execEngineAPI.
func newEthAPI(blockStore BlockStore, ethRPC, peptideRPC client.RPC, logger log.Logger) *ethServer {
	return &ethServer{blockStore, ethRPC, peptideRPC, logger, nil}
}

// TODO: delete hack
func (e *ethServer) SetEngineServer(engineServer *engineServer) {
	e.engineServer = engineServer
}

func GetEthAPI(blockStore BlockStore, ethRPC, peptideRPC client.RPC, logger log.Logger) rpc.API {
	return rpc.API{
		Namespace: "eth",
		Service:   newEthAPI(blockStore, ethRPC, peptideRPC, logger),
	}
}

func (e *ethServer) ChainId() (hexutil.Big, error) { // nolint: revive, stylecheck
	e.logger.Info("trying: ChainID")

	var id hexutil.Big
	err := e.ethRPC.CallContext(context.TODO(), &id, "eth_chainId")

	e.logger.Info("completed: ChainID", "id", id, "error", err)
	return id, err
}

// Docu yanked from go-eth for fullTx.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (e *ethServer) GetBlockByNumber(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByNumber", "id", id)

	var gethResult map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "eth_getBlockByNumber", id, fullTx)
	if err != nil {
		e.logger.Error("failed to call geth", "error", err)
		// TODO(jim): What do we do if geth for some reason errs and we dont? This happens when
		// GetBlockByNumber is called with a label of 'finalized'. For some reason ABCI engine
		// does _not_ return an error.
		return nil, err
	}

	var abciResult map[string]any
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "eth_getBlockByNumber", id, fullTx)
	if err != nil {
		e.logger.Error("failed to call abci", "error", err)
	}

	// Combine the hashes and store the composite block, return the composite hash as the geth["hash"] field.
	// See monomers ToEthBlock for fields populated in the abci call.
	gethHash := common.HexToHash(gethResult["hash"].(string))
	abciHash := common.HexToHash(abciResult["hash"].(string))
	compositeBlock := eetypes.NewCompositeBlock(gethHash, abciHash)
	e.blockStore.SaveCompositeBlock(compositeBlock)

	gethResult["hash"] = compositeBlock.Hash()

	e.logger.Info("composite block", "compositeHash", compositeBlock.Hash().Hex())
	e.logger.Info("completed: GetBlockByNumber", "result", gethResult)
	return gethResult, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *ethServer) GetBlockByHash(id any, fullTx bool) (map[string]any, error) {
	e.logger.Info("trying: GetBlockByHash", "id", id)

	hash := common.Hash{}
	switch id := id.(type) {
	case string:
		hash = common.HexToHash(id)
	case []byte:
		hash = common.BytesToHash(id)
	default:
		e.logger.Error("invalid type for id", "id", id)
	}
	compositeBlock := e.blockStore.GetCompositeBlock(hash)

	var gethResult map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &gethResult, "eth_getBlockByHash", compositeBlock.GethHash, fullTx)
	if err != nil {
		e.logger.Error("failed to call geth", "error", err)
		return nil, err
	}

	// NOTE: Do we even need to do forwarding? We don't use this block currently.
	var abciResult map[string]any
	err = e.peptideRPC.CallContext(context.TODO(), &abciResult, "eth_getBlockByHash", compositeBlock.ABCIHash, fullTx)
	if err != nil {
		e.logger.Error("failed to call abci", "error", err)
		return nil, err
	}

	gethResult["hash"] = compositeBlock.Hash()

	e.logger.Info("completed: GetBlockByHash", "result", gethResult)
	return gethResult, err
}

// Added for completeness -- tests do not appear to invoke for time being.
func (e *ethServer) GetProof(address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (map[string]any, error) {
	e.logger.Info("trying: GetProof")

	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getProof", address, storageKeys, blockNrOrHash)

	e.logger.Info("completed: GetProof", "result", result)
	return result, err
}

// Added for completeness -- tests do not appear to invoke for time being.
// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (e *ethServer) GetTransactionReceipt(txHash common.Hash) (map[string]any, error) {
	e.logger.Info("trying: GetTransactionReceipt")
	var result map[string]any
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getTransactionReceipt", txHash)

	e.logger.Info("completed: GetTransactionReceipt", "error", err, "result", result)
	return result, err
}

// Added to be able to intercept and forward eth transactions.
func (e *ethServer) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	e.logger.Info("trying: SendRawTransaction")

	tx := &ethtypes.Transaction{}

	err := tx.UnmarshalBinary(data)
	if err != nil {
		panic(err)
	}
	e.logger.Info("unmarshaled tx")

	var result common.Hash
	err = e.ethRPC.CallContext(context.TODO(), &result, "eth_sendRawTransaction", data)

	e.engineServer.AddTxHash(tx.Hash())

	e.logger.Info("completed: SendRawTransaction", "error", err, "result", result)
	return result, err
}

func (e *ethServer) MaxPriorityFeePerGas() (hexutil.Big, error) {
	e.logger.Info("trying: MaxPriorityFeePerGas")

	var result hexutil.Big
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_maxPriorityFeePerGas")

	e.logger.Info("completed: MaxPriorityFeePerGas", "result", result, "error", err)
	return result, err
}

func (e *ethServer) GetCode(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	e.logger.Info("trying: GetCode")

	var result hexutil.Bytes
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getCode", address, blockNrOrHash)

	e.logger.Info("completed: GetCode", "result", result, "error", err)
	return result, err
}

func (e *ethServer) EstimateGas(arg1 any) (hexutil.Uint64, error) {
	e.logger.Info("trying: EstimateGas")

	var result hexutil.Uint64
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_estimateGas", arg1)
	if err != nil {
		return 0, err
	}

	e.logger.Info("completed: EstimateGas", "result", result, "error", err)
	return result, nil
}

func (e *ethServer) GetTransactionCount(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	e.logger.Info("trying: GetTransactionCount")

	var result hexutil.Uint64
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_getTransactionCount", address, blockNrOrHash)

	e.logger.Info("completed: GetTransactionCount", "result", result, "error", err)
	return result, err
}

func (e *ethServer) Call(msg any, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	e.logger.Info("trying: Call")

	var result hexutil.Bytes
	err := e.ethRPC.CallContext(context.TODO(), &result, "eth_call", msg, blockNrOrHash)

	e.logger.Info("completed: Call", "result", result, "error", err)
	return result, err
}
