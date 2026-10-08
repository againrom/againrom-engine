package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func nativeSubjectFixture(t *testing.T) (*FrontEnd, []byte) {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	f.Difficulty = mapload.DifficultyNormal
	mapBytes := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: []synth.ALMUnit{
		{X: 15<<8 | 128, Y: 16<<8 | 128, ClassID: 35},
		{X: 16<<8 | 128, Y: 16<<8 | 128, ClassID: 35},
	}})
	rows := rawSections1092(t, mapBytes)[6]
	binary.LittleEndian.PutUint16(rows[64:], 91)
	binary.LittleEndian.PutUint16(rows[70+64:], 92)
	f.Archives.Containers = poolFixtureFrontMap(t, mapBytes).Archives.Containers
	params := make([]int32, 41)
	for i := range params {
		params[i] = -1
	}
	params[0], params[4], params[8], params[9], params[10] = 30, 60, 18, 16, 6
	params[11], params[12], params[14], params[15], params[16] = 3, 5, 17, 9, 2
	params[29], params[30], params[32] = 35, 0, 0
	units := f.Table.Units.(dbCollection)
	for i := range units {
		units[i] = dbEntry{name: "equal native subject", params: slices.Clone(params)}
	}
	if err := f.App("native actor subject").OpenMission(f.MissionOpenerWith(10, []mapload.PartyMember{})); err != nil {
		t.Fatal(err)
	}
	entities := f.live.world.Entities()
	if len(entities) != 2 || entities[0].ID != 0 || entities[0].HP != 60 || entities[1].HP != 60 {
		t.Fatal("real placed constructors did not consume the authored inputs", entities)
	}
	for _, e := range entities {
		if e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) || !e.NativeBasis.BasePresent || e.NativeBasis.BaseKnown != 0x003fffff {
			t.Fatal("fixture must use actual native constructor admission", e)
		}
	}
	raw, _, _ := saveCurrentEffect(t, f)
	return f, raw
}

func nativeSubjectCold(t *testing.T, source *FrontEnd, raw []byte) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	f.Archives.Containers, f.Table, f.Humans = source.Archives.Containers, source.Table, source.Humans
	open, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("native subject cold LOAD").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("native subject cold LOAD", town, err)
	}
	return f
}

func nativeSubjectRead(t *testing.T, raw []byte) ([]actor1161Record, *sackByteReader, map[uint16]uint16, unit1158Set) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	roots, reader, origins, err := readActorRoots(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := readUnitCombatExpected(file, raw)
	if err != nil || len(roots) != 2 || len(want.records) != 2 {
		t.Fatal("independent raw actor population", len(roots), len(want.records), err)
	}
	return roots, reader, origins, want
}

func nativeSubjectCompare(t *testing.T, raw []byte, ms *Mission, observed []sim.Entity, contexts ...*Mission) ([]string, []string, int, int) {
	t.Helper()
	roots, reader, origins, want := nativeSubjectRead(t, raw)
	context := ms
	if len(contexts) != 0 {
		context = contexts[0]
	}
	actorDiffs, population := actorRootEntityDifferences(roots, reader, origins, ms.savedDocument, ms.World, observed, context)
	combatDiffs, _, compared := want.entityDifferences(observed, ms.savedDocument, context)
	return actorDiffs, combatDiffs, population.live, compared
}

func TestNativeActorSubjectColdSaveCorrespondence(t *testing.T) {
	source, raw := nativeSubjectFixture(t)
	cold := nativeSubjectCold(t, source, raw)
	ms := cold.live.mission.state
	before, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	a, c, actors, combat := nativeSubjectCompare(t, raw, ms, ms.World.Entities())
	if actors != 2 || combat != 2 || strings.Contains(strings.Join(append(a, c...), ";"), "eligible raw actor missing") || strings.Contains(strings.Join(c, ";"), "expected live source basis unavailable") {
		t.Fatal("native subjects were discarded by source-only correspondence", actors, combat, a, c)
	}
	for _, e := range ms.World.Entities() {
		if e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) {
			t.Fatal("correspondence promoted source arithmetic", e)
		}
	}
	after, err := ms.World.MarshalBinary()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("comparison changed the World", err)
	}
	t.Log("remaining independent field failures", a, c)
}

func TestNativeActorSubjectExactReceiptLossControls(t *testing.T) {
	source, raw := nativeSubjectFixture(t)
	for _, name := range []string{"registry-missing", "document-missing", "registry-swap-equal-actors", "document-swap-equal-actors", "duplicate-ID", "duplicate-object", "registry-alias", "cross-class", "identity-byte", "runtime-byte", "missing-live", "no-context", "wrong-context"} {
		t.Run(name, func(t *testing.T) {
			cold := nativeSubjectCold(t, source, raw)
			ms := cold.live.mission.state
			observed := ms.World.Entities()
			var contexts []*Mission
			switch name {
			case "registry-missing":
				ms.actorRegistry = nil
			case "document-missing":
				ms.savedDocument.Actors = ms.savedDocument.Actors[1:]
			case "registry-swap-equal-actors":
				rows := ms.actorRegistry.actors
				rows[0].ID, rows[1].ID = rows[1].ID, rows[0].ID
			case "document-swap-equal-actors":
				rows := ms.savedDocument.Actors
				rows[0].ObjectIndex, rows[1].ObjectIndex = rows[1].ObjectIndex, rows[0].ObjectIndex
			case "duplicate-ID":
				observed[1].ID = observed[0].ID
			case "duplicate-object":
				ms.savedDocument.Actors = append(ms.savedDocument.Actors, ms.savedDocument.Actors[0])
			case "registry-alias":
				ms.actorRegistry.actors[1].ID = ms.actorRegistry.actors[0].ID
			case "cross-class":
				ms.actorRegistry.actors[0].Source.Class = "Human"
			case "identity-byte":
				ms.actorRegistry.actors[0].Source.Identity ^= 1
			case "runtime-byte":
				ms.actorRegistry.actors[0].Source.RuntimeID ^= 1
			case "missing-live":
				observed = observed[1:]
			case "no-context":
				contexts = []*Mission{nil}
			case "wrong-context":
				other := *ms
				other.savedDocument = cloneSavedDocumentFixture(t, ms.savedDocument)
				contexts = []*Mission{&other}
			}
			a, c, _, _ := nativeSubjectCompare(t, raw, ms, observed, contexts...)
			joined := strings.Join(append(a, c...), ";")
			if !strings.Contains(joined, "subject") && !strings.Contains(joined, "eligible raw actor missing") && !strings.Contains(joined, "expected live source basis unavailable") {
				t.Fatal("exact subject receipt loss was accepted", name, a, c)
			}
		})
	}
}

func TestNativeActorSubjectRawValuesRemainIndependent(t *testing.T) {
	source, raw := nativeSubjectFixture(t)
	for _, name := range []string{"ToHit", "Skill", "Base-known-byte", "Base-mask", "Base-presence", "book", "ordinary-known-byte"} {
		t.Run(name, func(t *testing.T) {
			cold := nativeSubjectCold(t, source, raw)
			ms := cold.live.mission.state
			observed := ms.World.Entities()
			switch name {
			case "ToHit":
				observed[0].ToHit++
			case "Skill":
				observed[0].Skill[0]++
			case "Base-known-byte":
				observed[0].NativeBasis.Base[0] = 0x63
			case "Base-mask":
				observed[0].NativeBasis.BaseKnown &^= 1
			case "Base-presence":
				observed[0].NativeBasis = sim.NativeActorBasis{}
			case "book":
				observed[0].Book.State = sim.BookPresent
			case "ordinary-known-byte":
				file, err := sav.Open(raw)
				if err != nil {
					t.Fatal(err)
				}
				locations, err := file.DocumentActorLocations()
				if err != nil {
					t.Fatal(err)
				}
				changed, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				_, _, origins, _ := nativeSubjectRead(t, raw)
				block, err := savedActorRaw(&changed.Objects[origins[locations[0].ArchiveIndex]-1], "U114", 24)
				if err != nil {
					t.Fatal(err)
				}
				block[0] = 0x63
				input, err := sav.EncodeDocumentData(changed)
				if err != nil {
					t.Fatal(err)
				}
				_, c, _, n := nativeSubjectCompare(t, input, ms, observed)
				if n != 2 || !strings.Contains(strings.Join(c, ";"), "native U114 byte0 differs") {
					t.Fatal("unchanged continuation or retained Document became a value oracle", n, c)
				}
				return
			}
			a, c, _, _ := nativeSubjectCompare(t, raw, ms, observed)
			joined := strings.Join(append(a, c...), ";")
			needle := map[string]string{"ToHit": "ToHit:", "Skill": "Skill[0]:", "Base-known-byte": "native U114 byte0 differs", "Base-mask": "native basis presence/mask differs", "Base-presence": "native basis presence/mask differs", "book": "native legacy book mode"}[name]
			if !strings.Contains(joined, needle) {
				t.Fatal("raw field control was hidden by correspondence", name, needle, a, c)
			}
		})
	}
}

func TestNativeActorSubjectRawAliasAndEntityZero(t *testing.T) {
	source, raw := nativeSubjectFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	for i := range doc.Objects {
		for j := range doc.Objects[i].Groups {
			group := &doc.Objects[i].Groups[j]
			for k := range group.RefSlots {
				if group.RefSlots[k].Name == "Actors" && len(group.RefSlots[k].Objects) >= 2 {
					group.RefSlots[k].Objects = append(group.RefSlots[k].Objects, group.RefSlots[k].Objects[0], 0)
					for q := range group.Counts {
						if group.Counts[q].Name == "Actors" {
							group.Counts[q].Count += 2
						}
					}
					for q := range doc.Objects[i].Counts {
						if doc.Objects[i].Counts[q].Name == "Actors" {
							doc.Objects[i].Counts[q].Count += 2
						}
					}
					changed = true
				}
			}
		}
	}
	if !changed {
		t.Fatal("real fresh SAV has no actor Group to repeat")
	}
	aliased, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := nativeSubjectCold(t, source, aliased)
	ms := cold.live.mission.state
	_, _, actors, combat := nativeSubjectCompare(t, aliased, ms, ms.World.Entities())
	if actors != 2 || combat != 2 {
		t.Fatal("raw alias duplicated an exact native subject", actors, combat)
	}
	if e, present := ms.World.Entity(0); !present || e.SourceBinding.Class != 0 || e.ActorLoad.Source.Class != 0 {
		t.Fatal("real native ID0 was lost or promoted", e, present)
	}
	// Identity keys remain ordinary metadata. This control does not rewrite IDs.
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	locations, err := file.DocumentActorLocations()
	if err != nil {
		t.Fatal(err)
	}
	if len(locations) != 2 || binary.LittleEndian.Uint32(file.Body[locations[0].Off+29:]) == binary.LittleEndian.Uint32(file.Body[locations[1].Off+29:]) {
		t.Fatal("equal actor values must carry different exact ordinary identities")
	}
	t.Log(fmt.Sprintf("native IDs and raw archive IDs remain separate: %d, %d", locations[0].ArchiveIndex, locations[1].ArchiveIndex))
}
