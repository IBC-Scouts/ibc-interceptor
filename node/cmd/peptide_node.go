package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/onsi/gomega/gexec"
)

const (
	// Default address for peptide app rpc.
	appRpcAddress = "localhost:0"
	// Default address for peptide engine rpc.
	eeHttpServerAddress = "localhost:35462"
	// Same as eeHttpServerAddress, used by us to connect to peptide.
	eeWsUrl = "ws://127.0.0.1:35462/websocket"
)

// Get the binary path for the peptide binary.
func getBinaryPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	basePath := strings.SplitAfter(wd, "op-e2e/")[0]
	binPath := fmt.Sprintf("%s/%s", basePath, "interceptor-node/peptide")

	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("could not locate interceptor in working directory: %w", err)
	}

	return binPath, nil
}

// Run the peptide binary using in memory dbs. I do not want to debug the cause behind init/seal/start
// invocation.
func PeptideStart() error {
	binPath, err := getBinaryPath()
	if err != nil {
		return fmt.Errorf("could not get binary path: %w", err)
	}

	// ./binary start-in-mem --app-rpc-address localhost:35463 --ee-http-server-address localhost:35462
	cmd := exec.Command(
		binPath,
		"start-in-mem",
		"--app-rpc-address", appRpcAddress,
		"--ee-http-server-address", eeHttpServerAddress,
	)
	fmt.Printf("Running start-in-mem command: %s\n", cmd.String())

	// Run the command.
	_, err = gexec.Start(cmd, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("could not run start-in-mem command: %w", err)
	}

	return nil
}
