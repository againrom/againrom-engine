package game

import (
	"bytes"
	"fmt"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
)

// deadBookRow is one dead record's book as a SAV document holds it: the
// presence flag, the opaque header word and each slot's Spell operands.
type deadBookRow struct {
	has, header string
	slots       []string
}

func deadBookRows(t *testing.T, d *sav.DocumentData) (map[string]deadBookRow, map[string]*sav.DocumentRecordData, int) {
	t.Helper()
	rows := map[string]deadBookRow{}
	records := map[string]*sav.DocumentRecordData{}
	for _, root := range d.DeadActors {
		if root == 0 || int(root) > len(d.Objects) {
			t.Fatalf("dead root %d out of range", root)
		}
		r := &d.Objects[root-1]
		key, err := savedStructureValue(r, "Identity")
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("%s %d", r.Class, key)
		if _, repeated := rows[name]; repeated {
			t.Fatalf("dead record %s repeats", name)
		}
		row := deadBookRow{has: "absent", header: "absent"}
		if v, err := savedStructureValue(r, "HasSpellbook"); err == nil {
			row.has = fmt.Sprint(v)
		}
		if v, err := savedStructureValue(r, "SpellsHeader"); err == nil {
			row.header = fmt.Sprint(v)
		}
		for _, refs := range r.RefSlots {
			if refs.Name != "Spells" {
				continue
			}
			for slot, ref := range refs.Objects {
				if ref == 0 {
					continue
				}
				spell, err := savedSpellRecord(&d.Objects[ref-1])
				if err != nil {
					t.Fatal(err)
				}
				row.slots = append(row.slots, fmt.Sprintf("%d:%+v", slot, spell.Value))
			}
		}
		rows[name], records[name] = row, r
	}
	spells := 0
	for i := range d.Objects {
		if d.Objects[i].Class == "Spell" {
			spells++
		}
	}
	return rows, records, spells
}

// After LOAD of an original first-game SAV and an immediate SAVE, every dead
// record keeps the spellbook the loaded record held: a corpse entity holds
// the book LOAD gave it, and a body the World holds as no entity takes its
// loaded record's book. The Spell object count is the loaded one, and the
// SAVE loads cold.
func TestReleaseOriginalResaveKeepsDeadSpellbooks(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, c := range []struct{ name, hash string }{
		{"2026-09-24/game0002-bigsack.sav", "8842212e16cbb97ad05af059bdd637df2ab777b4c7f5bb5430353b68b5312c51"},
		{"2026-09-24/game0017-victory.sav", "a30de01ec3d96d58a3a244b9a536a3e49c04a4d45098cb828dabb908756488c3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, payload := groundCorpusFile(t, c.name, c.hash)
			loaded, err := sav.DecodeDocumentData(payload)
			if err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			a := f.App("dead spellbooks " + c.name)
			a.Layout(1024, 768)
			open, _, err := f.RestoreOriginal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			s, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			written, _, err := f.playerMissionSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
			out, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			entity := map[string]bool{}
			for _, d := range f.live.world.OriginalDeadActors() {
				if _, held := f.live.world.Entity(d.ID); held {
					entity[fmt.Sprintf("%s %d", savedActorClass(d.Source.Class), d.Source.Identity)] = true
				}
			}
			before, beforeRecords, beforeSpells := deadBookRows(t, &loaded)
			after, afterRecords, afterSpells := deadBookRows(t, &out)
			books, kept, moved := 0, 0, 0
			for name, want := range before {
				got, ok := after[name]
				if !ok {
					t.Errorf("%s: no dead record after SAVE", name)
					continue
				}
				if want.has == "1" {
					books++
				}
				if !entity[name] {
					kept++
				}
				if fmt.Sprint(got) != fmt.Sprint(want) {
					moved++
					t.Errorf("%s (entity %v): book after SAVE %+v, loaded %+v", name, entity[name], got, want)
				}
			}
			if len(after) != len(before) {
				t.Errorf("dead records after SAVE %d, loaded %d", len(after), len(before))
			}
			if afterSpells != beforeSpells {
				t.Errorf("Spell objects after SAVE %d, loaded %d", afterSpells, beforeSpells)
			}
			t.Logf("%d dead records, %d with a book; %d with no entity; %d entity bodies; %d write another book; Spell objects %d, loaded %d",
				len(before), books, kept, len(entity), moved, afterSpells, beforeSpells)
			// The actor values of a dead body the World holds as no entity
			// come from its placement and the tables (DIV-4120); they are
			// reported, not required.
			for name, w := range beforeRecords {
				g := afterRecords[name]
				if g == nil || name != "Human 397802536" && name != "Human 385308024" {
					continue
				}
				var moved []string
				for _, field := range []string{"U12C", "U8E", "U90", "UA3"} {
					want, err1 := savedStructureValue(w, field)
					got, err2 := savedStructureValue(g, field)
					if err1 != nil || err2 != nil || got != want {
						moved = append(moved, fmt.Sprintf("%s %d>%d", field, want, got))
					}
				}
				want, err1 := savedActorRaw(w, "U154", 180)
				got, err2 := savedActorRaw(g, "U154", 180)
				if err1 != nil || err2 != nil || !bytes.Equal(got, want) {
					moved = append(moved, "U154")
				}
				t.Logf("%s actor values moved (DIV-4120): %v", name, moved)
			}
			cold := releaseFront(t)
			if _, _, err := cold.RestoreOriginal(written); err != nil {
				t.Fatalf("cold LOAD of the SAVE: %v", err)
			}
		})
	}
}
