package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func testCommands() []Command {
	return []Command{
		{Op: OpPoolCreate, Pool: "webapp-api", Strategy: "round_robin"},
		{Op: OpBackendAdd, Pool: "webapp-api", Backend: "api-1", Address: "10.0.0.1:3000", Weight: 5},
		{Op: OpBackendAdd, Pool: "webapp-api", Backend: "api-2", Address: "10.0.0.2:3000", Weight: 1},
		{Op: OpRouteAdd, Host: "app.example.com", PathPrefix: "/api/", Pool: "webapp-api"},
	}
}

func TestStoreSurvivesReopen(t *testing.T) {
	dir := t.TempDir()

	s1, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range testCommands() {
		if err := s1.Do(c); err != nil {
			t.Fatalf("Do %+v: %v", c, err)
		}
	}
	want := s1.state
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	if !reflect.DeepEqual(s2.state, want) {
		t.Fatalf("state after reopen differs:\n got %+v\nwant %+v", s2.state, want)
	}
	if s2.lastSeq != uint64(len(testCommands())) {
		t.Fatalf("lastSeq: got %d, want %d", s2.lastSeq, len(testCommands()))
	}
}

func TestStoreSecondOpenFails(t *testing.T) {
	dir := t.TempDir()

	s1, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()

	if _, err := Open(dir, nil); err == nil {
		t.Fatal("second Open on the same folder should fail")
	}
}

func TestStoreRejectedCommandIsNotLogged(t *testing.T) {
	dir := t.TempDir()

	s1, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	bad := Command{Op: OpBackendAdd, Pool: "missing", Backend: "x", Address: "1.1.1.1:1", Weight: 1}
	if err := s1.Do(bad); err == nil {
		t.Fatal("command for a missing pool should be rejected")
	}
	s1.Close()

	s2, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if s2.lastSeq != 0 || len(s2.state.Pools) != 0 {
		t.Fatalf("rejected command leaked into the log: lastSeq=%d pools=%d", s2.lastSeq, len(s2.state.Pools))
	}
}

func TestStoreRecoversFromTornTail(t *testing.T) {
	dir := t.TempDir()

	s1, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range testCommands()[:3] {
		if err := s1.Do(c); err != nil {
			t.Fatal(err)
		}
	}
	s1.Close()

	// simulate a crash in the middle of a write: a few junk bytes at the end of the log
	f, err := os.OpenFile(filepath.Join(dir, walFileName), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte{1, 2, 3, 4, 5})
	f.Close()

	s2, err := Open(dir, nil)
	if err != nil {
		t.Fatalf("Open should recover from a torn tail: %v", err)
	}
	if s2.lastSeq != 3 {
		t.Fatalf("lastSeq: got %d, want 3", s2.lastSeq)
	}

	// the junk must be gone, so a new record is readable after another restart
	if err := s2.Do(testCommands()[3]); err != nil {
		t.Fatal(err)
	}
	s2.Close()

	s3, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s3.Close()
	if s3.lastSeq != 4 {
		t.Fatalf("lastSeq after third open: got %d, want 4", s3.lastSeq)
	}
}
