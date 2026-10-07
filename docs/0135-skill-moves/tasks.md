# 0135-skill-moves — tasks

Kinds: **impl** (one commit, trailer `SDD-Task: 0135-skill-moves/T<n>`).
`verification.md` and the build are stages, not entries here.

---

## T1 — impl — the level becomes state `pkg/sim` carries

Files: `pkg/sim/world.go`, `pkg/sim/binary.go`, a new `pkg/sim/skill.go`,
`pkg/sim/binary_test.go`, `pkg/sim/nostate_test.go` and whichever pinned tests
the width change moves.

Covers FR-1, FR-2, P-2. Carries DD-1, DD-2.

- `Entity.Skill [skillSlots]int32`, doc'd on `Mind`'s carried-whole rule.
- `skill.go`: a 101-entry `int32` table of `S(n)`, generated once from
  `trunc((1.1ⁿ − 1) × 1000)` and pasted as literals, plus one accessor that
  answers rather than panics outside `[0, 100]`. Three checkpoints to generate
  against: `S(0) = 0`, `S(10) = 1593`, `S(100) = 13779612`.
- Byte form: `formatVersion` **39** — allocated, do not mint another. Six
  `int32` after `KnownSpells`; the record widens 187 → 211. Update the layout
  comment block and the width prose beside it.
- Re-pin every digest and byte pin the width moves, and add the
  pre-story-pin-plus-the-block test the package writes at every such crossing.

Fence: no feed, no raise, no reader outside this package. Nothing seeds the
field yet and that is correct for this commit.

Done when: the field round-trips, two worlds differing only in one level hash
differently, the previous version is refused, and the whole suite is green.

---

## T2 — impl — every producer of a level writes one, and every reader reads it

Files: `pkg/data/recompute.go`, `pkg/mapload/fromalm.go`, `pkg/mapload/start.go`,
`pkg/mapload/carry.go`, `pkg/game/world.go` (the entity-draw overlay only),
`cmd/missionrun/main.go`, plus tests.

Covers FR-3, FR-4, FR-14, P-1. Carries DD-8.

- `pkg/data`: export the forward curve as `SkillXPFor(level int32) int32`,
  called from `Recompute`'s experience step and from `SkillLevelFor`.
- `spawnBlock` gains the six levels, off the same `Recompute` its combat block
  already comes off; the placement literal carries them.
- The party mint writes the member's six levels beside `SkillXP`.
- `CarryParty` writes `Hero.Skill` from the entity's **stored** levels instead
  of from `SkillLevelFor`, and still carries `SkillXP` exactly.
- The draw overlay reads the stored level, not `SkillLevelFor`; the experience
  total is unchanged.
- P-1's test lives in `pkg/mapload`, the lowest package importing both.
- `missionrun` prints each party slot's six levels before and after a drive.

Fence: no award, no raise, no announcement.

Done when: a placed person and a minted member both start at their row's
levels with `xp == S(level)` per slot, a carried member arrives at exactly the
levels and experiences he left with, P-1 passes, and `missionrun -mission 10`
prints the levels on both lawful roots.

---

## T3 — impl — the award, its gates, and the two feeds

Files: `pkg/sim/skill.go`, `pkg/sim/combat.go`, `pkg/sim/spell.go`, tests.

Covers FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, P-3. Carries DD-3, DD-4,
DD-5.

- One sink, `(*World).awardSkill(ai int, named int32, amount int64, srcIdx
  int) bool`, returning whether a level moved. `srcIdx < 0` means no source.
- `payExperience` keeps its own two caller refusals and its `xpRaw`, and hands
  the raw amount to the sink naming slot 0. The three refusals FR-5 now owns —
  the gains flag, the owner slot, the locked relation — move into the sink and
  are not written twice.
- The cast feed is the last act of an applied cast, naming the spell's school,
  with the victim as the source.
- `spellPower` gains the school-level term.

Fence: no kill feed. No new door on `pkg/sim`, no float, no new import.

Done when: FR-5's four refusals, FR-6's two steps, FR-7's two arms and FR-8's
strict test each have a test that fails when that statement alone is reverted;
`AC-1`, `AC-3`, `AC-4`, `AC-5`, `AC-6`, `AC-7`, `AC-8`, `P-3` are witnessed.

---

## T4 — impl — the derive follows the raise, and the raise says so

Files: `pkg/game/world.go` (the per-frame refresh and the pick-up post),
`pkg/game/equip_test.go`, `cmd/missionrun/main.go`, tests.

Covers FR-12, FR-13, AC-10. Carries DD-6, DD-7.

- The per-frame refresh gains a second trigger beside the equipment one: the
  inventory subject's own six levels. One recompute serves both, seeded with
  the levels read off the entity, and pushed through the one existing door.
- The announcement: per known character, the six levels last posted against the
  live ones; each slot that rose posts one row, text `"<skill> <level>"`, count
  1. The skill's name comes from the mage set for an entity with a mana pool
  and the warrior set otherwise.

- One `pkg/game` test pins the pre-raise absence and is now false; correct it
  in place, keeping its name, to state what the raise actually does.
- `missionrun` gains one flag for how long to keep stepping after the last
  waypoint, defaulting to what it does today so no existing invocation moves.

Fence: no new overlay, no new per-frame hook in `pkg/ui`, no widening of the
derive's door.

Done when: a raise moves the subject's damage and to-hit through the existing
door; a tick with no raise posts nothing; AC-9 is witnessed; and the drive on
both lawful roots prints a level moving (AC-10).
