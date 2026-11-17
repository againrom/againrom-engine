package modrt

import (
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/mod"
)

func TestSecondSpellTablesFixtureCarriesItsTablesOnTheSecondGameOnly(t *testing.T) {
	entry := mod.Load(filepath.Join("testdata", "mods", "second-spell-tables"))
	if entry.Err != nil {
		t.Fatal(entry.Err)
	}
	for _, base := range []string{"rom2-en", "rom2-ru"} {
		res, err := Load([]mod.Entry{entry}, base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		rows := res.Spells.Rows
		if len(rows) != 2 || rows[0].Target != "Ice_Missile" || !slices.Equal(rows[0].DamageFactor.Vals, []int32{30, 45, 90}) ||
			rows[1].Target != "Invisibility" || !slices.Equal(rows[1].DurationFactor.Vals, []int32{1000, 2000, 5000}) {
			t.Errorf("%s rows %+v", base, rows)
		}
	}
	if _, err := Load([]mod.Entry{entry}, "rom1-en", nil, Options{}); err == nil {
		t.Error("the first game took a mod that applies to the second")
	}
}
