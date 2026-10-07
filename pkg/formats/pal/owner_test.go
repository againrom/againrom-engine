package pal_test

import (
	"strings"
	"testing"

	"againrom/pkg/formats/pal"
)

func ownerStream() []byte {
	out := make([]byte, pal.OwnerSize)
	for table := 0; table < pal.OwnerTableCount; table++ {
		for i := 0; i < pal.EntryCount; i++ {
			e := out[table*pal.TableSize+i*pal.EntrySize:]
			e[0] = byte(table*11 + i)
			e[1] = byte(table*17 + 255 - i)
			e[2] = byte(table*23 + i*7)
			e[3] = byte(table*29 + i*13)
		}
	}
	return out
}

func TestDecodeOwnerTablesReadsEveryTableAndEntry(t *testing.T) {
	raw := ownerStream()
	got, err := pal.DecodeOwnerTables(raw)
	if err != nil {
		t.Fatalf("DecodeOwnerTables: %v", err)
	}
	for table := 0; table < pal.OwnerTableCount; table++ {
		for i := 0; i < pal.EntryCount; i++ {
			e := raw[table*pal.TableSize+i*pal.EntrySize:]
			want := pal.Color{R: e[2], G: e[1], B: e[0]}
			if got[table][i] != want {
				t.Fatalf("table %d entry %d = %+v, want %+v", table, i, got[table][i], want)
			}
		}
	}
}

func TestDecodeOwnerTablesRefusesEveryOtherLength(t *testing.T) {
	full := ownerStream()
	for n := 0; n < pal.OwnerSize; n++ {
		_, err := pal.DecodeOwnerTables(full[:n])
		if err == nil || !strings.Contains(err.Error(), "exactly") {
			t.Fatalf("%d bytes: error = %v, want exact-length refusal", n, err)
		}
	}
	for _, extra := range []int{1, pal.TableSize, pal.OwnerSize} {
		_, err := pal.DecodeOwnerTables(append(append([]byte(nil), full...), make([]byte, extra)...))
		if err == nil || !strings.Contains(err.Error(), "exactly") {
			t.Fatalf("+%d bytes: error = %v, want exact-length refusal", extra, err)
		}
	}
}

func TestOwnerGeometryConstants(t *testing.T) {
	if pal.OwnerTableCount != 16 || pal.OwnerSize != 0x4000 {
		t.Fatalf("owner tables = %d, size %#x; want 16 and 0x4000", pal.OwnerTableCount, pal.OwnerSize)
	}
}
