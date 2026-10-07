package video

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeHeaderRejectsUnsafeExtentsAndTiming(t *testing.T) {
	for _, tc := range []struct {
		w, h, n, ms uint32
		valid       bool
	}{
		{320, 200, 90, 67, true}, {4, 4, 1, uint32(0xffffffff - 9999), true},
		{0, 200, 90, 67, false}, {321, 200, 90, 67, false}, {2049, 4, 1, 1, false},
		{4, 4, 0, 10, false}, {4, 4, MaxFrames + 1, 10, false}, {4, 4, 1, 0, false},
		{4, 4, 1, 1001, false}, {4, 4, 100000, 1000, false},
	} {
		h := make([]byte, 104)
		copy(h, "SMK2")
		for n, v := range []uint32{tc.w, tc.h, tc.n, tc.ms} {
			binary.LittleEndian.PutUint32(h[4+n*4:], v)
		}
		path := filepath.Join(t.TempDir(), "authored.smk")
		if err := os.WriteFile(path, h, 0600); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := InspectNativeInput(f)
		f.Close()
		if (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
		if tc.valid && (got.Interval < time.Millisecond || got.Interval > time.Second) {
			t.Fatal(got)
		}
	}
}
