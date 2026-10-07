# Tasks — 0096 the mission script's group command

`impl` = production code plus its tests, one commit each. T2 follows T1.

## T1 — the group order is state, and a decision reads it `impl`

Give the group record an order byte and a commanded cell, write the order at construction, carry
both in the byte form and the digest at the new version, and make the decision fork on the stored
byte instead of on the owner's slot. No writer of an order exists yet beyond construction, so every
world behaves exactly as before.

Files: `pkg/sim/{world,binary,engage}.go` and the tests of that package that pin the field set, the
byte form, the digests and the form version.

Covers FR-1, FR-2, FR-3, FR-12, FR-17, FR-18, FR-19, FR-20, FR-21, FR-23; DD-1, DD-2, DD-3, DD-4,
DD-5, DD-6, DD-7, DD-16, DD-17.

Scope fence: no script arm, no new engagement arm, no change to the clip, the candidate sweep, the
scorers or the distribution. Orders 2, 4 and 5 are unreachable after this task and need no arm.

Done when: the derivation from the owner's slot exists nowhere on a tick path, the missing-record
case included; the pinned world's bytes and digest are a hand transcription carrying both fields; a
decode refuses an order outside the five with the value and the record named; the version is read
from the constant by every test.

## T2 — the command, and the three arms it can select `impl`

Add the script's group command as a dispatch on its own first plain parameter with five writers, a
gap report keyed by sub-command, and the three engagement arms the new orders need. Lift the
group-move distribution so the two setters and the player's own order share one body.

Files: `pkg/sim/{script,group,engage}.go`, `pkg/mapload/script.go`, `cmd/almtool`,
`cmd/missionrun`, and the tests of each.

Covers FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, FR-13, FR-14, FR-15, FR-16, FR-22; DD-8,
DD-9, DD-10, DD-11, DD-12, DD-13, DD-14, DD-15, DD-18, DD-19.

Scope fence: the record, its two fields, the form and the version are T1's and are not touched. The
distribution's own rules — the formation gate, the offsets, the rate term, the clamp — are moved,
not changed. Nothing writes a group order on the player's path.

Done when: each of the five sub-commands writes what the table in the plan says and each of the
others leaves the world bit-for-bit unchanged; a script authoring one implemented and one
unimplemented sub-command reports exactly one gap, naming the unimplemented one; the three arms
differ from one another only in the clip, the scorer, the walk and Swarm 2's empty-list gate; the
tenth mission's outcome and tick are unchanged.

## Traceability

| Task | Requirements | Decisions |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-12, FR-17, FR-18, FR-19, FR-20, FR-21, FR-23 | DD-1, DD-2, DD-3, DD-4, DD-5, DD-6, DD-7, DD-16, DD-17 |
| T2 | FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, FR-13, FR-14, FR-15, FR-16, FR-22 | DD-8, DD-9, DD-10, DD-11, DD-12, DD-13, DD-14, DD-15, DD-18, DD-19 |
