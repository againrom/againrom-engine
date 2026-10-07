# Story 1136 — Sack container tail and Effect-bearing items restored from an original SAV

## Status

Gates run on the exact candidate commit, current master (`edfe4641`, story
1134) merged in; see "Proof" for the full list and exact counts. `gofmt -l .`
clean; `go test -trimpath -count=1 ./...` (whole repository, asset-free)
clean, 49 packages, 0 FAIL, including `internal/gatedtests`' checked-in
`testdata/population.txt` census (net +1 from this story's own witness on
top of master's 201 after story 1134, 202 registered names —
`population.txt` itself is 206 lines, 4 of them comments);
`scripts/check-no-game-assets.sh` clean; `pipeline/check-release-tests.sh`
against both lawful roots, 8 packages, 202 gated tests, 202 of 202 ran and 0
lacked a subject on EN and again on RU; `scripts/check-claim-citations.sh`
clean after merging current master; `pipeline/check-div-claims.sh` exit 0;
mission 10/20 `-trace -ticks 1` UNSUPPORTED-node count 0 on both missions,
both roots, unchanged from master — this story's own restoration is reached
only through an original-save resume, a path the fresh mission drive does
not exercise (story 1132/1133's own precedent, "Proof").

## Player result

A mission loaded from an original ROM1 SAV no longer refuses the whole load
because an item carries Effect state: an item's own unsupported Effect
record — a nonzero Token state, or a reference to a class other than
`Effect` such as `Effect_DirectDamage` — no longer stops the document walk.
The item restores with its supported state intact (code, stack, price,
weight, every ordered state-0 Effect) and the one unsupported record
diverted onto `sav.Piece.UnsupportedEffectStates`, carried but not
interpreted, exactly like every other item in the same Sack or the same
actor's holdings batch. No claim binds an item's own Effect reference to the
top-level SpellEffect graph story 1132 restores — they are structurally
distinct classes under a shared `Token` base (`SAV-CLASS-033`,
`SAV-TOKENLOAD-093`), read through unrelated walkers, and
`SAV-TOKENLOAD-094` states plainly that "nested Item/Effect reach is not
established" — so no binding is built; carrying the raw record and reporting
it is the B1-compliant behaviour research supports today.

Every ground Sack's own container tail (`+0x1c` insertion index, `+0x20`
running weight sum, `ITEM-CONT-004`'s own class, confirmed distinct from
`Item`'s numerically coincident fields by `SAV-670`) already had a live,
hashed carrier before this story (STORY 1136 HOTFIX correction):
`sim.SavedObjectContainer` (story 1115) carries the pair for `SavedOwnerSack`,
is filled on both original LOAD doors by `importSavedSackObjects`, sits
inside `World.encode()`, and is enacted — `putGroundObject` mints a new
ground Sack's container at the `+0x1c` sentinel and calls `weight()` on drop
and pickup, and a dying actor's pack spill carries that actor's own tail
verbatim into the new ground Sack's container. This story adds a second
carrier, `sim.World.savedGroundContainers`, restored on both original LOAD
doors from the same two file fields but outside `World.encode()`, so it
cannot move the world hash. Its only reader today is the unshipped
`exportOriginalGroundContainers` (see below), which re-exports this second
carrier's file-time value rather than the first carrier's live one — open
debt, harmless only because nothing calls it in production (see "Open
debt"). A native re-export through the new carrier still reproduces the
source file's two tail dwords per Sack and changes no other byte: the
corpus's own real content is comprehensive here, unlike story 1133's single
sample — 51 of 51 world-half saves in the preserved corpus carry a
non-empty Sack population with a real container tail on both EN and RU, and
the two carriers agree on every one of them (see "Proof").

An actor's own container tail (`ActorLoad.InsertIndex`/`Accumulator`) is
unchanged by this story: it was already carried, restored and live —
consumed by `sourceDerive`'s carry-weight speed penalty and mutated by
equip/pickup/drop — since story 1110's holdings work
(`pkg/sim/actorload.go`, `pkg/sim/sourceequipmove.go`). The brief's "unit
containers restore their container tail" clause was already true; this
story's own scope is exclusively the ground-Sack side, which had no carrier
at all before it.

No Sack-side rule reads or recomputes either carrier's insertion index or
weight sum (STORY 1136 HOTFIX correction): `pourSack` (`pkg/sim/sack.go`)
always appends a picked-up or dropped item at the live tail regardless of
the carried insertion index, and no Sack-side speed, capacity or other
display exists to read the weight sum back — the original's own load derive
(`ITEM-LOAD-005`) reaches a container's own `+0x20` only through an actor's
own `+0x7c` reference, which no Sack carries (DIV-962). That is narrower
than "no live computation touches either value": `SavedObjectContainer`'s
own `weight()` updates the Sack-side accumulator on every drop and pickup,
and on the actor side `ActorLoad.CurrentLoad()` reads its own accumulator
for the carry-weight speed penalty, as stated above. Native mission SAVE
itself is not a wired player-facing feature: no interactive path calls
`exportOriginalGroundContainers` (see "Open debt").

## Authority

- **`SAV-670`** (High, promoted, EXP-0313): the container tail embedded at
  `Sack+0x40` and, when present, `Unit+0x7c` is `ITEM-CONT-004`'s own
  container class, a different object from `Item` sharing no fields with
  `Item`'s numerically coincident `+0x1c`/`+0x20`.
- **`SAV-671`** (High, promoted, EXP-0313): `+0x1c` defaults to sentinel
  `10000`, forcing every ordinary insert to append; the pick-up order and
  the equip arm are its only non-default writers, both already exercised.
  The corpus census behind this claim finds the sentinel on every one of 307
  sampled Sack-class containers, zero exceptions.
- **`SAV-672`** (High, promoted, EXP-0313): `+0x20` is the running
  weight-times-count sum `ITEM-STACK-003` already gives five maintenance
  sites for, defaults to 0, copied verbatim by a transfer helper and zeroed
  by a reset pair alongside `+0x1c`.
- **`SAV-EQUIPEFFECT-553`** (High, promoted, EXP-0288): equipment Effect
  traversal is forward and incremental, not a pre-collected batch — consulted
  for the shape of the `Effects` reference list `piece()` walks, not
  extended.
- **`SAV-TOKEN-034`** (High, active): the 37-byte placeable head programme
  Item/Effect/Sack/Unit/Building all share, confirming an Effect reference
  is read through the same class-agnostic head before any subtype dispatch,
  never a hard error.
- **`SAV-CLASS-033`, `SAV-TOKENLOAD-093`, `SAV-TOKENLOAD-094`** (High/active):
  `Effect` and `SpellEffect` are siblings under `Token`, not the same class
  or a parent/child pair; nested Item/Effect reach from the top-level
  SpellEffect graph's own lifecycle dispatch is explicitly not established.
  Consulted to confirm no binding exists to build, not to build one.

## As-built behaviour

**`pkg/formats/sav`** (`ground.go`, `program.go`, `party.go`): `GroundSack`
gained `InsertIndex`/`Accumulator`, read from `Contents1C`/`Contents20`.
`Record.ContainerTailOff` records the body offset of a walked
`stpContainer`'s own tail during the walk (`program.go`), letting
`File.SetGroundContainerTails` patch exactly those two dwords per Sack for
re-export without touching anything else — verified by a shape check against
each Sack's own `Identity` before writing. `piece()`'s Effect dispatch
condition widened from "nonzero state" alone to `effect.Class != "Effect" ||
state != 0`: a reference to any class other than the bare `Effect` type
(such as `Effect_DirectDamage`) now diverts the same as a nonzero state, on
`TestPieceDivertsANonEffectClassEvenAtStateZero`'s own regression, even when
that record's own `E0C` head reads zero — reading the head at all assumes
the `Effect` layout, and it never becomes a dispatch error for a subtype.
`ActorLoadState`'s own `InsertIndex`/`Accumulator` fields (from
`Inventory1C`/`Inventory20`) predate this story (story 1110); this story's
diff there is doc-comment and dispatch-delegation only, no new decode.

**`pkg/sim`** (`savedgroundcontainer.go`, new): `SavedGroundContainer`
mirrors `sav.GroundContainerTail` with no `pkg/formats/sav` import.
`World.savedGroundContainers` is carried-not-wire-form, `savedProjectiles`'
own class (story 1133): `UnmarshalBinary`'s composite literal carries the
receiver's own prior value across a decode rather than reading it from the
encoded bytes, so a decode into a FRESH receiver needs priming
(`SetSavedGroundContainers`) first or the value is silently lost.
`ImportOriginalLivingActors` (`pkg/sim/originalliving.go`) and
`restoreOriginalActorStock` (`pkg/game/originalholdings.go`) both stage
every original actor admission through exactly that round trip and both now
prime it. `resumeWorld`'s own `.ags` resume (`pkg/game/resume.go`) needs NO
priming call, unlike those two: it calls `ms.World.UnmarshalBinary(form)`
directly on the SAME receiver the composite literal reads
`savedGroundContainers` off — proven by
`TestUnmarshalBinaryOntoTheSameReceiverNeedsNoGroundContainerPriming`
(`pkg/sim/savedgroundcontainer_test.go`), not merely inferred.

**`pkg/game`** (`originalgroundcontainers.go`, new): `originalGroundContainers`
reads every Sack's own tail off `sav.File.GroundSacks()` into
`[]sim.SavedGroundContainer`, sorted by identity.
`applyOriginalGroundContainers` imports it into `sim.World` on both original
LOAD doors (`ResumeOriginalSave`, `FrontEnd.RestoreOriginal`) right after the
existing ground-Sack import, and into `restoreOriginalActorStock`'s staging
round trip. `OriginalSaveResume` reports `GroundContainers`/
`GroundContainersApplied` alongside the existing fields.
`exportOriginalGroundContainers` re-exports the carried tails through
`sav.SetGroundContainerTails` — a content patch touching only the two tail
dwords per Sack, never a full re-encode — but has NO production caller: this
project has no production mission SAVE path yet (owner rule), so it is an
unshipped round-trip building block, checked only by the corpus audit and
the release witness' own native-export half. `originalground.go`'s and
`originalholdings.go`'s refusal conditions both narrowed from including
`len(piece.UnsupportedEffectStates) != 0` to leaving it out entirely — an
unsupported Effect record no longer refuses the Sack or the actor holdings
batch that carries it.

## Divergence rows

- **DIV-962** (new, FIDELITY-DEBT, OPEN; STORY 1136 HOTFIX correction): this
  story's own `savedGroundContainers` is carried, never enacted, but it is a
  SECOND carrier — `sim.SavedObjectContainer` (story 1115) already carries
  the same `+0x1c`/`+0x20` pair, hashed, and enacted by `putGroundObject`'s
  sentinel mint, `weight()` on drop/pickup, and a dying actor's pack spill.
  No Sack-side rule reads either carrier's own value back: the original's
  own load derive (`ITEM-LOAD-005`) reaches a container's own `+0x20` only
  through an actor's own `+0x7c` reference, which no Sack carries. Neither
  carrier validates a Sack's own tail against its item list on LOAD; a
  hostile weight sum or out-of-range insertion index is carried unchanged,
  because `SAV-671`'s census finds the sentinel on all 307 sampled records.
  A file-vs-carried comparison over the full preserved corpus (72 owner
  saves, EN and RU), both sides through `sav.File.GroundSacks`, finds 51 of
  51 world-half saves carry a nonempty Sack population, zero unexplained
  differences, and a native-export byte comparison against the same file
  matches for every save; a second, genuinely independent comparison against
  `sim.SavedObjects().Containers` (story 1115's own decode, never
  `GroundSacks`) agrees on the same 51 of 51 saves.
- **DIV-594** (updated; STORY 1136 HOTFIX correction): now records that an
  item's own unsupported Effect record no longer refuses the Sack that holds
  it, and that the container's own `+0x1c`/`+0x20` pair already was a native
  Sack field on `sim.SavedObjectContainer` (story 1115) — `sim.SavedGroundContainer`
  (DIV-962) is a second, unhashed carrier of the same pair, not the first.
- **DIV-748** (updated): now records the same no-longer-refuses behaviour
  for the actor holdings admission batch, the same per-item standing
  DIV-594 records for ground Sacks.
- DIV-963 through DIV-967 (reserved, unused) are returned unclaimed.
- DIV-944 (story 1133, projectile store) and DIV-755 (actor carried weight)
  are different, narrower surfaces; DIV-962 restates DIV-755's own
  carried-state/live-recompute distinction for a Sack rather than an actor.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean
  (net +1 from this story's own witness on top of master's 201 after story
  1134; 202 registered names; `population.txt` is 206 lines, 4 of them
  comments).
- `scripts/check-no-game-assets.sh`: clean.
- `TestGroundContainerCorpusAudit1136` (`-tags sessioncorpusaudit`, opt-in,
  both lawful roots): 72 files audited (51 world-half, 20 between-mission, 1
  unreadable), 51 non-empty, 0 mismatching, identical on EN and RU (`audited
  72 file(s): 51 world-half, 20 between-mission, 1 unreadable, 51 non-empty,
  0 mismatching`). All five pre-existing `sessioncorpusaudit` suites
  (sessions 1130, cell records 1131, SpellEffect graph 1132, projectiles
  1133, mover/route 1134) re-run clean alongside it on both roots after
  merging current master.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation, run on the merge
  commit): 8 packages, 202 gated tests, 2 roots; **202 of 202 ran and 0
  lacked a subject, on EN and again on RU.**
- `scripts/check-claim-citations.sh`: ok, 1636 distinct citations resolve
  against 1985 claims and 311 experiments under 1012 prefixes, clean on the
  merge commit after merging current master (`edfe4641`, story 1134).
- `pipeline/check-div-claims.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, run on the merge commit): exit 0, 107 rows flagged
  (informational; DIV-962 cites `ITEM-STACK-003`, which carries an unrelated
  partial retraction — the Potion stack-merge enchantment-separator clause,
  not the five weight-times-count maintenance sites this row leans on).
- `pipeline/check-milestone.sh`: not run. This story's restoration is
  reached only when resuming an original save, a path the census's own
  fresh, non-resumed mission drive never exercises — story 1132/1133's own
  reasoning for the same population.
- `go build ./cmd/missionrun` (from this worktree) plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master,
  whose own recorded census carries no `cannot run` line for either mission
  (`pipeline/milestone-baseline.txt`).

## Open debt

**Native mission SAVE has no interactive path yet.**
`exportOriginalGroundContainers` is exercised only by this story's own
corpus audit and release witness. No production menu, hotkey or session
flow calls it; the owner's own standing rule against inventing a production
mission SAVE entry point applies here the same way it did to
`exportOriginalProjectiles`/`exportOriginalSpellEffects`/
`exportOriginalCellRecords` in stories 1131-1133. The building block is
tested and correct; wiring it to a player-visible SAVE action is a separate,
later decision.

**No live consumer reads the carried tail** (DIV-962). Neither field feeds
any Sack-side rule today: `pourSack` always appends at the live tail
regardless of the carried insertion index, and no Sack-side weight display
or capacity check exists to read the accumulator. A future story that wants
either value to matter needs a claim naming a Sack-side consumer, which does
not exist today — `ITEM-LOAD-005`'s own load derive only ever reaches a
container through an actor's `+0x7c` reference.

**`savedGroundContainers` duplicates a carrier the world already has, live**
(STORY 1136 HOTFIX, F-1/F-8). `sim.SavedObjectContainer` (story 1115)
already carries a ground Sack's own `+0x1c`/`+0x20` pair, inside
`World.encode()`, enacted by `putGroundObject`'s sentinel mint and
`weight()` calls on drop/pickup and by a dying actor's pack spill.
`sim.World.savedGroundContainers` (this story) is a second, unhashed
carrier of the same pair, read only by the unshipped
`exportOriginalGroundContainers`, which therefore re-exports the file-time
value rather than the first carrier's live one — harmless only while
nothing calls it in production. Removing the duplicate, or re-pointing the
export at the live `SavedObjectContainer` ledger, is a separately scoped
story (DIV-962). Neither carrier validates a Sack's own tail against its
item list on LOAD: a hostile weight sum or an insertion index outside the
list is carried unchanged, because `SAV-671`'s census finds the `+0x1c`
sentinel on all 307 sampled records with zero exceptions and a refusal here
would reject loads the original itself accepts — narrower than the original
story brief's own requested cross-check refusal, not previously recorded as
a departure.

**No preserved save exercises the Effect-dispatch change, on either path**
(STORY 1136 HOTFIX, F-4/F-5). Scanning all 72 corpus saves' item records
finds 172 `Effects` references across 60 files, all class `Effect` at state
0 — zero non-`Effect` classes, zero nonzero states. So neither the ground
and holdings "no longer refuses" behaviour nor the party path's own
consequences below has been observed on real data; every acceptance in this
story's own tests is synthetic. On the party path specifically, an item
whose last ordered Effect reference diverts loses source equip eligibility
(`SourceEquipment.EffectsUnsupported`, read by `sourceEquipmentSlot`) if it
is a Weapon/Armor/Shield, and — for any item — its own `Effects` list going
empty flips `sim.ItemEqual`'s stack test from keep-apart to merge
(`HasEnchantment`, `pkg/sim/item.go`), which the original never does because
it keeps the diverted record in the item's own list.
`TestOriginalPartyLastEffectDivertingChangesEquipAndStackEquality`
(`pkg/game/originalitem_test.go`) pins both as current, accepted behaviour,
not a fix. Three questions are open for research, no expected answer given:

1. In an original ROM1 save, can an item's serialized `Effects` list
   reference a class other than `Effect`, and what in-game action writes
   such a record?
2. Can an item's own `Effect` record carry a nonzero Token `+0x0c`, and
   which routine writes that value?
3. `SAV-671` and `SAV-672` cite EXP-0313's census keys
   `container_owner_class_world_half_<class>,<bool>` and render them in
   prose as "town-half" and "mission-half", reporting `Sack` 307 town-half
   and 0 mission-half. Which of the two save populations does
   `world_half=true` name?

**The `.ags` resume drops this store while nothing reads it**, the same gap
DIV-932/DIV-938/DIV-944 name for `savedCellRecords`/`savedSpellEffects`/
`savedProjectiles`: `World.MarshalBinary` does not encode
`savedGroundContainers`, so no `.ags` byte form carries it, and
`resumeWorld`'s same-receiver decode carries the receiver's own prior
(empty) value across a mid-mission resume. Not exercised end to end by this
story's own tests, on the same reasoning as its three precedents: no
simulation step reads this field today, so no hash or player-visible
behaviour depends on it surviving a resume.

**Item Effect state has no binding to the SpellEffect graph, and none is
supported by any claim read for this story.** `SAV-TOKENLOAD-094` states
plainly that nested Item/Effect reach from the top-level graph's own
lifecycle dispatch is not established. If a future claim names such a
binding, `sav.Piece.UnsupportedEffectStates` and the ordered
`Effects []ItemEffect` this story leaves unchanged are both still available
to build it from without a decode-time redesign.

## Touched surfaces

- `pkg/formats/sav/ground.go`: `GroundSack.InsertIndex`/`Accumulator`,
  `GroundContainerTail`, `File.SetGroundContainerTails`.
- `pkg/formats/sav/ground_test.go`: walk assertions plus round-trip,
  leave-alone and shape-mismatch tests for the new writer.
- `pkg/formats/sav/program.go`: `Record.ContainerTailOff`.
- `pkg/formats/sav/holdings.go`: doc comment only; decode predates this
  story.
- `pkg/formats/sav/party.go`, `pkg/formats/sav/itemeffect_test.go`: widened
  Effect dispatch condition and its regression.
- `pkg/sim/savedgroundcontainer.go` (new), `pkg/sim/savedgroundcontainer_test.go`
  (new): `SavedGroundContainer` and `World` accessors, round-trip/detach
  contract, the staging-carry regression, and the same-receiver
  no-priming proof.
- `pkg/sim/world.go`: `savedGroundContainers` field.
- `pkg/sim/binary.go`: carry it across `UnmarshalBinary`'s composite
  literal.
- `pkg/sim/originalliving.go`: prime it before `ImportOriginalLivingActors`'
  own staging decode.
- `pkg/sim/nostate_test.go`, `pkg/sim/world_test.go`: field/method-set pins.
- `pkg/game/originalgroundcontainers.go` (new): the sav<->sim converters,
  LOAD and export entry points.
- `pkg/game/originalgroundcontainers1136_corpus_test.go` (new,
  `sessioncorpusaudit`): file-vs-live and whole-file native-export audit.
- `pkg/game/originalgroundcontainers1136_release_test.go` (new): release
  witness over `game0021.sav`.
- `pkg/game/originalground.go`, `pkg/game/originalholdings.go`: narrowed
  refusal conditions; staging prime in `restoreOriginalActorStock`.
- `pkg/game/originalground_test.go`, `pkg/game/originalholdings_test.go`:
  updated refusal-enumeration cases, new restore-despite-unsupported-Effect
  test, new staging-carry regression.
- `pkg/game/originalsave.go`: wire both original LOAD doors and
  `OriginalSaveResume`'s report/`String()`.
- `pkg/game/resume.go`: doc comment naming the same-receiver no-priming
  witness.
- `internal/gatedtests/testdata/population.txt`: register the release
  witness.
- `docs/DIVERGENCES.md`: DIV-962 (new), DIV-594 and DIV-748 (updated).

**STORY 1136 HOTFIX** (`pipeline/reviews/story1136-pass1.md`, F-1/F-2/F-3/F-4/F-5/F-6/F-7/F-8; no player or hashed-state effect, all debt):

- `docs/1136/story.md` (this file), `docs/DIVERGENCES.md` (DIV-962, DIV-594):
  correct the "first and only carrier"/"never enacted"/"not a native Sack
  field at all" claims F-1/F-2/F-8 name; record the duplicate-carrier and
  unvalidated-hostile-tail debt; name the two new party-path debt questions
  alongside the three B1 questions copied verbatim.
- `pkg/game/originalgroundcontainers1136_corpus_test.go`: F-3, cross-check
  every adopted Sack's tail against `sim.SavedObjects().Containers` (story
  1115's own independent decode), not only against itself through
  `GroundSacks` twice; corrected doc comment.
- `pkg/game/originalground.go`, `pkg/game/originalholdings.go`,
  `pkg/game/originalsave.go`: F-6, `originalGroundState` and
  `applyOriginalGround` return and fold a per-record diverted-Effect count;
  `restoreOriginalActorStock`'s own closure counts directly;
  `OriginalSaveResume.UnsupportedItemEffects` (new field, wired into
  `String()`).
- `pkg/game/originalground_test.go`: F-6, counter assertion on the existing
  restore-despite-unsupported-Effect test.
- `pkg/game/originalholdings_test.go`: F-6, new
  `TestRestoreOriginalActorStockRestoresDespiteUnsupportedItemEffectAndCountsIt`.
- `pkg/game/originalitem_test.go`: F-4/F-5, new
  `TestOriginalPartyLastEffectDivertingChangesEquipAndStackEquality` pinning
  the party path's own equip-refusal and stack-merge debt.
- `pkg/formats/sav/ground.go`: F-7, `SetGroundContainerTails` validates
  every identity and offset before the first write.
- `pkg/formats/sav/ground_test.go`: F-7, new
  `TestSetGroundContainerTailsRefusesALaterIndexMismatchWithNoPartialWrite`
  (three Sacks, a later-index mismatch, whole-file `bytes.Equal`).
