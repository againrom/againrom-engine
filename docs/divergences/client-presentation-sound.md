# Divergences -- Client -- presentation, sound

2 row(s). Part of the split described in `docs/DIVERGENCES.md`: read that file first for what a row means, what each column holds, and how a row is found.

## Divergences

Implementation and researched ROM1 behaviour differ, or the owner ruled against a claim.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|

## Authored where research is silent

No claim answers the question, so the implementation authored one.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-502 | client / weapon-spell cast-animation sound when attack and cast messages share a tick | — | `VIDEO-SFX-019` and `VIDEO-SFX-018` establish the cast-animation selector `500+actionspell` at the class `AttackDelay`. `MAGIC-CASTANIM-029` establishes that the even cast message's action-start arm is skipped while the client unit already has an action and explicitly leaves ordering against the ordinary attack message Unknown. | A mage's weapon replacement uses the existing `AttackCasting` run and emits the cast selector when that run reaches `AttackDelay`, without starting a second run at spell application. A fighter's spell rider keeps its physical swing and has no cast-animation hook; its applied spell still emits the direct message selector. | UNKNOWN | The two simulation arms are explicit and one-shot, but choosing which client message wins an original same-tick action guard would invent the unresolved ordering. This keeps presentation on the run each arm already displays. | A promoted claim fixing `0x72` versus even-`0x86` arrival order and the resulting `unit+0xa0` guard outcome for weapon replacement and rider casts. | OPEN |
| DIV-501 | client / spell message sound ordering relative to transient-object construction | — | `VIDEO-SFX-019` establishes the `500+picture` selectors, non-null and picture-51 gates, and the fixed phase-8 storm arm. It explicitly leaves all three client tails' ordering relative to object construction Unknown. `MAGIC-AREADRAW-049` establishes one blast message and one staged message per accepted cell. | The direct even-picture selector plays at the source when an applied `CastEvent` or resolved temporary-script cast is observed. A blast effect selector plays at the target when its transient burst begins; a staged selector plays once per reported accepted cell when that cell object is created. Picture 51 voices slot 551 when its common projectile driver reaches actionphase 8, including an ordinary SAV LOAD before that boundary. A record saved at or beyond actionphase 8 does not replay the cue. | UNKNOWN | These are the narrow production events the implementation actually owns. They preserve the decoded event population and exact phase-8 exception without claiming whether ROM1's terminal call precedes or follows allocation by a few instructions. | A promoted claim retaining each `0x86`, `0x8b` and `0x8c` prelude through construction and terminal call, including their exact order. | OPEN |
