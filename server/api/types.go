// types.go holds any additional required type definitions for the server implementations.
package api

// TODO(jim): Ethereum JSON/RPC dictates responses should either return 0, 1 (response or error) or 2 (response and error).
// For now, we return 2 just to keep separated.
type SendCosmosTxResult struct{}
