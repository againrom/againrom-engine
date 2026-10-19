# Floating damage numerals

## Intent and authority

Merge damage on the same victim only within one raw simulation tick. Keep
figures from different ticks separate, including ticks delivered in one frame.
Owner direction supersedes the interim per-application behaviour of DIV-976.

Authority read through the pinned claim tool: ANIM-NUM-020, ANIM-071,
ANIM-073, ANIM-BLOW-019 (including its partial retraction), ANIM-PACE-017,
ANIM-072, SESS-TICK-026 and ANIM-075 (including its partial retraction).
ANIM-071 establishes exact victim/raw-counter equality and no refresh on merge.
ANIM-073 establishes nominal speed counts, not realised timing on a stalled
original. ANIM-BLOW-019's numeral branch stands after its sound-band correction.
ANIM-075's Poison inference is retracted; its direct-call population stands.

## As built

The one production route is `sim.StepReported` -> `Report.Damages` plus
`Report.Tick` -> `mapWorld.observeDamage` -> `pendingDamage` -> `push` ->
`Viewer.AppendDamageMessages` -> client stream consumption ->
`receiveDamageNumeral`. The existing sound stream carries the tick beside each
causal application. Sounds retain their existing health-memory and dispatch
rules. A numeral uses the whole application delta; it does not narrow to the
sound client's signed health word.

A matching victim/tick adds to the existing amount and re-composes its text.
Birth, flight, colour and drift direction remain unchanged. Different victims
or raw ticks append independent records. Snapshot replacement, passive health
changes and corpse decay create no figure without a causal message. The old
`ingestDamage`, `addDamageNumeral` and `numeralHP` detector are removed.

Life remains 1000 ms, inclusive, on the presentation clock. Drift remains one
pixel sideways and two upward per paced animation tick. A delayed record starts
with one drift step for each later raw tick already projected. Active player
pause keeps the pre-existing retained-flight and remaining-life policy.
The nine original periods are 125, 100, 83, 71, 62, 50, 41, 35 and 31 ms;
nominal tick lives are 8..9, 10..11, 12..13, 14..15, 16..17, 20..21,
24..25, 28..29 and 32..33. No speed slider, damage rule or health bar changes.

The shared viewer changes ROM1 and ROM2. Neither `World` nor a save schema gains
an event field. `Report.Tick` is a return value; the projection queue, messages
and numeral records are transient. No hash pin or scenario expectation changes.

## Builder census

The three existing simulation emitters are `damageObservation.loss`,
`World.reportHealthMessage` and `World.reportPhysicalBlow`; no damage producer
is added. There is one event projection builder in `mapWorld.push`, one client append
method, one numeral builder (`receiveDamageNumeral`) and one placement method
(`numeralPlacements`). Normal frames, catch-up, command damage and physical
combat use that same route. The installed witness observes its emitted records;
it does not rebuild numerals in the game tier. `HeadlessMapFrame` composites the
production numeral placements at native scale after terrain and art.

## Proof

`TestDamageNumeralMergeKeyAndLossControls` compares same victim/tick with
next-tick, same-full-tick and other-victim controls, including multiple events
before one frame. A snapshot-only decrease adds nothing.
`TestDamageNumeralMergeKeepsBirthFlightAndColour` checks the summed glyph and
retained birth, drift, colour and expiry. `TestDamageNumeralCatchUpOffsetsFollowRawTicks`
checks delayed offsets. `TestDamageNumeralKeepsWholeApplicationAmount` rejects
signed-word narrowing. `TestDamageNumeralLifeAtEveryShippedSpeed` checks each
period's literal nominal drift and both sides of the 1000 ms boundary.

The existing physical-application simulation test compares reported and plain
World bytes/hash, checks the report's raw tick, and retains zero-hit controls.
Fog, text smoothing, toggle, corpse-decay and active-pause tests use explicit
causal messages where they intend a blow.

Installed release subjects:

- `TestReleaseDamageNumeralsKeepRepeatedGarrisonTicks`: mission 41, hero placed
  at the gate, roaming pack removed, healing below half, eight simulation ticks
  per presentation frame, actual garrison damage without injected damage.
  Each emitted amount is checked against reported events keyed by victim/tick.
  Two live records for one victim reject frame/window merging. Replayed
  snapshots preserve World bytes and hash.
- `TestReleaseDamageNumeralsMergeCommandsWithinRawTick`: installed mission 20,
  damage controls 7+13 in one tick and 3 in the next; expect records 20 and 3.
- `TestReleaseSecondGameDamageNumeralsMergeCommandsWithinRawTick`: the same
  controls through ROM2 mission 10.
- `TestReleaseDamageNumeralsStartAtTheUnitWithInstalledFont`: repeated explicit
  events retain their birth origins and installed glyphs.

All three new subjects are declared in the gated population. EN/RU release
receipts, ROM2 receipts, cropped 320x224 CPU frames, raw tick/figure JSON and
final gate results live under the external review directory
`review/story1472-damage-numerals/`. CPU frames omit shroud, HUD and GPU blend
modes; they are not native display captures. `PROGRESS.md` records attempts and
`RETURN.md` records exact candidate results.

The storyguard baseline falls by two numeric comment forms, five long comment
groups and 8931 comment bytes after removing the obsolete detector commentary.
Composition and command-literal counters do not change.

## Divergences and debt

DIV-976 is closed in `DIVERGENCES-CLOSED.md`. DIV-3703 retains the existing
palette, scalar-one offsets and shadow where the claim gives no exact values.
DIV-3704 bounds delayed receipt timing, realised cadence and the existing
active-pause policy. DIV-3705 and DIV-3706 are unused. The existing windowed GPU release subject explicitly skips when
`AGAINROM_HEADLESS_ONLY=1`; the release analyzer reports that environment skip,
so a headless set cannot claim full GPU acceptance. Its damage producer now
uses a causal simulation command. No original program, window or audio is used
by this lane. No install bytes or generated images
enter the repository.
