package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"sort"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// SAV-UNITPROG-156 owns the four consecutive widths. Amended SAV-HEROXP-063
// owns the six raw dwords, independently of aggregate XP. Only decompression,
// structural starts and the identity-only archive/DTO permutation come from
// production. No decoded block, ActorHoldings or imported population is expected.
type unitCombatRecord struct {
	archive           uint16
	class             string
	off               int
	identity, runtime uint32
	position          [4]byte
	raw               map[string][]byte
	// These independently read scalar bytes only name the existing non-live
	// boundary.
	stage   byte
	hp      int16
	current *milestoneActorCurrent
}

type unit1158Set struct {
	records []unitCombatRecord
	origins map[uint16]uint16
}

var unit1158Blocks = [...]struct {
	name  string
	width int
}{{"UA6", 24}, {"UBE", 22}, {"U114", 24}, {"UD4", 64}, {"H1CC", 24}}

func readUnitCombatExpected(f *sav.File, source []byte) (unit1158Set, error) {
	locations, err := f.DocumentActorLocations()
	if err != nil {
		return unit1158Set{}, err
	}
	objects, err := f.DocumentObjectLocations()
	if err != nil {
		return unit1158Set{}, err
	}
	document, origins, err := sav.DecodeDocumentDataWithOrigins(source)
	if err != nil {
		return unit1158Set{}, err
	}
	out, err := unit1158Read(f.Body, locations, objects, origins)
	if err != nil {
		return out, err
	}
	current, err := milestoneActorCurrentInputs(&document)
	if err != nil {
		return out, err
	}
	for i := range out.records {
		out.records[i].current = current[out.origins[out.records[i].archive]]
	}
	return out, nil
}

func unit1158Read(body []byte, locations []sav.DocumentActorLocation, objects []sav.DocumentObjectLocation, origins []sav.DocumentObjectOrigin) (unit1158Set, error) {
	out := unit1158Set{origins: map[uint16]uint16{}}
	all, pending := map[uint16]sav.DocumentObjectLocation{}, map[uint16]sav.DocumentObjectLocation{}
	for _, object := range objects {
		if object.ArchiveIndex == 0 || all[object.ArchiveIndex].ArchiveIndex != 0 {
			return out, fmt.Errorf("duplicate/zero tagged archive %d", object.ArchiveIndex)
		}
		all[object.ArchiveIndex] = object
		if unit1156Class(object.Class) {
			pending[object.ArchiveIndex] = object
		}
	}
	if len(origins) != len(objects) {
		return out, fmt.Errorf("origin population %d, raw tagged population %d", len(origins), len(objects))
	}
	reverse := map[uint16]bool{}
	for _, origin := range origins {
		if all[origin.ArchiveIndex].ArchiveIndex == 0 || origin.ObjectIndex == 0 || int(origin.ObjectIndex) > len(objects) || out.origins[origin.ArchiveIndex] != 0 || reverse[origin.ObjectIndex] {
			return out, fmt.Errorf("non-bijective origin archive %d -> DTO %d", origin.ArchiveIndex, origin.ObjectIndex)
		}
		out.origins[origin.ArchiveIndex], reverse[origin.ObjectIndex] = origin.ObjectIndex, true
	}
	for _, loc := range locations {
		object, exists := pending[loc.ArchiveIndex]
		if !exists || loc.Class != object.Class || loc.Off != object.Off {
			return out, fmt.Errorf("actor locator %d differs from complete tagged population", loc.ArchiveIndex)
		}
		delete(pending, loc.ArchiveIndex)
		if loc.Off < 0 || loc.Off > len(body)-37 || loc.RawBlocksOff < loc.Off+37 || loc.RawBlocksOff > len(body)-134 || loc.ControlOff < loc.RawBlocksOff+134 || loc.StateOff < loc.ControlOff || loc.StateOff >= len(body) {
			return out, fmt.Errorf("archive %d invalid block/state starts", loc.ArchiveIndex)
		}
		n := int(body[loc.StateOff])
		if n == 255 || n > len(body)-loc.StateOff-1-55 {
			return out, fmt.Errorf("archive %d truncated/unsupported diagnostic state", loc.ArchiveIndex)
		}
		state := body[loc.StateOff+1+n : loc.StateOff+1+n+55]
		r := unitCombatRecord{archive: loc.ArchiveIndex, class: loc.Class, off: loc.Off,
			identity: binary.LittleEndian.Uint32(body[loc.Off+29:]), runtime: binary.LittleEndian.Uint32(body[loc.Off+12:]),
			raw: map[string][]byte{}, stage: state[46], hp: int16(binary.LittleEndian.Uint16(state[16:]))}
		copy(r.position[:], body[loc.Off:loc.Off+4])
		at := loc.RawBlocksOff
		for _, block := range unit1158Blocks[:4] {
			r.raw[block.name] = slices.Clone(body[at : at+block.width])
			at += block.width
		}
		if loc.Class == "Unit" {
			if loc.HumanoidXPOff != 0 {
				return out, fmt.Errorf("archive %d Unit has a Humanoid XP start", loc.ArchiveIndex)
			}
		} else {
			if loc.HumanoidXPOff < loc.StateOff+1+n+55 || loc.HumanoidXPOff > len(body)-24 {
				return out, fmt.Errorf("archive %d invalid Humanoid XP start", loc.ArchiveIndex)
			}
			r.raw["H1CC"] = slices.Clone(body[loc.HumanoidXPOff : loc.HumanoidXPOff+24])
		}
		out.records = append(out.records, r)
	}
	if len(pending) != 0 {
		return out, fmt.Errorf("actor locator omitted %d tagged Unit-family objects", len(pending))
	}
	sort.Slice(out.records, func(i, j int) bool { return out.records[i].archive < out.records[j].archive })
	return out, nil
}

func (want unit1158Set) documentDifferences(state *SnapshotSAVDocument) []string {
	if state == nil || state.Document == nil || state.Unavailable != "" {
		return []string{"complete Unit combat Document unavailable"}
	}
	var differences []string
	population := 0
	for _, object := range state.Document.Objects {
		if unit1156Class(object.Class) {
			population++
		}
	}
	if population != len(want.records) || len(state.Document.Objects) != len(want.origins) {
		differences = append(differences, fmt.Sprintf("retained population actors/objects %d/%d, raw %d/%d", population, len(state.Document.Objects), len(want.records), len(want.origins)))
	}
	seen := map[uint16]bool{}
	for _, r := range want.records {
		index := want.origins[r.archive]
		prefix := fmt.Sprintf("archive %d %s", r.archive, r.class)
		if index == 0 || int(index) > len(state.Document.Objects) || seen[index] {
			differences = append(differences, prefix+": absent/aliased DTO binding")
			continue
		}
		seen[index] = true
		got := state.Document.Objects[index-1]
		if got.Class != r.class {
			differences = append(differences, prefix+": retained class differs")
		}
		for _, block := range unit1158Blocks {
			count := 0
			var value []byte
			for _, field := range got.Raw {
				if field.Name == block.name {
					count++
					value = field.Bytes
				}
			}
			expected, present := r.raw[block.name]
			if !present && count != 0 || present && (count != 1 || !bytes.Equal(value, expected)) {
				differences = append(differences, fmt.Sprintf("%s %s: retained count=%d bytes=%x, raw=%x", prefix, block.name, count, value, expected))
			}
		}
	}
	return differences
}

// This checks the already implemented runtime projections, not a new derive.
// Non-live records keep their complete Document comparison. Their scalar
// stage/HP bytes identify the existing admission boundary without using an
// importer-produced count. Native IDs, DTO indices and archive IDs never mix.
func (want unit1158Set) entityDifferences(entities []sim.Entity, state *SnapshotSAVDocument, contexts ...*Mission) (differences, excluded []string, compared int) {
	subjects, subjectDifferences := newMilestoneActorSubjects(entities, state, contexts...)
	differences = append(differences, subjectDifferences...)
	byArchive := subjects.byArchive
	for _, r := range want.records {
		prefix := fmt.Sprintf("archive %d %s offset %d", r.archive, r.class, r.off)
		e, exists, native, issue := subjects.actor(r.archive, r.off, r.class, r.identity, r.runtime, want.origins[r.archive], r.current)
		if issue != "" {
			differences = append(differences, prefix+": "+issue)
		}
		if !exists {
			owned, ownerIssue := subjects.terminal(r.archive, r.off, r.class, r.identity, r.runtime, want.origins[r.archive], r.current, r.stage, r.hp, r.position[:])
			if ownerIssue != "" {
				differences = append(differences, prefix+": "+ownerIssue)
			}
			if owned {
				excluded = append(excluded, prefix+": exact current terminal owner; raw tuple and complete Document compared")
				continue
			}
			if native || r.stage == 1 || r.stage == 0 && r.hp > 0 {
				differences = append(differences, prefix+": expected live source basis unavailable")
			} else {
				excluded = append(excluded, fmt.Sprintf("%s: no live combat basis (stage=%d signed HP=%d); complete Document compared", prefix, r.stage, r.hp))
			}
			continue
		}
		compared++
		class := uint8(2)
		if r.class == "Unit" {
			class = 1
		}
		if !native && (savedActorClass(e.SourceBinding.Class) != r.class || !e.ActorLoad.Present || e.ActorLoad.Source.Class != class) {
			differences = append(differences, prefix+": live class/basis presence differs")
		}
		bound := 0
		if state != nil {
			for _, binding := range state.Actors {
				if binding.EntityID == e.ID || binding.ObjectIndex == want.origins[r.archive] {
					bound++
					if binding.EntityID != e.ID || binding.ObjectIndex != want.origins[r.archive] || binding.Retired {
						differences = append(differences, prefix+": archive/DTO/entity projection binding differs")
					}
				}
			}
		}
		if bound != 1 {
			differences = append(differences, prefix+": absent/duplicate live DTO projection")
		}
		s := e.ActorLoad.Source
		// A slot whose saved experience lies above its level's band loads at
		// the level that experience implies; the expected blocks carry that
		// repair, which is the one place the live words differ from the raw.
		expectedRaw := map[string][]byte{}
		for name, raw := range r.raw {
			expectedRaw[name] = raw
		}
		if xp, present := r.raw["H1CC"]; !native && present && class == 2 {
			attack, base := bytes.Clone(r.raw["UA6"]), bytes.Clone(r.raw["U114"])
			for i := 1; i <= 5; i++ {
				stored := int32(int16(binary.LittleEndian.Uint16(attack[2+2*i:])))
				fixed := data.RepairSkillLevel(stored, int32(binary.LittleEndian.Uint32(xp[4*i:])))
				if fixed != stored {
					wasBase := int32(int16(binary.LittleEndian.Uint16(base[2+2*i:])))
					binary.LittleEndian.PutUint16(attack[2+2*i:], uint16(fixed))
					binary.LittleEndian.PutUint16(base[2+2*i:], uint16(wasBase+fixed-stored))
				}
			}
			expectedRaw["UA6"], expectedRaw["U114"] = attack, base
		}
		if native {
			rules := sim.Rules{}
			if subjects.mission != nil && subjects.mission.World != nil {
				rules = subjects.mission.World.Rules()
			}
			expectedRaw = unitNativePartyCombatRaw(expectedRaw, r.current, r.class, rules)
			differences = append(differences, milestoneNativeCombatDifferences(prefix, e, r.current, expectedRaw)...)
		} else {
			for _, block := range []sav.DocumentRawData{{Name: "UA6", Bytes: s.Attack[:]}, {Name: "UBE", Bytes: s.Defence[:]}, {Name: "U114", Bytes: s.Base[:]}, {Name: "UD4", Bytes: s.Modifier[:]}} {
				if !bytes.Equal(block.Bytes, expectedRaw[block.Name]) {
					differences = append(differences, prefix+": live source "+block.Name+" bytes differ")
				}
			}
		}
		check := func(name string, got, expected int32) {
			if got != expected {
				differences = append(differences, fmt.Sprintf("%s %s: live=%d raw=%d", prefix, name, got, expected))
			}
		}
		a, d, m := expectedRaw["UA6"], r.raw["UBE"], r.raw["UD4"]
		word := func(b []byte, at int) int32 { return int32(int16(binary.LittleEndian.Uint16(b[at:]))) }
		// A Human whose stored level sits at the original cap with a base plus
		// bonus above it fights at the higher level; to-hit and damage base
		// carry the active skill's extra terms. The raw block keeps the
		// original's values and is compared as bytes above.
		lifted := [6]int32{}
		toHitLift, damageLift := int32(0), int32(0)
		if !native && class == 2 {
			for i := 1; i <= 5; i++ {
				stored := word(a, 2+2*i)
				if stored != 100 {
					continue
				}
				if eff := min(min(word(expectedRaw["U114"], 2+2*i), 100)+word(m, 20+2*i), 255); eff > stored {
					lifted[i] = eff - stored
					if int32(a[16]) == int32(i) {
						toHitLift, damageLift = 3*(eff-stored), eff/5-stored/5
					}
				}
			}
		}
		for i := range 6 {
			check(fmt.Sprintf("Skill[%d]", i), e.Skill[i], word(a, 2+2*i)+lifted[i])
			if xp, present := r.raw["H1CC"]; present {
				expected := int32(binary.LittleEndian.Uint32(xp[4*i:]))
				if !native {
					check(fmt.Sprintf("basis.SkillXP[%d]", i), int32(s.SkillXP[i]), expected)
				}
				check(fmt.Sprintf("entity.SkillXP[%d]", i), e.SkillXP[i], expected)
			}
		}
		for _, field := range []struct {
			name          string
			got, expected int32
		}{
			{"ToHit", e.ToHit, word(a, 0) + toHitLift}, {"Defence", e.Defence, word(d, 0)}, {"Absorption", e.Absorption, word(d, 2)},
			{"DamageBase", e.DamageBase, int32(a[14]) + damageLift}, {"DamageSpread", e.DamageSpread, int32(a[15])}, {"XPSlot", int32(e.XPSlot), int32(a[16])},
			{"SecondBase", int32(e.SecondBase), int32(a[17])}, {"SecondSpread", int32(e.SecondSpread), int32(a[18])},
			{"ElementalBase", int32(e.SecondaryDamage.Base), int32(a[19])}, {"ElementalSpread", int32(e.SecondaryDamage.Spread), int32(a[20])},
			{"HealthRegeneration", e.HealthRegeneration, word(m, 10)}, {"ManaRegeneration", e.ManaRegeneration, word(m, 14)},
		} {
			check(field.name, field.got, field.expected)
		}
		// HERO-DMG2-029: only an active third damage component consumes the
		// raw selector; the full byte is still compared in Document and basis.
		if a[19] != 0 || a[20] != 0 {
			if a[21] < 1 || a[21] > 5 {
				differences = append(differences, prefix+": unsupported active elemental selector")
			} else {
				check("ElementalSelector", int32(e.SecondaryDamage.Selector), [...]int32{0, 3, 2, 1, 4}[a[21]-1])
			}
		}
		for i := range 5 {
			check(fmt.Sprintf("Protection[%d]", i), e.Protection[i], word(d, 6+2*i))
			check(fmt.Sprintf("Resistance[%d]", i), int32(e.Resistance[i]), int32(d[17+i]))
		}
	}
	if n := subjects.unexpectedNative(); n != 0 {
		differences = append(differences, fmt.Sprintf("unexpected native actor subject population%d", n))
	}
	for archive := range byArchive {
		differences = append(differences, fmt.Sprintf("live archive %d absent from raw Unit population", archive))
	}
	return
}

func unitNativePartyCombatRaw(raw map[string][]byte, current *milestoneActorCurrent, class string, rules sim.Rules) map[string][]byte {
	if current == nil || !current.NativeParty || current.SourceBound || current.SourceClass != 0 || class == "Unit" || len(raw["U114"]) != 24 || len(raw["H1CC"]) != 24 {
		return raw
	}
	out := make(map[string][]byte, len(raw))
	for name, value := range raw {
		out[name] = value
	}
	base := bytes.Clone(raw["U114"])
	for slot := 1; slot <= 5; slot++ {
		xp := int32(binary.LittleEndian.Uint32(raw["H1CC"][4*slot:]))
		level := int32(int16(binary.LittleEndian.Uint16(base[2+2*slot:])))
		if xp <= 0 || level > 0 && xp == rules.SkillXP(level)+1 {
			continue
		}
		covered := rules.SkillCap()
		for rank := int32(0); rank <= rules.SkillCap(); rank++ {
			if rules.SkillXP(rank) >= xp {
				covered = rank
				break
			}
		}
		if covered > level {
			binary.LittleEndian.PutUint16(base[2+2*slot:], uint16(covered))
		}
	}
	out["U114"] = base
	return out
}
