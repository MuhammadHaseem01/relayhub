package health

import (
	"sync"
	"time"
)

type State string

const (
	StateHealthy   State = "healthy"
	StateUnhealthy State = "unhealthy"
)

const (
	DefaultFailThreshold = 5
	DefaultOpenDuration  = 60 * time.Second
)

type providerState struct {
	consecutiveFailures int
	isOpen              bool
	openedAt            time.Time
	halfOpenInFlight    bool
}

type Registry struct {
	mu            sync.Mutex
	providers     map[string]*providerState
	failThreshold int
	openDuration  time.Duration
}

func NewRegistry(providerNames []string) *Registry {
	return NewRegistryWithConfig(providerNames, DefaultFailThreshold, DefaultOpenDuration)
}

func NewRegistryWithConfig(providerNames []string, failThreshold int, openDuration time.Duration) *Registry {
	if failThreshold <= 0 {
		failThreshold = DefaultFailThreshold
	}
	if openDuration <= 0 {
		openDuration = DefaultOpenDuration
	}

	m := make(map[string]*providerState, len(providerNames))
	for _, name := range providerNames {
		m[name] = &providerState{}
	}

	return &Registry{
		providers:     m,
		failThreshold: failThreshold,
		openDuration:  openDuration,
	}
}

func (r *Registry) IsHealthy(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	ps, ok := r.providers[name]
	if !ok {
		return true
	}

	if !ps.isOpen {
		return true
	}

	if time.Since(ps.openedAt) >= r.openDuration {
		if !ps.halfOpenInFlight {
			ps.halfOpenInFlight = true
			return true
		}
		return false
	}

	return false
}

func (r *Registry) RecordSuccess(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ps, ok := r.providers[name]
	if !ok {
		return
	}

	ps.consecutiveFailures = 0
	ps.isOpen = false
	ps.halfOpenInFlight = false
}

func (r *Registry) RecordFailure(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ps, ok := r.providers[name]
	if !ok {
		return
	}

	ps.consecutiveFailures++
	ps.halfOpenInFlight = false

	if ps.isOpen || ps.consecutiveFailures >= r.failThreshold {
		ps.isOpen = true
		ps.openedAt = time.Now()
	}
}

func (r *Registry) Snapshot() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	res := make(map[string]string, len(r.providers))
	now := time.Now()

	for name, ps := range r.providers {
		if ps.isOpen && now.Sub(ps.openedAt) < r.openDuration {
			res[name] = string(StateUnhealthy)
		} else {
			res[name] = string(StateHealthy)
		}
	}
	return res
}
