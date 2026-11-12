package store

import (
	"fmt"
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
		// shape: pool name is valid (not empty, not too long, safe characters)
		if err := checkName("pool", c.Pool); err != nil {
			return err
		}
		// shape: strategy is one of the known strategies
		return checkStrategy(c.Strategy)
		// state: a pool with this name must NOT already exist
		if _, ok := s.Pool[c.Pool]; ok {
			return fmt.Errorf("pool %q already exist", c.Pool)
		}

	case OpPoolDelete:
		// state: the pool must exist
		// state: no route may still point to this pool
		//        (loop over s.Routes, compare r.Pool)

	case OpStrategySet:
		// shape: strategy is one of the known strategies
		// state: the pool must exist

	case OpBackendAdd:
		// shape: backend name is valid
		// shape: address is a valid host:port (net.SplitHostPort)
		// shape: weight is at least 1
		// state: the pool must exist
		// state: no backend with this name may already be in that pool
		// (optional, your decision) state: reject a duplicate address in the pool

	case OpBackendRemove:
		// state: the pool must exist
		// state: a backend with this name must exist in that pool

	case OpWeightSet:
		// shape: weight is at least 1
		// state: the pool must exist
		// state: the backend must exist in that pool

	case OpRouteAdd:
		// shape: host is not empty
		// shape: path prefix starts with "/"
		// state: the pool it points to must exist
		// state: no route with the same host AND path prefix may already exist

	case OpRouteRemove:
		// state: a route with this host AND path prefix must exist

	default:
		// unknown op (including 0): return an error
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
