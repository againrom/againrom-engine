package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type dead1163Source struct {
	roots      []uint16                 // full counted sequence, including repeated references
	actors     []sim.OriginalDeadSource // distinct archives in first-occurrence order
	origins    map[uint16]uint16
	scalars    unit1156Set
	actorRoots []actor1161Record
	children   *sack1151Reader
}

// SAV-DEADLOAD-124/126/129, SAV-STREAM-013 and SAV-UNITPROG-156. Only
// decompression, starts and a bijective archive/DTO permutation are shared.
// Independent1154 finishes the Player stream,1156 reads actor scalars,
// 1161 reads variable actor roots/tails and1151 reads item children from Body.
func dead1163Expected(f *sav.File, raw []byte) (dead1163Source, error) {
	var out dead1163Source
	players, err := players1154Expected(f)
	if err != nil {
		return out, err
	}
	loc, err := f.DocumentDeadRootLocation()
	if err != nil {
		return out, err
	}
	if loc.CountOff != players.end {
		return out, fmt.Errorf("dead count start%d != independent Player end%d", loc.CountOff, players.end)
	}
	scalars, err := unit1156Expected(f, raw)
	if err != nil {
		return out, err
	}
	roots, children, origins, err := actor1161Read(f, raw)
	if err != nil {
		return out, err
	}
	out.origins = origins
	out.scalars, out.actorRoots, out.children = scalars, roots, children
	byArchive := map[uint16]unit1156Record{}
	tails := map[uint16]actor1161Record{}
	for _, a := range scalars.records {
		byArchive[a.archive] = a
	}
	for _, a := range roots {
		tails[a.loc.ArchiveIndex] = a
	}
	r := player1154Reader{body: f.Body}
	p := loc.CountOff
	n := r.number(&p, 4)
	if r.err != nil {
		return out, r.err
	}
	if uint64(n)*2 > uint64(len(f.Body)-p) || uint64(n) != uint64(len(loc.RefOffs)) {
		return out, fmt.Errorf("raw dead count%d vs located%d", n, len(loc.RefOffs))
	}
	seen := map[uint16]bool{}
	last := p - 1
	for _, off := range loc.RefOffs {
		if off < p || off <= last {
			return out, fmt.Errorf("dead tag starts not strictly ordered")
		}
		last = off
		archive := r.referenceAt(off, children.byOff, children.byIndex)
		if r.err != nil {
			return out, r.err
		}
		a, exists := byArchive[archive]
		if archive == 0 || !exists {
			return out, fmt.Errorf("dead root lacks Unit-family archive%d", archive)
		}
		out.roots = append(out.roots, archive)
		if seen[archive] {
			continue
		}
		seen[archive] = true
		tail := tails[archive]
		position := a.raw["Block12"]
		d := sim.OriginalDeadSource{ArchiveIndex: archive, Class: map[string]uint8{"Unit": 1, "Human": 2, "Humanoid": 3}[a.class],
			Identity: a.values["Identity"]}
		// RuntimeID belongs to State, not the identity metadata.
		d.MapUnitID, d.TerrainKey, d.OwnerKey = uint16(a.values["T08"]), binary.LittleEndian.Uint32(position[8:]), a.values["Reference"]
		d.State = sim.DeadActorState{RuntimeID: a.values["RuntimeID"], Cell: binary.LittleEndian.Uint16(position), FineX: position[4], FineY: position[5], Stage: tail.stage, HP: tail.hp, Timer: int8(a.values["U6C"])}
		d.References = [5]uint32{tail.values["U5C"], tail.values["U64"], tail.values["U44"], 0, tail.values["U40"]}
		if refs := tail.refs["U68"]; len(refs) != 1 {
			return out, fmt.Errorf("dead U68 has no raw reference slot")
		} else if refs[0] != 0 {
			other, ok := byArchive[refs[0]]
			if !ok {
				return out, fmt.Errorf("dead U68 has non-actor target")
			}
			d.References[3] = other.values["Identity"]
		}
		d.ContainerPresent = tail.values["HasInventory"] != 0
		d.ContainerTail = [2]uint32{tail.values["Inventory1C"], tail.values["Inventory20"]}
		if refs := tail.refs["HeldWeapon"]; len(refs) != 1 {
			return out, fmt.Errorf("dead held weapon has no raw slot")
		} else if refs[0] != 0 {
			d.HeldWeapon, err = dead1163Weapon(children.source.rows[refs[0]])
			if err != nil {
				return out, err
			}
		}
		out.actors = append(out.actors, d)
	}
	return out, nil
}

func dead1163Weapon(r *sackByteRecord) (sim.OriginalDeadWeapon, error) {
	if r == nil || r.class != "Weapon" || r.counts["Effects"] != 0 || len(r.raw["Block12"]) != 12 || len(r.raw["W52"]) != 24 || len(r.raw["W6A"]) != 22 {
		return sim.OriginalDeadWeapon{}, fmt.Errorf("dead weapon outside bounded raw shape")
	}
	p, v := r.raw["Block12"], r.values
	w := sim.OriginalDeadWeapon{Present: true, ArchiveIndex: r.index, Cell: binary.LittleEndian.Uint16(p), PackedCell: binary.LittleEndian.Uint16(p[2:]), FineX: p[4], FineY: p[5], PositionU06: binary.LittleEndian.Uint16(p[6:]), TerrainKey: binary.LittleEndian.Uint32(p[8:]), RuntimeID: v["RuntimeID"], T0C: uint8(v["T0C"]), T0E: uint16(v["T0E"]), T08: v["T08"], T18: uint16(v["T18"]), T1C: v["T1C"], Identity: v["Identity"], Reference: v["Reference"], F40: uint16(v["F40"]), F42: uint16(v["F42"]), F44: uint8(v["F44"]), F45: uint8(v["F45"]), F46: uint8(v["F46"]), F47: uint8(v["F47"]), F48: uint16(v["F48"]), F4A: uint16(v["F4A"]), W50: uint8(v["W50"])}
	copy(w.W52[:], r.raw["W52"])
	copy(w.W6A[:], r.raw["W6A"])
	return w, nil
}

func (want dead1163Source) documentDifferences(state *SnapshotSAVDocument) []string {
	if state == nil || state.Document == nil {
		return []string{"dead roots lack complete Document"}
	}
	var expected []uint16
	for _, archive := range want.roots {
		index := want.origins[archive]
		if index == 0 || int(index) > len(state.Document.Objects) {
			return []string{"dead root lost archive/DTO binding"}
		}
		expected = append(expected, index)
	}
	if !slices.Equal(expected, state.Document.DeadActors) {
		return []string{fmt.Sprintf("dead root sequence: Document%v raw%v", state.Document.DeadActors, expected)}
	}
	var differences []string
	for _, a := range want.actors {
		o := state.Document.Objects[want.origins[a.ArchiveIndex]-1]
		identity, err := savedStructureValue(&o, "Identity")
		if err != nil || identity != a.Identity || map[string]uint8{"Unit": 1, "Human": 2, "Humanoid": 3}[o.Class] != a.Class {
			differences = append(differences, fmt.Sprintf("dead archive%d lost class/identity", a.ArchiveIndex))
		}
	}
	// Reuse the landed independent readers for the complete selected actor
	// fields and every reachable child, including non-live dead subjects.
	differences = append(differences, want.scalars.documentDifferences(state)...)
	differences = append(differences, actor1161DocumentDifferences(want.actorRoots, want.children, want.origins, state.Document)...)
	return differences
}

func (want dead1163Source) liveDifferences(got []sim.OriginalDeadRecord, entities []sim.Entity) []string {
	var differences []string
	fail := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if len(got) != len(want.actors) {
		fail("dead distinct population%d != raw%d", len(got), len(want.actors))
	}
	byArchive := map[uint16]sim.OriginalDeadRecord{}
	byNative := map[sim.EntityID]bool{}
	live := map[sim.EntityID]sim.Entity{}
	for _, e := range entities {
		live[e.ID] = e
	}
	for i, d := range got {
		if _, duplicate := byArchive[d.Source.ArchiveIndex]; duplicate || byNative[d.ID] {
			fail("dead archive/native identity duplicated")
		}
		byArchive[d.Source.ArchiveIndex], byNative[d.ID] = d, true
		if i < len(want.actors) && d.Source.ArchiveIndex != want.actors[i].ArchiveIndex {
			fail("dead first-occurrence order differs at%d", i)
		}
	}
	for _, a := range want.actors {
		d, ok := byArchive[a.ArchiveIndex]
		if !ok {
			fail("dead archive%d omitted", a.ArchiveIndex)
			continue
		}
		if d.Source != a {
			fail("dead archive%d source tuple differs", a.ArchiveIndex)
		}
		if d.Current != a.State {
			fail("dead archive%d initial current tuple differs", a.ArchiveIndex)
		}
		e, present := live[d.ID]
		// Disclosed native corpse admission, not original occupancy evidence.
		if present != (a.MapUnitID != 0 && a.State.Stage < 5) {
			fail("dead archive%d materialization policy differs", a.ArchiveIndex)
		}
		if present && e.MapUnitID != a.MapUnitID {
			fail("dead archive%d joined the wrong authored identity", a.ArchiveIndex)
		}
	}
	return differences
}

// Snapshot may renumber objects. Compare each retained source identity in
// raw root order, never an obsolete original archive/DTO permutation.
func (want dead1163Source) currentRootDifferences(state *SnapshotSAVDocument) []string {
	if state == nil || state.Document == nil || len(state.Document.DeadActors) != len(want.roots) {
		return []string{"current dead-root population differs"}
	}
	keys := map[uint16]uint32{}
	for _, a := range want.actors {
		keys[a.ArchiveIndex] = a.Identity
	}
	var differences []string
	for i, archive := range want.roots {
		index := state.Document.DeadActors[i]
		if index == 0 || int(index) > len(state.Document.Objects) {
			differences = append(differences, "current dead root lacks object")
			continue
		}
		key, err := savedStructureValue(&state.Document.Objects[index-1], "Identity")
		if err != nil || key != keys[archive] {
			differences = append(differences, fmt.Sprintf("current dead root%d identity differs", i))
		}
	}
	return differences
}
