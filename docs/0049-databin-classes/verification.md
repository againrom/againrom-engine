# Verification

## The finding, first

**R-1 fired.** The contract read a Units or Humans entry's trailing strings as a counted array,
because the group title arrays are counted and the source claim names the entry's field a
`CStringArray`. It is not counted on the wire: the strings are a fixed run, two on a Units row and
ten on a Humans row. The counted walk desynced inside Units entry 26 of the shipped file and
surfaced at `+0x4400` as an undefined length escape — the other half of the same open question
catching this one. The residue test is what made it visible, exactly as the risk said it would.

Repaired: the contract's I/O example (one byte, no overrun), the grammar table and the entry
reader, and both synthetic fixtures, which now spell **both** string shapes so a walk that confused
them could not be tested against a stream making the same mistake. `provenance.md` carries the
falsification as a dated append beside the question, under a declared overrun. The walk then tiles
the shipped file exactly and reproduces every collection count the source publishes — a whole-file
agreement the story had before only as a promise.

## Gate at the tip

```
$ go build ./... && go vet ./...                       clean
$ go test -trimpath -count=1 ./...                     31 packages: 28 ok, 3 no test files, 0 FAIL
$ sh scripts/check-no-game-assets.sh                   clean (tree scan)
$ sh scripts/check-doc-budget.sh                       ok; 0049 under a DECLARED OVERRUN (provenance 8192)
$ sh scripts/check-sdd-audit.sh                        ok (35 notes/warnings, none enforced); FAIL set EMPTY
$ gofmt -l $(git ls-files '*.go')                      0 of 303 files
```

Between T5 landing and this file existing the audit reported exactly one FAIL — `0049: every task
in tasks.md has landed and there is no verification.md` — which is the state this file closes.

## Criteria

| id | witnessed by | result |
|---|---|---|
| AC-1 | `TestParseReadsBackWhatWasWritten` — all eight groups, every entry kind, both counting rules, the shared title arrays, an empty parameter array, an empty name, a 300-byte name at the escape, `Consumed == len(in)` | pass |
| AC-2 | `TestParseRefusesMalformedStreams` — five streams, each refused with the group and collection named, each yielding a nil table | pass |
| AC-3 | `TestAllSentinelRowIsExactlyTheDefaults` (field by field, not one equality) and `TestEveryMappedFieldHoldsItsOwnSlot` (a hand-written expected value; experience lands on slot 37) | pass |
| AC-4 | `TestTheDamageSelectorRoutesAndRefuses` — `-1`, `0`, `3` give `(40, 30)`; `3` alone marks always-hit; `1` and `2` refuse and name the entry | pass |
| AC-5 | `TestEveryArmIsTakenAndCounted` — eleven placements, each asserted on arm **and** entry — plus the four search tests in the definition tier | pass |
| AC-6 | `TestTheAdjustmentTable` over `{0,1,99,100,65535} x {1,2,3}` against a hand-written table, and `TestAFourthDifficultyIsRefused` | pass |
| AC-7 | `TestTheSingleArgumentEntryPointIsUnmoved` (digest and byte-form length pinned to values read off the tree **before** T4 ran) and `TestAWorldBuiltWithATableCarriesTheResolvedHealth` | pass |
| AC-8 | the tool over a lawful install and ten shipped maps — figures below | pass |
| P-1 | `TestNoLoadedFieldHoldsTheSentinel` — three rows, every field walked by reflection | pass |
| P-2 | `TestParseIsAFunctionOfItsBytes` — one stream parsed twice equal, and the table survives its input being overwritten | pass |
| P-3 | `TestEachSlotHasExactlyOneOutcome` — each of the 38 slots driven alone out of an empty row against a table of expected outcomes, the four dropped slots asserted to move nothing | pass |
| P-4 | `TestTheAdjustmentIsPure` — two applications equal at every value, the argument unmoved, value 2 the identity | pass |
| SC-1 | AC-1 and AC-2's tests | pass |
| SC-2 | AC-3's two tests | pass |
| SC-3 | `TestEveryMappedFieldHoldsItsOwnSlot` (experience 137, reachable only if 33–36 were consumed) and P-1's test | pass |
| SC-4 | AC-4's test | pass |
| SC-5 | AC-5's tests — nine outcomes plus two more (a definition id naming nothing, and a table holding one collection) | pass |
| SC-6 | AC-6's test | pass |
| SC-7 | AC-7's tests, plus `FromALMWith` over a nil table hashing equal to `FromALM` | pass |
| SC-8 | the gate above, including the two import-graph rows added for the new package | pass |
| SC-9 | the install run — figures below | pass |
| SC-10 | both mutants, below | pass |

## Mutants

Two applied, one at a time, whole tree run, each reverted and the file's SHA-256 re-checked
(`7b4a8d0170fb8596314c47d6baa281062215567d761bcb207b24cafcea647275`). Neither survived its first
run.

```
(a) the -1 test removed from the slot helper, so a sentinel is stored
    killed by  TestAllSentinelRowIsExactlyTheDefaults
               TestNoLoadedFieldHoldsTheSentinel  (subtests: every_cell_empty,
                                                   the_sentinel_only_where_it_hurts)
               TestEachSlotHasExactlyOneOutcome
(b) the case for slot 33 deleted, so the cursor advances one slot short
    killed by  TestEveryMappedFieldHoldsItsOwnSlot   (XPValue reads slot 36's value)
               TestEachSlotHasExactlyOneOutcome      (slot 36 moves XPValue; slot 37 moves nothing)
```

## The install run

`classdump -databin <world.res> [<map.alm> [<difficulty>]]`, built into the untracked
`builds/0049-databin-classes/`. Counts only — no name, no string, no parameter value is printed.

```
world/data/data.bin: 88327 of 88327 byte(s) consumed, 0 left over
collection    written   titles   params        Units parameter widths: 55 x56
Shapes              5       11        0        Units rows with parameters: 26 27 64..116 118
Materials          16       11        0
Magic              50       30       50        Horror.alm, 1815 placements, difficulty 2:
Armors             30       18       30          npc 0/0  server-id 474/474
Shields             9       18        9          humans 1/1  units 1340/1340
Weapons            27       18       27
MagicItems         49        4       49        13 distinct adjusted maxima over the units arm,
Units             118       57       56        none of them the provisional 100:
Humans            215       28      210          d1  27 40 54 81 108 112 135 162 165 216 540 675 811
Buildings          66        9       66          d2  41 61 82 123 164 170 205 246 250 328 819 1024 1229
Spells             28       24       28          d3  61 91 123 184 246 255 307 369 375 492 1228 1536 1843

ten shipped loose maps: 5761 placements, 5761 on a named arm, 5761 resolved, 0 residue
  Forester 672 (53 + 619)   Horror 1815 (474 + 1 + 1340)   Islands 434 (58 + 376)
  Kids 51 (51)              LuMoir 228 (15 + 213)          Waters 273 (48 + 225)
  Beast 964 (186 + 778)     Cross 774 (148 + 626)          Kids2 88 (1 + 87)
  Tomb 462 (58 + 404)
```

Every entry and title count reproduces the published figures for this file, and all 56
parameterised Units rows carry 55 values. The health census is a second, independent check on the
arithmetic: `1229 -> 811 / 1229 / 1843` and `819 -> 540 / 819 / 1228` are the integer forms' answers
read off the shipped population rather than off a test.

## Movement domains, and what this story does not do

An open question against a sibling story asks what the three movement domains may enter. **This
story reaches none of them.** The movement-type column is carried on the definition and read by
nothing: `pkg/mapload` sets no `Domain` on any entity, so every placement a world builds stays the
zero value, ground. The measured census is therefore **0 non-ground movers on every map**, at every
difficulty, and a world's canonical byte form is unchanged — which the pinned digest above says
outright. The column is decoded and parked; whatever that question settles, it settles for the
story that consumes it.

## Not witnessed

- **A selector outside `{<=0, 1, 2, 3}`.** The contract enumerates four arms and refuses two; it
  says nothing about 4 and above, so those take arm 0, the same reading its `<= 0` arm has. No
  shipped row exercises it (48 rows absent, 8 at `3`), so this is a decision recorded, not measured.
- **The length escape past the 16-bit form** stays refused rather than guessed, and is now measured
  unexercised on this root: the walk tiles the file without meeting one.
- **The campaign maps.** The tool reads a loose `.alm`; the 28 embedded maps live inside an archive
  it does not open, so the placement census above covers the ten loose maps alone.
- **A second shipped root.** One root was read. The source records that the Units collection is
  identical across roots and that Humans differ on six slots, so the humans-arm figures are this
  root's.
