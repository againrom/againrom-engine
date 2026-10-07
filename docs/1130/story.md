# Story 1130 — session trigger-result registers and the two raw session regions

## Status

Gates run on the exact candidate commit; see "Proof" for the full list and
exact counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository, asset-free) clean, including `internal/gatedtests`' checked-in
`testdata/population.txt` census (net +2: the two new release witnesses named
below); `scripts/check-no-game-assets.sh` clean; `pipeline/check-release-tests.sh`
against both lawful roots (8 packages, 192 gated tests, 192 of 192 ran and 0
lacked a subject on each of EN and RU); `pipeline/check-milestone.sh` (seat
script,
`AGAINROM_MILESTONE_DRIVE` pointed at a `missionrun` built from this
worktree) exit 0, unchanged on both roots — expected, since this story's own
restoration only reaches an original-save resume, a path the census's fresh
mission drive does not exercise.

## Player result

A player resuming an original mid-mission save now gets the mission's own
hundred trigger-result registers back — the values compiled checks, authored
variables and the constant preset left in the file — instead of a freshly
recompiled, constant-only set. This joins the cell triggers, latches,
diplomacy and outcome earlier stories already restore. The two further
session-block spans research locates but assigns no meaning to (RawHead,
RawMid) now round-trip byte for byte through resume, including through the
actor-stock staging step every original resume with restorable actors passes
through — closing a real loss this story found reaching every such resume
today, not a hypothetical one.

Native mission SAVE itself is not a wired player-facing feature yet: no
interactive path calls the native writer this story adds (see "Open debt").
Mid-mission continuation remains exclusively the existing lossless `.ags`
envelope; this story's own native writer is a tested building block for a
later story in the same milestone sequence (`pipeline/SAV-COMPLETION.md`,
owner milestone 2).

## Authority

- **`SAV-SESS-031`** (High): the world-half session `Serialize` is 4374 bytes;
  its store arm writes, in order, the 100 trigger-result registers, the 1000
  fire-once latches, a 48-byte span, a 400-byte span, the 50×50 diplomacy
  matrix (with an 8-byte lead-in before the matrix proper), a further 6-byte
  stretch (`u8`, `u8`, `u32`, none named), and the three outcome integers Won,
  an explicitly-unnamed 4-byte gap, and Lose. It locates the 48- and 400-byte
  spans (RawHead, RawMid) precisely but promotes no meaning for either.
- **`TRIG-STORE-002`** (High): the script compiles into the same 100-int
  register file and a 1000-byte latch array; a compiled check owns exactly
  one slot by its own subscript and writes it once per pass; an instant
  writes a slot only through opcodes 3 and 8; the builder bounds-checks
  neither array.
- **`TRIG-SAVE-008`** (current text, partially retracted — load order and
  corpus count amended by EXP-0252/EXP-0254; the field programme and
  world-half scope stand): the session block, including the 100 registers, is
  restored *before* the map trigger programme is rebuilt, reversing an
  earlier, now-refuted order. The rebuild "reconstructs checks, instants and
  patterns around the already loaded arrays and its map-variable arm can
  preset an individual result slot" — i.e. the original's own rebuild
  overwrites only the constant-preset class of slot, after restoring.
  Graded High for the field programme and exact call order, Medium for the
  corrected path distribution; a full post-builder slot differential and a
  running-original trigger witness remain Unknown.
- **`MISSION-SLOT-008`** (Medium): a mission variable (authored, instant-
  written) and a compiled check's own result share the same 100-int array,
  indexed by the same authored number; the shipped corpus already contains a
  real collision (`60.alm`, variable indices 32/33 inside its own 64
  check-node range). The slot law is read from the builder's own counter,
  established for the general upper bound but confirmed on the compiled-order
  reading specifically for `60.alm`.

## As-built behaviour

**1. `pkg/formats/sav` carries the two structures.** `SessionState()` gains
`Registers [100]int32`, `RawHead [48]byte` and `RawMid [400]byte` alongside
the existing latch/diplomacy/outcome fields; `RawHead()`/`SetRawHead()` and
`RawMid()`/`SetRawMid()` accessors read and write the two spans on `*File` at
block offsets 1400 and 1448. `TestSessionBlockLayoutAccountsForEveryByte`
(`pkg/formats/sav/session1130_test.go`) locks the full 4374-byte sum,
including the three small stretches SAV-SESS-031 locates but does not name
(DIV-927).

**2. `sim.OriginalSession` carries both structures into the world.**
`Registers [scriptRegisters]int32`, `RawHead`/`RawMid` are new fields;
`ValidateOriginalSession` explicitly does not check any of the three — each
is a fixed array, so there is no length to police, and every value the file
can hold is legitimate. `ImportOriginalSession` copies `Registers` into
`w.registers`, then re-runs `presetRegisters()` — the same call a mission's
own construction already ran once, over the map's own compiled constants —
and carries `RawHead`/`RawMid` opaquely into `w.rawSessionHead`/
`w.rawSessionMid`.

This project's own restore order is compile-then-overlay-then-represet, not
corrected `TRIG-SAVE-008`'s restore-then-rebuild: a mission is already
compiled (and `presetRegisters` already run once) by the time
`ImportOriginalSession` runs. The two orders reach the same final state for
every slot class, because `presetRegisters` is a pure function of each
compiled constant node alone (`w.registers[c.Register] = c.Args[0]`) and
never reads another slot, so running it a second time after the saved copy
lands each constant-owned slot on exactly the value a real rebuild would give
it. `TestOriginalSessionRestoresRegistersPerSlotClass` and
`TestOriginalSessionRegistersSurviveARealCheckOnlyUntilItRuns`
(`pkg/sim/originalsession_test.go`) witness the three slot classes directly:

- **Constant-owned** (a `ScriptCheckConstant` node's own register): the
  file's saved value is discarded; the slot reads the fresh compile's own
  constant immediately after import (witnessed: file value 999 at register 5,
  reads 73 — the compiled constant — right after `ImportOriginalSession`).
- **Check-owned** (an ordinary runtime check's own register): the file's
  saved value survives import and reads back unchanged until that check's own
  next full tick recomputes it (witnessed: file value -777 at register 9
  reads back -777 immediately after import, then 1 — the check's own answer —
  after one full script cycle).
- **Authored-variable** (an instant-written index that names no compiled
  check, or one that collides with a check's own subscript per
  `MISSION-SLOT-008`): the file's saved value survives import with no further
  overwrite from `presetRegisters` (witnessed: file value 4242 at register 50
  reads back 4242, unchanged).

**3. The native writer (building block).** `exportOriginalMissionSession`
(`pkg/game/originalsave.go`) writes a world's live registers and the two
carried raw regions back into a `*sav.File` — the counterpart to
`ImportOriginalSession`/`applyOriginalSession` on the read side. No
interactive SAVE path calls it (see "Open debt"); it is exercised directly by
function call, by the corpus acceptance instrument, and by the two release
witnesses below.

**4. A real defect found and fixed: the actor-stock staging round trip
silently dropped both raw spans.** `restoreOriginalActorStock`
(`pkg/game/originalholdings.go`) stages the whole holdings handoff through
`ms.World.MarshalBinary()` into a fresh `sim.World`, `UnmarshalBinary`'d back,
so a late refusal (an unrearmable saved loadout) touches neither prior
canonical state nor its weights. `rawSessionHead`/`rawSessionMid` are
carried-not-wire-form — the same choice DIV-041 already made for the ghost
template — so `UnmarshalBinary`'s own composite literal carries the
*receiver's* prior value across a decode, not one the encoded bytes state.
The fresh `staged` receiver's own value starts zero regardless of what
`ms.World` already held, exactly like the ghost template into a fresh world.
Unlike the ghost template, nothing primed it: `mapload.BindSourceDerive`
already primes `sourceDerive` for this exact function, but nothing did the
same for the two raw spans before this story. Every original-save resume
with any restorable actor stock crosses this staging step, so the loss was
not hypothetical — it reached the live in-memory `ms.World` on every such
resume, one call after `ImportOriginalSession` had restored both spans
correctly.

This was found by actually running story 1130's own corpus acceptance
instrument against a real preserved EN save (`game0000.sav`), not by any test
that existed before this story: `RawSessionHead()` read back all-zero despite
the file carrying real non-zero bytes and `SessionApplied == true`. The fix
adds `SetRawSessionHead`/`SetRawSessionMid` (`pkg/sim/script.go`) and calls
both on `staged` before its `UnmarshalBinary`, in `restoreOriginalActorStock`,
priming the fresh receiver the same way `BindSourceDerive` already primes
`sourceDerive` there. `TestResumeOriginalSaveCarriesRawSessionSpansThroughActorStockStaging`
(`pkg/game/originalsave_test.go`) is the regression witness: confirmed to
fail with the exact all-zero-versus-expected mismatch when the fix is
reverted, and to pass with it restored.

**5. Acceptance instrument.** `TestSessionCorpusAudit1130`
(`pkg/game/originalsession1130_corpus_test.go`, build tag
`sessioncorpusaudit`, opt-in — not part of any default gate) is an
independent file-vs-live comparison: for every preserved owner save under
`gameversions/saves` that carries a world half, it reads the file's own
`SessionState()` directly (never through `ResumeOriginalSave`), resumes the
same file through the ordinary front-end path, and compares live registers/
RawHead/RawMid against the file, then re-opens a native export of the live
state and compares that against the file too. The walk is recursive over the
corpus, not one directory level deep. Result: **72 files audited (51
world-half, 20 between-mission, 1 unreadable non-`Asg&` file skipped by that
failure, not by name), 0 mismatching, on both EN and RU** (exact log line
identical on both roots: `audited 72 file(s): 51 world-half, 20
between-mission, 1 unreadable, 0 mismatching`).

**6. Release witnesses.** `TestReleaseOriginalSessionRegistersRestoreOnLoad1130`
and `TestReleaseOriginalSessionRawSpansSurviveActorStockStaging1130`
(`pkg/game/originalsession1130_release_test.go`, new) run through the
ordinary App LOAD path (`releaseFront`, `groundCorpusFile`, `groundAppLoad`,
the same helpers `TestReleaseOriginalCellTriggers1106...` already uses) over
the owner's own `2026-08-24/game0021.sav`, hash-pinned. The registers witness
asserts the live world's exact 100-slot register file matches the source
file (14/100 non-zero in this save) after LOAD, and that a native export
reproduces the source file's register span byte for byte. The raw-spans
witness does the same for RawHead (17/48 non-zero, real ROM1 content in this
save) and, because RawMid is all-zero in every preserved save (DIV-928),
writes one explicitly named diagnostic byte pattern into RawMid before LOAD —
the same limited role the 1106 witness's own `changed-source` variant plays
for cell triggers, a probe of the carry-through mechanism, not a claim about
original runtime content. Both witnesses are registered in
`internal/gatedtests/testdata/population.txt` and pass on EN and RU.

## Divergence rows

Amended (`docs/DIVERGENCES.md`):

- **DIV-026** (persistence / original saves): the "trigger result registers
  ... remain unrestored" clause is removed; a STORY 1130 paragraph records
  the three-slot-class restore, the raw-span carry, and the corpus audit
  result.

New, from the reserved DIV-926 through DIV-931 range
(`pipeline/ALLOCATIONS.md`):

- **DIV-926** (raw session span carry-across): records the staging-round-trip
  defect found and fixed by this story (item 4 above), the same class DIV-041
  already names for the ghost template. Narrowed, not closed: a different
  fresh-World decode outside `restoreOriginalActorStock` — a read-only save
  instrument, or a genuine cross-process `.ags` resume — still loses both
  spans unless it primes them the same way. Remains OPEN for that remainder.
- **DIV-927** (three small unpromoted session gaps): records the 18 bytes
  `SAV-SESS-031` locates but names no meaning for (8 before the diplomacy
  matrix, 6 after it, 4 between Won and Lose), accounted for in the byte-sum
  lock test but neither read nor carried by any path.
- **DIV-928** (RawMid real-content coverage): records that RawMid is 0/400
  non-zero in every one of the 51 world-half saves in the corpus's 72
  preserved owner saves, so the carry-through mechanism is proven but a
  genuinely non-zero RawMid's fidelity has no authentic sample to check
  against.

**DIV-929 through DIV-931 remain unused and retired**: no further named gap
this story found was a new, unclaimed owner-direction choice of its own
distinct from DIV-926 through DIV-928 above.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean
  (net +2, 192 total names).
- `scripts/check-no-game-assets.sh`: clean.
- `TestSessionCorpusAudit1130` (`-tags sessioncorpusaudit`, opt-in, both
  lawful roots): 72 files audited (51 world-half, 20 between-mission, 1
  unreadable), 0 mismatching, on EN and again on RU.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 192 gated
  tests, 2 roots; **192 of 192 ran and 0 lacked a subject, on EN and again on
  RU.**
- `pipeline/check-milestone.sh` (seat script, run because a restored register
  can change a shipped map's tick outcome; `AGAINROM_MILESTONE_DRIVE` pointed
  at a `missionrun` built from this worktree, `AGAINROM_MILESTONE_ALLOW_STALE=1`
  because that override's own binary has no relationship to the unrelated
  `implementation/` checkout's HEAD timestamp the stale-drive check reads by
  default): **exit 0, unchanged from `pipeline/milestone-baseline.txt` on
  both roots** — this story's registers/raw-span restoration is reached only
  when resuming an original save, a path the census's own fresh, non-resumed
  mission drive never exercises, so neither the 28-map script-node count nor
  the mission-10 escort drive's own tick/outcome/census line moved.
- `go build ./cmd/missionrun` (from this worktree) plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master,
  whose own recorded census already carries no `cannot run` line for either
  mission (`pipeline/milestone-baseline.txt`).

## Open debt

**Native mission SAVE has no interactive path yet.** `exportOriginalMissionSession`
is a tested building block, not a wired feature: no UI SAVE action calls it,
confirmed by tracing every caller in this codebase before writing it. This
story's own scope (`pipeline/SAV-COMPLETION.md`, owner milestone 2, step 1 of
an 8-story sequence) stops at the building-block level by design; a later
story in the same sequence wires an interactive path.

**The raw-span carry-across gap is narrowed, not closed (DIV-926).** This
story closes it for `restoreOriginalActorStock`'s own staging round trip,
the one path proven to reach a player today. A different fresh-World decode
— a read-only save instrument, or a genuine cross-process `.ags` resume —
still loses both spans unless a future caller primes them with
`SetRawSessionHead`/`SetRawSessionMid` first, the same class of gap DIV-041
still names for the ghost template.

**RawMid's real-content fidelity is unverified beyond one diagnostic
mutation (DIV-928).** Every preserved owner save available to this project
carries RawMid as all-zero. The carry-through mechanism is proven identical
to RawHead's (same code path), but no authentic non-zero sample exists in the
accessible corpus to check byte-for-byte fidelity against.

**18 bytes of the session block remain neither restored nor carried
(DIV-927).** Three small stretches `SAV-SESS-031` locates but does not name
— an 8-byte lead-in before the diplomacy matrix, a 6-byte stretch after it, a
4-byte stretch between Won and Lose — are accounted for only in the
byte-width lock test, not read into any field or carried opaquely.

Out of scope, unchanged, per the brief: general dead-object graphs, remaining
saved cell fields (DIV-730), remaining actor state, orders, casts, area
effects — DIV-026's own pre-existing remainder, unaffected by this story
beyond the registers/raw-span clause it amends.

## Touched surfaces

`pkg/formats/sav/session.go`, `pkg/formats/sav/world.go` (`SessionState`
gains `Registers`/`RawHead`/`RawMid`; `RawHead`/`SetRawHead`/`RawMid`/
`SetRawMid` new), `pkg/formats/sav/session1130_test.go` (new),
`pkg/formats/sav/sav_test.go` (accessor/offset pins extended).

`pkg/sim/originalsession.go` (`OriginalSession` gains `Registers`/`RawHead`/
`RawMid`; `ImportOriginalSession` restores registers and carries the raw
spans), `pkg/sim/originalsession_test.go` (new: per-slot-class witnesses),
`pkg/sim/script.go` (`SetRawSessionHead`/`SetRawSessionMid` new setters),
`pkg/sim/world.go` (`rawSessionHead`/`rawSessionMid` doc comment corrected:
the carry-across gap is not hypothetical, and is now narrowed by
`restoreOriginalActorStock`'s own priming), `pkg/sim/world_test.go`
(`worldMethods`/`worldWriters` pins extended), `pkg/sim/binary.go`
(`UnmarshalBinary`'s composite literal carries both raw spans across a decode
alongside the ghost template, doc comment extended), `pkg/sim/nostate_test.go`
(pin extended).

`pkg/game/originalsave.go` (`exportOriginalMissionSession` new, native writer
building block), `pkg/game/originalholdings.go`
(`restoreOriginalActorStock` primes `staged`'s raw spans before its decode —
the fix), `pkg/game/originalsave_test.go`
(`TestResumeOriginalSaveCarriesRawSessionSpansThroughActorStockStaging` new),
`pkg/game/originalsession1130_corpus_test.go` (new, build-tag
`sessioncorpusaudit`, opt-in acceptance instrument),
`pkg/game/originalsession1130_release_test.go` (new: two release witnesses).

`internal/gatedtests/testdata/population.txt` (net +2: the two new release
witness names). `docs/DIVERGENCES.md` (DIV-026 amended; DIV-926, DIV-927,
DIV-928 new).

No `formatVersion` or pinned-digest constant moved.
