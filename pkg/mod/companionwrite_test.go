package mod

import (
	"strings"
	"testing"
)

func TestCompanionsWriterRoundTripsRowsThroughItsReader(t *testing.T) {
	src := "# keep\n[[join]]\nkey       = \"plagat_mage\"\ncompanion = 22\nchapter   = 30\nbuilding  = \"tavern\"\ntalk      = 22\n"
	var d Data
	if err := readCompanions(DataReader{Mod: "m"}, &d, CompanionsFile, []byte(src)); err != nil {
		t.Fatal(err)
	}
	if same, err := writeCompanions(CompanionsFile, []byte(src), d); err != nil || string(same) != src {
		t.Fatalf("an unchanged row is rewritten: %q, %v", same, err)
	}
	d.Companions.Joins[0].Talk = 23
	d.Companions.Joins = append(d.Companions.Joins, CompanionJoin{Key: "second", Companion: 23, Chapter: 31, Building: BuildingTavern, Talk: 5})
	out, err := writeCompanions(CompanionsFile, []byte(src), d)
	if err != nil || !strings.HasPrefix(string(out), "# keep\n[[join]]\nkey       = \"plagat_mage\"") {
		t.Fatalf("%q, %v", out, err)
	}
	back, err := ParseCompanions("m", CompanionsFile, out)
	if err != nil || len(back.Joins) != 2 || back.Joins[0].Talk != 23 || back.Joins[1].Companion != 23 || back.Joins[1].Chapter != 31 || back.Joins[1].Talk != 5 {
		t.Fatalf("read back %+v, %v", back, err)
	}
	d.Companions.Joins = nil
	out, err = writeCompanions(CompanionsFile, out, d)
	if err != nil || strings.Contains(string(out), "[[join]]") {
		t.Fatalf("emptied: %q, %v", out, err)
	}
	if got, err := writeCompanions(CompanionsFile, nil, Data{}); err != nil || len(got) != 0 {
		t.Fatalf("an empty mod writes %q, %v", got, err)
	}
}
