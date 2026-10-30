package game

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// presentationWorld is a map world of the first or the second game with a
// sheet for each picture the cases below release.
func presentationWorld(t *testing.T, second bool) *mapWorld {
	t.Helper()
	mw := sbWorld(t)
	if second {
		mw.world = secondGameScriptWorld(t, nil, nil)
	}
	sheets := map[int]*terrain.EffectSheet{}
	for _, id := range []int{10, 18, 23, 28, 30, 54, 56, 60} {
		sheets[id] = &terrain.EffectSheet{Frames: spFrames(9, 8), Phases: 9, RotationPhases: 1, CenterX: 4, CenterY: 4}
	}
	mw.projectiles = &terrain.EffectSet{Sheets: sheets}
	return mw
}

// castSegments releases one cast of row and returns the segment count each
// record keeps after its first driver call, in id order. A record of no life
// retires on that call and keeps none.
func castSegments(t *testing.T, second bool, row uint16) []int32 {
	t.Helper()
	mw := presentationWorld(t, second)
	releaseCastEvent(mw, sim.CastEvent{Caster: 1, Target: 2, AtCell: true, Spell: row, FromX: 1, FromY: 1, ToX: 9, ToY: 1})
	var out []int32
	for _, p := range mw.world.SavedProjectiles().Items {
		out = append(out, p.ActionSegments)
	}
	return out
}

// The cast site takes the life, path and second record of the World's own
// game (R2-ENGINE-445, R2-ENGINE-447). The first game's rows of the same
// identifiers are the loss control: they build no living object. The cast
// covers 8 cells, 2048 units.
func TestTheCastSiteTakesTheWorldsSet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		row          uint16
		second, want []int32
	}{
		// Ice Missile, identifier 18: travels distance/200 in the second game.
		{"ice missile", 5, []int32{2048/200 - 1}, nil},
		// Lightning, identifier 28: the 13-update path.
		{"lightning", 10, []int32{13 - 1}, nil},
		// Teleport, identifier 54: two records living the sheet's 9 phases.
		{"teleport", 23, []int32{9 - 1, 9 - 1}, nil},
	} {
		if got := castSegments(t, true, tc.row); !equalSegments(got, tc.second) {
			t.Errorf("%s in the second game: segments %v, want %v", tc.name, got, tc.second)
		}
		if got := castSegments(t, false, tc.row); !equalSegments(got, tc.want) {
			t.Errorf("%s row in the first game: segments %v, want %v", tc.name, got, tc.want)
		}
	}
}

func equalSegments(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The sound sites take the World's set: row 29 voices 566 and the Blizzard
// cell queues 523 for nine cue counts in the second game; the first
// game keeps row 29 silent and row 7's cell immediate.
func TestTheSoundSitesTakeTheWorldsSet(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{true, false} {
		mw := presentationWorld(t, second)
		plays := recordSpellSounds(mw)
		set := mw.spells()
		wantCast := map[bool]int{true: 566, false: 0}[second]
		if got := castSpellSoundSlot(set, 29); got != wantCast {
			t.Errorf("second=%v: row 29 cast slot %d, want %d", second, got, wantCast)
		}
		mw.observeAreaPaints([]sim.AreaPaint{{Spell: 7, Cells: []sim.CellPoint{{X: 3, Y: 3}}}})
		switch {
		case second && (len(*plays) != 0 || len(mw.spellSoundCues) != 1 ||
			mw.spellSoundCues[0].slot != 523 || mw.spellSoundCues[0].ticks != 9):
			t.Errorf("second game: row 7 cell played %v, queued %+v; want 523 after 9", *plays, mw.spellSoundCues)
		case !second && (len(*plays) != 1 || (*plays)[0].slot != 523 || len(mw.spellSoundCues) != 0):
			t.Errorf("first game: row 7 cell played %v, queued %+v; want 523 at once", *plays, mw.spellSoundCues)
		}
	}
}

// cellSoundFrame is the frame a cell object created by queue shows when its
// sound plays: the creating update counts the cue down once at frame 0, each
// later update once more.
func cellSoundFrame(t *testing.T, mw *mapWorld, queue func()) int {
	t.Helper()
	plays := recordSpellSounds(mw)
	queue()
	for frame := 0; frame < 32; frame++ {
		mw.advanceSpellSoundCues()
		if len(*plays) != 0 {
			return frame
		}
	}
	t.Fatal("the cell sound never played")
	return -1
}

// The second game's Blizzard cell voices 523 with frame 8: the original's
// counter starts at -1 and sounds when it reaches 8 (R2-ENGINE-446). Loss
// control: a cue of 8 counts plays with frame 7.
func TestTheBlizzardCellSoundsWithFrameEight(t *testing.T) {
	t.Parallel()
	mw := presentationWorld(t, true)
	if got := cellSoundFrame(t, mw, func() {
		mw.observeAreaPaints([]sim.AreaPaint{{Spell: 7, Cells: []sim.CellPoint{{X: 3, Y: 3}}}})
	}); got != 8 {
		t.Errorf("the Blizzard cell sounds with frame %d, want 8", got)
	}
	control := presentationWorld(t, true)
	if got := cellSoundFrame(t, control, func() {
		control.queueSpellSound(523, 0, image.Pt(3, 3), 8)
	}); got == 8 {
		t.Error("a cue of 8 counts also sounds with frame 8")
	}
}

// The light site takes the World's set: Ice Missile's 18 lights its cell at
// radius 0, level 16 in the second game (R2-ENGINE-448) and not in the first.
func TestTheLightSiteTakesTheWorldsSet(t *testing.T) {
	t.Parallel()
	cell := image.Pt(5, 5)
	second := pictureLightStamps(nil, presentationWorld(t, true).spells(), 18, cell, 0)
	if len(second) != 4 || second[0].Level != 16 {
		t.Errorf("second game: identifier 18 stamps %+v, want four vertices at 16", second)
	}
	if first := pictureLightStamps(nil, presentationWorld(t, false).spells(), 18, cell, 0); len(first) != 0 {
		t.Errorf("first game: identifier 18 stamps %+v, want none", first)
	}
}

// The draw site takes the World's set: Heal's 56 draws nothing itself in the
// second game; the first game draws its 56 as any record.
func TestTheHiddenCastTakesTheWorldsSet(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{true, false} {
		mw := presentationWorld(t, second)
		mw.world.ReleaseCast(sim.CastRecord{Picture: 56, Phases: 9, X: 300, Y: 300, AimX: 600, AimY: 300, Segments: 4})
		got := len(mw.savedProjectileScene(false))
		if want := map[bool]int{true: 0, false: 1}[second]; got != want {
			t.Errorf("second=%v: picture 56 drew %d sprites, want %d", second, got, want)
		}
	}
}

// spellTextFixture is an install whose spell.txt names row r "row r" on line
// r-1 for 29 rows and whose spells.txt names slot s "slot s#detail".
func spellTextFixture() installFixture {
	names, book := map[int]string{}, map[int]string{}
	for r := 1; r <= 29; r++ {
		names[r-1] = fmt.Sprintf("row %d", r)
	}
	for s := 0; s < 24; s++ {
		book[s] = fmt.Sprintf("slot %d#detail", s)
	}
	return installFixture{SpellNamesTextPath: textFile(29, names), SpellBookNamesTextPath: textFile(24, book)}
}

// The second game names all 29 rows from spell.txt and joins each spells.txt
// line to the row of its slot (R2-ENGINE-444); the first game names 28 rows
// and joins by its own slot table. A table shifted one slot must fail.
func TestEachGameNamesItsSpellsFromItsOwnTables(t *testing.T) {
	second := []uint16{1, 2, 3, 4, 24, 20, 21, 26, 13, 12, 11, 10, 5, 6, 7, 8, 25, 22, 23, 27, 19, 18, 17, 16}
	first := []uint16{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}
	shifted := append(append([]uint16{}, second[1:]...), second[0])
	check := func(w ui.Words, rows []uint16, named int) error {
		for r := 1; r < len(w.ItemSpellNames); r++ {
			want := ""
			if r <= named {
				want = fmt.Sprintf("row %d", r)
			}
			if w.ItemSpellNames[r] != want {
				return fmt.Errorf("item name of row %d = %q, want %q", r, w.ItemSpellNames[r], want)
			}
		}
		for s, r := range rows {
			if want := fmt.Sprintf("slot %d", s); w.SpellBookNames[r] != want {
				return fmt.Errorf("book name of row %d = %q, want %q", r, w.SpellBookNames[r], want)
			}
		}
		return nil
	}
	words := LoadInstallWords(spellTextFixture(), TextCode{})
	if err := check(words.Words(), first, 28); err != nil {
		t.Fatalf("no game named: %v", err)
	}
	words.Spells = data.FirstSpellPresentation()
	if err := check(words.Words(), first, 28); err != nil {
		t.Fatalf("first game: %v", err)
	}
	words.Spells = data.SpellPresentationOf(rules.SpellGameSecond)
	w := words.Words()
	if err := check(w, second, 29); err != nil {
		t.Fatalf("second game: %v", err)
	}
	if check(w, shifted, 29) == nil || check(w, first, 29) == nil || check(w, second, 28) == nil {
		t.Fatal("a shifted slot table, the first game's table or 28 named rows also passed")
	}
}

// The second game's item card keeps the spell its attack releases: the
// duration and ray lines follow the row's arm, its Stone Curse row 18 and its
// Prismatic Spray row 11, and the range is the spell's (DIV-1977). The first
// game's table, the loss control, composes the Weapon's stream instead.
func TestItemSpellLinesFollowTheArm(t *testing.T) {
	second := base.GameROM2.Edition()
	words := ui.AuthoredWords()
	lines := func(id int, name string, power int32, duration int32, edition *base.Edition) []string {
		table, weapon := staffTooltipTable(t, id, name, power, 6, 5, 15, duration)
		if edition != nil {
			table.Edition, table.SpellArms = edition, mapload.SecondGameSpellArms
		}
		item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{castSpellEffect(uint16(id), power)}}
		return itemInstanceInfoLines(item, table, words)
	}
	if got := lines(18, "Stone Curse", 0, 10, &second); !slices.Contains(got, fmt.Sprintf("%s %d", words.ItemRange, 6)) ||
		!slices.Contains(got, "Stone Curse 10 "+words.DurationUnit) {
		t.Fatalf("second-game Stone Curse card = %q", got)
	}
	if got := lines(11, "Prismatic Spray", 40, 0, &second); !slices.Contains(got, fmt.Sprintf("%s %d", words.ItemRays, 4)) ||
		!slices.Contains(got, fmt.Sprintf("%s %d", words.ItemRange, 6)) {
		t.Fatalf("second-game Prismatic Spray card = %q", got)
	}
	if got := lines(11, "Prismatic Spray", 40, 0, nil); slices.Contains(got, fmt.Sprintf("%s %d", words.ItemRays, 4)) ||
		!slices.Contains(got, words.ItemRange+": 5") {
		t.Fatalf("first-game row 11 card = %q", got)
	}
}

// The draw site takes the falling piece from the World's set
// (R2-ENGINE-447): the second game's Blizzard cell draws frame 8 raised
// 28-4f pixels while its frame f is below 8, then frame f-8. The first game's
// identifier 23 draws its frame where it stands: the loss control.
func TestTheFallingPieceTakesTheWorldsSet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		second             bool
		phase, frame, rise int
	}{
		{true, 0, 8, 28}, {true, 1, 8, 24}, {true, 7, 8, 0}, {true, 8, 0, 0}, {true, 15, 7, 0},
		{false, 0, 0, 0}, {false, 3, 3, 0}, {false, 8, 8, 0},
	} {
		mw := presentationWorld(t, tc.second)
		mw.world.ReleaseCast(sim.CastRecord{Picture: 23, Phases: 9, X: 300, Y: 300, AimX: 300, AimY: 300, Segments: 16})
		store := mw.world.SavedProjectiles()
		store.Items[0].Phase = int32(tc.phase)
		mw.world.SetSavedProjectiles(store)
		draws := mw.savedProjectileScene(false)
		if len(draws) != 1 {
			t.Fatalf("second=%v phase %d: %d draws, want one", tc.second, tc.phase, len(draws))
		}
		if d := draws[0]; d.Frame != tc.frame || d.Raise != tc.rise {
			t.Errorf("second=%v phase %d: frame %d raised %d, want frame %d raised %d", tc.second, tc.phase, d.Frame, d.Raise, tc.frame, tc.rise)
		}
	}
}
