package game

import "image"

// firstRouteStages is the first game's step table, in route order: the
// second game's rows reduced. A new game opens the first mission from the
// generator, so the first town comes after the missions before it, which
// the walk wins with the chat cheat; the town has no statue and no blank
// square row. The town rows are the walk a town save takes after its load.
var firstRouteStages = []routeStage{
	{"menu", "menu", routeKindMenu, routeData{}},
	{"generator-precreate", "chargen/precreate", routeKindPreCreate, routeData{}},
	{"generator-detail", "chargen/detailed", routeKindDetail, routeData{play: "map"}},
	{"mission-dialogue", "map", routeKindMissionDialogue, routeData{newGame: true}},
	{"mission-screen", "map", routeKindMissionScreen, routeData{}},
	{"mission-hero", "map", routeKindMissionHero, routeData{}},
	{"to-town", "map", routeKindToTown, routeData{}},
	{"town-square", "town", routeKindSquare, routeData{arrive: "town", tip: routeTipClosePoint, town: true}},
	{"tavern", "town/square", routeKindRoom, routeData{tip: routeTipClosePoint, town: true, room: routeRoomData{door: "TAVERN", room: "tavern", surface: true}}},
	{"school", "town/square", routeKindRoom, routeData{tip: routeTipClosePoint, town: true, room: routeRoomData{door: "SCHOOL", room: "school", surface: true}}},
	{"shop", "town/square", routeKindShop, routeData{tip: routeTipClosePoint, town: true}},
	{"way-out", "town/square", routeKindWayOut, routeData{}},
	{"world-map", "town/worldmap", routeKindWorldMap, routeData{choose: true}},
	{"next-mission-dialogue", "map", routeKindMissionDialogue, routeData{}},
}

// routeSideScreen is a menu side screen a trip found on the mask opens:
// whether the walk waits for its roll before Esc, and its own buttons at
// their middles.
type routeSideScreen struct {
	roll    bool
	buttons []routeSidePoint
}

// routeSidePoint is a side screen's button and its middle.
type routeSidePoint struct {
	what string
	at   image.Point
}

// routeSideScreens is the first game's side screens by name: the Load
// window's Cancel, the Hall of Fame's art button and its Back, and the
// movie library's Cancel. Credits end on any press.
var routeSideScreens = map[string]routeSideScreen{
	"load":      {buttons: []routeSidePoint{{"cancel", image.Pt(441, 380)}}},
	"ending":    {buttons: []routeSidePoint{{"hall art button", image.Pt(575, 435)}, {"back", image.Pt(98, 446)}}},
	"cutscenes": {buttons: []routeSidePoint{{"cancel", image.Pt(404, 372)}}},
	"credits":   {roll: true, buttons: []routeSidePoint{{"press", image.Pt(320, 240)}}},
}
