# 0165-script-cast — verification

## The result someone can point at

The script-gap census falls by **78 nodes per preserved root**, from 366 to 288 — the same figure on
both roots, so 732 to 576 over the pair. Measured by driving each of the 28 campaign maps at
`-trace -ticks 1` and counting the compiled script's own `UNSUPPORTED instant` lines, which is what
`pipeline/check-milestone.sh` counts. Before is `implementation/builds/current/missionrun.exe`, the
binary master carried when this story branched; after is this branch's own build.

| opcode | before, per root | after | fall |
|---|---|---|---|
| 21, cast at a cell | 35 | 0 | 35 |
| 24, cast at a unit | 20 | 0 | 20 |
| 29, set an effect's lifetime | 23 | 0 | 23 |
| every other opcode | 288 | 288 | 0 |
| **total** | **366** | **288** | **78** |

No other census line moved. Both roots produce identical counts, before and after.

`pipeline/milestone-baseline.txt` is not updated here: it lives outside this repository and is
regenerated at the landing, against `builds/current/` rebuilt from master.

### Mission 10 and mission 20, the two the lane brief asks for

```
go build -o /tmp/mr ./cmd/missionrun
for m in 10 20; do AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
```

| mission | master before | this branch |
|---|---|---|
| 10 | 17 | 17 |
| 20 | 11 | 11 |

**Both are unchanged, and that is the expected result.** Neither map authors a node of opcode 21, 24
or 29 — the three opcodes appear on maps 80, 90, 91, 101, 131, 140 and 151 only. The stories these
two missions' remaining gaps belong to are opcode 2, opcode 20 and group sub-commands 11 and 15.

The mission-10 drive `check-milestone.sh` runs is unchanged in every stage it asserts and in the two
lines it records:

```
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (41,43), Chebyshev 22, after 224 ticks
outcome lost at tick 224
census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
```

### What the maps that do cast now do

`missionrun -census` reports standing area effects and pending casts. Driven unattended, over the
lawful EN install:

```
$ /tmp/mr -mission 101 -census -ticks 60
  effect spell 19 at (21,21), 346 tick(s) left
  effect spell 19 at (60,25), 346 tick(s) left
  effect spell 19 at (40,36), 346 tick(s) left
  effect spell 19 at (33,41), 346 tick(s) left
  effect spell 19 at (46,41), 346 tick(s) left
  effect spell 19 at (40,48), 346 tick(s) left
  effect spell 19 at (21,59), 346 tick(s) left
  effect spell 19 at (60,59), 346 tick(s) left

$ /tmp/mr -mission 91 -census -ticks 60
  effect spell 19 at (38,33), 346 tick(s) left
  effect spell 19 at (41,46), 346 tick(s) left
$ /tmp/mr -mission 91 -census -ticks 400
  effect spell 19 at (38,33), 6 tick(s) left
  effect spell 19 at (41,46), 6 tick(s) left
```

Every cell named is a cell that map's own instant-21 node casts at, and spell 19 is Wall of Earth,
the spell 23 of the 35 shipped instant-21 nodes name. The lifetime is FR-4's arithmetic:
`(15 << 4) + (99 << 4)/10 = 398` ticks, less the 52 ticks elapsed between the pass on tick 7 and the
report on tick 60. Mission 131 shows one wall at (110,91) on the same terms. Mission 140's casting
triggers do not fire in an unattended drive.

**Nothing was watched on a screen.** No windowed build was launched and no synthetic input was sent
to any window. The evidence above is `missionrun`'s own text output over the preserved installs.

## Gates

Run from the worktree on a clean tree at the branch head.

```
go build ./...                        clean
go vet ./...                          clean
gofmt -l $(git ls-files '*.go')       prints nothing
go test -trimpath -count=1 ./...      ok, all packages, 0 failures
```

Every `scripts/check-*.sh` by glob:

| script | result |
|---|---|
| `check-doc-budget.sh` | ok — provenance 8237/20480, spec 10370/24576, plan 8789/24576, verification prose 12261/20480, plan ≤ 1.2 × spec |
| `check-hotfix-ledger.sh` | ok |
| `check-no-game-assets.sh` | clean (tree scan) |
| `check-sdd-audit-selftest.sh` | ok |
| `check-sdd-audit.sh` | ok — no trailers on this branch, no `Co-Authored-By`, every id witnessed |

`git log --format='%h %(trailers:key=Co-Authored-By)' 9171a1e..HEAD` prints no trailer on any commit.
This story wrote no `tasks.md`: one lane implemented its own slice, so every `FR` and `DD` is
accounted for below instead.

The note and warning counts from `check-sdd-audit.sh` are not comparable from a worktree, which has
no `builds/` tree. Only the FAIL set is claimed, and it is empty.

## Every id, and what witnesses it

`AC-n`, `P-n` and `SC-n` are witnessed by a named test or by a measurement above. `FR-n` and `DD-n`
are named against the code that carries them, there being no `tasks.md`.

| id | witnessed by |
|---|---|
| FR-1 | `pkg/sim/scriptcast.go` `castAtCell`; `TestInstantTwentyOneAppendsOneCastCarryingItsOwnBytes` |
| FR-2 | `pkg/sim/scriptcast.go` `castAtUnit`; `TestInstantTwentyFourNamesItsTargetAndAnUnresolvedReferenceCastsNothing` |
| FR-3 | `pkg/sim/scriptcast.go` `stepScriptCasts`, called from `stepWorld`; `TestAPendingCastResolvesOnTheNextTickAndLeavesAnAreaEffect` |
| FR-4 | `pkg/sim/scriptcast.go` `resolveScriptCast`, `landAreaCast`, `landPointCast`; `TestACellCastOfAPointRowLandsNothing`, `TestAUnitCastAppliesTheTwoArmsThisBuildHasAndNothingElse`, `TestASpellIdNamingNoRowLandsNothing`, `TestAUnitCastWhoseTargetIsGoneLandsNothing` |
| FR-5 | `pkg/sim/celleffect.go` `placeCellEffect`, `decayCellEffects`; `TestAnAreaEffectsLifetimeFallsByOneATickAndIsRemovedAtZero`, `TestASeventhEffectOnOneCellIsDropped` |
| FR-6 | `pkg/sim/celleffect.go` `setCellEffectTime`; `TestInstantTwentyNineWritesEveryMatchOnItsOwnCellAndNoOther`, `TestAnIdentifierAboveAByteMatchesNothing` |
| FR-7 | `pkg/data/spell.go` slots 8 and 11; `pkg/sim/spell.go` `SpellRule.Area`, `.AreaDuration`; `TestASpellTableReachesTheWorldFieldForField` (pkg/mapload) |
| FR-8 | `pkg/sim/castbinary.go`, `pkg/sim/binary.go` version 49; `TestAWorldHoldingCastsAndEffectsRoundTrips`, `TestTheCastingSectionsSixRefusals` |
| AC-1 | `TestInstantTwentyOneAppendsOneCastCarryingItsOwnBytes`, `TestAnAuthoredPowerIsCarriedAndOnlyZeroIsSubstituted` |
| AC-2 | `TestInstantTwentyFourNamesItsTargetAndAnUnresolvedReferenceCastsNothing` |
| AC-3 | `TestAPendingCastResolvesOnTheNextTickAndLeavesAnAreaEffect`, `TestAUnitCastOfAnAreaRowLandsAtTheTargetsOwnCell` |
| AC-4 | `TestACellCastOfAPointRowLandsNothing`, three sub-cases |
| AC-5 | `TestAUnitCastAppliesTheTwoArmsThisBuildHasAndNothingElse`, three sub-cases |
| AC-6 | `TestAnAreaEffectsLifetimeFallsByOneATickAndIsRemovedAtZero`, `TestAnEffectWrittenToZeroIsRemovedRatherThanWrapped` |
| AC-7 | `TestInstantTwentyNineWritesEveryMatchOnItsOwnCellAndNoOther`, `TestAnIdentifierAboveAByteMatchesNothing` |
| AC-8 | `TestTheCellKeyIsSixteenBitArithmetic` |
| AC-9 | `TestAWorldHoldingCastsAndEffectsRoundTrips`, `TestTwoWorldsDifferingOnlyInACastHashDifferently`, `TestAVersionFortyEightFormIsRefused` |
| AC-10 | `TestTheCastingSectionsSixRefusals`, `TestTheCastingSectionsCountsAreBoundedAgainstTheBuffer` |
| AC-11 | the census table above; `TestTheThreeOpcodesAreReportedAsSupported` inside `pkg/sim` |
| P-1 | `TestAnObservedStepAndAnUnobservedStepAreOneWorld`, `TestAScriptCastIsNotReportedAsACastEvent` |
| P-2 | `TestACastThatLandsNothingLeavesTheWorldWhereItWas` |
| P-3 | `TestNothingHereReadsAnOwner` |
| DD-1 | `pkg/sim/world.go` `casts []scriptCast`; `pkg/sim/scriptcast.go`'s own head comment |
| DD-2 | `pkg/sim/celleffect.go` `cellKey`, `keyCell`; `TestTheCellKeyIsSixteenBitArithmetic` |
| DD-3 | `pkg/sim/celleffect.go` `placeCellEffect`; `TestTheEffectSliceStaysSortedByKey` |
| DD-4 | `pkg/sim/scriptcast.go` `resolveScriptCast`; `TestAUnitCastOfAnAreaRowLandsAtTheTargetsOwnCell` |
| DD-5 | `pkg/sim/spell.go` `SpellRule.Area`'s own doc; `pkg/data/spell.go` |
| DD-6 | `pkg/sim/step.go`, the three calls at the head of `stepWorld` |
| DD-7 | `TestAScriptCastIsNotReportedAsACastEvent` |
| DD-8 | `pkg/sim/binary.go`, `decodeCasting` called before `decodeScript` |
| DD-9 | the tree sweep below |
| DD-10 | the revert table below |
| SC-1 | `TestAUnitCastAppliesTheTwoArmsThisBuildHasAndNothingElse`'s third sub-case; and the limitation paragraph below |
| SC-2 | no code places, damages or draws from an area effect; `TestACastThatLandsNothingLeavesTheWorldWhereItWas` shows an entity set unchanged across a resolve |
| SC-3 | `TestAPendingCastResolvesOnTheNextTickAndLeavesAnAreaEffect` measures exactly one tick of latency |
| SC-4 | `TestAUnitCastWhoseTargetIsGoneLandsNothing` — the record is removed, not retried |
| SC-5 | `TestASeventhEffectOnOneCellIsDropped` — the cap is witnessed; no stacking rule is asserted |
| SC-6 | the gate table above |
| SC-7 | the census table above |
| SC-8 | the revert table below |
| SC-9 | the mission 10 and 20 table above |

### DD-9, checked rather than assumed

```
$ grep -rhoE "func Test[A-Za-z0-9_]+" --include=*_test.go . | grep -E "Version[0-9]|[0-9]{2}(Form|Bytes|Digest|Pin)" | sort -u
func TestAVersion20FormIsRefused
func TestAVersion22FormIsRefused
func TestAVersion26FormNamingAPlayerIsRefused
func TestAVersion37FormIsRefused
```

All four name an **old** version they refuse, where the number is permanent. No test name spells the
live version, so nothing was renamed. The story that this rule exists for — a name carrying the
current number and going stale — does not apply to this tree.

## SC-8: every gate reverted, and what failed

Each row reverts one comparison in the source, runs the named test, and restores the file. A row is
`witnessed` when the test went red.

| gate | file | test | result |
|---|---|---|---|
| a cast naming spell id 0 | `castbinary.go` | `TestTheCastingSectionsSixRefusals` | witnessed |
| a cast with a power of 0 | `castbinary.go` | same | witnessed |
| the target flag byte | `castbinary.go` | same | witnessed |
| an effect naming spell id 0 | `castbinary.go` | same | witnessed |
| ascending cell-key order | `castbinary.go` | same | witnessed |
| the six-slot cap | `castbinary.go` | same | witnessed |
| the pending-cast count bound | `castbinary.go` | `TestTheCastingSectionsCountsAreBoundedAgainstTheBuffer` | witnessed |
| the area-effect count bound | `castbinary.go` | same | witnessed |
| the version-49 gate | `binary.go` | `TestAVersionFortyEightFormIsRefused` | witnessed |
| a spell both damaging and restorative | `binary.go` | `TestUnmarshalRefusesTheSpellTableShapesNoTableMay` | witnessed |
| an undefined spell flag bit | `binary.go` | same | witnessed |
| the zero-lifetime removal guard | `celleffect.go` | `TestAnEffectWrittenToZeroIsRemovedRatherThanWrapped` | witnessed |
| the 16-bit ADD cell key, reverted to an OR | `celleffect.go` | `TestTheCellKeyIsSixteenBitArithmetic` | witnessed |
| the full-dword id comparison, reverted to `(u8)p2` | `celleffect.go` | `TestAnIdentifierAboveAByteMatchesNothing` | witnessed |
| the power-99 substitution | `scriptcast.go` | `TestInstantTwentyOneAppendsOneCastCarryingItsOwnBytes` | witnessed |

The last two revert to the **superseded** readings `TRIG-EFFECTTIME-034` carried before
`TRIG-CELLEFFECT-045` narrowed it, so this build's disagreement with the withdrawn row is asserted
rather than merely stated.

## Byte-form movement, checked against pre-story literals

Version 48 → **49**. The section adds eight bytes to a world holding no cast and no effect, and the
spell record grows four bytes for the area-duration column.

The peel tests are what say nothing else moved. `strippedOfTheCasting` was added as the newest peel
in `pkg/sim/binary_test.go` and in `pkg/mapload/fromalm_test.go`, and every existing pre-story digest
in both packages — twenty-odd literals reaching back to version 4 — is reached unchanged from the
new form once this story's section and version byte are lifted back off.
`TestThePinIsThePreStoryPinPlusTheCastingSection` is the new one, against version 48's own
`pinDigest` `0x459b1d6824831221` and `rtfDigest` `0x4765e86973904fdc`.

Live digests re-pinned, each because byte 0 moved and FNV-1a is a left fold: `pinDigest`
`0x6ab15b9bda10306c`, `rtfDigest` `0x27295dd98025ede1`, `gfDigest` `0x552da6feb991a5a2`, and four
fixture digests in `release_test.go`, `commanded_test.go` and `relaxation_test.go`.

## A repair this story made on the way

`SpellRule.Restorative` reached no bit of the byte form's spell flags byte. It was added by 0154 and
never encoded, so a world holding a heal row decoded with the flag clear and its heal arm gone. This
story's own point-cast arm forks on that flag, so the hole is load-bearing here; bit 2 of the flags
byte now carries it, at the version that was already moving the record. A form setting both bit 1 and
bit 2 is refused, the two being exact complements.

## The story's chief limitation, stated plainly

**All 20 shipped instant-24 nodes construct a cast, resolve it at their target, and land no state.**
The six spells they name — Stone Curse, Bless and the four Protections — are point effects, and
none is damaging or restorative, which are the only two arms this build has. `MAGIC-CEIL-013`
enumerates thirteen distinct expressions the spell power feeds, one or more per spell, and building
them is not this story (spec SC-1).

So the census counter falls by 20 for opcode 24 over nodes whose visible outcome is unchanged. The
arm is implemented and its terminal effect is not. Instant 21's 35 nodes and instant 29's 23 do land
state: the walls above are them.

An area effect also does nothing to a unit standing in it (spec SC-2), and the cast-time countdown is
one tick rather than the spell's own (spec SC-3), because no claim in the pin `aae16ee` names the
column that time comes from.
