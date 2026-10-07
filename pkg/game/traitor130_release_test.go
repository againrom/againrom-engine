package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type betrayal130 struct {
	f                                *FrontEnd
	app                              *ui.App
	hero, companion, veglud, breeder sim.EntityID
}

// Prior wins and exact approach positions are controlled prerequisites. The
// real chapter AddHero grant supplies NPC22, and the ordinary App door compiles
// and runs the installed mission. No ownership, diplomacy or verdict is supplied.
func startBetrayal130(t *testing.T, female bool) *betrayal130 {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	sex := 0
	if female {
		sex = 1
	}
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Betrayal witness", Choices: []int{sex, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	for _, mission := range f.Campaign.Value().Main {
		if mission >= 130 {
			break
		}
		f.addChapterCompanions(mission)
		f.Town.Won(mission)
	}
	if f.Town.Chapter() != 130 {
		t.Fatalf("controlled prerequisite chapter=%d, want130", f.Town.Chapter())
	}
	takeCampaignOffer(t, f, 130)
	app := f.App("Mission 130 betrayal")
	if err := app.OpenMission(f.MissionOpenerWith(130, f.NextParty())); err != nil {
		t.Fatal("offered mission entry", err)
	}
	x := &betrayal130{f: f, app: app, hero: f.live.mission.ids[0]}
	primary := f.live.mission.party[0]
	primaryDir := data.FigureDir(primary.FigureDir)
	if primary.Mage || primaryDir != data.FigureDirFor(false, female) {
		t.Fatalf("chargen primary mage=%v figure=%q, want female=%v fighter", primary.Mage, primaryDir, female)
	}
	found := false
	for i, p := range f.live.mission.party {
		if p.CompanionNPC == 22 {
			x.companion, found = f.live.mission.ids[i], true
			// NPC22's published selector is Mage,!MySex. Read both
			// character axes from the decoded figure directory, not a name.
			companionDir := data.FigureDir(p.FigureDir)
			if !p.PlayerCharacter || !p.Mage || companionDir != data.FigureDirFor(true, !primaryDir.Female()) {
				t.Fatalf("real AddHero22 mage=%v figure=%q must oppose primary figure=%q", p.Mage, companionDir, primaryDir)
			}
			t.Logf("NPC22 mage=%v primary_female=%v companion_female=%v", p.Mage, primaryDir.Female(), companionDir.Female())
		}
	}
	if !found {
		t.Fatal("real chapter AddHero path did not supply NPC22")
	}
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	var ok bool
	if x.veglud, ok = refs[135]; !ok {
		t.Fatal("installed mission lacks Veglud unit135")
	}
	if x.breeder, ok = refs[110]; !ok {
		t.Fatal("installed mission lacks breeder unit110")
	}
	src, err := f.live.mission.state.Map.Script()
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, a := range src.Actions {
		if a.ID == 39 {
			found = a.Opcode == 19 && a.Type[0] == 4 && a.Value[0] == 10002 && a.Type[1] == 3 && a.Value[1] == 8
		}
	}
	if !found {
		t.Fatal("installed action39 no longer gives the second hero to player8")
	}
	return x
}

func (x *betrayal130) entity(t *testing.T, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := x.f.live.entity(id)
	if !ok {
		t.Fatalf("entity%d is missing", id)
	}
	return e
}

func (x *betrayal130) running(t *testing.T, stage string) {
	t.Helper()
	w := x.f.live.world
	_, lost := w.ScriptCounters()
	if x.f.live.mission.outcome != sim.OutcomeUndecided || w.Outcome() != sim.OutcomeUndecided || lost != 0 {
		t.Fatalf("%s: client=%d world=%d lost=%d state50=%d tick=%d", stage,
			x.f.live.mission.outcome, w.Outcome(), lost, w.ScriptRegister(50), w.Tick())
	}
	if _, kind, open := x.f.LiveNotice(); open && kind != ui.NoticeDialogue {
		t.Fatalf("%s: unexpected outcome notice%d", stage, kind)
	}
}

// Advance the real App and acknowledge only dialogue pages. Outcome notices
// remain visible so a test cannot accidentally dismiss a failure and continue.
func (x *betrayal130) drive(t *testing.T, ticks uint64) {
	t.Helper()
	target := x.f.live.world.Tick() + ticks
	for frames := 0; frames < 4*int(ticks)+100; frames++ {
		if x.f.live.world.Tick() >= target {
			return
		}
		if _, kind, open := x.f.LiveNotice(); open {
			if kind != ui.NoticeDialogue {
				return
			}
			if err := x.app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if err := x.app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("App did not advance to tick%d", target)
}

func (x *betrayal130) cold(t *testing.T, stage string, expectedWorld []byte) {
	t.Helper()
	var store SaveStore
	if expectedWorld == nil {
		var saved Snapshot
		store, saved, _ = campaignSave(t, x.f, true)
		expectedWorld = saved.World
	} else {
		// This is a legacy input fixture for unresolved script references.
		// Ordinary player SAVE above always uses the current SAV producer.
		saved, _, err := x.f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := EncodeSave(saved, "Legacy unresolved program")
		if err != nil {
			t.Fatal(err)
		}
		store = SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(store.Dir, "legacy.ags"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	f, app := campaignCold(t, store)
	if f.live == nil || f.liveMission != 130 || app.Screen() != ui.ScreenMap {
		t.Fatalf("%s: cold SAV did not open mission130: screen=%v message=%q", stage, app.Screen(), app.HeadlessMessage())
	}
	x.f, x.app = f, app
	actual, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expectedWorld) {
		t.Fatalf("%s: cold LOAD changed World beyond the expected binding repair", stage)
	}
	t.Logf("cold LOAD %s tick=%d state50=%d", stage, f.live.world.Tick(), f.live.world.ScriptRegister(50))
}

func (x *betrayal130) prerequisite(t *testing.T, breederFirst bool) {
	t.Helper()
	want := int32(0)
	if breederFirst {
		if err := x.f.live.world.HeadlessKill(x.breeder); err != nil {
			t.Fatal(err)
		}
		want = 1
	}
	x.drive(t, 48)
	x.running(t, "distant companion control")
	if got := x.f.live.world.ScriptRegister(50); got != want {
		t.Fatalf("before approach state=%d, want%d", got, want)
	}
	if got := x.entity(t, x.companion).Owner; got != sim.SelfSlot {
		t.Fatalf("distant companion owner=%d", got)
	}
}

func (x *betrayal130) transfer(t *testing.T, breederFirst bool) {
	t.Helper()
	v := x.entity(t, x.veglud)
	livePlaceAndWalk(t, x.f.live, x.companion, v.X+1, v.Y)
	x.drive(t, 48)
	x.running(t, "authored transfer")
	wantState, latch := int32(2), int32(9)
	if breederFirst {
		wantState, latch = 4, 8
	}
	if got := x.entity(t, x.companion).Owner; got != 8 {
		t.Fatalf("installed Give Unit did not transfer companion: owner=%d", got)
	}
	if got := x.f.live.world.ScriptRegister(50); got != wantState || !x.f.live.world.ScriptLatched(latch) {
		t.Fatalf("betrayal state=%d want%d, authored latch%d=%v", got, wantState, latch, x.f.live.world.ScriptLatched(latch))
	}
}

func (x *betrayal130) hostile(t *testing.T, breederFirst bool) {
	t.Helper()
	if !breederFirst {
		if err := x.f.live.world.HeadlessKill(x.breeder); err != nil {
			t.Fatal(err)
		}
		x.drive(t, 32)
		if got := x.f.live.world.ScriptRegister(50); got != 3 {
			t.Fatalf("breeder death after deal: state=%d want3", got)
		}
		h, v := x.entity(t, x.hero), x.entity(t, x.veglud)
		if err := x.f.live.world.HeadlessPlace(x.hero, v.X-1, v.Y-1); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 64 && x.f.live.world.ScriptRegister(50) != 5; i++ {
			x.drive(t, 1)
		}
		if err := x.f.live.world.HeadlessPlace(x.hero, h.X, h.Y); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 16 && !x.f.live.world.Relations().Hostile(8, 1); i++ {
			x.drive(t, 4)
		}
	}
	x.running(t, "permitted hostile stage")
	want := int32(5)
	if breederFirst {
		want = 4
	}
	if got := x.f.live.world.ScriptRegister(50); got != want || !x.f.live.world.Relations().Hostile(8, 1) {
		t.Fatalf("authored hostility missing: state=%d want%d relation8to1=%d", got, want, x.f.live.world.Relations().Byte(8, 1))
	}
}

func (x *betrayal130) requireGuardLoss(t *testing.T, victim sim.EntityID) {
	t.Helper()
	if err := x.f.live.world.HeadlessKill(victim); err != nil {
		t.Fatal(err)
	}
	x.drive(t, 32)
	_, lost := x.f.live.world.ScriptCounters()
	_, kind, open := x.f.LiveNotice()
	if x.f.live.mission.outcome != sim.OutcomeLost || x.f.live.world.Outcome() != sim.OutcomeUndecided || lost != 0 || !open || kind != ui.NoticeFailure {
		t.Fatalf("active character%d death: client=%d world=%d lost=%d notice=%v/%d", victim,
			x.f.live.mission.outcome, x.f.live.world.Outcome(), lost, open, kind)
	}
}

// At the start of a legacy fixture, replace only program bytes. Two controlled
// copies locate those bytes without depending on serialized offsets; their
// reset execution state is never installed. Original clock/registers/latches,
// counters, outcomes and every non-program section remain byte-identical.
func betrayal130ProgramForm(t *testing.T, world *sim.World, script *sim.Script) []byte {
	t.Helper()
	original, err := world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	before, err := sim.NewControlledScriptWorld(world, world.Script())
	if err != nil {
		t.Fatal(err)
	}
	after, err := sim.NewControlledScriptWorld(world, script)
	if err != nil {
		t.Fatal(err)
	}
	a, err := before.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b, err := after.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(original) != len(a) || len(a) != len(b) {
		t.Fatal("legacy program replacement changed the world shape")
	}
	for i := range a {
		if a[i] != b[i] {
			if original[i] != a[i] {
				t.Fatal("legacy fixture attempted to overwrite runtime state")
			}
			original[i] = b[i]
		}
	}
	var check sim.World
	if err := check.UnmarshalBinary(original); err != nil {
		t.Fatal("legacy program form", err)
	}
	if check.Tick() != world.Tick() || check.ScriptRegisters() != world.ScriptRegisters() || check.Outcome() != world.Outcome() {
		t.Fatal("legacy program replacement changed execution state")
	}
	for i := int32(0); i < 1000; i++ {
		if check.ScriptLatched(i) != world.ScriptLatched(i) {
			t.Fatal("legacy program replacement changed latches")
		}
	}
	if !reflect.DeepEqual(check.Script().Checks(), script.Checks()) || !reflect.DeepEqual(check.Script().Instants(), script.Instants()) || !reflect.DeepEqual(check.Script().Triggers(), script.Triggers()) {
		t.Fatal("legacy form did not carry the exact requested program")
	}
	return original
}

func (x *betrayal130) legacyProgram(t *testing.T) *sim.Script {
	t.Helper()
	w := x.f.live.world
	correct := w.Script()
	checks, instants := correct.Checks(), correct.Instants()
	src, err := x.f.live.mission.state.Map.Script()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	clearRef := func(n alm.ScriptNode, unit, unit2 *sim.EntityID, has, has2 *bool) {
		ordinal := 0
		for slot, kind := range n.Type {
			if kind != 4 {
				continue
			}
			ordinal++
			if n.Value[slot] != 10002 {
				continue
			}
			if ordinal == 1 {
				if !*has || *unit != x.companion {
					t.Fatal("fresh program did not bind the companion")
				}
				*unit, *has = 0, false
			} else if ordinal == 2 {
				if !*has2 || *unit2 != x.companion {
					t.Fatal("fresh distance check did not bind the companion")
				}
				*unit2, *has2 = 0, false
			} else {
				t.Fatal("unexpected third unit reference")
			}
			count++
		}
	}
	for i, n := range src.Conditions {
		c := &checks[i]
		clearRef(n, &c.Unit, &c.Unit2, &c.HasUnit, &c.HasUnit2)
	}
	i := 0
	for _, n := range src.Actions {
		if n.Opcode >= 0x10002 {
			continue
		}
		in := &instants[i]
		clearRef(n, &in.Unit, &in.Unit2, &in.HasUnit, &in.HasUnit2)
		i++
	}
	if count != 3 {
		t.Fatalf("legacy mission130 fixture cleared%d references, want3", count)
	}
	legacy, err := sim.NewScript(checks, instants, correct.Triggers())
	if err != nil {
		t.Fatal(err)
	}
	form := betrayal130ProgramForm(t, w, legacy)
	if err := w.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	return correct
}

// The betrayal runs as four release tests, one per primary sex and branch,
// so the release gate can give each its own process. The three control
// cases run with the breeder-first branch.
func TestReleaseMission130CompanionBetrayal(t *testing.T) { betrayal130Release(t, false, false) }
func TestReleaseMission130CompanionBetrayalBreederFirst(t *testing.T) {
	betrayal130Release(t, false, true)
}
func TestReleaseMission130CompanionBetrayalFemale(t *testing.T) { betrayal130Release(t, true, false) }
func TestReleaseMission130CompanionBetrayalFemaleBreederFirst(t *testing.T) {
	betrayal130Release(t, true, true)
}

func betrayal130Release(t *testing.T, female, breederFirst bool) {
	variant := "male-primary"
	if female {
		variant = "female-primary"
	}
	t.Run(variant, func(t *testing.T) {
		runBetrayal130(t, female, breederFirst)
	})
}

func runBetrayal130(t *testing.T, female, breederFirstPart bool) {
	t.Helper()
	for _, breederFirst := range []bool{false, true} {
		if breederFirst != breederFirstPart {
			continue
		}
		branch := "secret-talk"
		if breederFirst {
			branch = "breeder-first"
		}
		t.Run(branch, func(t *testing.T) {
			x := startBetrayal130(t, female)
			x.prerequisite(t, breederFirst)
			x.cold(t, "before transfer", nil)
			x.transfer(t, breederFirst)
			x.cold(t, "after transfer", nil)
			x.hostile(t, breederFirst)
			if err := x.f.live.world.HeadlessKill(x.companion); err != nil {
				t.Fatal(err)
			}
			x.drive(t, 32)
			x.running(t, "enemy companion death")
			x.cold(t, "dead enemy body", nil)
			x.running(t, "cold dead enemy body")
			// Advance the actual corpse ladder; no direct removal or health
			// write skips the late missing-entity guard. At -10 the native
			// ladder needs under20k subticks to pass its final -600 floor.
			start := x.f.live.world.Tick()
			for x.f.live.world.Tick()-start < 20000 {
				if _, exists := x.f.live.entity(x.companion); !exists {
					break
				}
				x.drive(t, 128)
				x.running(t, "enemy corpse decay")
			}
			if _, exists := x.f.live.entity(x.companion); exists {
				t.Fatal("enemy body did not finish its bounded decay")
			}
			t.Logf("enemy removed after%d subticks", x.f.live.world.Tick()-start)
			x.cold(t, "enemy removed", nil)
			if !female && !breederFirst {
				var companionAfterLoad sim.EntityID
				foundCompanion := false
				for i, member := range x.f.live.mission.party {
					if member.CompanionNPC == 22 && i < len(x.f.live.mission.ids) {
						companionAfterLoad, foundCompanion = x.f.live.mission.ids[i], true
						break
					}
				}
				if !foundCompanion || !x.f.live.mission.departed[companionAfterLoad] {
					t.Fatalf("cold SAV lost NPC22's departed identity: id=%d found=%v departed=%v", companionAfterLoad,
						foundCompanion, x.f.live.mission.departed[companionAfterLoad])
				}
			}
			if !female && !breederFirst {
				x.f, x.app = requireGeneratedTombstonesSAV(t, x.f, x.app)
			}
			x.drive(t, 32)
			x.running(t, "cold removed enemy")
			x.requireGuardLoss(t, x.hero)
		})
		t.Run(branch+"/legacy-AGS", func(t *testing.T) {
			x := startBetrayal130(t, female)
			correct := x.legacyProgram(t)
			x.prerequisite(t, breederFirst)
			// The old writer has already run: variable50, counter93, the
			// Start latch and (in breeder-first) a nonzero branch all survive.
			if !x.f.live.world.ScriptLatched(0) || x.f.live.world.Tick() == 0 {
				t.Fatal("legacy AGS has no runtime state to preserve")
			}
			want := betrayal130ProgramForm(t, x.f.live.world, correct)
			x.cold(t, "legacy unresolved program", want)
			x.transfer(t, breederFirst)
			x.hostile(t, breederFirst)
			if err := x.f.live.world.HeadlessKill(x.companion); err != nil {
				t.Fatal(err)
			}
			x.drive(t, 32)
			x.running(t, "repaired legacy betrayal")
		})
	}
	if !breederFirstPart {
		return
	}
	t.Run("active-companion", func(t *testing.T) {
		x := startBetrayal130(t, female)
		x.drive(t, 32)
		x.cold(t, "active companion", nil)
		x.drive(t, 32)
		x.cold(t, "active companion second cycle", nil)
		x.requireGuardLoss(t, x.companion)
	})
	t.Run("current-script-ordinary-position", func(t *testing.T) {
		x := startBetrayal130(t, female)
		x.prerequisite(t, false)
		_, _, raw := campaignSave(t, x.f, true)
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil {
			t.Fatal("missing exact actor bindings", err)
		}
		var object uint16
		for _, binding := range a.Bindings {
			if binding.ID == x.companion && !binding.Structure && !binding.Missing {
				object = binding.Object
			}
		}
		if object == 0 {
			t.Fatal("companion lacks ordinary actor binding")
		}
		policy, _, _ := sav.NativeActions(doc.State)
		veglud := x.entity(t, x.veglud)
		cell := uint16(veglud.Y<<8 | (veglud.X + 1))
		position, err := savedActorRaw(&doc.Objects[object-1], "Block12", 12)
		if err != nil {
			t.Fatal(err)
		}
		binary.LittleEndian.PutUint16(position[:2], cell)
		binary.LittleEndian.PutUint16(position[2:4], cell)
		if err := savedStructureSetRaw(&doc.Objects[object-1], "Block12", position); err != nil {
			t.Fatal(err)
		}
		raw, err = sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		unchanged, _, _ := sav.NativeActions(doc.State)
		if !bytes.Equal(policy, unchanged) {
			t.Fatal("ordinary position edit changed the current continuation")
		}
		store := SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(store.Dir, "position.sav"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		x.f, x.app = campaignCold(t, store)
		companion := x.entity(t, x.companion)
		if companion.X != veglud.X+1 || companion.Y != veglud.Y {
			t.Fatal("current party role restoration replaced the ordinary position", companion.X, companion.Y)
		}
		x.drive(t, 48)
		if x.entity(t, x.companion).Owner != 8 || x.f.live.world.ScriptRegister(50) != 2 || !x.f.live.world.ScriptLatched(9) {
			t.Fatal("ordinary position did not drive authored distance and Give Unit")
		}
		x.cold(t, "ordinary approach after transfer", nil)
		x.hostile(t, false)
	})
	t.Run("early-attack", func(t *testing.T) {
		x := startBetrayal130(t, female)
		x.drive(t, 32)
		x.running(t, "early attack initial state")
		h, v := x.entity(t, x.hero), x.entity(t, x.veglud)
		livePlaceAndWalk(t, x.f.live, x.hero, v.X+1, v.Y)
		// Queue an ordinary attack; combat itself changes diplomacy.
		x.f.live.strike(uint32(h.ID), uint32(v.ID))
		for i := 0; i < 12 && x.f.live.mission.outcome != sim.OutcomeLost; i++ {
			x.drive(t, 48)
		}
		_, lost := x.f.live.world.ScriptCounters()
		if lost != 1 || !x.f.live.world.ScriptLatched(11) || x.f.live.world.Outcome() != sim.OutcomeLost || x.f.live.mission.outcome != sim.OutcomeLost {
			t.Fatalf("early attack did not retain authored defeat: lost=%d latch11=%v world=%d client=%d", lost,
				x.f.live.world.ScriptLatched(11), x.f.live.world.Outcome(), x.f.live.mission.outcome)
		}
		if x.f.live.guardedCharacterLost() {
			t.Fatal("character death contaminated the scripted early-attack control")
		}
		x.cold(t, "authored early-attack defeat", nil)
		if x.f.live.mission.outcome != sim.OutcomeLost {
			t.Fatal("cold SAV cleared authored defeat")
		}
	})
}

func TestRestoreCurrentDepartedCharactersUsesBoundIdentity(t *testing.T) {
	t.Run("reminted guarded identity", func(t *testing.T) {
		mw := &mapWorld{mission: &missionNotices{guarded: []sim.EntityID{17}}}
		if err := restoreCurrentDepartedCharacters(mw, []uint32{116}, map[sim.EntityID]sim.EntityID{116: 17}); err != nil {
			t.Fatal(err)
		}
		if !mw.mission.departed[17] || mw.mission.departed[116] {
			t.Fatalf("departed marker was not attached to the bound guarded identity: %v", mw.mission.departed)
		}
	})

	t.Run("unbound identity", func(t *testing.T) {
		mw := &mapWorld{mission: &missionNotices{guarded: []sim.EntityID{17}}}
		if err := restoreCurrentDepartedCharacters(mw, []uint32{116}, nil); err == nil {
			t.Fatal("accepted a departed identity with no actor binding")
		}
	})

	t.Run("unguarded identity", func(t *testing.T) {
		mw := &mapWorld{mission: &missionNotices{guarded: []sim.EntityID{17}}}
		if err := restoreCurrentDepartedCharacters(mw, []uint32{116}, map[sim.EntityID]sim.EntityID{116: 18}); err == nil {
			t.Fatal("accepted a departed identity that is not guarded")
		}
	})

	t.Run("duplicate identity", func(t *testing.T) {
		if err := validateCurrentDepartedCharacters([]uint32{116, 116}); err == nil {
			t.Fatal("accepted a repeated departed identity")
		}
	})

	t.Run("malformed list does not partially restore", func(t *testing.T) {
		mw := &mapWorld{mission: &missionNotices{guarded: []sim.EntityID{17}}}
		identities := map[sim.EntityID]sim.EntityID{116: 17, 117: 18}
		if err := restoreCurrentDepartedCharacters(mw, []uint32{116, 117}, identities); err == nil {
			t.Fatal("accepted a list whose second mapped identity is not guarded")
		}
		if len(mw.mission.departed) != 0 {
			t.Fatalf("malformed list partially restored departed state: %v", mw.mission.departed)
		}
	})
}
