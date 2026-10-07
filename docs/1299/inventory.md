# Story 1299: Spell mod surface, inventory and proposal

> Historical proposal at source 36ba2033. This inventory preserves its original decisions and Unknowns; current spell APIs and implemented behavior belong to the later story documents and source.

## Result

The engine implements 28 spells (ids 1 to 28, one `Data.bin` Spells row each).
Today a mod can change none of their values: `game.rules` exposes one
parameter (`skill_cap`) and no data file touches a spell. This document lists,
for each spell and for all spells, what the engine could expose to mods, the
code or data that holds each item now, and the limits that block large values.
It proposes the API (TOML keys, Starlark hooks, SAV handling) and splits the
work into six stories. No production code changes in this pass; the owner
approves the list first.

Scope: this story extends the knobs the engine exposes. It does not write a
mod. A fixture mod is allowed only inside a test, to prove one knob.

The three owner examples:

| Example | What it needs | Blocking limit |
|---|---|---|
| Prismatic Spray with 100 rays | A ray-count rule instead of `min(power/20+2, 7)`; the 10-slot winner loop lifted; a policy for rays beyond the number of distinct enemies; one training award per cast | `limit := uint8(min(power/20+2, 7))` and `min(limit, 10)` at `pkg/sim/prismatic.go:64,85`; rays are distinct actors, so 100 rays need 100 candidates. The renderer has no ray cap |
| Heal on enemies | One predicate override in the apply arm and one in autocast target choice | `pkg/sim/spell.go:259-266` refuses a hostile pair; `pkg/sim/spell.go:2197` skips hostile targets for an unbidden heal |
| Fire_Ball radius | A radius key; correct cell walk for radius above the wrap point; a burst drawn per covered area | `blastCells` truncates coordinates to bytes (`pkg/sim/celleffect.go:592`); the burst is one object at the anchor (`pkg/sim/burst.go:62`); `SpellRule.Radius` and the saved area radius are `uint8` |

## Sources

- Engine tree at `36ba2033` (branch `story-1299-spell-mod`), read directly.
- `Data.bin` Spells rows read through `data.LoadSpells` from the EN root with a
  throwaway tool that was not committed. The 28 rows of default numbers are not
  repeated here: they are game data and live in the untracked owner file
  `review/mod-spells/SUMMARY-RU.md`. This document states defaults only as code
  constants and as the formula that reads a column.
- Pinned knowledge: `formats/magic/` (`casting.md`, `application.md`,
  `area.md`, `marks.md`, `gates.md`) and the claims they cite
  (`MAGIC-SPRAY-134..137`, `MAGIC-TARGET-017`, `MAGIC-ARM-014`,
  `MAGIC-SING-019`, `MAGIC-AREACELL-039`).
- Line numbers are measured on `36ba2033`. Literal-id counts are a rough
  non-test grep: `pkg/sim` about 119 case or compare lines on spell ids,
  `pkg/game` about 43, `pkg/ui` about 12, `pkg/mapload` about 7.

## Existing mod framework

| Part | State |
|---|---|
| `pkg/mod` | Manifest, settings, ordering, content digest, data parsers for items, screens, companions, characters |
| `pkg/modrt` | Starlark runtime; the only importer of the interpreter. `init(game, settings)` sees `game.rules` and `game.data.add(path)`. Step limit 20 000 000. No file, clock or randomness |
| `game.rules` | One parameter, `skill_cap` (`ruleFields` in `pkg/modrt/modrt.go`); each parameter is an integer inside a declared range |
| `pkg/rules` | Immutable `Rules` built once from `Params`; the simulation receives the finished value and never sees the interpreter |
| Data files | `data/items.toml`, `screens.toml`, `companions.toml`, `characters.toml`, loaded by `game.data.add`; rows are keyed by a name, edits name a target row |
| Table seam | Mod item data enters `mapload.Table.Mods` (`ModContext`); spell rows are read by `mapload.SpellRules(t)` from `data.LoadSpells` (`pkg/mapload/spell.go:120`) |
| SAV | `AgainromMods` leaf (`pkg/game/modmark.go`, `pkg/formats/sav/native_mods.go`): mod set digest, true skill values beyond the original domain, item stand-ins, layers. Load requires the same mod set; a different set is refused naming each difference |

Owner rulings that bind the surface: a mod is files only; data rows are TOML
and code is Starlark; nothing mod-specific is compiled into `againrom.exe`;
SAV is the only save format.

## Where a spell value comes from

One path feeds every consumer. `Data.bin` row -> `data.LoadSpells`
(`pkg/data/spell.go`) -> `mapload.spellRules` (`pkg/mapload/spell.go:70`) ->
`[]sim.SpellRule` -> `World.spells`, which is part of the world byte form
(36 bytes per row, `pkg/sim/binary.go:1515`, `formatVersion = 95`, line 937).
Book casts, staff and scroll casts, weapon riders, creature spells and the
spellbook popup all read that one table. A change made at the table therefore
reaches every consumer and every cast route. A formula change must be made in
`pkg/sim` and presentation, because the arithmetic is code.

## Knobs every spell row has

Each row carries these columns. "Source" is the code that reads the value.

| Knob | Column or constant | Source | Width and limit |
|---|---|---|---|
| Mana cost | col 1 | `debitBook`, `bookAffords` (`pkg/sim/spellbook.go`) | `int32` in the table; the SAV book slot holds `uint16` read as `int16` (`BookSpell.ManaCost`, `pkg/sim/spellbook.go:22`) |
| Cast range | col 6, bonus `power/30` (Teleport `power/3`) | `spellRange` (`pkg/sim/spell.go:765`) | `uint8` in `SpellRule.MaxRange`; SAV book slot `Range uint8` (`RefreshBook`, `pkg/sim/spellbook.go:140`) |
| Recovery addend | col 0 | `bookRecoveryTicks` (`pkg/sim/spell.go:1305`) | `uint8`; the sum clamps at 255 |
| School | col 2 | `rule.School` picks the skill slot and the protection byte | 1..5; skills `[6]` in SAV |
| Damage pair | cols 16, 17 | `spellDamage` (`pkg/sim/spell.go:781`): `base = dmin*(power+30)/30`, `spread = dmax*(power+30)/30 - base` | `int32` in the table; an in-flight payload stores base and spread as bytes (`pkg/sim/spelldeliverysave.go:23`) |
| Delivery and speed | cols 5, 7 | `spellDeliveryTicks` (`pkg/sim/spelldelivery.go:35`) | Lightning and Prismatic Spray take a flat 10 ticks by id |
| Target flag | col 4 | `castSpell` target gate (`pkg/sim/spell.go:863`) lists nine ids by literal | `bool` |
| Defensive flag | col 18 | autocast target tier, SAV book slot | `bool`, `uint8` in SAV |
| Shape and radius | cols 8, 9 | `areaModeFor` (`pkg/sim/celleffect.go:246`), cell producers | `Radius uint8`, `Distribution uint8` |
| Area duration | col 11 | `areaLife` (`pkg/sim/celleffect.go:271`): `(AreaDuration<<4) + (power<<4)/10 + 1` | `int32` row, `uint16` live counter |
| Spell duration | col 14 | `lastingTicks` (`pkg/sim/spell.go:203`): `1.025^power * SpellDuration * 16`, power 0..100 table | `uint16` result |
| Effect string | trailing text | `parseSpellEffect` (`pkg/data/spell.go`): kind, magnitude, mode, duration | duration `uint16`; kind from a closed list of nine |

## Per-spell inventory

Each entry names the knobs beyond the common columns. Presentation limits
that apply to the whole group follow the table.

| Id | Spell | Changeable data | Changeable formula | Targeting rule | Presentation or hard limit |
|---|---|---|---|---|---|
| 1 | Fire Arrow | damage pair, range, speed, mana | damage factor `(power+30)/30` | unit target, not self (`spell.go:872`); hostile or any | flight picture 10 flies, length `dist/200`, 6-point trail (`data/projectile.go:293`); in-flight damage byte payload |
| 2 | Fire Ball | damage pair, radius (col 9), range, speed, mana | per-cell damage divided by target footprint area (`celleffect.go:963`); delay by Euclidean distance (`spelldelivery.go:53`) | cell or unit aim; no owner or hostility filter in the cell walk, so allies and the caster's own units are hit | square `(2r+1)^2` cells, coordinates truncated to bytes (`celleffect.go:592`); one burst picture at the anchor (`burst.go:62`); scorched cells for ids 2 and 3 only (`scorched.go:17`); `Radius uint8` |
| 3 | Wall of Fire | damage pair, area duration, range, mana | life `(AreaDuration<<4)+(power<<4)/10+1`; pulse every 16 ticks (`celleffect.go:180`) | cells only; wall shape ignores Radius | fixed 10-cell and 9-cell wall tables (`celleffect.go:687`); layer 3 of 6 (`arealayers.go:12`); overlay art for ids 3, 7, 8, 19 only (`data/projectile.go:195`) |
| 4 | Fire Sacrifice | mana (0), range (0), complication | damage built from caster HP plus mana, bytes capped at 255 (`celleffect.go:886`); sets caster HP to 1 and mana to 0 | cell | two fixed shells of 8 and 12 cells; stage count 2 by id (`celleffect.go:601`) |
| 5 | Protection from Fire | magnitude, duration, range, mana | magnitude `power/2` (`spell.go:395`); duration `lastingTicks` slow | friendly buff; autocast gives it to own team, then locked allies | protection clamps at 100 per element; effect id bit in a 32-bit mask |
| 6 | Heal | damage pair (heal amount), range, mana | amount from `spellDamage`, clamped at max health (`spell.go:1208`) | hostile pair refused at apply (`spell.go:259-266`); autocast skips hostile (`spell.go:2197`); needs `MaxHP > 0` | heal burst literal ids 6 and 11 in `pkg/game/spellbolt.go:168` |
| 7 | Freezing Cloud | radius, area duration, effect duration, speed magnitude | speed `-(power/15+1)` (`spell.go:401`) | cell; layer 7 conflicts with layer 3 (`arealayers.go:16`) | overlay art; layer 7 of 6 slots |
| 8 | Poison Cloud | radius, area duration, magnitude, tick duration | magnitude `mag*(power+30)/30` (`spell.go:403`); pulse every 8 ticks (`effect.go:466`); water protection reduces (`poison.go:5`) | cell; blocked by Wall of Fire cells (`celleffect.go:858`) | duration word `<<4` in `mapload/spell.go`; `Remaining` above 9600 never ticks (`effect.go:461`) |
| 9 | Acid Stream | damage pair, range, mana | six stage wedges fixed by id (`celleffect.go:627`) | cell, direction from caster | Radius column unused by the program; 6 stages; ring cadence 3 ticks (`celleffect.go:185`); burst picture 27, life 18 |
| 10 | Protection from Water | as 5 | magnitude `power/2` | as 5 | as 5 |
| 11 | Drain Life | damage pair, range, mana | roll capped at victim HP plus 10, moved to caster, caster capped at max (`spell.go:269-290`) | unit target; excluded from `Damaging` by literal id (`data/spell.go:healSpellID`) | heal burst literal id 11 |
| 12 | Light | radius, area duration, scan magnitude | scan `power/30+1` (`spell.go:405`) | cell; layer 12 conflicts with layer 17 | scan-range effect of one tick |
| 13 | Lightning | damage pair, range, mana | factor as 1; delay flat 10 ticks by id | unit target | picture 34, 13-tick path figure (`data/projectile.go:232`); in-flight byte payload |
| 14 | Prismatic Spray | damage pair, range, mana, ray count | ray cap `min(power/20+2, 7)` (`prismatic.go:64`); winners `min(limit,10)` (`prismatic.go:85`); score `(edgeDistance<<8 + turnCost)&0xffff` | primary bypasses filters; secondaries come from the caster group's sight and hostility (`prismatic.go:15`); a body is never primary | one figure per victim, colour `index%7` (`pkg/game/spellbolt.go:256`); tooltip `RayCount uint8` (`spell.go:657`) |
| 15 | Invisibility | duration | duration `1.05^power*3*16`, base 3 literal, ends on a cast at another actor (`spell.go:389`, `spell.go:829`) | friendly | none beyond duration word |
| 16 | Protection from Air | as 5 | as 5 | as 5 | as 5 |
| 17 | Darkness | radius, area duration, scan magnitude | scan `-1-power/30` (`spell.go:407`); victims order an attack on the caster (`celleffect.go:978`) | cell | scan-range effect of one tick |
| 18 | Shield | duration, magnitude, mana | absorption `power/10+3` (`spell.go:397`) | self by range 0 and a literal id list | none beyond duration word |
| 19 | Wall of Earth | area duration, range, mana | wall tables as 3; skips cells holding a ground actor (`celleffect.go:737`) | cell | layer 19; overlay art; no damage |
| 20 | Stone Curse | magnitudes, duration | duration reduced by target earth protection (`spell.go:420`); training award 3 percent of max health (`spell.go:360`) | unit target | immobilise gate is one 32-bit mask, spell ids bounded at 32 (`gates.md`) |
| 21 | Meteor Storm | damage pair, range, mana | 32 stages, one random cell in a 5x5 square per stage (`celleffect.go:663`) | cell | inset gate 8 cells from the map edge (`celleffect.go:672`); stage count by id |
| 22 | Protection from Earth | as 5 | as 5 | as 5 | as 5 |
| 23 | Bless | magnitude, duration | to-hit `power*4/5+20` (`spell.go:388`) | friendly | none beyond duration word |
| 24 | Haste | magnitude, duration | speed `power/15+1` (`spell.go:399`) | friendly | speed floor of 1 (`effect.go:120`) |
| 25 | Control Spirit | mana, range | ghost from the `Ghost` Units row; reaction `/2+1`, Mind and Spirit copied, health `/2` (`spell.go:488`) | bones corpse only (`spell.go:451`) | ghost template is one row |
| 26 | Teleport | mana, range | range bonus `power/3` (`spell.go:772`) | cell; excluded from autocast (`spell.go:1913`) | cast picture 60 draws two flashes (`spellbolt.go:294`) |
| 27 | Curse | as 23 | to-hit `-(power*4/5+20)` | hostile; 3 percent award | as 23 |
| 28 | Slow | as 7 | speed `-(power/15+1)` (`spell.go:401`) | hostile; 3 percent award | as 7 |

Presentation limits that bind every spell:

- A spell's pictures are computed from its id: cast `2*id+8`, burst `2*id+9`
  (`pkg/data/projectile.go:176`). Only seven pictures have a flight length
  (`CastFlight`, `pkg/data/projectile.go:232`), so 21 spells draw no flying
  object. Which pictures fly, trail or draw a path is a function of the
  picture number, not of data.
- Sound slots are `spellSoundBase + picture` (`pkg/game/spellsound.go:40`).
- Spell names, icons and the 28-name book arrays are indexed by id
  (`bookNames [29]string`, `pkg/game/spell.go:20`; `pkg/ui/words.go:127`).
- Quick-spell keys hold four ids; quick and current spell ids are `-1` or
  0..23 in SAV.

## Spells with several effects at once

- Prismatic Spray, rays: see the next section.
- Fire Ball is the only area spell whose cell walk applies per-footprint
  division (`celleffect.go:957-964`).
- Heal and Drain Life carry damage columns and are excluded from the
  damaging arm by literal ids 6 and 11 in `pkg/data/spell.go`. A mod that
  re-marks another row as healing needs a row flag, not an id.

## Prismatic Spray with 100 rays

Observation: a ray is a selected victim. The original and this engine pick
the primary plus the best secondaries from the caster group's visible
candidates; one figure and one delivery is made per selected actor
(`applyPrismaticItem`, `pkg/sim/celleffect.go:1029`; `preparePrismatic`,
`pkg/sim/spelldelivery.go:101`; `spawnCastSet`, `pkg/game/spellbolt.go:196`).
The original caps the list at `(u8)min(power/20+2, 7)` (`MAGIC-SPRAY-134`).

What blocks 100:

1. Cap formula and width. `limit` is a `uint8` (`prismatic.go:64`), and the
   winner loop is capped at 10 (`prismatic.go:85`). The tooltip repeats the
   formula in a third place (`spell.go:695`, `RayCount uint8`). Three copies
   must become one rule table value.
2. Population. With 3 enemies in sight the spray reaches the primary and at
   most 2 others. A count above the number of candidates changes nothing
   until a rule says what an extra ray does. Options: (a) none (rays are
   distinct actors); (b) repeat the selected victims in rank order, each ray
   a separate hit and figure; (c) rays land on cells near victims. Owner
   decision 1.
3. Training. Each queued ray pays one cast award (`spelldelivery.go:109`), so
   100 rays pay 100 awards. A rule must pay once per cast.
4. Persistence. Each ray is one queued delivery saved with a delivery policy
   record (`spelldeliverysave.go`). The saved-object bound is `1<<20`
   (`pkg/sim/savedobjects.go:10`), so 100 per cast fit. Whether the original
   object graph accepts 100 pending transports in one frame is Unknown and is
   a named test in the story.
5. Drawing. The renderer has no ray cap: `spawnCastSet` loops over
   `ev.Victims`, `CastEvent.Victims` is an unbounded slice, and each figure
   gets its own seed. Colours cycle through seven blocks, so rays 8 to 100
   repeat colours (`chainTagCount = 7`). Cost is 100 path figures for 13
   ticks. The viewer needs a measured check at 100.

## Area spells with large radius (Fire_Ball)

- Radius column to cell set: `blastCells(x, y, r)` loops `-r..r` on both axes
  and builds each key from `uint8(cx+dx)` (`celleffect.go:592`). A radius
  that crosses the map edge wraps to the opposite side. This is a faithful
  byte quirk and must be bounded by the map for a custom radius.
- Damage walk: `applyAreaCells` visits every covered cell, every occupant in
  entity-id order, with no owner or hostility test (`celleffect.go:893-990`).
- Drawing: one burst object, picture 13, at the anchor cell
  (`pkg/sim/burst.go:62`, `pkg/game/spellbolt.go:302`). A radius of 5 would
  show an 11-by-11 damage area under a 3-cell explosion. The proposal draws
  one burst per covered cell (or per 3-by-3 tile) and keeps the record count in
  the world byte form.
- Scorch: every covered cell is scorched (`scorched.go:17`); the list is part
  of the hash and the SAV footer.
- Widths: `SpellRule.Radius`, the saved area radius and the area-effect byte
  are `uint8`, so 255 is the ceiling for any mod radius; the cell walk costs
  `(2r+1)^2` and needs a declared range.

## Heal on enemies

The only gates are the apply arm (`pkg/sim/spell.go:259-266`: a restorative
cast at a unit whose owner is hostile to the caster's owner pays, marks and
heals nothing) and the unbidden target choice (`spell.go:2197`). The hotfix
that put the refusal in the shared path applies to book, staff and scroll
alike. A knob must name both gates. It must also say whether a hostile heal
flips diplomacy (it does not today; a damaging cast does, `flipOnBlow`). Owner
decision 2.

## All-spells knobs

Value multipliers apply to the table at build time in `mapload`, change no
`pkg/sim` code and are the smallest step. Integers only: a multiplier is a
numerator and a denominator, rounded toward zero, with the original value as
`1/1`.

| Knob | Applies to | Where | Limit |
|---|---|---|---|
| `damage` multiplier | damage pair of every damaging row | table | in-flight byte payload; table `int32` |
| `heal` multiplier | damage pair of restorative rows | table | as above |
| `mana_cost` multiplier | col 1 | table | SAV book slot `uint16` read as `int16` |
| `duration` multiplier | col 14, col 11, effect duration | table | `uint16` words; `Remaining > 9600` never ticks |
| `radius` multiplier or addend | col 9 | table | `uint8` |
| `range` multiplier or addend | col 6 | table | `uint8` |
| `cast_speed` | `AttackCharge` floor 8, recovery | rules | `uint8` counters, clamp 255 |
| `training` multiplier | `(ManaCost+1)/2` award | rules | per-cast award site list below |
| `power` clamp | `spellPower`, 0..100 | rules | `lastingTicks` tables are 101 entries; five more tables to extend |
| Target-filter overrides | heal on hostile; unbidden heal targets; area hits (all, hostile, not own team); damaging self-cast; Prismatic secondaries (any, hostile, sight) | rules, new `SpellRule` flags | no data field today |
| Formula hooks | power, damage factor, range bonus, duration factor, per-spell magnitude arms | rules tables | below |

Award sites to merge for the training knob: `castSpell`, `castBookAt`,
`applyPrismaticItem`, `preparePrismatic` and the 3 percent awards of ids
20, 27, 28.

## Hard limits found

| Limit | Where | What it blocks |
|---|---|---|
| 28 spell ids: `Spellbook.Slots [28]`, `KnownSpells uint32`, `id > 28` checks | `pkg/sim/spellbook.go:28`, `pkg/mapload/spell.go:32`, about 37 sites | Any new spell (not part of this story) |
| Spell ids bounded at 32 by one `uint32` mask for effects and the immobilise gate | `gates.md`; `pkg/sim/effect.go` | Any id above 31 |
| Six area layers by literal id `{3,7,8,19,12,17}`; overlay art ids `{3,7,8,19}` | `pkg/sim/arealayers.go:12`, `pkg/data/projectile.go:195` | A new cloud or wall spell |
| Staged programs by id (4, 9, 21) and fixed offset tables | `pkg/sim/celleffect.go:601,617`; `area.md` | Changed geometry or stage counts |
| Pictures computed from id; seven flying pictures | `pkg/data/projectile.go:176,232` | New flight, trail or path art |
| Book slot: range `uint8`, defensive `uint8`, mana `uint16` read as `int16` | `pkg/sim/spellbook.go:22` | Range above 255, mana above 32767 in SAV |
| In-flight payload: base and spread bytes; effect magnitude `uint16`, duration `uint16` | `pkg/sim/spelldeliverysave.go:23,26-30` | Damage above 255 per roll and magnitudes above 32767 in the original object |
| Area radius, shape and mode bytes; cell key is `x + y<<8` | `pkg/sim/celleffect.go:53`, `blastCells` | Radius above 255; maps above 256 cells |
| Effect `Remaining` word, ticks stop above 9600 | `pkg/sim/effect.go:461` | Durations above about 600 seconds |
| `spellPower` clamp 0..100 and 101-entry duration tables | `pkg/sim/spell.go:752,155-203` | Powers above 100 |
| `SpellRule` is 36 bytes in the world byte form, `formatVersion = 95` | `pkg/sim/binary.go:1515,937` | Any new rule field needs a version bump and an old-save default |
| Presentation: `chainTagCount = 7` colours; one burst object per area | `pkg/game/spellbolt.go:256` | More colours; a large explosion |

## Proposed mod API

### TOML

New data file `data/spells.toml`, loaded by `game.data.add("data/spells.toml")`
like the existing data files. A row names its target by the spell's name
(`Data.bin` name, underscores as in the item grammar); keys are optional and
an absent key leaves the original value.

```toml
[global]                          # every spell row
damage_mul     = [3, 2]           # numerator, denominator
mana_cost_mul  = [1, 2]
duration_mul   = [2, 1]
radius_add     = 1
heal_hostile   = true             # heal applies to hostile owners
training_per_cast = true

[[spell]]
target   = "Fire Ball"
radius   = 5                      # cells, 0..255
damage   = { min = 7, max = 13 }
range    = 14
mana     = 20

[[spell]]
target   = "Prismatic Spray"
rays     = { base = 2, per_power = 20, max = 100 }   # ray cap rule
rays_extra = "repeat"             # none | repeat | cells
```

Per-row keys: `mana`, `range`, `complication`, `damage {min,max}`, `speed`,
`radius`, `area_duration`, `duration`, `effect {kind, magnitude, mode,
duration}`, `rays`, `heal_hostile`, `area_hits`, `self_cast`,
`autocast_targets`, `training`. Settings from `settings.toml` reach these
keys through `settings` in `init`, as `skill_cap` does.

Refusals name the mod, file and line, as the existing parsers do: an unknown
spell, an unknown key, a value outside its declared range, a duplicate edit of
one field by two mods (the later declares `load-after`).

### Starlark hooks

Three tiers. The proposal builds the first two.

1. Data keys (TOML). Constants and multipliers. No script.
2. Tabulated formulas. `game.rules.spells.power(fn)` and the like take a
   Starlark function. The runtime calls it once per input over a finite
   integer domain at load, range-checks every result, and freezes the table in
   `rules.Rules`, as `skillXP` is frozen today. Domains: power 0..100 for
   damage factor, range bonus, duration factor and the per-spell magnitude
   arms (`power/2`, `power/10+3`, `power/15+1`, `power*4/5+20`); `skill+Mind`
   0..170 for `spellPower`. The simulation reads integer tables and never
   calls the interpreter, so the determinism wall and the `pkg/sim` import
   rules hold.
3. Live hooks (only if the owner wants state-dependent formulas, decision 5).
   A Starlark function called per cast with an integer snapshot (caster
   level, Mind, target owner relation) for target filters and per-cast
   magnitudes. It needs an injected function type in `pkg/sim`, step-limited
   calls and the script digest in the world hash input. Not proposed for the
   first stories.

### Saving values beyond the original limits

SAV stays the only format and one producer writes it from the current state.
Fields the original cannot hold are written as their projection into the
original domain (clamped, never refused). The true values go into the
`AgainromMods` leaf, which already does this for skills:

- Book slot range, mana: projected to `uint8` and `0..32767`; true values in a
  new `spells` section of the mark (mark format 2, with format 1 still read).
- In-flight deliveries: base and spread bytes projected; the true operands
  already have a home in `CurrentDeliveryPolicy.Special`
  (`pkg/sim/spelldeliverysave.go:67`). The story must test that a damage above
  255 survives SAVE and LOAD through it.
- Area radius, magnitudes, durations: projected; true values in the mark.
- Spell rows themselves are not stored. The mark's mod set digest must equal
  the active set on LOAD, so the table is re-derived identically. A save from
  a different set is refused with the existing message.

What the original does with the projected values stays Medium until one
owner check, as in the milestone proposal.

### Determinism and hash

- The spell table is hashed (`encode`, `binary.go:2003`). A mod that changes a
  row changes the world hash. This is by design and is recorded with the mod
  digest.
- An unmodded game stays byte-identical. A new rule field is added with the
  original value as default. If the world byte form gains a field,
  `formatVersion` moves to 96 and 95 still decodes with the defaults. A
  `go test` of the existing pinned hashes proves it.
- Multipliers are integer num/den, rounded toward zero. No float, map order or
  wall time. Starlark runs only in `pkg/modrt`.

## Story split

Ordered smallest vertical result first. Each result is something a test fixture
mod and a headless run can show.

| # | Story | Result | Size | Depends on |
|---|---|---|---|---|
| 1 | Spell data keys and global multipliers | `data/spells.toml` per-row `mana`, `range`, `damage`, `radius`, `duration`, `effect`, and `[global]` multipliers applied at the table; settings reach them; refusals with file and line; mark format 2 carries projected book values; unmodded hash unchanged | M | none |
| 2 | Large-radius area spells | Fire_Ball radius up to 255 hits the right cells inside the map, draws one burst per covered area, scorch and SAV radius through the mark; ally and caster handling stated | M | 1 |
| 3 | Target-filter knobs | `heal_hostile` (apply arm and autocast choice), `area_hits`, `self_cast` as `SpellRule` flags; `formatVersion` 96 with default 95 load | M | 1 |
| 4 | Prismatic ray rule | One ray-count rule used by the selector, the tooltip and the draw; 10-slot loop lifted; extra-ray policy; one award per cast; 100 rays drawn and saved in a real-install witness | M | 1, 3 |
| 5 | Tabulated formula hooks | `power`, damage factor, range bonus, duration factor and magnitude arms as Starlark tables frozen into `Rules`; power above 100 | L | 1 |
| 6 | Live hooks (optional) | Per-cast Starlark filters and magnitudes | L | 5, owner decision 5 |

Not scheduled and outside this request: new spells beyond id 28, new layers,
new pictures and new staged programs. They need the spell-id bound, the layer
array and the picture arithmetic lifted first.

## Open decisions for the owner

1. Rays beyond the number of distinct enemies: none, repeat victims, or cells.
   Recommended: `repeat`.
2. Heal on an enemy: apply the heal only, or also treat it as a friendly act
   (no diplomacy change). Recommended: heal only, no diplomacy change.
3. Training per ray or per cast. Recommended: per cast.
4. Multiplier form: `[num, den]` integers as drawn above, or percent.
   Recommended: `[num, den]`.
5. Live hooks (tier 3): build or not. Recommended: not in the first five
   stories.
6. One original-game check of a projected save, to move that confidence from
   Medium.

## Unknowns

- Whether the original object graph accepts 100 pending spell transports in
  one frame (story 4 names a test).
- Whether a saved in-flight damage above 255 survives SAVE and LOAD through
  `CurrentDeliveryPolicy.Special` for every payload class (story 1 tests it).
- What the original does with a clamped book slot (Medium; owner check).
- The user-facing cost of a 255-radius burst per cell; story 2 measures it.
