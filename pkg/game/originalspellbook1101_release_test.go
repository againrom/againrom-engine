package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestMageSpellbookProcess1101(t *testing.T) {
	f := releaseFront(t)
	mode := os.Getenv("AGAINROM_1101_PROCESS")
	if mode == "" {
		return
	}
	input, err := ReadSaveFile(os.Getenv("AGAINROM_1101_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	if mode == "load" {
		app := openLocalTownSAV(t, f, filepath.Dir(os.Getenv("AGAINROM_1101_INPUT")), filepath.Base(os.Getenv("AGAINROM_1101_INPUT")))
		m := trainingPartyMember(t, f, os.Getenv("AGAINROM_1101_MEMBER"))
		if got := fmt.Sprint(m.KnownSpells, m.Book); got != os.Getenv("AGAINROM_1101_HASH") {
			t.Fatalf("fresh LOAD book %s", got)
		}
		_, _, raw := menuSAVE(t, f, app, OriginalStore{})
		if err := os.WriteFile(os.Getenv("AGAINROM_1101_OUTPUT"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("AGAINROM_1101_OUTPUT")+".trace", mageMissionTrace(t, f, m.ID), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if mode == "native" {
		opener, town, err := f.RestoreOriginal(input)
		if err != nil || town {
			t.Fatalf("native mission LOAD %v %v", town, err)
		}
		if err := f.App("1101-native").OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		for i, p := range f.live.mission.party {
			e, _ := f.live.entity(f.live.mission.ids[i])
			if e.Book != p.Book {
				t.Fatal("native LOAD lost source book parameters")
			}
		}
		ticks, _ := strconv.Atoi(os.Getenv("AGAINROM_1101_TICKS"))
		for range ticks {
			sim.Step(f.live.world, nil)
		}
		if got := fmt.Sprintf("%x", f.live.world.Hash()); got != os.Getenv("AGAINROM_1101_HASH") {
			t.Fatalf("fresh continuation hash %s", got)
		}
		return
	}
	t.Fatal("unknown mode", mode)
}

func exerciseBookMission1101(t *testing.T, f *FrontEnd, id sim.EntityID, memberID, path string) {
	t.Helper()
	e, _ := f.live.entity(id)
	// Light targets the caster's visible cell without needing a particular
	// enemy placement. Admission and wind-up are the real mission World's.
	const spell = 12
	if e.KnownSpells&(1<<spell) == 0 {
		t.Skip("this source mage does not know Light; city/native conversion already witnessed")
	}
	if reason := f.live.world.BookSpellCellRefusal(id, e.X, e.Y, spell); reason != "" {
		t.Fatal("mission cast", reason)
	}
	sim.Step(f.live.world, []sim.Command{{Kind: sim.KindCastAt, Entity: id, X: e.X, Y: e.Y, Spell: spell}})
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteConvertedSave(path, raw, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	ticks, released := 0, false
	for ticks < 512 && !released {
		ticks++
		for _, event := range sim.StepObserved(f.live.world, nil) {
			if event.Caster == id && event.Spell == spell && !event.Weapon {
				released = true
			}
		}
	}
	after, _ := f.live.entity(id)
	if !released {
		t.Fatalf("mission cast did not release within512ticks: mana%d->%d charge%d", e.Mana, after.Mana, e.AttackCharge)
	}
	childMage1101(t, "native", path, "", memberID, fmt.Sprintf("%x", f.live.world.Hash()), ticks)
	t.Logf("mission Light cast: mana %d->%d; release after%d ticks; fresh native wind-up continuation identical", e.Mana, after.Mana, ticks)
}

func childMage1101(t *testing.T, mode, in, out, member, hash string, ticks ...int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMageSpellbookProcess1101$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_1101_PROCESS="+mode, "AGAINROM_1101_INPUT="+in,
		"AGAINROM_1101_OUTPUT="+out, "AGAINROM_1101_MEMBER="+member, "AGAINROM_1101_HASH="+hash)
	if len(ticks) != 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("AGAINROM_1101_TICKS=%d", ticks[0]))
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh %s: %v\n%s", mode, err, output)
	}
	if bytes.Contains(output, []byte("--- SKIP:")) {
		t.Fatalf("fresh %s did not witness required continuation\n%s", mode, output)
	}
	t.Logf("fresh %s PASS", mode)
}

func TestReleaseMageSpellbookTrainSAVEAndContinuation(t *testing.T) {
	// Both lawful sources are replayed with each asset root; these are not
	// independent EN/RU original-runtime recordings.
	for _, fixture := range []struct{ path, hash string }{
		{"2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4"},
	} {
		t.Run(filepath.Base(fixture.path), func(t *testing.T) {
			f := releaseFront(t)
			path, source := groundCorpusFile(t, fixture.path, fixture.hash)
			store := SaveStore{Dir: t.TempDir()}
			app := f.App("1101-mage-training")
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			var label string
			for _, entry := range list() {
				if entry.Name == filepath.Base(path) {
					label = entry.Label
				}
			}
			if label == "" {
				t.Fatal("source absent in original picker")
			}
			if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
				t.Fatalf("App LOAD %v", err)
			}
			var member mapload.PartyMember
			memberIndex := -1
			for i, p := range f.Carried {
				if p.Mage {
					member, memberIndex = p, i
					break
				}
			}
			if memberIndex < 0 {
				t.Fatal("source has no mage")
			}
			h, ok := member.OriginalHumanState()
			if !ok {
				t.Fatalf("mage %q is not source-backed: %+v", member.Name, member.OriginalHuman)
			}
			gold, slot, price := f.Town.Gold(), 0, 0
			for i := 1; i <= 5; i++ {
				if p := memberSchoolPrice(member, i); p > 0 && p <= gold && (slot == 0 || p < price) {
					slot, price = i, p
				}
			}
			if slot == 0 {
				t.Fatal("no affordable mage school slot")
			}
			s := f.TownScreen().(*townScreen)
			s.room, s.schoolCell, s.shopMember = roomSchool, slot+4, memberIndex
			if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil {
				t.Fatal("App Train", err)
			}
			trained := trainingPartyMember(t, f, member.ID)
			next, valid := trained.OriginalHumanState()
			if !valid || next.Base.Skill[slot] != h.Base.Skill[slot]+1 || f.Town.Gold() != gold-price {
				t.Fatalf("App Train failed: %s", app.HeadlessMessage())
			}
			changedRanges := 0
			for _, rule := range mapload.SpellRules(f.Table) {
				if member.KnownSpells&(uint32(1)<<rule.ID) == 0 {
					continue
				}
				before, after := member.Book.Slots[rule.ID-1], trained.Book.Slots[rule.ID-1]
				power := max(0, min(100, int(next.Attack.Skill[rule.School])+int(next.Mind)-30))
				rangeWant := int(rule.MaxRange)
				divisor := 30
				if rule.ID == 26 {
					divisor = 3
				}
				if rangeWant != 0 {
					rangeWant += power / divisor
				}
				if after.Range != uint8(rangeWant) || before.ManaCost != after.ManaCost || before.Defensive != after.Defensive {
					t.Fatalf("spell %d refresh: %+v -> %+v", rule.ID, before, after)
				}
				if before.Range != after.Range {
					changedRanges++
				}
			}
			for range 2 {
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal("App SAVE", err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name, ".sav") {
				t.Fatalf("SAVE %v %v %s", entries, err, app.HeadlessMessage())
			}
			savedPath := filepath.Join(store.Dir, entries[0].Name)
			raw, err := ReadSaveFile(savedPath)
			if err != nil || !bytes.HasPrefix(raw, []byte(sav.Magic)) {
				t.Fatal("ordinary SAVE not Asg&", err)
			}
			saved := savedCityMage(t, raw, member.Name)
			requireSavedBook(t, saved.KnownSpells, saved.Spells, trained.KnownSpells, trained.Book)
			backPath := filepath.Join(t.TempDir(), "again.sav")
			childMage1101(t, "load", savedPath, backPath, member.ID, fmt.Sprint(trained.KnownSpells, trained.Book))
			back, err := ReadSaveFile(backPath)
			if err != nil {
				t.Fatal(err)
			}
			again := savedCityMage(t, back, member.Name)
			requireSavedBook(t, again.KnownSpells, again.Spells, trained.KnownSpells, trained.Book)
			f.SetDeterministicFrames(true)
			live := strings.Split(string(mageMissionTrace(t, f, member.ID)), "\n")
			freshTrace, err := os.ReadFile(backPath + ".trace")
			if err != nil {
				t.Fatal(err)
			}
			cold := strings.Split(string(freshTrace), "\n")
			if len(cold) != len(live) {
				t.Fatalf("fresh continuation ran %d ticks, live %d", len(cold), len(live))
			}
			for tick := range live {
				if live[tick] != cold[tick] {
					t.Fatalf("tick %d: live %s, fresh %s", tick, live[tick], cold[tick])
				}
			}
			control := saved.Spells[0]
			altered := alterSAV(t, raw, func(doc *sav.DocumentData) bool { return setSavedSpellRange(doc, control.Key, control.Range+1) })
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "altered.sav"), altered, 0o600); err != nil {
				t.Fatal(err)
			}
			lossy := releaseFront(t)
			openLocalTownSAV(t, lossy, dir, "altered.sav")
			if got := trainingPartyMember(t, lossy, member.ID).Book.Slots[control.ID-1]; got.Range != control.Range+1 || got == trained.Book.Slots[control.ID-1] {
				t.Fatalf("altered spell %d loaded %+v; the LOAD does not read the saved book", control.ID, got)
			}
			if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, source) {
				t.Fatal("source was changed")
			}
			t.Logf("source=%x mage=%s slot=%d gold=%d->%d HPmax=%d->%d ManaMax=%d->%d book=%d changedRanges=%d SAV=%d bytes; App LOAD/Train/SAVE; fresh-process LOAD/SAVE keeps %d spells; %d-tick mission continuation equal; altered spell %d range loads", sha256.Sum256(source), member.Name, slot, gold, gold-price, h.HealthMax, next.HealthMax, h.ManaMax, next.ManaMax, member.Book.State, changedRanges, len(raw), len(again.Spells), len(live)-1, control.ID)
		})
	}
}

func TestReleaseFergardSpellbookInstancesAndNativeCast(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	opener, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("Fergard mission LOAD %v %v", town, err)
	}
	if err := f.App("1101-fergard").OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	file, _ := sav.Open(raw)
	characters, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	var source sav.Character
	for _, c := range characters {
		if len(c.Spells) == 28 {
			source = c
		}
	}
	if len(source.Spells) != 28 {
		t.Fatal("Fergard's 28 source Spell bodies absent")
	}
	for i, spell := range source.Spells {
		// Independent source-body locations supplied with this lawful fixture;
		// no writer or semantic reader computes these offsets.
		body := file.Body[1119+11*i : 1119+11*i+5]
		if body[0] != spell.ID || body[1] != spell.Range || body[2] != spell.Defensive || binary.LittleEndian.Uint16(body[3:]) != spell.ManaCost {
			t.Fatalf("literal source Spell%d body %x disagrees with instance %+v", i+1, body, spell)
		}
	}
	var found bool
	for i, p := range f.live.mission.party {
		if p.Name != source.Name {
			continue
		}
		found = true
		id := f.live.mission.ids[i]
		e, _ := f.live.entity(id)
		if e.Book.State != sim.BookPresent || e.KnownSpells != source.KnownSpells() || p.KnownSpells != source.KnownSpells() || e.Book != p.Book {
			t.Fatal("Fergard book not canonical")
		}
		for _, spell := range source.Spells {
			want := sim.BookSpell{Range: spell.Range, Defensive: spell.Defensive, ManaCost: spell.ManaCost}
			if e.Book.Slots[spell.ID-1] != want {
				t.Fatalf("Fergard spell%d instance", spell.ID)
			}
		}
		exerciseBookMission1101(t, f, id, p.ID, filepath.Join(t.TempDir(), "fergard-windup.sav"))
		t.Logf("lawful Fergard source=%x; 28 saved Spell parameter sets imported; actual mission cast and fresh native continuation", sha256.Sum256(raw))
	}
	if !found {
		t.Fatal("Fergard not in restored party")
	}
}
