package fame

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestReadPreservesSignedStoredOrderAndNameSpans(t *testing.T) {
	b := []byte{3, 0, 0, 0}
	for _, score := range []int32{-7, 90, 90} {
		b = binary.LittleEndian.AppendUint32(b, 4)
		b = append(b, 'A', 0, 'z', 0)
		b = binary.LittleEndian.AppendUint32(b, uint32(score))
		b = binary.LittleEndian.AppendUint32(b, 12)
		b = binary.LittleEndian.AppendUint32(b, 0xffffffff)
	}
	b = append(b, 99)
	rows, err := Parse(b)
	if err != nil || len(rows) != 3 {
		t.Fatal(rows, err)
	}
	for i, want := range []int32{-7, 90, 90} {
		if rows[i].Score != want || rows[i].Name != "A" || !reflect.DeepEqual(rows[i].NameSpan, []byte{'A', 0, 'z', 0}) || rows[i].Tail != [2]uint32{12, 0xffffffff} {
			t.Fatal(rows)
		}
	}
	if rows, err := Parse([]byte{0, 0, 0, 0}); err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}

func TestReadRejectsUnboundedAndTruncatedFields(t *testing.T) {
	valid := []byte{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for n := 0; n < len(valid); n++ {
		if _, err := Parse(valid[:n]); err == nil {
			t.Fatalf("accepted length %d", n)
		}
	}
	for _, at := range []int{0, 4} {
		b := append([]byte(nil), valid...)
		binary.LittleEndian.PutUint32(b[at:], 0xffffffff)
		if _, err := Parse(b); err == nil {
			t.Fatal("unchecked field", at)
		}
	}
	valid[8] = 'x'
	if _, err := Parse(valid); err == nil {
		t.Fatal("accepted missing NUL")
	}
}
