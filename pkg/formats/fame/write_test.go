package fame

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

func TestMarshalNormalizesReadSpansAndPreservesThreeWords(t *testing.T) {
	rows := []Record{{Name: string([]byte{0x80, 'A'}), NameSpan: []byte{0x80, 'A', 0, 'x', 0}, Score: -17, Tail: [2]uint32{0x12345678, 0xffffffff}}}
	written, err := Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 0, 0, 0, 3, 0, 0, 0, 0x80, 'A', 0}
	for _, v := range []uint32{0xffffffef, 0x12345678, 0xffffffff} {
		want = binary.LittleEndian.AppendUint32(want, v)
	}
	if !bytes.Equal(written, want) {
		t.Fatalf("writer bytes %x, want %x", written, want)
	}
	got, err := Parse(written)
	if err != nil || len(got) != 1 || got[0].Name != rows[0].Name || got[0].Score != -17 || got[0].Tail != rows[0].Tail {
		t.Fatal(got, err)
	}
}

func TestInsertionUsesSignedFirstMatchTiesAndStoredLimit(t *testing.T) {
	old := []Record{{Name: "same", Score: 90, Tail: [2]uint32{1, 2}}, {Name: "negative", Score: -7, Tail: [2]uint32{3, 4}}, {Name: "unsorted", Score: 100, Tail: [2]uint32{5, 6}}}
	for _, tc := range []struct {
		name  string
		score int32
		limit int
		want  []string
	}{
		{"tie", 90, 10, []string{"new", "same", "negative", "unsorted"}},
		{"first match", 50, 10, []string{"same", "new", "negative", "unsorted"}},
		{"signed low trimmed", math.MinInt32, 3, []string{"same", "negative", "unsorted"}},
		{"signed high", math.MaxInt32, 3, []string{"new", "same", "negative"}},
		{"zero limit", 90, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Insert(old, Record{Name: "new", Score: tc.score}, tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, r := range got {
				names = append(names, r.Name)
				for _, source := range old {
					if r.Name == source.Name && r.Tail != source.Tail {
						t.Fatal("lost carried words", r)
					}
				}
			}
			if !reflect.DeepEqual(names, tc.want) {
				t.Fatal(names, tc.want)
			}
		})
	}
	got, err := Insert(old, Record{Name: "same", Score: 90}, 10)
	if err != nil || len(got) != 4 || got[0].Name != "same" || got[1].Name != "same" {
		t.Fatal("deduplicated name", got, err)
	}
}

func TestWriterAndInsertionBounds(t *testing.T) {
	for _, name := range []string{"a\x00b", string(make([]byte, 1024))} {
		if _, err := Marshal([]Record{{Name: name}}); err == nil {
			t.Fatal("accepted invalid name")
		}
	}
	if _, err := Marshal(make([]Record, 4097)); err == nil {
		t.Fatal("accepted oversized table")
	}
	if _, err := Insert(nil, Record{}, -1); err == nil {
		t.Fatal("accepted negative limit")
	}
}
