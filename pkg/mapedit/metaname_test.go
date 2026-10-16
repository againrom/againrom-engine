package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

func TestEncodedMapNameIsBoundedAndKeepsOtherBytes(t *testing.T) {
	doc, err := alm.NewMap(alm.DefaultNewMap())
	if err != nil {
		t.Fatal(err)
	}
	b := doc.Write()
	const at = 20 + 20 + 0x30
	for n, name := range [][]byte{[]byte{0xca, 0xe0, 0xf0, 0xf2, 0xe0}, bytes.Repeat([]byte{0xe0}, 63), []byte("short")} {
		e, err := mapedit.New(b)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.SetEncodedName(name); err != nil {
			t.Fatal(err)
		}
		want := bytes.Clone(b)
		clear(want[at : at+64])
		copy(want[at:], name)
		if !bytes.Equal(e.Bytes(), want) {
			t.Fatalf("name %d changed bytes outside its field", n)
		}
		out, err := e.SaveBytes()
		if err != nil || !bytes.Equal(out, want) {
			t.Fatalf("save name %d: %v", n, err)
		}
		if !e.Undo() || !bytes.Equal(e.Bytes(), b) || !e.Redo() || !bytes.Equal(e.Bytes(), want) {
			t.Fatal("name history lost a byte")
		}
	}
	for _, bad := range [][]byte{bytes.Repeat([]byte{0xe0}, 64), []byte("a\x00b"), {0x98}} {
		e, _ := mapedit.New(b)
		if err := e.SetEncodedName(bad); err == nil || e.CanUndo() || !bytes.Equal(e.Bytes(), b) {
			t.Fatalf("accepted invalid name %x", bad)
		}
	}
}
