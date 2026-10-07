# Corpse entry death cue

## Intent and authority

Hear one death cue for every stage-1 corpse on session entry and ordinary SAV
LOAD. Authority: owner B5 direction; `ANIM-126` (High for the entry projection)
and `ANIM-DEATH-007` (High for the independent server stage), in k149 at
`dfb90596b92a54846a607dafaa6e3db57fa3d0c7`.

## As built

The common `App` entry path captures the finished World projection. Each
stage-1 entity queues one event 3 through the existing speech sink, bank,
visibility and placement rules. `MapEntity.CorpseStage` comes from
`Entity.Decay` or a retained actor's `Current.Stage`, never from HP.
The final opener snapshot includes restored state and resolved sound slots.
The queue survives a stage change on the first tick and is consumed once.

A successful entry or LOAD resets presentation sound memory, even with the
same actor ID and saved tick. Routine projections, cancelled choosers and
failed loads do not. Stage 0 and stages 2 through 5 stay silent on entry.
Live fall, startup strike suppression, wound gates and sound sliders retain
their existing routes. Sound adds no World state or SAV field.

## Proof

`TestAStageOneCorpseVoicesOnceOnMissionEntry` failed at version commit
`ac25ef24` with no cue for a stage-1 body at HP -15. The fixture's final
projection now includes resolved sound metadata before entry.
The UI family checks two corpses, stages 0 through 5, same-ID/time entries,
repeat projections, first-tick stage advance, ungated entry and failed or
cancelled LOAD. The game family checks World stage propagation and a literal
original-format SAV with independent Stage=1, Health=-15 and timer=7.

`TestReleaseCorpseEntrySAVLoad` reads each lawful EN/RU install. It uses the
ordinary named SAVE dialog and LOAD chooser, joins the saved Stage/Health
bytes to the restored actor, and observes the installed sample at the speech
sink. Two loads of the same file each emit once; repeats, cancellation and a
failed file read stay silent. Playback preserves World hash and SAV bytes;
the next production tick changes both current state and its saved output.
Fresh composition roots also restore an independent source Stage edit from 1
to 2 at identical HP=0, despite retained continuation bytes. Only stage 1
emits a cue; its next tick matches the ordinary LOAD. Output is outside the install through
`AGAINROM_CORPSE_ENTRY_OUT`.

Lane receipts are under `review/story1317-corpse-entry/`. Direct EN/RU runs
are focused installed proof. The sole adversarial review and complete merged
final chain are separate acceptance steps. Physical audibility and original
native runtime behaviour were not measured by this lane.

## Open debt

`DIV-1489` retains the 48 untraced projector senders and stage-1 reruns beyond
the identified bleed. `DIV-2153` retains per-entity frame coalescing;
`DIV-2154` retains original attenuation words. No claim about those clauses
is strengthened here. No supported SAV admission boundary changes.
