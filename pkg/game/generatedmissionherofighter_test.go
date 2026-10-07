package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// heroExportedU4C exports mission's current save and returns the exported
// U4C (ClassSelector/fighter-mage flag byte) value of the party's own hero
// Human record. DecodeDocumentData's own object numbering is not the
// archive's ArchiveIndex (PartyWalk's own key, per the sibling test in
// generatedmissionsavfields_test.go), so this matches the hero by cell
// position the same way that test does: the live SelfSlot-owned entity's own
// cell against each Human record's own Block12 raw field.
func heroExportedU4C(t *testing.T, f *FrontEnd, party []mapload.PartyMember, mission int, label string) uint32 {
	t.Helper()
	app := f.App(label)
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	if f.live == nil || f.live.mission == nil || f.live.mission.state == nil {
		t.Fatal("mission state unavailable after OpenMission")
	}
	heroCell := map[[2]byte]bool{}
	for _, e := range f.live.mission.state.World.Entities() {
		if e.Owner == sim.SelfSlot {
			heroCell[[2]byte{byte(e.X), byte(e.Y)}] = true
		}
	}
	snapshot, snapLabel, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.NativeMissionTerrain = true
	raw, err := f.ExportCurrentSave(snapshot, snapLabel)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, obj := range doc.Objects {
		if obj.Class != "Human" {
			continue
		}
		var objPos []byte
		for _, r := range obj.Raw {
			if r.Name == "Block12" {
				objPos = r.Bytes
			}
		}
		if len(objPos) < 2 || !heroCell[[2]byte{objPos[0], objPos[1]}] {
			continue
		}
		v, err := savedStructureValue(&obj, "U4C")
		if err != nil {
			t.Fatalf("hero record has no U4C: %v", err)
		}
		return v
	}
	t.Fatal("no hero record found in exported document")
	return 0
}

// TestReleaseGeneratedMissionHeroFighterClassFlags asserts DIV-1389's fix:
// a generated mission SAV's own hero (MissionParty's Table-less repro of the
// "generated mission SAV" kit path) carries the fighter archetype's class
// flags, not the all-zero "cannot resolve a base row" fallback's silent
// non-fighter default. Owner-witnessed game9248/9249 (docs/1224/story.md)
// both carried U4C=6 (the non-fighter/mage bit HERO-HP-072 identifies) for a
// party opened with no chargen screen; a real corpus mission-10 hero sharing
// the same row (gameversions/saves) and the byte-corrected mission-10
// reference (review/owner-sav-exp0397-r5/candidates/game9237.sav) both
// carry U4C=0.
func TestReleaseGeneratedMissionHeroFighterClassFlags(t *testing.T) {
	for _, mission := range []int{10, 20} {
		t.Run(missionSubtestName(mission), func(t *testing.T) {
			f := releaseFront(t)
			party := MissionParty(nil, nil, nil)
			got := heroExportedU4C(t, f, party, mission, "generated mission hero fighter class flags")
			if got != 0 {
				t.Errorf("hero record U4C=%#x, want 0 (fighter archetype)", got)
			}
		})
	}
}

// TestReleaseChargenPartyFighterClassFlagsUnaffected confirms DIV-1389's fix
// changes nothing for the party every real player builds: ChargenParty
// always resolves a base row from the loaded install (ok is always true in
// chargenProfile), so its own arm is untouched by the fallback's now-
// archetype-matched Fighter default. A fighter chargen choice still exports
// U4C=0, exactly as before this story.
func TestReleaseChargenPartyFighterClassFlagsUnaffected(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "FighterProbe", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	got := heroExportedU4C(t, f, party, 10, "chargen party fighter class flags unaffected")
	if got != 0 {
		t.Errorf("hero record U4C=%#x, want 0 (fighter archetype)", got)
	}
}

// A native city save carries no Human/Unit record at all (it has no live
// mission World to export from currentworldbuild.go's actor walk), and its
// own party is always built by ChargenParty like
// TestReleaseChargenPartyFighterClassFlagsUnaffected above — the same
// always-ok branch DIV-1389 leaves untouched. There is no U4C field for a
// city save to regress.
