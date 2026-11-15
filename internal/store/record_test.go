package store

import "testing"

func sampleCommands() []Command {
	return []Command{
		{Op: OpPoolCreate, Pool: "webapp-api", Strategy: "smooth_weighted_round_robin"},
		{Op: OpBackendAdd, Pool: "webapp-api", Backend: "api-1", Address: "10.0.0.1:3000", Weight: 5},
		{Op: OpWeightSet, Pool: "webapp-api", Backend: "api-1", Weight: 3},
		{Op: OpBackendRemove, Pool: "webapp-api", Backend: "api-1"},
		{Op: OpStrategySet, Pool: "webapp-api", Strategy: "round_robin"},
		{Op: OpRouteAdd, Host: "app.example.com", PathPrefix: "/api/", Pool: "webapp-api"},
		{Op: OpRouteRemove, Host: "app.example.com", PathPrefix: "/api/"},
		{Op: OpPoolDelete, Pool: "webapp-api"},
	}
}

func TestRoundTrip(t *testing.T) {
	for i, c := range sampleCommands() {
		c.Seq = uint64(i + 1)
		b, err := encodeCommand(c)
		if err != nil {
			t.Fatalf("encode %d: %v", i, err)
		}
		got, err := decodeCommand(b)
		if err != nil {
			t.Fatalf("decode %d: %v", i, err)
		}
		if got != c {
			t.Fatalf("round trip %d: got %+v, want %+v", i, got, c)
		}
	}
}

func TestDecodeTruncated(t *testing.T) {
	for i, c := range sampleCommands() {
		c.Seq = uint64(i + 1)
		b, _ := encodeCommand(c)
		for n := 0; n < len(b); n++ { // every shorter length must fail
			if _, err := decodeCommand(b[:n]); err == nil {
				t.Fatalf("command %d cut to %d bytes should fail", i, n)
			}
		}
	}
}

func TestDecodeUnknownOp(t *testing.T) {
	b := make([]byte, 9) // seq = 0, op = 0
	if _, err := decodeCommand(b); err == nil {
		t.Fatal("op 0 should fail")
	}
	b[8] = 99
	if _, err := decodeCommand(b); err == nil {
		t.Fatal("op 99 should fail")
	}
}
