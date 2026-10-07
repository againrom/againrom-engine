//go:build sessioncorpusaudit

package game

import (
	"testing"

	"againrom/pkg/mapload"
)

// Session-block sub-region byte offsets, this instrument's own transcription
// of SAV-SESS-031's store-arm enumeration (registers, latches, rawHead,
// rawMid, the diplomacy matrix, Won, Lose, each in the order and at the
// running sum SAV-SESS-031 gives), independent of
// pkg/formats/sav/world.go's own unexported sess* constants of the same
// values. DIV-927 names three stretches this enumeration leaves with no
// promoted meaning -- 8 bytes before the matrix, 6 after it (u8, u8, u32),
// 4 between Won and Lose -- 18 bytes total, neither read into a typed field
// nor carried opaquely anywhere in this project; they are excluded from
// comparison below, cited by that row rather than guessed at.
const (
	milestone2SessTriggerResults = 0    // 100 signed int32 (SAV-SESS-031, TRIG-STORE-002)
	milestone2SessTriggerLatches = 400  // 1000 fire-once bytes (TRIG-FIRE-007)
	milestone2SessRawHead        = 1400 // 48 opaque bytes
	milestone2SessRawMid         = 1448 // 400 opaque bytes
	milestone2SessDiplomacy      = 1856 // 50x50 matrix (AI-DIPLO-004)
	milestone2SessWon            = 4362 // u32, "the win counter" (SAV-SESS-031)
	milestone2SessLose           = 4370 // u32, "Lose" (SESS-LOSE-012, session+0xb3b4)

	milestone2TriggerResultCount = 100
	milestone2TriggerLatchCount  = 1000
	milestone2DiplomacySide      = 50
	milestone2RawHeadLen         = 48
	milestone2RawMidLen          = 400
)

// TestMilestone2SessionBlock is the independent acceptance instrument for the
// "session block (4,374 B)" row of pipeline/SAV-COMPLETION.md.
//
// Fields compared: the world head's clock pair CounterA/CounterB
// (SAV-HEAD-025, this row's "clock"; read at body offset 0/4, the same
// layout TestMilestone2MapReopen uses independently for the same head), the
// hundred trigger-result registers, the thousand trigger latches, the 50x50
// diplomacy matrix and the Won counter (SAV-SESS-031, "the win counter") and
// Lose counter (offset located by SAV-SESS-031, named by SESS-LOSE-012 --
// see the offset table's own doc comment above). The three DIV-927 gaps (18
// bytes total) are read by nothing in this project and are excluded, not
// compared.
//
// Every field is read here at f.World.SessionOff plus this file's own
// constants above, using this file's own byte arithmetic -- never through
// sav.File.SessionState()/TriggerResult/TriggerLatch/Diplomacy/Counters,
// pkg/formats/sav/world.go and session.go's decode of the identical bytes.
// f.World.SessionOff is used only as the structure's own start offset
// (WorldHalf's own doc comment), on this instrument's independence rule.
func TestMilestone2SessionBlock(t *testing.T) {
	filesChecked := 0
	clockMismatches, registerMismatches, latchMismatches := 0, 0, 0
	rawHeadMismatches, rawMidMismatches := 0, 0
	diplomacyMismatches, wonMismatches, loseMismatches := 0, 0, 0
	diplomacyRosterZeroNonzero := 0
	var refused []milestone2ResumeRefusal

	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		w := mf.f.World
		if w == nil {
			t.Fatal("mf.present true but sav.File.World is nil")
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, mapload.DifficultyNormal, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		filesChecked++
		base := w.SessionOff

		// Clock (SAV-HEAD-025): body offset 0/4, ahead of the session block
		// itself but grouped under this row by SAV-COMPLETION.md.
		counterA := rawU32(mf.body, 0)
		counterB := rawU32(mf.body, 4)
		clock, ok := ms.World.SessionClock()
		if !ok {
			t.Error("resumed world carries no session clock")
			clockMismatches++
		} else if clock.SubTick != counterA || clock.FullTick != counterB {
			clockMismatches++
			t.Errorf("clock: live %d/%d, file %d/%d", clock.SubTick, clock.FullTick, counterA, counterB)
		}

		// Trigger-result registers.
		live := ms.World.ScriptRegisters()
		for i := 0; i < milestone2TriggerResultCount; i++ {
			want := int32(rawU32(mf.body, base+milestone2SessTriggerResults+4*i))
			if live[i] != want {
				registerMismatches++
				t.Errorf("register %d: live %d, file %d", i, live[i], want)
			}
		}

		// Trigger latches.
		for i := 0; i < milestone2TriggerLatchCount; i++ {
			raw := mf.body[base+milestone2SessTriggerLatches+i]
			if got, want := ms.World.ScriptLatched(int32(i)), raw != 0; got != want {
				latchMismatches++
				t.Errorf("latch %d: live %v, file byte %d", i, got, raw)
			}
		}

		// RawHead/RawMid: opaque, but carried live; compared byte for byte.
		wantHead := [milestone2RawHeadLen]byte{}
		copy(wantHead[:], mf.body[base+milestone2SessRawHead:base+milestone2SessRawHead+milestone2RawHeadLen])
		if got := ms.World.RawSessionHead(); got != wantHead {
			rawHeadMismatches++
			t.Errorf("RawSessionHead: live %x, file %x", got, wantHead)
		}
		wantMid := [milestone2RawMidLen]byte{}
		copy(wantMid[:], mf.body[base+milestone2SessRawMid:base+milestone2SessRawMid+milestone2RawMidLen])
		if got := ms.World.RawSessionMid(); got != wantMid {
			rawMidMismatches++
			t.Errorf("RawSessionMid: live %x, file %x", got, wantMid)
		}

		// Diplomacy matrix. Roster slot 0 has no live cell at all
		// (sim.relationIndex excludes from==0 and to==0; AI-DIPLO-004 names
		// slot 0 as no roster entry), so Relations.Byte returns 0 there
		// regardless of the file's own byte -- logged if that ever disagrees,
		// not silently assumed zero.
		relations := ms.World.Relations()
		for from := 0; from < milestone2DiplomacySide; from++ {
			for to := 0; to < milestone2DiplomacySide; to++ {
				raw := mf.body[base+milestone2SessDiplomacy+from*milestone2DiplomacySide+to]
				got := relations.Byte(uint32(from), uint32(to))
				if (from == 0 || to == 0) && raw != 0 {
					diplomacyRosterZeroNonzero++
				}
				if got != raw {
					diplomacyMismatches++
					t.Errorf("diplomacy[%d][%d]: live %#02x, file %#02x", from, to, got, raw)
				}
			}
		}

		// Won/Lose.
		won, lost := ms.World.ScriptCounters()
		if wantWon := rawU32(mf.body, base+milestone2SessWon); won != wantWon {
			wonMismatches++
			t.Errorf("Won: live %d, file %d", won, wantWon)
		}
		if wantLost := rawU32(mf.body, base+milestone2SessLose); lost != wantLost {
			loseMismatches++
			t.Errorf("Lose: live %d, file %d", lost, wantLost)
		}
	})

	t.Logf("session block: %d world-half file(s) checked, %d refused", filesChecked, len(refused))
	milestone2LogRefusals(t, "session block", refused)
	t.Logf("session block mismatches: %d clock, %d register, %d latch, %d RawHead, %d RawMid, %d diplomacy, %d Won, %d Lose",
		clockMismatches, registerMismatches, latchMismatches, rawHeadMismatches, rawMidMismatches, diplomacyMismatches, wonMismatches, loseMismatches)
	t.Logf("diplomacy roster-slot-0 rows/columns with a nonzero file byte: %d (excluded from comparison, sim.relationIndex/AI-DIPLO-004)", diplomacyRosterZeroNonzero)
	if clockMismatches+registerMismatches+latchMismatches+rawHeadMismatches+rawMidMismatches+diplomacyMismatches+wonMismatches+loseMismatches != 0 {
		t.Fatal("one or more session block mismatches, see above")
	}
}
