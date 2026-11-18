package store

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
)

// One record on disk:
//
//	| crc (4 bytes) | length (4 bytes) | body (length bytes) |
//
// The body is what encodeCommand returns. The crc is calculated over the
// length field plus the body, so damage to either one is detected.
const (
	headerSize    = 8
	minBodySize   = 9       // seq (8 bytes) + op (1 byte)
	maxRecordSize = 1 << 20 // 1 MiB: refuse bigger lengths before allocating memory
)

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

type WAL struct {
	f *os.File
}

// openWAL opens the log file for appending, creating it if it doesn't exist.
func openWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &WAL{f: f}, nil
}

// Append writes one command as a record and forces it onto the disk.
// When it returns nil, the record survives a crash or power loss.
func (w *WAL) Append(c Command) error {
	body, err := encodeCommand(c)
	if err != nil {
		return err
	}
	if len(body) > maxRecordSize {
		return errors.New("record too large")
	}

	// build the whole record in one buffer, so it goes out in a single Write
	buf := make([]byte, headerSize+len(body))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(body)))
	copy(buf[headerSize:], body)
	binary.LittleEndian.PutUint32(buf[0:4], crc32.Checksum(buf[4:], castagnoli))

	if _, err := w.f.Write(buf); err != nil {
		return err
	}
	return w.f.Sync()
}

// Close closes the log file.
func (w *WAL) Close() error {
	return w.f.Close()
}

// readLog reads the log from the start and calls fn for each command, in order.
//
// It stops quietly at the first record that is incomplete or damaged, and
// returns validEnd: the byte offset just after the last good record. The caller
// decides what to do with the bytes after it (the Store cuts them off).
//
// It returns an error only for real problems: the file can't be opened or read,
// or fn itself returns an error. A missing file rwturns an error that satisfies
// errors.Is(err, os.ErrNotExist).
func readLog(path string, fn func(Command) error) (validEnd int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var hdr [headerSize]byte

	for {
		// the header
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			if isShortRead(err) {
				return validEnd, nil // clean end of file, or a half-written header
			}
			return validEnd, err
		}
		wantCRC := binary.LittleEndian.Uint32(hdr[0:4])
		length := binary.LittleEndian.Uint32(hdr[4:8])
		if length < minBodySize || length > maxRecordSize {
			return validEnd, nil // the length itself is damaged
		}

		// the body
		body := make([]byte, length)
		if _, err := io.ReadFull(r, body); err != nil {
			if isShortRead(err) {
				return validEnd, nil // the file ends in the middle of this record
			}
			return validEnd, err
		}

		// 3. the checksum
		sum := crc32.New(castagnoli)
		sum.Write(hdr[4:8])
		sum.Write(body)
		if sum.Sum32() != wantCRC {
			return validEnd, nil // bytes don't match what was written
		}

		// the command
		cmd, err := decodeCommand(body)
		if err != nil {
			return validEnd, nil
		}
		if err := fn(cmd); err != nil {
			return validEnd, err
		}
		validEnd += int64(headerSize) + int64(length)
	}
}

// isShortRead reports whether err just means "the file ended before we got
// all the bytes we asked for".
func isShortRead(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
