package game_test

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
)

// sfxRegBytes builds a synthetic sfx.reg: [Global] SfxCount — the HIGHEST
// id, not the entry count (REG-SFX-057) — and [Sfx] holding three entries: a
// low id, a SPARSE id far above it, and a value with the original's own
// backslash separator. The three exercise sfxSlotTable's whole rule in one
// fixture: it walks the section's own children rather than looping to
// SfxCount, and it folds a backslash to a forward slash and appends .wav.
func sfxRegBytes() []byte {
	return synth.Reg(0x11, []synth.RegNode{
		{Name: "Global", Kind: 0x01, Children: []synth.RegNode{
			{Name: "SfxCount", Kind: 0x02, Int: 564},
		}},
		{Name: "Sfx", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Sfx1", Kind: 0x00, Str: "click00"},
			{Name: "Sfx57", Kind: 0x00, Str: `units\sword`},
			{Name: "Sfx564", Kind: 0x00, Str: `ambient\river`},
		}},
	})
}

// tinyWAV is a minimal 16-bit mono RIFF/WAVE stream, two frames at an
// arbitrary source rate — built only to give audio.DecodeWAV something it
// accepts. What it decodes TO is not this file's contract to assert on;
// pkg/audio's own tests own that. It exists here, rather than importing a
// helper from pkg/audio, because that package exports no encoder — only
// DecodeWAV, the consumer side.
func tinyWAV() []byte {
	pcm := []int16{0, 4000}
	data := make([]byte, 2*len(pcm))
	for i, v := range pcm {
		binary.LittleEndian.PutUint16(data[2*i:], uint16(v))
	}
	fm := make([]byte, 16)
	binary.LittleEndian.PutUint16(fm[0:], 1)     // format 1: PCM
	binary.LittleEndian.PutUint16(fm[2:], 1)     // 1 channel
	binary.LittleEndian.PutUint32(fm[4:], 22050) // source rate
	binary.LittleEndian.PutUint32(fm[8:], 22050*2)
	binary.LittleEndian.PutUint16(fm[12:], 2)
	binary.LittleEndian.PutUint16(fm[14:], 16) // 16 bits/sample

	out := append([]byte("RIFF"), le32(uint32(4+8+len(fm)+8+len(data)))...)
	out = append(out, "WAVE"...)
	out = append(out, "fmt "...)
	out = append(out, le32(uint32(len(fm)))...)
	out = append(out, fm...)
	out = append(out, "data"...)
	out = append(out, le32(uint32(len(data)))...)
	out = append(out, data...)
	return out
}

func le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// sfxArchiveBytes packs sfxRegBytes beside two of its three named leaves —
// "click00.wav" and "units/sword.wav" — leaving "ambient/river.wav" (slot
// 564) UNWRITTEN on purpose: it is how this fixture reaches "the registry
// names a slot but the archive does not hold its leaf" without a second
// archive.
func sfxArchiveBytes() []byte {
	return synth.Archive([]synth.File{
		{Path: "sfx.reg", Data: sfxRegBytes()},
		{Path: "click00.wav", Data: tinyWAV()},
		{Path: "units/sword.wav", Data: tinyWAV()},
	})
}

func TestOpenSounds(t *testing.T) {
	t.Run("resolves a low id, a sparse id, and folds backslashes", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{game.SfxArchive: sfxArchiveBytes()})
		bank := game.OpenSounds(dir)
		if bank == nil {
			t.Fatalf("OpenSounds(%s) = nil, want a bank", dir)
		}
		if _, ok := bank.Sample(1); !ok {
			t.Errorf("slot 1 (click00, no subdirectory): Sample = false, want true")
		}
		if _, ok := bank.Sample(57); !ok {
			t.Errorf("slot 57 (units\\sword, a sparse id and a folded backslash): Sample = false, want true")
		}
	})

	t.Run("a slot with no registry entry is silence", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{game.SfxArchive: sfxArchiveBytes()})
		bank := game.OpenSounds(dir)
		if _, ok := bank.Sample(2); ok {
			t.Errorf("slot 2 names no [Sfx] entry, Sample = true, want false")
		}
	})

	t.Run("a slot naming a leaf the archive does not hold is silence, repeatedly", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{game.SfxArchive: sfxArchiveBytes()})
		bank := game.OpenSounds(dir)
		for i := 0; i < 2; i++ {
			if _, ok := bank.Sample(564); ok {
				t.Fatalf("call %d: slot 564 names a leaf the archive lacks, Sample = true, want false", i)
			}
		}
	})

	t.Run("a missing archive answers nil, not an error", func(t *testing.T) {
		dir := installDir(t, nil)
		if bank := game.OpenSounds(dir); bank != nil {
			t.Errorf("OpenSounds with no sfx.res = %v, want nil (spec FR-10, P-2)", bank)
		}
	})

	t.Run("an archive that will not open answers nil", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{game.SfxArchive: []byte("not a .res archive")})
		if bank := game.OpenSounds(dir); bank != nil {
			t.Errorf("OpenSounds over a malformed archive = %v, want nil", bank)
		}
	})

	t.Run("an archive with no registry entry answers nil", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{
			game.SfxArchive: synth.Archive([]synth.File{{Path: "other.bin", Data: []byte("x")}}),
		})
		if bank := game.OpenSounds(dir); bank != nil {
			t.Errorf("OpenSounds with no sfx.reg entry = %v, want nil", bank)
		}
	})

	t.Run("a registry that will not parse answers nil", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{
			game.SfxArchive: synth.Archive([]synth.File{{Path: "sfx.reg", Data: []byte("not a registry")}}),
		})
		if bank := game.OpenSounds(dir); bank != nil {
			t.Errorf("OpenSounds with an unparseable registry = %v, want nil", bank)
		}
	})

	t.Run("a nil bank is a safe, silent SoundBank (P-2)", func(t *testing.T) {
		var bank *game.SoundBank
		if _, ok := bank.Sample(1); ok {
			t.Errorf("(*SoundBank)(nil).Sample(1) = true, want false")
		}
	})
}

// unitSoundClass is one [UnitN] section's keys for LoadUnitSounds' own two
// fields: ID always, Sound only when sound is non-nil (an absent key rather
// than an empty array, so the "carries none" case is the registry's actual
// absence and not a zero-length array standing in for it), and AttackDelay
// always.
func unitSoundClass(id int32, sound []int32, delay int32) []synth.RegNode {
	keys := []synth.RegNode{{Name: "ID", Kind: 0x02, Int: id}}
	if sound != nil {
		keys = append(keys, synth.RegNode{Name: "Sound", Kind: 0x06, Ints: sound})
	}
	return append(keys, synth.RegNode{Name: "AttackDelay", Kind: 0x02, Int: delay})
}

func TestLoadUnitSounds(t *testing.T) {
	t.Run("keeps the sound array and the attack delay LoadUnits drops", func(t *testing.T) {
		unitsReg := synth.UnitsReg(nil,
			unitSoundClass(1, []int32{11, 0, 22, 33, 0}, 7),
			unitSoundClass(2, nil, 0),
		)
		fs := openContainers(t, synth.Archive([]synth.File{
			{Path: graphicsEntry(t, game.UnitRegistry), Data: unitsReg},
		}))

		got := game.LoadUnitSounds(fs)
		want := map[int32]game.UnitSound{
			1: {Slots: []int32{11, 0, 22, 33, 0}, AttackDelay: 7},
			2: {Slots: nil, AttackDelay: 0},
		}
		if len(got) != len(want) {
			t.Fatalf("LoadUnitSounds returned %d classes, want %d: %+v", len(got), len(want), got)
		}
		for id, w := range want {
			g, ok := got[id]
			if !ok {
				t.Fatalf("class %d missing from the result: %+v", id, got)
			}
			if !slices.Equal(g.Slots, w.Slots) || g.AttackDelay != w.AttackDelay {
				t.Errorf("class %d = %+v, want %+v", id, g, w)
			}
		}
	})

	t.Run("a nil source answers a nil map", func(t *testing.T) {
		if got := game.LoadUnitSounds(nil); got != nil {
			t.Errorf("LoadUnitSounds(nil) = %v, want nil", got)
		}
	})

	t.Run("an archive with no units.reg entry answers a nil map", func(t *testing.T) {
		fs := openContainers(t, synth.Archive(nil))
		if got := game.LoadUnitSounds(fs); got != nil {
			t.Errorf("LoadUnitSounds with no units.reg = %v, want nil", got)
		}
	})

	t.Run("a registry that will not parse answers a nil map", func(t *testing.T) {
		fs := openContainers(t, synth.Archive([]synth.File{
			{Path: graphicsEntry(t, game.UnitRegistry), Data: []byte("not a registry")},
		}))
		if got := game.LoadUnitSounds(fs); got != nil {
			t.Errorf("LoadUnitSounds with an unparseable registry = %v, want nil", got)
		}
	})
}
