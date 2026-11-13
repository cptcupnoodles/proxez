package store

import (
	"errors"
	"fmt"
	"net"
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

		// state: no backend with this name may already be in that pool
		// (optional, your decision) state: reject a duplicate address in the pool
		for _, b := range p.Backends {
			if b.Name == c.Backend {
				return fmt.Errorf("backend %q already in pool %q", c.Backend, c.Pool)
			}
		}

	case OpBackendRemove:
		// state: the pool must exist
		p, ok := s.Pools[c.Pool]
		if !ok {
			return fmt.Errorf("pool %q does not exist", c.Pool)
		}

		// state: a backend with this name must exist in that pool
		found := false
		for _, b := range p.Backends {
			if b.Name == c.Backend {
				found = true
			}
		}

	case OpWeightSet:
		if c.Weight <= 0 {
			return fmt.Errorf("weight should be atleast 1")
		}

		p, ok := s.Pools[c.Pool]
		if !ok {
			return fmt.Errorf("pool %q dies not exist", c.Pool)
		}

		found := false
		for _, b := range p.Backends {
			if b.Name == c.Backend {
				found = true
			}
		}

	case OpRouteAdd:
		// shape: host is not empty
		// shape: path prefix starts with "/"
		// state: the pool it points to must exist
		// state: no route with the same host AND path prefix may already exist

	case OpRouteRemove:
		// state: a route with this host AND path prefix must exist

	default:
		return fmt.Errorf("unknown op %d", c.Op)
	}
	return nil
}

func (s *State) Apply(c Command) {

}

const maxNameLen = 100

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

var knownStrategies = map[string]bool{
	"round_robin":                 true,
	"smooth_weighted_round_robin": true,
	"least_connection":            true,
}

func checkStrategy(s string) error {
	if !knownStrategies[s] {
		return fmt.Errorf("unknown strategy %q", s)
	}

	return nil
}

func checkRoute(host, prefix string) error {
	if host == "" {
		return fmt.Errorf("route hostis required")
	}

	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("path prefix %q must start with /", prefix)
	}
	return nil
}
