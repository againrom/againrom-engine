# Plan — one constant, one field, one readout, and four written premises it overturns

## Approach

The defect is one unwritten field. The design work is not the fix. It is (a) making sure the number
it takes cannot come apart from the number the engine already compares against, (b) finding every
place the tree reasons from the party's ABSENCE of an owner, because each of those becomes false the
moment it has one, and (c) building a witness for a map's own diplomacy that the import DAG allows.

## DD-1 — the slot is `pkg/sim`'s, exported, not a second literal in `pkg/mapload`

`pkg/sim/engage.go` already holds `const selfSlot = 1` with the whole derivation on it, and
`pkg/mapload` already imports `pkg/sim`. So the tree already states the player's slot, in the same
tree that creates the party at 0.

**Chosen:** export it as `sim.SelfSlot` and have the party placement write that. FR-2 is then
structural rather than a promise. It stays untyped, so the existing `selfSlot + 1` keeps working and
a `uint32` field takes it without a conversion.

**Rejected:** a `mapload.PartySlot = 1` beside it. Two constants that agree today is the failure the
tree's own comments name elsewhere. Rejected also: passing the slot in on `PartyMember`, which makes
the player's identity a caller's choice before any caller exists that could choose differently.

The rename reaches one use, the declaration, one doc mention, and two occurrences on one line of
`engage_test.go`.

## DD-2 — the field, and the three comments that assert its absence

`Owner` joins the party composite literal in `start.go`. `Group` is not written: FR-3 fixes it at 0,
which is the zero value, and writing `Group: 0` would state as a decision what is the absence of one.

Three comments assert the party stands at slot 0, and all three are this task's own input:

- `pkg/mapload/start.go`, the `ScanRange` field comment — *"An entity of roster slot 0 belongs to no
  group and takes no engagement decision"*, written to explain why a hero's sight is filled and
  unread. It is now read.
- `pkg/sim/engage.go`, `aiGroups`'s doc — *"AN ENTITY OF SLOT 0 BELONGS TO NO GROUP … the hero, the
  party, everything this tree spawns"*. The rule is unchanged; the population it names is not.
- `pkg/mapload/fromalm.go`, the owner field comment — *"THE OWNER IS CARRIED HERE AND NOWHERE ELSE.
  Every other way an entity reaches a world — the party placement, a test's own literal, a decode —
  leaves the field at zero"*. This is the sentence the story most directly falsifies.

A fence that forbade editing `fromalm.go` would leave the tree asserting the opposite of what it
does, so the fence there is on the CODE, not on the comment.

## DD-3 — the front end is told, because not telling it is a regression

`pkg/game/world.go` pushes `SetLocalOwner(0)` on the written premise that this build cannot know
which slot the local participant holds. That premise is now false for the campaign, and leaving the
zero is not neutral: `pkg/ui/numeral.go` compares `e.Owner != v.localOwner`, which is false for a
party member today only because both sides are zero. Move one and the player's own damage numerals
flip their drift direction.

**Chosen:** push `sim.SelfSlot`. It preserves the numeral behaviour FR-5 protects, and it turns
`canArmAttack`'s gate live — which is what that function's own comment says happens the day a
nonzero value is pushed, "with no other change anywhere".

**Not chosen:** leaving it and disclosing. The disclosure would be of a regression this story
introduces, and that is not what a disclosure is for.

## DD-4 — the readout re-derives, because the DAG forbids the alternative

`internal/archtest` grants `cmd/almtool -> {pkg/formats/alm, pkg/mapload}` and asserts
`cmd/almtool -> pkg/sim` is a FORBIDDEN edge. `relationFrom` returns `sim.Relations`, so exporting
it would be unusable from the tool: the tool cannot hold the loader's answer at all.

**Chosen:** the mode re-derives the effective row from `alm.Group` by the published rule, in
`cmd/almtool/roster.go`, and FR-8 says so — it witnesses the map, not the loader. One case in the
dispatch switch and one usage line in `main.go`, so the diff against a concurrently edited tool is a
case label.

**Rejected:** refactoring `relationFrom` to build plain byte rows a `mapload` helper could export.
It is the better shape and it is a change to a hashed-state path for a developer tool's benefit; the
drift it would prevent is two lines wide and is bounded by AC-12 running against shipped maps.

**The fixture is the work, not the verb.** `minimalALM()` builds a document with no type-5 and no
type-6 content. The mode's test extends it with 76-byte roster records — the name written as bytes,
never as literal non-ASCII text — and 70-byte unit records carrying an owner at `+0x14`.

## DD-5 — the evidence is synthetic for the rule and real for the map, on both roots

AC-4 through AC-8a, AC-10 and AC-14 are properties of the simulation and the viewer and get fixtures
built in test code. No install.

AC-9 and AC-12 read the `.alm` alone, and `10.alm` is byte-identical on both roots, so one reading is
both. **SC-1 and SC-2 are not** — a drive resolves every placement's health, sight, speed and combat
block out of the install's own `Data.bin` and registries, so under G1 they are run twice.

AC-11 is **already witnessed**: `pkg/sim/binary_test.go`'s
`TestTwoWorldsDifferingOnlyInAnOwnerHashDifferently` asserts the digest difference including the
0-versus-1 pair. Re-asserting it would add no coverage, so the story's byte-form task witnesses what
that test does not: a world built BY A START round-trips with the party's slot intact.

## DD-6 — no version is taken

The owner slot is already encoded at the entity's `+87` and already decoded. The field set does not
change, so the version literal does not. Witnessed by the round trip and by the decoded field, never
by reading the literal.

## Risks

**R-1 — a frozen digest of a world WITH a party must be re-taken; one of a world WITHOUT a party
must not move.** If a no-party digest moves, DD-2 is wrong.

**R-2 — a test can keep passing while ceasing to test what it names.**
`start_test.go`'s `TestAStartedMissionFightsWithoutBeingTold` places a slot-1 unit 2 cells from the
hostile and drops the party 5 cells from it. Both are now candidates; cost ordering still picks the
nearer, so the test would pass for a reason it does not state. AC-8a is that reason, stated.

**R-3 — the AI becomes a writer of orders on units the human commands.** `orderAttack` ends in an
unconditional `clearOrder`, so a party member that acquires a target loses its walk. Measured: a
single move order past an adjacent hostile leaves the member where it stood; the same order re-issued
every eight ticks carries it to the cell a slot-0 member reaches. It is FR-6a, and it is disclosed
rather than repaired — the original branches inside this arm on whether the unit belongs to a human
participant, and that branch is undecoded. Authoring one here would be inventing the answer in the
one place the whole engagement layer reads.

**R-4 — `ScriptInstantGiveGroup` writes an owner to every entity whose group word matches, and the
party's is 0.** A shipped `GiveGroup` naming group 0 would re-own the party mid-mission. Measured
over both roots: 28 nodes, **none** names group 0. Bounded, recorded, not fixed.

## Success criteria

- FR-1, FR-2, FR-3 -> AC-1, AC-2, AC-3.
- FR-4 -> AC-4, AC-5, AC-6, AC-8a.
- FR-5 -> AC-8.
- FR-6, FR-6a -> AC-7, AC-7a.
- FR-7 -> AC-10, AC-11.
- FR-8, FR-9 -> AC-12, AC-13, AC-9.
- FR-10 -> AC-14.
- SC-1, SC-2 and SC-3 are measured from the tools and the corpus and recorded, not asserted in the suite.
