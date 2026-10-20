package game

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"againrom/pkg/mapedit"
)

// The unit popup's override block: an empty field writes the stored
// "inherit" value, Health 0 writes 0, a typed 0 in a byte field is refused
// with its reason and writes nothing, and the Inherit button clears an
// override back to the bytes before. Each write equals SetField's bytes, the
// one write path the popup had before the block was regrouped.
func TestEditorUnitBlockWritesInheritAndExplicitZero(t *testing.T) {
	raw, e := openAuthoring(t)
	rec := recordIndex(t, e, "Unit", 0)
	e.Select(rec, true)
	typer := overrideTyper{t: t, c: &editorClicker{t: t, now: time.Unix(1, 0)}, e: e, i: 0}
	witness := func(before []byte, key string, v int64) []byte {
		t.Helper()
		m, err := mapedit.New(before)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.SetField(mapedit.KindUnit, 0, key, v); err != nil {
			t.Fatal(err)
		}
		return m.Bytes()
	}
	value := func(key string) int64 {
		t.Helper()
		fields := objectEdits(t, e).Fields("Unit", 0)
		return fields[fieldIndex(t, fields, key)].Value
	}

	for _, c := range []struct {
		key  string
		text string
		want int64
	}{{"hp", "0", 0}, {"hp", "", -1}, {"mana", "0", 0}, {"mana", "", -1}, {"body", "5", 5}, {"body", "", 0}, {"protection.fire", "9", 9}} {
		before := e.Document().SourceBytes
		typer.text(c.key, c.text)
		e.Select(rec, true)
		if got := value(c.key); got != c.want || !bytes.Equal(e.Document().SourceBytes, witness(before, c.key, c.want)) {
			t.Fatalf("%s %q: value %d, want %d with SetField's bytes", c.key, c.text, got, c.want)
		}
	}
	if bytes.Equal(witness(raw, "hp", 0), raw) {
		t.Fatal("loss control: Health 0 writes the bytes of inherit")
	}

	typer.clear("protection.fire")
	e.Select(rec, true)
	if value("protection.fire") != 0 || !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("Inherit did not clear the protection back to the bytes before")
	}

	typer.text("body", "0")
	note := "Body: 0 cannot be set"
	if !bytes.Equal(e.Document().SourceBytes, raw) || !strings.Contains(popupText(e), note) {
		t.Fatalf("Body 0 was not refused with %q: %s", note, popupText(e))
	}
	typer.clear("body")
	if !bytes.Equal(e.Document().SourceBytes, raw) || strings.Contains(popupText(e), note) {
		t.Fatal("Inherit did not take back the refused 0")
	}
}
