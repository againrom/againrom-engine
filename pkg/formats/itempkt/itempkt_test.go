package itempkt

import (
	"encoding/binary"
	"testing"
)

func packetRecord(id uint16, flag byte, attrs ...byte) []byte {
	b := make([]byte, 7)
	binary.LittleEndian.PutUint16(b, id)
	b[4], b[6] = flag, byte(len(attrs))
	return append(b, attrs...)
}

func TestPacketRecordsAndLastPrice(t *testing.T) {
	b := make([]byte, 9)
	b = append(b, packetRecord(0x1201, 2, 1, 43, 0, 0, 0, 99, 7, 1, 44, 0, 0, 0)...)
	b = append(b, packetRecord(0x1601, 4, 5, 8)...)
	got, err := Parse(b)
	if err != nil || len(got) != 2 || got[0x1201] != (Record{Flags: 2, Price: 44, HasPrice: true}) || got[0x1601].HasPrice {
		t.Fatalf("records %v error %v", got, err)
	}
}

func TestPacketRejectsTruncation(t *testing.T) {
	valid := append(make([]byte, 9), packetRecord(0x1201, 2, 1, 44, 0, 0, 0)...)
	for n := 0; n < len(valid); n++ {
		if n == 9 {
			continue
		}
		if _, err := Parse(valid[:n]); err == nil {
			t.Errorf("accepted %d bytes", n)
		}
	}
	bad := append(make([]byte, 9), packetRecord(1, 2, 1, 3)...)
	if _, err := Parse(bad); err == nil {
		t.Fatal("accepted a truncated price attribute")
	}
}
