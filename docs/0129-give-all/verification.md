# 0129 — give all: verification

Branch `0129-give-all`, three task commits: T1 `12f94a2`, T2 `7023b0a`, T3 `b57ded1`.

## The acceptance criteria

| Id | Where | What it asserts |
|---|---|---|
| AC-1 | `pkg/sim/giveall_test.go`, `TestGiveAllMovesTheWholeContainerInOrder` | Driven through a real script pass, the receiver ends holding `[7 0xe06 0xe06 0xe06]` — its own code first, then the giver's three in the giver's order — and the giver holds nothing. Exact lists, both sides. |
| AC-2 | same file, `TestGiveAllLeavesEquipmentAlone` | Both twelve-slot equipment records are equal to what they were, and the giver still wears hers with an empty container. |
| AC-3 | same file, `TestGiveAllsFourRefusalsChangeNothing` | One sub-test per FR-6 case; the whole `MarshalBinary` output is equal across the call, and the generator state with it. A third empty-handed entity sits at id 0 in every fixture, so a dropped presence guard acting on the zero value moves a real code and is caught rather than masked. |
| AC-4 | same file, `TestGiveAllPreservesTheReceiversOwnOrderAcrossASecondPour` | Two nodes in one trigger leave the receiver holding `[9 1 2 3 4]`, and both givers empty. |
| AC-5 | `pkg/mapload/script_test.go`, `TestAnActionNodesTwoUnitReferencesReachTheCompiledInstant` and `TestAnActionNodesUnresolvableSecondUnitReferenceIsReportedAndAbsent` | An action node naming two unit parameters compiles to an instant carrying both, in parameter order; one naming a single unit leaves the second absent; an unresolvable second parameter leaves it absent and appears in the report. |
| AC-6 | `pkg/sim/giveall_test.go`, `TestGiveAllIsNoLongerReportedAsAGap` | `Unsupported()` and `InertTriggers()` are both empty for a script carrying opcode 28. |
| AC-7 | `pkg/sim/scriptform_test.go`, three tests | A script carrying second references round-trips instant-for-instant and hashes equal; second reference `0` and *no* second reference come back distinguishable; two worlds differing only in that field hash differently; a buffer declaring version 37 is refused. |

**P-1** is a property of the arm reading both references through `indexOfEntity` rather than an index
into storage; **P-2** is asserted inside AC-3's own sub-tests, which compare `w.rng.state` across a
refusal; **P-3** is asserted by AC-1's exact-empty assertion on the giver and by AC-7's round trip,
which carries a giver holding nothing across the form and back.

## What was witnessed by reverting, not by reading

Each of T1's guards and both halves of T2's serialization were removed one at a time and the suite
re-run, and the intended test went red each time. The one honest gap, reported by the executor rather
than hidden: T3's second test — the unresolvable second parameter — passes with or without the line
T3 adds, because `HasUnit2` is false either way when resolution fails. It pins AC-5's third clause and
guards a future regression, but only the first test discriminates the added line.

## SC-1 — the tool no longer reports the gap

`go run ./cmd/almtool script <N.alm>` over both roots. `10.alm` on the English root reported
`instant arm 28: 1 node(s) not implemented` before this story and does not now; its four other
unimplemented lines (arm 2 × 13, arm 20 × 1, arm 6 sub-command 11 × 1, sub-command 15 × 2) are
unchanged, so the arm was removed from that list and nothing else was. All three maps
`TRIG-GIVEALL-025` names — `10.alm`, `50.alm`, `80.alm` — were run on the Russian root as well, and
arm 28 appears in none of their reports.

## SC-2 — the potions reach the hero, on both installs

Mission 10 started through `game.StartMission` against a lawful install. The compiled node is

    instant 25: Op=28 Unit=2 HasUnit=true Unit2=35 HasUnit2=true

Entity 2 is what the map calls unit **21** and entity 35 is the party's hero — which is
`TRIG-GIVEALL-025`'s `10.alm 21 → 10001` resolved through `TRIG-REC-011`'s hero band, and the
agreement was checked before anything was built on it. Before the node runs, entity 2 carries
`[3590 3590 3590]` — three times `0x0e06`, class 14, exactly the map's own stock record at (36,51) —
and the hero carries nothing while wearing `259`. After it runs, the hero carries `[3590 3590 3590]`,
entity 2 carries nothing, and both equipment records are what they were. Identical on the English and
the Russian root.

**The substitution, stated so the evidence is not read as more than it is.** The map's authored
trigger is an escort: its only condition is `distance(unit 2, (56,21)) <= 3`. The run above rebuilds
a world from mission 10's **own** entities, **own** stock and **own** compiled node, and replaces
that condition alone with an unconditional one. Everything the arm reads is the real map's; only what
makes the trigger fire is not.

The escort itself was attempted first and did not arrive: ordered to (56,21), the witch walks to
(39,41), stops there, and is killed over the following ticks — her health running deep into the
negatives while she stands still. That is a pathing and a damage question and neither belongs to this
story, but it is why the firing condition was substituted rather than played out, and it is the
reason this story's build cannot yet be demonstrated by playing the mission through.

## SC-3 — the gate

Run on a clean tree at `b57ded1`:

    go build ./...                      clean
    go vet ./...                        clean
    gofmt -l $(git ls-files '*.go')     printed nothing
    go test -count=1 -trimpath ./...    exit 0, 32 packages ok, no FAIL
    scripts/check-no-game-assets.sh     clean (tree scan)
    scripts/check-doc-budget.sh         no FAIL
    scripts/check-sdd-audit.sh          no FAIL

`git log --format='%h %(trailers:key=Co-Authored-By)' master..HEAD` prints no trailer on any of the
four commits.

## What this story did not do

The **notification packet** is not implemented and nothing stands where it would go: the routine the
original calls once per owner sits behind a `(u16)actor+0x0e` in `[0x21,0x40)` gate that research has
located and not interpreted, and grades **Medium**. Drawing, the inventory window and spawn-time
equipment are all elsewhere. No player-issued transfer exists; this is a script arm and no command
kind was added.

**`formatVersion` 38 was needed and taken.** Moving codes between two containers needs no byte-form
change on its own, but the record that names the two containers had nowhere to put the second one, so
the widening — and the version — is the story rather than an incidental of it.
