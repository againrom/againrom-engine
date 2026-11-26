package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// sessionLoot is the gold the map's loot adds to the purse.
const sessionLoot = 250

// sessionEndCases are the session cases over maps wide enough to hold the
// start's fallback drop cell, so the hero stands on the map.
func sessionEndCases() []sessionCase {
	cases := sessionCases()
	cases[0].mapB = func(t *testing.T) []byte {
		b := synth.ALM(synth.ALMOptions{Width: 128, Height: 128, Name: "Arena"})
		binary.LittleEndian.PutUint32(b[alm.FileHeaderSize+alm.RecordHeaderSize+alm.MetaWord70:], 2)
		return b
	}
	cases[1].mapB = func(t *testing.T) []byte { return secondMissionMapOf(128, 128, secondMissionContent{}) }
	for i := range cases {
		front := cases[i].front
		cases[i].front = func(t *testing.T) *FrontEnd {
			f := front(t)
			f.Font = resolved(missionFont(), nil)
			armed := eqDefsTable(t)
			if f.Table == nil {
				f.Table = armed
				return f
			}
			f.Table.Shapes, f.Table.Materials, f.Table.Weapons = armed.Shapes, armed.Materials, armed.Weapons
			f.Table.Shields, f.Table.Armors = armed.Shields, armed.Armors
			return f
		}
	}
	return cases
}

// sessionGains are what the first map left: the hero's experience, the purse,
// the carried and worn items, and the file's bytes after the end.
type sessionGains struct {
	xp    [data.SkillSlots]int32
	purse uint32
	items []sim.ItemInstance
	worn  [sim.EquipSlots]sim.ItemInstance
	file  []byte
}

// sessionGain has the hero wear a sword through the inventory's equip route,
// kill a monster that pays experience, and take the map's loot into the
// purse. It answers the hero's experience, which the kill raised.
func sessionGain(t *testing.T, f *FrontEnd) [data.SkillSlots]int32 {
	t.Helper()
	w, mw := f.live.world, f.live
	hero := secondMissionActor(t, w)
	if !w.CheatAddItem(hero.ID, sim.ItemStack{Code: eqSwordCode, Count: 1}) {
		t.Fatal("the sword was refused")
	}
	carried, _ := w.CarriedItems(hero.ID)
	mw.enqueueEquip(len(carried) - 1)
	mw.tick()
	if armed := secondMissionActor(t, w); armed.DamageBase == 0 {
		t.Fatal("the hero wears no weapon")
	}
	victim, err := w.CheatSummon(sim.Entity{Owner: 3, HP: 1, MaxHP: 1, DyingTime: 8, XPValue: 4},
		nil, [sim.EquipSlots]sim.ItemInstance{}, hero.X+1, hero.Y)
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 3000 && secondMissionActor(t, w).SkillXP == hero.SkillXP; tick++ {
		mw.strike(uint32(hero.ID), uint32(victim))
		mw.tick()
	}
	xp := secondMissionActor(t, w).SkillXP
	if xp == hero.SkillXP {
		t.Fatalf("the kill paid no experience, still %v", xp)
	}
	if !w.CheatAddGold(sim.SelfSlot, sessionLoot) {
		t.Fatal("the loot was refused")
	}
	return xp
}

// sessionHolding is the hero's carried and worn items in the live world.
func sessionHolding(t *testing.T, f *FrontEnd) ([]sim.ItemInstance, [sim.EquipSlots]sim.ItemInstance) {
	t.Helper()
	w, hero := f.live.world, secondMissionActor(t, f.live.world).ID
	items, _ := w.CarriedItems(hero)
	worn, _ := w.EquippedItems(hero)
	for i := range items {
		items[i].ObjectID = 0
	}
	for i := range worn {
		worn[i].ObjectID = 0
	}
	return items, worn
}

// sessionEnd ends the running session's map by route: "leave" is the game
// menu's exit, "win" the success notice's Victory over a won mission.
func sessionEnd(t *testing.T, f *FrontEnd, app *ui.App, route string) {
	t.Helper()
	if route == "win" {
		f.live.recordMissionOutcome(sim.OutcomeWon, lossCause{})
		f.live.showOutcome()
		if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
			t.Fatal("the won map opened no success notice")
		}
		app.Layout(1024, 768)
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate("victory"); err != nil {
			t.Fatal(err)
		}
	} else {
		sessionKey(t, app, "escape")
		if err := app.HeadlessGameMenuAction("end"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("exit-main"); err != nil {
			t.Fatal(err)
		}
	}
	if !app.CharacterList() || app.HeadlessListSelection() != 0 {
		t.Fatalf("the %s left screen %v list %v selection %d, want the character list on the character's row: %s",
			route, app.Screen(), app.CharacterList(), app.HeadlessListSelection(), app.HeadlessMessage())
	}
}

// sessionFirstMap enters c's map with the stored character, gains by a kill
// and the loot, and ends the map by route.
func sessionFirstMap(t *testing.T, c sessionCase, route string) (*FrontEnd, *ui.App, string, sessionGains) {
	t.Helper()
	f, app, stored := sessionApp(t, c)
	sessionKey(t, app, "enter", "enter")
	if app.Screen() != ui.ScreenMap || f.live == nil {
		t.Fatalf("no map entered: %s", app.HeadlessMessage())
	}
	var g sessionGains
	g.xp = sessionGain(t, f)
	if g.xp == c.xp {
		t.Fatalf("experience %v is the stored %v", g.xp, c.xp)
	}
	g.purse = f.live.world.Purse(sim.SelfSlot)
	g.items, g.worn = sessionHolding(t, f)
	sessionEnd(t, f, app, route)
	b, err := os.ReadFile(filepath.Join(stored, c.file))
	if err != nil {
		t.Fatal(err)
	}
	g.file = b
	return f, app, stored, g
}

// sessionWantFile checks the rewritten file holds what its game's file
// holds of the gains, and keeps every other field of the stored document.
func sessionWantFile(t *testing.T, c sessionCase, g sessionGains) {
	t.Helper()
	stored := c.record(t)
	if c.name == base.ROM1EN {
		got, err := alch.Read(g.file)
		want, _ := alch.Read(stored)
		if err != nil || got.Stats != [4]uint32{31, 27, 22, 26} || !bytes.Equal(g.file, stored) {
			t.Fatalf("first game file %+v, %v; want the stored document with its statistics %+v", got, err, want)
		}
		return
	}
	got, err := a2c.Read(g.file)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := a2c.Read(stored)
	want.E.Money = g.purse
	for i := range want.E.SkillXP {
		want.E.SkillXP[i] = uint32(g.xp[data.SkillBlade+int32(i)])
	}
	if got.E != want.E || string(got.A.Name) != "Stored" || !reflect.DeepEqual(got.A, want.A) ||
		!bytes.Equal(got.Kills, want.Kills) || !bytes.Equal(got.Hotkeys, want.Hotkeys) || got.Worn != nil || got.Bag != nil {
		t.Fatalf("second game file %+v, want %+v", got, want)
	}
}

// sessionWantParty checks the running map's party holds the gains.
func sessionWantParty(t *testing.T, f *FrontEnd, g sessionGains, items bool) {
	t.Helper()
	h := secondMissionActor(t, f.live.world)
	if h.SkillXP != g.xp {
		t.Fatalf("experience %v, want the gains %v", h.SkillXP, g.xp)
	}
	for s := data.SkillBlade; s < data.SkillSlots; s++ {
		if got := f.LiveParty()[0].Hero.Skill[s]; got < data.SkillLevelFor(g.xp[s]) {
			t.Fatalf("skill %d level %d, want at least %d for %d experience", s, got, data.SkillLevelFor(g.xp[s]), g.xp[s])
		}
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != g.purse {
		t.Fatalf("purse %d, want the gains %d", got, g.purse)
	}
	if !items {
		return
	}
	carried, worn := sessionHolding(t, f)
	if !reflect.DeepEqual(carried, g.items) || !reflect.DeepEqual(worn, g.worn) {
		t.Fatalf("items %+v worn %+v, want %+v and %+v", carried, worn, g.items, g.worn)
	}
}

// A won map and a leave through the game menu each end the session on the
// character list with the character's row selected; the file is rewritten
// with what it holds of the gains, the next map of the same process starts
// with the gains and the items, and an unchanged character writes the same
// bytes again.
func TestMultiplayerMapEndWritesTheGainsForTheNextMap(t *testing.T) {
	for _, c := range sessionEndCases() {
		for _, route := range []string{"leave", "win"} {
			t.Run(c.name+"/"+route, func(t *testing.T) {
				f, app, stored, g := sessionFirstMap(t, c, route)
				sessionWantFile(t, c, g)
				sessionKey(t, app, "enter", "enter")
				if app.Screen() != ui.ScreenMap || f.live == nil {
					t.Fatalf("the second map was not entered: %s", app.HeadlessMessage())
				}
				sessionWantParty(t, f, g, true)
				if _, _, err := f.Snapshot(true); err == nil {
					t.Fatal("SAVE was offered on the session's map")
				}
				sessionEnd(t, f, app, "leave")
				again, err := os.ReadFile(filepath.Join(stored, c.file))
				if err != nil || !bytes.Equal(again, g.file) {
					t.Fatalf("a second write of the unchanged character changed the file: %v", err)
				}
			})
		}
	}
}

// A second game's rewritten file alone gives a new process's party the
// experience, levels and purse it holds.
func TestRewrittenSecondCharacterFileGivesTheGains(t *testing.T) {
	c := sessionEndCases()[1]
	_, _, _, g := sessionFirstMap(t, c, "leave")
	c.record = func(*testing.T) []byte { return g.file }
	f, app, _ := sessionApp(t, c)
	sessionKey(t, app, "enter", "enter")
	if f.live == nil {
		t.Fatalf("no map entered: %s", app.HeadlessMessage())
	}
	sessionWantParty(t, f, g, false)
}

// The same route twice gives equal files and equal second-map world hashes.
func TestMultiplayerMapEndIsDeterministic(t *testing.T) {
	for _, c := range sessionEndCases() {
		t.Run(c.name, func(t *testing.T) {
			var files [2][]byte
			var hashes [2]uint64
			for i := range hashes {
				f, app, _, g := sessionFirstMap(t, c, "leave")
				sessionKey(t, app, "enter", "enter")
				if f.live == nil {
					t.Fatalf("run %d entered no second map: %s", i, app.HeadlessMessage())
				}
				files[i], hashes[i] = g.file, f.live.world.Hash()
			}
			if !bytes.Equal(files[0], files[1]) || hashes[0] != hashes[1] {
				t.Fatalf("runs differ: files equal %v, hashes %x and %x", bytes.Equal(files[0], files[1]), hashes[0], hashes[1])
			}
		})
	}
}
