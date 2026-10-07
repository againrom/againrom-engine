package game

import (
	"encoding/binary"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// cityGroupsByName reads a town SAV's single Player: its key and each group's
// members named by their actors' names.
func cityGroupsByName(t *testing.T, data sav.CityData) (uint32, []sav.CityGroupData, [][]string) {
	t.Helper()
	if len(data.Players) != 1 || data.Objects[data.Players[0]-1].Player == nil {
		t.Fatal("town SAV has no single Player")
	}
	player := data.Objects[data.Players[0]-1].Player
	names := make([][]string, len(player.Groups))
	for g, group := range player.Groups {
		for _, a := range group.Actors {
			names[g] = append(names[g], data.Objects[a-1].Unit.Name)
		}
	}
	return binary.LittleEndian.Uint32(player.Fixed[47:51]), player.Groups, names
}

func writtenTownData(t *testing.T, f *FrontEnd) sav.CityData {
	t.Helper()
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportOriginalSave(snapshot, "groups")
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	return p.Data()
}

// openOriginalSalesTown opens the CitySales source town through the ordinary
// load route and returns the front end and the source bytes.
func openOriginalSalesTown(t *testing.T) (*FrontEnd, []byte) {
	t.Helper()
	f := releaseFront(t)
	sourcePath, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	app := f.App("city groups")
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(sourcePath)}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	label := ""
	for _, e := range list() {
		if e.Name == filepath.Base(sourcePath) {
			label = e.Label
		}
	}
	if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("open the original town", err, app.Screen())
	}
	return f, source
}

// A town SAV written from a loaded original town keeps that town's Player
// groups: the same members in the same groups and order, every other group
// field as loaded, and the owner key naming the written Player.
func TestReleaseCitySAVKeepsTheLoadedPlayerGroups(t *testing.T) {
	f, source := openOriginalSalesTown(t)
	file, err := sav.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	loadedKey, loadedGroups, loadedNames := cityGroupsByName(t, loaded.Data())
	if len(loadedGroups) != 2 {
		t.Fatalf("source town has %d Player groups, want 2", len(loadedGroups))
	}
	writtenKey, writtenGroups, writtenNames := cityGroupsByName(t, writtenTownData(t, f))
	if len(writtenGroups) != len(loadedGroups) {
		t.Fatalf("written town has groups %v, loaded %v", writtenNames, loadedNames)
	}
	for g := range loadedGroups {
		want, got := loadedGroups[g], writtenGroups[g]
		if !slices.Equal(writtenNames[g], loadedNames[g]) {
			t.Fatalf("group %d members %v, loaded %v", g+1, writtenNames[g], loadedNames[g])
		}
		if !slices.Equal(got.Words20, want.Words20) || !slices.Equal(got.Raw80, want.Raw80) || !slices.Equal(got.Words3c, want.Words3c) || got.F1c != want.F1c {
			t.Fatalf("group %d fields differ from the loaded group", g+1)
		}
		if want.F44 != loadedKey || got.F44 != writtenKey || want.F40 != 0 || got.F40 != 0 {
			t.Fatalf("group %d keys F40 %#x F44 %#x, loaded F40 %#x F44 %#x (Player %#x, written Player %#x)", g+1, got.F40, got.F44, want.F40, want.F44, loadedKey, writtenKey)
		}
	}
}

// A town reached without a loaded town document writes its one constructed
// group holding the whole party (DIV-1411).
func TestReleaseCitySAVWithoutALoadedTownWritesOneGroup(t *testing.T) {
	f := currentTown(t, nil, nil)
	if f.originalCity != nil {
		t.Skip("this town route retains a loaded town document")
	}
	_, groups, names := cityGroupsByName(t, writtenTownData(t, f))
	if len(groups) != 1 || len(names[0]) != len(f.Carried) {
		t.Fatalf("town without a loaded document has groups %v, want one group of %d", names, len(f.Carried))
	}
}

// hireInTown hires each squad type through the production tavern toggle.
func hireInTown(t *testing.T, f *FrontEnd, types ...int) {
	t.Helper()
	f.Town.gold = 5_000_000
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	for _, typ := range types {
		f.Town.mercEnabled[typ] = true
		if msg, ok := s.toggleMercenary(typ); !ok {
			t.Fatalf("hire type %d refused: %s", typ, msg)
		}
	}
}

// wantHireGroups checks that the written groups after the first `kept` are
// one group per hired squad, in hire order, each holding exactly that squad's
// members in party order, with the hire-group fields and the written Player.
func wantHireGroups(t *testing.T, f *FrontEnd, data sav.CityData, kept int, types ...int) {
	t.Helper()
	key, groups, names := cityGroupsByName(t, data)
	if len(groups) != kept+len(types) {
		t.Fatalf("written town has groups %v, want %d kept groups and %d hire groups", names, kept, len(types))
	}
	hire := cityHireGroup()
	seen := map[uint16]bool{}
	identities := map[uint32]bool{}
	for gi, group := range groups {
		for _, actor := range group.Actors {
			if actor == 0 || int(actor) > len(data.Objects) || seen[actor] {
				t.Fatalf("group %d repeats or loses actor reference %d", gi+1, actor)
			}
			seen[actor] = true
			unit := data.Objects[actor-1].Unit
			if unit == nil || len(unit.Token) != 37 {
				t.Fatalf("group %d actor %d has no complete Unit token", gi+1, actor)
			}
			identity := binary.LittleEndian.Uint32(unit.Token[29:33])
			if identity == 0 || identities[identity] {
				t.Fatalf("group %d actor %d has absent or repeated identity %#x", gi+1, actor, identity)
			}
			identities[identity] = true
		}
	}
	for k, typ := range types {
		g := groups[kept+k]
		at := 0
		for _, member := range f.Carried {
			if int(member.MercenaryType) != typ {
				continue
			}
			if at >= len(g.Actors) {
				t.Fatalf("hire group %d lost type-%d member %q", k+1, typ, member.ID)
			}
			actor := data.Objects[g.Actors[at]-1]
			if actor.Class != "Human" || actor.Unit == nil || len(actor.Unit.Token) != 37 || len(actor.Unit.Scalar2) != 55 {
				t.Fatalf("hire group %d actor %d lacks its complete Human record", k+1, at)
			}
			unit := actor.Unit
			row := member.DefinitionRow
			if row == 0 || int(row) >= f.Table.Humans.Len() {
				t.Fatalf("fixture hire %q has no independent Humans row", member.ID)
			}
			wantName := member.Name
			// Exact template names are anonymous on the ordinary wire. The
			// definition and hire type remain separate from a custom name.
			if wantName == f.Table.Humans.EntryName(int(row)) {
				wantName = ""
			}
			if unit.Name != wantName || unit.Token[16] != row || binary.LittleEndian.Uint16(unit.Token[17:19]) != uint16(member.Class) ||
				binary.LittleEndian.Uint32(unit.Scalar2[47:51]) != uint32(typ) || binary.LittleEndian.Uint16(unit.Token[23:25]) != 2 ||
				binary.LittleEndian.Uint32(unit.Token[33:37]) != key {
				t.Fatalf("hire group %d member %d: Name=%q row=%d class=%d hire=%d publication=%d owner=%#x; want Name=%q row=%d class=%d hire=%d publication=2 owner=%#x",
					k+1, at, unit.Name, unit.Token[16], binary.LittleEndian.Uint16(unit.Token[17:19]), binary.LittleEndian.Uint32(unit.Scalar2[47:51]),
					binary.LittleEndian.Uint16(unit.Token[23:25]), binary.LittleEndian.Uint32(unit.Token[33:37]), wantName, row, member.Class, typ, key)
			}
			at++
		}
		if at == 0 || at != len(g.Actors) {
			t.Fatalf("hire group %d has %d actors, want exactly %d type-%d members", k+1, len(g.Actors), at, typ)
		}
		if !slices.Equal(g.Raw80, hire.Raw80) || g.F1c != 0 || g.F40 != 0 || g.F44 != key || len(g.Words20) != 0 || len(g.Words3c) != 0 {
			t.Fatalf("hire group %d fields %+v, want the hire group under Player %#x", k+1, g, key)
		}
	}
}

// saveAndReloadPartyIDs saves the town through the ordinary save seam, loads
// it into a fresh front end and returns the loaded party's IDs.
func saveAndReloadPartyIDs(t *testing.T, f *FrontEnd) []string {
	t.Helper()
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("town save = %q, %v; want a SAV", name, err)
	}
	restored := releaseFront(t)
	_, list, load := restored.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := list()
	if len(entries) != 1 {
		t.Fatalf("save list %+v, want one entry", entries)
	}
	if _, town, err := load(entries[0].Name); err != nil || !town {
		t.Fatalf("load the town SAV: town %v err %v", town, err)
	}
	_, ids := partyTypeAndIDSequence(restored.Carried)
	return ids
}

// A native town's SAV writes the hero's group and then one group per hired
// squad, in hire order, holding exactly that squad (SAV-617); the reload keeps
// party and hire order.
func TestReleaseCitySAVWritesEachNativeHireInItsOwnGroup(t *testing.T) {
	f, _ := hireForOrderTest(t, "Group Hero")
	hireInTown(t, f, 14, 6)
	for i := range f.Carried {
		if f.Carried[i].MercenaryType == 6 {
			f.Carried[i].Name = "Named hired guard"
			break
		}
	}
	_, _, names := cityGroupsByName(t, writtenTownData(t, f))
	var first []string
	for _, member := range f.Carried {
		if !member.Hired() {
			first = append(first, member.Name)
		}
	}
	if len(names) == 0 || !slices.Equal(names[0], first) {
		t.Fatalf("first group %v, want the unhired party %v", names, first)
	}
	wantHireGroups(t, f, writtenTownData(t, f), 1, 14, 6)
	_, want := partyTypeAndIDSequence(f.Carried)
	if got := saveAndReloadPartyIDs(t, f); !slices.Equal(got, want) {
		t.Fatalf("reloaded party %v, want %v", got, want)
	}
}

// A loaded original town's SAV keeps the loaded groups and adds one group for
// a squad hired after the load (SAV-617); the reload keeps party and hire
// order.
func TestReleaseCitySAVAddsAHireGroupAfterTheLoadedGroups(t *testing.T) {
	f, source := openOriginalSalesTown(t)
	file, err := sav.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	_, loadedGroups, loadedNames := cityGroupsByName(t, loaded.Data())
	offers := f.TownScreen().(*townScreen).tavernMercenaries()
	if len(offers) == 0 {
		t.Fatal("fixture assumption broke: the source town's tavern offers no squad")
	}
	hireInTown(t, f, offers[0].Type)
	data := writtenTownData(t, f)
	_, _, names := cityGroupsByName(t, data)
	for g := range loadedGroups {
		if g >= len(names) || !slices.Equal(names[g], loadedNames[g]) {
			t.Fatalf("written groups %v, want the loaded groups %v first", names, loadedNames)
		}
	}
	wantHireGroups(t, f, data, len(loadedGroups), offers[0].Type)
	_, want := partyTypeAndIDSequence(f.Carried)
	if got := saveAndReloadPartyIDs(t, f); !slices.Equal(got, want) {
		t.Fatalf("reloaded party %v, want %v", got, want)
	}
}
