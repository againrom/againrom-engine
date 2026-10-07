# Verification — 0051

Pin `acb8fb0`, branch `impl/0051-structure-footprints`. Every figure below was produced by running
the command shown; nothing is quoted from a plan.

## Gate

```text
(go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
 sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
 sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))")
EXIT=0        FAIL set: empty
```

**SC-1** ✅ **SC-8** ✅. `-trimpath` is load-bearing here: without it Windows Defender quarantines a
test binary. **SC-4** ✅ — `check-no-game-assets.sh` clean on the tree; `--history` clean over the
full log. **SC-3** ✅ — `internal/archtest` passes with `cmd/mapview`'s row widened by three; no other
row moved and no package left the DAG.

The note and warning **counts** from `check-sdd-audit.sh` are not recorded: a worktree has no
`builds/` (untracked, not ignored), so it warns once per story that has one. Only the FAIL set is
comparable across trees, and it is empty.

## Readiness — the three questions the pin answered

Each was open at `9ff259c`, each is answered at `acb8fb0`, each at **High**, and all three confirm
the stance the contract already carried. Folded into `provenance.md`.

| | Claim | Grade | What it settles |
|---|---|---|---|
| R-2 | `ALM-OBJ-061` | High | The **first** of a placement's two fixed-point coordinates is the column. Nine named instructions from the loader's two `LEA`s to the packed cell key, with no permutation on the chain. |
| R-3 | `ALM-OBJ-062` | High | The extension's **first four** bytes carry the width and its **second four** the height, each narrowed to a low byte; the other four reach no instruction. |
| R-4 | `ALM-CLS-063` | High | The shop-class placements **do** attach — through their base-class constructor, which is why a call-site count could never have shown it — and always take the resolver's table arm. |

`TERR-STRUCT-090` arrived with them and is the one that **moved the contract**: the override arm is
selected by the **sum of the two extension bytes**, not by the kind, so a `0x21` record carrying two
zero bytes takes the table arm. FR-2 and AC-4 are narrowed to it. No shipped record exercises it —
all 8 carry non-zero bytes in both — which is why only the branch could have said so.

`ALM-OBJ-034`'s row is SUPERSEDED on one clause and not the other, and the clause FR-2 cites — the
*discriminator*, the whole key against `0x21` — is **not** the one that moved.

## Acceptance criteria

Every unit criterion runs headlessly, with no game install present (**FR-8**).

```text
go test -trimpath -count=1 ./pkg/mapload ./pkg/render/terrain ./pkg/ui ./cmd/mapview ./cmd/classdump
ok  againrom/pkg/mapload  ok  againrom/pkg/render/terrain  ok  againrom/pkg/ui
ok  againrom/cmd/mapview  ok  againrom/cmd/classdump
```

| AC | Where | Outcome |
|---|---|---|
| AC-1 | `pkg/mapload/structures_test.go` `TestFootprintResolution` | ✅ 3 resolved, 2 unresolved, 1 short; the `0x103` width narrows to 3; `0x121` reaches entry 33 with no extension; the collection is byte-identical after the call |
| AC-2 | `TestFootprintWalkRowMajor` | ✅ five of six cells attach, row-major right and down; 2 closed, 3 opened; the unnamed sixth byte is untouched and uncounted |
| AC-3 | `TestFootprintAliasesPast32Cells` | ✅ on an 11x4, cell 32 attaches and closes exactly as bit 0 says; an unmasked index leaves it alone |
| AC-4 | `TestExtensionResolution`, `TestExtensionOpensItsWholeRectangle` | ✅ all five shapes, including the **both-bytes-zero** case falling through to the entry's own row and sets |
| AC-5 | `TestOverlapRefusesAndAbandons` | ✅ refusal, abandonment, order reversal, and a 3x2 clashing at the end of its top row — the case that pins the walk as row-major |
| AC-6 | `TestPassOverridesEveryArm` | ✅ over water, mountain, scenery and the ring: opened cells open, closed cells closed, every air bit the arms' own, no byte above bit 1 |
| AC-7 | `TestFootprintOutsideTheExtent` | ✅ 4 attached / 2 dropped on an overhang; the two ring cells a wrapping walk would have opened are untouched; an off-map anchor moves no byte |
| AC-8 | `pkg/mapload/fromalm_test.go` `TestStructurePassLeavesATablelessWorldWhereItWas` | ✅ see **SC-5** |
| AC-9 | below | ✅ 16 map files, both locales |
| AC-10 | `TestBlockingSetPolarity` | ✅ the named cell closes, the unnamed opens, on a pair the arms left open |
| AC-11 | `cmd/mapview/blocked_test.go` `TestBlockedCells` | ✅ row-major, ground-closed only; the air-only cell is not in the list |

P-1 `TestFootprintResolutionTotal` over six table shapes and 302 placements · P-2/P-3/P-4/P-7/P-8
`TestStructurePassProperties` over 5x5 set pairs x 6 extents x 7 anchors = 1050 derivations, each
checked for the counter partition, the air bits and the reserved bits · P-5 see SC-5 · P-6 asserted
in AC-1 and in every `TestFootprintResolutionTotal` case.

**SC-5** ✅ — the digest was taken by running the fixture against the **previous** revision of
`fromalm.go` and copied in by hand, so it says the new code agrees with the old one rather than with
itself:

```text
pinDigest = 0x7f65b8eca4e34a24
  no table / nil table / Buildings absent  -> 0x7f65b8eca4e34a24 (all three)
  the same table carrying Buildings        -> differs, so the pass does reach FromALMWith
```

## Mutation (SC-2)

Applied to the lines the task wrote, run against that task's tests, reverted.

```text
T1  entry guard's upper bound as <=                       killed
T1  extension test on the low byte, not the whole key     killed
T1  sum gate replaced by the kind                         killed
T1  extents taken as the whole parameter                  killed
T1  zero-extent test hoisted above the override           killed
T1  parameter slots 4 and 5 exchanged                     killed (by AC-2, not AC-10)
T2  bit index taken without the modulus                   killed
T2  abandon latch dropped                                 killed
T2  a dropped cell latching the abandon                   killed
T2  closing and opening arms exchanged                    killed (AC-10)
T2  opening arm written as a whole-byte zero              killed
T2  row and column loops exchanged                        SURVIVED, then killed
T3  FromALMWith building from the arms alone              killed
T3  Buildings omitted from the tool's table               killed
T3  census sequenced after the world build                killed
T3  FromALM passing a table instead of nil                EQUIVALENT
T4  tint's cells taken from bit 1                         killed
T4  pass appended last instead of prepended               killed
T4  colour made opaque                                    killed
T4  flag defaulted on                                     killed
T4  table resolved from a compiled-in path                killed
```

Two entries are not "killed" and neither is rounded up.

**T2's loop exchange survived the first pass.** Every overlap fixture was a single row, where
column-major and row-major visit the same cells in the same order. A 3x2 footprint clashing at the
end of its top row was added — row by row the clash is the third cell walked and three are
abandoned; column by column it is the fifth and only one is — and it kills the mutant. Recorded
because the gap was real for as long as it existed.

**T3's fourth mutant is not expressible.** `tasks.md` names "`FromALM` passing the caller's table
instead of nil", and `FromALM` has no table in scope. Its nearest expressible form — a nil `*Table`
for an empty `&Table{}` — is **equivalent by contract**, both being "no table", and it survived. The
intent is covered by T3's second mutant, which is killed.

**Two more of AC-10's limits, measured rather than assumed.** AC-6 also discriminates the polarity
exchange, from the other side: on arms-*closed* cells the swapped build opens what AC-6 says it
closes. AC-10's own contribution is doing it on arms-*open* cells, where the failure is a visible
inversion rather than a subtraction. And AC-10 is **blind** to the two parameter slots being
exchanged — that fixture passes a slot-swapped build — which AC-2 catches instead. `plan.md` R-1 is
written to point at the right criterion for each.

## AC-9 / SC-6 — the corpus

`classdump -databin <world.res> <map.alm>`, every `.alm` of both preserved installs. **Every map
censused without error**, sixteen files, two locales. No game bytes are committed; these are counts.

```text
EN, 10 maps                      placements  attached  closed  opened  dropped  refused  abandoned
  Beast.ALM      256x256               478      1911    1660     251        0        0          0
  Cross.ALM      256x256               292      1653    1296     357        0        1          3
  Forester.alm   256x256               207      1235     995     240        0        0          0
  Horror.alm     256x256               415      2227    1799     428        0        1          0
  Islands.alm    256x256               287      1582    1284     298        0        0          0
  Kids.alm        80x80                 16        92      70      22        0        0          0
  Kids2.ALM       80x80                 19       112      83      29        0        0          0
  LuMoir.alm     144x144               107       522     430      92        0        0          0
  Tomb.ALM       256x256               256      1413    1109     304        0        0          0
  Waters.alm     144x144               113       711     515     196        0        0          0
  TOTAL                               2190     11458    9241    2217        0        2          3

RU, 6 maps                       placements  attached  closed  opened  dropped  refused  abandoned
  FORESTER/ISLANDS/KIDS/LUMOIR/WATERS   730      4142    3294     848        0        0          0
  Horror.alm                              0         0       0       0        0        0          0
```

P-6 and P-7 hold on every row: `resolved + skips == placements` (no skip fired anywhere in either
corpus), `closed + opened == attached`, and `attached + dropped + refused + abandoned == 11463` named
cells on EN.

**Two findings worth their own line.** The RU corpus's five non-empty maps reproduce their EN
counterparts' figures **exactly**, placement for placement and cell for cell — the same maps under a
different locale, and an independent consistency check nobody designed. And RU `Horror.alm` is a
different, smaller file carrying **zero** placements of either kind; it is not a defect in the walk,
it is what that file contains.

### Setting the figures against the pin's own — and what could not be set

The published totals are **3141 placements / 17 057 cells over 38 maps**. Our corpus is 16 files
holding 11 distinct maps, so those totals are **not comparable** and no agreement is claimed from
them. That is reported rather than reconciled, per AC-9.

Two published figures **are** comparable, and both are discriminating rather than incidental.
`ALM-OBJ-061` states that re-running the pass with the anchor transposed "puts 178 footprint cells
off the map … raises attach refusals **2 → 17**". We ran exactly that, by transposing `Col`/`Row` in
`resolve`, rebuilding, and re-censusing the EN corpus:

```text
                       as built     anchor transposed
  attached               11458                 11414
  refused                    2                    11
  abandoned                  3                    38
  dropped                    0                     0
```

- **Refusals: 2 as built.** The pin's figure for the published reading is **2**, and for the
  transposed reading 17 over 38 maps. Ours is 2 over 10, rising to 11 — the same value, the same
  direction, the same order of magnitude for a subset. This is R-2 re-derived through our own build
  against our own corpus, and it is the one corpus check in this story that could have failed.
- **Dropped: 0 under both readings.** Also as published, and it is the reason the statistic
  `TERR-STRUCT-075` originally rested its axis clause on was withdrawn: 37 of the 38 shipped maps are
  square, so off-plane cells cannot tell the two readings apart. Our corpus is square throughout and
  reproduces that blindness exactly.

The source was reverted immediately; the transposed binary lives outside the repository and no
commit carries it.

## SC-7 — the owner can see it

```text
mapview -assets <install> -map <install>\LuMoir.alm -blocked
mapview: LuMoir 144x144 cells (20736), tile slots 52/128, blocked 9505, water speed 4 (...)
```

Run headlessly with `-check` over all ten EN maps: every one loads, opens its table and reports a
blocked count. `builds/0051-structure-footprints/README.md` carries the invocation, the map, the
exact cells, and the two things about the picture that are correct and look odd.

`LuMoir.alm` is the named map because it is unambiguous rather than impressive: **exactly one**
structure on it subtracts a block — a 6x2 deck at columns 111..116, rows 55..56, every cell water,
12 cells out of 20 736. `Horror.alm` carries the other bridge mechanism, a 5x4 at columns 111..115,
rows 80..83 from the extension arm; `Cross.ALM` has the most crossings at 95 subtracted cells.

Three before/after renders of the plane are in `review/0051-structure-footprints/`, outside both
repositories. They are **schematics** of the plane rather than screenshots, and say so: the plane is
what the story changed and it has no pixels of its own.

## Limitations, stated rather than papered over

- **No automated test drives `-blocked` end to end through an archive.** `blockedCells`, the viewer
  overlay, the pass order, the colour and the failure paths are each tested; the join from a parsed
  table to a washed cell is witnessed by the runs above against a lawful install, not by CI. Building
  a parseable eight-group `data.bin` fixture inside `cmd/mapview` was judged out of proportion, and
  `plan.md` DD-11 says so.
- **`-blocked` selecting flat is witnessed by the run, not by a test.** It is one boolean
  composition in `run()`; a unit test of it would be a test of `||`.
- **The census cannot test the contract.** Every figure above is computed from this contract, so
  agreement with itself means nothing. Its value is the two comparisons above, which could have
  disagreed.
- **`plan.md` DD-2 was corrected mid-story.** Its sketch had `Passability` defined as
  `PassabilityWith(m, nil)`; the dependency runs the other way, so `Passability`'s body is never
  edited and P-5's subject does not move. The plan was fixed rather than the divergence noted.
- **`tasks.md` T4's file list gained `cmd/mapview/flagset_test.go`**, the designated inventory of the
  tool's flag surface, which says in its own header that a new flag belongs in it.
