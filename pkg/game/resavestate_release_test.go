package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// withoutEngineState is raw without the three engine-state leaves the
// original drops when it resaves an engine SAV.
func withoutEngineState(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	dropped := 0
	doc.State.ValueRecords = slices.DeleteFunc(slices.Clone(doc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		for _, path := range []string{sav.NativeActionsPath, sav.NativeRandomPath, sav.NativeSessionPath} {
			if sav.StateRecordIs(doc.State.RootKind, doc.State.DirectoryRecords, r.Path, path) {
				dropped++
				return true
			}
		}
		return false
	})
	if dropped == 0 {
		t.Fatal("the engine SAV carries no engine-state leaf")
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// resaveTownView is the town state an original resave must not change: each
// line comes from the original's own bytes when the engine-state leaf is
// absent. An offer's Index is a handle, so only its sign is kept.
func resaveTownView(f *FrontEnd) []string {
	t := f.Town
	var out []string
	add := func(format string, v ...any) { out = append(out, fmt.Sprintf(format, v...)) }
	add("chapter %d selected %d open %t gold %d available %v", t.Chapter(), t.selectedMission(), t.Open(), t.Gold(), t.Available())
	offers := func(rows []TownOffer) string {
		var b strings.Builder
		for _, r := range rows {
			fmt.Fprintf(&b, "[%t %d %d]", r.Index >= 0, r.Mission, r.NPC)
		}
		return b.String()
	}
	for b := TownTavern; b <= TownSchool; b++ {
		add("building %d offers %s", b, offers(t.Offers(b)))
	}
	add("tavern roster %s", offers(t.TavernRoster()))
	var done []int
	for _, n := range t.camp.Main {
		if t.Done(n) {
			done = append(done, n)
		}
	}
	add("done mains %v documents %v", done, t.Documents())
	for typ := 1; typ < 16; typ++ {
		add("mercenary %d pool %d capacity %d enabled %t hired %t", typ, t.MercenaryPool(typ), t.MercenaryCapacity(typ), t.MercenaryEnabled(typ), t.MercenaryHired(typ))
	}
	add("quick spells %v", f.quickSpells)
	for i, m := range f.NextParty() {
		p := partyPanelSubject(m, f.Table)
		add("member %d %s speed %d weight %d hp %d mana %d", i, m.ID, p.Speed, p.Weight, p.MaxHP, p.MaxMana)
	}
	return out
}

// resaveTownDebt is the town state the original's bytes do not carry:
// completed side missions (SAV-CAMPAIGN-077) and the count the shop seed
// reads.
func resaveTownDebt(f *FrontEnd) string {
	var side []int
	for _, n := range f.Town.camp.Side {
		if f.Town.Done(n) {
			side = append(side, n)
		}
	}
	return fmt.Sprintf("done sides %v finished %d", side, f.Town.finishedCount())
}

func resaveViewDiff(a, b []string) string {
	if len(a) != len(b) {
		return fmt.Sprintf("%d lines against %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Sprintf("%q against %q", a[i], b[i])
		}
	}
	return ""
}

// The owner's engine town SAV and the original's resave of it reload to the
// same town view; the original's file carries no engine-state leaf.
func TestReleaseOriginalResaveReloadsTheSameTown(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := os.Getenv("AGAINROM_CITY_RESAVE_DIR")
	if dir == "" {
		t.Skip("AGAINROM_CITY_RESAVE_DIR is not set")
	}
	var views [2][]string
	for i, name := range []string{"citysaved.sav", "cityresaved.sav"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		f := currentTownReload(t, raw)
		views[i] = resaveTownView(f)
		t.Logf("%s: %s", name, resaveTownDebt(f))
	}
	if d := resaveViewDiff(views[0], views[1]); d != "" {
		t.Fatalf("the resave reloads a different town: %s", d)
	}
}

// An engine town SAV with taken offers reloads to the same town view with its
// engine-state leaves removed: a tavern pair in chapter 50, a shop and a school
// element in chapter 130. Loss control: the same leafless file with the
// chapter's registry arrays in place of the shortened ones offers the taken
// missions again.
func TestReleaseLeaflessTownReloadKeepsTheTown(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, c := range []struct {
		chapter int
		take    []int
	}{{50, []int{51}}, {130, []int{130, 131}}} {
		t.Run(fmt.Sprint(c.chapter), func(t *testing.T) {
			f := kitOwnerWalkToChapter(t, c.chapter)
			var took []int
			for b := TownTavern; b <= TownSchool; b++ {
				for _, offer := range f.Town.Offers(b) {
					if slices.Contains(c.take, offer.Mission) {
						if m, ok := f.Town.Take(b, offer.Index); ok {
							took = append(took, m)
						}
					}
				}
			}
			if len(took) != len(c.take) {
				t.Fatalf("took %v in chapter %d, want %v", took, f.Town.Chapter(), c.take)
			}
			raw := currentTownSave(t, f)
			engine := currentTownReload(t, raw)
			want := resaveTownView(engine)
			leafless := withoutEngineState(t, raw)
			got := currentTownReload(t, leafless)
			if d := resaveViewDiff(want, resaveTownView(got)); d != "" {
				t.Fatalf("the leafless reload differs: %s", d)
			}
			t.Logf("engine %s; leafless %s", resaveTownDebt(engine), resaveTownDebt(got))

			doc, err := sav.DecodeDocumentData(leafless)
			if err != nil {
				t.Fatal(err)
			}
			ch := f.Campaign.Value().Chapters[engine.Town.Chapter()]
			u16 := func(v []int) []uint16 {
				out := make([]uint16, len(v))
				for i, n := range v {
					out[i] = uint16(n)
				}
				return out
			}
			doc.Campaign.Arrays[2], doc.Campaign.Arrays[3] = u16(ch.InnNPC), u16(ch.Inn)
			doc.Campaign.Arrays[4], doc.Campaign.Arrays[5] = u16(ch.School), u16(ch.Shop)
			control, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			if resaveViewDiff(want, resaveTownView(currentTownReload(t, control))) == "" {
				t.Fatal("loss control: the registry arrays reload the same offers")
			}
		})
	}
}

// A native hero carrying table-weighted items keeps his pack's running
// weight through a reload without the engine-state leaf: SAVE writes the
// container sum the world keeps. Loss control: the same file with the
// instance-weight sum reloads lighter.
func TestReleaseLeaflessReloadKeepsTableWeightedPackLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("table pack").OpenMission(f.MissionOpenerWith(101, overloadOrderParty(t, f))); err != nil {
		t.Fatal(err)
	}
	hero := overloadOrderHero(t, f)
	pack, _ := f.live.world.CarriedStacks(hero.ID)
	instance := int32(0)
	for _, st := range pack {
		if st.WeightPresent {
			instance += int32(st.Weight) * int32(st.Count)
		}
	}
	carried, _ := f.live.world.ContainerWeight(hero.ID)
	own, _ := f.live.world.WornWeight(hero.ID)
	if carried == instance || hero.Load != sim.CarriedLoad(own, carried) {
		t.Fatalf("pack weight %d, instance weight %d, load %d: the witness needs table-weighted items", carried, instance, hero.Load)
	}
	store := SaveStore{Dir: t.TempDir()}
	name, doc := deadPatrolSave(t, f, store)
	if got := overloadOrderRecordAccumulator(t, doc); got != carried {
		t.Fatalf("SAV running weight %d, want the container sum %d", got, carried)
	}
	raw := withoutEngineState(t, mustReadSave(t, store, name))
	odoc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture caster's mana pool has no regeneration period, which an
	// original profile refuses; give it one so the file loads.
	for i := range odoc.Objects {
		r := &odoc.Objects[i]
		if r.Class == "Human" && humanSpeedValue(t, r, "ManaMax") != 0 && humanSpeedValue(t, r, "ManaRegen") == 0 {
			savedObjectSetValue(r, "ManaRegen", 1)
		}
	}
	if raw, err = sav.EncodeDocumentData(odoc); err != nil {
		t.Fatal(err)
	}
	reload := func(raw []byte) sim.Entity {
		g := releaseFront(t)
		g.SetDeterministicFrames(true)
		mission, town, err := g.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatalf("leafless LOAD: town=%t err=%v", town, err)
		}
		if err := g.App("table pack leafless").OpenMission(mission); err != nil {
			t.Fatal(err)
		}
		g.live.tick()
		return overloadOrderHero(t, g)
	}
	if got := reload(raw); got.ActorLoad.Accumulator != carried || got.Load != hero.Load {
		t.Fatalf("leafless reload running weight %d load %d, want %d and %d", got.ActorLoad.Accumulator, got.Load, carried, hero.Load)
	}
	overloadOrderSetAccumulator(t, &odoc, uint32(instance))
	control, err := sav.EncodeDocumentData(odoc)
	if err != nil {
		t.Fatal(err)
	}
	if got := reload(control); got.ActorLoad.Accumulator == carried {
		t.Fatalf("loss control: the instance-weight word %d reloads running weight %d", instance, got.ActorLoad.Accumulator)
	}
}

func overloadOrderRecordAccumulator(t *testing.T, doc sav.DocumentData) int32 {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class == "Human" && humanSpeedValue(t, r, "Reaction") == 15 {
			return int32(humanSpeedValue(t, r, "Inventory20"))
		}
	}
	t.Fatal("no saved hero record")
	return 0
}
