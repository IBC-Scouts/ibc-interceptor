package cmd

/*
import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/bytes"
	bfttypes "github.com/cometbft/cometbft/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/polymerdao/polymerase/chain/app/node"
	eetypes "github.com/polymerdao/polymerase/chain/app/node/types"
	peptest "github.com/polymerdao/polymerase/chain/testutil/peptide"
	"github.com/polymerdao/polymerase/chain/testutil/peptide/eeclient"
	polyibctypes "github.com/polymerdao/polymerase/chain/x/polyibc/core/types"
	"github.com/polymerdao/polymerase/chain/x/polyibc/light-clients/sim"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

const (
	cosmosUrl = "tcp://localhost:1234"
	eeUrl     = "http://localhost:8080"
	l1Hash    = "0x0000000000000000000000000000000000000000000000000000000000000001"
)

type appTestSuite struct {
	suite.Suite

	cmd         *cobra.Command
	node        *peptest.OpNodeMock
	homedir     string
	genesisTime time.Time
	wg          sync.WaitGroup
}

func (a *appTestSuite) SetupTest() {
	homedir, err := os.MkdirTemp("", "test-api-*")
	a.NoError(err)
	a.homedir = homedir
	// set genesis time to a fixed number to have a reproducible genesis block between test runs
	a.genesisTime = time.UnixMilli(1699646689 * 100)
	a.cmd = RootCmd()

	// generate genesis
	a.cmd.SetArgs([]string{"init",
		"--home", a.homedir,
		"--l1-hash", l1Hash,
		"--l1-height", "1",
		"--chain-id", "123",
	})
	a.NoError(a.cmd.Execute())

	a.cmd.SetArgs([]string{"seal",
		"--home", a.homedir,
		"--genesis-time", strconv.FormatInt(a.genesisTime.Unix(), 10),
	})
	a.NoError(a.cmd.Execute())

	a.cmd.SetArgs([]string{"start",
		"--home", a.homedir,
		"--app-rpc-address", cosmosUrl,
		"--ee-http-server-address", eeUrl,
		"--admin-api",
	})
	a.run()
}

func (a *appTestSuite) TearDownTest() {
	a.quit()
	os.RemoveAll(a.homedir)
}

func (a *appTestSuite) TestStartStop() {
	a.restart()
}

func (a *appTestSuite) TestStartStopBlockHeight() {
	a.node.RunDerivationLoop(5, peptest.Accounts[0], peptest.Accounts[1])
	lastBlockBefore, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)

	a.restart()

	lastBlockAfter, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)

	// last block before stopping the server must be the same one after starting it again.
	a.Equal(lastBlockBefore.Height(), lastBlockAfter.Height())
	a.Equal(lastBlockBefore.Hash(), lastBlockAfter.Hash())

	a.restart()

	lastBlock, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	a.Equal(lastBlockAfter.Height(), lastBlock.Height())

	a.node.ProduceBlocks(1)

	lastBlock, err = a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)

	a.Equal(lastBlockAfter.Height()+1, lastBlock.Height())

}

func (a *appTestSuite) TestStartStopBalance() {
	a.node.RunDerivationLoop(5, peptest.Accounts[0], peptest.Accounts[1])
	lastBlockBefore, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	balanceBefore := a.node.QueryBalance(peptest.Accounts[1].GetAddress().String(), lastBlockBefore.Height())

	a.restart()

	lastBlockAfter, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	balanceAfter := a.node.QueryBalance(peptest.Accounts[1].GetAddress().String(), lastBlockAfter.Height())

	// balance (app state) must be the same one after starting it again.
	a.Equal(balanceBefore.Amount, balanceAfter.Amount)
}

func (a *appTestSuite) TestStartStopStartWithBalance() {
	balances := make(map[int64]*sdktypes.Coin)
	a.node.RunDerivationLoop(5, peptest.Accounts[0], peptest.Accounts[1])
	lastBlockBefore, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	balanceBefore := a.node.QueryBalance(peptest.Accounts[1].GetAddress().String(), lastBlockBefore.Height())
	balances[lastBlockBefore.Height()] = balanceBefore

	a.restart()

	// run a derivation loop with no transactions. Balance should remain
	a.node.ProduceBlocks(5)
	lastBlockAfter, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)

	balanceAfter := a.node.QueryBalance(peptest.Accounts[1].GetAddress().String(), lastBlockAfter.Height())
	a.Equal(balanceBefore.Amount, balanceAfter.Amount)
	balances[lastBlockAfter.Height()] = balanceAfter

	a.restart()

	a.node.ProduceBlocks(1)

	lastBlockAfter, err = a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	balanceAfter = a.node.QueryBalance(peptest.Accounts[1].GetAddress().String(), lastBlockAfter.Height())

	a.Equal(balanceBefore.Amount, balanceAfter.Amount)
	balances[lastBlockAfter.Height()] = balanceAfter

	a.Equal(3, len(balances))
	for height, expected := range balances {
		balance := a.node.QueryBalance(peptest.Accounts[1].GetAddress().String(), height)
		a.Equal(expected, balance)
	}
}

func (a *appTestSuite) TestGenesis() {
	genesis, err := node.PeptideGenesisFromFile(a.homedir)
	a.NoError(err)

	a.Equal("123", genesis.ChainID)
	a.Equal(uint64(1), genesis.L1.Number)
	a.Equal(l1Hash, genesis.L1.Hash.Hex())
	a.NotEqual(eetypes.ZeroHash, genesis.GenesisBlock.Hash)
	a.Equal(uint64(1), genesis.GenesisBlock.Number)
	a.Equal(a.genesisTime.Unix(), genesis.GenesisTime.Unix())
	a.True(len(genesis.AppState) > 0)
}

func (a *appTestSuite) TestGenesisReproducibility() {
	genesisBefore, err := node.PeptideGenesisFromFile(a.homedir)
	a.NoError(err)

	sender := peptest.Accounts[0]
	sequence := sender.GetSequence()

	a.node.RunDerivationLoop(2, sender, peptest.Accounts[1])
	lastBlockBefore, err := a.node.BlockByNumber(eth.Unsafe)
	a.NoError(err)

	a.quit()
	// reset the sequence number since it will have been changed by the loop derivation
	sender.SetSequence(sequence)
	a.SetupTest()

	lastGenesis, err := node.PeptideGenesisFromFile(a.homedir)
	a.NoError(err)

	a.node.RunDerivationLoop(2, sender, peptest.Accounts[1])
	lastBlock, err := a.node.BlockByNumber(eth.Unsafe)
	a.NoError(err)

	a.Equal(genesisBefore, lastGenesis)

	// transactions record the time when were first seen locally, to avoid spam.
	// this is not part of the tx hash but it will make the assert below fail
	for i, tx := range lastBlockBefore.Transactions() {
		tx.SetTime(lastBlock.Transactions()[i].Time())
	}
	a.Equal(lastBlockBefore, lastBlock)
}

func (a *appTestSuite) TestDeterministicGenesisBlock() {
	homedir, err := os.MkdirTemp("", "test-deterministic-genesis-*")
	a.NoError(err)
	defer os.RemoveAll(homedir)

	cmd := RootCmd()
	// generate other genesis using the same input used in SetupTest
	cmd.SetArgs([]string{"init",
		"--home", homedir,
		"--l1-hash", l1Hash,
		"--l1-height", "1",
		"--chain-id", "123",
	})
	a.NoError(cmd.Execute())

	// must use the same genesis time to seal the genesis block
	cmd.SetArgs([]string{"seal",
		"--home", homedir,
		"--genesis-time", strconv.FormatInt(a.genesisTime.Unix(), 10),
	})
	a.NoError(cmd.Execute())

	genesis, err := node.PeptideGenesisFromFile(a.homedir)
	a.NoError(err)

	otherGenesis, err := node.PeptideGenesisFromFile(homedir)
	a.NoError(err)

	a.Equal(genesis.GenesisBlock.Hash, otherGenesis.GenesisBlock.Hash)
	a.Equal(genesis.GenesisBlock.Number, otherGenesis.GenesisBlock.Number)
	a.Equal(genesis.GenesisTime, otherGenesis.GenesisTime)
}

func (a *appTestSuite) TestStatus() {
	status, err := a.node.Status()
	a.NoError(err)
	a.NotEmpty(status.ValidatorInfo.Address.String())
}

func (a *appTestSuite) TestChainID() {
	chainID, err := a.node.ChainID()
	a.NoError(err)
	a.Equal("123", chainID.String())
}

func (a *appTestSuite) TestCreateSimLightClient() {
	a.node.ProduceBlocks(1)

	block, err := a.node.BlockByNumber(eth.Unsafe)
	a.NoError(err)

	clientState := sim.NewClientState("123", polyibctypes.NewHeight(0, block.NumberU64()), "memo")
	consensusState := sim.NewSimHeaderAlwaysValid(0, block.NumberU64(), "").ToConsensusState()

	sender := peptest.Accounts[0]
	msg, err := polyibctypes.NewMsgCreateClient(
		clientState,
		consensusState,
		sender.GetAddress().String(),
		"memo",
	)
	a.NoError(err)

	response, err := a.node.SendMsg(sender, msg)
	a.NoError(err)
	a.Equal(uint32(0), response.Code)

	a.node.ProduceBlocks(1)
	tx, err := a.node.Tx(response.Hash, false)
	a.NoError(err)

	var nativeClientID string
	for _, evt := range tx.TxResult.Events {
		if evt.Type == "create_client" {
			for _, attr := range evt.Attributes {
				if attr.GetKey() == "client_id" {
					nativeClientID = attr.GetValue()
				}
			}
		}
	}
	a.Equal("sim-test-0", nativeClientID)
}

func (a *appTestSuite) TestStartStopTxIndexer() {
	numOfLoops := 5
	a.node.RunDerivationLoop(uint64(numOfLoops), peptest.Accounts[0], peptest.Accounts[1])
	block, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)

	txHashes := make([]bytes.HexBytes, 0)
	allTxCount := len(block.L1Txs) + len(block.Txs)
	a.Equal(numOfLoops, allTxCount)
	for i := 0; i < allTxCount; i++ {
		var tx bfttypes.Tx
		if i < len(block.L1Txs) {
			tx = bfttypes.Tx(block.L1Txs[i])
		} else {
			tx = block.Txs[i-len(block.L1Txs)]
		}
		txHash := tx.Hash()
		// even failed txs are included
		txFound, err := a.node.Tx(txHash, false)
		a.node.NoError(err)
		a.node.NotNil(txFound)
		a.Equal(txFound.Index, uint32(i))
		// a.T().Logf("txIndexed %v: %x", i, []byte(txFound.Tx))
		txHashes = append(txHashes, txFound.Hash)
	}

	a.restart()

	for _, hash := range txHashes {
		txFound, err := a.node.TxSearch(fmt.Sprintf("tx.hash='%v'", hash), false, nil, nil, "asc")
		a.node.NoError(err)
		a.node.NotNil(txFound)
	}
}

// this tests the full rollback done via the admin API. It checks that the blockstore, txstore and
// appstate rollback as expected.
// the setup queries the current balance, sends tokens and checks that they
// are received. Then, the rollback is done and the test checks that the tx that send the token
// is no longer available and that the balance went back to the original value.
func (a *appTestSuite) TestRollback() {
	sender := peptest.Accounts[0]
	receiver := peptest.Accounts[1]
	a.node.ProduceBlocks(3)

	// this is the block we'll roll back to. The latest at this point
	rollbackBlock, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	safeBlock, err := a.node.PeptideBlock(eth.Safe)
	a.NoError(err)
	finalizedBlock, err := a.node.PeptideBlock(eth.Finalized)
	a.NoError(err)
	startingBalance := a.node.QueryBalance(receiver.GetAddress().String(), rollbackBlock.Height())
	log.Println(">>> block1", "hash", rollbackBlock.Hash().Hex(), "height", rollbackBlock.Height(), "balance", startingBalance)

	// send tokens and verify they have been received
	tx, err := a.node.SendTokens(sender, receiver, 1000)
	a.NoError(err)

	a.node.ProduceBlocks(2)
	blockBefore, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	balance2 := a.node.QueryBalance(receiver.GetAddress().String(), blockBefore.Height())
	log.Println(">>> block2", "hash", blockBefore.Hash().Hex(), "height", blockBefore.Height(), "balance", balance2)

	a.Equal(startingBalance.AddAmount(math.NewInt(1000)).Amount, balance2.Amount)

	_, err = a.node.Tx(tx.Hash, false)
	a.NoError(err) // no error == tx found

	// here's where the rollback happens
	a.NoError(a.node.Rollback(rollbackBlock.Height(), safeBlock.Height(), finalizedBlock.Height()))

	// the new latest head after the rollback is the same as the one from the starting point
	afterRollback, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)
	a.Equal(rollbackBlock.Hash(), afterRollback.Hash())
	a.Equal(rollbackBlock.Height(), afterRollback.Height())

	safeBlockAfterRollback, err := a.node.PeptideBlock(eth.Safe)
	a.NoError(err)
	a.Equal(safeBlock.Hash(), safeBlockAfterRollback.Hash())
	a.Equal(safeBlock.Height(), safeBlockAfterRollback.Height())

	finalizedBlockAfterRollback, err := a.node.PeptideBlock(eth.Finalized)
	a.NoError(err)
	a.Equal(finalizedBlock.Hash(), finalizedBlockAfterRollback.Hash())
	a.Equal(finalizedBlock.Height(), finalizedBlockAfterRollback.Height())

	a.node.ProduceBlocks(2)
	block3, err := a.node.PeptideBlock(eth.Unsafe)
	a.NoError(err)

	// the balance is back to what it was before sending the tokens
	balanceAfterRollback := a.node.QueryBalance(receiver.GetAddress().String(), block3.Height())
	log.Println(">>> block3", "hash", block3.Hash().Hex(), "height", block3.Height(), "balance", balanceAfterRollback)
	a.Equal(startingBalance, balanceAfterRollback)

	// after producing the same amount of blocks (after the rollback) the block at height blockBefore.Height()
	// has been mined again with the same hash
	a.node.ProduceBlocks(2)
	blockAfter, err := a.node.PeptideBlock(blockBefore.Height())
	a.NoError(err)
	log.Println(">>> block4", "hash", blockAfter.Hash().Hex(), "height", blockAfter.Height())
	a.NotEqual(blockBefore.Hash(), blockAfter.Hash())

	// the tx that sent the tokens has now been rolled back so it is not available.
	_, err = a.node.Tx(tx.Hash, false)
	a.Error(err) // err == tx NOT found
}

// cmd.Execute() will block so run it within its own goroutine.
func (a *appTestSuite) run() {
	log.Println("starting...")
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.NoError(a.cmd.Execute())
	}()

	// create a new client every time the server is started
	client, err := eeclient.NewEeClient(context.Background(), cosmosUrl, eeUrl)
	a.NoError(err)
	a.node = peptest.NewOpNodeMock(&a.Suite, client, rand.New(rand.NewSource(int64(1234))))
	log.Println("started.")
}

// send SIGINT to the running process so the server running on its own goroutine quits after receiving it
func (a *appTestSuite) quit() {
	log.Println("quitting...")
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	// wait until the goroutine is actually done
	a.wg.Wait()
	a.node.Close()
	log.Println("quit.")
}

func (a *appTestSuite) restart() {
	a.quit()
	a.run()
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(appTestSuite))
}
*/
