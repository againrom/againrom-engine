package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func actorRegistryTable() *mapload.Table {
	unit := make([]int32, 41)
	for i := range unit {
		unit[i] = -1
	}
	human := make([]int32, 32)
	for i := range human {
		human[i] = -1
	}
	human[16], human[17], human[18] = 3, 2, 0
	return &mapload.Table{
		Units:  dbCollection{{name: "map creature", params: append([]int32(nil), unit...)}, {name: "source creature", params: unit}},
		Humans: dbCollection{{}, {name: "NPC_temporary", params: human}, {}, {}, {}, {name: "NPC_persistent", params: human}},
	}
}

func actorRegistryFront1111(t *testing.T) *FrontEnd {
	return actorRegistryFront(t)
}

func actorRegistryFront(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91)
	f.Table = actorRegistryTable()
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	f.Units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{}}
	// Class 1 is what an unarmed party member's equipment selects
	// (HERO-APPEAR-042); the others are the fixture's wire TypeIDs.
	for _, class := range []int32{1, 3, 33, 35} {
		art := worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
		art.Name = "table name must not replace saved name"
		f.Units.Classes[class] = art
	}
	return f
}

// Literal SAV offsets from the independent fixture writer: Token row/type
// at16/17; Unit store-A at509; mover facing at179. No importer output supplies
// these expected operands. The two nonmatching MapUnitIDs deliberately collide.
func actorRegistrySave1111() []byte {
	body, _ := actorRegistryBody1111()
	return savedContainer(body)
}

func actorRegistryBody1111() ([]byte, []*poolFixtureActor) {
	load, runtime := &[4]int16{100, 0, 0, 300}, &[3]byte{1, 8, 4}
	a := &poolFixtureActor{cell: 0x100f, hp: 7, maxHP: 31, name: "Source creature", loadWords: load, equipmentRuntime: runtime}
	b := &poolFixtureActor{mapID: 900, cell: 0x120f, hp: 9, maxHP: 41, human: true, name: "Source temporary", loadWords: load, equipmentRuntime: runtime}
	c := &poolFixtureActor{mapID: 900, cell: 0x140f, hp: 11, maxHP: 51, human: true, name: "Source persistent", loadWords: load, equipmentRuntime: runtime}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b, a, c}}}}, nil)
	for i, actor := range []*poolFixtureActor{a, b, c} {
		binary.LittleEndian.PutUint32(body[actor.off+12:], uint32(501+i))
		body[actor.off+16] = 1
		binary.LittleEndian.PutUint16(body[actor.off+17:], []uint16{35, 3, 33}[i])
		body[actor.off+509], body[actor.off+510], body[actor.off+511] = 1, 1, 3
		body[actor.off+179] = 64
		body[actor.off+189] = 100 // literal saved mover rate, independent of own speed word
	}
	return body, []*poolFixtureActor{a, b, c}
}

func registryActors1111(t *testing.T, w *sim.World) map[uint16]sim.Entity {
	t.Helper()
	out := map[uint16]sim.Entity{}
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 {
			out[e.SourceBinding.TypeID] = e
		}
	}
	if len(out) != 3 {
		t.Fatalf("source population = %+v", out)
	}
	for _, typ := range []uint16{35, 3, 33} {
		e, ok := out[typ]
		if !ok || e.Owner != 1 || e.Facing != 64 || e.TokenSize != 1 || e.SourceBinding.Face != 3 || e.SourceBinding.GroupIndex != 1 {
			t.Fatalf("source identity/physical inputs for %d: %+v", typ, e)
		}
		if e.MapUnitID != 900 && typ != 35 || typ == 35 && e.MapUnitID != 0 {
			t.Fatalf("invented MapUnitID: %+v", e)
		}
	}
	return out
}

func registryIdentity1111(t *testing.T, w *sim.World, first sim.EntityID) {
	t.Helper()
	for _, want := range []struct {
		typ, index, mapID uint16
		entity            sim.EntityID
		runtime           uint32
		class             uint8
	}{
		{35, 4, 0, 1, 501, 1}, {3, 6, 900, 2, 502, 2}, {33, 7, 900, 3, 503, 2},
	} {
		want.entity += first - 1
		var got sim.Entity
		for _, e := range w.Entities() {
			if e.ID == want.entity {
				got = e
			}
		}
		s := got.SourceBinding
		if got.ID != want.entity || got.MapUnitID != want.mapID || got.Owner != 1 || s.Class != want.class || s.ArchiveIndex != want.index || s.Identity != 1000+uint32(want.index) || s.RuntimeID != want.runtime || s.TypeID != want.typ || s.TokenRow != 1 || s.GroupIndex != 1 || s.GroupOwnerResolved || !got.ActorLoad.Source.HasOwner {
			t.Fatalf("independent source/native namespaces: %+v", got)
		}
	}
}

func TestActorRegistry1111OriginalDoorsNativeMenuFreshProcess(t *testing.T) {
	f := actorRegistryFront1111(t)
	if path := os.Getenv("AGAINROM_REGISTRY_1111_NATIVE"); path != "" {
		store := SaveStore{Dir: filepath.Dir(path)}
		app := f.App("fresh-registry-native")
		save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, filepath.Base(path))
		if got := fmt.Sprintf("%x", f.live.world.Hash()); got != os.Getenv("AGAINROM_REGISTRY_WORLD_HASH") {
			t.Fatalf("fresh SAV changed current World: %s", got)
		}
		var unit sim.Entity
		for _, e := range f.live.world.Entities() {
			if e.SourceBinding.TypeID == 35 {
				unit = e
			}
		}
		if unit.SourceBinding.Class != 1 || unit.X != 16 || unit.Y != 16 {
			t.Fatalf("fresh source-only position: %+v", unit)
		}
		assertRegistryPresentation1111(t, f)
		registryIdentity1111(t, f.live.world, 2)
		if !reflect.DeepEqual(f.live.mission.ids, []sim.EntityID{1, 4}) {
			t.Fatalf("fresh persistent party IDs %v", f.live.mission.ids)
		}
		f.live.enqueue(uint32(unit.ID), 17, 16)
		registryReach1111(t, f.live, unit.ID, 17, 16)
		after, _ := f.live.entity(unit.ID)
		if after.X != 17 || after.Y != 16 || after.Owner != 1 || after.MapUnitID != 0 {
			t.Fatalf("fresh next action: %+v", after)
		}
		f.live.enqueue(3, 17, 18)
		registryReach1111(t, f.live, 3, 17, 18)
		registryCarryBoundary1111(t, f)
		t.Log("source-only actors: ordinary menu SAVE, fresh-process App LOAD, rendered names/persons and next MapOrder PASS")
		return
	}
	payload := actorRegistrySave1111()
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, mapload.DifficultyNormal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := registryActors1111(t, ms.World)
	registryIdentity1111(t, ms.World, 1)
	if len(ms.Start.IDs) != 0 {
		t.Fatalf("new allies became initial party: %v", ms.Start.IDs)
	}
	if source[35].ID != 1 || source[3].ID != 2 || source[33].ID != 3 || ms.World.Entities()[0].ID != 0 || ms.World.Entities()[0].MapUnitID != 91 {
		t.Fatalf("existing placement/new ID namespaces changed: %+v", ms.World.Entities())
	}
	if source[35].HP != 7 || source[3].HP != 9 || source[33].HP != 11 || source[33].SourceBinding.DefinitionRow() != 5 {
		t.Fatal("saved pools/unsigned Human row override lost")
	}
	_, carryIDs := mapload.CarryRosterIDs(ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster)
	if len(carryIDs) != 1 || carryIDs[0] != source[33].ID {
		t.Fatalf("temporary actors promoted to party: %v", carryIDs)
	}
	originals := t.TempDir()
	if err := os.WriteFile(filepath.Join(originals, "game1111.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("registry-original")
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1111.sav")
	source = registryActors1111(t, f.live.world)
	assertRegistryPresentation1111(t, f)
	f.live.enqueue(uint32(source[35].ID), 16, 16)
	registryReach1111(t, f.live, source[35].ID, 16, 16)
	if e, _ := f.live.entity(source[35].ID); e.X != 16 || e.Y != 16 {
		t.Fatalf("initial actual action: %+v", e)
	}
	f.live.enqueue(uint32(source[3].ID), 16, 18)
	registryReach1111(t, f.live, source[3].ID, 16, 18)
	registryIdentity1111(t, f.live.world, 2)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		logRegistryGroupAuthority(t, f)
		t.Fatalf("menu SAVE: %+v %v; %s", entries, err, app.HeadlessMessage())
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestActorRegistry1111OriginalDoorsNativeMenuFreshProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_REGISTRY_1111_NATIVE="+filepath.Join(store.Dir, entries[0].Name),
		fmt.Sprintf("AGAINROM_REGISTRY_WORLD_HASH=%x", f.live.world.Hash()))
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("next MapOrder PASS")) {
		t.Fatalf("fresh process: %v\n%s", err, output)
	}
	t.Log(string(output))
}

func logRegistryGroupAuthority(t *testing.T, f *FrontEnd) {
	t.Helper()
	players, present := f.live.world.SavedGroupPlayers()
	t.Logf("current Players present=%t rows=%+v", present, players)
	groups, _, _ := f.live.world.SavedGroups()
	for _, group := range groups {
		t.Logf("current Group ID=%d selector=%d authored=%t container=%d ownerID=%d owner=%+v members=%+v", group.ID, group.Selector, group.Authored, group.ContainerID, group.OwnerID, group.Owner, group.Members)
	}
	if state := f.live.mission.state.savedDocument; state != nil && state.GroupBindings != nil {
		t.Logf("ordinary Player bindings=%+v", state.GroupBindings.Players)
	}
}

func registryCarryBoundary1111(t *testing.T, f *FrontEnd) {
	t.Helper()
	mw := f.live
	f.FinishMissionWithRoster(10, mw.mission.party, mw.world, mw.mission.ids, mw.mission.state.Start.Roster)
	if len(f.Carried) != 2 || f.Carried[1].Name != "Source persistent" || f.Carried[1].Carry.LiveLoad == nil {
		t.Fatalf("actual mission/city boundary promoted temporary actors or lost persistent source: %+v", f.Carried)
	}
}

func TestActorRegistry1111LateOriginalFailureKeepsLiveGame(t *testing.T) {
	for _, mode := range []string{"late movement domain", "unmatched Humanoid"} {
		t.Run(mode, func(t *testing.T) {
			f := actorRegistryFront1111(t)
			app := f.App("old running game")
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			live, town, hash := f.live, f.Town, f.live.world.Hash()
			body, actors := actorRegistryBody1111()
			payload := savedContainer(body)
			if mode == "late movement domain" {
				body[actors[2].off+510] = 0
				payload = savedContainer(body)
			} else {
				payload = poolFixtureSave(&poolFixtureActor{mapID: 91, hp: 7, maxHP: 31}, &poolFixtureActor{mapID: 999, hp: 9, maxHP: 41, class: "Humanoid"})
			}
			open, _, err := f.RestoreOriginal(payload)
			if err == nil && open != nil {
				err = app.OpenMission(open)
			}
			if err == nil || f.live != live || f.Town != town || f.live.world.Hash() != hash {
				t.Fatalf("late original failure published: %v", err)
			}
		})
	}
}

func TestActorRegistry1111MappedHumanoidNativeCompatibility(t *testing.T) {
	a := &poolFixtureActor{mapID: 91, cell: 0x100f, hp: 7, maxHP: 31, class: "Humanoid", name: "Exact Humanoid"}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil)
	body[a.off+16] = 254 // no supported SAV constructor row: only the ALM arm may admit it
	binary.LittleEndian.PutUint16(body[a.off+17:], 3)
	payload := savedContainer(body)
	f := actorRegistryFront1111(t)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, mapload.DifficultyNormal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	check := func(w *sim.World) {
		e := poolEntity(t, w, 91)
		if e.SourceBinding.Class != 3 || e.SourceBinding.TokenRow != 254 || e.ActorLoad.Source.Class != 2 || !e.Humanoid || e.HP != 7 || e.MaxHP != 31 {
			t.Fatalf("raw Humanoid/native policy identity: %+v", e)
		}
	}
	check(ms.World)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game1111.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("Humanoid")
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1111.sav")
	check(f.live.world)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatalf("menu SAVE: %+v %v; %s", entries, err, app.HeadlessMessage())
	}
	fresh := actorRegistryFront1111(t)
	freshApp := fresh.App("fresh Humanoid")
	s, l, next := fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(s, l, next)
	groundAppLoad(t, freshApp, l, entries[0].Name)
	check(fresh.live.world)
	if fresh.live.world.Hash() != f.live.world.Hash() || !reflect.DeepEqual(fresh.live.mission.state.ActorManifest, f.live.mission.state.ActorManifest) {
		t.Fatalf("Humanoid native reconstruction changed: World %x/%x; manifest before=%+v after=%+v", f.live.world.Hash(), fresh.live.world.Hash(), f.live.mission.state.ActorManifest, fresh.live.mission.state.ActorManifest)
	}
	binary.LittleEndian.PutUint32(body[a.off+19:], 999)
	if _, _, err := fresh.RestoreOriginal(savedContainer(body)); err == nil || !strings.Contains(err.Error(), "Humanoid") {
		t.Fatalf("unmatched exact Humanoid admitted: %v", err)
	}
}

func TestActorRegistry1111EffectiveOwnerSupersedesUnresolvedToken(t *testing.T) {
	body, actors := actorRegistryBody1111()
	binary.LittleEndian.PutUint32(body[actors[1].off+33:], 0xdeadbeef)
	f := actorRegistryFront1111(t)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, savedContainer(body), f.Table, mapload.DifficultyNormal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	registryIdentity1111(t, ms.World, 1)
}

func TestActorRegistry1111SourceOnlyHumanEquipmentNativeNextCommand(t *testing.T) {
	armor := &holdingFixtureItem{class: "Armor", code: 0x0701, row: 1, ownKind: 7,
		count: 2, kind: 1, weight: 2, defence: [22]byte{7}}
	acc := int32(4)
	a := &poolFixtureActor{cell: 0x100f, hp: 9, maxHP: 41, human: true, name: "Source equipment",
		loadWords: &[4]int16{100, 0, 2, 300}, equipmentRuntime: &[3]byte{1, 8, 4},
		holdings: &holdingFixture{items: []*holdingFixtureItem{armor}, accumulator: &acc}}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil)
	binary.LittleEndian.PutUint16(body[a.off+17:], 33)
	wantItem := sim.ItemInstance{Code: 0x0701, Kind: 1, WeightPresent: true, Weight: 2,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 7, Defence: [22]byte{7}}}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game1111.sav"), savedContainer(body), 0600); err != nil {
		t.Fatal(err)
	}
	f := actorRegistryFront1111(t)
	app := f.App("source equipment")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1111.sav")
	var id sim.EntityID
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class == 2 {
			id = e.ID
			if e.MapUnitID != 0 || e.Group != 0 || e.SourceBinding.GroupIndex != 1 {
				t.Fatal("source-only identity/native unbound policy", e)
			}
		}
	}
	if id == 0 {
		t.Fatal("source-only Human missing")
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindEquip, Entity: id, X: 0, Y: 7})
	f.live.tick()
	check := func(w *sim.World, worn bool) {
		t.Helper()
		eq, _ := w.EquippedItems(id)
		pack, _ := w.CarriedStacks(id)
		wantCount, wantDefence, wantOwn := uint32(2), uint16(0), int16(0)
		if worn {
			// Armor adds literal7 to saved defence0. Neither load crosses a
			// capacity quotient, so this attach does not request a full derive.
			wantCount, wantDefence, wantOwn = 1, 7, 2
			if !reflect.DeepEqual(eq[6], wantItem) {
				t.Fatal("source armor not equipped", eq[6])
			}
		} else if !eq[6].Empty() {
			t.Fatal("source armor not removed")
		}
		if len(pack) != 1 || pack[0].Count != wantCount || !reflect.DeepEqual(pack[0].Instance(), wantItem) {
			t.Fatal("source quantity/operands", pack)
		}
		for _, e := range w.Entities() {
			if e.ID == id && (e.ActorLoad.OwnWeight != wantOwn || e.ActorLoad.Source.Stats[7] != 300 || binary.LittleEndian.Uint16(e.ActorLoad.Source.Defence[:]) != wantDefence) {
				t.Fatal("source Human next-equipment arithmetic", e.ActorLoad)
			}
		}
	}
	check(f.live.world, true)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatalf("menu SAVE: %+v %v; %s", entries, err, app.HeadlessMessage())
	}
	fresh := actorRegistryFront1111(t)
	freshApp := fresh.App("fresh source equipment")
	s, l, next := fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(s, l, next)
	groundAppLoad(t, freshApp, l, entries[0].Name)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("source equipment native identity/state changed")
	}
	check(fresh.live.world, true)
	fresh.live.pending = append(fresh.live.pending, sim.Command{Kind: sim.KindUnequip, Entity: id, X: 7})
	fresh.live.tick()
	check(fresh.live.world, false)
}

func registryReach1111(t *testing.T, mw *mapWorld, id sim.EntityID, x, y int32) {
	t.Helper()
	for i := 0; i < 96; i++ {
		mw.tick()
		if e, ok := mw.entity(id); ok && e.X == x && e.Y == y {
			return
		}
	}
	for _, e := range mw.world.Entities() {
		t.Logf("actor %d at%d,%d target%d,%d/%v", e.ID, e.X, e.Y, e.TargetX, e.TargetY, e.HasTarget)
	}
	t.Fatal("source actor did not reach ordered destination")
}

func assertRegistryPresentation1111(t *testing.T, f *FrontEnd) {
	t.Helper()
	names := map[uint16]string{35: "Source creature", 3: "Source temporary", 33: "Source persistent"}
	draws := map[uint32]ui.MapEntity{}
	for _, draw := range f.live.entityDraws() {
		draws[draw.ID] = draw
	}
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class == 0 {
			continue
		}
		draw := draws[uint32(e.ID)]
		if draw.Name != names[e.SourceBinding.TypeID] || draw.Art == nil || draw.Owner != 1 {
			t.Fatalf("missing render/control binding: %+v", draw)
		}
		if e.Humanoid {
			ch, ok := f.live.chars[e.ID]
			if !ok || ch.Body != 30 || ch.Reaction != 20 || f.live.figures[e.ID].Face != 3 {
				t.Fatalf("missing saved person: %+v figure=%+v", ch, f.live.figures[e.ID])
			}
		}
	}
	if f.live.view.LocalOwner() != sim.SelfSlot {
		t.Fatal("not a controllable player view")
	}
}
