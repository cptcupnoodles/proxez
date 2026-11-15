package store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

var errShort = errors.New("record body too short")

// ---------- writing helpers ----------

func putString(b []byte, s string) []byte {
	b = binary.LittleEndian.AppendUint16(b, uint16(len(s)))
	return append(b, s...)
}

func putU32(b []byte, v uint32) []byte {
	return binary.LittleEndian.AppendUint32(b, v)
}

// ---------- encode ----------

func encodeCommand(c Command) ([]byte, error) {
	// the length field is only 2 bytes, so a string can't exceed 65535
	for _, s := range []string{
		c.Pool,
		c.Strategy,
		c.Backend,
		c.Address,
		c.Host,
		c.PathPrefix,
	} {
		if len(s) > math.MaxUint16 {
			return nil, errors.New("field too long")
		}
	}

	b := make([]byte, 0, 64)
	b = binary.LittleEndian.AppendUint64(b, c.Seq)
	b = append(b, byte(c.Op))

	switch c.Op {
	case OpPoolCreate:
		b = putString(b, c.Pool)
		b = putString(b, c.Strategy)
	case OpPoolDelete:
		b = putString(b, c.Pool)
	case OpStrategySet:
		b = putString(b, c.Pool)
		b = putString(b, c.Strategy)
	case OpBackendAdd:
		b = putString(b, c.Pool)
		b = putString(b, c.Backend)
		b = putString(b, c.Address)
		b = putU32(b, c.Weight)
	case OpBackendRemove:
		b = putString(b, c.Pool)
		b = putString(b, c.Backend)
	case OpWeightSet:
		b = putString(b, c.Pool)
		b = putString(b, c.Backend)
		b = putU32(b, c.Weight)
	case OpRouteAdd:
		b = putString(b, c.Host)
		b = putString(b, c.PathPrefix)
		b = putString(b, c.Pool)
	case OpRouteRemove:
		b = putString(b, c.Host)
		b = putString(b, c.PathPrefix)
	default:
		return nil, fmt.Errorf("unknown op %d", c.Op)
	}
	return b, nil
}

// ---------- reading helper ----------

// reader walks through the bytes from left to right.
// If anything goes wrong it remembers the FIRST error and returns
// zero values from then on, so check r.err once at the end
// instead of after every single read.
type reader struct {
	b   []byte
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if len(r.b) < n {
		r.err = errShort
		return nil
	}
	out := r.b[:n]
	r.b = r.b[n:]
	return out
}

func (r *reader) u8() uint8 {
	p := r.take(1)
	if r.err != nil {
		return 0
	}
	return p[0]
}

func (r *reader) u16() uint16 {
	p := r.take(2)
	if r.err != nil {
		return 0
	}
	return binary.LittleEndian.Uint16(p)
}

func (r *reader) u32() uint32 {
	p := r.take(4)
	if r.err != nil {
		return 0
	}
	return binary.LittleEndian.Uint32(p)
}

func (r *reader) u64() uint64 {
	p := r.take(8)
	if r.err != nil {
		return 0
	}
	return binary.LittleEndian.Uint64(p)
}

func (r *reader) str() string {
	n := r.u16()
	p := r.take(int(n))
	if r.err != nil {
		return ""
	}
	return string(p)
}

// ---------- decode ----------

func decodeCommand(body []byte) (Command, error) {
	r := &reader{b: body}
	c := Command{Seq: r.u64(), Op: Op(r.u8())}

	switch c.Op {
	case OpPoolCreate:
		c.Pool = r.str()
		c.Strategy = r.str()
	case OpPoolDelete:
		c.Pool = r.str()
	case OpStrategySet:
		c.Pool = r.str()
		c.Strategy = r.str()
	case OpBackendAdd:
		c.Pool = r.str()
		c.Backend = r.str()
		c.Address = r.str()
		c.Weight = r.u32()
	case OpBackendRemove:
		c.Pool = r.str()
		c.Backend = r.str()
	case OpWeightSet:
		c.Pool = r.str()
		c.Backend = r.str()
		c.Weight = r.u32()
	case OpRouteAdd:
		c.Host = r.str()
		c.PathPrefix = r.str()
		c.Pool = r.str()
	case OpRouteRemove:
		c.Host = r.str()
		c.PathPrefix = r.str()
	default:
		return Command{}, fmt.Errorf("unknown op %d", c.Op)
	}

	if r.err != nil {
		return Command{}, r.err
	}
	if len(r.b) != 0 {
		return Command{}, errors.New("extra bytes after command")
	}
	return c, nil
}
