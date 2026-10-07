package ui

import (
	"image"
	"math"
	"strconv"
	"time"

	"againrom/pkg/audio"
)

type SoundBank interface {
	Sample(slot int) (audio.Sample, bool)
}

// UISoundSlot is one fixed interface selector recovered by VIDEO-SFX-016.
// These are archive slot numbers, not physical paths and not the 0xdc
// priority argument consumed by the playback terminal (VIDEO-SFX-013).
type UISoundSlot int

const (
	UISoundCampaignPanel   UISoundSlot = 1
	UISoundCommonControl   UISoundSlot = 2
	UISoundBookToggle      UISoundSlot = 7
	UISoundCommandPanel    UISoundSlot = 8
	UISoundMissionComplete UISoundSlot = 14
	UISoundMissionFailed   UISoundSlot = 16
	UISoundOptionsTest     UISoundSlot = 100
)

// playUISound resolves a fixed selector through the same shipped SoundBank as
// map sounds, but plays it centred: an interface event has no map cell from
// which a positional placement could lawfully be derived. Missing device,
// bank or registry entry is the existing silent degradation.
func playUISound(p audio.Player, b SoundBank, slot UISoundSlot) {
	if p == nil || b == nil || slot <= 0 {
		return
	}
	s, ok := b.Sample(int(slot))
	if !ok {
		return
	}
	audio.Dispatch(p, s, audio.FixedRequest("fixed-interface", "registry:"+strconv.Itoa(int(slot)),
		audio.EffectsChannel, 220, false, audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
}

// The hurt events a strike, a wound, a bleed and a fall emit (ANIM-095),
// numbered as the hurt hook's own argument. Event 0 plays Sound[1] of the
// drawn class on the effects device, whoever the entity is. Events 1 to 3 play
// on the speech device: a bank-voiced entity plays hurtLeaves[k] from its Voice
// bank, a class-voiced one Sound[k+1] of its drawn class (ANIM-094,
// ANIM-128).
const (
	hurtNone = 0
	hurtEasy = 1
	hurtHard = 2
	hurtDie  = 3
)

// hurtLeaves names each event's recording in a human voice bank (ANIM-094).
var hurtLeaves = [...]string{hurtEasy: "easy", hurtHard: "hard", hurtDie: "die"}

// GruntThrottle is the hurt hook's gate: a wound is silent within 1500 ms of
// the drawable's last voice, and a voiced wound stores the time (ANIM-094).
const GruntThrottle = 1500 * time.Millisecond

// BleedCeiling is the lowest health of a fallen body that still bleeds: it
// loses one point every fourth full tick from -1 down to -10, and each loss
// is sent to its owner alone (ANIM-126).
const BleedCeiling = -9

// GruntFloor is the stored health at and below which a blow emits no hurt
// event (ANIM-095).
const GruntFloor = -10

// hurtEvent is the event a blow emits, chosen by the health the victim held
// BEFORE it (ANIM-095): none at or below GruntFloor, hard below half the
// maximum, easy otherwise. Half truncates toward zero, as Go's division does.
func hurtEvent(stored, maxHP int) (int, bool) {
	if stored <= GruntFloor {
		return 0, false
	}
	if stored < maxHP/2 {
		return hurtHard, true
	}
	return hurtEasy, true
}

// VoiceBank answers a human voice bank's recording by its name inside the
// sound archive, such as "mf_hero/easy.wav" (ANIM-094). A SoundBank that does
// not implement it leaves every bank-voiced event silent.
type VoiceBank interface {
	VoiceSample(name string) (audio.Sample, bool)
}

// voiceMemory is what the viewer keeps per drawable for its voice: the one
// voice timestamp its wounds and its replies share (ANIM-094), and the
// health and life it held at the last handled message.
// framed is false for an entry only a reply has written.
type voiceMemory struct {
	at      time.Time
	framed  bool
	hp      int
	damaged bool
	alive   bool
}

// claim takes the voice timestamp for a voice that waits gap since the last
// one, and refuses while the gap has not passed.
func (m *voiceMemory) claim(now time.Time, gap time.Duration) bool {
	if !m.at.IsZero() && now.Sub(m.at) < gap {
		return false
	}
	m.at = now
	return true
}

func classSlot(sounds []int, idx int) int {
	if idx < 0 || idx >= len(sounds) {
		return 0
	}
	return sounds[idx]
}

func (v *Viewer) listenerCell() (image.Point, bool) {
	if v.cam == nil {
		return image.Point{}, false
	}
	r := v.cam.VisibleTiles()
	if r.Empty() {
		return image.Point{}, false
	}
	return image.Pt((r.Col0+r.Col1)/2, (r.Row0+r.Row1)/2), true
}

func (v *Viewer) playSlotAt(slot int, owner uint32, cell image.Point) {
	if slot <= 0 || v.soundBank == nil {
		return
	}
	v.PlayEffectAtFine("map-effect", slot, owner, cell, cell.Mul(256).Add(image.Pt(128, 128)))
}

// playSpeechSlotAt is playSlotAt on the speech device (ANIM-128).
func (v *Viewer) playSpeechSlotAt(slot int, owner uint32, cell image.Point) {
	if slot <= 0 || v.soundBank == nil {
		return
	}
	v.placeAndPlay(v.speechPlayer, owner, cell, cell.Mul(256).Add(image.Pt(128, 128)), "unit-hurt", "registry:"+strconv.Itoa(slot), audio.SpeechChannel, func() (audio.Sample, bool) { return v.soundBank.Sample(slot) })
}

// playVoiceAt plays one recording of a human voice bank, named inside the
// sound archive, from cell on the speech device, under playSlotAt's gate and
// placement.
func (v *Viewer) playVoiceAt(name string, owner uint32, cell image.Point) {
	bank, ok := v.soundBank.(VoiceBank)
	if !ok {
		return
	}
	v.placeAndPlay(v.speechPlayer, owner, cell, cell.Mul(256).Add(image.Pt(128, 128)), "unit-hurt", "voice:"+name, audio.SpeechChannel, func() (audio.Sample, bool) { return bank.VoiceSample(name) })
}

// placeAndPlay plays the sample resolve answers at cell, placed against the
// listener, on player. The sample is resolved only after every
// gate has passed.
func (v *Viewer) placeAndPlay(player audio.Player, owner uint32, cell, fine image.Point, source, selector string, group audio.Channel, resolve func() (audio.Sample, bool)) {
	if player == nil {
		return
	}
	if !v.fogGateEntity(owner, cell.X, cell.Y) {
		return
	}
	geometry, ok := v.soundGeometry()
	if !ok {
		return
	}
	s, ok := resolve()
	if !ok {
		return
	}
	request, ok := audio.PositionalRequest(source, selector, group, fine, geometry)
	if !ok {
		return
	}
	audio.Dispatch(player, s, request)
}

func (v *Viewer) soundGeometry() (audio.ViewGeometry, bool) {
	if v.cam == nil || v.cam.Zoom <= 0 {
		return audio.ViewGeometry{}, false
	}
	geometry := audio.ViewGeometry{Origin: image.Pt(int(math.Floor(v.cam.X/32)), int(math.Floor(v.cam.Y/32))),
		Span: image.Pt(int(float64(v.cam.ViewW)/v.cam.Zoom/32), int(float64(v.cam.ViewH)/v.cam.Zoom/32))}
	return geometry, geometry.Span.X > 0 && geometry.Span.Y > 0
}

func (v *Viewer) PlayEffectAtFine(source string, slot int, owner uint32, cell, fine image.Point) {
	if slot <= 0 || v.soundBank == nil {
		return
	}
	v.placeAndPlay(v.soundPlayer, owner, cell, fine, source, "registry:"+strconv.Itoa(slot), audio.EffectsChannel,
		func() (audio.Sample, bool) { return v.soundBank.Sample(slot) })
}

func entitySoundFine(e MapEntity) image.Point {
	fine := e.Cell.Mul(256).Add(image.Pt(128, 128))
	if e.FinePosition {
		fine = e.Cell.Mul(256).Add(image.Pt(int(e.FineX), int(e.FineY)))
	}
	return fine
}

// PlaySlotAt is playSlotAt's EXPORTED form for pkg/game's positional map
// emissions: swings, cast actions and spell effects all cross here with
// their selected archive slot, owner and sounding cell.
func (v *Viewer) PlaySlotAt(slot int, owner uint32, cell image.Point) {
	v.playSlotAt(slot, owner, cell)
}

// PlayUISound exposes the non-positional interface path to the mission driver,
// which creates the completion/failure child and therefore owns that event.
func (v *Viewer) PlayUISound(slot UISoundSlot) {
	playUISound(v.soundPlayer, v.soundBank, slot)
}

func (v *Viewer) SetAudio(p audio.Player, b SoundBank) {
	v.soundPlayer = p
	v.soundBank = b
}

// SetSpeechAudio hands the viewer the speech device a human's wounds, bleeding
// and fall play on (ANIM-128). A nil device keeps those voices silent; the
// no-damage strike cue plays on the effects device SetAudio hands over.
func (v *Viewer) SetSpeechAudio(p audio.Player) {
	v.speechPlayer = p
}

// ClaimVoice takes the one voice timestamp a drawable keeps for a voice that
// waits gap since its last voice, and refuses while the gap has not passed.
// Wounds and replies store the same timestamp (ANIM-094), so a unit that has
// just cried out does not answer an order at once, and the reverse.
func (v *Viewer) ClaimVoice(id uint32, now time.Time, gap time.Duration) bool {
	m := v.voices[id]
	if !m.claim(now, gap) {
		return false
	}
	if v.voices == nil {
		v.voices = make(map[uint32]voiceMemory)
	}
	v.voices[id] = m
	return true
}

// playHurt plays hurt event k for e. Event 0 is index 1 of the drawn class's
// Sound array on the effects device, for every entity (ANIM-094). Events 1 to
// 3 play on the speech device: the named leaf of a bank-voiced entity's voice
// bank, or for a class-voiced entity index k+1 of its drawn class's Sound
// array, so its fall plays index 4 (ANIM-094, ANIM-128).
func (v *Viewer) playHurt(e MapEntity, k int) {
	source, group := "unit-hurt", audio.SpeechChannel
	if k == hurtNone {
		source, group = "unit-strike", audio.EffectsChannel
	}
	if k == hurtDie {
		source = "unit-death"
	}
	fine := entitySoundFine(e)
	switch {
	case k == hurtNone:
		slot := classSlot(e.Sound, 1)
		v.PlayEffectAtFine(source, slot, e.Owner, e.Cell, fine)
	case e.Voice != "":
		if bank, ok := v.soundBank.(VoiceBank); ok {
			name := e.Voice + "/" + hurtLeaves[k] + ".wav"
			v.placeAndPlay(v.speechPlayer, e.Owner, e.Cell, fine, source, "voice:"+name, group,
				func() (audio.Sample, bool) { return bank.VoiceSample(name) })
		}
	default:
		slot := classSlot(e.Sound, k+1)
		if slot > 0 && v.soundBank != nil {
			v.placeAndPlay(v.speechPlayer, e.Owner, e.Cell, fine, source, "registry:"+strconv.Itoa(slot), group,
				func() (audio.Sample, bool) { return v.soundBank.Sample(slot) })
		}
	}
}

// Explicit entry projects a fresh drawable. Stage 1 queues the ungated die
// cue; health alone cannot identify it (ANIM-126, ANIM-DEATH-007). Capture the
// entry projection before the next simulation step can advance its stage.
func (v *Viewer) beginSoundEntry() {
	v.voices = nil
	v.soundMessages = nil
	for _, e := range v.entities {
		v.soundMessages = append(v.soundMessages, soundMessage{entity: e})
	}
	v.entrySounds = v.entrySounds[:0]
	for _, e := range v.entities {
		if e.CorpseStage == 1 {
			v.entrySounds = append(v.entrySounds, e)
		}
	}
}

// DamageMessage carries a causal health application with its drawable context.
// Entity.HP is server context; the receiver compares AfterHP to client memory.
type DamageMessage struct {
	Entity  MapEntity
	AfterHP int
}

type soundMessage struct {
	entity  MapEntity
	damage  bool
	afterHP int
}

// AppendDamageMessages copies messages into the pending client stream. Snapshot
// replacement cannot overwrite them; stepSound consumes the stream once.
func (v *Viewer) AppendDamageMessages(messages []DamageMessage) {
	for _, message := range messages {
		v.soundMessages = append(v.soundMessages, soundMessage{entity: message.Entity, damage: true, afterHP: message.AfterHP})
	}
}

func (v *Viewer) stepSound(now time.Time) {
	for _, e := range v.entrySounds {
		v.playHurt(e, hurtDie)
	}
	v.entrySounds = nil
	if v.voices == nil {
		v.voices = make(map[uint32]voiceMemory)
	}
	for _, message := range v.soundMessages {
		e := message.entity
		m := v.voices[e.ID]
		if message.damage {
			hp := int(int16(message.afterHP))
			if m.framed {
				if m.hp == hp {
					if v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
						v.playHurt(e, hurtNone)
					}
				} else if k, ok := hurtEvent(m.hp, int(int16(e.MaxHP))); ok && m.claim(now, GruntThrottle) {
					v.voices[e.ID] = m
					if v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
						v.playHurt(e, k)
					}
				}
			}
			m.hp, m.damaged = hp, true
		} else {
			fell := m.framed && m.alive && e.Life != LifeAlive
			bled := m.framed && !m.damaged && !fell && e.Owner == v.localOwner && m.hp < 0 && m.hp >= BleedCeiling && int(int16(e.HP)) < m.hp
			if bled && m.claim(now, GruntThrottle) {
				v.voices[e.ID] = m
				if v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
					v.playHurt(e, hurtHard)
				}
			}
			if fell && v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
				v.playHurt(e, hurtDie)
			}
			m.framed, m.hp, m.alive, m.damaged = true, int(int16(e.HP)), e.Life == LifeAlive, false
		}
		v.voices[e.ID] = m
	}
	v.soundMessages = nil
	next := make(map[uint32]voiceMemory, len(v.entities))
	for _, e := range v.entities {
		next[e.ID] = v.voices[e.ID]
	}
	v.voices = next
}
