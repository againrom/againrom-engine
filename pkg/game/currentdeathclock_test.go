package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func currentDeathClockActor(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.Alive() && e.MaxHP > 0 {
			return e
		}
	}
	t.Fatal("fixture has no living actor with health")
	return sim.Entity{}
}

func currentDeathClockFrame(t *testing.T, f *FrontEnd, id sim.EntityID, art *terrain.UnitClass) int {
	t.Helper()
	if f.live.art == nil {
		f.live.art = make(map[sim.EntityID]*terrain.UnitClass)
	}
	f.live.art[id] = art
	draw := deathDraw(t, f.live, id)
	if draw.Art != art.Corpse {
		t.Fatalf("actor %d drew %p, want corpse art %p", id, draw.Art, art.Corpse)
	}
	return frameIndex(art.Corpse, draw.Frame)
}

func TestCurrentDeathClockKeepsFallenFrameAcrossColdLoadsAndTicks(t *testing.T) {
	f := currentArchiveFixture(t)
	e := currentDeathClockActor(t, f)
	if err := f.live.world.HeadlessDamage(e.ID, e.HP); err != nil {
		t.Fatal(err)
	}
	if body, ok := f.live.world.Entity(e.ID); !ok || body.HP != 0 || body.Decay != sim.DecayFallen {
		t.Fatal("fixture did not reach a downed fallen body", body, ok)
	}
	art := deathBundle().Classes[1]
	currentDeathClockFrame(t, f, e.ID, art)
	f.live.scene += 4
	want := currentDeathClockFrame(t, f, e.ID, art)
	if want != deathFrameAt(sheetOctant(e.DrawnFacing()), 4) {
		t.Fatal("fixture did not reach the third death frame", want)
	}
	for cycle := 0; cycle < 2; cycle++ {
		s, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		cold := openCurrentArchive(t, raw)
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatal("SAVE/LOAD changed the World", cycle)
		}
		if got := currentDeathClockFrame(t, cold, e.ID, art); got != want {
			t.Fatalf("cycle %d cold death frame %d, want %d", cycle, got, want)
		}
		for tick := 0; tick < 2; tick++ {
			f.live.tick()
			cold.live.tick()
			if f.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("next tick changed the World", cycle, tick)
			}
			want = currentDeathClockFrame(t, f, e.ID, art)
			if got := currentDeathClockFrame(t, cold, e.ID, art); got != want {
				t.Fatalf("cycle %d tick %d death frame %d, want %d", cycle, tick, got, want)
			}
		}
		f = cold
	}
}

func TestCurrentDeathClockRestartsAfterHealingAndRefalling(t *testing.T) {
	f := currentArchiveFixture(t)
	e := currentDeathClockActor(t, f)
	art := deathBundle().Classes[1]
	if err := f.live.world.HeadlessDamage(e.ID, e.HP); err != nil {
		t.Fatal(err)
	}
	currentDeathClockFrame(t, f, e.ID, art)
	f.live.scene += 4
	if got := currentDeathClockFrame(t, f, e.ID, art); got != deathFrameAt(sheetOctant(e.DrawnFacing()), 4) {
		t.Fatal("first fall did not advance", got)
	}
	if err := f.live.world.HeadlessHeal(e.ID); err != nil {
		t.Fatal(err)
	}
	f.live.entityDraws()
	if _, ok := f.live.died[e.ID]; ok {
		t.Fatal("healed actor kept its death stamp")
	}
	healed, ok := f.live.world.Entity(e.ID)
	if !ok || !healed.Alive() {
		t.Fatal("actor did not recover", healed, ok)
	}
	if err := f.live.world.HeadlessDamage(e.ID, healed.HP); err != nil {
		t.Fatal(err)
	}
	if got := currentDeathClockFrame(t, f, e.ID, art); got != deathFrameAt(sheetOctant(e.DrawnFacing()), 0) {
		t.Fatal("second fall did not start on its first frame", got)
	}
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := openCurrentArchive(t, raw)
	if got := currentDeathClockFrame(t, cold, e.ID, art); got != deathFrameAt(sheetOctant(e.DrawnFacing()), 0) {
		t.Fatal("second fall changed on cold load", got)
	}
}

func TestCurrentDeathClockKeepsBoneAndBonelessFallbackFrames(t *testing.T) {
	for _, tc := range []struct {
		name    string
		art     *terrain.UnitClass
		hasBone bool
	}{
		{"bone block", bnBundle().Classes[1], true},
		{"death frame fallback", bnBundle().Classes[3], false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := currentArchiveFixture(t)
			e := currentDeathClockActor(t, f)
			if err := f.live.world.HeadlessDamage(e.ID, e.HP+15); err != nil {
				t.Fatal(err)
			}
			body, ok := f.live.world.Entity(e.ID)
			for tick := 0; tick < 128 && ok && body.Decay < sim.DecayBones; tick++ {
				f.live.tick()
				body, ok = f.live.world.Entity(e.ID)
			}
			if !ok || body.Decay < sim.DecayBones {
				t.Fatal("fixture did not reach the bone stage", body.Decay, body.HP, ok)
			}
			f.live.scene += 4
			want := currentDeathClockFrame(t, f, e.ID, tc.art)
			if want < 0 {
				t.Fatal("fixture did not select a corpse frame")
			}
			oct := sheetOctant(body.DrawnFacing())
			expected := bnDyingBase + oct*bnDyingSlot + bnDyingSlot - 1
			if tc.hasBone {
				expected = bnBoneFrameAt(oct, int(body.Decay))
			}
			if want != expected {
				t.Fatalf("fixture selected frame %d, want %d", want, expected)
			}
			s, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
			cold := openCurrentArchive(t, raw)
			if got := currentDeathClockFrame(t, cold, e.ID, tc.art); got != want {
				t.Fatalf("cold frame %d, want %d", got, want)
			}
			f.live.tick()
			cold.live.tick()
			if got, next := currentDeathClockFrame(t, cold, e.ID, tc.art), currentDeathClockFrame(t, f, e.ID, tc.art); got != next {
				t.Fatalf("next tick frame %d, want %d", got, next)
			}
		})
	}
}

func TestOriginalDeathClockStartsWithoutCurrentAgeLeaf(t *testing.T) {
	actor := &poolFixtureActor{mapID: 91, cell: 0x100f, stage: 1, hp: 65528, maxHP: 31, timer: 7}
	f := currentRetainedRuntimeFixture(t, nil, actor)
	_, present, err := sav.NativeActions(f.live.mission.state.savedDocument.Document.State)
	if err != nil || present {
		t.Fatal("fixture unexpectedly carries current actions", present, err)
	}
	e := poolEntity(t, f.live.world, 91)
	if e.HP != -8 || e.Decay != sim.DecayFallen || e.Dwell != 7 {
		t.Fatal("original SAV did not retain its downed state", e.HP, e.Decay, e.Dwell)
	}
	if got := currentDeathClockFrame(t, f, e.ID, deathBundle().Classes[1]); got != deathFrameAt(sheetOctant(e.DrawnFacing()), 0) {
		t.Fatal("original SAV without death age did not start at frame zero", got)
	}
}

func TestCurrentDeathClockRejectsMalformedRowsBeforeCommit(t *testing.T) {
	f := currentArchiveFixture(t)
	e := currentDeathClockActor(t, f)
	if err := f.live.world.HeadlessDamage(e.ID, e.HP); err != nil {
		t.Fatal(err)
	}
	f.live.entityDraws()
	f.live.scene += 4
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"duplicate", "negative", "excessive", "unbound"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.DeathAges) != 1 {
				t.Fatal("fixture lacks one death-age row", err, a.DeathAges)
			}
			switch change {
			case "duplicate":
				a.DeathAges = append(a.DeathAges, a.DeathAges[0])
			case "negative":
				a.DeathAges[0].Age = -1
			case "excessive":
				a.DeathAges[0].Age = maxCurrentDeathAge + 1
			case "unbound":
				a.DeathAges[0].Entity = 4000000
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := f.RestoreOriginal(candidate); err == nil {
				t.Fatal("malformed death age accepted")
			}
			after, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("refused LOAD changed current state")
			}
		})
	}
}

func TestCurrentDeathClockRestoresByBoundActorIdentityAtomically(t *testing.T) {
	mw := deathWorld(t, nil, deathUnit(1, 1, 2, 2), deathUnit(2, 1, 3, 3))
	if err := mw.world.HeadlessDamage(1, 100); err != nil {
		t.Fatal(err)
	}
	mw.entityDraws()
	mw.scene = 20
	old := mw.died[1]
	rows := []currentDeathAge{{Entity: 900, Age: 5}, {Entity: 901, Age: 4}}
	ids := map[sim.EntityID]sim.EntityID{900: 1, 901: 2}
	if err := mw.restoreCurrentDeathAges(rows, ids); err == nil || mw.died[1] != old {
		t.Fatal("invalid second row partially changed a death clock", err, mw.died[1], old)
	}
	if err := mw.restoreCurrentDeathAges(rows[:1], ids); err != nil {
		t.Fatal(err)
	}
	if got := mw.observeDeath(1); got != 5 {
		t.Fatal("saved label did not remap to the fallen actor", got)
	}
}
