# Plan — 0096, the mission script's group command

## Approach

Two slices, in order, each one commit.

The first makes the **order into state**: the group record grows an order byte and a commanded
cell, the constructor writes the order the owner's slot already implies, the byte form and the
digest carry both, and `engage.go` stops deriving a stance and starts reading one. Nothing on any
shipped map behaves differently after it, because nothing yet writes an order that construction did
not — which is what makes it separately checkable.

The second gives the script the command and the arms it can select. The command is one dispatch with
five writers, three of which are one line each and two of which call the group-move distribution
that already exists. The arms are a second fork in `decide`, and the shape they take is the shape
the four published arms take: **two independent choices, a radius clip and a scorer**, of which this
build had two corners and now has three.

| order | clip to the notice circle | scorer | members scored | walk when unengaged |
|---|---|---|---|---|
| 1 Guard | yes | ordinary | all | — (absent, 0086 FR-20) |
| 2 Swarm | no | ordinary | all | to the commanded cell, unoffset |
| 3 Stand Ground | no | reach-vetoing | all | none |
| 4 Move | no | reach-vetoing | arrived only | — (the setter already issued one) |
| 5 Swarm 2 | no | ordinary | all | none — **unless the list is empty, when the row above runs** |

Every row of that table is either what the build already does or one predicate away from it, which
is why the second slice is one commit, not three. Slice one carries the form and digest clauses
(FR-17) and leaves Guard and Stand Ground untouched (FR-12); slice two the dispatch and its two
refusals (FR-4, FR-5), the one-line setters (FR-7, FR-8), and FR-16's keep-what-you-held.

## Design decisions

- **DD-1 — the two fields go on the existing group record (FR-1).** `0095` put a `(owner, group)`
  record on the world with a frozen notice base; this adds `Order uint8` and a commanded cell beside
  it. A second table keyed the same way would have to be kept in step with the first by hand, and
  the record's key set is already exactly the set of groups a decision can be taken for.
- **DD-2 — the commanded cell is a pair of `int32`, not a packed word (FR-20, D-5).** The entity's
  own coordinates are `int32` and the record sits beside entity records in the form. Packing it into
  the law's two bytes would put a narrowing on a path no loadable map reaches and would make the
  field's own round-trip test a test of the packing.
- **DD-3 — construction writes the order; `stance()` disappears (FR-2, FR-3).** The derivation
  `owner == SelfSlot ? StandGround : Guard` moves from `aiGroup.stance` into `freezeGroups`. It is
  the same rule at a different moment, and moving it is what makes FR-3 checkable by *deletion*: if
  a derivation is left anywhere on a tick path, a command's write is silently overridden.
- **DD-16 — a pair with no record decides nothing, and the lookup says so in its type (FR-21).**
  0095 landed `groupBase(owner, group)`, which returns **zero** for a pair no record names and
  documents why that zero is derived rather than chosen: a group object is allocated with its whole
  AI record zeroed, and only an installer writes into it. **The order byte lives in that same zeroed
  span**, so the identical argument gives it zero — and `AI-CMD-033` says so directly, the group
  constructor writing 0 from a register zeroed at its head. The pair-keying that makes this state
  reachable is faithful and not an artefact: the law's own change-of-owner routine is one of
  `RemoveMember`'s four callers, so a handed-over actor leaves its group there too. The lookup
  therefore returns `(order, base, bool)` rather than a bare byte — a caller that cannot tell a
  stored 1 from a missing record would give a hand-over survivor the guard arm, which is exactly
  the derivation FR-3 removes, arriving by the back door.
- **DD-4 — `stance` stops being a two-value enum and becomes the order byte itself.** The five
  orders are the law's own numbering, and a separate internal enumeration would need a mapping in
  both directions and a decision about what to do with a byte outside it. The byte the form carries
  and the byte the decision switches on are one value.
- **DD-5 — an order outside the five is refused on decode (FR-19, P-5).** Opposite to `0095`'s
  treatment of the base, and for the opposite reason: every byte is a base some geometry produces,
  while only five bytes are orders this build writes. Accepting a sixth would build a world with no
  arm for one of its groups, which P-2 forbids.
- **DD-6 — the section grows in place (FR-18).** Each record gains nine bytes and the record count,
  the section's position and every offset before it are unmoved. The script section still closes the
  form and still consumes what is left of it exactly, so the existing length checks keep their
  meaning.
- **DD-7 — the version is 20 and no test states it as a literal (FR-18, AC-13).** The number is a
  project-wide allocation any lane may move; a literal holds when a story adds a field and forgets
  to bump, and fails when another lane legitimately takes the next one. Tests read the constant.
- **DD-8 — `scriptInstantSupported` takes the instant, not the opcode (FR-11).** Opcode 6 is a
  second dispatch, so support is a property of `(op, Args[0])`. `ScriptGap` gains a sub-command
  field, meaningful only for that opcode. The two tables that decide what runs stay the *only* place
  either answer is given — a switch with a silent default is how an unimplemented arm becomes an
  evaluated-false one, and that hazard doubles when one opcode covers ten behaviours.
- **DD-9 — the two setters call the existing group-move distribution (FR-9, FR-10).** The law's
  `Par0 = 4` and the player's own move opcode enter the *same routine*, and `Par0 = 5` enters its
  twin — which is now known to be **the same bytes**, differing in the order immediate alone. So
  `groupOrder`'s body is lifted into a function taking a member list, an ordered cell and the world,
  and both callers use it. The player path's own membership rule — the tag scan over the command
  slice — stays where it is; the script path's membership is the record's living members.
- **DD-10 — the player path does not write an order (D-3).** In the law a player order allocates a
  fresh group whose byte the setter moves; this tree cannot allocate one, so writing the placed
  group's byte would be a behaviour the law does not have. The lifted distribution therefore does
  **not** write an order, and the script's two setters write theirs beside the call.
- **DD-11 — sub-command 1 re-freezes through the same function construction uses (FR-6).** `0095`
  made the freeze a function of the members' live geometry; this calls it a second time. There is no
  second rule and no override parameter — the law's override is 0 on every path this build has.
- **DD-12 — "arrived and idle" is `!HasTarget && Transit == 0` (FR-13, FR-15).** Arrival already
  clears the target in `step.go`, and a crossing counter is already what says a mover is between
  cells. Both conditions exist; naming a third would be a second representation of the same fact.
- **DD-13 — the Swarm walk goes through the plain destination write, not the clamp (FR-13).** The
  law's move-to-the-cell branch stores the commanded cell directly, where the setters' per-member
  issue clamps into the playable rectangle. Following it means an authored cell outside the map
  reaches the member unchanged and the mover gives up by the existing rule, which is a defined
  outcome rather than a silent correction.
- **DD-14 — Move's arm scores only arrived members, and does so with the reach-vetoing scorer
  (FR-15, D-2).** The law gates acquisition on its own arrival latch and its acquisition rule
  discards any pick past reach. Scoring a walking member would make a Move command a Swarm command
  with extra steps; using the ordinary scorer would let a moving group acquire across the map.
- **DD-15 — Swarm 2's gate is `len(cands) == 0` on the list `candidates` returns, and it is written
  as an explicit branch into Move's arm (FR-14, D-4).** The law's field is that list's own element
  count, read on the instruction after the builder returns, and the builder empties the collection
  before its own early exit — so nothing stale reaches it and the analogue here is exact. The
  branch is a no-op in this build (D-4) and is written anyway, because the story that builds the
  re-issue must find the gate already in the right place rather than rediscover where it goes. What
  keeps it from rotting is AC-17: our `candidates` returns the corpse list when the living list is
  empty, exactly as the law moves collection B back into A, so a gate written over the living
  members instead would fail a test rather than pass silently.

- **DD-17 — the release keys on the owner, not the stance (FR-23, AC-21).** `0098` wrote its
  exemption as `st == stanceStandGround`, and its own FR-6 says the rule is really about the
  member's owner, "expressed through the one stance this build has for it". This story destroys that
  identity, so the condition becomes `g.owner != SelfSlot`. A **correction to a landed contract**,
  not a change of behaviour: on every world 0098 could build the two agree, which is why nothing
  goes red and only AC-21's worlds separate them.
- **DD-18 — `aiGroups` excludes a commanded unit only when its group's order issues no destination
  (FR-22, AC-20).** `0097`'s `underCommand` is exact only because the two writers were player
  orders; its own doc names "any script-ordered move" as the writer that breaks it, and this is
  that writer. The discriminator is the stored order: under 1 or 3 a destination is a player's,
  under 2, 4 or 5 the group's own. **`internal/archtest`'s `wantDestinationWriters` pin is widened
  with the triage its failure message demands.**
- **DD-19 — order 5's gate reads the count whole; `decide`'s emptiness test keeps its byte (D-7).**
  Two instructions in the law; `0098` built the byte one. The gate goes **before** `decide`'s
  empty-list branch, which now releases.

## Files

| Path | Change |
|---|---|
| `pkg/sim/world.go` | the record's two new fields; the constructor writes the order; `SelfSlot`'s comment stops naming a stance |
| `pkg/sim/binary.go` | the group section carries both fields; version 20; the order refusal |
| `pkg/sim/engage.go` | `stance()` deleted; `groupBase` becomes the order+base lookup; `decide` forks on the stored order; the three new arms |
| `pkg/sim/group.go` | the distribution lifted to a function two callers share |
| `pkg/sim/script.go` | opcode 6, its sub-dispatch, the five writers, per-sub-command support |
| `pkg/mapload/script.go`, `cmd/almtool`, `cmd/missionrun` | the gap report and the trace print the sub-command |
| `pkg/sim/*_test.go` | the acceptance criteria; the pinned form and digest re-transcribed |

## The tests that pin what this story changes

Listed because two previous lanes paid for not listing them. **T1:** `release_test.go`
`TestTheByteFormsVersionAndAFixedWorldsDigestAreUnchanged` — the **only** place in `pkg/sim` that
states a version as a literal (`b[0] != 19`), against this story's own AC-13; it is rewritten to
read the constant, not renumbered. Its digest and `TestTheAlwaysScoringPopulationMatchesTheGoldenDigest`'s
are the two golden digests version 20 moves. `TestAStandGroundMemberKeepsAVictimThatStepsPastReach`
and `partyslot_test.go` `TestThePlayerStandsHisGroundAndDoesNotChase` pin the stance the constructor
now stores — both must stay green unchanged, which is what says FR-2 preserved the rule.
**T2:** `internal/archtest` `TestDestinationWritersMatchFR2` fails the moment the distribution is
lifted or the swarm walk is added (DD-18). `commanded_test.go`'s eleven tests pin `underCommand`;
`TestACommandedFighterStillDecidesForItsGroup` and `TestARepeatedMoveOrderStaysUnderCommand` must
stay green, since FR-22 narrows the exclusion without removing it.
`release_test.go` `TestTheCandidateCountNarrowsToAByte` pins the byte width DD-19 keeps.

## Success criteria

- **SC-1** Every acceptance criterion has a test, and each arm's test fails if that arm is replaced
  by any other arm in the table above.
- **SC-2** `go test -trimpath -count=1 ./...` green with no game install present, plus
  `go build ./...`, `go vet ./...`, `gofmt -l`, `check-no-game-assets.sh`, `check-doc-budget.sh`,
  `check-sdd-audit.sh`.
- **SC-3** The 135-node census is re-run on **both roots** and reported by opcode as *runs* against
  *still skipped*, from the build's own gap report rather than from a separate count.
- **SC-4** `TestTheTenthMissionIsDrivenToAWin` and `missionrun -mission 10 -trace` are run on
  **both roots**, before and after, and the outcome and tick are reported whichever way they go.
  The prediction on record is unchanged: mission 10's only reachable group commands are Patrol.
- **SC-5** A **revert check** on the two hashed fields: with the order byte removed from the form
  and nothing else changed, a named test fails; likewise for the commanded cell. A version literal
  is not a witness.
- **SC-6** A map that authors an implemented sub-command on a first-pass trigger is driven far
  enough to show the group under its new order doing something the old build did not.
- **SC-7** The Swarm 2 gate is witnessed the only two ways it can be: AC-17 discriminates the list
  the count is taken over, and AC-18 the branch that does not walk. Neither is claimed to witness
  the fallback itself, and `verification.md` says so rather than implying otherwise.

## Risks

- **The digest moves for every world.** Two fields enter the hash, so every pinned digest in the
  suite changes. Mitigated by DD-6 keeping the section's position fixed and by AC-15 re-transcribing
  the pin by hand rather than from the code that writes it.
- **Deleting `stance()` can silently reintroduce the derivation.** A later reader may re-add
  `owner == SelfSlot` for convenience — and the likeliest place is the missing-record case of
  DD-16, where falling back to the owner *looks* like robustness. Mitigated by SC-1's replacement
  test, by AC-19, and by the constructor being the only site the rule appears at.
- **The lifted distribution has two callers with different membership rules.** Mitigated by DD-9
  keeping membership at each caller and passing only the resolved member list in.
- **Swarm 2's gate is a branch no test can enter (D-4, DD-15).** A dead branch rots. Mitigated by
  AC-17, which fails if the gate is written over the wrong list, and by DD-15 stating in the code
  what the branch is for.
- **Two of the rows this revision rests on are newer than the submodule pin.** `provenance.md` marks
  which and where they were read; a pin bump before Phase 4 is what closes it, and no number from
  them is implemented that the bumped pin will not show.
