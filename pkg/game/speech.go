package game

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/namefold"
	"againrom/pkg/vfs"
)

// SpeechBank indexes the selected install lazily and retains only the last
// decoded line. Missing speech never prevents its text from being read.
type SpeechBank struct {
	root string
	once sync.Once
	src  *vfs.FS
	last string
	line soundCacheEntry
}

func OpenSpeech(root string) *SpeechBank { return &SpeechBank{root: root} }

// has reports whether the bank holds name.
func (b *SpeechBank) has(name string) bool {
	_, found := b.Sample(name)
	return found
}

func (b *SpeechBank) Sample(name string) (audio.Sample, bool) {
	if b == nil || !strings.HasPrefix(name, "speech/") || path.Clean(name) != name || strings.ContainsAny(name, `\:`) {
		return audio.Sample{}, false
	}
	b.once.Do(func() {
		file, err := cutsceneInstallPath(b.root, "speech.res")
		if err == nil {
			b.src, _ = vfs.OpenFileBacked([]string{file}, nil)
		}
	})
	if b.src == nil {
		return audio.Sample{}, false
	}
	if b.last == name {
		return b.line.sample, b.line.ok
	}
	b.last, b.line = name, soundCacheEntry{}
	for _, entry := range b.src.Entries() {
		if entry.Address == name && entry.Size > 0 && entry.Size <= 16<<20 {
			if raw, err := b.src.ReadFile(name); err == nil {
				b.line.sample, err = audio.DecodeWAV(raw, audio.DeviceRate)
				b.line.ok = err == nil && len(b.line.sample.PCM) > 0
			}
			break
		}
	}
	return b.line.sample, b.line.ok
}

// TownSpeechPath uses the same selected tag as the displayed text. The
// basename/part convention and sound= override are DLG-SOUND-028.
func TownSpeechPath(textPath string, payload []byte, part int, audience EventAudience) (string, bool) {
	tag, _, ok := eventPartTag(payload, part, audience)
	if !ok {
		return "", false
	}
	return townSpeechTagPath(firstTownTalkProfile.speech, textPath, tag, part, audience)
}

// townSpeechNames is one game's rule for the recording a town dialogue page
// asks speech.res for. A rule with families names it from the section key the
// page was cut from; one without, from the text file's leaf.
type townSpeechNames struct {
	// leafDirs are the main/text/ directories whose pages name their leaf.
	leafDirs []string
	// families are the section-key words in the order they are tried.
	families []townSpeechFamily
	// heroNPC's pages and a header with an iam* token take heroLetters at
	// the hero's herokind index; other pages take plainLetter.
	heroNPC     int
	heroLetters string
	plainLetter byte
}

// townSpeechFamily is a key word's recording, dir/<npc><middle>p<part>: the
// header's NPC in four digits, then fixed or, when fixed is "", the page's
// letter and the key's trailing digits padded with 0 to three.
type townSpeechFamily struct{ word, dir, fixed string }

// townSpeechName is speech/<dir><stem>p<part>, part padded with 0 to width;
// sound= replaces <stem>p<part> in a leaf scope, else <dir><stem>p<part>.
type townSpeechName struct {
	dir, stem string
	width     int
	leaf      bool
}

// townSpeechTagPath is the one name of a town page's recording: source is the
// text path or the section key, tag the page's accepted header.
func townSpeechTagPath(names townSpeechNames, source, tag string, part int, audience EventAudience) (string, bool) {
	if part < 1 {
		return "", false
	}
	var n townSpeechName
	if names.families == nil {
		textPath := namefold.ArchiveText(strings.ReplaceAll(source, `\`, "/"))
		relative, ok := strings.CutPrefix(textPath, "main/text/")
		if !ok || path.Clean(textPath) != textPath || !slices.ContainsFunc(names.leafDirs, func(d string) bool { return strings.HasPrefix(relative, d+"/") }) {
			return "", false
		}
		n = townSpeechName{dir: path.Dir(relative) + "/", stem: strings.TrimSuffix(path.Base(relative), ".txt"), width: 1, leaf: true}
	} else {
		n = names.sectionName(namefold.ArchiveText(source), tag, audience)
	}
	return n.path(tag, part)
}

func (n townSpeechName) path(tag string, part int) (string, bool) {
	dir, name := n.dir, ""
	if n != (townSpeechName{}) {
		name = fmt.Sprintf("%sp%0*d", n.stem, n.width, part)
	}
	if value := townSpeechOverride(tag); value != "" {
		name = namefold.ArchiveText(strings.ReplaceAll(value, `\`, "/"))
		if !n.leaf {
			dir = ""
		}
	}
	full := "speech/" + dir + name + ".wav"
	if name == "" || path.Clean(name) != name || path.Clean(full) != full || strings.Contains(name, ":") || n.leaf && strings.Contains(name, "/") {
		return "", false
	}
	return full, true
}

// sectionName is the family default of key's page, zero when key has no family.
func (names townSpeechNames) sectionName(key, tag string, audience EventAudience) townSpeechName {
	at := slices.IndexFunc(names.families, func(f townSpeechFamily) bool { return strings.Contains(key, f.word) })
	if at < 0 {
		return townSpeechName{}
	}
	f := names.families[at]
	npc, _ := audience.tagRules().speaker(tag)
	middle := f.fixed
	if middle == "" {
		letter := names.plainLetter
		if npc == names.heroNPC || slices.ContainsFunc(eventArms[:4], func(a eventArm) bool { return containsFold(tag, a.lit) }) {
			letter = names.heroLetters[herokind.Of(audience.HeroMage, audience.HeroFemale).Index()]
		}
		digits := key[len(strings.TrimRight(key, "0123456789")):]
		middle = string(letter) + strings.Repeat("0", max(0, 3-len(digits))) + digits
	}
	return townSpeechName{dir: f.dir + "/", stem: fmt.Sprintf("%04d%s", npc, middle), width: 2}
}

func townSpeechOverride(tag string) string {
	at := indexFold(tag, "sound=")
	if at < 0 {
		return ""
	}
	value := strings.TrimSpace(tag[at+len("sound="):])
	end := ";"
	if strings.HasPrefix(value, `"`) {
		value, end = value[1:], `"`
	}
	if n := strings.Index(value, end); n >= 0 {
		value = value[:n]
	}
	return value
}

// townSpeechSource is what the profile's speech rule names a page's recording
// from: the open section's key under families, else the text path.
func (t *townScreen) townSpeechSource() (string, bool) {
	if t.talkProfile().speech.families == nil {
		return t.townTextPath()
	}
	if c := t.campaignTalk(); c != nil && c.key != "" {
		return c.key, true
	}
	return "", false
}

func (t *townScreen) townSpeechPath() (string, bool) {
	textPath, ok := t.townSpeechSource()
	if !ok {
		return "", false
	}
	tag := t.dialogue.tag
	name, ok := townSpeechTagPath(t.talkProfile().speech, textPath, tag, t.dialogue.displayPart, t.talkAudience())
	if !ok {
		return "", false
	}
	speaker, named := tagSpeaker(tag)
	rec, known := t.in.NPCFaces[int32(speaker)]
	if !strings.HasPrefix(textPath, "main/text/inn/npc/") || !named || !known || rec.Kind != data.NPCNoPicture || townSpeechOverride(tag) != "" {
		return name, true
	}
	// The installed hero recordings replace the "npc" prefix with the
	// speaker's sex/class pair (ff, fm, mf, mm). Reuse the figure predicate
	// to choose that speaker from the carried party, not the selected doll.
	cast := newSpeakerCast(t.sess.Carried, nil)
	actor, found := cast.resolveNPC(int32(speaker), rec)
	leaf := path.Base(name)
	if !found || !strings.HasPrefix(leaf, "npc") {
		return name, true
	}
	sex, class := "m", "f"
	if actor.fig.Dir.Female() {
		sex = "f"
	}
	if actor.fig.Dir.Mage() {
		class = "m"
	}
	// A speaker whose sex/class recording the install lacks plays the
	// pager's own leaf name (DLG-SOUND-028, DLG-VOICE-074).
	if hero := path.Dir(name) + "/" + sex + class + "_" + strings.TrimPrefix(leaf, "npc"); t.in.SpeechBank.has(hero) {
		return hero, true
	}
	return name, true
}

type townSpeech struct {
	key     string
	voice   audio.Voice
	paused  bool
	room    townRoom
	pending []string
}

func (t *townScreen) stopTownSpeech() {
	t.speech.pending = nil
	t.speech.paused = false
	if t.speech.voice != nil {
		audio.DestroyVoice(t.speech.voice)
		t.speech.voice = nil
	}
}

func (t *townScreen) pauseTownSpeech() {
	if t.speech.paused || t.speech.voice == nil {
		return
	}
	if voice, ok := t.speech.voice.(interface{ Pause() bool }); ok {
		t.speech.paused = voice.Pause()
	} else {
		t.stopTownSpeech()
	}
}

func (t *townScreen) resumeTownSpeech() bool {
	if !t.speech.paused {
		return true
	}
	if voice, ok := t.speech.voice.(interface{ Resume() bool }); ok && voice.Resume() {
		t.speech.paused = false
		return true
	}
	return false
}

func (t *townScreen) resetTownSpeech() {
	t.stopTownSpeech()
	t.speech = townSpeech{}
}

// TownDialogueActive is driven by App after input. Rendering cannot start or
// repeat a line; changing part, leaving the dialogue or exiting stops it.
func (t *townScreen) TownDialogueActive(active bool) {
	if t != nil && !t.hostRules().DialogueSpeech {
		return
	}
	if t == nil || t.sess == nil {
		return
	}
	if !active {
		t.pauseTownSpeech()
		return
	}
	if !t.campaignTown().talking() {
		if t.speech.room != t.room {
			t.stopTownSpeech()
		} else if t.resumeTownSpeech() {
			t.advanceTownResponse()
		}
		return
	}
	if t.talkDiagnostic != "" {
		t.stopTownSpeech()
		return
	}
	textPath, ok := t.townSpeechSource()
	if !ok {
		t.stopTownSpeech()
		return
	}
	key := fmt.Sprintf("%s#%d", textPath, t.dialogue.displayPart)
	if t.speech.key == key {
		if !t.resumeTownSpeech() {
			return
		}
		if t.speech.voice != nil && !t.speech.voice.Playing() {
			t.stopTownSpeech()
		}
		return
	}
	t.stopTownSpeech()
	t.speech.key, t.speech.room = key, roomTalk
	name, ok := t.townSpeechPath()
	if !ok {
		return
	}
	t.startTownVoice(name)
}

func (t *townScreen) startTownVoice(name string) bool {
	if sample, found := t.in.SpeechBank.Sample(name); found {
		source := "town-dialogue"
		if t.room != roomTalk {
			source = "town-response"
		}
		t.speech.voice = audio.Dispatch(t.roomSoundPlayer(audio.SpeechChannel), sample,
			audio.FixedRequest(source, name, audio.SpeechChannel, 128, false,
				audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	}
	return t.speech.voice != nil
}
