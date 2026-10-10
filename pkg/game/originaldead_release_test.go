package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseOriginalDead1100ExactFiveActorsBothDoorsAndNativeContinuation(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-02/game0002.sav", "b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761")
	source1163, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	roots1163, err := dead1163Expected(source1163, payload)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		mapID uint16
		stage uint8
		hp    int16
		x, y  int32
	}{{20, 4, -237, 32, 64}, {19, 4, -209, 24, 56}, {30, 5, -10014, 12, 54}, {36, 5, -10017, 49, 59}, {32, 5, -10011, 52, 47}}
	f.SetDeterministicFrames(true)
	app := f.App("1100-original-dead")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for _, v := range want {
		if !entityRef(poolEntity(t, f.live.world, v.mapID)).Alive() {
			t.Fatal("fresh baseline no longer witnesses resurrection")
		}
	}
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	w := f.live.world
	if d := roots1163.documentDifferences(f.live.mission.state.savedDocument); len(d) != 0 {
		t.Fatal("raw dead roots BEFORE Snapshot", d)
	}
	if d := roots1163.liveDifferences(w.OriginalDeadActors(), w.Entities()); len(d) != 0 {
		t.Fatal("raw dead sources", d)
	}
	assertDeadWorld1100(t, w, want)
	if len(w.Sacks()) != 2 || w.Purse(1) != 100 {
		t.Fatalf("dead import replayed sacks/gold: sacks=%d gold=%d", len(w.Sacks()), w.Purse(1))
	}
	if w.Tick() != rawSavedSubTick1112(t, payload) {
		t.Fatal("import advanced before publication")
	}
	ms, report, err := loadOriginalMission(f, payload)
	if err != nil || report.CorpsesRestored != 2 || report.TerminalRestored != 3 {
		t.Fatalf("diagnostic door: %+v %v", report, err)
	}
	assertDeadWorld1100(t, ms.World, want)
	if !reflect.DeepEqual(w.Sacks(), ms.World.Sacks()) {
		t.Fatal("load doors disagree on bags")
	}
	for i, r := range w.OriginalDeadActors() {
		if r.Source.ArchiveIndex != uint16(90+i) || !r.Source.ContainerPresent || r.Source.ContainerTail != [2]uint32{10000, 0} {
			t.Fatalf("source provenance differs from independent oracle: %+v", r)
		}
	}
	for range 33 {
		f.live.tick()
	}
	records := w.OriginalDeadActors()
	if records[0].Current.HP >= -237 {
		t.Fatal("corpse did not decay")
	}
	seam := originalDeadSeam(w)
	store, name, written := menuSAVE(t, f, app, OriginalStore{})
	requireWrittenDead(t, written, records)
	fresh := loadLocalLegacySave(t, store, name)
	if got := originalDeadSeam(fresh.live.world); got != seam {
		t.Fatalf("SAV LOAD changed dead state:\n%s\n%s", seam, got)
	}
	if d := roots1163.currentRootDifferences(fresh.live.mission.state.savedDocument); len(d) != 0 {
		t.Fatal("loaded dead roots", d)
	}
	for tick := range 64 {
		f.live.tick()
		fresh.live.tick()
		if got, live := originalDeadSeam(fresh.live.world), originalDeadSeam(w); got != live {
			t.Fatalf("tick %d continued dead state diverged:\n%s\n%s", tick, live, got)
		}
	}
	for i := 2; i < 5; i++ {
		if got := fresh.live.world.OriginalDeadActors()[i]; got.Current != records[i].Current {
			t.Fatal("terminal residue changed after ticks")
		}
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool { return setSavedActorHealth(doc, records[0].Current.RuntimeID, -300) })
	if originalDeadSeam(lost.live.world) == seam {
		t.Fatal("loss control: a SAV with the corpse HP altered still matches")
	}
	t.Logf("source=%s fresh authored living=5; loaded late corpses=2 terminal absent=3; exact archive indices=90..94; menu SAVE %s carries decayed HP %d; continued 64 ticks matched", path, name, records[0].Current.HP)
}

func TestReleaseOriginalDead1100TerminalUnitWeaponsRemainInert(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-02/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Independent transcription of the exact shared-archive dead list at
	// body[33995:48603], not a scan of heuristic Unit headers. Entries 0/1
	// are terminal Units with inert held Weapons; all later entries are empty.
	want := []struct {
		mapID uint16
		stage uint8
		hp    int16
		x, y  int32
	}{
		{59, 5, -10007, 32, 27}, {60, 5, -10001, 32, 25},
		{4, 4, -170, 43, 39}, {6, 4, -176, 44, 37}, {5, 4, -166, 41, 39},
		{17, 4, -179, 26, 63}, {18, 4, -155, 41, 75}, {57, 4, -141, 54, 80},
		{36, 4, -169, 53, 84}, {42, 4, -128, 55, 79}, {38, 4, -149, 54, 79},
		{39, 4, -107, 49, 83}, {35, 4, -132, 56, 79}, {34, 4, -124, 57, 79},
		{58, 4, -123, 58, 79}, {40, 4, -101, 49, 80}, {37, 4, -114, 51, 79},
		{41, 4, -106, 53, 79}, {12, 4, -87, 75, 93}, {10, 4, -49, 76, 92},
		{11, 4, -123, 76, 91}, {13, 4, -50, 87, 104},
	}
	f.SetDeterministicFrames(true)
	app := f.App("1100-terminal-unit-weapons")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	for _, v := range want {
		if !entityRef(poolEntity(t, f.live.world, v.mapID)).Alive() {
			t.Fatal("fresh baseline no longer witnesses resurrection")
		}
	}
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	w := f.live.world
	assertDeadWorld1100(t, w, want)
	assertTerminalWeapons1100(t, w, source.Body)
	ms, report, err := loadOriginalMission(f, payload)
	if err != nil || report.CorpsesRestored != 20 || report.TerminalRestored != 2 {
		t.Fatalf("diagnostic door: %+v %v", report, err)
	}
	assertDeadWorld1100(t, ms.World, want)
	assertTerminalWeapons1100(t, ms.World, source.Body)
	if !reflect.DeepEqual(w.Sacks(), ms.World.Sacks()) || !reflect.DeepEqual(w.OriginalDeadActors(), ms.World.OriginalDeadActors()) {
		t.Fatal("load doors disagree on dead source or bags")
	}
	if w.Tick() != rawSavedSubTick1112(t, payload) || len(w.Sacks()) != 10 || w.Purse(1) != 600 {
		t.Fatalf("dead import replayed world/loot: tick=%d sacks=%d gold=%d", w.Tick(), len(w.Sacks()), w.Purse(1))
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if !f.live.mission.delayedVictory || f.live.view.NoticeOpen() {
		t.Fatal("saved Victory did not admit Continue")
	}
	for range 33 {
		f.live.tick()
	}
	records, tick := w.OriginalDeadActors(), w.Tick()
	if tick != rawSavedSubTick1112(t, payload)+33 || records[2].Current.HP >= -170 {
		t.Fatal("prefix did not decay the corpse")
	}
	seam := originalDeadSeam(w)
	store, native, written := menuSAVE(t, f, app, OriginalStore{})
	requireWrittenDead(t, written, records)
	fresh := loadLocalLegacySave(t, store, native)
	if got := originalDeadSeam(fresh.live.world); got != seam {
		t.Fatalf("SAV LOAD changed dead state:\n%s\n%s", seam, got)
	}
	assertTerminalWeapons1100(t, fresh.live.world, source.Body)
	for range 64 {
		f.live.tick()
		fresh.live.tick()
		if got, live := originalDeadSeam(fresh.live.world), originalDeadSeam(w); got != live {
			t.Fatalf("continued dead state diverged:\n%s\n%s", live, got)
		}
		assertTerminalWeapons1100(t, fresh.live.world, source.Body)
	}
	g := fresh.live.world
	if g.Tick() != w.Tick() || g.Purse(1) != 600 || len(g.Sacks()) != 10 {
		t.Fatalf("continuation tick=%d/%d sacks=%d gold=%d", g.Tick(), w.Tick(), len(g.Sacks()), g.Purse(1))
	}
	for i := 0; i < 2; i++ {
		if got := g.OriginalDeadActors()[i]; got.Current != records[i].Current || got.Source.HeldWeapon != records[i].Source.HeldWeapon {
			t.Fatal("terminal Unit/Weapon provenance changed during gameplay")
		}
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool { return setSavedActorHealth(doc, records[2].Current.RuntimeID, -300) })
	if originalDeadSeam(lost.live.world) == seam {
		t.Fatal("loss control: a SAV with the corpse HP altered still matches")
	}
	t.Logf("source=%s fresh living=22; loaded corpses=20 terminal absent=2; Weapon archives=110/112 identities=0x2ca4c80/0x2ca7640 T0E=715/28769; all 98 bytes retained; sacks10 gold600; menu SAVE %s at tick %d; 64 ticks matched", path, native, tick)
}

func originalDeadSeam(w *sim.World) string {
	var b strings.Builder
	for _, r := range w.OriginalDeadActors() {
		fmt.Fprintf(&b, "mu%d arch%d %+v", r.Source.MapUnitID, r.Source.ArchiveIndex, r.Current)
		for _, e := range w.Entities() {
			if e.MapUnitID == r.Source.MapUnitID {
				fmt.Fprintf(&b, " decay%d hp%d at%d,%d alive%t", e.Decay, e.HP, e.X, e.Y, e.Alive())
			}
		}
		b.WriteString(";\n")
	}
	return b.String()
}

func requireWrittenDead(t *testing.T, raw []byte, records []sim.OriginalDeadRecord) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := file.DeadActors()
	if err != nil || len(dead) != len(records) {
		t.Fatalf("written dead population %d, live %d: %v", len(dead), len(records), err)
	}
	for i, d := range dead {
		r := records[i]
		if d.MapUnitID != r.Source.MapUnitID || d.RuntimeID != r.Current.RuntimeID || d.Stage != r.Current.Stage || d.HP != r.Current.HP || d.Cell != r.Current.Cell {
			t.Fatalf("written dead[%d] %+v, live %+v", i, d, r.Current)
		}
	}
}

func setSavedActorHealth(doc *sav.DocumentData, runtime uint32, hp int16) bool {
	for i := range doc.Objects {
		if id, err := savedStructureValue(&doc.Objects[i], "RuntimeID"); err == nil && id == runtime && runtime != 0 && (doc.Objects[i].Class == "Human" || doc.Objects[i].Class == "Unit") {
			return savedStructureSetValue(&doc.Objects[i], "Health", uint32(uint16(hp))) == nil
		}
	}
	return false
}

func assertTerminalWeapons1100(t *testing.T, w *sim.World, body []byte) {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// The independently tested action-clock suffix is outside this Weapon view.
	form = beforeStructureBlockingForm(t, form)
	form = beforeActionClockForm1146(t, form)
	// This witness compares the dead-Weapon byte section, not a historical
	// whole World. Peel both separate bounded suffixes without weakening the
	// genuine predecessor helpers downstream (beforeObjects1115Form via
	// beforeStrideStateForm1115's chain), which must still reject any populated
	// tail on an actual pre-1139/pre-1115 fixture.
	if len(form) >= 5 && form[0] == 85 {
		span := uint64(binary.LittleEndian.Uint32(form[len(form)-4:]))
		if span > uint64(len(form)-5) {
			t.Fatal("invalid current-resume suffix")
		}
		form = append([]byte(nil), form[:len(form)-int(span)-4]...)
		form[0] = 84
	}
	if len(form) >= 5 && form[0] == 84 {
		span := uint64(binary.LittleEndian.Uint32(form[len(form)-4:]))
		if span > uint64(len(form)-5) {
			t.Fatal("invalid current-object suffix")
		}
		form = append([]byte(nil), form[:len(form)-int(span)-4]...)
		form[0] = 83
	}
	form = beforeStrideStateForm1115(t, form)
	records := w.OriginalDeadActors()
	if len(records) != 22 {
		t.Fatal("lost dead source population")
	}
	const stride, prefix, relation = 173, 74, 2500
	// SavedObjects84, SavedCellPlanes83, SavedMotions82 and NativeStrides81
	// are already peeled above, down to form80, before this point (the manual
	// form84 peel plus the beforeStrideStateForm1115 chain). Player80,
	// Structure79 and Group78 are each one more length-prefixed span, appended
	// in that order after form77's clock suffix; peel them independently,
	// outermost first, the same shape as the peel already run above.
	end := len(form)
	for _, name := range []string{"Player", "Structure", "Group"} {
		if end < 4 {
			t.Fatal("missing native suffix", name)
		}
		span := binary.LittleEndian.Uint32(form[end-4:])
		if uint64(span) > uint64(end-4) {
			t.Fatal("invalid native suffix", name)
		}
		end -= int(span) + 4
	}
	const sessionClockSuffixLen = 5 // pkg/sim.sessionClockLen: form77's clock suffix, unexported across packages
	// Form77 appends only a final five-byte clock suffix after the relation
	// trailer, stripped by splitSessionClock before UnmarshalBinary reaches
	// this chain in production; this helper parses the raw MarshalBinary
	// output directly and so must peel that suffix here too. Each section's
	// own 4-byte length word sits immediately before its payload, not before
	// the next section's length word, so loadStart (not loadEnd) is what the
	// weight section's length word precedes — mirroring UnmarshalBinary's own
	// loadEnd/loadStart/weightEnd chain exactly.
	loadEnd := end - sessionClockSuffixLen - relation - 4
	if loadEnd < 4 {
		t.Fatal("missing native actor-load span")
	}
	loadHeader := binary.LittleEndian.Uint32(form[loadEnd:])
	loadStart := loadEnd - int(loadHeader&0x7fffffff)
	weightEnd := loadStart - 4
	if weightEnd < 4 {
		t.Fatal("missing native weight span")
	}
	weightSpan := binary.LittleEndian.Uint32(form[weightEnd:])
	if weightSpan%6 != 0 || uint64(weightSpan) > uint64(weightEnd-4) {
		t.Fatal("wrong native weight span")
	}
	weightStart := weightEnd - int(weightSpan)
	deadEnd := weightStart - 4
	start := deadEnd - len(records)*stride
	if start < 0 || binary.LittleEndian.Uint32(form[deadEnd:]) != 22*stride {
		t.Fatal("wrong native dead span")
	}
	for i, v := range []struct {
		bodyOff  int
		archive  uint16
		identity uint32
		t0e      uint16
	}{{34539, 110, 0x2ca4c80, 715}, {35272, 112, 0x2ca7640, 28769}} {
		weapon := records[i].Source.HeldWeapon
		if !weapon.Present || weapon.ArchiveIndex != v.archive || weapon.Identity != v.identity || weapon.T0E != v.t0e || weapon.F40 != 61720 {
			t.Fatalf("wrong independent weapon identity: %+v", weapon)
		}
		// Fixed source anchors and member lengths are independent of production
		// projection/encoding: Token37, empty Effects4, Item12 + Weapon47,
		// null Spell2. Copying these named source ranges makes every unknown
		// W52/W6A byte observable without committing any lawful-install bytes.
		source := body[v.bodyOff : v.bodyOff+102]
		if !bytes.Equal(source[37:41], []byte{0, 0, 0, 0}) || !bytes.Equal(source[100:102], []byte{0, 0}) {
			t.Fatal("source Weapon left the zero-effects/null-reference boundary")
		}
		want := []byte{byte(v.archive), byte(v.archive >> 8)}
		want = append(want, source[:37]...)
		want = append(want, source[41:100]...)
		at := start + i*stride + prefix
		if form[at] != 1 || !bytes.Equal(form[at+1:at+99], want) {
			t.Fatal("native held-Weapon lost an independently transcribed source member")
		}
		for _, e := range w.Entities() {
			if e.MapUnitID == uint16(59+i) {
				t.Fatal("terminal Unit59/60 revived")
			}
		}
	}
	for i := 2; i < len(records); i++ {
		if records[i].Source.HeldWeapon != (sim.OriginalDeadWeapon{}) || !bytes.Equal(form[start+i*stride+prefix:start+(i+1)*stride], make([]byte, 99)) {
			t.Fatal("absent Weapon acquired state")
		}
	}
}
