# Equipment checkpoint verification

Tested code: `0578d8d47cc0745a8900f9206148fe6cecb389c3`, merging published
implementation `e20d054e` into source-equipment checkpoint `22f8ea2e`.
Research pin: `659c3f69d4cb579006d4b584e0246cf309a40fc7`.

`go test -trimpath -count=1 ./...` passes. The asset guard prints
`check-no-game-assets: clean (tree scan)`. Focused source action tests use
literal expected values, including noncommuting Effects and late atomic refusal.
Natural original-save/App/native/death/re-equip tests pass both EN and RU.
These are implementation witnesses, not original-runtime observations.

The player-facing result exercised outside the new synthetic tests is the
existing natural-save App drive: the valued saved staff keeps its own runtime
blocks, price981 and owned Spell through both original LOAD doors and ordinary
native SAVE/fresh LOAD, then source death/removal clears the owned Spell and
re-equip constructs it again. EN/RU agree on native hashes
`795813289d2c5782` before re-equip and `8e9fac852d493841` afterward. The earlier
retention-only checkpoint did not execute source equipment arithmetic.

The worktree-built mission runner used lawful EN assets and one tick with
trace: mission10 UNSUPPORTED=0; mission20 UNSUPPORTED=0. Script populations are
unchanged against `pipeline/milestone-baseline.txt`: M10 16 checks/27 instants/
12 triggers, M20 14/15/11. That baseline does not separately store an historical
UNSUPPORTED count. This story does not claim to lower script gaps.
The build used `-buildvcs=false` after protected Git configuration prevented Go
VCS stamping; the exact tested commit is recorded above. No GUI was driven.

Outputs are outside Git in `review/story1110/equipment-reconciled-go.log`,
`equipment-reconciled-natural-en.log`, `equipment-reconciled-natural-ru.log`
and `mission-{10,20}-equipment.log`. No install byte or owner save was written.

This is a checkpoint, not final story acceptance. The parent coordinates final
dependency composition, the paired release/headless gate and existing review
budget. Unclosed casting/constructor-tail/source-identity boundaries are named
in `handoff.md`; green tests are not used to close them.

## Correction pass

`pipeline/reviews/story1110-pass1.md` RETURNed candidate `4ec7b424` for one
player-visible defect. This section records the correction; the checkpoint
text above is unchanged history.

**Returning defect.** A character carrying two items sharing Class and
DefinitionRow but differing retained runtime operands (DIV-762's own case,
e.g. the same weapon row in two materials) could not sell at the city shop.
`pkg/game/originalcity_sales.go`'s `inventoryInstance` patched equip-time
columns from the first `MemberCarriedItems` entry matching Class+DefinitionRow
only, which is not a unique key; the second colliding item received the
first's operands, so the `citySaleRemainders` fail-closed `DeepEqual` guard
against the baseline correctly refused the mismatched blend and every sale in
the pack was declined.

**Fix.** `inventoryInstance` now takes the object's own position in
`b.inventory` and patches from the baseline entry at that position's running
unit offset (`sum of Stack over all prior positions`), since `expanded` is
already built positionally against `MemberCarriedItems(b.baseline)` and one
stack's units always share one set of patched columns. All 6 call sites
(`citySaleRemainders`'s expand loop and its group-collision scan,
the sale-position lookup, `legacyCitySoldMember`, `replayCityMember`,
`prepareOriginalCitySale`'s group scan) pass the position instead of a
`sav.Piece`. `EffectsUnsupported` continues to come from `p` (the city
document), never blended from the baseline — unchanged.

**Regression proof.** `pkg/game/originalcity_sales_collision_test.go` adds
`TestCitySaleRemaindersAdmitsDistinctStacksSharingClassAndRow` (same row,
different own-kind; same row, one bound weapon spell — both must sell with
own-kind and Spell attributed to the correct stack and never grouped into one
sale target) and `TestCitySaleRemaindersStillRefusesAGenuineMismatch` (a
genuine baseline Code/container disagreement, and a city document's own
`UnsupportedEffectStates` — both must still be refused, proving the guard is
narrowed, not weakened). All 4 subtests pass against the fix. Verified
independently against the returned commit itself: a detached worktree at
`4ec7b424` running the pre-fix `inventoryInstance(sav.Piece)` against the same
two collision cases reproduces the review's exact failure — error
`original city pack is not its exact source container` for both cases, and
where the guard is bypassed to inspect the blend directly, own-kind blends to
`(4,4)` instead of the correct `(4,6)`. The fix changes only which baseline
entry is read; it does not touch `sav.CityInventoryItem` decode, the
`DeepEqual` guard itself, or `EffectsUnsupported`'s source.

**Selector mapping disagreement (review observation 2).** Claim
`HERO-DMG2-029` was read directly (`cd research && go run ./tools/claim
HERO-DMG2-029`): its five-way jump table reads raw selector 1→+0xc4, 2→+0xca,
3→+0xc8, 4→+0xc6, 5→+0xcc. At the Protection block's own two-byte-per-slot
column spacing (`pkg/sim/sourceactor.go`'s decode, `pkg/data/unitdef.go`'s
`protectionNames`/`UNIT-COMBAT-015`), those offsets name Fire, Earth, Air,
Water, Astral — a permutation `{0,3,2,1,4}` against canonical column order
Fire/Water/Air/Earth/Astral (`pkg/sim/spell.go:19-22`), not a plain
decrement. `pkg/game/originalprofile.go:67` already used this permutation;
`pkg/data/humanstate.go:315-319`'s plain `Selector--` was the outlier and is
the one the claim contradicts. Both paths now share one constant,
`pkg/data.ElementalSelectorOrder = [5]uint8{0, 3, 2, 1, 4}`
(`pkg/data/unitdef.go`); `humanstate.go`'s `Derived` uses it whenever the raw
kind is in the valid 1..5 range demanded by `HumanState.ProjectionError`, and
keeps the byte-identical prior decrement for an already out-of-range kind
that reaches `Derived` regardless, so no new out-of-range table read is
introduced. `pkg/data/elementalselector1110_test.go` pins all of raw 1..5 on
the `pkg/data` side (`TestHumanDerivedElementalSelectorMatchesResolverPermutation`)
plus the constant itself
(`TestElementalSelectorOrderPinsResolverPermutation`);
`pkg/game/originalprofile_test.go`'s new
`TestOriginalProfile1110ElementalSelectorMatchesResolverPermutation` pins the
same table through `applyOriginalActorProfiles`. All pass.

Dormancy was re-measured on this correction, not only trusted from the
review: every one of the 161 corpus actors across the 58 saves under
`gameversions/saves` carries elemental kind 0 with zero base and spread, so
no saved byte, world hash or player-visible value moves with this fix in the
current corpus. The change only stops silently mis-mapping a save that carries
a nonzero elemental base/spread with raw kind 2 or 4 (Water/Earth swapped),
which does not exist in the corpus today.

**Honesty repairs (review observation 1 comment, and observation 4).**
`pkg/sim/world.go`'s `reachFault` doc comment asserted unconditional refusal
of a decoded reach of 0; the function itself was already widened (before this
pass) to admit it when `e.ActorLoad.Source.EquipmentRuntimePresent`
(`docs/1110/handoff.md`: "EquipmentRuntimePresent discriminates saved
Reach/Charge/Relax bytes, including reach zero"). The comment now states the
exemption and its ground; the function body is unchanged. The uint8 reach
underflow in `sourceequipmove.go:225/284` that the review measured
unreachable in real data (137/137 worn weapons have saved reach equal to
`W50`, shipped weapon Range 1-20 across 27 rows) is untouched, per the
review's own classification as a queued hardening gap, not this pass's scope.

`pkg/game/originalprofile_test.go:206` (`TestOriginalProfile1107LateRefusalLeavesActiveSession`)
added `kind != "bad active"` as a blanket exemption from its stock/books/
profile assertion without re-pinning what that kind actually leaves in the
report. Both `"ambiguous source"` and `"bad active"` refuse inside
`restoreOriginalActorStock` before it ever assigns `*report` — measured
directly (temporary instrumentation, since removed): every one of
`Stocked`, `StockDead`, `StockOffMap`, `StockParty`, `StockUnbound`,
`StockUnmatched`, `Books.Restored` and `ProfilesRestored` is 0 for both kinds.
The exemption is replaced with a `switch` that pins that measured all-zero
report for both early-refusal kinds and keeps the prior `Stocked==2,
Books.Restored==2, ProfilesRestored==0` assertion for the remaining three
kinds (`"bad elemental"`, `"zero mana period"`, `"full mana zero period"`),
so the subtest again fails if a future change lets a partial stock/book/
profile report leak past either refusal gate.

**Gates, exact candidate.** All four worktree gates and all four seat gates
were re-run on the corrected tree and every number matches the
pre-correction baseline exactly: `gofmt -l .` clean; `go test -trimpath
-count=1 ./...` 49 ok, 25 no test files, 0 FAIL; `check-no-game-assets.sh`
clean; `check-claim-citations.sh` 1558 citations against 1890 claims (298
experiments, 1008 prefixes); `check-div-claims.sh` 336 of 336 live rows citing
489 distinct claim ids; `check-release-tests.sh` EN+RU one invocation, 159/159
and 159/159, 0 lacking a subject; `check-scenarios.sh` EN 50 of 50;
`check-preserved-installs.sh` 181 files both roots. None of the three items
change a gated population, so none of these numbers were expected to move,
and none did.

The worktree-built mission runner, lawful EN assets, one tick with trace:
mission10 UNSUPPORTED=0, mission20 UNSUPPORTED=0 — identical to the checkpoint
value above and to master's own recorded value
(`pipeline/SAV-COMPLETION.md`: "M10/M20 remain UNSUPPORTED0/0"). Unchanged;
none of the three correction items touch script-node execution.

No saved byte, world hash or pinned constant moves anywhere in this
correction: Item 1 corrects which baseline entry a comparison/replay value is
read from without changing the wire format, decode, or the guard's own logic;
Item 2 is dormant in the full corpus as shown above; Item 3 is comment and
test-assertion only. No GUI was driven for this pass.
