package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Genesis is used in OpApp's genesis.
// It is a clone of PeptideGenesis,
// https://github.com/polymerdao/monomer/blob/f57e4e014d24fb2538926efd5232c9dd383543cc/app/node/genesis.go#L15
type Genesis struct {
	GenesisTime   time.Time   `json:"genesis_time"`
	GenesisBlock  eth.BlockID `json:"genesis_block"`
	ChainID       string      `json:"chain_id"`
	AppState      []byte      `json:"app_state"`
	L1            eth.BlockID `json:"l1"`
	InitialHeight int64       `json:"initial_height"`
	// TODO: add missing stuff
}

func (p *Genesis) Validate() error {
	if p.GenesisBlock.Hash.Cmp(common.Hash{}) == 0 {
		return fmt.Errorf("genesis block hash must not be empty")
	}
	if p.GenesisBlock.Number == uint64(0) {
		return fmt.Errorf("genesis block height must not be zero")
	}
	if p.L1.Hash.Cmp(common.Hash{}) == 0 {
		return fmt.Errorf("l1 hash must not be empty")
	}
	if p.GenesisTime.Unix() == int64(0) {
		return fmt.Errorf("genesis time must not be empty")
	}
	if _, err := strconv.ParseUint(p.ChainID, 10, 64); err != nil {
		return fmt.Errorf("chain-id must be numerical")
	}
	if len(p.AppState) == 0 {
		return fmt.Errorf("app state must not be empty")
	}
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (p *Genesis) Save(homedir string, override bool) error {
	genFile := filepath.Join(homedir, "config", "genesis.json")
	if err := os.MkdirAll(filepath.Dir(genFile), 0o700); err != nil {
		return err
	}

	// return error if the genesis file exist already
	if !override && fileExists(genFile) {
		return fmt.Errorf("refusing to override existent genesis file %v", genFile)
	}

	genesis, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	//nolint:gosec
	if err := os.WriteFile(genFile, genesis, 0o644); err != nil {
		return err
	}

	return nil
}

func OpGenesisFromFile(homedir string) (*Genesis, error) {
	genFile := filepath.Join(homedir, "config", "genesis.json")

	bz, err := os.ReadFile(genFile)
	if err != nil {
		return nil, fmt.Errorf("could not read genesis file: %w", err)
	}

	var genesis Genesis
	if err := json.Unmarshal(bz, &genesis); err != nil {
		return nil, fmt.Errorf("could not unmarshal genesis file: %w", err)
	}

	if err := genesis.Validate(); err != nil {
		return nil, err
	}

	return &genesis, nil
}
