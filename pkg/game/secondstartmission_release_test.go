package game

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func secondStartMissionWitness(t *testing.T) (*FrontEnd, *ui.App, *routeWitness) {
	t.Helper()
	f := secondGameFront(t)
	f.Options = OptionsStore{}
	app := f.App("start mission")
	app.Layout(640, 480)
	f.TownAnimationNow = app.HeadlessSceneNow
	s, ok := f.TownScreen().(*secondCampaignScreen)
	if !ok {
		t.Fatal("the second game's town is not the second campaign's screen")
	}
	app.SetTown(s)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	return f, app, &routeWitness{t: t, f: f, app: app, town: s, place: s, stage: "start mission", steps: map[string]int{}}
}

func secondStartMissionTalk(t *testing.T, app *ui.App, w *routeWitness, talk string) {
	t.Helper()
	for _, target := range []string{"TAVERN", talk} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
	for pages := 0; app.HeadlessActivate("notice") == nil; pages++ {
		if pages >= 64 {
			t.Fatalf("%s: more than 64 pages", talk)
		}
	}
	if err := app.HeadlessActivate("EXIT"); err != nil || w.screen() != "town/square" {
		t.Fatalf("EXIT after %s: %v, %s", talk, err, w.screen())
	}
}

func secondStartMissionClick(t *testing.T, w *routeWitness) time.Time {
	t.Helper()
	r := w.sceneControl("GATES")
	w.pointer("hover", r.at)
	w.pointer("press", r.at)
	at := w.app.HeadlessSceneNow()
	w.pointer("release", r.at)
	return at
}

func TestReleaseSecondGameStartMissionShowsMapThenMissionThenDialogue(t *testing.T) {
	f, app, w := secondStartMissionWitness(t)
	secondStartMissionTalk(t, app, w, "TALK 517")
	click := secondStartMissionClick(t, w)
	if w.screen() != "town/worldmap" {
		t.Fatalf("START MISSION shows %s, want the world map", w.screen())
	}
	var (
		order                         []string
		paintAt                       []int
		missionAt, dialogueAt         = -1, -1
		routeLen, frame, dialogueTick int
	)
	for step := 0; step < 1000 && dialogueAt < 0; step++ {
		at := int(app.HeadlessSceneNow().Sub(click) / time.Millisecond)
		w.run(1)
		shown := w.screen()
		switch {
		case shown == "town/worldmap":
			s := w.town.worldMap
			routeLen = max(routeLen, len(s.route))
			if s.frame != frame {
				frame = s.frame
				paintAt = append(paintAt, at)
			}
		case shown == "map/notice":
			_, kind, _ := f.LiveNotice()
			if kind != ui.NoticeDialogue {
				t.Fatalf("the mission opened notice kind %v, want the dialogue", kind)
			}
			dialogueAt, dialogueTick = at, int(f.live.world.Tick())
			shown = "dialogue"
		case app.Screen() == ui.ScreenMap:
			if missionAt < 0 {
				missionAt = at
				frame = w.town.worldMap.frame
				paintAt = append(paintAt, at)
			}
			shown = "mission"
		default:
			t.Fatalf("at %d ms the transition shows %s", at, shown)
		}
		if n := len(order); n == 0 || order[n-1] != shown {
			order = append(order, shown)
		}
	}
	paints := frame
	t.Logf("App clock step 20 ms, map paint gate 100 ms, mission cadence rung %d of %d us per tick",
		terrain.DefaultCadenceRung, terrain.CadencePeriod(terrain.DefaultCadenceRung))
	t.Logf("route L=%d; %d paints at %v ms; mission at %d ms; dialogue at %d ms, World tick %d; order %v",
		routeLen, paints, paintAt, missionAt, dialogueAt, dialogueTick, order)
	if strings.Join(order, ",") != "town/worldmap,mission,dialogue" {
		t.Fatalf("order %v, want the map, the mission without dialogue, the dialogue", order)
	}
	if want := max(15, (routeLen-8)/8+1); routeLen == 0 || paints != want || len(paintAt) != paints {
		t.Fatalf("route %d points ended on paint %d (%d timed), want %d", routeLen, paints, len(paintAt), want)
	}
	for i := 1; i < len(paintAt); i++ {
		if paintAt[i]-paintAt[i-1] != 100 {
			t.Fatalf("paint %d came %d ms after paint %d, want 100", i+1, paintAt[i]-paintAt[i-1], i)
		}
	}
	if missionAt != paintAt[len(paintAt)-1] || missionAt > 2600 || dialogueAt <= 2600 || dialogueAt > 11000 {
		t.Fatalf("mission at %d ms, dialogue at %d ms: want the mission by 2600 ms on the last paint and the dialogue after 2600 ms and by 11000 ms",
			missionAt, dialogueAt)
	}
}

func TestReleaseSecondGameStartMissionGateRefusesBeforeTheMage(t *testing.T) {
	for _, talk := range []string{"TALK 207", "TALK 2108"} {
		t.Run(strings.ReplaceAll(talk, " ", ""), func(t *testing.T) {
			f, app, w := secondStartMissionWitness(t)
			secondStartMissionTalk(t, app, w, talk)
			c := f.Town.second
			if c.bank[769] != 0 {
				t.Fatalf("%s set bank 769", talk)
			}
			want, err := readSecondTownTalk(w.town.install, secondGateText)
			if err != nil || want == nil {
				t.Fatalf("no gate guard text: %v", err)
			}
			secondStartMissionClick(t, w)
			for step := range 100 {
				if s := w.screen(); s != "town/square/dialogue" {
					t.Fatalf("step %d after the click shows %s, want the guard's box over the square", step, s)
				}
				w.run(1)
			}
			if !bytes.Equal(c.talk.payload, want) {
				t.Fatal("the box is not the gate guard's text")
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatalf("Ok: %v", err)
			}
			w.run(5)
			if w.screen() != "town/square" || c.current != (partyPlace{kind: placeTown, id: 1}) || c.mapOpen != nil {
				t.Fatalf("Ok shows %s at %+v, map %v; want town 1's square", w.screen(), c.current, c.mapOpen)
			}
		})
	}
}
