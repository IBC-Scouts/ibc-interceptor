package types

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadStore interface {
	Add(payload *Payload) error
	Get(id PayloadID) (*Payload, bool)
	Current() *Payload
	RollbackToHeight(height int64) error
}

type pstore struct {
	mutex    sync.Mutex
	payloads map[PayloadID]*Payload
	heights  map[int64]PayloadID
	current  *Payload
}

var _ PayloadStore = (*pstore)(nil)

func NewPayloadStore() PayloadStore {
	return &pstore{
		mutex:    sync.Mutex{},
		payloads: make(map[PayloadID]*Payload),
		heights:  make(map[int64]PayloadID),
	}
}

func (p *pstore) Add(payload *Payload) error {
	if payload == nil {
		return fmt.Errorf("could not add invalid payload")
	}
	id, err := payload.GetPayloadID()
	if err != nil {
		return fmt.Errorf("could not add payload, %w", err)
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if _, ok := p.payloads[*id]; !ok {
		p.heights[payload.Height] = *id
		p.payloads[*id] = payload
		p.current = payload
	}
	return nil
}

func (p *pstore) Get(id PayloadID) (*Payload, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if payload, ok := p.payloads[id]; ok {
		return payload, true
	}
	return nil, false
}

func (p *pstore) Current() *Payload {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.current
}

func (p *pstore) RollbackToHeight(height int64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// nuke everything in memory
	p.current = nil
	p.heights = make(map[int64]PayloadID)
	p.payloads = make(map[PayloadID]*Payload)

	return nil
}

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
