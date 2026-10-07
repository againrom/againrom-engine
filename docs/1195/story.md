# The permanent SAV round-trip regression instrument

## Intent

`pipeline/SAV-ENDGAME.md`'s "Permanent instrument" names the measuring device
the whole SAV endgame is scored against: "A round-trip regression over the
discovered corpus and our own save points: load, resave, compare semantic
state, reload. It runs beside the existing tagged milestone-2 gate and names
every refusal instead of failing on the first one, so a growing corpus stays
honest." This story adds that instrument. It changes no production behaviour;
it only measures the two SAV interoperability paths that already exist.

The first pass (`90e2dc9`) was RETURNED by its sole adversarial review
(`pipeline/reviews/story1195-review.md`) on two blocking findings: the
instrument's own doc comment claimed a regression bound the code did not
enforce (F1), and it ran nowhere a gate would ever see it (F2). Every
measured number in that pass was independently reproduced and found correct;
nothing here changes the reported baseline counts. This document describes
the corrected, landed shape.

## As-built behaviour

`pkg/game/savroundtrip1195_corpus_test.go`, behind the `sessioncorpusaudit`
build tag (the milestone2 family's own tag).

Two tests, one per interoperability path in `pipeline/SAV-COMPLETION.md`'s
"Broader SAV interoperability paths" table, both driving the exact production
entry point `FrontEnd.ConvertSave` that `saveconvert.exe` also calls — this
measures what a player's own conversion run would do, not a second
approximation of it:

- `TestSAVRoundTrip1195AGSCorpus` walks `AGAINROM_AGS_CORPUS` (the owner's own
  `.ags` corpus, ordinarily `engine/saves`; read-only, never written) and, per
  file: decode, restore (ordinary native LOAD), `ConvertSave(raw, "sav", nil)`
  ("AGS to SAV"), reload the resulting SAV bytes on a fresh `FrontEnd`
  (`RestoreOriginal`), compare semantic state.
- `TestSAVRoundTrip1195OriginalCorpus` reuses `milestone2Corpus`
  (`milestone2_acceptance_reader_test.go`) to walk `AGAINROM_SAVE_CORPUS` (the
  original `.sav` corpus, ordinarily `gameversions/saves`) and, per file:
  restore (original LOAD), `ConvertSave(raw, "ags", nil)` ("SAV to AGS"),
  reload the resulting AGS bytes (`Restore`), compare semantic state.

Both tests discover their corpus by walking a directory, in `t.Run(rel, ...)`
subtests, on `milestone2Corpus`'s own shape; neither hard-codes a file list.

**Refusal handling (point 3).** A file that cannot complete the cycle is
recorded as a `sav1195Refusal{rel, stage, err}` and logged by name and stage
(`t.Logf`, never `t.Fatal`); it does not stop the rest of the population and
does not fail the test on its own.

## The correction pass: findings and the two seat rulings

**F1 (blocking) — the instrument now actually enforces the bound its doc
comment claims.** The first pass's doc comment asserted the test "fails only
on a genuine regression ... a save that round-trips today and stops doing
so"; nothing in the code compared any count against anything, so that clause
was false. **Seat ruling: build the real detector, not the one-clause
deletion.** `sav1195Baseline` (one per corpus) now commits the round-tripped
count and the refusal-reason histogram the census already prints — no owner
save name, only reason text and counts, since `rel` (the file name) never
leaves `t.Logf`. `sav1195CheckBaseline` fails the whole test if round-tripped
falls below its baseline, or if any refusal reason (known or newly appeared)
rises above its baseline. Verified by deliberately mutating each baseline
number in turn and confirming the test fails with the expected message, then
restoring the committed values (not part of the landed diff). The doc
comment on `sav1195Baseline` states plainly what this cannot see — a
simultaneous swap where one file starts round-tripping in the same pass that
a different file stops, leaving both totals unmoved — and is the one place
the baseline's update procedure is written down.

**F2 (blocking) — the instrument now runs somewhere a gate sees it.** Neither
test appeared in any script; `pipeline/SAV-ENDGAME.md` claims this instrument
"runs beside the existing tagged milestone-2 gate," which was not true.
`engine/scripts/check-milestone2-acceptance.sh` now also selects
`^TestSAVRoundTrip1195` (widened `-run` pattern), exports an
`AGAINROM_AGS_CORPUS` default (`engine/saves`, beside the script's existing
`AGAINROM_SAVE_CORPUS` default) and guards it with the same
`[ -d ... ] || fail` the SAV corpus already gets. The script's own
success-path `grep` filter — which is what actually hid a passing test's
`t.Logf` output, not the already-present `-v` flag — now also whitelists this
instrument's `SAV-ROUNDTRIP-*` summary lines, so a clean run prints its
numbers instead of nothing.

**F3 — the AGS census's `mismatched=0` is now labeled for what it is.** Every
one of the 105 AGS files refuses before reaching the compare step this pass,
so `sav1195Compare` runs zero times on that corpus; `mismatched=0` was an
empty-set result, not evidence of zero semantic loss. Both census lines now
also print `comparator-exercised=<roundtripped+mismatches>`, and either test
logs an explicit `-NOTE` line when that count is zero.

**F4 — the stale cross-reference moved out of the runtime log line.** The
`review/sav-export-census/RESULT.md` comparison used to print
inside the AGS census's own `t.Logf` format string, reading as a live
recomputation. It is now only in the file's static doc comment, marked
"PINNED HISTORICAL" — a fact about main `86d55b0`, not something this
instrument recomputes.

**F5 (seat ruling, decided rather than deferred) — the comparator was
extended to `Fame` and `Offered`, not merely re-labeled.** story1194 turns
two AGS-corpus refusals into disclosed approximations in exactly those two
`Snapshot` fields, which `sav1195Compare` did not read; the review named this
as the first blind spot this instrument would certify as clean. **Chosen:
extend, not defer.** Both fields are already produced by `FrontEnd.Snapshot`
on both sides of every round trip in this file, so comparing them costs
nothing to build, closes exactly the gap story1194 would otherwise pass
through unnoticed, and — the one sentence defending the choice — nothing in
this story's own worktree (still base `dd39145`, pre-1194) depends on
story1194's actual landed diff to add or verify the comparison: re-running
both tests after the change reproduces `mismatched=0` on both corpora
unchanged, so the extension is free today and armed for whenever 1194 lands.
The doc comment's "what it does not compare" list is also now exhaustive by
name (every remaining `Snapshot` field), not the broader "anything else no
existing comparator reaches" the review found overclaimed.

## Proof

Run from this worktree, `AGAINROM_GOCACHE=<seat>\.gocache`:

```
go test -tags sessioncorpusaudit -count=1 -run TestSAVRoundTrip1195AGSCorpus      ./pkg/game/... -v
go test -tags sessioncorpusaudit -count=1 -run TestSAVRoundTrip1195OriginalCorpus ./pkg/game/... -v
```

**AGS corpus (`engine/saves`, 105 files), EN and RU assets — identical on
both roots:**

```
SAV-ROUNDTRIP-AGS-CENSUS discovered=105 round-tripped=0 refused=105 mismatched=0 comparator-exercised=0
```

Six refusal reasons, with the exact same per-reason counts
`review/sav-export-census/RESULT.md` measured with `saveconvert.exe -to sav`
on main `86d55b0` over the same 105 files (a pinned historical cross-check,
independently reconfirmed file-by-file and reason-by-reason by the review):
never-imported worlds 79, unknown campaign score history 9, local UI
settings 8, unreachable Group graph 4, native-campaign-required city 4,
non-integer camera cell 1. `comparator-exercised=0`: no file reached the
compare step this pass, so `mismatched=0` is an empty-set result, stated as
such by the test's own `-NOTE` line.

**Original `.sav` corpus (`gameversions/saves`, 103 files, 1 unreadable — the
same population `milestone2Corpus` reports), EN and RU assets — identical on
both roots:**

```
SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=102 round-tripped=94 refused=8 mismatched=0 comparator-exercised=94
```

Eight refusals, three reasons: `SpellTransport/PointEffect scheduling ...
remain unbound` (2, a named world-writer boundary from `ExportCurrentWorldSave`
per `pipeline/SAV-COMPLETION.md`), and `original-compatible town save
semantic roster has N characters but only M were faithfully restored` (6,
`2027-09-07`'s own six-person-roster city saves, a known native-writer roster
limit). Zero semantic mismatches among the 94 files that did round-trip
(including the now-compared `Fame`/`Offered` fields), on either lawful root.

Both `go test` runs, all subtests, exit 0 on EN and on RU. The baseline
mechanism itself was exercised by deliberately raising
`sav1195AGSBaseline.roundTripped` and separately lowering one of its refusal
counts and confirming `TestSAVRoundTrip1195AGSCorpus` fails with the expected
message in each case, then restoring the committed values before this commit.

**Regression gates, same worktree:**

- `gofmt -l pkg/game/savroundtrip1195_corpus_test.go` — clean.
- `go build -tags sessioncorpusaudit ./...` — clean.
- `go test -trimpath -count=1 ./...` — all packages `ok`.
- `go vet -tags sessioncorpusaudit ./pkg/game/...` — the same pre-existing
  unkeyed-`BookSpell`-literal findings other files already carry; nothing new
  from this file.
- `bash scripts/check-no-game-assets.sh` — clean.
- `bash engine/scripts/check-milestone2-acceptance.sh <EN root> <RU root>`
  (this worktree's own copy of the script, absolute roots) — `ok` on both
  roots; prints both the 17 `TestMilestone2*` summary lines and this
  instrument's own `SAV-ROUNDTRIP-*` census/reason lines.
- `bash pipeline/check-preserved-installs.sh` — 554 files, every root as
  recorded.

## Open debt

- The baseline mechanism is an aggregate floor/ceiling, not a per-file
  ledger; it cannot see a simultaneous swap (one file starts round-tripping
  while a different one stops in the same pass) that leaves both totals
  unmoved. `sav1195Baseline`'s own doc comment names this and the reason a
  per-file ledger was not built instead: it would have to commit owner save
  names (or a hash of them) into a published repository.
- `sav1195Compare` now reaches `Fame` and `Offered` in addition to
  `city1166Check`'s original field list. The remaining `Snapshot` fields
  (`World`, `QuickSpells`, `Difficulty`, `Open`, `Available`, `Taken`,
  `HeroGrantState`/`ConsumedHeroGrants`, `MercenaryState`,
  `Documents`/`DocumentMission`, `Campaign`/`CampaignState`/
  `CampaignMarkerSelected`, `WorldSelectedOnce`, `WorldMapReturn`,
  `OriginalCity`, `ActorManifest`, `SavedDocument`, `Residue`) are still not
  compared; no existing helper reaches them, and battlefield/world producer
  coverage is separately tracked in `pipeline/SAV-COMPLETION.md` (M4, M6,
  M7). A future story extending real coverage should extend
  `sav1195Compare`, not add a parallel comparator.
- No divergence rows were needed. DIV-1323 through DIV-1325 were reserved for
  this story and are returned unused; this is a measurement instrument with
  no new player-visible or hashed-state behaviour to record.
- The eight original-corpus refusals and the 105 AGS-corpus refusals are
  today's committed baseline, not a target this story closes; closing them is
  the ordered work in `pipeline/SAV-ENDGAME.md` (M3 through M9). When a later
  story changes those counts on purpose, it must update both baselines in the
  same commit — the procedure is written once, in `sav1195Baseline`'s own doc
  comment.
- story1194 (city-side SAV-always, running concurrently) is expected to
  change the AGS census once it lands: it turns the 9 "unknown campaign score
  history" and some of the 4 "native-campaign-required city" refusals into
  disclosed approximations. Landing it must re-measure both censuses and
  update both baselines in that same reconciliation, not silently absorb the
  shift into a stale number.
