package types

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
)

type CompositeBlock struct {
	GethHash common.Hash
	ABCIHash common.Hash
}

func NewCompositeBlock(gethHash, abciHash common.Hash) CompositeBlock {
	return CompositeBlock{
		GethHash: gethHash,
		ABCIHash: abciHash,
	}
}

func (b CompositeBlock) Hash() common.Hash {
	buf := b.GethHash.Bytes()
	buf = append(buf, b.ABCIHash.Bytes()...)

	hash := sha256.Sum256(buf)
	return common.BytesToHash(hash[:])
}
