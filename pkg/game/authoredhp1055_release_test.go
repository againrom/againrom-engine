package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type rawAuthoredHealth1055 struct {
	Index   int
	UnitID  uint16
	ClassID int16
	X, Y    uint32
	HP      int16
}

// rawAuthoredHealthWalk1055 is intentionally independent of pkg/formats/alm:
// it walks only the file and record frames and reads the six-record stride and
// field offsets directly. A decoder regression therefore cannot generate both
// production state and this witness's oracle.
func rawAuthoredHealthWalk1055(data []byte) ([]rawAuthoredHealth1055, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("file is %d bytes, shorter than its 20-byte header", len(data))
	}
	records := int(binary.LittleEndian.Uint32(data[0x0c:0x10]))
	cursor := 20
	count6 := -1
	var type6 []byte
	for i := 0; i < records; i++ {
		if cursor+20 > len(data) {
			return nil, fmt.Errorf("record %d header overruns EOF", i)
		}
		size := int(binary.LittleEndian.Uint32(data[cursor+8 : cursor+12]))
		typeID := binary.LittleEndian.Uint32(data[cursor+12 : cursor+16])
		start, end := cursor+20, cursor+20+size
		if size < 0 || end < start || end > len(data) {
			return nil, fmt.Errorf("record %d payload overruns EOF", i)
		}
		switch typeID {
		case 0:
			if size < 0x28 {
				return nil, fmt.Errorf("type-0 payload is %d bytes", size)
			}
			count6 = int(binary.LittleEndian.Uint32(data[start+0x24 : start+0x28]))
		case 6:
			type6 = data[start:end]
		}
		cursor = end
	}
	if count6 < 0 {
		return nil, fmt.Errorf("no type-0 count")
	}
	if len(type6) != count6*70 {
		return nil, fmt.Errorf("type-6 payload is %d bytes, want %d", len(type6), count6*70)
	}
	var out []rawAuthoredHealth1055
	for i := 0; i < count6; i++ {
		rec := type6[i*70 : (i+1)*70]
		hp := int16(binary.LittleEndian.Uint16(rec[0x20:0x22]))
		if hp == -1 {
			continue
		}
		out = append(out, rawAuthoredHealth1055{
			Index: i, UnitID: binary.LittleEndian.Uint16(rec[0x40:0x42]),
			ClassID: int16(binary.LittleEndian.Uint16(rec[0x08:0x0a])),
			X:       binary.LittleEndian.Uint32(rec[0x00:0x04]),
			Y:       binary.LittleEndian.Uint32(rec[0x04:0x08]), HP: hp,
		})
	}
	return out, nil
}

func TestReleaseAuthoredCurrentHealthCorpusAndRescueTriggers(t *testing.T) {
	f := releaseFront(t)
	maps := releaseMissionPopulation(t, f)
	if len(maps) != 28 {
		t.Fatalf("campaign map population = %d, want 28", len(maps))
	}
	byMission := make(map[int][]rawAuthoredHealth1055)
	for _, mission := range maps {
		addr, _ := MissionMap(mission)
		raw, err := f.Archives.Containers.ReadFile(addr)
		if err != nil {
			t.Fatalf("read %s: %v", addr, err)
		}
		rows, err := rawAuthoredHealthWalk1055(raw)
		if err != nil {
			t.Fatalf("raw walk %s: %v", addr, err)
		}
		if len(rows) != 0 {
			byMission[mission] = rows
		}
	}
	var total int
	var populated []int
	for mission, rows := range byMission {
		total += len(rows)
		populated = append(populated, mission)
	}
	sort.Ints(populated)
	if total != 30 || len(populated) != 7 {
		t.Fatalf("authored current-health corpus = %d overrides on %d maps %v; want 30 on 7",
			total, len(populated), populated)
	}
	wantCorpus := map[int][]rawAuthoredHealth1055{
		30: {
			{Index: 35, UnitID: 64, ClassID: 1, X: 16256, Y: 3712, HP: 0},
			{Index: 36, UnitID: 68, ClassID: 1, X: 16512, Y: 4480, HP: 0},
			{Index: 37, UnitID: 69, ClassID: 1, X: 16768, Y: 5760, HP: 0},
			{Index: 38, UnitID: 70, ClassID: 1, X: 16000, Y: 5248, HP: 0},
		},
		71: {
			{Index: 1, UnitID: 42, ClassID: 21, X: 13440, Y: 17024, HP: 0},
			{Index: 2, UnitID: 43, ClassID: 21, X: 14208, Y: 17536, HP: 0},
			{Index: 39, UnitID: 80, ClassID: 14, X: 14976, Y: 12416, HP: 0},
			{Index: 40, UnitID: 81, ClassID: 14, X: 17536, Y: 12416, HP: 0},
			{Index: 41, UnitID: 82, ClassID: 14, X: 16512, Y: 13184, HP: 0},
			{Index: 42, UnitID: 83, ClassID: 14, X: 17536, Y: 12928, HP: 0},
			{Index: 43, UnitID: 86, ClassID: 11, X: 16256, Y: 13440, HP: 0},
			{Index: 44, UnitID: 89, ClassID: 11, X: 15744, Y: 12672, HP: 0},
			{Index: 45, UnitID: 90, ClassID: 11, X: 17280, Y: 12416, HP: 0},
		},
		81:  {{Index: 17, UnitID: 50, ClassID: 66, X: 10624, Y: 11136, HP: 0}},
		100: {{Index: 112, UnitID: 245, ClassID: 1, X: 33152, Y: 3712, HP: 0}},
		120: {{Index: 83, UnitID: 160, ClassID: 68, X: 4224, Y: 5504, HP: -10}},
		131: {
			{Index: 10, UnitID: 32, ClassID: 1, X: 20352, Y: 17024, HP: 0},
			{Index: 20, UnitID: 47, ClassID: 76, X: 8320, Y: 14976, HP: 0},
			{Index: 21, UnitID: 48, ClassID: 76, X: 7808, Y: 17280, HP: 0},
			{Index: 22, UnitID: 49, ClassID: 76, X: 5248, Y: 16000, HP: 0},
			{Index: 34, UnitID: 61, ClassID: 76, X: 10624, Y: 22144, HP: 0},
		},
		140: {
			{Index: 124, UnitID: 443, ClassID: 24, X: 62080, Y: 29568, HP: 0},
			{Index: 145, UnitID: 887, ClassID: 76, X: 25984, Y: 20864, HP: -10},
			{Index: 146, UnitID: 888, ClassID: 76, X: 25472, Y: 21120, HP: -10},
			{Index: 147, UnitID: 889, ClassID: 76, X: 26496, Y: 21632, HP: -10},
			{Index: 148, UnitID: 890, ClassID: 76, X: 27776, Y: 21120, HP: -10},
			{Index: 149, UnitID: 891, ClassID: 76, X: 27520, Y: 21888, HP: -10},
			{Index: 150, UnitID: 892, ClassID: 76, X: 26496, Y: 21376, HP: -10},
			{Index: 151, UnitID: 893, ClassID: 76, X: 25216, Y: 21632, HP: -10},
			{Index: 152, UnitID: 894, ClassID: 76, X: 26752, Y: 21888, HP: -10},
		},
	}
	if !reflect.DeepEqual(byMission, wantCorpus) {
		t.Fatalf("authored current-health rows =\n%+v\nwant exact installed corpus\n%+v", byMission, wantCorpus)
	}

	releaseRescueMission1055(t, f, 71, []uint16{42, 43}, []int32{248, 248},
		mapload.Cell{X: 54, Y: 67}, []int32{1}, "Horsemans Healed", 3)
	releaseRescueMission1055(t, f, 81, []uint16{50}, []int32{320},
		mapload.Cell{X: 42, Y: 43}, []int32{5, 7}, "n-d Ogre Healed.2", 7)
}

func releaseRescueMission1055(t *testing.T, f *FrontEnd, mission int, unitIDs []uint16,
	wantMax []int32, cell mapload.Cell, latches []int32, triggerName string, event int32) {
	t.Helper()
	m := releaseMissionMap(t, f, mission)
	rawScript, err := m.Script()
	if err != nil {
		t.Fatalf("mission %d script: %v", mission, err)
	}
	for _, latch := range latches {
		if int(latch) >= len(rawScript.Triggers) {
			t.Fatalf("mission %d has %d triggers, no latch %d", mission, len(rawScript.Triggers), latch)
		}
	}
	if rawScript.Triggers[latches[len(latches)-1]].Name != triggerName {
		t.Fatalf("mission %d trigger %d = %q, want %q", mission, latches[len(latches)-1],
			rawScript.Triggers[latches[len(latches)-1]].Name, triggerName)
	}

	hero := data.Hero{Body: 100, Reaction: 100, Mind: 100, Spirit: 100}
	hero.Skill[5] = 100
	party := []mapload.PartyMember{{
		ID: "1055-rescue-mage", Class: 100, Mage: true, StartingHero: true, PlayerCharacter: true,
		Hero: hero, Profile: data.Profile{HealthColumn: true, ManaColumn: true}, KnownSpells: 1 << 6,
		Saved: &mapload.Saved{Cell: cell, HP: 1000, MaxHP: 1000, Mana: 10000, MaxMana: 10000,
			HealthRegenPeriod: 100, ManaRegenPeriod: 50},
	}}
	start := func() *Mission {
		ms, err := StartMissionFrom(m, fmt.Sprintf("scenario%d.alm", mission), mission,
			f.Table, mapload.DifficultyNormal, party)
		if err != nil {
			t.Fatalf("StartMissionFrom(%d): %v", mission, err)
		}
		return ms
	}
	ms := start()
	if len(ms.Start.IDs) != 1 {
		t.Fatalf("mission %d party ids = %v, want one mage", mission, ms.Start.IDs)
	}
	targets := make([]sim.EntityID, len(unitIDs))
	for k, unitID := range unitIDs {
		index := -1
		for i, u := range m.Units {
			if u.UnitID == unitID {
				index = i
				break
			}
		}
		if index < 0 {
			t.Fatalf("mission %d has no map unit %d", mission, unitID)
		}
		targets[k] = sim.EntityID(index)
		e := releaseWorldEntity1055(t, ms.World, targets[k])
		if e.HP != 0 || e.MaxHP != wantMax[k] || e.Decay != sim.DecayFallen {
			t.Fatalf("mission %d unit %d starts at HP %d/%d decay %d, want 0/%d fallen",
				mission, unitID, e.HP, e.MaxHP, e.Decay, wantMax[k])
		}
	}

	// Exercise the native save envelope and the exact in-mission resume seam
	// before play. The fresh ALM world is replaced by the saved world; its hash
	// and body state must remain byte-identical.
	ms = releaseResumeMission1055(t, f, start(), party, mission, ms.World)
	caster := ms.Start.IDs[0]
	seenHeal := make(map[sim.EntityID]bool)
	for ticks := 0; ticks < 2048 && len(seenHeal) < len(targets); ticks++ {
		var cmds []sim.Command
		_, _, casting := ms.World.CastingSpell(caster)
		for _, target := range targets {
			// One click starts one cast. Reissuing every tick now deliberately
			// replaces its wind-up under the owner's manual-priority rule.
			if casting || seenHeal[target] || releaseWorldEntity1055(t, ms.World, target).HP > 0 {
				continue
			}
			if ms.World.BookSpellRefusal(caster, target, 6) == "" {
				cmds = []sim.Command{{Kind: sim.KindCast, Entity: caster, X: int32(target), Y: 6}}
				break
			}
		}
		for _, cast := range sim.StepObserved(ms.World, cmds) {
			if cast.Caster == caster && cast.Spell == 6 && cast.HealthRestored > 0 {
				for _, target := range targets {
					if cast.Target == target {
						seenHeal[target] = true
					}
				}
			}
		}
	}
	if len(seenHeal) != len(targets) {
		t.Fatalf("mission %d production Heal reached %v of targets %v", mission, seenHeal, targets)
	}
	for k, target := range targets {
		e := releaseWorldEntity1055(t, ms.World, target)
		if e.HP <= 0 || e.Decay != sim.DecayNone || e.Dwell != 0 {
			t.Fatalf("mission %d healed unit %d = HP %d decay %d dwell %d",
				mission, unitIDs[k], e.HP, e.Decay, e.Dwell)
		}
	}

	fired := make(map[int32]bool)
	for tick := 0; tick < 4*16; tick++ {
		tr := sim.StepTraced(ms.World, nil)
		for _, firing := range tr.Firings {
			fired[firing.Latch] = true
		}
	}
	for _, latch := range latches {
		if !fired[latch] || !ms.World.ScriptLatched(latch) {
			t.Fatalf("mission %d rescue latch %d did not fire: trace=%v", mission, latch, fired)
		}
	}
	foundRaise := false
	for _, raise := range ms.Raises {
		foundRaise = foundRaise || raise.Latch == latches[len(latches)-1] && raise.Event == event
	}
	if !foundRaise {
		t.Fatalf("mission %d rescue raises = %+v, want latch %d event %d",
			mission, ms.Raises, latches[len(latches)-1], event)
	}

	// A healed save is resumed over a newly started ALM world whose bodies are
	// zero again. The saved positive health and fired latch must win; reapplying
	// the map after unmarshal would fail both the hash and these assertions.
	resumed := releaseResumeMission1055(t, f, start(), party, mission, ms.World)
	for k, target := range targets {
		if e := releaseWorldEntity1055(t, resumed.World, target); e.HP <= 0 || e.Decay != sim.DecayNone {
			t.Fatalf("mission %d resumed healed unit %d = HP %d decay %d",
				mission, unitIDs[k], e.HP, e.Decay)
		}
	}
	for _, latch := range latches {
		if !resumed.World.ScriptLatched(latch) {
			t.Fatalf("mission %d resumed without fired latch %d", mission, latch)
		}
	}
}

func releaseResumeMission1055(t *testing.T, f *FrontEnd, fresh *Mission,
	party []mapload.PartyMember, mission int, source *sim.World) *Mission {
	t.Helper()
	form, err := source.MarshalBinary()
	if err != nil {
		t.Fatalf("mission %d MarshalBinary: %v", mission, err)
	}
	wantHash := source.Hash()
	disk, err := EncodeSave(Snapshot{Mission: mission, Party: party, World: form}, "1055")
	if err != nil {
		t.Fatalf("mission %d EncodeSave: %v", mission, err)
	}
	snapshot, label, err := DecodeSave(disk)
	if err != nil || label != "1055" || !bytes.Equal(snapshot.World, form) {
		t.Fatalf("mission %d DecodeSave = label %q bytes=%v err=%v", mission, label,
			bytes.Equal(snapshot.World, form), err)
	}
	if err := resumeWorld(fresh, &snapshot, f.Table); err != nil {
		t.Fatalf("mission %d resumeWorld err=%v", mission, err)
	}
	if fresh.World.Hash() != wantHash {
		t.Fatalf("mission %d resume hash = %016x, want %016x", mission, fresh.World.Hash(), wantHash)
	}
	return fresh
}

func releaseWorldEntity1055(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("world has no entity %d", id)
	return sim.Entity{}
}
