# Provenance — a fight is legible

Every research fact below is cited by **claim id** at the pin recorded in the submodule (research
`414bc29`). No experiment folder is cited: a claim carries its own amendment and retraction state, an
experiment cannot tell you it has been superseded.

**THE PIN MOVED MID-STORY, from `20921e2` to `414bc29`**, on the orchestrator's instruction and as a
disclosed deviation from the freeze that normally holds a pin for a story's length. The freeze exists
so facts do not shift underfoot; it is not there to let a story ship an invention that has since been
decoded, and `EXP-0111` decoded three things this contract had authored. What the bump changed is
recorded in the removed register at the foot of this file, and `verification.md` names both hashes.

## What the contract rests on

| spec anchor | id | confidence | what this story takes from it |
|---|---|---|---|
| FR-1, FR-2 | `MOVE-TURN-031` | High | **Facing is a byte in 32-unit steps, eight directions over 256.** `R1356(actor, nextCell)` returns `dirIndex << 5`, and the desired facing is stored at `mover+0x01` against the current one at `mover+0x00`. That is the field's width, its units and its quantum — not ours. |
| FR-2 | `MOVE-DIR-034` | High | **The eight directions.** `dx = [0,+1,+1,+1,0,-1,-1,-1]`, `dy = [-1,-1,0,+1,+1,+1,0,-1]`, clockwise from north, all in `{-1,0,+1}` — two 8-byte tables written once by the world init, read at 4 and 5 sites with one write each and no orphan. Index 0 is north, and that is what makes the zero value of a facing byte a direction rather than an absence. |
| FR-2 | `MOVE-RATE-029` | High | **The byte-to-direction rounding**, `dir = ((facing + 0x10) >> 5) & 7`, transcribed at `L06859`…`L06860` as the first thing the rate law does with a facing. It is used here for exactly what it is: the total map from any byte onto one of the eight, so a facing this build never writes still names a direction. |
| FR-3 | `MOVE-TURN-031` | High | **A mover faces where it is going**: the step is computed and taken only on equality of the two facing bytes, and the desired one is derived from the **next cell**. So the facing a walk leaves behind is the direction of the step just taken, and it is adjacent by construction. |
| FR-8 | `TERR-SPR-047` | High for the switch and the arms / Medium for the state names | **There is an attack animation state and it is one arm.** The drawn frame comes from a nine-entry switch on `unit+0x74` with six live arms; states 3, 7 and 8 share **one attack arm**, the game's own command strings distinguishing `Attack`, `Shoot` and `Cast`. This is what establishes that a swing is a drawn state of its own rather than a variation of the move block. |
| FR-7 | `SPR256-UNIT-024` | High | **The attack block's place in the sheet**: base `S + D*(MB+MV)`, length `D*AT`, one direction slot of `AT` frames. Already implemented for the move, dying and tail blocks by `0024`; this story adds no arithmetic to it, it reads the slot length the same expression already computes a base from. |
| Out-of-Scope | `AI-RETAL-056` | High for the hook and the arm / Medium for "exactly one reader" | **Refutes retaliation-as-an-order**, and is cited here for what it forbids. A blow sets `ord+0x54`; its single consumer is the idle-turn arm, where the flag skips a `rand()` gate. The facing that arm then picks is `mover+0x00 + 0x21` — one step from where the victim already faced, **not** the attacker's bearing. It writes no `actor+0x5c`, no `ord+0x08` and no `actor+0x50`. |
| Out-of-Scope | `AI-STRIKE-055` | Medium | Being struck is **not** among the causes that make a unit attempt a blow; ordinary acquisition is. Cited to fix that the enemy's reply is an acquisition story and not a rule that hangs off the blow. |
| FR-4 | `AI-FACE-067` | High for the routine and the caller set / Medium for "none is reached from them" | **The complete turn-to-face producer set is six routines and not one is on the swing path.** `R0056` is the only routine in the image that aims an actor; its `callto:` is 7 hits / 6 owners / 0 orphan, and none of the six is the swing start, the strike or the actor's own tick. One of the six IS the approach's stand-and-face. This is what put the turn on the pursuit's stop arm and took it off the attack cycle, and the claim states the failure in terms: a consumer that turns the attacker as part of the swing has invented a coupling the original does not have. |
| FR-4, FR-5 | `AI-FACE-066` | High for the two guard sets and the test / Medium for *every route* | **Facing is a PRECONDITION of an attack and never a consequence of one.** The swing start and the strike are read end to end and their complete guard sets contain no facing test, no facing write and no turn call; the test lives one level up, where the direction from self to target must equal the current facing before the distance is even compared. It is re-run every tick, so a victim that circles its attacker drops it back to the arm that turns and stands. This settles the question FR-5 was written around, and it is why the un-built gate is now a **named divergence with a citation** rather than an open item. |
| FR-8, FR-11 | `ANIM-RUN-004` | High | **An attack run's length is `len(AttackAnimTime expansion)` ticks** — the art's, not the duration the simulation ships, which neither client arm reads at all. This is the number the swing plays for, and it is why the selection refuses past the track's own length instead of looping. |
| FR-8, FR-11 | `ANIM-PHASE-003` | High | **The attack arm's clock is `+1` per tick from zero and indexes with NO modulo** — the move arm is the only one of the switch that takes one. The authored modulus is gone because of this row. |
| FR-8, FR-11 | `ANIM-STATE-023` | High | A swing is action code 3 at reach 1, it is **entered by the message that starts it** and **left when the run counter reaches 0 and the state is forced to 0**. That is the restart and the ending: the clock is zeroed at the swing start and the drawing falls through once the run is spent. |
| FR-8 | `ANIM-CLOCK-024` | High for the two clocks / Medium for the histograms | **The swing frame, the swing sound and the damage are scheduled from three different numbers**, and binding any two reproduces the original on at most 2 of the 157 shipped pairs that answer. The independence this story ships is therefore fidelity, not a shortfall — which is the opposite of what the first draft called it. |

## What is OURS by choice, and where the seam is

- **The rotation between the two direction orderings.** `MOVE-DIR-034` numbers directions clockwise
  from north; the drawing tier's octants, shipped since `0024`, run S=0 … SE=7. Nothing in research
  states the relation. Ours is `octant = (dir + 4) mod 8`, and it is not a chosen constant: it is the
  one rotation that maps each of the eight deltas onto the octant the shipped derivation already
  gives that same delta, which is what `AC-9` asserts over all eight rather than over a sample.
- **A facing gates nothing HERE.** No blow, no step and no search reads the field. That it should
  gate a blow is no longer ours and no longer open — `AI-FACE-066` settles it as a precondition — but
  the site it is tested at is the order machine's act-state entry, and this tree has no act-state
  machine. What is ours is the decision not to invent that site in order to host the rule. Read by
  nothing, the field stands on the class key's and the owner slot's footing, and the seam is
  `Entity.Facing`'s own doc block.
- **The turn is instant and costs no tick.** `MOVE-TURN-031`'s small-arc arm snaps a facing for
  free; its large-arc arm costs `ceil(arc / RotationSpeed)` ticks and destroys the route. This build
  reproduces the free arm for every arc. The divergence is named in `spec.md` (FR-5) and it is a
  divergence of **duration**, not of direction: the facing reached is the facing that arm reaches.
- **Which drawn state a swing is.** Ours: an entity holding a victim that did not move this tick.
  The engine holds ONE action code at a time and this tree derives two independent readings, so an
  ordering between them is ours whatever the codes are; the state byte itself is not modelled here.
- **Which tick of OUR cycle is the swing start.** The run's length, its indexing and its ending are
  decoded; what is ours is the mapping onto this tree's own three-phase cycle, where the run is
  restarted at the transition into the charging phase. That is the tick our cycle loads its countdown
  on, which is the instant the engine's own swing start sends its message — but our phases are ours
  (their numbering is already declared ours in `combat.go`), so the correspondence is engineering.
- **A never-turned unit faces north.** The map's placed unit record carries no facing — the decoded
  `alm.Unit` is position, two class keys, flags, a definition id, an owner and two script ids — so a
  placement states none. North is the direction table's own index 0 rather than a choice among
  eight.

## What is OPEN, and deliberately assigned no meaning

- **`RotationSpeed` as a per-class column.** Decoded, named, and read by nothing in this tree. The
  `pkg/data` unit definition already carries the field; no entity is given one.
- **The facing precondition on a blow.** Decoded (`AI-FACE-066`) and not built: the gate sits in the
  order machine's act-state entry and this tree has no act-state machine. Open as a divergence with a
  citation, which is a different thing from an undecoded question.
- **The two sounds and the numeral a landed blow emits.** Decoded by the same experiment and
  allocated elsewhere; nothing here asserts anything about either.

## What was REMOVED, and why

- **"A struck unit turns toward its attacker."** Drafted from a summary of `AI-RETAL-056` and cut
  after the claim was read whole: the arm turns the victim one direction step from where it already
  faced, and the attacker's cell reaches only the sight stamp. Keeping it would have shipped an
  invention under a High citation.
- **A turn-in-progress pair on the entity.** Drafted to carry `MOVE-TURN-031`'s `mover+0xa0`/`+0xa4`
  so the turn could cost ticks later without a second form change. Cut: two hashed fields no tick
  reads and no test can exercise are exactly what the field-set pin exists to refuse.

### Removed by the mid-story pin bump

Three things this story had AUTHORED were decoded by `EXP-0111` and replaced. Each is listed with
what it was, because the point of a seam is that it can be found again — and all three were found
where the seam said they would be.

- **The attacker's turn inside the attack cycle.** Authored at the top of the attacker's own turn,
  guarded by the reach test. `AI-FACE-067` refutes the placement outright. Moved to the pursuit's
  stop arm, which is the original's own stand-and-face, and the attack cycle now holds no facing code.
- **The swing clock's modulus.** Authored as a euclidean reduction over the track's period, because
  the arm's arithmetic was transcribed nowhere. `ANIM-RUN-004` and `ANIM-PHASE-003` give the run its
  length and remove the modulus; the run now plays once and ends.
- **"The swing/blow desynchronisation is a divergence."** Authored as a disclosed shortfall.
  `ANIM-CLOCK-024` makes it the *faithful* answer: the three events are scheduled from three numbers
  and binding any two would have been the divergence. The behaviour did not change; what it is called
  did, which is the correction that mattered most.
