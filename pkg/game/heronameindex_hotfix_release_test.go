package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseHeroNameIndexFollowsCurrentEquipmentThroughColdLoad(t *testing.T) {
	// dropped is the name index after the first dropped item: game0076's
	// Danath keeps his shield when his sword is dropped (DIV-1537), so he
	// stands as the unarmed fighter with a shield.
	cases := []struct {
		path, hash     string
		id             sim.EntityID
		index, dropped int
	}{
		{"2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b", 46, 5, 1},
		{"2026-08-30/EXP-0278-human-runtime-en/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b", 46, 5, 1},
		{"2026-09-09/game0076.sav", "ca6f2980986859fb19c7b602a00b92b0e3ae95b1d00adc6c757152d596ed556c", 81, 4, 2},
	}
	root := os.Getenv("AGAINROM_ASSETS")
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			_, raw := groundCorpusFile(t, tc.path, tc.hash)
			open := func(front *FrontEnd, bytes []byte) {
				t.Helper()
				mission, town, err := front.RestoreOriginal(bytes)
				if err != nil || town {
					t.Fatal("restore", town, err)
				}
				app := front.App("hero unit name")
				app.Layout(1024, 768)
				if err := app.OpenMission(mission); err != nil {
					t.Fatal("open mission", err)
				}
			}
			find := func(front *FrontEnd) ui.MapEntity {
				t.Helper()
				for _, draw := range front.live.entityDraws() {
					if draw.ID == uint32(tc.id) {
						return draw
					}
				}
				t.Fatalf("entity %d has no visible unit", tc.id)
				return ui.MapEntity{}
			}
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			open(f, raw)
			initial := find(f)
			initialEntity, _ := f.live.world.Entity(tc.id)
			if initial.Name == "" || initial.Name != f.live.actorNames[tc.id] || initial.UnitNameIndex != tc.index || initialEntity.Class != int32(tc.index) {
				t.Fatalf("initial unit %d name=%q explicit=%q index=%d class=%d, want index/class %d", tc.id, initial.Name, f.live.actorNames[tc.id], initial.UnitNameIndex, initialEntity.Class, tc.index)
			}
			f.LiveAdvance(30)
			hero := f.live.mission.ids[0]
			e, _ := f.live.entity(hero)
			at := sim.CellPoint{X: e.X, Y: e.Y}
			tries := []sim.Command{sim.DropCarried(hero, 0, at)}
			for slot := sim.EquipSlot(1); slot <= 12; slot++ {
				tries = append(tries, sim.DropWorn(hero, slot, at))
			}
			beforeSacks := len(f.live.world.Sacks())
			for _, cmd := range tries {
				f.live.pending = append(f.live.pending, cmd)
				f.LiveAdvance(1)
				if len(f.live.world.Sacks()) != beforeSacks {
					break
				}
			}
			afterDrop := find(f)
			afterDropEntity, _ := f.live.world.Entity(tc.id)
			if afterDrop.UnitNameIndex != tc.dropped || afterDrop.Name != initial.Name || afterDrop.Art == initial.Art || afterDropEntity.Class != int32(tc.index) {
				t.Fatalf("after drop unit %d name=%q index=%d class=%d body=%q, initial body=%q", tc.id, afterDrop.Name, afterDrop.UnitNameIndex, afterDropEntity.Class, afterDrop.Art.Name, initial.Art.Name)
			}
			party := map[sim.EntityID]bool{}
			for _, id := range f.live.mission.ids {
				party[id] = true
			}
			killed := 0
			for _, e := range f.live.world.Entities() {
				if killed == 5 {
					break
				}
				if party[e.ID] || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone {
					continue
				}
				f.LiveKill(uint32(e.ID))
				killed++
			}
			f.LiveAdvance(2400)
			before := find(f)
			actor, _ := f.live.world.Entity(tc.id)
			if before.UnitNameIndex != tc.dropped || before.Name != initial.Name || before.Art != afterDrop.Art || actor.HP <= 0 || actor.Class != int32(tc.index) {
				t.Fatalf("pre-SAVE unit %d name=%q index=%d HP=%d class=%d body=%q", tc.id, before.Name, before.UnitNameIndex, actor.HP, actor.Class, before.Art.Name)
			}
			dir := t.TempDir()
			seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
			prepared, err := seams.Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: "hero-name", Format: ui.SaveSAV})
			if err != nil {
				t.Fatal("prepare SAVE", err)
			}
			if _, err := prepared.Commit(true); err != nil {
				t.Fatal("commit SAVE", err)
			}
			written, err := ReadSaveFile(filepath.Join(dir, "hero-name.sav"))
			if err != nil {
				t.Fatal("read SAVE", err)
			}
			g, err := NewFrontEnd(root)
			if err != nil {
				t.Fatal("cold front end", err)
			}
			g.SetDeterministicFrames(true)
			open(g, written)
			check := func(stage string) {
				t.Helper()
				got := find(g)
				loaded, _ := g.live.world.Entity(tc.id)
				if got.Name != before.Name || got.Name != g.live.actorNames[tc.id] || got.UnitNameIndex != before.UnitNameIndex || got.Art.Name != before.Art.Name || loaded.Class != actor.Class {
					t.Fatalf("%s unit %d name=%q/%q index=%d/%d body=%q/%q class=%d/%d", stage, tc.id, got.Name, before.Name, got.UnitNameIndex, before.UnitNameIndex, got.Art.Name, before.Art.Name, loaded.Class, actor.Class)
				}
			}
			check("cold LOAD")
			f.LiveAdvance(120)
			g.LiveAdvance(120)
			check("next ticks")
			continued := find(f)
			if continued.Name != find(g).Name || continued.UnitNameIndex != find(g).UnitNameIndex {
				t.Fatalf("continued copies disagree: source name=%q index=%d, loaded name=%q index=%d", continued.Name, continued.UnitNameIndex, find(g).Name, find(g).UnitNameIndex)
			}
		})
	}
}
