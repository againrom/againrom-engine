# Plan — a placed unit's combat numbers

## Approach

The whole change is one seam: the loop in `pkg/mapload` that turns one placement record into one
`sim.Entity`. That loop already resolves a `data.UnitDef`, already applies the difficulty to it, and
already reads three things off it — the health maximum, the rate and the movement column. Every one
of the eight values FR-1 asks for is a field of that same resolved definition, and every one
has a counterpart field on `sim.Entity` that a tick already reads. So the design is not "carry a new
number from a file to a simulation"; it is "stop dropping eight fields that both ends already have",
and the whole of it lands inside one composite literal and its unresolved fallback. FR-2's damage
pair and mark arrive with them, already built by the tier that decodes the row.

The one real decision is what that fallback is, and the contract's constraint table settles it: an
unresolved placement takes the **whole** constructor definition rather than eight separate defaults,
which is FR-4's eight.
That turns the fallback from a growing list of named constants into a single value with a single
source, and it removes an inconsistency the current code already carries (see DD-3).

## Facts verified during planning

Baseline, frozen at planning time.

1. `sim.Entity` already declares all eight — the two cadence numbers, to-hit, defence, absorption,
   the damage pair and the always-hits mark. All eight are canonical: they enter the byte form and
   the digest, and a tick reads them. The record is at form version 10 and this story adds no field,
   so **no version bump and no offset move is possible** — FR-6 discharged by the shape of the
   change rather than by a check.
2. `data.UnitDef` already decodes all eight from the definition row, including FR-2's routing switch
   that builds the damage pair and sets the mark, and including the refusal of the two arms this
   tree does not model.
3. The difficulty adjustment already adds its constant to the definition's to-hit and defence and
   already scales the health maximum. It touches none of the other six. **It is applied once,
   inside the resolution, before anything is built** — so no change is needed to satisfy FR-3, only
   a test that can fail if one of the six ever moves.
4. The entity loop currently reads `def.HealthMax` and `def.Speed` **only when the placement
   resolved**, and reads the movement column off the definition **unconditionally** — which means
   the unresolved arm today hands the movement lookup a *zero* definition while handing the rate a
   *constructor-defaults* definition. Two different meanings of "unresolved" in one statement.
5. The party-placement path builds its own entities in a second composite literal, with the spawn
   health and the default rate spelled there by name. It is the second site that has to agree with
   the fallback.
6. Two world digests are pinned by hand in `pkg/mapload`'s tests — one for a table-free load and
   one for a load with a table. Both carry a written history of their earlier values and the
   derivation that produced each. Both move.
7. `pkg/sim` holds pinned digests of its own. **None of them moves**: every fixture there is a
   hand-built world and no hand-built world's fields change.
8. The attack cycle advances only for an entity that holds an attack target, and nothing in this
   tree gives a loaded map's entity one. So every value this story writes is, today, observable in
   the byte form and the digest and **nowhere else** — and FR-7's "nothing else moves them" is
   true of the tree as it stands, which is what makes it a witness rather than a hope.

## Files to touch

| path | intent | what |
|---|---|---|
| `pkg/mapload/fromalm.go` | MODIFY | the unresolved definition (DD-1); the eight added to the entity literal (DD-2); `FromALMWith`'s contract doc, which currently says which three values a table contributes; `DefaultSpeed`'s doc, which is now one field of a value taken whole |
| `pkg/mapload/spawn.go` | MODIFY | the movement-column helper's doc only. Its paragraph explaining that an unresolved placement arrives as a *zero* definition stops being true (DD-3); the code is unchanged and the answer it gives is unchanged |
| `pkg/mapload/start.go` | MODIFY | the party entity literal takes the same eight (DD-4) |
| `pkg/mapload/combat_test.go` | ADD | the per-field, per-arm and per-difficulty witnesses |
| `pkg/mapload/fromalm_test.go` | MODIFY | the two pinned digests and the zeroing re-derivation |
| `pkg/mapload/start_test.go` | MODIFY | the party/placement agreement witness |

Nothing under `pkg/sim`, `pkg/data` or `pkg/formats` is touched. If a change appears to be needed
there, the design is wrong — stop.

## Design decisions

**DD-1 — the unresolved arm substitutes the whole constructor definition, once, before anything is
read off it.** *Rejected:* eight package-level defaults beside the existing rate default. That is the
shape the current code is already in for one field, and extending it to nine means nine names, nine
docs and nine chances for the movement lookup and the rate to disagree about what "unresolved"
means again. The substitution also makes the existing rate default fall out as one field of the same
value rather than as a second statement of it.

**DD-2 — the eight are written in the same composite literal as the health, rate and domain.** There
is no helper that fills "the combat fields" and no second statement. *Rejected:* a `applyCombat(e,
def)` mutator, which reads better and is exactly what makes P-1 unenforceable — a mutator has a call
site that can be forgotten, and a literal field that is absent is a field the reader can see is
absent.

**DD-3 — the movement-column lookup keeps its code and loses its explanation.** It is total over
every column value, so the constructor definition (column 1, ground) and the zero definition
(column 0, no code, default arm) give the **same** answer and no behaviour moves. What must change
is the doc: it currently argues the unresolved case is carried *because* the zero definition's
column is 0, and after DD-1 that premise is false while the conclusion stands. *Rejected:* leaving
the doc alone — a rule whose conclusion is right and whose stated reason is wrong survives every
review, because everyone checks the conclusion.

**DD-4 — the party literal reaches the fallback through the same exported source the placement loop
does, not through a copy of the two numbers.** *Rejected:* spelling 8 and 4 in the party literal.
The contract requires the two populations to be identical, and two literals are how they stop being.

**DD-5 — both pinned digests are re-derived by hand and each keeps its superseded value in the note
beside it.** The re-derivation is the contract's own AC-8 recipe: lift the eight fields' bytes out of
every record of the new form, confirm the result hashes to the pre-story pin, and record that. The
new pin is then the old pin plus this story's bytes and nothing else, checked against a literal this
story's own code did not produce. *Rejected:* running the fixture and pasting what comes out. That
regenerates a wrong number as readily as a right one and has no failure mode.

**DD-6 — no new exported constant carries 8 or 4.** They are the definition tier's own constructor
values and are reached through it. *Rejected:* a `DefaultAttackCharge` pair beside the rate default,
which would be a second copy of a number the definition tier owns and would go stale silently.

**DD-7 — the developer-run tool prints the template's numbers and says so in its own output.** It is
not a fidelity check on a class's fighting numbers and must not read as one, because equipment moves
five of the eight on shipped classes and this tree equips nothing. *Rejected:* printing the numbers
with no caveat, which would turn an honest stat dump into a false claim the moment anyone compared
it with a class the game arms.

## Risks

**R-1 — a regenerated pin.** A digest pasted from a run is a pin that cannot fail, and the two here
are the only assertion that this story's bytes are the bytes intended. *Mitigation:* DD-5's zeroing
recipe is a second, independent derivation from a literal that predates this story.

**R-2 — a divergence that was unreachable becomes reachable.** The blow resolution already treats a
damage that is not positive after absorption as removing nothing, and whether the original clamps at
that point was not read. Until now every loaded map's absorption was zero, so the branch could not
be taken on anything but a hand-built world. Filling absorption from a class makes it reachable.
*Mitigation:* nothing about the resolution changes, the divergence is carried forward and restated,
and no test in this story asserts anything about what the original does there. It is a standing
research request, not something to design around.

**R-3 — an equipped class's numbers read as reproduced.** A printed stat block invites the reading
that the tree fights the way the class does. It does not: equipment assigns over the cadence pair,
adds to the damage pair, the absorption and the defence, and moves the reach, on a substantial
minority of shipped classes. *Mitigation:* DD-7, the contract's own out-of-scope entry, and the
owner-run row's explicit statement that the cadence pair is witnessed by nothing.

**R-4 — the two unresolved populations drift.** A later story that changes the placement fallback and
not the party's, or the reverse, reintroduces exactly the split fact 4 above records.
*Mitigation:* DD-4's single source, plus a witness that compares the two **inside one world** rather
than against a constant, so a change to the constant cannot make both sides move together and stay
wrong.

**R-5 — scope creep into reach or equipment.** Both are one small edit away and both look like
finishing the job. *Mitigation:* FR-5 is a requirement rather than an omission, and the reach
constant's own doc already names the story that turns it into a field — which is not this one.

## Success criteria

| | criterion | how |
|---|---|---|
| **SC-1** | a resolved placement carries all eight, each asserted separately, and its health, rate and domain are unmoved | automated — AC-1 |
| **SC-2** | the damage pair is `(min, max−min)` on every admitted routing arm and the mark is set on exactly one | automated — AC-2 |
| **SC-3** | hard moves to-hit and defence by the constant and moves none of the other six; easy moves none of the eight | automated — AC-3, AC-4 |
| **SC-4** | the three unresolved shapes carry 8, 4 and six zeros, and carry the identical eight as each other | automated — AC-5 |
| **SC-5** | every party member's eight equal an unresolved placement's eight in the same world | automated — AC-6 |
| **SC-6** | a table-built world round-trips field for field and by digest, at the unchanged version byte | automated — AC-7 |
| **SC-7** | both pinned digests hold, and zeroing the eight fields' bytes in each new form reproduces its pre-story pin exactly | automated — AC-8 |
| **SC-8** | an unmodelled routing arm is refused by name, no world is returned, and no partial entity exists | automated — AC-9 |
| **SC-9** | the local gate is clean: build, vet, gofmt, the full test suite with no game install, the asset scan, the doc budget and the SDD audit | release-gate |
| **SC-10** | a run against a lawful install prints every placement's eight beside its class name, at each of the three difficulties, and the owner compares the six the game's own panel shows | developer-run verification — AC-10 |
| **SC-11** | the deletion set between this branch's base and its tip is empty | release-gate |
