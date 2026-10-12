package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondSpellShowRecord is one armed record: its identifier and phase and
// the sheet frame and raise the set draws it with.
type secondSpellShowRecord struct {
	Picture      int32
	Phase        int32
	Drawn, Raise int
}

// secondSpellShowTick is one update of a cast as the map draws and voices it.
type secondSpellShowTick struct {
	Records []secondSpellShowRecord
	Draws   []ui.SpellBolt
	Sounds  []int
}

// secondSpellShow casts row on the first campaign mission and records ticks
// updates of armed records, their draws and the sound slots queued. The cast
// goes through the World's cast command, at the cell four columns east for a
// cell spell and at the nearest admitted actor otherwise.
func secondSpellShow(t *testing.T, row uint16, atCell bool, ticks int) []secondSpellShowTick {
	t.Helper()
	f, _ := secondCheatMission(t, true, false)
	mw := f.live
	// The mission's one mage stands where its owner sees nobody; a copy owned
	// by the player stands three cells east of the player's first unit and
	// casts at the nearest actor it admits.
	lead, _ := mw.entity(mw.mission.ids[0])
	var mage sim.Entity
	for _, e := range mw.world.Entities() {
		if e.Alive() && e.MaxMana > mage.MaxMana {
			mage = e
		}
	}
	mage.Owner = lead.Owner
	hero, err := mw.world.CheatSummon(mage, nil, [sim.EquipSlots]sim.ItemInstance{}, lead.X+3, lead.Y)
	if err != nil || !mw.world.CheatSpell(hero, row) {
		t.Fatalf("row %d: no mage to cast it: %v", row, err)
	}
	h, _ := mw.entity(hero)
	victim, near := sim.EntityID(0), int32(-1)
	for _, e := range mw.world.Entities() {
		d := max(e.X-h.X, h.X-e.X) + max(e.Y-h.Y, h.Y-e.Y)
		if e.ID != hero && e.Alive() && d >= 2 && (near < 0 || d < near) && (atCell || mw.world.BookSpellRefusal(hero, e.ID, uint32(row)) == "") {
			victim, near = e.ID, d
		}
	}
	if near < 0 {
		t.Fatalf("row %d: the mage admits the spell at no actor", row)
	}
	var sounds []int
	mw.semanticSound = func(_ string, slot int, _ uint32, _ image.Point, _ image.Point) { sounds = append(sounds, slot) }
	if atCell {
		mw.pending = append(mw.pending, sim.CastAt(hero, sim.SpellID(row), sim.CellPoint{X: h.X + 4, Y: h.Y}))
	} else {
		mw.pending = append(mw.pending, sim.Cast(hero, victim, sim.SpellID(row)))
	}
	set := mw.spells()
	var out []secondSpellShowTick
	for range ticks {
		sounds = nil
		mw.tick()
		var tick secondSpellShowTick
		records, _ := mw.armedRecords()
		for _, p := range records {
			drawn, raise := set.DrawnFrame(p.Picture, int(p.Phase))
			tick.Records = append(tick.Records, secondSpellShowRecord{p.Picture, p.Phase, drawn, raise})
		}
		tick.Draws = mw.savedProjectileScene(false)
		tick.Sounds = slices.Clone(sounds)
		out = append(out, tick)
	}
	return out
}

// secondSpellShowArmed counts the updates a record of picture is armed and
// those of them that draw anything.
func secondSpellShowArmed(ticks []secondSpellShowTick, picture int32) (armed, drawn int) {
	for _, tick := range ticks {
		if slices.ContainsFunc(tick.Records, func(r secondSpellShowRecord) bool { return r.Picture == picture }) {
			armed++
			if len(tick.Draws) != 0 {
				drawn++
			}
		}
	}
	return armed, drawn
}

func secondSpellShowSounds(ticks []secondSpellShowTick) []int {
	var out []int
	for _, tick := range ticks {
		out = append(out, tick.Sounds...)
	}
	return out
}

// secondSpellShowSheet is a contact sheet of the draws: one 96-pixel column
// per update, each frame placed by its sheet halves and raised by its Raise.
func secondSpellShowSheet(ticks []secondSpellShowTick) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 96*len(ticks), 160))
	for i := range pic.Pix {
		pic.Pix[i] = 32
		if i%4 == 3 {
			pic.Pix[i] = 255
		}
	}
	for col, tick := range ticks {
		for x := 0; x < 96; x += 8 {
			pic.SetRGBA(col*96+x, 120, color.RGBA{128, 128, 128, 255})
		}
		for _, d := range tick.Draws {
			if d.Sheet == nil {
				continue
			}
			frame := d.Sheet.Frame(d.Frame)
			if frame == nil {
				continue
			}
			x0, y0 := col*96+48-d.Sheet.CenterX, 120-d.Sheet.CenterY-d.Raise
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					c := frame.Pixels[y*frame.Width+x]
					if c.A == 0 {
						continue
					}
					sx := x
					if d.Mirror {
						sx = frame.Width - 1 - x
					}
					if p := image.Pt(x0+sx, y0+y); p.In(image.Rect(col*96, 0, col*96+96, 160)) {
						pic.SetRGBA(p.X, p.Y, c)
					}
				}
			}
		}
	}
	return pic
}

// secondSpellShowWrite keeps the witness of one spell under the review
// directory AGAINROM_SPELL_SHOW_DIR names: the records and sounds of each
// update as text, with whether sfx.reg holds each slot, and the contact
// sheet. Nothing is written without it.
func secondSpellShowWrite(t *testing.T, name string, ticks []secondSpellShowTick, bank *SoundBank) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SPELL_SHOW_DIR")
	if dir == "" {
		return
	}
	dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for i, tick := range ticks {
		fmt.Fprintf(&b, "update %2d draws %d;", i+1, len(tick.Draws))
		for _, r := range tick.Records {
			fmt.Fprintf(&b, " id %d phase %d frame %d raise %d;", r.Picture, r.Phase, r.Drawn, r.Raise)
		}
		for _, slot := range tick.Sounds {
			_, held := bank.entries[slot]
			fmt.Fprintf(&b, " sound %d in sfx.reg %v;", slot, held)
		}
		b.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, name+".txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, secondSpellShowSheet(ticks)); err != nil {
		t.Fatal(err)
	}
}

// On the first campaign mission a mage's casts draw and voice the second
// game's own identifiers (R2-ENGINE-445..447): row r arms identifier 2r+8
// and queues slot 500+2r+8; a typed identifier draws; Blizzard's cells are
// identifier 23, drawn as frame 8 raised 28-4f while f < 8, then frame f-8,
// and queue slot 523 with frame 8, which sfx.reg holds, while its cast
// slot 522 is not held and so stays silent; row 12's identifiers 32 and 33
// have no projectiles.reg type and draw nothing. Loss control: the identifier the
// first game's arm of the same row arms (Lightning 34, Prismatic Spray 36,
// Blizzard cells 51) is never armed.
func TestReleaseSecondGameSpellsDrawAndSoundTheirOwn(t *testing.T) {
	bank := OpenSounds(os.Getenv("AGAINROM_ASSETS"))
	for _, tc := range []struct {
		name  string
		row   uint16
		ticks int
		cast  int32
		lost  int32
	}{
		{"fire-arrow", 1, 40, 10, 0},
		{"ice-missile", 5, 40, 18, 0},
		{"lightning", 10, 40, 28, 34},
		{"prismatic-spray", 11, 40, 30, 36},
		{"blizzard", 7, 60, 23, 51},
		{"untyped-row-12", 12, 40, 0, 0},
	} {
		ticks := secondSpellShow(t, tc.row, tc.row == 7, tc.ticks)
		if bank == nil {
			t.Fatal("the second game's sound bank does not open")
		}
		secondSpellShowWrite(t, tc.name, ticks, bank)
		armed, drawn := secondSpellShowArmed(ticks, tc.cast)
		if tc.cast != 0 && (armed == 0 || drawn == 0) {
			t.Errorf("%s: identifier %d armed %d updates, drew in %d", tc.name, tc.cast, armed, drawn)
		}
		if tc.cast == 0 && slices.ContainsFunc(ticks, func(k secondSpellShowTick) bool { return len(k.Draws) != 0 }) {
			t.Errorf("%s: an untyped row drew", tc.name)
		}
		if lost, _ := secondSpellShowArmed(ticks, tc.lost); tc.lost != 0 && lost != 0 {
			t.Errorf("%s: the first game's identifier %d armed", tc.name, tc.lost)
		}
		sounds := secondSpellShowSounds(ticks)
		if want := 500 + 2*int(tc.row) + 8; !slices.Contains(sounds, want) {
			t.Errorf("%s: sounds %v lack the cast slot %d", tc.name, sounds, want)
		}
		if tc.row == 7 {
			secondSpellShowFall(t, ticks, sounds, bank)
		}
	}
}

// secondSpellShowFall checks every Blizzard cell record against the falling
// piece and the cell sound.
func secondSpellShowFall(t *testing.T, ticks []secondSpellShowTick, sounds []int, bank *SoundBank) {
	t.Helper()
	falling, shattering := 0, 0
	for _, tick := range ticks {
		for _, r := range tick.Records {
			if r.Picture != 23 {
				continue
			}
			wantDrawn, wantRaise := int(r.Phase)-8, 0
			if r.Phase < 8 {
				wantDrawn, wantRaise = 8, 28-4*int(r.Phase)
				falling++
			} else {
				shattering++
			}
			if r.Drawn != wantDrawn || r.Raise != wantRaise {
				t.Errorf("Blizzard cell at phase %d draws frame %d raised %d", r.Phase, r.Drawn, r.Raise)
			}
		}
		// R2-ENGINE-446: a cell voices 523 on the update its piece shows
		// frame 8, so every update that queues 523 holds a phase-8 record.
		if slices.Contains(tick.Sounds, 523) && !slices.ContainsFunc(tick.Records,
			func(r secondSpellShowRecord) bool { return r.Picture == 23 && r.Phase == 8 }) {
			t.Errorf("Blizzard voiced 523 with no cell at frame 8: %+v", tick.Records)
		}
		for _, d := range tick.Draws {
			if d.Raise < 0 || d.Raise > 28 {
				t.Errorf("a draw raised %d", d.Raise)
			}
		}
	}
	if falling == 0 || shattering == 0 {
		t.Errorf("Blizzard cells: %d falling and %d shattering records", falling, shattering)
	}
	_, held523 := bank.entries[523]
	_, held522 := bank.entries[522]
	if !slices.Contains(sounds, 523) || !held523 || held522 {
		t.Errorf("Blizzard sounds %v; sfx.reg holds 523 %v, 522 %v", sounds, held523, held522)
	}
}
