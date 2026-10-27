package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// runtimeIDFileRecord is one Token-headed record of a save as its own bytes
// hold it: the class and the dword at Token +0x04, body offset 12 of the head
// (SAV-TOKEN-034).
type runtimeIDFileRecord struct {
	class string
	id    uint32
}

// runtimeIDOwnerClasses are the classes whose +0x04 an object carries while it
// lives (SAV-1368); an Item or Effect value is only marked at LOAD.
var runtimeIDOwnerClasses = map[string]bool{"Human": true, "Humanoid": true, "Unit": true,
	"Building": true, "Shop": true, "Tavern": true, "Outpost": true, "Sack": true}

func runtimeIDFileRecords(f *sav.File) ([]runtimeIDFileRecord, error) {
	locations, err := f.DocumentObjectLocations()
	if err != nil {
		return nil, err
	}
	var out []runtimeIDFileRecord
	for _, l := range locations {
		if !sav.Lookup(l.Class).Head {
			continue
		}
		if l.Off < 0 || l.Off+16 > len(f.Body) {
			return nil, fmt.Errorf("record %d %s at %d past the body", l.ArchiveIndex, l.Class, l.Off)
		}
		out = append(out, runtimeIDFileRecord{l.Class, binary.LittleEndian.Uint32(f.Body[l.Off+12:])})
	}
	return out, nil
}

// runtimeIDFileReplay is SAV-1371 computed from the records alone: bit 0, then
// in archive order each non-zero id marks its low 16 bits, a Sack first taking
// the lowest clear bit. It returns the marked bits ascending and the lowest
// clear bit, the id the next allocation gives.
func runtimeIDFileReplay(records []runtimeIDFileRecord) ([]uint16, uint32) {
	var bits [1 << 16]bool
	bits[0] = true
	lowest := func() uint32 {
		for i := range bits {
			if !bits[i] {
				return uint32(i)
			}
		}
		return 0
	}
	for _, r := range records {
		if r.class == "Sack" {
			bits[lowest()] = true
		}
		if r.id != 0 {
			bits[uint16(r.id)] = true
		}
	}
	var marks []uint16
	for i := range bits {
		if bits[i] {
			marks = append(marks, uint16(i))
		}
	}
	return marks, lowest()
}

// runtimeIDOwners lists the owner records' ids, sorted, and the Item and
// Effect ids, sorted.
func runtimeIDOwners(records []runtimeIDFileRecord) (owners, others []uint32) {
	for _, r := range records {
		if runtimeIDOwnerClasses[r.class] {
			owners = append(owners, r.id)
		} else {
			others = append(others, r.id)
		}
	}
	slices.Sort(owners)
	slices.Sort(others)
	return owners, others
}

// runtimeIDReplayTrace names, in archive order, each record that marks a bit
// below limit and each Sack constructor's bit.
func runtimeIDReplayTrace(records []runtimeIDFileRecord, limit uint32) string {
	var bits [1 << 16]bool
	bits[0] = true
	var out []string
	for i, r := range records {
		if r.class == "Sack" {
			low := 0
			for bits[low] {
				low++
			}
			bits[low] = true
			out = append(out, fmt.Sprintf("%d:Sack+%d", i, low))
		}
		if r.id != 0 {
			bits[uint16(r.id)] = true
			if uint16(r.id) < uint16(limit) {
				out = append(out, fmt.Sprintf("%d:%s=%d", i, r.class, r.id))
			}
		}
	}
	return fmt.Sprint(out)
}

// runtimeIDClassCounts names, per class, the records and how many hold 0.
func runtimeIDClassCounts(records []runtimeIDFileRecord) map[string][2]int {
	out := map[string][2]int{}
	for _, r := range records {
		c := out[r.class]
		c[0]++
		if r.id == 0 {
			c[1]++
		}
		out[r.class] = c
	}
	return out
}

func firstMarkDifference(a, b []uint16) string {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return fmt.Sprintf("at %d: %d against %d", i, a[i], b[i])
		}
	}
	return fmt.Sprintf("lengths %d against %d", len(a), len(b))
}

// runtimeIDOlderEngineFiles are corpus files an earlier engine wrote, whose
// SAVE gave ids by its own document allocation, also to departed actors. A
// current SAVE writes a departed actor's record with no id and writes no
// record for one that leaves no body, so only their LOAD replay is required.
var runtimeIDOlderEngineFiles = map[string]bool{"2026-10-07/saveorcsdontgo.sav": true}

func mustOpenSAV(t *testing.T, raw []byte) *sav.File {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// runtimeIDClaimExamples are the SAV-1371 examples by the first 16 hex digits
// of the file's SHA-256 and the lowest clear bit the claim's replay gives.
var runtimeIDClaimExamples = map[string]uint32{"3407c9fce2ac9ce0": 23}

// A World LOAD leaves the allocator as the file's own replay gives it
// (SAV-1371), for an original SAV and one this engine wrote; SAVE keeps every
// carried id; a cold LOAD of that SAVE holds the same allocator as the World
// that wrote it, before and after one tick.
func TestReleaseRuntimeIDsAfterLoadFollowTheCorpusReplay(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	corpus, assets := os.Getenv("AGAINROM_SAVE_CORPUS"), os.Getenv("AGAINROM_ASSETS")
	if corpus == "" || assets == "" {
		t.Skip("AGAINROM_SAVE_CORPUS and AGAINROM_ASSETS are required")
	}
	front, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	front.SetDeterministicFrames(true)
	cleanupFrontAudio(t, front)
	type tally struct{ files, refused, replayed, kept, cold, hashed, otherMoved, resaved int }
	counts := map[string]*tally{}
	examples := 0
	check := func(t *testing.T, kind, rel string, raw []byte, f *sav.File) {
		c := counts[kind]
		if c == nil {
			c = &tally{}
			counts[kind] = c
		}
		c.files++
		records, err := runtimeIDFileRecords(f)
		if err != nil {
			t.Fatal(err)
		}
		marks, next := runtimeIDFileReplay(records)
		sum := sha256.Sum256(raw)
		if want, found := runtimeIDClaimExamples[hex.EncodeToString(sum[:8])]; found {
			examples++
			if next != want {
				t.Errorf("the claim example replays to %d, want %d", next, want)
			}
		}
		live, err := loadOriginalFront(front, raw)
		if err != nil {
			c.refused++
			t.Logf("LOAD refused: %v", err)
			return
		}
		w := live.CurrentMission().World
		if got := w.RuntimeIDMarks(); !slices.Equal(got, marks) {
			t.Errorf("LOAD marks differ from the file replay: %s", firstMarkDifference(got, marks))
		} else if got := w.NextRuntimeID(); got != next {
			t.Errorf("LOAD gives %d next, the file replay %d", got, next)
		} else {
			c.replayed++
		}
		written := currentRuntimeSave(t, live)
		wf, err := sav.Open(written)
		if err != nil {
			t.Fatal(err)
		}
		writtenRecords, err := runtimeIDFileRecords(wf)
		if err != nil {
			t.Fatal(err)
		}
		owners, others := runtimeIDOwners(records)
		gotOwners, gotOthers := runtimeIDOwners(writtenRecords)
		older := runtimeIDOlderEngineFiles[rel]
		if !slices.Equal(gotOwners, owners) {
			report := t.Errorf
			if older {
				report = t.Logf
			}
			report("SAVE wrote owner ids\n%v\nwant\n%v\nby class %v against %v", gotOwners, owners, runtimeIDClassCounts(writtenRecords), runtimeIDClassCounts(records))
		} else {
			c.kept++
		}
		// An Item or Effect the SAVE built anew holds 0 (SAV-1369), so the next
		// LOAD's replay lacks the mark the loaded value made; the cold World
		// then follows the written file's replay, not the continued World.
		moved := older || !slices.Equal(gotOthers, others)
		if moved {
			c.otherMoved++
			t.Logf("SAVE wrote Item and Effect ids %v against %v loaded", gotOthers, others)
		}
		cold, err := loadOriginalFront(front, written)
		if err != nil {
			t.Fatalf("cold LOAD of the SAVE: %v", err)
		}
		cw := cold.CurrentMission().World
		writtenMarks, writtenNext := runtimeIDFileReplay(writtenRecords)
		if got := cw.RuntimeIDMarks(); !slices.Equal(got, writtenMarks) || cw.NextRuntimeID() != writtenNext {
			t.Errorf("cold LOAD differs from the written file's replay: %s", firstMarkDifference(got, writtenMarks))
		}
		// The file this engine wrote, loaded and saved again, keeps every id.
		again, err := runtimeIDFileRecords(mustOpenSAV(t, currentRuntimeSave(t, cold)))
		if err != nil {
			t.Fatal(err)
		}
		againOwners, againOthers := runtimeIDOwners(again)
		if !slices.Equal(againOwners, gotOwners) || !slices.Equal(againOthers, gotOthers) {
			t.Errorf("the second SAVE wrote ids\n%v %v\nagainst\n%v %v", againOwners, againOthers, gotOwners, gotOthers)
		} else {
			c.resaved++
		}
		same := func(when string) bool {
			if a, b := w.RuntimeIDMarks(), cw.RuntimeIDMarks(); !slices.Equal(a, b) {
				report := t.Errorf
				if moved {
					report = t.Logf
				}
				report("%s: cold LOAD marks differ: %s\nloaded %s\nwritten %s", when, firstMarkDifference(b, a), runtimeIDReplayTrace(records, 24), runtimeIDReplayTrace(writtenRecords, 24))
				return false
			}
			if a, b := w.NextRuntimeID(), cw.NextRuntimeID(); a != b {
				t.Errorf("%s: cold LOAD gives %d next, the continued World %d", when, b, a)
				return false
			}
			return true
		}
		before := same("after LOAD")
		sim.Step(w, nil)
		sim.Step(cw, nil)
		if same("after one tick") && before {
			c.cold++
		}
		if w.Hash() == cw.Hash() {
			c.hashed++
		} else if moved {
			t.Log("the cold World's hash differs after one tick")
		} else {
			t.Error("the cold World's hash differs after one tick")
		}
	}
	walkOriginalSaveCorpus(t, corpus, func(t *testing.T, mf milestone2File) {
		if !mf.present {
			return
		}
		kind := "original"
		if mf.f.HasNativeActionsLeaf() {
			kind = "engine"
		}
		check(t, kind, mf.rel, mf.raw, mf.f)
	})
	installed, err := filepath.Glob(filepath.Join(assets, "*.sav"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range installed {
		t.Run("install/"+filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := sav.Open(raw)
			if err != nil || f.World == nil {
				t.Logf("not a world-half SAV: %v", err)
				return
			}
			check(t, "install", "install/"+filepath.Base(path), raw, f)
		})
	}
	for _, kind := range []string{"original", "engine", "install"} {
		if c := counts[kind]; c != nil {
			t.Logf("%s: %d world-half file(s), %d refused, %d match the replay, %d keep owner ids, %d cold LOAD equal, %d equal hash after one tick, %d moved Item or Effect ids, %d keep every id through a second SAVE",
				kind, c.files, c.refused, c.replayed, c.kept, c.cold, c.hashed, c.otherMoved, c.resaved)
		}
	}
	t.Logf("%d SAV-1371 claim example(s) met", examples)
	if counts["original"] == nil || counts["original"].replayed == 0 {
		t.Fatal("no original corpus file was replayed")
	}
}

// runtimeIDPairs are the corpus LOAD then SAVE pairs whose both sides the
// corpus holds: the file the original loaded and the file it saved.
var runtimeIDPairs = [][2]string{
	{"2026-09-24/exp-engine-lineage/game1017-engine-from-0017.sav", "2026-09-24/exp-engine-lineage/game0001-original-resave1017.sav"},
}

// An owner id the original's SAVE holds that its LOAD did not is one it gave
// after LOAD; the replay of the loaded file predicts the lowest such id.
func TestReleaseRuntimeIDCorpusPairs(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("AGAINROM_SAVE_CORPUS is required")
	}
	read := func(rel string) []runtimeIDFileRecord {
		raw, err := os.ReadFile(filepath.Join(corpus, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		f, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		records, err := runtimeIDFileRecords(f)
		if err != nil {
			t.Fatal(err)
		}
		return records
	}
	for _, pair := range runtimeIDPairs {
		source, target := read(pair[0]), read(pair[1])
		marks, next := runtimeIDFileReplay(source)
		had := map[uint32]bool{}
		for _, r := range source {
			had[r.id] = true
		}
		var created []string
		for _, r := range target {
			if runtimeIDOwnerClasses[r.class] && r.id != 0 && !had[r.id] {
				created = append(created, fmt.Sprintf("%s %d", r.class, r.id))
				// A new id is a bit the loaded file's replay left clear.
				if _, marked := slices.BinarySearch(marks, uint16(r.id)); marked {
					t.Errorf("%s: the original gave %s %d, a bit the replay of %s marks", pair[1], r.class, r.id, pair[0])
				}
			}
		}
		owners, _ := runtimeIDOwners(source)
		targetOwners, _ := runtimeIDOwners(target)
		t.Logf("%s -> %s: %d marks, next %d; %d and %d owner records; created %v", pair[0], pair[1], len(marks), next, len(owners), len(targetOwners), created)
	}
}

// The original builds one Outpost per opcode-0x10003 check at a map's start
// and at World LOAD, each taking two bits (SAV-1373). This engine builds none,
// so it counts the checks in the shipped maps.
func TestReleaseRuntimeIDOutpostChecksInShippedMaps(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	maps := ArchiveMaps(f.Archives.Containers)
	actions, conditions, read := 0, 0, 0
	for _, name := range maps.Names() {
		raw, err := maps.Read(name)
		if err != nil {
			t.Fatal(name, err)
		}
		m, err := alm.Open(raw)
		if err != nil {
			t.Logf("%s: %v", name, err)
			continue
		}
		s, err := m.Script()
		if err != nil {
			t.Logf("%s: script: %v", name, err)
			continue
		}
		read++
		a, c := 0, 0
		for _, n := range s.Actions {
			if n.Opcode == 0x10003 {
				a++
			}
		}
		for _, n := range s.Conditions {
			if n.Opcode == 0x10003 {
				c++
			}
		}
		if a+c != 0 {
			t.Logf("%s: %d action and %d condition node(s) with opcode 0x10003", name, a, c)
		}
		actions, conditions = actions+a, conditions+c
	}
	t.Logf("%d of %d shipped maps read: %d action and %d condition node(s) with opcode 0x10003", read, len(maps.Names()), actions, conditions)
}
