package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func worldMapRecordValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface || v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.IsNil() {
		return nil
	}
	if v.CanInterface() {
		if pic, ok := v.Interface().(image.Image); ok {
			r := pic.Bounds()
			h := sha256.New()
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					r, g, b, a := pic.At(x, y).RGBA()
					fmt.Fprintf(h, "%04x%04x%04x%04x", r, g, b, a)
				}
			}
			return []any{r, fmt.Sprintf("%x", h.Sum(nil))}
		}
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return worldMapRecordValue(v.Elem())
	case reflect.Struct:
		out := map[string]any{}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				out[v.Type().Field(i).Name] = worldMapRecordValue(v.Field(i))
			}
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, v.Len())
		for i := range out {
			out[i] = worldMapRecordValue(v.Index(i))
		}
		return out
	default:
		return v.Interface()
	}
}

func worldMapPageRecords(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	record := func(name string, s *townScreen, act ui.TownAction) {
		v := s.WorldMapView()
		state := s.worldMapNow()
		snap := Snapshot{}
		snapshotWorldSelectedOnce(s, &snap)
		first, restored := s.sess.Town.firstMapPoint()
		mode, cross := -1, -1
		if state != nil {
			mode, cross = state.mode, state.cross
		}
		value := map[string]any{
			"view": worldMapRecordValue(reflect.ValueOf(v)), "mode": mode, "cross": cross,
			"atMap": s.AtWorldMap(), "room": s.roomName(), "position": s.worldPosition,
			"positionSet": s.worldPositionSet, "selectedOnce": snap.WorldSelectedOnce,
			"firstMapPoint": first, "restored": restored, "selectedMission": s.sess.Town.selectedMission(),
			"open": act.Open != nil, "message": act.Msg, "info": act.Info,
			"frame": worldMapRecordValue(reflect.ValueOf(ui.ComposeWorldMap(v))),
		}
		if c := s.state(); c != nil {
			value["current"], value["bank"], value["available"] = c.current, c.bank, c.available
		}
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&out, "%s %x\n", name, sha256.Sum256(data))
	}
	first := func() (*FrontEnd, *townScreen) {
		c := townCampaign(t)
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: worldMapFixtureFS(t)}, Font: resolved(missionFont(), nil), Words: ui.AuthoredWords()}, CampaignSession: CampaignSession{Town: NewTown(c)}}
		f.Town.Arrive()
		for _, id := range []int{30, 31, 32} {
			f.Town.announceMission(id)
		}
		s := f.TownScreen().(*townScreen)
		s.Choose(3)
		a := s.worldMap.assets
		a.available, a.cross, a.modes[0].party = crossSheet(2), crossSheet(3), crossSheet(4)
		a.markers[30], a.markerTried[30] = crossSheet(1)[0], true
		a.data.Missions[99] = 2
		a.markers[99], a.markerTried[99] = crossSheet(2)[1], true
		s.markWorldSelected(99)
		return f, s
	}
	_, s := first()
	record("chapter/open", s, ui.TownAction{})
	s.WorldMapHover(ui.WorldMapCardRect(0).Min.Add(image.Pt(2, 2)))
	record("chapter/hover", s, ui.TownAction{})
	s.WorldMapMove(1)
	record("chapter/move", s, ui.TownAction{})
	record("chapter/choose-before-tick", s, s.WorldMapChoose())
	record("chapter/tick", s, s.WorldMapTick())
	record("chapter/miss", s, s.WorldMapClick(image.Pt(2, 2)))
	record("chapter/arrival", s, s.WorldMapTick())
	_, s = first()
	record("chapter/click-disabled", s, s.WorldMapClick(ui.WorldMapCardRect(1).Min.Add(image.Pt(2, 2))))
	s.leaveWorldMap()
	record("chapter/leave", s, ui.TownAction{})
	for _, home := range []bool{true, false} {
		f := restoredWorldMapFront(t, home, true)
		s := f.TownScreen().(*townScreen)
		s.Choose(3)
		record(fmt.Sprintf("chapter/load-home-%t", home), s, ui.TownAction{})
	}
	f, s := victoryWorldMapFront(t)
	ms, mw, _ := victoryContinueDriver(t, 30)
	f.continuity(30, ms, mw.advanceNotice)(ui.NoticeVictory)
	record("chapter/victory", s, ui.TownAction{})
	record("chapter/return-tick", s, s.WorldMapTick())
	record("chapter/return-choose", s, s.WorldMapChoose())
	record("chapter/return-arrival", s, s.WorldMapTick())

	f, s = travelFront(t, travelCatalog)
	s.Choose(3)
	record("chapter/town-record", s, ui.TownAction{})
	s.chooseWorldRecord(len(s.WorldMapView().Missions) - 1)
	record("chapter/town-travel", s, ui.TownAction{})
	s.WorldMapTick()
	s.WorldMapChoose()
	record("chapter/town-arrival", s, s.WorldMapTick())

	for _, opening := range []string{"select", "route", "load"} {
		f, _, s := secondCampaignFixture(t, true)
		art := *s.mapArt()
		sample := secondMapState(t, secondMapSelect, image.Point{}).assets
		art.modes, art.cross, art.rules = sample.modes, sample.cross, sample.rules
		art.recordTextTried, art.recordText = true, "#town2\r\nKaarg\r\n#description2\r\nA town.\r\n"
		s.mapArt = func() *worldMapAssets { return &art }
		c := newSecondCampaign()
		c.current, c.available = partyPlace{}, []partyPlace{{2, 3}, {2, 2}, {1, 30}, {1, 999}}
		if opening != "load" {
			from := partyPlace{2, 3}
			if opening == "route" {
				from = partyPlace{2, 1}
			}
			c.openMap(from)
		}
		f.Town.second = c
		prefix := "destination/" + opening
		record(prefix+"/open", s, ui.TownAction{})
		s.WorldMapHover(image.Pt(370, 305))
		record(prefix+"/hover", s, ui.TownAction{})
		record(prefix+"/click", s, s.WorldMapClick(image.Pt(370, 305)))
		s.WorldMapMove(1)
		record(prefix+"/move", s, ui.TownAction{})
		record(prefix+"/choose", s, s.WorldMapChoose())
		record(prefix+"/tick", s, s.WorldMapTick())
		s.leaveWorldMap()
		record(prefix+"/leave", s, ui.TownAction{})
	}
	return out.String()
}

func TestWorldMapPageRecords(t *testing.T) {
	got := worldMapPageRecords(t)
	want, err := os.ReadFile("testdata/worldmap-page-records.txt")
	if err != nil {
		t.Fatalf("read records: %v\n%s", err, got)
	}
	if got != string(want) {
		t.Fatalf("world-map records changed\ngot:\n%swant:\n%s", got, want)
	}
}
