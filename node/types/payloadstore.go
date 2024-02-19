package types

import (
	"fmt"
	"sync"
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
