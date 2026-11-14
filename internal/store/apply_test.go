package store

import "testing"

func TestApplyPoolCreate(t *testing.T) {
	s := NewState()
	c := Command{
		Op:       OpPoolCreate,
		Pool:     "storage",
		Strategy: "round_robin",
	}

	// the command must be valid before we apply it
	if err := s.Validate(c); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	s.Apply(c)

	// the pool now exist
	p, ok := s.Pools["storage"]
	if !ok {
		t.Fatal("Pool storage is not created")
	}

	// it has the values from the command
	if p.Name != "storage" {
		t.Fatalf("name: got %q, want %q", p.Name, "storage")
	}
	if p.Strategy != "round_robin" {
		t.Fatalf("strategy: got %q, want %q", p.Strategy, "round_robin")
	}

	// a new pool starts with no backends
	if len(p.Backends) != 0 {
		t.Fatalf("backends: got %d, want %d", len(p.Backends), 0)
	}

	// add exactly one pool exists
	if len(s.Pools) != 1 {
		t.Fatalf("pools: got %d, want %d", len(s.Pools), 1)
	}
}

func TestApplyPoolCreateTwice(t *testing.T) {
	s := NewState()
	first := Command{Op: OpPoolCreate, Pool: "a", Strategy: "round_robin"}
	second := Command{Op: OpPoolCreate, Pool: "b", Strategy: "least_connection"}

	for _, c := range []Command{first, second} {
		if err := s.Validate(c); err != nil {
			t.Fatalf("validate %+v: %v", c, err)
		}
		s.Apply(c)
	}

	// the second create must not overwrite or change the first
	if got := s.Pools["a"].Strategy; got != "round_robin" {
		t.Fatalf("pool a strategy: got %q, want round_robin", got)
	}
	if got := s.Pools["b"].Strategy; got != "least_connection" {
		t.Fatalf("pool b strategy: got %q, want least_connection", got)
	}

	// creating the same name again is rejected by Validate
	if err := s.Validate(first); err == nil {
		t.Fatal("creating pool a again should be rejected")
	}
}
