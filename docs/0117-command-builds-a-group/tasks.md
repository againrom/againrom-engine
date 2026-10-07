# Tasks — 0117

Trailer: `SDD-Task: 0117-command-builds-a-group/T<n>`.

## T1 — the command group as state

Add `CommandGroup uint32` to `sim.Entity` (world.go), documented beside `Group`: zero means the
actor is in its placed group, `Group` never moves, and the new field is read by engage.go alone.

Add `effectiveGroup(e Entity) uint32` to engage.go and route through it the six places that ask
which group an actor is in — the skip lookup and the partition in `aiGroups`, the pair scan in
`groupKeys`, the member scan in `freezeGroups`, and `groupLivingMembers`.

binary.go: `formatVersion` to the next number free of every sibling lane, `entityLen` 145 → 149, the
field at `+145`, four little-endian bytes carried whole with no value refused. Write the version's
own paragraph above `formatVersion` in the shape every version before it has.

Nothing writes the field here, so no tick may answer differently. Pinned forms and digests are
re-taken by running the suite; a **behavioural** assertion that has to change is a finding and stops
the task. `TestThePreviousVersionFormIsRefused` needs its builder rewritten to strip the new tail.

Tests: a round trip carrying assorted command groups including zero, the maximum and one equal to a
placed id; the previous version's refusal naming both numbers; and that with nothing written, every actor's
effective group is its placed group.

Do not touch script.go, the move arms, or any id allocation.

## T2 — a player move order builds one

group.go gains `commandFloor`, `freeCommandGroup` and `commandGroup(members []int, order uint8,
ordered cell)`, per plan.md's T2 section.

Two call sites and no others: `groupOrder` before `issueGroupDestination`, and `stepWorld`'s
`KindMoveTo` arm beside its destination write. Both pass `orderMove`.

script.go: the alive-count check's note says nothing in this package moves an actor between groups.
That is now false and it is the argument for having no membership index. Correct it in place — the
count reads the placed group, which still never moves.

Tests, synthetic, in pkg/sim: AC-2 through AC-8. AC-2 is a **pair** — the commanded guard that is
not walked home beside an uncommanded one of the same world that still is. AC-10 is checked by
deleting the two calls and confirming the package builds and only this task's tests fail; restore.

AC-1 in cmd/missionrun, guarded on `AGAINROM_ASSETS` and skipped without it: start mission 10, take
script unit 21, order it a few cells off its placement, step, and assert it is never given its own
post as a destination — with a fatal check that it left the post at all.

## Traceability

| Spec | Task |
|---|---|
| FR-1, FR-2, FR-9 | T1 |
| FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-10 | T2 |
| DD-1, DD-2 | T1 |
| DD-3, DD-4, DD-5, DD-6 | T2 |
| AC-9 | T1 |
| AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-10 | T2 |
