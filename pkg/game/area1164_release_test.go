package game

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Installed App proof starts with untouched original bytes, then explicitly
// changes one private Light template to Poison. The original has no Poison
// area: this endpoint is controlled, not claimed as an original observation.
func TestReleaseAreaDamageLifecycle1164(t *testing.T) {
	defer area1164FreshCasts(t)
	_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
	f := releaseFront(t)
	_, _, _ = openWorldEffectsTestSave(t, f, raw, "unchanged-light.sav")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := spell1152Expected(source)
	if err != nil {
		t.Fatal(err)
	}
	if d := light1162Differences(f.live.world, f.live.mission.state.savedDocument, graph, 0); len(d) != 0 {
		t.Fatal("untouched original", d)
	}
	var target sim.EntityID
	for _, e := range f.live.world.ActiveEffects() {
		if e.Spell == 12 && e.Remaining == 12 {
			target = e.Target
		}
	}
	if target == 0 {
		t.Fatal("authentic Light actor not found")
	}
	area1164SourceCast(t, raw, target)

	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Isolate Poison HP from ordinary regeneration, automatic Heal (zero mana) and the pre-existing
	// Light expiry derive: these private endpoint changes are explicit.
	var actorIndex uint16
	for _, row := range f.live.mission.state.savedDocument.Actors {
		if row.EntityID == target {
			actorIndex = row.ObjectIndex
		}
	}
	if actorIndex == 0 {
		t.Fatal("target Document binding missing")
	}
	actor := &doc.Objects[actorIndex-1]
	savedObjectSetValue(actor, "HealthRegen", 32767)
	savedObjectSetValue(actor, "Mana", 0)
	savedObjectSetValue(actor, "ManaRegen", 32767)
	priorRefs, _ := savedObjectRefs(actor, "Effects")
	for _, ref := range priorRefs {
		e := &doc.Objects[ref-1]
		spell, _ := savedStructureValue(e, "E0C")
		if spell == 12 {
			value, _ := savedStructureValue(e, "E40")
			savedObjectSetValue(e, "E40", value&65535|uint32(9601)<<16)
		}
	}
	root := &doc.Objects[doc.World.Effects[0]-1]
	identity, _ := savedStructureValue(root, "Identity")
	savedObjectSetValue(root, "T0C", 8)
	savedObjectSetValue(root, "AE4C", 17)
	refs, _ := savedObjectRefs(root, "AE44")
	child := &doc.Objects[refs[0]-1]
	for name, value := range map[string]uint32{"T0C": 8, "E0C": 8, "E3C": 6, "E3D": 2, "E40": 0x0080fffc} {
		savedObjectSetValue(child, name, value)
	}
	for i := range doc.World.Cells {
		c := &doc.World.Cells[i]
		if c.Layers[4] == identity {
			c.Layers[2], c.Layers[4] = identity, 0
		}
	}
	if releaseEntity(t, f.live, target).Protection[1] != 21 {
		t.Fatal("authentic target water resistance changed")
	}
	controlled, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	f = releaseFront(t)
	app, store, path := openWorldEffectsTestSave(t, f, controlled, "controlled-poison.sav")
	before := releaseEntity(t, f.live, target)
	if before.ActorLoad.Source.Class == 0 {
		t.Fatal("controlled target is not source-bound")
	}
	if err := f.live.world.HeadlessPlace(target, 19, 40); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	e := releaseEntity(t, f.live, target)
	if e.HP != before.HP-6 {
		t.Fatalf("first retained Poison plus same-tick actor phase: HP%d want%d (water%d)", e.HP, before.HP-6, e.Protection[1])
	}
	check := func(front *FrontEnd, wantHP int32, wantRemaining uint16) {
		t.Helper()
		got := releaseEntity(t, front.live, target)
		if got.HP != wantHP {
			t.Fatal("source current HP", got.HP, wantHP)
		}
		found := false
		for _, a := range front.live.world.ActiveEffects() {
			if a.Target == target && a.Spell == 8 {
				found = true
				if a.Magnitude != -4 || a.Remaining != wantRemaining || a.HasCaster {
					t.Fatal("retained Poison payload/phase/source", a)
				}
			}
		}
		if !found {
			t.Fatal("Poison attachment lost")
		}
		snapshot := producer1162Snapshot(t, front)
		if d := cellProducer1162Differences(front.live.world, snapshot.SavedDocument); len(d) != 0 {
			t.Fatal("current cells/Document", d)
		}
		if snapshot.SavedDocument.ActorEffects == nil || snapshot.SavedDocument.ActorEffects.Unavailable != "" {
			t.Fatal("current caster-free Poison attachment was not projected")
		}
		bindings := 0
		for _, row := range snapshot.SavedDocument.ActorEffects.Rows {
			if row.Entity != target || row.Spell != 8 || row.ObjectIndex == 0 {
				continue
			}
			bindings++
			value, err := savedEffectRecord(&snapshot.SavedDocument.Document.Objects[row.ObjectIndex-1])
			if err != nil || value.E0C != 8 || value.Value.Kind != 6 || value.Value.Mode != 2 || value.Value.Operand != uint32(0xfffc)|uint32(wantRemaining)<<16 {
				t.Fatalf("current Poison raw fields = %+v: %v", value, err)
			}
		}
		if bindings != 1 {
			t.Fatalf("current Poison owner bindings = %d, want one", bindings)
		}
		for _, row := range snapshot.SavedDocument.Actors {
			if row.EntityID != target {
				continue
			}
			record := &snapshot.SavedDocument.Document.Objects[row.ObjectIndex-1]
			health, err := savedStructureValue(record, "Health")
			if err != nil || health != uint32(uint16(wantHP)) {
				t.Fatal("independent current Document HP", health, wantHP, err)
			}
			bad, err := cloneSavedDocument(snapshot.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			savedObjectSetValue(&bad.Document.Objects[row.ObjectIndex-1], "Health", uint32(uint16(wantHP+1)))
			// This independent oracle detects a stale projector. General
			// scalar native admission is not claimed: World HP remains the
			// native authority even when historical DTO Health is stale.
			stale, _ := savedStructureValue(&bad.Document.Objects[row.ObjectIndex-1], "Health")
			if stale == uint32(uint16(wantHP)) {
				t.Fatal("stale HP escaped direct Document oracle")
			}
		}
	}
	check(f, before.HP-6, 127)
	fresh := producer1162Fresh(t, f, app, store, path)
	check(fresh, before.HP-6, 127)
	for tick := 2; tick <= 18; tick++ {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("source-free Poison continuation", tick)
		}
	}
	// Pulses at old128/120, then final cloud zero refresh128 before actor phase.
	check(f, before.HP-12, 126)
	check(fresh, before.HP-12, 126)
	if len(f.live.world.SavedSpellEffects()) != 0 || f.live.world.SavedWorldEffectDrivers() != nil {
		t.Fatal("retained Poison root/layers did not retire")
	}
	for _, c := range f.live.world.SavedCellRecords() {
		if slices.Contains(c.SpellEffects[:], identity) {
			t.Fatal("retired pointer remains")
		}
	}
	t.Log("controlled retained Poison: signed water21 gives damage3; first -6/127, old-counter pulses, final-zero refresh, cleanup without attachment revocation; current World/Document and ordinary menu SAVE/source removal/fresh LOAD")
}

// A new cast from an original-loaded source actor must keep native ownership
// and current raw Cells consistent without inventing an original area address.
func area1164SourceCast(t *testing.T, raw []byte, caster sim.EntityID) {
	t.Helper()
	t.Run("original-source-new-area-document-boundary", func(t *testing.T) {
		f := releaseFront(t)
		app, store, path := openWorldEffectsTestSave(t, f, raw, "source-new-light.sav")
		if err := f.live.world.HeadlessPlace(caster, 19, 40); err != nil {
			t.Fatal(err)
		}
		f.live.attackOrCast(uint32(caster), 0, 12, 19, 40, true)
		if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindCastAt {
			t.Fatal("new Light input not queued")
		}
		for tick := 0; !f.live.world.HasNativeAreaEffects(); tick++ {
			if tick > 128 {
				t.Fatal("source actor did not cast Light", f.live.world.BookSpellCellRefusal(caster, 19, 40, 12))
			}
			f.live.tick()
		}
		snapshot := producer1162Snapshot(t, f)
		if snapshot.SavedDocument.WorldEffects == nil || snapshot.SavedDocument.WorldEffects.Unavailable != "" {
			t.Fatal("current area constructor was refused")
		}
		if d := cellProducer1162Differences(f.live.world, snapshot.SavedDocument); len(d) != 0 {
			t.Fatal("new native layer current raw cells/planes disagree", d)
		}
		cold := producer1162Fresh(t, f, app, store, path)
		for range 20 {
			f.live.tick()
			cold.live.tick()
			if f.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("new area from original source changed after native LOAD")
			}
		}
		t.Log("source-bound MapAttack Light: current raw Cells/planes and explicit AGS SAVE/source removal/fresh LOAD/20 steps agree")
	})
}

func area1164NativeFresh(t *testing.T, f *FrontEnd, app *ui.App) *FrontEnd {
	return area1164NativeFreshWith(t, f, app, releaseFront)
}

func area1164NativeFreshWith(t *testing.T, f *FrontEnd, app *ui.App, newFront func(*testing.T) *FrontEnd) *FrontEnd {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	hash := f.live.world.Hash()
	mover1160Menu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		s, label, snapshotErr := f.Snapshot(true)
		_, encodeErr := EncodeSave(s, label)
		t.Fatal("native menu SAVE", entries, err, snapshotErr, encodeErr, app.HeadlessMessage())
	}
	fresh := newFront(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Area native continuation")
	app2.Layout(1024, 768)
	save2, list2, load2 := agsSaveSeams(fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save2, list2, load2)
	groundAppLoad(t, app2, list2, entries[0].Name)
	if f.live.world.Hash() != hash || fresh.live.world.Hash() != hash {
		currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
		t.Fatal("fresh native LOAD changed current world")
	}
	return fresh
}

func area1164FreshCasts(t *testing.T) {
	t.Helper()
	for _, tc := range []struct {
		name            string
		spell, duration uint16
	}{{"fresh-fire", 3, 0}, {"fresh-poison", 8, 0}, {"signed-poison-zero-crossing", 8, 0}} {
		spell := tc.spell
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
			hero.Skill[1], hero.Skill[2] = 100, 100
			party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
				Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << spell,
				Saved: &mapload.Saved{Cell: mapload.Cell{X: 53, Y: 54}, HP: 1000, MaxHP: 1000, Mana: 1000, MaxMana: 1000}},
				{ID: "companion", Profile: data.Profile{HealthColumn: true}, Hero: data.Hero{Body: 60, Reaction: 60},
					Saved: &mapload.Saved{Cell: mapload.Cell{X: 54, Y: 55}, HP: 1000, MaxHP: 1000}}}
			app := f.App("Area damage lifecycle")
			app.Layout(1024, 768)
			if err := app.OpenMission(f.MissionOpenerWith(101, party)); err != nil {
				t.Fatal(err)
			}
			caster, target := f.live.mission.ids[0], f.live.mission.ids[1]
			victim := releaseEntity(t, f.live, target)
			rule, ok := f.live.world.Spell(uint32(spell))
			if !ok {
				t.Fatal("installed spell missing")
			}
			if spell == 8 && (rule.EffectDuration != 128 || rule.EffectMagnitude != -2) {
				t.Fatal("installed Poison duration/magnitude producer", rule.EffectDuration, rule.EffectMagnitude)
			}
			if tc.name == "signed-poison-zero-crossing" {
				area1164SignedCloudApp(t, f, app, caster, target)
				return
			}
			initialMana := releaseEntity(t, f.live, caster).Mana
			owned := func(front *FrontEnd) []sim.CellEffect {
				var out []sim.CellEffect
				for _, e := range front.live.world.CellEffects() {
					if e.Spell == spell && e.HasCaster && e.Caster == caster {
						out = append(out, e)
					}
				}
				return out
			}
			// Both calls are the MapAttack callback installed on the active mission.
			for n := 0; n < 2; n++ {
				for tick := 0; releaseEntity(t, f.live, caster).CastWait != 0; tick++ {
					if tick > 256 {
						t.Fatal("recovery stuck")
					}
					f.live.tick()
				}
				f.live.attackOrCast(uint32(caster), 0, uint32(spell), int(victim.X), int(victim.Y), true)
				if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindCastAt {
					t.Fatal("map input did not queue point cast")
				}
				for tick := 0; len(owned(f)) < n+1; tick++ {
					if tick > 256 {
						t.Fatal("cast did not land", f.live.world.BookSpellCellRefusal(caster, victim.X, victim.Y, uint32(spell)))
					}
					f.live.tick()
				}
			}
			effects := owned(f)
			if len(effects) != 2 || len(effects[0].Cells) != 0 || len(effects[1].Cells) == 0 {
				t.Fatal("new area overwrote independent clock instead of layer", effects)
			}
			if releaseEntity(t, f.live, caster).Mana >= initialMana {
				t.Fatal("two casts did not spend mana")
			}
			cold := area1164NativeFresh(t, f, app)
			if !slices.EqualFunc(owned(cold), effects, func(a, b sim.CellEffect) bool {
				return a.Spell == b.Spell && a.Remaining == b.Remaining && slices.Equal(a.Cells, b.Cells)
			}) {
				t.Fatal("native owner/counter cut differs")
			}
			for tick := 0; len(owned(f)) != 0; tick++ {
				if tick > 1600 {
					t.Fatal("areas did not expire")
				}
				f.live.tick()
				cold.live.tick()
				if f.live.world.Hash() != cold.live.world.Hash() {
					t.Fatal("cast lifecycle diverged after source-free native LOAD", tick)
				}
			}
			if got := releaseEntity(t, f.live, target).HP; got >= victim.HP {
				t.Fatal("installed area did no damage", got, victim.HP)
			}
			// Save once more after both independent objects disappear, while a Poison
			// attachment may still be present; next ordinary input must remain viable.
			resumed := area1164NativeFresh(t, f, app)
			// The caster's spell power now exceeds 100, so its own Poison cloud may
			// fell it; the next ordinary wound goes to a living entity instead.
			wounded := caster
			for _, e := range f.live.world.Entities() {
				if e.HP > 0 {
					wounded = e.ID
					break
				}
			}
			for _, w := range []*sim.World{f.live.world, resumed.live.world} {
				headlessDamage(t, w, wounded, 1)
				sim.Step(w, nil)
			}
			if f.live.world.Hash() != resumed.live.world.Hash() {
				t.Fatal("post-expiry native action differs")
			}
			t.Logf("installed spell%d: two MapAttack casts spent mana, independent clocks/current newest paint; overlap and post-expiry ordinary menu SAVE/fresh source-free LOAD/next damage, targetHP%d to%d", spell, victim.HP, releaseEntity(t, f.live, target).HP)
		})
	}
}

func area1164SignedCloudApp(t *testing.T, f *FrontEnd, app *ui.App, caster, target sim.EntityID) {
	t.Helper()
	// Controlled current native inputs, not a claim that stock equipment can
	// produce protection150. SetCombat clamps at100; the supported byte form
	// retains signed values. Alter only defence and water protection in the
	// independently spelled form92 actor record, then use ordinary admission.
	b, err := f.live.world.MarshalBinary()
	if err != nil || beforeAutoHealing1191(t, b)[0] != 95 {
		t.Fatal("controlled native form", err)
	}
	base := 34 + 3*int(binary.LittleEndian.Uint32(b[30:]))
	for i, e := range f.live.world.Entities() {
		if e.ID == target {
			o := base + 492*i
			if binary.LittleEndian.Uint32(b[o:]) != uint32(target) {
				t.Fatal("independent actor offset")
			}
			binary.LittleEndian.PutUint32(b[o+66:], 40)
			binary.LittleEndian.PutUint32(b[o+234:], 150)
		}
	}
	if err := f.live.world.UnmarshalBinary(b); err != nil {
		t.Fatal("controlled signed native inputs refused", err)
	}
	victim := releaseEntity(t, f.live, target)
	f.live.attackOrCast(uint32(caster), 0, 8, int(victim.X), int(victim.Y), true)
	effect := func(front *FrontEnd) sim.ActiveEffect {
		for _, e := range front.live.world.ActiveEffects() {
			if e.Spell == 8 && e.Target == target {
				return e
			}
		}
		return sim.ActiveEffect{}
	}
	for tick := 0; effect(f).Remaining == 0; tick++ {
		if tick > 300 {
			t.Fatal("MapAttack cloud did not attach Poison")
		}
		f.live.tick()
	}
	attached := effect(f)
	damage := (int64(-attached.Magnitude)*(-50) + 50) / 100
	if attached.Remaining != 127 || damage >= -1 {
		t.Fatal("installed signed Poison first pulse", attached, damage)
	}
	headlessDamage(t, f.live.world, target, releaseEntity(t, f.live, target).HP+1)
	sim.Step(f.live.world, nil)
	if e := releaseEntity(t, f.live, target); e.HP != -1 || e.Decay != sim.DecayFallen || e.Defence != 20 || effect(f).Remaining != 126 {
		t.Fatal("ordinary damage must leave a loadable fallen cut", e, effect(f))
	}
	cold := area1164NativeFresh(t, f, app)
	for range 7 {
		sim.Step(f.live.world, nil)
		sim.Step(cold.live.world, nil)
	}
	for _, front := range []*FrontEnd{f, cold} {
		e := releaseEntity(t, front.live, target)
		if e.HP != -1-int32(damage) || e.Decay != sim.DecayNone || e.Dwell != 0 || e.Defence != 40 || effect(front).Remaining != 119 {
			t.Fatal("later signed pulse must clear native decay once", e, effect(front), damage)
		}
	}
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("fallen-cut native continuation differs")
	}
	cold = area1164NativeFresh(t, f, app)
	for _, front := range []*FrontEnd{f, cold} {
		headlessDamage(t, front.live.world, target, 1)
		sim.Step(front.live.world, nil)
	}
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("healed-cut next ordinary action differs")
	}
	t.Logf("installed MapAttack Poison magnitude%d/protection150/computed damage%d: ordinary damage leavesHP-1; menu SAVE/source-free LOAD; seven steps cross toHP%d with defence40/DecayNone; second menu SAVE/fresh LOAD/next damage1", attached.Magnitude, damage, -1-int32(damage))
}
