# Verification — a placed unit's combat numbers

Environment: Windows 11, Go 1.26.1 (`go.mod`'s pin), worktree `wt-0067-spec`, branch
`impl/0067-combat-numbers` off master `45716c4`. The install-facing runs read the two preserved
roots read-only and wrote every byte of output outside the repository; no game data entered the
tree, and no binary, output or run note is committed (golden rules 1 and 3).

## The gate — SC-9

Judged by EXIT CODE, not by grepping for `FAIL` — two of these scripts emit no `FAIL` line at all
and their verdict *is* the status.

```
go build ./...                       clean                       EXIT=0
go vet ./...                         clean                       EXIT=0
gofmt -l $(git ls-files '*.go')      printed nothing             EXIT=0
go test -count=1 -trimpath ./...     29 ok, 0 fail, 3 no tests   EXIT=0

bash scripts/check-no-game-assets.sh                             EXIT=0
bash scripts/check-doc-budget.sh                                 EXIT=0
bash scripts/check-sdd-audit.sh                                  EXIT=0
```

`-trimpath` is required locally: Windows Defender quarantines one test binary without it. Note and
warning **counts** are not recorded, because a worktree has no `builds/` and the audit's builds
check is guarded by `[ -d builds ]` — only the FAIL set is comparable, and it is empty.

Deletion sweep over the whole branch, `git diff --diff-filter=D --name-only 45716c4 HEAD`:
**empty**. 10 files changed, 1109 insertions, 52 deletions, none of them a file. **SC-11.**

Three trailered commits, one per task, each touching a file outside `docs/`:

```
f90f395  SDD-Task: 0067-combat-numbers/T1
e9cad08  SDD-Task: 0067-combat-numbers/T2
4d901af  SDD-Task: 0067-combat-numbers/T3
```

## The form did not move — AC-7, AC-8, SC-6, SC-7, P-4, P-5

The eight were already declared, already carried in the byte form at their own offsets and widths,
and already read by a tick; only the load-time write was missing. So the version byte stays at
**10** and no record grows. **AC-7** is a marshal/unmarshal round trip of a **table-built** world —
built at the hard setting, so its records carry the eight at values a table produced rather than at
the constructor's, which a round trip over an all-defaults world would not distinguish from a
decoder reading them as zero. It comes back equal field for field and equal by digest, and the
version byte is asserted against a literal written out from the form's documented layout rather than
read from the package that emits it. **P-4** is that same literal.

**P-5** has two halves and both hold. Two loads of one `(map, table, difficulty)` inside one process
hash identically, asserted beside the round trip. Across processes, the three pins below are
literals re-checked by every `go test` invocation — each a separate process — and the tool runs in
the next section are further separate processes that reproduce the same numbers from the same
inputs.

**Three** pinned digests move, not the two `plan.md` fact 6 recorded — `gridform_test.go` holds a
third. Every one of them was derived FORWARD, from the pre-story bytes, **before** the loader was
changed: take the pre-story form, write 8 and 4 into each record's two cadence slots, hash the
result. Each derivation was then confirmed by running the changed loader and finding the same
number.

| fixture | pre-story | now | length |
|---|---|---|---|
| `fixtureMap`, 5 records | `0x2f029f5557deeb6a` | `0xb2373204d2c5e176` | 6786, unmoved |
| `pinMap`, 2 records | `0xf0db1255eab165cc` | `0xc8456b5d673be21c` | 2205, unmoved |
| `gfMap`, 1 record | `0x9761ea070c9b922c` | `0xb5f72b1253c19040` | 2118, unmoved |

The backward check is **AC-8**, it runs on **every** test run, and it is the whole of R-1's
mitigation: zeroing the
eight fields' bytes — the contiguous span `+54…+82` closing each record — in every record of each
new form reproduces that fixture's pre-story digest **exactly**, against a literal this story's code
did not produce. `gfMap`'s runs through the file's own second FNV-1a implementation, which owes
nothing to `hash/fnv`. A pin pasted out of a run cannot fail; these can.

The pre-story spans were also confirmed all-zero before being written to, which is what makes the
zeroing recipe exact rather than approximate.

## The witnesses — SC-1 to SC-5, SC-8

All in `pkg/mapload`, all synthetic, all green with no install present.

- **SC-1 / AC-1.** One row whose nine source columns hold eight values, no two equal and none equal
  to a constructor default, asserted **one field at a time** — so a field crossed with another and a
  field left unwritten are two different failures. Health, rate and domain asserted unmoved beside
  them, at values of their own rather than at zero.
- **SC-2 / AC-2.** Four rows: the empty cell, the first pair, the always-hits arm, and a value no
  arm names. The pair is `(min, max−min)` on all four and the mark is set on exactly one — the test
  counts its own expectations, so a table that stopped expecting the mark anywhere fails.
- **SC-3 / AC-3, AC-4.** Two **built worlds** compared, never arithmetic on a definition. Hard moves
  to-hit and defence by exactly the constant; the other six are asserted one at a time. Easy moves
  none of the eight.
- **SC-4 / AC-5.** The three unresolved shapes — no table, a humans-arm placement, a key naming
  nothing — each carry 8, 4 and six zeros, and are asserted equal **to each other**, which is the
  half a constant cannot catch.
- **SC-5 / AC-6.** A party of four compared against a placement **in the same world**, with a guard
  that fails if that placement carries eight zeros — otherwise the comparison would hold for a party
  carrying nothing either.
- **SC-8 / AC-9, P-3.** Both refused arms: a nil world, no partial entity, and a refusal naming the
  entry.
- **P-2.** A world advanced 25 ticks, one of them carrying a walking order, still holds the eight
  the load wrote.
- **P-1** is the completeness property, and it is carried **structurally** rather than by a test
  that could only sample: the eight are written in the same composite literal as the health, the
  rate and the domain, off a single resolution, with the unresolved arm substituting the whole
  constructor definition once before anything is read off it. There is no mutator, so there is no
  call site to forget, and a field left unwritten is a field a reader can see is absent. AC-1 and
  AC-5 witness the two populations; the install run below adds 1815 real placements, every one of
  which carries a complete set. No code path in `pkg/mapload` produces a mixture, because no code
  path writes any of the eight anywhere else.

**Mutation kills.** Three run against the committed suite, each reverted after:

| mutation | caught by |
|---|---|
| to-hit and defence swapped in the entity literal | AC-1's per-field table |
| the `AttackRelax` write dropped | 8 tests, including all three digest pins |
| the always-hits mark forced false | AC-1 and AC-2 |

Each single-field omission is caught **by construction** as well as by these three: every one of the
eight source values differs from the constructor default that would stand in its place, so dropping
any one write moves that field to a value AC-1 does not expect.

## The dump the owner reads — SC-10, AC-10

`classdump -databin <world.res> <map.alm> <difficulty>`, run against **both** preserved roots. The
report is appended to the verb; its existing rows, totals and exit codes are unchanged, asserted by
a test that also requires the new section to come after them.

EN, `Kids.alm`, difficulty 2 — the heading is in the output, not only in this repository:

```
Kids.alm: template combat numbers, 51 placement(s) at difficulty 2
  THESE ARE THE TEMPLATE'S NUMBERS AND EQUIPMENT IS NOT APPLIED.
  A class the game arms fights at its weapon's cadence and at a damage, absorption,
  defence and reach its equipment moves; none of that is read here, so these rows are
  NOT what such a class fights at. The charge and relax columns are on no panel the
  game displays and are compared against nothing.
  index           key  charge  relax  toHit  defence  absorb  dmgBase  dmgSpread  always  entry
      0 0x0042/0x0001      12      6     80       70       5       20         20   false  "Ogre"
      2 0x0040/0x0001       7      3     15       30       0        4          4   false  "Goblin_Pike"
      4 0x0041/0x0001      21      7     55       45       2        6          6   false  "Orc_Bow"
```

51 placements, 14 distinct entries, every one resolved. **EN and RU produce byte-identical rows for
the same map** — a claim made on one root is not made.

**FR-3 on shipped data, over all 51 placements at once.** Differencing the three runs column by
column: easy against normal is **empty** — all eight identical on every row — and hard against
normal is `(+50, +50)` on to-hit and defence with **0** other columns moving, on all 51 of 51 rows.
That is the requirement measured against an install rather than against a fixture.

**FR-4 on shipped data.** `Horror.alm`, 1815 placements, of which **421** resolve to no unit
definition. Every one of the 421 carries exactly `8 4 0 0 0 0 0 false` — one distinct row, 421 times
— and each is printed as `none - the server-id arm reached no unit definition` rather than as a row
of zeros a reader would have to interpret. The constructor's 8 and 4 are a real class's numbers too,
which is why the words are load-bearing.

Every run exited 0. A refused difficulty still refuses, printing no report at all.

## What is NOT witnessed, and by what

**AC-10's comparison itself is the owner's and has not been made.** This run produces the numbers;
the row is closed only when the owner sets six of the eight against the game's own unit information
panel for a unit they select. Two of the eight — the attack charge and the attack relax — are **on
no panel the game displays** and are therefore witnessed by no run of any kind, this one included.
The report says so in its own output.

**Mission 10 was not the map used.** `10.alm` lives inside `scenario.res` and this verb reads a map
from the filesystem, so the runs above use loose maps. The property AC-10 tests is not map-specific,
but the substitution is named rather than glossed.

**Equipment is not modelled and this story does not approach it.** It assigns over the cadence pair
and adds to the damage pair, the absorption and the defence, and moves the reach, on a substantial
minority of shipped classes. Any row above compared against such a class will disagree, and should.
The named seam moves all five together; it is not this story.

**R-2 stands, carried forward.** The blow resolution treats a damage that is not positive after
absorption as removing nothing, and whether the original clamps there was not read. Until now every
loaded map's absorption was zero and the branch was unreachable outside a hand-built world; filling
absorption from a class makes it reachable. Nothing about the resolution changed here and no test in
this story asserts anything about it. It is a standing research request.

**FR-5** is met by adding nothing: no per-entity reach exists and no reach is read from a
definition. The class tables carry no reach column at all, so a reach filled from one would be a
single constructor value wearing a decode's clothes.

## Two defects in the planning artifacts, found by implementing them

Neither changed what the story does; both are recorded because a file list that is wrong in one
place is worth checking in others.

1. **`plan.md` fact 6 says two world digests are pinned by hand in `pkg/mapload`'s tests. There are
   three** — `gridform_test.go`'s `gfDigest` is the third, and it moves for exactly the same reason
   the other two do. Found by the test failing, not by reading the plan.
2. **`tasks.md` T3 names `cmd/classdump/campaign.go` and `campaign_test.go`.** The verb T3 describes
   — "the existing table verb, which already takes an archive, an optional map and a difficulty" —
   is `-databin`, and it lives in `databin.go` / `databin_test.go`. `campaign.go` is a different
   verb that takes an asset root and no difficulty. The task was implemented where its own prose
   points; `main.go` was modified as listed.

A third, smaller one: `cmd/classdump`'s row in the import DAG does not include `pkg/sim`, so the
report is assembled out of a row type of this tool rather than the simulation's entity. No entry was
added to the allow-map.
