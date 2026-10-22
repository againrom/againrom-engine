package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// moverStartRecords SAVEs the mission the front end holds and returns each
// map-placed actor's SAV record by its map unit.
func moverStartRecords(t *testing.T, f *FrontEnd, label string) map[uint16][]byte {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return moverRecordsByMapUnit(doc)
}

func moverRecordsByMapUnit(doc sav.DocumentData) map[uint16][]byte {
	out := map[uint16][]byte{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		unit, err := savedStructureValue(r, "T08")
		if err != nil || unit == 0 {
			continue
		}
		if mover, err := savedActorRaw(r, "U154", 180); err == nil {
			out[uint16(unit)] = mover
		}
	}
	return out
}

// TestReleaseEveryMissionStartHoldsItsConstructedMovers opens every mission
// the install ships and SAVEs it at once. Each map-placed actor's Mover holds
// the constructor's drawn facing at +0x00 and +0x01 (0x40..0xC0), the mask of
// its domain, 5 at +0x08 for every actor and 0xff at +0x09 for another
// player's actor, and the step cost the World derives (MOVE-119..MOVE-121,
// AI-692). Loss control: a mission
// with two or more placed actors holds two or more facings, which a start
// leaving every facing 0 cannot.
func TestReleaseEveryMissionStartHoldsItsConstructedMovers(t *testing.T) {
	f := releaseFront(t)
	f.SetTipsOff(true)
	a := f.App("mission start Movers")
	a.SetCutscenes(nil)
	opened, actors := 0, 0
	for mission := 1; mission <= 150; mission++ {
		if err := a.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
			continue
		}
		opened++
		w := f.live.world
		records := moverStartRecords(t, f, fmt.Sprintf("Mover start %d", mission))
		facings := map[byte]bool{}
		placed := 0
		for _, e := range w.Entities() {
			mover, ok := records[e.MapUnitID]
			if e.MapUnitID == 0 || !ok {
				continue
			}
			placed++
			var want [180]byte
			if !w.ApplyMover(e.ID, &want) {
				t.Fatalf("mission %d actor %d holds no Mover record", mission, e.ID)
			}
			facings[mover[0]] = true
			bad := mover[0] != mover[1] || mover[0] < 0x40 || mover[0] > 0xc0 || mover[0] != e.Facing ||
				mover[5] != sim.MoverPassabilityMask(uint32(e.Domain)+1) ||
				binary.LittleEndian.Uint16(mover[0x72:]) != binary.LittleEndian.Uint16(want[0x72:])
			if mover[8] != 5 || e.Owner != sim.SelfSlot && mover[9] != 0xff {
				bad = true
			}
			if bad {
				t.Fatalf("mission %d actor %d (unit %d) Mover % x, step cost %#x", mission, e.ID, e.MapUnitID, mover[:11], binary.LittleEndian.Uint16(mover[0x72:]))
			}
		}
		actors += placed
		if placed >= 2 && len(facings) < 2 {
			t.Fatalf("mission %d: %d placed actors share one facing", mission, placed)
		}
	}
	if opened < 20 {
		t.Fatalf("opened only %d missions", opened)
	}
	t.Logf("%d missions, %d placed actors", opened, actors)
}

// TestReleaseMissionStartMoversAgainstTheRestartSlots compares a fresh start
// of each restart slot's mission (game9999.sav, SAV-1213, SAV-1225) with the
// original's own start: for every map-placed actor both hold, every Mover
// byte except the drawn facing +0x00/+0x01. Each difference is listed by
// offset and count. The controlled player's +0x09 and +0x82..+0x85 differ
// by DIV-1388 and DIV-3035; its +0x08 search radius does not (AI-692). An AI actor whose speed word differs
// differs at +0x0a and +0x72 together and is listed; any other difference
// fails.
func TestReleaseMissionStartMoversAgainstTheRestartSlots(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: release integration needs a lawful install")
	}
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if root == "" {
		root = filepath.Join(os.Getenv("AGAINROM_ASSETS"), "..", "saves")
	}
	paths, _ := filepath.Glob(filepath.Join(root, "*", "game9999.sav"))
	if len(paths) == 0 {
		t.Skip("no restart slot in the save corpus")
	}
	compared := 0
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		probe := releaseFront(t)
		probe.SetTipsOff(true)
		opener, town, err := probe.RestoreOriginal(raw)
		if err != nil || town {
			t.Logf("%s: not this edition's mission (%v)", path, err)
			continue
		}
		pa := probe.App("restart slot probe")
		pa.SetCutscenes(nil)
		if err := pa.OpenMission(opener); err != nil {
			t.Logf("%s: not opened (%v)", path, err)
			continue
		}
		mission := probe.live.mission.state.Number
		f := releaseFront(t)
		f.SetTipsOff(true)
		a := f.App("restart slot fresh start")
		a.SetCutscenes(nil)
		if err := a.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
			t.Fatal(err)
		}
		ours := moverStartRecords(t, f, "Mover restart compare")
		theirs := moverRecordsByMapUnit(doc)
		owner := map[uint16]uint32{}
		for _, e := range f.live.world.Entities() {
			owner[e.MapUnitID] = e.Owner
		}
		diffs := map[string]int{}
		actors := 0
		for unit, want := range theirs {
			got, ok := ours[unit]
			if !ok {
				continue
			}
			actors++
			if got[10] != want[10] {
				t.Logf("unit %d owner %d: rate ours %#x original %#x, step cost %#x / %#x", unit, owner[unit], got[10], want[10], binary.LittleEndian.Uint16(got[0x72:]), binary.LittleEndian.Uint16(want[0x72:]))
			}
			for at := 2; at < 180; at++ {
				if got[at] != want[at] {
					side := "ai"
					if owner[unit] == sim.SelfSlot {
						side = "self"
					}
					diffs[fmt.Sprintf("%s+%#02x", side, at)]++
					explained := side == "self" && (at == 9 || at >= 0x82 && at <= 0x85) ||
						side == "ai" && (at == 10 || at == 0x72 || at == 0x73) && got[10] != want[10]
					if !explained {
						t.Errorf("%s unit %d: Mover +%#02x ours %#x original %#x", path, unit, at, got[at], want[at])
					}
				}
			}
		}
		compared++
		var keys []string
		for k := range diffs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var list []string
		for _, k := range keys {
			list = append(list, fmt.Sprintf("%s:%d", k, diffs[k]))
		}
		t.Logf("%s mission %d: %d actors; differences %s", path, mission, actors, strings.Join(list, " "))
	}
	if compared == 0 {
		t.Skip("no restart slot of this edition")
	}
}
