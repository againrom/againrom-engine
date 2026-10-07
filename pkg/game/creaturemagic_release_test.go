package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const magicWitnessMission = 51

// magicWitnessOpen opens the mission through the player's own opener and
// returns the session and the one hostile creature that holds class spell
// slots.
func magicWitnessOpen(t *testing.T) (*FrontEnd, *ui.App, sim.Entity) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "SlotProbe", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("creature magic")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := app.OpenMission(f.missionOpener(magicWitnessMission, party, nil, nil, nil)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 8; k++ {
		if _, _, open := f.LiveNotice(); !open {
			break
		}
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	var holders []sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.CreatureSpells[0].ID != 0 && e.Owner != sim.SelfSlot {
			holders = append(holders, e)
		}
	}
	if len(holders) != 1 {
		t.Fatalf("mission %d places %d creatures holding class spell slots, want the one ogre turtle", magicWitnessMission, len(holders))
	}
	return f, app, holders[0]
}

// magicWitnessSaved reads the creature's record out of a SAV and reports its
// three slots and its experience value (the record's +0x1c).
func magicWitnessSaved(t *testing.T, raw []byte, unit uint16) ([sim.CreatureSpellSlots]sim.CreatureSpell, uint32) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Objects {
		if r.Class != "Unit" || savedRecordValueForTest(t, r, "T08") != uint32(unit) {
			continue
		}
		var slots [sim.CreatureSpellSlots]sim.CreatureSpell
		order := savedRecordRawForTest(t, r, "U158")
		for i := range slots {
			slots[i] = sim.CreatureSpell{
				ID:        binary.LittleEndian.Uint32(order[sav.CreatureSpellIDOffset+4*i:]),
				Threshold: binary.LittleEndian.Uint32(order[sav.CreatureSpellThresholdOffset+4*i:]),
			}
		}
		return slots, savedRecordValueForTest(t, r, "T1C")
	}
	t.Fatalf("the SAV holds no Unit record for map unit %d", unit)
	return [sim.CreatureSpellSlots]sim.CreatureSpell{}, 0
}

// magicWitnessZero returns the SAV with the one occurrence of pattern in its
// body overwritten by zeros.
func magicWitnessZero(t *testing.T, raw, pattern []byte, name string) []byte {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(file.Body, pattern); n != 1 {
		t.Fatalf("%s pattern occurs %d times in the SAV body, want 1", name, n)
	}
	copy(file.Body[bytes.Index(file.Body, pattern):], make([]byte, len(pattern)))
	return file.Marshal()
}

// magicWitnessCaption reports whether the information card of entity id paints
// the installed spellcaster caption.
func magicWitnessCaption(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID) bool {
	t.Helper()
	live := f.live
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	for _, e := range live.world.Entities() {
		if e.ID == id {
			inspectionCentre(live, int(e.X), int(e.Y))
		}
	}
	live.push()
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	subject, ok := live.view.InspectionPanel()
	if !ok || subject.ID != uint32(id) {
		t.Fatalf("panel subject %d/%v, want %d", subject.ID, ok, id)
	}
	caption := f.Words.PanelCaptions[190]
	if caption == "" {
		t.Fatal("the install states no spellcaster caption")
	}
	for _, row := range ui.CharacterPanelReport(ui.CompactPanelLayout(nil), f.tipFont(), subject) {
		if row.Label == caption {
			return true
		}
	}
	return false
}

// magicWitnessPlain is a hostile creature of the mission with no spell slots
// and no book.
func magicWitnessPlain(t *testing.T, f *FrontEnd) sim.EntityID {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.Owner != sim.SelfSlot && e.Owner != 0 && e.Alive() && e.TypeID >= 0x1a && e.XPValue != 0 &&
			e.KnownSpells == 0 && e.CreatureSpells == [sim.CreatureSpellSlots]sim.CreatureSpell{} {
			return e.ID
		}
	}
	t.Fatal("the mission holds no plain creature")
	return 0
}

// TestReleaseMission51OgreTurtleKeepsItsMagic opens mission 51 and follows the
// ogre turtle's magic through the information card, a constructed fight, the
// SAVE and a cold LOAD. The card's caption and the hover list read the install
// words; the SAVE carries the three class slots and the experience value the
// original's caption tests; the loss controls remove each in turn.
func TestReleaseMission51OgreTurtleKeepsItsMagic(t *testing.T) {
	f, app, turtle := magicWitnessOpen(t)
	live := f.live
	if turtle.XPValue == 0 || !turtle.Book.HasInstances() || turtle.KnownSpells&(1<<turtle.CreatureSpells[0].ID) == 0 {
		t.Fatalf("turtle %d holds slots %v without its book or experience value: known %#x xp %d",
			turtle.ID, turtle.CreatureSpells, turtle.KnownSpells, turtle.XPValue)
	}

	// The card: the installed spellcaster caption paints and the hover list is
	// the installed heading and the spell's installed name.
	plain := magicWitnessPlain(t, f)
	if !magicWitnessCaption(t, f, app, turtle.ID) {
		t.Fatalf("the turtle's card does not paint the installed caption %q", f.Words.PanelCaptions[190])
	}
	if magicWitnessCaption(t, f, app, plain) {
		t.Fatalf("the card of plain creature %d paints the spellcaster caption", plain)
	}
	magicWitnessCaption(t, f, app, turtle.ID)
	subject, _ := live.view.InspectionPanel()
	lines, ok := ui.MonsterSpellHint(f.Words, subject.KnownSpells)
	name := f.Words.ItemSpellNames[turtle.CreatureSpells[0].ID]
	if !ok || len(lines) != 1 || !strings.HasPrefix(lines[0], f.Words.Hover[192]) || !strings.Contains(lines[0], name) || name == "" {
		t.Fatalf("hover list %q/%v lacks the install heading %q and spell %q", lines, ok, f.Words.Hover[192], name)
	}

	// The fight: with the party beside it the turtle casts its class spell
	// inside a bounded window; with the slots cleared it never does.
	m := releaseMissionMap(t, f, magicWitnessMission)
	ms, err := StartMissionFrom(m, "witness.alm", magicWitnessMission, f.Table, mapload.DifficultyNormal, creatureWitnessParty())
	if err != nil {
		t.Fatal(err)
	}
	hero := ms.Start.IDs[0]
	x, y, ok := creatureFreeCellNear(ms.World, mapload.Planes(m, f.Table), m.Width, m.Height, turtle.X, turtle.Y)
	if !ok {
		t.Fatal("no free cell beside the turtle")
	}
	const window = 1500
	castFrom := func(edit ...func(*sim.Entity)) (int, uint16) {
		w := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y, edit...)
		for tick := 0; tick < window; tick++ {
			for _, ev := range sim.StepObserved(w, nil) {
				if ev.Caster == turtle.ID {
					return tick, ev.Spell
				}
			}
		}
		return -1, 0
	}
	at, spell := castFrom()
	if at < 0 || uint32(spell) != turtle.CreatureSpells[0].ID {
		t.Fatalf("the turtle cast spell %d at tick %d, want its slot spell %d inside %d ticks", spell, at, turtle.CreatureSpells[0].ID, window)
	}
	if got, _ := castFrom(func(e *sim.Entity) {
		if e.ID == turtle.ID {
			e.CreatureSpells = [sim.CreatureSpellSlots]sim.CreatureSpell{}
		}
	}); got >= 0 {
		t.Fatalf("the turtle cast at tick %d with its slots cleared", got)
	}
	t.Logf("the turtle cast spell %d at tick %d of %d", spell, at, window)

	// SAVE: the record carries the slots and the experience value.
	dir, saved := castOrderF2Save(t, f, app, "magic")
	slots, xp := magicWitnessSaved(t, saved, turtle.MapUnitID)
	if slots != turtle.CreatureSpells || xp != uint32(turtle.XPValue) {
		t.Fatalf("SAVE wrote slots %v and experience %d, want %v and %d", slots, xp, turtle.CreatureSpells, turtle.XPValue)
	}

	// Loss controls on the file: a zeroed slot window and a zeroed experience
	// value each fail the same reader.
	slotPattern := make([]byte, 24)
	for i, s := range turtle.CreatureSpells {
		binary.LittleEndian.PutUint32(slotPattern[4*i:], s.ID)
		binary.LittleEndian.PutUint32(slotPattern[12+4*i:], s.Threshold)
	}
	xpPattern := binary.LittleEndian.AppendUint32(nil, uint32(turtle.XPValue))
	for name, pattern := range map[string][]byte{"slot window": slotPattern, "experience value": xpPattern} {
		broken := magicWitnessZero(t, saved, pattern, name)
		if bs, bx := magicWitnessSaved(t, broken, turtle.MapUnitID); bs == turtle.CreatureSpells && bx == uint32(turtle.XPValue) {
			t.Fatalf("zeroing the %s left the witness reading the original values", name)
		}
	}

	// Cold LOAD keeps the slots and the next SAVE keeps the experience value.
	g, gapp := castOrderSession(t, dir)
	var loaded sim.Entity
	for _, e := range g.live.world.Entities() {
		if e.MapUnitID == turtle.MapUnitID {
			loaded = e
		}
	}
	if loaded.CreatureSpells != turtle.CreatureSpells || loaded.KnownSpells != turtle.KnownSpells || loaded.XPValue != turtle.XPValue {
		t.Fatalf("cold LOAD holds slots %v known %#x xp %d, want %v %#x %d", loaded.CreatureSpells, loaded.KnownSpells,
			loaded.XPValue, turtle.CreatureSpells, turtle.KnownSpells, turtle.XPValue)
	}
	if !magicWitnessCaption(t, g, gapp, turtle.ID) || magicWitnessCaption(t, g, gapp, magicWitnessPlain(t, g)) {
		t.Fatal("after cold LOAD the turtle's card lacks the caption or a plain creature's card has it")
	}
	_, resaved := castOrderF2Save(t, g, gapp, "magic again")
	if rs, rx := magicWitnessSaved(t, resaved, turtle.MapUnitID); rs != turtle.CreatureSpells || rx != uint32(turtle.XPValue) {
		t.Fatalf("the resave wrote slots %v and experience %d", rs, rx)
	}

	// A SAV whose slot window is zero (an engine SAVE written before the slots
	// existed) loads with the class slots the creature is built with.
	zeroDir := t.TempDir()
	zeroed := magicWitnessZero(t, saved, slotPattern, "slot window")
	if err := os.WriteFile(filepath.Join(zeroDir, "magic.sav"), zeroed, 0o600); err != nil {
		t.Fatal(err)
	}
	h, _ := castOrderSession(t, zeroDir)
	for _, e := range h.live.world.Entities() {
		if e.MapUnitID == turtle.MapUnitID && e.CreatureSpells != turtle.CreatureSpells {
			t.Fatalf("LOAD of a zero slot window holds %v, want the class slots %v", e.CreatureSpells, turtle.CreatureSpells)
		}
	}
}
