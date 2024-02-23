package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/ibc-scouts/ibc-interceptor/node"
	"github.com/ibc-scouts/ibc-interceptor/types"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "interceptor",
		Short: "Interceptor execution engine for OP stack.",
	}

	// TODO(colin): decide necessary commands
	// jim: Can go with start for now. It calls peptide.
	rootCmd.AddCommand(startCmd())

	return rootCmd
}

// startCmd is responsible for setting up the interceptor node.
// It does the following:
//  1. Reads the config file containing high level configuration for the interceptor node.
//  2. Starts the peptide node which is the sdk execution engine for the interceptor node.
//  3. Starts the interceptor node.
func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Interceptor Node",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFilePath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			config, err := types.ConfigFromFilePath(configFilePath)
			if err != nil {
				return err
			}

			// Uses eeHttpServerAddress const, we can control where peptide binds.
			if err = PeptideStart(); err != nil {
				return err
			}
			config.PeptideEngineAddr = eeWsUrl
			// Sleep for a bit before starting the interceptor node.
			time.Sleep(2 * time.Second)

			gethEngineAddr, err := cmd.Flags().GetString("geth-engine-addr")
			if err != nil {
				return err
			}

			if gethEngineAddr != "" {
				config.GethEngineAddr = gethEngineAddr
			}
			node := node.NewInterceptorNode(config)

			if err := node.Start(); err != nil {
				return err
			}

			// Wait for interrupt signal to gracefully shut down the server
			quit := make(chan os.Signal, 1)
			// catch SIGINT (Ctrl+C) and SIGTERM
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit // Block until a signal is received

			// Perform any cleanup and shutdown tasks here
			// For example, gracefully shutting down the RPC server
			if err := node.Stop(); err != nil {
				return err // or log the error instead of returning it
			}

			return nil
		},
	}

	cmd.Flags().String("geth-engine-addr", "", "RPC address of geth execution engine")
	cmd.Flags().String("config", types.DefaultConfigFilePath, "Path to the interceptor config file")

	return cmd
}
