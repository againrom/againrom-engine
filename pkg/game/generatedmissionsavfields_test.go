package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// TestReleaseGeneratedMissionSAVOriginalConstraints asserts the field-level
// constraints SAV-1091..1098 establish an original mission LOAD requires,
// against the exact document ExportCurrentSave writes for a freshly opened,
// never-saved generated mission (the M7 path). It does not assert byte
// identity with any owner file; it asserts the same field CLASS the corpus
// establishes: no row-0 Weapon/Armor/Shield definition, a nonzero mover
// RotationSpeed, the mover's passability mask derived from its own domain
// byte, a Unit's Capacity and own weight, publication mask 2 on every
// Human/Unit/Building/Sack token, the Building type word at T0E (not T0C),
// the sack/ground-actor block-plane bits with no full-map dump, each actor's
// AI-start post, and AutoGetMission naming the scenario's own declared
// successor (10 -> 20, every other campaign mission -> -1). It runs for both
// mission 10 (the one declared successor) and mission 20 (a no-successor
// mission), and confirms a native city save carries none of the mission-only
// fields this story changed.
func TestReleaseGeneratedMissionSAVOriginalConstraints(t *testing.T) {
	for _, tc := range []struct {
		mission  int
		wantAuto uint32
	}{
		{10, 20},
		{20, 0xFFFFFFFF},
	} {
		t.Run(missionSubtestName(tc.mission), func(t *testing.T) {
			f := releaseFront(t)
			party := MissionParty(nil, data.BodyList{}, nil)
			app := f.App("generated mission SAV constraints")
			if err := app.OpenMission(f.MissionOpenerWith(tc.mission, party)); err != nil {
				t.Fatal(err)
			}
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			// SAV-BLOCK-011/TERR-PASS-053: this is the ROM1-byte-matching kit
			// export F3 names, not an ordinary round-trip SAVE, so it opts into
			// the narrow sweep-window block-plane delta the original writer
			// itself keeps rather than this engine's own full per-cell backfill.
			snapshot.NativeMissionTerrain = true
			raw, err := f.ExportCurrentSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			if doc.World == nil {
				t.Fatal("generated mission document has no world half")
			}

			// DIV-1388: the party's own hero is excluded from the AI-start
			// mover/order repairs below (savactorproject.go), so this subtest
			// needs to tell the hero's own Human record apart from every
			// other Human/Unit record the same repair still covers. The
			// live world's own SelfSlot-owned entity names the hero's cell;
			// doc.Objects carries no owner field to read this back from
			// directly, and DecodeDocumentData's own object numbering is not
			// the archive's ArchiveIndex (PartyWalk/ActorGraph's own key),
			// so position is the one join key both sides agree on.
			if f.live == nil || f.live.mission == nil || f.live.mission.state == nil {
				t.Fatal("mission state unavailable after OpenMission")
			}
			heroCell := map[[2]byte]bool{}
			for _, e := range f.live.mission.state.World.Entities() {
				if e.Owner == sim.SelfSlot {
					heroCell[[2]byte{byte(e.X), byte(e.Y)}] = true
				}
			}
			isHero := func(pos []byte) bool {
				return len(pos) >= 2 && heroCell[[2]byte{pos[0], pos[1]}]
			}

			// SAV-1091: a Weapon/Armor/Shield definition row of 0 has no
			// parameters and terminates the original at LOAD.
			for _, obj := range doc.Objects {
				if obj.Class != "Weapon" && obj.Class != "Armor" && obj.Class != "Shield" {
					continue
				}
				if v, err := savedStructureValue(&obj, "T0C"); err == nil && v == 0 {
					t.Errorf("%s record has definition row 0", obj.Class)
				}
			}

			// SAV-1092/TERR-PASS-051: a mover RotationSpeed byte of 0 divides
			// by zero on the actor's first turn, and the passability mask
			// (+0x05) must match the domain byte (U4A) this same export
			// writes rather than stay at the zero LOAD never repairs.
			for _, obj := range doc.Objects {
				if obj.Class != "Human" && obj.Class != "Unit" {
					continue
				}
				var objPos []byte
				for _, raw := range obj.Raw {
					if raw.Name == "Block12" {
						objPos = raw.Bytes
					}
				}
				hero := isHero(objPos)
				domain, err := savedStructureValue(&obj, "U4A")
				if err != nil {
					t.Errorf("%s record has no U4A domain: %v", obj.Class, err)
					continue
				}
				for _, raw := range obj.Raw {
					if raw.Name != "U154" || len(raw.Bytes) <= 10 {
						continue
					}
					if raw.Bytes[10] == 0 {
						t.Errorf("%s record has zero mover RotationSpeed", obj.Class)
					}
					if want := sim.MoverPassabilityMask(domain); raw.Bytes[5] != want {
						t.Errorf("%s record mover mask=%#x, want %#x for domain %d", obj.Class, raw.Bytes[5], want, domain)
					}
					// SAV-1096: every AI-owned mission-start original carries
					// 5,255 at +0x08/+0x09 and 0x80,0x80 at +0x84/+0x85. DIV-1388:
					// the party's own hero carries zero at both instead (SAV-1096's
					// own corpus is entirely AI actors; a real corpus hero record
					// and the byte-corrected game9237 reference agree it stays
					// zero), matching savactorproject.go's owner exclusion.
					if len(raw.Bytes) > 0x85 {
						if hero {
							if raw.Bytes[8] != 0 || raw.Bytes[9] != 0 {
								t.Errorf("hero record mover +0x08/+0x09=%d,%d, want 0,0", raw.Bytes[8], raw.Bytes[9])
							}
							if raw.Bytes[0x84] != 0 || raw.Bytes[0x85] != 0 {
								t.Errorf("hero record mover +0x84/+0x85=%#x,%#x, want 0,0", raw.Bytes[0x84], raw.Bytes[0x85])
							}
							continue
						}
						if raw.Bytes[8] != 5 || raw.Bytes[9] != 255 {
							t.Errorf("%s record mover +0x08/+0x09=%d,%d, want 5,255", obj.Class, raw.Bytes[8], raw.Bytes[9])
						}
						if raw.Bytes[0x84] != 0x80 || raw.Bytes[0x85] != 0x80 {
							t.Errorf("%s record mover +0x84/+0x85=%#x,%#x, want 0x80,0x80", obj.Class, raw.Bytes[0x84], raw.Bytes[0x85])
						}
					}
				}
			}

			// UNIT-CTOR-004/SAV-UNITFLD-049: every Unit carries Capacity 300
			// (nothing in this engine ever derives one), and own weight (U8E)
			// mirrors the load (U90) already exported rather than a
			// never-populated zero.
			for _, obj := range doc.Objects {
				if obj.Class != "Unit" && obj.Class != "Human" {
					continue
				}
				u8e, err := savedStructureValue(&obj, "U8E")
				if err != nil {
					t.Errorf("%s record has no U8E: %v", obj.Class, err)
					continue
				}
				u90, err := savedStructureValue(&obj, "U90")
				if err != nil {
					t.Errorf("%s record has no U90: %v", obj.Class, err)
					continue
				}
				if u8e != u90 {
					t.Errorf("%s record own weight U8E=%d, load U90=%d, want equal", obj.Class, u8e, u90)
				}
				if obj.Class == "Unit" {
					if cap, err := savedStructureValue(&obj, "Capacity"); err != nil {
						t.Errorf("Unit record has no Capacity: %v", err)
					} else if cap != 300 {
						t.Errorf("Unit record Capacity=%d, want 300", cap)
					}
				}
			}

			// SAV-1093: every verified original Building/Human/Unit/Sack
			// token carries publication mask 2 at Token+0x18.
			for _, obj := range doc.Objects {
				switch obj.Class {
				case "Human", "Unit", "Building", "Sack":
					if v, err := savedStructureValue(&obj, "T18"); err != nil {
						t.Errorf("%s record has no T18 field: %v", obj.Class, err)
					} else if v != 2 {
						t.Errorf("%s record has publication mask %d, want 2", obj.Class, v)
					}
				}
			}

			// A Building's type word is Token+17 (T0E), not Token+16 (T0C,
			// which must stay 0): a T0C-only writer draws no structure.
			for _, obj := range doc.Objects {
				if obj.Class != "Building" {
					continue
				}
				if v, err := savedStructureValue(&obj, "T0C"); err != nil || v != 0 {
					t.Errorf("Building record T0C=%v (err %v), want 0", v, err)
				}
				if v, err := savedStructureValue(&obj, "T0E"); err != nil || v == 0 {
					t.Errorf("Building record T0E=%v (err %v), want nonzero structure kind", v, err)
				}
			}

			// SAV-BLOCK-011/TERR-PASS-051: a sack or ground-actor cell's
			// static byte carries bit 0x20 (a hash-table record exists
			// there); a ground actor's cell also carries dynamic bit 0x40,
			// and an air actor's carries 0x80. The block plane is a DELTA:
			// its row count must stay far under a full per-cell dump, and
			// bit 0x20 must appear on exactly the cell-record rows, neither
			// fewer nor more.
			blockByCell := map[uint16]sav.BlockRecord{}
			for _, b := range doc.World.Blocks {
				blockByCell[b.Cell] = b
			}
			if f.live == nil || f.live.mission == nil || f.live.mission.state == nil {
				t.Fatal("mission state unavailable after OpenMission")
			}
			mapCells := f.live.mission.state.Map.Width * f.live.mission.state.Map.Height
			if got, bound := len(doc.World.Blocks), mapCells*3/4; got >= bound {
				t.Errorf("block plane has %d rows over a %d-cell map, want well under a full per-cell dump (bound %d)", got, mapCells, bound)
			}
			recordCells := map[uint16]bool{}
			for _, idx := range doc.World.Sacks {
				r := doc.Objects[idx-1]
				var pos []byte
				for _, raw := range r.Raw {
					if raw.Name == "Block12" {
						pos = raw.Bytes
					}
				}
				if len(pos) < 2 {
					t.Errorf("Sack record has no position block")
					continue
				}
				cell := uint16(pos[0]) | uint16(pos[1])<<8
				recordCells[cell] = true
				if b, ok := blockByCell[cell]; !ok || b.Static&0x20 == 0 {
					t.Errorf("sack cell %04x has no static record bit", cell)
				}
			}
			for _, c := range doc.World.Cells {
				if c.Sack == 0 && c.GroundActor == 0 && c.AirActor == 0 && c.Building == 0 {
					continue
				}
				recordCells[c.Cell] = true
				b, ok := blockByCell[c.Cell]
				if !ok || b.Static&0x20 == 0 {
					t.Errorf("cell-record cell %04x has no static record bit", c.Cell)
					continue
				}
				if c.GroundActor != 0 && b.Dyn&0x40 == 0 {
					t.Errorf("ground-actor cell %04x has no dynamic occupant bit", c.Cell)
				}
				if c.AirActor != 0 && b.Dyn&0x80 == 0 {
					t.Errorf("air-actor cell %04x has no dynamic air bit", c.Cell)
				}
			}
			for cell, b := range blockByCell {
				if b.Static&0x20 != 0 && !recordCells[cell] {
					t.Errorf("cell %04x carries static record bit 0x20 with no cell record", cell)
				}
			}

			// SAV-1096/AI-POST-042: every AI-owned actor's AI-start order
			// object and mover both name its own cell as its post, not the
			// zero construction default that lets a hostile actor fight
			// early. DIV-1388: the party's own hero is excluded (its post
			// stays 0,0), matching savactorproject.go's owner gate.
			for _, obj := range doc.Objects {
				if obj.Class != "Human" && obj.Class != "Unit" {
					continue
				}
				var pos, order, mover []byte
				for _, raw := range obj.Raw {
					switch raw.Name {
					case "Block12":
						pos = raw.Bytes
					case "U158":
						order = raw.Bytes
					case "U154":
						mover = raw.Bytes
					}
				}
				if len(pos) < 2 || len(order) < 2 || len(mover) <= 0x83 {
					t.Errorf("%s record is missing a position/order/mover block", obj.Class)
					continue
				}
				if isHero(pos) {
					if order[0] != 0 || order[1] != 0 {
						t.Errorf("hero record order post=%d,%d, want 0,0", order[0], order[1])
					}
					if mover[0x82] != 0 || mover[0x83] != 0 {
						t.Errorf("hero record mover post=%d,%d, want 0,0", mover[0x82], mover[0x83])
					}
					continue
				}
				if order[0] != pos[0] || order[1] != pos[1] {
					t.Errorf("%s record order post=%d,%d, want own cell %d,%d", obj.Class, order[0], order[1], pos[0], pos[1])
				}
				if mover[0x82] != pos[0] || mover[0x83] != pos[1] {
					t.Errorf("%s record mover post=%d,%d, want own cell %d,%d", obj.Class, mover[0x82], mover[0x83], pos[0], pos[1])
				}
			}
			// SAV-1096/AI-GRPGUARD-074: a Unit is never the party's own hero,
			// so every Unit record's raw state word (U50) is asserted at the
			// idle-turn/guard default every mission-start original carries.
			for _, obj := range doc.Objects {
				if obj.Class != "Unit" {
					continue
				}
				for _, raw := range obj.Raw {
					if raw.Name == "U50" && len(raw.Bytes) == 4 {
						if raw.Bytes[0] != 0x0b || raw.Bytes[1] != 0 || raw.Bytes[2] != 0 || raw.Bytes[3] != 0 {
							t.Errorf("Unit record U50=%v, want [11 0 0 0]", raw.Bytes)
						}
					}
				}
			}
			// DIV-1388: a Human record naming the party's own hero carries
			// the raw state word (U50) untouched at 0 — the AI idle-turn
			// default above is excluded for it, the same as its post above.
			for _, obj := range doc.Objects {
				if obj.Class != "Human" {
					continue
				}
				var objPos []byte
				for _, raw := range obj.Raw {
					if raw.Name == "Block12" {
						objPos = raw.Bytes
					}
				}
				if !isHero(objPos) {
					continue
				}
				for _, raw := range obj.Raw {
					if raw.Name == "U50" && len(raw.Bytes) == 4 {
						if raw.Bytes[0] != 0 || raw.Bytes[1] != 0 || raw.Bytes[2] != 0 || raw.Bytes[3] != 0 {
							t.Errorf("hero record U50=%v, want [0 0 0 0]", raw.Bytes)
						}
					}
				}
			}

			// SAV-1094/REG-SCN-063: AutoGetMission names the scenario's own
			// declared successor, or -1 where none is declared.
			f2, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			camp, present, err := f2.Campaign()
			if err != nil || !present {
				t.Fatalf("campaign: present=%v err=%v", present, err)
			}
			if camp.AutoGetMission != tc.wantAuto {
				t.Errorf("AutoGetMission=%d, want %d", camp.AutoGetMission, tc.wantAuto)
			}
		})
	}
}

func missionSubtestName(mission int) string {
	switch mission {
	case 10:
		return "mission10"
	case 20:
		return "mission20"
	default:
		return "mission"
	}
}

// TestReleaseNativeCitySAVAutoGetMissionUnchangedByMissionDocumentFix confirms
// the mission-document AutoGetMission fix (REG-SCN-063) reaches only the
// mission SAV path: a native city save at chapter 0 (before any main mission
// is won) still carries no declared successor, exactly as it did before.
func TestReleaseNativeCitySAVAutoGetMissionUnchangedByMissionDocumentFix(t *testing.T) {
	f := releaseFront(t)
	projection, err := nativeCampaignProjectionForChapter(f, Snapshot{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if projection.AutoGetMission != 0xFFFFFFFF {
		t.Errorf("native city chapter-0 AutoGetMission=%d, want -1 (no chapter-0 [Mission0] section declares a successor)", projection.AutoGetMission)
	}
}

// TestReleaseCampaignAutoGetMissionMatchesScenarioData asserts, per the
// owner's own account of the campaign (after mission 20 the player returns
// to town, not to a next mission, and every mission from 30 on always ends
// the same way; mission 10 is the one case that advances directly), that
// autoGetMissionValue agrees with the installed scenario data for EVERY
// declared campaign mission, not only 10 and 20: a successor only where
// [Mission<n>] actually declares one, and -1 everywhere else. A mission
// whose own data disagrees with this account fails loudly here rather than
// being silently overridden.
func TestReleaseCampaignAutoGetMissionMatchesScenarioData(t *testing.T) {
	f := releaseFront(t)
	camp := f.Campaign.Value()
	missions := append(append([]int{}, camp.Main...), camp.Side...)
	if len(missions) == 0 {
		t.Fatal("installed scenario declares no missions")
	}
	for _, mission := range missions {
		next, hasNext := camp.AutoAdvance(mission)
		got := autoGetMissionValue(camp, mission)
		switch {
		case mission == 10:
			if !hasNext || next != 20 || got != 20 {
				t.Errorf("mission 10: scenario successor=%d(present=%v), autoGetMissionValue=%d, want 20/20", next, hasNext, got)
			}
		case hasNext:
			// The owner's account allows no declared successor anywhere but
			// mission 10. A different mission naming one is a genuine
			// conflict with that account, not a case to silently resolve.
			t.Errorf("CONFLICT: mission %d declares successor %d in the installed scenario data, contradicting the owner's account that only mission 10 advances directly", mission, next)
		case got != 0xFFFFFFFF:
			t.Errorf("mission %d: autoGetMissionValue=%d, want -1 (no declared successor)", mission, got)
		}
	}
}
