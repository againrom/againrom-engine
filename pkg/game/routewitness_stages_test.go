package game

import (
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/base"
)

// routeStageRow is a stage row as the test states it: its name, the screen
// it opens on and its stage kind.
type routeStageRow struct {
	name, screen string
	kind         routeKind
}

func routeStageRows(stages []routeStage) []routeStageRow {
	out := make([]routeStageRow, 0, len(stages))
	for _, s := range stages {
		out = append(out, routeStageRow{s.name, s.screen, s.kind})
	}
	return out
}

// TestRouteStageRows pins each game's step table in route order: the second
// game's route with its statue and blank square, and the first game's
// reduced rows, mission first, with the town after the win; and the town
// rows a town save walks.
func TestRouteStageRows(t *testing.T) {
	secondRows := []routeStageRow{
		{"menu", "menu", routeKindMenu},
		{"generator-precreate", "chargen/precreate", routeKindPreCreate},
		{"generator-detail", "chargen/detailed", routeKindDetail},
		{"town-square", "town/square", routeKindSquare},
		{"town-statue", "town/square", routeKindStatue},
		{"tavern", "town/square", routeKindRoom},
		{"shop", "town/square", routeKindShop},
		{"town-blank", "town/square", routeKindBlank},
		{"way-out", "town/square", routeKindWayOut},
		{"world-map", "town/worldmap", routeKindWorldMap},
		{"mission-dialogue", "map", routeKindMissionDialogue},
		{"mission-screen", "map", routeKindMissionScreen},
		{"mission-menu", "map", routeKindMissionMenu},
		{"mission-pack", "map", routeKindMissionPack},
		{"mission-signpost", "map", routeKindSignpost},
		{"mission-exit", "map", routeKindMissionExit},
	}
	firstRows := []routeStageRow{
		{"menu", "menu", routeKindMenu},
		{"generator-precreate", "chargen/precreate", routeKindPreCreate},
		{"generator-detail", "chargen/detailed", routeKindDetail},
		{"mission-dialogue", "map", routeKindMissionDialogue},
		{"mission-screen", "map", routeKindMissionScreen},
		{"mission-hero", "map", routeKindMissionHero},
		{"to-town", "map", routeKindToTown},
		{"town-square", "town", routeKindSquare},
		{"tavern", "town/square", routeKindRoom},
		{"school", "town/square", routeKindRoom},
		{"shop", "town/square", routeKindShop},
		{"way-out", "town/square", routeKindWayOut},
		{"world-map", "town/worldmap", routeKindWorldMap},
		{"next-mission-dialogue", "map", routeKindMissionDialogue},
	}
	if got := routeStageRows(campaignOf(base.GameROM2).routeStages()); !slices.Equal(got, secondRows) {
		t.Errorf("second game's rows %v, want %v", got, secondRows)
	}
	if got := routeStageRows(campaignOf(base.GameROM1).routeStages()); !slices.Equal(got, firstRows) {
		t.Errorf("first game's rows %v, want %v", got, firstRows)
	}
	town := []routeStageRow{firstRows[7], firstRows[8], firstRows[9], firstRows[10]}
	if got := routeStageRows(routeTownRows(campaignOf(base.GameROM1).routeStages())); !slices.Equal(got, town) {
		t.Errorf("first game's town save rows %v, want %v", got, town)
	}
	if got := routeTownRows(campaignOf(base.GameROM2).routeStages()); len(got) != 0 {
		t.Errorf("second game's town save rows %v, want none", routeStageRows(got))
	}
}

// TestRouteStageRowData pins what the two tables' rows of one kind do
// differently: the second game's menu trips and launch movie, its inn and
// shop, and the first game's surface rooms, tip points and mission choice.
func TestRouteStageRowData(t *testing.T) {
	row := func(stages []routeStage, name string) routeData {
		for _, s := range stages {
			if s.name == name {
				return s.data
			}
		}
		t.Fatalf("no row %s", name)
		return routeData{}
	}
	rom2, rom1 := campaignOf(base.GameROM2).routeStages(), campaignOf(base.GameROM1).routeStages()
	if m := row(rom2, "menu").menu; m.launch.movie != "logos/nival.smk" || len(m.trips) != 4 || len(m.notes) != 1 || m.buttons == 0 {
		t.Errorf("second game's menu %+v", m)
	}
	if m := row(rom1, "menu").menu; m.launch.movie != "" || m.trips != nil || m.notes != nil || m.buttons != 0 {
		t.Errorf("first game's menu %+v, want the reduced walk", m)
	}
	if d := row(rom2, "tavern"); d.room != (routeRoomData{door: "TAVERN", room: "inn", exit: secondInnExit, pane: true, talks: true}) || d.tip != routeTipCloseButton {
		t.Errorf("second game's tavern %+v", d)
	}
	for _, name := range []string{"tavern", "school"} {
		if d := row(rom1, name); !d.room.surface || d.room.exit != "" || d.room.pane || d.room.talks || d.tip != routeTipClosePoint || d.room.door != strings.ToUpper(name) {
			t.Errorf("first game's %s %+v", name, d)
		}
	}
	if d := row(rom2, "shop").shop; d != (routeShopData{pane: true, inspect: true, named: true, undo: true}) {
		t.Errorf("second game's shop %+v", d)
	}
	if d := row(rom1, "shop"); d.shop != (routeShopData{}) || d.tip != routeTipClosePoint {
		t.Errorf("first game's shop %+v", d)
	}
	if !row(rom1, "world-map").choose || row(rom2, "world-map").choose {
		t.Error("only the first game's world map chooses its mission")
	}
	if d := row(rom2, "mission-screen"); !d.spellbook || !d.heroShown {
		t.Errorf("second game's mission screen %+v", d)
	}
	if d := row(rom1, "mission-screen"); d.spellbook || d.heroShown {
		t.Errorf("first game's mission screen %+v", d)
	}
	if row(rom2, "generator-detail").play != "town/square" || row(rom1, "generator-detail").play != "map" {
		t.Error("the generator's Play opens the second game's town and the first game's mission")
	}
}

// TestRouteStageKindsHaveOneWalk states that every stage kind has exactly
// one walk, a routeWitness method, no two kinds share one, and both tables
// use every kind.
func TestRouteStageKindsHaveOneWalk(t *testing.T) {
	seen := map[string]routeKind{}
	for k, walk := range routeKindWalks {
		if walk == nil {
			t.Errorf("kind %d has no walk", k)
			continue
		}
		name := runtime.FuncForPC(reflect.ValueOf(walk).Pointer()).Name()
		if !strings.Contains(name, "(*routeWitness).") || strings.HasSuffix(name, "-fm") || strings.Contains(name, ".func") {
			t.Errorf("kind %d walks %s, want a routeWitness method", k, name)
		}
		if other, ok := seen[name]; ok {
			t.Errorf("kinds %d and %d share the walk %s", other, k, name)
		}
		seen[name] = routeKind(k)
	}
	used := map[routeKind]bool{}
	for _, stages := range [][]routeStage{campaignOf(base.GameROM1).routeStages(), campaignOf(base.GameROM2).routeStages()} {
		for _, s := range stages {
			if s.kind < 0 || s.kind >= routeKindCount {
				t.Errorf("row %s has kind %d outside the kinds", s.name, s.kind)
			}
			used[s.kind] = true
		}
	}
	for k := range routeKindCount {
		if !used[k] {
			t.Errorf("kind %d has no row", k)
		}
	}
}
