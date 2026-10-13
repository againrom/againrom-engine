package alm

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestPlayerNameIsBoundedLegacyText(t *testing.T) {
	for _, name := range []string{"Стража", strings.Repeat("Я", 32)} {
		encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(name))
		if err != nil {
			t.Fatal(err)
		}
		rec := make([]byte, 76)
		copy(rec[12:44], encoded)
		copy(rec[44:], bytes.Repeat([]byte{1}, 32))
		m := &Map{Meta: Meta{Count5: 1}}
		if err := m.decodeGroups(rec); err != nil {
			t.Fatal(err)
		}
		if got := m.Groups[0].Name; got != string(encoded) {
			t.Fatalf("name bytes = %x, want %x", got, encoded)
		}
		if got := DecodeText([]byte(m.Groups[0].Name)); got != name {
			t.Fatalf("display name = %q, want %q", got, name)
		}
	}
}
