package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// This fixture writes the archive programme directly, without consulting the
// pool reader, importer, template statistics, or any original bytes. Pointer
// reuse below deliberately emits an archive back-reference, not a copied body.
type poolFixtureActor struct {
	mapID, cell, hp, maxHP, mana, maxMana uint16
	stage                                 byte
	human                                 bool
	humanoid                              bool
	name                                  string
	runtime                               *uint32
	timer                                 int8
	spells                                []byte // nil absent; non-nil slots 1..n, zero null
	class                                 string // explicit Humanoid fixture; otherwise the old human flag
	profile                               *profileFixture1107
	holdings                              *holdingFixture
	book                                  []*poolFixtureSpell
	bookOff, off                          int       // literal writer positions for corruption tests
	loadWords                             *[4]int16 // speed, own-weight, current load, capacity
	equipmentRuntime                      *[3]byte  // source reach, charge, relax
}

type poolFixtureSpell struct {
	id, rangeByte, defensive byte
	cost                     uint16
	actorRef                 bool
	off                      int
}

type poolFixturePlayer struct {
	groups  [][]*poolFixtureActor
	hero    *poolFixtureActor // identity key is patched after its object is emitted
	reserve uint32            // literal Player F58, separate from Token owner
}

func poolFixtureBody(players []*poolFixturePlayer, dead []*poolFixtureActor) []byte {
	return poolFixtureBodyWithSession(players, dead, nil)
}

func poolFixtureBodyWithSession(players []*poolFixturePlayer, dead []*poolFixtureActor, session []byte) []byte {
	b := append([]byte(nil), savedBody(10, nil)[:75]...)
	binary.LittleEndian.PutUint32(b[71:], uint32(len(players)))
	u16 := func(v uint16) { b = binary.LittleEndian.AppendUint16(b, v) }
	u32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	raw := func(n int) { b = append(b, make([]byte, n)...) }
	next := uint16(1)
	classes := map[string]uint16{}
	obj := func(class string) uint16 {
		if index := classes[class]; index != 0 {
			u16(0x8000 | index)
		} else {
			u16(0xffff)
			u16(1)
			u16(uint16(len(class)))
			b = append(b, class...)
			classes[class] = next
			next++
		}
		index := next
		next++
		return index
	}
	actors := map[*poolFixtureActor]uint16{}
	items := map[*holdingFixtureItem]uint16{}
	item := func(p *holdingFixtureItem) {
		if p == nil {
			u16(0)
			return
		}
		if index := items[p]; index != 0 {
			u16(index)
			return
		}
		index := obj(p.class)
		items[p] = index
		raw(16)
		b = append(b, p.row)
		raw(8)
		u32(uint32(p.price))
		u32(uint32(index) + 2000)
		u32(0)
		u32(uint32(len(p.effects)))
		for _, e := range p.effects {
			obj("Effect")
			raw(37)
			b = append(b, e.Kind, e.Mode)
			u32(e.Operand)
			b = append(b, p.state)
		}
		u16(p.code)
		u16(p.count)
		b = append(b, p.kind, 0, 0)
		u16(0)
		u16(uint16(p.weight))
		b = append(b, 0)
		switch p.class {
		case "Weapon":
			b = append(b, p.attack[:]...)
			b = append(b, p.defence[:]...)
			b = append(b, p.ownKind)
			u16(0)
		case "Shield":
			b = append(b, p.defence[:]...)
		case "Armor":
			b = append(b, p.defence[:]...)
			kind := p.ownKind
			if kind == 0 {
				kind = byte(p.code >> 8 & 15)
			}
			b = append(b, kind)
		}
	}
	spells := map[*poolFixtureSpell]uint16{}
	actor := func(a *poolFixtureActor) {
		if a == nil {
			u16(0)
			return
		}
		if index := actors[a]; index != 0 {
			u16(index)
			return
		}
		class := "Unit"
		if a.humanoid {
			class = "Humanoid"
		}
		if a.human {
			class = "Human"
		}
		if a.holdings != nil && a.holdings.humanoid {
			class = "Humanoid"
		}
		if a.class != "" {
			class = a.class
		}
		index := obj(class)
		actors[a] = index
		a.off = len(b)
		u16(a.cell)
		u16(a.cell)
		b = append(b, 128, 128)
		u16(0)
		u32(0x1234a020)
		runtime := uint32(index)
		if a.runtime != nil {
			runtime = *a.runtime
		}
		u32(runtime)
		b = append(b, 1) // definition row, deliberately unrelated to map ID
		u16(0)
		u32(uint32(a.mapID))
		u16(0)
		u32(0)
		u32(1000 + uint32(index))
		u32(0)
		raw(4 + 2 + 2)
		if a.profile == nil {
			raw(24 + 22 + 24 + 64)
		} else {
			b = append(b, a.profile.attack[:]...)
			b = append(b, a.profile.defence[:]...)
			raw(24)
			b = append(b, a.profile.modifier[:]...)
		}
		raw(180 + 148 + 2)
		raw(19)
		b[len(b)-19], b[len(b)-18] = 1, 1 // literal saved footprint and ground domain
		b[len(b)-1] = byte(a.timer)
		if a.holdings == nil {
			u16(0)
			u16(0)
		} else {
			item(a.holdings.weapon)
			item(a.holdings.shield)
		}
		name := a.name
		if name == "" {
			name = "NPC"
		}
		b = append(b, byte(len(name)))
		b = append(b, name...)
		stats := []uint16{30, 20, 10, 5, 16, 0, 0, 300, a.hp, a.maxHP, 333, a.mana, a.maxMana, 444}
		if a.loadWords != nil {
			for i, v := range a.loadWords {
				stats[4+i] = uint16(v)
			}
		}
		if a.profile != nil {
			stats[1], stats[2], stats[3] = a.profile.stats[0], a.profile.stats[1], a.profile.stats[2]
			stats[10], stats[13] = a.profile.periods[0], a.profile.periods[1]
		}
		for _, word := range stats {
			u16(word)
		}
		if a.profile == nil {
			raw(2)
		} else {
			b = append(b, a.profile.fractions[:]...)
		}
		u16(0)       // mana floor
		u16(8 * 256) // explicit saved sight for this fixture's cast/visibility routes
		runtimeStart := len(b)
		raw(12)
		if a.equipmentRuntime != nil {
			b[runtimeStart], b[runtimeStart+5], b[runtimeStart+6] = a.equipmentRuntime[0], a.equipmentRuntime[1], a.equipmentRuntime[2]
		}
		b = append(b, a.stage)
		raw(8 + 2) // remaining Unit state, U68
		if a.holdings == nil {
			b = append(b, 0)
		} else {
			b = append(b, 1)
			u32(uint32(len(a.holdings.items)))
			for _, p := range a.holdings.items {
				item(p)
			}
			index, accumulator := uint32(10000), int32(0)
			if a.holdings.insertIndex != nil {
				index = *a.holdings.insertIndex
			}
			if a.holdings.accumulator != nil {
				accumulator = *a.holdings.accumulator
			}
			u32(index)
			u32(uint32(accumulator))
		}
		a.bookOff = len(b)
		if a.book != nil {
			b = append(b, 1)
			u32(0)
			u32(uint32(len(a.book) + 1))
			for _, spell := range a.book {
				if spell == nil {
					u16(0)
					continue
				}
				if spell.actorRef {
					u16(index)
					continue
				}
				if ref := spells[spell]; ref != 0 {
					u16(ref)
					continue
				}
				ref := obj("Spell")
				spells[spell] = ref
				spell.off = len(b)
				b = append(b, spell.id, spell.rangeByte, spell.defensive)
				u16(spell.cost)
				u32(1000 + uint32(ref))
			}
		} else if a.spells == nil {
			b = append(b, 0)
		} else {
			b = append(b, 1)
			u32(0)
			u32(uint32(len(a.spells) + 1))
			for _, id := range a.spells {
				if id == 0 {
					u16(0)
					continue
				}
				index := obj("Spell")
				b = append(b, id, 13, 1)
				u16(17)
				u32(1000 + uint32(index))
			}
		}
		raw(17)
		if class != "Unit" {
			raw(24)
			for slot := 0; slot < 12; slot++ {
				if a.holdings == nil {
					u16(0)
				} else {
					item(a.holdings.worn[slot])
				}
			}
			u16(0)
		}
	}
	seenPlayers := map[*poolFixturePlayer]uint16{}
	for i, player := range players {
		if player == nil {
			u16(0)
			continue
		}
		if index := seenPlayers[player]; index != 0 {
			u16(index)
			continue
		}
		seenPlayers[player] = obj("Player")
		b = append(b, 1, 'P')
		u16(uint16(i + 1))
		u32(uint32(i + 1))
		raw(8)
		b = append(b, 2)
		participant := uint32(1)
		if len(seenPlayers) == 1 {
			participant = 0
		}
		u32(participant)
		u16(2)
		u32(700 ^ 0x5c073f4d)
		raw(2)
		u32(0x5c073f4d)
		raw(4 + 2 + 2)
		u32(player.reserve)
		heroOff := len(b)
		u32(0)
		u32(uint32(900 + i))
		u32(uint32(len(player.groups)))
		for _, group := range player.groups {
			raw(2 + 80 + 2)
			u32(uint32(len(group)))
			for _, a := range group {
				actor(a)
			}
			raw(12)
		}
		if player.hero != nil {
			binary.LittleEndian.PutUint32(b[heroOff:], 1000+uint32(actors[player.hero]))
		}
		raw(32 + 2 + 2 + 4) // raw Player tail and inline Diary
	}
	u32(uint32(len(dead)))
	for _, a := range dead {
		actor(a)
	}
	b = append(b, 1)
	raw(8 + 2 + 2 + 4) // world, empty lists/terrain and terrain identity
	if session == nil {
		raw(4374)
	} else {
		if len(session) != 4374 {
			panic("fixture session must have exactly 4374 bytes")
		}
		b = append(b, session...)
	}
	u32(0) // empty Sack list
	return savedTrailer(b)
}

func poolFixtureSave(actors ...*poolFixtureActor) []byte {
	return savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{actors}}}, nil))
}

// Assign the authored IDs independently at the ALM type-6 wire offsets. synth
// supplies the container and otherwise-zero placed-unit records.
func poolFixtureMap(ids ...uint16) []byte {
	units := make([]synth.ALMUnit, len(ids))
	for i := range units {
		units[i] = synth.ALMUnit{X: uint32(15+i)<<8 | 128, Y: 16<<8 | 128}
	}
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: units})
	for off := 20; off+20 <= len(b); {
		size := int(binary.LittleEndian.Uint32(b[off+8:]))
		if binary.LittleEndian.Uint32(b[off+12:]) == 6 {
			for i, id := range ids {
				binary.LittleEndian.PutUint16(b[off+20+i*70+64:], id)
			}
			return b
		}
		off += 20 + size
	}
	panic("synthetic map has no type-6 record")
}

func poolFixtureFront(t *testing.T, ids ...uint16) *FrontEnd {
	t.Helper()
	return poolFixtureFrontMap(t, poolFixtureMap(ids...))
}

func currentPoolFixtureFront(t *testing.T, ids ...uint16) *FrontEnd {
	t.Helper()
	f := poolFixtureFront(t, ids...)
	f.Table = actorRegistryTable()
	units := f.Table.Units.(dbCollection)
	f.Table.Units = dbCollection{units[1], units[1]}
	f.Humans = fourBaseHumans()
	f.Table.Humans = append(fourBaseHumans(), dbEntry{name: "actor runtime", params: humansParams(50, 0, 5)})
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	return f
}

func poolFixtureFrontMap(t *testing.T, mapBytes []byte) *FrontEnd {
	t.Helper()
	f := missionFrontEnd(t)
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	b := synth.Archive([]synth.File{{Path: "10.alm", Data: mapBytes}, {Path: "npc.reg", Data: synth.NPCReg(nil)}})
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	f.SetDeterministicFrames(true)
	return f
}

func poolEntity(t *testing.T, w *sim.World, mapID uint16) sim.Entity {
	t.Helper()
	var result sim.Entity
	n := 0
	for _, entity := range w.Entities() {
		if entity.MapUnitID == mapID {
			result, n = entity, n+1
		}
	}
	if n != 1 {
		t.Fatalf("MapUnitID %d names %d entities", mapID, n)
	}
	return result
}

func assertPools(t *testing.T, entity sim.Entity, want [4]int32) {
	t.Helper()
	if got := [4]int32{entity.HP, entity.MaxHP, entity.Mana, entity.MaxMana}; got != want {
		t.Fatalf("MapUnitID %d pools %v, want %v", entity.MapUnitID, got, want)
	}
}

func TestOriginalPools1094AllPlayersSparseAndRepeatedReferences(t *testing.T) {
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, mana: 5, maxMana: 23}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, mana: 6, maxMana: 29, human: true}
	first := &poolFixturePlayer{groups: [][]*poolFixtureActor{{a, nil, a}}}
	last := &poolFixturePlayer{groups: [][]*poolFixtureActor{{}, {nil, b, a}}}
	body := poolFixtureBody([]*poolFixturePlayer{first, nil, last, first}, []*poolFixtureActor{{mapID: 93, hp: 1, maxHP: 2}})
	sf, err := sav.Open(savedContainer(body))
	if err != nil || len(sf.Players) != 2 {
		t.Fatalf("exact sparse Player index: %v", err)
	}
	pools, err := sf.ActorPools()
	if err != nil || len(pools) != 2 || pools[0].MapUnitID != 91 || pools[1].MapUnitID != 92 ||
		pools[0].HP != 7 || pools[0].MaxHP != 31 || pools[0].Mana != 5 || pools[0].MaxMana != 23 || pools[1].HP != 9 {
		t.Fatalf("all-Player pool projection = %+v, %v", pools, err)
	}
	if _, present, err := sf.GroundSacks(); err != nil || !present {
		t.Fatalf("shared traversal changed ground document endpoint: %t %v", present, err)
	}
	party, err := sf.Party()
	if err != nil || len(party) != 1 || party[0].MapUnitID != 91 {
		t.Fatalf("first-Player unique Party: %+v %v", party, err)
	}
	rawParty, _, err := sf.PartyWalk()
	if err != nil || len(rawParty) != 2 || rawParty[0].MapUnitID != 91 || rawParty[1].MapUnitID != 91 {
		t.Fatalf("raw first-Player PartyWalk lost occurrences: %+v %v", rawParty, err)
	}
	pools[0].HP = 99
	again, _ := sf.ActorPools()
	if again[0].HP != 7 {
		t.Fatal("returned values alias archive state")
	}
	for end := 0; end < pools[1].Off+612; end++ {
		truncated := &sav.File{Body: body[:end], Head: sf.Head}
		if partial, err := truncated.ActorPools(); err == nil || len(partial) != 0 {
			t.Fatalf("truncated actor list at %d accepted/returned partial pools: %+v %v", end, partial, err)
		}
	}
	for _, head := range []sav.Head{{End: -1, PlayerCount: 1}, {End: len(body) + 1, PlayerCount: 1}, {End: 75, PlayerCount: 65537}} {
		if _, err := (&sav.File{Body: body, Head: head}).ActorPools(); err == nil {
			t.Fatalf("accepted unbounded Player head: %+v", head)
		}
	}
}

func TestOriginalPools1094AppLoadAndNativeSaveKeepFourValues(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("original-pools")
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, mana: 5, maxMana: 23}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, mana: 6, maxMana: 29, human: true}
	payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{nil, a, a}, {}, {b, nil}}}}, nil))
	originals := t.TempDir()
	if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	assertPools(t, poolEntity(t, f.live.world, 91), [4]int32{7, 31, 5, 23})
	assertPools(t, poolEntity(t, f.live.world, 92), [4]int32{9, 41, 6, 29})
	if f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
		t.Fatal("LOAD advanced before imported pools were visible")
	}
	hash := f.live.world.Hash()
	priorWorld := f.live.world
	priorDriver := f.live
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatalf("ordinary mission SAVE = %+v, %v: %s", entries, err, app.HeadlessMessage())
	}
	groundAppLoad(t, app, list, localOriginalSaveToken(entries[0].Name))
	assertPools(t, poolEntity(t, f.live.world, 91), [4]int32{7, 31, 5, 23})
	assertPools(t, poolEntity(t, f.live.world, 92), [4]int32{9, 41, 6, 29})
	if f.live.world.Hash() != hash {
		t.Fatalf("native SAVE/LOAD changed the imported world hash: %x -> %x", hash, f.live.world.Hash())
	}
	// Advance the actual canonical worlds identically after the save boundary.
	// A transient importer cache or reset remainder cannot survive this check.
	for range 32 {
		priorDriver.tick()
		f.live.tick()
		if priorWorld.Hash() != f.live.world.Hash() {
			t.Fatalf("native continuation diverged at tick %d", priorWorld.Tick())
		}
	}
	ms, report, err := loadOriginalMission(f, payload)
	if err != nil || report.PoolsRestored != 2 {
		t.Fatalf("diagnostic resume: %+v %v", report, err)
	}
	assertPools(t, poolEntity(t, ms.World, 91), [4]int32{7, 31, 5, 23})
}

func TestOriginalPools1094SharedActorHandoffRunsAfterRearm(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 0, X: 5, Y: 6, MapUnitID: 91, HP: 20, MaxHP: 20}})
	if err != nil {
		t.Fatal(err)
	}
	member := mapload.PartyMember{Hero: data.Hero{Body: 30, Reaction: 20, Mind: 10, Spirit: 5}}
	ms := &Mission{Map: &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{UnitID: 91}}}, World: w,
		Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{0: member}}}
	chars := []sav.ActorHoldings{{MapUnitID: 91, HP: 20, Cell: 0x0605}}
	if err := restoreOriginalActorStock(ms, chars, nil, &OriginalSaveResume{}); err != nil {
		t.Fatal("fixture failed to restock its non-party character")
	}
	rearmed := poolEntity(t, w, 91)
	if rearmed.MaxHP == 20 || rearmed.MaxHP == 31 || !rearmed.Alive() {
		t.Fatalf("fixture did not exercise a distinct, living rearm maximum: %d/%d", rearmed.HP, rearmed.MaxHP)
	}
	var report OriginalSaveResume
	if err := restoreOriginalActors(ms, chars,
		[]sav.ActorPools{{MapUnitID: 91, Cell: 0x0605, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}}, nil, nil, &report, nil); err != nil {
		t.Fatal(err)
	}
	if report.Stocked != 1 || report.PoolsRestored != 1 {
		t.Fatalf("shared handoff counts: %+v", report)
	}
	assertPools(t, poolEntity(t, w, 91), [4]int32{7, 31, 5, 23})
}

func TestOriginalPools1094BoundariesAndExactUniqueJoin(t *testing.T) {
	m := &alm.Map{Width: 40, Height: 40}
	entities := []sim.Entity{
		{ID: 1, X: 5, Y: 6, MapUnitID: 91, HP: 20, MaxHP: 20},
		{ID: 2, X: 6, Y: 6, MapUnitID: 92, HP: 21, MaxHP: 21},
		{ID: 3, X: 7, Y: 6, MapUnitID: 93, HP: 22, MaxHP: 22, OffMap: true},
		{ID: 4, X: 8, Y: 6, MapUnitID: 94, HP: 0, MaxHP: 23},
	}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{Map: m, World: w, Party: []mapload.PartyMember{{Saved: &mapload.Saved{MapUnitID: 92}}}}
	valid := sav.ActorPools{MapUnitID: 91, Cell: 0x0605, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}
	source := []sav.ActorPools{valid,
		{MapUnitID: 92, Cell: 0x0606, HP: 1, MaxHP: 1},   // existing party owns it
		{MapUnitID: 999, Cell: 0x0605, HP: 99, MaxHP: 1}, // unmatched, no guessed cell match
		{MapUnitID: 0, Cell: 0x0605, HP: 1, MaxHP: 1},
		{MapUnitID: 91, Cell: 0xffff, HP: 1, MaxHP: 1},
		{MapUnitID: 91, Cell: 0x0605, HP: 0, MaxHP: 1},
		{MapUnitID: 91, Cell: 0x0605, HP: 65535, MaxHP: 1},
		{MapUnitID: 91, Cell: 0x0605, HP: 1, MaxHP: 1, Stage: 3},
		{MapUnitID: 93, Cell: 0x0607, HP: 1, MaxHP: 1},
		{MapUnitID: 94, Cell: 0x0608, HP: 1, MaxHP: 1}}
	var report OriginalSaveResume
	if err := applyOriginalActorPools(ms, source, &report); err != nil {
		t.Fatal(err)
	}
	assertPools(t, poolEntity(t, w, 91), [4]int32{7, 31, 5, 23})
	assertPools(t, poolEntity(t, w, 92), [4]int32{21, 21, 0, 0})
	if report.PoolsRestored != 1 || report.PoolsParty != 1 || report.PoolsExcluded != 7 || report.PoolsUnmatched != 1 {
		t.Fatalf("boundary report: %+v", report)
	}
	before := w.Hash()
	if err := applyOriginalActorPools(ms, []sav.ActorPools{valid, valid}, &OriginalSaveResume{}); err == nil || !strings.Contains(err.Error(), "ambiguous") || w.Hash() != before {
		t.Fatalf("duplicate source refusal: %v", err)
	}
	entities[1].MapUnitID = 91
	w, err = sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	ms.World, ms.Party = w, nil
	before = w.Hash()
	if err := applyOriginalActorPools(ms, []sav.ActorPools{valid}, &OriginalSaveResume{}); err == nil || !strings.Contains(err.Error(), "ambiguous") || w.Hash() != before {
		t.Fatalf("duplicate target refusal: %v", err)
	}
}

func TestOriginalPools1094RefusedAppLoadRetainsOldSession(t *testing.T) {
	for _, kind := range []string{"health above maximum", "mana above maximum", "zero maximum", "target collision", "truncated actor"} {
		t.Run(kind, func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			if kind == "target collision" {
				f = currentPoolFixtureFront(t, 91, 91)
			}
			app := f.App("refused-pools")
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			f.Offered, f.Carried = 10, mapload.CloneParty(f.liveParty)
			a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, mana: 5, maxMana: 23}
			b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, mana: 6, maxMana: 29}
			switch kind {
			case "health above maximum":
				b.hp = 42
			case "mana above maximum":
				b.mana = 30
			case "zero maximum":
				b.maxHP = 0
			}
			payload := poolFixtureSave(a, b)
			if kind == "truncated actor" {
				body := poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{a, b}}}}, nil)
				payload = savedContainer(body[:1100])
			}
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0o600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			if kind == "truncated actor" {
				before, _, err := f.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				priorLive, priorTown := f.live, f.Town
				if _, _, err := f.RestoreOriginal(payload); err == nil {
					t.Fatal("truncated archive accepted")
				}
				after, _, err := f.Snapshot(true)
				if err != nil || !reflect.DeepEqual(before, after) || f.live != priorLive || f.Town != priorTown {
					t.Fatal("early rejection changed session")
				}
				if len(list()) != 0 {
					t.Fatal("malformed archive offered as LOAD candidate")
				}
				return
			}
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			oldLive, oldTown, oldHash := f.live, f.Town, f.live.world.Hash()
			rows := list()
			if len(rows) != 1 {
				t.Fatalf("LOAD rows = %+v", rows)
			}
			if err := app.HeadlessActivate(rows[0].Label); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" {
				t.Fatalf("bad LOAD accepted: %s %q", app.Screen(), app.HeadlessMessage())
			}
			after, _, err := f.Snapshot(true)
			if err != nil || !reflect.DeepEqual(after, before) || f.live != oldLive || f.Town != oldTown || f.live.world.Hash() != oldHash || f.Offered != 10 {
				t.Fatalf("refused LOAD changed old session: %v", err)
			}
			name, err := save(true)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := store.Read(name)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Ext(name) != ".sav" {
				t.Fatalf("post-refusal SAVE did not write SAV: %s", name)
			}
			if _, err := sav.DecodeDocumentData(stored); err != nil {
				t.Fatalf("post-refusal SAVE is not an ordinary document: %v", err)
			}
			ids := []uint16{91, 92}
			if kind == "target collision" {
				ids[1] = 91
			}
			fresh := currentPoolFixtureFront(t, ids...)
			freshApp := fresh.App("post-refusal cold SAV")
			fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, ff)
			groundAppLoad(t, freshApp, fl, localOriginalSaveToken(name))
			expectedParty := mapload.CloneParty(before.Party)
			for i, id := range before.CurrentPartyIDs {
				items, ok := f.live.world.CarriedItems(id)
				member := &expectedParty[i]
				if !ok || member.Carry == nil || len(items) != len(member.CarriedItems) || len(items) != len(member.Carry.ItemInstances) || len(items) != len(member.Carry.OrderedStacks) {
					t.Fatal("current party item view has no matching World roots")
				}
				for j, item := range items {
					member.CarriedItems[j].ObjectID = item.ObjectID
					member.Carry.ItemInstances[j].ObjectID = item.ObjectID
					member.Carry.OrderedStacks[j].ObjectID = item.ObjectID
				}
			}
			cold, _, err := fresh.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if fresh.live.world.Hash() != oldHash || fresh.Offered != before.Offered || !reflect.DeepEqual(cold.Party, expectedParty) {
				currentItemFieldDiagnostics(t, "Party", reflect.ValueOf(expectedParty), reflect.ValueOf(cold.Party))
				t.Fatalf("post-refusal SAV cold LOAD lost original session: world %x -> %x; offered %d -> %d; party %t", oldHash, fresh.live.world.Hash(), before.Offered, fresh.Offered, reflect.DeepEqual(cold.Party, expectedParty))
			}
		})
	}
}
