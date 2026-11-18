package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func walTestCommands() []Command {
	return []Command{
		{Op: OpPoolCreate, Pool: "webapp-api", Strategy: "round_robin"},
		{Op: OpBackendAdd, Pool: "webapp-api", Backend: "api-1", Address: "10.0.0.1:3000", Weight: 5},
		{Op: OpBackendAdd, Pool: "webapp-api", Backend: "api-2", Address: "10.0.0.2:3000", Weight: 1},
		{Op: OpRouteAdd, Host: "app.example.com", PathPrefix: "/api/", Pool: "webapp-api"},
		{Op: OpWeightSet, Pool: "webapp-api", Backend: "api-1", Weight: 3},
	}
}

// writes the commands with seq 1, 2, 3... and returns the file size after each
// record: ends[k] is the byte offset where the first k records end (ends[0] = 0)
func writeTestLog(t *testing.T, path string, cmds []Command) []int64 {
	t.Helper()
	w, err := openWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	ends := []int64{0}
	for i, c := range cmds {
		c.Seq = uint64(i + 1)
		if err := w.Append(c); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		ends = append(ends, info.Size())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return ends
}

func collect(path string) ([]Command, int64, error) {
	var got []Command
	end, err := readLog(path, func(c Command) error {
		got = append(got, c)
		return nil
	})
	return got, end, err
}

func TestWALRoundTrip(t *testing.T) {
	//path := filepath.Join(t.TempDir(), "wal.log")
	path := filepath.Join(testDir(t), "wal.log")
	cmds := walTestCommands()
	ends := writeTestLog(t, path, cmds)

	got, validEnd, err := collect(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(cmds) {
		t.Fatalf("got %d records, want %d", len(got), len(cmds))
	}
	for i, c := range cmds {
		c.Seq = uint64(i + 1)
		if got[i] != c {
			t.Fatalf("record %d: got %+v, want %+v", i, got[i], c)
		}
	}
	if validEnd != ends[len(ends)-1] {
		t.Fatalf("validEnd: got %d, want %d", validEnd, ends[len(ends)-1])
	}
}

// The crash simulation: cut the file at EVERY possible length.
// Reading must never fail, and must return exactly the records that fit completely.
func TestWALTornWrite(t *testing.T) {
	//dir := t.TempDir()
	dir := testDir(t)
	ends := writeTestLog(t, filepath.Join(dir, "wal.log"), walTestCommands())
	data, err := os.ReadFile(filepath.Join(dir, "wal.log"))
	if err != nil {
		t.Fatal(err)
	}

	cutPath := filepath.Join(dir, "cut.log")
	for n := 0; n <= len(data); n++ {
		if err := os.WriteFile(cutPath, data[:n], 0o600); err != nil {
			t.Fatal(err)
		}
		got, validEnd, err := collect(cutPath)
		if err != nil {
			t.Fatalf("cut at %d: unexpected error: %v", n, err)
		}

		want := 0 // how many complete records fit in n bytes
		for want+1 < len(ends) && ends[want+1] <= int64(n) {
			want++
		}
		if len(got) != want {
			t.Fatalf("cut at %d: got %d records, want %d", n, len(got), want)
		}
		if validEnd != ends[want] {
			t.Fatalf("cut at %d: validEnd %d, want %d", n, validEnd, ends[want])
		}
	}
}

// One flipped byte in the middle: reading stops there, earlier records survive.
func TestWALCorruption(t *testing.T) {
	//dir := t.TempDir()
	dir := testDir(t)

	path := filepath.Join(dir, "wal.log")
	ends := writeTestLog(t, path, walTestCommands())

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[ends[2]+headerSize+2] ^= 0xFF // a byte inside the 3rd record's body
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	got, validEnd, err := collect(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2 (the ones before the damage)", len(got))
	}
	if validEnd != ends[2] {
		t.Fatalf("validEnd: got %d, want %d", validEnd, ends[2])
	}
}

// Closing and opening again must continue the same log.
func TestWALReopenAndContinue(t *testing.T) {
	//path := filepath.Join(t.TempDir(), "wal.log")
	path := filepath.Join(testDir(t), "wal.log")
	cmds := walTestCommands()

	w, err := openWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cmds[:3] {
		c.Seq = uint64(i + 1)
		if err := w.Append(c); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()

	w, err = openWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cmds[3:] {
		c.Seq = uint64(i + 4)
		if err := w.Append(c); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()

	got, _, err := collect(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d records, want 5", len(got))
	}
	if !reflect.DeepEqual(got[4].Backend, "api-1") || got[4].Seq != 5 {
		t.Fatalf("last record wrong: %+v", got[4])
	}
}

func TestReadLogMissingFile(t *testing.T) {
	_, _, err := collect(filepath.Join(t.TempDir(), "nope.log"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want an os.ErrNotExist error, got %v", err)
	}
}

func TestReadLogCallbackErrorStops(t *testing.T) {
	//path := filepath.Join(t.TempDir(), "wal.log")
	path := filepath.Join(testDir(t), "wal.log")
	writeTestLog(t, path, walTestCommands())

	boom := errors.New("boom")
	count := 0
	_, err := readLog(path, func(Command) error {
		count++
		if count == 2 {
			return boom
		}
		return nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("want the callback's error, got %v", err)
	}
	if count != 2 {
		t.Fatalf("callback ran %d times, want 2", count)
	}
}

// testDir returns a fresh, empty folder for one test. Normally Go deletes it
// afterwards. With KEEP_TEST_DATA=1 it is kept, and its path is logged.
func testDir(t *testing.T) string {
	t.Helper()

	if os.Getenv("KEEP_TEST_DATA") == "" {
		return t.TempDir()
	}

	dir, err := os.MkdirTemp("", "proxez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("keeping test data in %s", dir)
	return dir
}
