# Control Spirit raised actor and item route

## Intent and authority

The actor Control Spirit raises, and the weapon, item and scroll routes that
reach it, follow the original where the claims are High. Authority: `MAGIC-249`
(template and corpse stores), `MAGIC-250` (difficulty, Medium), `MAGIC-251`
(item entry and arm), `MAGIC-252` (no authored carrier; shop Scrolls and Books
can carry id 25), `MAGIC-SING-019` (c), `MAGIC-ITEM-007`. Rows: `DIV-037`
narrowed, `DIV-056` closed, `DIV-2157` opened. Owner direction: change state
only where a claim is High and disclose the rest.

## As built

- The template is the first exact-name, case-sensitive Units row `Ghost`. The
  corpse supplies reaction/2+1, Mind, Spirit, healthMax/2 with health equal to
  it, and the first word of the to-hit and defence blocks. Before, the to-hit
  and defence came from the row. The other columns still come from the row
  (Medium row streaming); the engine's values stand.
- Difficulty has no effect on the raised actor (`MAGIC-250`, Medium). It changed
  only health, to-hit and defence, and the corpse now supplies all three. The
  template still carries the adjusted to-hit and defence, which saves persist;
  the raise does not read them.
- A weapon, fighter-rider or item release of id 25 is no longer refused. The arm
  uses only the target's cell. It finds the first bones corpse in that cell,
  consumes it, and places the actor in the corpse's own cell. The engine never
  fails a placement: the original's radius-zero placement can fail and consume
  the corpse, but its blockers are Unknown (`MAGIC-251`), so no assumed blocker
  set destroys a corpse. The target object is not written, and its spell mark
  is skipped.
- The release only queues the cell and the caster. The corpse is consumed and
  the actor placed after the attack pass, before the decay pass, because the
  strike and release callers and the pass loop hold entity indices (`DIV-2157`).
- A Scroll of Control Spirit resolves its corpse by the target's cell the same
  way. A book cast keeps its corpse-target admission.
- The corpse leaves through the existing removal, which already retires it to
  the saved dead list as a stage-5 body. That the original keeps it on the dead
  list is Medium; no change was needed.

## Save effect

No saved field was added. The raised actor's to-hit and defence words now hold
the corpse's values, so a save made after a raise differs from before in those
two words. A save writes from current state, as for every actor.

## Proof

- `TestControlSpiritConsumesBonesAndCreatesANewOwnedGhost` fails on the old
  template values (row to-hit 41, defence 42).
- `TestItemReleaseOfControlSpiritRaisesFromTheCorpseAtTheTargetCell` fails on
  the old refusal; with `...WithAGroundTargetInTheCellStillRaises`, `TestBookCastOfControlSpiritRaisesWithAGroundActorInTheCorpseCell`, `TestScrollOfControlSpiritConsumesTheTargetsOwnCorpse`,
  `...WithoutACorpseAtTheCellDoesNothing`, `TestTwoItemReleasesShareOneCorpseOnce`,
  `TestFighterRiderOfControlSpiritRaisesThroughAStrike` and
  `TestScrollOfControlSpiritRaisesFromTheCorpseAtTheTargetCell`.

## Open debt

- `DIV-2157`: raise one pass later; the original's dead-list order, its
  no-corpse exit, radius-zero placement failure and blockers are Unknown.
- `DIV-037`: other row columns (Medium); a health maximum of 1 raises health 1
  where the original gives 0.
- The weapon characteristics projection still reports no characteristics for
  id 25; the spell has no damage or duration.
