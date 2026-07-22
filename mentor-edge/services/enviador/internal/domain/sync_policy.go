package domain

import "sync"

type SyncPolicy struct {
	BatchSize    int
	PollInterval int
	mu           sync.RWMutex
}

func NewDefaultSyncPolicy() *SyncPolicy {
	return &SyncPolicy{
		BatchSize:    100,
		PollInterval: 300,
	}
}

func (p *SyncPolicy) GetBatchSize() int {
	return p.BatchSize
}

func (p *SyncPolicy) GetPollInterval() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.PollInterval
}

func (p *SyncPolicy) SetPollInterval(seconds int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if seconds >= 10 {
		p.PollInterval = seconds
	}
}
