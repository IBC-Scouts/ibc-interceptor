# IBC Interceptor

This is a simple interceptor for the proof of concept integration of `ibc-go` with [`optimism`](https://github.com/ethereum-optimism/optimism).

The interceptor is a simple server that receives requests from `op-node` and forwards them to `geth` and an ABCI app (that uses `ibc-go`).

## Sequencing Mode Diagram

![Sequencing Mode Diagram](./op-seq-light.svg#gh-light-mode-only)
![Sequencing Mode Diagram](./op-seq-dark.svg#gh-dark-mode-only)
