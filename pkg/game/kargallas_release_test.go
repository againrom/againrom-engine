package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func kargallasInput(t *testing.T, key, digest string) []byte {
	t.Helper()
	path := os.Getenv(key)
	if path == "" {
		t.Skip(key + " required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != digest {
		t.Fatalf("%s SHA256=%s", key, got)
	}
	return raw
}

func kargallasLoad(t *testing.T, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("mission LOAD: town=%v err=%v", town, err)
	}
	a := f.App("Kargallas")
	a.Layout(1024, 768)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f, a
}

func TestReleaseKargallasDragonHasTheInstalledEmptyBook(t *testing.T) {
	f := releaseFront(t)
	rows := 0
	for i := 1; i < f.Table.Units.Len(); i++ {
		p := f.Table.Units.EntryParams(i)
		if len(p) < 54 || f.Table.Units.EntryName(i) != "Dragon" && f.Table.Units.EntryName(i) != "Dragon.2" && f.Table.Units.EntryName(i) != "Dragon.3" && f.Table.Units.EntryName(i) != "Dragon.4" {
			continue
		}
		rows++
		for _, v := range p[48:54] {
			if v != -1 {
				t.Fatalf("%s class spell cells=%v", f.Table.Units.EntryName(i), p[48:54])
			}
		}
	}
	if rows != 4 {
		t.Fatalf("Dragon rows=%d, want four", rows)
	}
	check := func(t *testing.T, f *FrontEnd, a *ui.App, interact bool) {
		t.Helper()
		var e sim.Entity
		found := 0
		for _, candidate := range f.live.world.Entities() {
			if candidate.MapUnitID == 187 {
				e = candidate
				found++
			}
		}
		if found != 1 {
			t.Fatalf("dragon records=%d", found)
		}
		id := e.ID
		if e.TypeID != 71 || e.Owner != 9 || e.Book.State != sim.BookPresent || e.KnownSpells != 0 || e.Book.Slots != ([28]sim.BookSpell{}) || e.CreatureSpells != ([sim.CreatureSpellSlots]sim.CreatureSpell{}) {
			t.Fatalf("dragon book/owner=%+v", e)
		}
		if interact {
			spellbookShow(t, a, f.live, id)
		}
		entries, fixed := selectedSpellbook(f.live.world.Rules(), []sim.Entity{e}, f.live.world.Spells(), f.live.spellNames, f.live.view.Words(), nil)
		if !fixed || len(entries) != 24 {
			t.Fatalf("book catalog=%d fixed=%v", len(entries), fixed)
		}
		for _, entry := range entries {
			if !entry.Unavailable {
				t.Fatalf("dragon knows spell %d", entry.ID)
			}
			if !interact {
				continue
			}
			x, y, err := a.HeadlessSpellPoint(entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			if _, current, armed := f.live.view.QuickSpellState(); current != 0 || armed {
				t.Fatalf("empty book arms %d (%v)", current, armed)
			}
		}
		if interact && len(f.live.pending) != 0 {
			t.Fatal("empty book issued an order")
		}
	}
	t.Run("fresh", func(t *testing.T) {
		a := f.App("Kargallas fresh")
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(90)); err != nil {
			t.Fatal(err)
		}
		check(t, f, a, true)
	})
	for _, c := range []struct{ key, hash string }{
		{"AGAINROM_KARGALLAS_SAV", "7298b24b9d72c3ac1e149b5b01f8dbbcb9fb1d44f3450621d2d62b41263d01f8"},
		{"AGAINROM_KARGALLAS_END_SAV", "b5cff799f5d5470bcc3d45442987121f41574754df60f875ed6a60967a6ae050"},
	} {
		t.Run(c.key, func(t *testing.T) {
			raw := kargallasInput(t, c.key, c.hash)
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			books, err := file.ActorSpellbooks()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, b := range books {
				if b.MapUnitID == 187 {
					found = true
					if !b.HasSpellbook || len(b.Spells) != 0 {
						t.Fatalf("source dragon book=%+v", b)
					}
				}
			}
			if !found {
				t.Fatal("source has no dragon book")
			}
			f, a := kargallasLoad(t, raw)
			check(t, f, a, false)
		})
	}
}

func TestReleaseKargallasNativeLootSharesPackCells(t *testing.T) {
	for _, input := range []struct {
		key, hash string
		totals    [3]uint32
	}{
		{"AGAINROM_KARGALLAS_SAV", "7298b24b9d72c3ac1e149b5b01f8dbbcb9fb1d44f3450621d2d62b41263d01f8", [3]uint32{14, 11, 6}},
		{"AGAINROM_KARGALLAS_END_SAV", "b5cff799f5d5470bcc3d45442987121f41574754df60f875ed6a60967a6ae050", [3]uint32{17, 13, 8}},
	} {
		t.Run(input.key, func(t *testing.T) {
			raw := kargallasInput(t, input.key, input.hash)
			f, _ := kargallasLoad(t, raw)
			check := func(label string, stacks []sim.ItemStack) {
				t.Helper()
				for i, code := range []uint16{0x1626, 0x0202, 0xac3c} {
					cells := 0
					var count uint32
					for _, st := range stacks {
						if st.Code == code {
							cells++
							count += st.Count
						}
					}
					if cells != 1 || count != input.totals[i] {
						t.Fatalf("%s code=%04x cells=%d count=%d, want one x%d", label, code, cells, count, input.totals[i])
					}
				}
			}
			stacks, _ := f.live.world.CarriedStacks(f.live.mission.ids[2])
			check("mission LOAD", stacks)
			_, counts, _ := buildInventoryPack(nil, stacks, nil)
			if len(counts) != len(stacks) {
				t.Fatal("mission pack omits stack quantities")
			}
			for i, count := range counts {
				if count != stacks[i].Count {
					t.Fatal("mission pack displays a different stack quantity")
				}
			}
			g := f
			if input.key == "AGAINROM_KARGALLAS_SAV" {
				path := saveCorpseMission(t, f, t.TempDir())
				g = loadAreaContinuation(t, path)
				stacks, _ = g.live.world.CarriedStacks(g.live.mission.ids[2])
				check("mission cold LOAD", stacks)
			}
			if _, _, err := g.LiveCompleteCampaign(); err != nil {
				t.Fatal(err)
			}
			check("town", cityMemberStacks(g.Carried[2], g.Table))
			h := currentTownReload(t, currentTownSave(t, g))
			check("town cold LOAD", cityMemberStacks(h.Carried[2], h.Table))
		})
	}
}
