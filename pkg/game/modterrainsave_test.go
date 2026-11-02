package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// modGroupMission is the cell-state fixture mission with cell (20,20)
// painted with the snow-field mod's group 31 and cell (21,20) with its group
// 13; rows nil leaves the front without the mod.
func modGroupMission(t *testing.T, rows *mod.TerrainData, all mod.TerrainData) *FrontEnd {
	t.Helper()
	return modGroupCells(t, rows, all, groupCell{31, 20}, groupCell{13, 21})
}

// groupCell is cell (x,20) of the fixture map painted with a mod group.
type groupCell struct{ group, x int }

// modGroupCells is modGroupMission with the given cells painted under the
// rows of all.
func modGroupCells(t *testing.T, rows *mod.TerrainData, all mod.TerrainData, cells ...groupCell) *FrontEnd {
	t.Helper()
	table := &mapload.Table{}
	table.Mods.Terrain = all
	paint := func(raw []byte) []byte {
		ed, err := mapedit.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cells {
			op := ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: mapedit.GroupBrush(c.group), Cells: []image.Point{{X: c.x, Y: 20}}}
			if _, err := paintStroke(ed, op, table); err != nil {
				t.Fatal(err)
			}
		}
		return ed.Bytes()
	}
	f := cellStateFrontWith(t, paint)
	if rows != nil {
		f.Table.Mods.Terrain = *rows
	}
	return f
}

// A mission on a map holding mod groups opens with the rows' land kinds in
// its cell planes, saves, and loads on a fresh front into the same cells and
// planes; both worlds then run on alike. Without the mod the map is refused
// naming the group.
func TestModGroupMissionSavesAndLoads(t *testing.T) {
	rows, _, err := checkSnowField(t, snowFieldMod(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	f := modGroupMission(t, &rows, rows)
	app := f.App("mod group mission")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	m := f.live.mission.state.Map
	ice, snow := m.Tiles[20*m.Width+20], m.Tiles[20*m.Width+21]
	if alm.TileGroup(ice) != 31 || alm.TileGroup(snow) != 13 {
		t.Fatalf("mission map words %#04x %#04x", ice, snow)
	}
	planes, present := f.live.world.SavedCellPlanes()
	if !present || planes.Static[20*256+20] != 1 || planes.Static[20*256+21] != 0 || planes.Cost[20*256+21] != planes.Costs[2] {
		t.Fatalf("mod group cells' planes: present %t static %d %d cost %d", present, planes.Static[20*256+20], planes.Static[20*256+21], planes.Cost[20*256+21])
	}
	f.live.world.PruneGroupsForSave() // SAVE prunes first; the rest of it changes nothing
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	name := crossingMenuSave(t, app, store)

	fresh := modGroupMission(t, &rows, rows)
	freshApp := fresh.App("fresh mod group LOAD")
	save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, localOriginalSaveToken(name))
	loaded := fresh.live.mission.state.Map
	if loaded.Tiles[20*loaded.Width+20] != ice || loaded.Tiles[20*loaded.Width+21] != snow {
		t.Fatal("LOAD read other words into the mod group cells")
	}
	after, present := fresh.live.world.SavedCellPlanes()
	if !present || after.Static[20*256+20] != planes.Static[20*256+20] || after.Cost[20*256+21] != planes.Cost[20*256+21] {
		t.Fatal("LOAD rebuilt the mod group cells' planes otherwise")
	}
	for tick := 1; tick <= 10; tick++ {
		f.live.tick()
		fresh.live.tick()
		if adoptLoadedAllocator(t, f.live.world, fresh.live.world).Hash() != fresh.live.world.Hash() {
			t.Fatal("the loaded mod group mission diverged", tick)
		}
	}

	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	bare := modGroupMission(t, nil, rows)
	open, _, err := bare.RestoreOriginal(raw)
	if err == nil {
		err = bare.App("LOAD without the mod").OpenMission(open)
	}
	if err == nil || !strings.Contains(err.Error(), "10.alm") || !strings.Contains(err.Error(), "terrain group 13") {
		t.Fatalf("LOAD without the mod: %v", err)
	}
}
