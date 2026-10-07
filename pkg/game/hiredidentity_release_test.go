package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func hiredOrdinaryIdentity(t *testing.T, raw []byte, party []mapload.PartyMember, table *mapload.Table) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var actors []sav.Character
	if doc.World == nil {
		city, err := f.CityProvenance()
		if err != nil {
			t.Fatal(err)
		}
		actors, err = city.SourceParty()
		if err != nil {
			t.Fatal(err)
		}
	} else {
		actors, err = f.Party()
		if err != nil {
			t.Fatal(err)
		}
	}
	type identity struct {
		row   uint8
		hire  uint8
		name  string
		class int
	}
	want, got := map[identity]int{}, map[identity]int{}
	for _, member := range party {
		if !member.Hired() || member.MercenaryType <= 2 {
			continue
		}
		row := member.DefinitionRow
		if row == 0 {
			row = uint8(data.FindHumanByName(table.Humans, member.Name))
		}
		name := member.Name
		if name == table.Humans.EntryName(int(row)) && strings.HasPrefix(name, "NPC") {
			name = ""
		}
		want[identity{row, member.MercenaryType, name, int(member.Class)}]++
	}
	for _, actor := range actors {
		if actor.Class != "Human" || uint8(actor.DisplayBacking) == 0 {
			continue
		}
		got[identity{actor.DefRow, uint8(actor.DisplayBacking), actor.Name, int(actor.Basis.Human.TypeID)}]++
	}
	if len(want) == 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("ordinary hired identity got=%v want=%v", got, want)
	}
}

func TestReleaseHiredIdentityCityAndMission(t *testing.T) {
	f, screen := hireForOrderTest(t, "Hired identity")
	for _, typ := range []int{6, 14} {
		if _, ok := screen.toggleMercenary(typ); !ok {
			t.Fatal("hire", typ)
		}
	}
	for i := range f.Carried {
		if f.Carried[i].MercenaryType == 6 {
			f.Carried[i].Name = fmt.Sprintf("Veteran %d", i)
		}
	}
	want := mapload.CloneParty(f.Carried)
	for cycle := 0; cycle < 2; cycle++ {
		raw := currentTownSave(t, f)
		hiredOrdinaryIdentity(t, raw, want, f.Table)
		ordinary := currentTownReload(t, hiredActorWithoutSupplement(t, raw))
		if len(ordinary.Carried) != len(want) {
			t.Fatal("ordinary city roster", len(ordinary.Carried), len(want))
		}
		f = currentTownReload(t, raw)
		for _, typ := range []int{6, 14} {
			s := ordinary.TownScreen().(*townScreen)
			for _, hired := range []bool{false, true} {
				if _, ok := s.toggleMercenary(typ); !ok || ordinary.Town.MercenaryHired(typ) != hired {
					t.Fatal("ordinary cancel/rehire", typ, hired)
				}
			}
		}
		if len(ordinary.Carried) != len(want) {
			t.Fatal("ordinary rehire duplicated squad")
		}
	}
	hiredMissionCycles(t, f, want)
}

func hiredMissionCycles(t *testing.T, f *FrontEnd, want []mapload.PartyMember) {
	t.Helper()
	app := f.App("hired mission")
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		store := SaveStore{Dir: t.TempDir()}
		f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
		raw := cityRosterF2Save(t, app, store, fmt.Sprintf("hired-mission-%d", cycle))
		hiredOrdinaryIdentity(t, raw, want, f.Table)
		cold := releaseFront(t)
		opener, town, err := cold.RestoreOriginal(raw)
		if err != nil || town || opener == nil {
			t.Fatal("mission LOAD", err, town)
		}
		coldApp := cold.App("hired mission LOAD")
		if err := coldApp.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 4; tick++ {
			if f.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("mission continuation changed", cycle, tick)
			}
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
		}
		f, app = cold, coldApp
	}
}

func TestReleaseHiredIdentityOwnerResaves(t *testing.T) {
	root := os.Getenv("AGAINROM_HIRED_CORPUS")
	if root == "" {
		t.Skip("AGAINROM_HIRED_CORPUS is not set")
	}
	for _, name := range []string{"hired.sav", "game0007.sav", "game0008.sav"} {
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			raw, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			opener, town, err := f.RestoreOriginal(raw)
			if err != nil {
				t.Fatal(err)
			}
			wantFirst, wantSecond := 3, 4
			if name != "hired.sav" {
				wantFirst, wantSecond = 6, 8
			}
			if town {
				hiredActorCounts(t, f.Carried, wantFirst, wantSecond)
				want := mapload.CloneParty(f.Carried)
				for cycle := 0; cycle < 2; cycle++ {
					raw = currentTownSave(t, f)
					hiredOrdinaryIdentity(t, raw, want, f.Table)
					ordinary := currentTownReload(t, hiredActorWithoutSupplement(t, raw))
					hiredActorCounts(t, ordinary.Carried, wantFirst, wantSecond)
					f = currentTownReload(t, raw)
				}
				hiredMissionCycles(t, f, want)
			} else {
				app := f.App("owner hired mission")
				if err := app.OpenMission(opener); err != nil {
					t.Fatal(err)
				}
				hiredActorCounts(t, f.liveParty, wantFirst, wantSecond)
				want := mapload.CloneParty(f.liveParty)
				for cycle := 0; cycle < 2; cycle++ {
					store := SaveStore{Dir: t.TempDir()}
					f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
					raw = cityRosterF2Save(t, app, store, fmt.Sprintf("owner-hired-%d", cycle))
					hiredOrdinaryIdentity(t, raw, want, f.Table)
					f = releaseFront(t)
					opener, _, err = f.RestoreOriginal(raw)
					if err != nil {
						t.Fatal(err)
					}
					app = f.App("owner hired reLOAD")
					if err := app.OpenMission(opener); err != nil {
						t.Fatal(err)
					}
					hiredActorCounts(t, f.liveParty, wantFirst, wantSecond)
				}
			}
		})
	}
}

func TestReleaseHiredIdentityLegacyMissionNames(t *testing.T) {
	if os.Getenv("AGAINROM_HIRED_CORPUS") == "" {
		t.Skip("AGAINROM_HIRED_CORPUS is not set")
	}
	f := releaseFront(t)
	raw, err := os.ReadFile(filepath.Join(os.Getenv("AGAINROM_HIRED_CORPUS"), "game0008.sav"))
	if err != nil {
		t.Fatal(err)
	}
	opener, town, err := f.RestoreOriginal(raw)
	if err != nil || town || opener == nil {
		t.Fatal("source load", err, town)
	}
	app := f.App("legacy hire probe")
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	raw = cityRosterF2Save(t, app, store, "source-bound-current")
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		row, typ := uint32(0), uint32(0)
		for _, v := range r.Values {
			if v.Name == "T0C" {
				row = v.Value
			}
			if v.Name == "U148" {
				typ = v.Value
			}
		}
		if typ != 12 && typ != 13 {
			continue
		}
		for j := range r.Values {
			if r.Values[j].Name == "U148" {
				r.Values[j].Value = 0
			}
		}
		for j := range r.Texts {
			if r.Texts[j].Name == "Name" {
				r.Texts[j].Value = f.Table.Humans.EntryName(int(row))
			}
		}
		changed++
	}
	if changed != 14 {
		t.Fatal("legacy probe population", changed)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	opener, town, err = cold.RestoreOriginal(raw)
	if err != nil || town || opener == nil {
		t.Fatal("legacy load", err, town)
	}
	coldApp := cold.App("legacy hire LOAD")
	if err := coldApp.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	store = SaveStore{Dir: t.TempDir()}
	cold.ConfigureSaveSeams(coldApp, store, OriginalStore{}, nil)
	raw = cityRosterF2Save(t, coldApp, store, "legacy-repaired")
	hiredOrdinaryIdentity(t, raw, cold.liveParty, cold.Table)
}
