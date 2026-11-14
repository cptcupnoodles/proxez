package strategy

import (
	"fmt"
	"proxez/m/internal/backend"
	"sync"
)

type Strategy struct {
	mu            sync.Mutex
	next          int
	currentWeight map[*backend.Backend]int
}

func NewStrategy() *Strategy {
	return &Strategy{
		currentWeight: make(map[*backend.Backend]int),
	}
}

func (s *Strategy) RoundRobin(backends []*backend.Backend) (*backend.Backend, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends configured")
	}

	for range len(backends) {
		index := s.next
		s.next = (s.next + 1) % len(backends)

		if backends[index].Health() {
			return backends[index], nil
		}
	}

	return nil, fmt.Errorf("no healthy backends available")
}

func (s *Strategy) WeightedRoundRobin(backends []*backend.Backend) (*backend.Backend, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var totalWeight int

	// Add each healthy backend's weight.
	for _, backend := range backends {
		if !backend.Health() || backend.Weight <= 0 {
			continue
		}

		s.currentWeight[backend] += backend.Weight
		totalWeight += backend.Weight
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	var selected *backend.Backend
	// Select the backend with the highest current weight.
	for _, backend := range backends {
		if !backend.Health() || backend.Weight <= 0 {
			continue
		}

		if selected == nil || s.currentWeight[backend] > s.currentWeight[selected] {
			selected = backend
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no healthy backends available")
	}

	// subtract one complete scheduling cycle
	s.currentWeight[selected] -= totalWeight

	// Forward the request to the selected backend.
	return selected, nil
}
