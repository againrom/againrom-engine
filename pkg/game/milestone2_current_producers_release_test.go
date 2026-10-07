package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// P1/P2/P3 in the finite milestone-2 map concern existing current-value
// producers. Every subcase first checks an untouched installed original LOAD.
// Controlled endpoints below are native test setup, never original observations.
func producerSAVLoad(t *testing.T) (*FrontEnd, *ui.App, string, *sav.File) {
	t.Helper()
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	scalars, err := unit1156Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	combat, err := unit1158Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, path := openOriginalSAVApp(t, f, raw, "game1162.sav")
	unit1156InitialCheck(t, scalars, f.live.mission.state)
	unit1158InitialCheck(t, combat, f.live.mission.state)
	planes, cells, _, _ := originalCellPlaneReference1115(t, f, source)
	originalCellPlanesCheck1115(t, "untouched LOAD", f.live.world, planes)
	originalCellNodesCheck1115(t, "untouched LOAD", f.live.world, cells)
	wants, links, err := buildings1145Expected(source)
	if err != nil {
		t.Fatal(err)
	}
	structures, structureCells, present := f.live.world.SavedStructures()
	if d := buildings1145Differences(wants, links, f.live.world.Structures(), structures, structureCells, present); len(d) != 0 {
		t.Fatal(d)
	}
	// Own literal Session offsets, identical to the independent acceptance
	// instrument, without SessionState() or snapshotSavedDocument as oracle.
	b := source.Body[source.World.SessionOff:]
	w, doc := f.live.world, f.live.mission.state.savedDocument.Document.World.Session
	won, lost := w.ScriptCounters()
	if won != binary.LittleEndian.Uint32(b[4362:]) || lost != binary.LittleEndian.Uint32(b[4370:]) || doc.Won != won || doc.Lost != lost {
		t.Fatal("untouched Session counters")
	}
	for i := 0; i < 1000; i++ {
		if w.ScriptLatched(int32(i)) != (b[400+i] != 0) || doc.Latches[i] != b[400+i] {
			t.Fatalf("untouched latch%d", i)
		}
	}
	for from := 0; from < 50; from++ {
		for to := 0; to < 50; to++ {
			v := b[1856+50*from+to]
			if w.Relations().Byte(uint32(from), uint32(to)) != v || doc.Diplomacy[from][to] != v {
				t.Fatal("untouched diplomacy", from, to)
			}
		}
	}
	return f, app, path, source
}

func installTestScript(t *testing.T, f *FrontEnd, script *sim.Script) {
	t.Helper()
	w, err := sim.NewControlledScriptWorld(f.live.world, script)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(w)
	*f.live.world = *w
}

func producer1162Snapshot(t *testing.T, f *FrontEnd) Snapshot {
	t.Helper()
	hash := f.live.world.Hash()
	s, _, err := f.Snapshot(true)
	if err != nil || f.live.world.Hash() != hash {
		t.Fatal("current Snapshot", err)
	}
	return s
}

func producer1162Fresh(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, path string) *FrontEnd {
	t.Helper()
	hash := f.live.world.Hash()
	_, nativeList, nativeLoad := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(nativeCheckpoint1170(f, store), nativeList, nativeLoad)
	t.Log("App menu with explicit native checkpoint codec")
	mover1160Menu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatal("ordinary menu SAVE", entries, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("private source remains", err)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Current producers source-free LOAD")
	app2.Layout(1024, 768)
	save, list, load := agsSaveSeams(fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, entries[0].Name)
	if fresh.live.world.Hash() != hash || f.live.world.Hash() != hash {
		t.Fatal("SAVE/fresh LOAD changed World")
	}
	return fresh
}

func producerSAVFresh(t *testing.T, f *FrontEnd, app *ui.App, path string) (*FrontEnd, []byte) {
	t.Helper()
	hash := f.live.world.Hash()
	mover1160Menu(t, app)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: filepath.Dir(path)}, func() time.Time { return time.Unix(1, 0) })
	app.SetSaveSeams(save, list, load)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatal("ordinary menu SAVE", entries, err, app.HeadlessMessage())
	}
	if f.live.world.Hash() != hash {
		t.Fatal("SAVE changed the live World")
	}
	written, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	return loadLocalLegacySave(t, store, entries[0].Name), written
}

func sessionProducer1162Differences(w *sim.World, state *SnapshotSAVDocument, diplomacy [50][50]byte) []string {
	var d []string
	add := func(ok bool, name string) {
		if !ok {
			d = append(d, name)
		}
	}
	won, lost := w.ScriptCounters()
	add(won == 2 && lost == 3, "World counters2/3")
	if state == nil || state.Document == nil {
		return append(d, "missing Document")
	}
	s := state.Document.World.Session
	add(s.Won == 2 && s.Lost == 3, "Document counters2/3")
	for i := 0; i < 1000; i++ {
		want := i == 17 || i == 918
		add(w.ScriptLatched(int32(i)) == want, fmt.Sprintf("World latch%d", i))
		add(s.Latches[i] == map[bool]byte{false: 0, true: 1}[want], fmt.Sprintf("Document latch%d", i))
	}
	for from := 0; from < 50; from++ {
		for to := 0; to < 50; to++ {
			add(w.Relations().Byte(uint32(from), uint32(to)) == diplomacy[from][to], "World diplomacy")
			add(s.Diplomacy[from][to] == diplomacy[from][to], "Document diplomacy")
		}
	}
	return d
}

func producerEntity(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("missing bound actor%d", id)
	return sim.Entity{}
}

// The current basis belongs to World. Neither SourceNow, savedActorValueRecord,
// a second Snapshot, nor a re-import of the projected Document supplies this
// oracle. Initial raw-byte checks already establish the field mapping; this
// joins each named current owner to its independent Document field at a cut.
func actorProducer1162Differences(w *sim.World, state *SnapshotSAVDocument) []string {
	var diff []string
	if state == nil || state.Document == nil {
		return []string{"missing actor Document"}
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		entities[e.ID] = e
	}
	seenActors, seenObjects := map[sim.EntityID]bool{}, map[uint16]bool{}
	for _, binding := range state.Actors {
		if binding.Retired {
			continue
		}
		e, ok := entities[binding.EntityID]
		if !ok || seenActors[binding.EntityID] || seenObjects[binding.ObjectIndex] || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(state.Document.Objects) {
			diff = append(diff, "actor binding")
			continue
		}
		seenActors[binding.EntityID], seenObjects[binding.ObjectIndex] = true, true
		r := state.Document.Objects[binding.ObjectIndex-1]
		s := e.ActorLoad.Source
		values := map[string]uint32{}
		for _, v := range r.Values {
			values[v.Name] = v.Value
		}
		want := map[string]uint32{
			"Body": uint32(s.Stats[0]), "Reaction": uint32(s.Stats[1]), "Mind": uint32(s.Stats[2]), "Spirit": uint32(s.Stats[3]),
			"Speed": uint32(uint16(e.HumanMovement.RawSpeed)), "U8E": uint32(uint16(e.ActorLoad.OwnWeight)), "U90": uint32(uint16(e.Load)), "Capacity": uint32(uint16(e.Capacity)),
			"Health": uint32(uint16(e.HP)), "HealthMax": uint32(uint16(e.MaxHP)), "Mana": uint32(uint16(e.Mana)), "ManaMax": uint32(uint16(e.MaxMana)),
			"HealthRegen": uint32(s.Stats[10]), "ManaRegen": uint32(s.Stats[13]), "UA2": uint32(e.HealthHundredths), "UA3": uint32(e.ManaHundredths),
			"UA0": uint32(s.ManaFloor), "UA4": uint32(s.Sight), "U130": s.Experience,
		}
		for name, v := range want {
			got, present := values[name]
			if !present || got != v {
				diff = append(diff, fmt.Sprintf("actor%d %s=%d want%d", e.ID, name, got, v))
			}
		}
		defence := s.Defence
		binary.LittleEndian.PutUint16(defence[:], uint16(e.Defence))
		binary.LittleEndian.PutUint16(defence[2:], uint16(e.Absorption))
		var xp [24]byte
		for i, v := range e.SkillXP {
			binary.LittleEndian.PutUint32(xp[4*i:], uint32(v))
		}
		blocks := map[string][]byte{"UA6": s.Attack[:], "UBE": defence[:], "U114": s.Base[:], "UD4": s.Modifier[:]}
		if r.Class != "Unit" {
			blocks["H1CC"] = xp[:]
		}
		for name, want := range blocks {
			found := false
			for _, block := range r.Raw {
				if block.Name == name {
					found = true
					if !reflect.DeepEqual(block.Bytes, want) {
						diff = append(diff, fmt.Sprintf("actor%d %s", e.ID, name))
					}
				}
			}
			if !found {
				diff = append(diff, "missing block "+name)
			}
		}
	}
	for id, e := range entities {
		if e.SourceBinding.Class != 0 && !seenActors[id] {
			diff = append(diff, fmt.Sprintf("lost current actor%d", id))
		}
	}
	return diff
}

func producerBasis(t *testing.T, w *sim.World, id sim.EntityID, building bool) {
	t.Helper()
	e := producerEntity(t, w, id)
	load := e.CurrentActorLoad()
	if load == nil {
		t.Fatal("controlled source basis absent")
	}
	s := &load.Inventory.Source
	// Explicit native endpoint: retain all identities/children/container state,
	// but give this actor distinct current pools/blocks and a fighter XP slot.
	// This is not a claim that the original mage changes class during play.
	s.Fighter = true
	s.Stats[2], s.Stats[8], s.Stats[9], s.Stats[10], s.Stats[11], s.Stats[12], s.Stats[13] = 30, 37, 123, 1000, 1, 97, 2000
	// The authentic book graph remains intact. One mana point cannot pay the
	// installed Heal cost5; it must not mix with this physical/regen witness.
	s.Attack[14], s.Attack[15], s.Attack[16], s.Attack[17], s.Attack[18], s.Attack[19], s.Attack[20], s.Attack[21] = 5, 0, 3, 0, 0, 0, 0, 0
	binary.LittleEndian.PutUint16(s.Attack[:], 500)
	binary.LittleEndian.PutUint16(s.Defence[:], 7)
	binary.LittleEndian.PutUint16(s.Defence[2:], 2)
	binary.LittleEndian.PutUint16(s.Modifier[10:], 0)
	binary.LittleEndian.PutUint16(s.Modifier[14:], 0)
	for i, v := range []uint16{3, 4, 5, 0, 7, 8} {
		binary.LittleEndian.PutUint16(s.Base[2+2*i:], v)
	}
	s.SkillXP = [6]uint32{11, 22, 33, 0, 55, 66}
	s.Experience = 0x12340000
	s.EquipmentRuntimePresent, s.Reach, s.AttackCharge, s.AttackRelax = true, 1, 1, 2
	if building {
		s.Attack[19], s.Attack[20], s.Attack[21] = 20, 1, 1
	}
	if err := w.RestoreActorLoad(id, *load); err != nil {
		t.Fatal(err)
	}
	e = producerEntity(t, w, id)
	if err := w.ImportOriginalActorProfiles([]sim.OriginalActorProfile{{ID: id, Reaction: int16(e.Reaction), Mind: 30, Spirit: int16(e.Spirit), ToHit: 500, Defence: 7, Absorption: 2, DamageBase: 5, XPSlot: 3,
		SecondaryDamage: e.SecondaryDamage, HealthPeriod: 1000, ManaPeriod: 2000, HealthHundredths: 13, ManaHundredths: 27}}); err != nil {
		t.Fatal(err)
	}
}

func cellProducer1162Differences(w *sim.World, state *SnapshotSAVDocument) []string {
	if state == nil || state.Document == nil {
		return []string{"missing cell Document"}
	}
	p, present := w.SavedCellPlanes()
	_, cells, _, current := w.SavedActorMotions()
	if !present || !current {
		return []string{"missing current cell authority"}
	}
	// The literal scan is the same independent rule as cellStateCheck1115,
	// applied to the installed current planes instead of its synthetic map.
	var blocks []sav.BlockRecord
	for cell := 0x807; cell <= 0xeded; cell++ {
		if p.Dynamic[cell] > 15 {
			blocks = append(blocks, sav.BlockRecord{Cell: uint16(cell), Static: p.Static[cell], Dyn: p.Dynamic[cell]})
		}
	}
	var diff []string
	if !reflect.DeepEqual(blocks, state.Document.World.Blocks) {
		diff = append(diff, "current terrain block scan")
	}
	if len(cells) != len(state.Document.World.Cells) {
		diff = append(diff, "current cell population")
	}
	for i, cell := range cells {
		if i >= len(state.Document.World.Cells) {
			break
		}
		doc := state.Document.World.Cells[i]
		if cell.Cell != doc.Cell || cell.Payload != doc.Payload() {
			diff = append(diff, fmt.Sprintf("current52-byte cell%04x", cell.Cell))
		}
	}
	return diff
}

func buildingProducer1162Differences(w *sim.World, state *SnapshotSAVDocument, id sim.StructureID, hp uint16) []string {
	var diff []string
	sources, _, present := w.SavedStructures()
	if !present || state == nil || state.Document == nil {
		return []string{"missing structure authority"}
	}
	var key uint32
	for _, s := range sources {
		if s.ID == id {
			key = s.SourceKey
		}
	}
	if key == 0 {
		return []string{"missing structure identity"}
	}
	liveFound, docFound := false, false
	for _, s := range w.Structures() {
		if s.ID == id {
			liveFound = true
			if s.Field42 != hp {
				diff = append(diff, "live structure health")
			}
		}
	}
	for _, index := range state.Document.World.Buildings {
		r := state.Document.Objects[index-1]
		values := map[string]uint32{}
		for _, v := range r.Values {
			values[v.Name] = v.Value
		}
		if values["Identity"] == key {
			docFound = true
			if values["B42"] != uint32(hp) {
				diff = append(diff, "Document structure health")
			}
		}
	}
	if !liveFound || !docFound {
		diff = append(diff, "structure identity join")
	}
	return diff
}

func TestReleaseMilestone2CurrentProducers1162(t *testing.T) {
	t.Run("P1 Session", func(t *testing.T) {
		f, app, path, _ := producerSAVLoad(t)
		diplomacy := f.live.mission.state.savedDocument.Document.World.Session.Diplomacy
		old := diplomacy[49][48]
		addend := (old + 1) & 3
		diplomacy[49][48] = (old & 0xfc) + addend
		// Two one-shot triggers actually increment both counters and change one
		// directional relation. Counters2/3 deliberately leave outcome undecided.
		script, err := sim.NewScript(nil, []sim.ScriptInstant{
			{Op: sim.ScriptInstantWin}, {Op: sim.ScriptInstantLose},
			{Op: sim.ScriptInstantRelation, Args: [10]int32{49, 48, int32(addend)}},
		}, []sim.ScriptTrigger{
			{Instants: [4]int32{0, 0, 1, 1}, Once: true, Latch: 17},
			{Instants: [4]int32{1, 2, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 918},
		})
		if err != nil {
			t.Fatal(err)
		}
		installTestScript(t, f, script)
		for range 16 {
			sim.Step(f.live.world, nil)
			if f.live.world.ScriptLatched(918) {
				break
			}
		}
		current := producer1162Snapshot(t, f)
		if d := sessionProducer1162Differences(f.live.world, current.SavedDocument, diplomacy); len(d) != 0 {
			t.Fatal(d)
		}
		for _, field := range []string{"Won", "Lost", "latch", "diplomacy"} {
			bad, err := cloneSavedDocument(current.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			s := &bad.Document.World.Session
			switch field {
			case "Won":
				s.Won = 0
			case "Lost":
				s.Lost = 0
			case "latch":
				s.Latches[918] = 0
			case "diplomacy":
				s.Diplomacy[49][48] = old
			}
			if d := sessionProducer1162Differences(f.live.world, bad, diplomacy); len(d) == 0 {
				t.Fatal("stale Session oracle control accepted", field)
			}
		}
		fresh, written := producerSAVFresh(t, f, app, path)
		if d := sessionProducer1162Differences(fresh.live.world, fresh.live.mission.state.savedDocument, diplomacy); len(d) != 0 {
			t.Fatal("fresh stored Document before re-projection", d)
		}
		if d := sessionProducer1162Differences(f.live.world, fresh.live.mission.state.savedDocument, diplomacy); len(d) != 0 {
			t.Fatal("written Document differs from changed live Session", d)
		}
		lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
			doc.World.Session.Won = 0
			return true
		})
		if d := sessionProducer1162Differences(f.live.world, lost.live.mission.state.savedDocument, diplomacy); len(d) == 0 {
			t.Fatal("loss control: a SAV with Won cleared still matches")
		}
		if err := fresh.live.world.RestoreScriptProgram(script); err != nil {
			t.Fatal(err)
		}
		for range 20 {
			sim.Step(f.live.world, nil)
			sim.Step(fresh.live.world, nil)
			for _, front := range []*FrontEnd{f, fresh} {
				if d := sessionProducer1162Differences(front.live.world, producer1162Snapshot(t, front).SavedDocument, diplomacy); len(d) != 0 {
					t.Fatal(d)
				}
			}
		}
		t.Log("literal counters0/0->2/3, latches17/918, directional49->48; stale controls4; changed menu SAVE/source-free LOAD/next20 ticks")
	})
	t.Run("P2 Unit", func(t *testing.T) {
		f, app, path, _ := producerSAVLoad(t)
		// The retained source relation can be locked; the ordinary script
		// instant makes this controlled opponent hostile before the first blow.
		quiet, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantRelation, Args: [10]int32{1, 5, 1}}}, []sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 917}})
		if err != nil {
			t.Fatal(err)
		}
		installTestScript(t, f, quiet)
		const actor, target sim.EntityID = 35, 3
		producerBasis(t, f.live.world, actor, false)
		if clock := producerEntity(t, f.live.world, actor).ActionClock; clock != (sim.ActionClock{Known: true, End: 0}) {
			t.Fatal("source Fergard U138 deadline", clock)
		}
		if err := f.live.world.HeadlessPlace(actor, 20, 20); err != nil {
			t.Fatal(err)
		}
		// Damage is the script's health write; the restored fractional pool consumer
		// then runs at its ordinary full-tick phase, with literal non-round divisors.
		headlessDamage(t, f.live.world, actor, 7)
		sim.Step(f.live.world, nil)
		if e := producerEntity(t, f.live.world, actor); e.HP != 30 {
			t.Fatal("damage37->30", e.HP)
		}
		for f.live.world.Tick() < 397 {
			sim.Step(f.live.world, nil)
		}
		e := producerEntity(t, f.live.world, actor)
		// Source U138=0 selects the idle rate3 (SAV-REGENORDER-531).
		// HP:13+trunc(123*2*100*3/1000)=86; MP:27+2*trunc(97*100*3/2000)=55.
		if e.HP != 30 || e.HealthHundredths != 86 || e.Mana != 1 || e.ManaHundredths != 55 {
			t.Fatalf("literal first health/two mana passes: HP%d.%d MP%d.%d", e.HP, e.HealthHundredths, e.Mana, e.ManaHundredths)
		}
		regen := producer1162Snapshot(t, f)
		if d := actorProducer1162Differences(f.live.world, regen.SavedDocument); len(d) != 0 {
			t.Fatal("changed regeneration cut", d)
		}
		// A native mage weapon diverts its physical release into a spell. End
		// the controlled pool subcase with a fighter's zero mana maximum before
		// the separate physical-XP action. The installed weapon rider still runs
		// after the physical component: its prepared child now lands separately.
		// This controlled first attack removes5 physical HP and then7 rider HP.
		if err := f.live.world.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: actor, HP: 30, MaxHP: 123}}); err != nil {
			t.Fatal(err)
		}
		if err := f.live.world.HeadlessPlace(target, 21, 20); err != nil {
			t.Fatal(err)
		}
		if err := f.live.world.ImportOriginalActorProfiles([]sim.OriginalActorProfile{{ID: target, HealthPeriod: 1000, ManaPeriod: 1}}); err != nil {
			t.Fatal(err)
		}
		if err := f.live.world.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: target, HP: 50, MaxHP: 50}}); err != nil {
			t.Fatal(err)
		}
		beforeTarget := producerEntity(t, f.live.world, target)
		sim.Step(f.live.world, []sim.Command{{Kind: sim.KindAttack, Entity: actor, X: int32(target)}})
		for n := 0; n < 64 && producerEntity(t, f.live.world, target).HP == 50; n++ {
			sim.Step(f.live.world, nil)
		}
		e = producerEntity(t, f.live.world, actor)
		victim := producerEntity(t, f.live.world, target)
		if e.SkillXP[3] != 2 || victim.HP != beforeTarget.HP-5 || f.live.world.PendingSpellDeliveries() != 1 {
			t.Fatalf("physical release before rider: XP%d target%d deliveries%d", e.SkillXP[3], victim.HP, f.live.world.PendingSpellDeliveries())
		}
		if d := actorProducer1162Differences(f.live.world, producer1162Snapshot(t, f).SavedDocument); len(d) != 0 {
			t.Fatal("physical release before rider projection", d)
		}
		for n := 0; n < 16 && f.live.world.PendingSpellDeliveries() != 0; n++ {
			sim.Step(f.live.world, nil)
		}
		e = producerEntity(t, f.live.world, actor)
		victim = producerEntity(t, f.live.world, target)
		if e.SkillXP != ([6]int32{11, 22, 33, 4, 55, 66}) || e.ActorLoad.Source.Experience != 0x12340004 || binary.LittleEndian.Uint16(e.ActorLoad.Source.Base[8:]) != 1 || victim.HP != beforeTarget.HP-12 {
			t.Fatalf("first blow XP/base/HP: XP%v base%d target%d->%d type%d band%v relation%d fighter%v phase%d credit%d/%v", e.SkillXP, binary.LittleEndian.Uint16(e.ActorLoad.Source.Base[8:]), beforeTarget.HP, victim.HP, e.TypeID, sim.InPersistBand(e.TypeID), f.live.world.Relations().Byte(1, 5), e.ActorLoad.Source.Fighter, e.AttackPhase, victim.KillCreditSource, victim.HasKillCredit)
		}
		current := producer1162Snapshot(t, f)
		if d := actorProducer1162Differences(f.live.world, current.SavedDocument); len(d) != 0 {
			t.Fatal(d)
		}
		for _, field := range []string{"Health", "Mana", "HealthMax", "ManaMax", "HealthRegen", "ManaRegen", "U130", "UA2", "UA3", "UA6", "UBE", "U114", "UD4", "H1CC", "lost actor"} {
			bad, err := cloneSavedDocument(current.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			for _, b := range bad.Actors {
				if b.EntityID == actor {
					if field == "lost actor" {
						continue
					}
					r := &bad.Document.Objects[b.ObjectIndex-1]
					for i := range r.Values {
						if r.Values[i].Name == field {
							r.Values[i].Value ^= 1
						}
					}
					for i := range r.Raw {
						if r.Raw[i].Name == field {
							r.Raw[i].Bytes[0] ^= 1
						}
					}
				}
			}
			if field == "lost actor" {
				var keep []SnapshotSAVActor
				for _, b := range bad.Actors {
					if b.EntityID != actor {
						keep = append(keep, b)
					}
				}
				bad.Actors = keep
			}
			if d := actorProducer1162Differences(f.live.world, bad); len(d) == 0 {
				t.Fatal("stale actor control accepted", field)
			}
		}
		fresh, written := producerSAVFresh(t, f, app, path)
		if d := actorProducer1162Differences(fresh.live.world, fresh.live.mission.state.savedDocument); len(d) != 0 {
			t.Fatal("fresh stored actor Document", d)
		}
		if d := actorProducer1162Differences(f.live.world, fresh.live.mission.state.savedDocument); len(d) != 0 {
			t.Fatal("written Document differs from changed live actor", d)
		}
		runtime := producerEntity(t, f.live.world, actor).SourceBinding.RuntimeID
		lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
			for i := range doc.Objects {
				if id, _ := savedStructureValue(&doc.Objects[i], "RuntimeID"); id == runtime && doc.Objects[i].Class == "Human" {
					return savedStructureSetValue(&doc.Objects[i], "U130", 1) == nil
				}
			}
			return false
		})
		if d := actorProducer1162Differences(f.live.world, lost.live.mission.state.savedDocument); len(d) == 0 {
			t.Fatal("loss control: a SAV with the actor's U130 altered still matches")
		}
		if err := fresh.live.world.RestoreScriptProgram(quiet); err != nil {
			t.Fatal(err)
		}
		for range 5 {
			sim.Step(f.live.world, nil)
			sim.Step(fresh.live.world, nil)
			a, b := producerEntity(t, f.live.world, actor), producerEntity(t, fresh.live.world, actor)
			if a.HP != b.HP || a.HealthHundredths != b.HealthHundredths || a.Mana != b.Mana || a.ManaHundredths != b.ManaHundredths || a.SkillXP != b.SkillXP {
				t.Fatal("restored actor continuation differs", a.HP, b.HP)
			}
			for _, front := range []*FrontEnd{f, fresh} {
				if d := actorProducer1162Differences(front.live.world, producer1162Snapshot(t, front).SavedDocument); len(d) != 0 {
					t.Fatal(d)
				}
			}
		}
		t.Log("current HP37->30, fractions13->37/27->35; first installed-weapon attack12 HP including rider, XP slot3 0->4/base0->1/aggregate+4;15 stale field/block/loss controls; changed SAVE/fresh LOAD/next5 ticks")
	})
	t.Run("P3 Cells Building", func(t *testing.T) {
		f, app, path, _ := producerSAVLoad(t)
		old := producer1162Snapshot(t, f)
		// The untouched original's actual20-tick movement exercises cell
		// construction/deletion. Headless relocation below is only the separate
		// Building attack setup; it does not claim to rewrite saved cell state.
		for range 20 {
			f.live.tick()
		}
		quiet, err := sim.NewScript(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		installTestScript(t, f, quiet)
		const actor sim.EntityID = 35
		producerBasis(t, f.live.world, actor, true)
		if err := f.live.world.HeadlessPlace(actor, 32, 49); err != nil {
			t.Fatal(err)
		}
		sim.Step(f.live.world, []sim.Command{{Kind: sim.KindAttackStructure, Entity: actor, X: 4}})
		for n := 0; n < 128 && f.live.world.Structures()[4].Field42 == 1000; n++ {
			sim.Step(f.live.world, nil)
		}
		current := producer1162Snapshot(t, f)
		if d := buildingProducer1162Differences(f.live.world, current.SavedDocument, 4, 984); len(d) != 0 {
			t.Fatal("literal first structure blow1000->984", f.live.world.Structures()[4].Field42, d)
		}
		if d := cellProducer1162Differences(f.live.world, current.SavedDocument); len(d) != 0 {
			t.Fatal(d)
		}
		if reflect.DeepEqual(old.SavedDocument.Document.World.Blocks, current.SavedDocument.Document.World.Blocks) || reflect.DeepEqual(old.SavedDocument.Document.World.Cells, current.SavedDocument.Document.World.Cells) {
			t.Fatal("movement did not change both block plane and current cells")
		}
		for _, field := range []string{"Blocks", "Cells", "lost cell", "B42"} {
			bad, err := cloneSavedDocument(current.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			switch field {
			case "Blocks":
				bad.Document.World.Blocks = old.SavedDocument.Document.World.Blocks
			case "Cells":
				bad.Document.World.Cells = old.SavedDocument.Document.World.Cells
			case "lost cell":
				bad.Document.World.Cells = bad.Document.World.Cells[1:]
			case "B42":
				for _, index := range bad.Document.World.Buildings {
					r := &bad.Document.Objects[index-1]
					for i := range r.Values {
						if r.Values[i].Name == "B42" {
							r.Values[i].Value = 1000
						}
					}
				}
			}
			if len(cellProducer1162Differences(f.live.world, bad))+len(buildingProducer1162Differences(f.live.world, bad, 4, 984)) == 0 {
				t.Fatal("stale cell/structure control accepted", field)
			}
		}
		fresh, written := producerSAVFresh(t, f, app, path)
		if d := cellProducer1162Differences(fresh.live.world, fresh.live.mission.state.savedDocument); len(d) != 0 {
			t.Fatal("fresh stored cells", d)
		}
		if d := buildingProducer1162Differences(fresh.live.world, fresh.live.mission.state.savedDocument, 4, 984); len(d) != 0 {
			t.Fatal("fresh stored building", d)
		}
		if d := append(cellProducer1162Differences(f.live.world, fresh.live.mission.state.savedDocument), buildingProducer1162Differences(f.live.world, fresh.live.mission.state.savedDocument, 4, 984)...); len(d) != 0 {
			t.Fatal("written Document differs from changed live cells or building", d)
		}
		lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
			for _, index := range doc.World.Buildings {
				r := &doc.Objects[index-1]
				if v, err := savedStructureValue(r, "B42"); err == nil && v == 984 {
					return savedStructureSetValue(r, "B42", 1000) == nil
				}
			}
			return false
		})
		if d := buildingProducer1162Differences(f.live.world, lost.live.mission.state.savedDocument, 4, 984); len(d) == 0 {
			t.Fatal("loss control: a SAV with Building4 B42 restored to 1000 still matches")
		}
		if err := fresh.live.world.RestoreScriptProgram(quiet); err != nil {
			t.Fatal(err)
		}
		for range 5 {
			sim.Step(f.live.world, nil)
			sim.Step(fresh.live.world, nil)
			if a, b := f.live.world.Structures()[4].Field42, fresh.live.world.Structures()[4].Field42; a != b {
				t.Fatal("restored Building4 continuation differs", a, b)
			}
			for _, front := range []*FrontEnd{f, fresh} {
				now := producer1162Snapshot(t, front)
				if d := cellProducer1162Differences(front.live.world, now.SavedDocument); len(d) != 0 {
					t.Fatal(d)
				}
				if d := buildingProducer1162Differences(front.live.world, now.SavedDocument, 4, front.live.world.Structures()[4].Field42); len(d) != 0 {
					t.Fatal(d)
				}
			}
		}
		t.Log("untouched original next20 ticks change planes/nodes; controlled attack placement near32,49; complete current block/52-byte cell oracle; Building4 first physical blow1000->984;4 stale/loss controls; changed SAVE/fresh LOAD/next5 ticks")
	})
	t.Run("P3 Building After Move", func(t *testing.T) {
		f, _, _, _ := producerSAVLoad(t)
		for range 20 {
			f.live.tick()
		}
		quiet, err := sim.NewScript(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		installTestScript(t, f, quiet)
		const actor sim.EntityID = 35
		w := f.live.world
		producerBasis(t, w, actor, true)
		if err := w.HeadlessPlace(actor, 32, 49); err != nil {
			t.Fatal(err)
		}
		// The walk helpers wait out a crossing; this order must arrive during
		// one, so the walk stops when the actor first stands on 32,49.
		order := []sim.Command{sim.MoveTo(actor, sim.CellPoint{X: 32, Y: 49})}
		for n := 0; ; n++ {
			e, _ := w.Entity(actor)
			if e.X == 32 && e.Y == 49 {
				break
			}
			if n >= headlessReachLimit || n > 0 && !e.HasTarget && e.Transit == 0 {
				t.Fatalf("actor %d stopped at (%d,%d) short of 32,49", actor, e.X, e.Y)
			}
			sim.Step(w, order)
			order = nil
		}
		if e, _ := w.Entity(actor); e.Transit == 0 {
			t.Fatal("the move already ended; the order would not arrive mid-crossing")
		}
		sim.Step(w, []sim.Command{sim.AttackStructure(actor, 4)})
		n := 0
		for ; n < 128 && w.Structures()[4].Field42 == 1000; n++ {
			sim.Step(w, nil)
		}
		e, _ := w.Entity(actor)
		t.Logf("first blow after %d ticks: 1000->%d, attack held %v", n, w.Structures()[4].Field42, e.HasAttackTarget)
		if w.Structures()[4].Field42 != 984 || !e.HasAttackTarget {
			t.Fatal("attack given during the move did not land", w.Structures()[4].Field42)
		}
	})
}
