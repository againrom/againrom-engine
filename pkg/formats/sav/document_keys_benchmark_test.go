package sav

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func BenchmarkReserveDocumentKeys(b *testing.B) {
	doc := DocumentData{Objects: []DocumentRecordData{{Raw: []DocumentRawData{{Name: "U158", Bytes: make([]byte, 512)}}}}}
	for i := 0; i < 128; i++ {
		binary.LittleEndian.PutUint32(doc.Objects[0].Raw[0].Bytes[i*4:], 0x01000000+uint32(3*i))
	}
	for _, count := range []int{8, 65536} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				keys, err := ReserveDocumentKeys(doc, count)
				if err != nil || len(keys) != count {
					b.Fatal(len(keys), err)
				}
			}
		})
	}
}

func TestDocumentKeyReservationRetainsAscendingSequence(t *testing.T) {
	doc := DocumentData{Objects: []DocumentRecordData{{
		Values: []DocumentValueData{{Name: "Reference", Value: 0x01000000}},
		Raw:    []DocumentRawData{{Name: "opaque", Bytes: []byte{0x77, 3, 0, 0, 1}}},
		Groups: []DocumentRecordData{{Values: []DocumentValueData{{Name: "Reference", Value: 0x01000002}}}},
	}}, World: &DocumentWorldData{TerrainIdentity: 0x01000005, Cells: []DocumentCellData{{GroundActor: 0x01000006}}}}
	keys, err := ReserveDocumentKeys(doc, 65536)
	if err != nil {
		t.Fatal(err)
	}
	excluded := map[uint32]bool{0x01000000: true, 0x01000002: true, 0x01000003: true, 0x01000005: true, 0x01000006: true}
	next := uint32(0x01000000)
	for i, key := range keys {
		for excluded[next] {
			next++
		}
		if key != next {
			t.Fatalf("key %d = %#x, want %#x", i, key, next)
		}
		next++
	}
	short, err := ReserveDocumentKeys(doc, 8)
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range short {
		if key != keys[i] {
			t.Fatal("reservation prefix changed")
		}
	}
}
