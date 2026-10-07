package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseOriginalResaveReplacesMapWeaponSpell(t *testing.T) {
	f := releaseFront(t)
	path, _ := groundCorpusFile(t, "2026-09-09/game0076.sav", "ca6f2980986859fb19c7b602a00b92b0e3ae95b1d00adc6c757152d596ed556c")
	f.SetDeterministicFrames(true)
	app := f.App("original mission resave")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	if f.live.mission.number != 111 || len(f.live.mission.ids) != 5 {
		t.Fatalf("mission/party: %d/%d", f.live.mission.number, len(f.live.mission.ids))
	}
	actors, found := 0, false
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class != 0 {
			actors++
		}
		if e.SourceBinding.ArchiveIndex != 106 {
			continue
		}
		found = true
		weapon, ok := f.live.world.EquippedItems(e.ID)
		if !ok || e.TypeID != 27 || e.WeaponSpell != 2 || e.WeaponSpellLevel != 40 || e.WeaponSpellSource != sim.WeaponSpellItem || weapon[0].Code != 0xf11a {
			t.Fatalf("saved NPC weapon: actor=%d type=%d spell=%d/%d/%d item=%+v", e.ID, e.TypeID, e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource, weapon[0])
		}
	}
	if !found || actors != 85 {
		t.Fatalf("restored actors=%d found NPC=%t", actors, found)
	}
	// Take an ordinary AGS checkpoint before the first tick, then resume it
	// through a fresh front end and compare both live continuations.
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	for range 128 {
		f.LiveAdvance(1)
		fresh.LiveAdvance(1)
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("original/AGS continuation diverged")
		}
	}
	t.Logf("M111 original LOAD: 85 actors, 5 party; NPC weapon 2/40; AGS reload and 128 ticks matched")
}
