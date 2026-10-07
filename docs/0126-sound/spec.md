# Spec — a landed blow makes a sound

## Terms

**Slot** — a small integer naming one sound in the install's sound archive. A class names its sounds
by slot, never by path.

**Sample** — decoded mono PCM at a stated rate: the playable form of one slot.

**Bank** — whatever answers "the sample for slot n", or refuses. A bank that refuses every slot is a
lawful bank.

**Device** — whatever a sample and a stereo placement can be handed to. A device that discards
everything is a lawful device and is the state of every headless run.

**Placement** — one play's left and right gains, each an integer in [0, 10000]. The unit is 10000
because that is the clamp the original applies to both of its own positional terms.

**Listener** — the cell the player is deemed to be listening from.

**Sound slots of a class** — the ordered array a unit class names, by index:

| index | what it is |
|---|---|
| 0 | the **swing**: the attacker's own blow leaving |
| 1 | the grunt for a blow that changed no health — **out of scope**, see below |
| 2 | the grunt for a wound leaving the victim at or above half its maximum |
| 3 | the grunt for a wound leaving the victim below half its maximum |
| 4 | read by nothing this contract knows of |

A zero element is **silence and not an error**: three shipped classes carry a zero swing. An array
shorter than an index, or absent altogether, is silence by the same rule.

## Why

Sound is a subsystem with no code at all, and breadth comes before fidelity: a mechanism moved from
nothing to something is worth more now than a mechanism tuned. A blow that lands is the one place in
the game where audio is load-bearing rather than atmospheric — it is the feedback that says the
click did something — so it is where the subsystem starts. What this story owes beyond audibility is
a **seam**: an asset root that is still configurable, a volume and a mute in one named place, and a
game that keeps running when there is no sound device at all.

## Functional requirements

**FR-1.** A unit whose health has fallen since the front end last saw it plays that unit's class
grunt: slot index **3** when the new health is below half that unit's maximum, otherwise index
**2**. Integer halving; a maximum of zero or less makes the test false.

*Folded from hotfix `29c2f77` — see `docs/hotfix/ARCHIVE.md#29c2f77`.* A grunt is gated on the
health the front end already held and is refused when that health was at or below zero, so a
decaying corpse is silent while the killing blow still sounds. The floor alone was never enough
(FR-2).

**FR-2.** FR-1 applies only while the new health is **above -10**. At -10 or below, nothing plays.

**FR-3.** FR-1 is throttled **per victim** to one grunt per **1500 ms of wall clock**. A grunt
refused by the throttle does not reserve the next slot: the clock runs from the grunt that played.

**FR-4.** A unit seen for the first time plays nothing. Arriving is not a wound.

**FR-5.** An attacker plays its class's slot index **0** on the tick its attack-run counter equals
its class's attack delay, once per run.

*Folded from hotfix `4ac0744` — see `docs/hotfix/ARCHIVE.md#4ac0744`.* A swing is voiced only
when the attacker could reach what it is swinging at. The cycle itself is not narrowed and
nothing hashed moves; `AC-14`'s count is read under that gate.

**FR-6.** FR-1 and FR-5 both hold whether or not the damage-numeral display is on. The two are
independent instruments and the numeral's toggle is not an audio switch.

**FR-7.** Every play is **positional**. Its placement is computed from the sounding cell relative to
the listener: gain falls with distance and is zero at or beyond a stated radius, and the left and
right gains differ by the horizontal offset. A play whose gain is zero on both sides is not made at
all.

**FR-8.** A **master volume** and a **mute** are values in one named place, applied to every play by
one code path. A mute plays nothing. Neither may be a literal at a call site.

**FR-9.** The asset root reaches this subsystem the way it reaches every other: the existing flag or
environment variable, resolved by the one function that already owns that precedence. No shipped
source names an install path.

**FR-10.** If the sound archive is absent, unreadable, or its registry unparseable, **the game runs
silently**. It does not refuse to start and it reports no failure to the player.

**FR-11.** If the audio device cannot be opened — no device, no driver, a headless machine — **the
game runs silently**, by FR-10's rule and through the same silent state.

**FR-12.** A slot resolves through the archive's own registry: the registry's `[Sfx]` section holds
`Sfx<n>` entries whose values are archive-relative paths **with no extension**, and the sample is
the archive entry at that path with a `.wav` extension appended. A slot with no entry is silence.

**FR-13.** No sample is read or decoded at startup. A slot is read on first play and the decoded
sample is kept, so a second play of the same slot decodes nothing.

**FR-14.** The decoder accepts RIFF/WAVE holding uncompressed PCM: one or two channels, 8 or 16 bits
per sample, any rate. Two channels are mixed down to one. Anything else is refused, and a refusal is
silence by FR-10's rule.

**FR-15.** A sample whose rate differs from the device's is resampled to the device's rate on
decode, so one device rate serves the whole corpus.

**FR-16.** The command line carries a switch that turns sound off entirely and one that sets the
master volume. Off is indistinguishable from FR-11's silent state from the game's side.

## Acceptance criteria

**AC-1.** A viewer holding a recording device and a class with slots: pushing an entity at full
health, then the same entity at 60% of its maximum, plays slot index 2 exactly once.

**AC-2.** The same, pushed at 40% of its maximum, plays slot index 3.

**AC-3.** Health pushed unchanged plays nothing. Health pushed **higher** plays nothing.

**AC-4.** An entity whose new health is -10 plays nothing; at -9 it plays.

**AC-5.** Two wounds 100 ms apart play once; two 1600 ms apart play twice.

**AC-6.** An entity present in the first push at low health plays nothing on that push.

**AC-7.** A class whose slot array is empty, shorter than the index, or zero at the index plays
nothing, and the push does not fail.

**AC-8.** With the numeral display toggled off, AC-1 still plays.

**AC-9.** A sounding cell to the left of the listener yields a left gain strictly greater than its
right; to the right, the reverse; on the listener, the two are equal. A cell beyond the falloff
radius yields no play at all.

**AC-10.** Mute yields no play. Master volume zero yields no play. Halving the master volume halves
the amplitude of the synthesised buffer.

**AC-11.** A nil device and a nil bank both leave every push silent and failing nothing.

**AC-12.** A synthetic RIFF/WAVE built in test code — 16-bit mono, then 8-bit mono, then 16-bit
stereo — decodes to the expected frame count and values, and a truncated or non-PCM header is
refused rather than panicking.

**AC-13.** A sample at twice the device rate decodes to half as many frames; at the device rate, to
exactly its own.

**AC-14.** A swing fires on the tick the attack-run counter reaches the class's attack delay and on
no other tick of that run.

**AC-15.** `go test ./...` is green with no game install present, and no game asset enters the repo.

## Design decisions

**DD-1.** The subsystem is **entirely outside the determinism wall**. Nothing about sound reaches the
simulation package, no simulation state is added, and the serialized byte form does not change. A
sound is a consequence of state the front end can already see, never a state of its own.

**DD-2.** The decision to play and the ability to play are **separate**. The rules — which slot, when,
how often, how loud — are pure functions over values a caller already holds, and the device is an
interface behind them. That is what makes every acceptance criterion above testable with no audio
hardware, and it is what makes FR-11's silent state one line rather than a mode.

**DD-3.** **Panning is synthesised, not requested.** The playback library offers a volume and no pan,
so a positional play is a stereo buffer this tree builds from a mono sample and two gains. The gains
are integers over a unit of 10000 and the synthesis is a pure function.

**DD-4.** **The device rate is 22050 Hz** — the rate 288 of the corpus's 290 leaves already carry —
and everything else is resampled to it. One rate is not a preference: a playback context has exactly
one, so the choice is between resampling two files and resampling 288.

**DD-5.** **The grunt is decided where the health memory already is** — the front end's own
per-entity health, the same value the damage numeral subtracts. There is no second memory and no new
seam. A class's slot array crosses the existing entity seam as plain integers, interpreted by index
and by nothing else.

**DD-6.** **The swing is decided where the swing clock already is** — the tier that counts the attack
run for the animation. It is not moved across the entity seam, because the counter and the class's
attack delay both live there and neither is a drawing.

**DD-7.** **The attack-delay tick is a disclosed divergence.** The original schedules the swing frame,
the swing sound and the damage from three different numbers; this tree's attack cycle already binds
two of them, so the counter and the threshold are the original's and the *cycle they ride* is ours.
This is stated rather than hidden, and it is the only place this story approximates a timing.

**DD-8.** **The unchanged-health grunt is refused, not approximated.** It exists in the original
because a blow arrives at the client as a message; in this tree only health levels cross the seam, so
"struck for nothing" and "not struck" are the same observation. Playing it on some substitute event
would invent a rule the original does not have.

**DD-9.** **The throttle runs on the wall clock**, matching the original's own millisecond timer
literally, and matching the house rule that a front-end life runs on the wall clock while a drift
runs on the tick. It is on the near side of the determinism wall, so a clock is free there.

**DD-10.** **The sound archive is optional and opened apart from the required set.** The four archives
the front end demands are demanded because a missing one is a broken game. A missing sound archive is
a quiet game, and demanding it would turn every install without one into a failure to start.

**DD-11.** **Volume, mute, the falloff radius, the pan radius, the throttle and the device rate are
named constants or named fields in one place each.** The owner asked for customisation ahead of
polish; a number spelt at a call site is a number no one can change.

## Properties

**P-1.** No sound decision is a function of anything the front end cannot reproduce: the same health
sequence, the same swing counter and the same clock yield the same plays.

**P-2.** Every failure in this subsystem degrades to silence. There is no input — a missing archive, a
malformed registry, a corrupt sample, an absent device, a nil bank, a class with no slots — that can
make the game fail to start, fail to open a map, or fail a frame.

**P-3.** Nothing is decoded on a frame that plays nothing.

## Scope claims

**SC-1.** Only the swing and the two wound grunts. No ambience, no music, no interface sound, no spell
sound, no death sound, no footstep — the archive holds all of them and nothing here asks for one.

**SC-2.** No channel policy: no voice limit, no priority, no stealing. The original has a channel
manager whose policy this contract does not state.

**SC-3.** Slot index 4 is unused and is named so rather than guessed at.

**SC-4.** No in-game volume control surface. The seam is a flag and a named field; a slider is a
different story.
