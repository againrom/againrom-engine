package game

import (
	"encoding/json"
	"image"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/town"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// travelCatalog is a fixture catalog of two first-game towns: West, town 1
// at the first registry object, and East, town 2 at the third, under a
// description of its own name, offering mission 31.
const travelCatalog = `{"catalog": "fixture", "cite": [], "towns": [
  {"id": 1, "description": "rom1", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 1},
   "name": "West", "cite": []},
  {"id": 2, "description": "rom1-east", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 3},
   "name": "East", "missions": [31], "cite": []}]}`

// The points of the travel fixture's registry objects.
var (
	travelWest      = image.Pt(2, 3)
	travelMission30 = image.Pt(32, 23)
	travelEast      = image.Pt(2, 23)
	travelMission31 = image.Pt(32, 3)
)

// travelBook decodes raw over the shipped descriptions and rom1-east, the
// first game's description under a second name.
func travelBook(t *testing.T, raw string) *townBook {
	t.Helper()
	data := maps.Clone(townDescriptionData)
	data["rom1-east"] = rom1TownJSON
	b, err := decodeTownBook([]byte(raw), data)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// travelMapFS is an install whose world map has four objects at the corners
// of one square corridor: West, mission 30's, East and mission 31's.
func travelMapFS(t *testing.T) *vfs.FS {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, files []synth.File) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	object := func(n int, p image.Point) synth.RegNode {
		return synth.RegNode{Name: "MapObject" + string(rune('0'+n)), Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{int32(p.X), int32(p.Y)}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{int32(p.X), int32(p.Y), 4, 4}},
		}}
	}
	registry := synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{{Name: "ObjectCount", Kind: kindInt, Int: 4}}},
		object(1, travelWest), object(2, travelMission30), object(3, travelEast), object(4, travelMission31),
		{Name: "MissionObjects", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mission30", Kind: kindInt, Int: 2},
			{Name: "Mission31", Kind: kindInt, Int: 4},
		}},
	})
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), worldMapPalette)
	for x := 2; x <= 32; x++ {
		mask.SetColorIndex(x, 3, 1)
		mask.SetColorIndex(x, 23, 1)
	}
	for y := 3; y <= 23; y++ {
		mask.SetColorIndex(2, y, 1)
		mask.SetColorIndex(32, y, 1)
	}
	for _, p := range []image.Point{travelWest, travelMission30, travelEast, travelMission31} {
		mask.SetColorIndex(p.X, p.Y, 2)
	}
	main := write(MainArchive, []synth.File{{Path: "graphics/global.map/gmap.bmp", Data: synth.BMP24(image.NewRGBA(image.Rect(0, 0, 640, 480)))}})
	graphics := write(GraphicsArchive, []synth.File{{Path: "global.map/pathmap.bmp", Data: synth.BMP8(mask)}})
	scenario := write(ScenarioArchive, []synth.File{{Path: "globalmap.reg", Data: registry}})
	fsys, err := vfs.Open([]string{main, graphics, scenario}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

// travelFront is a front end over the travel fixture's map and catalog, its
// party in West with the town open.
func travelFront(t *testing.T, raw string) (*FrontEnd, *townScreen) {
	t.Helper()
	c := townCampaign(t)
	c.MercenaryCount = []int{4}
	c.Towns = travelBook(t, raw)
	tw := NewTown(c)
	tw.Arrive()
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: travelMapFS(t)},
		Font: resolved(missionFont(), nil)}, CampaignSession: CampaignSession{Town: tw}}
	return f, f.TownScreen().(*townScreen)
}

// travelCard clicks record number of kind on the open map and runs the
// travel to its end; it returns the arrival's action and the route.
func travelCard(t *testing.T, s *townScreen, kind, number int) (ui.TownAction, []image.Point) {
	t.Helper()
	v := s.WorldMapView()
	i := slices.IndexFunc(v.Missions, func(m ui.WorldMapMission) bool { return m.Kind == kind && m.Number == number })
	if i < 0 {
		t.Fatalf("the map shows no record %d/%d: %+v", kind, number, v.Missions)
	}
	card := ui.WorldMapCardRect(i)
	s.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	route := s.WorldMapView().Route
	if len(route) < 2 {
		t.Fatalf("record %d/%d: no route", kind, number)
	}
	for range 200 {
		if act := s.WorldMapTick(); act.Open != nil || s.AtTownSquare() {
			return act, route
		}
	}
	t.Fatalf("the travel to record %d/%d never arrived", kind, number)
	return ui.TownAction{}, nil
}

// runHome runs a homeward trip to its end and answers where the party
// stands.
func runHome(t *testing.T, s *townScreen) image.Point {
	t.Helper()
	for range 200 {
		s.WorldMapTick()
		if s.AtTownSquare() {
			return s.worldPosition
		}
	}
	t.Fatal("the homeward trip never arrived")
	return image.Point{}
}

func offerMissions(offers []TownOffer) []int {
	var out []int
	for _, o := range offers {
		out = append(out, o.Mission)
	}
	return out
}

// The world map shows the other open town at its entry's map point; the
// party travels from West to East and back, each town keeping its offers
// and mercenaries and composing from its own description; a mission taken in
// East returns the party to East on its win.
func TestPartyTravelsBetweenTwoChapterTowns(t *testing.T) {
	f, s := travelFront(t, travelCatalog)
	tw := f.Town
	if got := offerMissions(tw.Offers(TownTavern)); !slices.Equal(got, []int{30, 0}) || len(tw.Offers(TownShop)) != 0 {
		t.Fatalf("West offers tavern %v shop %v, want tavern [30 0] and no shop offer", got, tw.Offers(TownShop))
	}
	s.Choose(3)
	v := s.WorldMapView()
	if !v.AtHome || v.Position != travelWest || len(v.Missions) != 1 || v.Missions[0].Kind != placeTown ||
		v.Missions[0].Anchor != travelEast || v.Missions[0].Title != "East" || !v.Missions[0].Enabled {
		t.Fatalf("West's map: home %v at %v, records %+v; want the party at %v and East at %v", v.AtHome, v.Position, v.Missions, travelWest, travelEast)
	}
	if act, route := travelCard(t, s, placeTown, 2); act.Open != nil || route[0] != travelWest || route[len(route)-1] != travelEast {
		t.Fatalf("the travel to East opened %v along %v", act.Open != nil, route)
	}
	if tw.townID() != 2 || !s.AtTownSquare() || s.worldPosition != travelEast {
		t.Fatalf("arrival: town %d, square %v, position %v", tw.townID(), s.AtTownSquare(), s.worldPosition)
	}
	if got := s.townDescription(); got == nil || got != tw.camp.Towns.descriptions["rom1-east"] || got == TownDescription(s.profile()) {
		t.Fatal("East's screen does not compose from East's own description")
	}
	if got := offerMissions(tw.Offers(TownShop)); !slices.Equal(got, []int{31}) || len(tw.Offers(TownTavern)) != 0 {
		t.Fatalf("East offers shop %v tavern %v, want shop [31] and no tavern offer", got, tw.Offers(TownTavern))
	}
	if _, ok := tw.Take(TownTavern, 0); ok {
		t.Fatal("East's tavern handed out West's mission")
	}
	tw.gold = 1 << 20
	if m, ok := tw.Take(TownShop, 0); !ok || m != 31 {
		t.Fatalf("East's shop take: %d %v", m, ok)
	}
	tw.mercenaryHire(1, 0)

	s.Choose(3)
	v = s.WorldMapView()
	if !v.AtHome || v.Position != travelEast || v.Words.WorldHomeTitle != "East" {
		t.Fatalf("East's map: home %v at %v titled %q, want home East at %v", v.AtHome, v.Position, v.Words.WorldHomeTitle, travelEast)
	}
	travelCard(t, s, placeTown, 1)
	if tw.townID() != 1 || tw.MercenaryHired(1) || !slices.Equal(offerMissions(tw.Offers(TownTavern)), []int{30, 0}) {
		t.Fatalf("back in West: town %d, hired %v, tavern %v", tw.townID(), tw.MercenaryHired(1), offerMissions(tw.Offers(TownTavern)))
	}
	if s.townDescription() != tw.camp.Towns.descriptions["rom1"] {
		t.Fatal("West's screen does not compose from West's description")
	}
	s.Choose(3)
	travelCard(t, s, placeTown, 2)
	if tw.townID() != 2 || !tw.MercenaryHired(1) || len(tw.Offers(TownShop)) != 0 {
		t.Fatalf("East again: town %d, hired %v, shop %v", tw.townID(), tw.MercenaryHired(1), tw.Offers(TownShop))
	}

	s.Choose(3)
	if act, _ := travelCard(t, s, 0, 31); act.Open == nil {
		t.Fatal("the travel to mission 31 opened nothing")
	}
	tw.Won(31)
	// Loss control: the same return toward a fixed home point, the starting
	// town's, ends in West.
	s.beginWorldMapReturn(31)
	s.worldMap.homeTown = town.Entry{}
	s.worldMap.route = worldMapRoute(s.worldMap.assets.mode(s.worldMap.mode).graph, s.worldPosition, s.worldMap.home(), worldMapFallbackStep)
	if at := runHome(t, s); at != travelWest {
		t.Fatalf("loss control: a fixed home point ends the return at %v, want West %v", at, travelWest)
	}
	s.beginWorldMapReturn(31)
	if got := s.worldMap.route; len(got) == 0 || got[0] != travelMission31 || got[len(got)-1] != travelEast {
		t.Fatalf("the return route %v does not run from mission 31 to East", got)
	}
	if at := runHome(t, s); at != travelEast || tw.townID() != 2 {
		t.Fatalf("the win returned the party to %v in town %d, want East %v", at, tw.townID(), travelEast)
	}
}

// A town that opens after a mission is no travel record until the mission is
// done.
func TestTownOpensAfterItsMission(t *testing.T) {
	_, s := travelFront(t, strings.Replace(travelCatalog, `"missions": [31]`, `"missions": [31], "opens-after": 30`, 1))
	gateOpen := townConditions["gate-open"]
	if gateOpen(s) || len(s.sess.Town.travelTowns()) != 0 {
		t.Fatal("the gate opens on a town not yet open")
	}
	if s.sess.Town.travelTo(2) {
		t.Fatal("the party travelled to a town not yet open")
	}
	s.sess.Town.Won(30)
	if !gateOpen(s) {
		t.Fatal("the gate stays closed with East open")
	}
	s.Choose(3)
	if v := s.WorldMapView(); !slices.ContainsFunc(v.Missions, func(m ui.WorldMapMission) bool { return m.Kind == placeTown && m.Number == 2 }) {
		t.Fatalf("records after mission 30: %+v, want East", v.Missions)
	}
}

// The shipped first game holds one town: no record, no state besides its
// own, no town record in a SAVE, and its home at the first MapObject.
func TestShippedFirstGameHoldsOneTown(t *testing.T) {
	c := townCampaign(t)
	c.Towns = townBooks["rom1"]
	tw := NewTown(c)
	tw.Arrive()
	if tw.others != nil || tw.travelTowns() != nil || tw.townsRecord() != nil || tw.townEntry().MapPoint.Object != 1 {
		t.Fatalf("shipped town: others %v, travel %v, record %+v, entry %+v", tw.others, tw.travelTowns(), tw.townsRecord(), tw.townEntry())
	}
	a := &worldMapAssets{data: &globalMapData{Objects: []globalMapObject{{Point: image.Pt(7, 9), Valid: true}}}}
	if got := worldMapHome(tw.townEntry(), a); got != image.Pt(7, 9) {
		t.Fatalf("home %v, want the first MapObject's point", got)
	}
	if got := worldMapHome(town.Entry{}, &worldMapAssets{}); got != worldMapNoPoint {
		t.Fatalf("home without a registry %v, want the map's centre", got)
	}
}

// The town record a SAVE writes carries the party's town and the other
// towns' states through the session leaf's JSON; no record is town 1 with
// fresh towns. The loss control drops the record: the LOAD lands in town 1
// with East's hire lost.
func TestTownRecordCarriesThePartysTownAcrossLoad(t *testing.T) {
	f, s := travelFront(t, travelCatalog)
	tw := f.Town
	tw.mercenaryHire(1, 0)
	s.Choose(3)
	travelCard(t, s, placeTown, 2)
	tw.gold = 1 << 20
	tw.Take(TownShop, 0)
	r := tw.townsRecord()
	if r == nil || r.Town != 2 || len(r.Others) != 1 || r.Others[0].ID != 1 || !r.Others[0].Hired[0] {
		t.Fatalf("record %+v, want town 2 and West's hire", r)
	}
	raw, err := json.Marshal(&currentSessionData{Towns: r})
	if err != nil {
		t.Fatal(err)
	}
	load := func(raw []byte) *Town {
		var back currentSessionData
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatal(err)
		}
		loaded := NewTown(tw.camp)
		loaded.Arrive()
		loaded.townState.taken = maps.Clone(tw.taken)
		if err := loaded.restoreTowns(back.Towns); err != nil {
			t.Fatal(err)
		}
		return loaded
	}
	loaded := load(raw)
	if loaded.townID() != 2 || len(loaded.Offers(TownShop)) != 0 || loaded.travelTo(1) == false || !loaded.MercenaryHired(1) {
		t.Fatalf("LOAD: town %d, shop %v, West's hire %v", loaded.townID(), loaded.Offers(TownShop), loaded.MercenaryHired(1))
	}
	dropped, _ := json.Marshal(&currentSessionData{})
	if strings.Contains(string(dropped), "Towns") {
		t.Fatalf("a session with no record writes %s", dropped)
	}
	if lost := load(dropped); lost.townID() != 1 || lost.MercenaryHired(1) {
		t.Fatal("loss control: a dropped record still lands in town 2 or keeps West's hire")
	}
	if !reflect.DeepEqual(r, r.clone()) {
		t.Fatal("clone differs")
	}
	bad := NewTown(tw.camp)
	for _, rec := range []*currentTowns{{Town: 3}, {Town: 2, Others: []currentTownState{{ID: 2}}}, {Town: 2, Others: []currentTownState{{ID: 1}, {ID: 1}}}} {
		if err := bad.restoreTowns(rec); err == nil {
			t.Errorf("record %+v restored", rec)
		}
	}
}

// A catalog entry whose description its screen cannot compose fails the
// decode, as do opening and mission data on a scenario-bank town and chapter
// towns that do not start in town 1.
func TestCatalogRefusesATownWithNoUsableDescription(t *testing.T) {
	var desc map[string]any
	if err := json.Unmarshal(rom1TownJSON, &desc); err != nil {
		t.Fatal(err)
	}
	// The school room is renamed, so the description decodes but holds no
	// room the chapters hook set enters.
	for _, r := range desc["rooms"].([]any) {
		if r := r.(map[string]any); r["name"] == "school" {
			r["name"] = "library"
		}
	}
	for _, h := range desc["hotspots"].([]any) {
		for _, c := range h.(map[string]any)["click"].([]any) {
			if c := c.(map[string]any); c["room"] == "school" {
				c["room"] = "library"
			}
		}
	}
	noSchool, err := json.Marshal(desc)
	if err != nil {
		t.Fatal(err)
	}
	data := maps.Clone(townDescriptionData)
	data["rom1-no-school"] = noSchool
	entry := `{"id": 1, "description": "rom1", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 1}, "cite": []}`
	catalog := func(entries ...string) []byte {
		return []byte(`{"catalog": "fixture", "cite": [], "towns": [` + strings.Join(entries, ",") + `]}`)
	}
	for _, c := range []struct{ name, want string }{
		{strings.Replace(entry, `"rom1"`, `"rom1-no-school"`, 1), `no room "school"`},
		{strings.Replace(entry, `"chapters"`, `"destinations"`, 1), "no usable description"},
		{strings.Replace(entry, `"id": 1`, `"id": 2`, 1), "start in town 1"},
		{strings.Replace(entry, `"cite"`, `"opens-after": 30, "cite"`, 1), "start in town 1"},
	} {
		if _, err := decodeTownBook(catalog(c.name), data); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want one naming %q", c.name, err, c.want)
		}
	}
	bank := `{"id": 2, "description": "rom2-town2", "hooks": "destinations", "offers": "scenario-bank", "map-point": {"record": true}, "opens-after": 20, "cite": []}`
	if _, err := decodeTownBook(catalog(bank), data); err == nil || !strings.Contains(err.Error(), "scenario bank") {
		t.Errorf("a scenario-bank town with an opening mission: %v", err)
	}
}
