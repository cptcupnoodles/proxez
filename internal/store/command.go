// Package store holds the load balancer's saved configuration: which pools
// exist, which backends each pool has, and which routes send traffic to which pool.

package store

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
)

type Op uint8

const (
	OpPoolCreate Op = iota + 1
	OpPoolDelete
	OpStrategySet
	OpBackendAdd
	OpBackendRemove
	OpWeightSet
	OpRouteAdd
	OpRouteRemove
)

type Command struct {
	Seq        uint64
	Op         Op
	Pool       string
	Strategy   string
	Backend    string
	Address    string
	Weight     uint32
	Host       string
	PathPrefix string
}

func (s *State) Validate(c Command) error {
	switch c.Op {

	case OpPoolCreate:
		if err := checkName("pool", c.Pool); err != nil {
			return err
		}

		if err := checkStrategy(c.Strategy); err != nil {
			return err
		}

		if _, ok := s.Pools[c.Pool]; ok {
			return fmt.Errorf("pool %q already exist", c.Pool)
		}

	case OpPoolDelete:
		if _, ok := s.Pools[c.Pool]; !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

		for _, r := range s.Routes {
			if r.Pool == c.Pool {
				return fmt.Errorf("pool %q is used by route %s%s", c.Pool, r.Host, r.PathPrefix)
			}
		}

	case OpStrategySet:
		if err := checkStrategy(c.Strategy); err != nil {
			return err
		}

		if _, ok := s.Pools[c.Pool]; !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

	case OpBackendAdd:
		if err := checkName("backend", c.Backend); err != nil {
			return err
		}
		// address is a valid host:port (net.SplitHostPort)
		if _, _, err := net.SplitHostPort(c.Address); err != nil {
			return fmt.Errorf("invalid address %q: %w", c.Address, err)
		}

		if c.Weight == 0 {
			return errors.New("weight must be at least 1")
		}

		p, ok := s.Pools[c.Pool]
		if !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

		for _, b := range p.Backends {
			if b.Name == c.Backend {
				return fmt.Errorf("backend %q already in pool %q", c.Backend, c.Pool)
			}
		}

	case OpBackendRemove:
		p, ok := s.Pools[c.Pool]
		if !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

		if findBackend(p, c.Backend) < 0 {
			return fmt.Errorf("backend %q does not exist in pool %q", c.Backend, c.Pool)
		}

	case OpWeightSet:
		if c.Weight == 0 {
			return fmt.Errorf("weight should be at least 1")
		}

		p, ok := s.Pools[c.Pool]
		if !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

		if findBackend(p, c.Backend) < 0 {
			return fmt.Errorf("backend %q does not exist in pool %q", c.Backend, c.Pool)
		}

	case OpRouteAdd:
		if err := checkRoute(c.Host, c.PathPrefix); err != nil {
			return err
		}

		if _, ok := s.Pools[c.Pool]; !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

		for _, v := range s.Routes {
			if v.Host == c.Host && v.PathPrefix == c.PathPrefix {
				return fmt.Errorf("route %q already exists in pool %q", v.Host, c.Pool)
			}
		}
	case OpRouteRemove:
		found := false
		for _, v := range s.Routes {
			if v.Host == c.Host && v.PathPrefix == c.PathPrefix {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("route %q%q does not exist", c.Host, c.PathPrefix)
		}

	default:
		return fmt.Errorf("unknown op %d", c.Op)
	}
	return nil
}

func (s *State) Apply(c Command) {
	switch c.Op {

	case OpPoolCreate:
		// add a new PoolState to s.Pools: name, strategy, no backends yet
		s.Pools[c.Pool] = &PoolState{Name: c.Pool, Strategy: c.Strategy}

	case OpPoolDelete:
		// delete the pool from s.Pools
		delete(s.Pools, c.Pool)

	case OpStrategySet:
		// set the Strategy field on the pool
		s.Pools[c.Pool].Strategy = c.Strategy

	case OpBackendAdd:
		// append a BackendState (name, address, weight) to the pool's Backends
		p := s.Pools[c.Pool]
		p.Backends = append(s.Pools[c.Pool].Backends, BackendState{
			Name:    c.Backend,
			Address: c.Address,
			Weight:  c.Weight,
		})

	case OpBackendRemove:
		// find the backend's index with findBackend
		p, _ := s.Pools[c.Pool]
		i := findBackend(p, c.Backend)
		// remove it from the slice:
		p.Backends = slices.Delete(p.Backends, i, i+1)

	case OpWeightSet:
		// find the backend's index with findBackend
		p, _ := s.Pools[c.Pool]
		i := findBackend(p, c.Backend)
		// set Backends[i].Weight
		p.Backends[i].Weight = c.Weight

	case OpRouteAdd:
		// append a RouteState (host, prefix, pool) to s.Routes
		s.Routes = append(s.Routes, RouteState{
			Host:       c.Host,
			PathPrefix: c.PathPrefix,
			Pool:       c.Pool,
		})

	case OpRouteRemove:
		// find the route with this host AND prefix
		for i, r := range s.Routes {
			if r.Host == c.Host && r.PathPrefix == c.PathPrefix {
				// remove it from s.Routes (same slice trick as the backend removal)
				s.Routes = slices.Delete(s.Routes, i, i+1)
				break
			}
		}

	default:
		panic(fmt.Sprintf("store: Apply called with unknown op %d", c.Op))
	}
}

// ---------------------------------------------------------------------------
// Helpers
//
// Small functions shared by Validate and Apply. They hold the rules
// that several ops have in common, so each rule is written in one place:
//
//   - checkName, checkStrategy and checkRoute check the shape of a value.
//     They look only at the value itself, never at the current State.
//   - findBackend looks up an item in the State and returns its index.
//
// ---------------------------------------------------------------------------

const maxNameLen = 100

var knownStrategies = map[string]bool{
	"round_robin":                 true,
	"smooth_weighted_round_robin": true,
	"least_connection":            true,
}

func checkName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is required", kind)
	}

	if len(name) > maxNameLen {
		return fmt.Errorf("%s name too long (max %d)", kind, maxNameLen)
	}

	for _, r := range name {
		ok := r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '-' || r == '_' || r == '.'
		if !ok {
			return fmt.Errorf("%s name %q has invalid character %q", kind, name, r)
		}
	}
	return nil

}

func checkStrategy(s string) error {
	if !knownStrategies[s] {
		return fmt.Errorf("unknown strategy %q", s)
	}

	return nil
}

func checkRoute(host, prefix string) error {
	if host == "" {
		return fmt.Errorf("route host required")
	}

	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("path prefix %q must start with /", prefix)
	}
	return nil
}

func findBackend(p *PoolState, name string) int {
	for i, b := range p.Backends {
		if b.Name == name {
			return i
		}
	}
	return -1
}
