package game

import (
	"path/filepath"
	"strconv"
	"strings"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// A landed blow's SOUND (0126): the archive this tier resolves a slot
// through, the per-class table a swing and a grunt are looked up in, and the
// process-wide configuration cmd/againrom's two flags write.
//
// pkg/audio decodes and mixes with no notion of a slot, an archive or a
// class; pkg/ui's sound.go decides WHICH slot plays, for whom, how often and
// where, with no notion of a registry or a file. This file is the seam
// between them: it is what turns "slot 57" into bytes, and what turns a
// class's own registry section into the two numbers the swing and the grunt
// rules read (spec Terms, plan T3).

// SfxArchive is the sound archive under the asset root — sfx.res — the
// fourth archive this front-end knows the name of and the one OpenArchives
// does NOT open (archives.go's own OpenArchives comment): a missing sfx.res
// is a quiet game, not a broken one, so it is resolved on its own by
// OpenSounds below rather than folded into the required set.
const SfxArchive = "sfx.res"

// sfxPrefix is the address prefix of everything this file reads out of
// SfxArchive: that container's identity segment — SfxArchive's own stem,
// by the derivation graphicsPrefix already states its reason in
// (archives.go) — and the separator.
//
// It is spelt ONCE, here, beside the archive name it belongs to, and both
// addresses this file builds are composed from it: SfxRegistry below, and
// every leaf SoundBank.decode resolves. A second spelling elsewhere is how a
// renamed container would leave one of the two naming an identity nothing in
// the set answers (applied to the archive this story adds).
const sfxPrefix = "sfx/"

// SfxRegistry is the slot table's own address — sfx.res's identity segment
// plus its registry entry — UnitRegistry's own shape (units.go) applied to
// this archive.
const SfxRegistry = sfxPrefix + "sfx.reg"

// OpenSounds opens the sound archive alone and resolves its slot table:
// which archive-relative, extension-bearing address answers which slot
// (REG-SFX-057, High, plan T3).
//
// IT RETURNS NIL ON EVERY FAILURE AND REPORTS NONE. A root with no sfx.res,
// an sfx.res whose registry will not read, and one whose registry will not
// parse are three different causes with one answer: a nil *SoundBank IS the
// silent state this whole subsystem degrades to, and there is deliberately
// no error return here for some future caller to forward correctly for one
// failure and forget for another — the caller cannot tell them apart, on
// purpose, because the game must not either.
//
// NOTHING IS READ OR DECODED BEYOND THE REGISTRY ITSELF: this call resolves
// addresses and reads no leaf, so opening a sound bank costs one archive
// open and one small registry parse regardless of how many of the corpus's
// slots a session ever plays.
func OpenSounds(root string) *SoundBank {
	containers, err := OpenContainers(filepath.Join(root, SfxArchive))
	if err != nil {
		return nil
	}
	stream, err := containers.ReadFile(SfxRegistry)
	if err != nil {
		return nil
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return nil
	}
	return &SoundBank{src: containers, entries: sfxSlotTable(r), cache: make(map[int]soundCacheEntry)}
}

// SoundBank is this tree's one implementation of ui.SoundBank (spec Terms;
// plan T2's own interface, this file's implementation): OpenSounds' resolved
// slot table, plus a per-slot decode cache.
//
// EVERY METHOD IS SAFE ON A NIL RECEIVER, and that is load-bearing rather
// than defensive habit. OpenSounds returns nil on every failure, and
// (*ui.Viewer).SetAudio takes its bank as the ui.SoundBank INTERFACE — so
// a FrontEnd handing over a nil *SoundBank is boxing a NIL POINTER INSIDE A
// NON-NIL INTERFACE VALUE, the classic Go trap: an interface equals nil only
// when BOTH its type and its value are nil, and the moment a *SoundBank(nil)
// crosses into a SoundBank-typed variable the type half is no longer nil,
// whatever the pointer inside is. pkg/ui's own playSlotAt guards
// `v.soundBank == nil` before ever calling Sample (sound.go, T2), and that
// guard cannot see through the boxing from this side — so the one thing
// standing between a failed archive open and a nil-pointer panic on the very
// first grunt is Sample testing its OWN receiver, first, below, rather than
// trusting a caller across the interface to have tested it already.
type SoundBank struct {
	src     *vfs.FS
	entries map[int]string // slot -> archive-relative address, .wav appended
	cache   map[int]soundCacheEntry
	named   map[string]soundCacheEntry
}

// NamedSample is namedSample for pkg/ui's chrgen requests (ui.NamedSoundBank).
func (b *SoundBank) NamedSample(path string) (audio.Sample, bool) {
	return b.namedSample(path)
}

// namedSample resolves a directly named interface sound such as TOWN-147's
// Rotate.wav. Such a filename is not evidence for an invented registry slot.
// The archive and its absence policy are shared with numbered sounds.
func (b *SoundBank) namedSample(path string) (audio.Sample, bool) {
	if b == nil {
		return audio.Sample{}, false
	}
	path = strings.ToLower(strings.ReplaceAll(path, `\`, "/"))
	if c, ok := b.named[path]; ok {
		return c.sample, c.ok
	}
	if b.named == nil {
		b.named = make(map[string]soundCacheEntry)
	}
	c := soundCacheEntry{}
	if b.src != nil {
		if raw, err := b.src.ReadFile(sfxPrefix + path); err == nil {
			c.sample, err = audio.DecodeWAV(raw, audio.DeviceRate)
			c.ok = err == nil
		}
	}
	b.named[path] = c
	return c.sample, c.ok
}

// VoiceSample satisfies ui.VoiceBank: one recording of a human voice bank by
// its name inside the archive, such as "mf_hero/easy.wav" (ANIM-094).
func (b *SoundBank) VoiceSample(name string) (audio.Sample, bool) {
	return b.namedSample(name)
}

// soundCacheEntry is one slot's memoised answer: the decoded sample and
// whether decoding it succeeded. Both are cached, not only the sample — see
// Sample's own comment for why a failure is worth remembering too.
type soundCacheEntry struct {
	sample audio.Sample
	ok     bool
}

// Sample answers slot's decoded sound, lazily, satisfying ui.SoundBank.
//
// A SLOT IS READ AND DECODED ON FIRST PLAY AND NEVER AT OPEN: OpenSounds
// resolves only addresses, and this is the one place a leaf's bytes are read
// and audio.DecodeWAV is ever called for it — at audio.DeviceRate, so the
// corpus resamples on the way in and every Sample this bank returns already
// plays at the rate the caller's device opened with.
//
// THE FAILURE IS CACHED, NOT ONLY THE SUCCESS: a slot with no registry
// entry, a leaf the archive does not hold, and a stream the decoder refuses
// all answer false and are all remembered on the first ask, so a class whose
// swing names a bad leaf costs one failed decode per process and not one per
// swing.
func (b *SoundBank) Sample(slot int) (audio.Sample, bool) {
	if b == nil {
		return audio.Sample{}, false
	}
	if c, cached := b.cache[slot]; cached {
		return c.sample, c.ok
	}
	s, ok := b.decode(slot)
	b.cache[slot] = soundCacheEntry{sample: s, ok: ok}
	return s, ok
}

// decode is Sample's uncached half: the registry-resolved address, read and
// decoded, or false at either step.
func (b *SoundBank) decode(slot int) (audio.Sample, bool) {
	path, ok := b.entries[slot]
	if !ok {
		return audio.Sample{}, false
	}
	data, err := b.src.ReadFile(sfxPrefix + path)
	if err != nil {
		return audio.Sample{}, false
	}
	s, err := audio.DecodeWAV(data, audio.DeviceRate)
	if err != nil {
		return audio.Sample{}, false
	}
	return s, true
}

// sfxSlotTable is the archive-relative, extension-bearing address for every
// slot the registry's [Sfx] section names (REG-SFX-057, High): the section
// holds Sfx<n> string entries whose VALUES are archive paths WITH NO
// EXTENSION and the original's own backslash separator, sparse over a
// highest id far past the count actually present — 115 of a highest slot
// of 564 on the shipped corpus.
//
// IT WALKS THE SECTION'S OWN CHILDREN rather than looping ids up to [Global]
// SfxCount and calling (*reg.Reg).GetString per id. SfxCount IS THE HIGHEST
// ID AND NOT THE ENTRY COUNT (REG-SFX-057) — a naive build that sized a loop
// or an array from it would be sizing from the wrong quantity — so a loop
// bound taken from it would still have to tolerate 449 misses on the shipped
// registry to find the 115 hits; walking the section directly costs exactly
// what it finds, and never reads [Global] at all. A child that is not a
// string, or whose name is not "Sfx" followed by a positive decimal, is
// skipped rather than treated as a malformed registry: nothing in the
// contract says the section holds nothing else, and this function's job is
// the slots it CAN resolve, not a validator for the ones it cannot.
func sfxSlotTable(r *reg.Reg) map[int]string {
	sec := regSection(r, "Sfx")
	if sec == nil {
		return nil
	}
	out := make(map[int]string, len(sec.Children))
	for _, n := range sec.Children {
		if n.Dir || n.Type != reg.TypeString || n.Str == "" {
			continue
		}
		id, ok := sfxSlotID(n.Name)
		if !ok {
			continue
		}
		out[id] = strings.ReplaceAll(n.Str, `\`, "/") + ".wav"
	}
	return out
}

// regSection finds r's top-level child named name, matching ASCII case
// insensitively — pkg/formats/reg's own findChild convention
// (lookup.go), reimplemented here because that fold is unexported and this
// function reads Root directly rather than this story adding a second
// exported accessor to a leaf package for its one caller.
func regSection(r *reg.Reg, name string) *reg.Node {
	if r == nil || r.Root == nil {
		return nil
	}
	for _, c := range r.Root.Children {
		if c.Dir && strings.EqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

// sfxSlotID parses a [Sfx] section key into its slot id: "Sfx" followed by a
// positive decimal, case-insensitively on the prefix since findChild's own
// fold is (lookup.go). Anything else — a differently spelled key, or a zero
// or negative id no slot can be — is refused rather than guessed at.
func sfxSlotID(name string) (int, bool) {
	if len(name) <= 3 || !strings.EqualFold(name[:3], "sfx") {
		return 0, false
	}
	id, err := strconv.Atoi(name[3:])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

type UnitSound struct {
	Slots       []int32
	AttackDelay int32
}

// LoadUnitSounds is the two units.reg fields LoadUnits drops on the way to the
// render tier's bundle: a class's sound-slot array and its attack delay, keyed
// by class ID. A front end takes it from its install share, which derives it
// and the unit set from one parse.
//
// IT RETURNS NO ERROR, on OpenSounds' own rule: a nil src, an unreadable
// registry and one that will not parse all yield a nil map, and a nil map
// answers every class lookup with "no class" — the swing's silent state.
func LoadUnitSounds(src terrain.EntrySource) map[int32]UnitSound {
	classes, err := loadUnitClasses(src)
	if err != nil {
		return nil
	}
	return unitSounds(classes)
}

func unitSounds(classes *data.UnitClasses) map[int32]UnitSound {
	all := classes.All()
	out := make(map[int32]UnitSound, len(all))
	for _, c := range all {
		out[c.ID] = UnitSound{Slots: c.Sound, AttackDelay: c.AttackDelay}
	}
	return out
}

// SoundOptions is the master volume and the mute in ONE NAMED PLACE:
// cmd/againrom's -sound and -volume flags write it, through SetSoundOptions,
// and NewFrontEnd reads it when it opens the effects and music devices —
// so neither term is a literal at any call site in this tree.
type SoundOptions struct {
	Enabled bool
	// Volume is audio.Settings.Master's own unit, audio.MasterUnit — 100 is
	// full and 0 is silent — so cmd/againrom's -volume flag reaches
	// audio.Stereo with no translation (plan T3).
	Volume int
}

// soundOptions is this process's sound configuration: PartySkillSlot's own
// shape (hero.go), for the reason stated there. It is package state and not
// a NewFrontEnd parameter, because opening the sound device is one step deep
// inside that constructor, and widening its signature for two numbers a
// caller sets once, if at all, would touch every hand-built front-end this
// package's own tests construct, for a feature none of them exercises. The
// contract is partySkillSlot's own: a caller sets it, if at all, before
// NewFrontEnd runs and never again — nothing downstream re-reads it
// mid-mission.
var soundOptions = SoundOptions{Enabled: true, Volume: audio.MasterUnit}

// DIV-505
func SetSoundOptions(o SoundOptions) {
	soundOptions = o
}
