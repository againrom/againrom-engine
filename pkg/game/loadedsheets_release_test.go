package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A mission SAVE taken after creatures have died and their bodies have left
// the world is loaded with fewer placements than the ids were minted from, so
// an entity's id no longer indexes its own placement. Each creature must still
// carry the sheet of its own placement: on mission 10 three creatures stood
// after such a load with the person sheet Body 30, Reaction 25, Mind 25,
// Spirit 25 where their own is Body 5, Reaction 60, Mind 1, Spirit 2.
//
// The state is reached through ordinary input (the selection, the attack key
// and a click on the target, a click on the ground) and ordinary ticks; each
// mission notice closes with Enter. SAVE is the ordinary producer and LOAD the
// load dialog of a cold front end. The fresh mission is the reference: each
// creature's statistics card, hovered and read from the mission's character
// pane after the J key, must be the same picture in the loaded mission as in
// the fresh one.
func TestReleaseLoadedCreaturesKeepTheirSheets(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("loaded sheets")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	fresh := readPlacedSheets(t, a, live, "fresh mission 10")
	if n := len(live.mission.state.Map.Units); n != len(fresh) {
		t.Fatalf("mission 10 places %d units and %d of them read a sheet", n, len(fresh))
	}

	r := placedSpeakerRoute{t: t, a: a, live: live, hero: live.mission.ids[0]}
	r.kill(0)
	r.kill(1)
	r.walk(42, 47)
	gone := []sim.EntityID{15, 14, 12}
	for _, target := range gone {
		r.kill(target)
	}
	for n := 0; ; n++ {
		left := 0
		for _, target := range gone {
			if _, ok := live.entity(target); ok {
				left++
			}
		}
		if left == 0 {
			break
		}
		if n == 2000 {
			t.Fatalf("%d of the three creatures still stand in the world", left)
		}
		r.advance()
	}

	path, _ := writeOrdinarySAV(t, f, "loaded-sheets.sav")
	g, b, cold := placedSpeakerLoad(t, filepath.Dir(path), filepath.Base(path))
	compareLoadedSheets(t, g, fresh, b, cold, "mission 10 after LOAD", true)
}

// The owner's mission-10 save is a real file whose load withdraws placements
// of creatures that are gone. The creatures it keeps stand beside creatures of
// their own kind, so an index-keyed lookup lands on the right sheet by
// chance and the constructed route above is the case that tells the pairings
// apart; this file shows that a real save, and the save written from it,
// load each creature on its own sheet and hover to the same card as a fresh
// mission at the save's own difficulty.
func TestReleaseOwnerSaveCreaturesKeepTheirSheets(t *testing.T) {
	path := os.Getenv("AGAINROM_SARINDAR_SAV")
	if path == "" {
		t.Skip("no AGAINROM_SARINDAR_SAV: the owner's mission-10 save is an owner input")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "18e126de76e914ae1da420eacaf0da53917d931755f2bd815df3337c13bf1152" {
		t.Fatalf("owner save changed: %s", got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mission10.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	g, b, loaded := placedSpeakerLoad(t, dir, "mission10.sav")

	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.Difficulty = g.Difficulty
	f.SetDeterministicFrames(true)
	a := f.App("loaded sheets reference")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	fresh := readPlacedSheets(t, a, f.live, "fresh mission 10")

	compareLoadedSheets(t, g, fresh, b, loaded, "owner save", false)
	resaved, _ := writeOrdinarySAV(t, g, "owner-resaved.sav")
	h, c, cold := placedSpeakerLoad(t, filepath.Dir(resaved), filepath.Base(resaved))
	compareLoadedSheets(t, h, fresh, c, cold, "owner save after SAVE and LOAD", false)
}

// placedSheetRead is what one placed actor of a mission states: the sheet its
// character map holds and, for a living creature, the statistics card the
// character pane draws when the pointer hovers it.
type placedSheetRead struct {
	entity sim.EntityID
	sheet  ui.UnitCharacter
	card   *image.RGBA
	hp     int32
}

// readPlacedSheets reads every placed actor of live, keyed by the map unit id
// its placement carries, with the character pane in its statistics mode. The
// presentation fog is opened for the reading and put back, and the world hash
// must not move.
func readPlacedSheets(t *testing.T, a *ui.App, live *mapWorld, when string) map[uint16]placedSheetRead {
	t.Helper()
	before := live.world.Hash()
	visible, explored := slices.Clone(live.fog.visible), slices.Clone(live.fog.explored)
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	// The cards are read at full knowledge: the kills that reach the loaded
	// state raise the Diary, and a card at a lower level draws fewer groups.
	if err := a.HeadlessKey("shift-f4"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := a.HeadlessKey("shift-f4"); err != nil {
			t.Fatal(err)
		}
	}()
	if _, stats, err := a.HeadlessCharacterPane(); err != nil {
		t.Fatalf("%s: %v", when, err)
	} else if !stats {
		if err := a.HeadlessKey("j"); err != nil {
			t.Fatal(err)
		}
	}
	out := make(map[uint16]placedSheetRead)
	for _, e := range live.world.Entities() {
		if e.MapUnitID == 0 || live.missionPartyMember(e.ID) != nil {
			continue
		}
		if _, dup := out[e.MapUnitID]; dup {
			t.Fatalf("%s: map unit %d names two entities", when, e.MapUnitID)
		}
		read := placedSheetRead{entity: e.ID, sheet: live.chars[e.ID], hp: e.HP}
		if !e.Humanoid && e.Alive() {
			inspectionCentre(live, int(e.X), int(e.Y))
			live.push()
			releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(e.ID)})
			pic, stats, err := a.HeadlessCharacterPane()
			if err != nil || !stats {
				t.Fatalf("%s: entity %d: statistics card unavailable: stats=%t err=%v", when, e.ID, stats, err)
			}
			read.card = image.NewRGBA(pic.Bounds())
			copy(read.card.Pix, pic.Pix)
		}
		out[e.MapUnitID] = read
	}
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	copy(live.fog.visible, visible)
	copy(live.fog.explored, explored)
	if live.world.Hash() != before {
		t.Fatalf("%s: reading the sheets changed the world", when)
	}
	return out
}

// compareLoadedSheets checks every placed actor of a loaded mission against
// the fresh mission's reading of the same placement: a creature's whole sheet
// and its hovered statistics card, a person's band and four statistics. The
// loaded map must be shorter than the fresh one and place at least three
// creatures at another index than their entity id. When tellsPairingsApart is
// set, an index-keyed lookup over that map, computed here from the map-loading
// tier's own sheets, must also name another placement's sheet for at least
// three creatures and ten hovered cards must match at equal health.
func compareLoadedSheets(t *testing.T, g *FrontEnd, fresh map[uint16]placedSheetRead, a *ui.App, live *mapWorld, when string, tellsPairingsApart bool) {
	t.Helper()
	m := live.mission.state.Map
	if len(m.Units) >= len(fresh) {
		t.Fatalf("%s: the load kept %d placements of %d; nothing was withdrawn", when, len(m.Units), len(fresh))
	}
	byIndex := mapload.PlacedSheets(m, g.Table)
	got := readPlacedSheets(t, a, live, when)
	units := make([]uint16, 0, len(got))
	for unit := range got {
		units = append(units, unit)
	}
	slices.Sort(units)
	persons, creatures, cards, moved, indexWrong := 0, 0, 0, 0, 0
	for _, unit := range units {
		l := got[unit]
		want, ok := fresh[unit]
		if !ok {
			t.Errorf("%s: unit %d is loaded as entity %d and was not placed in the fresh mission", when, unit, l.entity)
			continue
		}
		if want.card == nil {
			persons++
			if statisticsOf(l.sheet) != statisticsOf(want.sheet) {
				t.Errorf("%s: person unit %d (entity %d, fresh entity %d) carries %+v, want its own placement's %+v",
					when, unit, l.entity, want.entity, statisticsOf(l.sheet), statisticsOf(want.sheet))
			}
			continue
		}
		creatures++
		if l.sheet != want.sheet {
			t.Errorf("%s: creature unit %d (entity %d, fresh entity %d) carries %+v, want its own placement's %+v",
				when, unit, l.entity, want.entity, l.sheet, want.sheet)
		}
		if i := slices.IndexFunc(m.Units, func(u alm.Unit) bool { return u.UnitID == unit }); i != int(l.entity) {
			moved++
		}
		if s, found := byIndex[l.entity]; !found || sheetCharacter(s) != want.sheet {
			indexWrong++
		}
		if l.card == nil || l.hp != want.hp {
			continue
		}
		cards++
		if l.card.Bounds() != want.card.Bounds() || !bytes.Equal(l.card.Pix, want.card.Pix) {
			t.Errorf("%s: creature unit %d (entity %d): the hovered statistics card differs from the fresh mission's", when, unit, l.entity)
		}
	}
	t.Logf("%s: %d persons and %d creatures read; %d creatures placed at another index than their entity id, %d of them named by that id as another placement's sheet; %d hovered cards compared at equal health",
		when, persons, creatures, moved, indexWrong, cards)
	if persons < 10 || creatures < 10 || moved < 3 {
		t.Errorf("%s: %d persons, %d creatures, %d moved: too few to discriminate", when, persons, creatures, moved)
	}
	if tellsPairingsApart && (indexWrong < 3 || cards < 10) {
		t.Errorf("%s: %d index-wrong creatures, %d cards: too few to tell the pairings apart", when, indexWrong, cards)
	}
}

// statisticsOf is the part of a person's sheet the roster states for a placed
// person and the loaded mission must state alike.
func statisticsOf(c ui.UnitCharacter) [5]int {
	return [5]int{int(c.Band), c.Body, c.Reaction, c.Mind, c.Spirit}
}
