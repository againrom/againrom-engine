//go:build release

package game

import (
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// realChoices are the choices that are rows, not headings.
func realChoices(cs []ui.InspectionChoice) []ui.InspectionChoice {
	var out []ui.InspectionChoice
	for _, c := range cs {
		if c.Value >= 0 {
			out = append(out, c)
		}
	}
	return out
}

func TestReleaseSecondPalettesListTheCatalogAndNameEveryPlacement(t *testing.T) {
	x := installedInspector(t)
	if x.installGame() != base.GameROM2 {
		t.Skip("requires a second-game install")
	}
	named := func(c interface {
		Len() int
		EntryName(int) string
	}) (n int) {
		for i := 1; i < c.Len(); i++ {
			if c.EntryName(i) != "" {
				n++
			}
		}
		return n
	}
	units, humans, buildings := named(x.table.Units), named(x.table.Humans), named(x.table.Buildings)
	archive, loose := ArchiveMaps(x.archives.Containers), x.archives.Loose
	var first string
	unitOK, unitBad, structOK, structBad, maps := 0, 0, 0, 0, 0
	for _, entry := range x.Maps {
		source, read := ui.MapSourceKey(ui.SourceLoose, entry.Source), loose.ReadFile
		if entry.FromArchive {
			source, read = ui.MapSourceKey(ui.SourceScenario, entry.Source), archive.Read
			if first == "" {
				first = source
			}
		}
		raw, err := read(entry.Source)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.OpenROM2(raw)
		if err != nil {
			t.Fatal(err)
		}
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(entry.Source, err)
		}
		// The map takes the terrain brush; every other tool is refused.
		pal, second := d.Edits.(secondEdits)
		if !second {
			t.Fatal("a second-game map's edit host is the terrain-brush host")
		}
		if _, err := d.Edits.Delete(ui.KindUnit, 0); err == nil {
			t.Fatal("a second-game map must refuse a record delete")
		}
		if _, err := pal.AddRecord(ui.InspectionNewRecord{Kind: ui.KindUnit}); err == nil {
			t.Fatal("a second-game map must refuse an add")
		}
		maps++
		for _, r := range d.Records {
			switch r.Kind {
			case ui.KindUnit:
				u := m.Units[r.Index]
				if mapload.Resolve(u, x.table).Found() {
					unitOK++
					if r.Name == "" || strings.Contains(r.Name, "no catalog row") {
						t.Fatalf("%s unit %d resolves but is named %q", entry.Source, r.Index, r.Name)
					}
				} else {
					unitBad++
					if !strings.Contains(r.Name, "no catalog row") {
						t.Fatalf("%s unit %d unresolved but named %q", entry.Source, r.Index, r.Name)
					}
				}
			case ui.KindStructure:
				o := m.Objects[r.Index]
				if _, name, ok := mapload.BuildingName(x.table, o.Kind); ok {
					structOK++
					if !strings.Contains(r.Name, name) || !strings.Contains(strings.Join(r.Details, "\n"), name) {
						t.Fatalf("%s structure %d: %q lacks %q", entry.Source, r.Index, r.Name, name)
					}
				} else {
					structBad++
					if !strings.Contains(r.Name, "no catalog row") {
						t.Fatalf("%s structure %d unresolved but named %q", entry.Source, r.Index, r.Name)
					}
				}
			}
		}
	}
	d, err := x.Open(first)
	if err != nil {
		t.Fatal(err)
	}
	pal, _ := d.Edits.(secondEdits)
	identity := pal.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
	creatures, persons := 0, 0
	for _, c := range realChoices(identity.Choices) {
		id, ok := x.identityOf(c.Value)
		if !ok || c.Label == "" {
			t.Fatalf("identity %d has no row or name", c.Value)
		}
		if id.Kind == mapload.UnitCreature {
			creatures++
		} else {
			persons++
		}
	}
	kinds := pal.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindStructure}, "kind")
	items := pal.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindSack}, "item")
	if creatures != units || persons != humans || len(kinds.Choices) != buildings || len(items.Choices) == 0 {
		t.Fatalf("palette counts: units %d/%d, humans %d/%d, structures %d/%d, items %d", creatures, units, persons, humans, len(kinds.Choices), buildings, len(items.Choices))
	}
	if _, err := pal.AddRecord(ui.InspectionNewRecord{Kind: ui.KindUnit}); err == nil {
		t.Fatal("a viewed map accepted a placement")
	}
	e := x.Editor()
	e.Layout(1920, 1080)
	if err := e.Open(first); err != nil {
		t.Fatal(err)
	}
	shown, _ := e.PaletteHolds(ui.PaletteStructures)
	more, _ := e.PaletteHolds(ui.PaletteObjects)
	if len(shown)+len(more) != buildings {
		t.Fatalf("structure palettes hold %d+%d of %d rows", len(shown), len(more), buildings)
	}
	t.Logf("PASS: %d maps; unit placements resolved %d unresolved %d; structure placements resolved %d unresolved %d; palette rows units %d humans %d structures %d items %d",
		maps, unitOK, unitBad, structOK, structBad, creatures, persons, len(kinds.Choices), len(items.Choices))
}

func TestReleaseFirstPalettesStillListAndPlace(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	d, err := x.Open("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	if d.Edits == nil || d.Palettes != nil {
		t.Fatal("first-game map edits and lists through its own host")
	}
	host := d.Edits.(ui.InspectionNewRecords)
	identity := host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
	if want := len(mapload.Identities(x.table, x.identityMission)); len(realChoices(identity.Choices)) != want {
		t.Fatalf("identities %d, want %d", len(realChoices(identity.Choices)), want)
	}
	before := len(d.Records)
	m, err := alm.Open(d.SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	var pick int64 = -1
	for _, c := range realChoices(identity.Choices) {
		pick = c.Value
		break
	}
	next, err := host.AddRecord(ui.InspectionNewRecord{Kind: ui.KindUnit, Cell: mapload.Playable(m).Min, HasCell: true,
		Values: []ui.InspectionFieldValue{{Key: "identity", Value: pick}}})
	if err != nil || next == nil || len(next.Records) <= before {
		t.Fatalf("first-game placing: %v", err)
	}
}
