package game

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"againrom/pkg/audio"
	"againrom/pkg/data"
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
	return townSpeechTagPath(textPath, tag, part)
}

func townSpeechTagPath(textPath, tag string, part int) (string, bool) {
	if part < 1 {
		return "", false
	}
	textPath = strings.ToLower(strings.ReplaceAll(textPath, `\`, "/"))
	if !strings.HasPrefix(textPath, "main/text/") || path.Clean(textPath) != textPath {
		return "", false
	}
	relative := strings.TrimPrefix(textPath, "main/text/")
	if !strings.HasPrefix(relative, "inn/") && !strings.HasPrefix(relative, "shop/") && !strings.HasPrefix(relative, "training/") {
		return "", false
	}
	name := fmt.Sprintf("%sp%d", strings.TrimSuffix(path.Base(relative), ".txt"), part)
	if value := townSpeechOverride(tag); value != "" {
		name = strings.ToLower(strings.ReplaceAll(value, `\`, "/"))
	}
	if name == "" || path.Clean(name) != name || strings.ContainsAny(name, ":/") {
		return "", false
	}
	return "speech/" + path.Dir(relative) + "/" + name + ".wav", true
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

func (t *townScreen) townSpeechPath() (string, bool) {
	textPath, ok := t.townTextPath()
	if !ok {
		return "", false
	}
	tag := t.dialogue.tag
	name, ok := townSpeechTagPath(textPath, tag, t.dialogue.displayPart)
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
	cast := speakerCast{}
	for _, member := range t.sess.Carried {
		if primaryPlayerHero(member) {
			cast.playerDir, _ = memberFigure(member)
			cast.hasPlayer = true
		}
	}
	for _, member := range t.sess.Carried {
		dir, face := memberFigure(member)
		actor := speakerActor{me: primaryPlayerHero(member), hero: member.PlayerCharacter && member.MercenaryType == 0,
			fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: member.Class}
		if !cast.matches(rec, actor) {
			continue
		}
		sex, class := "m", "f"
		if dir.Female() {
			sex = "f"
		}
		if dir.Mage() {
			class = "m"
		}
		leaf := path.Base(name)
		if strings.HasPrefix(leaf, "npc") {
			return path.Dir(name) + "/" + sex + class + "_" + strings.TrimPrefix(leaf, "npc"), true
		}
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
	if t == nil || t.sess == nil {
		return
	}
	if !active {
		t.pauseTownSpeech()
		return
	}
	if t.room != roomTalk {
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
	textPath, ok := t.townTextPath()
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
