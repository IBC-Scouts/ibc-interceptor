package types

import (
	"crypto/sha256"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// -------------- Composite Payload --------------

type CompositePayload struct {
	// NOTE!: Both payloads may be nil.
	GethPayload *eth.PayloadID
	ABCIPayload *eth.PayloadID
}

func NewCompositePayload(gethPayload, abciPayload *eth.PayloadID) CompositePayload {
	return CompositePayload{
		GethPayload: gethPayload,
		ABCIPayload: abciPayload,
	}
}

func (p CompositePayload) Payload() *eth.PayloadID {
	// NOTE!: Guarantees uniqueness, no?
	s := ""
	if p.GethPayload != nil {
		s = p.GethPayload.String()
	}

	if p.ABCIPayload != nil {
		s += p.ABCIPayload.String()
	}

	hash := sha256.Sum256([]byte(s))
	payloadID := eth.PayloadID(hash[:8])

	return &payloadID
}
