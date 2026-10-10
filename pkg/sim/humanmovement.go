package sim

// HumanMovement retains a source-backed signed Human+8c speed, after the
// original load-before-modifier fold. It is not the low byte in Mover+a and
// not a terrain-adjusted step rate (TERR-MOVE-056). Native Speed remains the
// unencumbered fallback. Any load, equipment derive or speed-effect producer
// retires this context. A formation order's group term, which already carries
// the members' penalties, replaces it while set (moverSpeed).
type HumanMovement struct {
	Present                     bool
	RawSpeed                    int16
	NativeSpeed, Load, Capacity int32
}

func (e Entity) retainedHumanMovement() bool {
	_, valid := e.RetainedHumanSpeed()
	return valid && e.GroupSpeed == 0
}

// RetainedHumanSpeed is this actor's signed Human statistic while its source
// context remains current. A temporary group override changes movement, not
// this own-stat value. Native actors have no separate retained statistic.
func (e Entity) RetainedHumanSpeed() (int16, bool) {
	h := e.HumanMovement
	return h.RawSpeed, h.Present && h.NativeSpeed == e.Speed && h.Load == e.Load && h.Capacity == e.Capacity
}

// SetHumanMovement is a projection door for a validated source Human. Source
// load is retained until an actual item/weight mutation recomputes it.
func (w *World) SetHumanMovement(id EntityID, raw int16, load int32) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || load < 0 {
		return false
	}
	e := &w.entities[i]
	e.Load = load
	e.HumanMovement = HumanMovement{true, raw, e.Speed, load, e.Capacity}
	return true
}

// SpeedWord is the actor's speed word as the original stores it: a native
// Human's derived word (aloneSpeed), a retained Human word, else Speed.
func (e Entity) SpeedWord() int32 {
	if e.Humanoid && e.ActorLoad.Source.Class == 0 {
		return aloneSpeed(e)
	}
	if raw, ok := e.RetainedHumanSpeed(); ok {
		return int32(raw)
	}
	return e.Speed
}

// SpeedModifierWord is the modifier speed word a SAVE writes for a native
// Human: each byte the native basis knows, else SpeedModifier's byte.
func (e Entity) SpeedModifierWord() uint16 {
	b := [2]byte{byte(e.SpeedModifier), byte(e.SpeedModifier >> 8)}
	for n := range b {
		if e.NativeBasis.ModifierByteKnown(4 + n) {
			b[n] = e.NativeBasis.Modifier[4+n]
		}
	}
	return uint16(b[0]) | uint16(b[1])<<8
}
