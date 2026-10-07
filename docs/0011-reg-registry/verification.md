# Verification — `.reg` binary registry parser (ROM1)

Evidence about the implementation as landed. Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` →
`plan.md` §Success criteria; `DDx` → `plan.md` §Design decisions; `R-x` → `plan.md` §Risks; `Tn` →
`tasks.md`; `REG-*` → the `research/` submodule's claim ledger.

Environment: Go 1.26.1 (the toolchain pinned in `go.mod`), Windows 11, 2026-07-25. Automated evidence
was produced with **no game install visible to the test suite**. The developer-run evidence used a
lawful GOG *Rage of Mages* install, its root passed on the command line; the root is not recorded here.
**No game byte, `.reg` entry, archive or dump output is committed** — the binary, both probes and every
byte of output stayed outside the repository. The `research/` submodule stayed pinned at `8b14881`.
Every figure below was measured in this pass.

## Gate results

Run from the repo root at `9b3d012`:

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `go test -count=1 ./...` | **all packages ok** — `cmd/againrom`, `cmd/mapview`, `cmd/regtool`, `cmd/terraintool`, `internal/archtest`, `internal/notices`, `internal/synth`, `pkg/formats/alm`, `pkg/formats/reg`, `pkg/formats/res`, `pkg/formats/spr256`, `pkg/game`, `pkg/render/camera`, `pkg/render/frame`, `pkg/render/menu`, `pkg/render/terrain`, `pkg/ui` |
| `go test ./internal/archtest` | ok |
| `gofmt -l $(git ls-files '*.go')` | empty |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash scripts/check-no-game-assets.sh --history` | `clean (history scan)` |
| `SDD-Task` bijection over `60d70c6..9b3d012` | 5 IDs (`T1`…`T5`), 0 duplicates; the story's four docs commits carry none |
| `Co-Authored-By` trailers over the same range | 0 |
| `research/` submodule | `8b14881`, unchanged |

That is SC-12, with the suite run on a tree whose tests read no install.

## Automated criteria

Every named test of the story, at `-count=1`. This table records that each test exists and passed here;
what each asserts is the record of the task that authored it.

| SC | Test | Result |
|---|---|---|
| SC-1, SC-2, SC-3, SC-4, SC-5, SC-6, SC-8 | `TestParseTree`, `TestParseFloat`, `TestParseRejectsFraming`, `TestParseRejectsRanges`, `TestParseRejectsValues`, `TestParseBytesVerbatim`, `TestReachability` (`pkg/formats/reg`) | **pass**, all seven |
| P-1's fuzz witness | `FuzzParse` (`pkg/formats/reg`), seed corpus in the ordinary run | **pass** |
| SC-7 | `TestAccessors` (`pkg/formats/reg`) | **pass** |
| SC-9 | `TestDumpRender` (`cmd/regtool`) | **pass** |
| SC-10 | `TestCheckNamesOffendingEdge`, `TestLiveTreeClean`, `TestCheckSimTests`, `TestLoadSkipsNestedModule` (`internal/archtest`) | **pass**; the doc half — `docs/ARCHITECTURE.md` agreeing with the allow-map — is a reviewed edit, not a test |
| R-1's two hand-laid witnesses | `TestRegBuilders`, `TestRegHandLaidStream` (`internal/synth`), plus `TestParseTree`'s own hand-laid stream | **pass** |

## Developer-run evidence (T6 / AC-8 / SC-11)

`regtool` built from `9b3d012` to a scratch path outside the repository. Two throwaway probes were
built in a scratch module with a `replace` to this module, importing `pkg/formats/reg` and
`pkg/formats/res`, and run from outside the tree.

### (a) The census

    $ regtool sweep <install root>
    ...
    44 regs: 44 parsed, 0 failed
    exit=0

    real  0m0.215s

That summary line is the last line of stdout, verbatim. **stderr was empty (0 bytes)**: no archive
failed to open, no entry failed to parse. stdout carried 45 lines — one per registry plus the summary.
Run three times; the three stdout streams are byte-identical (`cmp`).

Per-archive distribution, aggregated from the 44 per-entry lines. The node figure is each line's
`N nodes` field — `Reg.NodeCount`, the header's `nodeCount`, the whole node table at all depths:

| archive | registries | nodes |
|---|---|---|
| `Allods/VIDEO4.RES` | 18 | 239 |
| `Allods/VIDEO8.RES` | 15 | 222 |
| `graphics.res` | 5 | 3 217 |
| `scenario.res` | 3 | 790 |
| `world.res` | 2 | 35 |
| `sfx.res` | 1 | 118 |
| **total** | **44** | **4 621** |

**AC-8's pass condition is met**: 44 registries, 44 parsed, 0 failed, exit 0, and the distribution is
AC-8's — 33 cutscene registries across the two video archives, 5, 3, 2 and 1 elsewhere. 4 621 matches
`REG-REC-032`'s record count. The **18/15** split of the 33 is recorded as **observed**, not as a pass
condition, and matches `REG-LOC-038`.

**Containers.** `sweep` selects by the `.res` extension, case-folded (DD15): **11** such files exist
under the install root, 6 contributing registries and 5 (`Allods/MUSIC.RES`, `main.res`, `movies.res`,
`patch.res`, `speech.res`) none. `REG-LOC-038` counts **12** `&YA1` containers, by signature. Measured
here: of the **120** regular files under the root, **12** begin `26 59 41 31` — the eleven `.res` files
plus `KIDS.LM`, which is **24 bytes**, a header and no payload. The narrower rule reaches one fewer
container and no fewer registry. The census is over physical archives with no VFS layering; `patch.res`
holds no `.reg` entry, so nothing is counted twice or shadowed.

**Reachability (P-3 on real data).** `regtool dump` was then run on all 44 registries — a second pass
through the parser by a different path (`ReadFile` by name rather than entry iteration). All 44 exited
0. The renderer emits one line per node, so the line count is the number of nodes the walk reached:
**4 621 lines in total, and per registry the line count equals that registry's `NodeCount` in 44 of
44**. Zero orphans, zero nodes reached twice, corpus-wide — `REG-VAL-028`'s figure, reproduced.

### (b) The spot-checks

Read off `regtool dump` of `graphics.res` `units/units.reg` (799 nodes) and `Allods/VIDEO4.RES`
`start/01.reg` (14 nodes).

| check (AC-8) | observed | |
|---|---|---|
| `[Global] UnitCount` | `UnitCount = 34` | **34** |
| `[Global] FileCount` | `FileCount = 33` | **33** |
| `[Unit0] DescText` reads as coherent text | a 15-byte string value, every byte in `0x20`–`0x7E`, reading as a coherent two-word English noun phrase naming a unit type. The string is game text and is **not reproduced here** | **yes** |
| `[Unit0] AttackPhases` = element count of `AttackAnimFrame` | `AttackPhases = 7`, `AttackAnimFrame` renders 7 elements | **7 = 7** |
| `[Unit0] MovePhases` = element count of `MoveAnimFrame` | `MovePhases = 8`, `MoveAnimFrame` renders 8 elements, unabbreviated | **8 = 8** |
| a cutscene `startfade`/`endfade` pair reads exactly `0.0` and `1.0` | `start/01.reg` `Fading1`: `startfade = 0.0`, `endfade = 1.0`; `Fading2`: `startfade = 1.0`, `endfade = 0.0` | **yes** |

Two further counts from the same dump: `units.reg`'s root holds **36** sections — `Global`, `Files`, and
**34** named `Unit0`…`Unit33` — and `Files` holds **33** keys. `UnitCount` equals the number of `Unit*`
sections and `FileCount` the number of `Files` keys.

**The phase/frame agreement holds for `Unit0` and is not a corpus invariant.** Over all 34 `Unit*`
sections there are 68 possible (`*Phases`, `*AnimFrame`) pairs: **39 agree**, **3 disagree** (the frame
array holds more elements than the phase count), **1** has a zero-length string where the array would
be, and in **25** one or both keys are absent. AC-8 asserts the agreement for `Unit0` only; the wider
sweep bounds what the check proves. *Inference, labelled:* in the three disagreements the frame array
repeats an earlier element instead of counting up.

### Corpus properties measured from the parsed trees

From the probe walking `Node.Type` over all 44 registries — measured on the type, not inferred from how
`dump` renders it.

| property | measured over | observed | claim |
|---|---|---|---|
| type histogram | all 4 621 nodes | string 873 · directory 512 · int32 2 792 · float64 **122** · int32-array 322 (sum 4 621) | `REG-KIND-034`: 873 / 512 / 2792 / 122 / 322 — **five for five** |
| any other type | all 4 621 nodes | **0** (a type-8 or type-10 node is a rejection; nothing was rejected) | `REG-KIND-034`: kinds 8 and 10 ×0 |
| kind-4 occurrences | all 44 registries | **122**, all in cutscene registries; **31 of the 33** carry at least one — 30 carry 4 each, `LOGOS/nival.reg` carries 2 (30×4 + 2 = 122); the two without are `LOGOS/1c.reg` and `LOGOS/buka.reg` | `REG-DBL-035`, `REG-LOC-038`: 122 in 31 of 33, same two exceptions |
| distinct kind-4 values | all 122 | exactly **two**: `0` ×**61** and `1` ×**61** | `REG-DBL-035`: 0.0 ×61, 1.0 ×61 |
| byte range | every name and every string value in all 44 | `0x20` – `0x7a`; **no byte ≥ 0x80** | `REG-TEXT-037`: `0x20…0x7A`, 0 bytes ≥ 0x80 |
| longest name; bit 28 | all 4 621 nodes | 15 bytes; **0** set bit 28 | `REG-KIND-033`: no shipped record carries bit 28 |
| bit 4 (sorted children) | all 4 621 non-root nodes | **0** set it | — |
| root `kind` word | all 44 headers | **17** in 44 of 44 | `REG-KIND-033`: every `.reg` carries 17 at `+0x0C` |
| tree depth below the root | all 44 | maximum **2** | — |
| `heapSize`, read from the payload bytes independently of our framing | all 44 | **35 have `heapSize == 0`**, none of those 35 holding a type-0 or type-6 node; **9** have a non-empty heap (`graphics.res` 5, `scenario.res` 3, `sfx.res` 1) | — |
| stream ends exactly at heap end | all 44 | `0x18 + 32·nodeCount + 4 + heapSize == len(payload)` in **44 of 44**, 0 exceptions | `REG-VAL-024`: same closure, 44/44 |

`dump` emitted **no `\xNN` escape** in its 4 621 lines, as that byte range requires; the two literal
`\x` sequences that do appear, on one line in `units.reg`'s `Files` section, are a printable backslash
followed by a printable `x` inside a string value — DD14's non-injectivity, reached by the corpus.

*Inference, labelled:* 61 kind-4 values read back as exactly `1.0`; under the opposite word order the
same eight bytes assemble to a denormal near `5.3e-315`, which cannot render `1.0`.

## Correction to a plan-level baseline fact

`plan.md` R-2 lists "no registry with an empty heap" among the things the corpus cannot exercise. It
does exercise it: **35 of the 44 registries have `heapSize == 0` and hold no type-0 or type-6 node** —
AC-9's fixture shape — so that path is corpus-backed 35 times, not synthetic-only.

## Limitations and what this run does not claim

- **The defensive rules stay unexercised by real data, the empty-heap item struck.** The corpus holds
  **0** orphans and **0** doubly-referenced nodes (above), **nothing deeper than 2 levels**, **no byte
  ≥ 0x80**, **no type-8 and no type-10 node**. The depth cap, the single-parent rule, the orphan
  tolerance, byte fidelity above `0x7E` and both the *unsupported* and *unrecognised* rejections are
  carried by the synthetic criteria alone.
- **No rejection path ran on real data.** Every shipped registry is well formed, so the validation half
  of `Parse` is witnessed only by SC-3…SC-5 and `FuzzParse`.
- **Not measured here:** `REG-VAL-025`'s pool-*tiling* property. Our reader checks each reference
  against the heap window, not the partition, and this run did not test the partition.
- **Left out as data rather than evidence:** the text of `[Unit0] DescText`; the string value carrying
  the literal `\x`; the element values of any array; every dump, extracted entry and archive byte.
- **No observation here was made by a human at a screen**, and none is required.

## Conclusion

`regtool sweep` over a lawful install reports **44 registries, 44 parsed, 0 failed, exit 0**, stderr
empty, distribution as AC-8 states, reproducing byte for byte across three runs; each of AC-8's
spot-checks passes on the values observed; every node of every registry is reached exactly once. Against
the claim ledger this run agrees on every figure it touched — record count, type histogram, kind-4
census and its two values, container count, cutscene split, byte range, root kind word, bit 28,
reachability and structural closure. **No figure disagreed and none was adjusted.**

**AC-8 / SC-11 pass in full**, and with them the corpus half of R-1. R-2 stands except for its
empty-heap item. Every automated criterion (SC-1…SC-10) passes and SC-12's gates are clean. What remains
unevidenced anywhere is the behaviour of the parser's rejection and defensive paths on real data, which
no shipped registry can exercise.
