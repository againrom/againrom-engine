# 0069 — verification

Environment: Windows 11, Go toolchain pinned in `go.mod`. Tests run `-trimpath` (Windows Defender
quarantines an untrimmed test binary here). Install evidence is from two lawful installs preserved
outside either repo; no asset entered the tree.

## Gates

Run from the story worktree, judged by **exit code**:

```
go build ./...                     EXIT=0
go vet ./...                       EXIT=0
gofmt -l $(git ls-files '*.go')    EXIT=0, printed nothing
go test -count=1 -trimpath ./...   EXIT=0
scripts/check-no-game-assets.sh    EXIT=0
scripts/check-doc-budget.sh        EXIT=0
scripts/check-sdd-audit.sh         EXIT=0
```

Note/warning counts are not comparable from a worktree — it has no `builds/` — so only the FAIL set
is reported, and it is empty.

## The contract, criterion by criterion

**AC-1, AC-2, SC-1 — the hero resolves, and it is the hero.** `TestTheBoundHeroIsThePartysFirstMember`
asserts the compiled check's reference is present, that it names `PartyEntity(m, 0)`, and that the
entity carrying that id stands on `Start.Cells[0]`. The cell is the half that discriminates: an
off-by-one naming the second party member carries a plausible id and stands somewhere else.

On the real corpus, mission 10 through `game.StartMissionFrom` — 5 of 5 distance checks resolve
with a party, 1 of 5 with none (that one names a placed unit and never depended on a hero):

```
=== EN mission 10                                    === RU mission 10 (byte-identical map)
party=0  distanceChecksBound=1/5  raises=11          party=0  distanceChecksBound=1/5  raises=11
party=1  distanceChecksBound=5/5  raises=11          party=1  distanceChecksBound=5/5  raises=11
party=3  distanceChecksBound=5/5  raises=11          party=3  distanceChecksBound=5/5  raises=11
```

**AC-3, SC-2 — the win discriminates. This is the story's result.**
`TestAMissionIsWonOnlyWhenTheHeroArrives` drives a map whose only winning trigger asks whether the
hero is within three cells of a point:

| party | hero dropped | reference resolved | outcome |
|---|---|---|---|
| one member | far from the point | yes | **undecided** |
| one member | on the point | yes | won |
| none | far from the point | no | won |
| none | on the point | no | won |

Rows 3 and 4 are the control and they are what every mission did before this story: nothing is
bound, the check writes no register, register 0 keeps its initial zero, and `0 <= 3` holds.

**The assertion discriminates against the pre-change code, and this was executed rather than
assumed.** Reverting `pkg/game/mission.go` alone and re-running the table:

```
--- FAIL: .../hero_bound,_standing_far_from_the_objective
    mission_test.go:444: the distance check's reference resolved = false, want true
```

and with that first assertion removed so the outcome itself is reached:

```
--- FAIL: .../hero_bound,_standing_far_from_the_objective
    mission_test.go:446: outcome = 1, want won = false — the distance is measured and exceeds the radius
```

Both files were restored before the commit; the working tree at the T2 commit is the fixed one.

The same table over **mission 10's own compiled script**, driven on a world whose entity positions
are chosen so both clauses can be exercised (the hero 60 cells from the objective in the `false`
rows):

```
heroBound=false heroAtObjective=false -> outcome=1 at tick 32     <- the defect
heroBound=false heroAtObjective=true  -> outcome=1 at tick 32
heroBound=true  heroAtObjective=false -> outcome=0 at tick -1     <- fixed
heroBound=true  heroAtObjective=true  -> outcome=1 at tick 32
```

Identical before and after 0067 merged in.

**AC-4, SC-1 — no party, no binding.** Rows 3 and 4 above assert the reference is unresolved.

**AC-5 — one ordinal only.** `TestAnOrdinalAboveOneStaysUnresolved` runs 10002, 10006 and 11000
through a mission started with a party; none resolves.

**AC-6, SC-3 — one rule, not two.** `TestPartyEntityIsTheIdTheStartAssigns` asserts `PartyEntity` equals
the id the **started world** gave that member, over placement counts 0/1/5/35 and party sizes
0/1/3/6, by cell as well as by id. On the installs, checked over every map of both — the built world
holds one entity per placement, member 0 receives the placement count, and it stands on the start's
first cell:

```
en: checked 38 maps, 0 mismatches
ru: checked 34 maps, 0 mismatches
```

**AC-7 — a script that will not decode is still not fatal.** `TestStartMissionCarriesTheCompilesAnswer`
and `TestStartMissionWithAnUndecodableScriptRunsNone` are unchanged and pass.

**AC-8, P-2, SC-4 — nothing without a party moved.** `TestAMissionWithNoPartyKeepsItsDigest` asserts a
mission started with no party hashes exactly what the explicit no-hero compile builds.

## The two digests

Mission 10, party of three, `nil` table, `DifficultyNormal`. **An absolute digest is a property of
the whole tree, not of this story**, so the pair is given for both trees this story was measured on
— this branch's base, and the same branch after 0067 merged into it:

```
                                    at 9fbda2d          merged with 0067
compiled with no hero               0x1cfea58bd60be74d  0xbe26ab43b0cca725
compiled with the hero bound        0xb46bf8105de39d2d  0x994f4cc4d3f40fc5
```

0067 gives a placed unit its class's combat numbers, which are entity fields and therefore in the
digest; it moved both rows and moved neither conclusion. **What this story owns is the delta** — the
two rows differ, and they differ for the same reason on either tree.

The lower-right value is what `game.StartMissionFrom` produces for that mission and party: the
mission path and the explicit bound compile agree, which is the point. Identical on both installs,
the maps being byte-identical (`md5 2d983ccbf249c5336ebb7ccc41fc405c`). On the merged tree a party
of one gives `0xe0be4e642f0e545c` and a party of none `0x9814835669edc4cc`.

Every other measurement in this file was re-run on the merged tree and is unchanged: the four-row
table, both id-rule sweeps, the bound-check counts, and the whole suite.

**P-1, FR-9, SC-4 — the form did not move.** `formatVersion` is untouched; `ScriptCheck.Unit` and
`HasUnit` are already encoded at fixed widths and only their values changed. No digest pinned as a
literal anywhere in the tree moved — the tree pins none, every comparison being world-against-world.

**P-3 — idempotence.** The binding introduces no dependence on iteration order or on when the
compile ran: the id is arithmetic over the map's placement count, and the unit table is a lookup
that is never iterated. Measured rather than argued — the probe above run twice in separate
processes prints the same three digests, and the two tests carrying the binding pass under
`go test -count=3`. The EN and RU columns are a third instance of the same equality.

## Corpus effect

Over both installs, hero-band references number 196 and static-name-band references 0. Binding
ordinal 1 resolves 135 of them; **61 remain unresolved**, every one naming an ordinal between 2 and
6 and every one in a campaign map. That residue is the contract's declared fence, not a regression.

## Limitations

Mission 10 is not driven to its win from a real install here — that needs orders the party has no
driver for yet, and the outcome evidence above is over a world whose positions were placed
directly. What is established is that the winning trigger now measures the hero and no longer holds
on an unwritten register.

What ordinals 2..6 name is not decided by this story and is not measured here.
