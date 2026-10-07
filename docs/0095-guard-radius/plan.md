# Plan — 0095 the frozen guard radius

## Approach

One package, `pkg/sim`, in two commits. The first gives the world the record, freezes it at
construction, carries it in the byte form at version 19 and lets it into the digest — with nothing
in a tick reading it yet. The second makes the engagement clip read the frozen base instead of
recomputing one, which is the behaviour, and deletes the per-tick computation's role as a reader.

The split is real and not a slice of one change: the first commit is form-and-state work whose
tests are the round trip, the digest and the field-set pin, and it leaves the tree coherent and
green against a build whose behaviour is unchanged. The second commit is one call site and its
behavioural tests. Neither passes tests against a half-built version of the other.

Nothing outside `pkg/sim` moves. No loader fills the record, because the record is derived from
entities the loader already places; no caller passes one in, because there is no parameter to pass
it with.

## Design decisions

- **DD-1 — the record is `World.groups []groupAI`, keyed and ordered by `(owner, group)`
  (FR-1, FR-5).** A slice and not a map, for the reason `aiGroups` is a scan: Go randomises map iteration
  order per process and this is a path that touches a world and its digest. Ascending order is what
  makes the byte form and the digest depend on the logical world alone rather than on the order the
  entities arrived in.
- **DD-2 — the key set is every OWNED entity, alive or not (FR-1, D-5).** A felled member is
  unlinked from its group in the law but the group survives it, and nothing here appends an entity
  to a world — so keying on the whole placed set makes the set a constant no tick has to maintain,
  and keeps a group whose members have all fallen writable in the byte form. Slot 0 names no group
  and is excluded, which is `aiGroups`' own rule.
- **DD-3 — the record holds the BASE and not the working radius (FR-2, FR-4, D-2).** The law stores
  the installed base and derives the working radius from it at the arm, adding the margin and the
  roll. Storing the working radius instead would fold a constant into hashed state and would make
  the roll — the next story that touches this — a change to the byte form rather than a change to a
  reader.
- **DD-4 — one byte for the base (FR-2, FR-7).** It is what a byte compare reads in the law, and
  the computation already narrows each member's distance-plus-range to a byte before the maximum.
  A wider field could hold bases the rule that produces one cannot. The two key words are 32-bit,
  matching the entity's own `Group` and `Owner`.
- **DD-5 — the geometry and the margin become two functions, and the margin is added once
  (FR-2, FR-3, FR-4).** The computation this file already had folded the margin into its result.
  That was right while one caller wanted the finished radius and wrong the moment a second caller
  wanted the base: the freeze would have to subtract the margin back off, which is exact under the
  byte arithmetic and puts one constant in three places. So it splits — a function computing the
  base alone, which the constructor calls once per group, and a reader turning a stored base into a
  radius, which the clip calls. The arithmetic is bit-for-bit what it was, and the byte-form and
  digest pins are what proves that rather than a claim. The function's own comment said this file
  was the seam and that a story adding the record would change what reads it — this is that change,
  and the comment is replaced rather than left standing.
- **DD-6 — the section sits between the routes and the script (FR-6).** The script section is
  required to consume what is left of the buffer exactly and the relation is taken off the end, so
  a new counted block can only go where both contracts still hold: after the routes, whose own
  reader reports what it consumed, and before the script, which then consumes the rest. Every
  offset the previous version documented is unmoved, which is the trade every bump since 12 has
  made.
- **DD-7 — the decoder checks the section's SHAPE and not its key set (FR-8).** Carrying the keys
  buys a form that can be read record by record and a decode that fails loudly on a wrong count
  instead of parsing a base out of the middle of the script section, so the count is bounded
  against the buffer and the records are required to strictly ascend. The key set is deliberately
  **not** checked against the entities: the mission script's hand-over arms write an entity's owner
  inside a tick, so a world this package's own `Step` produces can hold an entity whose current
  pair no frozen record names, and a cross-check would refuse a world the encoder writes — the
  exact asymmetry every other refusal in that file exists to prevent. The gap is pinned by a test
  rather than left implicit, because an unchecked invariant that nothing measures is a preference.
- **DD-12 — a pair with no record clips at base zero (FR-9, P-2).** A group object in the law is
  allocated with its whole AI record zeroed, and the base lives inside that zeroed span; only the
  installer writes it. So a group that has never had guard issued to it carries zero, and this is
  the derived answer rather than a chosen sentinel. The two rejected alternatives: the floor, which
  is what the *installer* raises to and therefore assumes an install that never happened; and
  recomputing from live geometry, which is the behaviour this whole story removes and would leave
  one path on which the radius still follows the group.
- **DD-8 — no base value is refused (FR-7, P-4).** Every byte is a base some geometry produces, so
  a refusal would make a world the constructor builds a world the decoder will not read back. This
  is the facing's and the sight range's rule and it is taken for their reason.
- **DD-9 — the version's pins are recomputed outside the tree.** The pinned bytes are a hand
  transcription that gains the new section by hand, and the pinned digest is derived from those
  bytes by an implementation of the hash that is not this package's. A second test strips the
  section back off and sets the version byte to the previous one, and requires the previous
  version's pinned digest — so the transcription and the constant each have to be right, and the
  byte-form pin cannot come to agree with the encoder.
- **DD-10 — the version is 19 and no test states it as a literal (FR-6, AC-8).** The number is a
  project-wide allocation any lane may move. A test asserting `formatVersion == 19` would hold when
  a story added a field and forgot to bump, and would fail when another lane legitimately took 20;
  what the tests state is that the form opens at whatever this build defines and that every other
  byte is refused.
- **DD-11 — the pinned world's own records are the evidence for the two edges (AC-3).** Its three
  entities are one in slot 0, one alive and alone in its group at sight 255, and one not alive at
  all. So the transcription carries a base of 255 — the byte's top, reached by geometry — and a
  base at the floor reached by a group with no living member, without a fourth fixture.

## Files

- `pkg/sim/engage.go` — the record type, the freeze, the clip's new reader, and the comment that
  named this file the seam.
- `pkg/sim/world.go` — the `World` field, its construction in `newWorld`, and the doc that says why
  it is canonical.
- `pkg/sim/binary.go` — version 19, the section's layout, its encode and its decode with the five
  refusals.
- `pkg/sim/nostate_test.go` — the field-set pin: the new `World` row and the record type's own
  table.
- `pkg/sim/binary_test.go`, `pkg/sim/hash_test.go`, `pkg/sim/budget_test.go` — the byte-form pin,
  the digest pin, and the strip-back test.
- New tests for the freeze, the two edges, the constancy under stepping and the malformed sections.

## Success criteria

- **SC-1** Every acceptance criterion has a test, and the freeze tests fail if the radius is
  recomputed anywhere on a decision path.
- **SC-2** `go test -trimpath -count=1 ./...` is green with no game install present.
- **SC-3** The census of load-time circles is run on BOTH roots and reported whichever way it goes:
  how many guard groups on the shipped campaign maps hold a hostile inside their frozen circle,
  under the geometry reading and under D-3's floor-only reading, plus the distribution of each
  group's distance to its nearest hostile. If the count is large, `verification.md` states which of
  the two it is — the freeze is not the whole answer, or the initial radius is wrong — and says so
  with the numbers rather than leaving the reader to choose.
- **SC-4** A drift measurement is run on BOTH roots, before and after, over the shipped campaign
  maps stepped with no commands: entities moved, total displacement, deaths. The result is recorded
  as a result; nothing is tuned toward it.
- **SC-5** `TestTheTenthMissionIsDrivenToAWin` and the mission-10 drive under `-trace` are run on
  both roots and their outcome and tick reported. If either moves, the new tick and the arm that
  fired are reported and neither the test nor any predicate is edited.
- **SC-6** A revert check: with the freeze reverted to a per-tick computation and nothing else
  changed, the new behavioural tests fail. A line nobody can break is not witnessed.

## Risks

- **R-1** *Freezing is not the fix for the report that prompted the story.* Measured before the
  work began, and stated in the spec's own Why. Mitigated by SC-3 and SC-4 stating the numbers
  rather than a verdict, and by the story claiming only the divergence it removes.
- **R-2** *A frozen base is WIDER than the recomputed one late in a mission,* because a group
  converging on its target closes up and its spread term falls. So freezing may admit candidates
  this build currently drops. Mitigated by SC-4 measuring both directions rather than assuming one.
- **R-3** *The record set and the live partition come apart.* `aiGroups` partitions the living and
  the record set keys on all owned entities; a group whose members are all dead is a record with no
  partition, which is fine, but the reverse would be a lookup that misses. Mitigated by P-2 and by
  the decoder's key check, which makes the two derivable from one another.
- **R-4** *The digest pin is recomputed from the encoder by accident,* which would make the
  byte-form test agree with itself and witness nothing. Mitigated by DD-9's out-of-tree derivation
  and by the strip-back test, which cannot pass unless the transcription is right.
