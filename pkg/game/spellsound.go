package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// The spell book is a fixed 1..28 id space. Both decoded selector families use
// the picture arithmetic that already owns that space: the caster/action
// picture is even and the transient effect picture is odd.
const (
	firstSpellSoundID = 1
	lastSpellSoundID  = 28
	spellSoundBase    = 500
	stormBurstPicture = 51 // ANIM-149, DIV-501
	stormSoundSlot    = 551
	stormSoundPhase   = 8
)

// spellSoundCue is one future one-shot, counted only by paced world ticks.
// Keeping this out of draw and push is the idempotence rule: any number of
// render frames, including paused ones, can observe a cue without consuming or
// repeating it.
type spellSoundCue struct {
	source string
	slot   int
	owner  uint32
	cell   image.Point
	ticks  int

	// caster/run/phase are populated only for the delayed book-cast animation
	// hook. The run identity prevents an old cue from voicing a replacement;
	// phase proves the same live run actually reached its class AttackDelay.
	caster sim.EntityID
	run    *castRunIdentity
	phase  int
}

func castSpellSoundSlot(spell uint16) int {
	if spell < firstSpellSoundID || spell > lastSpellSoundID {
		return 0
	}
	return spellSoundBase + data.CastPicture(int(spell))
}

func effectSpellSoundSlot(spell uint16) int {
	if spell < firstSpellSoundID || spell > lastSpellSoundID {
		return 0
	}
	picture := data.BurstPicture(int(spell))
	if picture == stormBurstPicture {
		return stormSoundSlot
	}
	return spellSoundBase + picture
}

func (mw *mapWorld) playSpellSound(slot int, owner uint32, cell image.Point, source ...string) {
	if mw.spellSound == nil && mw.semanticSound == nil || slot <= 0 {
		return
	}
	family := "spell-effect"
	if len(source) != 0 {
		family = source[0]
	}
	if mw.semanticSound != nil {
		mw.semanticSound(family, slot, owner, cell, cell.Mul(256).Add(image.Pt(128, 128)))
	} else {
		mw.spellSound(slot, owner, cell)
	}
}

func (mw *mapWorld) actorSoundFine(e sim.Entity) image.Point {
	fine := image.Pt(int(e.X)*256+128, int(e.Y)*256+128)
	if x, y, ok := mw.world.ActorFinePosition(e.ID); ok {
		fine = image.Pt(int(e.X)*256+int(x), int(e.Y)*256+int(y))
	}
	return fine
}

func (mw *mapWorld) playActorSound(source string, slot int, e sim.Entity) {
	if slot <= 0 {
		return
	}
	cell := image.Pt(int(e.X), int(e.Y))
	if mw.semanticSound != nil {
		mw.semanticSound(source, slot, e.Owner, cell, mw.actorSoundFine(e))
	} else if source == "unit-swing" && mw.swingSound != nil {
		mw.swingSound(slot, e.Owner, cell)
	} else if mw.spellSound != nil {
		mw.spellSound(slot, e.Owner, cell)
	}
}

// queueSpellSound waits for exactly ticks subsequent calls to
// advanceSpellSoundCues. A non-positive delay is the current logical event and
// plays immediately. The callback is tested before retaining a cue, so a world
// constructed with audio disabled cannot accumulate old sounds for a device
// attached later.
func (mw *mapWorld) queueSpellSound(slot int, owner uint32, cell image.Point, ticks int, source ...string) {
	if mw.spellSound == nil && mw.semanticSound == nil || slot <= 0 {
		return
	}
	if ticks <= 0 {
		mw.playSpellSound(slot, owner, cell, source...)
		return
	}
	family := "spell-effect"
	if len(source) != 0 {
		family = source[0]
	}
	mw.spellSoundCues = append(mw.spellSoundCues, spellSoundCue{
		source: family, slot: slot, owner: owner, cell: cell, ticks: ticks,
	})
}

func (mw *mapWorld) advanceSpellSoundCues() {
	live := mw.spellSoundCues[:0]
	for _, cue := range mw.spellSoundCues {
		var caster sim.Entity
		if cue.run != nil {
			var ok bool
			caster, ok = mw.liveBookCastSoundCaster(cue)
			if !ok {
				// The caster died, disappeared, turned out of the visible run, or
				// replaced/cancelled it. The animation event no longer exists.
				continue
			}
		}
		cue.ticks--
		if cue.ticks <= 0 {
			if cue.run == nil {
				mw.playSpellSound(cue.slot, cue.owner, cue.cell, cue.source)
			} else if mw.swing[cue.caster] == cue.phase {
				// Unlike the independent direct message at CastEvent.FromX/Y,
				// this hook belongs to the live caster animation. Teleport has
				// already moved that caster, so both owner and cell are live.
				mw.playActorSound("book-cast-animation", cue.slot, caster)
			}
			continue
		}
		live = append(live, cue)
	}
	mw.spellSoundCues = live
}

func (mw *mapWorld) liveBookCastSoundCaster(cue spellSoundCue) (sim.Entity, bool) {
	e, ok := mw.entity(cue.caster)
	if !ok || !e.Alive() || e.Turning() {
		return sim.Entity{}, false
	}
	r, ok := mw.castRun[cue.caster]
	if !ok || r.left <= 0 || r.identity != cue.run {
		return sim.Entity{}, false
	}
	return e, true
}

// scheduleBookCastSound places the cast-animation hook on the caster's class
// AttackDelay phase. The cast run enters phase zero later in this same world
// tick, so phase n is the (n+1)th cue advance.
func (mw *mapWorld) scheduleBookCastSound(ev sim.CastEvent, span int) {
	e, ok := mw.entity(ev.Caster)
	if !ok {
		return
	}
	r, running := mw.castRun[ev.Caster]
	snd, ok := mw.sounds[mw.spellClientClass(e.ID, e.Class)]
	if !running || r.identity == nil || !ok || snd.AttackDelay < 0 || int(snd.AttackDelay) >= span ||
		mw.spellSound == nil && mw.semanticSound == nil {
		return
	}
	slot := castSpellSoundSlot(ev.Spell)
	if slot <= 0 {
		return
	}
	phase := int(snd.AttackDelay)
	mw.spellSoundCues = append(mw.spellSoundCues, spellSoundCue{
		slot: slot, ticks: phase + 1, caster: ev.Caster, run: r.identity, phase: phase,
	})
}

// observeScriptCasts voices only temporary-caster records that the simulation
// actually resolved. They have no client unit and therefore no cast-animation
// hook; their carried source cell is the spatial origin of the direct message.
func (mw *mapWorld) observeScriptCasts(events []sim.ScriptCastEvent) {
	for i, ev := range events {
		from := image.Pt(int(ev.FromX), int(ev.FromY))
		to := image.Pt(int(ev.ToX), int(ev.ToY))
		seed := uint32(from.X|from.Y<<8) ^ uint32(to.X|to.Y<<8)<<16 ^ uint32(i+1)
		// UNIT-M10CAST-056 / MAGIC-DELIVER-035: direct-client Lightning carries
		// flight parameter 5, not the normal caster's 13 calls; the source-cell
		// Prismatic Spray links each resolved victim (MAGIC-281).
		mw.releaseScriptCast(ev, seed)
		mw.playSpellSound(castSpellSoundSlot(ev.Spell), 0, from, "scripted-cast")
		mw.scheduleBlastSound(ev.Spell, 0, from, to, false)
	}
}

// scheduleBlastSound voices the one transient object emitted by a blast row.
// This build deliberately holds a travelling Fire Ball burst until its drawn
// projectile reaches the target; its effect sound is held by that same number
// so picture and sound cannot arrive on different world ticks.
func (mw *mapWorld) scheduleBlastSound(spell uint16, owner uint32, from, to image.Point, weaponReplacement bool) {
	rule, ok := mw.world.Spell(uint32(spell))
	if !ok || rule.AreaMode() != sim.AreaModeBlast {
		return
	}
	delay := 0
	if !weaponReplacement {
		picture := data.CastPicture(int(spell))
		if data.CastTravels(picture) {
			delay = data.CastFlight(picture, castDistance(from, to))
		}
	}
	source := "spell-effect"
	if delay > 0 {
		source = "deferred-spell-effect"
	}
	mw.queueSpellSound(effectSpellSoundSlot(spell), owner, to, delay, source)
}
