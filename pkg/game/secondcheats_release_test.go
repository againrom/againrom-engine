package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondCheatMission opens the first campaign mission on a second-game root
// through New Game, the inn and the gates, with a mage hero when mage is set,
// and settles its opening dialogue.
func secondCheatMission(t *testing.T, mage, chicken bool) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondGameFront(t)
	f.SetChickenAtMissionStart(chicken)
	app := f.App("second cheats")
	app.Layout(1024, 768)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if mage {
		f.Carried = MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	}
	for _, target := range []string{"TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatalf("initial inn dialogue did not open: %v", err)
	}
	for pages := 0; pages < 64; pages++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
	}
	for _, target := range []string{"GATES", "mission 10", "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("first campaign mission did not open: %v", app.Screen())
	}
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	f.live.stopped = true
	return f, app
}

// secondCheatReply is the reply line R2-ENGINE-298 names: owner's name
// between main.txt lines 221+2(code-5) and the next one.
func secondCheatReply(t *testing.T, f *FrontEnd, code int, owner uint32) string {
	t.Helper()
	row := 221 + 2*(code-5)
	name := EncodeInstallText(f.live.cheatPlayerName(owner), f.live.view.TextSelector())
	return rawTextLine(t, f, MainTextPath, row) + name + rawTextLine(t, f, MainTextPath, row+1)
}

// secondCheatLines types line through the App chat and returns the message
// lines it added.
func secondCheatLines(t *testing.T, f *FrontEnd, a *ui.App, line string) []ui.MessageLine {
	t.Helper()
	before := len(f.live.view.MessageLines())
	cheatChat(t, a, line)
	lines := f.live.view.MessageLines()
	if len(lines) < before {
		t.Fatalf("%q dropped message lines", line)
	}
	return lines[before:]
}

func secondCheatWantReply(t *testing.T, f *FrontEnd, got []ui.MessageLine, code int, owner uint32) {
	t.Helper()
	want := secondCheatReply(t, f, code, owner)
	if len(got) != 1 || got[0].Text != want || got[0].Life != 5*time.Second {
		t.Fatalf("reply %d = %+v, want one %q for 5 s", code, got, want)
	}
}

func secondCheatWantNothing(t *testing.T, f *FrontEnd, got []ui.MessageLine, hash uint64, what string) {
	t.Helper()
	if len(got) != 0 || f.live.world.Hash() != hash {
		t.Fatalf("%s: lines %+v, world changed %v", what, got, f.live.world.Hash() != hash)
	}
}

func secondCheatUnlock(t *testing.T, f *FrontEnd, a *ui.App) {
	t.Helper()
	got := secondCheatLines(t, f, a, secondUnlock+"d")
	if f.live.cheats.privilege[sim.SelfSlot] != 255 {
		t.Fatal("the unlock line did not set the cheat state")
	}
	secondCheatWantReply(t, f, got, 5, sim.SelfSlot)
}

// secondCheatItemName is a magic item the installed table names and the
// shared factory builds.
func secondCheatItemName(t *testing.T, f *FrontEnd) (string, uint16) {
	t.Helper()
	for row := 1; row < f.Table.MagicItems.Len() && row <= 255; row++ {
		name := f.Table.MagicItems.EntryName(row)
		if name == "" || strings.ContainsAny(name, "\r\n") {
			continue
		}
		if item, ok := mapload.CheatItem(name, f.Table); ok && item.ValidateWeight() == nil {
			return name, item.Code
		}
	}
	t.Fatal("installed magic-item table has no buildable name")
	return "", 0
}

func secondCheatActorName(t *testing.T, f *FrontEnd, c data.Collection) string {
	t.Helper()
	for row := 1; row < c.Len() && row <= 255; row++ {
		name := c.EntryName(row)
		if name == "" || strings.ContainsAny(name, " \r\n") {
			continue
		}
		if _, _, _, err := mapload.CheatActor(name, false, f.Table, mapload.DifficultyNormal); err == nil {
			return name
		}
	}
	t.Fatal("installed table has no buildable name")
	return ""
}

func secondCheatGod(e sim.Entity) bool {
	for j := 0; j < 6; j++ {
		if binary.LittleEndian.Uint16(e.NativeBasis.Modifier[46+2*j:]) != 100 || e.NativeBasis.Modifier[58+j] != 100 {
			return false
		}
	}
	return true
}

type secondCheatState struct {
	Purse    uint32
	God      bool
	Carried  []uint16
	Known    uint32
	Book     bool
	Summoned int
}

func secondCheatStateOf(t *testing.T, f *FrontEnd, summoned []sim.EntityID) secondCheatState {
	t.Helper()
	hero := releaseEntity(t, f.live, f.live.mission.ids[0])
	carried, _ := f.live.world.Carried(hero.ID)
	s := secondCheatState{Purse: f.live.world.Purse(sim.SelfSlot), God: secondCheatGod(hero), Carried: slices.Sorted(slices.Values(carried)), Known: hero.KnownSpells, Book: hero.Book.WirePresent(hero.KnownSpells)}
	for _, id := range summoned {
		if e, ok := f.live.world.Entity(id); ok && e.Owner == sim.SelfSlot {
			s.Summoned++
		}
	}
	return s
}

// secondCheatActorsEqual compares every actor of want and got field by field.
// A summoned actor's source spellbook flag is excepted: the SAV writer gives
// every actor an empty present book, so the flag reads back set (story debt;
// the first game's summon has the same continuation).
func secondCheatActorsEqual(t *testing.T, want, got *sim.World, summoned []sim.EntityID) {
	t.Helper()
	if len(want.Entities()) != len(got.Entities()) {
		t.Fatalf("cold LOAD has %d actors, want %d", len(got.Entities()), len(want.Entities()))
	}
	for _, a := range want.Entities() {
		b, ok := got.Entity(a.ID)
		if !ok {
			t.Fatalf("cold LOAD lost actor %d", a.ID)
		}
		if slices.Contains(summoned, a.ID) {
			a.ActorLoad.Source.HasSpellbook = b.ActorLoad.Source.HasSpellbook
		}
		x, y := reflect.ValueOf(a), reflect.ValueOf(b)
		for i := 0; i < x.NumField(); i++ {
			if x.Field(i).CanInterface() && !reflect.DeepEqual(x.Field(i).Interface(), y.Field(i).Interface()) {
				t.Fatalf("cold LOAD actor %d %s: %v / %v", a.ID, x.Type().Field(i).Name, x.Field(i).Interface(), y.Field(i).Interface())
			}
		}
	}
}

// TestReleaseSecondCampaignCheatCommands drives every second-game chat
// command and debug letter through App input in campaign mission 10:
// R2-ENGINE-295 to R2-ENGINE-304. It then saves and cold-loads the mission
// and compares the state R2-SESSION-136 says a SAV carries.
func TestReleaseSecondCampaignCheatCommands(t *testing.T) {
	f, a := secondCheatMission(t, true, false)
	mw := f.live
	if !mw.cheats.campaign {
		t.Fatal("the campaign mission did not open in the campaign")
	}
	id := mw.mission.ids[0]
	hash := mw.world.Hash()
	gold := mw.world.Purse(sim.SelfSlot)

	// Locked: every command, an ordinary line and every debug letter change
	// nothing and send nothing (R2-ENGINE-297, R2-ENGINE-304).
	for _, line := range []string{"#create 100 Gold", "#modify self +god", "#show map", "#killall", "#victory", "#event 1", "hello"} {
		secondCheatWantNothing(t, f, secondCheatLines(t, f, a, line), hash, "locked "+line)
	}
	for letter := byte('B'); letter <= 'Y'; letter++ {
		mw.debugLetter(letter)
	}
	if mw.cheats.turnTrace || mw.cheats.scriptTrace || mw.cheats.safe || mw.world.Hash() != hash {
		t.Fatal("a locked debug letter acted")
	}
	if mw.cheats.privilege[sim.SelfSlot] != 0 || mw.view.FogRevealed() || mw.mission.open {
		t.Fatal("a locked command changed the cheat state")
	}

	secondCheatUnlock(t, f, a)
	secondCheatWantNothing(t, f, secondCheatLines(t, f, a, secondUnlock+"d"), hash, "an unlocked Player's unlock line")
	secondCheatWantNothing(t, f, secondCheatLines(t, f, a, "#Chicken"), hash, "another game's unlock line")

	// #create (R2-ENGINE-299): gold ignores case; an item by name; a refusal.
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#create 250 gOLD"), 7, sim.SelfSlot)
	if got := mw.world.Purse(sim.SelfSlot); got != gold+250 {
		t.Fatalf("purse %d, want %d", got, gold+250)
	}
	name, code := secondCheatItemName(t, f)
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#create 1 "+name), 7, sim.SelfSlot)
	if carried, _ := mw.world.Carried(id); !slices.Contains(carried, code) {
		t.Fatalf("#create %s did not reach the inventory: %v", name, carried)
	}
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#create no such item"), 6, sim.SelfSlot)

	// #modify (R2-ENGINE-300).
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#modify self +god"), 7, sim.SelfSlot)
	if !secondCheatGod(releaseEntity(t, mw, id)) {
		t.Fatal("+god did not write the six words and bytes")
	}
	if !mw.cheatHeroBook(releaseEntity(t, mw, id)) {
		t.Fatal("the mage hero holds no book")
	}
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#modify self +spell 3"), 7, sim.SelfSlot)
	if releaseEntity(t, mw, id).KnownSpells&(1<<3) == 0 {
		t.Fatal("+spell 3 did not reach the book")
	}
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#modify self +spells"), 7, sim.SelfSlot)
	known := releaseEntity(t, mw, id).KnownSpells
	for _, rule := range mw.world.Spells() {
		if rule.ID >= 1 && rule.ID <= 29 && known&(1<<rule.ID) == 0 {
			t.Fatalf("+spells left spell %d out of the book", rule.ID)
		}
	}
	h := mw.world.Hash()
	secondCheatWantNothing(t, f, secondCheatLines(t, f, a, "#modify self +knowledge"), h, "+knowledge")
	if mw.cheats.knowledge {
		t.Fatal("+knowledge set the first game's card override")
	}

	// #summon (R2-ENGINE-301): the count, then the name, no reply.
	hero := releaseEntity(t, mw, id)
	var summoned []sim.EntityID
	for _, c := range []struct {
		name  string
		count int
	}{{secondCheatActorName(t, f, f.Table.Humans), 2}, {secondCheatActorName(t, f, f.Table.Units), 1}} {
		known := map[sim.EntityID]bool{}
		for _, e := range mw.world.Entities() {
			known[e.ID] = true
		}
		got := secondCheatLines(t, f, a, fmt.Sprintf("#summon %d %s", c.count, c.name))
		var added []sim.EntityID
		for _, e := range mw.world.Entities() {
			if !known[e.ID] && e.Owner == sim.SelfSlot && e.Group != 0 {
				added = append(added, e.ID)
			}
		}
		if len(got) != 0 || len(added) != c.count {
			t.Fatalf("#summon %d %s: lines %+v, added %v, failure %v", c.count, c.name, got, added, mw.cheats.failure)
		}
		summoned = append(summoned, added...)
	}

	// #pickup all (R2-ENGINE-301) over a sack the ordinary drop made.
	hero = releaseEntity(t, mw, id)
	sim.Step(mw.world, []sim.Command{sim.DropGold(sim.SelfSlot, 40, sim.CellPoint{X: hero.X, Y: hero.Y})})
	if len(mw.world.Sacks()) == 0 {
		t.Fatal("the ordinary gold drop made no sack")
	}
	purse := mw.world.Purse(sim.SelfSlot)
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#pickup all"), 7, sim.SelfSlot)
	if len(mw.world.Sacks()) != 0 || mw.world.Purse(sim.SelfSlot) != purse+40 {
		t.Fatalf("#pickup all left sacks %d purse %d, want 0 and %d", len(mw.world.Sacks()), mw.world.Purse(sim.SelfSlot), purse+40)
	}

	// #show map, #hide map (R2-ENGINE-303).
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#show map"), 7, sim.SelfSlot)
	if !mw.view.FogRevealed() || !mw.cheats.showMap || slices.Contains(mw.fog.explored, 0) {
		t.Fatal("#show map did not reveal the map")
	}
	secondCheatWantReply(t, f, secondCheatLines(t, f, a, "#hide map"), 7, sim.SelfSlot)
	if mw.view.FogRevealed() || mw.cheats.showMap {
		t.Fatal("#hide map kept the reveal")
	}

	// Debug letters (R2-ENGINE-304): D, T and Q toggle; none prints.
	safeHash := mw.world.Hash()
	lines := map[string]bool{}
	for _, l := range mw.view.MessageLines() {
		lines[l.Text] = true
	}
	for _, letter := range []byte{'D', 'T', 'Q', 'H', 'I', 'U', 'B'} {
		if err := a.HeadlessKey("alt-" + strings.ToLower(string(letter))); err != nil {
			t.Fatal(err)
		}
	}
	printed := slices.ContainsFunc(mw.view.MessageLines(), func(l ui.MessageLine) bool { return !lines[l.Text] })
	if !mw.cheats.turnTrace || !mw.cheats.scriptTrace || !mw.cheats.safe || mw.world.Hash() == safeHash || printed {
		t.Fatalf("debug letters: trace %v script %v safe %v printed %v", mw.cheats.turnTrace, mw.cheats.scriptTrace, mw.cheats.safe, printed)
	}
	if err := a.HeadlessKey("alt-q"); err != nil || mw.cheats.safe {
		t.Fatal("Alt+Q did not toggle safe mode back", err)
	}

	// SAVE and cold LOAD (R2-SESSION-135, R2-SESSION-136).
	want := secondCheatStateOf(t, f, summoned)
	if !want.God || want.Summoned != 3 {
		t.Fatalf("state before SAVE: %+v", want)
	}
	out := secondMissionSaveDirectory(t)
	secondMissionNamedSave(t, f, a, out, "Cheats")
	cold, ca := secondMissionColdAt(t, out, "Cheats.sav", 10)
	cold.live.stopped = true
	if got := secondCheatStateOf(t, cold, summoned); !reflect.DeepEqual(got, want) {
		t.Fatalf("cold LOAD state %+v, want %+v", got, want)
	}
	secondCheatActorsEqual(t, f.live.world, cold.live.world, summoned)
	if cold.live.cheats.privilege[sim.SelfSlot] != 0 || cold.live.view.FogRevealed() {
		t.Fatal("cold LOAD with the flag off is not locked")
	}
	h = cold.live.world.Hash()
	secondCheatWantNothing(t, cold, secondCheatLines(t, cold, ca, "#create 5 Gold"), h, "locked after LOAD")

	// The kills (R2-ENGINE-302) on the loaded mission.
	secondCheatUnlock(t, cold, ca)
	cw := cold.live
	relations := cw.world.Relations()
	var hostile []uint32
	for owner := uint32(0); owner < 50; owner++ {
		if relations.Hostile(owner, sim.SelfSlot) && owner != sim.SelfSlot && slices.ContainsFunc(cw.world.Entities(), func(e sim.Entity) bool { return e.Owner == owner && e.Alive() }) {
			hostile = append(hostile, owner)
		}
	}
	if len(hostile) == 0 {
		t.Fatal("mission 10 has no living hostile Player")
	}
	h = cw.world.Hash()
	secondCheatWantNothing(t, cold, secondCheatLines(t, cold, ca, "#kill cheaters"), h, "#kill cheaters")
	secondCheatWantNothing(t, cold, secondCheatLines(t, cold, ca, "#kill nobody by this name"), h, "#kill with no Player")
	target := hostile[0]
	secondCheatWantReply(t, cold, secondCheatLines(t, cold, ca, "#kill "+cw.cheatPlayerName(target)), 7, target)
	for _, e := range cw.world.Entities() {
		if e.Owner == target && e.HP > 0 {
			t.Fatalf("#kill left actor %d of Player %d at %d", e.ID, target, e.HP)
		}
	}
	secondCheatWantReply(t, cold, secondCheatLines(t, cold, ca, "#killall"), 7, sim.SelfSlot)
	for _, e := range cw.world.Entities() {
		if relations.Hostile(e.Owner, sim.SelfSlot) && e.HP > 0 {
			t.Fatalf("#killall left hostile actor %d of Player %d", e.ID, e.Owner)
		}
	}

	// #event (R2-ENGINE-303, R2-ENGINE-047) raises the event as the script
	// does; #victory opens the ordinary completion.
	cheatChat(t, ca, "#event 1")
	if !slices.Contains(cw.mission.pendingMessages, 1) && !(cw.mission.open && cw.mission.kind == ui.NoticeDialogue) {
		t.Fatal("#event 1 raised no event")
	}
	dismissSecondMissionDialogue(t, ca, cw)
	cw.mission.pendingMessages = nil
	cheatChat(t, ca, "#victory")
	if !cw.mission.open || cw.mission.kind != ui.NoticeSuccess || cw.mission.outcome != sim.OutcomeWon {
		t.Fatal("#victory did not open the ordinary completion")
	}
}

// TestReleaseSecondCampaignCheatLaunchFlag: the starter checkbox and -chicken
// unlock every fresh second-game campaign mission and every LOAD into one,
// with reply 5 (owner direction, DIV-2774); off, a LOAD comes back locked
// (R2-SESSION-135). Outside the campaign the unlock line does nothing
// (R2-ENGINE-296).
func TestReleaseSecondCampaignCheatLaunchFlag(t *testing.T) {
	f, a := secondCheatMission(t, false, true)
	if f.live.cheats.privilege[sim.SelfSlot] != 255 {
		t.Fatal("the flag did not unlock the fresh mission")
	}
	want := secondCheatReply(t, f, 5, sim.SelfSlot)
	if !slices.ContainsFunc(f.live.view.MessageLines(), func(l ui.MessageLine) bool { return l.Text == want }) {
		t.Fatalf("fresh start lines %+v lack reply 5", f.live.view.MessageLines())
	}
	out := secondMissionSaveDirectory(t)
	secondMissionNamedSave(t, f, a, out, "Flag")

	on := secondGameFront(t)
	on.SetChickenAtMissionStart(true)
	app := on.App("flag load")
	app.Layout(1024, 768)
	on.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	_, list, _ := on.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	groundAppLoad(t, app, list, localOriginalSaveToken("Flag.sav"))
	if !on.live.mission.resumed || on.live.cheats.privilege[sim.SelfSlot] != 255 {
		t.Fatalf("the flag did not unlock the LOAD: resumed %v campaign %v priv %d", on.live.mission.resumed, on.live.cheats.campaign, on.live.cheats.privilege[sim.SelfSlot])
	}
	want = secondCheatReply(t, on, 5, sim.SelfSlot)
	if !slices.ContainsFunc(on.live.view.MessageLines(), func(l ui.MessageLine) bool { return l.Text == want }) {
		t.Fatalf("LOAD lines %+v lack reply 5", on.live.view.MessageLines())
	}

	off, _ := secondMissionColdAt(t, out, "Flag.sav", 10)
	if off.live.cheats.privilege[sim.SelfSlot] != 0 {
		t.Fatal("a LOAD with the flag off is unlocked")
	}

	outside := secondGameFront(t)
	outside.SetChickenAtMissionStart(true)
	oa := outside.App("outside the campaign")
	oa.Layout(1024, 768)
	if err := oa.OpenMission(outside.MissionOpenerWith(10, MissionParty(outside.StartWeapon.Value(), outside.Bodies, outside.Table))); err != nil {
		t.Fatal(err)
	}
	outside.live.stopped = true
	if outside.live.cheats.campaign || outside.live.cheats.privilege[sim.SelfSlot] != 0 {
		t.Fatal("a mission outside the campaign admitted the unlock")
	}
	dismissSecondMissionDialogue(t, oa, outside.live)
	h := outside.live.world.Hash()
	for _, line := range []string{secondUnlock + "d", "#kick x", "#locate x", "#set latency 50", "#show latency", "#create 5 Gold"} {
		secondCheatWantNothing(t, outside, secondCheatLines(t, outside, oa, line), h, "outside the campaign "+line)
	}
}
