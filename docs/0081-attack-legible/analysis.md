# Analysis — a fight is legible

## What a fight looks like today

Two units stand still, one health bar falls, and nothing on screen says which unit is doing it. The
attack order landed with its cycle, its roll and its damage; the drawing tier was never told a fight
was happening at all. Four things were reported missing after the first run against a lawful install:
no animation, no damage numbers, no turning toward the enemy, and no reply from the enemy. Two of
them are this story's; the other two are named in Out-of-Scope with the reason each is somebody
else's rather than merely later.

## What we did not know, and what we looked at

**Is there a facing anywhere?** `grep -i facing pkg/sim/` finds prose and no field. `pkg/sim.Entity`
carries a position, a destination, a class, a domain, a rate, a crossing, a group, an owner and
twelve attack fields — and no direction. So nothing in the simulation can turn, and nothing in it
could be asked which way a unit is pointing.

**Then what is the drawn direction today?** `pkg/game/world.go` keeps a per-entity octant memory,
`mw.facing`, written from the cell delta an entity was observed to move by and read by an entity that
did not move. It is the seam's own state, not the world's — deliberately so at the time, because
there was no world field to read. Two consequences we had to check rather than assume: it survives
a round trip through nothing at all (it is rebuilt only by observation), and its never-observed
default is octant 0, which that table calls **south**.

**Is there an attack timeline in the descriptor?** `pkg/render/terrain.UnitAnim` knows `MoveSlot`,
`IdleSlot` and `DyingSlot`, with `MoveTrack`/`IdleTrack` and their gates. `AttackBase` is present and
the file says why: it "rides along so the mirror stays field-for-field". There is no attack slot, no
attack track and no gate — so a swing has a base to be drawn from and nothing to index past it.

**Is the data behind one already decoded?** Yes, and this was the finding that decided the shape.
`pkg/data.UnitClass` already carries `AttackPhases`, `AttackAnimTime` and `AttackAnimFrame`, loaded
through the same key table as their move and idle counterparts, and `Anim()` already computes
`AttackBase` from `AttackPhases` in the same expression that computes the dying and tail bases. The
attack block's arithmetic is therefore not something this story invents; it is one derivation the
descriptor stops one field short of.

**Which of the eight directions is which?** Two orderings exist in the tree and they are not the
same. `MOVE-DIR-034`'s movement tables run clockwise from north — `dx = [0,+1,+1,+1,0,-1,-1,-1]`,
`dy = [-1,-1,0,+1,+1,+1,0,-1]`. The drawing tier's octants, shipped since `0024`, run
S=0, SW=1, W=2, NW=3, N=4, NE=5, E=6, SE=7. Laid side by side they differ by a constant rotation of
four and by nothing else, direction for direction — which is why the translation is one addition and
why a test can assert it against the derivation already shipped rather than against a number we
chose.

**What does being struck do?** `AI-RETAL-056` was read in full rather than summarised, and it does
not say what a summary of it would. A blow sets one flag; the flag's single consumer is the idle-turn
arm, where it skips a `rand()` gate so the actor re-picks a facing on this evaluation instead of on
about one in 160. The facing it picks is `mover+0x01 = mover+0x00 + 0x21` — one direction step from
where it already faced — **not** the attacker's direction. So "a struck unit turns toward its
attacker" is not established by that claim, and building it would have been an invention wearing the
claim's citation.

**What indexes an attack frame?** `TERR-SPR-047` gives the nine-state switch and names states 3, 7
and 8 as one shared attack arm, but the arm's own arithmetic is not transcribed — where the move arm
is quoted down to its `IDIV` and the idle arm down to its lack of one. So the attack track's clock is
undecoded. `EXP-0111` is asking exactly this, together with which state a swing is and whether
facing is a precondition of a blow or a consequence of one.

## What we chose to leave open

The turn's own cost. `MOVE-TURN-031` establishes a second per-unit rate, a `RotationSpeed` column, a
free arc of one direction step, `ceil(arc / RotationSpeed)` ticks for anything larger, and route
destruction as part of the large turn. All of it is movement law and all of it lands in the two files
another story is holding. Reproducing the rate here would have put a second author in the movement
rate law for the sake of a story about drawing; leaving it out costs a named divergence and no
structure.

## Baseline recorded before any change

`go build`, `go vet`, `gofmt -l`, `go test -count=1 -trimpath ./...`, `check-no-game-assets.sh`,
`check-doc-budget.sh` and `check-sdd-audit.sh` all exit 0 at the branch point. The audit reports 35
notes and warnings, none enforced. The byte form stands at version 12 and the entity record at 91
bytes.
