# Story `1041` — campaign-script execution closure

This story closes an execution-evidence gap. It does not turn an observed visit into proof that the
visited behaviour is faithful.

## Result

Every runtime operation family reachable from the 28 shipped campaign maps has a production-path
witness on both preserved roots. The witness starts from a normally loaded mission, crosses the
ordinary script pass and dispatch, and checks the operation's externally visible state change or
deliberate no-op. The two build-time operation rows have builder-boundary witnesses.

The result is one machine-readable matrix. For every authored check, trigger, instant and instant-6
group sub-command it distinguishes:

- present in the shipped file;
- accepted by the builder;
- reachable from an accepted trigger;
- observed during an ordinary headless play drive;
- exercised through the controlled production seam when ordinary play did not reach it;
- dispatched to the expected arm;
- verified at the arm's effect boundary.

No row may use “supported”, a switch-case visit, a passing helper test or a source-coverage percentage
as a synonym for the last two states.

## Why this story exists

`pipeline/check-milestone.sh` loads all 28 maps on EN and RU, but its campaign-wide half asks only
whether every opcode is present in the supported tables. At one tick it prints each script's counts and
any `UNSUPPORTED` row. It does not say whether a check produced the right value, whether either side of
a comparison occurred, whether a trigger fired, whether its instant reached the dispatch, or whether
the dispatch changed the right state. Its zero is a support census, not execution closure.

The lane measured implementation base `e3466269cdd4b7fa23b46f2103265297ce17e8d7` with research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209` before implementation:

| Measure | EN | RU |
|---|---:|---:|
| checks compiled across 28 maps | 680 | 678 |
| instants compiled | 759 | 759 |
| triggers compiled | 398 | 397 |
| reachable instant nodes in the research census | 614 | 614 |
| reachable instant-6 sub-command nodes | 89 | 89 |
| triggers firing in an unattended 4,000-tick drive of every map | 72 | 72 |
| distinct instant nodes run by that drive | 147 | 147 |
| runtime instant opcode families run by that drive | 17 of 25 | 17 of 25 |

The unattended drive observed 72 distinct trigger nodes, 147 distinct instant nodes and 17 of the 25
runtime instant opcode families on each root. It missed eight reachable instant families: 5, 13, 18,
22, 23, 25, 30 and 33.
It is not proposed as a complete play strategy; it is the counterexample to treating a campaign load
or an `UNSUPPORTED == 0` report as execution coverage.

The ordinary Go suite is not the missing instrument either. On the same tree `go test ./pkg/sim`
reports 91.8% statement coverage, `runCheck` 98.0% and `runInstant` 97.9%. Those are useful local
numbers and say nothing about whether shipped data reaches the statements through the production
binder and mission loop.

## Authority and denominator

Read `TRIG-CLOSURE-037` whole. Its active amended row owns the shipped operation vocabulary,
builder reachability and EN/RU differences. The current pinned census has 52 reachable operation
rows per root: 17 runtime check arms, one build-time constant row, 25 runtime instant arms, one
build-time drop row, and eight instant-6 sub-command arms. Checks are evaluated even when no trigger
names them; instants and sub-commands are reachable only through accepted trigger slots.

The effect oracle for each runtime row is the active claim for that arm, never current implementation
code. The two build-time rows use claim-backed builder postconditions. The lane starts from the claim
join behind `TRIG-CLOSURE-037`, re-reads every active row with `go run ./tools/claim <ID>`, and records
the ids in the implementation matrix. In particular:

| Claim | Boundary it prevents the witness from guessing |
|---|---|
| `TRIG-TAKEITEM-038` | instant 13 detaches one item before freeing it and still notifies on a miss |
| `TRIG-OFFMAP-041` | instant 16 removes map presence while preserving the named untouched state |
| `TRIG-GRPARM-047` | group sub-commands 10, 11 and 15, including the authored-unreachable veto |
| `TRIG-GRPLIMIT-048` | byte-width escort range and its zero-to-three coercion |

An active typed divergence is also a valid expected result, but the matrix names its `DIV-` id and
labels the row `DIVERGES`. An uncited copy of current behaviour is not an oracle.

## The seven behaviours

### B1 — one census with stable identities

Add a developer command, provisionally `cmd/scriptcoverage`, which loads the campaign through the
same asset, map and mission constructors as the game. It walks all 28 maps and emits one stable row
per root, map, family and authored node index. An instant-6 row also carries its sub-command.

The command refuses a corpus drift rather than silently changing its denominator. It reports the
current totals beside the expected claim-backed totals and identifies added, removed and reclassified
rows. EN and RU stay separate: their operation sets agree, while two check counts and one trigger
count do not.

The two build-time rows are closed at the builder boundary rather than forced through a dispatcher
they do not enter. The constant row proves its value is preset in the owned register; the drop row
proves the authored record is deliberately absent from the runtime instant list and from every
trigger slot after compilation.

### B2 — complete observation of a script pass

Extend the return-value observer behind `StepTraced`; do not add a hook or mutable trace field to
`World`. On every script pass it records:

- each check's dispatch, whether it wrote, the value written, or the exact silence reason;
- each trigger's decision: inert, skipped by a spent once-latch, failed pair, held or fired;
- the pair values actually compared and the first pair that short-circuited;
- each instant slot entered, its opcode and sub-command, and the dispatch outcome;
- enough before/after data to join the dispatch to B5's independent effect oracle.

Untraced `Step` remains the same execution with nil observers. Tracing adds no field to the world,
byte form or digest. A traced and untraced copy started from identical bytes must remain byte- and
digest-identical after every step.

### B3 — ordinary shipped-mission drives remain honestly incomplete

Run deterministic headless drives through ordinary commands and the production front end. The
matrix labels every operation and node they naturally reach. It never promotes an unobserved row
because another node with the same opcode fired.

The lane must improve the drive set beyond “stand still for 4,000 ticks”, but it does not invent a
single supposed playthrough that claims to make mutually exclusive mission branches all occur. A
natural witness is kept only when the exact authored node fires from ordinary commands and state.

### B4 — controlled production seam for the remainder

Every reachable operation row without a natural witness gets a controlled witness. It uses a real
mission world and the exact compiled check or instant record from that mission. It enters through
`NewScript` and `StepTraced`; it does not call `runCheck`, `runInstant`, `cmdGroupOrder` or an effect
helper directly.

For an instant, the harness may place the exact instant behind an always-true one-shot probe trigger.
It may not rewrite the instant's opcode, parameters or resolved references. For a check, it may seed
the minimum world state needed to reach the claimed side, but the check is still evaluated by the
normal script pass. Every such seed and every predicate replacement is printed in the row, so a
controlled witness cannot be mistaken for ordinary campaign play.

An operation absent from shipped data or a semantic arm proved authored-unreachable is synthetic
coverage, not shipped coverage. Keep it in a separate section. `TRIG-GRPARM-047`'s target veto and
`TRIG-GRPLIMIT-048`'s zero-range coercion are the model: exercise them through the production seam and
label why no shipped node can reach them.

### B5 — effects, not visits

Each runtime row among the 52 reachable operation rows has an independent postcondition at the state
boundary its claim names. Examples are a register value, latch transition, outcome counter, relation
cell, owner, inventory element and stack count, map-presence and occupancy pair, attached effect time,
structure field, group order, per-member order or emitted event. The two build-time rows carry B1's
builder postconditions. A row whose only assertion is “the case was entered” is GAP.

For conditional arms, cover both results when shipped state can reach both. When only one result is
shipped-reachable, cover that result on the shipped node and put the other in the explicitly
synthetic section. Reference-present and reference-missing paths are separate wherever the active
claim gives them different effects.

### B6 — a durable release gate

The matrix is exercised by an asset-gated release test discovered automatically by
`pipeline/check-release-tests.sh`. A normal `go test ./...` with no lawful install skips it and says
why. The seat runs it once per preserved root and reports selected, ran and skipped counts.

Keep `check-milestone.sh`'s support census labelled as support. Do not make its old baseline imply
execution closure. The new test fails when a reachable operation row lacks a dispatch witness or an
effect oracle, when a denominator changes, or when a controlled row is relabelled natural.

Mutation proof is at the production use sites. Removing a supported-table entry, bypassing one
dispatch case, making the observer report a visit it did not see, or suppressing a representative
effect must each make the release witness fail. Restore every mutation byte-identically before the
branch is pushed.

### B7 — findings are dispositioned, not hidden in the matrix

This story closes the execution-evidence gap. If the matrix finds current behaviour different from
an active claim, the lane fixes it only when it is a bounded Campaign & Scripts defect. A defect that
changes another domain's mechanics becomes a separately scoped story or hotfix and a typed
`DIVERGENCES.md` row before `1041` lands. Its matrix row says `DIVERGES`; it never says `PASS` merely
because the mismatch has been recorded.

The lane also corrects the stale comments at `pkg/sim/script.go:293-304` and `1998-2000`. They still
describe four published group-command arms as unimplemented even though the supported table and
dispatch now carry sub-commands 10, 11 and 15.

## Twelve-aspect closure

| Aspect | Verdict owed |
|---|---|
| data | PASS: all shipped nodes on all 28 maps and both roots enter the census |
| runtime state | PASS: tracing is return-only and traced/untraced worlds remain identical |
| simulation | PASS: every reachable operation row reaches its production dispatch and effect boundary |
| input | PASS: natural drives use ordinary commands; controlled rows are labelled, never presented as play |
| AI | PASS where group commands or script-owned orders cross AI; otherwise N/A by row |
| UI/HUD | N/A: the matrix is headless and changes no presentation |
| triggers/scripts | PASS: checks, comparisons, latch arms, instants and group sub-dispatch are all accounted |
| inventory/equipment | PASS for item instants' effect rows; otherwise N/A by row |
| persistence | PASS: no observer state enters bytes or hash, and a mid-mission round trip preserves script state |
| campaign/session | PASS: production mission construction and outcome reporting are used |
| shipped content | PASS: 28 maps on EN and 28 on RU, with denominator drift refused |
| interactions | PASS: each cross-domain effect is checked at its owning boundary or typed as a divergence |

## Domains

Assets, Sim Core, Campaign & Scripts, and AI & Orders. Party, Items & Heroes, Combat & Magic and
Persistence are walked as effect boundaries but are changed only if B7's bounded-fix rule admits the
specific defect.

This work is not split by opcode. One observer and one denominator are the feature; splitting the
families would recreate the exact blind spot, because each partial story could report its own green
subset while nothing proved the union was complete.

## Byte form

No version is reserved. Trace and coverage state are not simulation state. If implementation needs
to serialize any new field, stop and ask the seat before choosing a version.

## Out of scope

- Proving that one unattended drive can win every campaign mission.
- Treating operations absent from the shipped campaign as shipped coverage.
- Fixing an unrelated Combat, Items, Client or Persistence mismatch found by the new instrument.
- Importing lawful game data, generated maps or save files into the repository.
- Using implementation behaviour as evidence of ROM1 behaviour.

## Divergence ids

The seat reserves `DIV-385` through `DIV-400`. Use an id only for a real difference exposed or chosen
by this story. Amend an existing row when the subject is already there. Stop and ask if the range is
spent; an unmerged lane is invisible to the allocator.

## Review ceiling and stopping condition

**Four adversarial passes.** The observer crosses more than three domains and can expose hashed-state
errors even though it adds no hashed state.

The chain stops at the first independent pass with no P finding after the remaining-surface list is
empty. The reviewer must cover: denominator completeness; natural-versus-controlled labelling; all
52 operation rows; check write/silence outcomes; comparison and latch branches; instant and group
sub-dispatch; independent effect oracles; traced/untraced determinism; both shipped roots; and the
asset-gated release path. W becomes a ledger row and D is fixed in place without another pass. Only P
returns the story.

If the ceiling is reached, land the complete truthful instrument, keep every unresolved behaviour
row labelled GAP or DIVERGES, and cut those mechanics into their own stories. Do not weaken the
denominator to make `1041` green.
