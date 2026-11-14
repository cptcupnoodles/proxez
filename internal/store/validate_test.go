package store

import (
	"reflect"
	"testing"
)

func baseState() *State {
	s := NewState()
	s.Pools["p"] = &PoolState{
		Name:     "p",
		Strategy: "round_robin",
		Backends: []BackendState{
			{Name: "alpha", Address: "127.0.0.1:9000", Weight: 1},
		},
	}
	s.Routes = []RouteState{
		{Host: "h", PathPrefix: "/", Pool: "p"},
	}
	return s
}

func TestCreatePoolOnEmptyState(t *testing.T) {
	s := NewState()
	c := Command{Op: OpPoolCreate, Pool: "storage", Strategy: "round_robin"}
	if err := s.Validate(c); err != nil {
		t.Fatalf("Error creating pool: %v", err)
	}
}

func TestValidateAccepts(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
	}{
		{"pool create", Command{Op: OpPoolCreate, Pool: "new", Strategy: "round_robin"}},
		{"strategy set", Command{Op: OpStrategySet, Pool: "p", Strategy: "least_connection"}},
		{"backend add", Command{Op: OpBackendAdd, Pool: "p", Backend: "b2", Address: "10.0.0.2:80", Weight: 2}},
		{"backend remove", Command{Op: OpBackendRemove, Pool: "p", Backend: "alpha"}},
		{"weight set", Command{Op: OpWeightSet, Pool: "p", Backend: "alpha", Weight: 9}},
		{"route add", Command{Op: OpRouteAdd, Host: "h", PathPrefix: "/media/", Pool: "p"}},
		{"route remove", Command{Op: OpRouteRemove, Host: "h", PathPrefix: "/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := baseState().Validate(tt.cmd); err != nil {
				t.Fatalf("should be accepted, got: %v", err)
			}
		})
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		Name string
		cmd  Command
	}{
		{"pool create: already exists", Command{Op: OpPoolCreate, Pool: "p", Strategy: "round_robin"}},
		{"pool create: empty name", Command{Op: OpPoolCreate, Pool: "", Strategy: "round_robin"}},
		{"pool create: bad character", Command{Op: OpPoolCreate, Pool: "a b", Strategy: "round_robin"}},
		{"pool create: unknown strategy", Command{Op: OpPoolCreate, Pool: "new", Strategy: "nope"}},

		{"pool delete: missing", Command{Op: OpPoolDelete, Pool: "nope"}},
		{"pool delete: used by a route", Command{Op: OpPoolDelete, Pool: "p"}},

		{"strategy set: missing pool", Command{Op: OpStrategySet, Pool: "nope", Strategy: "round_robin"}},
		{"strategy set: unknown strategy", Command{Op: OpStrategySet, Pool: "p", Strategy: "nope"}},

		{"backend add: missing pool", Command{Op: OpBackendAdd, Pool: "nope", Backend: "b2", Address: "10.0.0.2:80", Weight: 1}},
		{"backend add: duplicate name", Command{Op: OpBackendAdd, Pool: "p", Backend: "alpha", Address: "10.0.0.9:80", Weight: 1}},
		{"backend add: bad address", Command{Op: OpBackendAdd, Pool: "p", Backend: "b2", Address: "nope", Weight: 1}},
		{"backend add: zero weight", Command{Op: OpBackendAdd, Pool: "p", Backend: "b2", Address: "10.0.0.2:80", Weight: 0}},
		{"backend add: empty name", Command{Op: OpBackendAdd, Pool: "p", Backend: "", Address: "10.0.0.2:80", Weight: 1}},

		{"backend remove: missing pool", Command{Op: OpBackendRemove, Pool: "nope", Backend: "alpha"}},
		{"backend remove: missing backend", Command{Op: OpBackendRemove, Pool: "p", Backend: "nope"}},

		{"weight set: missing pool", Command{Op: OpWeightSet, Pool: "nope", Backend: "alpha", Weight: 2}},
		{"weight set: missing backend", Command{Op: OpWeightSet, Pool: "p", Backend: "nope", Weight: 2}},
		{"weight set: zero weight", Command{Op: OpWeightSet, Pool: "p", Backend: "alpha", Weight: 0}},

		{"route add: missing pool", Command{Op: OpRouteAdd, Host: "h", PathPrefix: "/x/", Pool: "nope"}},
		{"route add: duplicate", Command{Op: OpRouteAdd, Host: "h", PathPrefix: "/", Pool: "p"}},
		{"route add: prefix without slash", Command{Op: OpRouteAdd, Host: "h", PathPrefix: "x", Pool: "p"}},
		{"route add: empty host", Command{Op: OpRouteAdd, Host: "", PathPrefix: "/x/", Pool: "p"}},

		{"route remove: missing", Command{Op: OpRouteRemove, Host: "h", PathPrefix: "/nope/"}},

		{"unknown op 0", Command{Op: 0}},
		{"unknown op 99", Command{Op: 99}},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if err := baseState().Validate(tt.cmd); err == nil {
				t.Fatal("should be rejected, but Validate returned nil")
			}
		})
	}
}

func TestValidateDoesNotMutate(t *testing.T) {
	s := baseState()

	_ = s.Validate(Command{Op: OpPoolCreate, Pool: "new", Strategy: "round_robin"})
	_ = s.Validate(Command{Op: OpBackendAdd, Pool: "p", Backend: "b2", Address: "10.0.0.2:80", Weight: 1})
	_ = s.Validate(Command{Op: OpBackendRemove, Pool: "p", Backend: "b1"})
	_ = s.Validate(Command{Op: OpPoolDelete, Pool: "p"})

	if !reflect.DeepEqual(s, baseState()) {
		t.Fatal("Validate changed the state")
	}
}
