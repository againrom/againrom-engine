package archtest

import "testing"

// TestLiveSpellDeletionsMatchTheirBaseline measures the real tree.
func TestLiveSpellDeletionsMatchTheirBaseline(t *testing.T) {
	files := liveProductionSources(t)
	for _, name := range []string{WornChangeFile, WornTownHolderFile} {
		if _, ok := files[name]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", name)
		}
	}
	got, vs := SpellDeletionSites(files)
	for _, v := range vs {
		t.Error(v)
	}
	for _, v := range CheckSpellDeletionSites(got, CommittedSpellDeletionSites) {
		t.Errorf("%s (measured %v)", v, got)
	}
}

// TestSpellDeletionSitesAreFound drives the scan with synthetic sources: each
// deletion shape counts outside the transaction, a Spell write that builds one
// and the transaction's own files do not.
func TestSpellDeletionSitesAreFound(t *testing.T) {
	files := map[string]string{
		"pkg/game/a.go":      "package game\nfunc f(item *sim.ItemInstance) { item.SourceEquipment.Spell = sim.SourceItemSpell{} }\n",
		"pkg/game/b.go":      "package game\nfunc f(g *topology) { g.Items[0].Spell = 0 }\n",
		"pkg/sim/c.go":       "package sim\nfunc (w *World) f(item *ItemInstance) bool { return w.replaceObjectSpell(item, SourceItemSpell{}) }\n",
		"pkg/sim/d.go":       "package sim\nfunc f(item *ItemInstance, s SourceItemSpell) { item.SourceEquipment.Spell = s; item.SourceEquipment.Spell = SourceItemSpell{Present: true} }\n",
		WornChangeFile:       "package sim\nfunc f(item *ItemInstance) { item.SourceEquipment.Spell = SourceItemSpell{} }\n",
		WornTownHolderFile:   "package game\nfunc f(item *sim.ItemInstance) { item.SourceEquipment.Spell = sim.SourceItemSpell{} }\n",
		"pkg/game/e_test.go": "package game\nfunc f(item *sim.ItemInstance) { item.SourceEquipment.Spell = sim.SourceItemSpell{} }\n",
	}
	got, vs := SpellDeletionSites(files)
	if len(vs) != 0 {
		t.Fatal(vs)
	}
	want := map[string]int{"pkg/game/a.go": 1, "pkg/game/b.go": 1, "pkg/sim/c.go": 1}
	if len(got) != len(want) {
		t.Fatalf("SpellDeletionSites = %v, want %v", got, want)
	}
	for name, n := range want {
		if got[name] != n {
			t.Fatalf("SpellDeletionSites = %v, want %v", got, want)
		}
	}
	if vs := CheckSpellDeletionSites(got, map[string]int{"pkg/game/a.go": 1, "pkg/game/b.go": 1, "pkg/sim/c.go": 1}); len(vs) != 0 {
		t.Fatalf("the committed counts refused: %v", vs)
	}
	if vs := CheckSpellDeletionSites(got, map[string]int{"pkg/game/a.go": 1, "pkg/game/b.go": 1}); len(vs) != 1 {
		t.Fatalf("a new deletion file: %v, want one violation", vs)
	}
	if vs := CheckSpellDeletionSites(got, map[string]int{"pkg/game/a.go": 1, "pkg/game/b.go": 1, "pkg/sim/c.go": 2}); len(vs) != 1 {
		t.Fatalf("a fallen count: %v, want one violation", vs)
	}
}
