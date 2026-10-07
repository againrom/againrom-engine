# Verification — a fight is legible

Environment: Windows 11, Go as pinned in `go.mod`, branch `impl/0081-attack-legible` rebased onto
`origin/master` **`e209e95`**. Research submodule `414bc29`. Every figure below was produced after
that rebase.

Master moved twice under this branch and it was rebased onto each: `2a15d03` (0079, 0080 and the pin
bump) and then `e209e95` (0082). The second rebase conflicted in `pkg/game/world.go`, where both
stories add fields to one struct and one changes a constructor's signature; both sides were kept and
the field this story deletes was dropped from the literal. The whole gate was re-run after each
rebase rather than only before the first, and the figures below are the second run's.

**A DISCLOSED DEVIATION FROM S-5's PIN FREEZE.** This story began at research `20921e2` and was
finished at `414bc29`. The bump was made mid-story on the orchestrator's instruction, because
`EXP-0111` decoded three things the contract had AUTHORED. What changed as a result is listed under
"What the pin bump moved" below; the freeze exists so facts do not shift underfoot, not so a story
ships an invention that has since been decoded.

## The gate

Run from the worktree root after the rebase. Each command's own exit code, not a pipeline's:

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
(no output)
$ bash scripts/check-no-game-assets.sh   ; echo $?
0
$ bash scripts/check-doc-budget.sh       ; echo $?
0
$ bash scripts/check-sdd-audit.sh        ; echo $?
0
```

`gofmt -l` printed nothing, `go vet` printed nothing, and the suite printed no `FAIL` line. The
audit's note/warning count is not recorded here: this is a worktree, `builds/` is untracked and so
absent from a fresh checkout, and the count is therefore meaningless from this seat. Only the FAIL
set is enforced and it is empty.

## SC-1 — the field, the conversions and the form (AC-1, AC-2, AC-6, P-1)

`FacingDir` is asserted over **all 256 bytes** against the rounding written out a second time in the
test, and lands in `[0,8)` for every one; the eight directions round-trip `dir << 5`; each of the
eight deltas names its direction and the zero delta names none, at magnitudes 1, 3 and 1000 — the
magnitude never enters. The production delta table is checked against a second transcription of
`MOVE-DIR-034` in the test file, so a transposed row fails rather than being agreed with. That is
**AC-2**.

**AC-1**: an entity naming no facing faces north; every one of the 256 values survives a round trip
through the byte form, on a live unit and on a felled one; and two worlds differing only in one
facing hash differently and marshal to different bytes. **P-1**: `pkg/sim` still imports the standard
library alone and names no float — the package's own import and determinism checks are part of the
green suite above, and the facing arithmetic is integer throughout.

The byte form: version **14**, record **91 → 92 bytes**, the facing at the record's tail. Version 13
is refused, and so is every other byte — the sweep runs all 256. With the record width and the
version byte, that is **AC-6**, whose remaining clause — a felled unit keeps the facing it died with
across the blow, the constructor and a round trip — is asserted in three places.

**The two pinned worlds and their digests moved, and each moved for a reason that was checked rather
than accepted.** A pin and a digest computed from that pin agree by construction, so both were
re-derived backwards onto a literal that predates the story: `strippedOfTheFacing` removes exactly one
byte per record and puts version 13 back, and the result must hash to the digest the tree carried
before. It does, for both fixtures, in `pkg/sim` and again in `pkg/mapload`.

```
pkg/sim   pinDigest  0x2a12cf3dcfcd82fc -> 0xcc2e3f970c53ff2e   (strip -> 0x2a12cf3dcfcd82fc)
pkg/sim   rtfDigest  0xea98475193d1cb4c -> 0x26357be9f95bb0d4   (strip -> 0xea98475193d1cb4c)
mapload   gfDigest   0x1c835cfb072e60d1 -> 0xdba36943468cc4e2   (strip -> 0x1c835cfb072e60d1)
```

**Two digests were re-derived OUTSIDE this tree, as their own doc comments require.** The relaxation
fixtures' tick-1 digests are assembled from the layout description by a program written in another
language that shares no code with the encoder, and hashed there:

```
rlx len 1642 want 1642
rlxTick1Digest  = 0x2e6cc59a440729ab
hyb len 1799 want 1799
hybTick1Digest  = 0x34d011a1cb394bef
```

Both equal what the Go encoder produces. That is two independent implementations of one layout
agreeing, which is the whole point of those two constants.

**A limitation, stated:** the external assembler was written for this story and is not preserved in
the repo, so the derivation is reproducible from the doc comment and this transcript rather than by
re-running a checked-in tool. That is the same footing the constants stood on before.

## SC-2 — a mover faces the cell it stepped to (AC-3)

Asserted in all eight directions, one order per direction, on the entity the world hands back. The
four ways a tick can leave a unit where it was are each a separate case — ordered nowhere, walled in
through the give-up limit, felled, and paying for a crossing — and each keeps the facing it held,
which is what pins that no site other than the step writes one. That is **AC-3**.

The felled case is the sharp one: a blow clears four fields beside this one, and the body keeps the
facing it died with.

## SC-3 — the attacker's turn, and that a facing gates nothing (AC-4, AC-5)

An attacker faces an adjacent victim from **all eight relative positions**, each starting from the
opposite direction so a case that turned nothing would fail rather than pass on its initial value.
An attacker whose victim is far away is checked on **every tick of the walk**: its facing is the step
it just took, never the victim's bearing — the fixture places the victim eight cells east and one
south precisely so the two differ. A victim on the attacker's own cell leaves the facing alone. An
order whose victim died this tick turns nothing. Together those are **AC-4**.

**AC-5, the property FR-5 exists for**: two worlds differing only in one unit's facing, advanced
through one 60-tick schedule, compared every tick on the whole entity slice with the facings masked
out and on both stored routes. They never differ. The victim takes damage during it, so the case is
measuring a fight and not two idle worlds. Their digests DO differ at the start, because the field is
canonical — and converge as the schedule turns both attackers the same way, which is the story
working rather than the field failing to reach the form.

## SC-4, SC-5 — the descriptor and the swing selection (AC-7, AC-8, P-2)

The attack slot, track and gate are what a class's keys imply, over six shapes including the
clamped-at-zero absent phase, the empty-track gate failure and two arrays of unequal length. The
block's extent is asserted as a RELATION — `AttackBase + D*AttackSlot == DyingBase` — over four
classes, so the slot is the same scalar the base beside it was computed from and not a coincidence of
literals.

The selection returns the block's own frames for every octant and every tick of the run; refuses at
and past the run's length; refuses for no block, a negative slot, a failed gate, an empty track, an
index past the sheet's frames, a zero frame count and a base before the sheet; and is total over a
negative and a huge clock. The mirroring layout's fold is written out as eight cases rather than read
off the production rule. That is **AC-8**, and its totality half is **P-2**'s first clause: no input
panics and none divides by zero.

The two tiers' descriptors are compared field for field by reflection, over a class whose every
derived field is pairwise distinct — a property this story had to RESTORE, because the fixture set an
attack phase count and no attack arrays, so the new track and gate would have mirrored nil onto nil
and false onto false and passed for exactly the two fields being added. With the six key shapes
above, that is **AC-7**.

## SC-6 — the seam (AC-9, AC-10, AC-11, P-2, P-3, P-4)

**AC-9, the rotation.** `sheetOctant` is asserted over all eight deltas against this package's own
independent transcription of the sheet ordering, and the facing it is given is one a real world
walked away with — one unit, one order, one tick — rather than one built through any conversion. All
eight agree. The assertion landed in the same commit that deleted the derivation it replaces. That
is **AC-9**, and with the deletion of `mw.facing`, `signOctant`, `signOctants` and `signIndex` it is
**P-3**: the direction index, the sheet octant and the mirror bit are computed per call and stored
nowhere, and no second remembered facing exists in the tree.

**AC-10 and AC-11.** An attacker beside its victim draws the attack block for exactly the run's
length and its idle drawing on every tick after it; the count of attack frames drawn is asserted, so
a build that looped the swing would fail rather than pass. A class with no attack block falls through
to the live selection on every tick. A corpse draws no frame of the attack block at all. A schedule
run with two pushes per tick and a schedule run with none produce the same digest. Those are
**AC-10** and **AC-11**, and the digest half is **P-4**: neither the swing clock nor the phase memory
is in the world, the byte form or the digest. **P-2**'s second clause — the three selections are
jointly total and ordered — is the dispatcher's own shape: death first, swing second, live last, each
refusal falling through, so every entity reaches exactly one drawing.

**One visible change, disclosed rather than absorbed.** A unit that has never turned now draws NORTH
where it drew SOUTH. Three shipped assertions pinned the old default and all three were REWRITTEN
rather than deleted — `pkg/game/facing_test.go`'s never-moved case and two frame tables in
`pkg/game/world_test.go` and `pkg/game/death_test.go`.

## SC-7 — measured against both lawful roots (AC-12)

`missionrun` reports the direction an attacker finished facing. Mission 10, EN and RU:

```
$ missionrun -assets ..\gameversions\en --mission 10 -ticks 4000 \
    -waypoint u28:79:79:200 -waypoint u29:79:79:200 -attack u28:u29
waypoint 1  u28 -> (79,79) r200 : reached (69,51), Chebyshev 28, after 1 ticks
waypoint 2  u29 -> (79,79) r200 : reached (64,46), Chebyshev 33, after 1 ticks
attack 1  u28 -> u29 : FELLED it after 128 ticks, victim at -1 hp, attacker facing NW
outcome undecided at tick 194
```

RU prints the same four lines, byte for byte. `u28` stands at (69,51) and `u29` at (64,46): a delta
of (-5,-5), which is **north-west**, and north-west is what the attacker finished facing. The victim
is at -1 hp, so a blow landed and the fight is real rather than a walk that timed out. That is
**AC-12**, on both roots.

**RE-MEASURED after the rebase onto `e209e95`, and unchanged in all four lines.** That story gave the
hero a derived movement speed, which moves every timing measured on a party member; this fixture
names two SCRIPT units and no party member at all, so its 128 ticks are the same 128. The
re-measurement was made rather than argued, because "my fixture should not be affected" is exactly
the belief a re-run is cheap enough to check.

**An assumption this story does NOT make**, named because a defect in the neighbourhood is being
looked at: nothing here decides or assumes WHICH class a party member resolves to. The swing
selection reads the descriptor of whatever class an entity's own id resolved to, the facing is the
simulation's own field, and the fixture above touches no party member. A unit drawn as the wrong
class would draw the wrong class's swing, and that would be the resolution's defect, not this
selection's.

**A false positive found and discarded, recorded because it nearly became the evidence.** The first
run used `-attack p0:p1` and reported `FELLED it after 1 ticks, victim at 0 hp`. It is not a kill:
`hpOf` answers 0 for an entity the world does not hold, and the party on this mission holds no second
member. The pair above was chosen instead by reading every resolvable script unit's cell and taking
the closest two.

## SC-8 — nothing else moved (P-5)

The suite is green in every package, which includes every digest, byte-form, cadence, order- and
draw-invariance pin the tree carries. `pkg/ui` gained no field and still names no simulation type. The gate at the head of this file is
**P-5**, and it is clean.

## What the pin bump moved

Three things this story had authored were replaced by decoded ones. All three were found where the
seam said they would be, which is the only claim the seam ever made:

| authored | replaced by | what changed |
|---|---|---|
| the turn at the top of the attack cycle | `AI-FACE-066`, `AI-FACE-067` | moved to the pursuit's stop arm; the strike now holds no facing code, and the turn no longer reaches a mover mid-crossing |
| the swing clock reduced modulo the track | `ANIM-RUN-004`, `ANIM-PHASE-003`, `ANIM-STATE-023` | the run is the art's own length, indexed one per tick with no modulo, and it ends |
| "the swing/blow desynchronisation is a divergence" | `ANIM-CLOCK-024` | it is the FAITHFUL answer; binding the two would have been the divergence. The behaviour did not change, only what it is called |

`AI-FACE-066` also settles the question FR-5 was written around: facing IS a precondition of a blow.
It is still not built, and the reason is now specific rather than provisional — the gate's site is the
order machine's act-state entry, and this tree has no act-state machine.

## Limitations

- **The blow is not gated on facing**, and **a turn costs no tick and destroys no route**. Both are
  decoded and both are disclosed (FR-5). Neither is measured here, because neither is built.
- **The swing does not coincide with the blow.** Reproduced deliberately (`ANIM-CLOCK-024`); the two
  are scheduled from different numbers in the original.
- **The drawn result was not looked at on a screen by this lane.** Everything above is a test or a
  headless run. What a fight LOOKS like is the owner's to judge, and `builds/0081-attack-legible/`
  exists for that.
- **`swinging` does not ask whether the victim is in reach**, so an attacker standing still and out of
  reach draws a swing. Named in the plan (DD-12) and not fixed here: the reach is a simulation law and
  a copy of it in the drawing tier is what this tree already refuses elsewhere.
