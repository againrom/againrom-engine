# Story `1045` — original action cadence

This is the canonical as-built specification. `contract.md` records the dispatched result and
research boundary. Research pin `23daf74f`; reviewed story-1044 master `69c0985b` merged at
`13ede1c7`. Later implementation master `8f6e68e3` merged at `ebb8ec30`, then exact master
`5e5cc97a` merged at `1c5615b1`; neither merge changed this story's simulation state. Simulation
form 62 is current.

## 1. Canonical inputs

**B1.** Every entity carries a canonical `Humanoid` predicate. Human placements and party members
receive it from their typed producers; raised actors copy the Ghost template's explicit value.
The predicate is saved, loaded and hashed. It is not inferred from mana, ownership, equipment or
class-number ranges during a tick.

**B2.** Every normalized Spell rule carries the decoded `Complication Level`. The data loader,
mapload conversion, simulation table, byte form and hash preserve the unsigned byte. Simulation
does not reopen an install to recover it.

**B3.** The equipped primary item and the declared runtime item-weight table remain the only weapon
identity and weight sources. Reaction remains the entity's canonical statistic.

## 2. Physical cadence

**B4.** A retained physical order loads charge plus the existing ranged-flight term before
application. Successful application loads recovery as

`relax + U[0,3] + humanoidPenalty`.

The recovered action consumes one completion boundary tick, re-arms on the next actor tick and
then loads charge. The uninterrupted application interval is therefore

`charge + rangedExtra + relax + U[0,3] + humanoidPenalty + 2 actor ticks`.

**B5.** `humanoidPenalty` is zero for a non-Humanoid or an unarmed Humanoid. An equipped Humanoid
uses

`clamp(IDIV(runtimeWeaponWeight + 5*(30 - Reaction), 12), 0, 12)`.

Go's signed division supplies truncation toward zero. The clamp follows the division.

**B6.** A fighter's weapon Spell is a rider on the physical application and starts no second
recovery. A mage's weapon Spell diverts before the physical strike. The diversion uses decoded
charge without ranged extra, book floor or Complication, then uses ordinary recovery and the two
scheduler boundaries.

## 3. Retained book cadence

**B7.** A retained book order is one sorted canonical record per caster. It carries unit or cell
target identity, Spell, fixed aim, phase, countdown, completion, retry progress and retention.
One tick performs at most one transition and at most one application. A later explicit unit or cell
command cannot replace that record during recovery, either scheduler boundary or re-arm.
An application that removes lower, current or higher records from the sorted book set does not
transfer the releasing transition to another record or skip a surviving record.

**B8.** Ordinary book charge retains the owner-authored visible-swing floor:
`max(decoded charge, 8)`. After application, recovery is

`relax + U[0,3] + humanoidPenalty + ComplicationLevel`.

The uninterrupted interval adds the same two completion and re-arm boundary ticks. The floor is
an againrom rule, not ROM1 evidence; `DIV-425` records the accepted conflict.

**B9.** Insufficient mana is a failed admission. It pays no mana, writes no recovery and consumes
no recovery jitter. An incomplete retained order retries every actor tick. A completed order
attempts on three actor ticks, consumes completion and skips admission once, then re-arms the same
Spell and target. If mana becomes insufficient during charge, release returns a retained record to
incomplete pending state and removes a one-shot record. Later mana regeneration admits the retained
order without another command.

**B10.** Player commands, offensive autocast and idle Heal retain their existing source-specific
selection and priority. Once armed, player and AI physical and book orders use the common
lifecycle. `DIV-022` continues to own the authored priority, facing and current-perception
boundary. Offensive and player-armed restorative autocasts are one-shot selections. Insufficient
mana stores no pending record and the next attempt selects a current target. The unarmed idle-Heal
affordability gate remains its producer-side exclusion. `DIV-028` continues to own movement admitted
during a pending cast.

## 4. Cancellation and revalidation

**B11.** Leaving the living state clears physical charge, recovery, retained book state and
`CastWait` immediately. This applies to command damage, later-ID combat, equipment replacement,
unequip, worn-item drop and derived-health recomputation. A book or weapon-area application that
includes its source completes its deterministic recovery draw but writes no book, cast or physical
recovery on the dead source. Recovery aging does not mutate an actor that is not alive.

**B12.** A physical action checks actor and target presence, linkage and current reach at start and
application. Target health alone does not refuse an application. Existing target teardown and
group-order replacement still decide when the retained target ends. The approach phase applies the
same rule, so a linked negative-health target reaches both the physical and fighter-rider paths.

**B13.** Book admission and release keep their established target-form, applicability, visibility,
range, terrain and action-busy checks. No shared rule is invented for an external writer replacing
progress 1 or 2, or for target loss across every Spell arm; those remain the Unknown bounds in
`HERO-CADENCE-115` and `MAGIC-CADENCE-127`.

## 5. Canonical form and migration

**B14.** Form 62 appends one `Humanoid` boolean to each entity record, one `Complication` byte to
each Spell record and four lifecycle bytes to each tagged book-cast record: phase, retry progress,
completion and retained-order flags. The decoder rejects non-booleans, progress outside 0 through
3, invalid phase/countdown combinations, inconsistent completion or retention, any non-retained
pending record and a cell target
that also carries a unit id. It deliberately does not require a stored unit target to remain
present; release revalidation owns that lawful target-loss boundary. It also refuses `CastWait` on
an actor that is not alive, and refuses a book record whose caster is absent or not alive. The
constructor normalizes dead `CastWait` residue to zero.

**B15.** Hashing is the canonical byte form. Worlds differing only in Humanoid, Complication,
phase, progress or completion therefore differ in digest. Every lifecycle boundary round-trips
and resumes with the same next events, tick and random position. Dead `CastWait` residue normalizes
to the same form and digest as zero. A death transition removes its book record before the next
save or hash observation.

**B16.** Every supported pre-62 form migrates through the existing ordered upgrade chain. Form 61
copies its item-instance section byte-exact, widens entity and Spell records with zero for facts it
never stored, and widens an old book record to its one-shot charging meaning: incomplete,
non-retained and progress zero. Older forms rebuild only the sections they already lacked. One
save-form loss notice says that Humanoid, Complication and retained retry facts were absent; none is
guessed from unrelated bytes.

## Design decisions

**DD1.** Physical and book actions keep their existing state shapes where they already differ. The
shared lifecycle is their scheduler ordering, completion boundary and canonical retry semantics,
not one union record that makes irrelevant fields meaningful.

**DD2.** Failed retry and successful completion remain one story because failure consumes the
completion and progress state success writes. Splitting them would create an intermediate form
that cannot reproduce its own retained order.

**DD3.** The book floor remains isolated at the book-charge writer. Weapon diversion reads decoded
charge directly, so the presentation choice cannot leak into a weapon release or physical ranged
term.

**DD4.** Migration preserves carried bytes before deriving defaults. Form 61's item-instance state
is copied after validation rather than reconstructed from its older code projections.

**DD5.** The shipped integration witness starts real missions through production archives and
typed tables, then isolates the decoded actors so mission AI cannot replace the retained command
being measured. It opens no GUI.

## Bounds

This story does not change tick speed, client animation clocks, damage, spell effects, target
selection, mana regeneration, routing or campaign state. External order replacement and
spell-specific target-loss rules remain research Unknowns. The eight-tick ordinary-book floor is
owner-authored and is not stated as ROM1 behaviour.
