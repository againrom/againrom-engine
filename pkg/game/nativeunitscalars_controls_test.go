package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The dispatch keeps this test compiling before and after Mission context is
// added. The unchanged source-only comparator must fail the subject count.
func nativeUnitScalarCompare(want unitScalarSet, entities []sim.Entity, manifest *SnapshotActorManifest, contexts ...*Mission) ([]string, []string, int) {
	if compare, ok := any(want.entityDifferences).(func([]sim.Entity, *SnapshotActorManifest, ...*Mission) ([]string, []string, int)); ok {
		return compare(entities, manifest, contexts...)
	}
	return want.entityDifferences(entities, manifest)
}

func nativeUnitScalarRead(t *testing.T, raw []byte) unitScalarSet {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := unitScalarExpected(file, raw)
	if err != nil || len(want.records) != 2 {
		t.Fatal("raw scalar population", len(want.records), err)
	}
	return want
}

func nativeUnitScalarFixture(t *testing.T) (*FrontEnd, []byte) {
	t.Helper()
	f, _ := nativeSubjectFixture(t)
	for _, e := range f.live.world.Entities() {
		inventory := e.ActorLoad
		inventory.Present, inventory.OwnWeight = true, 13
		load := sim.ActorLoadSnapshot{Inventory: inventory,
			Load: 13, Capacity: 317, Speed: e.Speed, HealthHundredths: e.HealthHundredths, ManaHundredths: e.ManaHundredths}
		if err := f.live.world.RestoreActorLoad(e.ID, load); err != nil {
			t.Fatal("independent native load operands", err)
		}
	}
	raw, _, _ := saveCurrentEffect(t, f)
	return f, raw
}

func TestNativeUnitScalarsColdRawSubjects(t *testing.T) {
	source, raw := nativeUnitScalarFixture(t)
	cold := nativeSubjectCold(t, source, raw)
	ms := cold.live.mission.state
	want := nativeUnitScalarRead(t, raw)
	if differences := want.documentDifferences(ms.savedDocument); len(differences) != 0 {
		t.Fatal("raw ordinary scalar bytes", differences)
	}
	before, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	differences, excluded, n := nativeUnitScalarCompare(want, ms.World.Entities(), ms.ActorManifest, ms)
	if n != 2 || len(excluded) != 0 {
		t.Fatal("exact native Class0 scalar subjects were excluded", n, excluded, differences)
	}
	joined := strings.Join(differences, ";")
	if strings.Contains(joined, "expected live scalar") || strings.Contains(joined, "live class/basis presence differs") {
		t.Fatal("native identity join demanded source arithmetic", differences)
	}
	for _, name := range []string{"T0C", "U4B", "U148", "UA0", "UA4", "U130", "U50", "U54", "U58"} {
		if !strings.Contains(joined, "native "+name+" current") {
			t.Fatal("unrepresented ordinary field became silent success", name, differences)
		}
	}
	for _, e := range ms.World.Entities() {
		if e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) {
			t.Fatal("comparison fabricated source arithmetic", e)
		}
	}
	after, err := ms.World.MarshalBinary()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("scalar comparison mutated World", err)
	}
}

func TestNativeUnitScalarsCurrentFieldLossControls(t *testing.T) {
	source, raw := nativeUnitScalarFixture(t)
	cold := nativeSubjectCold(t, source, raw)
	ms := cold.live.mission.state
	want := nativeUnitScalarRead(t, raw)
	controls := []struct {
		name string
		edit func(*sim.Entity)
	}{
		{"Reaction", func(e *sim.Entity) { e.Reaction++ }},
		{"Mind", func(e *sim.Entity) { e.Mind++ }},
		{"Spirit", func(e *sim.Entity) { e.Spirit++ }},
		{"Speed", func(e *sim.Entity) { e.Speed++ }},
		{"U8E", func(e *sim.Entity) { e.ActorLoad.OwnWeight++ }},
		{"U90", func(e *sim.Entity) { e.Load++ }},
		{"Capacity", func(e *sim.Entity) { e.Capacity++ }},
		{"Health", func(e *sim.Entity) { e.HP++ }},
		{"HealthMax", func(e *sim.Entity) { e.MaxHP++ }},
		{"HealthRegen", func(e *sim.Entity) { e.HealthRegenPeriod++ }},
		{"Mana", func(e *sim.Entity) { e.Mana++ }},
		{"ManaMax", func(e *sim.Entity) { e.MaxMana++ }},
		{"ManaRegen", func(e *sim.Entity) { e.ManaRegenPeriod++ }},
		{"UA2", func(e *sim.Entity) { e.HealthHundredths++ }},
		{"UA3", func(e *sim.Entity) { e.ManaHundredths++ }},
		{"U12C", func(e *sim.Entity) { e.Reach++ }},
		{"U134", func(e *sim.Entity) { e.AttackCharge++ }},
		{"U135", func(e *sim.Entity) { e.AttackRelax++ }},
		{"T0E", func(e *sim.Entity) { e.TypeID++ }},
		{"T1C", func(e *sim.Entity) { e.XPValue++ }},
		{"U49", func(e *sim.Entity) { e.TokenSize++ }},
		{"domain", func(e *sim.Entity) { e.Domain = (e.Domain + 1) % 3 }},
		{"MapUnitID", func(e *sim.Entity) { e.MapUnitID++ }},
		{"cell X", func(e *sim.Entity) { e.X++ }},
		{"cell Y", func(e *sim.Entity) { e.Y++ }},
		{"ScanRange", func(e *sim.Entity) { e.ScanRange++ }},
		{"U4C off-map", func(e *sim.Entity) { e.OffMap = !e.OffMap }},
		{"Stage", func(e *sim.Entity) { e.Decay = sim.DecayFallen }},
		{"native Body", func(e *sim.Entity) { e.NativeBasis.Body++ }},
	}
	for _, control := range controls {
		t.Run(control.name, func(t *testing.T) {
			bad := ms.World.Entities()
			control.edit(&bad[0])
			differences, excluded, n := nativeUnitScalarCompare(want, bad, ms.ActorManifest, ms)
			needle := control.name + ": World="
			if n != 2 || len(excluded) != 0 || !strings.Contains(strings.Join(differences, ";"), needle) {
				t.Fatal("independent current field loss was hidden", control.name, n, excluded, differences)
			}
		})
	}
	for _, name := range []string{"Body-known", "Body-presence", "Base-mask", "Modifier-mask", "native-source-residue", "load-presence", "native-class-presence"} {
		t.Run(name, func(t *testing.T) {
			bad := ms.World.Entities()
			switch name {
			case "Body-known":
				bad[0].NativeBasis.BodyKnown = false
			case "Body-presence":
				bad[0].NativeBasis.BodyPresent, bad[0].NativeBasis.BodyKnown, bad[0].NativeBasis.Body = false, false, 0
			case "Base-mask":
				bad[0].NativeBasis.BaseKnown ^= 1
			case "Modifier-mask":
				bad[0].NativeBasis.ModifierKnown ^= 1
			case "native-source-residue":
				bad[0].ActorLoad.Source.Stats[0] = 1
			case "load-presence":
				bad[0].ActorLoad.Present = !bad[0].ActorLoad.Present
			case "native-class-presence":
				bad[0].NativeClass.Present = !bad[0].NativeClass.Present
			}
			differences, _, _ := nativeUnitScalarCompare(want, bad, ms.ActorManifest, ms)
			joined := strings.Join(differences, ";")
			if !strings.Contains(joined, "native basis presence/mask differs") && !strings.Contains(joined, "native actor subject arithmetic absence/class differs") {
				t.Fatal("current presence/mask loss was accepted", name, differences)
			}
		})
	}
}

func TestNativeUnitScalarsRawByteLossControls(t *testing.T) {
	source, raw := nativeUnitScalarFixture(t)
	cold := nativeSubjectCold(t, source, raw)
	ms := cold.live.mission.state
	for _, field := range []struct {
		name      string
		at, width int
	}{
		{"native Body", 0, 2}, {"Reaction", 2, 2}, {"Mind", 4, 2}, {"Spirit", 6, 2}, {"Speed", 8, 2},
		{"U8E", 10, 2},
		{"U90", 12, 2}, {"Capacity", 14, 2}, {"Health", 16, 2}, {"HealthMax", 18, 2}, {"HealthRegen", 20, 2},
		{"Mana", 22, 2}, {"ManaMax", 24, 2}, {"ManaRegen", 26, 2}, {"UA2", 28, 1}, {"UA3", 29, 1},
		{"U12C", 34, 1}, {"U134", 39, 1}, {"U135", 40, 1}, {"Stage", 46, 1},
	} {
		for byteIndex := range field.width {
			t.Run(fmt.Sprintf("%s-byte%d", field.name, byteIndex), func(t *testing.T) {
				file, err := sav.Open(raw)
				if err != nil {
					t.Fatal(err)
				}
				locations, err := file.DocumentActorLocations()
				if err != nil || len(locations) != 2 {
					t.Fatal("raw starts", locations, err)
				}
				at := locations[0].StateOff
				at += 1 + int(file.Body[at])
				file.Body[at+field.at+byteIndex] ^= 1
				// The unmodified transport supplies identity/presence only.
				// Expected scalar words are reread from the changed ordinary Body.
				want, err := unitScalarExpected(file, raw)
				if err != nil {
					t.Fatal(err)
				}
				differences, excluded, n := nativeUnitScalarCompare(want, ms.World.Entities(), ms.ActorManifest, ms)
				if n != 2 || len(excluded) != 0 || !strings.Contains(strings.Join(differences, ";"), field.name+": World=") {
					t.Fatal("continuation or retained Document replaced raw value oracle", n, excluded, differences)
				}
			})
		}
	}
}

func TestNativeUnitScalarsExactReceiptLossControls(t *testing.T) {
	source, raw := nativeUnitScalarFixture(t)
	want := nativeUnitScalarRead(t, raw)
	for _, name := range []string{"registry-missing", "document-missing", "registry-swap-equal-actors", "document-swap-equal-actors", "duplicate-ID", "duplicate-object", "registry-alias", "cross-class", "identity", "runtime", "archive", "offset", "missing-live", "no-context", "wrong-context", "ambiguous-context"} {
		t.Run(name, func(t *testing.T) {
			cold := nativeSubjectCold(t, source, raw)
			ms := cold.live.mission.state
			observed := ms.World.Entities()
			contexts := []*Mission{ms}
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
			case "identity":
				ms.actorRegistry.actors[0].Source.Identity ^= 1
			case "runtime":
				ms.actorRegistry.actors[0].Source.RuntimeID ^= 1
			case "archive":
				ms.actorRegistry.actors[0].Source.ArchiveIndex++
			case "offset":
				ms.actorRegistry.actors[0].Source.Off++
			case "missing-live":
				observed = observed[1:]
			case "no-context":
				contexts = nil
			case "wrong-context":
				other := *ms
				other.savedDocument = cloneSavedDocumentFixture(t, ms.savedDocument)
				other.savedDocument.Actors[0].ObjectIndex = other.savedDocument.Actors[1].ObjectIndex
				contexts = []*Mission{&other}
			case "ambiguous-context":
				contexts = append(contexts, ms)
			}
			differences, excluded, n := nativeUnitScalarCompare(want, observed, ms.ActorManifest, contexts...)
			if n >= 2 || len(excluded) != 0 || !strings.Contains(strings.Join(differences, ";"), "subject") {
				t.Fatal("identity receipt loss preserved scalar admission", name, n, excluded, differences)
			}
		})
	}
}

func TestNativeUnitScalarsRawIdentityLossControls(t *testing.T) {
	source, raw := nativeUnitScalarFixture(t)
	cold := nativeSubjectCold(t, source, raw)
	ms := cold.live.mission.state
	for _, at := range []int{12, 13, 14, 15, 29, 30, 31, 32} {
		t.Run(fmt.Sprintf("Token-byte%d", at), func(t *testing.T) {
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			locations, err := file.DocumentActorLocations()
			if err != nil || len(locations) != 2 {
				t.Fatal("raw starts", locations, err)
			}
			file.Body[locations[0].Off+at] ^= 1
			want, err := unitScalarExpected(file, raw)
			if err != nil {
				t.Fatal(err)
			}
			differences, excluded, n := nativeUnitScalarCompare(want, ms.World.Entities(), ms.ActorManifest, ms)
			if n != 1 || len(excluded) != 0 || !strings.Contains(strings.Join(differences, ";"), "native actor subject registry metadata/ID differs") {
				t.Fatal("raw provenance bytes redirected or collapsed the native join", n, excluded, differences)
			}
		})
	}
}

func TestNativeUnitScalarsRawAliasAndEntityZero(t *testing.T) {
	source, raw := nativeUnitScalarFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	for i := range doc.Objects {
		for j := range doc.Objects[i].Groups {
			group := &doc.Objects[i].Groups[j]
			for k := range group.RefSlots {
				if group.RefSlots[k].Name != "Actors" || len(group.RefSlots[k].Objects) < 2 {
					continue
				}
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
	if !changed {
		t.Fatal("fresh SAV lacks a repeated actor Group fixture")
	}
	aliased, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := nativeSubjectCold(t, source, aliased)
	ms := cold.live.mission.state
	want := nativeUnitScalarRead(t, aliased)
	differences, excluded, n := nativeUnitScalarCompare(want, ms.World.Entities(), ms.ActorManifest, ms)
	if n != 2 || len(excluded) != 0 || len(want.records) != 2 {
		t.Fatal("references duplicated tagged scalar subjects", n, excluded, differences)
	}
	if e, ok := ms.World.Entity(0); !ok || e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) {
		t.Fatal("native entity zero was lost or promoted", e, ok)
	}
	file, err := sav.Open(aliased)
	if err != nil {
		t.Fatal(err)
	}
	locations, err := file.DocumentActorLocations()
	if err != nil || len(locations) != 2 || binary.LittleEndian.Uint32(file.Body[locations[0].Off+29:]) == binary.LittleEndian.Uint32(file.Body[locations[1].Off+29:]) {
		t.Fatal("equal-valued subjects lost independent keys", err)
	}
	// Removing a current carrier remains a failure despite a surviving actor.
	differences, excluded, n = nativeUnitScalarCompare(want, slices.Delete(ms.World.Entities(), 0, 1), ms.ActorManifest, ms)
	if n != 1 || len(excluded) != 0 || !strings.Contains(strings.Join(differences, ";"), "expected live scalar") {
		t.Fatal("raw aliases concealed missing native entity", n, excluded, differences)
	}
}
