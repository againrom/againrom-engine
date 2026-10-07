package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The counts come from raw ALM zero words paired with positive DAT maximum
// health. Opening a mission here tests placement, not campaign reachability.
func TestReleaseBuilding1175AuthoredZeroAndColdSave(t *testing.T) {
	for _, tc := range []struct{ mission, zeros int }{
		{40, 3}, {50, 4}, {61, 6}, {81, 4}, {111, 3}, {120, 8}, {150, 11},
	} {
		t.Run(fmt.Sprint(tc.mission), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.NextParty()
			prepareAcceptedCampaignMission(t, f, tc.mission)
			f.Carried = mapload.CloneParty(party)
			address, _ := MissionMap(tc.mission)
			raw, err := f.Archives.Containers.ReadFile(address)
			if err != nil {
				t.Fatal(err)
			}
			m, err := alm.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			app := f.App("1175-building-placement")
			app.Layout(1024, 768)
			if err := app.OpenMission(f.MissionOpener(tc.mission)); err != nil {
				t.Fatal(err)
			}
			// Inspect every installed placement, including unexplored cells.
			// Reveal affects this viewer only; it does not change simulation fog.
			f.live.view.SetFogReveal(true)
			structures := f.live.world.Structures()
			if len(structures) != len(m.Objects) {
				t.Fatalf("%d live structures for %d records", len(structures), len(m.Objects))
			}
			var zeroIDs []sim.StructureID
			ruinEntries := 0
			for i, object := range m.Objects {
				definition := int(uint8(object.Kind))
				if definition < 1 || definition >= f.Table.Buildings.Len() {
					t.Fatalf("unresolved installed definition %d", definition)
				}
				params := f.Table.Buildings.EntryParams(definition)
				if len(params) < 4 {
					t.Fatal("short installed Building definition")
				}
				maximum := uint16(params[3])
				current := maximum
				if object.Field0C == 0 && uint16(object.Kind) != 34 && uint16(object.Kind) != 35 {
					current = 0
					if maximum != 0 {
						zeroIDs = append(zeroIDs, sim.StructureID(i))
					}
				}
				if got := structures[i]; got.Field42 != current || got.MaxHealth != maximum {
					t.Fatalf("structure%d kind%d raw%04x = %d/%d, want %d/%d", i,
						object.Kind, object.Field0C, got.Field42, got.MaxHealth, current, maximum)
				}
				class := f.Structures.Classes[definition]
				if current == 0 && maximum != 0 && class != nil && !class.Indestructible &&
					class.GridCells() > 0 && len(class.Frames) >= 2*class.GridCells() {
					entries, ruins := f.live.view.StructureRuinFrames(uint32(i))
					if entries == 0 || ruins != entries {
						t.Fatalf("authored-zero structure%d draw: %d entries, %d ruins", i, entries, ruins)
					}
					ruinEntries += ruins
				}
			}
			if len(zeroIDs) != tc.zeros {
				t.Fatalf("authored-zero positive-max structures = %d, want %d", len(zeroIDs), tc.zeros)
			}
			refs, err := mapload.StructureFieldRefs(m, f.Table)
			if err != nil {
				t.Fatal(err)
			}
			zeroRefs := 0
			for _, ref := range refs {
				if ref.Placed && ref.Value != structures[ref.Ref].Field42 {
					t.Fatalf("compiled script node %+v disagrees with live structure", ref)
				}
				if ref.Placed && slices.Contains(zeroIDs, sim.StructureID(ref.Ref)) {
					zeroRefs++
				}
			}
			t.Logf("%d authored-zero structures; %d ruin entries; %d/%d script references reach those structures",
				len(zeroIDs), ruinEntries, zeroRefs, len(refs))
			coldBuilding1175(t, f)

			// A saved live value takes precedence over the ALM starting value.
			// This isolated import changes one existing word without a damage or
			// regeneration action; restoring it must not reseed the placement.
			id := zeroIDs[0]
			if err := f.live.world.ImportOriginalStructureHealth([]sim.OriginalStructureHealth{
				{ID: id, Health: 1, MaxHealth: structures[id].MaxHealth},
			}); err != nil {
				t.Fatal(err)
			}
			f.live.push()
			coldBuilding1175(t, f)
		})
	}
}

func coldBuilding1175(t *testing.T, f *FrontEnd) {
	t.Helper()
	want := f.live.world.Structures()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := DecodeSave(disk)
	if err != nil {
		t.Fatal(err)
	}
	fresh := releaseFront(t)
	opener, town, err := fresh.Restore(saved)
	if err != nil || town || opener == nil {
		t.Fatalf("cold Restore: town=%v opener=%v error=%v", town, opener != nil, err)
	}
	app := fresh.App("1175-building-cold-load")
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	fresh.live.view.SetFogReveal(true)
	if got := fresh.live.world.Structures(); !slices.Equal(got, want) {
		t.Fatal("cold AGS changed structure state")
	}
	for _, structure := range want {
		before, beforeRuins := f.live.view.StructureRuinFrames(uint32(structure.ID))
		after, afterRuins := fresh.live.view.StructureRuinFrames(uint32(structure.ID))
		if before != after || beforeRuins != afterRuins {
			t.Fatalf("structure%d draw changed after cold AGS: %d/%d -> %d/%d",
				structure.ID, before, beforeRuins, after, afterRuins)
		}
	}
}
