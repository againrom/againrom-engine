# Verification — 0105, the equipment channel

Branch `story-0105-equipment-channel`, four trailered task commits and this one. Research pin
`53f8bb7`, unmoved: master's gitlink was already research master, so no pin commit was owed.

**Method.** A clause is called witnessed only where a mutation of the line that implements it turned
a test red. Where a mutation left the suite green the survivor is named below with the reason, and
two of them were survivors this stage repaired rather than excused. The install measurements were
taken with a throwaway tool built **outside** the repo against a `replace` of this worktree; it
reads `-assets` and writes nothing, and no shipped string was copied out of it (P-5).

## The gate

At `be1d4aa` plus this commit, clean tree, real exit codes captured directly and not through a pipe.

```
go build ./...                                   0
go vet ./...                                     0
gofmt -l $(git ls-files '*.go')                  0   (no output)
go test -trimpath -count=1 ./...                 0   (31 packages ok)
bash scripts/check-no-game-assets.sh             0   clean (tree scan)
bash scripts/check-doc-budget.sh                 0
bash scripts/check-sdd-audit.sh                  0   FAIL set empty
```

`git diff --diff-filter=D --name-only origin/master..HEAD` is **empty**. `PartyBody` was deleted as
a function; no file went with it.

`git diff --stat origin/master..HEAD -- pkg/render pkg/sim` is **empty output** — SC-5 and P-2 hold
structurally rather than by assertion.

`git log --format='%h %(trailers:key=Co-Authored-By)' origin/master..HEAD` prints an empty trailer
for every commit.

## The two installs

`main/text/heropicture.txt`, read through the container filesystem from
`gameversions/en` and `gameversions/ru`. **Every line below was identical on both roots.**

```
bytes         248
md5           984975778768dcaa81806f89712623df
CRLF lines    26        bare LF 0        trailing newline yes
file lines    26        (counted from the raw bytes, not by the parser under test)
list entries  26        equal to the file's line count: true
blank entry   index 22
entry lengths [7 9 9 9 9 11 7 7 7 7 6 8 7 7 7 7 7 6 8 6 6 7 0 10 13 9]
```

This is the byte count `HERO-APPEAR-052` itself states, and the blank at index 22 is the divergence
`provenance.md` discloses: the row publishes 25 non-blank lines, this build keeps 26 (DD-4). Only
the entry *lengths* are recorded here, never the names — a transcription of the payload is the thing
P-5 and SC-3 forbid.

## Acceptance

**AC-1 — twelve slots, numbered, empty.** `pkg/data/equip_test.go`. Mutation **M6**: `SetRow` widened
to admit a thirteenth slot. `TestASlotOutsideOneToTwelveIsRefusedByAllThree` went red.

**AC-2 — both installs, every line an entry.** The table above, on both roots. The entry count is
compared against a line count computed from the raw bytes by the measurement tool, so it is not the
parser agreeing with itself.

**AC-3 — an occupied row answers the entry one before it.** Mutation **M2**: `HeroBodyFor`'s
`i := int(row) - 1` changed to `i := int(row)`. Three `pkg/data` tests and four `pkg/game` tests went
red, including `TestAnOccupiedFirstSlotDerivesTheEntryOneBeforeItsRow`.

**AC-4 — an empty first slot answers the list's first entry.** Same mutation M2;
`TestAnEmptyFirstSlotDerivesTheBareHandedName` went red. Measured on both installs: a nil weapon
yields `"unarmed"` and class key 1.

**AC-5 — a row naming no entry produces no name.** Same mutation M2;
`TestARowNamingNoEntryProducesNoName` went red on the before-the-start and past-the-end arms and on
the empty-entry arm.

**AC-6 — every trained skill this front end can generate for.** Five skill slots, both roots,
identical:

| slot | row | range | body | class key | law's key for that name | list entry matches |
|---|---|---|---|---|---|---|
| Blade | 3 | 1 | swordsman | 3 | 3 | yes |
| Axe | 18 | 1 | axeman | 7 | 7 | yes |
| Bludgen | 9 | 1 | clubman | 10 | 10 | yes |
| Pike | 15 | 1 | pikeman | 12 | 12 | yes |
| Shoot | 20 | 4 | archer | 14 | 14 | yes |
| — bare — | — | — | unarmed | 1 | 1 | l[0] |

The last column is the tool's own `list[Row-1]`, computed beside `HeroBodyFor` rather than through
it. Every one of the five matched an arm of the law, so none fell to `HeroUnmatchedClass`. Note that
the blade arm — the party this build actually ships — lands on `swordsman`, which is exactly the name
`PartyBody` used to return. **The retired constant was right, and that is a result rather than a
reason to have kept it**: it is now the shipped list's answer instead of ours, and four of the five
arms it never covered are answered too.

**AC-7 — the reach a member leaves the start with.** Mission 10 on both roots, a three-member party:
a bow (range 4), a sword (range 1) and a bare hand.

```
member 0  id 35  reach 4      member 1  id 36  reach 1      member 2  id 37  reach 1
```

Mutation **M1**: the mint's `Reach: reachOf(c.Reach)` line deleted from `pkg/mapload/start.go`.
Member 0's reach fell from **4 to 1** and the world's byte form changed
(`44f34e3d…` → `978cb50a…`). That single number is the whole of FR-7: 1 is what the constructor's
repair answers, 4 is what the weapon says.

**AC-8 — no entity leaves a start with a zero reach, and no member's reach is the repair.** 38
entities in the started world, **0** carrying reach 0. **The zero count alone does not discriminate**
and is recorded as such: under M1 it was still 0, because the repair fires and produces 1. The half
that discriminates is member 0's 4 above.

**AC-9 — the byte form does not move.** The form's version byte reads **23** on both roots, before
and after. `pkg/sim`'s pin tests were re-run unchanged and none moved; nothing in `pkg/sim` was
edited, which the empty diff-stat above states more strongly than an assertion could.

**AC-10 — no body-name literal on the party path.**
`grep -rn 'BodySwordsman|"swordsman"' pkg/game/ cmd/ --include=*.go` outside `_test.go` returns
**nothing**. See the M5 survivor below for how this went from asserted to witnessed.

## Requirements, properties, decisions

**FR-1, FR-2** — M6, above. A slot carries an `int32` row and nothing else; there is no second field
to carry a name or a statistic.

**FR-3** — two halves. Parse: mutation **M3**, `ParseBodyList` rewritten to drop empty lines;
`TestParseBodyListSplitsEveryLineIncludingAnEmptyOne` went red. Read: mutation **M8**, `ReadBodyList`
made to report absence unconditionally; `TestReadBodyListIsSilentWhenNothingShips` went red.

**FR-4** — M2, above, on all three arms.

**FR-5** — M5, below, and AC-10's grep.

**FR-6** — mutation **M4**: `ResolveWeapon`'s `Row: int32(row)` changed to `Row: 0`.
`TestResolveWeaponFillsRowFromTheIndexItAlreadyFound` and
`TestAWeaponWithNoShapeWordScalesThroughIndexZero` went red in `pkg/data`, and
`TestLoadDefinitionsResolvesTheStartingWeapon` in `pkg/game`. The ranged refusal was not touched:
the bow above resolves through `ResolveWeapon` exactly as it did before, and `pkg/mapload`'s
`firstWeapon` predicate is unchanged.

**FR-7** — three mutations. **M7a**: `c.Reach = w.Range` deleted from `Hero.Derive` —
`TestAnArmedPersonsReachIsHisWeaponsRange` and `TestDeriveSetsReachFromItsOwnWeaponArgument` went
red. **M7b**: `Derive`'s base `Reach: 1` changed to `0` — four `pkg/data` tests went red including
`TestABarePersonsReachIsOne`. **M1**: the mint line, above.

**FR-8** — the empty `pkg/render`/`pkg/sim` diff, the version byte reading 23 on both installs, and
`pkg/sim`'s own pin tests re-run. Byte-form version **24 was allocated to this story and is not
used**.

**P-1** — M5, below. **P-2** — the empty `pkg/render` diff. **P-3** — `HeroBodyFor` takes a list and
an `Equipment` and nothing else; there is no parameter for a skill, a statistic or a shape word to
arrive through. **P-4** — `pkg/data/humandef_test.go` is byte-identical to master and passes
unedited, which is what makes the reach a *move*: `HumanDef.Combat`'s first statement calls `Derive`,
so a placed person's number comes out of the same two lines it came out of before.
`pkg/data/humandef.go` now contains no `c.Reach` assignment at all. **P-5** — no shipped payload,
and no transcription of one, is in the diff; `check-no-game-assets.sh` is clean.

**DD-1** — eleven slots modelled and left empty; nothing enforces that only slot 1 is filled, and
`Equipment`'s doc says why. **DD-2** — the derivation runs once, in `partyBody`, where the party is
assembled. **DD-3** — no code; the dying substitution is still in `HeroBodyName` and still unreached.
**DD-4** — M3, and the blank at index 22 measured on both roots. **DD-5** — M2's third arm; the
refusal is written as authored at the function.

**SC-1** — the gate above. **SC-2** — no unit test reads an install; the two-root numbers are this
document's. **SC-3** — the only lists in test code are invented ones. **SC-4** —
`pkg/data/appearance.go` is not in the diff. **SC-5** — the empty diff-stat. **SC-6** — AC-10's grep.

## Two mutations that survived, and what was done about them

Both were repaired at this stage rather than excused, and both repairs are test-only.

**M5 — `partyBody` returning the retired literal left the whole suite green.** Replacing
`return data.HeroBodyFor(list, eq)` with `return data.BodySwordsman, true` passed every test in the
tree. The cause is that **every fixture list in `pkg/game` derived `"swordsman"`** — the exact name
`PartyBody` used to return — so the constant and the derivation were observationally identical.
FR-5, P-1 and AC-10 were asserted and not witnessed. Added
`TestMovingTheWeaponMovesTheDerivedBody` in `pkg/game/heroappear_test.go`: one invented list, four
rows, four different names. Re-run under M5 it goes red.

**M1 — deleting the mint's reach line left `go test ./...` green.** No synthetic test asserted a
started party member's reach; only the install measurement caught it, and an install measurement is
not a gate. Added `TestAStartedPartyMembersReachIsHisWeaponsRange` in `pkg/mapload/hero_test.go`,
with a range-4 member because a range-1 member reads 1 under the repair as well. Re-run under M1 it
goes red.

One mutation is recorded as **equivalent rather than unwitnessed**: within that new test the bare
member's 1 cannot be told from the repair's 1 at the loader tier, by construction. The arm that pins
it is `pkg/data`'s `TestABarePersonsReachIsOne`, and the test says so at the fixture rather than
implying a discrimination it does not make.

## What was changed in the inherited artifacts, and one stale comment

Nothing in `spec.md`, `plan.md` or `tasks.md` was found false, and none was edited.

Two departures from `tasks.md`'s **file assignment**, both disclosed rather than quiet:

1. `pkg/game/frontend.go` and `pkg/game/table.go` were done in **T3**, not T4. Deleting `PartyBody`
   and changing `MissionParty`'s signature breaks both, and `frontend.go` cannot pass a list that
   `LoadDefinitions` does not yet load — so the three files are one atomic change and splitting them
   would have made T3 a commit that could not compile its own package. Every file `tasks.md` names is
   still touched by exactly one task.
2. `cmd/againrom/main_test.go` was reworked in T4 and is not in any entry's file list. It called the
   deleted `PartyBody` in three places; it now derives through `MissionParty` over a synthetic list
   written into a synthetic install.

**T3's commit does not build `cmd/`**, and that is the plan's own design — it calls T4 "the two call
sites T3 breaks". It is recorded here because a commit that does not build is a bisect hazard and
nothing else in the story says so.

One code comment was corrected at this stage, in `pkg/mapload/start.go`: `PartyMember`'s doc said a
member "keeps SpawnHP and the simulation's one-cell reach", naming both as divergences owed to a
later story. This **is** that later story for the reach, so the sentence had become false about the
tree three files away from where it was written. It was true when written; it is the failure mode of
reading a comment as documentation rather than as a premise.

## Still open, and named rather than left silent

- **DD-4 is a disclosed divergence and not a resolved question.** Whether the original's own list
  reader keeps the blank line is not established by any row at this pin. Nothing this build ships can
  tell the two readings apart: every weapon character generation can hand out resolves to a row below
  index 22, which the AC-6 table shows directly — the highest row reached is 20.
- **DD-5 is authored.** What the original does with a slot-1 index past the end of the list is
  `HERO-APPEAR-052`'s standing Unknown.
- **Slots 2 through 12 are modelled and empty**, so the shield suffix and the armour arm of the
  appearance law are still reached with the same "no second slot, no armour" arguments as before.
- **Nothing can pick an item up, drop one or swap one**, so a character's appearance is still derived
  once, at the start (DD-2).
