# verification — 0150 the explored map

## The result

A save the owner made in the original game loads in this build with the ground he had already
uncovered still uncovered. On his own `game0009.sav`, labelled `666`, mission 20, the build restores
**3475 of 20736 cells, 16.8 %**, which is the file's own figure.

This story's result is visible in a build, not in the script-gap census. The census is unchanged and
is recorded below.

## Gate

Run in the worktree at `<seat>\wt-0150`, on the branch head.

| Command | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l $(git ls-files '*.go')` | 0, no output |
| `go test -count=1 -trimpath ./...` | 0 |
| `bash scripts/check-no-game-assets.sh` | 0 |
| `bash scripts/check-doc-budget.sh` | 0 |
| `bash scripts/check-sdd-audit.sh` | 0 |
| `bash scripts/check-hotfix-ledger.sh` | 0 |

`check-sdd-audit`'s note and warning **counts** are not comparable from a worktree, which has no
`builds/`. Only its FAIL set is enforced and it is empty.

There is no `tasks.md`: one lane implemented the whole slice in one context. Every FR and DD is
accounted for below.

## Script-gap census

Baseline, `pipeline/milestone-baseline.txt`: mission 10 **17**, mission 20 **13**, on both roots.

    go build -o <bin>/mr ./cmd/missionrun
    for m in 10 20; do AGAINROM_ASSETS=<root>/gameversions/en <bin>/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done

Measured on this branch: mission 10 **17**, mission 20 **13**. **Unchanged**, which is the intended
result: this story runs no script node. The census drives a fresh start, and a fresh start carries no
save and therefore no explored record.

## Evidence against the preserved corpus

Produced by `cmd/savtool fog` and `cmd/savtool verify` (golden rule 2: tests are synthetic, evidence
comes from a tool under `cmd/`). No save bytes are in this repository.

Corpus as it stands now: **23 files, 18 distinct by MD5**, over four directories —
`gameversions/en` (4), `gameversions/ru` (4), `gameversions/saves/2026-08-02` (12),
`gameversions/saves/2026-08-12` (3). The EN and RU install saves are the same four files by
content. Research measured 15 files in `SAV-FOG-061`; the corpus has grown since, and nothing in it
refutes that claim.

### The three regions (AC-1, FR-1)

On all 23 files, the state store's declared extent plus the trailing region equals the tail exactly.
The store is **28 records / 9 sections / 19 leaves** on all 22 mid-mission files and **22 / 7 / 15**
on the one between-mission file, `saves/2026-08-02/game0010.sav`. The trailing region is **268**
bytes on every mid-mission file and **310** on the between-mission one, and its first dword is **10**
on the `10.alm` files, **20** on the `20.alm` files and **30** on the between-mission one.

This reproduces `SAV-TAILEXT-062` on 23 files against the 15 it was published on.

### The explored record (AC-2, AC-4, FR-4)

`sum(Data)` equals **6400** on every `10.alm` save and **20736** on every `20.alm` save, which are the
two maps' cell counts. `FirstState` is **0** on all 22 files that carry the section. The
between-mission file carries no `Fog` section and is reported as carrying none, not as an error.

| File | Mission | Runs | Cells | Explored | % |
|---|---|---|---|---|---|
| `en/game0000.sav`, `ru/game0000.sav` | 10 | 35 | 6400 | 347 | 5.4 |
| `en/game0001.sav`, `ru/game0001.sav` | 10 | 51 | 6400 | 666 | 10.4 |
| `en/game0002.sav`, `ru/game0002.sav` | 10 | 81 | 6400 | 1514 | 23.7 |
| `en/game9999.sav`, `ru/game9999.sav` | 10 | 1 | 6400 | 0 | 0.0 |
| `saves/2026-08-02/game0000.sav` | 20 | 65 | 20736 | 686 | 3.3 |
| `saves/2026-08-02/game0001.sav` | 20 | 33 | 20736 | 231 | 1.1 |
| `saves/2026-08-02/game0002.sav` | 10 | 81 | 6400 | 1514 | 23.7 |
| `saves/2026-08-02/game0003.sav` | 10 | 81 | 6400 | 1603 | 25.0 |
| `saves/2026-08-02/game0004.sav` | 10 | 119 | 6400 | 2255 | 35.2 |
| `saves/2026-08-02/game0005.sav` | 20 | 111 | 20736 | 1212 | 5.8 |
| `saves/2026-08-02/game0006.sav` | 10 | 137 | 6400 | 2494 | 39.0 |
| `saves/2026-08-02/game0007.sav` | 20 | 181 | 20736 | 2152 | 10.4 |
| `saves/2026-08-02/game0008.sav` | 20 | 225 | 20736 | 2888 | 13.9 |
| `saves/2026-08-02/game0009.sav` | 20 | 273 | 20736 | 3475 | 16.8 |
| `saves/2026-08-02/game0010.sav` | 0 | — | — | no `Fog` section | — |
| `saves/2026-08-02/game9999.sav` | 20 | 1 | 20736 | 0 | 0.0 |
| `saves/2026-08-12/game0011.sav` | 10 | 29 | 6400 | 152 | 2.4 |
| `saves/2026-08-12/game0012.sav` | 10 | 35 | 6400 | 248 | 3.9 |
| `saves/2026-08-12/game9999.sav` | 10 | 1 | 6400 | 0 | 0.0 |

`game0009.sav` reproduces `SAV-FOG-061`'s published **273 runs / 3475 set cells** exactly, and
`game0011.sav` and `game0012.sav` reproduce its **29/152** and **35/248**.

### Re-emission (AC-3, FR-3)

`savtool verify` over all 23 files: **23/23 byte-identical**. Splitting the tail moved no byte.

## The restore, on a real save (AC-5, AC-6, FR-5, FR-7)

Three separate processes, so nothing crosses in memory.

Process 1 — load the original save through the LOAD GAME seam and write our own save from what came
back:

    savecheck -assets <root>/gameversions/en -saves <tmp> \
      -orig <root>/gameversions/saves/2026-08-02 -name game0009.sav -resave load

    savecheck: loaded game0009.sav: town=false gold=0 party=1
    resume: mission 20 map "20.alm" label "666" outcome 1 money 600
            ...
            explored map: 3475 of 20736 cells (16.8%) RESTORED
    savecheck: resumed at tick 0, hash 8e2e433d9e24acd8, 57 entities, 0 commanded,
               3475 of 20736 cells explored (16.8%)
    savecheck: RESAVED as save-20260812-214704.ags (ours), 3475 of 20736 cells explored

Process 2 — load our own save, knowing nothing but the directory:

    savecheck -assets <root>/gameversions/en -saves <tmp> load

    savecheck: loaded save-20260812-214704.ags: town=false gold=0 party=5
    savecheck: resumed at tick 0, hash 8e2e433d9e24acd8, 57 entities, 0 commanded,
               3475 of 20736 cells explored (16.8%)

The count is read off the plane the map screen draws from (`FrontEnd.LiveExplored`), not off the
bytes the restore was handed, so it measures the restore rather than the decode. The world hash is
identical across the two loads, which is the byte form saying it did not move: exploration is not
simulation state in this tree.

## The 16.8 % against the owner's estimate

`SAV-FOG-061` records an open discrepancy: the owner's estimate from the original's screen was 40 to
50 %, and the file records 16.8 %. It is not resolved here and nothing was tuned to close it.

What this build restores is the file's own recorded cells, 3475 of 20736. If the screen looks sparser
than 40 to 50 %, that is the expected state. How the original expands recorded cells at draw time is
Unknown in the claim — its render gate ORs four neighbouring corner words, so the drawn extent may
exceed the set cells — and authoring a rule for that was out of scope (FR-6). The render gate is
unchanged.

**Nobody has watched the LOAD window on a screen.** No synthetic input was sent to the owner's
desktop. What the window would show is witnessed through the same seam by `savecheck`, which calls
the same `SaveSeams` list and load functions the window calls, and prints the same caveat line the
window puts on its message line. That is not the same as having seen it.

## Requirement accounting

| Id | Where it landed | What witnesses it |
|---|---|---|
| FR-1 | `pkg/formats/sav/container.go` `splitTail`, `pkg/formats/reg/reg.go` `Size` | `TestTheTailSplitsAtTheStoreOwnExtent`, `TestSizeReportsTheFramedExtent`; corpus AC-1 above |
| FR-2 | `splitTail`'s two error arms | `TestAnUnframedTailIsCarriedWhole`, and every pre-existing `sav` fixture, whose 4-byte tail takes that arm |
| FR-3 | `File.Marshal` | `TestMarshalPutsTheTailBackWhole`; `savtool verify` 23/23 |
| FR-4 | `pkg/formats/sav/fog.go` `File.Fog` | `TestFogDecodesRunsIntoCells`, `TestFogRefusesANegativeRun`, `TestNoFogSectionIsNotAnError` |
| FR-5 | `pkg/game/originalsave.go` `readOriginalFog`/`fogForMap`, `pkg/game/resume.go` `applyExplored`, `pkg/game/frontend.go` `missionOpener` | `TestFogForMapSizesTheRecordAgainstTheMap`, `TestApplyExploredOnlyEverORs`, `TestApplyExploredRefusesAnotherMapExtent`; the `savecheck` run above |
| FR-6 | No change made to `pkg/ui/fog.go`, `pkg/render/terrain` or any drawable gate | `git diff --stat 695cc60..HEAD` touches neither package; `pkg/ui`'s own fog tests are unmodified and green |
| FR-7 | `SnapshotResidue.FogCols/FogRows/FogExplored`, which 0143 already carries | `TestARestoredExploredPlaneSurvivesOurOwnSave`; the two-process `savecheck` round trip above |
| FR-8 | `cmd/savtool` `fog` verb, `cmd/savecheck` `-resave` and the explored line, `FrontEnd.LiveExplored` | The outputs quoted above |
| FR-9 | `OriginalSaveNote`, `OriginalSaveResume.fogLine`, the type doc in `originalsave.go`, the writer note in `fog.go` | `TestTheOriginalSaveCaveatIsSayableOnOneLine` (now requires `explored map`), `TestTheFogLineNamesWhichOfTheThreeCasesHolds` |
| AC-1..AC-4 | — | Corpus tables above |
| AC-5, AC-6 | — | The three-process `savecheck` run above |
| AC-7 | `applyExplored` ORs and clears nothing; `fogPlane.refresh` unchanged | `TestApplyExploredOnlyEverORs`, `TestExploredNeverShrinksAndVacatedCellsStayExplored` (pre-existing) |
| AC-8 | — | Census unchanged, 17 and 13 |
| DD-1 | `internal/archtest/dag.go`: `pkg/formats/sav -> pkg/formats/reg` | `internal/archtest`'s own import-graph test, which is fail-closed |
| DD-2 | `reg.Size` | `TestSizeReportsTheFramedExtent`, `TestSizeRefusesWhatItCannotSplit` |
| DD-3 | `File.Store`, `File.TailRest`, `File.StateStore` | `TestMarshalPutsTheTailBackWhole`; the `sav` fixture gained a `tail` field so both arms are exercised |
| DD-4 | `splitTail` returns rather than erroring | `TestAnUnframedTailIsCarriedWhole` |
| DD-5 | `Fog.Cells` is flat; `fogForMap` supplies the width | `TestFogForMapSizesTheRecordAgainstTheMap`'s second half |
| DD-6 | `pkg/sim/binary.go` untouched | See below |
| DD-7 | `prepare func(*alm.Map) (*originalFog, error)`; apply after `openMission` | The `savecheck` run: the report prints `RESTORED`, which is written from the opener's own answer |
| DD-8 | `applyExplored` is the single writer; `applyResidue` delegates | `TestAFogPlaneOfAnotherSizeIsRefused` (pre-existing, still green through the new path) |
| DD-9 | `savtool fog` prints the trailing region's first dword | The corpus output above |

## The byte form did not change, and version 43 is returned unused

The brief allocated a serialized byte-form version on the premise that persisting the explored map
would widen `pkg/sim`'s form. **That premise is refuted.**

Exploration is not simulation state in this tree. Story 0118 put it in `pkg/game`'s `fogPlane`
because it is per-participant view, and `pkg/game`'s own save envelope has carried it since 0143 as
`SnapshotResidue.FogCols/FogRows/FogExplored`. Spec FR-7 was therefore already met by the envelope,
and this story witnesses it rather than building it: `TestARestoredExploredPlaneSurvivesOurOwnSave`
goes through `EncodeSave`/`DecodeSave`, and the `savecheck` round trip above crosses two processes.

`pkg/sim/binary.go` is not modified on this branch. `formatVersion` stays **41**, and **43** joins
28, 30, 33, 37, 40 and 42 as a permanent gap. The world hash `8e2e433d9e24acd8` is identical whether
the world was built from the original save or from our own save of it, which is the byte form saying
the same thing.

The brief first named 42 and the orchestrator corrected it to 43 mid-story. 42 was allocated to 0147
and returned unused; reusing it would put two byte forms behind one number. No file in this branch
ever contained either number as this story's allocation.

## What was not done

- The trailing region is carried verbatim and not decoded past its first dword, which `savtool fog`
  prints. `SAV-TAILEXT-062` grades its contents Unknown.
- Nothing writes a `Fog` section into an original save. This tree writes no `.sav`.
- Bit 14 is not restored. The save masks `0x8000` alone and `TERR-FOG-145` states the consequence
  directly.
- The `Unit` subtree desync and everything else in the compressed body are untouched.
