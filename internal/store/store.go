package store

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

const walFileName = "wal.log"

var errClosed = errors.New("store is closed")

// Store keeps the saved config. It owns the data folder: it holds the lock on
// it, replays the log into memory at startup, and is the only way to change
// the config afterwards (through Do).
type Store struct {
	mu       sync.Mutex
	dir      string
	lock     *os.File // held open for the Store's whole life; closing it releases the lock
	wal      *WAL
	state    *State
	lastSeq  uint64
	onChange func(*State)
	failed   error // set after a write failure or Close; blocks further writes
}

// Open opens the data folder (creating it if needed), takes the lock on it,
// rebuilds the state by replaying the log, and returns a Store ready for Do.
// onChange is called once after recovery and again after every successful
// command. It runs while the Store's lock is held and receives the live state,
// so it must copy whatever it needs and must not keep the pointer.
func Open(dir string, onChange func(*State)) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	lock, err := lockDir(dir)
	if err != nil {
		return nil, err
	}

	s := &Store{
		dir:      dir,
		lock:     lock,
		state:    NewState(),
		onChange: onChange,
	}

	if err := s.recover(); err != nil {
		lock.Close()
		return nil, err
	}

	w, err := openWAL(filepath.Join(dir, walFileName))
	if err != nil {
		err := lock.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	s.wal = w

	// a newly created log file only survives a crash once the folder entry is on disk too
	if err := syncDir(dir); err != nil {
		err := w.Close()
		if err != nil {
			return nil, err
		}
		err = lock.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	if s.onChange != nil {
		s.onChange(s.state)
	}
	return s, nil
}

// recover replays the log into s.state and cuts off an incomplete record at the end.
func (s *Store) recover() error {
	path := filepath.Join(s.dir, walFileName)

	validEnd, err := readLog(path, func(c Command) error {
		if c.Seq != s.lastSeq+1 {
			return fmt.Errorf("log is out of order: expected seq %d, found %d", s.lastSeq+1, c.Seq)
		}
		if err := s.state.Validate(c); err != nil {
			return fmt.Errorf("log record %d is not valid: %w", c.Seq, err)
		}
		s.state.Apply(c)
		s.lastSeq = c.Seq
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if dropped := info.Size() - validEnd; dropped > 0 {
		log.Printf("store: cutting %d bytes of incomplete record from the end of %s", dropped, path)
		if err := os.Truncate(path, validEnd); err != nil {
			return err
		}
	}
	return nil
}

// Do runs one command: check it, write it to the log, then apply it to memory.
// If it returns nil, the command is on disk.
func (s *Store) Do(cmd Command) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failed != nil {
		return fmt.Errorf("store cannot accept changes: %w", s.failed)
	}

	// validate command, changing nothing
	if err := s.state.Validate(cmd); err != nil {
		return err
	}

	// write to disk
	cmd.Seq = s.lastSeq + 1
	if err := s.wal.Append(cmd); err != nil {
		// after a failed write we can't trust what is on disk: stop accepting changes
		s.failed = err
		return err
	}

	// change memory (cannot fail: Validate already passed)
	s.state.Apply(cmd)
	s.lastSeq = cmd.Seq

	if s.onChange != nil {
		s.onChange(s.state)
	}
	return nil
}

// Close closes the log and releases the lock on the data folder.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if errors.Is(s.failed, errClosed) {
		return nil
	}
	s.failed = errClosed

	err := s.wal.Close()
	if e := s.lock.Close(); err == nil { // ...then release the lock last
		err = e
	}
	return err
}

// syncDir forces the folder's own entries (file names) onto the disk.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// lockDir takes an exclusive lock on the data folder, so that only one
// process can use it at a time. It creates the LOCK file if it doesn't exist
// and asks the operating system for a lock on it (flock).
func lockDir(dir string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(dir, "LOCK"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("data dir %s is in use by another process", dir)
		}
		return nil, err
	}
	return f, nil
}
