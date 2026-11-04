package sim

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// gameProfileRow is one fixture World's answers to which game it runs, whether
// it holds no Group registry, and each per-feature choice, with its hash when
// built and after stepping.
type gameProfileRow struct {
	second, native bool
	route, aim     string
	spell          int
	built, stepped uint64
}

func (r gameProfileRow) String() string {
	return fmt.Sprintf("{second: %v, native: %v, route: %q, aim: %q, spell: %d, built: %#x, stepped: %#x}",
		r.second, r.native, r.route, r.aim, r.spell, r.built, r.stepped)
}

// gameProfileRows are the answers each build path gives a fixture World of
// each game, recorded before the game was a profile chosen once.
var gameProfileRows = map[string]gameProfileRow{
	"first decoded":                 {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x65b50eb30e0d455b, stepped: 0xd593e83892201017},
	"first loaded":                  {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x9d503846558ecf57, stepped: 0x55553abfbbf67745},
	"first loaded decoded":          {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x9d503846558ecf57, stepped: 0x55553abfbbf67745},
	"first loaded script copy":      {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x9d503846558ecf57, stepped: 0x55553abfbbf67745},
	"first script copy":             {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x65b50eb30e0d455b, stepped: 0xd593e83892201017},
	"first script copy other game":  {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0xb2e30ca9d6830599, stepped: 0xf2c5d89bc50bfef7},
	"first start":                   {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x65b50eb30e0d455b, stepped: 0xd593e83892201017},
	"second decoded":                {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0xb2e30ca9d6830599, stepped: 0xf2c5d89bc50bfef7},
	"second loaded":                 {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0x687df14b7092c10b, stepped: 0xfca3dff7a64b58a},
	"second loaded decoded":         {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0x687df14b7092c10b, stepped: 0xfca3dff7a64b58a},
	"second loaded script copy":     {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0x687df14b7092c10b, stepped: 0xfca3dff7a64b58a},
	"second script copy":            {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0xb2e30ca9d6830599, stepped: 0xf2c5d89bc50bfef7},
	"second script copy other game": {second: false, native: false, route: "first", aim: "first", spell: 0, built: 0x65b50eb30e0d455b, stepped: 0xd593e83892201017},
	"second start":                  {second: true, native: false, route: "second", aim: "second", spell: 1, built: 0xb2e30ca9d6830599, stepped: 0xf2c5d89bc50bfef7},
}

// gameProfileFixture is a hero and a hostile fighter two cells apart, each in
// its own Group, at mission start.
func gameProfileFixture(t *testing.T, second bool) *World {
	t.Helper()
	hero := engFighter(guardRouteHero, SelfSlot, 20, 20)
	foe := engFighter(guardRouteFoe, guardRouteSlot, 22, 20)
	foe.Group = 1
	rel := engRel(t, [3]uint32{guardRouteSlot, SelfSlot, 1}, [3]uint32{SelfSlot, guardRouteSlot, 1})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero, foe}, gameProfileScript(t, second), rel)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func gameProfileScript(t *testing.T, second bool) *Script {
	t.Helper()
	if second {
		return secondGameScript(t, nil, nil, nil)
	}
	s, err := NewScript(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// gameProfileLoaded is the fixture after the Group import an original LOAD
// runs: the hostile's Group at the guard route, the hero's at Stand Ground.
func gameProfileLoaded(t *testing.T, second bool) *World {
	t.Helper()
	w := gameProfileFixture(t, second)
	guard := SavedGroup{ID: 1, Selector: 1, Owner: SavedGroupReference{Class: 1, Owner: guardRouteSlot},
		Members: []SavedGroupMember{{Archive: 2, Entity: guardRouteFoe, Bound: true}}}
	guard.AI[0x20], guard.AI[0x2d], guard.AI[0x38], guard.AI[0x45] = orderGuard, 10, 10, 1
	if second {
		guard.AI[0x20] = guardRouteB
	}
	stand := SavedGroup{ID: 2, Owner: SavedGroupReference{Class: 1, Owner: SelfSlot},
		Members: []SavedGroupMember{{Archive: 1, Entity: guardRouteHero, Bound: true}}}
	stand.AI[0x20], stand.AI[0x45] = orderStandGround, 1
	var orders []SavedActorOrder
	if !second {
		orders = []SavedActorOrder{constructedOrder(laEnt(t, w, guardRouteFoe), uint32(actorStateGuard))}
	}
	if err := w.ImportOriginalGroups([]SavedGroup{guard, stand}, orders); err != nil {
		t.Fatal(err)
	}
	return w
}

func gameProfileDecoded(t *testing.T, w *World) *World {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var out World
	if err := out.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	return &out
}

func gameProfileCopy(t *testing.T, w *World, second bool) *World {
	t.Helper()
	out, err := NewControlledScriptWorld(w, gameProfileScript(t, second))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// gameProfileWorlds are the fixture Worlds of both games on every build path:
// mission start, decoded, after an original LOAD's Group import, decoded
// after it, and the controlled script copy in the same and the other game.
func gameProfileWorlds(t *testing.T) map[string]*World {
	t.Helper()
	out := map[string]*World{}
	for _, second := range []bool{false, true} {
		game := "first"
		if second {
			game = "second"
		}
		out[game+" start"] = gameProfileFixture(t, second)
		out[game+" decoded"] = gameProfileDecoded(t, gameProfileFixture(t, second))
		out[game+" loaded"] = gameProfileLoaded(t, second)
		out[game+" loaded decoded"] = gameProfileDecoded(t, gameProfileLoaded(t, second))
		out[game+" script copy"] = gameProfileCopy(t, gameProfileFixture(t, second), second)
		out[game+" script copy other game"] = gameProfileCopy(t, gameProfileFixture(t, second), !second)
		out[game+" loaded script copy"] = gameProfileCopy(t, gameProfileLoaded(t, second), second)
	}
	return out
}

// playsSecondGame reports whether w holds the second game's whole profile.
func playsSecondGame(w *World) bool {
	return reflect.DeepEqual(w.game, newGameProfile(ScriptROM2))
}

func gameProfileAnswers(w *World) gameProfileRow {
	route, aim := "first", "first"
	if w.game.groupRoute == secondGameGroupRoute {
		route = "second"
	}
	if reflect.DeepEqual(w.game.creatureAim, secondGameCreatureAim) {
		aim = "second"
	}
	if !playsSecondGame(w) && !reflect.DeepEqual(w.game, gameProfile{}) {
		route, aim = "mixed", "mixed"
	}
	return gameProfileRow{second: playsSecondGame(w), native: w.groupDialectNative(), route: route, aim: aim, spell: int(w.SpellGame()), built: w.Hash()}
}

// Every build path of every fixture World answers the game, the Group
// registry, the group route, the creature aim and the spell game as recorded,
// and its hash when built and after 300 steps is the recorded one.
func TestEveryBuildPathAnswersTheRecordedGameProfile(t *testing.T) {
	worlds := gameProfileWorlds(t)
	names := make([]string, 0, len(worlds))
	for name := range worlds {
		names = append(names, name)
	}
	sort.Strings(names)
	var got []string
	for _, name := range names {
		w := worlds[name]
		row := gameProfileAnswers(w)
		for range 300 {
			Step(w, nil)
		}
		row.stepped = w.Hash()
		if want, ok := gameProfileRows[name]; !ok || want != row {
			t.Errorf("%s: %v, want %v", name, row, want)
		}
		got = append(got, fmt.Sprintf("\t%q: %v,", name, row))
	}
	if len(gameProfileRows) != len(worlds) {
		t.Errorf("%d recorded rows, %d fixture Worlds", len(gameProfileRows), len(worlds))
	}
	if t.Failed() {
		t.Logf("answers:\n%s", strings.Join(got, "\n"))
	}
}
