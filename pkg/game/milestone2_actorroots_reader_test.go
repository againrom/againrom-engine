package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// SAV-UNITLEN-045, SAV-UNITPROG-156, SAV-SPELL-044 and MAGIC-BOOK-002.
// Only structure starts and the identity-only origin permutation come from
// production. Expected fields/counts/edges come from Body. The independent
// 1158 actor population proof rejects omitted/duplicated locators; 1151 reads
// every reachable Item/Effect/Spell field. No imported survivor list supplies
// the expected actor population or the live/late-dead admission decision.
type actor1161Record struct {
	loc    sav.DocumentActorLocation
	stage  byte
	hp     int16
	values map[string]uint32
	counts map[string]uint32
	refs   map[string][]uint16
}

func actor1161Reference(r *sack1151Reader, p *int) uint16 {
	start := *p
	tag := uint16(r.number(p, 2))
	if tag == 0 || r.err != nil {
		return 0
	}
	if tag < 0x8000 {
		loc, ok := r.byIndex[tag]
		if !ok || loc.Off >= start {
			r.err = fmt.Errorf("invalid archive backref %d at %d", tag, start)
			return tag
		}
		switch loc.Class {
		case "Unit", "Human", "Humanoid", "Diary":
			return tag
		}
		*p = start
		return r.reference(p, 0)
	}
	if tag == 0xffff {
		schema, n := r.number(p, 2), r.number(p, 2)
		if schema != 1 || n == 0 || n > 32 {
			r.err = fmt.Errorf("class header at %d", start)
			return 0
		}
		name := string(r.take(p, int(n)))
		if loc, ok := r.byOff[*p]; !ok || loc.Class != name {
			r.err = fmt.Errorf("class header/location mismatch at %d", start)
			return 0
		}
	}
	loc, ok := r.byOff[*p]
	if !ok {
		r.err = fmt.Errorf("no object at %d", *p)
		return 0
	}
	if loc.Class == "Diary" {
		for _, width := range []int{4, 2} {
			n := r.number(p, 2)
			if n == 65535 {
				n = r.number(p, 4)
			}
			if r.err != nil || n > 65536 || uint64(n)*uint64(width) > uint64(len(r.body)-*p) {
				r.err = fmt.Errorf("Diary count at %d", *p)
				return 0
			}
			r.take(p, int(n)*width)
		}
		r.take(p, 4)
		return loc.ArchiveIndex
	}
	*p = start
	return r.reference(p, 0)
}

func actor1161Read(f *sav.File, raw []byte) ([]actor1161Record, *sack1151Reader, map[uint16]uint16, error) {
	_, os, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		return nil, nil, nil, err
	}
	locs, err := f.DocumentObjectLocations()
	if err != nil {
		return nil, nil, nil, err
	}
	actors, err := f.DocumentActorLocations()
	if err != nil {
		return nil, nil, nil, err
	}
	return actor1161ReadLocations(f.Body, actors, locs, os)
}

type actor1161Population struct{ live, rawOnly, books, spellRefs, effects, u68 int }

func actor1161WorldDifferences(roots []actor1161Record, r *sack1151Reader, origins map[uint16]uint16, state *SnapshotSAVDocument, world *sim.World) ([]string, actor1161Population) {
	return actor1161EntityDifferences(roots, r, origins, state, world, world.Entities())
}

func actor1161EntityDifferences(roots []actor1161Record, r *sack1151Reader, origins map[uint16]uint16, state *SnapshotSAVDocument, world *sim.World, observed []sim.Entity) ([]string, actor1161Population) {
	var differences []string
	var pop actor1161Population
	add := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if state == nil || state.Objects == nil {
		return []string{"actor item identities absent"}, pop
	}
	entities := map[uint16]sim.Entity{}
	for _, e := range observed {
		if e.SourceBinding.Class != 0 {
			if _, exists := entities[e.SourceBinding.ArchiveIndex]; exists {
				add("duplicate live actor archive%d", e.SourceBinding.ArchiveIndex)
			}
			entities[e.SourceBinding.ArchiveIndex] = e
		}
	}
	ids := map[uint16]sim.SavedObjectID{}
	for _, b := range state.Objects.Items {
		ids[b.ObjectIndex] = b.ID
	}
	bindings := map[sim.EntityID]uint16{}
	for _, b := range state.Actors {
		bindings[b.EntityID] = b.ObjectIndex
	}
	active := map[[2]uint32]sim.ActiveEffect{}
	for _, e := range world.ActiveEffects() {
		active[[2]uint32{uint32(e.Target), uint32(e.Spell)}] = e
	}
	expectedEffects := 0
	for _, a := range roots {
		prefix := fmt.Sprintf("actor%d %s", a.loc.ArchiveIndex, a.loc.Class)
		e, ok := entities[a.loc.ArchiveIndex]
		delete(entities, a.loc.ArchiveIndex)
		if !ok {
			if a.stage == 1 || a.stage == 0 && a.hp > 0 {
				add("%s eligible raw actor missing live carrier", prefix)
			} else {
				pop.rawOnly++
			}
			continue
		}
		pop.live++
		class := uint8(1)
		if a.loc.Class == "Human" {
			class = 2
		}
		if a.loc.Class == "Humanoid" {
			class = 3
		}
		if e.SourceBinding.Class != class || e.SourceBinding.Identity != binary.LittleEndian.Uint32(r.body[a.loc.Off+29:]) || e.SourceBinding.RuntimeID != binary.LittleEndian.Uint32(r.body[a.loc.Off+12:]) || bindings[e.ID] != origins[a.loc.ArchiveIndex] {
			add("%s source/Document identity differs", prefix)
		}
		pack, ok := world.CarriedStacks(e.ID)
		if !ok {
			add("%s pack absent", prefix)
		}
		if len(pack) != len(a.refs["Inventory"]) {
			add("%s pack length %d raw%d", prefix, len(pack), len(a.refs["Inventory"]))
		}
		for i, ref := range a.refs["Inventory"] {
			if i >= len(pack) {
				break
			}
			row := r.source.rows[ref]
			if row == nil {
				add("%s raw pack item%d absent", prefix, ref)
				continue
			}
			id := ids[origins[ref]]
			if id == 0 {
				add("%s pack item%d lost identity", prefix, ref)
			}
			if !sack1151SameItem(pack[i], r.source.item(row, id)) {
				add("%s pack slot%d item differs", prefix, i)
			}
		}
		worn, ok := world.EquippedItems(e.ID)
		if !ok {
			add("%s equipment absent", prefix)
		}
		expected := make([]uint16, sim.EquipSlots)
		copy(expected, a.refs["Worn"])
		expected[0] = a.refs["HeldWeapon"][0]
		expected[1] = a.refs["HeldShield"][0]
		for slot, ref := range expected {
			if ref == 0 {
				if !worn[slot].Empty() {
					add("%s worn slot%d expected empty", prefix, slot)
				}
				continue
			}
			row := r.source.rows[ref]
			if row == nil {
				add("%s raw worn item%d absent", prefix, ref)
				continue
			}
			id := ids[origins[ref]]
			if id == 0 {
				add("%s worn item%d lost identity", prefix, ref)
			}
			want := r.source.item(row, id)
			want.Count = 1
			if !sack1151SameItem(sim.StackItem(worn[slot], 1), want) {
				add("%s worn slot%d item differs", prefix, slot)
			}
		}
		n := 0
		for _, c := range world.SavedObjects().Containers {
			if c.Owner.Kind != sim.SavedOwnerActorPack || c.Owner.Entity != e.ID {
				continue
			}
			n++
			want := make([]sim.SavedObjectID, len(a.refs["Inventory"]))
			for i, x := range a.refs["Inventory"] {
				want[i] = ids[origins[x]]
			}
			if c.Present != (a.values["HasInventory"] != 0) || c.InsertIndex != a.values["Inventory1C"] || c.Accumulator != int32(a.values["Inventory20"]) || !slices.Equal(c.Items, want) {
				add("%s live container presence/header/identities differ", prefix)
			}
		}
		if n != 1 {
			add("%s live container population%d", prefix, n)
		}
		book := sim.Spellbook{State: sim.BookAbsent}
		mask := uint32(0)
		if a.values["HasSpellbook"] != 0 {
			pop.books++
			book.State = sim.BookPresent
		}
		for slot, ref := range a.refs["Spells"] {
			if ref == 0 {
				continue
			}
			pop.spellRefs++
			s := r.source.rows[ref]
			if s == nil {
				add("%s raw spell%d absent", prefix, ref)
				continue
			}
			id := s.values["S08"]
			if id < 1 || id > 28 || id != uint32(slot+1) {
				add("%s unsupported spell slot%d/id%d", prefix, slot+1, id)
				continue
			}
			mask |= 1 << id
			book.Slots[id-1] = sim.BookSpell{Range: uint8(s.values["S09"]), Defensive: uint8(s.values["S0A"]), ManaCost: uint16(s.values["S0C"])}
		}
		if e.Book != book || e.KnownSpells != mask {
			add("%s live spellbook differs", prefix)
		}
		if a.refs["U68"][0] != 0 {
			pop.u68++
		}
		for _, ref := range a.refs["Effects"] {
			pop.effects++
			expectedEffects++
			raw := r.source.rows[ref]
			if raw == nil {
				add("%s raw Effect%d absent", prefix, ref)
				continue
			}
			id, kind, mode, operand := raw.values["E0C"], raw.values["E3C"], raw.values["E3D"], raw.values["E40"]
			kinds := map[uint32]sim.EffectKind{6: sim.EffectHealth, 8: sim.EffectHealthRegeneration, 11: sim.EffectManaRegeneration, 16: sim.EffectAbsorption, 17: sim.EffectSpeed, 19: sim.EffectScanRange, 21: sim.EffectProtectionFire, 22: sim.EffectProtectionWater, 23: sim.EffectProtectionAir, 24: sim.EffectProtectionEarth, 38: sim.EffectInvisible, 39: sim.EffectBless, 40: sim.EffectCurse}
			got, found := active[[2]uint32{uint32(e.ID), id}]
			if !found || kinds[kind] == sim.EffectNone || got.HasCaster || got.Caster != 0 || got.Kind != kinds[kind] || uint32(got.Mode) != mode || got.Magnitude != int32(int16(operand)) || got.Remaining != uint16(operand>>16) {
				add("%s live Effect%d differs from raw id%d/kind%d/mode%d/operand%d", prefix, ref, id, kind, mode, operand)
			}
			bound := 0
			if state.ActorEffects != nil {
				for _, b := range state.ActorEffects.Rows {
					if b.Entity == e.ID && uint32(b.Spell) == id && b.ObjectIndex == origins[ref] {
						bound++
					}
				}
			}
			if bound != 1 {
				add("%s Effect%d exact binding population%d", prefix, ref, bound)
			}
		}
	}
	if len(entities) != 0 {
		add("unexpected live actor population%d", len(entities))
	}
	if len(active) != expectedEffects {
		add("live effect population%d expected%d", len(active), expectedEffects)
	}
	if state.ActorEffects == nil || state.ActorEffects.Unavailable != "" {
		add("initial actor-effect projection unavailable")
	}
	return differences, pop
}

func actor1161ReadLocations(body []byte, actors []sav.DocumentActorLocation, locs []sav.DocumentObjectLocation, origins []sav.DocumentObjectOrigin) ([]actor1161Record, *sack1151Reader, map[uint16]uint16, error) {
	population, err := unit1158Read(body, actors, locs, origins)
	if err != nil {
		return nil, nil, nil, err
	}
	admission := map[uint16]unit1158Record{}
	for _, row := range population.records {
		admission[row.archive] = row
	}
	r := &sack1151Reader{body: body, byIndex: map[uint16]sav.DocumentObjectLocation{}, byOff: map[int]sav.DocumentObjectLocation{}, source: sackByteSource{rows: map[uint16]*sackByteRecord{}}}
	for _, loc := range locs {
		if loc.ArchiveIndex == 0 || loc.Off < 0 || loc.Off >= len(body) {
			return nil, r, nil, fmt.Errorf("invalid object location")
		}
		if _, ok := r.byIndex[loc.ArchiveIndex]; ok {
			return nil, r, nil, fmt.Errorf("duplicate object identity")
		}
		if _, ok := r.byOff[loc.Off]; ok {
			return nil, r, nil, fmt.Errorf("duplicate object start")
		}
		r.byIndex[loc.ArchiveIndex] = loc
		r.byOff[loc.Off] = loc
	}
	out := make([]actor1161Record, 0, len(actors))
	for _, loc := range actors {
		a := actor1161Record{loc: loc, stage: admission[loc.ArchiveIndex].stage, hp: admission[loc.ArchiveIndex].hp, values: map[string]uint32{}, counts: map[string]uint32{}, refs: map[string][]uint16{}}
		p := loc.Off + 37
		a.counts["Effects"], a.refs["Effects"] = r.list(&p, 0)
		if p != loc.RoutesOff && r.err == nil {
			r.err = fmt.Errorf("actor %d effects endpoint %d != routes %d", loc.ArchiveIndex, p, loc.RoutesOff)
		}
		p = loc.ControlOff + 19
		a.refs["HeldWeapon"] = []uint16{actor1161Reference(r, &p)}
		a.refs["HeldShield"] = []uint16{actor1161Reference(r, &p)}
		if p != loc.StateOff && r.err == nil {
			r.err = fmt.Errorf("actor %d held endpoint %d != state %d", loc.ArchiveIndex, p, loc.StateOff)
		}
		p = loc.StateOff
		n := r.number(&p, 1)
		if n == 255 {
			return nil, r, nil, fmt.Errorf("extended CString not covered by this bounded actor reader")
		}
		r.take(&p, int(n)+55)
		a.refs["U68"] = []uint16{actor1161Reference(r, &p)}
		a.values["HasInventory"] = r.number(&p, 1)
		if a.values["HasInventory"] != 0 {
			a.counts["Inventory"], a.refs["Inventory"] = r.list(&p, 0)
			a.values["Inventory1C"] = r.number(&p, 4)
			a.values["Inventory20"] = r.number(&p, 4)
		}
		a.values["HasSpellbook"] = r.number(&p, 1)
		if a.values["HasSpellbook"] != 0 {
			a.values["SpellsHeader"] = r.number(&p, 4)
			a.counts["Spells"] = r.number(&p, 4)
			n := a.counts["Spells"]
			if r.err != nil || n > 65536 || n > 0 && uint64(n-1)*2 > uint64(len(r.body)-p) {
				return nil, r, nil, fmt.Errorf("book count exceeds bounded tags")
			}
			a.refs["Spells"] = nil
			for k := uint32(1); k < a.counts["Spells"]; k++ {
				a.refs["Spells"] = append(a.refs["Spells"], r.reference(&p, 0))
			}
		}
		for _, name := range []string{"U5C", "U64", "U44", "U40"} {
			a.values[name] = r.number(&p, 4)
		}
		a.values["U48"] = r.number(&p, 1)
		if loc.Class != "Unit" {
			if p != loc.HumanoidXPOff && r.err == nil {
				r.err = fmt.Errorf("actor %d Unit endpoint %d != XP %d", loc.ArchiveIndex, p, loc.HumanoidXPOff)
			}
			r.take(&p, 24)
			for range 12 {
				a.refs["Worn"] = append(a.refs["Worn"], r.reference(&p, 0))
			}
			a.refs["Diary"] = []uint16{actor1161Reference(r, &p)}
		}
		if r.err != nil {
			return nil, r, nil, fmt.Errorf("actor %d %s: %w", loc.ArchiveIndex, loc.Class, r.err)
		}
		out = append(out, a)
	}
	return out, r, population.origins, nil
}

func actor1161DocumentDifferences(roots []actor1161Record, r *sack1151Reader, origins map[uint16]uint16, doc *sav.DocumentData) []string {
	if doc == nil {
		return []string{"missing actor-root Document"}
	}
	var differences []string
	// The reused child comparator owns item roots only. This detached read view
	// removes the unrelated World sack root assertion, never an actual object.
	detached := *doc
	if doc.World != nil {
		world := *doc.World
		detached.World = &world
		world.Sacks = nil
	} else {
		// This comparator checks child rows, with an empty unrelated sack
		// root list. City actor children do not require a live mission world.
		detached.World = &sav.DocumentWorldData{}
	}
	differences = append(differences, sackDocumentDifferences(r.source, origins, &detached)...)
	for _, a := range roots {
		prefix := fmt.Sprintf("actor %d %s", a.loc.ArchiveIndex, a.loc.Class)
		local := origins[a.loc.ArchiveIndex]
		if local == 0 || int(local) > len(doc.Objects) {
			differences = append(differences, prefix+": missing DTO")
			continue
		}
		got := &doc.Objects[local-1]
		if got.Class != a.loc.Class {
			differences = append(differences, prefix+": DTO class differs")
		}
		for name, want := range a.values {
			n := 0
			for _, v := range got.Values {
				if v.Name == name {
					n++
					if v.Value != want {
						differences = append(differences, fmt.Sprintf("%s %s doc=%d raw=%d", prefix, name, v.Value, want))
					}
				}
			}
			if n != 1 {
				differences = append(differences, fmt.Sprintf("%s %s field count %d", prefix, name, n))
			}
		}
		for name, want := range a.counts {
			v, n := uint32(0), 0
			for _, c := range got.Counts {
				if c.Name == name {
					v = c.Count
					n++
				}
			}
			if v != want || n != 1 {
				differences = append(differences, fmt.Sprintf("%s count %s %d/%d raw=%d", prefix, name, v, n, want))
			}
		}
		for name, refs := range a.refs {
			expected := make([]uint16, len(refs))
			for i, x := range refs {
				expected[i] = origins[x]
				if x != 0 && expected[i] == 0 {
					differences = append(differences, fmt.Sprintf("%s missing target %d", prefix, x))
				}
			}
			var actual []uint16
			n := 0
			for _, v := range got.RefSlots {
				if v.Name == name {
					actual = v.Objects
					n++
				}
			}
			if !slices.Equal(actual, expected) || n != 1 {
				differences = append(differences, fmt.Sprintf("%s refs %s doc=%v raw=%v fields=%d", prefix, name, actual, expected, n))
			}
		}
	}
	return differences
}
