# Analysis — a landed blow makes a sound

## What we did not know

Sound is one of the subsystems with no code at all. Before this story the tree could not make a
noise of any kind, and `pkg/ui/numeral.go` said so in its own header: *"there is no audio layer in
this tree at all, so the sounds are declared out of scope rather than approximated."*

Three questions had to be answered before a contract could be written, and all three were answered
by measurement rather than by assumption.

## What the leaves are

`sfx.res` was extracted whole on both lawful roots and every leaf's header read. The result is the
same on each root:

| count | shape |
|---|---|
| 288 | RIFF/WAVE, `audioFormat` 1 (PCM), mono, 22050 Hz, 16-bit |
| 1 | RIFF/WAVE, PCM, mono, **44100 Hz**, 16-bit — `town/shop/nofit.wav` |
| 1 | not RIFF — `sfx.reg`, the slot registry, not a sample |

So **there is nothing to decode**. The sound corpus is plain uncompressed PCM; a reader for it is a
header walk, not a reverse-engineering problem. The one 44100 Hz leaf is a shop refusal, not a unit
sound, and it is on no `Sound[]` array — but the decoder handles it generically rather than
special-casing it, because a rate the corpus contains is a rate the corpus contains.

That measurement is why this story is small. It also settles the RU question that had to be asked:
`RES-HDR-012` records the RU `SFX.RES` breaking a header law elsewhere, so the two roots were
surveyed separately rather than assumed equal. They agree.

## What can play them

`github.com/hajimehoshi/ebiten/v2` is already a required module and already ships `ebiten/v2/audio`
in the same download. **This story adds no dependency.** What that package does *not* offer is a pan
control: `*audio.Player` has `SetVolume` and no `SetPan`, and a context has exactly one sample rate.
Both consequences are the contract's, not an implementation detail — a positional sound has to be
synthesised into a stereo buffer by us, and every sample has to arrive at one rate.

## What the front end can already see

The grunt's whole input is a health level and its predecessor. `pkg/ui`'s viewer already holds
`numeralHP`, *"the health this viewer last saw for each entity, and it is the ONLY source of a
damage figure in the tree"*, and `ui.MapEntity` already carries `HP` and `MaxHP`. So the grunt costs
one comparison in a walk that already runs.

The swing looked like the harder half and was not. `pkg/game`'s `advanceSwings` already keeps
`swing map[sim.EntityID]int` — *"how many ticks each entity's current attack RUN has been playing"*,
counted from zero at the transition into the charging phase — which is the same counter the decoded
swing-sound test reads. What was missing was only the class-side number beside it: `AttackDelay` and
`Sound[]` are both parsed into `data.UnitClass` and both dropped at the `terrain.UnitClass` boundary,
because the render tier has no use for either.

## What is not here

`ANIM-CLOCK-024` is why the swing is a *named divergence* rather than a reproduction: the swing
frame, the swing sound and the damage are scheduled from three different numbers in the original,
and this tree's attack cycle already binds two of them. Our swing clock is the right counter and
`AttackDelay` is the right threshold; what we cannot claim is that the tick it lands on is the
original's tick, because the cycle it rides is our own.

And one grunt cannot be built at all. The original's grunt hook is a **message** arm: the engine
sends the victim's new health and the client compares, so "struck for no damage" is a state the
client is told about. This tree has no blow message — health levels cross the seam every tick and
nothing else does — so an unchanged health is indistinguishable from no blow. That arm is refused
rather than approximated.
