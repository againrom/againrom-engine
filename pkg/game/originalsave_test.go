package game

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// savedFile builds a whole original-game save in test code from the format
// contract. Nothing here reads an install.
func savedFile(mission uint32, actors []savedActor) []byte {
	b := savedBody(mission, actors)
	b = append(b, 0) // no world
	return savedContainer(savedTrailer(b))
}

func savedBody(mission uint32, actors []savedActor) []byte {
	var b []byte
	le32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	le16 := func(v uint16) { b = binary.LittleEndian.AppendUint16(b, v) }
	le32(0x50)
	le32(0x05)
	b = append(b, 6)
	b = append(b, "10.alm"...)
	for i := 0; i < 11; i++ {
		le32(0)
	}
	le32(mission)
	le32(2)
	le32(6)
	le32(1)
	le16(0xffff)
	le16(1)
	le16(uint16(len("Player")))
	b = append(b, "Player"...)
	b = append(b, 4)
	b = append(b, "Hero"...)
	le16(1)
	le32(1)
	b = append(b, 1, 2, 3, 4, 5, 6, 7, 8)
	b = append(b, 0x02)
	le32(0)
	le16(2)
	le32(700 ^ 0x5c073f4d)
	b = append(b, 1, 1) // outcome COMPLETE, +0x3d
	le32(0 ^ 0x5c073f4d)
	le32(0)
	le16(0)
	le16(0)
	le32(0)
	le32(0)
	le32(0)
	le32(1) // Player group
	b = append(b, make([]byte, 2+80+2)...)
	le32(uint32(len(actors))) // owner-graph actors
	for i, a := range actors {
		if i == 0 {
			le16(0xffff)
			le16(1)
			le16(4)
			b = append(b, "Unit"...)
		} else {
			le16(0x8003) // Player class/object are archive indices 1/2
		}
		le16(a.cell)
		le16(a.cell)
		b = append(b, a.fineX, a.fineY)
		le16(0x0777)
		le32(0x1234a020)
		le32(a.runtimeID)
		b = append(b, 0xaa, 0xbb, 0xcc)
		le32(uint32(a.mapUnitID))
		le16(0)
		le32(0)
		le32(uint32(1000 + i))
		le32(0)
		// Empty effects and two word lists; six opaque Unit spans; third
		// word list; store-A; held references; empty name; store-B; U68;
		// absent inventory/book; store-C. Widths from SAV-UNITLEN-045.
		start := len(b)
		b = append(b, make([]byte, 4+2+2+462+2+19+2+2+1+55+2+1+1+17)...)
		binary.LittleEndian.PutUint16(b[start+4+2+2+462+2+19+2+2+1+16:], 1)
		binary.LittleEndian.PutUint16(b[start+4+2+2+462+2+19+2+2+1+18:], 1)
	}
	b = append(b, make([]byte, 12+32+2+2+4)...)
	le32(0) // dead list
	return b
}

func savedTrailer(b []byte) []byte {
	b = binary.LittleEndian.AppendUint32(b, 0xbadface1)
	b = append(b, make([]byte, 4+400)...)
	if len(b)%2 != 0 {
		b = append(b, 0)
	}
	return b
}

func savedContainer(b []byte) []byte {
	blob := sav.Compress(b)
	out := make([]byte, 16)
	copy(out, sav.Magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(16+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], sav.MinVersion)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	out = append(out, make([]byte, 0x100)...)
	return append(out, "&YA1"...)
}

// savedFileWithSession adds a synthetic world-half tail to savedFile. The
// offsets are deliberately stated in the fixture's own units rather than read
// from production constants: 400 bytes of trigger results, then 1000 latches;
// the 50x50 diplomacy matrix begins at session+1856; the session is 4374 bytes.
func savedFileWithSession(t *testing.T, mission uint32, actors []savedActor,
	latch int, latchValue byte, from, to int, relation byte) []byte {
	t.Helper()
	body := savedBody(mission, actors)
	body = append(body, 1)                  // world present
	body = append(body, make([]byte, 8)...) // empty Buildings / SpellEffects

	// One block record whose static byte promises one matching 54-byte cell
	// record. That exact key-set agreement is the world-half discriminator.
	body = binary.LittleEndian.AppendUint16(body, 1)
	body = binary.LittleEndian.AppendUint32(body, uint32(0x0810)<<16|0x20)
	body = binary.LittleEndian.AppendUint16(body, 1)
	record := make([]byte, 54)
	binary.LittleEndian.PutUint16(record, 0x0810)
	body = append(body, record...)
	body = append(body, 0xde, 0xad, 0xbe, 0xef)
	session := make([]byte, 4374)
	if latch >= 0 && latch < 1000 {
		session[400+latch] = latchValue
	}
	if from >= 0 && from < 50 && to >= 0 && to < 50 {
		session[1856+from*50+to] = relation
	}
	body = append(body, session...)
	body = append(body, make([]byte, 4)...) // authoritative empty Sack list
	return savedContainer(savedTrailer(body))
}

type savedActor struct {
	cell         uint16
	fineX, fineY uint8
	runtimeID    uint32
	mapUnitID    uint16
}

// tenActors is a run long enough for the reader's own calibration to see one.
func tenActors() []savedActor {
	out := make([]savedActor, 10)
	for i := range out {
		out[i] = savedActor{
			cell:      uint16(0x0a00 + i*0x101),
			fineX:     0x80,
			fineY:     0x80,
			runtimeID: uint32(i + 1),
			mapUnitID: uint16(100 + i),
		}
	}
	return out
}

// resumeMap is a decoded map with units whose ids the save can join to. It is a
// struct literal rather than a parsed file: what is under test here is the
// transfer, not the map decoder.
func resumeMap() *alm.Map {
	m := &alm.Map{Width: 80, Height: 80}
	for i := 0; i < 4; i++ {
		m.Units = append(m.Units, alm.Unit{
			X:      uint32(5+i)<<8 | 0x80,
			Y:      uint32(5+i)<<8 | 0x80,
			UnitID: uint16(100 + i),
		})
	}
	// One unit the save says nothing about: it must stay where the map puts
	// it, because a save with no word about a unit is not a save that moved
	// it to the origin.
	m.Units = append(m.Units, alm.Unit{X: 0x1e80, Y: 0x2280, UnitID: 900})
	return m
}

func TestApplyPositionsTransfersTheSavedCellAndFine(t *testing.T) {
	f, err := sav.Open(savedFile(10, tenActors()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	m := resumeMap()
	var r OriginalSaveResume
	applyOriginalPositions(m, f, &r)
	if r.Joined != 4 || r.Moved != 4 {
		t.Fatalf("joined %d moved %d, want 4/4", r.Joined, r.Moved)
	}
	// The head's packing IS the map record's fixed point: cell in the high
	// byte, fine in the low. A transfer that needed arithmetic would be
	// visible here as a wrong cell.
	for i := 0; i < 4; i++ {
		wantCol := uint32(0x0a00+i*0x101) & 0xff
		wantRow := uint32(0x0a00+i*0x101) >> 8
		if m.Units[i].X != wantCol<<8|0x80 || m.Units[i].Y != wantRow<<8|0x80 {
			t.Fatalf("unit %d placed at %#x,%#x", i, m.Units[i].X, m.Units[i].Y)
		}
	}
	if m.Units[4].X != 0x1e80 || m.Units[4].Y != 0x2280 {
		t.Fatal("a unit the save says nothing about was moved")
	}
}

// TestASavedPositionEQUALtoTheMapIsNotCountedAsMoved: the count is what a caller
// reads to see whether the resume did anything, so it must mean what it says.
func TestASavedPositionEqualToTheMapIsNotCountedAsMoved(t *testing.T) {
	actors := tenActors()
	for i := 0; i < 4; i++ {
		actors[i].cell = uint16(5+i)<<8 | uint16(5+i)
	}
	f, err := sav.Open(savedFile(10, actors))
	if err != nil {
		t.Fatal(err)
	}
	m := resumeMap()
	var r OriginalSaveResume
	applyOriginalPositions(m, f, &r)
	if r.Joined != 4 || r.Moved != 0 {
		t.Fatalf("joined %d moved %d, want 4/0", r.Joined, r.Moved)
	}
}

// TestADeadSavedObjectIsNotResumed: a dead object keeps its head and its unit
// id, so a join that did not skip it would put a corpse where a living unit is.
func TestADeadSavedObjectIsNotResumed(t *testing.T) {
	actors := tenActors()
	actors[1].runtimeID = 0
	f, err := sav.Open(savedFile(10, actors))
	if err != nil {
		t.Fatal(err)
	}
	m := resumeMap()
	before := m.Units[1].X
	var r OriginalSaveResume
	applyOriginalPositions(m, f, &r)
	if r.Joined != 3 {
		t.Fatalf("joined %d, want 3", r.Joined)
	}
	if m.Units[1].X != before {
		t.Fatal("a dead object was resumed into the map")
	}
}

func TestPlayerOriginalSaveRestoresSessionBeforeAnnouncerAndFirstTick(t *testing.T) {
	// This fixture isolates session state from actor materialization.
	saved := savedFileWithSession(t, 10, nil, 7, 1, 3, 4, 1)
	src := missionSource{"scenario/10.alm": synth.ALM(synth.ALMOptions{Width: 40, Height: 40})}
	ms, r, err := loadOriginalMission(originalMissionFixture(src), saved)
	if err != nil {
		t.Fatalf("RestoreOriginal: %v", err)
	}
	if ms.World.Tick() != rawSavedSubTick1112(t, saved) {
		t.Fatalf("resumed world already advanced to tick %d", ms.World.Tick())
	}
	if !r.SessionApplied || r.Latches != 1 || r.DiplomacyCells != 2500 {
		t.Fatalf("resume report session = applied %v, latches %d, diplomacy %d",
			r.SessionApplied, r.Latches, r.DiplomacyCells)
	}
	if !ms.World.ScriptLatched(7) || !ms.World.Relations().Hostile(3, 4) ||
		ms.World.Relations().Hostile(4, 3) {
		t.Fatalf("world session = latch %v, relation %#x/%#x",
			ms.World.ScriptLatched(7), ms.World.Relations().Byte(3, 4),
			ms.World.Relations().Byte(4, 3))
	}

	// NewAnnouncer seeds from the already imported latch. Its first sample and
	// the first gameplay tick therefore cannot present an old message again.
	a := NewAnnouncer(ms.World, []mapload.ScriptRaise{{Latch: 7, Event: 99}})
	if got := a.Sample(ms.World); got != nil {
		t.Fatalf("NewAnnouncer re-raised the saved latch immediately: %v", got)
	}
	sim.Step(ms.World, nil)
	if got := a.Sample(ms.World); got != nil {
		t.Fatalf("first gameplay tick re-raised the saved latch: %v", got)
	}
}

func TestPlayerOriginalSaveCarriesRawSessionSpansThroughActorStockStaging(t *testing.T) {
	saved := savedFileWithSession(t, 10, nil, -1, 0, -1, 0, 0)
	f, err := sav.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	var head [48]byte
	head[0], head[47] = 0xab, 0xcd
	if err := f.SetRawHead(head); err != nil {
		t.Fatal(err)
	}
	var mid [400]byte
	mid[0], mid[399] = 0xef, 0x12
	if err := f.SetRawMid(mid); err != nil {
		t.Fatal(err)
	}
	saved = f.Marshal()

	src := missionSource{"scenario/10.alm": synth.ALM(synth.ALMOptions{Width: 40, Height: 40})}
	ms, r, err := loadOriginalMission(originalMissionFixture(src), saved)
	if err != nil {
		t.Fatalf("RestoreOriginal: %v", err)
	}
	if !r.SessionApplied {
		t.Fatal("session was not applied")
	}
	if got := ms.World.RawSessionHead(); got != head {
		t.Fatalf("RawSessionHead after resume = %x, want %x (restoreOriginalActorStock's staging must carry it across)", got, head)
	}
	if got := ms.World.RawSessionMid(); got != mid {
		t.Fatalf("RawSessionMid after resume = %x, want %x (restoreOriginalActorStock's staging must carry it across)", got, mid)
	}
}

func TestPlayerOriginalSaveRejectsAnInvalidLatchBeforeOpeningTheMap(t *testing.T) {
	saved := savedFileWithSession(t, 10, tenActors(), 999, 2, 3, 4, 1)
	_, r, err := loadOriginalMission(originalMissionFixture(missionSource{}), saved)
	if err == nil || !strings.Contains(err.Error(), "latch 999 is 2") {
		t.Fatalf("RestoreOriginal error = %v, want invalid latch refusal", err)
	}
	if r.SessionApplied {
		t.Fatal("a refused original session was reported as applied")
	}
}

func TestFrontEndRejectsAnInvalidOriginalSessionBeforeResettingTheLiveGame(t *testing.T) {
	saved := savedFileWithSession(t, 10, tenActors(), 999, 2, 3, 4, 1)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{{Name: "the current game's hero", Class: 7}}}}
	f.Town.gold = 321
	beforeTown := f.Town

	open, town, err := f.RestoreOriginal(saved)
	if err == nil || !strings.Contains(err.Error(), "latch 999 is 2") {
		t.Fatalf("RestoreOriginal error = %v, want invalid latch refusal", err)
	}
	if open != nil || town {
		t.Fatalf("refused load returned opener %v, town %v", open != nil, town)
	}
	if f.Town != beforeTown || f.Town.Gold() != 321 || len(f.Carried) != 1 ||
		f.Carried[0].Name != "the current game's hero" || f.Carried[0].Class != 7 {
		t.Fatalf("refused load changed live game: town changed=%v gold=%d party=%+v",
			f.Town != beforeTown, f.Town.Gold(), f.Carried)
	}
}

func TestFrontEndLoadsABetweenMissionOriginalSaveIntoTown(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{{Body: "existing"}}}}
	open, town, err := f.RestoreOriginal(savedFile(0, tenActors()))
	if err != nil {
		t.Fatalf("RestoreOriginal: %v", err)
	}
	if open != nil || !town {
		t.Fatalf("load returned opener %v town=%v, want nil/true", open != nil, town)
	}
	if f.Town == nil || !f.Town.Open() {
		t.Fatal("the restored town is not open")
	}
	if got := f.Town.Gold(); got != 700 {
		t.Errorf("town gold = %d, want the Player record's 700", got)
	}
	if len(f.Carried) == 0 {
		t.Fatal("the restored town lost the party")
	}
}

func TestClearRestoredMissionPositionDropsOnlyThePosition(t *testing.T) {
	party := []mapload.PartyMember{
		{Name: "Danath", Saved: &mapload.Saved{
			Cell: mapload.Cell{X: 16, Y: 12}, MapUnitID: 136,
			HP: 107, MaxHP: 131, Mana: 30, MaxMana: 90,
			HealthRegenPeriod: 100, ManaRegenPeriod: 50,
		}},
		{Name: "Reniesta", Saved: &mapload.Saved{
			Cell: mapload.Cell{X: 16, Y: 11}, HP: 22, MaxHP: 22,
		}},
		{Name: "Generated"},
	}
	clearRestoredMissionPosition(party)

	for i, want := range []mapload.Saved{
		{HP: 107, MaxHP: 131, Mana: 30, MaxMana: 90, HealthRegenPeriod: 100, ManaRegenPeriod: 50},
		{HP: 22, MaxHP: 22},
	} {
		if got := *party[i].Saved; got != want {
			t.Errorf("member %d kept %+v, want the pools alone: %+v", i, got, want)
		}
	}
	if party[2].Saved != nil {
		t.Error("a member who carried no Saved was given one")
	}
}

func TestResumeRefusesWhatItCannotOpen(t *testing.T) {
	if _, _, err := loadOriginalMission(originalMissionFixture(missionSource{}), []byte("not a save")); err == nil {
		t.Fatal("a file that is not a save was resumed")
	}
	// A save naming a mission number no campaign map answers to.
	if _, _, err := loadOriginalMission(originalMissionFixture(missionSource{}), savedFile(999, tenActors())); err == nil {
		t.Fatal("a save naming no campaign mission was resumed")
	}
	// And a mission whose map the archive does not hold.
	if _, _, err := loadOriginalMission(originalMissionFixture(missionSource{}), savedFile(10, tenActors())); err == nil {
		t.Fatal("a mission whose map is absent was resumed")
	}
}

// TestTheReportNamesWhatIsNOTCarried. The divergence is the point of the report:
// a resume that printed only what it did carry would be one whose divergence a
// reader has to know to look for.
func TestTheReportNamesWhatIsNotCarried(t *testing.T) {
	r := OriginalSaveResume{Mission: 10, MapName: "10.alm", Label: "x", HasWorld: true,
		Heads: 9, Dead: 1, Joined: 4, Moved: 2, BlockRecords: 3, Latches: 7,
		DiplomacyCells: 2500, SessionApplied: true, Diaries: 5, DiariesApplied: true}
	s := r.String()
	for _, want := range []string{"NOT CARRIED",
		"18 bytes of still-unpromoted session gaps", "MOVED 2", "blocks 3", "latches 7 set",
		"diplomacy 2500 cells", "RESTORED (100 registers, 448 raw bytes)",
		"WIN/LOSE 0/0 and Player outcome 0 RESTORED", "diaries 5 carried"} {
		if !strings.Contains(s, want) {
			t.Fatalf("the report does not mention %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "trigger result registers") {
		t.Fatalf("the report still calls restored trigger result registers not carried:\n%s", s)
	}
	if strings.Contains(s, "trigger latches, which") {
		t.Fatalf("the report still calls restored trigger latches not carried:\n%s", s)
	}
	if strings.Contains(s, "the journal") {
		t.Fatalf("the report still calls the carried journal not carried:\n%s", s)
	}
	// And it names the party, which is the axis whose silence cost the owner a
	// save: every other count in this report was already right.
	if !strings.Contains(s, "party:") {
		t.Fatalf("the report does not name the party at all:\n%s", s)
	}
	if !strings.Contains(OriginalSaveResume{}.String(), "NO WORLD HALF") {
		t.Fatal("a save with no world half is not said to have none")
	}
}

// TestAFallbackPartyIsSaidOutLoud. A save whose object walk reaches no character
// opens with the fresh party, and the one thing that must not happen is that it
// does so silently — that is exactly the substitution this story exists to
// undo.
func TestAFallbackPartyIsSaidOutLoud(t *testing.T) {
	s := RestoredParty{Fallback: true, Err: errNoWalk}.String()
	for _, want := range []string{"NO CHARACTER RESTORED", "fresh party"} {
		if !strings.Contains(s, want) {
			t.Fatalf("the fallback is not stated: %q", s)
		}
	}
}

var errNoWalk = errors.New("the walk read nothing")

func TestOriginalSaveLabelCarriesNoSourceMarker(t *testing.T) {
	for _, tc := range []struct {
		mission uint32
		suffix  string
	}{{7, " - mission 7"}, {0, " - between missions"}} {
		saved := savedFile(tc.mission, tenActors())
		sf, err := sav.Open(saved)
		if err != nil {
			t.Fatal(err)
		}
		want := asciiLabel(sf.Label, 0)
		if want == "" {
			want = "(no label)"
		}
		want += tc.suffix
		got, err := OriginalSaveLabel(saved, 0)
		if err != nil || got != want {
			t.Fatalf("mission %d: OriginalSaveLabel = %q, %v; want %q", tc.mission, got, err, want)
		}
	}
}
