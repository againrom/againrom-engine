# Hotfix archive

**This file is frozen.** Nothing needs to read it — the same precedent as `pipeline/archive/` one
level up: it exists so the record is not lost, not so it is consulted. A later hotfix does not
append here; it writes its own detail in its own commit message and gets its own one line in
`docs/hotfix/LEDGER.md`.

The text below is the ledger's own `Boundary`, `What` and `Owes` cells, verbatim, as they stood at
commit `524d059` — the commit before this story, `0130`, moved them here and cut the ledger's table
down to one line per row. Nothing is summarised, paraphrased, shortened or dropped in the move; the
only alteration is re-wrapping each cell to 100 columns.

Three sentences below carry a dated correction beside them (FR-9): the original stands, and a boxed
note under it says what is now known. Two of the three corrections cite `ITEM-CORPSE-034` and
`ITEM-SUIT-035`, which are published on research `master` **after** this repository's submodule pin
`0908589`. Both are cited here only as corrections to the historical record — no clause folded into
any `spec.md` by this story depends on either of them.

## 9729459

### Boundary

after `0098` landed and the pin went to `e6f9ee6`; `0096` in Phase 4

### What

A damage numeral is gated on the health the viewer **already held**, not the health that just
arrived — so the killing blow still shows and the decay walk's six hundred `-1`s do not.
`ANIM-DEATH-007`: a dead actor's falling health reaches the client as the corpse stage byte and
never as a number.

### Owes

`0083` (the numeral's FR set)

## c5274d7

### Boundary

after `0096` landed; `0099` stopped

### What

**The placeholder walkabout is out of the game.** `mapload.Schedule` — 0020's order script, six laps
of a closed square per unit, staggered 7 ticks per entity — was wired into **both** front-end paths,
so every mob on every map paced a square for ~1000 ticks and then stood still forever. The owner saw
it and reported it. The generator survives only as unexported test support in
`pkg/game/schedule_test.go`, where the digest comparison still needs a command stream that is a pure
function of a map. Both paths pinned against its return.

### Owes

`0020` (`FR-2`, `FR-4`, `DD-2`, `DD-3`)

## 2ad4c29

### Boundary

after `0096` landed; `0099` stopped

### What

**Two debug surfaces stop shipping switched on.** `-markers` (the diagnostic cross on every
structure, unit and static) defaulted ON; the debug readout box was SHOWN. Both are now off in the
game and unchanged in `cmd/mapview`, which is a developer's window; `-markers` and F1 bring them
back. The readout is switched off at the two front-end call sites rather than at the viewer's
default, so its test asserts **where** the switch is thrown.

### Owes

`0017` (`FR-11`), `0060` (`FR-7`, `FR-8`)

## 65dfdf7

### Boundary

after `0096` landed; `0099` in Phase 4

### What

`missionrun -census`: report every unit that changed cell, per tick rather than per snapshot, with
its slot and group. The owner watched the tenth mission in the original and counted the units that
move without being told to — **two, both peasants in the village** — and this tree could not produce
that number for itself, so several stories argued about autonomous movement with nothing to measure.
First reading: on the milestone drive **exactly one** unit moves, `u32` slot 4 group 7 from (48,44),
and it is what kills the witch.

### Owes

`0093` (the tool's own FR set)

## 7e4245c

### Boundary

after `0096` landed; `0099` in Phase 4

### What

`mapBorderDim` 0.5 → **0**: the map edge is black. The owner ruled it on 2026-08-01 («в оригинале в
принципе край карты черный») and it sat unbuilt in `OWNER-RULINGS.md` for six days until he said it
again. `0038`'s `DD-4` — dim by *multiplying* so the margin keeps its relief — is void rather than
retuned, so its `FR-4` bound and two tests were rewritten to what is now true.

### Owes

`0038` (`FR-4`, `DD-4`, `AC-3`)

## eef792f

### Boundary

after `0111` stages 1–3 landed; `0111` stage 4 in the lane

### What

**A mission's world carries the map's ground sacks.** `StartMission`'s party branch and
`StartMissionScripted` both rebuild the world, and both rebuilt through `sim.NewRelatedWorld`, which
names no sacks — so every shipped mission opened with an **empty** list while a plain load of the
same map carried the records the map authored. Measured on mission 10 through the shipped path:
`FromALMWith` 4, `mapload.StartMission` 4, `StartMissionScripted` **0**. `0103` landed the list
after both rebuilds were written; the comment above the second already warns in capitals that the
relation is lost if a rebuild does not name it, and the sack list is that lesson one field later.
Silent: nothing drew a sack until `0111`, so the only symptom was `FR-16`'s check opcode 14
answering 0 on every mission for every cell — indistinguishable from a map that authored no loot.
Both rebuilds now take `sim.NewLootWorld` with `base.Sacks()`; the regression test is the sibling of
the relation's own fence, at the same six shapes, and reverting either rebuild reddens a different
subset of it. The milestone drive is byte-identical on both roots and no landed test moves.

### Owes

`0103` (`FR-16`, `FR-20`)

## e09d2bb

### Boundary

after `0113` landed; `0112` in correction, `0116`/`0117` in their lanes

### What

**`HANDOFF.md` is out of the repository.** A lane's handoff note reached master inside `0089`'s
verification commit (`8680408`) and sat at the repo root for **24 stories**. Every line of it was
false: it named worktree `wt-0089` and branch `impl/0089-corpse-decay`, both long gone; the
submodule pin `96f0b15` against a live `87a256d`; `formatVersion` "was 16 and is 17 here" against a
live 25; and a `builds/0089-corpse-decay/` that does not exist. It was written in the imperative —
*"What is left for the orchestrator"* — so it read as work still owed, at the second file anyone
opening the repository sees. Nothing referenced it; one commit ever touched it. The owner found it
and named it exactly: a file that lies. `/HANDOFF.md` is now gitignored, with the reason and the
right home for a handoff written beside the entry.

### Owes

**no story** — this is process, not a contract: the rule is `AGENTS.md`'s existing one, that lane
state lives in the lane's return text or in that story's own `docs/<NNNN>/`

## 872487b

### Boundary

after `0119` landed; `0124`/`0125`/`0126` in their lanes. **Authored in `0124`'s lane as `7666149`
and cherry-picked to master by the seat**, because a hotfix that sits on a story branch is not a
hotfix: the owner reported this same defect a second time while the fix waited behind a story that
had already taken a byte-form version

### What

**The open inventory window swallows its own clicks.** It was drawn over the running world and every
button fell straight through it: a press on the figure selected whatever unit stood underneath, a
right press walked the party to the cell under the frame, and a drag begun on the window panned the
camera and drew a marquee. The window's rectangle now has one spelling — `inventoryWindowRect`,
which `inventoryPresent` also takes its origin from, so the rectangle a press is judged against is
the rectangle the picture is drawn on. A press or a release inside it, and any gesture latched as
having begun inside it, reach neither `decide` nor the camera; one pixel outside is the map's
exactly as before, because the world still runs beneath it (0110 `FR-6`) and this is not a
full-screen modal. The wheel is untouched: it is not a click, and gating it would be a second rule
nobody asked for. Witnessed by reverting the guard — a right press over the window then issues one
move order and a tap clears the selection.

### Owes

`0110` (`FR-6`, and the fence "do not let the binding reach the command path, the selection or the
camera", which the binding kept and the WINDOW did not)

## 7e3176c

### Boundary

after `0119` landed and `872487b`'s inventory hotfix; `0125`/`0126` in their lanes

### What

**The generation screen says what a point buys.** It showed sex, class, skill, the four statistics,
the cost and the budget — and no consequence at all, so moving Body told the player nothing about
what Body bought (owner, 2026-08-09). Six lines now stand under the footer: Health, Mana, To hit,
Defence, Damage (as the roll `base-base+spread`, hero.go's own pair), Speed. Left out deliberately:
sight, absorption, the five protections, the five resistances, the per-slot experience and the two
attack timings — each either constant across every spread the screen can reach, or a five-wide row
no single line carries, and the screen has no scroll window. **The seam is a `Derive` field on
`ChargenSetup`, not a third argument to `OpenChargen`** — cleaner because the wiring tier already
builds the whole setup in one call, so the one place that states what the screen needs stays one
place and `main.go` does not move; `pkg/ui` still imports `fmt` and nothing else, and every label
and value arrives already rendered (`0119` FR-3, AC-11). **The preview is the mint's own
arithmetic**: `mapload.PartySpawn` is now the exported single expression `StartMission`'s loop mints
from and `ChargenDerived` reads, so a previewed health cannot drift from the character the player
gets. While the spread is illegal the block says there is nothing to derive and `Derive` is never
called — a stale health read as the one about to be granted is worse than none.

### Owes

`0119` (`FR-3`, `FR-15`, `FR-16`, and a new FR for the block itself)

## c9d0a2c

### Boundary

after `0126` landed and master was `b267c4a`; `0125` in its lane

### What

**A unit's class-row weapon reaches the simulation, so a body has something to leave and the first
equip has something to displace.** Two owner reports, one root. The death path was already correct
and already fired at the right instant — `DecayNone → DecayFallen`, inside the guard that carries
"once per death", which is the same instant a second blow on the body is refused — but it poured
`w.carried[i]` and every container was empty. The only filler was the map's own type-8 owned
records: **43 across the 28 shipped maps, 26 of them empty, 25 elements in all** (`ITEM-OWNED-028`,
reproduced from the file side), and `ITEM-SPAWN-027` closes the other door, the random scatter being
multiplayer-only. The class row was the only source left and it reached nothing: `unitWeapon` and
`firstWeapon` resolved the row's equipment string to a `*data.Weapon` and fed the definition's
**combat numbers alone**, so a clubman died leaving nothing and a party member's first equip
superseded his starting weapon instead of displacing it into the pack (`0124` P-4, its own disclosed
hole). Three things now happen. **(1)** The resolved weapon's `Code` — which `ResolveWeapon` already
composes, so no inverse of `WeaponFromCode` was needed and its unchecked collection indices are not
made reachable — travels as `sim.Stock.Equipped`, a field carried **inside the record both mission
rebuilds already name**, which is what keeps a starting loadout from becoming the fourth state a
rebuild silently drops (the sack list in `0103`, the containers in `0112`). **(2)** A body unequips
slot 2 then slot 1 into its container before the container becomes a sack — `ITEM-DEATH-012`'s own
order, `actor+0x78` then `actor+0x74`, appended at the tail — all-or-nothing behind the existing
bounds guard, and the ten humanoid armour slots at `actor+0x198` are left on the corpse exactly as
that routine leaves them. Asking one predicate for both "is this a holding" and "would this body
drop" planted an **empty** sack for a body wearing armour alone; the two questions are now
`holdsSomething` and `dropsSomething` and each is asked by name. **(3)** The `"NPC"` suppression is
built. `ITEM-DEATH-012` deletes an NPC-templated corpse's whole container before anything drops;
`pkg/sim` holds no template name and could not gain a suppression bit without widening the byte
form, so the gate is applied to the **loadout the loader composes** — such a unit still fights with
his mace and leaves nothing behind. Measured on the EN root over all 28 campaign maps: 2333
placements, **1198 armed, 117 of them NPC-templated and therefore withheld**; what the gate cannot
reach is a map-authored owned record sitting on an NPC unit, of which there is **exactly one
non-empty in the whole campaign**. **Declined by name.** The row's other item cells: the `Humans`
collection's ten string cells hold **909 items over 216 rows, of which 156 resolve as weapons — one
per row, always the first — and 753 do not** (helm, mail, boots, bracers); `Units` names two cells
and 26 of 26 resolve, so nothing is left on that band. Composing a code for a helm needs field B,
and `ITEM-CODE-029` publishes B=3..13 → `Armor` without saying **which** of the eleven a helm takes
— undecoded, so a research question and not a value to invent — and under `ITEM-DEATH-012` those ten
slots are not dropped in any case, so the owner's «булава и сапоги» is the mace and not yet the
boots.

> **Correction, 2026-08-10 (0130):** answered by `ITEM-ARMSLOT-031` — the destination is a column of
> the piece's own `Armors` row, parameter 4, titled `Slot`.

> **Correction, 2026-08-10 (0130):** `ITEM-CORPSE-034` shows worn armour DOES reach the sack,
> through a `vt+0x44` call the death claim's published sequence steps over. This tree leaving the
> ten armour fields on the corpse is now a known divergence, and it owes a story.

Also declined: parameter 15's gate on slot 1, because that column lives in `pkg/data` and the
determinism wall puts it out of `pkg/sim`'s reach, so slot 1 always drops; and the gold roll for
`typeID > 0x40` (`HERO-KILL-027`'s three treasure columns plus a draw), so a bee still leaves
nothing. **No byte-form version moves**: `formatVersion` stays 34 and the equipment section already
existed — this fills it. The milestone drive is **byte-identical on both roots**, `lost at tick 272`
and `4 of 36 moved, 1 fell`, because the equipment set feeds no combat number in `pkg/sim` and the
weapon was already folded into the definition; the one unit that falls on that drive is one of the
unarmed ones. Witnessed on the lawful install by driving `p0` onto an armed guard: same outcome,
same tick, same 34 entities on master and here, and **two extra sacks holding `33031` where the two
armed units fell**, identical on both roots.

### Owes

`0123` (`FR-1` to `FR-5` — the drop set is the container **and** the two base-actor slots, plus the
`"NPC"` and parameter-15 gates); `0124` (`P-4`, closed here); `0112` (`Stock` now states a loadout
as well as a container). The remaining item cells and the death gold each owe a story of their own

## 637e513

### Boundary

after `0119` landed and `872487b`'s inventory hotfix; `0125`/`0126` in their lanes

### What

**No enemy indicator survives the fog.** The owner saw a health bar over an enemy nothing else drew
(2026-08-09); *«никакие индикаторы»* is the instruction, so the fix is the set. `0118`'s
`fogGateEntity` was correct and was asked at **two** of the nine walks over the entity snapshot —
`entityLayer` and `minimapMarks` — and every other walk was free to ignore it, silently, because a
leak draws pixels rather than errors. Six more sites now ask it: the health bar, the shot mark, the
damage numeral (**twice** — creation in `ingestDamage` so no arrears accrue in the dark, and
placement in `numeralPlacements` so a victim struck in the light and walking into the dark stops
being painted; one gate cannot do both), the attack cursor's outline, **the selection rim and the
route** — the last two found here and not in the brief: nothing scopes the selection to the local
participant's own units, so tapping an enemy and watching him walk into the dark left a rim and his
whole route standing. **Named and deliberately NOT fixed:** the press behind the attack outline is
still free to target a unit in the fog — gating `topAt`'s input changes what the player can *target*
rather than what he can see, and whether the original refuses that is not decoded, so the outline's
contract weakens in one safe direction (everything outlined is still what would be attacked) and the
rest is a story's question; and the info panel, the inventory's eligibility test and the debug
readout keep reading a selected enemy through the fog, because they are panels rather than map
glyphs and blanking a panel mid-inspection is a rule this hotfix does not make. The three gates that
deliberately differ are untouched: `fogGateGround` keeps a structure drawn once explored,
`fogGateSack` has no owner to except it, and the `owner == v.localOwner` arm is decoded behaviour.
**The structural exposure is now fenced rather than merely named**: `pkg/ui/fogwalk_test.go` parses
the package's own non-test source and fails, by function name, on any function that touches
`v.entities` and is not classified in its table — and a `gated` verdict is checked rather than
taken, since such a function must also call `fogGateEntity`. It is **lexical**, on
`internal/archtest`'s determinism-scan precedent, and two gaps are left open on purpose and written
down there: a walk reached through a helper (`entityByID`) is invisible, and a gate asked about the
wrong entity passes.

### Owes

`0118` (`FR-7`, `D-5` — the gate is theirs and the rule is "every consumer of the snapshot in a draw
path owes it", not "the one call site"); the targeting question owes a story of its own

## dbd5c89

### Boundary

after `0125` landed and master was `b93d793`; `0127-first-spell` and two hotfix lanes in flight

### What

**A character survives a mission, and finishing one names what follows.** Every mission opened on a
freshly minted party, so winning mission 10 and starting mission 20 threw away everything the player
had just earned — the owner asked for the RPG link between missions and this is its cheap half.
**What is carried** (`mapload.Carry`, a nil-able field on `PartyMember`, read by the mint and by
nothing else): the six per-slot experiences **as the exact integers the entity ended on**, the skill
levels those integers imply through `data.SkillLevelFor`, the container, and the twelve equipment
slots. The exact integer and the level are both carried because they are not the same fact —
`S(n)`'s inverse FLOORS, so a member carried through his level alone would hand back every point of
progress toward the next one at every mission boundary (700 becomes 610, 1700 becomes 1593; both
numbers are in the test). The levels are carried because nothing else in a member reaches his combat
block, his pools, his rate or his sight, so experience that moved without them would buy nothing.
**Named and deliberately NOT carried.** The two pools: health and mana are re-minted at the
recompute's own maxima, because there is no town and no healer between missions and carrying a wound
would be an authored difficulty rule nobody asked for. The purse: `pkg/sim` has no constructor that
can build a world with a non-zero purse — `TakeSack` is its only writer — so gold would need a `sim`
change, and nothing in this tree spends or displays it. Which weapon the combat derive reads: a
carried member wears what he ended in, but his numbers still come off the `*data.Weapon` he was
minted with — resolving a code back to a definition is `data.WeaponFromCode`'s unchecked path, and
equipping does not move a combat block INSIDE a mission either (`equip.go`, FR-11's own single call
site), so this hole is that hole and not a new one. A member whose entity did not survive: carried
forward exactly as he ENTERED that mission, nothing invented and nothing taken away. **Nothing is
written to disk** — a carry lives for the life of the process, and persisting one is a different
goal with a different cost. **The order comes out of `scenario.reg`, which this tree reads for the
first time** (`pkg/game/campaign.go`; the sixth registry beside `npc`, `sfx`, `objects`,
`structures` and `units`): 24 `[Mission<n>]` sections, 15 main and 9 side, and the union of
`InnMission`/`ShopMission`/`TCMission` — 26 offered missions, 30…151. **The town's boundary is READ
and not known.** The missions no building offers are exactly 10 and 20, on both roots, measured from
the file rather than taken from the claim; that is the owner's own account («после 10 миссии игрок
не попадает в город, а попадает сразу в 20 … окончание 20 приводит к городу») corroborated from the
other side by `REG-SCN-064`'s corpus, whose coverage starts at 30. So **10 → 20 is the campaign's
own successor and is stated plainly, and 30 → 40 → … → 150 is ascending order this tree AUTHORED as
a placeholder for the town's gate choice** — `Offer.Town` carries which of the two it is, read off
the offer set and never off a mission number compared against a literal, and the sentence the map
list is given says so in the player's own words. `[Mission10] AutoGetMission = 20` and `[Mission150]
LastMission = 1` agree with both ends of that ladder and are read as a **cross-check only**:
`REG-SCN-059` locates each key's reader without saying what either means, so interpreting the names
would be a fact this tree is not entitled to. **The town is not built and nothing here is a step
toward one** — it is three buildings consuming offer lists (`REG-SCN-064`), a global map
(`globalmap.reg`, 35 objects and a 28-entry mission table) and an explicit choice at the gate; what
stands in its place is a sentence naming a number. **`pkg/game/world.go` IS NOT TOUCHED**: the
driver still decides what a notice does and `world_test.go` still asserts its destination and its
own sentence — `MissionOpenerWith` WRAPS the advance seam it already returns, recognises a win from
`World.Outcome()` rather than from the destination alone, and calls one method on the front end.
**No byte-form version moves** — `formatVersion` stays 35 and `pkg/sim` is not touched at all; a
carry reaches values in records that already exist. The milestone drive is **byte-identical on both
roots**, `lost at tick 272` and `4 of 36 moved, 1 fell`, as expected: the milestone opens mission 10
with no prior mission, where a nil carry takes the arm the mint always took. Witnessed by reverting
ten lines one at a time — the exact-experience override, the level write, the carry arm of the
equipment block, the aliveness guard, the array-shaped key, the zero sentinel, the town flag, the
won-outcome guard, the capture and the party swap — each reddening a named assertion, none
surviving.

### Owes

`0066` (`FR-11` — where a dismissed win leads and what it says; the sentence is now two sentences
and carries a consequence); `0119-chargen` (`FR-15`, `FR-16` — the mint's pools and its one
recompute now stand beside a carry); `0125` (`FR-13` — an entity's `SkillXP` is seeded from a
carried integer as well as from the levels' own sum); `0124` (`Stock.Equipped` now states a carried
loadout as well as a class row's). **The town owes a story of its own**, and so do the purse and the
weapon a carried member's numbers are derived from

## 026e936

### Boundary

after `0125` landed and master was `b93d793`; `0127` in its lane, two more hotfix lanes and a
continuity lane live

### What

**A creature's own attack is not an item, so it never enters the equipment set.** The owner killed a
bat on mission 20 and found its **Sonic Beam** in the sack. `c9d0a2c` **declined this gate by name**
— `ITEM-DEATH-012` (High) unequips `actor+0x74` only when the weapon's own `Data.bin` parameter 15
is non-zero, and that column lives in `pkg/data`, past the determinism wall from `pkg/sim`'s side,
so slot 1 always dropped. The decline was the defect. **The wall is not in the way, because the
decision does not belong inside it**: it is the LOADER that chose to write a code into an equipment
slot, and `pkg/mapload` is outside the wall — so `startingLoadout` now asks `carriable` and an
innate weapon never becomes an item at all. **The column separates the two populations exactly**,
re-measured here over the whole shipped Weapons collection and identical cell for cell on both
lawful roots: index 15 of the parameter array is 1 on the nineteen blades, clubs, polearms, axes and
bows, 2 on the two staves, 3 on `BareHands` and `Plasma Sword`, -1 on the removed row `rem` — and
**0 on Sonic Beam, Flame Thrower and Boulder Thrower and on nothing else**. The row has seventeen
cells against eighteen titles, so column i is title i+1 and index 15 is the one the file itself
spells `sutableFor`; that is a NAME and not a decode — the claim still grades the column's meaning
Unknown and only the behaviour is claimed.

> **Correction, 2026-08-10 (0130):** `ITEM-SUIT-035` decodes it — `sutableFor` is a two-bit mask
> over consumer classes, and the death gate's whole-value test means *neither bit set*.

The compare is the claim's own `!= 0` and not `> 0`, so `rem`'s empty cell reads as carriable
exactly as the original reads it, and **a row too short to carry the column is carried**: fourteen
and fifteen-cell rows resolve, and silence is not zero. **The alternative was considered and
refused**: a droppable bit on the entity consulted by `pkg/sim` models an innate attack as an item
and then remembers not to drop it — one more piece of hashed state, and a byte-form version, for a
question the loader has already answered. **No version moves; `formatVersion` stays 35.** **It is
not a trade**: the fold is `definitionFor`'s and runs before the gate is asked, and the equipment
set feeds nothing in `pkg/sim` but the death drop, the equip move and the inventory window — the
tree's one `SetCombat` call site recomputes from the inventory SUBJECT's slots, which is a party
member and never a creature. Measured on the lawful install: each of mission 20's five bats is
byte-identical field for field before and after, `Reach:4 DamageBase:1 DamageSpread:4 AttackCharge:8
AttackRelax:2 ToHit:55 Defence:45 AlwaysHits:true`, with `eq=[280 …]` becoming `eq=[0 …]` and
nothing else changing. Mission 20 on both roots: 57 entities, **36 armed before and 31 after**, the
five being class 70 at reach 4 all holding code `0x0118` `Common Iron Sonic Beam`; killing all 57
plants **40 sacks before and 35 after**, and killing only the five bats plants **5 sacks before and
none at all after**. Over the five missions `scenario.res` carries (10, 20, 30, 31, 40), 212
placements, **114 armed before and 106 after**. **NOT DONE, and named rather than hidden.**
`pkg/sim/step.go`'s death-drop comment still says the parameter-15 divergence is open and that this
build "drops it always" — true of that package and now incomplete about the tree, and it was left
alone because `0127` holds `pkg/sim`; it owes one sentence saying the loader closes it.
`mapload/start.go`'s party path composes its slot-1 code through `weaponCode` directly and is NOT
gated, because a minted hero's weapon comes from chargen and no generated character can name an
innate attack; if that ever stops being true the gate belongs there too. And the column index is
transcribed in `pkg/mapload` while its eight siblings live in `pkg/data/weapon.go` — the second copy
is a scope decision, not its right home. **The owner's «с клабменов падает дубина И САПОГИ» is
untouched by this and still open**: nothing here reaches the ten armour cells, `startingLoadout`
composes slot 1 alone, and `ITEM-DEATH-012` leaves `actor+0x198` on the corpse — so the boots are
still the same open question `c9d0a2c` left, needing field B for a helm (`ITEM-CODE-029` publishes
B=3..13 → `Armor` without saying which) and an image question about what the original actually
drops.

> **Correction, 2026-08-10 (0130):** `ITEM-CORPSE-034` shows worn armour DOES reach the sack,
> through a `vt+0x44` call the death claim's published sequence steps over. This tree leaving the
> ten armour fields on the corpse is now a known divergence, and it owes a story.

### Owes

`0123` (`FR-1` to `FR-5` — the parameter-15 gate `c9d0a2c` declined, now closed on the loader side);
`0124` (`P-4`); the boots and the death gold each still owe a story

## b55f111

### Boundary

after `0125` landed and master was `b93d793`; `0127` in its lane

### What

**The credited skill slot follows the weapon.** The owner took a mace off a corpse, equipped it,
started killing and reported «опыт капает, но навык не растет» — experience accrues, the skill does
not. It was right: `sim.CombatBlock` carried **nine** fields and `SetCombat`'s own doc said in
capitals that it wrote "THOSE NINE FIELDS AND NOTHING ELSE", while `XPSlot` was written once, at the
map load or the party mint, and by nothing after. `rearm` (`pkg/game/world.go`) computed a whole
`Derived` whose `Combat.SkillSlot` is `activeSkill(l.Weapon)` — the same call that produces the
damage pair and the reach — and then dropped it, having nowhere to put it. So a hero fighting with a
mace went on paying every blow into Blade, which a generated hero starts at level 10: the panel
shows S(10) = **1593** and the next level is S(11) = **1853**, so **260** further experience at
roughly one per blow before the number could move, while Bludgen sat at 0 with S(1) = **100** a few
fights away. Experience visibly moved and the skill visibly did not. **`XPSlot` is now the tenth
field of `CombatBlock`**, written through the door the nine already use rather than a second one
beside it — no byte-form version: the field already exists on `Entity` and already sits in the byte
form at `+181`, and `formatVersion` stays 35. **The bounds guard is `experienceFault`, not a second
comparison**: `SetCombat` builds the entity it would leave behind, asks the one predicate the
constructor and the decoder already ask, and on a refusal writes **none** of the ten rather than
nine of them. That is not defensive — `activeSkill` answers a melee weapon's attack type and "melee"
is `AttackType < 0xa` against six slots, and the shipped Weapons table **has** such a row: measured
on both lawful roots, 28 rows of which row 23 carries attack type **-1** and still resolves through
`data.WeaponFromCode` (code `0x0117`) as melee, which `uint8` narrows to **255** and
`payExperience`'s `a.SkillXP[a.XPSlot]` would have indexed with. `pkg/game`'s own `creditedSlot`
narrows first, to `SkillGeneral` outside `(SkillGeneral, SkillSlots)` — **`recompute.go` step 6a's
own window**, where the skill's to-hit and damage terms are added only inside it, read one field
further rather than invented here. **The mission-start path was checked and is right**: a party
member takes `p.Hero.Derive(p.Weapon)`'s own slot (`start.go`) and a placed person `h.Combat(w)` →
`Hero.Derive(w)` (`fromalm.go`), both the recompute's; the creature and unresolved arms leave slot 0
because `UnitDef.Combat` derives none, and that is inert rather than a defect since both carry
`gainsXP: false`. Measured on both roots, **zero** of the 156 weapons the 216 `Humans` rows name and
**zero** of the 26 the 119 `Units` rows name resolve outside 0..5 — so the `-1` row is reachable
through an item **code** in a container, which is the equip path, and not through any definition's
equipment string. Witnessed by reverting: delete `e.XPSlot = c.XPSlot` and the end-to-end test
reddens with "the credited slot is 1, want 3"; delete the fault guard and a block carrying slot 255
writes itself into the world and moves the hash. **Not heard, only measured** — the owner's own
reproduction needs a double-click this lane cannot deliver, so the equip is driven through
`enqueueEquip`, the command path that click raises.

### Owes

`0125` (`FR-2`, `FR-14` — "resolved once at the map load or the party mint" is the clause this makes
false, and the credited slot owes the same "follows the loadout" rule the nine combat numbers have);
`0124` (`FR-11` — the rearm's field set)

## 4ac0744

### Boundary

after `0125` landed and master was `b93d793`, on the same branch as `b55f111`; `0127` in its lane

### What

**A swing is voiced only when the attacker could reach what it is swinging at.** The owner heard
them «сильно раньше чем стартовала битва — враги еще идут, а свинги уже слышны»: enemies still
crossing the map, already audible. **The sound was faithful to the cycle and the cycle runs out of
reach.** `pkg/sim`'s step advances the attack cycle for every alive entity that holds an attack
target, with **no distance test at all**; `advanceAttack` enters `AttackCharging` and counts down;
only `resolveBlow` refuses, on `inReach`, at the very end. And `combat.go`'s own comment says that
is deliberate — *"a cycle that cannot reach anything runs at the same rate as one that can, and the
draw count does not depend on where the victim happens to stand"* — so `advanceSwings`, correct on
its own terms and gated on `AttackPhase == AttackCharging`, voiced every charge of an attacker that
was still walking. **The voicing is narrowed, the cycle is not**: no phase, no countdown, no draw
and nothing the world hashes moves, and the run counter still advances, so a swing that comes into
reach mid-run is not restarted. **AUTHORED, and not claimed as decoded**: `ANIM-SND-022` (High)
establishes that the swing is fired by the driver's attack arm and shoot arm on the `AttackDelay`
tick and says nothing whatever about whether that driver enters the arm out of contact — nobody has
read it — so this is our conservative choice, that a sound which cannot correspond to a blow is not
played. **The bigger question is named and NOT answered here**: whether this tree's attack cycle
should be starting at all before contact is a fidelity question about `pkg/sim`, larger than a
hotfix, and it is visible only because sound made it audible — the sound did not create the defect,
it exposed one. It owes a story. `sim.InReach` is **exported** rather than a second distance
expression written in `pkg/game`: `strikeDistance` is deliberately written in the law's own
footprint-aware terms rather than the `max(1, Chebyshev)` it currently equals, so that a later wider
footprint has ONE place to change, and a copy across the wall would silently not be that place. A
victim the snapshot no longer holds answers false and is voiced not at all. Witnessed by reverting
the gate: an attacker eight cells from its victim emits **1** where it must emit **0**, at the cell
`x=9` it had walked to — mid-approach, which is the report — while the adjacent control still emits
exactly **1** and the run counter still passes the class's delay in both. **Not heard, only
counted**: this lane cannot play audio, so the witness is emission counts.

### Owes

`0126` (`FR-5`, `AC-14` — the swing's own emission rule owes the reach clause); the attack cycle's
own pre-contact question owes a story of its own

## 29c2f77

### Boundary

after `0125` landed and master was `b93d793`, on the same branch as `b55f111` and `4ac0744`; `0127`
in its lane

### What

**A decaying corpse does not grunt.** The owner: «поправить звук когда враг получает урон от
истлевания — тут не должно быть звука». **This is hotfix `9729459`'s defect one instrument over**,
and the fix is the same sentence: gate on the health the front end **already held**, not on the
value that just arrived. `GruntFloor` was `ANIM-SND-022`'s own gate and right as far as it goes —
refuse at or below **-10** — but the decay ladder walks a corpse down from 0 and **every rung from
-1 to -9 clears that floor**, so a body grunted its way to the bottom after it was already dead. In
the original those decrements never reach the client as a number at all: a dead actor's falling
health arrives as the corpse **stage** byte and *"the thresholds never reach the client as numbers"*
(`ANIM-DEATH-007`, High), so the `0x73` arm that decides a grunt is never entered for them. This
tree's front end sees the same state the simulation holds, which is why it heard what the original's
client is never told. One line — `if had <= 0 { continue }` in `stepSound`, the exact line
`ingestDamage` already carries — and the **killing blow still grunts**, because it lands on a living
victim and only the decreases after it are dropped. Witnessed by reverting it: a killing blow plus
nine rungs makes **10** plays where it must make **1**. The nine pushes are spaced **2000 ms**
apart, clear of `GruntThrottle`'s 1500 ms, on purpose — pushed at one instant the throttle alone
would have suppressed them and the test would have passed against the unfixed tree. **Not heard,
only counted**: this lane cannot play audio, so the witness is the recording player's own play count
and the slot each play resolved to.

### Owes

`0126` (`FR-1`, `FR-2` — the grunt's gate owes the numeral's already-held-health clause, and
`GruntFloor` alone was never enough); `0083` (the same clause, already owed there by `9729459`)

## 8e13e8f

### Boundary

after `0125` landed and master was `b93d793`, on the same branch as `b55f111`, `4ac0744` and
`29c2f77`; `0127` in its lane

### What

**A sack is painted over the body that dropped it.** The owner: «мешок всегда самый верхний
относительно тела, а сейчас он не виден» — a body drops a sack and the sack is underneath it,
invisible. `DepthOrder`'s interleave chose `sRow <= eRow` at both of its two sites, **sack first at
an equal row**, and the list is painted in order, so first means underneath; a corpse and the sack
it just dropped share a cell and therefore a row. `0111` plan DD-2 states that tie-break and **gives
no reason for it** — no claim, nothing that depended on it — a choice made when nothing could tell
the difference. Something can now. **The owner's ruling as author (2026-08-09) is the whole rule and
it is three-way, not a flipped tie**: «на клетке стоит герой или юнит — он выше всех. если есть
мешок, то он под героем. если есть труп, то он под мешочком» — living unit above everything, sack
under it, corpse under the sack. A single comparison cannot express that, so a `DepthTie` rides on
`StaticPlacement` (`TieCorpse` -1, `TieGround` 0, `TieUnit` 1) and the order is lexicographic on
**(row, tie)**. **Two halves, both needed and each witnessed alone**: `sackFirst` — now ONE function
called from both sites, because this rule has already been wrong twice in exactly the shape of a
comparison written twice — and `rowOrder`'s own tie term, which is what lets ONE entity stream carry
two tiers; without it a corpse and a living unit reach the interleave in list order and the sack can
only land on one side of the pair. **The zero value is the ground tier**, so a sack placement and
every caller that names no tie get exactly the order `0111` gave them — asserted, not inferred from
the frozen fixtures passing. **AUTHORED throughout**: nothing decoded establishes a paint order
between a body and a ground item, and this is named as his ruling rather than as a reproduction.
**Only `LifeDead` takes the corpse tier**: a downed unit — exactly zero health, a body in the way,
not yet dead and having dropped nothing — stays with the living, the narrowest reading that
satisfies the ruling, and it is disclosed here because nothing decides it. Unchanged by design: the
flat decoration prefix, the cross-row order, and the interleave itself (a near sack must not be
drawn in front of a far entity). The tie is set in `pkg/ui`'s `entityLayer` and **not** in
`statics.go`, because the sprite list drops entities — off-grid, fogged, no art — so its index
cannot be mapped back to the snapshot anywhere downstream; the life state crosses the seam already
classified and this package makes no classification of its own. Witnessed by reverting each half:
`sackFirst` back to `sRow <= eRow` reddens "corpse then sack" and the three-way case **on both sides
of an art ref** while "sack then living unit" stays green, which is the shape of the defect;
`rowOrder`'s tie term alone reddens **only** the three-way case, which is why that case is asserted
directly rather than left to compose out of the two pairs. **Not seen, only ordered**: this lane has
no rendered frame — `SetForegroundWindow` is refused — so the witness is the emitted `DepthRef`
sequence by kind and index, never a picture. He found it by looking and he is the one who will see
it.

### Owes

`0111` (`FR-7`, plan `DD-2` — DD-2's equal-row clause is **void** rather than retuned, and the three
tiers replace it); `0123`/`c9d0a2c` (a body's drop is what makes the case reachable at all)

## ea870ef

### Boundary

after `0125` landed and master was `81e1e9e`; `0128` in its lane

### What

**The info window names the weapon he is actually holding.** The owner equipped a different weapon
out of the inventory and the panel's WEAPON row went on naming the old one — the damage pair,
to-hit, defence and the swing timings all moved, because `rearm` recomputes and calls `SetCombat`,
and the NAME is not a combat number and lives somewhere else: `ui.UnitCharacter.Weapon`, in
`mw.chars`, a map `partyCharacters` writes once at the open and **nothing wrote again**. This is
`b55f111`'s defect a second field over — that hotfix's own row says it, *"everything a mace changed
about the blow arrived, and the skill the blow trained did not"* — and the name is the third thing
that did not arrive. **The `entityDraws` comment is why it survived**: it listed the weapon name
among the members that *"stay exactly what `mw.chars` gave them"*, an assertion of stability nothing
was maintaining, so a reader checking whether the row could go stale was told it could not. That
sentence is now void and the loop says instead that the name is neither overlaid nor stable. **Shape
chosen: write it in `rearm`, not overlay it in `entityDraws`.** The overlay is the uniform-looking
half — it is how `Skills` and `Experience` already refresh — but it would re-resolve an item code to
a definition on every tick for every entity, and only the inventory SUBJECT has equipment this tree
can change or an `invPartyGear` to resolve against, so for every other entity the per-tick resolve
answers nothing. The write reuses the resolve `rearm` is already doing, runs once per equipment
change, and its cost is stated rather than hidden: it covers the subject alone, which is exactly the
set whose weapon can move. It sits AFTER `SetCombat` and inside none of `rearm`'s early returns, so
a nil table or an unresolvable code leaves the name exactly where it leaves the combat block. **A
bare hero reads as bare**: `w` falls back to `invParty.startWeapon`, nil for a character generated
holding nothing, and the write clears the row rather than leaving the last name standing —
`buildInventorySubject`'s own "a nil weapon is a bare hero" (`inventory.go`), applied at the other
end. The lookup is a comma-ok, so a subject the load knows no character for writes nothing instead
of minting one. **No byte-form version moves** and `pkg/sim` is not touched at all: `mw.chars` is
front-end state and the simulation already had every value this reads. **Witnessed by reverting,
both halves separately**: drop the write and the first test reddens with `the panel names "Sword"
after equipping the mace, want "Mace"` — the defect verbatim — while the second reddens on the
seeded stale name; drop the nil guard alone and the second **panics**. Both read `mw.entityDraws()`,
the readout the panel is built from, and both compare one resolved NAME against another rather than
against emptiness, so writing something merely non-empty into the row would not pass. **Named and
NOT done**: this tree has no unequip — `sim`'s `equip` moves a code INTO slot 1 and only ever swaps
what was there back into the container — so the named-to-bare transition cannot be driven through
the command path and its test seeds the stale name instead of reaching it, which the test says in
its own words. Nothing here reaches a second party member's weapon row, because nothing can change
one. **Not seen, only asserted**: this lane has no rendered frame, so the witness is the string in
the readout and never a picture.

### Owes

`0124` (`FR-11` — the rearm's field set, the same clause `b55f111` already owes: a recompute's every
consumer follows the loadout, and the panel's own name is one); `0113` (`FR-17` — `partyCharacters`'
map is no longer load-time-only)

