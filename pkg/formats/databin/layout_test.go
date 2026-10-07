package databin_test

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/databin"
)

var wideTail = []byte{0xee, 0xee, 0xee, 0xee}

func wideStream() []byte {
	in := goodStream()
	for _, raw := range [][]byte{armorRaw, shieldRaw} {
		at := bytes.Index(in, raw) + len(raw)
		in = append(in[:at:at], append(append([]byte(nil), wideTail...), in[at:]...)...)
	}
	return in
}

func TestParseWithWideArmorBlock(t *testing.T) {
	in := wideStream()
	f, err := databin.ParseWith(in, databin.ROM2Layout)
	if err != nil {
		t.Fatal(err)
	}
	if f.Consumed != len(in) {
		t.Fatalf("consumed %d of %d", f.Consumed, len(in))
	}
	armors := f.Collection(databin.Armors)
	want := append(append([]byte(nil), armorRaw...), wideTail...)
	if got := armors.EntryRaw(1); !bytes.Equal(got, want) {
		t.Fatalf("Armors[1] raw = % x, want % x", got, want)
	}
	if got := len(f.Collection(databin.Shields).EntryRaw(1)); got != 14 {
		t.Fatalf("Shields[1] raw is %d bytes, want 14", got)
	}
	if got := f.Collection(databin.Units).EntryName(1); got != "Un1" {
		t.Fatalf("a later group read %q after the wide block", got)
	}
}

func TestEachLayoutRefusesTheOthersFile(t *testing.T) {
	if _, err := databin.Parse(wideStream()); err == nil {
		t.Fatal("the narrow layout walked a wide-block file")
	}
	if _, err := databin.ParseWith(goodStream(), databin.ROM2Layout); err == nil {
		t.Fatal("the wide layout walked a narrow-block file")
	}
	if _, err := databin.ParseWith(goodStream(), databin.ROM1Layout); err != nil {
		t.Fatalf("the narrow layout refused its own file: %v", err)
	}
}
