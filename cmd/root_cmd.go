package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "interceptor",
		Short: "Interceptor execution engine for OP stack.",
	}

	// TODO(colin): decide necessary commands
	rootCmd.AddCommand(
		startCmd(),
		exportCmd(),
		genAccountsCmd(),
		initCmd(),
		sealCmd(),
	)
	return rootCmd
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the Interceptor Node",
		RunE: func(cmd *cobra.Command, args []string) error {
			//	logger := server.DefaultLogger()
			//	config := server.NewConfig(cmd).
			//		WithHomeDir().
			//		WithL1().
			//		WithChainId().
			//		WithDbBackend().
			//		WithLogger(logger)

			// create an in memory db backed app only to generate an initial genesis file
			// TODO(colin): uncomment when we use abci app
			///app := app.New(config.ChainId, "", tmdb.NewMemDB(), logger)

			// TODO use these testing accounts for now until we add proper account management
			//	appState := app.SimpleGenesis(peptest.Accounts, peptest.ValidatorAccounts)
			//	appStateBytes, err := json.Marshal(appState)
			//	if err != nil {
			// return err
			//}

			// genesis state will be validated when sealed.
			//	genesis := node.PeptideGenesis{}

			//	genesis.L1.Hash = config.L1.Hash
			//	genesis.L1.Number = config.L1.Number
			//	genesis.AppState = appStateBytes

			// use a dummy genesis block for now so the validation during seal passes
			//	genesis.GenesisBlock = eth.BlockID{
			//		Hash:   eetypes.HashOfEmptyHash,
			//		Number: uint64(1),
			//	}

			// TODO add override option
			//		if err := genesis.Save(config.HomeDir, false); err != nil {
			//			return err
			//		}

			// TODO write basic config to file?

			//		logger.Info("genesis initialized")
			return nil
		},
	}

	//	server.AddInitCommandFlags(cmd)
	return cmd
}

func sealCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seal",
		Short: "Seals the node",
		RunE: func(cmd *cobra.Command, args []string) error {
			/*
				logger := server.DefaultLogger()
				config := server.NewConfig(cmd).
					WithHomeDir().
					WithDbBackend().
					WithGenesisTime().
					WithLogger(logger)

				genesis, err := node.PeptideGenesisFromFile(config.HomeDir)
				if err != nil {
					return err
				}
				genesis.GenesisTime = config.GenesisTime

				appdb := server.OpenDB(node.AppStateDbName, config)
				defer appdb.Close()
				app := peptide.New(genesis.ChainID, config.HomeDir, appdb, logger)

				bsdb := server.OpenDB(node.BlockStoreDbName, config)
				defer bsdb.Close()

				// provide the partially generated genesis state so we can generate a block
				block, err := node.InitChain(app, bsdb, genesis)
				if err != nil {
					return err
				}

				genesis.GenesisBlock = eth.BlockID{
					Hash:   block.Hash(),
					Number: uint64(block.Height()),
				}

				if err := genesis.Validate(); err != nil {
					return err
				}

				if err := genesis.Save(config.HomeDir, true); err != nil {
					return err
				}

				logger.Info("genesis blocked sealed",
					"height", block.Height(),
					"hash", block.Hash().Hex(),
					"timestamp", genesis.GenesisTime.Unix(),
				)
			*/
			return nil
		},
	}

	//	server.AddSealCommandFlags(cmd)
	return cmd
}

// startCmd is responsible for setting up the interceptor node.
// It must setup the geth engine client.
func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Interceptor Node",
		RunE: func(cmd *cobra.Command, args []string) error {
			gethEngineAddr, err := cmd.Flags().GetString("geth-engine-addr")
			if err != nil {
				return err
			}

			_, err = newEngineClient(gethEngineAddr)
			if err != nil {
				return err
			}

			/*
				logger := server.DefaultLogger()
				config := server.NewConfig(cmd).
					WithHomeDir().
					WithAbciServerRpc().
					WithAbciServerGrpc().
					WithPeptideCometServerRpc().
					WithPeptideEngineServerRpc().
					WithGenesisConfig(*eetypes.NewGenesisConfig(eetypes.ZeroHash, 0)).
					WithDbBackend().
					WithPrometheusRetentionTime().
					WithAdminApi().
					WithCpuProfile().
					WithIavlLazyLoading().
					WithIavlDisableFastNode().
					WithPprofRpc().
					WithLogger(logger)

				genesis, err := node.PeptideGenesisFromFile(config.HomeDir)
				if err != nil {
					return err
				}

				appdb := server.OpenDB(node.AppStateDbName, config)
				defer appdb.Close()

				_, err = telemetry.New(
					telemetry.Config{Enabled: true, EnableHostname: false, EnableHostnameLabel: false,
						PrometheusRetentionTime: config.PrometheusRetentionTime},
				)
				if err != nil {
					return err
				}

				app := peptide.NewWithOptions(peptide.PeptideAppOptions{
					ChainID:             genesis.ChainID,
					HomePath:            config.HomeDir,
					DB:                  appdb,
					IAVLDisableFastNode: config.IavlDisableFastNode,
					IAVLLazyLoading:     config.IavlLazyLoading,
				}, logger)

				txIndexerDb := server.OpenDB(node.TxStoreDbName, config)
				defer txIndexerDb.Close()

				bsdb := server.OpenDB(node.BlockStoreDbName, config)
				defer bsdb.Close()

				peptideNode, err := node.NewPeptideNodeFromConfig(app, bsdb, txIndexerDb, genesis, config)
				if err != nil {
					return fmt.Errorf("failed to create peptide node: %w", err)
				}

				stopCpuProfiling, err := server.StartCpuProfiler(config)
				if err != nil {
					return err
				}
				defer stopCpuProfiling()

				if err := peptideNode.Service().Start(); err != nil {
					return err
				} else {
					logger.Info("Press Ctrl+C to stop the server", "homedir", config.HomeDir)
				}

				// Listen for the kill signals
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

				// Wait for the signal from sigCh, then stop Peptide Node gracefully
				sig := <-sigCh
				logger.Info("Received signal", "signal", sig)
				peptideNode.Service().Stop()
			*/

			return nil
		},
	}


	cmd.Flags().String("geth-engine-addr", "", "RPC address of geth execution engine")
	return cmd
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export chain app states as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			/*
				config := server.NewConfig(cmd).
					WithHomeDir().
					WithOuput().
					WithDbBackend().
					WithLogger(server.DefaultLogger())

				defer config.Output.Close()
				genesis, err := node.PeptideGenesisFromFile(config.HomeDir)
				if err != nil {
					return err
				}

				appdb := server.OpenDB(node.AppStateDbName, config)
				defer appdb.Close()
				app := peptide.New(genesis.ChainID, config.HomeDir, appdb, config.Logger)

				appStates := app.ExportGenesis()
				stateJSON, err := json.MarshalIndent(appStates, "", "  ")
				if err != nil {
					return err
				}

				_, err = config.Output.Write(stateJSON)
				return err
			*/
			return nil
		},
	}


	return cmd
}

func genAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-accounts",
		Short: "Generate accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			/*
				numAccounts, err := cmd.Flags().GetUint64("num-accounts")
				if err != nil {
					return err
				}

				startingSeqNum, err := cmd.Flags().GetUint64("starting-seq-num")
				if err != nil {
					return err
				}

				app := peptide.New("", "", tmdb.NewMemDB(), server.DefaultLogger())
				accounts := peptide.NewSignerAccounts(numAccounts, startingSeqNum)

				// accountsJSON, err := json.MarshalIndent(accounts, "", "  ")
				accountsJSON, err := app.AppCodec().MarshalJSON(accounts[0].PrivKey)
				if err != nil {
					return err
				}

				fmt.Println(string(accountsJSON))
			*/
			return nil
		},
	}

	//	cmd.Flags().Uint64("num-accounts", 5, "Number of accounts to generate")
	//	cmd.Flags().Uint64("starting-seq-num", 0, "Starting sequence number")

	return cmd
}

func newEngineClient(gethEngineAddr string) (*sources.EngineClient, error) {
	// necessary setup args
	ctx, m, log := context.Background(), metrics.NewMetrics(""), log.New()

	if strings.TrimSpace(gethEngineAddr) == "" {
		return nil, fmt.Errorf("geth execution engine address must be non-empty")
	}

	// TODO(colin): enable auth?
	//	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(cfg.L2EngineJWTSecret))
	opts := []client.RPCOption{
		//		client.WithGethRPCOptions(auth),
		client.WithDialBackoff(10),
	}
	rpcClient, err := client.NewRPC(ctx, log, gethEngineAddr, opts...)
	if err != nil {
		return nil, err
	}

	// TODO(colin): figure out how to populate rollupConfig
	rollupCfg := &rollup.Config{}
	rpcCfg := sources.EngineClientDefaultConfig(rollupCfg)

	engineClient, err := sources.NewEngineClient(
		client.NewInstrumentedRPC(rpcClient, m), log, m.L2SourceCache, rpcCfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Engine client: %w", err)
	}

	return engineClient, nil
}
