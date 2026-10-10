package game

// Tests for the front-end's own marker selection: the field is read on every
// load, so what the game's flag writes is what the map screen shows.
//
// This is an internal test because the seam it needs is the picker's loader,
// which is unexported — and it is the seam that matters: the defect this covers
// was a load path that worked and a front-end that never reached it.
//
// The FrontEnd is assembled directly rather than through NewFrontEnd, so the
// test needs no menu asset set to ask a question about map loading. NewFrontEnd's
// own default is asserted where an install fixture already exists, in
// cmd/againrom.

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// looseOver is the loose filesystem a HAND-ASSEMBLED FrontEnd needs: a
// picked row's bytes come from its bare name as an address now, resolved
// under the directories that filesystem lists, and not from a path joined
// onto the asset root. A front-end out of NewFrontEnd carries one already.
func looseOver(t *testing.T, dir string) *vfs.FS {
	t.Helper()
	loose, err := vfs.Open(nil, []string{dir})
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return loose
}

func TestFrontEndMarkers(t *testing.T) {
	dir := t.TempDir()
	data := synth.ALM(synth.ALMOptions{
		Width: 6, Height: 5, Name: "Marked",
		Objects: []synth.ALMObject{{X: 0x0280, Y: 0x01c0}},
		Units:   []synth.ALMUnit{{X: 0x0500, Y: 0x0080}, {X: 0x0180, Y: 0x0300}},
	})
	if err := os.WriteFile(filepath.Join(dir, "a.alm"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The map is reachable through the LOOSE FILESYSTEM alone.
	loose := looseOver(t, dir)
	empty := t.TempDir()
	front := func(m Markers) *FrontEnd {
		return &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: empty, Loose: loose}, Tiles: &terrain.Tileset{}, Maps: []MapEntry{{Source: "a.alm", Name: "Marked"}}}, Presentation: Presentation{Markers: m}}
	}

	t.Run("the field reaches the map the picker opens", func(t *testing.T) {
		v, _, _, _, _, _, _, _, _, _, err := front(Markers{Objects: true, Units: true}).loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if on, cells := v.ObjectOverlay(); !on || cells != 1 {
			t.Errorf("object overlay = (on %v, %d cells), want (on true, 1 cell)", on, cells)
		}
		if on, cells := v.UnitOverlay(); !on || cells != 2 {
			t.Errorf("unit overlay = (on %v, %d cells), want (on true, 2 cells)", on, cells)
		}
	})

	// THE GAME OPENS WITH NO DEBUG READOUT, and F1 is what brings it back.
	//
	// The viewer's own default is SHOWN and stays that way (0060: a readout that
	// had to be switched on could not report the moment a map opens), which is
	// right for cmd/mapview and wrong for the game. So this asserts WHERE the
	// switch is thrown and not what the default is — a bare ui.Viewer still
	// answers shown, and asking that instead would pass vacuously.
	t.Run("the readout box is off on the map the picker opens", func(t *testing.T) {
		v, _, _, _, _, _, _, _, _, _, err := front(Markers{}).loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if v.ReadoutShown() {
			t.Error("the game's map screen opened with the debug readout showing")
		}
		if !v.ToggleReadout() {
			t.Error("F1 did not bring the readout back")
		}
	})

	t.Run("switched off, the map screen is the one that shipped", func(t *testing.T) {
		v, _, _, _, _, _, _, _, _, _, err := front(Markers{}).loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if on, _ := v.ObjectOverlay(); on {
			t.Errorf("object overlay is on with markers off")
		}
		if on, _ := v.UnitOverlay(); on {
			t.Errorf("unit overlay is on with markers off")
		}
	})
}

// TestTheHeadlessCheckLineIsWhatItWas — 0028 SC-10's check-line clause
// (AC-11): the game's headless mode prints the line it printed before this
// story, character for character.
//
// IT IS A CHARACTERIZATION PIN and the literal is deliberate: nothing about a
// selection, a pick, a highlight or a pending order may reach the one line the
// headless mode prints, and a substring check would let a token be appended to
// the end of it while staying green. The whole line is therefore the assertion.
//
// The mask is HAND-BUILT rather than loaded, so this needs no menu asset set and
// no install: MaskRegions counts frame pixels carrying a button's hot index, and
// four pixels are enough to state a count. Which button each index selects is
// read back through the package's own ButtonAt rather than assumed here, so the
// premise "these two bytes are two different buttons' regions" is checked
// instead of transcribed — and the second number in the line is then a COUNT of
// what the mask holds and not a constant that happens to read right.
func TestTheHeadlessCheckLineIsWhatItWas(t *testing.T) {
	// Two hot indices, one index that names no button, and one plain zero.
	mask := image.NewPaletted(image.Rect(0, 0, 4, 1), nil)
	copy(mask.Pix, []byte{0x80, 0x00, 0xc0, 0x07})
	assets := &menu.Assets{Mask: mask}

	buttons := map[int]bool{}
	for x := 0; x < 4; x++ {
		if n := assets.ButtonAt(image.Pt(x, 0)); n != 0 {
			buttons[n] = true
		}
	}
	if len(buttons) != 2 {
		t.Fatalf("the hand-built mask selects %d distinct buttons %v, want the 2 the line below is written "+
			"over — the pin would otherwise state a count the fixture does not hold", len(buttons), buttons)
	}

	f := &FrontEnd{InstallResources: InstallResources{Assets: assets, Maps: []MapEntry{{Source: "a.alm", Name: "A"}, {Source: "b.alm", Name: "B"}, {Source: "c.alm", Name: "C"}}}}
	// The hand-built front-end resolved no weapon, so its hero is BARE and the
	// line says so rather than stating a band it does not have (0078 AC-15).
	// His CHARACTER is stated either way: a bare hero still has one, and the
	// spread is what the numbers beside it are a function of.
	const want = "againrom: 3 map rows, 2 of 8 buttons have a mask region; " +
		"hero Body 43, Reaction 26, Mind 15, Spirit 15, Blade 10, bare"
	if got := f.CheckLine(); got != want {
		t.Errorf("the headless check line changed:\n got %q\nwant %q", got, want)
	}

	// Armed, the same line states the band the party's hero starts at — which is
	// the one figure a headless run can print that says whether the player's own
	// unit can fight.
	f.StartWeapon = resolved(&data.Weapon{
		Name: "Iron Short Sword", DamageBase: 5, DamageSpread: 3,
		ToHit: 5, AttackType: data.SkillBlade, ChargeTime: 9, RelaxTime: 5, Range: 1,
	}, nil)
	const armed = "againrom: 3 map rows, 2 of 8 buttons have a mask region; " +
		"hero Body 43, Reaction 26, Mind 15, Spirit 15, Blade 10, " +
		"Iron Short Sword 10-16, to-hit 49, defence 8"
	if got := f.CheckLine(); got != armed {
		t.Errorf("the armed check line:\n got %q\nwant %q", got, armed)
	}

	// AND THE NUMBERS ARE THE PARTY MEMBER'S OWN (0082 AC-18). The literal above
	// pins the wording; this pins that the wording is about the hero a mission
	// actually places. A check line built from a SECOND construction of the hero
	// is the defect this story is worst at noticing — it would print the
	// generation start's 7-10 and to-hit 39, which are correct answers to a
	// different question and are a criterion of this very story.
	p := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	c := p.Hero.Derive(p.Weapon)
	clause := fmt.Sprintf("Body %d, Reaction %d, Mind %d, Spirit %d, %s %d, %s %d-%d, to-hit %d, defence %d",
		p.Hero.Body, p.Hero.Reaction, p.Hero.Mind, p.Hero.Spirit,
		data.SkillName(PartySkillSlot()), p.Hero.Skill[PartySkillSlot()],
		f.StartWeapon.Value().Name, c.DamageBase, c.DamageBase+c.DamageSpread, c.ToHit, c.Defence)
	if got := f.CheckLine(); !strings.Contains(got, clause) {
		t.Errorf("the check line does not state the party member's own sheet:\n got %q\nwant it to carry %q",
			got, clause)
	}
}

// TestMapBytesAddresses pins the two addresses a picked row's bytes come
// from: an archive row's the scenario identity and the remainder the row
// listed under, a loose row's its bare file name, which carries no identity
// segment and so resolves under the asset root.
//
// Both cases are arranged so a WRONG address still finds bytes — a container
// holding the same entry name, and a copy of the file under Root — so what each
// assertion distinguishes is the route, not that a read happened to succeed.
func TestMapBytesAddresses(t *testing.T) {
	t.Run("an archive row reads the scenario identity's entry", func(t *testing.T) {
		dir := t.TempDir()
		lay := func(name string, files []synth.File) string {
			t.Helper()
			p := filepath.Join(dir, name)
			if err := os.WriteFile(p, synth.Archive(files), 0o644); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
			return p
		}
		campaign := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "Campaign"})
		decoy := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "Decoy"})
		// The decoy container is listed FIRST and holds the same entry name, so a
		// read that named any identity but the campaign container's would resolve —
		// to the wrong bytes.
		mainPath := lay(MainArchive, []synth.File{{Path: "a.alm", Data: decoy}})
		scenarioPath := lay(ScenarioArchive, []synth.File{{Path: "a.alm", Data: campaign}})
		containers, err := vfs.Open([]string{mainPath, scenarioPath}, nil)
		if err != nil {
			t.Fatalf("vfs.Open: %v", err)
		}

		f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: dir, Containers: containers}}}
		got, err := f.mapBytes(MapEntry{Source: "a.alm", FromArchive: true})
		if err != nil {
			t.Fatalf("mapBytes: %v", err)
		}
		if !bytes.Equal(got, campaign) {
			t.Errorf("an archive row's bytes are not %s's; the address names the wrong container identity",
				ScenarioArchive)
		}
	})

	t.Run("a loose row reads its bare name on the loose filesystem, folded", func(t *testing.T) {
		root := t.TempDir()
		looseDir := t.TempDir()
		beside := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "Beside The Route"})
		addressed := synth.ALM(synth.ALMOptions{Width: 5, Height: 5, Name: "Addressed"})
		// Root holds the row's name in the case a scan would have listed it, and the
		// filesystem's own directory holds it FOLDED. A host read joined onto Root
		// returns the first; only the address returns the second.
		if err := os.WriteFile(filepath.Join(root, "Tomb.ALM"), beside, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := os.WriteFile(filepath.Join(looseDir, "tomb.alm"), addressed, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: root, Loose: looseOver(t, looseDir)}}}
		got, err := f.mapBytes(MapEntry{Source: "Tomb.ALM"})
		if err != nil {
			t.Fatalf("mapBytes: %v", err)
		}
		if !bytes.Equal(got, addressed) {
			t.Errorf("a loose row's bytes did not come from the loose filesystem at the folded name; "+
				"%q must resolve as an address, not as a path joined onto the asset root", "Tomb.ALM")
		}
	})
}

func TestABlowBecomesACommandInTheQueueTheOrdersUse(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	w, err := sim.NewWorld(1, b, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
		{ID: 1, X: 3, Y: 1, HP: 25, MaxHP: 25},
		{ID: 2, X: 5, Y: 1, HP: 4, MaxHP: 4}, // a maximum below ten
		{ID: 3, X: 7, Y: 1},                  // no health system at all
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: make(map[sim.EntityID]int), commanded: make(map[sim.EntityID]bool)}

	before := w.Hash()
	mw.affect(0, false)
	mw.affect(1, false)
	mw.affect(2, false)
	mw.affect(3, false)
	mw.affect(1, true)
	mw.affect(99, true) // an entity the world does not hold

	want := []sim.Command{
		{Kind: sim.KindDamage, Entity: 0, X: 10}, // a tenth of 100
		{Kind: sim.KindDamage, Entity: 1, X: 2},  // a tenth of 25, truncated
		{Kind: sim.KindDamage, Entity: 2, X: 1},  // a tenth of 4 is 0, and a chip is never less than 1
		{Kind: sim.KindDamage, Entity: 3, X: 1},  // and never less than 1 on a maximum of 0 either
		{Kind: sim.KindKill, Entity: 1},
	}
	if !reflect.DeepEqual(mw.pending, want) {
		t.Errorf("the queue holds\n %+v\nwant\n %+v", mw.pending, want)
	}

	if got := w.Hash(); got != before {
		t.Errorf("the world hashes %#016x after six blows and %#016x before them", got, before)
	}
	if len(mw.commanded) != 0 {
		t.Errorf("a blow marked %d entity/entities commanded; a blow is not an order", len(mw.commanded))
	}
}

func TestABlowReachesTheWorldOnlyThroughAnAdvance(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))

	mw.affect(0, true)
	mw.affect(1, false)
	quiet := mw.world.Hash()
	if len(mw.pending) != 2 {
		t.Fatalf("the queue holds %d command(s), want 2", len(mw.pending))
	}

	mw.tick()
	if len(mw.pending) != 0 {
		t.Errorf("the queue still holds %d command(s) after an advance", len(mw.pending))
	}
	if mw.world.Hash() == quiet {
		t.Errorf("the world hashes %#016x after the advance that applied two blows, unchanged", quiet)
	}

	ents := mw.world.Entities()
	if !ents[0].Dead() {
		t.Errorf("entity 0 is at %d/%d after a kill", ents[0].HP, ents[0].MaxHP)
	}
	if want := mapload.SpawnHP - mapload.SpawnHP/10; ents[1].HP != want {
		t.Errorf("entity 1 is at %d health after one chip, want %d", ents[1].HP, want)
	}

	// And a second advance applies nothing further: the queue was emptied by the
	// tick that used it, not by the one after.
	afterFirst := mw.world.Entities()[1].HP
	mw.tick()
	if got := mw.world.Entities()[1].HP; got != afterFirst {
		t.Errorf("entity 1 is at %d health a tick later, want the %d one advance left", got, afterFirst)
	}
}

// AC-13's second half: a front-end that could not load a font still opens maps.
// Assembled directly, as everything else in this file is, because the question
// is what a Font-less front-end DOES and not how one comes about.
func TestAFontlessFrontEndStillOpensAMap(t *testing.T) {
	dir := t.TempDir()
	data := synth.ALM(synth.ALMOptions{Width: 6, Height: 5, Name: "Fontless",
		Units: []synth.ALMUnit{{X: 0x0180, Y: 0x0300}}})
	if err := os.WriteFile(filepath.Join(dir, "a.alm"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: t.TempDir(), Loose: looseOver(t, dir)}, Tiles: &terrain.Tileset{}, Maps: []MapEntry{{Source: "a.alm", Name: "Fontless"}}, Font: resolved[*text.Font](nil, errors.New("no font in this install"))}}

	v, tick, order, cadence, affect, _, _, _, _, _, err := f.loadMap(0)
	if err != nil {
		t.Fatalf("loadMap on a fontless front-end: %v", err)
	}
	if v == nil || tick == nil || order == nil || cadence == nil || affect == nil {
		t.Fatal("a fontless front-end yielded an incomplete map screen")
	}
	// The map is live: it advances, and the viewer holds the entity the map
	// places. A front-end missing a font must lose the panel and nothing else.
	tick()
	if got := v.EntityMarkers(); got != 1 {
		t.Errorf("the viewer holds %d entity, want 1 — the map did not open", got)
	}
}

// The mission door's front-end half (AC-1, AC-4).
//
// The FrontEnd is assembled by hand over a scenario archive holding one map, so
// these need no menu asset set to ask a question about opening a mission.

// missionFrontEnd is a front end whose scenario container holds `10.alm` and
// nothing else that matters.
func missionFrontEnd(t *testing.T) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	data := synth.Archive([]synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{
			Width: 24, Height: 24,
			Units: []synth.ALMUnit{{X: 0x0280, Y: 0x01c0}, {X: 0x0500, Y: 0x0080}},
		})},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	})
	path := filepath.Join(dir, ScenarioArchive)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	containers, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: containers}, Tiles: &terrain.Tileset{}}}
}

// AC-1: the flag opens THAT mission's map screen, over a world built from its
// map — and it does it through the picker's own entry, so the map screen is the
// one a pick would have produced.
func TestMissionOpenerShowsTheMapScreen(t *testing.T) {
	f := missionFrontEnd(t)
	a := f.App("t")
	if a.Screen() != ui.ScreenMenu {
		t.Fatalf("setup: the app opens on %v, want the menu", a.Screen())
	}
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatalf("OpenMission: %v", err)
	}
	if a.Screen() != ui.ScreenMap {
		t.Fatalf("screen = %v, want ScreenMap", a.Screen())
	}
}

// AC-3: without the flag the front-end is untouched — the app opens on the main
// menu, which is what a run with no -mission does.
func TestNoMissionLeavesTheAppOnTheMenu(t *testing.T) {
	f := missionFrontEnd(t)
	if a := f.App("t"); a.Screen() != ui.ScreenMenu {
		t.Fatalf("screen = %v, want ScreenMenu", a.Screen())
	}
}

// AC-4: the started world holds ONE party member more than the map's own
// placements, carrying the class key its body DERIVES, standing on the
// mission's start cell.
func TestTheMissionPartyIsOneMemberAtTheStartCell(t *testing.T) {
	f := missionFrontEnd(t)
	// A small invented body list (SC-3): index 0 is the bare-handed name a
	// nil weapon's derivation answers.
	f.Bodies = data.NewBodyList("swordsman")
	party := MissionParty(nil, f.Bodies, nil)
	ms, err := StartMission(f.Archives.Containers, 10, f.Table, openDifficulty, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	ents := ms.World.Entities()
	if got, want := len(ents), len(ms.Map.Units)+1; got != want {
		t.Fatalf("%d entities, want %d — the map's placements plus one member", got, want)
	}
	// The party is APPENDED, so the member is the last entity — 0065's own
	// disclosure, relied on here rather than re-derived.
	member := ents[len(ents)-1]
	wantBody, ok := data.HeroBodyFor(f.Bodies, data.Equipment{})
	if !ok {
		t.Fatalf("fixture: the bare-handed row derived no name")
	}
	want, matched := data.HeroBodyClass(wantBody)
	if !matched {
		t.Fatalf("the fixture body %q matches no arm of the name chain", wantBody)
	}
	if member.Class != want {
		t.Errorf("the member carries class %d, want the %q body's own %d",
			member.Class, wantBody, want)
	}
	if member.X != ms.Start.Drop.X || member.Y != ms.Start.Drop.Y {
		t.Errorf("the member stands at (%d, %d), want the start's own (%d, %d)",
			member.X, member.Y, ms.Start.Drop.X, ms.Start.Drop.Y)
	}
	if len(ms.Start.Cells) != 1 || ms.Start.Cells[0] != ms.Start.Drop {
		t.Errorf("start = %+v, want exactly one member cell, on the drop cell", ms.Start)
	}
}

// The headless report reaches the same mission and says what it built.
func TestMissionLineReportsTheStart(t *testing.T) {
	f := missionFrontEnd(t)
	line, err := f.MissionLine(10)
	if err != nil {
		t.Fatalf("MissionLine: %v", err)
	}
	for _, want := range []string{"mission 10", "scenario/10.alm", "24x24", "party at"} {
		if !bytes.Contains([]byte(line), []byte(want)) {
			t.Errorf("line %q does not carry %q", line, want)
		}
	}
	if _, err := f.MissionLine(11); err == nil {
		t.Error("a mission the archive does not hold reported no error")
	}
}

// The opener's own failures, at the front end rather than at the command: each
// names what it failed at and no two are equal.
func TestMissionOpenerFailuresAreDistinct(t *testing.T) {
	f := missionFrontEnd(t)
	seen := map[string]bool{}
	for _, n := range []int{-3, 11} {
		_, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(n)()
		if err == nil {
			t.Fatalf("mission %d opened", n)
		}
		if seen[err.Error()] {
			t.Errorf("mission %d repeats an earlier message %q", n, err)
		}
		seen[err.Error()] = true
	}
	// The refused number names the number, because there is no address for it
	// to name; the absent entry names the address.
	if len(seen) != 2 {
		t.Fatalf("two failures produced %d distinct messages", len(seen))
	}
}

func TestLoadMapRoutesAMissionRowToTheMissionDoor(t *testing.T) {
	f := missionFrontEnd(t)
	f.Maps = []MapEntry{
		{Source: "10.alm", FromArchive: true, Mission: 10},
		{Source: "10.alm", FromArchive: true},
	}

	_, _, _, _, _, missionAdvance, _, _, _, _, err := f.loadMap(0)
	if err != nil {
		t.Fatalf("loadMap(0) (mission row): %v", err)
	}
	if missionAdvance == nil {
		t.Error("a mission row's loadMap returned a nil MapAdvance; DD-5's delegation to MissionOpener is missing")
	}

	_, _, _, _, _, mapAdvance, _, _, _, _, err := f.loadMap(1)
	if err != nil {
		t.Fatalf("loadMap(1) (map row): %v", err)
	}
	if mapAdvance != nil {
		t.Error("a map row's loadMap returned a non-nil MapAdvance; FR-4 forbids an advance-notice seam on a plain map")
	}
}

// missionFrontEndWithBodies is missionFrontEnd's own shape, widened with a
// SECOND container, main.res, serving list at HeroPictureAddress — the
// two-container layout a real install has, and the one openMission's own
// ReadBodyList call (world.go, D-6) actually needs: that call reads
// "main/text/heropicture.txt", and missionFrontEnd's own scenario-only
// archive answers every such address with a miss, which is fine for every
// test that does not open a mission's OWN paced path but says nothing
// about it.
func missionFrontEndWithBodies(t *testing.T, list data.BodyList) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	scenario := synth.Archive([]synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 24, Height: 24})},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	})
	names := make([]string, list.Len())
	for i := range names {
		names[i] = string(list.Entry(i))
	}
	main := synth.Archive([]synth.File{
		{Path: "text/heropicture.txt", Data: []byte(strings.Join(names, "\n") + "\n")},
	})

	hosts := make([]string, 0, 2)
	for name, content := range map[string][]byte{ScenarioArchive: scenario, MainArchive: main} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		hosts = append(hosts, path)
	}
	containers, err := vfs.Open(hosts, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: containers}, Tiles: &terrain.Tileset{}}}
}

// AC-18 — one equipment set derives the same body, directory and class at
// the open and at a refresh: assembleParty's own HeroAppearance call, at
// mission start, and refreshAppearance's own, over the SAME worn set the
// mint folded into the world, are two independent expressions that must not
// come to disagree.
//
// THE FIXTURE IS A REAL, RESOLVABLE WEAPON (equip_test.go's own eqDefsTable
// and eqMace) rather than a bare hero, and its own Weapons row is 2 — so the
// worn set's slot 1 lands on HeroBodyFor's row-2 arm, list[1] "swordsman", a
// discriminating case rather than the bare-handed row every empty fixture
// would agree on for free.
func TestAMissionsOpenAndItsFirstRefreshAgreeOnOneEquipmentSet(t *testing.T) {
	list := data.NewBodyList("unarmed", "swordsman")
	f := missionFrontEndWithBodies(t, list)
	table := eqDefsTable(t)
	weapon := eqMace(t, table)
	party := MissionParty(weapon, list, table)

	ms, err := StartMission(f.Archives.Containers, 10, table, openDifficulty, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	mw := openMission(ms, table, nil, worldFixtureViewer(t, ms.Map), f.Archives.Containers, nil, nil)

	// SETUP'S OWN CHECK, not the claim: openMission actually read the
	// archive's shipped list rather than carrying invParty.list at its zero
	// value, which would let the assertion below pass over two empty lists
	// agreeing about nothing.
	if !reflect.DeepEqual(mw.invParty.list, list) {
		t.Fatalf("setup: invParty.list = %#v, want %#v off main.res — openMission's own ReadBodyList call",
			mw.invParty.list, list)
	}

	gotBody, gotDir, gotClass, _ := data.HeroAppearance(mw.invParty.list, mw.currentEquipment(), mw.invParty.mage, false)
	if string(gotBody) != party[0].Body || gotDir != party[0].BodyDir || gotClass != party[0].Class {
		t.Errorf("the refresh derives (%q, %q, %d), want the open's own (%q, %q, %d)",
			gotBody, gotDir, gotClass, party[0].Body, party[0].BodyDir, party[0].Class)
	}
}
