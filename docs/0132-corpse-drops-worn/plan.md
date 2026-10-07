# 0132-corpse-drops-worn — plan

## Approach

Three edits, in `pkg/sim`'s death path, in its one sack-worthiness predicate, and in
`cmd/missionrun`. Nothing new is introduced anywhere: the drop already exists and already walks two
slots, and the widening is the loop the original's own dispatch runs.

The brownfield discipline applies to the first two. The behaviour being changed is pinned by a
test that asserts the *old* rule (`TestTheTenArmourSlotsAreNotDropped`, `pkg/sim`), so the change
is visible as that test turning over rather than as a silent widening — see DD-5.

## Design decisions

**DD-1 — The strip is written at the death path's own site (FR-1, FR-2).** The two existing
unequips in the crossing block gain a loop over slots 3 to 12, ascending, appending each non-zero
code and zeroing the field. It is not extracted into a helper: the block already holds the
container hand-over, the bounds refusal and the two hand slots as one statement, and a helper would
put half of "what a body gives up" in a second place free to disagree with the first about order.
Slots 1 and 2 keep their named constants and their own two lines, because *which* slot is the
weapon is a decoded fact and an index is not; slots 3 to 12 are a range and are written as one.

The loop sits **inside** the bounds refusal and inside the once-per-crossing guard, after the two
hand slots and before the hand-over to the sack — so FR-5's all-or-nothing covers the ten new
fields by placement rather than by a second guard, and FR-4's once-per-death covers them by the
guard that already holds the rest of the block. FR-6's tie-break is likewise placement and not a
rule: the crossing block runs where the tick reaches it, so two bodies on one cell append in the
order their crossings were resolved, and nothing here chooses an order or could.

*That tie-break is **authored**, not decoded.* No source read for this story says what the original
does when two bodies die on one tick on one cell. What the contract fixes is the property that
matters here — the order is a consequence of the tick's own resolution order and of nothing
unordered — rather than a number taken from somewhere it was not measured.

**DD-2 — `dropsSomething` retires and the death path asks `holdsSomething` (FR-3).** The two
predicates existed because the questions differed: a body could hold armour it would never drop, so
asking the wider question planted an empty sack. Every slot is now droppable, so the two questions
have one answer, and keeping two spellings of it would be two places to keep in agreement for a
distinction that no longer exists. `holdsSomething` survives because `Stock()` also asks it; its
doc block records that the split died and why, so the next reader does not re-derive the removed
distinction from the ledger.

*Rejected:* keeping `dropsSomething` as a one-line forwarder. It reads as a live distinction and
would invite a future edit to make the two diverge again with no fact behind it.

**DD-3 — The order is written as it is read, not sorted (FR-2).** Container, then slot 2, then slot
1, then the ascending walk. The ascending walk is the loop's own direction; nothing re-orders and
nothing sorts, so the sack's byte order — and therefore the world's digest — is decided in exactly
one place.

**DD-4 — No byte-form change and no version bump (FR-7).** A sack already carries a list of codes
and an entity already carries twelve slots; this story changes which values occur, not what the
form can express. The version byte is untouched, and no version-shaped test is re-spelled.

**DD-5 — The pinning test turns over rather than being deleted (FR-1, FR-3).** The test that
asserts the ten armour slots stay on the corpse is replaced, in place, by its inverse: same fixture,
opposite expectation, and a doc block naming what changed. Deleting it would leave the widening
unpinned in exactly the file that pinned it.

**DD-6 — `missionrun` reports through `pkg/data`'s existing code readers (FR-8).** The seven-digit
name and the slot come from an item code's own accessors; the weapon name comes from the existing
code-to-weapon recovery against the loaded tables. Nothing new is added to `pkg/data`, and no
unexported name in `pkg/game` is exported for it. This adds `pkg/data` to `cmd/missionrun`'s
allow-list in the import-graph check — a planned, named boundary change, in the direction the DAG
already runs.

*Rejected:* exporting `pkg/game`'s existing item namer. It sits in a file another lane holds this
cycle, and the tool needs no more than the two accessors it would wrap.

**DD-7 — `0123`'s contract is corrected, not left to contradict the code (FR-1, FR-3).** Two of its
requirements state that the armour stays on the corpse and that a body wearing only armour plants no
sack. Both become false when this lands, so each gains one sentence naming this story as what
supersedes it. Nothing else of `0123` moves, and no identifier is renumbered.

The authority is the playbook's own S-6 — reconcile a discrepancy **into** the spec, never leave an
out-of-scope item contradicting the code — read with *Revisions*: a material change starts a
revision at the earliest affected stage, and the earliest stage affected by "a corpse keeps its
armour" being false is the artifact that says it. The alternative, leaving the old contract
standing and correcting it only here, is the failure S-6 names: a reader of `docs/0123` would build
against a requirement the code refutes. This story's own spec is what the code is built from; the
edit to `0123` is a pointer, not a second contract.

**DD-8 — The loader's suppression gate keeps its width and loses its reason (spec divergence 2).**
`pkg/mapload` empties the two hand slots of an NPC-templated person at load time, and its doc block
gives as the reason that a corpse gives up only those two. That reason is what this story removes,
so the comment is corrected to say what is now true: the gate is narrower than the drop, and the
consequence is stated where a reader of that file will meet it. **The gate itself does not move** —
widening it would strip a living NPC of the armour he is seen wearing, which is a separate question
from what his corpse leaves, and this story has no mandate for it.

*Rejected:* widening the gate here. It reaches another package, changes what living units wear, and
would fold an unasked decision into a story about the death path. It is named for the seat above to
schedule, not decided in passing.

## Risks

- **The digest of any world where an armoured body dies changes.** That is the story, not a
  hazard, but it means a saved world produced before this lands and stepped after it diverges. No
  save format changes, so nothing refuses to load; the divergence is in what happens next. Accepted
  and stated (FR-7 bounds it to values, not form).
- **A body whose template the original would suppress now drops its armour** — a divergence that
  existed for the container and widens here. Disclosed in the contract; no code guards it.
- **The tool's report is only as good as the tables it was handed.** A code the loaded tables cannot
  name still prints its seven digits and its slot, so a gap is visible rather than silent.

## Success criteria

- **SC-1** — `go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1
  -trimpath ./...` all clean, plus `check-no-game-assets.sh`, `check-doc-budget.sh`,
  `check-sdd-audit.sh` and `check-hotfix-ledger.sh`.
- **SC-2** — the unit tests in `pkg/sim` witness AC-1 to AC-9, each by name.
- **SC-3** — on **both** lawful roots, the tenth mission is driven headlessly and a clubman is
  felled, **twice**: once with the ten-slot loop reverted and once with it in. The criterion is the
  **difference** between the two outputs, not a reading of one — reverted, the boots code is
  reported worn before the blow and absent from the sack; in, the same code is reported worn before
  the blow and present in the sack, with the club. A report that printed the same thing either way
  fails it, and so does one that prints a code the body was never wearing.
- **SC-4** — reverting the ten-slot loop alone turns over the tests that claim it, and nothing else.
  A line is witnessed because removing it fails something, not because an assertion mentions it.

## Traceability

| FR | Design | Criterion |
|---|---|---|
| FR-1 | DD-1, DD-5, DD-7 | SC-2, SC-3, SC-4 |
| FR-2 | DD-1, DD-3 | SC-2, SC-4 |
| FR-3 | DD-2, DD-5, DD-7 | SC-2, SC-4 |
| FR-4 | DD-1 | SC-2 |
| FR-5 | DD-1 | SC-2 |
| FR-6 | DD-1, DD-3 | SC-2 |
| FR-7 | DD-4 | SC-1, SC-2 |
| FR-8 | DD-6 | SC-3 |
