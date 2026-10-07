# 1062 — spell cast and effect SFX

## Player result

Successful spells now use the installed one-shot samples at their actual map
events. An applied cast voices the even-picture selector at the caster; the
caster's animation voices the same selector independently at its class
`AttackDelay`. Fire Ball's odd-picture effect sounds at its delayed visual
impact, staged effects sound once per accepted cell, and every Meteor object
waits for its decoded phase 8 before playing slot 551.

`VIDEO-SFX-019` is the selector authority, with `VIDEO-SFX-018`,
`MAGIC-CASTANIM-029`, `MAGIC-BURSTLIFE-034` and `MAGIC-AREADRAW-049` fixing the
event population and known timing. The unresolved client-tail construction
order and weapon-message race are recorded as `DIV-501` and `DIV-502`; no
terminal-site, slot-15, music, ambience or looping-player mapping is added.

## As built

`sim.Report` exposes only successfully resolved temporary-caster spells in
addition to its existing applied casts and accepted area cells. `pkg/game`
turns those observations into positional tick-owned cues and sends them through
the existing `Viewer.PlaySlotAt` path, so the existing archive/device,
sound-enable, volume, fog, distance and missing-selector refusals remain the one
set of gates. Cues advance only in a paced world tick; redraw, pause and repeated
pushes cannot consume or duplicate them. A book-animation cue also retains the
exact caster and cast-run identity: at `AttackDelay` it plays only if that same
living visible run is still at the expected phase, and uses the caster's current
cell. Teleport therefore voices its animation at the destination, while death,
disappearance, cancellation or replacement drops the stale cue. A failed cast
has no observation and therefore no sound.

## Proof and debt

Focused synthetic tests cover selector arithmetic, source/target/cell position,
book and weapon timing, staged and phase-8 events, refusal, audio-disabled
operation, redraw idempotence, Teleport relocation, death and run replacement,
and the complete one-command Fire Ball lifecycle.
The install-gated test checks the exact 50/56 cast and 6/56 effect candidate
population on each lawful root, including refusal of terminal slot 565.

Final reconciled Go, no-assets, EN/RU release and mission census evidence is in
`verification.md`. No install bytes or generated sample data enter the tree.
