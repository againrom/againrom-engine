# Plan — 0117

## Shape

Two commits. The first adds the state and carries it in the form and changes **no behaviour**: with
nothing writing a command group, every effective group equals its placed group and every tick
answers exactly as it did. The second adds the writer, and it is one function with two call sites.

Splitting there is what makes AC-10 checkable: after T1 the whole read side exists and is exercised,
so deleting T2's two calls returns the build to T1's behaviour without touching a line of the
decision.

## T1 — the state and the form

**`pkg/sim/world.go`.** `Entity` gains `CommandGroup uint32` — DD-1's second word rather than a
rewrite of the first — documented beside `Group` with FR-1's sentence and DD-2's: zero means the actor is in its placed group, and zero is not a chosen sentinel
because the allocator cannot produce it. Say there where the field is read — the engagement layer
alone (FR-2) — and say that `Group` is unchanged and is what the script names.

**`pkg/sim/engage.go`.** Add

```go
func effectiveGroup(e Entity) uint32
```

and route the six sites that ask which group an actor is in through it (FR-2): the skip lookup and
the partition in `aiGroups`, the pair scan in `groupKeys`, the member scan in `freezeGroups`, and
`groupLivingMembers`. Nothing else in the package reads `Group`, and nothing else should start.

**`pkg/sim/binary.go`** (FR-9). `formatVersion` to the next number free of every sibling lane;
`entityLen` 145 → 149; the command group at `+145`, four bytes, little-endian, carried whole with no
refusal. Write the version's own paragraph in the block above `formatVersion` in the shape every
version before it has: what the version before it says nothing about, why that silence is not a gap this
build can fill, and that the records moved by four bytes so an old buffer read against these offsets
is a misparse rather than a wrong value.

**Re-pin.** Every pinned form and digest in the suite moves. They are found by running the package
and reading the failures; do not hand-edit one without re-running it. `TestThePreviousVersionFormIsRefused`
builds the previous version's own stream and needs its builder rewritten to strip the new tail.

**Tests (T1).** Round trip a world whose actors carry assorted command groups, including zero, the
maximum and one equal to a placed id. Assert the refusal of the version replaced names both numbers (AC-9). Assert
that with no command group written, `effectiveGroup` is the placed group for every actor and the
world's digest over a fixed schedule is a single pinned value.

**Do not** touch `script.go`, the two move arms, or any group id allocation in T1.

## T2 — the writer

**`pkg/sim/group.go`** gains the whole seam, three functions:

```go
func (w *World) commandFloor() uint32              // FR-5, DD-3
func (w *World) freeCommandGroup() uint32          // FR-5, DD-4
func (w *World) commandGroup(members []int, order uint8, ordered cell)
```

`commandFloor` is one above every group id the world's script names — over both the check array and
the instant array, each read only where its own `HasGroup` is set — and at least 1. A world with no
script answers 1. If the maximum is the largest a group id can hold, no id is allocatable; the
command then builds no group and leaves every actor where it was, which is a state a map cannot
reach and is written down rather than left to wrap.

`freeCommandGroup` is the lowest id at or above the floor that no actor's **effective** group names.
It is called after the members have been released, which is DD-4 and is the whole of why the id
list does not grow with the click count.

`commandGroup` is, in order: drop every member's command group; take the id; write it on every
member whose owner is not slot 0 (FR-7); then, for each distinct owner among those members, upsert
the record at (owner, id) — order, the ordered cell whole (DD-5), and the notice base over that
owner's share of the members by `noticeBase` and `groupCentroid`, exactly as `freezeGroups`
computes one. Upsert means: overwrite in place if the key exists, otherwise insert at the position
that keeps `w.groups` ascending (FR-6, P-3). A call with no member does nothing.

**Call sites, two** (FR-4, and FR-10 by omission — no third arm gains one). `groupOrder` calls it
with the resolved members and the ordered cell before
`issueGroupDestination`; `stepWorld`'s `KindMoveTo` arm calls it with the one member and the
commanded cell, beside the destination write it already makes. Both pass `orderMove` and no other
value, which is DD-6. Neither
`issueGroupDestination` nor either script setter gains a call — they command groups that already
exist, and a group built there would be the law's own scenario command allocating one, which this
tree cannot do.

**`pkg/sim/script.go`.** The alive-count check's own note states that nothing in this package moves
an actor between groups. That is now false and the note is load-bearing — it is the argument for
there being no membership index. Correct it in place: the count reads the **placed** group, which
still never moves, and say why the distinction is what keeps a player's click out of a win chain
(FR-3).

**Tests (T2).** AC-2 through AC-8 and AC-10, in `pkg/sim`, synthetic. The two that carry the story:

- the guard actor that is not walked home after a command, beside an uncommanded actor of the same
  world that still is (AC-2) — the *pair* is the test, because a single actor standing still proves
  nothing about whether the arm ran;
- the id taken twice being the same id, with the record count unmoved (AC-5).

FR-8 is a claim about what is ABSENT — nothing returns an actor to its placed group — so it is
witnessed by the first of those two and by there being no such writer to point at, not by a test of
its own.

AC-10 is checked by deletion, not asserted: remove the two calls, confirm the package builds and
only the T2 tests fail, restore.

**AC-1, the reproduction, in `cmd/missionrun`.** Guarded on `AGAINROM_ASSETS` and skipped without
it, on the tenth-mission drive's own precedent. It starts mission 10, takes script unit 21, records
its cell, orders it a few cells away with one ordinary move command, and steps. The assertion is
that the unit is **never given its own post as a destination** over the run, plus a fatal check that
it actually left the post — without which the test would pass on a unit that never moved. Before
this story the same test fails at the tick after arrival.

## The three claims to check at the end

**SC-1** is checked by reading the diff of `engage.go`: `walkHome` must be untouched. **SC-2** by
grepping the tree for the new field outside `pkg/sim`. **SC-3** needs no work — it is the state the
build is left in, and it is true because nothing was added to sweep a record.

## Order and risk

T1 before T2: T2's tests read the field T1 adds.

The one risk worth naming is the re-pin in T1. A pinned digest that is edited to whatever the build
now prints asserts the build against itself. The rule for this story: a digest may be re-taken, but
every *behavioural* assertion beside it must be left exactly as it stands and must still pass. If a
behavioural assertion has to change in T1, that is a finding — T1 is not supposed to move behaviour
— and it stops the task rather than being edited.

## Verification

Beyond the gate: the mission-10 drive under `-mission 10 -census -waypoint u21:56:21:3 -waypoint
p0:66:16:3`, both roots, compared against the recorded outcome. This story changes which group
decides for a driven unit, so the drive may legitimately move; what it may not do is move
unremarked.
