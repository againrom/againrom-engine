# Provenance — 0095, the frozen guard radius

Research pin: submodule `research/` at `03d3ac2`. Every row below was read through
`go run ./tools/claim <ID>` at that pin, never out of a ledger by hand.

## What the story rests on

| Claim | Confidence | What it gives this story |
|---|---|---|
| `AI-RADFREEZE-075` | High | FR-3 and FR-4. The per-tick geometry pass writes `grpAI+0x2c` and it is never read again; what the arm rolls from is `grpAI+0x38`, whose three writers are all inside the guard setter `R0125`, which runs at map load, on the player's guard command and on the script's `Par0 = 1`. So the radius is a load-time constant, and a group that spreads out, loses members or crosses the map keeps it. It amends `AI-RADIUS-014`, whose Medium rested on there being one producer. |
| `AI-RADIUS-014` | High for the arithmetic, Medium for the roll's cadence | FR-2, in full. The geometry is the centroid at `grpAI+0x28`, the max spread at `+0x2a`, the max member sight at `+0x2b`, and `+0x2c` = **max over members of (that member's distance from the centroid + that member's own sight)**. The setter then **installs** `grpAI+0x2d = grpAI+0x38 = grpAI+0x2c`, raising both to the caller's override or, when that override is 0, to `[Scanning] MinimalGuardRange`. Shipped `World\Data\ai.reg` sets that to **8** against the code default 10, so the file's value is the only floor ever reached. |
| `AI-GUARD-021` | High for the branches, Medium for the cadence | The margin and the roll. `grpAI+0x2d = grpAI+0x38 + 4 + r`, `r` in {-1,0,1}, once, on the tick the has-members latch `grpAI+0x30` flips; the empty-group edge rolls the same expression **without** the `+4`. This build has no latch, so the `+4` stands and `r` is taken at the roll's midpoint — 0086 D-4, restated as D-2 because the base it is added to is now stored rather than recomputed. |
| `AI-GRPGUARD-074` | High | That the clip's origin is the group's **centroid**, `grpAI+0x28`, read fresh every guard tick — so the circle follows the group and only its radius is frozen. Also the walk-home target this story does not build (D-4). |
| `AI-GROUP-009` | High for the mechanism | That a group is formed by equality of the map's own type-6 group id **within one player**, which is what makes `(owner, group)` the record's key rather than the word alone. Its corpus figure — 2264 groups over 38 maps, mean 3.58 members, 686 singletons — is what sizes the section. |
| `AI-ROAM-025` | High | The negative keeping Roam out of scope: order `0x11` is the only genuine wander in the image and `Par0 = 17` ships 0 nodes over 38 maps. |
| `AI-POST-042` | High | Why the walk home is out of scope, and it is **not** the reason this story was handed. `ord+0x00` is the post; it is emergent rather than authored, set to the actor's own cell the first time the initialiser runs on a zero — so for a placed creature it is **its spawn cell**, not the map's origin. `AI-GRPGUARD-074`'s "nothing has ever written it" means no *group-order* path writes it; the per-actor initialiser does. A faithful walk home therefore sends an idle guard back to where the map put it, which is a behaviour worth having and needs a per-actor field this tree has not got. |
| `AI-REISSUE-077` | High for the mechanism | That there is no break-off under either implemented group order — the arm re-issues unconditionally — so a frozen circle releases nothing already acquired, and narrowing one cannot undo a fight that has started. |

## What is ours by choice

- **The record's key set is every owned entity, living or not.** The law's group object outlives
  its members and a felled member is unlinked from it. Nothing in this tree appends an entity to a
  world, so keying on the whole placed set makes the record set a constant of the world rather than
  something a death has to maintain — and it keeps a group whose members have all fallen
  representable, which the byte form otherwise could not write down.
- **The record's own width.** One byte for the base, because `grpAI+0x38` is what a byte compare
  reads and because the existing computation already narrows each summand to a byte before taking
  the maximum. The two key words are carried at the placed record's 32-bit width, matching the
  entity's `Group` and `Owner` beside them.
- **Where the section sits in the byte form** — between the routes and the script — and that the
  decoder checks its key set against the form's own entities rather than trusting it.

## Open — named, not filled

- **Whether the load-time install sees a computed geometry.** `AI-RADIUS-014` fixes the install as
  `grpAI+0x38 = grpAI+0x2c`, and `AI-RADFREEZE-075` fixes the only per-tick producer of `+0x2c` as
  the guard arm's first act. Whether that arm has run before the load-time call is not established,
  so a reading under which every group's load base is the bare floor is not excluded by anything
  read here. This build takes the geometry, which is what both rows' own statements describe. The
  difference over the shipped corpus is measured in `verification.md`: 235 groups against 141.
- **The setter's caller-supplied override.** It raises the base above the geometry, and no path in
  this tree supplies one — there is no guard command and no group-order script command — so the
  override is 0 wherever this build can ask, and only the floor arm is reachable (D-1).

## Pending at this pin — named, and not implemented from

`EXP-0125` closed after this submodule pin (`03d3ac2`) and its rows — `AI-POST-095`, `AI-POST-096`,
`AI-JITTER-103`, `AI-RANGE-102`, `AI-CENTRE-101` — are **not readable here**. Two of them bear on
this story and both were reported to this lane by the orchestrator rather than read at a pin, so
neither is built on:

- **The jitter's divisor.** The `+/-1` this story does not implement (D-2) is stated from
  `AI-RADIUS-014` and `AI-GUARD-021` **at this pin**, both of which already carry the expression
  and the `0x8000` divisor with its writer at `L00371`. Nothing here rests on the newer row, and
  no number in this build changed for it — the roll is absent either way.
- **A re-issued guard moves the post,** which refutes `AI-POST-042`'s "for a guard it never moves".
  This story neither models the post nor the re-issue, and the clause it does use — that the post
  is the spawn cell at map load — survives the refutation. Recorded so the next story starts from
  the amended row rather than from the one above.

The pin is the orchestrator's to bump at the story boundary.

## Removed from scope while writing

- Reproducing the has-members latch and its ±1 roll. It is group state, and this story adds the
  record it would live on, so it became reachable — and it was left out because it is a *second*
  behaviour with its own open cadence (Medium in two rows), and a story that adds a record should
  not also spend the first thing the record makes possible.
