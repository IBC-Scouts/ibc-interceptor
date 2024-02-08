package engine

import "github.com/ethereum-optimism/optimism/op-service/sources"

type Node interface {
	GetGethClient() *sources.EngineClient
}
