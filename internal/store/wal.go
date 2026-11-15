package store

import (
	"hash/crc32"
	"os"
)

const maxRecordSize = 1 << 20 // 1 MiB: refuse bigger records before allocating

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

type WAL struct {
	f *os.File
}

// openWAL opens (or creates) the log file, positioned to append at the end.
func openWAL(path string) (*WAL, error) {
	// os.OpenFile with O_CREATE | O_RDWR | O_APPEND, permission 0o644

	return nil, nil
}

// Append writes one command as a record and forces it onto the disk.
func (w *WAL) Append(c Command) error {
	// 1. body, err := encodeCommand(c)
	// 2. build ONE buffer: 8-byte header + body
	//      - length = len(body), little endian
	//      - crc over (length bytes + body)
	// 3. w.f.Write(buffer) in a single call
	// 4. w.f.Sync()  <- this is what makes it durable

	return nil
}

func (w *WAL) Close() error {
	return nil
}

// readLog reads records from the start of the file and calls fn for each
// command, in order. It stops at the first record that is incomplete or damaged.
// It returns validEnd: the byte offset just after the last good record, so the
// caller can cut the file there.
func readLog(path string, fn func(Command) error) (validEnd int64, err error) {
	// loop:
	//   read 8 header bytes (io.ReadFull)
	//     - 0 bytes read and io.EOF  -> clean end, stop, no problem
	//     - fewer than 8 bytes       -> torn write, stop
	//   length > maxRecordSize or length < 9 -> damaged, stop
	//   read `length` body bytes     -> fewer than expected: torn write, stop
	//   recompute the crc, compare   -> different: damaged, stop
	//   decodeCommand(body)          -> error: damaged, stop
	//   call fn(cmd), then validEnd += 8 + length

	return 0, err
}
