package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"sort"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The oracle owns these wire offsets. Production supplies decompression,
// tagged-object/run starts and an identity-only archive-to-DTO permutation.
// No ActorHoldings, ActorGraph, decoded statistics or ownership is an oracle.
// SAV-TOKEN-034, SAV-UNITPROG-156, amended SAV-UNITFLD-049 and SAV-REGENWIRE-532.
type unit1156Record struct {
	archive uint16
	class   string
	off     int
	values  map[string]uint32
	raw     map[string][]byte
	name    string
	current *milestoneActorCurrent
}

type unitScalarSet struct {
	records []unit1156Record
	origins map[uint16]uint16
}

var unit1156Stats = [...]string{"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen"}

func unit1156Class(class string) bool {
	return class == "Unit" || class == "Human" || class == "Humanoid"
}

func unitScalarExpected(f *sav.File, source []byte) (unitScalarSet, error) {
	locations, err := f.DocumentActorLocations()
	if err != nil {
		return unitScalarSet{}, err
	}
	objects, err := f.DocumentObjectLocations()
	if err != nil {
		return unitScalarSet{}, err
	}
	// Decoded ordinary scalar values are discarded. The supplement supplies
	// only exact identity/admission/presence; ordinary Body remains the oracle.
	document, origins, err := sav.DecodeDocumentDataWithOrigins(source)
	if err != nil {
		return unitScalarSet{}, err
	}
	out, err := unit1156Read(f.Body, locations, objects, origins)
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

func unit1156Read(body []byte, locations []sav.DocumentActorLocation, objects []sav.DocumentObjectLocation, origins []sav.DocumentObjectOrigin) (unitScalarSet, error) {
	out := unitScalarSet{origins: map[uint16]uint16{}}
	byArchive := map[uint16]sav.DocumentObjectLocation{}
	wanted := map[uint16]sav.DocumentObjectLocation{}
	for _, object := range objects {
		if object.ArchiveIndex == 0 || byArchive[object.ArchiveIndex].ArchiveIndex != 0 {
			return out, fmt.Errorf("duplicate/zero raw object archive %d", object.ArchiveIndex)
		}
		byArchive[object.ArchiveIndex] = object
		if unit1156Class(object.Class) {
			wanted[object.ArchiveIndex] = object
		}
	}
	if len(origins) != len(objects) {
		return out, fmt.Errorf("origin population %d, raw tagged objects %d", len(origins), len(objects))
	}
	reverse := map[uint16]uint16{}
	for _, origin := range origins {
		if byArchive[origin.ArchiveIndex].ArchiveIndex == 0 || origin.ObjectIndex == 0 || int(origin.ObjectIndex) > len(objects) || out.origins[origin.ArchiveIndex] != 0 || reverse[origin.ObjectIndex] != 0 {
			return out, fmt.Errorf("non-bijective origin archive %d -> DTO %d", origin.ArchiveIndex, origin.ObjectIndex)
		}
		out.origins[origin.ArchiveIndex], reverse[origin.ObjectIndex] = origin.ObjectIndex, origin.ArchiveIndex
	}
	for _, loc := range locations {
		object, exists := wanted[loc.ArchiveIndex]
		if !exists || object.Off != loc.Off || object.Class != loc.Class {
			return out, fmt.Errorf("actor locator %d differs from tagged-object population", loc.ArchiveIndex)
		}
		delete(wanted, loc.ArchiveIndex)
		if loc.Off < 0 || loc.Off > len(body)-37 || loc.ControlOff < loc.Off+37 || loc.ControlOff > len(body)-19 || loc.StateOff < loc.ControlOff+23 || loc.StateOff >= len(body) {
			return out, fmt.Errorf("actor %d has invalid Token/control/state starts", loc.ArchiveIndex)
		}
		r := unit1156Record{archive: loc.ArchiveIndex, class: loc.Class, off: loc.Off, values: map[string]uint32{}, raw: map[string][]byte{}}
		r.raw["Block12"] = slices.Clone(body[loc.Off : loc.Off+12])
		for _, field := range []struct {
			name       string
			off, width int
		}{{"RuntimeID", 12, 4}, {"T0C", 16, 1}, {"T0E", 17, 2}, {"T08", 19, 4}, {"T18", 23, 2}, {"T1C", 25, 4}, {"Identity", 29, 4}, {"Reference", 33, 4}} {
			r.values[field.name] = unit1156Unsigned(body[loc.Off+field.off:], field.width)
		}
		for i, name := range []string{"U49", "U4A", "U4B", "U4C"} {
			r.values[name] = uint32(body[loc.ControlOff+i])
		}
		for i, name := range []string{"U50", "U54", "U58"} {
			at := loc.ControlOff + 4 + i*4
			r.raw[name] = slices.Clone(body[at : at+4])
		}
		for i, name := range []string{"U60", "U61", "U6C"} {
			r.values[name] = uint32(body[loc.ControlOff+16+i])
		}
		n := int(body[loc.StateOff])
		if n == 255 || n > len(body)-loc.StateOff-1-55 {
			return out, fmt.Errorf("actor %d has truncated/unsupported CString or state", loc.ArchiveIndex)
		}
		r.name = string(body[loc.StateOff+1 : loc.StateOff+1+n])
		state := body[loc.StateOff+1+n : loc.StateOff+1+n+55]
		for i, name := range unit1156Stats {
			r.values[name] = uint32(binary.LittleEndian.Uint16(state[i*2:]))
		}
		for _, field := range []struct {
			name       string
			off, width int
		}{{"UA2", 28, 1}, {"UA3", 29, 1}, {"UA0", 30, 2}, {"UA4", 32, 2}, {"U12C", 34, 1}, {"U130", 35, 4}, {"U134", 39, 1}, {"U135", 40, 1}, {"U136", 41, 1}, {"U138", 42, 4}, {"Stage", 46, 1}, {"U148", 47, 4}, {"U144", 51, 4}} {
			r.values[field.name] = unit1156Unsigned(state[field.off:], field.width)
		}
		out.records = append(out.records, r)
	}
	if len(wanted) != 0 {
		return out, fmt.Errorf("actor locator omitted %d tagged Unit-family objects", len(wanted))
	}
	sort.Slice(out.records, func(i, j int) bool { return out.records[i].archive < out.records[j].archive })
	return out, nil
}

func unit1156Unsigned(b []byte, width int) uint32 {
	switch width {
	case 1:
		return uint32(b[0])
	case 2:
		return uint32(binary.LittleEndian.Uint16(b))
	default:
		return binary.LittleEndian.Uint32(b)
	}
}

func (want unitScalarSet) documentDifferences(state *SnapshotSAVDocument) []string {
	if state == nil || state.Document == nil || state.Unavailable != "" {
		return []string{"complete Unit document unavailable"}
	}
	var differences []string
	population := 0
	for _, object := range state.Document.Objects {
		if unit1156Class(object.Class) {
			population++
		}
	}
	if population != len(want.records) {
		differences = append(differences, fmt.Sprintf("retained actor population %d, raw %d", population, len(want.records)))
	}
	seen := map[uint16]bool{}
	for _, r := range want.records {
		index := want.origins[r.archive]
		prefix := fmt.Sprintf("archive %d %s", r.archive, r.class)
		if index == 0 || int(index) > len(state.Document.Objects) || seen[index] {
			differences = append(differences, prefix+": absent/duplicate retained object binding")
			continue
		}
		seen[index] = true
		got := state.Document.Objects[index-1]
		if got.Class != r.class {
			differences = append(differences, prefix+": retained class differs")
		}
		for _, name := range spell1152Keys(r.values) {
			count := 0
			var value uint32
			for _, field := range got.Values {
				if field.Name == name {
					count++
					value = field.Value
				}
			}
			if count != 1 || value != r.values[name] {
				differences = append(differences, fmt.Sprintf("%s %s: retained count=%d value=%#x, raw=%#x", prefix, name, count, value, r.values[name]))
			}
		}
		for _, name := range spell1152Keys(r.raw) {
			count := 0
			var value []byte
			for _, field := range got.Raw {
				if field.Name == name {
					count++
					value = field.Bytes
				}
			}
			if count != 1 || !bytes.Equal(value, r.raw[name]) {
				differences = append(differences, prefix+" "+name+": retained raw bytes differ")
			}
		}
		count, name := 0, ""
		for _, field := range got.Texts {
			if field.Name == "Name" {
				count++
				name = field.Value
			}
		}
		if count != 1 || name != r.name {
			differences = append(differences, prefix+": retained CString differs")
		}
	}
	return differences
}

// Raw stage and HP require a carrier for living stage-0 and supported stage-1
// records. A missing importer binding cannot change that population. Late-dead,
// terminal and other raw-only records remain in Document comparisons and are
// reported individually; native ALM-only entities are not invented SAV rows.
func (want unitScalarSet) worldDifferences(world *sim.World, manifest *SnapshotActorManifest, contexts ...*Mission) (differences, excluded []string, compared int) {
	return want.entityDifferences(world.Entities(), manifest, contexts...)
}

func (want unitScalarSet) entityDifferences(entities []sim.Entity, manifest *SnapshotActorManifest, contexts ...*Mission) (differences, excluded []string, compared int) {
	var state *SnapshotSAVDocument
	if len(contexts) == 1 && contexts[0] != nil {
		state = contexts[0].savedDocument
	}
	subjects, subjectDifferences := newMilestoneActorSubjects(entities, state, contexts...)
	differences = append(differences, subjectDifferences...)
	byArchive := subjects.byArchive
	for _, r := range want.records {
		prefix := fmt.Sprintf("archive %d %s offset %d", r.archive, r.class, r.off)
		e, exists, native, issue := subjects.actor(r.archive, r.off, r.class, r.values["Identity"], r.values["RuntimeID"], want.origins[r.archive], r.current)
		if issue != "" {
			differences = append(differences, prefix+": "+issue)
		}
		if !exists {
			stage, hp := r.values["Stage"], int16(r.values["Health"])
			owned, ownerIssue := subjects.terminal(r.archive, r.off, r.class, r.values["Identity"], r.values["RuntimeID"], want.origins[r.archive], r.current, byte(stage), hp, r.raw["Block12"][:4])
			if ownerIssue != "" {
				differences = append(differences, prefix+": "+ownerIssue)
			}
			if owned {
				excluded = append(excluded, prefix+": exact current terminal owner; raw tuple and complete Document compared")
				continue
			}
			if native || stage == 1 || stage == 0 && hp > 0 {
				differences = append(differences, fmt.Sprintf("%s: expected live scalar SourceBinding entity missing (stage=%d, signed HP=%d, runtime=%d)", prefix, stage, hp, r.values["RuntimeID"]))
			} else {
				excluded = append(excluded, fmt.Sprintf("%s: no scalar SourceBinding entity (stage=%d, signed HP=%d, runtime=%d); retained Document still compared", prefix, stage, hp, r.values["RuntimeID"]))
			}
			continue
		}
		delete(byArchive, r.archive)
		compared++
		if native {
			differences = append(differences, unit1156NativeDifferences(r, e, r.current, manifest)...)
			continue
		}
		check := func(name string, got, expected uint32) {
			if got != expected {
				differences = append(differences, fmt.Sprintf("%s %s: World=%#x raw=%#x", prefix, name, got, expected))
			}
		}
		if savedActorClass(e.SourceBinding.Class) != r.class || !e.ActorLoad.Present || e.ActorLoad.Source.Class == 0 {
			differences = append(differences, prefix+": live class/basis presence differs")
		}
		v, s, b := r.values, e.ActorLoad.Source, e.SourceBinding
		classAndPresence := uint32(b.ClassFlags)
		if e.OffMap {
			classAndPresence |= sav.ActorOffMapFlag
		}
		for i, name := range unit1156Stats {
			check("basis."+name, uint32(s.Stats[i]), v[name])
		}
		for _, field := range []struct {
			name string
			got  uint32
		}{
			{"Identity", b.Identity}, {"RuntimeID", b.RuntimeID}, {"T0C", uint32(b.TokenRow)}, {"T0E", uint32(b.TypeID)},
			{"U4B", uint32(b.Face)}, {"U4C", classAndPresence}, {"U148", b.DisplayBacking},
			{"U49", uint32(e.TokenSize)}, {"UA0", uint32(s.ManaFloor)}, {"UA4", uint32(s.Sight)}, {"U130", s.Experience},
			{"U12C", uint32(s.Reach)}, {"U134", uint32(s.AttackCharge)}, {"U135", uint32(s.AttackRelax)},
			{"UA2", uint32(e.HealthHundredths)}, {"UA3", uint32(e.ManaHundredths)},
		} {
			check(field.name, field.got, v[field.name])
		}
		check("MapUnitID (T08 low word)", uint32(e.MapUnitID), v["T08"]&0xffff)
		check("domain (U4A codes 1/2/3)", uint32(e.Domain)+1, v["U4A"])
		cell := binary.LittleEndian.Uint16(r.raw["Block12"])
		check("cell X", uint32(e.X), uint32(cell&255))
		check("cell Y", uint32(e.Y), uint32(cell>>8))
		check("ScanRange (UA4 high byte)", uint32(e.ScanRange), v["UA4"]>>8)
		check("Fighter (U4C bit2)", boolUint1156(s.Fighter), boolUint1156(v["U4C"]&4 == 0))
		for _, field := range []struct {
			name string
			got  uint16
		}{
			{"Reaction", uint16(e.Reaction)}, {"Mind", uint16(e.Mind)}, {"Spirit", uint16(e.Spirit)},
			{"Speed", uint16(e.HumanMovement.RawSpeed)}, {"U8E", uint16(e.ActorLoad.OwnWeight)}, {"U90", uint16(e.Load)}, {"Capacity", uint16(e.Capacity)},
			{"Health", uint16(e.HP)}, {"HealthMax", uint16(e.MaxHP)}, {"Mana", uint16(e.Mana)}, {"ManaMax", uint16(e.MaxMana)},
			{"HealthRegen", uint16(e.HealthRegenPeriod)}, {"ManaRegen", uint16(e.ManaRegenPeriod)},
			{"U12C", uint16(e.Reach)}, {"U134", uint16(e.AttackCharge)}, {"U135", uint16(e.AttackRelax)},
		} {
			check("entity."+field.name, uint32(field.got), v[field.name])
		}
		check("Stage", uint32(e.Decay), v["Stage"])
		if v["Stage"] == 1 {
			check("dying timer U6C", uint32(e.Dwell), v["U6C"])
		}
		if manifest != nil {
			count := 0
			for _, row := range manifest.Actors {
				if row.ID == e.ID {
					count++
					if row.Name != r.name {
						differences = append(differences, prefix+": live manifest Name differs")
					}
				}
			}
			if count != 1 {
				differences = append(differences, prefix+": missing/duplicate manifest Name")
			}
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

func unit1156NativeDifferences(r unit1156Record, e sim.Entity, current *milestoneActorCurrent, manifest *SnapshotActorManifest) []string {
	prefix := fmt.Sprintf("archive %d %s offset %d", r.archive, r.class, r.off)
	var differences []string
	check := func(name string, got, expected uint32) {
		if got != expected {
			differences = append(differences, fmt.Sprintf("%s %s: World=%#x raw=%#x", prefix, name, got, expected))
		}
	}
	unavailable := func(name string) {
		differences = append(differences, prefix+": native "+name+" current scalar carrier unavailable; complete Document compared")
	}
	v, b, mask := r.values, e.NativeBasis, current.Basis
	if err := b.Validate(); err != nil {
		differences = append(differences, prefix+": native basis invalid: "+err.Error())
	}
	if b.BasePresent != mask.BasePresent || b.BaseKnown != mask.BaseKnown || b.ModifierPresent != mask.ModifierPresent || b.ModifierKnown != mask.ModifierKnown || b.BodyPresent != mask.BodyPresent || b.BodyKnown != mask.BodyKnown {
		differences = append(differences, prefix+": native basis presence/mask differs")
	}
	if b.ScalarsPresent != mask.ScalarsPresent || b.ScalarKnown != mask.ScalarKnown || b.BlockPresent != mask.BlockPresent || b.BlockKnown != mask.BlockKnown {
		differences = append(differences, prefix+": native scalar presence/mask differs")
	}
	scalar := func(name string, index int, expected uint32) {
		if !b.ScalarsPresent || b.ScalarKnown&(uint32(1)<<index) == 0 || mask.ScalarKnown&(uint32(1)<<index) == 0 {
			unavailable(name)
			return
		}
		check("native "+name, b.Scalars[index], expected)
	}
	if b.BodyPresent && b.BodyKnown && mask.BodyPresent && mask.BodyKnown {
		check("native Body", uint32(b.Body), v["Body"])
	} else {
		unavailable("Body")
	}
	// The exact subject join compared raw Identity/RuntimeID to registry source
	// metadata and the independently restored ordinary binding, not EntityID.
	for _, field := range []struct {
		name string
		got  uint16
	}{
		{"Reaction", uint16(e.Reaction)}, {"Mind", uint16(e.Mind)}, {"Spirit", uint16(e.Spirit)},
		{"Speed", uint16(e.Speed)}, {"U90", uint16(e.Load)}, {"Capacity", uint16(e.Capacity)},
		{"Health", uint16(e.HP)}, {"HealthMax", uint16(e.MaxHP)}, {"Mana", uint16(e.Mana)}, {"ManaMax", uint16(e.MaxMana)},
		{"HealthRegen", uint16(e.HealthRegenPeriod)}, {"ManaRegen", uint16(e.ManaRegenPeriod)}, {"T0E", uint16(e.TypeID)},
	} {
		check(field.name, uint32(field.got), v[field.name])
	}
	if e.ActorLoad.Present {
		check("U8E", uint32(uint16(e.ActorLoad.OwnWeight)), v["U8E"])
	} else {
		scalar("U8E", sim.ScalarU8E, v["U8E"])
	}
	for _, field := range []struct {
		name string
		got  uint32
	}{
		{"U49", uint32(e.TokenSize)}, {"UA2", uint32(e.HealthHundredths)}, {"UA3", uint32(e.ManaHundredths)},
		{"U12C", uint32(e.Reach)}, {"U134", uint32(uint8(e.AttackCharge))}, {"U135", uint32(uint8(e.AttackRelax))},
	} {
		check(field.name, field.got, v[field.name])
	}
	check("MapUnitID", uint32(e.MapUnitID), v["T08"]&0xffff)
	check("domain", uint32(e.Domain)+1, v["U4A"])
	for _, difference := range unitNativePositionDifferences(e, r.raw["Block12"]) {
		differences = append(differences, prefix+" "+difference)
	}
	check("ScanRange", uint32(e.ScanRange), v["UA4"]>>8)
	check("U4C off-map", boolUint1156(e.OffMap), boolUint1156(v["U4C"]&sav.ActorOffMapFlag != 0))
	if r.class == "Human" && e.NativeClass.Present {
		check("U4C Fighter", boolUint1156(e.NativeClass.Fighter), boolUint1156(v["U4C"]&4 == 0))
	} else if !b.ScalarsPresent || b.ScalarKnown&(1<<sim.ScalarU4C) == 0 {
		unavailable("U4C bit2")
	}
	scalar("U4C remaining bits", sim.ScalarU4C, v["U4C"])
	check("Stage", uint32(e.Decay), v["Stage"])
	if v["Stage"] == 1 {
		check("dying timer U6C", uint32(e.Dwell), v["U6C"])
	} else {
		scalar("U6C", sim.ScalarU6C, v["U6C"])
	}
	if r.class == "Unit" && e.TypeID >= 0x1a {
		check("T1C", uint32(e.XPValue), v["T1C"])
	} else {
		scalar("T1C", sim.ScalarT1C, v["T1C"])
	}
	// No current writer DTO or retained constructor scalar is an expectation.
	// These obligations remain failures until an independent carrier is present.
	for _, field := range []struct {
		name  string
		index int
	}{
		{"T0C", sim.ScalarT0C}, {"T08 high word", sim.ScalarT08High}, {"T18", sim.ScalarT18}, {"Reference", sim.ScalarReference},
		{"U4B", sim.ScalarU4B}, {"U60", sim.ScalarU60}, {"U61", sim.ScalarU61}, {"UA0", sim.ScalarUA0}, {"UA4", sim.ScalarUA4},
		{"U130", sim.ScalarU130}, {"U136", sim.ScalarU136}, {"U138", sim.ScalarU138}, {"U148", sim.ScalarU148}, {"U144", sim.ScalarU144},
	} {
		expected := v[field.name]
		if field.index == sim.ScalarT08High {
			expected = v["T08"] >> 16
		}
		scalar(field.name, field.index, expected)
	}
	for n := range 10 {
		if !b.BlockPresent || b.BlockKnown&(1<<n) == 0 || mask.BlockKnown&(1<<n) == 0 {
			unavailable(fmt.Sprintf("Block12 byte%d", n+2))
		} else {
			check(fmt.Sprintf("native Block12 byte%d", n+2), uint32(b.Block[n]), uint32(r.raw["Block12"][n+2]))
		}
	}
	for _, field := range []struct {
		name  string
		index int
	}{{"U50", sim.ScalarU50}, {"U54", sim.ScalarU54}, {"U58", sim.ScalarU58}} {
		scalar(field.name, field.index, binary.LittleEndian.Uint32(r.raw[field.name]))
	}
	if manifest == nil {
		unavailable("Name")
	} else {
		count := 0
		for _, row := range manifest.Actors {
			if row.ID == e.ID {
				count++
				if row.Name != r.name {
					differences = append(differences, prefix+": live manifest Name differs")
				}
			}
		}
		if count != 1 {
			differences = append(differences, prefix+": missing/duplicate manifest Name")
		}
	}
	return differences
}

func boolUint1156(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}

func unitNativePositionDifferences(e sim.Entity, raw []byte) []string {
	if len(raw) != 12 {
		return []string{"position block length differs"}
	}
	x, y := e.X, e.Y
	var differences []string
	if e.Transit > 0 && e.Stride.Present {
		s := e.Stride
		if e.TransitTotal == 0 || e.Transit > e.TransitTotal || e.X != s.ToX || e.Y != s.ToY {
			return []string{"accepted stride destination/interval differs"}
		}
		elapsed := int32(e.TransitTotal - e.Transit)
		fineX := s.FromX*256 + 128 + int32(s.StepX)*elapsed
		fineY := s.FromY*256 + 128 + int32(s.StepY)*elapsed
		x, y = fineX>>8, fineY>>8
		if byte(fineX) != raw[4] || byte(fineY) != raw[5] {
			differences = append(differences, "accepted stride fine position differs")
		}
	}
	cell := binary.LittleEndian.Uint16(raw)
	if x != int32(cell&255) {
		differences = append(differences, fmt.Sprintf("cell X: World=%#x raw=%#x", x, cell&255))
	}
	if y != int32(cell>>8) {
		differences = append(differences, fmt.Sprintf("cell Y: World=%#x raw=%#x", y, cell>>8))
	}
	if e.Transit > 0 && e.Stride.Present && binary.LittleEndian.Uint16(raw[2:]) != cell {
		differences = append(differences, "accepted stride packed position differs")
	}
	return differences
}
