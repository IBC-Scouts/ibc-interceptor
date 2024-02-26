package api

import (
	"context"

	"github.com/cometbft/cometbft/libs/log"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/rpc"
)

func GetCosmosAPI(mempoolNode MempoolNode, peptideRPC client.RPC, logger log.Logger) rpc.API {
	return rpc.API{
		Namespace: "cosmos",
		Service:   newCosmosAPI(mempoolNode, peptideRPC, logger),
	}
}

// cosmosServer is the API for the underlying cosmos app.
type cosmosServer struct {
	mempoolNode MempoolNode
	peptideRPC client.RPC
	logger      log.Logger
}

// newCosmosAPI returns a new cosmosServer.
func newCosmosAPI(mempoolNode MempoolNode, peptideRPC client.RPC, logger log.Logger) *cosmosServer {
	return &cosmosServer{mempoolNode, peptideRPC, logger}
}

/* 'cosmos_' Namespace server methods:

Basically for any information we might want to send over from our e2es.
Currently we have entrypoints for each of the IBC messages we want to send over, these
could also be bulk added using a single method.
*/

func (e *cosmosServer) ChanOpenInit() error {
	e.logger.Info("trying: ChanOpenInit")

	// Create a Tx holding a chanopeninit message and add it to the mempool.
	// When we get a forkchoiceupdate call, we forward it to the peptide app (by another rpc call, not abci).
	msg := channeltypes.MsgChannelOpenInit{}
	msgBz, err := msg.Marshal()
	if err != nil {
		return err
	}

	e.mempoolNode.AddMsgToMempool(msgBz)

	return nil
}

func (e *cosmosServer) ChanOpenTry() error {
	e.logger.Info("trying: ChanOpenTry")

	// Create a Tx holding a chanopentry message and add it to the mempool.
	// When we get a forkchoiceupdate call, we forward it to the peptide app (by another rpc call, not abci).
	msg := channeltypes.MsgChannelOpenTry{}
	msgBz, err := msg.Marshal()
	if err != nil {
		return err
	}

	e.mempoolNode.AddMsgToMempool(msgBz)

	return nil
}

func (e *cosmosServer) ChanOpenAck() error {
	e.logger.Info("trying: ChanOpenAck")

	// Create a Tx holding a chanopenack message and add it to the mempool.
	// When we get a forkchoiceupdate call, we forward it to the peptide app (by another rpc call, not abci).
	msg := channeltypes.MsgChannelOpenAck{}
	msgBz, err := msg.Marshal()
	if err != nil {
		return err
	}

	e.mempoolNode.AddMsgToMempool(msgBz)

	return nil
}

func (e *cosmosServer) ChanOpenConfirm() error {
	e.logger.Info("trying: ChanOpenConfirm")

	// Create a Tx holding a chanopenconfirm message and add it to the mempool.
	// When we get a forkchoiceupdate call, we forward it to the peptide app (by another rpc call, not abci).
	msg := channeltypes.MsgChannelOpenConfirm{}
	msgBz, err := msg.Marshal()
	if err != nil {
		return err
	}

	e.mempoolNode.AddMsgToMempool(msgBz)

	return nil
}

// SendCosmosTx receives an opaque tx byte slice and adds it to the mempool.
func (e *cosmosServer) SendTransaction(tx []byte) (SendCosmosTxResult, error) {
	e.logger.Info("trying: SendTransaction", "tx", tx)

	// Try and parse it as a cm and dump it in our own mempool. When we get a forkchoiceupdate
	// call, we forward it to the peptide app (by another rpc call, not abci).

	var result bool
	err := e.peptideRPC.CallContext(context.TODO(), &result, "intercept_addTxToMempool", tx)
	if err != nil {
		e.logger.Error("forward SendTransaction to abci engine", "error", err)
	}

	e.logger.Info("success in forwarding SendTransaction to abci engine", "result", result)

	e.logger.Info("completed: SendTransaction")
	return SendCosmosTxResult{}, nil
}
