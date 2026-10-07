# Tasks — 0097

Two tasks, one commit each, trailer `SDD-Task: 0097-commanded-group/T<n>`. Both are in the
simulation tree; neither touches `docs/`, `builds/` or any tool.

**T1** carries FR-1, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11 and DD-1, DD-2, DD-3,
DD-4, DD-5.
**T2** carries FR-2 and DD-6. SC-1 to SC-4 belong to no task: they are measured in the verification
stage, which is not a task and takes no trailer.

## T1 — the state, the partition, and what follows

**Files:** `pkg/sim/engage.go`, new `pkg/sim/commanded_test.go`, `pkg/sim/partyslot_test.go`.

Add the predicate FR-1 defines and give the partition of deciders its second skip clause, beside the
one for an entity belonging to no roster slot. Nothing else in `engage.go` moves: no arm, no
scorer, no constant, no attack-order writer.

Both docs are deliverable. The predicate's says what the two fields stand for, why reading them is
exact **today**, and which future writer breaks that — the file's header names the branch this
build lacks, so point at it by the name used there. The partition's gains a paragraph in the same
voice: the unit is in the group its command built, that group takes no decision, and it is still a
candidate for everyone else.

Cover AC-1 to AC-10, AC-12 and AC-13 in the new file, on the package's synthetic worlds. AC-12 pins
the digest as a literal so a later version bump cannot pass quietly.

One existing test measures the behaviour this story removes and goes red: it drives a unit at the
player's slot under a single order and asserts it is held in contact. Rewrite it to AC-15 — the
assertion its own slot-0 control was already the comparison for — in this same commit, naming in
its doc whose disclosure it carried. Fixture, control and walkers stay; only the expected side of
the comparison moves.

## T2 — the tripwire that keeps FR-1 exact

**Files:** new `internal/archtest/destination.go` and `internal/archtest/destination_test.go`.

FR-1 is exact because of a property of `pkg/sim`'s source, so pin that property where such
properties already live. Follow the determinism check in the same package exactly: a pure function
over a map of filename to source, returning findings that name file, line and enclosing function in
file order, plus a test that runs it over the live tree through the loader that is already there.

What it collects: every assignment giving an entity's destination flag the value `true`, anywhere in
`pkg/sim`'s non-test files. The expected set of enclosing functions is pinned by name in the test,
and the failure message must say what appeared or vanished and why it matters — a reader who trips
this has not read `spec.md` and needs one sentence, not a diff.

Drive the check itself over literal source strings for the three cases that matter: an unexpected
writer, an expected one gone, and no writers at all. That is AC-11.
